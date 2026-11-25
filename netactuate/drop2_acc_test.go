//go:build acctest

package netactuate

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// Track composite IDs separately from the existing integer-only acceptance registry.
var drop2AccTracked = map[string]map[string]*schema.ResourceData{}

func init() {
	for kind, r := range drop2Resources() {
		kind, r := kind, r
		resource.AddTestSweepers("netactuate_"+kind, &resource.Sweeper{Name: "netactuate_" + kind, F: func(region string) error {
			if os.Getenv("NETACTUATE_API_KEY") == "" {
				return nil
			}
			for _, d := range drop2AccTracked[kind] {
				if diags := r.DeleteContext(context.Background(), d, testAccClients()); diags.HasError() {
					return fmt.Errorf("%v", diags)
				}
			}
			return nil
		}})
	}
}
func drop2AccPrecheck(t *testing.T, cluster bool) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC is required")
	}
	names := []string{"NETACTUATE_ACC_LOCATION_ID"}
	if cluster {
		names = append(names, "NETACTUATE_ACC_NKE_VERSION", "NETACTUATE_ACC_PLAN", "NETACTUATE_ACC_CONTRACT_ID")
	}
	testAccPreCheck(t, names...)
}
func drop2AccName() string { return "nah-q31-" + strings.TrimPrefix(testAccName("drop2"), "nah-") }
func drop2AccInventory(c *gona.V3Client, prefix string) ([]string, error) {
	out := []string{}
	vpcs, err := c.ListVPCs()
	if err != nil {
		return nil, err
	}
	for _, v := range vpcs {
		if strings.HasPrefix(v.Metadata.Label, prefix) {
			out = append(out, fmt.Sprintf("vpc/%d", v.VPCID))
		}
	}
	clusters, err := c.ListNKEClusters()
	if err != nil {
		return nil, err
	}
	for _, v := range clusters {
		if strings.HasPrefix(v.Name, prefix) {
			out = append(out, fmt.Sprintf("cluster/%d", v.ClusterID))
		}
	}
	sort.Strings(out)
	return out, nil
}
func drop2AccFixture(t *testing.T, name, kind string) string {
	location := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
 label = %q
 description = %q
 location_id = %d
 network_ipv4 = "10.77.0.0/24"
 enable_default_snat = true
}
`, name, name, location)
	if strings.HasPrefix(kind, "nke_") {
		config += fmt.Sprintf(`
