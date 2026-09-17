//go:build acctest

package netactuate

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateSSHKeyLifecycle(t *testing.T) {
	name := testAccName("sshkey-w70")
	createConfig := testAccSSHKeyConfig(name)
	modifiedConfig := testAccSSHKeyConfig(name + "-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSSHKeyDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_sshkey.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_sshkey", "netactuate_sshkey.test"),
			testAccCheckSSHKeyAPI("netactuate_sshkey.test", name, testAccSSHPublicKey),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_sshkey", "netactuate_sshkey.test"),
			testAccCheckSSHKeyAPI("netactuate_sshkey.test", name+"-updated", testAccSSHPublicKey),
		), testAccLifecycleOptions{}),
	})
}

func TestAccNetactuateSSHKeyOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("sshkey-w70-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSSHKeyDestroy,
		Steps: []resource.TestStep{{
			Config: testAccSSHKeyResourceOnlyConfig(name),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_sshkey", "netactuate_sshkey.test"),
				testAccCheckSSHKeyAPI("netactuate_sshkey.test", name, testAccSSHPublicKey),
				testAccDeleteSSHKeyOutOfBand("netactuate_sshkey.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateTagLifecycle(t *testing.T) {
	name := testAccName("tag-w70")
	createConfig := testAccTagConfig(name, "created by Terraform acceptance test")
	modifiedConfig := testAccTagConfig(name+"-updated", "updated by Terraform acceptance test")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckTagsDestroyed(name, name+"-updated"),
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_tag.test", resource.ComposeTestCheckFunc(
			testAccCheckTagAPI("netactuate_tag.test", name, "created by Terraform acceptance test"),
		), resource.ComposeTestCheckFunc(
			testAccCheckTagAPI("netactuate_tag.test", name+"-updated", "updated by Terraform acceptance test"),
		), testAccLifecycleOptions{}),
	})
}

func TestAccNetactuateTagOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("tag-w70-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckTagsDestroyed(name),
		Steps: []resource.TestStep{{
			Config: testAccTagConfig(name, "created by Terraform acceptance test"),
			Check: resource.ComposeTestCheckFunc(
				testAccCheckTagAPI("netactuate_tag.test", name, "created by Terraform acceptance test"),
				testAccDeleteTagOutOfBand("netactuate_tag.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateDNSZoneLifecycle(t *testing.T) {
	zoneName := testAccDNSZoneName(t)
	modifiedName := testAccDNSZoneName(t)
	createConfig := testAccDNSZoneConfig(zoneName)
	modifiedConfig := testAccDNSZoneConfig(modifiedName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDNSZoneDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_dns_zone.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
			testAccCheckDNSZoneFromAPI("netactuate_dns_zone.test", zoneName, "NATIVE"),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
			testAccCheckDNSZoneFromAPI("netactuate_dns_zone.test", modifiedName, "NATIVE"),
		), testAccLifecycleOptions{}),
	})
}

func TestAccNetactuateDNSZoneOutOfBandDeleteIdempotent(t *testing.T) {
	zoneName := testAccDNSZoneName(t)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDNSZoneDestroy,
		Steps: []resource.TestStep{{
			Config: testAccDNSZoneResourceOnlyConfig(zoneName),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
				testAccCheckDNSZoneFromAPI("netactuate_dns_zone.test", zoneName, "NATIVE"),
				testAccDeleteDNSZoneOutOfBand("netactuate_dns_zone.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateDNSRecordLifecycle(t *testing.T) {
	zoneName := testAccDNSZoneName(t)
	createConfig := testAccDNSRecordConfig(zoneName, "192.0.2.20", 3600)
	modifiedConfig := testAccDNSRecordConfig(zoneName, "192.0.2.21", 7200)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDNSRecordAndZoneDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_dns_record.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
			testAccTrackFromState("netactuate_dns_record", "netactuate_dns_record.test"),
			testAccCheckDNSRecordFromAPI("netactuate_dns_record.test", zoneName, "www", "A", "192.0.2.20", 3600, 0),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
			testAccTrackFromState("netactuate_dns_record", "netactuate_dns_record.test"),
			testAccCheckDNSRecordFromAPI("netactuate_dns_record.test", zoneName, "www", "A", "192.0.2.21", 7200, 0),
		), testAccLifecycleOptions{}),
	})
}

