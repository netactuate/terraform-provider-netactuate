//go:build acctest

package netactuate

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateOIDCClientLifecycle(t *testing.T) {
	label := testAccName("oidc-client-lifecycle")
	updatedLabel := label + "-updated"
	createConfig := testAccOIDCClientConfig(label, "initial", 300, "https://api.example.com/oidc-audience", false, true)
	modifiedConfig := testAccOIDCClientConfig(updatedLabel, "updated", 600, "https://api.example.com/oidc-audience-updated", true, true)

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: true,
		CheckDestroy:              testAccCheckOIDCClientDestroy("netactuate_oidc_client.test"),
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_oidc_client.test", resource.ComposeTestCheckFunc(
			testAccTrackOIDCClient("netactuate_oidc_client.test"),
			testAccCheckOIDCClientAPI("netactuate_oidc_client.test", label, "initial", 300, "https://api.example.com/oidc-audience", false),
		), resource.ComposeTestCheckFunc(
			testAccTrackOIDCClient("netactuate_oidc_client.test"),
			testAccCheckOIDCClientAPI("netactuate_oidc_client.test", updatedLabel, "updated", 600, "https://api.example.com/oidc-audience-updated", true),
		), testAccLifecycleOptions{}),
	})
}

func TestAccNetactuateOIDCClientOutOfBandDeleteIdempotent(t *testing.T) {
	label := testAccName("oidc-client-oob")

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: true,
		CheckDestroy:              testAccCheckOIDCClientDestroy("netactuate_oidc_client.test"),
		Steps: []resource.TestStep{{
			Config: testAccOIDCClientResourceOnlyConfig(label, "delete outside terraform"),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackOIDCClient("netactuate_oidc_client.test"),
				testAccCheckOIDCClientAPI("netactuate_oidc_client.test", label, "delete outside terraform", 300, "https://api.example.com/oidc-audience", false),
				testAccDeleteOIDCClient("netactuate_oidc_client.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateOIDCClientKeyLifecycle(t *testing.T) {
	label := testAccName("oidc-key-lifecycle")
	createConfig := testAccOIDCClientKeyConfig(label, "key one")
	modifiedConfig := testAccOIDCClientKeyConfig(label, "key updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: true,
		CheckDestroy:              testAccCheckOIDCClientDestroy("netactuate_oidc_client.test"),
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_oidc_client_key.test", resource.ComposeTestCheckFunc(
			testAccTrackOIDCClient("netactuate_oidc_client.test"),
			testAccCheckOIDCClientKeyAPI("netactuate_oidc_client_key.test", label+"-key", "key one"),
		), resource.ComposeTestCheckFunc(
			testAccTrackOIDCClient("netactuate_oidc_client.test"),
			testAccCheckOIDCClientKeyAPI("netactuate_oidc_client_key.test", label+"-key", "key updated"),
		), testAccLifecycleOptions{
			ImportStateIdFunc:       testAccOIDCStateImportID("netactuate_oidc_client_key.test"),
			ImportStateVerifyIgnore: []string{"public_key"},
		}),
	})
}

func TestAccNetactuateOIDCClientKeyOutOfBandDeleteIdempotent(t *testing.T) {
	label := testAccName("oidc-key-oob")

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: true,
		CheckDestroy:              testAccCheckOIDCClientDestroy("netactuate_oidc_client.test"),
		Steps: []resource.TestStep{{
			Config: testAccOIDCClientKeyResourceOnlyConfig(label, "key delete outside terraform"),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackOIDCClient("netactuate_oidc_client.test"),
				testAccCheckOIDCClientKeyAPI("netactuate_oidc_client_key.test", label+"-key", "key delete outside terraform"),
				testAccDeleteOIDCClientKeyOutOfBand("netactuate_oidc_client_key.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateOIDCClientVMAllowListLifecycle(t *testing.T) {
	mbpkgid := testAccEnvInt(t, "NETACTUATE_TEST_VM_MBPKGID")
	label := testAccName("oidc-vm-allow-lifecycle")
	createConfig := testAccOIDCClientVMAllowListConfig(label, mbpkgid)
	modifiedConfig := testAccOIDCClientVMAllowListResourceOnlyConfig(label, "client for VM allow list updated", mbpkgid)

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t, "NETACTUATE_TEST_VM_MBPKGID") },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: true,
		CheckDestroy:              testAccCheckOIDCClientDestroy("netactuate_oidc_client.test"),
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_oidc_client_vm_allow_list.test", resource.ComposeTestCheckFunc(
			testAccTrackOIDCClient("netactuate_oidc_client.test"),
			testAccCheckOIDCClientVMAllowListAPI("netactuate_oidc_client.test", mbpkgid, true),
		), resource.ComposeTestCheckFunc(
			testAccTrackOIDCClient("netactuate_oidc_client.test"),
			testAccCheckOIDCClientAPI("netactuate_oidc_client.test", label, "client for VM allow list updated", 300, "https://api.example.com/oidc-audience", true),
			testAccCheckOIDCClientVMAllowListAPI("netactuate_oidc_client.test", mbpkgid, true),
		), testAccLifecycleOptions{ImportStateIdFunc: testAccOIDCStateImportID("netactuate_oidc_client_vm_allow_list.test")}),
	})
}

