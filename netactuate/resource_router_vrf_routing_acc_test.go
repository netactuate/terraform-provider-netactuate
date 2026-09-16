//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuateRouterVRFRouting_importUpdateRemoveAndOutOfBandDelete(t *testing.T) {
	name := testAccName("router-routing")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	initialConfig := testAccRouterVRFRoutingConfig(name, locationID, plan, false)
	updateConfig := testAccRouterVRFRoutingConfig(name, locationID, plan, true)
	routerOnlyConfig := testAccRouterVRFRoutingRouterOnlyConfig(name, locationID, plan)
	ids := &testAccRouterVRFRoutingIDs{}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_PLAN",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterVRFRoutingDestroy,
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_router", "netactuate_router.test"),
					ids.capture,
					testAccCheckRouterVRFAPI("netactuate_router_vrf.test", name+" vrf", name+" vrf initial"),
					testAccCheckRouterVRFBGPAPI("netactuate_router_vrf_bgp.test", "64512", []string{"198.51.100.0/24"}),
					testAccCheckRouterPrefixListAPI("netactuate_router_prefix_list.test", testAccRouterPrefixListWant{
						Name:        name + "-pl",
						Description: name + " prefix initial",
						IPVersion:   4,
						Rules: []gona.PrefixListRule{
							{Action: "permit", Prefix: "198.51.100.0/24"},
						},
					}),
					testAccCheckRouterBGPNeighborAPI("netactuate_router_vrf_bgp_neighbor.test", testAccRouterBGPNeighborWant{
						Name:               name + "-neighbor",
						Description:        name + " neighbor initial",
						Address:            "192.0.2.2",
						RemoteASN:          64513,
						IPv4Enabled:        true,
						IPv6Enabled:        false,
						DoAsOverride:       false,
						DoNextHopSelf:      true,
						IsShutdown:         false,
						ImportDefaultDrop:  true,
						ExportDefaultDrop:  false,
						ImportAction:       "permit",
						ExportAction:       "deny",
						SetLocalPreference: 200,
						PrependLastAsn:     2,
					}),
					testAccCheckRouterStaticRouteAPI("netactuate_router_static_route.test", testAccRouterStaticRouteWant{
						Network:     "203.0.113.0/24",
						Description: name + " static initial",
						Distance:    10,
						NextHop:     "192.0.2.1",
					}),
				),
			},
			{
				ResourceName:      "netactuate_router_vrf.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_vrf.test"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_router_vrf_bgp.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_vrf_bgp.test"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_router_prefix_list.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_prefix_list.test"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_router_vrf_bgp_neighbor.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_vrf_bgp_neighbor.test"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_router_static_route.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_static_route.test"),
				ImportStateVerify: true,
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					ids.capture,
					testAccCheckRouterVRFAPI("netactuate_router_vrf.test", name+" vrf updated", name+" vrf updated"),
					testAccCheckRouterVRFBGPAPI("netactuate_router_vrf_bgp.test", "64514", []string{"198.51.100.0/24", "203.0.113.0/24"}),
					testAccCheckRouterPrefixListAPI("netactuate_router_prefix_list.test", testAccRouterPrefixListWant{
						Name:        name + "-pl-updated",
						Description: name + " prefix updated",
						IPVersion:   4,
						Rules: []gona.PrefixListRule{
							{Action: "deny", Prefix: "192.0.2.0/24"},
							{Action: "permit", Prefix: "198.51.100.0/24"},
						},
					}),
					testAccCheckRouterBGPNeighborAPI("netactuate_router_vrf_bgp_neighbor.test", testAccRouterBGPNeighborWant{
						Name:               name + "-neighbor-updated",
						Description:        name + " neighbor updated",
						Address:            "192.0.2.3",
						RemoteASN:          64515,
						IPv4Enabled:        true,
						IPv6Enabled:        false,
						DoAsOverride:       true,
						DoNextHopSelf:      false,
						IsShutdown:         true,
						ImportDefaultDrop:  false,
						ExportDefaultDrop:  true,
						ImportAction:       "deny",
						ExportAction:       "permit",
						SetLocalPreference: 300,
						PrependLastAsn:     3,
					}),
					testAccCheckRouterStaticRouteAPI("netactuate_router_static_route.test", testAccRouterStaticRouteWant{
						Network:     "198.51.100.0/24",
						Description: name + " static updated",
						Distance:    20,
						NextHop:     "192.0.2.254",
					}),
				),
			},
			{
				// The BGP endpoint has GET and PUT only. Removing the Terraform block drops the
				// provider state, and deleting the VRF removes the API surface that holds it.
				Config: routerOnlyConfig,
				Check: resource.ComposeTestCheckFunc(
					ids.checkRemovedFromAPI,
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					ids.capture,
					testAccCheckRouterVRFAPI("netactuate_router_vrf.test", name+" vrf updated", name+" vrf updated"),
					testAccCheckRouterBGPNeighborAPI("netactuate_router_vrf_bgp_neighbor.test", testAccRouterBGPNeighborWant{
						Name:               name + "-neighbor-updated",
						Description:        name + " neighbor updated",
						Address:            "192.0.2.3",
						RemoteASN:          64515,
						IPv4Enabled:        true,
						IPv6Enabled:        false,
						DoAsOverride:       true,
						DoNextHopSelf:      false,
						IsShutdown:         true,
						ImportDefaultDrop:  false,
						ExportDefaultDrop:  true,
						ImportAction:       "deny",
						ExportAction:       "permit",
						SetLocalPreference: 300,
						PrependLastAsn:     3,
					}),
				),
			},
			{
				PreConfig: func() {
					if err := ids.deleteOutOfBand(); err != nil {
						t.Fatalf("out of band router routing delete: %v", err)
					}
					testAccWaitGoneFromRouterRoutingAPI(t, "router routing resources", ids.goneFromAPI)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGone("netactuate_router_vrf.test"),
					testAccCheckGone("netactuate_router_vrf_bgp.test"),
					testAccCheckGone("netactuate_router_prefix_list.test"),
					testAccCheckGone("netactuate_router_vrf_bgp_neighbor.test"),
					testAccCheckGone("netactuate_router_static_route.test"),
				),
			},
		},
	})
}

