//go:build acceptance

package ceph

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccFS_basic(t *testing.T) {
	const (
		fsName   = "tfacc-fs"
		metaPool = "tfacc-fs-meta"
		dataPool = "tfacc-fs-data"
	)
	resourceName := "ceph_fs.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckFSDestroyed(fsName),
		Steps: []resource.TestStep{
			{
				Config: testAccFSConfig(fsName, metaPool, dataPool),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", fsName),
					resource.TestCheckResourceAttr(resourceName, "metadata_pool", metaPool),
					resource.TestCheckResourceAttr(resourceName, "data_pools.#", "1"),
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

func testAccFSConfig(fsName, metaPool, dataPool string) string {
	return testAccProviderBlock() + fmt.Sprintf(`
resource "ceph_osd_pool" "meta" {
  name = %q
}

resource "ceph_osd_pool" "data" {
  name = %q
}

resource "ceph_fs" "test" {
  name          = %q
  metadata_pool = ceph_osd_pool.meta.name
  data_pools    = [ceph_osd_pool.data.name]
}
`, metaPool, dataPool, fsName)
}

func testAccCheckFSDestroyed(name string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		config := testAccNewConfig()
		conn, err := config.GetCephConnection()
		if err != nil {
			return err
		}
		fs, err := fsGet(conn, name)
		if err != nil {
			return nil
		}
		if fs != nil {
			return fmt.Errorf("filesystem %q still exists after destroy", name)
		}
		return nil
	}
}
