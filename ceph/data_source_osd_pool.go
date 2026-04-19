package ceph

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &osdPoolDataSource{}
var _ datasource.DataSourceWithConfigure = &osdPoolDataSource{}

type osdPoolDataSource struct {
	config *Config
}

func newOSDPoolDataSource() datasource.DataSource {
	return &osdPoolDataSource{}
}

func (d *osdPoolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_osd_pool"
}

func (d *osdPoolDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.config = req.ProviderData.(*Config)
}

func (d *osdPoolDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source allows you to get information about an existing Ceph OSD pool.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the pool.",
			},
			"pg_num": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of placement groups.",
			},
			"size": schema.Int64Attribute{
				Computed:    true,
				Description: "Replication factor.",
			},
			"min_size": schema.Int64Attribute{
				Computed:    true,
				Description: "Minimum number of replicas required for I/O.",
			},
			"crush_rule": schema.StringAttribute{
				Computed:    true,
				Description: "CRUSH rule name for the pool.",
			},
			"application": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Application tags enabled on the pool (e.g. rbd, cephfs, rgw).",
			},
		},
	}
}

func (d *osdPoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config osdPoolDatasourceModel
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
	pool, status, err := osdPoolGetAll(conn, name)
	if err != nil {
		if strings.Contains(status, "ENOENT") {
			resp.Diagnostics.AddError("Pool not found", fmt.Sprintf("Pool %q does not exist", name))
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("Error reading pool %q", name), err.Error())
		return
	}

	apps, err := osdPoolApplicationGet(conn, name)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error reading applications for pool %q", name), err.Error())
		return
	}

	appList, diags := types.ListValueFrom(ctx, types.StringType, apps)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	model := osdPoolDatasourceModel{
		Name:        types.StringValue(pool.Pool),
		PgNum:       types.Int64Value(int64(pool.PgNum)),
		Size:        types.Int64Value(int64(pool.Size)),
		MinSize:     types.Int64Value(int64(pool.MinSize)),
		CrushRule:   types.StringValue(pool.CrushRule),
		Application: appList,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
