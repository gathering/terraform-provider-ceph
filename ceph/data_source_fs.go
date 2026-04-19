package ceph

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &fsDataSource{}
var _ datasource.DataSourceWithConfigure = &fsDataSource{}

type fsDataSource struct {
	config *Config
}

func newFSDataSource() datasource.DataSource {
	return &fsDataSource{}
}

func (d *fsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fs"
}

func (d *fsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.config = req.ProviderData.(*Config)
}

func (d *fsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source allows you to get information about an existing CephFS filesystem.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the filesystem.",
			},
			"metadata_pool": schema.StringAttribute{
				Computed:    true,
				Description: "Pool used for filesystem metadata.",
			},
			"data_pools": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Data pools attached to the filesystem.",
			},
		},
	}
}

func (d *fsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config fsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := d.config.GetCephConnection()
	if err != nil {
		resp.Diagnostics.AddError("Unable to connect to Ceph", err.Error())
		return
	}

	name := config.Name.ValueString()
	fs, err := fsGet(conn, name)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error reading filesystem %q", name), err.Error())
		return
	}
	if fs == nil {
		resp.Diagnostics.AddError("Filesystem not found", fmt.Sprintf("Filesystem %q does not exist", name))
		return
	}

	dpSet, diags := types.SetValueFrom(ctx, types.StringType, fs.DataPools)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, fsDataSourceModel{
		Name:         types.StringValue(fs.Name),
		MetadataPool: types.StringValue(fs.MetadataPool),
		DataPools:    dpSet,
	})...)
}
