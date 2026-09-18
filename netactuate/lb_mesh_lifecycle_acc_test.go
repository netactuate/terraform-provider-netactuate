//go:build acctest

package netactuate

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateNetworkLoadbalancerGroupLifecycle(t *testing.T) {
	name := testAccName("nlbg")
	updatedName := name + "-updated"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	createConfig := testAccNetworkLBGroupConfig(name, locationID, false, false)
	modifiedConfig := testAccNetworkLBGroupConfig(updatedName, locationID, true, false)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNetworkLBGroupDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_network_loadbalancer_group.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccTrackNetworkLBGroupFromState("netactuate_network_loadbalancer_group.test"),
			testAccCheckNetworkLBGroupFromAPI("netactuate_network_loadbalancer_group.test", testAccNetworkLBGroupWant{
				Name:           name,
				Description:    name + " description",
				Algorithm:      "round-robin",
				BackendNames:   []string{name + "-backend-a", name + "-backend-b"},
				RuleProtocols:  []string{"TCP"},
				HealthCheckTCP: false,
			}),
		), resource.ComposeTestCheckFunc(
			testAccTrackNetworkLBGroupFromState("netactuate_network_loadbalancer_group.test"),
			testAccCheckNetworkLBGroupFromAPI("netactuate_network_loadbalancer_group.test", testAccNetworkLBGroupWant{
				Name:           updatedName,
				Description:    updatedName + " description",
				Algorithm:      "least-connections",
				BackendNames:   []string{updatedName + "-backend-b", updatedName + "-backend-c"},
				RuleProtocols:  []string{"TCP", "UDP"},
				HealthCheckTCP: true,
			}),
		), testAccLifecycleOptions{ImportStateIdFunc: testAccLBMeshLifecycleImportID("netactuate_network_loadbalancer_group.test")}),
	})
}

func TestAccNetactuateNetworkLoadbalancerGroupOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("nlbg-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNetworkLBGroupDestroy,
		Steps: []resource.TestStep{{
			Config: testAccNetworkLBGroupConfig(name, locationID, false, false),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
				testAccTrackNetworkLBGroupFromState("netactuate_network_loadbalancer_group.test"),
				testAccCheckNetworkLBGroupFromAPI("netactuate_network_loadbalancer_group.test", testAccNetworkLBGroupWant{
					Name:           name,
					Description:    name + " description",
					Algorithm:      "round-robin",
					BackendNames:   []string{name + "-backend-a", name + "-backend-b"},
					RuleProtocols:  []string{"TCP"},
					HealthCheckTCP: false,
				}),
				testAccDeleteNetworkLBGroupOutOfBand("netactuate_network_loadbalancer_group.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateHTTPLoadbalancerGroupLifecycle(t *testing.T) {
	name := testAccName("htlbg")
	updatedName := name + "-updated"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	createConfig := testAccHTTPLBGroupConfig(name, locationID, false, false)
	modifiedConfig := testAccHTTPLBGroupConfig(updatedName, locationID, true, false)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPLBGroupDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_http_loadbalancer_group.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccTrackHTTPLBGroupFromState("netactuate_http_loadbalancer_group.test"),
			testAccCheckHTTPLBGroupFromAPI("netactuate_http_loadbalancer_group.test", testAccHTTPLBGroupWant{
				Name:                  name,
				Description:           name + " description",
				Algorithm:             "round-robin",
				StickySessionsEnabled: false,
				SSLToBackendEnabled:   false,
				InternalPort:          8080,
				MatchPorts:            "80",
				RuleDomains:           []string{name + ".example.invalid"},
				BackendNames:          []string{name + "-backend-a", name + "-backend-b"},
			}),
		), resource.ComposeTestCheckFunc(
			testAccTrackHTTPLBGroupFromState("netactuate_http_loadbalancer_group.test"),
			testAccCheckHTTPLBGroupFromAPI("netactuate_http_loadbalancer_group.test", testAccHTTPLBGroupWant{
				Name:                  updatedName,
				Description:           updatedName + " description",
				Algorithm:             "least-connections",
				StickySessionsEnabled: true,
				SSLToBackendEnabled:   true,
				InternalPort:          8443,
				MatchPorts:            "80+443",
				RuleDomains:           []string{updatedName + ".example.invalid", "alt-" + updatedName + ".example.invalid"},
				BackendNames:          []string{updatedName + "-backend-b", updatedName + "-backend-c"},
			}),
		), testAccLifecycleOptions{ImportStateIdFunc: testAccLBMeshLifecycleImportID("netactuate_http_loadbalancer_group.test")}),
	})
}

