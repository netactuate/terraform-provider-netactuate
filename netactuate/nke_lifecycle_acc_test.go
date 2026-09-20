//go:build acctest

package netactuate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// netactuate_nke_access_urls is intentionally omitted here. The provider calls a
// create endpoint, but Delete only removes Terraform state because the platform
// has no access URL delete operation. Treat it as a data-only lifecycle gap.

func TestAccNetactuateNKEClusterLifecycle(t *testing.T) {
	name := testAccName("nke-life")
	updatedName := name + "-updated"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	createConfig := testAccNKEClusterConfigID(name, locationID, version, plan, contractID)
	modifiedConfig := testAccNKEClusterConfigID(updatedName, locationID, version, plan, contractID)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccNKEAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNKEClusterDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_nke_cluster.test",
			resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
				testAccCheckNKEClusterFromAPI("netactuate_nke_cluster.test", name, locationID, version, plan, 1, 1),
			),
			testAccCheckNKEClusterFromAPI("netactuate_nke_cluster.test", updatedName, locationID, version, plan, 1, 1),
			testAccLifecycleOptions{ImportStateVerifyIgnore: []string{"tag_ids"}},
		),
	})
}

func TestAccNetactuateNKEClusterOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("nke-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	config := testAccNKEClusterConfigID(name, locationID, version, plan, contractID)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccNKEAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNKEClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
					testAccCheckNKEClusterFromAPI("netactuate_nke_cluster.test", name, locationID, version, plan, 1, 1),
				),
			},
			{
				PreConfig: func() {
					if err := testAccDeleteNKEClustersByName(name); err != nil {
						t.Fatalf("out of band NKE cluster delete: %v", err)
					}
					testAccWaitGoneFromNKEAPI(t, "NKE cluster", func() (bool, error) {
						exists, err := testAccNKEClusterExistsByName(name)
						return !exists, err
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_nke_cluster.test"),
			},
		},
	})
}

func TestAccNetactuateNKEWorkerNodeLifecycle(t *testing.T) {
	name := testAccName("nke-work")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	workerIDFile := filepath.Join(t.TempDir(), "worker_node_id")
	clusterConfig := testAccNKEClusterConfigID(name, locationID, version, plan, contractID)
	createConfig := testAccNKEWorkerNodeLifecycleConfig(name, locationID, version, plan, contractID, workerIDFile, "initial")
	modifiedConfig := testAccNKEWorkerNodeLifecycleConfig(name, locationID, version, plan, contractID, workerIDFile, "updated")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccNKEAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNKEClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: clusterConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
					testAccWriteNKEWorkerNodeID("netactuate_nke_cluster.test", workerIDFile),
				),
			},
			{
				Config: createConfig,
				Check:  testAccCheckNKEWorkerNodeAPI("netactuate_nke_worker_node.test"),
			},
			{
				Config:             createConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: modifiedConfig,
				Check:  testAccCheckNKEWorkerNodeAPI("netactuate_nke_worker_node.test"),
			},
			{
				Config:             modifiedConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:            "netactuate_nke_worker_node.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccRouterRoutingImportID("netactuate_nke_worker_node.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"label"},
			},
			// The worker node API does not return label or tag_ids, so import cannot
			// recover the update fields. A post-import no-op plan would pin a provider
			// bug instead of lifecycle stability.
		},
	})
}

func TestAccNetactuateNKEWorkerNodeOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("nke-woob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	workerIDFile := filepath.Join(t.TempDir(), "worker_node_id")
	clusterConfig := testAccNKEClusterConfigID(name, locationID, version, plan, contractID)
	config := testAccNKEWorkerNodeLifecycleConfig(name, locationID, version, plan, contractID, workerIDFile, "initial")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccNKEAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNKEClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: clusterConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
					testAccWriteNKEWorkerNodeID("netactuate_nke_cluster.test", workerIDFile),
				),
			},
			{
				Config: config,
				Check:  testAccCheckNKEWorkerNodeAPI("netactuate_nke_worker_node.test"),
			},
			{
				PreConfig: func() {
					clusterID, workerNodeID, err := testAccReadNKEWorkerNodeIDs(workerIDFile)
					if err != nil {
						t.Fatalf("read NKE worker node id: %v", err)
					}
					if err := testAccClients().V3.DeleteNKEWorkerNode(clusterID, workerNodeID); err != nil && !gona.IsV3NotFound(err) {
						t.Fatalf("out of band NKE worker node delete: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_nke_worker_node.test"),
			},
		},
	})
}

