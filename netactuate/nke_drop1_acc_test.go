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

func testAccDrop1ClusterID(t *testing.T) int {
	t.Helper()
	testAccPreCheck(t, "NETACTUATE_ACC_NKE_CLUSTER_ID")
	return testAccEnvInt(t, "NETACTUATE_ACC_NKE_CLUSTER_ID")
}
func testAccDrop1PoolConfig(clusterID, packageID int, name string) string {
	return fmt.Sprintf(`
resource "netactuate_nke_node_pool" "test" {
 cluster_id = %d
 name = %q
 type = "managed"
 package_id = %d
 autoscaling {
  enabled = false
  minimum_nodes = 0
  maximum_nodes = 2
 }
 desired_nodes = 0
 labels = { team = "terraform" }
}
`, clusterID, name, packageID)
}
func testAccDrop1CheckPool(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["netactuate_nke_node_pool.test"]
		if !ok {
			return fmt.Errorf("pool absent from state")
		}
		clusterID, id, err := parseSurfaceTwoPartID(rs.Primary.ID, "clusterId/poolId")
		if err != nil {
			return err
		}
		p, err := testAccClients().V3.GetNKENodePool(clusterID, id)
		if err != nil {
			return err
		}
		if p.Name != name {
			return fmt.Errorf("pool name %q, want %q", p.Name, name)
		}
		return nil
	}
}
func testAccDrop1CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "netactuate_nke_node_pool":
			clusterID, id, err := parseSurfaceTwoPartID(rs.Primary.ID, "clusterId/poolId")
			if err != nil {
				return err
			}
			_, err = testAccClients().V3.GetNKENodePool(clusterID, id)
			if !gona.IsV3NotFound(err) {
				return fmt.Errorf("pool still exists or lookup failed: %v", err)
			}
		case "netactuate_nke_autoscaler_addon":
			id, err := strconv.Atoi(rs.Primary.ID)
			if err != nil {
				return err
			}
			_, err = testAccClients().V3.GetClusterAddon(id, nkeAutoscalerAddonType)
			if !gona.IsV3NotFound(err) {
				return fmt.Errorf("autoscaler still exists or lookup failed: %v", err)
			}
		}
	}
	return nil
}
func TestAccNetactuateNKENodePool_lifecycle(t *testing.T) {
	clusterID := testAccDrop1ClusterID(t)
	packageID := testAccEnvInt(t, "NETACTUATE_ACC_PACKAGE_ID")
	name := testAccName("nah-pool")
	create := testAccDrop1PoolConfig(clusterID, packageID, name)
	update := testAccDrop1PoolConfig(clusterID, packageID, name+"-updated")
	resource.Test(t, resource.TestCase{PreCheck: func() { testAccPreCheck(t, "NETACTUATE_ACC_NKE_CLUSTER_ID", "NETACTUATE_ACC_PACKAGE_ID") }, ProviderFactories: testAccProviderFactories, CheckDestroy: testAccDrop1CheckDestroy, Steps: []resource.TestStep{
		{Config: create, Check: resource.ComposeTestCheckFunc(testAccTrackFromState("netactuate_nke_node_pool", "netactuate_nke_node_pool.test"), testAccDrop1CheckPool(name))},
		{Config: update, Check: testAccDrop1CheckPool(name + "-updated")},
		{ResourceName: "netactuate_nke_node_pool.test", ImportState: true, ImportStateVerify: true},
		{Config: update, PlanOnly: true, ExpectNonEmptyPlan: false},
	}})
}
func TestAccNetactuateNKENodePool_OutOfBandDeleteIdempotent(t *testing.T) {
	clusterID := testAccDrop1ClusterID(t)
	packageID := testAccEnvInt(t, "NETACTUATE_ACC_PACKAGE_ID")
	name := testAccName("nah-oob")
	resource.Test(t, resource.TestCase{PreCheck: func() { testAccPreCheck(t, "NETACTUATE_ACC_NKE_CLUSTER_ID", "NETACTUATE_ACC_PACKAGE_ID") }, ProviderFactories: testAccProviderFactories, CheckDestroy: testAccDrop1CheckDestroy, Steps: []resource.TestStep{
		{Config: testAccDrop1PoolConfig(clusterID, packageID, name), Check: resource.ComposeTestCheckFunc(testAccTrackFromState("netactuate_nke_node_pool", "netactuate_nke_node_pool.test"), testAccDrop1CheckPool(name))},
		{PreConfig: func() {
			pools, err := testAccClients().V3.ListNKENodePools(clusterID)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range pools {
				if p.Name == name {
					if err = testAccClients().V3.DeleteNKENodePool(clusterID, p.NodePoolID); err != nil {
						t.Fatal(err)
					}
				}
			}
		}, RefreshState: true, ExpectNonEmptyPlan: true, Check: testAccCheckGone("netactuate_nke_node_pool.test")},
	}})
}
func testAccDrop1AutoscalerConfig(clusterID int, expander string) string {
	return fmt.Sprintf(`
resource "netactuate_nke_autoscaler_addon" "test" {
 cluster_id = %d
 config {
  scale_down_enabled = false
  expander = %q
 }
}
`, clusterID, expander)
}
func TestAccNetactuateNKEAutoscalerAddon_lifecycle(t *testing.T) {
	clusterID := testAccDrop1ClusterID(t)
	check := func(expander string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			a, err := testAccClients().V3.GetClusterAddon(clusterID, nkeAutoscalerAddonType)
			if err != nil {
				return err
			}
			if a.Config.Expander == nil || *a.Config.Expander != expander {
				return fmt.Errorf("autoscaler config differs")
			}
			return nil
		}
	}
	resource.Test(t, resource.TestCase{PreCheck: func() { testAccPreCheck(t, "NETACTUATE_ACC_NKE_CLUSTER_ID") }, ProviderFactories: testAccProviderFactories, CheckDestroy: testAccDrop1CheckDestroy, Steps: []resource.TestStep{
		{Config: testAccDrop1AutoscalerConfig(clusterID, "least-waste"), Check: resource.ComposeTestCheckFunc(testAccTrackFromState("netactuate_nke_autoscaler_addon", "netactuate_nke_autoscaler_addon.test"), check("least-waste"))},
		{Config: testAccDrop1AutoscalerConfig(clusterID, "priority"), Check: check("priority")},
		{ResourceName: "netactuate_nke_autoscaler_addon.test", ImportState: true, ImportStateVerify: true},
		{Config: testAccDrop1AutoscalerConfig(clusterID, "priority"), PlanOnly: true, ExpectNonEmptyPlan: false},
	}})
}

