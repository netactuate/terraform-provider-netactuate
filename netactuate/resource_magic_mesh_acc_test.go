//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

var (
	testAccMagicMeshOutOfBandDeleteID int
	testAccMagicMeshRouterMeshID      int
	testAccMagicMeshRouterRouterID    int
)

func TestAccNetactuateMagicMesh_importPlanUpdateAndOutOfBandDelete(t *testing.T) {
	name := testAccName("magic-mesh")
	updatedName := name + "-updated"
	configInitial := testAccMagicMeshConfig(name)
	configUpdated := testAccMagicMeshConfig(updatedName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMagicMeshDestroy,
		Steps: []resource.TestStep{
			{
				Config: configInitial,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_magic_mesh", "netactuate_magic_mesh.test"),
					testAccRememberID("netactuate_magic_mesh.test", &testAccMagicMeshOutOfBandDeleteID),
					testAccCheckMagicMeshFromAPI("netactuate_magic_mesh.test", name, name+" description"),
				),
			},
			{
				ResourceName:      "netactuate_magic_mesh.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   configInitial,
				PlanOnly: true,
			},
			{
				Config: configUpdated,
				Check:  testAccCheckMagicMeshFromAPI("netactuate_magic_mesh.test", updatedName, updatedName+" description"),
			},
			{
				PreConfig: func() {
					meshID := testAccMagicMeshOutOfBandDeleteID
					if meshID == 0 {
						t.Fatal("no tracked magic mesh ID")
					}
					if err := testAccClients().V3.DeleteMagicMesh(meshID); err != nil && !gona.IsV3NotFound(err) {
						t.Fatalf("out of band magic mesh delete: %v", err)
					}
					testAccWaitMagicMeshGone(t, meshID)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_magic_mesh.test"),
			},
		},
	})
}

func TestAccNetactuateMagicMeshRouter_importPlanDetachAndOutOfBandDelete(t *testing.T) {
	name := testAccName("magic-mesh-router")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	configAttached := testAccMagicMeshRouterConfig(name, locationID, plan, true)
	configDetached := testAccMagicMeshRouterConfig(name, locationID, plan, false)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID", "NETACTUATE_ACC_PLAN")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMagicMeshRouterSuiteDestroy,
		Steps: []resource.TestStep{
			{
				Config: configAttached,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_magic_mesh", "netactuate_magic_mesh.test"),
					testAccTrackFromState("netactuate_router", "netactuate_router.test"),
					testAccRememberID("netactuate_magic_mesh.test", &testAccMagicMeshRouterMeshID),
					testAccRememberID("netactuate_router.test", &testAccMagicMeshRouterRouterID),
					testAccCheckRouterCanJoinMagicMeshFromAPI("netactuate_router.test", false),
					testAccCheckMagicMeshRouterFromAPI("netactuate_magic_mesh.test", "netactuate_router.test", true),
				),
			},
			{
				ResourceName:      "netactuate_magic_mesh_router.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   configAttached,
				PlanOnly: true,
			},
			{
				Config: configDetached,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMagicMeshRouterFromAPI("netactuate_magic_mesh.test", "netactuate_router.test", false),
					testAccCheckRouterCanJoinMagicMeshFromAPI("netactuate_router.test", true),
				),
			},
			{
				Config: configAttached,
				Check:  testAccCheckMagicMeshRouterFromAPI("netactuate_magic_mesh.test", "netactuate_router.test", true),
			},
			{
				PreConfig: func() {
					meshID := testAccMagicMeshRouterMeshID
					routerID := testAccMagicMeshRouterRouterID
					if meshID == 0 || routerID == 0 {
						t.Fatalf("missing tracked magic mesh router IDs: mesh=%d router=%d", meshID, routerID)
					}
					if err := testAccClients().V3.RemoveRouterFromMesh(meshID, routerID); err != nil && !gona.IsV3NotFound(err) {
						t.Fatalf("out of band magic mesh router detach: %v", err)
					}
					testAccWaitMagicMeshRouterMembership(t, meshID, routerID, false)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_magic_mesh_router.test"),
			},
		},
	})
}

func testAccCheckMagicMeshDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_magic_mesh" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetMagicMesh(id); err == nil {
			return fmt.Errorf("magic mesh still exists: %d", id)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return nil
}

