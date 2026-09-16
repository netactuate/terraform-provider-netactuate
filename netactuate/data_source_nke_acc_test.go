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

func TestAccNetactuateNKEDataSources_hydrateFromAPI(t *testing.T) {
	name := testAccName("nke-data-sources")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	config := testAccNKEDataSourcesConfig(name, locationID, version, plan, contractID)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_NKE_VERSION",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNKEClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
					testAccCheckNKEClusterFromAPI("netactuate_nke_cluster.test", name, locationID, version, plan, 1, 1),
					testAccCheckNKEVersionsDataSourceFromAPI("data.netactuate_nke_versions.test"),
					testAccCheckNKEAddonsDataSourceFromAPI("data.netactuate_nke_addons.test"),
					testAccCheckNKEKubeconfigDataSourceFromAPI("data.netactuate_nke_kubeconfig.test", "netactuate_nke_cluster.test"),
					testAccCheckNKEWorkerNodesDataSourceFromAPI("data.netactuate_nke_worker_nodes.test", "netactuate_nke_cluster.test"),
				),
			},
		},
	})
}

func testAccCheckNKEVersionsDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		versions, err := testAccClients().V3.ListNKEVersions()
		if err != nil {
			return err
		}
		if len(versions) == 0 {
			return fmt.Errorf("NKE versions API returned no versions")
		}
		if got := rs.Primary.Attributes["versions.#"]; got != strconv.Itoa(len(versions)) {
			return fmt.Errorf("%s state version count = %q, want API count %d", address, got, len(versions))
		}
		for i, version := range versions {
			key := fmt.Sprintf("versions.%d", i)
			if got := rs.Primary.Attributes[key]; got != version {
				return fmt.Errorf("%s state %s = %q, want API version %q", address, key, got, version)
			}
		}
		return nil
	}
}

func testAccCheckNKEAddonsDataSourceFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		addons, err := testAccClients().V3.ListAddonCatalog()
		if err != nil {
			return err
		}
		if len(addons) == 0 {
			return fmt.Errorf("NKE addon catalog API returned no addons")
		}
		if got := rs.Primary.Attributes["addons.#"]; got != strconv.Itoa(len(addons)) {
			return fmt.Errorf("%s state addon count = %q, want API count %d", address, got, len(addons))
		}
		for i, addon := range addons {
			prefix := fmt.Sprintf("addons.%d", i)
			if got := rs.Primary.Attributes[prefix+".addon_id"]; got != strconv.Itoa(addon.AddonID) {
				return fmt.Errorf("%s state %s.addon_id = %q, want API id %d", address, prefix, got, addon.AddonID)
			}
			if got := rs.Primary.Attributes[prefix+".addon_type"]; got != addon.AddonType {
				return fmt.Errorf("%s state %s.addon_type = %q, want API type %q", address, prefix, got, addon.AddonType)
			}
			if got := rs.Primary.Attributes[prefix+".version"]; got != addon.Version {
				return fmt.Errorf("%s state %s.version = %q, want API version %q", address, prefix, got, addon.Version)
			}
			if got := rs.Primary.Attributes[prefix+".channel"]; got != addon.Channel {
				return fmt.Errorf("%s state %s.channel = %q, want API channel %q", address, prefix, got, addon.Channel)
			}
			if got := rs.Primary.Attributes[prefix+".display_name"]; got != addon.DisplayName {
				return fmt.Errorf("%s state %s.display_name = %q, want API display_name %q", address, prefix, got, addon.DisplayName)
			}
			if got := rs.Primary.Attributes[prefix+".requires_vpc"]; got != strconv.FormatBool(addon.RequiresVpc) {
				return fmt.Errorf("%s state %s.requires_vpc = %q, want API requires_vpc %t", address, prefix, got, addon.RequiresVpc)
			}
		}
		return nil
	}
}

