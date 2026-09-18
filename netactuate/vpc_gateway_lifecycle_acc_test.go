//go:build acctest

package netactuate

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateVPCGatewayDNATRuleLifecycle(t *testing.T) {
	name := testAccName("vpc-dnat-lifecycle")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	createConfig := testAccVPCGatewayLifecycleDNATConfig(name, locationID, false)
	modifiedConfig := testAccVPCGatewayLifecycleDNATConfig(name, locationID, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCGatewayDNATRuleDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_vpc_gateway_dnat_rule.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccCheckVPCGatewayDNATRuleAPI("netactuate_vpc_gateway_dnat_rule.test", testAccVPCGatewayDNATWant{
				Description:          name + " dnat initial",
				Protocol:             "TCP",
				MatchAddress:         "203.0.113.10",
				MatchPortStart:       8443,
				MatchPortEnd:         8444,
				TranslationAddress:   "192.0.2.10",
				TranslationPortStart: 443,
				TranslationPortEnd:   444,
			}),
		), testAccCheckVPCGatewayDNATRuleAPI("netactuate_vpc_gateway_dnat_rule.test", testAccVPCGatewayDNATWant{
			Description:          name + " dnat updated",
			Protocol:             "UDP",
			MatchAddress:         "203.0.113.11",
			MatchPortStart:       5353,
			MatchPortEnd:         5354,
			TranslationAddress:   "192.0.2.11",
			TranslationPortStart: 53,
			TranslationPortEnd:   54,
		}), testAccLifecycleOptions{
			// Likely provider bug: DNAT state IDs are vpcId/ruleId, but the importer
			// currently requires vpcId/ruleId/ipVersion.
			ImportStateIdFunc: testAccVPCGatewayRuleImportID("netactuate_vpc_gateway_dnat_rule.test"),
		}),
	})
}

func TestAccNetactuateVPCGatewayDNATRuleOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("vpc-dnat-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCGatewayDNATRuleDestroy,
		Steps: []resource.TestStep{{
			Config: testAccVPCGatewayLifecycleDNATConfig(name, locationID, false),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
				testAccCheckVPCGatewayDNATRuleAPI("netactuate_vpc_gateway_dnat_rule.test", testAccVPCGatewayDNATWant{
					Description:          name + " dnat initial",
					Protocol:             "TCP",
					MatchAddress:         "203.0.113.10",
					MatchPortStart:       8443,
					MatchPortEnd:         8444,
					TranslationAddress:   "192.0.2.10",
					TranslationPortStart: 443,
					TranslationPortEnd:   444,
				}),
				testAccDeleteVPCGatewayDNATRuleOutOfBand("netactuate_vpc_gateway_dnat_rule.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateVPCGatewaySNATRuleLifecycle(t *testing.T) {
	name := testAccName("vpc-snat-lifecycle")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	createConfig := testAccVPCGatewayLifecycleSNATConfig(name, locationID, false)
	modifiedConfig := testAccVPCGatewayLifecycleSNATConfig(name, locationID, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCGatewaySNATRuleDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_vpc_gateway_snat_rule.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccCheckVPCGatewaySNATRuleAPI("netactuate_vpc_gateway_snat_rule.test", testAccVPCGatewaySNATWant{
				Description:             name + " snat initial",
				Protocol:                "UDP",
				MatchInternalCIDR:       "192.0.2.0/24",
				TranslationAddressStart: "198.51.100.10",
				TranslationAddressEnd:   "198.51.100.11",
				TranslationPortStart:    5300,
				TranslationPortEnd:      5301,
			}),
		), testAccCheckVPCGatewaySNATRuleAPI("netactuate_vpc_gateway_snat_rule.test", testAccVPCGatewaySNATWant{
			Description:             name + " snat updated",
			Protocol:                "TCP",
			MatchInternalCIDR:       "192.0.2.128/25",
			TranslationAddressStart: "198.51.100.20",
			TranslationAddressEnd:   "198.51.100.21",
			TranslationPortStart:    5400,
			TranslationPortEnd:      5401,
		}), testAccLifecycleOptions{
			// Likely provider bug: SNAT state IDs are vpcId/ruleId, but the importer
			// currently requires vpcId/ruleId/ipVersion.
			ImportStateIdFunc: testAccVPCGatewayRuleImportID("netactuate_vpc_gateway_snat_rule.test"),
		}),
	})
}