type testAccRouterVRFRoutingIDs struct {
	RouterID      int
	VRFID         int
	PrefixListID  int
	NeighborID    int
	StaticRouteID int
}

type testAccRouterPrefixListWant struct {
	Name        string
	Description string
	IPVersion   int
	Rules       []gona.PrefixListRule
}

type testAccRouterBGPNeighborWant struct {
	Name               string
	Description        string
	Address            string
	RemoteASN          int
	IPv4Enabled        bool
	IPv6Enabled        bool
	DoAsOverride       bool
	DoNextHopSelf      bool
	IsShutdown         bool
	ImportDefaultDrop  bool
	ExportDefaultDrop  bool
	ImportAction       string
	ExportAction       string
	SetLocalPreference int
	PrependLastAsn     int
}

type testAccRouterStaticRouteWant struct {
	Network     string
	Description string
	Distance    int
	NextHop     string
}

func (ids *testAccRouterVRFRoutingIDs) capture(s *terraform.State) error {
	routerID, err := testAccID(s, "netactuate_router.test")
	if err != nil {
		return err
	}
	_, vrfID, err := testAccCompositeID(s, "netactuate_router_vrf.test")
	if err != nil {
		return err
	}
	_, prefixListID, err := testAccCompositeID(s, "netactuate_router_prefix_list.test")
	if err != nil {
		return err
	}
	_, _, neighborID, err := testAccTripleID(s, "netactuate_router_vrf_bgp_neighbor.test")
	if err != nil {
		return err
	}
	_, _, staticRouteID, err := testAccTripleID(s, "netactuate_router_static_route.test")
	if err != nil {
		return err
	}

	ids.RouterID = routerID
	ids.VRFID = vrfID
	ids.PrefixListID = prefixListID
	ids.NeighborID = neighborID
	ids.StaticRouteID = staticRouteID
	return nil
}

func (ids *testAccRouterVRFRoutingIDs) checkRemovedFromAPI(s *terraform.State) error {
	if !ids.goneFromAPI() {
		return fmt.Errorf("router routing resources still present after Terraform removal")
	}
	return nil
}

