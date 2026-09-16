//go:build acctest

package netactuate

import (
	"fmt"
	"log"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateMetal_BuyBuildAPIReadLocationIDAndCancel(t *testing.T) {
	name := testAccName("metal")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	deviceID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_DEVICE_ID")
	profileID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_PROFILE_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	config := testAccMetalConfig(name, locationID, deviceID, profileID, password)
	defer testAccCancelTrackedMetal()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_DEDICATED_DEVICE_ID",
				"NETACTUATE_ACC_DEDICATED_PROFILE_ID",
				"NETACTUATE_ACC_SERVER_PASSWORD",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMetalCanceled,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_metal", "netactuate_metal.test"),
					testAccCheckMetalAPI("netactuate_metal.test", name+".example.invalid", locationID, deviceID),
				),
			},
		},
	})
}

func testAccCheckMetalAPI(address, hostname string, locationID, deviceID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		metal, err := testAccClients().V2.GetMetal(id)
		if err != nil {
			return err
		}
		if metal.Hostname != hostname {
			return fmt.Errorf("API metal hostname = %q, want %q", metal.Hostname, hostname)
		}
		if metal.DatacenterID != locationID {
			return fmt.Errorf("API metal datacenter_id = %d, want %d", metal.DatacenterID, locationID)
		}
		if metal.ID != deviceID {
			return fmt.Errorf("API metal device id = %d, want %d", metal.ID, deviceID)
		}
		return nil
	}
}

func testAccCheckMetalCanceled(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_metal" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		metal, err := clients.V2.GetMetal(id)
		if err != nil {
			if isNotFoundOrUnprocessable(err) {
				log.Printf("[INFO] Dedicated server %d cancellation confirmed by missing package", id)
				return nil
			}
			log.Printf("[ERROR] Dedicated server %d cancellation status could not be verified: %s", id, err)
			return err
		}
		if metal.Canceling == 1 {
			log.Printf("[INFO] Dedicated server %d cancellation confirmed by canceling flag", id)
			return nil
		}
		log.Printf("[ERROR] Dedicated server %d cancellation was not confirmed", id)
		return fmt.Errorf("dedicated server still exists and is not canceling: %d", id)
	}
	return nil
}

func testAccCancelTrackedMetal() {
	clients := testAccClients()
	for id := range testAccCreated["netactuate_metal"] {
		commentsValue := "Cancel from acceptance test defer"
		req := &gona.CancelRequest{
			MBPKGID:    id,
			CancelType: "Immediate",
			Agree:      1,
			Comments:   &commentsValue,
		}
		if _, err := clients.V2.CancelPackage(req); err != nil {
			if isNotFoundOrUnprocessable(err) {
				log.Printf("[INFO] Dedicated server %d cancellation defer confirmed package is gone", id)
				continue
			}
			log.Printf("[ERROR] Dedicated server %d cancellation defer failed: %s", id, err)
			continue
		}
		log.Printf("[INFO] Dedicated server %d cancellation defer succeeded", id)
	}
}

func testAccMetalConfig(name string, locationID, deviceID, profileID int, password string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_metal" "test" {
  hostname    = %q
  location_id = %d
  device_id   = %d
  profile     = %d
  password    = %q

  lifecycle {
    ignore_changes = [password]
  }
}
`, name+".example.invalid", locationID, deviceID, profileID, password)
}
