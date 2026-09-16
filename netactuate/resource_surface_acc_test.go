//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateCloudFloatingIPv4_importGrantUpdateAndOutOfBandDelete(t *testing.T) {
	name := testAccName("floating-ip")
	serverID := testAccEnvInt(t, "NETACTUATE_ACC_WRITABLE_SERVER_ID")
	vlanID := testAccEnvInt(t, "NETACTUATE_ACC_CUSTOMER_VLAN_ID")
	configOne := testAccCloudFloatingIPv4Config(name, vlanID, []int{serverID}, true)
	configEmpty := testAccCloudFloatingIPv4Config(name, vlanID, nil, true)
	var floatingID int

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_WRITABLE_SERVER_ID", "NETACTUATE_ACC_CUSTOMER_VLAN_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCloudFloatingIPv4Destroy,
		Steps: []resource.TestStep{
			{
				Config: configOne,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_cloud_floating_ipv4", "netactuate_cloud_floating_ipv4.test"),
					testAccRememberID("netactuate_cloud_floating_ipv4.test", &floatingID),
					testAccCheckCloudFloatingIPv4API("netactuate_cloud_floating_ipv4.test", name+".example.invalid"),
					testAccCheckCloudFloatingIPv4VMGrantAPI("netactuate_cloud_floating_ipv4_vm_grant.test", []int{serverID}),
				),
			},
			{
				ResourceName:      "netactuate_cloud_floating_ipv4.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:            "netactuate_cloud_floating_ipv4_vm_grant.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"revoke_existing"},
			},
			{
				Config: configEmpty,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFloatingIPv4API("netactuate_cloud_floating_ipv4.test", name+".example.invalid"),
					testAccCheckCloudFloatingIPv4VMGrantAPI("netactuate_cloud_floating_ipv4_vm_grant.test", nil),
				),
			},
			{
				PreConfig: func() {
					if floatingID == 0 {
						t.Fatal("no tracked floating IPv4 ID")
					}
					if err := testAccClients().V3.DeleteCloudFloatingIPv4(floatingID); err != nil {
						t.Fatalf("out of band floating IPv4 delete: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGone("netactuate_cloud_floating_ipv4.test"),
					testAccCheckGone("netactuate_cloud_floating_ipv4_vm_grant.test"),
				),
			},
		},
	})
}

func TestAccNetactuateCloudReverseDNS_importUpdateAndDelete(t *testing.T) {
	mbpkgID := testAccEnvInt(t, "NETACTUATE_ACC_WRITABLE_SERVER_ID")
	ipv4ID := testAccEnvInt(t, "NETACTUATE_ACC_IPV4_ADDRESS_ID")
	ipv6ID := testAccEnvInt(t, "NETACTUATE_ACC_IPV6_ADDRESS_ID")
	initial := testAccCloudReverseDNSConfig(mbpkgID, ipv4ID, ipv6ID, "initial")
	updated := testAccCloudReverseDNSConfig(mbpkgID, ipv4ID, ipv6ID, "updated")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_WRITABLE_SERVER_ID",
				"NETACTUATE_ACC_IPV4_ADDRESS_ID",
				"NETACTUATE_ACC_IPV6_ADDRESS_ID",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCloudReverseDNSDestroy,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudReverseDNSAPI("netactuate_cloud_ipv4_reverse_dns.test", false, "initial.example.invalid"),
					testAccCheckCloudReverseDNSAPI("netactuate_cloud_ipv6_reverse_dns.test", true, "initial6.example.invalid"),
				),
			},
			{
				ResourceName:      "netactuate_cloud_ipv4_reverse_dns.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_cloud_ipv6_reverse_dns.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: updated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudReverseDNSAPI("netactuate_cloud_ipv4_reverse_dns.test", false, "updated.example.invalid"),
					testAccCheckCloudReverseDNSAPI("netactuate_cloud_ipv6_reverse_dns.test", true, "updated6.example.invalid"),
				),
			},
		},
	})
}

