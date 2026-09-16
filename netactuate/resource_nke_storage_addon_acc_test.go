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

func TestAccNetactuateNKEStorageAddon_basic(t *testing.T) {
	clusterName := testAccName("nke-storage-addon")
	namespaceName := clusterName + "-namespace"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	storageLocationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	config := testAccNKEStorageAddonConfig(clusterName, namespaceName, locationID, storageLocationID, version, plan, contractID)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_STORAGE_LOCATION_ID",
				"NETACTUATE_ACC_NKE_VERSION",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNKEStorageAddonFixtureDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
					testAccTrackFromState("netactuate_storage_block_namespace", "netactuate_storage_block_namespace.test"),
					testAccCheckStorageBlockNamespaceFromAPI("netactuate_storage_block_namespace.test", namespaceName, storageLocationID, 1, false),
					testAccCheckNKEStorageAddonFromAPI("netactuate_nke_storage_addon.test", "netactuate_storage_block_namespace.test", "netactuate-block", true, "Delete"),
					testAccCheckNKEStorageAddonStatePinned("netactuate_nke_storage_addon.test", "netactuate_storage_block_namespace.test"),
				),
			},
			{
				ResourceName:      "netactuate_nke_storage_addon.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_nke_storage_addon.test"),
				ImportStateVerify: true,
				// The storage addon read response does not reliably return the block
				// namespace ID for an addon created through the API, so import cannot
				// recover it from the remote object.
				ImportStateVerifyIgnore: []string{"block_namespace_id"},
			},
			{
				Config:   config,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					if err := testAccDeleteNKEAddonByClusterName(clusterName, nkeStorageAddonType); err != nil {
						t.Fatalf("out of band NKE storage addon delete: %v", err)
					}
					testAccWaitGoneFromNKEAPI(t, "NKE storage addon", func() (bool, error) {
						exists, err := testAccNKEAddonExistsByClusterName(clusterName, nkeStorageAddonType)
						return !exists, err
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_nke_storage_addon.test"),
			},
		},
	})
}

func testAccCheckNKEStorageAddonFixtureDestroy(s *terraform.State) error {
	if err := testAccCheckNKEStorageAddonDestroy(s); err != nil {
		return err
	}
	if err := testAccCheckStorageBlockNamespaceDestroy(s); err != nil {
		return err
	}
	return testAccCheckNKEClusterDestroy(s)
}

func testAccCheckNKEStorageAddonDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_nke_storage_addon" {
			continue
		}
		clusterID, addonType, err := parseNKEAddonStateID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetClusterAddon(clusterID, addonType); err == nil {
			return fmt.Errorf("NKE storage addon still exists on cluster %d", clusterID)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return nil
}

func testAccCheckNKEStorageAddonFromAPI(address, namespaceAddress string, expectedStorageClassName string, expectedMakeDefault bool, expectedReclaimPolicy string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		expectedBlockNamespaceID, err := testAccID(s, namespaceAddress)
		if err != nil {
			return err
		}
		clusterID, addonType, err := parseNKEAddonStateID(rs.Primary.ID)
		if err != nil {
			return err
		}
		addon, err := testAccClients().V3.GetClusterAddon(clusterID, addonType)
		if err != nil {
			return err
		}
		if addon.ClusterID != 0 && addon.ClusterID != clusterID {
			return fmt.Errorf("%s API cluster_id = %d, want %d", address, addon.ClusterID, clusterID)
		}
		if addon.AddonType != "" && addon.AddonType != nkeStorageAddonType {
			return fmt.Errorf("%s API addon_type = %q, want %q", address, addon.AddonType, nkeStorageAddonType)
		}
		if len(addon.Config.Integrations) == 0 {
			return fmt.Errorf("%s API integrations is empty, want block namespace %d", address, expectedBlockNamespaceID)
		}
		in := addon.Config.Integrations[0]
		// PINNED DEFECT, asserted as it currently behaves so a fix is noticed.
		//
		// block_namespace_id comes back as 0 from this endpoint for an addon created
		// through the API, and STAYS 0. A provider side wait for a non zero value can
		// time out, so elapsed time is not the explanation.
		//
		// An ESTABLISHED cluster's addon DOES report a real id, 50107 on cluster 325, so
		// the field is returnable and something populates it. What, is unknown and is an
		// open platform question in review/30.
		//
		// This assertion is deliberately inverted: it expects the zero. When the platform
		// starts returning the id, THIS TEST FAILS, which is the alarm we want. Delete
		// this block and restore the real assertion at that point.
		if in.BlockNamespaceID != 0 {
			return fmt.Errorf(
				"%s API block_namespace_id = %d, and the pinned defect expected 0: the platform "+
					"appears to have started returning it. Restore the real assertion and close "+
					"the open question in review/30", address, in.BlockNamespaceID)
		}
		_ = expectedBlockNamespaceID

		if in.StorageIntegrationID == 0 {
			return fmt.Errorf("%s API storage_integration_id = 0, want nonzero", address)
		}
		if in.StorageClassName != expectedStorageClassName {
			return fmt.Errorf("%s API storage_class_name = %q, want %q", address, in.StorageClassName, expectedStorageClassName)
		}
		if in.IsDefaultClass != expectedMakeDefault {
			return fmt.Errorf("%s API make_default = %t, want %t", address, in.IsDefaultClass, expectedMakeDefault)
		}
		if in.ReclaimPolicy != expectedReclaimPolicy {
			return fmt.Errorf("%s API reclaim_policy = %q, want %q", address, in.ReclaimPolicy, expectedReclaimPolicy)
		}
		return nil
	}
}

func testAccCheckNKEStorageAddonStatePinned(address, namespaceAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		expectedBlockNamespaceID, err := testAccID(s, namespaceAddress)
		if err != nil {
			return err
		}
		blockNamespaceID := rs.Primary.Attributes["block_namespace_id"]
		if blockNamespaceID == "0" {
			return fmt.Errorf("%s state block_namespace_id = 0 after API read, want %d", address, expectedBlockNamespaceID)
		}
		if blockNamespaceID != strconv.Itoa(expectedBlockNamespaceID) {
			return fmt.Errorf("%s state block_namespace_id = %q, want %d", address, blockNamespaceID, expectedBlockNamespaceID)
		}
		if got := rs.Primary.Attributes["storage_integration_id"]; got == "" || got == "0" {
			return fmt.Errorf("%s state storage_integration_id = %q, want nonzero", address, got)
		}
		return nil
	}
}

func testAccNKEStorageAddonConfig(clusterName, namespaceName string, locationID, storageLocationID int, version, plan string, contractID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
data "netactuate_nke_addons" "test" {}

resource "netactuate_nke_cluster" "test" {
  name          = %q
  version       = %q
  location_id   = %d
  plan          = %q
  contract_id   = %d
  minimum_nodes = 1
  maximum_nodes = 1
}

resource "netactuate_storage_block_namespace" "test" {
  label       = %q
  location_id = %d
  capacity    = 1
}

resource "netactuate_nke_storage_addon" "test" {
  cluster_id          = netactuate_nke_cluster.test.cluster_id
  block_namespace_id  = netactuate_storage_block_namespace.test.block_namespace_id
  storage_class_name  = "netactuate-block"
  make_default        = true
  reclaim_policy      = "Delete"
}
`, clusterName, version, locationID, plan, contractID, namespaceName, storageLocationID)
}
