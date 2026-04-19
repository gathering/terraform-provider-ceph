package ceph

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
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
