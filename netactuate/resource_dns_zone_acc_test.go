//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateDNSZone_importDataSourcePlanAndOutOfBandDelete(t *testing.T) {
	zoneName := testAccDNSZoneName(t)
	config := testAccDNSZoneConfig(zoneName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
					testAccCheckDNSZoneFromAPI("netactuate_dns_zone.test", zoneName, "NATIVE"),
					testAccCheckDNSZoneListedFromAPI("netactuate_dns_zone.test", "NATIVE"),
					testAccCheckDNSZoneTTLFromAPI("netactuate_dns_zone.test", "NATIVE"),
					testAccCheckDNSZoneDataSourceFromAPI("data.netactuate_dns_zone.test", zoneName, "NATIVE"),
				),
			},
			{
				// No ImportStatePersist here. The framework imports into a FRESH state for
				// this step, which is what proves import fidelity. Persisting it instead
				// imports into the state step 1 already populated, and terraform refuses
				// with "Resource already managed by Terraform". The no-diff assertion is
				// the PlanOnly step below, not this one.
				ResourceName:      "netactuate_dns_zone.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_dns_zone"] {
						_ = clients.V2.DeleteZone(id)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_dns_zone.test"),
			},
		},
	})
}

func TestAccNetactuateDNSZone_nkeDNSAddonInteraction(t *testing.T) {
	clusterName := testAccName("dns-zone-nke-addon")
	zoneName := testAccDNSZoneName(t)
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	config := testAccDNSZoneNKEAddonConfig(clusterName, zoneName, locationID, version, plan, contractID)

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
		CheckDestroy:      testAccCheckDNSZoneAndNKEDNSAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
					testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
					testAccCheckDNSZoneFromAPI("netactuate_dns_zone.test", zoneName, "NATIVE"),
					testAccCheckNKEDNSAddonZoneFromAPI("netactuate_nke_dns_addon.test", zoneName, "subzone"),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckDNSZoneAndNKEDNSAddonDestroy(s *terraform.State) error {
	if err := testAccCheckDNSZoneDestroy(s); err != nil {
		return err
	}
	if err := testAccCheckNKEDNSAddonDestroy(s); err != nil {
		return err
	}
	return testAccCheckNKEClusterDestroy(s)
}

func testAccCheckDNSZoneDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_dns_zone" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V2.GetZone(id); err == nil {
			return fmt.Errorf("DNS zone still exists: %d", id)
		} else if !gona.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func testAccCheckDNSZoneFromAPI(address, expectedName, expectedType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		zone, err := testAccClients().V2.GetZone(id)
		if err != nil {
			return err
		}
		if zone.Name != expectedName {
			return fmt.Errorf("%s API name = %q, want %q", address, zone.Name, expectedName)
		}
		if zone.Type != expectedType {
			return fmt.Errorf("%s API type = %q, want %q", address, zone.Type, expectedType)
		}
		return nil
	}
}

func testAccCheckDNSZoneListedFromAPI(address, expectedType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		zones, err := testAccClients().V2.ListZones(expectedType)
		if err != nil {
			return err
		}
		for _, zone := range zones {
			if zone.ID != id {
				continue
			}
			if zone.Type != expectedType {
				return fmt.Errorf("%s API list type = %q, want %q", address, zone.Type, expectedType)
			}
			return nil
		}
		return fmt.Errorf("%s not found in API %s zone list", address, expectedType)
	}
}

func testAccCheckDNSZoneTTLFromAPI(address, zoneType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		zone, err := testAccClients().V2.GetZone(id)
		if err != nil {
			return err
		}
		if zone.TTL.Int() == 0 {
			return fmt.Errorf("%s API detail TTL = %q, want parseable nonzero TTL", address, string(zone.TTL))
		}
		zones, err := testAccClients().V2.ListZones(zoneType)
		if err != nil {
			return err
		}
		for _, listed := range zones {
			if listed.ID != id {
				continue
			}
			if listed.TTL.Int() == 0 {
				return fmt.Errorf("%s API list TTL = %q, want parseable nonzero TTL", address, string(listed.TTL))
			}
			return nil
		}
		return fmt.Errorf("%s not found in API %s zone list", address, zoneType)
	}
}

func testAccCheckDNSZoneDataSourceFromAPI(address, expectedName, expectedType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		zone, err := testAccClients().V2.GetZone(id)
		if err != nil {
			return err
		}
		if zone.Name != expectedName {
			return fmt.Errorf("%s API name = %q, want %q", address, zone.Name, expectedName)
		}
		if zone.Type != expectedType {
			return fmt.Errorf("%s API type = %q, want %q", address, zone.Type, expectedType)
		}
		return nil
	}
}

func testAccCheckNKEDNSAddonZoneFromAPI(address, expectedZone, expectedMode string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		clusterID, addonType, err := parseNKEAddonStateID(rs.Primary.ID)
		if err != nil {
			return err
		}
		addon, err := testAccClients().V3.GetClusterAddon(clusterID, addonType)
		if err != nil {
			return err
		}
		for _, zone := range addon.Config.Zones {
			if zone.Zone != expectedZone {
				continue
			}
			if zone.Mode != expectedMode {
				return fmt.Errorf("%s API mode = %q, want %q", address, zone.Mode, expectedMode)
			}
			if zone.DNSZoneID == 0 {
				return fmt.Errorf("%s API dnsZoneId = 0, want nonzero", address)
			}
			return nil
		}
		return fmt.Errorf("%s API response did not include zone %q", address, expectedZone)
	}
}

func testAccDNSZoneConfig(zoneName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dns_zone" "test" {
  name = %q
  type = "NATIVE"
}

data "netactuate_dns_zone" "test" {
  name = netactuate_dns_zone.test.name
}
`, zoneName)
}

func testAccDNSZoneNKEAddonConfig(clusterName, zoneName string, locationID int, version, plan string, contractID int) string {
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
`, clusterName, version, locationID, plan, contractID, zoneName)
}

func testAccDNSZoneName(t *testing.T) string {
	t.Helper()
	domain := testAccEnvString(t, "NETACTUATE_ACC_DNS_DOMAIN")
	domain = strings.TrimSuffix(domain, ".")
	// No trailing dot: the API's ValidFQDNRule regex ends at [a-zA-Z\d]{1,63}$,
	// so a canonical trailing-dot FQDN is rejected with a 422.
	return fmt.Sprintf("%s.%s", testAccName("dns"), domain)
}
