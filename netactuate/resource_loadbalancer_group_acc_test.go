//go:build acctest

package netactuate

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// NOTE on the VIP. A load balancer group's match_address must be an ALLOCATED FLOATING IP
// on the VPC. Using the gateway or bastion address is refused with a 400 that says exactly
// why:
//
//	"The address 192.73.244.27 is not a floating IP on this VPC. Allocate it as a floating
//	 IP before using it as a load balancer VIP."
//
// That is the clearest error the platform has produced in this programme: it names the
// field, the value and the fix. Worth copying elsewhere.
//
// So every config here allocates netactuate_vpc_floating_ip.lb and uses its address.

func TestAccNetactuateNetworkLoadbalancerGroup_importPlanUpdateBackendsClearAndOutOfBandDelete(t *testing.T) {
	name := testAccName("nlbg")
	updatedName := name + "-u"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	configInitial := testAccNetworkLBGroupConfig(name, locationID, false, false)
	configUpdated := testAccNetworkLBGroupConfig(updatedName, locationID, true, false)
	configCleared := testAccNetworkLBGroupConfig(updatedName, locationID, true, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNetworkLBGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: configInitial,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCLBIDsFromAPI("netactuate_vpc.test"),
					testAccTrackNetworkLBGroupFromState("netactuate_network_loadbalancer_group.test"),
					testAccCheckNetworkLBGroupFromAPI("netactuate_network_loadbalancer_group.test", testAccNetworkLBGroupWant{
						Name:           name,
						Description:    name + " description",
						Algorithm:      "round-robin",
						BackendNames:   []string{name + "-backend-a", name + "-backend-b"},
						RuleProtocols:  []string{"TCP"},
						HealthCheckTCP: false,
					}),
				),
			},
			{
				ResourceName:      "netactuate_network_loadbalancer_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   configInitial,
				PlanOnly: true,
			},
			{
				Config: configUpdated,
				Check: testAccCheckNetworkLBGroupFromAPI("netactuate_network_loadbalancer_group.test", testAccNetworkLBGroupWant{
					Name:           updatedName,
					Description:    updatedName + " description",
					Algorithm:      "least-connections",
					BackendNames:   []string{updatedName + "-backend-b", updatedName + "-backend-c"},
					RuleProtocols:  []string{"TCP", "UDP"},
					HealthCheckTCP: true,
				}),
			},
			{
				// Reduced to ONE backend, not zero: the schema declares MinItems 1 and a
				// config with no backends is rejected before it reaches the API. See
				// Addendum 1 in the criteria.
				Config: configCleared,
				Check: testAccCheckNetworkLBGroupFromAPI("netactuate_network_loadbalancer_group.test", testAccNetworkLBGroupWant{
					Name:        updatedName,
					Description: updatedName + " description",
					Algorithm:   "least-connections",
					// One backend remains, not zero: MinItems 1. See Addendum 1 in the criteria.
					BackendNames:   []string{updatedName + "-backend-a"},
					RuleProtocols:  []string{"TCP", "UDP"},
					HealthCheckTCP: true,
				}),
			},
			{
				PreConfig: func() {
					if err := testAccDeleteNetworkLBGroupsByName(updatedName); err != nil {
						t.Fatalf("out of band network LB group delete: %v", err)
					}
					testAccWaitGoneFromLBAPI(t, "network LB group", func() (bool, error) {
						exists, err := testAccNetworkLBGroupExistsByName(updatedName)
						return !exists, err
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_network_loadbalancer_group.test"),
			},
		},
	})
}