func TestAccNetactuateNKEDNSAddonLifecycle(t *testing.T) {
	clusterName := testAccName("nke-dns")
	zone := testAccDNSZoneName(t)
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	createConfig := testAccNKEDNSAddonLifecycleConfig(clusterName, zone, locationID, version, plan, contractID, "subzone")
	modifiedConfig := testAccNKEDNSAddonLifecycleConfig(clusterName, zone, locationID, version, plan, contractID, "full-zone")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccNKEAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNKEDNSAddonFixtureDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_nke_dns_addon.test",
			resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
				testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
				testAccCheckNKEDNSAddonZoneFromAPI("netactuate_nke_dns_addon.test", zone, "subzone"),
			),
			testAccCheckNKEDNSAddonZoneFromAPI("netactuate_nke_dns_addon.test", zone, "full-zone"),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_nke_dns_addon.test")},
		),
	})
}

func TestAccNetactuateNKEDNSAddonOutOfBandDeleteIdempotent(t *testing.T) {
	clusterName := testAccName("nke-dnsoob")
	zone := testAccDNSZoneName(t)
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	config := testAccNKEDNSAddonLifecycleConfig(clusterName, zone, locationID, version, plan, contractID, "subzone")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccNKEAccPreCheck(t, "NETACTUATE_ACC_DNS_DOMAIN")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNKEDNSAddonFixtureDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
					testAccTrackFromState("netactuate_dns_zone", "netactuate_dns_zone.test"),
					testAccCheckNKEDNSAddonZoneFromAPI("netactuate_nke_dns_addon.test", zone, "subzone"),
				),
			},
			{
				PreConfig: func() {
					if err := testAccDeleteNKEAddonByClusterName(clusterName, nkeDNSAddonType); err != nil {
						t.Fatalf("out of band NKE DNS addon delete: %v", err)
					}
					testAccWaitGoneFromNKEAPI(t, "NKE DNS addon", func() (bool, error) {
						exists, err := testAccNKEAddonExistsByClusterName(clusterName, nkeDNSAddonType)
						return !exists, err
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_nke_dns_addon.test"),
			},
		},
	})
}

func TestAccNetactuateNKEStorageAddonLifecycle(t *testing.T) {
	clusterName := testAccName("nke-stor")
	namespaceName := testAccName("nke-ns")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	storageLocationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	createConfig := testAccNKEStorageAddonLifecycleConfig(clusterName, namespaceName, locationID, storageLocationID, version, plan, contractID, "nke-block", true, "Delete")
	modifiedConfig := testAccNKEStorageAddonLifecycleConfig(clusterName, namespaceName, locationID, storageLocationID, version, plan, contractID, "nke-block-alt", false, "Retain")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccNKEStorageAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNKEStorageAddonFixtureDestroy,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
					testAccTrackFromState("netactuate_storage_block_namespace", "netactuate_storage_block_namespace.test"),
					testAccCheckStorageBlockNamespaceFromAPI("netactuate_storage_block_namespace.test", namespaceName, storageLocationID, 1, false),
					testAccCheckNKEStorageAddonFromAPI("netactuate_nke_storage_addon.test", "netactuate_storage_block_namespace.test", "nke-block", true, "Delete"),
					testAccCheckNKEStorageAddonStatePinned("netactuate_nke_storage_addon.test", "netactuate_storage_block_namespace.test"),
				),
			},
			{
				Config:             createConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: modifiedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNKEStorageAddonFromAPI("netactuate_nke_storage_addon.test", "netactuate_storage_block_namespace.test", "nke-block-alt", false, "Retain"),
					testAccCheckNKEStorageAddonStatePinned("netactuate_nke_storage_addon.test", "netactuate_storage_block_namespace.test"),
				),
			},
			{
				Config:             modifiedConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:            "netactuate_nke_storage_addon.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccRouterRoutingImportID("netactuate_nke_storage_addon.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"block_namespace_id"},
			},
			// The storage addon read response can return block_namespace_id as 0 for
			// API-created addons, so import cannot always recover the configured
			// namespace. Do not add the standard post-import no-op plan until that
			// provider or platform gap is closed.
		},
	})
}

