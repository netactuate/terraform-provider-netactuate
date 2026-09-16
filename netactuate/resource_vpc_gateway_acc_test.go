//go:build acctest

package netactuate

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateVPCGatewayDNATRule_importPlanReorderRemoveAndOutOfBandDelete(t *testing.T) {
	name := testAccName("vpc-dnat")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	singleConfig := testAccVPCGatewayDNATSingleConfig(name, locationID)
	initialOrderConfig := testAccVPCGatewayDNATOrderConfig(name, locationID, true, false)
	reorderedConfig := testAccVPCGatewayDNATOrderConfig(name, locationID, true, true)
	removedMiddleConfig := testAccVPCGatewayDNATOrderConfig(name, locationID, false, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCGatewayDNATRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: singleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCGatewayDNATRuleAPI("netactuate_vpc_gateway_dnat_rule.test", testAccVPCGatewayDNATWant{
						Description:          name + " dnat test",
						Protocol:             "TCP",
						MatchAddress:         "203.0.113.10",
						MatchPortStart:       8443,
						MatchPortEnd:         8444,
						TranslationAddress:   "192.0.2.10",
						TranslationPortStart: 443,
						TranslationPortEnd:   444,
					}),
					testAccCheckDataSourceListContainsAttribute("data.netactuate_vpc_gateway_dnat_rules.test", "rules", "dnat_rule_id", "netactuate_vpc_gateway_dnat_rule.test", "rule_id"),
				),
			},
			{
				ResourceName:      "netactuate_vpc_gateway_dnat_rule.test",
				ImportState:       true,
				ImportStateIdFunc: testAccVPCGatewayRuleImportID("netactuate_vpc_gateway_dnat_rule.test"),
				ImportStateVerify: true,
			},
			{
				Config:   singleConfig,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					if err := testAccDeleteVPCGatewayDNATRulesByDescription(name+" dnat test", 4); err != nil {
						t.Fatalf("out of band DNAT delete: %v", err)
					}
					testAccWaitGoneFromVPCList(t, "DNAT rule", func() bool {
						return !testAccVPCGatewayDNATRuleExistsByDescription(name+" dnat test", 4)
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_vpc_gateway_dnat_rule.test"),
			},
			{
				Config: initialOrderConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCGatewayDNATOrderAPI("netactuate_vpc.test", []string{name + " dnat a", name + " dnat b", name + " dnat c"}),
				),
			},
			{
				Config: reorderedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCGatewayDNATOrderAPI("netactuate_vpc.test", []string{name + " dnat c", name + " dnat a", name + " dnat b"}),
				),
			},
			{
				Config: removedMiddleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCGatewayDNATRuleAbsentAPI("netactuate_vpc.test", 4, name+" dnat a"),
					testAccCheckVPCGatewayDNATOrderAPI("netactuate_vpc.test", []string{name + " dnat c", name + " dnat b"}),
				),
			},
		},
	})
}