func TestAccNetactuateDNSRecordOutOfBandDeleteIdempotent(t *testing.T) {
	zoneName := testAccDNSZoneName(t)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDNSRecordAndZoneDestroy,
		Steps: []resource.TestStep{{
			Config: testAccDNSRecordConfig(zoneName, "192.0.2.22", 3600),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
				testAccTrackFromState("netactuate_dns_record", "netactuate_dns_record.test"),
				testAccCheckDNSRecordFromAPI("netactuate_dns_record.test", zoneName, "www", "A", "192.0.2.22", 3600, 0),
				testAccDeleteDNSRecordOutOfBand("netactuate_dns_record.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateSecretListLifecycle(t *testing.T) {
	name := testAccName("secret-list-w70")
	createConfig := testAccSecretListOnlyConfig(name)
	modifiedConfig := testAccSecretListOnlyConfig(name + "-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSecretListValueDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_secret_list.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_secret_list", "netactuate_secret_list.test"),
			testAccCheckSecretListAPI("netactuate_secret_list.test", name),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_secret_list", "netactuate_secret_list.test"),
			testAccCheckSecretListAPI("netactuate_secret_list.test", name+"-updated"),
		), testAccLifecycleOptions{}),
	})
}

func TestAccNetactuateSecretListOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("secret-list-w70-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSecretListValueDestroy,
		Steps: []resource.TestStep{{
			Config: testAccSecretListOnlyConfig(name),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_secret_list", "netactuate_secret_list.test"),
				testAccCheckSecretListAPI("netactuate_secret_list.test", name),
				testAccDeleteSecretListOutOfBand("netactuate_secret_list.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateSecretListValueLifecycle(t *testing.T) {
	name := testAccName("secret-list-value-w70")
	key := testAccName("secret-key-w70")
	createConfig := testAccSecretListValueConfig(name, key, testAccSecretValueInitial)
	modifiedConfig := testAccSecretListValueConfig(name+"-updated", key+"-updated", testAccSecretValueUpdated)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSecretListValueDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_secret_list_value.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_secret_list", "netactuate_secret_list.test"),
			testAccTrackFromState("netactuate_secret_list_value", "netactuate_secret_list_value.test"),
			testAccCheckSecretListValueAPI("netactuate_secret_list.test", "netactuate_secret_list_value.test", key, testAccSecretValueInitial),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_secret_list", "netactuate_secret_list.test"),
			testAccTrackFromState("netactuate_secret_list_value", "netactuate_secret_list_value.test"),
			testAccCheckSecretListValueAPI("netactuate_secret_list.test", "netactuate_secret_list_value.test", key+"-updated", testAccSecretValueUpdated),
		), testAccLifecycleOptions{ImportStateIdFunc: testAccSecretListValueImportID("netactuate_secret_list.test", "netactuate_secret_list_value.test")}),
	})
}

func TestAccNetactuateSecretListValueOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("secret-list-value-w70-oob")
	key := testAccName("secret-key-w70-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSecretListValueDestroy,
		Steps: []resource.TestStep{{
			Config: testAccSecretListValueConfig(name, key, testAccSecretValueInitial),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_secret_list", "netactuate_secret_list.test"),
				testAccTrackFromState("netactuate_secret_list_value", "netactuate_secret_list_value.test"),
				testAccCheckSecretListValueAPI("netactuate_secret_list.test", "netactuate_secret_list_value.test", key, testAccSecretValueInitial),
				testAccDeleteSecretListValueOutOfBand("netactuate_secret_list.test", "netactuate_secret_list_value.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateFirewallSetLifecycle(t *testing.T) {
	name := testAccName("fw-set-w70")
	createConfig := testAccFirewallSetOnlyConfig(name, true)
	modifiedConfig := testAccFirewallSetOnlyConfig(name+"-updated", false)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_firewall_set.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
			testAccCheckFirewallSetAPI("netactuate_firewall_set.test", name, true),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
			testAccCheckFirewallSetAPI("netactuate_firewall_set.test", name+"-updated", false),
		), testAccLifecycleOptions{}),
	})
}

func TestAccNetactuateFirewallSetOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("fw-set-w70-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: []resource.TestStep{{
			Config: testAccFirewallSetOnlyConfig(name, true),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
				testAccCheckFirewallSetAPI("netactuate_firewall_set.test", name, true),
				testAccDeleteFirewallSetOutOfBand("netactuate_firewall_set.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateFirewallRuleLifecycle(t *testing.T) {
	name := testAccName("fw-rule-w70")
	createConfig := testAccFirewallSetRulesConfig(name, []testAccFirewallRule{
		{Name: "alpha", Priority: 10, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
	})
	modifiedConfig := testAccFirewallSetRulesConfig(name, []testAccFirewallRule{
		{Name: "alpha", Priority: 20, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_firewall_rule.alpha", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
			testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
				{Name: "alpha", Priority: 10, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
			}, []string{"alpha"}),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
			testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
				{Name: "alpha", Priority: 20, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
			}, []string{"alpha"}),
		), testAccLifecycleOptions{
			ImportStateIdFunc:       testAccFirewallRuleImportID("netactuate_firewall_set.test", "netactuate_firewall_rule.alpha"),
			ImportStateVerifyIgnore: []string{"sync_after_publish"},
		}),
	})
}

func TestAccNetactuateFirewallRuleOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("fw-rule-w70-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: []resource.TestStep{{
			Config: testAccFirewallSetRulesConfig(name, []testAccFirewallRule{
				{Name: "alpha", Priority: 10, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
			}),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
				testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
					{Name: "alpha", Priority: 10, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
				}, []string{"alpha"}),
				testAccDeleteFirewallRuleOutOfBand(name+" alpha"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateAccessControlSubnetLifecycle(t *testing.T) {
	name := testAccName("uac-subnet-w70")
	createConfig := testAccAccessControlSubnetConfig(name, "192.0.2.0/24")
	modifiedConfig := testAccAccessControlSubnetConfig(name+"-updated", "198.51.100.0/24")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_ALLOW_SUBNET_WRITE") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAccessControlSubnetDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_access_control_subnet.test", resource.ComposeTestCheckFunc(
			testAccCheckAccessControlSubnetAPI("netactuate_access_control_subnet.test", name, "192.0.2.0/24"),
		), resource.ComposeTestCheckFunc(
			testAccCheckAccessControlSubnetAPI("netactuate_access_control_subnet.test", name+"-updated", "198.51.100.0/24"),
		), testAccLifecycleOptions{}),
	})
}

func TestAccNetactuateAccessControlSubnetOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("uac-subnet-w70-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_ALLOW_SUBNET_WRITE") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAccessControlSubnetDestroy,
		Steps: []resource.TestStep{{
			Config: testAccAccessControlSubnetConfig(name, "192.0.2.0/24"),
			Check: resource.ComposeTestCheckFunc(
				testAccCheckAccessControlSubnetAPI("netactuate_access_control_subnet.test", name, "192.0.2.0/24"),
				testAccDeleteAccessControlSubnetOutOfBand("netactuate_access_control_subnet.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateSSLCertificateLifecycle(t *testing.T) {
	name := testAccName("ssl-w70")
	createConfig := testAccSSLCertificateConfigWithDescription(name, "nah acceptance certificate")
	modifiedConfig := testAccSSLCertificateConfigWithDescription(name+"-updated", "nah acceptance certificate updated")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSSLCertificateDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_ssl_certificate.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_ssl_certificate", "netactuate_ssl_certificate.test"),
			testAccCheckSSLCertificateAPI("netactuate_ssl_certificate.test", name, "nah acceptance certificate"),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_ssl_certificate", "netactuate_ssl_certificate.test"),
			testAccCheckSSLCertificateAPI("netactuate_ssl_certificate.test", name+"-updated", "nah acceptance certificate updated"),
		), testAccLifecycleOptions{ImportStateVerifyIgnore: []string{"certificate", "private_key"}}),
	})
}

func TestAccNetactuateSSLCertificateOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("ssl-w70-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSSLCertificateDestroy,
		Steps: []resource.TestStep{{
			Config: testAccSSLCertificateConfigWithDescription(name, "nah acceptance certificate"),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_ssl_certificate", "netactuate_ssl_certificate.test"),
				testAccCheckSSLCertificateAPI("netactuate_ssl_certificate.test", name, "nah acceptance certificate"),
				testAccDeleteSSLCertificateOutOfBand("netactuate_ssl_certificate.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateVPCLifecycle(t *testing.T) {
	name := testAccName("vpc-w70")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	createConfig := testAccVPCConfigID(name, locationID)
	modifiedConfig := testAccVPCConfigIDWithDescription(name+"-updated", name+" description updated", locationID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_vpc.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccCheckVPCFromAPI("netactuate_vpc.test", name, locationID),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccCheckVPCFromAPI("netactuate_vpc.test", name+"-updated", locationID),
		), testAccLifecycleOptions{ImportStateVerifyIgnore: []string{"enable_default_snat"}}),
	})
}

func TestAccNetactuateVPCOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("vpc-w70-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{{
			Config: testAccVPCConfigID(name, locationID),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
				testAccCheckVPCFromAPI("netactuate_vpc.test", name, locationID),
				testAccDeleteVPCOutOfBand("netactuate_vpc.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateVPCNameserversLifecycle(t *testing.T) {
	name := testAccName("vpc-ns-w70")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	createConfig := testAccVPCNameserversConfig(name, locationID, []string{"192.0.2.53"}, []string{})
	modifiedConfig := testAccVPCNameserversConfig(name, locationID, []string{"192.0.2.54", "198.51.100.53"}, []string{})

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_vpc_nameservers.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccCheckVPCNameserversResourceAPI("netactuate_vpc_nameservers.test", []string{"192.0.2.53"}, []string{}),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccCheckVPCNameserversResourceAPI("netactuate_vpc_nameservers.test", []string{"192.0.2.54", "198.51.100.53"}, []string{}),
		), testAccLifecycleOptions{}),
	})
}

func TestAccNetactuateVPCNameserversOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("vpc-ns-w70-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{{
			Config: testAccVPCNameserversConfig(name, locationID, []string{"192.0.2.53"}, []string{}),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
				testAccCheckVPCNameserversResourceAPI("netactuate_vpc_nameservers.test", []string{"192.0.2.53"}, []string{}),
				testAccDeleteVPCNameserversOutOfBand("netactuate_vpc_nameservers.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateVPCGatewayFirewallRuleLifecycle(t *testing.T) {
	name := testAccName("vpc-fw-w70")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	createConfig := testAccVPCGatewayFirewallRuleSingleConfig(name, locationID, "TCP", "203.0.113.0/24", 8443, 8444)
	modifiedConfig := testAccVPCGatewayFirewallRuleSingleConfig(name, locationID, "UDP", "198.51.100.0/24", 5353, 5354)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_vpc_gateway_firewall_rule.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccCheckVPCGatewayFirewallRuleAPI("netactuate_vpc_gateway_firewall_rule.test", testAccVPCGatewayFirewallWant{
				Description: name,
				Direction:   "inbound",
				Protocol:    "TCP",
				Network:     "203.0.113.0/24",
				PortStart:   8443,
				PortEnd:     8444,
			}),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
			testAccCheckVPCGatewayFirewallRuleAPI("netactuate_vpc_gateway_firewall_rule.test", testAccVPCGatewayFirewallWant{
				Description: name,
				Direction:   "inbound",
				Protocol:    "UDP",
				Network:     "198.51.100.0/24",
				PortStart:   5353,
				PortEnd:     5354,
			}),
		), testAccLifecycleOptions{ImportStateIdFunc: testAccVPCGatewayFirewallImportID("netactuate_vpc_gateway_firewall_rule.test")}),
	})
}

func TestAccNetactuateVPCGatewayFirewallRuleOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("vpc-fw-w70-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{{
			Config: testAccVPCGatewayFirewallRuleSingleConfig(name, locationID, "TCP", "203.0.113.0/24", 8443, 8444),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
				testAccCheckVPCGatewayFirewallRuleAPI("netactuate_vpc_gateway_firewall_rule.test", testAccVPCGatewayFirewallWant{
					Description: name,
					Direction:   "inbound",
					Protocol:    "TCP",
					Network:     "203.0.113.0/24",
					PortStart:   8443,
					PortEnd:     8444,
				}),
				testAccDeleteVPCGatewayFirewallRuleOutOfBand("netactuate_vpc_gateway_firewall_rule.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateVPCGatewayFirewallRuleOutboundDirectionRejected(t *testing.T) {
	name := testAccName("vpc-fw-outbound-w70")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{{
			Config:             testAccVPCGatewayFirewallRuleOutboundConfig(name, locationID),
			ExpectError:        regexp.MustCompile(`expected direction to be one of \["inbound"\]`),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func testAccDeleteSSHKeyOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V2.DeleteSSHKey(id)
	}
}

func testAccDeleteTagOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V2.DeleteTag(id)
	}
}

func testAccDeleteDNSZoneOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V2.DeleteZone(id)
	}
}

func testAccDeleteDNSRecordOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V2.DeleteRecord(id)
	}
}

func testAccDeleteSecretListOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V2.DeleteSecretList(id)
	}
}

