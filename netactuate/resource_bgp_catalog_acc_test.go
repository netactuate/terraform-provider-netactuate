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

func TestAccNetactuateBGPCatalogDataSources_readFromAPI(t *testing.T) {
	groupID := testAccEnvInt(t, "NETACTUATE_ACC_BGP_GROUP_ID")
	asnID := testAccEnvInt(t, "NETACTUATE_ACC_BGP_ASN_ID")
	prefixID := testAccEnvInt(t, "NETACTUATE_ACC_BGP_PREFIX_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, "NETACTUATE_ACC_BGP_GROUP_ID", "NETACTUATE_ACC_BGP_ASN_ID", "NETACTUATE_ACC_BGP_PREFIX_ID")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      func(s *terraform.State) error { return nil },
		Steps: []resource.TestStep{{
			Config: testAccBGPCatalogDataSourcesConfig(groupID, asnID, prefixID),
			Check: resource.ComposeTestCheckFunc(
				testAccCheckBGPGroupsDataSourceFromAPI("data.netactuate_bgp_groups.test", groupID),
				testAccCheckBGPASNsDataSourceFromAPI("data.netactuate_bgp_asns.test"),
				testAccCheckBGPASNDataSourceFromAPI("data.netactuate_bgp_asn.test", asnID),
				testAccCheckBGPPrefixesDataSourceFromAPI("data.netactuate_bgp_prefixes.test"),
				testAccCheckBGPPrefixDataSourceFromAPI("data.netactuate_bgp_prefix.test", prefixID),
				testAccCheckBGPSummaryDataSourceFromAPI("data.netactuate_bgp_summary.test"),
				testAccCheckBGPDashboardDataSourceFromAPI("data.netactuate_bgp_dashboard.test"),
			),
		}},
	})
}

func TestAccNetactuateBGPGroup_createImportAndStateOnlyDelete(t *testing.T) {
	name := testAccName("bgp-group")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      func(s *terraform.State) error { return testAccCheckStateOnlyDestroy(s, "netactuate_bgp_group") },
		Steps: []resource.TestStep{
			{Config: testAccBGPGroupConfig(name, name+" description", "anycast"), Check: resource.ComposeTestCheckFunc(testAccTrackFromState("netactuate_bgp_group", "netactuate_bgp_group.test"), testAccCheckBGPGroupFromAPI("netactuate_bgp_group.test", name, name+" description", "anycast"))},
			{ResourceName: "netactuate_bgp_group.test", ImportState: true, ImportStateVerify: true},
		},
	})
}

func TestAccNetactuateBGPGroupFirewallSet_createImportAndDelete(t *testing.T) {
	groupID := testAccEnvInt(t, "NETACTUATE_ACC_BGP_GROUP_ID")
	firewallSetID := testAccEnvInt(t, "NETACTUATE_ACC_FIREWALL_SET_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_BGP_GROUP_ID", "NETACTUATE_ACC_FIREWALL_SET_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			// The API exposes bind and unbind, but no read endpoint for a single binding.
			return nil
		},
		Steps: []resource.TestStep{
			{Config: testAccBGPGroupFirewallSetConfig(groupID, firewallSetID, 0, 10), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("netactuate_bgp_group_firewall_set.test", "bgp_group_id", strconv.Itoa(groupID)), resource.TestCheckResourceAttr("netactuate_bgp_group_firewall_set.test", "firewall_set_id", strconv.Itoa(firewallSetID)), resource.TestCheckResourceAttr("netactuate_bgp_group_firewall_set.test", "interface_number", "0"), resource.TestCheckResourceAttr("netactuate_bgp_group_firewall_set.test", "set_priority", "10"), testAccCheckAttributePresent("netactuate_bgp_group_firewall_set.test", "binding_id"))},
			{ResourceName: "netactuate_bgp_group_firewall_set.test", ImportState: true, ImportStateVerify: true},
		},
	})
}

