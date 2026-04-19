package ceph

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &authDataSource{}
var _ datasource.DataSourceWithConfigure = &authDataSource{}

type authDataSource struct {
	config *Config
}

func newAuthDataSource() datasource.DataSource {
	return &authDataSource{}
}

func (d *authDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth"
}

func (d *authDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.config = req.ProviderData.(*Config)
}

func (d *authDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source allows you to get information about a ceph client.",
		Attributes: map[string]schema.Attribute{
			"entity": schema.StringAttribute{
				Required:    true,
				Description: "The entity name (i.e.: client.admin)",
			},
			"caps": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "The caps of the entity",
			},
			"keyring": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The cephx keyring of the entity",
			},
			"key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The cephx key of the entity",
			},
		},
	}
}

func (d *authDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config authDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := d.config.GetCephConnection()
	if err != nil {
		resp.Diagnostics.AddError("Unable to connect to Ceph", err.Error())
		return
	}

	command, err := json.Marshal(map[string]interface{}{
		"prefix": "auth get",
		"format": "json",
		"entity": config.Entity.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error building auth get command", err.Error())
		return
	}

	buf, _, err := conn.MonCommand(command)
	if err != nil {
		resp.Diagnostics.AddError("Error on auth get command", err.Error())
		return
	}

	var authResponses []authResponse
	if err = json.Unmarshal(buf, &authResponses); err != nil {
		resp.Diagnostics.AddError("Error unmarshaling auth response", err.Error())
		return
	}

	fullModel, diags := authModelFromResponse(ctx, authResponses)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, authDataSourceModel{
		Entity:  fullModel.Entity,
		Caps:    fullModel.Caps,
		Key:     fullModel.Key,
		Keyring: fullModel.Keyring,
	})...)
}