func TestAccNetactuateCloudFirewallSetBinding_importAndOutOfBandDelete(t *testing.T) {
	name := testAccName("cloud-fw-bind")
	// This test BINDS a firewall set to the server, so it needs a server that may be written to.
	// NETACTUATE_ACC_WRITABLE_SERVER_ID is read only and points at production; using it here bound a set
	// to a live host.
	serverID := testAccEnvInt(t, "NETACTUATE_ACC_WRITABLE_SERVER_ID")
	config := testAccCloudFirewallSetBindingConfig(name, serverID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_WRITABLE_SERVER_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
					testAccCheckCloudFirewallSetBindingAPI("netactuate_cloud_firewall_set_binding.test"),
				),
			},
			{
				ResourceName:            "netactuate_cloud_firewall_set_binding.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"raw_json", "set_priority"},
			},
			{
				PreConfig: func() {
					if err := testAccUnbindCloudFirewallSetFromState(serverID); err != nil {
						t.Fatalf("out of band cloud firewall set unbind: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_cloud_firewall_set_binding.test"),
			},
		},
	})
}

func TestAccNetactuateFirewallSetDetachAllVMs_import(t *testing.T) {
	name := testAccName("fw-detach-all")
	serverID := testAccEnvInt(t, "NETACTUATE_ACC_WRITABLE_SERVER_ID")
	config := testAccFirewallSetDetachAllVMsConfig(name, serverID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_WRITABLE_SERVER_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
					testAccCheckFirewallSetHasNoVMsAPI("netactuate_firewall_set.test"),
				),
			},
			{
				ResourceName:      "netactuate_firewall_set_detach_all_vms.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

func TestAccNetactuateDedicatedServerBuy_importAndOutOfBandDelete(t *testing.T) {
	deviceID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_BUY_DEVICE_ID")
	var mbpkgID int

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DEDICATED_BUY_DEVICE_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDedicatedServerResourcesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDedicatedServerBuyConfig(deviceID),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_dedicated_server_buy", "netactuate_dedicated_server_buy.test"),
					testAccRememberID("netactuate_dedicated_server_buy.test", &mbpkgID),
					testAccCheckDedicatedServerAPI("netactuate_dedicated_server_buy.test"),
				),
			},
			{
				ResourceName:            "netactuate_dedicated_server_buy.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"device_id"},
			},
			{
				PreConfig: func() {
					if mbpkgID == 0 {
						t.Fatal("no tracked dedicated server ID")
					}
					if err := testAccClients().V2.DeleteDedicatedServer(mbpkgID, nil); err != nil {
						t.Fatalf("out of band dedicated server delete: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_dedicated_server_buy.test"),
			},
		},
	})
}

func TestAccNetactuateDedicatedServerDeployment_import(t *testing.T) {
	mbpkgID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_SERVER_ID")
	profileID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_PROFILE_ID")
	diskLayoutID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_DISK_LAYOUT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_DEDICATED_ROOT_PASSWORD")
	name := testAccName("dedicated-deploy")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_DEDICATED_SERVER_ID",
				"NETACTUATE_ACC_DEDICATED_PROFILE_ID",
				"NETACTUATE_ACC_DEDICATED_DISK_LAYOUT_ID",
				"NETACTUATE_ACC_DEDICATED_ROOT_PASSWORD",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDedicatedServerResourcesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDedicatedServerDeploymentConfig(mbpkgID, profileID, diskLayoutID, name, password),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_dedicated_server_deployment", "netactuate_dedicated_server_deployment.test"),
					testAccCheckDedicatedServerAPI("netactuate_dedicated_server_deployment.test"),
				),
			},
			{
				ResourceName:            "netactuate_dedicated_server_deployment.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"profile", "disklayout", "fqdn", "root_password"},
			},
		},
	})
}

func TestAccNetactuateDedicatedServerBuyBuild_importAndOutOfBandDelete(t *testing.T) {
	deviceID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_BUY_BUILD_DEVICE_ID")
	profileID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_PROFILE_ID")
	diskLayoutID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_DISK_LAYOUT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_DEDICATED_ROOT_PASSWORD")
	name := testAccName("dedicated-buy-build")
	var mbpkgID int

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_DEDICATED_BUY_BUILD_DEVICE_ID",
				"NETACTUATE_ACC_DEDICATED_PROFILE_ID",
				"NETACTUATE_ACC_DEDICATED_DISK_LAYOUT_ID",
				"NETACTUATE_ACC_DEDICATED_ROOT_PASSWORD",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDedicatedServerResourcesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDedicatedServerBuyBuildConfig(deviceID, profileID, diskLayoutID, name, password),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_dedicated_server_buy_build", "netactuate_dedicated_server_buy_build.test"),
					testAccRememberID("netactuate_dedicated_server_buy_build.test", &mbpkgID),
					testAccCheckDedicatedServerAPI("netactuate_dedicated_server_buy_build.test"),
				),
			},
			{
				ResourceName:            "netactuate_dedicated_server_buy_build.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"device_id", "profile", "disklayout", "fqdn", "root_password"},
			},
			{
				PreConfig: func() {
					if mbpkgID == 0 {
						t.Fatal("no tracked dedicated server ID")
					}
					if err := testAccClients().V2.DeleteDedicatedServer(mbpkgID, nil); err != nil {
						t.Fatalf("out of band dedicated server delete: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_dedicated_server_buy_build.test"),
			},
		},
	})
}

