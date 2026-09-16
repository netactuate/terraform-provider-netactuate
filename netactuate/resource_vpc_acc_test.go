//go:build acctest

package netactuate

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateVPC_importPlanLocationAndOutOfBandDelete(t *testing.T) {
	name := testAccName("vpc")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	locationName := testAccEnvString(t, "NETACTUATE_ACC_LOCATION_NAME")
	differentLocationName := testAccDifferentLocationName(t, locationID)
	configID := testAccVPCConfigID(name, locationID)
	configName := testAccVPCConfigName(name, locationName)
	configDifferentLocation := testAccVPCConfigName(name, differentLocationName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID", "NETACTUATE_ACC_LOCATION_NAME") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: configID,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCFromAPI("netactuate_vpc.test", name, locationID),
					testAccCheckVPCNameserversAPI("netactuate_vpc.test"),
				),
			},
			{
				ResourceName:            "netactuate_vpc.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enable_default_snat"},
			},
			{
				Config:   configID,
				PlanOnly: true,
			},
			{
				Config:   configName,
				PlanOnly: true,
			},
			{
				Config:             configDifferentLocation,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_vpc"] {
						_ = clients.V3.DeleteVPC(id)
					}
				},
				// Refreshing an API-deleted VPC must remove it from state.
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_vpc.test"),
			},
		},
	})
}

func TestAccNetactuateVPC_locationNameCreate(t *testing.T) {
	name := testAccName("vpc-location-name")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	locationName := testAccEnvString(t, "NETACTUATE_ACC_LOCATION_NAME")
	config := testAccVPCConfigName(name, locationName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID", "NETACTUATE_ACC_LOCATION_NAME") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCFromAPI("netactuate_vpc.test", name, locationID),
				),
			},
		},
	})
}

func testAccCheckVPCDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_vpc" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetVPC(id); err == nil {
			return fmt.Errorf("VPC still exists: %d", id)
		}
	}
	return nil
}

func testAccCheckVPCFromAPI(address, expectedLabel string, expectedLocationID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		vpc, err := testAccClients().V3.GetVPC(id)
		if err != nil {
			return err
		}
		if vpc.Metadata.Label != expectedLabel {
			return fmt.Errorf("%s API label = %q, want %q", address, vpc.Metadata.Label, expectedLabel)
		}
		if vpc.Location.ID != expectedLocationID {
			return fmt.Errorf("%s API location_id = %d, want %d", address, vpc.Location.ID, expectedLocationID)
		}
		return nil
	}
}

func testAccCheckVPCNameserversAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		vpc, err := testAccClients().V3.GetVPC(id)
		if err != nil {
			return err
		}
		if vpc.DHCP == nil || vpc.DHCP.Nameservers == nil {
			return fmt.Errorf("%s API nameservers are missing", address)
		}
		if len(vpc.DHCP.Nameservers.IPv4) == 0 && len(vpc.DHCP.Nameservers.IPv6) == 0 {
			return fmt.Errorf("%s API nameservers are empty", address)
		}
		return nil
	}
}

func testAccDifferentLocationName(t *testing.T, currentLocationID int) string {
	t.Helper()
	if currentLocationID == 0 {
		return ""
	}
	locations, err := testAccClients().V2.GetLocations()
	if err != nil {
		if testAccIsAcceptanceRun() {
			t.Skipf("list locations for different-location check: %v", err)
		}
		return ""
	}
	for _, location := range locations {
		if location.ID != currentLocationID && location.Name != "" {
			return location.Name
		}
	}
	if testAccIsAcceptanceRun() {
		t.Skip("NETACTUATE account catalog has no second location for replacement plan check")
	}
	return ""
}

func testAccIsAcceptanceRun() bool {
	return os.Getenv(resource.EnvTfAcc) != ""
}

func testAccVPCConfigID(name string, locationID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}
`, name, name, locationID)
}

func testAccVPCConfigName(name, locationName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location    = %q
}
`, name, name, locationName)
}