func TestAccNetactuateVPCGatewaySNATRuleOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("vpc-snat-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCGatewaySNATRuleDestroy,
		Steps: []resource.TestStep{{
			Config: testAccVPCGatewayLifecycleSNATConfig(name, locationID, false),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
				testAccCheckVPCGatewaySNATRuleAPI("netactuate_vpc_gateway_snat_rule.test", testAccVPCGatewaySNATWant{
					Description:             name + " snat initial",
					Protocol:                "UDP",
					MatchInternalCIDR:       "192.0.2.0/24",
					TranslationAddressStart: "198.51.100.10",
					TranslationAddressEnd:   "198.51.100.11",
					TranslationPortStart:    5300,
					TranslationPortEnd:      5301,
				}),
				testAccDeleteVPCGatewaySNATRuleOutOfBand("netactuate_vpc_gateway_snat_rule.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateVPCBackendTemplateLifecycle(t *testing.T) {
	name := testAccName("bt-life")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	createConfig := testAccVPCGatewayLifecycleBackendTemplateConfig(name, locationID, false)
	modifiedConfig := testAccVPCGatewayLifecycleBackendTemplateConfig(name, locationID, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCBackendTemplateDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_vpc_backend_template.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccTrackBackendTemplateFromState("netactuate_vpc_backend_template.test"),
			testAccCheckVPCBackendTemplateAPI("netactuate_vpc_backend_template.test", name+"-initial", 1),
		), testAccCheckVPCBackendTemplateAPI("netactuate_vpc_backend_template.test", name+"-updated", 2), testAccLifecycleOptions{
			ImportStateIdFunc: testAccVPCGatewayLifecycleImportID("netactuate_vpc_backend_template.test"),
		}),
	})
}

func TestAccNetactuateVPCBackendTemplateOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("bt-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCBackendTemplateDestroy,
		Steps: []resource.TestStep{{
			Config: testAccVPCGatewayLifecycleBackendTemplateConfig(name, locationID, false),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
				testAccTrackBackendTemplateFromState("netactuate_vpc_backend_template.test"),
				testAccCheckVPCBackendTemplateAPI("netactuate_vpc_backend_template.test", name+"-initial", 1),
				testAccDeleteVPCBackendTemplateOutOfBand("netactuate_vpc_backend_template.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateVPCBackendTemplateBackendsLifecycle(t *testing.T) {
	name := testAccName("btb-life")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	createConfig := testAccVPCGatewayLifecycleBackendTemplateBackendsConfig(name, locationID, false)
	modifiedConfig := testAccVPCGatewayLifecycleBackendTemplateBackendsConfig(name, locationID, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCBackendTemplateDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_vpc_backend_template_backends.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccTrackBackendTemplateFromState("netactuate_vpc_backend_template.test"),
			testAccCheckVPCBackendTemplateBackendsAPI("netactuate_vpc_backend_template_backends.test", []string{"nah-backend-one", "nah-backend-two"}),
		), testAccCheckVPCBackendTemplateBackendsAPI("netactuate_vpc_backend_template_backends.test", []string{"nah-backend-three"}), testAccLifecycleOptions{
			ImportStateIdFunc: testAccVPCGatewayLifecycleImportID("netactuate_vpc_backend_template_backends.test"),
		}),
	})
}

func TestAccNetactuateVPCBackendTemplateBackendsOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("btb-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCBackendTemplateDestroy,
		Steps: []resource.TestStep{{
			Config: testAccVPCGatewayLifecycleBackendTemplateBackendsConfig(name, locationID, false),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
				testAccTrackBackendTemplateFromState("netactuate_vpc_backend_template.test"),
				testAccCheckVPCBackendTemplateBackendsAPI("netactuate_vpc_backend_template_backends.test", []string{"nah-backend-one", "nah-backend-two"}),
				testAccDeleteVPCBackendTemplateBackendsOutOfBand("netactuate_vpc_backend_template_backends.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateVPCFloatingIPLifecycle(t *testing.T) {
	name := testAccName("vpc-fip-lifecycle")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	createConfig := testAccVPCGatewayLifecycleFloatingIPConfig(name, locationID, false)
	modifiedConfig := testAccVPCGatewayLifecycleFloatingIPConfig(name, locationID, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCFloatingIPDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_vpc_floating_ip.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccCheckVPCFloatingIPAPI("netactuate_vpc_floating_ip.test", name+"-initial.example.invalid"),
		), testAccCheckVPCFloatingIPAPI("netactuate_vpc_floating_ip.test", name+"-updated.example.invalid"), testAccLifecycleOptions{
			ImportStateIdFunc: testAccVPCGatewayLifecycleImportID("netactuate_vpc_floating_ip.test"),
		}),
	})
}

func TestAccNetactuateVPCFloatingIPOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("vpc-fip-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCFloatingIPDestroy,
		Steps: []resource.TestStep{{
			Config: testAccVPCGatewayLifecycleFloatingIPConfig(name, locationID, false),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
				testAccCheckVPCFloatingIPAPI("netactuate_vpc_floating_ip.test", name+"-initial.example.invalid"),
				testAccDeleteVPCFloatingIPOutOfBand("netactuate_vpc_floating_ip.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateVPCSSHKeyLifecycle(t *testing.T) {
	name := testAccName("vpc-ssh-life")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	createConfig := testAccVPCGatewayLifecycleSSHKeyConfig(name, locationID, true)
	modifiedConfig := testAccVPCGatewayLifecycleSSHKeyConfig(name, locationID, false)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCSSHKeyDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_vpc_ssh_key.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccTrackFromState("netactuate_sshkey", "netactuate_sshkey.test"),
			testAccCheckVPCSSHKeyAPIEnabled("netactuate_vpc_ssh_key.test", name, true),
		), testAccCheckVPCSSHKeyAPIEnabled("netactuate_vpc_ssh_key.test", name, false), testAccLifecycleOptions{
			ImportStateIdFunc: testAccVPCGatewayLifecycleImportID("netactuate_vpc_ssh_key.test"),
		}),
	})
}

func TestAccNetactuateVPCSSHKeyOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("vpc-ssh-key-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCSSHKeyDestroy,
		Steps: []resource.TestStep{{
			Config: testAccVPCGatewayLifecycleSSHKeyConfig(name, locationID, true),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
				testAccTrackFromState("netactuate_sshkey", "netactuate_sshkey.test"),
				testAccCheckVPCSSHKeyAPIEnabled("netactuate_vpc_ssh_key.test", name, true),
				testAccDeleteVPCSSHKeyOutOfBand("netactuate_vpc_ssh_key.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func testAccVPCGatewayLifecycleImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return testAccStateID(s, address)
	}
}

func testAccDeleteVPCGatewayDNATRuleOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, ruleID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		if err := testAccClients().V3.DeleteVPCDNATRule(vpcID, ruleID); err != nil {
			return err
		}
		return testAccClients().V3.ApplyVPCDNATChanges(vpcID)
	}
}

func testAccDeleteVPCGatewaySNATRuleOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, ruleID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		if err := testAccClients().V3.DeleteVPCSNATRule(vpcID, ruleID); err != nil {
			return err
		}
		return testAccClients().V3.ApplyVPCSNATChanges(vpcID)
	}
}

func testAccDeleteVPCBackendTemplateOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, templateID, err := testAccBackendTemplateID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteVPCBackendTemplate(vpcID, templateID)
	}
}

func testAccDeleteVPCBackendTemplateBackendsOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, templateID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		_, err = testAccClients().V3.ReplaceVPCBackends(vpcID, templateID, &gona.ReplaceVPCBackendsRequest{BackendHosts: []gona.VPCBackend{}})
		return err
	}
}

func testAccDeleteVPCFloatingIPOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, fipID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteVPCFloatingIP(vpcID, fipID)
	}
}

func testAccDeleteVPCSSHKeyOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, sshKeyID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteVPCSSHKey(vpcID, sshKeyID)
	}
}

func testAccCheckVPCSSHKeyAPIEnabled(address, name string, enabled bool) resource.TestCheckFunc {
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
		if key.IsEnabled() != enabled {
			return fmt.Errorf("%s API enabled = %t, want %t", address, key.IsEnabled(), enabled)
		}
		if key.PublicKey == "" || key.Fingerprint == "" {
			return fmt.Errorf("%s API key detail missing public key or fingerprint", address)
		}
		return nil
	}
}

