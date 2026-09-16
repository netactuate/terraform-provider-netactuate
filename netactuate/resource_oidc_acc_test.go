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

const testAccOIDCPublicJWK = `{"kty":"RSA","kid":"nah-test","use":"sig","alg":"RS256","n":"sXchqG_U8TX12hy0I23WeEAvmk6FzY0P6f8gJObYdQz2YKeOiwR8pEc3ZoY0nWOb3XT7tqjXLgRfK5q23xgEwQ","e":"AQAB"}`

func TestAccOIDCClient_basic(t *testing.T) {
	testAccPreCheck(t)
	label := testAccName("oidc-client")
	updatedLabel := label + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: true,
		CheckDestroy:              testAccCheckOIDCClientDestroy("netactuate_oidc_client.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccOIDCClientConfig(label, "initial", 300, "https://api.example.com/oidc-audience", false, true),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackOIDCClient("netactuate_oidc_client.test"),
					resource.TestCheckResourceAttrSet("netactuate_oidc_client.test", "oidc_client_id"),
					resource.TestCheckResourceAttr("netactuate_oidc_client.test", "label", label),
					resource.TestCheckResourceAttr("netactuate_oidc_client.test", "ttl", "300"),
					resource.TestCheckResourceAttr("netactuate_oidc_client.test", "default_audience", "https://api.example.com/oidc-audience"),
					resource.TestCheckResourceAttr("netactuate_oidc_client.test", "account_default", "false"),
					resource.TestCheckResourceAttr("netactuate_oidc_client.test", "enforce_allow_list", "false"),
					resource.TestCheckResourceAttrSet("netactuate_oidc_client.test", "created_on"),
				),
			},
			{
				ResourceName:      "netactuate_oidc_client.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccOIDCClientConfig(updatedLabel, "updated", 600, "https://api.example.com/oidc-audience-updated", true, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("netactuate_oidc_client.test", "label", updatedLabel),
					resource.TestCheckResourceAttr("netactuate_oidc_client.test", "description", "updated"),
					resource.TestCheckResourceAttr("netactuate_oidc_client.test", "ttl", "600"),
					resource.TestCheckResourceAttr("netactuate_oidc_client.test", "default_audience", "https://api.example.com/oidc-audience-updated"),
					resource.TestCheckResourceAttr("netactuate_oidc_client.test", "account_default", "false"),
					resource.TestCheckResourceAttr("netactuate_oidc_client.test", "enforce_allow_list", "true"),
				),
			},
		},
	})
}

func TestAccOIDCClient_disappears(t *testing.T) {
	testAccPreCheck(t)
	label := testAccName("oidc-client-disappears")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOIDCClientResourceOnlyConfig(label, "delete outside terraform"),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackOIDCClient("netactuate_oidc_client.test"),
					testAccDeleteOIDCClient("netactuate_oidc_client.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccOIDCClientKey_basic(t *testing.T) {
	testAccPreCheck(t)
	label := testAccName("oidc-key")

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: testAccOIDCClientKeyConfig(label, "key one"),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackOIDCClient("netactuate_oidc_client.test"),
					resource.TestCheckResourceAttrSet("netactuate_oidc_client_key.test", "key_id"),
					resource.TestCheckResourceAttr("netactuate_oidc_client_key.test", "label", label+"-key"),
				),
			},
			{
				ResourceName:            "netactuate_oidc_client_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"public_key"},
			},
			{
				Config: testAccOIDCClientKeyConfig(label, "key updated"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("netactuate_oidc_client_key.test", "description", "key updated"),
				),
			},
		},
	})
}