func TestAccNetactuateVPCGatewaySNATRule_importPlanReorderRemoveAndOutOfBandDelete(t *testing.T) {
	name := testAccName("vpc-snat")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	singleConfig := testAccVPCGatewaySNATSingleConfig(name, locationID)
	initialOrderConfig := testAccVPCGatewaySNATOrderConfig(name, locationID, true, false)
	reorderedConfig := testAccVPCGatewaySNATOrderConfig(name, locationID, true, true)
	removedMiddleConfig := testAccVPCGatewaySNATOrderConfig(name, locationID, false, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCGatewaySNATRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: singleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCGatewaySNATRuleAPI("netactuate_vpc_gateway_snat_rule.test", testAccVPCGatewaySNATWant{
						Description:             name + " snat test",
						Protocol:                "UDP",
						MatchInternalCIDR:       "192.0.2.0/24",
						TranslationAddressStart: "198.51.100.10",
						TranslationAddressEnd:   "198.51.100.11",
						TranslationPortStart:    5300,
						TranslationPortEnd:      5301,
					}),
					testAccCheckDataSourceListContainsAttribute("data.netactuate_vpc_gateway_snat_rules.test", "rules", "snat_rule_id", "netactuate_vpc_gateway_snat_rule.test", "rule_id"),
				),
			},
			{
				ResourceName:      "netactuate_vpc_gateway_snat_rule.test",
				ImportState:       true,
				ImportStateIdFunc: testAccVPCGatewayRuleImportID("netactuate_vpc_gateway_snat_rule.test"),
				ImportStateVerify: true,
			},
			{
				Config:   singleConfig,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					if err := testAccDeleteVPCGatewaySNATRulesByDescription(name+" snat test", 4); err != nil {
						t.Fatalf("out of band SNAT delete: %v", err)
					}
					testAccWaitGoneFromVPCList(t, "SNAT rule", func() bool {
						return !testAccVPCGatewaySNATRuleExistsByDescription(name+" snat test", 4)
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_vpc_gateway_snat_rule.test"),
			},
			{
				Config: initialOrderConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCGatewaySNATOrderAPI("netactuate_vpc.test", []string{name + " snat a", name + " snat b", name + " snat c"}),
				),
			},
			{
				Config: reorderedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCGatewaySNATOrderAPI("netactuate_vpc.test", []string{name + " snat c", name + " snat a", name + " snat b"}),
				),
			},
			{
				Config: removedMiddleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCGatewaySNATRuleAbsentAPI("netactuate_vpc.test", 4, name+" snat a"),
					testAccCheckVPCGatewaySNATOrderAPI("netactuate_vpc.test", []string{name + " snat c", name + " snat b"}),
				),
			},
		},
	})
}

func TestAccNetactuateVPCFloatingIP_createAPIAndOutOfBandDelete(t *testing.T) {
	name := testAccName("vpc-fip")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	config := testAccVPCFloatingIPConfig(name, locationID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCFloatingIPDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCFloatingIPAPI("netactuate_vpc_floating_ip.test", name+".example.invalid"),
					testAccCheckDataSourceListContainsAttribute("data.netactuate_vpc_floating_ips.test", "floating_ips", "floating_ip_id", "netactuate_vpc_floating_ip.test", "floating_ip_id"),
				),
			},
			{
				PreConfig: func() {
					if err := testAccDeleteVPCFloatingIPsByPTR(name + ".example.invalid"); err != nil {
						t.Fatalf("out of band floating IP delete: %v", err)
					}
					testAccWaitGoneFromVPCList(t, "floating IP", func() bool {
						return !testAccVPCFloatingIPExistsByPTR(name + ".example.invalid")
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_vpc_floating_ip.test"),
			},
		},
	})
}

func TestAccNetactuateVPCSSHKey_importPlanAndAPIRead(t *testing.T) {
	name := testAccName("vpc-ssh-key")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	config := testAccVPCSSHKeyConfig(name, locationID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccTrackFromState("netactuate_sshkey", "netactuate_sshkey.test"),
					testAccCheckVPCSSHKeyAPI("netactuate_vpc_ssh_key.test", name),
				),
			},
			{
				ResourceName:      "netactuate_vpc_ssh_key.test",
				ImportState:       true,
				ImportStateIdFunc: testAccVPCSSHKeyImportID("netactuate_vpc_ssh_key.test"),
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

type testAccVPCGatewayDNATWant struct {
	Description          string
	Protocol             string
	MatchAddress         string
	MatchPortStart       int
	MatchPortEnd         int
	TranslationAddress   string
	TranslationPortStart int
	TranslationPortEnd   int
}

type testAccVPCGatewaySNATWant struct {
	Description             string
	Protocol                string
	MatchInternalCIDR       string
	TranslationAddressStart string
	TranslationAddressEnd   string
	TranslationPortStart    int
	TranslationPortEnd      int
}

func testAccCheckVPCGatewayDNATRuleAPI(address string, want testAccVPCGatewayDNATWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, ruleID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		rule, err := testAccClients().V3.GetVPCDNATRule(vpcID, ruleID, 4)
		if err != nil {
			return err
		}
		if rule.IPVersion != 4 || rule.Protocol != want.Protocol || rule.Description != want.Description {
			return fmt.Errorf("%s API fields = ip_version:%d protocol:%q description:%q", address, rule.IPVersion, rule.Protocol, rule.Description)
		}
		if rule.Match == nil || rule.Match.Port == nil {
			return fmt.Errorf("%s API match is incomplete", address)
		}
		if rule.Match.Address != want.MatchAddress || rule.Match.Port.Start != want.MatchPortStart || rule.Match.Port.End != want.MatchPortEnd {
			return fmt.Errorf("%s API match = %s %d-%d, want %s %d-%d", address, rule.Match.Address, rule.Match.Port.Start, rule.Match.Port.End, want.MatchAddress, want.MatchPortStart, want.MatchPortEnd)
		}
		if rule.Translation == nil || rule.Translation.Port == nil {
			return fmt.Errorf("%s API translation is incomplete", address)
		}
		if rule.Translation.Address != want.TranslationAddress || rule.Translation.Port.Start != want.TranslationPortStart || rule.Translation.Port.End != want.TranslationPortEnd {
			return fmt.Errorf("%s API translation = %s %d-%d, want %s %d-%d", address, rule.Translation.Address, rule.Translation.Port.Start, rule.Translation.Port.End, want.TranslationAddress, want.TranslationPortStart, want.TranslationPortEnd)
		}
		return nil
	}
}