func TestAccNetactuateHTTPLoadbalancerGroup_importPlanUpdateBackendsClearAndOutOfBandDelete(t *testing.T) {
	name := testAccName("htlbg")
	updatedName := name + "-u"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	configInitial := testAccHTTPLBGroupConfig(name, locationID, false, false)
	configUpdated := testAccHTTPLBGroupConfig(updatedName, locationID, true, false)
	configCleared := testAccHTTPLBGroupConfig(updatedName, locationID, true, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPLBGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: configInitial,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCLBIDsFromAPI("netactuate_vpc.test"),
					testAccTrackHTTPLBGroupFromState("netactuate_http_loadbalancer_group.test"),
					testAccCheckHTTPLBGroupFromAPI("netactuate_http_loadbalancer_group.test", testAccHTTPLBGroupWant{
						Name:                  name,
						Description:           name + " description",
						Algorithm:             "round-robin",
						StickySessionsEnabled: false,
						SSLToBackendEnabled:   false,
						InternalPort:          8080,
						MatchPorts:            "80",
						RuleDomains:           []string{name + ".example.invalid"},
						BackendNames:          []string{name + "-backend-a", name + "-backend-b"},
					}),
				),
			},
			{
				ResourceName:      "netactuate_http_loadbalancer_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   configInitial,
				PlanOnly: true,
			},
			{
				Config: configUpdated,
				Check: testAccCheckHTTPLBGroupFromAPI("netactuate_http_loadbalancer_group.test", testAccHTTPLBGroupWant{
					Name:                  updatedName,
					Description:           updatedName + " description",
					Algorithm:             "least-connections",
					StickySessionsEnabled: true,
					SSLToBackendEnabled:   true,
					InternalPort:          8443,
					MatchPorts:            "80+443",
					RuleDomains:           []string{updatedName + ".example.invalid", "alt-" + updatedName + ".example.invalid"},
					BackendNames:          []string{updatedName + "-backend-b", updatedName + "-backend-c"},
				}),
			},
			{
				// Reduced to ONE backend, not zero: the schema declares MinItems 1 and a
				// config with no backends is rejected before it reaches the API. See
				// Addendum 1 in the criteria.
				Config: configCleared,
				Check: testAccCheckHTTPLBGroupFromAPI("netactuate_http_loadbalancer_group.test", testAccHTTPLBGroupWant{
					Name:                  updatedName,
					Description:           updatedName + " description",
					Algorithm:             "least-connections",
					StickySessionsEnabled: true,
					SSLToBackendEnabled:   true,
					InternalPort:          8443,
					MatchPorts:            "80+443",
					RuleDomains:           []string{updatedName + ".example.invalid", "alt-" + updatedName + ".example.invalid"},
					// One backend remains, not zero: MinItems 1. See Addendum 1 in the criteria.
					BackendNames: []string{updatedName + "-backend-a"},
				}),
			},
			{
				PreConfig: func() {
					if err := testAccDeleteHTTPLBGroupsByName(updatedName); err != nil {
						t.Fatalf("out of band HTTP LB group delete: %v", err)
					}
					testAccWaitGoneFromLBAPI(t, "HTTP LB group", func() (bool, error) {
						exists, err := testAccHTTPLBGroupExistsByName(updatedName)
						return !exists, err
					})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_http_loadbalancer_group.test"),
			},
		},
	})
}

type testAccLBGroupID struct {
	LBID    int
	GroupID int
}

var testAccNetworkLBGroups = map[testAccLBGroupID]struct{}{}
var testAccHTTPLBGroups = map[testAccLBGroupID]struct{}{}

type testAccNetworkLBGroupWant struct {
	Name           string
	Description    string
	Algorithm      string
	MatchAddress   string
	BackendNames   []string
	RuleProtocols  []string
	HealthCheckTCP bool
}

type testAccHTTPLBGroupWant struct {
	Name                  string
	Description           string
	Algorithm             string
	StickySessionsEnabled bool
	SSLToBackendEnabled   bool
	InternalPort          int
	MatchAddress          string
	MatchPorts            string
	RuleDomains           []string
	BackendNames          []string
}

func testAccTrackNetworkLBGroupFromState(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		lbID, groupID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		testAccNetworkLBGroups[testAccLBGroupID{LBID: lbID, GroupID: groupID}] = struct{}{}
		return nil
	}
}

func testAccTrackHTTPLBGroupFromState(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		lbID, groupID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		testAccHTTPLBGroups[testAccLBGroupID{LBID: lbID, GroupID: groupID}] = struct{}{}
		return nil
	}
}

