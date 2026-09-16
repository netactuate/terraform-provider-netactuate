//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateImage_importPlan(t *testing.T) {
	name := testAccName("image")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	config := testAccImageConfig(name, locationID, imageID, plan, contractID, password)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_IMAGE_ID",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
				"NETACTUATE_ACC_SERVER_PASSWORD",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_server", "netactuate_server.source"),
					testAccTrackFromState("netactuate_image", "netactuate_image.test"),
					testAccCheckImageAPI("netactuate_image.test", name, name),
				),
			},
			{
				ResourceName: "netactuate_image.test",
				ImportState:  true,
				// The API does not expose server_id on image reads. Excluding it is
				// expected, and the following plan-only step guards against a diff.
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"keep_ssh_userdirs", "server_id"},
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckImageDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_image" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V2.GetImage(id); err == nil {
			return fmt.Errorf("image still exists: %d", id)
		}
	}
	return nil
}

func testAccImageConfig(name string, locationID, imageID int, plan, contractID, password string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_server" "source" {
  hostname                    = %q
  plan                        = %q
  location_id                 = %d
  image_id                    = %d
  password                    = %q
  package_billing_contract_id = %q

  lifecycle {
    ignore_changes = [password]
  }
}

resource "netactuate_image" "test" {
  name        = %q
  description = %q
  server_id   = netactuate_server.source.id
}
`, name+".example.invalid", plan, locationID, imageID, password, contractID, name, name)
}
