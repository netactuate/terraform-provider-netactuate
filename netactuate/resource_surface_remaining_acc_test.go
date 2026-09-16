//go:build acctest

package netactuate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateFirewallSetRuleOrder_importUpdateAndAPIRead(t *testing.T) {
	name := testAccName("fw-order")
	initial := testAccFirewallSetRuleOrderConfig(name, false)
	updated := testAccFirewallSetRuleOrderConfig(name, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
					testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
						{Name: "alpha", Priority: 10, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
						{Name: "beta", Priority: 20, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
						{Name: "gamma", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
					}, []string{"gamma", "alpha", "beta"}),
				),
			},
			{
				ResourceName:      "netactuate_firewall_set_rule_order.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: updated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFirewallRulesAPI("netactuate_firewall_set.test", []testAccFirewallRule{
						{Name: "alpha", Priority: 10, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
						{Name: "beta", Priority: 20, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
						{Name: "gamma", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
					}, []string{"alpha", "beta", "gamma"}),
				),
			},
		},
	})
}

func TestAccNetactuateFirewallSetVMDetach_importAndAPIRead(t *testing.T) {
	name := testAccName("fw-vm-detach")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	config := testAccFirewallSetVMDetachConfig(name, locationID, imageID, plan, contractID, password)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_IMAGE_ID",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
				"NETACTUATE_ACC_SERVER_PASSWORD",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetVMAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
					testAccTrackFromState("netactuate_server", "netactuate_server.test"),
					testAccCheckFirewallSetVMAPI("netactuate_firewall_set.test", "netactuate_server.test", false),
					resource.TestCheckResourceAttrPair("netactuate_firewall_set_vm_detach.test", "relation_id", "data.netactuate_firewall_set_vms.before", "vms.0.id"),
				),
			},
			{
				ResourceName:      "netactuate_firewall_set_vm_detach.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNetactuateNKEAccessURLs_importAndOutOfBandDelete(t *testing.T) {
	name := testAccName("nke-access")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	config := testAccNKEAccessURLsConfig(name, locationID, version, plan, contractID)

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
					testAccCheckNKEAccessURLsAPI("netactuate_nke_access_urls.test", "netactuate_nke_cluster.test"),
				),
			},
			{
				ResourceName:      "netactuate_nke_access_urls.test",
				ImportState:       true,
				ImportStateVerify: true,
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
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGone("netactuate_nke_access_urls.test"),
					testAccCheckGone("netactuate_nke_cluster.test"),
				),
			},
		},
	})
}

func TestAccNetactuateNKEWorkerNode_importAPIReadAndOutOfBandDelete(t *testing.T) {
	name := testAccName("nke-worker")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	version := testAccEnvString(t, "NETACTUATE_ACC_NKE_VERSION")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvInt(t, "NETACTUATE_ACC_CONTRACT_ID")
	config := testAccNKEWorkerNodeConfig(name, locationID, version, plan, contractID)

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
					testAccCheckNKEWorkerNodeAPI("netactuate_nke_worker_node.test"),
				),
			},
			{
				ResourceName:            "netactuate_nke_worker_node.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"label"},
			},
			{
				PreConfig: func() {
					clusterID := testAccCreatedID("netactuate_nke_cluster")
					if clusterID == 0 {
						t.Fatal("no tracked NKE cluster ID")
					}
					nodes, err := testAccClients().V3.ListNKEWorkerNodes(clusterID)
					if err != nil {
						t.Fatalf("list NKE worker nodes: %v", err)
					}
					if len(nodes) == 0 {
						t.Fatal("NKE worker node API returned no nodes")
					}
					if err := testAccClients().V3.DeleteNKEWorkerNode(clusterID, nodes[0].WorkerNodeID); err != nil && !gona.IsV3NotFound(err) {
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

func TestAccNetactuateRouterRoutingView_importAndOutOfBandDelete(t *testing.T) {
	name := testAccName("router-view")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	config := testAccRouterRoutingViewConfig(name, locationID, plan)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_PLAN",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_router", "netactuate_router.test"),
					testAccCheckRouterRoutingViewAPI("netactuate_router_routing_view.test"),
				),
			},
			{
				ResourceName:      "netactuate_router_routing_view.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_router"] {
						_ = clients.V3.DeleteRouter(id)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGone("netactuate_router_routing_view.test"),
					testAccCheckGone("netactuate_router.test"),
					testAccCheckGone("netactuate_router_vrf.test"),
				),
			},
		},
	})
}