// TF-10 covers the touched cluster, worker resource and worker listing together.
func TestAccNetactuateNKEDrop1_upgradeNoDiff(t *testing.T) {
	testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID", "NETACTUATE_ACC_NKE_VERSION", "NETACTUATE_ACC_PLAN", "NETACTUATE_ACC_CONTRACT_ID")
	config := testAccNKEClusterConfigID(testAccName("nah-upgrade"), testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID"), testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION"), testAccEnvString(t, "NETACTUATE_ACC_PLAN"), testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")) + `
data "netactuate_nke_worker_nodes" "upgrade" {
 cluster_id = netactuate_nke_cluster.test.id
}
resource "netactuate_nke_worker_node" "upgrade" {
 cluster_id = netactuate_nke_cluster.test.id
 worker_node_id = data.netactuate_nke_worker_nodes.upgrade.worker_nodes[0].worker_node_id
}
`
	resource.Test(t, resource.TestCase{CheckDestroy: testAccCheckNKEClusterDestroy, Steps: []resource.TestStep{
		{Config: config, Check: testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"), ExternalProviders: map[string]resource.ExternalProvider{"netactuate": {Source: "netactuate/netactuate", VersionConstraint: "= 0.4.10"}}},
		{Config: config, ProviderFactories: testAccProviderFactories, PlanOnly: true, ExpectNonEmptyPlan: false},
	}})
}
