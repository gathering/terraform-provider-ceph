//go:build acceptance

package ceph

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAuth_basic(t *testing.T) {
	const entity = "client.tfacc-auth"
	resourceName := "ceph_auth.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckAuthDestroyed(entity),
		Steps: []resource.TestStep{
			{
				Config: testAccAuthConfig(entity, `mon = "allow r"`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "entity", entity),
					resource.TestCheckResourceAttr(resourceName, "caps.mon", "allow r"),
					resource.TestCheckResourceAttrSet(resourceName, "key"),
					resource.TestCheckResourceAttrSet(resourceName, "keyring"),
				),
			},
			{
				Config: testAccAuthConfig(entity, `mon = "allow r", osd = "allow rw pool=rbd"`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "caps.mon", "allow r"),
					resource.TestCheckResourceAttr(resourceName, "caps.osd", "allow rw pool=rbd"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key", "keyring"},
			},
		},
	})
}

func testAccAuthConfig(entity, caps string) string {
	return testAccProviderBlock() + fmt.Sprintf(`
resource "ceph_auth" "test" {
  entity = %q
  caps = {
    %s
  }
}
`, entity, caps)
}

func testAccCheckAuthDestroyed(entity string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		config := testAccNewConfig()
		conn, err := config.GetCephConnection()
		if err != nil {
			return err
		}
		command, err := json.Marshal(map[string]interface{}{
			"prefix": "auth get",
			"format": "json",
			"entity": entity,
		})
		if err != nil {
			return err
		}
		buf, _, err := conn.MonCommand(command)
		if err != nil {
			return nil
		}
		var responses []authResponse
		if err := json.Unmarshal(buf, &responses); err != nil {
			return nil
		}
		if len(responses) > 0 {
			return fmt.Errorf("auth entity %q still exists after destroy", entity)
		}
		return nil
	}
}
