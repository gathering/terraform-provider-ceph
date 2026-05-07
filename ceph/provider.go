package ceph

import (
	"context"
	"os"

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
				Description: "Path to the ceph config. Can also be set via the CEPH_CONF environment variable.",
			},
			"entity": schema.StringAttribute{
				Optional:    true,
				Description: "The cephx entity to use to connect to Ceph (i.e.: client.admin). Can also be set via the CEPH_ENTITY environment variable.",
			},
			"cluster": schema.StringAttribute{
				Optional:    true,
				Description: "The name of the Ceph cluster to use. Can also be set via the CEPH_CLUSTER environment variable.",
			},
			"keyring": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The actual keyring (not a path to a file) to use to connect to Ceph. Can also be set via the CEPH_KEYRING environment variable.",
			},
			"key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The actual key (not a path to a file) to use to connect to Ceph. Can also be set via the CEPH_KEY environment variable.",
			},
			"mon_host": schema.StringAttribute{
				Optional:    true,
				Description: "Monitor address(es) to connect to. Can also be set via the CEPH_MON_HOST environment variable.",
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

	configPath := os.Getenv("CEPH_CONF")
	entity := os.Getenv("CEPH_ENTITY")
	cluster := os.Getenv("CEPH_CLUSTER")
	keyring := os.Getenv("CEPH_KEYRING")
	key := os.Getenv("CEPH_KEY")
	monHost := os.Getenv("CEPH_MON_HOST")

	if !data.ConfigPath.IsNull() {
		configPath = data.ConfigPath.ValueString()
	}
	if !data.Entity.IsNull() {
		entity = data.Entity.ValueString()
	}
	if !data.Cluster.IsNull() {
		cluster = data.Cluster.ValueString()
	}
	if !data.Keyring.IsNull() {
		keyring = data.Keyring.ValueString()
	}
	if !data.Key.IsNull() {
		key = data.Key.ValueString()
	}
	if !data.MonHost.IsNull() {
		monHost = data.MonHost.ValueString()
	}

	if cluster == "" {
		cluster = "ceph"
	}

	config := &Config{
		ConfigPath: configPath,
		Entity:     entity,
		Cluster:    cluster,
		Keyring:    keyring,
		Key:        key,
		MonHost:    monHost,
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
