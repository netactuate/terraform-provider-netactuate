//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateBGPGroupLifecycle(t *testing.T) {
	t.Skip("skipped: a BGP group has no delete endpoint, so running this lifecycle test would leave a group on the account")
	name := testAccName("bgpgrp")
	config := testAccBGPGroupConfig(name, name+" description", "anycast")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      func(s *terraform.State) error { return testAccCheckStateOnlyDestroy(s, "netactuate_bgp_group") },
		Steps: testAccLifecycleNoModifySteps(config, "netactuate_bgp_group.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_bgp_group", "netactuate_bgp_group.test"),
			testAccCheckBGPGroupFromAPI("netactuate_bgp_group.test", name, name+" description", "anycast"),
		), testAccLifecycleOptions{}),
	})
}

func TestAccNetactuateBGPGroupFirewallSetLifecycle(t *testing.T) {
	groupID := testAccEnvInt(t, "NETACTUATE_ACC_BGP_GROUP_ID")
	firewallSetID := testAccEnvInt(t, "NETACTUATE_ACC_FIREWALL_SET_ID")
	config := testAccBGPGroupFirewallSetConfig(groupID, firewallSetID, 0, 0)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_BGP_GROUP_ID", "NETACTUATE_ACC_FIREWALL_SET_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		Steps: testAccLifecycleNoModifySteps(config, "netactuate_bgp_group_firewall_set.test", testAccCheckBGPGroupFirewallSetState(groupID, firewallSetID, 0, 0), testAccLifecycleOptions{
			ImportStateIdFunc:       testAccBGPFirewallLifecycleImportID("netactuate_bgp_group_firewall_set.test"),
			ImportStateVerifyIgnore: []string{"binding_id"},
		}),
	})
}

func TestAccNetactuateBGPGroupFirewallSetOutOfBandDeleteIdempotent(t *testing.T) {
	groupID := testAccEnvInt(t, "NETACTUATE_ACC_BGP_GROUP_ID")
	firewallSetID := testAccEnvInt(t, "NETACTUATE_ACC_FIREWALL_SET_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_BGP_GROUP_ID", "NETACTUATE_ACC_FIREWALL_SET_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return nil
		},
		Steps: []resource.TestStep{{
			Config: testAccBGPGroupFirewallSetResourceOnlyConfig(groupID, firewallSetID, 0, 0),
			Check: resource.ComposeTestCheckFunc(
				testAccCheckBGPGroupFirewallSetState(groupID, firewallSetID, 0, 0),
				testAccDeleteBGPGroupFirewallSetOutOfBand("netactuate_bgp_group_firewall_set.test"),
			),
			// Provider bug: resourceBGPGroupFirewallSetRead only reparses the ID and
			// cannot currently detect that the API binding was deleted out of band.
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateFirewallSetRuleOrderLifecycle(t *testing.T) {
	name := testAccName("forder")
	createConfig := testAccFirewallSetRuleOrderConfig(name, false)
	modifiedConfig := testAccFirewallSetRuleOrderConfig(name, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_firewall_set_rule_order.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
			testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", testAccFirewallSetRuleOrderRules(), []string{"gamma", "alpha", "beta"}),
		), testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", testAccFirewallSetRuleOrderRules(), []string{"alpha", "beta", "gamma"}), testAccLifecycleOptions{
			ImportStateIdFunc: testAccBGPFirewallLifecycleImportID("netactuate_firewall_set_rule_order.test"),
		}),
	})
}

func TestAccNetactuateCloudFirewallSetBindingLifecycle(t *testing.T) {
	name := testAccName("cfwbind")
	serverID := testAccEnvInt(t, "NETACTUATE_ACC_WRITABLE_SERVER_ID")
	config := testAccCloudFirewallSetBindingConfig(name, serverID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_WRITABLE_SERVER_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: testAccLifecycleNoModifySteps(config, "netactuate_cloud_firewall_set_binding.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
			testAccCheckCloudFirewallSetBindingAPI("netactuate_cloud_firewall_set_binding.test"),
		), testAccLifecycleOptions{
			ImportStateIdFunc:       testAccBGPFirewallLifecycleImportID("netactuate_cloud_firewall_set_binding.test"),
			ImportStateVerifyIgnore: []string{"raw_json", "set_priority"},
		}),
	})
}