func TestAccNetactuateServerOptions_importUpdateAndOutOfBandDelete(t *testing.T) {
	name := testAccName("server-options")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	initial := testAccServerOptionsConfig(name, locationID, imageID, plan, contractID, password, false)
	updated := testAccServerOptionsConfig(name, locationID, imageID, plan, contractID, password, true)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_IMAGE_ID",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
				"NETACTUATE_ACC_SERVER_PASSWORD",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_server", "netactuate_server.test"),
					testAccCheckServerOptionsAPI("netactuate_server_options.test", name+"-options.example.invalid"),
				),
			},
			{
				ResourceName:            "netactuate_server_options.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"raw_json"},
			},
			{
				Config: updated,
				Check:  testAccCheckServerOptionsAPI("netactuate_server_options.test", name+"-options-updated.example.invalid"),
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_server"] {
						jobID, err := clients.V2.DeleteServer(id, true)
						if err != nil {
							continue
						}
						if d := wait4JobStatus("delete", jobID, clients.V2); d != nil && d.HasError() {
							continue
						}
						_ = clients.V2.UnlinkServer(id)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGone("netactuate_server_options.test"),
					testAccCheckGone("netactuate_server.test"),
				),
			},
		},
	})
}

func TestAccNetactuateVPCNameservers_importUpdateDeleteAndAPIRead(t *testing.T) {
	name := testAccName("vpc-ns")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	initial := testAccVPCNameserversConfig(name, locationID, []string{"192.0.2.53"}, []string{})
	updated := testAccVPCNameserversConfig(name, locationID, []string{"192.0.2.54", "198.51.100.53"}, []string{})
	vpcOnly := testAccVPCConfigID(name, locationID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCNameserversResourceAPI("netactuate_vpc_nameservers.test", []string{"192.0.2.53"}, []string{}),
				),
			},
			{
				ResourceName:      "netactuate_vpc_nameservers.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: updated,
				Check:  testAccCheckVPCNameserversResourceAPI("netactuate_vpc_nameservers.test", []string{"192.0.2.54", "198.51.100.53"}, []string{}),
			},
			{
				Config: vpcOnly,
				Check:  testAccCheckVPCNameserversResourceAbsentAPI("netactuate_vpc.test"),
			},
		},
	})
}

func TestAccNetactuateVPCGatewayStandby_importAndOutOfBandDelete(t *testing.T) {
	name := testAccName("vpc-standby")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	config := testAccVPCGatewayStandbyConfig(name, locationID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCFromAPI("netactuate_vpc.test", name, locationID),
				),
			},
			{
				ResourceName:      "netactuate_vpc_gateway_standby.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				PreConfig: func() {
					clients := testAccClients()
					for id := range testAccCreated["netactuate_vpc"] {
						_ = clients.V3.DeleteVPC(id)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGone("netactuate_vpc_gateway_standby.test"),
					testAccCheckGone("netactuate_vpc.test"),
				),
			},
		},
	})
}

func TestAccNetactuateVPCBackendTemplateBackends_importUpdateDeleteAndAPIRead(t *testing.T) {
	name := testAccName("backend-template-backends")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	initial := testAccVPCBackendTemplateBackendsConfig(name, locationID, false)
	updated := testAccVPCBackendTemplateBackendsConfig(name, locationID, true)
	templateOnly := testAccVPCBackendTemplateConfig(name, locationID, false)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCBackendTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccTrackBackendTemplateFromState("netactuate_vpc_backend_template.test"),
					testAccCheckVPCBackendTemplateBackendsAPI("netactuate_vpc_backend_template_backends.test", []string{"nah-backend-one", "nah-backend-two"}),
				),
			},
			{
				ResourceName:      "netactuate_vpc_backend_template_backends.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: updated,
				Check:  testAccCheckVPCBackendTemplateBackendsAPI("netactuate_vpc_backend_template_backends.test", []string{"nah-backend-three"}),
			},
			{
				Config: templateOnly,
				Check:  testAccCheckVPCBackendTemplateAPI("netactuate_vpc_backend_template.test", name, 0),
			},
		},
	})
}

func testAccCheckNKEAccessURLsAPI(address, clusterAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, address)
		if err != nil {
			return err
		}
		clusterID, err := testAccID(s, clusterAddress)
		if err != nil {
			return err
		}
		cluster, err := testAccClients().V3.GetNKECluster(clusterID)
		if err != nil {
			return err
		}
		if cluster.URLs.API == "" || cluster.URLs.Prometheus == "" || cluster.URLs.KubernetesDashboard == "" {
			return fmt.Errorf("%s API access URLs are incomplete", address)
		}
		if rs.Primary.Attributes["api"] != cluster.URLs.API || rs.Primary.Attributes["prometheus"] != cluster.URLs.Prometheus || rs.Primary.Attributes["kubernetes_dashboard"] != cluster.URLs.KubernetesDashboard {
			return fmt.Errorf("%s state access URLs do not match API", address)
		}
		return nil
	}
}

func testAccCheckNKEWorkerNodeAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clusterID, workerNodeID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		node, err := testAccClients().V3.GetNKEWorkerNode(clusterID, workerNodeID)
		if err != nil {
			return err
		}
		if node.WorkerNodeID != workerNodeID || node.ClusterID != clusterID {
			return fmt.Errorf("%s API worker node = cluster:%d node:%d, want cluster:%d node:%d", address, node.ClusterID, node.WorkerNodeID, clusterID, workerNodeID)
		}
		if node.Name == "" || node.Package.ID == 0 {
			return fmt.Errorf("%s API worker node detail is incomplete", address)
		}
		return nil
	}
}

func testAccCheckRouterRoutingViewAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResourceState(s, address)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(rs.Primary.Attributes["result_json"], "{") && !strings.HasPrefix(rs.Primary.Attributes["result_json"], "[") {
			return fmt.Errorf("%s result_json is not JSON: %q", address, rs.Primary.Attributes["result_json"])
		}
		return nil
	}
}

func testAccCheckServerOptionsAPI(address, fqdn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		mbpkgID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		server, err := testAccClients().V2.GetServer(mbpkgID)
		if err != nil {
			return err
		}
		if server.Name != fqdn {
			return fmt.Errorf("%s API fqdn = %q, want %q", address, server.Name, fqdn)
		}
		return nil
	}
}

func testAccCheckVPCNameserversResourceAPI(address string, wantIPv4, wantIPv6 []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		ns, err := testAccClients().V3.GetVPCNameservers(vpcID)
		if err != nil {
			return err
		}
		if strings.Join(flattenNameservers(ns.IPv4), ",") != strings.Join(wantIPv4, ",") {
			return fmt.Errorf("%s API IPv4 nameservers = %v, want %v", address, flattenNameservers(ns.IPv4), wantIPv4)
		}
		if strings.Join(flattenNameservers(ns.IPv6), ",") != strings.Join(wantIPv6, ",") {
			return fmt.Errorf("%s API IPv6 nameservers = %v, want %v", address, flattenNameservers(ns.IPv6), wantIPv6)
		}
		return nil
	}
}

func testAccCheckVPCNameserversResourceAbsentAPI(vpcAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, err := testAccID(s, vpcAddress)
		if err != nil {
			return err
		}
		ns, err := testAccClients().V3.GetVPCNameservers(vpcID)
		if err != nil {
			return err
		}
		if len(ns.IPv4) != 0 || len(ns.IPv6) != 0 {
			return fmt.Errorf("%s API nameservers = IPv4:%v IPv6:%v, want empty", vpcAddress, ns.IPv4, ns.IPv6)
		}
		return nil
	}
}

func testAccCheckVPCBackendTemplateBackendsAPI(address string, names []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vpcID, templateID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		backends, err := testAccClients().V3.ListVPCBackends(vpcID, templateID)
		if err != nil {
			return err
		}
		if len(backends) != len(names) {
			return fmt.Errorf("%s API backend count = %d, want %d", address, len(backends), len(names))
		}
		for i, name := range names {
			if backends[i].Name != name {
				return fmt.Errorf("%s API backend %d name = %q, want %q", address, i, backends[i].Name, name)
			}
		}
		return nil
	}
}

func testAccCreatedID(resourceType string) int {
	for id := range testAccCreated[resourceType] {
		return id
	}
	return 0
}

func testAccFirewallSetRuleOrderConfig(name string, updated bool) string {
	config := testAccFirewallSetRulesConfig(name, []testAccFirewallRule{
		{Name: "alpha", Priority: 10, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"203.0.113.10/32"}, DestinationPortStart: 22, DestinationPortEnd: 22},
		{Name: "beta", Priority: 20, Action: "DROP", Protocol: "udp", SourceNet: []string{"198.51.100.0/24"}, DestinationPortStart: 53, DestinationPortEnd: 53},
		{Name: "gamma", Priority: 30, Action: "ACCEPT", Protocol: "tcp", SourceNet: []string{"192.0.2.0/24"}, DestinationPortStart: 443, DestinationPortEnd: 443},
	})
	move := `
resource "netactuate_firewall_set_rule_order" "test" {
  firewall_set_id = netactuate_firewall_set.test.id
  move_id         = netactuate_firewall_rule.gamma.id
  before_id       = netactuate_firewall_rule.alpha.id

  depends_on = [
    netactuate_firewall_rule.alpha,
    netactuate_firewall_rule.beta,
    netactuate_firewall_rule.gamma,
  ]
}
`
	if updated {
		move = `
resource "netactuate_firewall_set_rule_order" "test" {
  firewall_set_id = netactuate_firewall_set.test.id
  move_id         = netactuate_firewall_rule.gamma.id
  after_id        = netactuate_firewall_rule.beta.id

  depends_on = [
    netactuate_firewall_rule.alpha,
    netactuate_firewall_rule.beta,
    netactuate_firewall_rule.gamma,
  ]
}
`
	}
	return config + move
}