func (ids *testAccRouterVRFRoutingIDs) deleteOutOfBand() error {
	clients := testAccClients()
	if ids.RouterID == 0 {
		return fmt.Errorf("router id was not captured")
	}
	if ids.NeighborID != 0 {
		if err := clients.V3.DeleteRouterVRFBGPNeighbor(ids.RouterID, ids.VRFID, ids.NeighborID); err != nil && !testAccRouterRoutingNotFound(err) {
			return err
		}
	}
	if ids.StaticRouteID != 0 {
		if err := clients.V3.DeleteRouterStaticRoute(ids.RouterID, ids.VRFID, ids.StaticRouteID); err != nil && !testAccRouterRoutingNotFound(err) {
			return err
		}
	}
	if ids.PrefixListID != 0 {
		if err := clients.V3.DeleteRouterPrefixList(ids.RouterID, ids.PrefixListID); err != nil && !testAccRouterRoutingNotFound(err) {
			return err
		}
	}
	if ids.VRFID != 0 {
		if err := clients.V3.DeleteRouterVRF(ids.RouterID, ids.VRFID); err != nil && !testAccRouterRoutingNotFound(err) {
			return err
		}
	}
	return nil
}

func (ids *testAccRouterVRFRoutingIDs) goneFromAPI() bool {
	clients := testAccClients()
	if ids.NeighborID != 0 {
		if _, err := clients.V3.GetRouterVRFBGPNeighbor(ids.RouterID, ids.VRFID, ids.NeighborID); err == nil || !testAccRouterRoutingNotFound(err) {
			return false
		}
	}
	if ids.StaticRouteID != 0 {
		if _, err := clients.V3.GetRouterStaticRoute(ids.RouterID, ids.VRFID, ids.StaticRouteID); err == nil || !testAccRouterRoutingNotFound(err) {
			return false
		}
	}
	if ids.PrefixListID != 0 {
		if _, err := clients.V3.GetRouterPrefixList(ids.RouterID, ids.PrefixListID); err == nil || !testAccRouterRoutingNotFound(err) {
			return false
		}
	}
	if ids.VRFID != 0 {
		if _, err := clients.V3.GetRouterVRF(ids.RouterID, ids.VRFID); err == nil || !testAccRouterRoutingNotFound(err) {
			return false
		}
	}
	return true
}

func testAccCheckRouterVRFAPI(address, name, description string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		vrf, err := testAccClients().V3.GetRouterVRF(routerID, vrfID)
		if err != nil {
			return err
		}
		if vrf.Name != name {
			return fmt.Errorf("VRF name: expected %q, got %q", name, vrf.Name)
		}
		if vrf.Description != description {
			return fmt.Errorf("VRF description: expected %q, got %q", description, vrf.Description)
		}
		if vrf.VrfID != vrfID {
			return fmt.Errorf("VRF id: expected %d, got %d", vrfID, vrf.VrfID)
		}
		return nil
	}
}

func testAccCheckRouterVRFBGPAPI(address, localASN string, networks []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		bgp, err := testAccClients().V3.GetRouterVRFBGP(routerID, vrfID)
		if err != nil {
			return err
		}
		if bgp.LocalAsn == nil || *bgp.LocalAsn != localASN {
			return fmt.Errorf("BGP local_asn: expected %q, got %#v", localASN, bgp.LocalAsn)
		}
		for _, network := range networks {
			if !testAccRouterBGPNetworkContains(bgp.Networks, network) {
				return fmt.Errorf("BGP networks missing %q: %#v", network, bgp.Networks)
			}
		}
		return nil
	}
}

func testAccCheckRouterPrefixListAPI(address string, want testAccRouterPrefixListWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, prefixListID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		list, err := testAccClients().V3.GetRouterPrefixList(routerID, prefixListID)
		if err != nil {
			return err
		}
		if list.Name != want.Name {
			return fmt.Errorf("prefix list name: expected %q, got %q", want.Name, list.Name)
		}
		if list.Description != want.Description {
			return fmt.Errorf("prefix list description: expected %q, got %q", want.Description, list.Description)
		}
		if list.IPVersion != want.IPVersion {
			return fmt.Errorf("prefix list ip_version: expected %d, got %d", want.IPVersion, list.IPVersion)
		}
		if len(list.Rules) != len(want.Rules) {
			return fmt.Errorf("prefix list rule count: expected %d, got %d", len(want.Rules), len(list.Rules))
		}
		for _, rule := range want.Rules {
			if !testAccRouterPrefixRulesContain(list.Rules, rule) {
				return fmt.Errorf("prefix list missing rule %#v: %#v", rule, list.Rules)
			}
		}
		return nil
	}
}

