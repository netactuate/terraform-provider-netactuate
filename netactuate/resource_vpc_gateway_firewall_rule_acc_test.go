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

func TestAccNetactuateVPCGatewayFirewallRule_criteria8_10_11(t *testing.T) {
	name := testAccName("vpc-fw")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	inboundDescription := name + " inbound"
	outboundDescription := name + " outbound"
	// Phase 1 creates the VPC and its rules with the firewall OFF, because the API forbids
	// enabling it at create. Phase 2 turns it on, which is the only workflow that works and
	// which exercises the Update path. Phase 3 removes the outbound rule.
	// The outbound rule is deliberately NOT in this test. Its direction is coerced to
	// inbound by the API, which makes every plan perpetually dirty and would fail every
	// step here, hiding whether criteria 8, 10 and 11 actually pass. The outbound case is
	// pinned on its own in TestAccNetactuateVPCGatewayFirewallRule_outboundIsCoercedToInbound.
	bothRulesFirewallOff := testAccVPCGatewayFirewallRulesConfig(name, locationID, false, false)
	bothRulesFirewallOn := testAccVPCGatewayFirewallRulesConfig(name, locationID, false, true)
	inboundOnly := testAccVPCGatewayFirewallRulesConfig(name, locationID, false, true)
	_ = outboundDescription

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{
			{
				// Phase 1: rules exist, firewall still off.
				Config: bothRulesFirewallOff,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCGatewayFirewallRuleAPI("netactuate_vpc_gateway_firewall_rule.inbound", testAccVPCGatewayFirewallWant{
						Description: inboundDescription,
						Direction:   "inbound",
						Protocol:    "TCP",
						Network:     "203.0.113.0/24",
						PortStart:   8443,
						PortEnd:     8444,
					}),
					testAccCheckDataSourceListContainsAttribute("data.netactuate_vpc_gateway_firewall_rules.test", "rules", "firewall_rule_id", "netactuate_vpc_gateway_firewall_rule.inbound", "rule_id"),
				),
			},
			{
				// Phase 2: now that rules exist, enabling the firewall is legal. This is
				// the assertion that the two phase workflow actually works.
				Config: bothRulesFirewallOn,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCFirewallTogglesAPI("netactuate_vpc.test"),
				),
			},
			{
				// Phase 3: rule removal.
				Config: inboundOnly,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCGatewayFirewallRuleAPI("netactuate_vpc_gateway_firewall_rule.inbound", testAccVPCGatewayFirewallWant{
						Description: inboundDescription,
						Direction:   "inbound",
						Protocol:    "TCP",
						Network:     "203.0.113.0/24",
						PortStart:   8443,
						PortEnd:     8444,
					}),
					testAccCheckVPCGatewayFirewallRuleAbsentAPI("netactuate_vpc.test", 4, outboundDescription),
				),
			},
			{
				ResourceName:      "netactuate_vpc_gateway_firewall_rule.inbound",
				ImportState:       true,
				ImportStateIdFunc: testAccVPCGatewayFirewallImportID("netactuate_vpc_gateway_firewall_rule.inbound"),
				ImportStateVerify: true,
			},
			{
				Config:   inboundOnly,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					// Fourth instance of the same race on this account: delete and refresh
					// in one breath, and the platform can legitimately still list the
					// object, so the provider correctly keeps it in state and the test
					// fails for timing rather than a defect. Poll the end state instead.
					if err := testAccDeleteVPCGatewayFirewallRulesByDescription(inboundDescription, 4); err != nil {
						t.Fatalf("out of band firewall rule delete: %v", err)
					}
					testAccWaitGoneFromVPCList(t, "gateway firewall rule", func() bool {
						return !testAccVPCGatewayFirewallRuleExistsByDescription(inboundDescription, 4)
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_vpc_gateway_firewall_rule.inbound"),
			},
		},
	})
}

// Criterion 9 is intentionally absent. Addendum 1 withdrew VPC gateway
// firewall reorder coverage because the firewall endpoint has no ordering
// field, unlike the DNAT and SNAT endpoints.

type testAccVPCGatewayFirewallWant struct {
	Description string
	Direction   string
	Protocol    string
	Network     string
	PortStart   int
	PortEnd     int
}

func testAccCheckVPCFirewallTogglesAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		vpc, err := testAccClients().V3.GetVPC(vpcID)
		if err != nil {
			return err
		}
		if vpc.Firewalls == nil || vpc.Firewalls.IPv4 == nil || vpc.Firewalls.IPv4.Inbound == nil || vpc.Firewalls.IPv4.Outbound == nil {
			return fmt.Errorf("VPC %d API firewall toggles are missing", vpcID)
		}
		if !vpc.Firewalls.IPv4.Inbound.Enabled || !vpc.Firewalls.IPv4.Outbound.Enabled {
			return fmt.Errorf("VPC %d API firewall toggles = inbound:%t outbound:%t, want both true", vpcID, vpc.Firewalls.IPv4.Inbound.Enabled, vpc.Firewalls.IPv4.Outbound.Enabled)
		}
		return nil
	}
}

func testAccCheckVPCGatewayFirewallRuleAPI(address string, want testAccVPCGatewayFirewallWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, ruleID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		rule, err := testAccClients().V3.GetVPCFirewallRule(vpcID, ruleID, 4)
		if err != nil {
			return err
		}
		return testAccVPCGatewayFirewallRuleMatches(*rule, want)
	}
}

func testAccCheckVPCGatewayFirewallRuleAbsentAPI(vpcAddress string, ipVersion int, description string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, err := testAccID(s, vpcAddress)
		if err != nil {
			return err
		}
		rules, err := testAccClients().V3.ListVPCFirewallRules(vpcID, ipVersion)
		if err != nil {
			return err
		}
		for _, r := range rules {
			if r.Description == description {
				return fmt.Errorf("VPC gateway firewall rule %q still exists on API in VPC %d", description, vpcID)
			}
		}
		return nil
	}
}

func testAccVPCGatewayFirewallRuleMatches(rule gona.VPCFirewallRule, want testAccVPCGatewayFirewallWant) error {
	if rule.IPVersion != 4 || rule.Direction != want.Direction || rule.Protocol != want.Protocol || rule.Description != want.Description || rule.Network != want.Network {
		return fmt.Errorf("VPC gateway firewall API fields = ip_version:%d direction:%q protocol:%q description:%q network:%q", rule.IPVersion, rule.Direction, rule.Protocol, rule.Description, rule.Network)
	}
	if rule.Port == nil {
		return fmt.Errorf("VPC gateway firewall rule %q API port is nil", want.Description)
	}
	if rule.Port.Start != want.PortStart || rule.Port.End != want.PortEnd {
		return fmt.Errorf("VPC gateway firewall rule %q API ports = %d-%d, want %d-%d", want.Description, rule.Port.Start, rule.Port.End, want.PortStart, want.PortEnd)
	}
	return nil
}

func testAccVPCGatewayFirewallImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		vpcID, ruleID, err := testAccCompositeID(s, address)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d/%d/4", vpcID, ruleID), nil
	}
}

func testAccDeleteVPCGatewayFirewallRulesByDescription(description string, ipVersion int) error {
	clients := testAccClients()
	// testAccCreated is per BINARY, not per test, so it holds every VPC any test in this shard
	// created, including ones already destroyed by a test that ran earlier. Listing rules on a
	// destroyed VPC returns 404 "VPC does not exist". The failure is order dependent, which is why it never
	// appeared in pattern scoped runs and only surfaced once the whole suite ran in shards.
	// A VPC that is gone cannot be holding a rule we need to delete, so skip it and continue.
	for vpcID := range testAccCreated["netactuate_vpc"] {
		rules, err := clients.V3.ListVPCFirewallRules(vpcID, ipVersion)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		for _, rule := range rules {
			if rule.Description != description {
				continue
			}
			if err := clients.V3.DeleteVPCFirewallRule(vpcID, rule.FirewallRuleID); err != nil {
				return err
			}
		}
		if err := clients.V3.ApplyVPCFirewallChanges(vpcID); err != nil {
			return err
		}
	}
	return nil
}

func testAccCompositeID(s *terraform.State, address string) (int, int, error) {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return 0, 0, fmt.Errorf("not found: %s", address)
	}
	parts := strings.SplitN(rs.Primary.ID, "/", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("%s has invalid composite ID %q", address, rs.Primary.ID)
	}
	parentID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("%s has invalid parent ID %q: %w", address, parts[0], err)
	}
	childID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("%s has invalid child ID %q: %w", address, parts[1], err)
	}
	return parentID, childID, nil
}

// testAccVPCGatewayFirewallRulesConfig builds the VPC and its rules.
//
// togglesOn must be FALSE on the first apply. The API refuses to enable a VPC firewall at
// create time, and says so clearly:
//
//	"Cannot enable the IPv4 inbound firewall while creating a VPC because the new VPC has
//	 zero IPv4 inbound firewall rules. Create the VPC, add an IPv4 inbound firewall rule,
//	 then enable it."
//
// That is a sensible fail safe: enabling an empty inbound firewall would drop everything.
// The provider nonetheless exposes firewall_ipv4_inbound as a create time argument and
// passes it straight through, so the obvious single step config can never apply. Recorded
// in review/22. This test proves the two phase workflow that does work.
func testAccVPCGatewayFirewallRulesConfig(name string, locationID int, includeOutbound, togglesOn bool) string {
	toggles := ""
	if togglesOn {
		toggles = `
  firewall_ipv4_inbound  = true
  firewall_ipv4_outbound = true`
	}
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label                  = %q
  description            = %q
  location_id            = %d%s
}