func testAccDeleteSecretListValueOutOfBand(listAddress, valueAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		listID, err := testAccID(s, listAddress)
		if err != nil {
			return err
		}
		valueID, err := testAccID(s, valueAddress)
		if err != nil {
			return err
		}
		return testAccClients().V2.DeleteSecretListValue(listID, valueID)
	}
}

func testAccDeleteFirewallSetOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V2.DeleteFirewallSet(id)
	}
}

func testAccDeleteFirewallRuleOutOfBand(adminComment string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return testAccDeleteFirewallRuleByAdminComment(adminComment)
	}
}

func testAccDeleteAccessControlSubnetOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccStateID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V2.DeleteAccessControlSubnet(id)
	}
}

func testAccDeleteSSLCertificateOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteSSLCertificate(id)
	}
}

func testAccDeleteVPCOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteVPC(id)
	}
}

func testAccDeleteVPCNameserversOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		_, err = testAccClients().V3.ReplaceVPCNameservers(vpcID, &gona.ReplaceVPCNameserversRequest{Nameservers: []gona.VPCNameserver{}})
		return err
	}
}

func testAccDeleteVPCGatewayFirewallRuleOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, ruleID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		if err := testAccClients().V3.DeleteVPCFirewallRule(vpcID, ruleID); err != nil {
			return err
		}
		return testAccClients().V3.ApplyVPCFirewallChanges(vpcID)
	}
}

func testAccCheckAccessControlSubnetAPI(address, wantLabel, wantSubnet string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccStateID(s, address)
		if err != nil {
			return err
		}
		subnet, err := testAccClients().V2.GetAccessControlSubnet(id)
		if err != nil {
			return err
		}
		if subnet.Label != wantLabel {
			return fmt.Errorf("access control subnet %s label = %q, want %q", id, subnet.Label, wantLabel)
		}
		if subnet.Subnet != wantSubnet {
			return fmt.Errorf("access control subnet %s subnet = %q, want %q", id, subnet.Subnet, wantSubnet)
		}
		return nil
	}
}

func testAccCheckAccessControlSubnetDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_access_control_subnet" {
			continue
		}
		if _, err := testAccClients().V2.GetAccessControlSubnet(rs.Primary.ID); err == nil {
			return fmt.Errorf("access control subnet still exists: %s", rs.Primary.ID)
		} else if !gona.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func testAccFirewallRuleImportID(setAddress, ruleAddress string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		// The firewall rule ID is already the composite "{set_id}/{rule_id}" that the
		// resource importer expects, so return it directly rather than reparsing it.
		return testAccStateID(s, ruleAddress)
	}
}

func testAccSecretListOnlyConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_secret_list" "test" {
  name = %q
}
`, name)
}

func testAccFirewallSetOnlyConfig(name string, enabled bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_firewall_set" "test" {
  name        = %q
  description = %q
  enabled     = %t
}
`, name, name, enabled)
}

func testAccSSLCertificateConfigWithDescription(name, description string) string {
	certificate, privateKey := testAccSelfSignedCertificate()
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_ssl_certificate" "test" {
  name        = %q
  description = %q
  certificate = <<EOT
%sEOT
  private_key = <<EOT
%sEOT
}
`, name, description, certificate, privateKey)
}

func testAccVPCConfigIDWithDescription(label, description string, locationID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}
`, label, description, locationID)
}

func testAccVPCGatewayFirewallRuleSingleConfig(name string, locationID int, protocol, network string, portStart, portEnd int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_gateway_firewall_rule" "test" {
  vpc_id      = netactuate_vpc.test.vpc_id
  ip_version  = 4
  direction   = "inbound"
  protocol    = %q
  description = %q
  network     = %q
  port_start  = %d
  port_end    = %d
}
`, name, name, locationID, protocol, name, network, portStart, portEnd)
}

func testAccVPCGatewayFirewallRuleOutboundConfig(name string, locationID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_gateway_firewall_rule" "test" {
  vpc_id      = netactuate_vpc.test.vpc_id
  ip_version  = 4
  direction   = "outbound"
  protocol    = "UDP"
  description = %q
  network     = "198.51.100.0/24"
  port_start  = 5353
  port_end    = 5354
}
`, name, name, locationID, name)
}

func testAccSSHKeyResourceOnlyConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_sshkey" "test" {
  name = %q
  key  = %q
}
`, name, testAccSSHPublicKey)
}

func testAccDNSZoneResourceOnlyConfig(zoneName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dns_zone" "test" {
  name = %q
  type = "NATIVE"
}
`, zoneName)
}