func testAccCheckVPCLBIDsFromAPI(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		vpcID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		vpc, err := testAccClients().V3.GetVPC(vpcID)
		if err != nil {
			return err
		}
		if vpc.LoadBalancers == nil {
			return fmt.Errorf("%s API loadBalancers missing", address)
		}
		// The VPC payload reports load balancers as counts, not as a list carrying identifiers,
		// so the balancer id cannot be asserted from here. The count is what this endpoint can
		// answer, and the balancer resources carry their own ids.
		if vpc.LoadBalancers.Network == nil || vpc.LoadBalancers.Network.Total == 0 {
			return fmt.Errorf("%s API reports no network load balancer in the VPC", address)
		}
		if vpc.LoadBalancers.HTTP == nil || vpc.LoadBalancers.HTTP.Total == 0 {
			return fmt.Errorf("%s API reports no HTTP load balancer in the VPC", address)
		}
		stateNLBCount, err := strconv.Atoi(rs.Primary.Attributes["network_loadbalancer_count"])
		if err != nil {
			return fmt.Errorf("%s state network_loadbalancer_count is not an integer: %w", address, err)
		}
		if stateNLBCount != vpc.LoadBalancers.Network.Total {
			return fmt.Errorf("%s network_loadbalancer_count = %d, API returned %d", address, stateNLBCount, vpc.LoadBalancers.Network.Total)
		}
		stateHTTPLBCount, err := strconv.Atoi(rs.Primary.Attributes["http_loadbalancer_count"])
		if err != nil {
			return fmt.Errorf("%s state http_loadbalancer_count is not an integer: %w", address, err)
		}
		if stateHTTPLBCount != vpc.LoadBalancers.HTTP.Total {
			return fmt.Errorf("%s http_loadbalancer_count = %d, API returned %d", address, stateHTTPLBCount, vpc.LoadBalancers.HTTP.Total)
		}
		return nil
	}
}

func testAccCheckNetworkLBGroupFromAPI(address string, want testAccNetworkLBGroupWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		lbID, groupID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		return testAccWaitNetworkLBGroup(lbID, groupID, want)
	}
}

func testAccWaitNetworkLBGroup(lbID, groupID int, want testAccNetworkLBGroupWant) error {
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		group, err := testAccClients().V3.GetNLBGroup(lbID, groupID)
		if err != nil {
			lastErr = err
		} else if err := testAccAssertNetworkLBGroup(group, want); err != nil {
			lastErr = err
		} else {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("network LB group %d/%d did not reach expected API state: %w", lbID, groupID, lastErr)
}

func testAccAssertNetworkLBGroup(group *gona.NLBGroup, want testAccNetworkLBGroupWant) error {
	if group.Name != want.Name || group.Description != want.Description || group.Algorithm != want.Algorithm {
		return fmt.Errorf("API network LB group fields = name:%q description:%q algorithm:%q", group.Name, group.Description, group.Algorithm)
	}
	if group.IPVersion != 4 {
		return fmt.Errorf("API network LB group ip_version = %d, want 4", group.IPVersion)
	}
	if want.MatchAddress != "" && group.Match.Address != want.MatchAddress {
		return fmt.Errorf("API network LB group match address = %q, want %q", group.Match.Address, want.MatchAddress)
	}
	if group.Match.Address == "" {
		return fmt.Errorf("API network LB group match address is empty")
	}
	method := "Ping"
	if want.HealthCheckTCP {
		method = "TCP"
	}
	if !group.HealthCheck.Enabled || group.HealthCheck.Method != method || group.HealthCheck.Interval != 10 || group.HealthCheck.Retries != 3 || group.HealthCheck.Delay != 5 || group.HealthCheck.Timeout != 5 {
		return fmt.Errorf("API network LB group health check = %#v", group.HealthCheck)
	}
	if got := testAccNLBBackendNames(group.Backends); !reflect.DeepEqual(got, want.BackendNames) {
		return fmt.Errorf("API network LB group backend names = %#v, want %#v", got, want.BackendNames)
	}
	if got := testAccNLBRuleProtocols(group.Rules); !reflect.DeepEqual(got, want.RuleProtocols) {
		return fmt.Errorf("API network LB group rule protocols = %#v, want %#v", got, want.RuleProtocols)
	}
	return nil
}

func testAccCheckHTTPLBGroupFromAPI(address string, want testAccHTTPLBGroupWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		lbID, groupID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		return testAccWaitHTTPLBGroup(lbID, groupID, want)
	}
}