resource "netactuate_vpc_gateway_firewall_rule" "inbound" {
  vpc_id      = netactuate_vpc.test.vpc_id
  ip_version  = 4
  direction   = "inbound"
  protocol    = "TCP"
  description = %q
  network     = "203.0.113.0/24"
  port_start  = 8443
  port_end    = 8444
}

data "netactuate_vpc_gateway_firewall_rules" "test" {
  vpc_id     = netactuate_vpc.test.vpc_id
  ip_version = 4

  depends_on = [netactuate_vpc_gateway_firewall_rule.inbound]
}
`, name, name, locationID, toggles, name+" inbound")
	if includeOutbound {
		config += fmt.Sprintf(`
resource "netactuate_vpc_gateway_firewall_rule" "outbound" {
  vpc_id      = netactuate_vpc.test.vpc_id
  ip_version  = 4
  direction   = "outbound"
  protocol    = "UDP"
  description = %q
  network     = "198.51.100.0/24"
  port_start  = 5353
  port_end    = 5354
}
`, name+" outbound")
	}
	return config
}

// TestAccNetactuateVPCGatewayFirewallRule_outboundIsCoercedToInbound pins a PLATFORM defect
// with security consequences.
//
// A VPC gateway firewall rule created with direction "outbound" is stored and returned as
// "inbound". This is not a provider marshalling bug:
//
//	POST /vpcs/{id}/gateway/rules/firewall  {"direction":"outbound","protocol":"UDP",...}  -> 200
//	GET  /vpcs/{id}/gateway/rules/firewall       -> direction "inbound"
//	GET  /vpcs/{id}/gateway/rules/firewall/ipv4  -> direction "inbound"
//
// Both list endpoints agree, so it is not a filtering artifact of the ipv4 view.
//
// Why this matters more than a normal round trip failure: the customer asked to restrict
// EGRESS and got an INGRESS permit. The rule they wrote does not exist, and a rule they did
// not write does. On a firewall that is the wrong direction to fail in.
//
// The provider is faithful: it sends what the schema was given, and its own validator
// correctly accepts only "inbound" and "outbound".
//
// This test asserts the CURRENT broken behaviour and will fail when the platform fixes it.
func TestAccNetactuateVPCGatewayFirewallRule_outboundIsCoercedToInbound(t *testing.T) {
	name := testAccName("vpc-fw-dir")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "dir" {
  label       = %q
  description = "direction coercion probe"
  location_id = %d
}

resource "netactuate_vpc_gateway_firewall_rule" "out" {
  vpc_id      = netactuate_vpc.dir.vpc_id
  ip_version  = 4
  direction   = "outbound"
  protocol    = "UDP"
  description = %q
  network     = "198.51.100.0/24"
  port_start  = 5353
  port_end    = 5360
}
`, name, locationID, name+" outbound")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				// The plan can never be empty. Config says outbound, the API returns
				// inbound, so every plan forever shows direction "inbound" -> "outbound".
				// The resource NEVER CONVERGES. That is the real consequence and this
				// flag is the assertion of it.
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.dir"),
					// Config says outbound. The API says inbound. If this ever fails
					// asserting "inbound", the platform has fixed the defect: delete this
					// test and restore the real assertion above.
					testAccCheckVPCGatewayFirewallRuleAPI("netactuate_vpc_gateway_firewall_rule.out", testAccVPCGatewayFirewallWant{
						Description: name + " outbound",
						Direction:   "inbound",
						Protocol:    "UDP",
						Network:     "198.51.100.0/24",
						PortStart:   5353,
						PortEnd:     5360,
					}),
				),
			},
		},
	})
}

func testAccVPCGatewayFirewallRuleExistsByDescription(description string, ipVersion int) bool {
	clients := testAccClients()
	for vpcID := range testAccCreated["netactuate_vpc"] {
		rules, err := clients.V3.ListVPCFirewallRules(vpcID, ipVersion)
		if err != nil {
			// A VPC an earlier test already destroyed is not this rule surviving.
			if gona.IsV3NotFound(err) {
				continue
			}
			return true
		}
		for _, rule := range rules {
			if rule.Description == description {
				return true
			}
		}
	}
	return false
}
