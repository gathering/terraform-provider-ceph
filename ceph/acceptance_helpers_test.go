//go:build acceptance

package ceph

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"ceph": providerserver.NewProtocol6WithError(New()()),
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