func testAccFirewallSetVMDetachConfig(name string, locationID, imageID int, plan, contractID, password string) string {
	return testAccFirewallSetVMConfig(name, locationID, imageID, plan, contractID, password, true) + `
data "netactuate_firewall_set_vms" "before" {
  firewall_set_id = netactuate_firewall_set.test.id

  depends_on = [netactuate_firewall_set_vm.test]
}

resource "netactuate_firewall_set_vm_detach" "test" {
  relation_id = data.netactuate_firewall_set_vms.before.vms[0].id
}
`
}

func testAccNKEAccessURLsConfig(name string, locationID int, version, plan string, contractID int) string {
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

resource "netactuate_nke_access_urls" "test" {
  cluster_id = netactuate_nke_cluster.test.cluster_id
}
`, name, version, locationID, plan, contractID)
}

func testAccNKEWorkerNodeConfig(name string, locationID int, version, plan string, contractID int) string {
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

data "netactuate_nke_worker_nodes" "test" {
  cluster_id = netactuate_nke_cluster.test.cluster_id
}

resource "netactuate_nke_worker_node" "test" {
  cluster_id     = netactuate_nke_cluster.test.cluster_id
  worker_node_id = data.netactuate_nke_worker_nodes.test.worker_nodes[0].worker_node_id
  label          = %q
}
`, name, version, locationID, plan, contractID, name+" worker")
}

func testAccRouterRoutingViewConfig(name string, locationID int, plan string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_router" "test" {
  name        = %q
  description = %q
  location_id = %d
  plan        = %q
}

resource "netactuate_router_vrf" "test" {
  router_id   = netactuate_router.test.router_id
  name        = %q
  description = %q
}

resource "netactuate_router_routing_view" "test" {
  router_id = netactuate_router.test.router_id
  vrf_id    = netactuate_router_vrf.test.vrf_id

  view {
    ip_version = 4
    name       = "routes"
    filter     = "192.0.2.0/24"
  }
}
`, name, name, locationID, plan, name+" vrf", name+" vrf")
}

func testAccServerOptionsConfig(name string, locationID, imageID int, plan, contractID, password string, updated bool) string {
	fqdn := name + "-options.example.invalid"
	if updated {
		fqdn = name + "-options-updated.example.invalid"
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_server" "test" {
  hostname                    = %q
  plan                        = %q
  location_id                 = %d
  image_id                    = %d
  password                    = %q
  package_billing_contract_id = %q

  lifecycle {
    ignore_changes = [password, hostname]
  }
}

resource "netactuate_server_options" "test" {
  mbpkgid = netactuate_server.test.id
  fqdn    = %q
}
`, name+".example.invalid", plan, locationID, imageID, password, contractID, fqdn)
}

func testAccVPCNameserversConfig(name string, locationID int, ipv4, ipv6 []string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_nameservers" "test" {
  vpc_id = netactuate_vpc.test.vpc_id
  ipv4   = %s
  ipv6   = %s
}
`, name, name, locationID, testAccStringList(ipv4), testAccStringList(ipv6))
}

func testAccVPCGatewayStandbyConfig(name string, locationID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_gateway_standby" "test" {
  vpc_id = netactuate_vpc.test.vpc_id
}
`, name, name, locationID)
}

func testAccVPCBackendTemplateBackendsConfig(name string, locationID int, updated bool) string {
	hosts := `
  backend_host {
    name    = "nah-backend-one"
    address = "192.0.2.10"
  }

  backend_host {
    name    = "nah-backend-two"
    address = "192.0.2.11"
  }
`
	if updated {
		hosts = `
  backend_host {
    name    = "nah-backend-three"
    address = "192.0.2.12"
  }
`
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %q
  description = %q
  location_id = %d
}

resource "netactuate_vpc_backend_template" "test" {
  vpc_id      = netactuate_vpc.test.vpc_id
  name        = %q
  description = "nah acceptance backend template"
}

resource "netactuate_vpc_backend_template_backends" "test" {
  vpc_id              = netactuate_vpc.test.vpc_id
  backend_template_id = netactuate_vpc_backend_template.test.backend_template_id
%s}
`, name, name, locationID, name, hosts)
}