func testAccCheckVPCGatewaySNATRuleAPI(address string, want testAccVPCGatewaySNATWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, ruleID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		rule, err := testAccClients().V3.GetVPCSNATRule(vpcID, ruleID, 4)
		if err != nil {
			return err
		}
		if rule.IPVersion != 4 || rule.Protocol != want.Protocol || rule.Description != want.Description {
			return fmt.Errorf("%s API fields = ip_version:%d protocol:%q description:%q", address, rule.IPVersion, rule.Protocol, rule.Description)
		}
		if rule.Match == nil || rule.Match.InternalCidr != want.MatchInternalCIDR {
			return fmt.Errorf("%s API match_internal_cidr = %#v, want %q", address, rule.Match, want.MatchInternalCIDR)
		}
		if rule.Translation == nil || rule.Translation.Address == nil || rule.Translation.Port == nil {
			return fmt.Errorf("%s API translation is incomplete", address)
		}
		if rule.Translation.Address.Start != want.TranslationAddressStart || rule.Translation.Address.End != want.TranslationAddressEnd {
			return fmt.Errorf("%s API translation address = %s-%s, want %s-%s", address, rule.Translation.Address.Start, rule.Translation.Address.End, want.TranslationAddressStart, want.TranslationAddressEnd)
		}
		if rule.Translation.Port.Start != want.TranslationPortStart || rule.Translation.Port.End != want.TranslationPortEnd {
			return fmt.Errorf("%s API translation port = %d-%d, want %d-%d", address, rule.Translation.Port.Start, rule.Translation.Port.End, want.TranslationPortStart, want.TranslationPortEnd)
		}
		return nil
	}
}

func testAccCheckVPCGatewayDNATOrderAPI(vpcAddress string, wantDescriptions []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, err := testAccID(s, vpcAddress)
		if err != nil {
			return err
		}
		rules, err := testAccClients().V3.ListVPCDNATRules(vpcID, 4)
		if err != nil {
			return err
		}
		got := testAccDNATDescriptionsInAPIOrder(rules, wantDescriptions)
		if !reflect.DeepEqual(got, wantDescriptions) {
			return fmt.Errorf("VPC %d API DNAT order = %v, want %v", vpcID, got, wantDescriptions)
		}
		return nil
	}
}

func testAccCheckVPCGatewaySNATOrderAPI(vpcAddress string, wantDescriptions []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, err := testAccID(s, vpcAddress)
		if err != nil {
			return err
		}
		rules, err := testAccClients().V3.ListVPCSNATRules(vpcID, 4)
		if err != nil {
			return err
		}
		got := testAccSNATDescriptionsInAPIOrder(rules, wantDescriptions)
		if !reflect.DeepEqual(got, wantDescriptions) {
			return fmt.Errorf("VPC %d API SNAT order = %v, want %v", vpcID, got, wantDescriptions)
		}
		return nil
	}
}