func testAccWaitHTTPLBGroup(lbID, groupID int, want testAccHTTPLBGroupWant) error {
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		group, err := testAccClients().V3.GetHTTPLBGroup(lbID, groupID)
		if err != nil {
			lastErr = err
		} else if err := testAccAssertHTTPLBGroup(group, want); err != nil {
			lastErr = err
		} else {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("HTTP LB group %d/%d did not reach expected API state: %w", lbID, groupID, lastErr)
}

func testAccAssertHTTPLBGroup(group *gona.HTTPLBGroup, want testAccHTTPLBGroupWant) error {
	if group.Name != want.Name || group.Description != want.Description || group.Algorithm != want.Algorithm {
		return fmt.Errorf("API HTTP LB group fields = name:%q description:%q algorithm:%q", group.Name, group.Description, group.Algorithm)
	}
	if group.StickySessionsEnabled != want.StickySessionsEnabled || group.SSLToBackendEnabled != want.SSLToBackendEnabled || group.InternalPort != want.InternalPort {
		return fmt.Errorf("API HTTP LB group toggles = sticky:%t ssl_to_backend:%t internal_port:%d", group.StickySessionsEnabled, group.SSLToBackendEnabled, group.InternalPort)
	}
	if want.MatchAddress != "" && group.Match.Address != want.MatchAddress {
		return fmt.Errorf("API HTTP LB group match address = %q, want %q", group.Match.Address, want.MatchAddress)
	}
	if group.Match.Address == "" || group.Match.Ports != want.MatchPorts {
		return fmt.Errorf("API HTTP LB group match = address:%q ports:%q", group.Match.Address, group.Match.Ports)
	}
	if got := testAccHTTPLBBackendNames(group.Backends); !reflect.DeepEqual(got, want.BackendNames) {
		return fmt.Errorf("API HTTP LB group backend names = %#v, want %#v", got, want.BackendNames)
	}
	wantRuleDomains := append([]string(nil), want.RuleDomains...)
	sort.Strings(wantRuleDomains)
	if got := testAccHTTPLBRuleDomains(group.Rules); !reflect.DeepEqual(got, wantRuleDomains) {
		return fmt.Errorf("API HTTP LB group rule domains = %#v, want %#v", got, wantRuleDomains)
	}
	return nil
}

func testAccCheckNetworkLBGroupDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_network_loadbalancer_group" {
			continue
		}
		lbID, groupID, err := parseNLBGroupID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetNLBGroup(lbID, groupID); err == nil {
			return fmt.Errorf("network LB group still exists: %d/%d", lbID, groupID)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return testAccCheckVPCDestroy(s)
}

func testAccCheckHTTPLBGroupDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_http_loadbalancer_group" {
			continue
		}
		lbID, groupID, err := parseHTTPLBGroupID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if _, err := clients.V3.GetHTTPLBGroup(lbID, groupID); err == nil {
			return fmt.Errorf("HTTP LB group still exists: %d/%d", lbID, groupID)
		} else if !gona.IsV3NotFound(err) {
			return err
		}
	}
	return testAccCheckVPCDestroy(s)
}

func testAccDeleteNetworkLBGroupsByName(name string) error {
	clients := testAccClients()
	for id := range testAccNetworkLBGroups {
		group, err := clients.V3.GetNLBGroup(id.LBID, id.GroupID)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		if group.Name == name {
			if err := clients.V3.DeleteNLBGroup(id.LBID, id.GroupID); err != nil && !gona.IsV3NotFound(err) {
				return err
			}
		}
	}
	return nil
}

func testAccDeleteHTTPLBGroupsByName(name string) error {
	clients := testAccClients()
	for id := range testAccHTTPLBGroups {
		group, err := clients.V3.GetHTTPLBGroup(id.LBID, id.GroupID)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		if group.Name == name {
			if err := clients.V3.DeleteHTTPLBGroup(id.LBID, id.GroupID); err != nil && !gona.IsV3NotFound(err) {
				return err
			}
		}
	}
	return nil
}

