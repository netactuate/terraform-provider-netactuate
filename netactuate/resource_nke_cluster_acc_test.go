//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateNKECluster_importPlanLocationName(t *testing.T) {
	name := testAccName("nke")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	locationName := testAccEnvString(t, "NETACTUATE_ACC_LOCATION_NAME")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	configID := testAccNKEClusterConfigID(name, locationID, version, plan, contractID)
	configName := testAccNKEClusterConfigName(name, locationName, version, plan, contractID)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_LOCATION_NAME",
				"NETACTUATE_ACC_NKE_VERSION",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNKEClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: configID,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
					testAccCheckNKEClusterFromAPI("netactuate_nke_cluster.test", name, locationID, version, plan, 1, 1),
				),
			},
			{
				ResourceName:            "netactuate_nke_cluster.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tag_ids"},
			},
			{
				Config:   configID,
				PlanOnly: true,
			},
			{
				Config:   configName,
				PlanOnly: true,
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

func testAccCheckNKEClusterDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_nke_cluster" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetNKECluster(id); err == nil {
			return fmt.Errorf("NKE cluster still exists: %d", id)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return nil
}

func testAccCheckNKEClusterFromAPI(address, expectedName string, expectedLocationID int, expectedVersion, expectedPlan string, expectedMinimumNodes, expectedMaximumNodes int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		cluster, err := testAccClients().V3.GetNKECluster(id)
		if err != nil {
			return err
		}
		if cluster.ClusterID != id {
			return fmt.Errorf("%s API cluster_id = %d, want %d", address, cluster.ClusterID, id)
		}
		if cluster.Name != expectedName {
			return fmt.Errorf("%s API name = %q, want %q", address, cluster.Name, expectedName)
		}
		if cluster.Location.ID != expectedLocationID {
			return fmt.Errorf("%s API location_id = %d, want %d", address, cluster.Location.ID, expectedLocationID)
		}
		if cluster.Version.Active != expectedVersion {
			return fmt.Errorf("%s API version = %q, want %q", address, cluster.Version.Active, expectedVersion)
		}
		if cluster.Package.Name != expectedPlan {
			return fmt.Errorf("%s API plan = %q, want %q", address, cluster.Package.Name, expectedPlan)
		}
		if cluster.Nodes.Minimum != expectedMinimumNodes {
			return fmt.Errorf("%s API minimum_nodes = %d, want %d", address, cluster.Nodes.Minimum, expectedMinimumNodes)
		}
		if cluster.Nodes.Maximum == nil {
			return fmt.Errorf("%s API maximum_nodes is nil, want %d", address, expectedMaximumNodes)
		}
		if *cluster.Nodes.Maximum != expectedMaximumNodes {
			return fmt.Errorf("%s API maximum_nodes = %d, want %d", address, *cluster.Nodes.Maximum, expectedMaximumNodes)
		}
		if cluster.URLs.API == "" {
			return fmt.Errorf("%s API url is empty", address)
		}
		return nil
	}
}

func testAccDeleteNKEClustersByName(name string) error {
	clients := testAccClients()
	for id := range testAccCreated["netactuate_nke_cluster"] {
		cluster, err := clients.V3.GetNKECluster(id)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		if cluster.Name == name {
			if err := clients.V3.DeleteNKECluster(id); err != nil && !gona.IsV3NotFound(err) {
				return err
			}
		}
	}
	return nil
}

func testAccNKEClusterExistsByName(name string) (bool, error) {
	for id := range testAccCreated["netactuate_nke_cluster"] {
		cluster, err := testAccClients().V3.GetNKECluster(id)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return false, err
		}
		if cluster.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func testAccWaitGoneFromNKEAPI(t *testing.T, label string, gone func() (bool, error)) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		ok, err := gone()
		if err != nil {
			lastErr = err
		} else if ok {
			return
		}
		time.Sleep(30 * time.Second)
	}
	if lastErr != nil {
		t.Fatalf("%s was not gone before timeout: %v", label, lastErr)
	}
	t.Fatalf("%s was not gone before timeout", label)
}

func testAccNKEClusterConfigID(name string, locationID int, version, plan string, contractID int) string {
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
`, name, version, locationID, plan, contractID)
}

func testAccNKEClusterConfigName(name, locationName, version, plan string, contractID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_nke_cluster" "test" {
  name          = %q
  version       = %q
  location      = %q
  plan          = %q
  contract_id   = %d
  minimum_nodes = 1
  maximum_nodes = 1
}
`, name, version, locationName, plan, contractID)
}