func testAccCheckVPCGatewayDNATRuleAbsentAPI(vpcAddress string, ipVersion int, description string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, err := testAccID(s, vpcAddress)
		if err != nil {
			return err
		}
		rules, err := testAccClients().V3.ListVPCDNATRules(vpcID, ipVersion)
		if err != nil {
			return err
		}
		for _, rule := range rules {
			if rule.Description == description {
				return fmt.Errorf("VPC DNAT rule %q still exists on API in VPC %d", description, vpcID)
			}
		}
		return nil
	}
}

func testAccCheckVPCGatewaySNATRuleAbsentAPI(vpcAddress string, ipVersion int, description string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, err := testAccID(s, vpcAddress)
		if err != nil {
			return err
		}
		rules, err := testAccClients().V3.ListVPCSNATRules(vpcID, ipVersion)
		if err != nil {
			return err
		}
		for _, rule := range rules {
			if rule.Description == description {
				return fmt.Errorf("VPC SNAT rule %q still exists on API in VPC %d", description, vpcID)
			}
		}
		return nil
	}
}

func testAccDNATDescriptionsInAPIOrder(rules []gona.VPCDNATRule, wantDescriptions []string) []string {
	want := testAccStringSet(wantDescriptions)
	var got []string
	for _, rule := range rules {
		if want[rule.Description] {
			got = append(got, rule.Description)
		}
	}
	return got
}

func testAccSNATDescriptionsInAPIOrder(rules []gona.VPCSNATRule, wantDescriptions []string) []string {
	want := testAccStringSet(wantDescriptions)
	var got []string
	for _, rule := range rules {
		if want[rule.Description] {
			got = append(got, rule.Description)
		}
	}
	return got
}

func testAccStringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func testAccCheckDataSourceListContainsAttribute(dataAddress, listAttr, idAttr, resourceAddress, resourceAttr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		dataResource, ok := s.RootModule().Resources[dataAddress]
		if !ok || dataResource.Primary == nil {
			return fmt.Errorf("%s not found in state", dataAddress)
		}
		createdResource, ok := s.RootModule().Resources[resourceAddress]
		if !ok || createdResource.Primary == nil {
			return fmt.Errorf("%s not found in state", resourceAddress)
		}

		wantID, ok := createdResource.Primary.Attributes[resourceAttr]
		if !ok || wantID == "" {
			return fmt.Errorf("%s.%s is empty", resourceAddress, resourceAttr)
		}
		countRaw, ok := dataResource.Primary.Attributes[listAttr+".#"]
		if !ok {
			return fmt.Errorf("%s.%s count is missing", dataAddress, listAttr)
		}
		count, err := strconv.Atoi(countRaw)
		if err != nil {
			return fmt.Errorf("%s.%s count %q is invalid: %w", dataAddress, listAttr, countRaw, err)
		}
		if count == 0 {
			return fmt.Errorf("%s.%s is empty", dataAddress, listAttr)
		}
		for i := 0; i < count; i++ {
			if dataResource.Primary.Attributes[fmt.Sprintf("%s.%d.%s", listAttr, i, idAttr)] == wantID {
				return nil
			}
		}
		return fmt.Errorf("%s.%s does not contain %s.%s %s", dataAddress, listAttr, resourceAddress, resourceAttr, wantID)
	}
}

func testAccCheckVPCFloatingIPAPI(address, ptr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, fipID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		fip, err := testAccClients().V3.GetVPCFloatingIP(vpcID, fipID)
		if err != nil {
			return err
		}
		if fip.PTR != ptr {
			return fmt.Errorf("%s API ptr = %q, want %q", address, fip.PTR, ptr)
		}
		if fip.Address == "" {
			return fmt.Errorf("%s API address is empty", address)
		}
		if fip.IPVersion != 4 {
			return fmt.Errorf("%s API ip_version = %d, want 4", address, fip.IPVersion)
		}
		return nil
	}
}