func TestAccNetactuateDedicatedServerIPv4Reverse_importUpdateAndDelete(t *testing.T) {
	mbpkgID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_SERVER_ID")
	ipv4ID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_IPV4_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DEDICATED_SERVER_ID", "NETACTUATE_ACC_DEDICATED_IPV4_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDedicatedIPv4ReverseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDedicatedIPv4ReverseConfig(mbpkgID, ipv4ID, "initial.example.invalid"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("netactuate_dedicated_server_ipv4_reverse.test", "reverse", "initial.example.invalid"),
					testAccCheckDedicatedServerAPIByID(mbpkgID),
				),
			},
			{
				ResourceName:            "netactuate_dedicated_server_ipv4_reverse.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"reverse"},
			},
			{
				Config: testAccDedicatedIPv4ReverseConfig(mbpkgID, ipv4ID, "updated.example.invalid"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("netactuate_dedicated_server_ipv4_reverse.test", "reverse", "updated.example.invalid"),
					testAccCheckDedicatedServerAPIByID(mbpkgID),
				),
			},
		},
	})
}

func TestAccNetactuateDedicatedServerPowerStatus_importPlanAndDelete(t *testing.T) {
	mbpkgID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_SERVER_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DEDICATED_SERVER_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDedicatedPowerStatusDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDedicatedPowerStatusConfig(mbpkgID, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTypedAttribute("netactuate_dedicated_server_power_status.test", "status_json", nonEmptyStringField),
					testAccCheckDedicatedPowerStatusAPI("netactuate_dedicated_server_power_status.test"),
				),
			},
			{
				ResourceName:            "netactuate_dedicated_server_power_status.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"status_json"},
			},
			{
				Config:   testAccDedicatedPowerStatusConfig(mbpkgID, true),
				PlanOnly: true,
			},
		},
	})
}

func testAccCloudFloatingIPv4Config(name string, vlanID int, mbpkgids []int, revokeExisting bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_cloud_floating_ipv4" "test" {
  ptr_domain = %q
  vlan_id    = %d
}

resource "netactuate_cloud_floating_ipv4_vm_grant" "test" {
  floating_ipv4_id = netactuate_cloud_floating_ipv4.test.floating_ipv4_id
  mbpkgids         = [%s]
  revoke_existing  = %t
}
`, name+".example.invalid", vlanID, testAccIntList(mbpkgids), revokeExisting)
}

func testAccCloudReverseDNSConfig(mbpkgID, ipv4ID, ipv6ID int, label string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_cloud_ipv4_reverse_dns" "test" {
  mbpkgid    = %d
  address_id = %d
  reverse    = %q
}

resource "netactuate_cloud_ipv6_reverse_dns" "test" {
  mbpkgid    = %d
  address_id = %d
  reverse    = %q
}
`, mbpkgID, ipv4ID, label+".example.invalid", mbpkgID, ipv6ID, label+"6.example.invalid")
}

func testAccCloudFirewallSetBindingConfig(name string, serverID int) string {
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

func testAccFirewallSetDetachAllVMsConfig(name string, serverID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_firewall_set" "test" {
  name        = %q
  description = %q
  enabled     = true
}

resource "netactuate_firewall_set_vm" "test" {
  firewall_set_id = netactuate_firewall_set.test.id
  mbpkgid         = %d
  set_priority    = 10
}

resource "netactuate_firewall_set_detach_all_vms" "test" {
  firewall_set_id = netactuate_firewall_set.test.id

  depends_on = [netactuate_firewall_set_vm.test]
}
`, name, name, serverID)
}

func testAccDedicatedServerBuyConfig(deviceID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dedicated_server_buy" "test" {
  device_id = %d
}
`, deviceID)
}