func testAccNetworkLBGroupExistsByName(name string) (bool, error) {
	clients := testAccClients()
	for id := range testAccNetworkLBGroups {
		group, err := clients.V3.GetNLBGroup(id.LBID, id.GroupID)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return true, err
		}
		if group.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func testAccHTTPLBGroupExistsByName(name string) (bool, error) {
	clients := testAccClients()
	for id := range testAccHTTPLBGroups {
		group, err := clients.V3.GetHTTPLBGroup(id.LBID, id.GroupID)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return true, err
		}
		if group.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func testAccWaitGoneFromLBAPI(t *testing.T, label string, gone func() (bool, error)) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		ok, err := gone()
		if err != nil {
			lastErr = err
		} else if ok {
			return
		}
		time.Sleep(5 * time.Second)
	}
	if lastErr != nil {
		t.Fatalf("%s was not gone before timeout: %v", label, lastErr)
	}
	t.Fatalf("%s was not gone before timeout", label)
}

func testAccNLBBackendNames(backends []gona.NLBGroupBackend) []string {
	names := make([]string, len(backends))
	for i, backend := range backends {
		names[i] = backend.Name
	}
	sort.Strings(names)
	return names
}

func testAccHTTPLBBackendNames(backends []gona.HTTPLBGroupBackend) []string {
	names := make([]string, len(backends))
	for i, backend := range backends {
		names[i] = backend.Name
	}
	sort.Strings(names)
	return names
}

func testAccNLBRuleProtocols(rules []gona.NLBGroupRule) []string {
	protocols := make([]string, len(rules))
	for i, rule := range rules {
		protocols[i] = rule.Protocol
	}
	sort.Strings(protocols)
	return protocols
}

func testAccHTTPLBRuleDomains(rules []gona.HTTPLBGroupRule) []string {
	domains := make([]string, len(rules))
	for i, rule := range rules {
		domains[i] = rule.Match.Domain
	}
	sort.Strings(domains)
	return domains
}

