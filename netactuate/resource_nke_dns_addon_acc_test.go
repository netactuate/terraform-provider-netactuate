//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateNKEDNSAddon_basic(t *testing.T) {
	clusterName := testAccName("nke-dns-addon")
	zone := testAccDNSZoneName(t)
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	config := testAccNKEDNSAddonConfig(clusterName, zone, locationID, version, plan, contractID)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_NKE_VERSION",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
				"NETACTUATE_ACC_DNS_DOMAIN",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNKEDNSAddonFixtureDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
					testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
					testAccCheckNKEDNSAddonZoneFromAPI("netactuate_nke_dns_addon.test", zone, "subzone"),
					testAccCheckNKEDNSZonesDataSourceFromAPI("data.netactuate_nke_dns_zones.test", "netactuate_nke_cluster.test"),
				),
			},
			{
				ResourceName:      "netactuate_nke_dns_addon.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_nke_dns_addon.test"),
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					if err := testAccDeleteNKEAddonByClusterName(clusterName, nkeDNSAddonType); err != nil {
						t.Fatalf("out of band NKE DNS addon delete: %v", err)
					}
					testAccWaitGoneFromNKEAPI(t, "NKE DNS addon", func() (bool, error) {
						exists, err := testAccNKEAddonExistsByClusterName(clusterName, nkeDNSAddonType)
						return !exists, err
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_nke_dns_addon.test"),
			},
		},
	})
}

func testAccCheckNKEDNSAddonFixtureDestroy(s *terraform.State) error {
	if err := testAccCheckNKEDNSAddonDestroy(s); err != nil {
		return err
	}
	if err := testAccCheckDNSZoneDestroy(s); err != nil {
		return err
	}
	return testAccCheckNKEClusterDestroy(s)
}

func testAccCheckNKEDNSAddonDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_nke_dns_addon" {
			continue
		}
		clusterID, addonType, err := parseNKEAddonStateID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetClusterAddon(clusterID, addonType); err == nil {
			return fmt.Errorf("NKE DNS addon still exists on cluster %d", clusterID)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return nil
}

func testAccDeleteNKEAddon(clusterID int, addonType string) error {
	if err := testAccClients().V3.DeleteClusterAddon(clusterID, addonType); err != nil && !gona.IsV3NotFound(err) {
		return err
	}
	return nil
}

func testAccNKEAddonExists(clusterID int, addonType string) (bool, error) {
	if _, err := testAccClients().V3.GetClusterAddon(clusterID, addonType); err != nil {
		if gona.IsV3NotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func testAccDeleteNKEAddonByClusterName(name, addonType string) error {
	clusterID, found, err := testAccNKEClusterIDByName(name)
	if err != nil || !found {
		return err
	}
	return testAccDeleteNKEAddon(clusterID, addonType)
}

func testAccNKEAddonExistsByClusterName(name, addonType string) (bool, error) {
	clusterID, found, err := testAccNKEClusterIDByName(name)
	if err != nil || !found {
		return false, err
	}
	return testAccNKEAddonExists(clusterID, addonType)
}

func testAccNKEClusterIDByName(name string) (int, bool, error) {
	for id := range testAccCreated["netactuate_nke_cluster"] {
		cluster, err := testAccClients().V3.GetNKECluster(id)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return 0, false, err
		}
		if cluster.Name == name {
			return id, true, nil
		}
	}
	return 0, false, nil
}

func testAccCheckNKEDNSZonesDataSourceFromAPI(address, clusterAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		expectedClusterID, err := testAccID(s, clusterAddress)
		if err != nil {
			return err
		}
		zones, err := testAccClients().V3.ListClusterDNSZones(expectedClusterID)
		if err != nil {
			return err
		}
		if len(zones) == 0 {
			return fmt.Errorf("NKE DNS zones API returned no zones for cluster %d", expectedClusterID)
		}
		if got := rs.Primary.Attributes["zones.#"]; got != strconv.Itoa(len(zones)) {
			return fmt.Errorf("%s state zone count = %q, want API count %d", address, got, len(zones))
		}
		for i, zone := range zones {
			prefix := fmt.Sprintf("zones.%d", i)
			if got := rs.Primary.Attributes[prefix+".dns_zone_id"]; got != strconv.Itoa(zone.DNSZoneID) {
				return fmt.Errorf("%s state %s.dns_zone_id = %q, want API id %d", address, prefix, got, zone.DNSZoneID)
			}
			if got := rs.Primary.Attributes[prefix+".cluster_id"]; got != strconv.Itoa(zone.ClusterID) {
				return fmt.Errorf("%s state %s.cluster_id = %q, want API cluster_id %d", address, prefix, got, zone.ClusterID)
			}
			if got := rs.Primary.Attributes[prefix+".zone"]; got != zone.Zone {
				return fmt.Errorf("%s state %s.zone = %q, want API zone %q", address, prefix, got, zone.Zone)
			}
			if got := rs.Primary.Attributes[prefix+".mode"]; got != zone.Mode {
				return fmt.Errorf("%s state %s.mode = %q, want API mode %q", address, prefix, got, zone.Mode)
			}
		}
		return nil
	}
}

func testAccNKEDNSAddonConfig(clusterName, zone string, locationID int, version, plan string, contractID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_nke_cluster" "test" {
  name          = %q
  version       = %q
  location_id   = %d
  plan          = %q
  contract_id   = %d
  minimum_nodes = 1
  maximum_nodes = 1
}

resource "netactuate_dns_zone" "test" {
  name = %q
  type = "NATIVE"
}

resource "netactuate_nke_dns_addon" "test" {
  cluster_id = netactuate_nke_cluster.test.cluster_id
  zone       = netactuate_dns_zone.test.name
  mode       = "subzone"
}

data "netactuate_nke_dns_zones" "test" {
  cluster_id = netactuate_nke_dns_addon.test.cluster_id

  depends_on = [netactuate_nke_dns_addon.test]
}
`, clusterName, version, locationID, plan, contractID, zone)
}
