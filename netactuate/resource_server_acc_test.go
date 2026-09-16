//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateServer_importWithVPCAndImageNamePlan(t *testing.T) {
	name := testAccName("server")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	locationName := testAccEnvString(t, "NETACTUATE_ACC_LOCATION_NAME")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	imageName := testAccEnvString(t, "NETACTUATE_ACC_IMAGE_NAME")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	configID := testAccServerConfigID(name, locationID, imageID, plan, contractID, password)
	configName := testAccServerConfigName(name, locationName, imageName, plan, contractID, password)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_LOCATION_NAME",
				"NETACTUATE_ACC_IMAGE_ID",
				"NETACTUATE_ACC_IMAGE_NAME",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
				"NETACTUATE_ACC_SERVER_PASSWORD",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: configID,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccTrackFromState("netactuate_server", "netactuate_server.test"),
					testAccCheckServerAPI("netactuate_server.test", "netactuate_vpc.test", name+".example.invalid", plan, locationID, imageID),
				),
			},
			{
				ResourceName: "netactuate_server.test",
				ImportState:  true,
				// E-01 failed before e2bd63f: an imported server with vpc_id set planned replacement.
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "allow_downsize_reboot", "package_billing", "params"},
			},
			{
				Config:   configID,
				PlanOnly: true,
			},
			{
				Config:   configName,
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckServerDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_server" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		// A server does not vanish the instant destroy returns: it goes through
		// termination first, so a single immediate GET can still find it and report a
		// false leak. Poll rather than assert once.
		gone := false
		for i := 0; i < 30; i++ {
			if _, err := clients.V2.GetServer(id); err != nil {
				gone = true
				break
			}
			time.Sleep(10 * time.Second)
		}
		if !gone {
			return fmt.Errorf("server still exists after waiting for termination: %d", id)
		}
	}
	return nil
}

func testAccServerConfigID(name string, locationID, imageID int, plan, contractID, password string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_server" "test" {
  hostname                    = %q
  plan                        = %q
  location_id                 = %d
  image_id                    = %d
  password                    = %q
  package_billing_contract_id = %q
  vpc_id                      = netactuate_vpc.test.vpc_id

  lifecycle {
    ignore_changes = [password]
  }
}
`, name, name, locationID, name+".example.invalid", plan, locationID, imageID, password, contractID)
}

func testAccServerConfigName(name, locationName, imageName, plan, contractID, password string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location    = %q
}

resource "netactuate_server" "test" {
  hostname                    = %q
  plan                        = %q
  location                    = %q
  image                       = %q
  password                    = %q
  package_billing_contract_id = %q
  vpc_id                      = netactuate_vpc.test.vpc_id

  lifecycle {
    ignore_changes = [password]
  }
}
`, name, name, locationName, name+".example.invalid", plan, locationName, imageName, password, contractID)
}