func testAccCheckNKEKubeconfigDataSourceFromAPI(address, clusterAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		clusterID, err := testAccID(s, clusterAddress)
		if err != nil {
			return err
		}
		// The cluster is still fetched, because a kubeconfig for a cluster that cannot be
		// read is not worth asserting about, but its URLs are deliberately NOT compared
		// against the kubeconfig. See the note below.
		if _, err := testAccClients().V3.GetNKECluster(clusterID); err != nil {
			return err
		}
		kubeconfig := rs.Primary.Attributes["kubeconfig"]
		if kubeconfig == "" {
			return fmt.Errorf("%s state kubeconfig is empty", address)
		}
		// Do NOT assert the kubeconfig contains cluster.URLs.API. They are different
		// endpoints by design:
		//
		//	urls.api           https://08b0747c9a378c20-api.nke.netactuate.com
		//	kubeconfig server  https://208.111.40.9:1024
		//
		// urls.api is the ingress hostname; the kubeconfig points at the control plane
		// directly. Asserting they match would have reported a defect that is not one.
		//
		// What IS worth asserting is that the kubeconfig is a usable document: it must
		// name a server, or it cannot reach any cluster at all.
		if !strings.Contains(kubeconfig, "server:") {
			return fmt.Errorf("%s state kubeconfig has no server entry, so it cannot reach a cluster:\n%s",
				address, kubeconfig)
		}
		if !strings.Contains(kubeconfig, "apiVersion: v1") || !strings.Contains(kubeconfig, "kind: Config") {
			return fmt.Errorf("%s state kubeconfig is not Kubernetes config YAML", address)
		}
		return nil
	}
}

func testAccCheckNKEWorkerNodesDataSourceFromAPI(address, clusterAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		clusterID, err := testAccID(s, clusterAddress)
		if err != nil {
			return err
		}
		nodes, err := testAccClients().V3.ListNKEWorkerNodes(clusterID)
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			return fmt.Errorf("NKE worker nodes API returned no nodes for cluster %d", clusterID)
		}
		if got := rs.Primary.Attributes["worker_nodes.#"]; got != strconv.Itoa(len(nodes)) {
			return fmt.Errorf("%s state worker node count = %q, want API count %d", address, got, len(nodes))
		}
		for i, node := range nodes {
			prefix := fmt.Sprintf("worker_nodes.%d", i)
			if got := rs.Primary.Attributes[prefix+".worker_node_id"]; got != strconv.Itoa(node.WorkerNodeID) {
				return fmt.Errorf("%s state %s.worker_node_id = %q, want API id %d", address, prefix, got, node.WorkerNodeID)
			}
			if got := rs.Primary.Attributes[prefix+".name"]; got != node.Name {
				return fmt.Errorf("%s state %s.name = %q, want API name %q", address, prefix, got, node.Name)
			}
			if got := rs.Primary.Attributes[prefix+".status_ready"]; got != strconv.FormatBool(node.Status.Ready) {
				return fmt.Errorf("%s state %s.status_ready = %q, want API status_ready %t", address, prefix, got, node.Status.Ready)
			}
			if node.Package.ID != 0 {
				if got := rs.Primary.Attributes[prefix+".package_id"]; got != strconv.Itoa(node.Package.ID) {
					return fmt.Errorf("%s state %s.package_id = %q, want API package_id %d", address, prefix, got, node.Package.ID)
				}
			}
		}
		return nil
	}
}

func testAccNKEDataSourcesConfig(name string, locationID int, version, plan string, contractID int) string {
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

data "netactuate_nke_versions" "test" {}

data "netactuate_nke_addons" "test" {}

data "netactuate_nke_kubeconfig" "test" {
  cluster_id          = netactuate_nke_cluster.test.cluster_id
  expiration_seconds  = 3600
}

data "netactuate_nke_worker_nodes" "test" {
  cluster_id = netactuate_nke_cluster.test.cluster_id
}
`, name, version, locationID, plan, contractID)
}
