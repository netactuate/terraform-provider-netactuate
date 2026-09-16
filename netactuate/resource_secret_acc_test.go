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

const testAccSecretValueInitial = "nah-secret-value-initial"
const testAccSecretValueUpdated = "nah-secret-value-updated"

// Secret redaction is asserted by the live runner grepping the outer test
// output. The SDK acceptance callbacks do not receive stdout or stderr, so this
// file keeps secret mismatch errors value-free and leaves output scanning to the
// caller.
func TestAccNetactuateSecretListAndValue_apiReadUpdateImportPlanAndDelete(t *testing.T) {
	name := testAccName("secret-list")
	updatedName := name + "-updated"
	key := testAccName("secret-key")
	updatedKey := key + "-updated"
	configInitial := testAccSecretListValueConfig(name, key, testAccSecretValueInitial)
	configUpdated := testAccSecretListValueConfig(updatedName, updatedKey, testAccSecretValueUpdated)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSecretListValueDestroy,
		Steps: []resource.TestStep{
			{
				Config: configInitial,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_secret_list", "netactuate_secret_list.test"),
					testAccTrackFromState("netactuate_secret_list_value", "netactuate_secret_list_value.test"),
					testAccCheckSecretListAPI("netactuate_secret_list.test", name),
					testAccCheckSecretListValueAPI("netactuate_secret_list.test", "netactuate_secret_list_value.test", key, testAccSecretValueInitial),
				),
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecretListAPI("netactuate_secret_list.test", updatedName),
					testAccCheckSecretListValueAPI("netactuate_secret_list.test", "netactuate_secret_list_value.test", updatedKey, testAccSecretValueUpdated),
				),
			},
			{
				ResourceName:      "netactuate_secret_list.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_secret_list_value.test",
				ImportState:       true,
				ImportStateIdFunc: testAccSecretListValueImportID("netactuate_secret_list.test", "netactuate_secret_list_value.test"),
				ImportStateVerify: true,
			},
			{
				Config:   configUpdated,
				PlanOnly: true,
			},
		},
	})
}

func TestAccNetactuateSecretListAndValue_outOfBandDeleteRefreshLeavesStateClean(t *testing.T) {
	name := testAccName("secret-list-oob")
	key := testAccName("secret-key-oob")
	config := testAccSecretListValueConfig(name, key, testAccSecretValueInitial)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSecretListValueDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_secret_list", "netactuate_secret_list.test"),
					testAccTrackFromState("netactuate_secret_list_value", "netactuate_secret_list_value.test"),
					testAccCheckSecretListValueAPI("netactuate_secret_list.test", "netactuate_secret_list_value.test", key, testAccSecretValueInitial),
				),
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_secret_list_value"] {
						for listID := range testAccCreated["netactuate_secret_list"] {
							_ = clients.V2.DeleteSecretListValue(listID, id)
						}
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_secret_list_value.test"),
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_secret_list_value", "netactuate_secret_list_value.test"),
					testAccCheckSecretListValueAPI("netactuate_secret_list.test", "netactuate_secret_list_value.test", key, testAccSecretValueInitial),
				),
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_secret_list"] {
						_ = clients.V2.DeleteSecretList(id)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGone("netactuate_secret_list_value.test"),
					testAccCheckGone("netactuate_secret_list.test"),
				),
			},
		},
	})
}

func testAccCheckSecretListValueDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "netactuate_secret_list":
			id, err := strconv.Atoi(rs.Primary.ID)
			if err != nil {
				return err
			}
			if _, err := clients.V2.GetSecretList(id); err == nil {
				return fmt.Errorf("secret list still exists: %d", id)
			} else if !gona.IsNotFound(err) {
				return err
			}
		case "netactuate_secret_list_value":
			valueID, err := strconv.Atoi(rs.Primary.ID)
			if err != nil {
				return err
			}
			listID, err := strconv.Atoi(rs.Primary.Attributes["secret_list_id"])
			if err != nil {
				return err
			}
			if _, err := clients.V2.GetSecretListValue(listID, valueID); err == nil {
				return fmt.Errorf("secret list value still exists: %d", valueID)
			} else if !gona.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func testAccCheckSecretListAPI(address, wantName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		list, err := testAccClients().V2.GetSecretList(id)
		if err != nil {
			return err
		}
		if list.Name != wantName {
			return fmt.Errorf("secret list API name mismatch for %d", id)
		}
		return nil
	}
}

func testAccCheckSecretListValueAPI(listAddress, valueAddress, wantKey, wantValue string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		listID, err := testAccID(s, listAddress)
		if err != nil {
			return err
		}
		valueID, err := testAccID(s, valueAddress)
		if err != nil {
			return err
		}
		value, err := testAccClients().V2.GetSecretListValue(listID, valueID)
		if err != nil {
			return err
		}
		if value.SecretListID != listID {
			return fmt.Errorf("secret list value API parent mismatch for %d", valueID)
		}
		if value.SecretKey != wantKey {
			return fmt.Errorf("secret list value API key mismatch for %d", valueID)
		}
		if value.SecretValue != wantValue {
			return fmt.Errorf("secret list value API value mismatch for %d", valueID)
		}
		return nil
	}
}

func testAccSecretListValueImportID(listAddress, valueAddress string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		listID, err := testAccID(s, listAddress)
		if err != nil {
			return "", err
		}
		valueID, err := testAccID(s, valueAddress)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d/%d", listID, valueID), nil
	}
}

func testAccSecretListValueConfig(name, key, value string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_secret_list" "test" {
  name = %q
}

resource "netactuate_secret_list_value" "test" {
  secret_list_id = netactuate_secret_list.test.id
  secret_key     = %q
  secret_value   = %q
}
`, name, key, value)
}
