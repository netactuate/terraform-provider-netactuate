//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateDedicatedServerBuyLifecycle(t *testing.T) {
	deviceID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_BUY_DEVICE_ID")
	config := testAccDedicatedServerBuyConfig(deviceID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DEDICATED_BUY_DEVICE_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDedicatedLifecycleCanceled,
		Steps: testAccLifecycleSteps(config, config, "netactuate_dedicated_server_buy.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_dedicated_server_buy", "netactuate_dedicated_server_buy.test"),
			testAccCheckDedicatedServerAPI("netactuate_dedicated_server_buy.test"),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_dedicated_server_buy", "netactuate_dedicated_server_buy.test"),
			testAccCheckDedicatedServerAPI("netactuate_dedicated_server_buy.test"),
		), testAccLifecycleOptions{
			ImportStateIdFunc:       testAccDedicatedLifecycleImportID("netactuate_dedicated_server_buy.test"),
			ImportStateVerifyIgnore: []string{"device_id"},
		}),
	})
}

func TestAccNetactuateDedicatedServerBuyOutOfBandDeleteIdempotent(t *testing.T) {
	deviceID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_BUY_DEVICE_ID")
	var mbpkgID int

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DEDICATED_BUY_DEVICE_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDedicatedLifecycleCanceled,
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
				PreConfig: func() {
					if mbpkgID == 0 {
						t.Fatal("no tracked dedicated server ID")
					}
					if err := testAccCancelDedicatedLifecycleServer(mbpkgID); err != nil {
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

func TestAccNetactuateDedicatedServerBuyBuildLifecycle(t *testing.T) {
	deviceID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_BUY_BUILD_DEVICE_ID")
	profileID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_PROFILE_ID")
	diskLayoutID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_DISK_LAYOUT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_DEDICATED_ROOT_PASSWORD")
	name := testAccName("ds-bb-lc")
	config := testAccDedicatedServerBuyBuildConfig(deviceID, profileID, diskLayoutID, name, password)

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
		CheckDestroy:      testAccCheckDedicatedLifecycleCanceled,
		Steps: testAccLifecycleSteps(config, config, "netactuate_dedicated_server_buy_build.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_dedicated_server_buy_build", "netactuate_dedicated_server_buy_build.test"),
			testAccCheckDedicatedServerAPI("netactuate_dedicated_server_buy_build.test"),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_dedicated_server_buy_build", "netactuate_dedicated_server_buy_build.test"),
			testAccCheckDedicatedServerAPI("netactuate_dedicated_server_buy_build.test"),
		), testAccLifecycleOptions{
			ImportStateIdFunc:       testAccDedicatedLifecycleImportID("netactuate_dedicated_server_buy_build.test"),
			ImportStateVerifyIgnore: []string{"device_id", "profile", "disklayout", "fqdn", "root_password"},
		}),
	})
}

func TestAccNetactuateDedicatedServerBuyBuildOutOfBandDeleteIdempotent(t *testing.T) {
	deviceID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_BUY_BUILD_DEVICE_ID")
	profileID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_PROFILE_ID")
	diskLayoutID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_DISK_LAYOUT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_DEDICATED_ROOT_PASSWORD")
	name := testAccName("ds-bb-oob")
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
		CheckDestroy:      testAccCheckDedicatedLifecycleCanceled,
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
				PreConfig: func() {
					if mbpkgID == 0 {
						t.Fatal("no tracked dedicated server ID")
					}
					if err := testAccCancelDedicatedLifecycleServer(mbpkgID); err != nil {
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

// Gaps:
// netactuate_dedicated_server_deployment deletes by canceling an existing mbpkgid supplied
// through NETACTUATE_ACC_DEDICATED_SERVER_ID. A lifecycle or out of band delete test would
// cancel a pre-existing dedicated server, not one purchased by the test.
// netactuate_dedicated_server_ipv4_reverse has update coverage, but delete is state only
// because the API has no clear/readback endpoint for dedicated reverse DNS.
// netactuate_dedicated_server_power_status is a read action marker. Delete only removes
// Terraform state and must not alter or cancel the dedicated server.

func testAccDedicatedLifecycleImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return testAccStateID(s, address)
	}
}

func testAccCheckDedicatedLifecycleCanceled(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "netactuate_dedicated_server_buy", "netactuate_dedicated_server_buy_build", "netactuate_dedicated_server_deployment":
			id, err := strconv.Atoi(rs.Primary.ID)
			if err != nil {
				return err
			}
			if err := testAccCancelDedicatedLifecycleServer(id); err != nil {
				return err
			}
		}
	}
	return nil
}

func testAccCancelDedicatedLifecycleServer(mbpkgID int) error {
	clients := testAccClients()
	if err := clients.V2.DeleteDedicatedServer(mbpkgID, nil); err != nil && !isNotFoundOrUnprocessable(err) {
		if verifyErr := testAccVerifyDedicatedLifecycleCanceled(mbpkgID); verifyErr == nil {
			return nil
		}
		return fmt.Errorf("cancel dedicated server %d: %w", mbpkgID, err)
	}
	return testAccVerifyDedicatedLifecycleCanceled(mbpkgID)
}

func testAccVerifyDedicatedLifecycleCanceled(mbpkgID int) error {
	server, err := testAccClients().V2.GetMetal(mbpkgID)
	if err != nil {
		if isNotFoundOrUnprocessable(err) {
			return nil
		}
		return err
	}
	if server.Canceling == 1 {
		return nil
	}
	return fmt.Errorf("dedicated server still exists and is not canceling: %d", mbpkgID)
}