func testAccCheckVPCSSHKeyAPI(address, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, sshKeyID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		key, err := testAccClients().V3.GetVPCSSHKey(vpcID, sshKeyID)
		if err != nil {
			return err
		}
		if key.Name != name {
			return fmt.Errorf("%s API name = %q, want %q", address, key.Name, name)
		}
		if !key.IsEnabled() {
			return fmt.Errorf("%s API key is not enabled", address)
		}
		if key.PublicKey == "" || key.Fingerprint == "" {
			return fmt.Errorf("%s API key detail missing public key or fingerprint", address)
		}
		return nil
	}
}

func testAccCheckVPCGatewayDNATRuleDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_vpc_gateway_dnat_rule" {
			continue
		}
		vpcID, ruleID, err := parseDNATRuleID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetVPCDNATRule(vpcID, ruleID, 4); err == nil {
			return fmt.Errorf("VPC DNAT rule still exists: %s", rs.Primary.ID)
		} else if !gona.IsV3NotFound(err) && !testAccVPCParentIsGone(err) {
			return err
		}
	}
	return testAccCheckVPCDestroy(s)
}

func testAccCheckVPCGatewaySNATRuleDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_vpc_gateway_snat_rule" {
			continue
		}
		vpcID, ruleID, err := parseSNATRuleID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetVPCSNATRule(vpcID, ruleID, 4); err == nil {
			return fmt.Errorf("VPC SNAT rule still exists: %s", rs.Primary.ID)
		} else if !gona.IsV3NotFound(err) && !testAccVPCParentIsGone(err) {
			return err
		}
	}
	return testAccCheckVPCDestroy(s)
}

func testAccCheckVPCFloatingIPDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_vpc_floating_ip" {
			continue
		}
		vpcID, fipID, err := parseFloatingIPID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetVPCFloatingIP(vpcID, fipID); err == nil {
			return fmt.Errorf("VPC floating IP still exists: %s", rs.Primary.ID)
		} else if !gona.IsV3NotFound(err) && !testAccVPCParentIsGone(err) {
			return err
		}
	}
	return testAccCheckVPCDestroy(s)
}

func testAccCheckVPCSSHKeyDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_vpc_ssh_key" {
			continue
		}
		vpcID, sshKeyID, err := parseVPCSSHKeyID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if key, err := clients.V3.GetVPCSSHKey(vpcID, sshKeyID); err == nil && key.IsEnabled() {
			return fmt.Errorf("VPC SSH key still enabled: %s", rs.Primary.ID)
		} else if err != nil && !gona.IsV3NotFound(err) && !testAccVPCParentIsGone(err) {
			return err
		}
	}
	if err := testAccCheckSSHKeyDestroy(s); err != nil {
		return err
	}
	return testAccCheckVPCDestroy(s)
}

func testAccVPCParentIsGone(err error) bool {
	return strings.Contains(err.Error(), "VPC") && strings.Contains(err.Error(), "not found")
}