func testAccDedicatedServerDeploymentConfig(mbpkgID, profileID, diskLayoutID int, name, password string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dedicated_server_deployment" "test" {
  mbpkgid       = %d
  fqdn          = %q
  profile       = %d
  disklayout    = %d
  root_password = %q
}
`, mbpkgID, name+".example.invalid", profileID, diskLayoutID, password)
}

func testAccDedicatedServerBuyBuildConfig(deviceID, profileID, diskLayoutID int, name, password string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dedicated_server_buy_build" "test" {
  device_id     = %d
  fqdn          = %q
  profile       = %d
  disklayout    = %d
  root_password = %q
}
`, deviceID, name+".example.invalid", profileID, diskLayoutID, password)
}

func testAccDedicatedIPv4ReverseConfig(mbpkgID, ipv4ID int, reverse string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dedicated_server_ipv4_reverse" "test" {
  mbpkgid  = %d
  ipv4_id  = %d
  reverse  = %q
}
`, mbpkgID, ipv4ID, reverse)
}

func testAccDedicatedPowerStatusConfig(mbpkgID int, force bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_dedicated_server_power_status" "test" {
  mbpkgid = %d
  force   = %t
}
`, mbpkgID, force)
}

func testAccCheckCloudFloatingIPv4API(address, ptrDomain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		floatingIPs, err := testAccClients().V3.ListCloudFloatingIPv4()
		if err != nil {
			return err
		}
		for _, floatingIP := range floatingIPs {
			if floatingIP.FloatingIPv4ID != id {
				continue
			}
			if floatingIP.PTRDomain == nil || *floatingIP.PTRDomain != ptrDomain {
				return fmt.Errorf("%s API ptr_domain = %v, want %q", address, floatingIP.PTRDomain, ptrDomain)
			}
			if floatingIP.Address == "" {
				return fmt.Errorf("%s API address is empty", address)
			}
			return nil
		}
		return fmt.Errorf("%s API floating IPv4 %d not found", address, id)
	}
}

func testAccCheckCloudFloatingIPv4VMGrantAPI(address string, want []int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		vms, err := testAccClients().V3.ListCloudFloatingIPv4VMs(id)
		if err != nil {
			return err
		}
		got := make([]int, len(vms))
		for i, vm := range vms {
			got[i] = vm.MBPkgID
		}
		if !testAccSameIntSet(got, want) {
			return fmt.Errorf("%s API mbpkgids = %v, want %v", address, got, want)
		}
		return nil
	}
}

func testAccCheckCloudReverseDNSAPI(address string, ipv6 bool, reverse string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		mbpkgID, addressID, err := testAccTwoPartStateID(s, address)
		if err != nil {
			return err
		}
		if ipv6 {
			items, err := testAccClients().V2.GetServerIPv6(mbpkgID)
			if err != nil {
				return err
			}
			for _, item := range items {
				if item.ID == addressID {
					if got := stringPtrOrEmpty(item.Reverse); got != reverse {
						return fmt.Errorf("%s API reverse = %q, want %q", address, got, reverse)
					}
					return nil
				}
			}
		} else {
			items, err := testAccClients().V2.GetServerIPv4(mbpkgID)
			if err != nil {
				return err
			}
			for _, item := range items {
				if item.ID == addressID {
					if got := stringPtrOrEmpty(item.Reverse); got != reverse {
						return fmt.Errorf("%s API reverse = %q, want %q", address, got, reverse)
					}
					return nil
				}
			}
		}
		return fmt.Errorf("%s API address %d not found on server %d", address, addressID, mbpkgID)
	}
}

func testAccCheckCloudFirewallSetBindingAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		mbpkgID, firewallSetID, err := testAccTwoPartStateID(s, address)
		if err != nil {
			return err
		}
		vms, err := testAccClients().V2.GetFirewallSetRelatedVMs(mbpkgID, nil)
		if err != nil {
			return err
		}
		for _, vm := range vms {
			if vm.FirewallSetID == firewallSetID && vm.Mbpkgid == mbpkgID {
				if vm.SetPriority != 40 {
					return fmt.Errorf("%s API set_priority = %d, want 40", address, vm.SetPriority)
				}
				return nil
			}
		}
		return fmt.Errorf("%s API binding %d/%d not found", address, mbpkgID, firewallSetID)
	}
}

func testAccCheckFirewallSetHasNoVMsAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		setID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		vms, err := testAccClients().V2.GetFirewallSetVMs(setID)
		if err != nil {
			return err
		}
		if len(vms) != 0 {
			return fmt.Errorf("%s API still has %d attached VM(s)", address, len(vms))
		}
		return nil
	}
}

func testAccCheckDedicatedServerAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		mbpkgID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccCheckDedicatedServerAPIByID(mbpkgID)(s)
	}
}