func TestAccNetactuateHTTPLoadbalancerGroupOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("htlbg-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPLBGroupDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHTTPLBGroupConfig(name, locationID, false, false),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
				testAccTrackHTTPLBGroupFromState("netactuate_http_loadbalancer_group.test"),
				testAccCheckHTTPLBGroupFromAPI("netactuate_http_loadbalancer_group.test", testAccHTTPLBGroupWant{
					Name:                  name,
					Description:           name + " description",
					Algorithm:             "round-robin",
					StickySessionsEnabled: false,
					SSLToBackendEnabled:   false,
					InternalPort:          8080,
					MatchPorts:            "80",
					RuleDomains:           []string{name + ".example.invalid"},
					BackendNames:          []string{name + "-backend-a", name + "-backend-b"},
				}),
				testAccDeleteHTTPLBGroupOutOfBand("netactuate_http_loadbalancer_group.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

// netactuate_cloud_floating_ipv4 has no update path: ptr_domain and vlan_id are ForceNew,
// and the resource has no UpdateContext. Lifecycle coverage would be a create/import test
// with no meaningful modify step, so this file covers out-of-band delete only.
func TestAccNetactuateCloudFloatingIPv4OutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("floating-ip-oob")
	vlanID := testAccEnvInt(t, "NETACTUATE_ACC_CUSTOMER_VLAN_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_CUSTOMER_VLAN_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCloudFloatingIPv4Destroy,
		Steps: []resource.TestStep{{
			Config: testAccCloudFloatingIPv4ResourceOnlyConfig(name, vlanID),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_cloud_floating_ipv4", "netactuate_cloud_floating_ipv4.test"),
				testAccCheckCloudFloatingIPv4API("netactuate_cloud_floating_ipv4.test", name+".example.invalid"),
				testAccDeleteCloudFloatingIPv4OutOfBand("netactuate_cloud_floating_ipv4.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateMagicMeshLifecycle(t *testing.T) {
	name := testAccName("magic-mesh-lifecycle")
	updatedName := name + "-updated"
	createConfig := testAccMagicMeshConfig(name)
	modifiedConfig := testAccMagicMeshConfig(updatedName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMagicMeshDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_magic_mesh.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_magic_mesh", "netactuate_magic_mesh.test"),
			testAccCheckMagicMeshFromAPI("netactuate_magic_mesh.test", name, name+" description"),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_magic_mesh", "netactuate_magic_mesh.test"),
			testAccCheckMagicMeshFromAPI("netactuate_magic_mesh.test", updatedName, updatedName+" description"),
		), testAccLifecycleOptions{ImportStateIdFunc: testAccLBMeshLifecycleImportID("netactuate_magic_mesh.test")}),
	})
}

func TestAccNetactuateMagicMeshOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("magic-mesh-oob")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMagicMeshDestroy,
		Steps: []resource.TestStep{{
			Config: testAccMagicMeshConfig(name),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_magic_mesh", "netactuate_magic_mesh.test"),
				testAccCheckMagicMeshFromAPI("netactuate_magic_mesh.test", name, name+" description"),
				testAccDeleteMagicMeshOutOfBand("netactuate_magic_mesh.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

// netactuate_magic_mesh_router has no mutable fields: mesh_id and router_id are ForceNew,
// and the resource has no UpdateContext. Lifecycle coverage would have no honest modify step,
// so this file covers out-of-band delete only.
func TestAccNetactuateMagicMeshRouterOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("magic-mesh-router-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID", "NETACTUATE_ACC_PLAN")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckMagicMeshRouterSuiteDestroy,
		Steps: []resource.TestStep{{
			Config: testAccMagicMeshRouterConfig(name, locationID, plan, true),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_magic_mesh", "netactuate_magic_mesh.test"),
				testAccTrackFromState("netactuate_router", "netactuate_router.test"),
				testAccCheckMagicMeshRouterFromAPI("netactuate_magic_mesh.test", "netactuate_router.test", true),
				testAccDeleteMagicMeshRouterOutOfBand("netactuate_magic_mesh_router.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func testAccLBMeshLifecycleImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return testAccStateID(s, address)
	}
}

func testAccCloudFloatingIPv4ResourceOnlyConfig(name string, vlanID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_cloud_floating_ipv4" "test" {
  ptr_domain = %q
  vlan_id    = %d
}
`, name+".example.invalid", vlanID)
}

func testAccDeleteNetworkLBGroupOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		lbID, groupID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		if err := testAccClients().V3.DeleteNLBGroup(lbID, groupID); err != nil && !gona.IsV3NotFound(err) {
			return err
		}
		return testAccWaitNetworkLBGroupGone(lbID, groupID)
	}
}

func testAccDeleteHTTPLBGroupOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		lbID, groupID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		if err := testAccClients().V3.DeleteHTTPLBGroup(lbID, groupID); err != nil && !gona.IsV3NotFound(err) {
			return err
		}
		return testAccWaitHTTPLBGroupGone(lbID, groupID)
	}
}

func testAccDeleteCloudFloatingIPv4OutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		if err := testAccClients().V3.DeleteCloudFloatingIPv4(id); err != nil && !gona.IsNotFound(err) {
			return err
		}
		return testAccWaitCloudFloatingIPv4Gone(id)
	}
}

func testAccDeleteMagicMeshOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		if err := testAccClients().V3.DeleteMagicMesh(id); err != nil && !gona.IsV3NotFound(err) {
			return err
		}
		return testAccWaitMagicMeshGoneCheck(id)
	}
}

func testAccDeleteMagicMeshRouterOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		meshID, routerID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		if err := testAccClients().V3.RemoveRouterFromMesh(meshID, routerID); err != nil && !gona.IsV3NotFound(err) {
			return err
		}
		return testAccWaitMagicMeshRouterMembership(nil, meshID, routerID, false)
	}
}

func testAccWaitNetworkLBGroupGone(lbID, groupID int) error {
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		_, err := testAccClients().V3.GetNLBGroup(lbID, groupID)
		if gona.IsV3NotFound(err) {
			return nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("network LB group still exists")
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for network LB group %d/%d to be deleted: %v", lbID, groupID, lastErr)
}

func testAccWaitHTTPLBGroupGone(lbID, groupID int) error {
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		_, err := testAccClients().V3.GetHTTPLBGroup(lbID, groupID)
		if gona.IsV3NotFound(err) {
			return nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("HTTP LB group still exists")
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for HTTP LB group %d/%d to be deleted: %v", lbID, groupID, lastErr)
}

func testAccWaitCloudFloatingIPv4Gone(id int) error {
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		floatingIPs, err := testAccClients().V3.ListCloudFloatingIPv4()
		if err != nil {
			lastErr = err
		} else {
			found := false
			for _, floatingIP := range floatingIPs {
				if floatingIP.FloatingIPv4ID == id {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
			lastErr = fmt.Errorf("cloud floating IPv4 still exists")
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for cloud floating IPv4 %d to be deleted: %v", id, lastErr)
}

func testAccWaitMagicMeshGoneCheck(id int) error {
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		_, err := testAccClients().V3.GetMagicMesh(id)
		if gona.IsV3NotFound(err) {
			return nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("magic mesh still exists")
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for magic mesh %d to be deleted: %v", id, lastErr)
}