func testAccDeleteVPCGatewayDNATRulesByDescription(description string, ipVersion int) error {
	clients := testAccClients()
	for vpcID := range testAccCreated["netactuate_vpc"] {
		rules, err := clients.V3.ListVPCDNATRules(vpcID, ipVersion)
		if err != nil {
			// testAccCreated accumulates VPC ids across every test in the run, and earlier
			// tests have already destroyed theirs, so listing one 404s. That is not a
			// failure of THIS delete. Skip a VPC that is gone rather than erroring on it.
			//
			// This only surfaced once the callers stopped discarding the error with _ =,
			// which is the argument for not discarding it.
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		for _, rule := range rules {
			if rule.Description == description {
				if err := clients.V3.DeleteVPCDNATRule(vpcID, rule.DNATRuleID); err != nil {
					return err
				}
			}
		}
		if err := clients.V3.ApplyVPCDNATChanges(vpcID); err != nil {
			return err
		}
	}
	return nil
}

func testAccDeleteVPCGatewaySNATRulesByDescription(description string, ipVersion int) error {
	clients := testAccClients()
	for vpcID := range testAccCreated["netactuate_vpc"] {
		rules, err := clients.V3.ListVPCSNATRules(vpcID, ipVersion)
		if err != nil {
			// testAccCreated accumulates VPC ids across every test in the run, and earlier
			// tests have already destroyed theirs, so listing one 404s. That is not a
			// failure of THIS delete. Skip a VPC that is gone rather than erroring on it.
			//
			// This only surfaced once the callers stopped discarding the error with _ =,
			// which is the argument for not discarding it.
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		for _, rule := range rules {
			if rule.Description == description {
				if err := clients.V3.DeleteVPCSNATRule(vpcID, rule.SNATRuleID); err != nil {
					return err
				}
			}
		}
		if err := clients.V3.ApplyVPCSNATChanges(vpcID); err != nil {
			return err
		}
	}
	return nil
}

func testAccDeleteVPCFloatingIPsByPTR(ptr string) error {
	clients := testAccClients()
	for vpcID := range testAccCreated["netactuate_vpc"] {
		fips, err := clients.V3.ListVPCFloatingIPs(vpcID)
		if err != nil {
			// testAccCreated accumulates VPC ids across every test in the run, and earlier
			// tests have already destroyed theirs, so listing one 404s. That is not a
			// failure of THIS delete. Skip a VPC that is gone rather than erroring on it.
			//
			// This only surfaced once the callers stopped discarding the error with _ =,
			// which is the argument for not discarding it.
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		for _, fip := range fips {
			if fip.PTR == ptr {
				if err := clients.V3.DeleteVPCFloatingIP(vpcID, fip.FloatingIPID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func testAccVPCGatewayRuleImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		vpcID, ruleID, err := testAccCompositeID(s, address)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d/%d/4", vpcID, ruleID), nil
	}
}

func testAccVPCSSHKeyImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		vpcID, sshKeyID, err := testAccCompositeID(s, address)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d/%d", vpcID, sshKeyID), nil
	}
}

func testAccVPCGatewayDNATSingleConfig(name string, locationID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_gateway_dnat_rule" "test" {
  vpc_id                 = netactuate_vpc.test.vpc_id
  ip_version             = 4
  protocol               = "TCP"
  description            = %q
  match_address          = "203.0.113.10"
  match_port_start       = 8443
  match_port_end         = 8444
  translation_address    = "192.0.2.10"
  translation_port_start = 443
  translation_port_end   = 444
}

data "netactuate_vpc_gateway_dnat_rules" "test" {
  vpc_id     = netactuate_vpc.test.vpc_id
  ip_version = 4

  depends_on = [netactuate_vpc_gateway_dnat_rule.test]
}
`, name, name, locationID, name+" dnat test")
}

func testAccVPCGatewaySNATSingleConfig(name string, locationID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_gateway_snat_rule" "test" {
  vpc_id                    = netactuate_vpc.test.vpc_id
  ip_version                = 4
  protocol                  = "UDP"
  description               = %q
  match_internal_cidr       = "192.0.2.0/24"
  translation_address_start = "198.51.100.10"
  translation_address_end   = "198.51.100.11"
  translation_port_start    = 5300
  translation_port_end      = 5301
}

data "netactuate_vpc_gateway_snat_rules" "test" {
  vpc_id     = netactuate_vpc.test.vpc_id
  ip_version = 4

  depends_on = [netactuate_vpc_gateway_snat_rule.test]
}
`, name, name, locationID, name+" snat test")
}

func testAccVPCGatewayDNATOrderConfig(name string, locationID int, includeMiddle, reordered bool) string {
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}
`, name, name, locationID)
	if includeMiddle {
		if reordered {
			config += testAccVPCGatewayDNATRuleBlock("a", name+" dnat a", "192.0.2.21", `priority_after_rule_id = netactuate_vpc_gateway_dnat_rule.c.rule_id`)
			config += testAccVPCGatewayDNATRuleBlock("b", name+" dnat b", "192.0.2.22", `priority_after_rule_id = netactuate_vpc_gateway_dnat_rule.a.rule_id`)
			config += testAccVPCGatewayDNATRuleBlock("c", name+" dnat c", "192.0.2.23", `priority_location = "start"`)
			return config
		}
		config += testAccVPCGatewayDNATRuleBlock("a", name+" dnat a", "192.0.2.21", `priority_location = "start"`)
		config += testAccVPCGatewayDNATRuleBlock("b", name+" dnat b", "192.0.2.22", `priority_after_rule_id = netactuate_vpc_gateway_dnat_rule.a.rule_id`)
		config += testAccVPCGatewayDNATRuleBlock("c", name+" dnat c", "192.0.2.23", `priority_after_rule_id = netactuate_vpc_gateway_dnat_rule.b.rule_id`)
		return config
	}
	config += testAccVPCGatewayDNATRuleBlock("b", name+" dnat b", "192.0.2.22", `priority_after_rule_id = netactuate_vpc_gateway_dnat_rule.c.rule_id`)
	config += testAccVPCGatewayDNATRuleBlock("c", name+" dnat c", "192.0.2.23", `priority_location = "start"`)
	return config
}

func testAccVPCGatewaySNATOrderConfig(name string, locationID int, includeMiddle, reordered bool) string {
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}
`, name, name, locationID)
	if includeMiddle {
		if reordered {
			config += testAccVPCGatewaySNATRuleBlock("a", name+" snat a", "192.0.2.31/32", `priority_after_rule_id = netactuate_vpc_gateway_snat_rule.c.rule_id`)
			config += testAccVPCGatewaySNATRuleBlock("b", name+" snat b", "192.0.2.32/32", `priority_after_rule_id = netactuate_vpc_gateway_snat_rule.a.rule_id`)
			config += testAccVPCGatewaySNATRuleBlock("c", name+" snat c", "192.0.2.33/32", `priority_location = "start"`)
			return config
		}
		config += testAccVPCGatewaySNATRuleBlock("a", name+" snat a", "192.0.2.31/32", `priority_location = "start"`)
		config += testAccVPCGatewaySNATRuleBlock("b", name+" snat b", "192.0.2.32/32", `priority_after_rule_id = netactuate_vpc_gateway_snat_rule.a.rule_id`)
		config += testAccVPCGatewaySNATRuleBlock("c", name+" snat c", "192.0.2.33/32", `priority_after_rule_id = netactuate_vpc_gateway_snat_rule.b.rule_id`)
		return config
	}
	config += testAccVPCGatewaySNATRuleBlock("b", name+" snat b", "192.0.2.32/32", `priority_after_rule_id = netactuate_vpc_gateway_snat_rule.c.rule_id`)
	config += testAccVPCGatewaySNATRuleBlock("c", name+" snat c", "192.0.2.33/32", `priority_location = "start"`)
	return config
}

