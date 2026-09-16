//go:build acctest

package netactuate

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateNKECluster_vpcNetworking(t *testing.T) {
	name := testAccName("nke-vpc")
	vpcName := name + "-vpc"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	// Derive the pod and service CIDRs per run instead of hardcoding 10.244.0.0/16 and
	// 10.96.0.0/16. Those are the Kubernetes defaults, and on a shared account the platform
	// refuses them once anything else holds them:
	//   400 networking.serviceCidr: "The service CIDR overlaps a network already in use by
	//   another cluster in your account."
	// It is intermittent rather than permanent, because a cluster that is still tearing down
	// keeps its network reserved, so the test can collide with its own previous
	// run. The account's five standing clusters use 10.1x and 100.6x ranges, so a per run octet
	// well clear of those makes the test independent of what else exists.
	octet := testAccCIDROctet(name)
	podCIDR := fmt.Sprintf("10.%d.0.0/16", 200+octet)
	serviceCIDR := fmt.Sprintf("10.%d.0.0/16", 220+octet)
	config := testAccNKEClusterVPCConfig(name, vpcName, locationID, version, plan, contractID, podCIDR, serviceCIDR)

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
		CheckDestroy:      testAccCheckNKEClusterVPCFixtureDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test"),
					testAccCheckVPCFromAPI("netactuate_vpc.test", vpcName, locationID),
					testAccCheckNKEClusterFromAPI("netactuate_nke_cluster.test", name, locationID, version, plan, 1, 1),
					testAccCheckNKEClusterVPCFromAPI("netactuate_nke_cluster.test", "netactuate_vpc.test", podCIDR, serviceCIDR),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckNKEClusterVPCFixtureDestroy(s *terraform.State) error {
	if err := testAccCheckNKEClusterDestroy(s); err != nil {
		return err
	}
	return testAccCheckVPCDestroy(s)
}

func testAccCheckNKEClusterVPCFromAPI(address, vpcAddress string, expectedPodCIDR, expectedServiceCIDR string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		expectedVPCID, err := testAccID(s, vpcAddress)
		if err != nil {
			return err
		}
		cluster, err := testAccClients().V3.GetNKECluster(id)
		if err != nil {
			return err
		}
		if cluster.VpcID != expectedVPCID {
			return fmt.Errorf("%s API vpc_id = %d, want %d", address, cluster.VpcID, expectedVPCID)
		}
		if cluster.Networks.Pod != expectedPodCIDR {
			return fmt.Errorf("%s API pod_cidr = %q, want %q", address, cluster.Networks.Pod, expectedPodCIDR)
		}
		if cluster.Networks.Service != expectedServiceCIDR {
			return fmt.Errorf("%s API service_cidr = %q, want %q", address, cluster.Networks.Service, expectedServiceCIDR)
		}
		return nil
	}
}

func testAccNKEClusterVPCConfig(name, vpcName string, locationID int, version, plan string, contractID int, podCIDR, serviceCIDR string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d

  # REQUIRED for NKE in a VPC. Without an outbound path the worker nodes cannot
  # bootstrap, and the platform refuses the cluster outright:
  #
  #   400 networking.vpcId: "The VPC has no outbound (SNAT) rule, so Kubernetes worker
  #   nodes could never bootstrap. Add an SNAT rule on the VPC's gateway (or enable the
  #   default SNAT rule) and apply changes, then create the cluster."
  #
  # This is a prerequisite a narrative guide would omit, and NKE in a VPC is the newer
  # functionality this wave exists to cover. Carried into review/30.
  enable_default_snat = true
}

resource "netactuate_nke_cluster" "test" {
  name          = %q
  version       = %q
  location_id   = %d
  plan          = %q
  contract_id   = %d
  minimum_nodes = 1
  maximum_nodes = 1
  vpc_id        = netactuate_vpc.test.vpc_id
  pod_cidr      = %q
  service_cidr  = %q
}
`, vpcName, vpcName, locationID, name, version, locationID, plan, contractID, podCIDR, serviceCIDR)
}

// testAccCIDROctet turns a run name into a stable octet offset in 0..19, so two runs and two
// tests do not pick the same network. A hash rather than a counter because acceptance tests have
// no shared state to count in.
func testAccCIDROctet(name string) int {
	sum := 0
	for _, c := range name {
		sum = (sum*31 + int(c)) % 20
	}
	return sum
}
