package ceph

import (
	"testing"
)

// TestProviderSchema validates that the provider schema is internally consistent.
// This catches typos in schema definitions, missing required fields, and invalid
// attribute combinations without needing a live Ceph cluster.
func TestProviderSchema(t *testing.T) {
	p := Provider()
	if err := p.InternalValidate(); err != nil {
		t.Fatalf("provider schema validation failed: %v", err)
	}
}

// TestProviderResources checks that all expected resources and data sources are registered.
func TestProviderResources(t *testing.T) {
	p := Provider()

	resources := []string{
		"ceph_auth",
		"ceph_osd_pool",
		"ceph_wait_online",
		"ceph_fs",
	}
	for _, name := range resources {
		if _, ok := p.ResourcesMap[name]; !ok {
			t.Errorf("provider is missing resource %q", name)
		}
	}

	dataSources := []string{
		"ceph_auth",
		"ceph_osd_pool",
		"ceph_fs",
	}
	for _, name := range dataSources {
		if _, ok := p.DataSourcesMap[name]; !ok {
			t.Errorf("provider is missing data source %q", name)
		}
	}
}