func testAccCheckMagicMeshRouterSuiteDestroy(s *terraform.State) error {
	if err := testAccCheckMagicMeshDestroy(s); err != nil {
		return err
	}
	return testAccCheckRouterDestroy(s)
}

func testAccCheckMagicMeshFromAPI(address, expectedName, expectedDescription string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccWaitMagicMesh(id, expectedName, expectedDescription)
	}
}

// testAccCheckRouterCanJoinMagicMeshFromAPI asserts the flag in BOTH directions, because it is
// derived rather than stored. vapi3 computes it as:
//
//	canJoinMagicMesh = !!(hasDefaultVrf && !magicMesh_id)
//
// so a router that is already in a mesh reports false BY DEFINITION. The first version of this
// test asserted true immediately after attaching, which is not a condition the platform can ever
// satisfy. This test pins that platform behavior directly.
//
// Asserting both directions is stronger than the original: false while attached proves the API
// reflects membership, true after detaching proves it is released again.
func testAccCheckRouterCanJoinMagicMeshFromAPI(address string, want bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		config, err := testAccClients().V3.GetRouterConfig(routerID)
		if err != nil {
			return err
		}
		if config.Metadata.CanJoinMagicMesh != want {
			return fmt.Errorf("%s API canJoinMagicMesh = %t, want %t", address, config.Metadata.CanJoinMagicMesh, want)
		}
		return nil
	}
}

func testAccCheckMagicMeshRouterFromAPI(meshAddress, routerAddress string, expectedPresent bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		meshID, err := testAccID(s, meshAddress)
		if err != nil {
			return err
		}
		routerID, err := testAccID(s, routerAddress)
		if err != nil {
			return err
		}
		return testAccWaitMagicMeshRouterMembership(nil, meshID, routerID, expectedPresent)
	}
}

func testAccWaitMagicMesh(meshID int, expectedName, expectedDescription string) error {
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		mesh, err := testAccClients().V3.GetMagicMesh(meshID)
		if err != nil {
			lastErr = err
		} else if mesh.Name != expectedName {
			lastErr = fmt.Errorf("magic mesh API name = %q, want %q", mesh.Name, expectedName)
		} else if mesh.Description == nil || *mesh.Description != expectedDescription {
			got := "<nil>"
			if mesh.Description != nil {
				got = *mesh.Description
			}
			lastErr = fmt.Errorf("magic mesh API description = %q, want %q", got, expectedDescription)
		} else {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for magic mesh %d: %v", meshID, lastErr)
}

func testAccWaitMagicMeshGone(t *testing.T, meshID int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		_, err := testAccClients().V3.GetMagicMesh(meshID)
		if gona.IsV3NotFound(err) {
			return
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("magic mesh still exists")
		}
		time.Sleep(5 * time.Second)
	}
	t.Fatalf("timed out waiting for magic mesh %d to be deleted: %v", meshID, lastErr)
}

func testAccWaitMagicMeshRouterMembership(t *testing.T, meshID, routerID int, expectedPresent bool) error {
	if t != nil {
		t.Helper()
	}
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		routers, err := testAccClients().V3.ListMeshRouters(meshID)
		if err != nil {
			lastErr = err
		} else {
			found := false
			for _, router := range routers {
				if router.RouterID == routerID {
					found = true
					break
				}
			}
			if found == expectedPresent {
				return nil
			}
			lastErr = fmt.Errorf("magic mesh router membership present = %t, want %t", found, expectedPresent)
		}
		time.Sleep(5 * time.Second)
	}
	err := fmt.Errorf("timed out waiting for magic mesh %d router %d membership: %v", meshID, routerID, lastErr)
	if t != nil {
		t.Fatal(err)
	}
	return err
}

func testAccMagicMeshConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_magic_mesh" "test" {
  name        = %q
  description = %q
}
`, name, name+" description")
}

func testAccMagicMeshRouterConfig(name string, locationID int, plan string, attach bool) string {
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_magic_mesh" "test" {
  name        = %q
  description = %q
}

resource "netactuate_router" "test" {
  name        = %q
  description = %q
  location_id = %d
  plan        = %q
}
`, name, name+" mesh description", name, name+" router description", locationID, plan)
	if attach {
		config += `
resource "netactuate_magic_mesh_router" "test" {
  mesh_id   = netactuate_magic_mesh.test.mesh_id
  router_id = netactuate_router.test.router_id
}
`
	}
	return config
}