func TestAccNetactuateNKEStorageAddonOutOfBandDeleteIdempotent(t *testing.T) {
	clusterName := testAccName("nke-stoob")
	namespaceName := testAccName("nke-ns")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	storageLocationID := testAccEnvInt(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	config := testAccNKEStorageAddonLifecycleConfig(clusterName, namespaceName, locationID, storageLocationID, version, plan, contractID, "nke-block", true, "Delete")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccNKEStorageAccPreCheck(t)
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
					testAccCheckNKEStorageAddonFromAPI("netactuate_nke_storage_addon.test", "netactuate_storage_block_namespace.test", "nke-block", true, "Delete"),
					testAccCheckNKEStorageAddonStatePinned("netactuate_nke_storage_addon.test", "netactuate_storage_block_namespace.test"),
				),
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

func testAccNKEAccPreCheck(t *testing.T, names ...string) {
	t.Helper()
	required := append([]string{
		"NETACTUATE_ACC_LOCATION_ID",
		"NETACTUATE_ACC_NKE_VERSION",
		"NETACTUATE_ACC_PLAN",
		"NETACTUATE_ACC_CONTRACT_ID",
	}, names...)
	testAccPreCheck(t, required...)
}

func testAccNKEStorageAccPreCheck(t *testing.T) {
	t.Helper()
	testAccNKEAccPreCheck(t, "NETACTUATE_ACC_STORAGE_LOCATION_ID")
}

func testAccWriteNKEWorkerNodeID(clusterAddress, path string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clusterID, err := testAccID(s, clusterAddress)
		if err != nil {
			return err
		}
		nodes, err := testAccClients().V3.ListNKEWorkerNodes(clusterID)
		if err != nil {
			return fmt.Errorf("list NKE worker nodes: %w", err)
		}
		if len(nodes) == 0 {
			return fmt.Errorf("NKE worker node API returned no nodes for cluster %d", clusterID)
		}
		body := fmt.Sprintf("%d/%d\n", clusterID, nodes[0].WorkerNodeID)
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			return fmt.Errorf("write NKE worker node id: %w", err)
		}
		return nil
	}
}

func testAccReadNKEWorkerNodeIDs(path string) (int, int, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	return parseSurfaceTwoPartID(strings.TrimSpace(string(body)), "clusterId/workerNodeId")
}

func testAccNKEWorkerNodeLifecycleConfig(name string, locationID int, version, plan string, contractID int, workerIDFile, labelSuffix string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_nke_cluster" "test" {
  name          = %q
  version       = %q
  location_id   = %d
  plan          = %q
  contract_id   = %d
  minimum_nodes = 1
  maximum_nodes = 1
}

resource "netactuate_nke_worker_node" "test" {
  cluster_id     = netactuate_nke_cluster.test.cluster_id
  worker_node_id = tonumber(split("/", trimspace(file(%q)))[1])
  label          = %q
}
`, name, version, locationID, plan, contractID, workerIDFile, name+"-"+labelSuffix)
}

func testAccNKEDNSAddonLifecycleConfig(clusterName, zoneName string, locationID int, version, plan string, contractID int, mode string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_nke_cluster" "test" {
  name          = %q
  version       = %q
  location_id   = %d
  plan          = %q
  contract_id   = %d
  minimum_nodes = 1
  maximum_nodes = 1
}

resource "netactuate_dns_zone" "test" {
  name = %q
  type = "NATIVE"
}

resource "netactuate_nke_dns_addon" "test" {
  cluster_id = netactuate_nke_cluster.test.cluster_id
  zone       = netactuate_dns_zone.test.name
  mode       = %q
}
`, clusterName, version, locationID, plan, contractID, zoneName, mode)
}

func testAccNKEStorageAddonLifecycleConfig(clusterName, namespaceName string, locationID, storageLocationID int, version, plan string, contractID int, storageClassName string, makeDefault bool, reclaimPolicy string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
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
  storage_class_name  = %q
  make_default        = %t
  reclaim_policy      = %q
}
`, clusterName, version, locationID, plan, contractID, namespaceName, storageLocationID, storageClassName, makeDefault, reclaimPolicy)
}
