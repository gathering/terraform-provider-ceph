package ceph

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderSchema(t *testing.T) {
	ctx := context.Background()
	p := New()()

	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("provider schema has errors: %v", schemaResp.Diagnostics)
	}

	for _, attr := range []string{"config_path", "entity", "cluster", "keyring", "key", "mon_host"} {
		if _, ok := schemaResp.Schema.Attributes[attr]; !ok {
			t.Errorf("provider schema missing attribute %q", attr)
		}
	}
}

func TestProviderResources(t *testing.T) {
	ctx := context.Background()
	p := New()()

	if got := len(p.Resources(ctx)); got != 4 {
		t.Errorf("expected 4 resources, got %d", got)
	}
	if got := len(p.DataSources(ctx)); got != 3 {
		t.Errorf("expected 3 data sources, got %d", got)
	}
}

// providerAttrTypes is the tftypes object shape matching the provider schema.
var providerAttrTypes = map[string]tftypes.Type{
	"config_path": tftypes.String,
	"entity":      tftypes.String,
	"cluster":     tftypes.String,
	"keyring":     tftypes.String,
	"key":         tftypes.String,
	"mon_host":    tftypes.String,
}

// nullProviderConfig returns a ConfigureRequest where every attribute is null
// (simulates an empty provider block).
func nullProviderConfig(t *testing.T) provider.ConfigureRequest {
	t.Helper()
	return providerConfigWith(t, nil)
}

// providerConfigWith returns a ConfigureRequest with the given attributes set
// and all others null.
func providerConfigWith(t *testing.T, vals map[string]string) provider.ConfigureRequest {
	t.Helper()
	ctx := context.Background()
	p := New()()

	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)

	attrVals := make(map[string]tftypes.Value, len(providerAttrTypes))
	for k := range providerAttrTypes {
		if v, ok := vals[k]; ok {
			attrVals[k] = tftypes.NewValue(tftypes.String, v)
		} else {
			attrVals[k] = tftypes.NewValue(tftypes.String, nil)
		}
	}

	raw := tftypes.NewValue(tftypes.Object{AttributeTypes: providerAttrTypes}, attrVals)
	return provider.ConfigureRequest{
		Config: tfsdk.Config{Raw: raw, Schema: schemaResp.Schema},
	}
}

func configure(t *testing.T, req provider.ConfigureRequest) *Config {
	t.Helper()
	ctx := context.Background()
	p := New()()

	var resp provider.ConfigureResponse
	p.Configure(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure diagnostics: %v", resp.Diagnostics)
	}
	return resp.ResourceData.(*Config)
}

func TestProviderConfigure_EnvVars(t *testing.T) {
	t.Setenv("CEPH_CONF", "/tmp/ceph.conf")
	t.Setenv("CEPH_ENTITY", "client.test")
	t.Setenv("CEPH_CLUSTER", "mycluster")
	t.Setenv("CEPH_KEYRING", "[client.test]\n\tkey = ABC")
	t.Setenv("CEPH_KEY", "ABC123")
	t.Setenv("CEPH_MON_HOST", "10.0.0.1:6789")

	got := configure(t, nullProviderConfig(t))

	if got.ConfigPath != "/tmp/ceph.conf" {
		t.Errorf("ConfigPath: got %q, want %q", got.ConfigPath, "/tmp/ceph.conf")
	}
	if got.Entity != "client.test" {
		t.Errorf("Entity: got %q, want %q", got.Entity, "client.test")
	}
	if got.Cluster != "mycluster" {
		t.Errorf("Cluster: got %q, want %q", got.Cluster, "mycluster")
	}
	if got.Keyring != "[client.test]\n\tkey = ABC" {
		t.Errorf("Keyring: got %q, want %q", got.Keyring, "[client.test]\n\tkey = ABC")
	}
	if got.Key != "ABC123" {
		t.Errorf("Key: got %q, want %q", got.Key, "ABC123")
	}
	if got.MonHost != "10.0.0.1:6789" {
		t.Errorf("MonHost: got %q, want %q", got.MonHost, "10.0.0.1:6789")
	}
}

func TestProviderConfigure_ConfigOverridesEnvVars(t *testing.T) {
	t.Setenv("CEPH_CONF", "/tmp/from-env.conf")
	t.Setenv("CEPH_ENTITY", "client.env")
	t.Setenv("CEPH_CLUSTER", "envcluster")
	t.Setenv("CEPH_KEY", "envkey")
	t.Setenv("CEPH_MON_HOST", "10.0.0.1:6789")

	req := providerConfigWith(t, map[string]string{
		"config_path": "/etc/ceph/ceph.conf",
		"entity":      "client.admin",
		"cluster":     "cfgcluster",
		"key":         "cfgkey",
		"mon_host":    "192.168.1.1:6789",
	})
	got := configure(t, req)

	if got.ConfigPath != "/etc/ceph/ceph.conf" {
		t.Errorf("ConfigPath: got %q, want %q", got.ConfigPath, "/etc/ceph/ceph.conf")
	}
	if got.Entity != "client.admin" {
		t.Errorf("Entity: got %q, want %q", got.Entity, "client.admin")
	}
	if got.Cluster != "cfgcluster" {
		t.Errorf("Cluster: got %q, want %q", got.Cluster, "cfgcluster")
	}
	if got.Key != "cfgkey" {
		t.Errorf("Key: got %q, want %q", got.Key, "cfgkey")
	}
	if got.MonHost != "192.168.1.1:6789" {
		t.Errorf("MonHost: got %q, want %q", got.MonHost, "192.168.1.1:6789")
	}
}

func TestProviderConfigure_ClusterDefault(t *testing.T) {
	got := configure(t, nullProviderConfig(t))

	if got.Cluster != "ceph" {
		t.Errorf("Cluster: got %q, want default %q", got.Cluster, "ceph")
	}
}

func TestProviderConfigure_ResourceDataAndDataSourceDataMatch(t *testing.T) {
	t.Setenv("CEPH_MON_HOST", "10.0.0.2:6789")

	ctx := context.Background()
	p := New()()

	var resp provider.ConfigureResponse
	p.Configure(ctx, nullProviderConfig(t), &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure diagnostics: %v", resp.Diagnostics)
	}

	if resp.ResourceData != resp.DataSourceData {
		t.Error("ResourceData and DataSourceData should be the same *Config pointer")
	}
}