func testAccVPCGatewayDNATRuleBlock(localName, description, translationAddress, priority string) string {
	lastOctet := strings.TrimPrefix(translationAddress, "192.0.2.")
	port, _ := strconv.Atoi(lastOctet)
	return fmt.Sprintf(`
resource "netactuate_vpc_gateway_dnat_rule" %q {
  vpc_id                 = netactuate_vpc.test.vpc_id
  ip_version             = 4
  protocol               = "TCP"
  description            = %q
  match_address          = "203.0.113.%d"
  match_port_start       = %d
  match_port_end         = %d
  translation_address    = %q
  translation_port_start = 80
  translation_port_end   = 80
  %s
}
`, localName, description, port, 8000+port, 8000+port, translationAddress, priority)
}

func testAccVPCGatewaySNATRuleBlock(localName, description, matchCIDR, priority string) string {
	lastOctet := strings.TrimSuffix(strings.TrimPrefix(matchCIDR, "192.0.2."), "/32")
	port, _ := strconv.Atoi(lastOctet)
	return fmt.Sprintf(`
resource "netactuate_vpc_gateway_snat_rule" %q {
  vpc_id                    = netactuate_vpc.test.vpc_id
  ip_version                = 4
  protocol                  = "TCP"
  description               = %q
  match_internal_cidr       = %q
  translation_address_start = "198.51.100.%d"
  translation_address_end   = "198.51.100.%d"
  translation_port_start    = %d
  translation_port_end      = %d
  %s
}
`, localName, description, matchCIDR, port, port, 1000+port, 1000+port, priority)
}

