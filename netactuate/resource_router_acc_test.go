//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateRouter_locationNamePlan(t *testing.T) {
	name := testAccName("router")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	locationName := testAccEnvString(t, "NETACTUATE_ACC_LOCATION_NAME")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	configID := testAccRouterConfigID(name, locationID, plan)
	configName := testAccRouterConfigName(name, locationName, plan)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_LOCATION_NAME",
				"NETACTUATE_ACC_PLAN",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterDestroy,
		Steps: []resource.TestStep{
			{
				Config: configID,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_router", "netactuate_router.test"),
					resource.TestCheckResourceAttr("netactuate_router.test", "name", name),
				),
			},
			{
				ResourceName:      "netactuate_router.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router.test"),
				ImportStateVerify: true,
				// The router read endpoints do not return the original package selection.
				ImportStateVerifyIgnore: []string{"package_id", "plan"},
			},
			{
				// A-04 failed before 4c3e734 for imported routers with location fields.
				Config:   configName,
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckRouterDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_router" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetRouter(id); err == nil {
			return fmt.Errorf("router still exists: %d", id)
		}
	}
	return nil
}

func testAccRouterConfigID(name string, locationID int, plan string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_router" "test" {
  name        = %q
  description = %q
  location_id = %d
  plan        = %q
}
`, name, name, locationID, plan)
}

func testAccRouterConfigName(name, locationName, plan string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_router" "test" {
  name        = %q
  description = %q
  location    = %q
  plan        = %q
}
`, name, name, locationName, plan)
}