func TestAccOIDCClientVMAllowList_basic(t *testing.T) {
	testAccPreCheck(t, "NETACTUATE_TEST_VM_MBPKGID")
	label := testAccName("oidc-vm-allow")
	mbpkgid := testAccEnvInt(t, "NETACTUATE_TEST_VM_MBPKGID")

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t, "NETACTUATE_TEST_VM_MBPKGID") },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: testAccOIDCClientVMAllowListConfig(label, mbpkgid),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackOIDCClient("netactuate_oidc_client.test"),
					resource.TestCheckResourceAttr("netactuate_oidc_client_vm_allow_list.test", "mbpkgid", strconv.Itoa(mbpkgid)),
				),
			},
			{
				ResourceName:      "netactuate_oidc_client_vm_allow_list.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// testAccOIDCClientResourceOnlyConfig omits the data sources deliberately.
//
// The bundled config reads the client back through four data sources, which is what makes it a
// good round trip test and a bad disappears test. Once the client is deleted out of band, the
// post-apply plan refreshes those data sources against an id that no longer exists and they fail
// with HTTP 404. That is correct behaviour: a data source pointing at a deleted object should
// error rather than return an empty result. So the disappears case gets the resource on its own,
// leaving the plan free to observe the resource leaving state.
func testAccOIDCClientResourceOnlyConfig(label, description string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_oidc_client" "test" {
  label              = %q
  description        = %q
  jwks_uri           = "https://keys.example.com/.well-known/jwks.json"
  account_default    = false
  enforce_allow_list = false
  ttl                = 300
  default_audience   = "https://api.example.com/oidc-audience"
}
`, label, description)
}

// withJWKS selects whether the client under test carries a JWKS URI.
//
// The two are mutually exclusive at the platform, which the spec does not document:
// POST /oidc/clients/{id}/keys returns HTTP 400 "While the OID
// client exists, it has a JWKS URI associated with it, so you can not provide public keys."
// A client fetching its keys from a JWKS URI cannot also have keys pushed to it, which is
// coherent. So the key test must build a client WITHOUT a JWKS URI, and passing false here is
// what makes that configuration legal rather than a 400 at apply time.
func testAccOIDCClientConfig(label, description string, ttl int, audience string, enforceAllowList bool, withJWKS bool) string {
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
  ttl                = %d
  default_audience   = %q
}

data "netactuate_oidc_clients" "all" {}

data "netactuate_oidc_client" "by_id" {
  oidc_client_id = netactuate_oidc_client.test.oidc_client_id
}

data "netactuate_oidc_client_auth_logs" "logs" {
  oidc_client_id = netactuate_oidc_client.test.oidc_client_id
}

data "netactuate_oidc_client_change_logs" "changes" {
  oidc_client_id = netactuate_oidc_client.test.oidc_client_id
}
`, label, description, jwks, enforceAllowList, ttl, audience)
}

func testAccOIDCClientKeyConfig(label, description string) string {
	return testAccOIDCClientConfig(label, "client for key", 300, "https://api.example.com/oidc-audience", false, false) + fmt.Sprintf(`
resource "netactuate_oidc_client_key" "test" {
  oidc_client_id = netactuate_oidc_client.test.oidc_client_id
  label          = %q
  description    = %q
  public_key     = %q
}

data "netactuate_oidc_client_keys" "keys" {
  oidc_client_id = netactuate_oidc_client.test.oidc_client_id
}
`, label+"-key", description, testAccOIDCPublicJWK)
}

func testAccOIDCClientVMAllowListConfig(label string, mbpkgid int) string {
	return testAccOIDCClientConfig(label, "client for VM allow list", 300, "https://api.example.com/oidc-audience", true, true) + fmt.Sprintf(`
resource "netactuate_oidc_client_vm_allow_list" "test" {
  oidc_client_id = netactuate_oidc_client.test.oidc_client_id
  mbpkgid        = %d
}
`, mbpkgid)
}

func testAccTrackOIDCClient(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		testAccTrack("netactuate_oidc_client", id)
		return nil
	}
}

func testAccDeleteOIDCClient(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteOIDCClient(id)
	}
}

func testAccCheckOIDCClientDestroy(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "netactuate_oidc_client" {
				continue
			}
			if !strings.HasPrefix(rs.Primary.Attributes["label"], "nah-") {
				return fmt.Errorf("refusing to inspect non acceptance OIDC client %q", rs.Primary.Attributes["label"])
			}
			id, err := strconv.Atoi(rs.Primary.ID)
			if err != nil {
				return err
			}
			_, err = testAccClients().V3.GetOIDCClient(id)
			if err == nil {
				return fmt.Errorf("%s still exists with ID %d", address, id)
			}
			if !gona.IsV3NotFound(err) {
				return err
			}
		}
		return nil
	}
}