func testAccNetworkLBGroupConfig(name string, locationID int, updated, clearBackends bool) string {
	algorithm := "round-robin"
	backendA := testAccNetworkLBBackendBlock(name+"-backend-a", "192.0.2.10")
	backendB := testAccNetworkLBBackendBlock(name+"-backend-b", "192.0.2.11")
	backendC := ""
	rules := testAccNetworkLBRuleBlock("TCP", 8080, 80)
	healthMethod := "Ping"
	if updated {
		algorithm = "least-connections"
		backendA = ""
		backendB = testAccNetworkLBBackendBlock(name+"-backend-b", "192.0.2.12")
		backendC = testAccNetworkLBBackendBlock(name+"-backend-c", "192.0.2.13")
		rules += testAccNetworkLBRuleBlock("UDP", 8081, 81)
		healthMethod = "TCP"
	}
	if clearBackends {
		// Reduce to exactly ONE backend, not zero. The schema declares MinItems 1, so a
		// config with no backends is rejected by Terraform before it reaches the API:
		//
		//	Error: Insufficient backend blocks
		//	At least 1 "backend" blocks are required.
		//
		// A correct constraint, not a defect: a group with no backends has nothing to
		// balance. The risk this step exists to catch is a collection update that
		// silently keeps the OLD membership, and going from three to one tests that just
		// as well. See Addendum 1 in the criteria.
		//
		// Set explicitly rather than by blanking the others: `updated` already blanks
		// backendA, so blanking B and C left ZERO and the step failed the same way twice.
		backendA = testAccNetworkLBBackendBlock(name+"-backend-a", "192.0.2.10")
		backendB = ""
		backendC = ""
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %[1]q
  description = %[1]q
  location_id = %[2]d
}

resource "netactuate_vpc_floating_ip" "lb" {
  vpc_id     = netactuate_vpc.test.vpc_id
  ip_version = 4
}

resource "netactuate_network_loadbalancer_group" "test" {
  network_loadbalancer_id = netactuate_vpc.test.network_loadbalancer_id
  name                    = %[3]q
  description             = %[4]q
  ip_version              = 4
  algorithm               = %[5]q
  match_address           = netactuate_vpc_floating_ip.lb.address

  health_check {
    enabled  = true
    method   = %[6]q
    interval = 10
    retries  = 3
    delay    = 5
    timeout  = 5
  }
%[7]s%[8]s%[9]s%[10]s
}
`, name, locationID, name, name+" description", algorithm, healthMethod, rules, backendA, backendB, backendC)
}

func testAccHTTPLBGroupConfig(name string, locationID int, updated, clearBackends bool) string {
	algorithm := "round-robin"
	sticky := false
	sslBackend := false
	internalPort := 8080
	matchPorts := "80"
	backendA := testAccHTTPLBBackendBlock(name+"-backend-a", "192.0.2.20")
	backendB := testAccHTTPLBBackendBlock(name+"-backend-b", "192.0.2.21")
	backendC := ""
	rules := testAccHTTPLBRuleBlock(name+".example.invalid", "/")
	if updated {
		algorithm = "least-connections"
		sticky = true
		sslBackend = true
		internalPort = 8443
		matchPorts = "80+443"
		backendA = ""
		backendB = testAccHTTPLBBackendBlock(name+"-backend-b", "192.0.2.22")
		backendC = testAccHTTPLBBackendBlock(name+"-backend-c", "192.0.2.23")
		rules += testAccHTTPLBRuleBlock("alt-"+name+".example.invalid", "/alt")
	}
	if clearBackends {
		// Reduce to exactly ONE backend, not zero. The schema declares MinItems 1, so a
		// config with no backends is rejected by Terraform before it reaches the API:
		//
		//	Error: Insufficient backend blocks
		//	At least 1 "backend" blocks are required.
		//
		// A correct constraint, not a defect: a group with no backends has nothing to
		// balance. The risk this step exists to catch is a collection update that
		// silently keeps the OLD membership, and going from three to one tests that just
		// as well. See Addendum 1 in the criteria.
		//
		// Set explicitly rather than by blanking the others: `updated` already blanks
		// backendA, so blanking B and C left ZERO and the step failed the same way twice.
		backendA = testAccHTTPLBBackendBlock(name+"-backend-a", "192.0.2.10")
		backendB = ""
		backendC = ""
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_vpc" "test" {
  label       = %[1]q
  description = %[1]q
  location_id = %[2]d
}

resource "netactuate_vpc_floating_ip" "lb" {
  vpc_id     = netactuate_vpc.test.vpc_id
  ip_version = 4
}

resource "netactuate_http_loadbalancer_group" "test" {
  http_loadbalancer_id           = netactuate_vpc.test.http_loadbalancer_id
  name                           = %[3]q
  description                    = %[4]q
  algorithm                      = %[5]q
  sticky_sessions_enabled        = %[6]t
  ssl_to_backend_enabled         = %[7]t
  internal_port                  = %[8]d
  match_address                  = netactuate_vpc_floating_ip.lb.address
  match_ports                    = %[9]q
  health_check_active_enabled    = true
  health_check_active_interval   = 10
  health_check_active_retries    = 3
  health_check_active_delay      = 5
  health_check_active_timeout    = 5
  health_check_active_path       = "/health"
  health_check_passive_enabled   = true
%[10]s%[11]s%[12]s%[13]s
}
`, name, locationID, name, name+" description", algorithm, sticky, sslBackend, internalPort, matchPorts, rules, backendA, backendB, backendC)
}

func testAccNetworkLBRuleBlock(protocol string, matchPort, internalPort int) string {
	return fmt.Sprintf(`
  rule {
    protocol      = %q
    port_match    = %d
    port_internal = %d
  }
`, protocol, matchPort, internalPort)
}

func testAccNetworkLBBackendBlock(name, address string) string {
	return fmt.Sprintf(`
  backend {
    name             = %q
    internal_address = %q
  }
`, name, address)
}

func testAccHTTPLBRuleBlock(domain, path string) string {
	return fmt.Sprintf(`
  rule {
    match_domain           = %q
    match_path             = %q
    ssl_enabled            = false
    https_redirect_enabled = false
  }
`, domain, path)
}

func testAccHTTPLBBackendBlock(name, address string) string {
	return fmt.Sprintf(`
  backend {
    name             = %q
    internal_address = %q
  }
`, name, address)
}