func testAccVPCGatewayLifecycleDNATConfig(name string, locationID int, updated bool) string {
	protocol := "TCP"
	description := name + " dnat initial"
	matchAddress := "203.0.113.10"
	matchPortStart := 8443
	matchPortEnd := 8444
	translationAddress := "192.0.2.10"
	translationPortStart := 443
	translationPortEnd := 444
	if updated {
		protocol = "UDP"
		description = name + " dnat updated"
		matchAddress = "203.0.113.11"
		matchPortStart = 5353
		matchPortEnd = 5354
		translationAddress = "192.0.2.11"
		translationPortStart = 53
		translationPortEnd = 54
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_gateway_dnat_rule" "test" {
  vpc_id                 = netactuate_vpc.test.vpc_id
  ip_version             = 4
  protocol               = %q
  description            = %q
  match_address          = %q
  match_port_start       = %d
  match_port_end         = %d
  translation_address    = %q
  translation_port_start = %d
  translation_port_end   = %d
}
`, name, name, locationID, protocol, description, matchAddress, matchPortStart, matchPortEnd, translationAddress, translationPortStart, translationPortEnd)
}

func testAccVPCGatewayLifecycleSNATConfig(name string, locationID int, updated bool) string {
	protocol := "UDP"
	description := name + " snat initial"
	matchCIDR := "192.0.2.0/24"
	translationAddressStart := "198.51.100.10"
	translationAddressEnd := "198.51.100.11"
	translationPortStart := 5300
	translationPortEnd := 5301
	if updated {
		protocol = "TCP"
		description = name + " snat updated"
		matchCIDR = "192.0.2.128/25"
		translationAddressStart = "198.51.100.20"
		translationAddressEnd = "198.51.100.21"
		translationPortStart = 5400
		translationPortEnd = 5401
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_gateway_snat_rule" "test" {
  vpc_id                    = netactuate_vpc.test.vpc_id
  ip_version                = 4
  protocol                  = %q
  description               = %q
  match_internal_cidr       = %q
  translation_address_start = %q
  translation_address_end   = %q
  translation_port_start    = %d
  translation_port_end      = %d
}
`, name, name, locationID, protocol, description, matchCIDR, translationAddressStart, translationAddressEnd, translationPortStart, translationPortEnd)
}

func testAccVPCGatewayLifecycleBackendTemplateConfig(name string, locationID int, updated bool) string {
	templateName := name + "-initial"
	backendHosts := `
  backend_host {
    name    = "nah-backend-one"
    address = "192.0.2.10"
  }
`
	if updated {
		templateName = name + "-updated"
		backendHosts = `
  backend_host {
    name    = "nah-backend-one"
    address = "192.0.2.10"
  }

  backend_host {
    name    = "nah-backend-two"
    address = "192.0.2.11"
  }
`
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_backend_template" "test" {
  vpc_id      = netactuate_vpc.test.vpc_id
  name        = %q
  description = "acceptance backend template"
%s}
`, name, name, locationID, templateName, backendHosts)
}

func testAccVPCGatewayLifecycleBackendTemplateBackendsConfig(name string, locationID int, updated bool) string {
	backendHosts := `
  backend_host {
    name    = "nah-backend-one"
    address = "192.0.2.10"
  }

  backend_host {
    name    = "nah-backend-two"
    address = "192.0.2.11"
  }
`
	if updated {
		backendHosts = `
  backend_host {
    name    = "nah-backend-three"
    address = "192.0.2.12"
  }
`
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_backend_template" "test" {
  vpc_id      = netactuate_vpc.test.vpc_id
  name        = %q
  description = "acceptance backend template"
}

resource "netactuate_vpc_backend_template_backends" "test" {
  vpc_id              = netactuate_vpc.test.vpc_id
  backend_template_id = netactuate_vpc_backend_template.test.backend_template_id
%s}
`, name, name, locationID, name, backendHosts)
}

func testAccVPCGatewayLifecycleFloatingIPConfig(name string, locationID int, updated bool) string {
	ptr := name + "-initial.example.invalid"
	if updated {
		ptr = name + "-updated.example.invalid"
	}
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
`, name, name, locationID, ptr)
}

func testAccVPCGatewayLifecycleSSHKeyConfig(name string, locationID int, enabled bool) string {
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
  enabled    = %t
}
`, name, testAccSSHPublicKey, name, name, locationID, enabled)
}