func testAccCheckRouterBGPNeighborAPI(address string, want testAccRouterBGPNeighborWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, neighborID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		neighbor, err := testAccClients().V3.GetRouterVRFBGPNeighbor(routerID, vrfID, neighborID)
		if err != nil {
			return err
		}
		if neighbor.Name != want.Name {
			return fmt.Errorf("BGP neighbor name: expected %q, got %q", want.Name, neighbor.Name)
		}
		if neighbor.Description != want.Description {
			return fmt.Errorf("BGP neighbor description: expected %q, got %q", want.Description, neighbor.Description)
		}
		if neighbor.Address != want.Address {
			return fmt.Errorf("BGP neighbor address: expected %q, got %q", want.Address, neighbor.Address)
		}
		if neighbor.ASN.Remote != want.RemoteASN {
			return fmt.Errorf("BGP neighbor remote_asn: expected %d, got %d", want.RemoteASN, neighbor.ASN.Remote)
		}
		if neighbor.EnabledIPVersion.IPv4 != want.IPv4Enabled || neighbor.EnabledIPVersion.IPv6 != want.IPv6Enabled {
			return fmt.Errorf("BGP neighbor IP versions: expected %t/%t, got %t/%t", want.IPv4Enabled, want.IPv6Enabled, neighbor.EnabledIPVersion.IPv4, neighbor.EnabledIPVersion.IPv6)
		}
		if neighbor.DoAsOverride != want.DoAsOverride || neighbor.DoNextHelpSelf != want.DoNextHopSelf || neighbor.IsShutdown != want.IsShutdown {
			return fmt.Errorf("BGP neighbor flags mismatch: %#v", neighbor)
		}
		if neighbor.Import == nil || neighbor.Import.DoDefaultDrop != want.ImportDefaultDrop {
			return fmt.Errorf("BGP neighbor import default drop: expected %t, got %#v", want.ImportDefaultDrop, neighbor.Import)
		}
		if neighbor.Export == nil || neighbor.Export.DoDefaultDrop != want.ExportDefaultDrop {
			return fmt.Errorf("BGP neighbor export default drop: expected %t, got %#v", want.ExportDefaultDrop, neighbor.Export)
		}
		if !testAccRouterImportRuleMatches(neighbor.Import.Rules, want.ImportAction, want.SetLocalPreference) {
			return fmt.Errorf("BGP neighbor import rules mismatch: %#v", neighbor.Import.Rules)
		}
		if !testAccRouterExportRuleMatches(neighbor.Export.Rules, want.ExportAction, want.PrependLastAsn) {
			return fmt.Errorf("BGP neighbor export rules mismatch: %#v", neighbor.Export.Rules)
		}
		return nil
	}
}

func testAccCheckRouterStaticRouteAPI(address string, want testAccRouterStaticRouteWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, routeID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		route, err := testAccClients().V3.GetRouterStaticRoute(routerID, vrfID, routeID)
		if err != nil {
			return err
		}
		if route.Network != want.Network {
			return fmt.Errorf("static route network: expected %q, got %q", want.Network, route.Network)
		}
		if route.Description != want.Description {
			return fmt.Errorf("static route description: expected %q, got %q", want.Description, route.Description)
		}
		if route.Distance == nil || *route.Distance != want.Distance {
			return fmt.Errorf("static route distance: expected %d, got %#v", want.Distance, route.Distance)
		}
		if route.Via.NextHop != want.NextHop {
			return fmt.Errorf("static route next_hop: expected %q, got %q", want.NextHop, route.Via.NextHop)
		}
		return nil
	}
}

func testAccCheckRouterVRFRoutingDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "netactuate_router_vrf":
			routerID, vrfID, err := parseRouterVRFImportID(rs.Primary.ID)
			if err != nil {
				return err
			}
			if _, err := clients.V3.GetRouterVRF(routerID, vrfID); err == nil {
				return fmt.Errorf("VRF still exists: %s", rs.Primary.ID)
			} else if !testAccRouterRoutingNotFound(err) {
				return err
			}
		case "netactuate_router_vrf_bgp":
			routerID, vrfID, err := parseRouterVRFBGPImportID(rs.Primary.ID)
			if err != nil {
				return err
			}
			if _, err := clients.V3.GetRouterVRFBGP(routerID, vrfID); err == nil {
				return fmt.Errorf("BGP config still exists: %s", rs.Primary.ID)
			} else if !testAccRouterRoutingNotFound(err) {
				return err
			}
		case "netactuate_router_prefix_list":
			routerID, prefixListID, err := parsePrefixListID(rs.Primary.ID)
			if err != nil {
				return err
			}
			if _, err := clients.V3.GetRouterPrefixList(routerID, prefixListID); err == nil {
				return fmt.Errorf("prefix list still exists: %s", rs.Primary.ID)
			} else if !testAccRouterRoutingNotFound(err) {
				return err
			}
		case "netactuate_router_vrf_bgp_neighbor":
			routerID, vrfID, neighborID, err := parseRouterVRFBGPNeighborImportID(rs.Primary.ID)
			if err != nil {
				return err
			}
			if _, err := clients.V3.GetRouterVRFBGPNeighbor(routerID, vrfID, neighborID); err == nil {
				return fmt.Errorf("BGP neighbor still exists: %s", rs.Primary.ID)
			} else if !testAccRouterRoutingNotFound(err) {
				return err
			}
		case "netactuate_router_static_route":
			routerID, vrfID, routeID, err := parseStaticRouteID(rs.Primary.ID)
			if err != nil {
				return err
			}
			if _, err := clients.V3.GetRouterStaticRoute(routerID, vrfID, routeID); err == nil {
				return fmt.Errorf("static route still exists: %s", rs.Primary.ID)
			} else if !testAccRouterRoutingNotFound(err) {
				return err
			}
		}
	}
	return testAccCheckRouterDestroy(s)
}

func testAccRouterRoutingImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[address]
		if !ok || rs.Primary == nil {
			return "", fmt.Errorf("not found: %s", address)
		}
		return rs.Primary.ID, nil
	}
}

func testAccTripleID(s *terraform.State, address string) (int, int, int, error) {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return 0, 0, 0, fmt.Errorf("not found: %s", address)
	}
	parts := strings.SplitN(rs.Primary.ID, "/", 3)
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("%s has invalid triple ID %q", address, rs.Primary.ID)
	}
	first, second, third, err := parseTripleIDParts(address, parts)
	if err != nil {
		return 0, 0, 0, err
	}
	return first, second, third, nil
}

func parseTripleIDParts(address string, parts []string) (int, int, int, error) {
	first, err := parsePositiveIDPart(address, "first", parts[0])
	if err != nil {
		return 0, 0, 0, err
	}
	second, err := parsePositiveIDPart(address, "second", parts[1])
	if err != nil {
		return 0, 0, 0, err
	}
	third, err := parsePositiveIDPart(address, "third", parts[2])
	if err != nil {
		return 0, 0, 0, err
	}
	return first, second, third, nil
}

func parsePositiveIDPart(address, name, raw string) (int, error) {
	out, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s has invalid %s ID %q: %w", address, name, raw, err)
	}
	return out, nil
}

func testAccRouterRoutingNotFound(err error) bool {
	if err == nil {
		return false
	}
	if gona.IsV3NotFound(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "does not exist")
}

func testAccWaitGoneFromRouterRoutingAPI(t *testing.T, what string, gone func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if gone() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s still present 2 minutes after it was deleted", what)
		}
		time.Sleep(5 * time.Second)
	}
}

func testAccRouterBGPNetworkContains(networks []gona.RouterVRFBGPNetwork, want string) bool {
	for _, network := range networks {
		if network.Subnet == want {
			return true
		}
	}
	return false
}

func testAccRouterPrefixRulesContain(rules []gona.PrefixListRule, want gona.PrefixListRule) bool {
	for _, rule := range rules {
		if rule.Action == want.Action && rule.Prefix == want.Prefix {
			return true
		}
	}
	return false
}

func testAccRouterImportRuleMatches(rules []gona.BGPNeighborRouteMapRule, action string, localPreference int) bool {
	for _, rule := range rules {
		if rule.Action != action {
			continue
		}
		if localPreference == 0 {
			return rule.SetLocalPreference == nil
		}
		if rule.SetLocalPreference != nil && *rule.SetLocalPreference == localPreference {
			return true
		}
	}
	return false
}

func testAccRouterExportRuleMatches(rules []gona.BGPNeighborRouteMapRule, action string, prependLastAsn int) bool {
	for _, rule := range rules {
		if rule.Action != action {
			continue
		}
		if prependLastAsn == 0 {
			return rule.PrependLastAsn == nil
		}
		if rule.PrependLastAsn != nil && *rule.PrependLastAsn == prependLastAsn {
			return true
		}
	}
	return false
}

