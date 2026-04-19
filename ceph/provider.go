package ceph

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &cephProvider{}

type cephProvider struct{}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &cephProvider{}
	}
}

func (p *cephProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ceph"
}

func (p *cephProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"config_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to the ceph config",
			},
			"entity": schema.StringAttribute{
				Optional:    true,
				Description: "The cephx entity to use to connect to Ceph (i.e.: client.admin).",
			},
			"cluster": schema.StringAttribute{
				Optional:    true,
				Description: "The name of the Ceph cluster to use.",
			},
			"keyring": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The actual keyring (not a path to a file) to use to connect to Ceph.",
			},
			"key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The actual key (not a path to a file) to use to connect to Ceph.",
			},
			"mon_host": schema.StringAttribute{
				Optional:    true,
				Description: "Monitor address(es) to connect to.",
			},
		},
	}
}

type providerModel struct {
	ConfigPath types.String `tfsdk:"config_path"`
	Entity     types.String `tfsdk:"entity"`
	Cluster    types.String `tfsdk:"cluster"`
	Keyring    types.String `tfsdk:"keyring"`
	Key        types.String `tfsdk:"key"`
	MonHost    types.String `tfsdk:"mon_host"`
}

func (p *cephProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster := "ceph"
	if v := data.Cluster.ValueString(); v != "" {
		cluster = v
	}

	config := &Config{
		ConfigPath: data.ConfigPath.ValueString(),
		Entity:     data.Entity.ValueString(),
		Cluster:    cluster,
		Keyring:    data.Keyring.ValueString(),
		Key:        data.Key.ValueString(),
		MonHost:    data.MonHost.ValueString(),
	}

	resp.DataSourceData = config
	resp.ResourceData = config
}

func (p *cephProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newWaitOnlineResource,
		newAuthResource,
		newOSDPoolResource,
		newFSResource,
	}
}

func (p *cephProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newAuthDataSource,
		newOSDPoolDataSource,
		newFSDataSource,
	}
}
