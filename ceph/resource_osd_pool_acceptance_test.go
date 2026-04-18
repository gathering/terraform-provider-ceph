//go:build acceptance

package ceph

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOSDPool_basic(t *testing.T) {
	const name = "tfacc-pool"
	resourceName := "ceph_osd_pool.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckOSDPoolDestroyed(name),
		Steps: []resource.TestStep{
			{
				Config: testAccOSDPoolConfig(name, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "type", "replicated"),
				),
			},
			{
				Config: testAccOSDPoolConfig(name, `application = ["rbd"]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "application.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccOSDPoolConfig(name, extra string) string {
	return testAccProviderBlock() + fmt.Sprintf(`
resource "ceph_osd_pool" "test" {
  name = %q
  %s
}
`, name, extra)
}

func testAccCheckOSDPoolDestroyed(name string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		config := testAccNewConfig()
		conn, err := config.GetCephConnection()
		if err != nil {
			return err
		}
		pool, status, err := osdPoolGetAll(conn, name)
		if err != nil {
			if strings.Contains(status, "ENOENT") {
				return nil
			}
			return nil
		}
		if pool != nil {
			return fmt.Errorf("osd pool %q still exists after destroy", name)
		}
		return nil
	}
}
