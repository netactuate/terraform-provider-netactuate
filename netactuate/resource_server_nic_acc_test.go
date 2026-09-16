//go:build acctest

package netactuate

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateServerNIC_attachUpdateImportAndDetach(t *testing.T) {
	name := testAccName("server-nic") + ".natest.io"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	vlanID := testAccNICVLANID(t)
	var mbpkgID int
	var nicID int

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_IMAGE_ID",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
				"NETACTUATE_ACC_SERVER_PASSWORD",
			)
			testAccNICVLANID(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckServerNICAndServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServerNICConfig(name, locationID, imageID, plan, contractID, password, vlanID, nil),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_server", "netactuate_server.test"),
					resource.TestCheckResourceAttrPair("netactuate_server_nic.test", "mbpkgid", "netactuate_server.test", "id"),
					resource.TestCheckResourceAttr("netactuate_server_nic.test", "customer_vlan_id", strconv.Itoa(vlanID)),
					resource.TestCheckResourceAttrSet("netactuate_server_nic.test", "attach_order"),
					resource.TestCheckResourceAttrSet("netactuate_server_nic.test", "nic_id"),
					testAccCheckServerNICAPI("netactuate_server_nic.test", vlanID, true),
					testAccRememberServerNIC("netactuate_server_nic.test", &mbpkgID, &nicID),
				),
			},
			{
				Config: testAccServerNICConfig(name, locationID, imageID, plan, contractID, password, vlanID, intPtr(2)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("netactuate_server_nic.test", "customer_vlan_id", strconv.Itoa(vlanID)),
					resource.TestCheckResourceAttr("netactuate_server_nic.test", "attach_order", "2"),
					testAccCheckServerNICAPI("netactuate_server_nic.test", vlanID, true),
				),
			},
			{
				ResourceName:      "netactuate_server_nic.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccServerNICConfigNoNIC(name, locationID, imageID, plan, contractID, password),
				Check:  testAccCheckServerNICGone(&mbpkgID, &nicID),
			},
		},
	})
}

func testAccNICVLANID(t *testing.T) int {
	t.Helper()
	v := os.Getenv("NETACTUATE_ACC_NIC_VLAN_ID")
	if v == "" {
		t.Skip("NETACTUATE_ACC_NIC_VLAN_ID is unset, so the server NIC write path is deliberately gated. Set it only after designating an existing account VLAN for this acceptance test, because the API can list VLANs but cannot create an isolated test VLAN.")
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		t.Skipf("NETACTUATE_ACC_NIC_VLAN_ID must be an integer for this acceptance test: %v", err)
	}
	return i
}

func testAccCheckServerNICAndServerDestroy(s *terraform.State) error {
	if err := testAccCheckServerNICDestroy(s); err != nil {
		return err
	}
	return testAccCheckServerDestroy(s)
}

func testAccCheckServerNICDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_server_nic" {
			continue
		}
		mbpkgID, nicID, err := parseServerNICID(rs.Primary.ID)
		if err != nil {
			return err
		}
		nic, err := findServerNICByID(clients.V2, mbpkgID, nicID)
		if err != nil {
			if gona.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("checking server %d NIC list after destroy: %w", mbpkgID, err)
		}
		if nic != nil {
			return fmt.Errorf("server %d still has NIC %d after destroy", mbpkgID, nicID)
		}
	}
	return nil
}

func testAccCheckServerNICAPI(address string, wantVLANID int, wantPresent bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			if wantPresent {
				return fmt.Errorf("not found: %s", address)
			}
			return nil
		}
		mbpkgID, nicID, err := parseServerNICID(rs.Primary.ID)
		if err != nil {
			return err
		}
		nic, err := findServerNICByID(testAccClients().V2, mbpkgID, nicID)
		if err != nil {
			return err
		}
		if nic == nil {
			if wantPresent {
				return fmt.Errorf("server %d does not have NIC %d", mbpkgID, nicID)
			}
			return nil
		}
		if !wantPresent {
			return fmt.Errorf("server %d still has NIC %d", mbpkgID, nicID)
		}
		if nic.CustomerVLANID != wantVLANID {
			return fmt.Errorf("server %d NIC %d customer_vlan_id = %d, want %d", mbpkgID, nicID, nic.CustomerVLANID, wantVLANID)
		}
		return nil
	}
}

func testAccRememberServerNIC(address string, mbpkgID, nicID *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		gotMBPkgID, gotNICID, err := parseServerNICID(rs.Primary.ID)
		if err != nil {
			return err
		}
		*mbpkgID = gotMBPkgID
		*nicID = gotNICID
		return nil
	}
}

func testAccCheckServerNICGone(mbpkgID, nicID *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *mbpkgID == 0 || *nicID == 0 {
			return fmt.Errorf("server NIC ID was not recorded before detach")
		}
		nic, err := findServerNICByID(testAccClients().V2, *mbpkgID, *nicID)
		if err != nil {
			return err
		}
		if nic != nil {
			return fmt.Errorf("server %d still has NIC %d", *mbpkgID, *nicID)
		}
		return nil
	}
}

func testAccServerNICConfig(name string, locationID, imageID int, plan, contractID, password string, vlanID int, attachOrder *int) string {
	attachOrderConfig := ""
	if attachOrder != nil {
		attachOrderConfig = fmt.Sprintf("\n  attach_order     = %d", *attachOrder)
	}

	return testAccServerNICConfigNoNIC(name, locationID, imageID, plan, contractID, password) + fmt.Sprintf(`
resource "netactuate_server_nic" "test" {
  mbpkgid          = tonumber(netactuate_server.test.id)
  customer_vlan_id = %d%s
}
`, vlanID, attachOrderConfig)
}

func testAccServerNICConfigNoNIC(name string, locationID, imageID int, plan, contractID, password string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_server" "test" {
  hostname                    = %q
  plan                        = %q
  location_id                 = %d
  image_id                    = %d
  password                    = %q
  package_billing_contract_id = %q

  lifecycle {
    ignore_changes = [password]
  }
}
`, name, plan, locationID, imageID, password, contractID)
}

func intPtr(v int) *int {
	return &v
}
