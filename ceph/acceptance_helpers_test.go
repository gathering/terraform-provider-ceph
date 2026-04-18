//go:build acceptance

package ceph

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviderFactories = map[string]func() (*schema.Provider, error){
	"ceph": func() (*schema.Provider, error) {
		return Provider(), nil
	},
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("CEPH_CONF") == "" {
		t.Fatal("CEPH_CONF must be set for acceptance tests")
	}
}

func testAccProviderBlock() string {
	return fmt.Sprintf(`
provider "ceph" {
  config_path = %q
  entity      = "client.admin"
  cluster     = "ceph"
}
`, testAccCephConfigPath())
}

func testAccCephConfigPath() string {
	if v := os.Getenv("CEPH_CONF"); v != "" {
		return v
	}
	return "/etc/ceph/ceph.conf"
}

func testAccNewConfig() *Config {
	return &Config{
		ConfigPath: testAccCephConfigPath(),
		Entity:     "client.admin",
		Cluster:    "ceph",
	}
}
