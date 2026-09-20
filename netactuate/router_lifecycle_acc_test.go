//go:build acctest

package netactuate

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

// Gaps: netactuate_router_vrf_bgp, netactuate_router_ntp,
// netactuate_router_vrf_dhcp and netactuate_router_ipsec are configuration
// overlays with update or reset semantics, but no standalone V3 delete method.
// netactuate_router_routing_view is query-only, has no standalone delete, and
// its importer cannot reconstruct the view request fields for lifecycle import
// verification.

func TestAccNetactuateRouterLifecycle(t *testing.T) {
	name := testAccName("router-lifecycle")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	createConfig := testAccRouterConfigID(name, locationID, plan)
	modifiedConfig := testAccRouterConfigID(name+"-updated", locationID, plan)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_router", "netactuate_router.test"),
			testAccCheckRouterAPI("netactuate_router.test", name, name),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_router", "netactuate_router.test"),
			testAccCheckRouterAPI("netactuate_router.test", name+"-updated", name+"-updated"),
		), testAccLifecycleOptions{
			ImportStateIdFunc:       testAccRouterLifecycleImportID("netactuate_router.test"),
			ImportStateVerifyIgnore: []string{"package_id", "plan"},
		}),
	})
}

func TestAccNetactuateRouterOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("router-oob")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterDestroy,
		Steps: []resource.TestStep{{
			Config: testAccRouterConfigID(name, locationID, plan),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_router", "netactuate_router.test"),
				testAccCheckRouterAPI("netactuate_router.test", name, name),
				testAccDeleteRouterOutOfBand("netactuate_router.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateRouterVRFLifecycle(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-vrf-lifecycle")
	createConfig := testAccRouterLifecycleVRFConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterLifecycleVRFConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterVRFRoutingDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_vrf.test",
			testAccCheckRouterVRFAPI("netactuate_router_vrf.test", name+" vrf", name+" vrf initial"),
			testAccCheckRouterVRFAPI("netactuate_router_vrf.test", name+" vrf updated", name+" vrf updated"),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_vrf.test")}),
	})
}