func TestAccNetactuateOIDCClientVMAllowListOutOfBandDeleteIdempotent(t *testing.T) {
	mbpkgid := testAccEnvInt(t, "NETACTUATE_TEST_VM_MBPKGID")
	label := testAccName("oidc-vm-allow-oob")

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t, "NETACTUATE_TEST_VM_MBPKGID") },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: true,
		CheckDestroy:              testAccCheckOIDCClientDestroy("netactuate_oidc_client.test"),
		Steps: []resource.TestStep{{
			Config: testAccOIDCClientVMAllowListResourceOnlyConfig(label, "client for VM allow list", mbpkgid),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackOIDCClient("netactuate_oidc_client.test"),
				testAccCheckOIDCClientVMAllowListAPI("netactuate_oidc_client.test", mbpkgid, true),
				testAccDeleteOIDCClientVMAllowListOutOfBand("netactuate_oidc_client_vm_allow_list.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateOIDCClientBareMetalAllowListLifecycle(t *testing.T) {
	mbpkgid := testAccEnvInt(t, "NETACTUATE_ACC_BARE_METAL_MBPKGID")
	label := testAccName("oidc-bm-allow-lifecycle")
	createConfig := testAccOIDCClientBareMetalAllowListConfig(label, mbpkgid)
	modifiedConfig := testAccOIDCClientBareMetalAllowListResourceOnlyConfig(label, "client for bare metal allow list updated", mbpkgid)

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t, "NETACTUATE_ACC_BARE_METAL_MBPKGID") },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: true,
		CheckDestroy:              testAccCheckOIDCBareMetalAllowListAndClientDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_oidc_client_bare_metal_allow_list.test", resource.ComposeTestCheckFunc(
			testAccTrackOIDCClient("netactuate_oidc_client.test"),
			testAccCheckOIDCBareMetalAllowListAPI("netactuate_oidc_client.test", mbpkgid, true),
		), resource.ComposeTestCheckFunc(
			testAccTrackOIDCClient("netactuate_oidc_client.test"),
			testAccCheckOIDCClientAPI("netactuate_oidc_client.test", label, "client for bare metal allow list updated", 300, "https://api.example.com/oidc-audience", true),
			testAccCheckOIDCBareMetalAllowListAPI("netactuate_oidc_client.test", mbpkgid, true),
		), testAccLifecycleOptions{ImportStateIdFunc: testAccOIDCStateImportID("netactuate_oidc_client_bare_metal_allow_list.test")}),
	})
}

