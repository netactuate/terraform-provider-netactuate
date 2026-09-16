//go:build acctest

package netactuate

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccNetactuateRouterVRFBGP_deleteIsolation exists to answer one question and nothing else.
//
// A VRF carrying BGP can fail deletion when an unguarded subscript in VyOS raises KeyError
// on the delete path and leaves a billable router behind.
//
// In both runs something else had already failed first: a rejected neighbour create in one, a
// rejected local ASN update in the other. So it was not possible to tell whether ANY VRF with
// BGP cannot be deleted, or only one left behind by a failed write.
//
// This test creates a router, a VRF, and a BGP configuration on that VRF, and then destroys.
// Nothing else. No neighbour, no prefix list, no static route, no update step. The teardown is
// the assertion:
//
//   - destroy succeeds: the delete crash is downstream of a failed write, and the fix belongs
//     with the write path.
//   - destroy fails the same way: no VRF carrying BGP can be deleted, and that is a release
//     blocker rather than a finding.
//
// Deliberately no Check functions. Anything asserted here could fail for its own reasons and
// muddy the single signal this test is for.
func TestAccNetactuateRouterVRFBGP_deleteIsolation(t *testing.T) {
	name := testAccName("vrf-bgp-del")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_PLAN",
			)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRouterVRFBGPDeleteIsolationConfig(name, locationID, plan),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_router", "netactuate_router.test"),
				),
			},
		},
	})
}

func testAccRouterVRFBGPDeleteIsolationConfig(name string, locationID int, plan string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_router" "test" {
  name        = %q
  description = %q
  location_id = %d
  plan        = %q
}

resource "netactuate_router_vrf" "test" {
  router_id   = netactuate_router.test.router_id
  name        = %q
  description = %q
}

resource "netactuate_router_vrf_bgp" "test" {
  router_id = netactuate_router.test.router_id
  vrf_id    = netactuate_router_vrf.test.vrf_id
  local_asn = "64512"

  networks {
    subnet = "198.51.100.0/24"
  }
}
`, name, name+" delete isolation", locationID, plan, name+" vrf", name+" vrf delete isolation")
}