func TestAccNetactuateRouterVRFOutOfBandDeleteIdempotent(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-vrf-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterVRFRoutingDestroy,
		Steps: []resource.TestStep{{
			Config: testAccRouterLifecycleVRFConfig(name, locationID, plan, false),
			Check: resource.ComposeTestCheckFunc(
				testAccCheckRouterVRFAPI("netactuate_router_vrf.test", name+" vrf", name+" vrf initial"),
				testAccDeleteRouterVRFOutOfBand("netactuate_router_vrf.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateRouterVRFBGPLifecycle(t *testing.T) {
	t.Skip("skipped: the platform rejects a VRF BGP configuration apply with HTTP 400; re-enable when the API is fixed")
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-vrf-bgp-lifecycle")
	createConfig := testAccRouterVRFRoutingConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterVRFRoutingConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterVRFRoutingDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_vrf_bgp.test",
			testAccCheckRouterVRFBGPAPI("netactuate_router_vrf_bgp.test", "64512", []string{"198.51.100.0/24"}),
			testAccCheckRouterVRFBGPAPI("netactuate_router_vrf_bgp.test", "64514", []string{"198.51.100.0/24", "203.0.113.0/24"}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_vrf_bgp.test")}),
	})
}

func TestAccNetactuateRouterVRFBGPNeighborLifecycle(t *testing.T) {
	t.Skip("skipped: the platform rejects a VRF BGP configuration apply with HTTP 400; re-enable when the API is fixed")
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-bgp-neighbor-lifecycle")
	createConfig := testAccRouterVRFRoutingConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterVRFRoutingConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterVRFRoutingDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_vrf_bgp_neighbor.test",
			testAccCheckRouterBGPNeighborAPI("netactuate_router_vrf_bgp_neighbor.test", testAccRouterBGPNeighborWant{Name: name + "-neighbor", Description: name + " neighbor initial", Address: "192.0.2.2", RemoteASN: 64513, IPv4Enabled: true, DoNextHopSelf: true, ImportDefaultDrop: true, ExportAction: "deny", ImportAction: "permit", SetLocalPreference: 200, PrependLastAsn: 2}),
			testAccCheckRouterBGPNeighborAPI("netactuate_router_vrf_bgp_neighbor.test", testAccRouterBGPNeighborWant{Name: name + "-neighbor-updated", Description: name + " neighbor updated", Address: "192.0.2.3", RemoteASN: 64515, IPv4Enabled: true, DoAsOverride: true, IsShutdown: true, ExportDefaultDrop: true, ImportAction: "deny", ExportAction: "permit", SetLocalPreference: 300, PrependLastAsn: 3}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_vrf_bgp_neighbor.test")}),
	})
}

func TestAccNetactuateRouterVRFBGPNeighborOutOfBandDeleteIdempotent(t *testing.T) {
	t.Skip("skipped: the platform rejects a VRF BGP configuration apply with HTTP 400; re-enable when the API is fixed")
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-bgp-neighbor-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterVRFRoutingDestroy,
		Steps: []resource.TestStep{{
			Config: testAccRouterVRFRoutingConfig(name, locationID, plan, false),
			Check: resource.ComposeTestCheckFunc(
				testAccDeleteRouterBGPNeighborOutOfBand("netactuate_router_vrf_bgp_neighbor.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateRouterStaticRouteLifecycle(t *testing.T) {
	t.Skip("skipped: the platform rejects a VRF BGP configuration apply with HTTP 400; re-enable when the API is fixed")
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-static-route-lifecycle")
	createConfig := testAccRouterVRFRoutingConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterVRFRoutingConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterVRFRoutingDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_static_route.test",
			testAccCheckRouterStaticRouteAPI("netactuate_router_static_route.test", testAccRouterStaticRouteWant{Network: "203.0.113.0/24", Description: name + " static initial", Distance: 10, NextHop: "192.0.2.1"}),
			testAccCheckRouterStaticRouteAPI("netactuate_router_static_route.test", testAccRouterStaticRouteWant{Network: "198.51.100.0/24", Description: name + " static updated", Distance: 20, NextHop: "192.0.2.254"}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_static_route.test")}),
	})
}

func TestAccNetactuateRouterStaticRouteOutOfBandDeleteIdempotent(t *testing.T) {
	t.Skip("skipped: the platform rejects a VRF BGP configuration apply with HTTP 400; re-enable when the API is fixed")
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-static-route-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterVRFRoutingDestroy,
		Steps: []resource.TestStep{{
			Config:             testAccRouterVRFRoutingConfig(name, locationID, plan, false),
			Check:              testAccDeleteRouterStaticRouteOutOfBand("netactuate_router_static_route.test"),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateRouterPrefixListLifecycle(t *testing.T) {
	t.Skip("skipped: the platform rejects a VRF BGP configuration apply with HTTP 400; re-enable when the API is fixed")
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-prefix-list-lifecycle")
	createConfig := testAccRouterVRFRoutingConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterVRFRoutingConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterVRFRoutingDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_prefix_list.test",
			testAccCheckRouterPrefixListAPI("netactuate_router_prefix_list.test", testAccRouterPrefixListWant{Name: name + "-pl", Description: name + " prefix initial", IPVersion: 4, Rules: []gona.PrefixListRule{{Action: "permit", Prefix: "198.51.100.0/24"}}}),
			testAccCheckRouterPrefixListAPI("netactuate_router_prefix_list.test", testAccRouterPrefixListWant{Name: name + "-pl-updated", Description: name + " prefix updated", IPVersion: 4, Rules: []gona.PrefixListRule{{Action: "deny", Prefix: "192.0.2.0/24"}, {Action: "permit", Prefix: "198.51.100.0/24"}}}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_prefix_list.test")}),
	})
}

func TestAccNetactuateRouterPrefixListOutOfBandDeleteIdempotent(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-prefix-list-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterVRFRoutingDestroy,
		Steps: []resource.TestStep{{
			Config:             testAccRouterLifecyclePrefixListConfig(name, locationID, plan, false),
			Check:              testAccDeleteRouterPrefixListOutOfBand("netactuate_router_prefix_list.test"),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateRouterNTPLifecycle(t *testing.T) {
	t.Skip("skipped: shares an interface config whose WireGuard peer read-back returns HTTP 500 on the platform; re-enable when fixed, or split the peer into its own config")
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-ntp-lifecycle")
	createConfig := testAccRouterInterfaceServicesConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterInterfaceServicesConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_ntp.test",
			testAccCheckRouterNTPAPI("netactuate_router_ntp.test", true, []string{"time.cloudflare.com"}),
			testAccCheckRouterNTPAPI("netactuate_router_ntp.test", true, []string{"time.google.com"}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_ntp.test")}),
	})
}

func TestAccNetactuateRouterVRFDHCPLifecycle(t *testing.T) {
	t.Skip("skipped: shares an interface config whose WireGuard peer read-back returns HTTP 500 on the platform; re-enable when fixed, or split the peer into its own config")
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-dhcp-lifecycle")
	createConfig := testAccRouterInterfaceServicesConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterInterfaceServicesConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_vrf_dhcp.test",
			testAccCheckRouterDHCPAPI("netactuate_router_vrf_dhcp.test", testAccRouterDHCPWant{Enabled: true, Subnet: "198.51.100.0/24", LeaseTimeout: 3600, DoPingCheck: true, DefaultRouterAddress: "198.51.100.1", ClientDomainName: "example.test", RangeFirst: "198.51.100.10", RangeLast: "198.51.100.20", DomainNameServers: []string{"198.51.100.53"}, NTPServers: []string{"198.51.100.123"}, StaticRouteNetwork: "203.0.113.0/24", StaticRouteNextHop: "198.51.100.1"}),
			testAccCheckRouterDHCPAPI("netactuate_router_vrf_dhcp.test", testAccRouterDHCPWant{Enabled: true, Subnet: "198.51.100.0/24", LeaseTimeout: 7200, DefaultRouterAddress: "198.51.100.1", ClientDomainName: "updated.example.test", RangeFirst: "198.51.100.30", RangeLast: "198.51.100.40", DomainNameServers: []string{"198.51.100.54"}, NTPServers: []string{"198.51.100.124"}, StaticRouteNetwork: "203.0.113.0/24", StaticRouteNextHop: "198.51.100.1"}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_vrf_dhcp.test")}),
	})
}

func TestAccNetactuateRouterVRFInterfaceLifecycle(t *testing.T) {
	t.Skip("skipped: shares an interface config whose WireGuard peer read-back returns HTTP 500 on the platform; re-enable when fixed, or split the peer into its own config")
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-interface-lifecycle")
	createConfig := testAccRouterInterfaceServicesConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterInterfaceServicesConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_vrf_interface.dummy",
			testAccCheckRouterInterfaceAPI("netactuate_router_vrf_interface.dummy", testAccRouterInterfaceWant{Type: "dummy", Name: name + "-dummy", Description: name + " dummy initial", IPv4CIDR: "192.0.2.1/32"}),
			testAccCheckRouterInterfaceAPI("netactuate_router_vrf_interface.dummy", testAccRouterInterfaceWant{Type: "dummy", Name: name + "-dummy-updated", Description: name + " dummy updated", IPv4CIDR: "192.0.2.11/32"}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_vrf_interface.dummy")}),
	})
}

func TestAccNetactuateRouterVRFInterfaceOutOfBandDeleteIdempotent(t *testing.T) {
	t.Skip("skipped: shares an interface config whose WireGuard peer read-back returns HTTP 500 on the platform; re-enable when fixed, or split the peer into its own config")
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-interface-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: []resource.TestStep{{
			Config:             testAccRouterInterfaceServicesConfig(name, locationID, plan, false),
			Check:              testAccDeleteRouterInterfaceOutOfBand("netactuate_router_vrf_interface.dummy"),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateRouterVRFInterfaceWireguardPeerLifecycle(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-wg-peer-lifecycle")
	createConfig := testAccRouterLifecycleWireguardPeerConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterLifecycleWireguardPeerConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_vrf_interface_wireguard_peer.test",
			testAccCheckWireguardPeerAPI("netactuate_router_vrf_interface_wireguard_peer.test", testAccWireguardPeerWant{Name: name + "-wg-peer", Description: name + " wireguard peer initial", AllowedIPs: []string{"198.51.100.0/24"}}),
			testAccCheckWireguardPeerAPI("netactuate_router_vrf_interface_wireguard_peer.test", testAccWireguardPeerWant{Name: name + "-wg-peer-updated", Description: name + " wireguard peer updated", AllowedIPs: []string{"203.0.113.0/24"}}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_vrf_interface_wireguard_peer.test")}),
	})
}

func TestAccNetactuateRouterVRFInterfaceWireguardPeerOutOfBandDeleteIdempotent(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-wg-peer-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: []resource.TestStep{{
			Config:             testAccRouterLifecycleWireguardPeerConfig(name, locationID, plan, false),
			Check:              testAccDeleteRouterWireguardPeerOutOfBand("netactuate_router_vrf_interface_wireguard_peer.test"),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateRouterVRFTunnelLifecycle(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-tunnel-lifecycle")
	createConfig := testAccRouterTunnelNATConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterTunnelNATConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_vrf_tunnel.test",
			testAccCheckRouterTunnelAPI("netactuate_router_vrf_tunnel.test", testAccRouterTunnelWant{Name: name + "-tunnel", Description: name + " tunnel initial", MTU: "1400", IPKey: 101, IPv4CIDR: "192.0.2.20/31", EndpointRemote: "198.51.100.1", IPVersion: 4}),
			testAccCheckRouterTunnelAPI("netactuate_router_vrf_tunnel.test", testAccRouterTunnelWant{Name: name + "-tunnel-updated", Description: name + " tunnel updated", MTU: "1410", IPKey: 102, IPv4CIDR: "192.0.2.22/31", EndpointRemote: "198.51.100.2", IPVersion: 4}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_vrf_tunnel.test")}),
	})
}

func TestAccNetactuateRouterVRFTunnelOutOfBandDeleteIdempotent(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-tunnel-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: []resource.TestStep{{
			Config:             testAccRouterTunnelNATConfig(name, locationID, plan, false),
			Check:              testAccDeleteRouterTunnelOutOfBand("netactuate_router_vrf_tunnel.test"),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateRouterVRFSNATRuleLifecycle(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-snat-lifecycle")
	createConfig := testAccRouterTunnelNATConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterTunnelNATConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_vrf_snat_rule.test",
			testAccCheckRouterSNATAPI("netactuate_router_vrf_snat_rule.test", testAccRouterNATWant{Protocol: "TCP", Description: name + " snat initial", MatchNetwork: "192.0.2.0/24", MatchPortStart: 1000, MatchPortEnd: 1005, TranslateNetwork: "198.51.100.10/32", TranslateStart: 2000, TranslateEnd: 2005}),
			testAccCheckRouterSNATAPI("netactuate_router_vrf_snat_rule.test", testAccRouterNATWant{Protocol: "TCP", Description: name + " snat updated", MatchNetwork: "192.0.2.0/24", MatchPortStart: 1010, MatchPortEnd: 1015, TranslateNetwork: "198.51.100.11/32", TranslateStart: 2010, TranslateEnd: 2015}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_vrf_snat_rule.test")}),
	})
}

func TestAccNetactuateRouterVRFSNATRuleOutOfBandDeleteIdempotent(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-snat-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: []resource.TestStep{{
			Config:             testAccRouterTunnelNATConfig(name, locationID, plan, false),
			Check:              testAccDeleteRouterSNATRuleOutOfBand("netactuate_router_vrf_snat_rule.test"),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateRouterVRFDNATRuleLifecycle(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-dnat-lifecycle")
	createConfig := testAccRouterTunnelNATConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterTunnelNATConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_vrf_dnat_rule.test",
			testAccCheckRouterDNATAPI("netactuate_router_vrf_dnat_rule.test", testAccRouterNATWant{Protocol: "UDP", Description: name + " dnat initial", MatchNetwork: "198.51.100.20/32", MatchPortStart: 3000, MatchPortEnd: 3005, TranslateNetwork: "192.0.2.20/32", TranslateStart: 4000, TranslateEnd: 4005}),
			testAccCheckRouterDNATAPI("netactuate_router_vrf_dnat_rule.test", testAccRouterNATWant{Protocol: "UDP", Description: name + " dnat updated", MatchNetwork: "198.51.100.21/32", MatchPortStart: 3010, MatchPortEnd: 3015, TranslateNetwork: "192.0.2.21/32", TranslateStart: 4010, TranslateEnd: 4015}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_vrf_dnat_rule.test")}),
	})
}

func TestAccNetactuateRouterVRFDNATRuleOutOfBandDeleteIdempotent(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-dnat-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: []resource.TestStep{{
			Config:             testAccRouterTunnelNATConfig(name, locationID, plan, false),
			Check:              testAccDeleteRouterDNATRuleOutOfBand("netactuate_router_vrf_dnat_rule.test"),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateRouterIPSecLifecycle(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-ipsec-lifecycle")
	createConfig := testAccRouterIPSecConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterIPSecConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_ipsec.test",
			testAccCheckRouterIPSecAPI("netactuate_router_ipsec.test", testAccRouterIPSecWant{AutoRenegotiation: true, KeyExchange: 2, IKELifetime: 28800, DHGroup: 14, IKEEncryption: "aes256", IKEHash: "sha256", IKEPRF: "prfsha256", ESPLifetime: 3600, ESPEncryption: "aes256", ESPHash: "sha256"}),
			testAccCheckRouterIPSecAPI("netactuate_router_ipsec.test", testAccRouterIPSecWant{KeyExchange: 1, IKELifetime: 14400, DHGroup: 15, IKEEncryption: "aes128", IKEHash: "sha1", IKEPRF: "prfsha1", ESPLifetime: 1800, ESPEncryption: "aes128", ESPHash: "sha1"}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_ipsec.test")}),
	})
}

func TestAccNetactuateRouterVRFIPSecPeerLifecycle(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-ipsec-peer-lifecycle")
	createConfig := testAccRouterIPSecConfig(name, locationID, plan, false)
	modifiedConfig := testAccRouterIPSecConfig(name, locationID, plan, true)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_router_vrf_ipsec_peer.test",
			testAccCheckRouterIPSecPeerAPI("netactuate_router_vrf_ipsec_peer.test", testAccRouterIPSecPeerWant{Name: name + "-peer", Description: name + " peer initial", RemoteID: "198.51.100.10", DoInitiateConnection: true, PeerAddress: "198.51.100.10", OverlayIPv4: "192.0.2.40/31"}),
			testAccCheckRouterIPSecPeerAPI("netactuate_router_vrf_ipsec_peer.test", testAccRouterIPSecPeerWant{Name: name + "-peer-updated", Description: name + " peer updated", RemoteID: "198.51.100.11", DoInitiateConnection: true, PeerAddress: "198.51.100.11", OverlayIPv4: "192.0.2.42/31"}),
			testAccLifecycleOptions{ImportStateIdFunc: testAccRouterLifecycleImportID("netactuate_router_vrf_ipsec_peer.test"), ImportStateVerifyIgnore: []string{"psk_secret"}}),
	})
}

func TestAccNetactuateRouterVRFIPSecPeerOutOfBandDeleteIdempotent(t *testing.T) {
	name, locationID, plan := testAccRouterLifecycleInputs(t, "router-ipsec-peer-oob")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccRouterLifecyclePreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: []resource.TestStep{{
			Config:             testAccRouterIPSecConfig(name, locationID, plan, false),
			Check:              testAccDeleteRouterIPSecPeerOutOfBand("netactuate_router_vrf_ipsec_peer.test"),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func testAccRouterLifecyclePreCheck(t *testing.T) {
	testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID", "NETACTUATE_ACC_PLAN")
}

func testAccRouterLifecycleInputs(t *testing.T, prefix string) (string, int, string) {
	return testAccName(prefix), testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID"), testAccEnvString(t, "NETACTUATE_ACC_PLAN")
}

func testAccRouterLifecycleImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return testAccStateID(s, address)
	}
}

func testAccCheckRouterAPI(address, name, description string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		router, err := testAccClients().V3.GetRouter(id)
		if err != nil {
			return err
		}
		if router.Name != name {
			return fmt.Errorf("router name: expected %q, got %q", name, router.Name)
		}
		if router.Description == nil || *router.Description != description {
			return fmt.Errorf("router description: expected %q, got %#v", description, router.Description)
		}
		return nil
	}
}

func testAccRouterLifecycleVRFConfig(name string, locationID int, plan string, updated bool) string {
	vrfName := name + " vrf"
	vrfDescription := name + " vrf initial"
	if updated {
		vrfName = name + " vrf updated"
		vrfDescription = name + " vrf updated"
	}
	return testAccRouterServiceRouterOnlyConfig(name, locationID, plan) + fmt.Sprintf(`
resource "netactuate_router_vrf" "test" {
  router_id   = netactuate_router.test.router_id
  name        = %q
  description = %q
}
`, vrfName, vrfDescription)
}

func testAccRouterLifecyclePrefixListConfig(name string, locationID int, plan string, updated bool) string {
	prefixName := name + "-pl"
	prefixDescription := name + " prefix initial"
	rules := `
  rule {
    action = "permit"
    prefix = "198.51.100.0/24"
  }`
	if updated {
		prefixName = name + "-pl-updated"
		prefixDescription = name + " prefix updated"
		rules = `
  rule {
    action = "deny"
    prefix = "192.0.2.0/24"
  }

  rule {
    action = "permit"
    prefix = "198.51.100.0/24"
  }`
	}
	return testAccRouterServiceRouterOnlyConfig(name, locationID, plan) + fmt.Sprintf(`
resource "netactuate_router_prefix_list" "test" {
  router_id   = netactuate_router.test.router_id
  name        = %q
  description = %q
  ip_version  = 4
%s
}
`, prefixName, prefixDescription, rules)
}

func testAccRouterLifecycleWireguardPeerConfig(name string, locationID int, plan string, updated bool) string {
	peerName := name + "-wg-peer"
	peerDescription := name + " wireguard peer initial"
	peerAllowedIP := "198.51.100.0/24"
	if updated {
		peerName = name + "-wg-peer-updated"
		peerDescription = name + " wireguard peer updated"
		peerAllowedIP = "203.0.113.0/24"
	}
	return testAccRouterServiceRouterOnlyConfig(name, locationID, plan) + fmt.Sprintf(`
resource "netactuate_router_vrf" "test" {
  router_id   = netactuate_router.test.router_id
  name        = %q
  description = %q
}

resource "netactuate_router_vrf_interface" "wireguard" {
  router_id      = netactuate_router.test.router_id
  vrf_id         = netactuate_router_vrf.test.vrf_id
  type           = "wireguard"
  name           = %q
  description    = %q
  ipv4_cidr      = "192.0.2.3/32"
  wireguard_port = 51820
}

resource "netactuate_router_vrf_interface_wireguard_peer" "test" {
  router_id    = netactuate_router.test.router_id
  vrf_id       = netactuate_router_vrf.test.vrf_id
  interface_id = netactuate_router_vrf_interface.wireguard.interface_id
  name         = %q
  description  = %q

  allowed_ips {
    network = %q
  }
}
`, name+" vrf", name+" vrf", name+"-wireguard", name+" wireguard", peerName, peerDescription, peerAllowedIP)
}

func testAccDeleteRouterOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteRouter(routerID)
	}
}

func testAccDeleteRouterVRFOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteRouterVRF(routerID, vrfID)
	}
}

func testAccDeleteRouterBGPNeighborOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, neighborID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteRouterVRFBGPNeighbor(routerID, vrfID, neighborID)
	}
}

func testAccDeleteRouterStaticRouteOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, routeID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteRouterStaticRoute(routerID, vrfID, routeID)
	}
}

func testAccDeleteRouterPrefixListOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, prefixListID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteRouterPrefixList(routerID, prefixListID)
	}
}

func testAccDeleteRouterInterfaceOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, interfaceID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteRouterVRFInterface(routerID, vrfID, interfaceID)
	}
}

func testAccDeleteRouterWireguardPeerOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, interfaceID, peerID, err := testAccQuadID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteRouterVRFInterfaceWireguardPeer(routerID, vrfID, interfaceID, peerID)
	}
}

func testAccDeleteRouterTunnelOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, tunnelID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteRouterVRFTunnel(routerID, vrfID, tunnelID)
	}
}

func testAccDeleteRouterSNATRuleOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, ruleID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteRouterVRFSNATRule(routerID, vrfID, ruleID)
	}
}

func testAccDeleteRouterDNATRuleOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, ruleID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteRouterVRFDNATRule(routerID, vrfID, ruleID)
	}
}

func testAccDeleteRouterIPSecPeerOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, peerID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		return testAccClients().V3.DeleteRouterVRFIPSecPeer(routerID, vrfID, peerID)
	}
}

// A cloud router needs at least 2 GB RAM and 2 vCPU to finish provisioning.
// VR1G (1 GB, 2 vCPU) is below the RAM floor, so create must refuse it before
// the router is built. The apply errors during plan resolution, so nothing is
// created and there is nothing to clean up.
func TestAccNetactuateRouterRejectsUndersizedPlan(t *testing.T) {
	name := testAccName("router-undersized")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRouterConfigID(name, locationID, "VR1G"),
				ExpectError: regexp.MustCompile(`too small for a cloud router`),
			},
		},
	})
}
