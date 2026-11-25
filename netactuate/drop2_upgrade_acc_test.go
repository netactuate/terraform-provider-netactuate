//go:build acctest

package netactuate

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// A6 (WS8-AC6): state written by the released v0.4.9 for netactuate_vpc and
// netactuate_nke_cluster plans with no diff on this build.
func TestAccNetactuateDrop2_upgradeNoDiff(t *testing.T) {
	drop2AccPrecheck(t, true)
	config := drop2AccFixture(t, drop2AccName(), "nke_upgrade")
	resource.Test(t, resource.TestCase{CheckDestroy: testAccCheckNKEClusterDestroy, Steps: []resource.TestStep{
		{Config: config, Check: testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"), ExternalProviders: map[string]resource.ExternalProvider{"netactuate": {Source: "netactuate/netactuate", VersionConstraint: "= 0.4.9"}}},
		{Config: config, ProviderFactories: testAccProviderFactories, PlanOnly: true, ExpectNonEmptyPlan: false},
	}})
}