func TestAccNetactuateCloudFirewallSetBindingOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("cfwbind-oob")
	serverID := testAccEnvInt(t, "NETACTUATE_ACC_WRITABLE_SERVER_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_WRITABLE_SERVER_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: []resource.TestStep{{
			Config: testAccCloudFirewallSetBindingResourceOnlyConfig(name, serverID),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
				testAccCheckCloudFirewallSetBindingAPI("netactuate_cloud_firewall_set_binding.test"),
				testAccDeleteCloudFirewallSetBindingOutOfBand("netactuate_cloud_firewall_set_binding.test"),
			),
			// Provider bug: resourceCloudFirewallSetBindingRead only reparses the ID and
			// cannot currently detect that the API binding was deleted out of band.
			ExpectNonEmptyPlan: true,
		}},
	})
}

// Not covered by OutOfBandDeleteIdempotent:
// netactuate_bgp_group has no API delete endpoint.
// netactuate_firewall_set_rule_order is an action marker. The API has no delete operation for a reorder.

func testAccLifecycleNoModifySteps(config, resourceAddress string, checkCreate resource.TestCheckFunc, opts testAccLifecycleOptions) []resource.TestStep {
	return []resource.TestStep{
		{
			Config: config,
			Check:  checkCreate,
		},
		{
			Config:             config,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			ResourceName:            resourceAddress,
			ImportState:             true,
			ImportStateIdFunc:       opts.ImportStateIdFunc,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: opts.ImportStateVerifyIgnore,
		},
		{
			Config:             config,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	}
}

func testAccBGPFirewallLifecycleImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return testAccStateID(s, address)
	}
}

func testAccCheckBGPGroupFirewallSetState(groupID, firewallSetID, interfaceNumber, setPriority int) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("netactuate_bgp_group_firewall_set.test", "bgp_group_id", strconv.Itoa(groupID)),
		resource.TestCheckResourceAttr("netactuate_bgp_group_firewall_set.test", "firewall_set_id", strconv.Itoa(firewallSetID)),
		resource.TestCheckResourceAttr("netactuate_bgp_group_firewall_set.test", "interface_number", strconv.Itoa(interfaceNumber)),
		resource.TestCheckResourceAttr("netactuate_bgp_group_firewall_set.test", "set_priority", strconv.Itoa(setPriority)),
		testAccCheckAttributePresent("netactuate_bgp_group_firewall_set.test", "binding_id"),
	)
}

func testAccDeleteBGPGroupFirewallSetOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		groupID, firewallSetID, err := testAccTwoPartStateID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V2.UnbindBGPGroupFirewallSet(groupID, firewallSetID)
	}
}

func testAccDeleteCloudFirewallSetBindingOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		mbpkgID, firewallSetID, err := testAccTwoPartStateID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V2.UnbindCloudFirewallSet(mbpkgID, strconv.Itoa(firewallSetID))
	}
}

func testAccFirewallSetRuleOrderRules() []testAccFirewallRule {
	return []testAccFirewallRule{
		{Name: "alpha", Priority: 10, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
		{Name: "beta", Priority: 20, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
		{Name: "gamma", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
	}
}

func testAccBGPGroupFirewallSetResourceOnlyConfig(groupID, firewallSetID, interfaceNumber, setPriority int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_bgp_group_firewall_set" "test" {
  bgp_group_id     = %d
  firewall_set_id  = %d
  interface_number = %d
  set_priority     = %d
}
`, groupID, firewallSetID, interfaceNumber, setPriority)
}

func testAccCloudFirewallSetBindingResourceOnlyConfig(name string, serverID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_firewall_set" "test" {
  name        = %q
  description = %q
  enabled     = true
}

resource "netactuate_cloud_firewall_set_binding" "test" {
  mbpkgid         = %d
  firewall_set_id = netactuate_firewall_set.test.id
  set_priority    = 40
}
`, name, name, serverID)
}
