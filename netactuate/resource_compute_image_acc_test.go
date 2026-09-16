//go:build acctest

package netactuate

import (
	"fmt"
	"log"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateImage_APIReadImportPlanServerIDExcludedAndOutOfBandDelete(t *testing.T) {
	name := testAccName("image")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	config := testAccImageWithDataSourceConfig(name, locationID, imageID, plan, contractID, password)

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
					testAccCheckImageDataSourceAPI("data.netactuate_image.test", name, name),
				),
			},
			{
				ResourceName:      "netactuate_image.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The API does not expose server_id on image reads. Excluding it is the
				// expected import behavior and the following plan-only step guards
				// against a spurious ForceNew diff for the configured source server.
				ImportStateVerifyIgnore: []string{"keep_ssh_userdirs", "server_id"},
			},
			{
				Config:   config,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_image"] {
						resp, err := clients.V2.DeleteImage(id)
						if err != nil {
							log.Printf("[WARN] out-of-band image %d delete failed: %s", id, err)
							continue
						}
						if _, err := clients.V2.WaitForImageQueue(resp.QueueID); err != nil {
							log.Printf("[WARN] out-of-band image %d delete wait failed: %s", id, err)
						}
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_image.test"),
			},
		},
	})
}

func testAccCheckImageAPI(address, name, description string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		image, err := testAccClients().V2.GetImage(id)
		if err != nil {
			return err
		}
		if image.Name != name {
			return fmt.Errorf("API image name = %q, want %q", image.Name, name)
		}
		gotDescription := ""
		if image.Description != nil {
			gotDescription = *image.Description
		}
		if gotDescription != description {
			return fmt.Errorf("API image description = %q, want %q", gotDescription, description)
		}
		return nil
	}
}

func testAccCheckImageDataSourceAPI(address, name, description string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		image, err := testAccClients().V2.GetImage(id)
		if err != nil {
			return err
		}
		if image.Name != name {
			return fmt.Errorf("API image data source name = %q, want %q", image.Name, name)
		}
		gotDescription := ""
		if image.Description != nil {
			gotDescription = *image.Description
		}
		if gotDescription != description {
			return fmt.Errorf("API image data source description = %q, want %q", gotDescription, description)
		}
		if rs.Primary.Attributes["name"] != image.Name {
			return fmt.Errorf("image data source name = %q, API has %q", rs.Primary.Attributes["name"], image.Name)
		}
		if rs.Primary.Attributes["description"] != gotDescription {
			return fmt.Errorf("image data source description = %q, API has %q", rs.Primary.Attributes["description"], gotDescription)
		}
		return nil
	}
}

func testAccImageWithDataSourceConfig(name string, locationID, imageID int, plan, contractID, password string) string {
	return testAccImageConfig(name, locationID, imageID, plan, contractID, password) + `
data "netactuate_image" "test" {
  id = netactuate_image.test.id
}
`
}