func TestAccNetactuateBGPPrefixPurchase_createImportAndStateOnlyDelete(t *testing.T) {
	name := testAccName("bgp-prefix")
	groupID := testAccEnvInt(t, "NETACTUATE_ACC_BGP_GROUP_ID")
	agreementID := testAccEnvInt(t, "NETACTUATE_ACC_BGP_AGREEMENT_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_BGP_GROUP_ID", "NETACTUATE_ACC_BGP_AGREEMENT_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckStateOnlyDestroy(s, "netactuate_bgp_prefix_purchase")
		},
		Steps: []resource.TestStep{
			{Config: testAccBGPPrefixPurchaseConfig(name, groupID, agreementID), Check: resource.ComposeTestCheckFunc(testAccTrackFromState("netactuate_bgp_prefix_purchase", "netactuate_bgp_prefix_purchase.test"), testAccCheckBGPPrefixPurchaseFromAPI("netactuate_bgp_prefix_purchase.test", name, groupID, agreementID))},
			{ResourceName: "netactuate_bgp_prefix_purchase.test", ImportState: true, ImportStateVerify: true},
		},
	})
}

func testAccCheckBGPGroupFromAPI(address, expectedName, expectedDescription, expectedGroupType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		group, err := testAccClients().V2.GetBGPGroup(id)
		if err != nil {
			return err
		}
		if group.Name != expectedName || group.Description != expectedDescription || group.GroupType != expectedGroupType {
			return fmt.Errorf("%s API group = %#v", address, group)
		}
		return nil
	}
}

func testAccCheckBGPPrefixPurchaseFromAPI(address, expectedName string, expectedGroupID, expectedAgreementID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		prefix, err := testAccClients().V2.GetBGPPrefix(id)
		if err != nil {
			return err
		}
		if prefix.Name != expectedName || prefix.GroupID != expectedGroupID {
			return fmt.Errorf("%s API prefix = %#v", address, prefix)
		}
		if prefix.AgreementID != 0 && prefix.AgreementID != expectedAgreementID {
			return fmt.Errorf("%s API agreement_id = %d, want %d", address, prefix.AgreementID, expectedAgreementID)
		}
		if prefix.Prefix == "" {
			return fmt.Errorf("%s API prefix is empty", address)
		}
		return nil
	}
}

func testAccCheckBGPASNDataSourceFromAPI(address string, asnID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		asn, err := testAccClients().V2.GetBGPASN(asnID)
		if err != nil {
			return err
		}
		return testAccCheckStateListHasInt(&terraform.State{Modules: s.Modules}, address, "", "bgp_asn_id", asn.ID)
	}
}

func testAccCheckBGPASNsDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		asns, err := testAccClients().V2.ListBGPASNs("")
		if err != nil {
			return err
		}
		return testAccCheckStateListCount(s, address, "asns", len(asns))
	}
}

func testAccCheckBGPGroupsDataSourceFromAPI(address string, groupID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		groups, err := testAccClients().V2.ListBGPGroups("")
		if err != nil {
			return err
		}
		if err := testAccCheckStateListCount(s, address, "groups", len(groups)); err != nil {
			return err
		}
		return testAccCheckStateListHasInt(s, address, "groups", "bgp_group_id", groupID)
	}
}

func testAccCheckBGPPrefixDataSourceFromAPI(address string, prefixID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		prefix, err := testAccClients().V2.GetBGPPrefix(prefixID)
		if err != nil {
			return err
		}
		return testAccCheckStateAttrEquals(s, address, "bgp_prefix_id", strconv.Itoa(prefix.ID))
	}
}

func testAccCheckBGPPrefixesDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		prefixes, err := testAccClients().V2.ListBGPPrefixes("")
		if err != nil {
			return err
		}
		return testAccCheckStateListCount(s, address, "prefixes", len(prefixes))
	}
}

func testAccCheckBGPSummaryDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		summary, err := testAccClients().V2.GetBGPSummary()
		if err != nil {
			return err
		}
		want, err := rawMapString(summary)
		if err != nil {
			return err
		}
		return testAccCheckStateAttrEquals(s, address, "summary_json", want)
	}
}

func testAccCheckBGPDashboardDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		dashboard, err := testAccClients().V2.GetBGPDashboard(gona.BGPDashboardOptions{})
		if err != nil {
			return err
		}
		want, err := rawMapString(dashboard)
		if err != nil {
			return err
		}
		return testAccCheckStateAttrEquals(s, address, "dashboard_json", want)
	}
}

func testAccBGPCatalogDataSourcesConfig(groupID, asnID, prefixID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
data "netactuate_bgp_groups" "test" {}
data "netactuate_bgp_asns" "test" {}
data "netactuate_bgp_asn" "test" {
  bgp_asn_id = %d
}
data "netactuate_bgp_prefixes" "test" {}
data "netactuate_bgp_prefix" "test" {
  bgp_prefix_id = %d
}
data "netactuate_bgp_summary" "test" {}
data "netactuate_bgp_dashboard" "test" {}
locals { bgp_group_id = %d }
`, asnID, prefixID, groupID)
}
func testAccBGPGroupConfig(name, description, groupType string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_bgp_group" "test" {
  name = %q
  description = %q
  group_type = %q
}
`, name, description, groupType)
}
func testAccBGPGroupFirewallSetConfig(groupID, firewallSetID, interfaceNumber, setPriority int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_bgp_group_firewall_set" "test" {
  bgp_group_id = %d
  firewall_set_id = %d
  interface_number = %d
  set_priority = %d
}
`, groupID, firewallSetID, interfaceNumber, setPriority)
}
func testAccBGPPrefixPurchaseConfig(name string, groupID, agreementID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_bgp_prefix_purchase" "test" {
  name = %q
  group_id = %d
  agreement_id = %d
}
`, name, groupID, agreementID)
}

func testAccCheckStateListCount(s *terraform.State, address, listAttr string, want int) error {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return fmt.Errorf("not found: %s", address)
	}
	got, err := strconv.Atoi(rs.Primary.Attributes[listAttr+".#"])
	if err != nil {
		return fmt.Errorf("%s %s count is invalid: %w", address, listAttr, err)
	}
	if got != want {
		return fmt.Errorf("%s %s count = %d, want API count %d", address, listAttr, got, want)
	}
	return nil
}

func testAccCheckStateListHasInt(s *terraform.State, address, listAttr, field string, want int) error {
	return testAccCheckStateListHasString(s, address, listAttr, field, strconv.Itoa(want))
}

func testAccCheckStateListHasString(s *terraform.State, address, listAttr, field, want string) error {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return fmt.Errorf("not found: %s", address)
	}
	if listAttr == "" {
		if got := rs.Primary.Attributes[field]; got == want {
			return nil
		}
		return fmt.Errorf("%s %s = %q, want %q", address, field, rs.Primary.Attributes[field], want)
	}
	count, err := strconv.Atoi(rs.Primary.Attributes[listAttr+".#"])
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if got := rs.Primary.Attributes[fmt.Sprintf("%s.%d.%s", listAttr, i, field)]; got == want {
			return nil
		}
	}
	return fmt.Errorf("%s %s has no item with %s = %q", address, listAttr, field, want)
}

func testAccCheckStateAttrEquals(s *terraform.State, address, attr, want string) error {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return fmt.Errorf("not found: %s", address)
	}
	got, ok := rs.Primary.Attributes[attr]
	if !ok {
		return fmt.Errorf("%s state missing %s", address, attr)
	}
	if got != want {
		return fmt.Errorf("%s state %s = %q, want API value %q", address, attr, got, want)
	}
	return nil
}

func testAccCheckStateOnlyDestroy(s *terraform.State, resourceType string) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == resourceType && rs.Primary != nil && rs.Primary.ID != "" {
			return fmt.Errorf("%s still in state with ID %s", resourceType, rs.Primary.ID)
		}
	}
	return nil
}
