//go:build acctest

package netactuate

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateDNSZoneDataSource_listsNativeAndSlaveZonesFromAPI(t *testing.T) {
	nativeName := testAccDNSZoneName(t)
	slaveName := testAccDNSZoneName(t)
	slaveMasterIP := testAccEnvString(t, "NETACTUATE_ACC_DNS_SLAVE_MASTER_IP")
	config := testAccDNSZoneDataSourceNativeAndSlaveConfig(nativeName, slaveName, slaveMasterIP)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN", "NETACTUATE_ACC_DNS_SLAVE_MASTER_IP") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.native"),
					testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.slave"),
					testAccCheckDNSZoneFromAPI("netactuate_dns_zone.native", nativeName, "NATIVE"),
					testAccCheckDNSZoneFromAPI("netactuate_dns_zone.slave", slaveName, "SLAVE"),
					testAccCheckDNSZoneListedFromAPI("netactuate_dns_zone.native", "NATIVE"),
					testAccCheckDNSZoneListedFromAPI("netactuate_dns_zone.slave", "SLAVE"),
					testAccCheckDNSZoneListedMasterFromAPI("netactuate_dns_zone.slave", slaveMasterIP),
					testAccCheckDNSZoneDataSourceFromAPI("data.netactuate_dns_zone.native", nativeName, "NATIVE"),
					testAccCheckDNSZoneDataSourceFromAPI("data.netactuate_dns_zone.slave", slaveName, "SLAVE"),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckDNSZoneListedMasterFromAPI(address, expectedMaster string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		zones, err := testAccClients().V2.ListZones("SLAVE")
		if err != nil {
			return err
		}
		for _, zone := range zones {
			if zone.ID != id {
				continue
			}
			if zone.Master != expectedMaster {
				return fmt.Errorf("%s API list master = %q, want %q", address, zone.Master, expectedMaster)
			}
			return nil
		}
		return fmt.Errorf("%s not found in API SLAVE zone list", address)
	}
}

func testAccDNSZoneDataSourceNativeAndSlaveConfig(nativeName, slaveName, slaveMasterIP string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dns_zone" "native" {
  name = %q
  type = "NATIVE"
}

resource "netactuate_dns_zone" "slave" {
  name = %q
  type = "SLAVE"
  ip   = %q
}

data "netactuate_dns_zone" "native" {
  name = netactuate_dns_zone.native.name
}

data "netactuate_dns_zone" "slave" {
  name = netactuate_dns_zone.slave.name
}
`, nativeName, slaveName, slaveMasterIP)
}