func testAccVPCFloatingIPConfig(name string, locationID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_floating_ip" "test" {
  vpc_id     = netactuate_vpc.test.vpc_id
  ip_version = 4
  ptr        = %q
}

data "netactuate_vpc_floating_ips" "test" {
  vpc_id = netactuate_vpc.test.vpc_id

  depends_on = [netactuate_vpc_floating_ip.test]
}
`, name, name, locationID, name+".example.invalid")
}

func testAccVPCSSHKeyConfig(name string, locationID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_sshkey" "test" {
  name = %q
  key  = %q
}

resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_ssh_key" "test" {
  vpc_id     = netactuate_vpc.test.vpc_id
  ssh_key_id = netactuate_sshkey.test.id
  enabled    = true
}
`, name, testAccSSHPublicKey, name, name, locationID)
}

// testAccWaitGoneFromVPCList polls until a predicate reports the object has disappeared from
// the VPC's own listing.
//
// WHY. Deleting through the API and refreshing immediately is a race: the platform is
// eventually consistent on deletes, so the refresh can legitimately still see the object,
// the provider correctly leaves it in state, and the test fails for a timing reason that
// looks exactly like a defect: a server after its delete job reported done, a VPC refusing
// deletion while its VMs detached, and here.
//
// The rule this encodes: poll the observable end state, never treat a delete call returning
// as proof the thing is gone.
func testAccWaitGoneFromVPCList(t *testing.T, what string, gone func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if gone() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s still present 2 minutes after it was deleted", what)
		}
		time.Sleep(5 * time.Second)
	}
}

// The three predicates testAccWaitGoneFromVPCList polls. Each answers one question against
// the API: is this object still listed by the VPC that owns it? A listing error counts as
// "still present" so a transient failure waits rather than declaring success.

func testAccVPCGatewayDNATRuleExistsByDescription(description string, ipVersion int) bool {
	clients := testAccClients()
	for vpcID := range testAccCreated["netactuate_vpc"] {
		rules, err := clients.V3.ListVPCDNATRules(vpcID, ipVersion)
		if err != nil {
			// testAccCreated accumulates VPC ids across every test in the run, and earlier
			// tests have already destroyed theirs, so listing one 404s. That is not a
			// failure of THIS delete. Skip a VPC that is gone rather than erroring on it.
			//
			// This only surfaced once the callers stopped discarding the error with _ =,
			// which is the argument for not discarding it.
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

func testAccVPCGatewaySNATRuleExistsByDescription(description string, ipVersion int) bool {
	clients := testAccClients()
	for vpcID := range testAccCreated["netactuate_vpc"] {
		rules, err := clients.V3.ListVPCSNATRules(vpcID, ipVersion)
		if err != nil {
			// testAccCreated accumulates VPC ids across every test in the run, and earlier
			// tests have already destroyed theirs, so listing one 404s. That is not a
			// failure of THIS delete. Skip a VPC that is gone rather than erroring on it.
			//
			// This only surfaced once the callers stopped discarding the error with _ =,
			// which is the argument for not discarding it.
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

func testAccVPCFloatingIPExistsByPTR(ptr string) bool {
	clients := testAccClients()
	for vpcID := range testAccCreated["netactuate_vpc"] {
		fips, err := clients.V3.ListVPCFloatingIPs(vpcID)
		if err != nil {
			// testAccCreated accumulates VPC ids across every test in the run, and earlier
			// tests have already destroyed theirs, so listing one 404s. That is not a
			// failure of THIS delete. Skip a VPC that is gone rather than erroring on it.
			//
			// This only surfaced once the callers stopped discarding the error with _ =,
			// which is the argument for not discarding it.
			if gona.IsV3NotFound(err) {
				continue
			}
			return true
		}
		for _, f := range fips {
			if f.PTR == ptr {
				return true
			}
		}
	}
	return false
}