func testAccRouterVRFRoutingRouterOnlyConfig(name string, locationID int, plan string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_router" "test" {
  name        = %q
  description = %q
  location_id = %d
  plan        = %q
}
`, name, name, locationID, plan)
}

func testAccRouterVRFRoutingConfig(name string, locationID int, plan string, updated bool) string {
	vrfName := name + " vrf"
	vrfDescription := name + " vrf initial"
	prefixName := name + "-pl"
	prefixDescription := name + " prefix initial"
	bgpASN := "64512"
	neighborName := name + "-neighbor"
	neighborDescription := name + " neighbor initial"
	neighborAddress := "192.0.2.2"
	neighborASN := 64513
	doAsOverride := false
	doNextHopSelf := true
	isShutdown := false
	importDefaultDrop := true
	exportDefaultDrop := false
	importAction := "permit"
	// vAPI3 lists 'next' as an allowed route-map rule action and then rejects it downstream with
	// "Action must be specified for route-map export-<id> rule 100", because the config generator
	// needs a permit or deny on every rule. That is a platform defect. Use 'deny'
	// here so the remaining steps run, and give 'next' its own test once the platform either
	// renders it or stops advertising it.
	exportAction := "deny"
	localPreference := 200
	prependLastAsn := 2
	staticNetwork := "203.0.113.0/24"
	staticDescription := name + " static initial"
	staticDistance := 10
	staticNextHop := "192.0.2.1"
	prefixRules := `
  rule {
    action = "permit"
    prefix = "198.51.100.0/24"
  }`
	bgpNetworks := `
  networks {
    subnet = "198.51.100.0/24"
  }`

	if updated {
		vrfName = name + " vrf updated"
		vrfDescription = name + " vrf updated"
		prefixName = name + "-pl-updated"
		prefixDescription = name + " prefix updated"
		bgpASN = "64514"
		neighborName = name + "-neighbor-updated"
		neighborDescription = name + " neighbor updated"
		neighborAddress = "192.0.2.3"
		neighborASN = 64515
		doAsOverride = true
		doNextHopSelf = false
		isShutdown = true
		importDefaultDrop = false
		exportDefaultDrop = true
		importAction = "deny"
		exportAction = "permit"
		localPreference = 300
		prependLastAsn = 3
		staticNetwork = "198.51.100.0/24"
		staticDescription = name + " static updated"
		staticDistance = 20
		staticNextHop = "192.0.2.254"
		prefixRules = `
  rule {
    action = "deny"
    prefix = "192.0.2.0/24"
  }

  rule {
    action = "permit"
    prefix = "198.51.100.0/24"
  }`
		bgpNetworks = `
  networks {
    subnet = "198.51.100.0/24"
  }

  networks {
    subnet = "203.0.113.0/24"
  }`
	}

	// The populated neighbor, static route and prefix list shapes below are from
	// vapi3 getter.parent.js and update.parent.js handlers, not from a live row.
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

resource "netactuate_router_prefix_list" "test" {
  router_id   = netactuate_router.test.router_id
  name        = %q
  description = %q
  ip_version  = 4
%s
}

resource "netactuate_router_vrf_bgp" "test" {
  router_id = netactuate_router.test.router_id
  vrf_id    = netactuate_router_vrf.test.vrf_id
  local_asn = %q
%s
}

resource "netactuate_router_vrf_bgp_neighbor" "test" {
  router_id            = netactuate_router.test.router_id
  vrf_id               = netactuate_router_vrf.test.vrf_id
  name                 = %q
  description          = %q
  address              = %q
  remote_asn           = %d
  ipv4_enabled         = true
  ipv6_enabled         = false
  do_as_override       = %t
  do_next_hop_self     = %t
  is_shutdown          = %t
  import_default_drop  = %t
  export_default_drop  = %t

  import_rules {
    prefix_list_id       = netactuate_router_prefix_list.test.prefix_list_id
    action               = %q
    set_local_preference = %d
  }

  export_rules {
    prefix_list_id   = netactuate_router_prefix_list.test.prefix_list_id
    action           = %q
    prepend_last_asn = %d
  }
}

resource "netactuate_router_static_route" "test" {
  router_id   = netactuate_router.test.router_id
  vrf_id      = netactuate_router_vrf.test.vrf_id
  network     = %q
  description = %q
  distance    = %d
  next_hop    = %q
}
`, name, name, locationID, plan, vrfName, vrfDescription, prefixName, prefixDescription, prefixRules, bgpASN, bgpNetworks, neighborName, neighborDescription, neighborAddress, neighborASN, doAsOverride, doNextHopSelf, isShutdown, importDefaultDrop, exportDefaultDrop, importAction, localPreference, exportAction, prependLastAsn, staticNetwork, staticDescription, staticDistance, staticNextHop)
}