func testAccCheckDedicatedServerAPIByID(mbpkgID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		server, err := testAccClients().V2.GetMetal(mbpkgID)
		if err != nil {
			return err
		}
		if server.MBPKGID != mbpkgID {
			return fmt.Errorf("dedicated server API mbpkgid = %d, want %d", server.MBPKGID, mbpkgID)
		}
		if server.Canceling != 0 {
			return fmt.Errorf("dedicated server API canceling = %d, want 0", server.Canceling)
		}
		return nil
	}
}

func testAccCheckDedicatedPowerStatusAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		mbpkgID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		status, err := testAccClients().V2.GetDedicatedServerPowerStatus(mbpkgID, nil)
		if err != nil {
			return err
		}
		if len(status) == 0 {
			return fmt.Errorf("%s API power status is empty", address)
		}
		return nil
	}
}

func testAccCheckCloudFloatingIPv4Destroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_cloud_floating_ipv4" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		floatingIPs, err := testAccClients().V3.ListCloudFloatingIPv4()
		if err != nil {
			return err
		}
		for _, floatingIP := range floatingIPs {
			if floatingIP.FloatingIPv4ID == id {
				return fmt.Errorf("cloud floating IPv4 still exists: %d", id)
			}
		}
	}
	return nil
}

func testAccCheckCloudReverseDNSDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_cloud_ipv4_reverse_dns" && rs.Type != "netactuate_cloud_ipv6_reverse_dns" {
			continue
		}
		mbpkgID, addressID, err := parseTwoPartIntID(rs.Primary.ID, "mbpkgid", "address_id")
		if err != nil {
			return err
		}
		ipv6 := rs.Type == "netactuate_cloud_ipv6_reverse_dns"
		check := testAccCheckCloudReverseDNSCleared(mbpkgID, addressID, ipv6)
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func testAccCheckDedicatedServerResourcesDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "netactuate_dedicated_server_buy", "netactuate_dedicated_server_buy_build":
			id, err := strconv.Atoi(rs.Primary.ID)
			if err != nil {
				return err
			}
			if _, err := testAccClients().V2.GetMetal(id); err == nil {
				return fmt.Errorf("%s still exists: %d", rs.Type, id)
			}
		}
	}
	return nil
}

func testAccCheckDedicatedIPv4ReverseDestroy(s *terraform.State) error {
	// The dedicated IPv4 reverse DNS endpoint has no readback and no clear operation.
	// The test can prove create and update reached the API, but destroy is state only.
	return nil
}

func testAccCheckDedicatedPowerStatusDestroy(s *terraform.State) error {
	// The power status endpoint is a read action and has no delete operation.
	return nil
}

func testAccCheckCloudReverseDNSCleared(mbpkgID, addressID int, ipv6 bool) func() error {
	return func() error {
		if ipv6 {
			items, err := testAccClients().V2.GetServerIPv6(mbpkgID)
			if err != nil {
				return err
			}
			for _, item := range items {
				if item.ID == addressID && stringPtrOrEmpty(item.Reverse) != "" {
					return fmt.Errorf("IPv6 reverse DNS still set for address %d", addressID)
				}
			}
			return nil
		}
		items, err := testAccClients().V2.GetServerIPv4(mbpkgID)
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.ID == addressID && stringPtrOrEmpty(item.Reverse) != "" {
				return fmt.Errorf("IPv4 reverse DNS still set for address %d", addressID)
			}
		}
		return nil
	}
}

func testAccUnbindCloudFirewallSetFromState(mbpkgID int) error {
	vms, err := testAccClients().V2.GetFirewallSetRelatedVMs(mbpkgID, nil)
	if err != nil {
		return err
	}
	for _, vm := range vms {
		if err := testAccClients().V2.UnbindCloudFirewallSet(mbpkgID, strconv.Itoa(vm.FirewallSetID)); err != nil {
			return err
		}
	}
	return nil
}

func testAccTwoPartStateID(s *terraform.State, address string) (int, int, error) {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return 0, 0, fmt.Errorf("not found: %s", address)
	}
	return parseTwoPartIntID(rs.Primary.ID, "left", "right")
}

func testAccIntList(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.Itoa(value)
	}
	return strings.Join(parts, ", ")
}

func testAccSameIntSet(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[int]int, len(got))
	for _, value := range got {
		seen[value]++
	}
	for _, value := range want {
		seen[value]--
		if seen[value] < 0 {
			return false
		}
	}
	return true
}