resource "netactuate_nke_cluster" "test" {
 name = %q
 version = %q
 location_id = %d
 plan = %q
 contract_id = %d
 minimum_nodes = 1
 maximum_nodes = 1
`, name, testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION"), location, testAccEnvString(t, "NETACTUATE_ACC_PLAN"), testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID"))
		if kind == "nke_cluster_api_access" {
			config += "  network_mode = \"edge-with-vpc\"\n  vpc_id = netactuate_vpc.test.vpc_id\n"
		}
		config += "}\n"
	}
	return config
}
func drop2AccConfig(kind, name string, named, updated bool) string {
	switch kind {
	case "vpc_ip_reservation":
		address := ""
		if named {
			address = "address = \"10.77.0.200\""
		}
		return fmt.Sprintf(`resource "netactuate_vpc_ip_reservation" "test" {
 vpc_id = netactuate_vpc.test.vpc_id
 description = %q
 %s
}`, name, address)
	case "nke_cluster_api_allow":
		return fmt.Sprintf(`resource "netactuate_nke_cluster_api_allow" "test" {
 cluster_id = netactuate_nke_cluster.test.cluster_id
 cidr = "192.0.2.0/24"
 label = %q
}
data "netactuate_nke_cluster_api_allow_list" "test" {
 cluster_id = netactuate_nke_cluster.test.cluster_id
 depends_on = [netactuate_nke_cluster_api_allow.test]
}`, name)
	case "nke_cluster_api_access":
		return fmt.Sprintf(`resource "netactuate_nke_cluster_api_access" "test" {
 cluster_id = netactuate_nke_cluster.test.cluster_id
 public_enabled = %t
 private_endpoint_enabled = %t
}`, updated, !updated)
	case "vpc_attachment":
		prefix := "192.0.2.0/24"
		if updated {
			prefix = "198.51.100.0/24"
		}
		return fmt.Sprintf(`resource "netactuate_vpc_attachment" "test" {
 vpc_id = netactuate_vpc.test.vpc_id
 type = "customer-router"
 address = "10.77.0.2"
 description = %q
 accept = [%q]
}
data "netactuate_vpc_attachments" "test" {
 vpc_id = netactuate_vpc.test.vpc_id
 depends_on = [netactuate_vpc_attachment.test]
}`, name, prefix)
	case "nke_cluster_private_network":
		return `resource "netactuate_nke_cluster_private_network" "test" {
 cluster_id = netactuate_nke_cluster.test.cluster_id
 vpc_id = netactuate_vpc.test.vpc_id
}`
	}
	panic("unknown Drop 2 resource")
}
func drop2AccCheckDestroy(s *terraform.State) error {
	c := testAccClients()
	for _, rs := range s.RootModule().Resources {
		kind := strings.TrimPrefix(rs.Type, "netactuate_")
		r := drop2Resources()[kind]
		if r == nil {
			continue
		}
		d := r.Data(rs.Primary)
		if kind == "nke_cluster_api_access" {
			id, err := strconv.Atoi(rs.Primary.ID)
			if err != nil {
				return err
			}
			a, err := c.V3.GetNKEAPIAccess(id)
			if err == nil && (!a.PublicEnabled || a.PrivateEndpointEnabled) {
				return fmt.Errorf("API endpoint settings not reset")
			}
			if err != nil && !gona.IsV3NotFound(err) {
				return err
			}
			continue
		}
		if kind == "nke_cluster_private_network" {
			continue
		}
		if diags := r.ReadContext(context.Background(), d, c); diags.HasError() {
			return fmt.Errorf("%v", diags)
		}
		if d.Id() != "" {
			return fmt.Errorf("%s still exists", rs.Type)
		}
	}
	if err := testAccCheckNKEClusterDestroy(s); err != nil {
		return err
	}
	return testAccCheckVPCDestroy(s)
}
func drop2AccRun(t *testing.T, kind string, oob, named bool) {
	t.Helper()
	cluster := strings.HasPrefix(kind, "nke_")
	drop2AccPrecheck(t, cluster)
	name := drop2AccName()
	c := testAccClients()
	before, err := drop2AccInventory(c.V3, name)
	if err != nil {
		t.Fatal(err)
	}
	// Inventory is scoped to this run so unrelated account activity cannot mask a leak.
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		err := drop2Wait(ctx, 10*time.Minute, func() (bool, error) {
			after, err := drop2AccInventory(c.V3, name)
			return reflect.DeepEqual(before, after), err
		})
		if err != nil {
			t.Errorf("inventory did not return to its pre-test contents: %v", err)
		}
	}()
	fixture := drop2AccFixture(t, name, kind)
	config := fixture + drop2AccConfig(kind, name, named, false)
	address := "netactuate_" + kind + ".test"
	savedID := ""
	clusterID := 0
	check := func(s *terraform.State) error {
		rs := s.RootModule().Resources[address]
		if rs == nil || rs.Primary.ID == "" {
			return fmt.Errorf("resource missing")
		}
		if savedID != "" && savedID != rs.Primary.ID {
			return fmt.Errorf("in-place update replaced %s", address)
		}
		savedID = rs.Primary.ID
		r := drop2Resources()[kind]
		if drop2AccTracked[kind] == nil {
			drop2AccTracked[kind] = map[string]*schema.ResourceData{}
		}
		drop2AccTracked[kind][savedID] = r.Data(rs.Primary)
		if err := testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test")(s); err != nil {
			return err
		}
		if cluster {
			if err := testAccTrackFromState("netactuate_nke_cluster", "netactuate_nke_cluster.test")(s); err != nil {
				return err
			}
			clusterID, _ = testAccID(s, "netactuate_nke_cluster.test")
		}
		if kind == "nke_cluster_api_allow" {
			rows, err := c.V3.ListNKEAPIAllowList(clusterID)
			if err != nil {
				return err
			}
			platform := false
			for _, r := range rows {
				platform = platform || r.IsPlatform
			}
			if !platform {
				return fmt.Errorf("allow-list did not include platform rows")
			}
			ds := s.RootModule().Resources["data.netactuate_nke_cluster_api_allow_list.test"]
			found := false
			if ds != nil {
				for k, v := range ds.Primary.Attributes {
					if strings.HasSuffix(k, ".is_platform") && v == "true" {
						found = true
					}
				}
			}
			if !found {
				return fmt.Errorf("data source omitted platform rows")
			}
		}
		if kind == "nke_cluster_private_network" {
			nodes, err := c.V3.ListNKEWorkerNodes(clusterID)
			if err != nil {
				return err
			}
			if len(nodes) != 1 {
				return fmt.Errorf("retrofit fixture must have one worker, got %d", len(nodes))
			}
		}
		return nil
	}
	steps := []resource.TestStep{{Config: config, Check: check}}
	if oob {
		steps = append(steps, resource.TestStep{PreConfig: func() {
			if kind == "nke_cluster_api_access" || kind == "nke_cluster_private_network" {
				if err := c.V3.DeleteNKECluster(clusterID); err != nil {
					t.Fatal(err)
				}
				err := drop2Wait(context.Background(), 30*time.Minute, func() (bool, error) {
					_, err := c.V3.GetNKECluster(clusterID)
					if gona.IsV3NotFound(err) {
						return true, nil
					}
					return false, err
				})
				if err != nil {
					t.Fatal(err)
				}
			} else {
				r := drop2Resources()[kind]
				d := r.Data(&terraform.InstanceState{ID: savedID})
				if diags := r.DeleteContext(context.Background(), d, c); diags.HasError() {
					t.Fatal(diags)
				}
			}
			r := drop2Resources()[kind]
			d := r.Data(&terraform.InstanceState{ID: savedID})
			if diags := r.DeleteContext(context.Background(), d, c); diags.HasError() {
				t.Fatalf("repeated delete: %v", diags)
			}
		}, RefreshState: true, ExpectNonEmptyPlan: true, Check: testAccCheckGone(address)})
	} else {
		if kind == "nke_cluster_api_access" || kind == "vpc_attachment" {
			config = fixture + drop2AccConfig(kind, name, named, true)
			steps = append(steps, resource.TestStep{Config: config, Check: check})
		}
		steps = append(steps, resource.TestStep{ResourceName: address, ImportState: true, ImportStateVerify: true}, resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: false})
		if kind == "vpc_attachment" {
			overlap := strings.Replace(config, "198.51.100.0/24", "10.77.0.0/24", 1)
			steps = append(steps, resource.TestStep{Config: overlap, ExpectError: regexp.MustCompile(`(?i)overlap`)}, resource.TestStep{Config: config, Check: check})
		}
	}
	resource.Test(t, resource.TestCase{PreCheck: func() { drop2AccPrecheck(t, cluster) }, ProviderFactories: testAccProviderFactories, CheckDestroy: drop2AccCheckDestroy, Steps: steps})
}
func TestAccNetactuateVPCIPReservation_lifecycle(t *testing.T) {
	for _, named := range []bool{false, true} {
		t.Run(fmt.Sprint(named), func(t *testing.T) { drop2AccRun(t, "vpc_ip_reservation", false, named) })
	}
}
func TestAccNetactuateVPCIPReservation_OutOfBandDeleteIdempotent(t *testing.T) {
	drop2AccRun(t, "vpc_ip_reservation", true, false)
}
func TestAccNetactuateNKEClusterAPIAllow_lifecycle(t *testing.T) {
	drop2AccRun(t, "nke_cluster_api_allow", false, false)
}
func TestAccNetactuateNKEClusterAPIAllow_OutOfBandDeleteIdempotent(t *testing.T) {
	drop2AccRun(t, "nke_cluster_api_allow", true, false)
}
func TestAccNetactuateNKEClusterAPIAccess_lifecycle(t *testing.T) {
	drop2AccRun(t, "nke_cluster_api_access", false, false)
}
func TestAccNetactuateNKEClusterAPIAccess_OutOfBandDeleteIdempotent(t *testing.T) {
	drop2AccRun(t, "nke_cluster_api_access", true, false)
}
func TestAccNetactuateVPCAttachment_lifecycle(t *testing.T) {
	drop2AccRun(t, "vpc_attachment", false, false)
}
func TestAccNetactuateVPCAttachment_OutOfBandDeleteIdempotent(t *testing.T) {
	drop2AccRun(t, "vpc_attachment", true, false)
}
func TestAccNetactuateNKEClusterPrivateNetwork_lifecycle(t *testing.T) {
	drop2AccRun(t, "nke_cluster_private_network", false, false)
}
func TestAccNetactuateNKEClusterPrivateNetwork_OutOfBandDeleteIdempotent(t *testing.T) {
	drop2AccRun(t, "nke_cluster_private_network", true, false)
}

// A service must already hold this reservation. The test verifies that it is not released.
func TestAccNetactuateVPCIPReservation_inUseDeleteRefused(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC is required")
	}
	testAccPreCheck(t, "NETACTUATE_ACC_Q31_IN_USE_VPC_ID", "NETACTUATE_ACC_Q31_IN_USE_RESERVATION_ID")
	v := testAccEnvInt(t, "NETACTUATE_ACC_Q31_IN_USE_VPC_ID")
	id := testAccEnvInt(t, "NETACTUATE_ACC_Q31_IN_USE_RESERVATION_ID")
	c := testAccClients().V3
	rows, err := c.ListVPCIPReservationsByType(v, "customerReserved")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rows {
		if r.IPReservationID == id {
			if !r.InUse || !strings.HasPrefix(r.Description, "nah-q31-") {
				t.Fatal("fixture must be an in-use nah-q31- reservation")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("in-use fixture missing")
	}
	if err = c.DeleteVPCIPReservation(v, id); err == nil {
		t.Fatal("in-use deletion unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "this address is used by a load balancer Service; delete the Service first") {
		t.Fatalf("unexpected refusal: %v", err)
	}
	rows, err = c.ListVPCIPReservationsByType(v, "customerReserved")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.IPReservationID == id && r.InUse {
			return
		}
	}
	t.Fatal("in-use reservation disappeared")
}