func TestAccNetactuateOIDCClientBareMetalAllowListOutOfBandDeleteIdempotent(t *testing.T) {
	mbpkgid := testAccEnvInt(t, "NETACTUATE_ACC_BARE_METAL_MBPKGID")
	label := testAccName("oidc-bm-allow-oob")

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t, "NETACTUATE_ACC_BARE_METAL_MBPKGID") },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: true,
		CheckDestroy:              testAccCheckOIDCBareMetalAllowListAndClientDestroy,
		Steps: []resource.TestStep{{
			Config: testAccOIDCClientBareMetalAllowListResourceOnlyConfig(label, "client for bare metal allow list", mbpkgid),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackOIDCClient("netactuate_oidc_client.test"),
				testAccCheckOIDCBareMetalAllowListAPI("netactuate_oidc_client.test", mbpkgid, true),
				testAccDeleteOIDCClientBareMetalAllowListOutOfBand("netactuate_oidc_client_bare_metal_allow_list.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func testAccOIDCStateImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return testAccStateID(s, address)
	}
}

func testAccOIDCClientKeyResourceOnlyConfig(label, description string) string {
	return testAccOIDCClientResourceOnlyConfigWithOptions(label, "client for key", false, false) + fmt.Sprintf(`
resource "netactuate_oidc_client_key" "test" {
  oidc_client_id = netactuate_oidc_client.test.oidc_client_id
  label          = %q
  description    = %q
  public_key     = %q
}
`, label+"-key", description, testAccOIDCPublicJWK)
}

func testAccOIDCClientVMAllowListResourceOnlyConfig(label, description string, mbpkgid int) string {
	return testAccOIDCClientResourceOnlyConfigWithOptions(label, description, true, true) + fmt.Sprintf(`
resource "netactuate_oidc_client_vm_allow_list" "test" {
  oidc_client_id = netactuate_oidc_client.test.oidc_client_id
  mbpkgid        = %d
}
`, mbpkgid)
}

func testAccOIDCClientBareMetalAllowListResourceOnlyConfig(label, description string, mbpkgid int) string {
	return testAccOIDCClientResourceOnlyConfigWithOptions(label, description, true, true) + fmt.Sprintf(`
resource "netactuate_oidc_client_bare_metal_allow_list" "test" {
  oidc_client_id = netactuate_oidc_client.test.oidc_client_id
  mbpkgid        = %d
}
`, mbpkgid)
}

func testAccOIDCClientResourceOnlyConfigWithOptions(label, description string, enforceAllowList bool, withJWKS bool) string {
	jwks := ""
	if withJWKS {
		jwks = "\n  jwks_uri           = \"https://keys.example.com/.well-known/jwks.json\""
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_oidc_client" "test" {
  label              = %q
  description        = %q%s
  account_default    = false
  enforce_allow_list = %t
  ttl                = 300
  default_audience   = "https://api.example.com/oidc-audience"
}
`, label, description, jwks, enforceAllowList)
}

func testAccCheckOIDCClientAPI(address, wantLabel, wantDescription string, wantTTL int, wantAudience string, wantEnforceAllowList bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		client, err := testAccClients().V3.GetOIDCClient(id)
		if err != nil {
			return err
		}
		if client.Label != wantLabel {
			return fmt.Errorf("OIDC client %d label = %q, want %q", id, client.Label, wantLabel)
		}
		if client.Description != wantDescription {
			return fmt.Errorf("OIDC client %d description = %q, want %q", id, client.Description, wantDescription)
		}
		if client.TTL != wantTTL {
			return fmt.Errorf("OIDC client %d ttl = %d, want %d", id, client.TTL, wantTTL)
		}
		if client.DefaultAudience != wantAudience {
			return fmt.Errorf("OIDC client %d default audience = %q, want %q", id, client.DefaultAudience, wantAudience)
		}
		if client.EnforceAllowList.Bool() != wantEnforceAllowList {
			return fmt.Errorf("OIDC client %d enforce allow list = %t, want %t", id, client.EnforceAllowList.Bool(), wantEnforceAllowList)
		}
		return nil
	}
}

func testAccCheckOIDCClientKeyAPI(address, wantLabel, wantDescription string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clientID, keyID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		keys, err := testAccClients().V3.GetOIDCClientKeys(clientID)
		if err != nil {
			return err
		}
		for _, key := range keys {
			if key.KeyID != keyID {
				continue
			}
			if key.Label != wantLabel {
				return fmt.Errorf("OIDC client %d key %d label = %q, want %q", clientID, keyID, key.Label, wantLabel)
			}
			if key.Description != wantDescription {
				return fmt.Errorf("OIDC client %d key %d description = %q, want %q", clientID, keyID, key.Description, wantDescription)
			}
			return nil
		}
		return fmt.Errorf("OIDC client %d key %d not found", clientID, keyID)
	}
}

func testAccCheckOIDCClientVMAllowListAPI(clientAddress string, mbpkgid int, want bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clientID, err := testAccID(s, clientAddress)
		if err != nil {
			return err
		}
		vms, err := testAccClients().V3.GetOIDCClientVMs(clientID)
		if err != nil {
			return err
		}
		for _, vm := range vms {
			if vm.MBPkgID == mbpkgid {
				if want {
					return nil
				}
				return fmt.Errorf("OIDC client %d still allows VM package %d", clientID, mbpkgid)
			}
		}
		if want {
			return fmt.Errorf("OIDC client %d does not allow VM package %d", clientID, mbpkgid)
		}
		return nil
	}
}

func testAccDeleteOIDCClientKeyOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clientID, keyID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteOIDCClientKey(clientID, keyID)
	}
}

func testAccDeleteOIDCClientVMAllowListOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clientID, mbpkgid, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.RemoveOIDCClientVM(clientID, mbpkgid)
	}
}

func testAccDeleteOIDCClientBareMetalAllowListOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clientID, mbpkgid, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.RemoveOIDCClientBareMetalServer(clientID, mbpkgid)
	}
}
