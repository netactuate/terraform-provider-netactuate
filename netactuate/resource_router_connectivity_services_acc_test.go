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

func TestAccNetactuateRouterInterfaceServices_importUpdateRemoveAndOutOfBandDelete(t *testing.T) {
	name := testAccName("router-iface-services")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	initialConfig := testAccRouterInterfaceServicesConfig(name, locationID, plan, false)
	updateConfig := testAccRouterInterfaceServicesConfig(name, locationID, plan, true)
	routerOnlyConfig := testAccRouterServiceRouterOnlyConfig(name, locationID, plan)
	ids := &testAccRouterInterfaceServicesIDs{}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_PLAN",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_router", "netactuate_router.test"),
					ids.capture,
					testAccCheckRouterInterfaceAPI("netactuate_router_vrf_interface.dummy", testAccRouterInterfaceWant{
						Type:        "dummy",
						Name:        name + "-dummy",
						Description: name + " dummy initial",
						IPv4CIDR:    "192.0.2.1/32",
					}),
					testAccCheckRouterInterfaceAPI("netactuate_router_vrf_interface.loopback", testAccRouterInterfaceWant{
						Type:        "dummy",
						Name:        name + "-dummy2",
						Description: name + " loopback initial",
						IPv4CIDR:    "198.51.100.1/24",
					}),
					testAccCheckRouterInterfaceAPI("netactuate_router_vrf_interface.wireguard", testAccRouterInterfaceWant{
						Type:          "wireguard",
						Name:          name + "-wireguard",
						Description:   name + " wireguard initial",
						IPv4CIDR:      "192.0.2.3/32",
						WireguardPort: 51820,
					}),
					testAccCheckWireguardPeerAPI("netactuate_router_vrf_interface_wireguard_peer.test", testAccWireguardPeerWant{
						Name:        name + "-wg-peer",
						Description: name + " wireguard peer initial",
						AllowedIPs:  []string{"198.51.100.0/24"},
					}),
					testAccCheckRouterDHCPAPI("netactuate_router_vrf_dhcp.test", testAccRouterDHCPWant{
						Enabled:              true,
						Subnet:               "198.51.100.0/24",
						LeaseTimeout:         3600,
						DoPingCheck:          true,
						DefaultRouterAddress: "198.51.100.1",
						ClientDomainName:     "example.test",
						RangeFirst:           "198.51.100.10",
						RangeLast:            "198.51.100.20",
						DomainNameServers:    []string{"198.51.100.53"},
						NTPServers:           []string{"198.51.100.123"},
						StaticRouteNetwork:   "203.0.113.0/24",
						StaticRouteNextHop:   "198.51.100.1",
					}),
					testAccCheckRouterNTPAPI("netactuate_router_ntp.test", true, []string{"time.cloudflare.com"}),
				),
			},
			{
				ResourceName:      "netactuate_router_vrf_interface.dummy",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_vrf_interface.dummy"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_router_vrf_interface.loopback",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_vrf_interface.loopback"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_router_vrf_interface.wireguard",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_vrf_interface.wireguard"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_router_vrf_interface_wireguard_peer.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_vrf_interface_wireguard_peer.test"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_router_vrf_dhcp.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_vrf_dhcp.test"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_router_ntp.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_ntp.test"),
				ImportStateVerify: true,
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					ids.capture,
					testAccCheckRouterInterfaceAPI("netactuate_router_vrf_interface.dummy", testAccRouterInterfaceWant{
						Type:        "dummy",
						Name:        name + "-dummy-updated",
						Description: name + " dummy updated",
						IPv4CIDR:    "192.0.2.11/32",
					}),
					testAccCheckRouterInterfaceAPI("netactuate_router_vrf_interface.loopback", testAccRouterInterfaceWant{
						Type:        "dummy",
						Name:        name + "-dummy2",
						Description: name + " loopback updated",
						IPv4CIDR:    "198.51.100.1/24",
					}),
					testAccCheckRouterInterfaceAPI("netactuate_router_vrf_interface.wireguard", testAccRouterInterfaceWant{
						Type:          "wireguard",
						Name:          name + "-wireguard-updated",
						Description:   name + " wireguard updated",
						IPv4CIDR:      "192.0.2.13/32",
						WireguardPort: 51821,
					}),
					testAccCheckRouterDHCPAPI("netactuate_router_vrf_dhcp.test", testAccRouterDHCPWant{
						Enabled:              true,
						Subnet:               "198.51.100.0/24",
						LeaseTimeout:         7200,
						DoPingCheck:          false,
						DefaultRouterAddress: "198.51.100.1",
						ClientDomainName:     "updated.example.test",
						RangeFirst:           "198.51.100.30",
						RangeLast:            "198.51.100.40",
						DomainNameServers:    []string{"198.51.100.54"},
						NTPServers:           []string{"198.51.100.124"},
						StaticRouteNetwork:   "203.0.113.0/24",
						StaticRouteNextHop:   "198.51.100.1",
					}),
					testAccCheckRouterNTPAPI("netactuate_router_ntp.test", true, []string{"time.google.com"}),
				),
			},
			{
				Config: routerOnlyConfig,
				Check: resource.ComposeTestCheckFunc(
					ids.checkRemovedFromAPI,
					testAccCheckRouterNTPDisabled(ids.RouterID),
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					ids.capture,
					testAccCheckRouterNTPAPI("netactuate_router_ntp.test", true, []string{"time.google.com"}),
				),
			},
			{
				PreConfig: func() {
					if err := ids.deleteRouterOutOfBand(); err != nil {
						t.Fatalf("out of band router delete: %v", err)
					}
					testAccWaitGoneFromRouterRoutingAPI(t, "router interface service resources", ids.goneFromAPI)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGone("netactuate_router.test"),
					testAccCheckGone("netactuate_router_vrf.test"),
					testAccCheckGone("netactuate_router_vrf_interface.dummy"),
					testAccCheckGone("netactuate_router_vrf_interface.loopback"),
					testAccCheckGone("netactuate_router_vrf_interface.wireguard"),
					testAccCheckGone("netactuate_router_vrf_interface_wireguard_peer.test"),
					testAccCheckGone("netactuate_router_vrf_dhcp.test"),
					testAccCheckGone("netactuate_router_ntp.test"),
				),
			},
		},
	})
}

func TestAccNetactuateRouterTunnelNAT_importUpdateRemoveAndOutOfBandDelete(t *testing.T) {
	name := testAccName("router-tunnel-nat")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	initialConfig := testAccRouterTunnelNATConfig(name, locationID, plan, false)
	updateConfig := testAccRouterTunnelNATConfig(name, locationID, plan, true)
	routerOnlyConfig := testAccRouterServiceRouterOnlyConfig(name, locationID, plan)
	ids := &testAccRouterTunnelNATIDs{}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_PLAN",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_router", "netactuate_router.test"),
					ids.capture,
					testAccCheckRouterTunnelAPI("netactuate_router_vrf_tunnel.test", testAccRouterTunnelWant{
						Name:           name + "-tunnel",
						Description:    name + " tunnel initial",
						MTU:            "1400",
						IPKey:          101,
						IPv4CIDR:       "192.0.2.20/31",
						EndpointRemote: "198.51.100.1",
						IPVersion:      4,
					}),
					testAccCheckRouterSNATAPI("netactuate_router_vrf_snat_rule.test", testAccRouterNATWant{
						Protocol:         "TCP",
						Description:      name + " snat initial",
						MatchNetwork:     "192.0.2.0/24",
						MatchPortStart:   1000,
						MatchPortEnd:     1005,
						TranslateNetwork: "198.51.100.10/32",
						TranslateStart:   2000,
						TranslateEnd:     2005,
					}),
					testAccCheckRouterDNATAPI("netactuate_router_vrf_dnat_rule.test", testAccRouterNATWant{
						Protocol:         "UDP",
						Description:      name + " dnat initial",
						MatchNetwork:     "198.51.100.20/32",
						MatchPortStart:   3000,
						MatchPortEnd:     3005,
						TranslateNetwork: "192.0.2.20/32",
						TranslateStart:   4000,
						TranslateEnd:     4005,
					}),
				),
			},
			{
				ResourceName:      "netactuate_router_vrf_tunnel.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_vrf_tunnel.test"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_router_vrf_snat_rule.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_vrf_snat_rule.test"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "netactuate_router_vrf_dnat_rule.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_vrf_dnat_rule.test"),
				ImportStateVerify: true,
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					ids.capture,
					testAccCheckRouterTunnelAPI("netactuate_router_vrf_tunnel.test", testAccRouterTunnelWant{
						Name:           name + "-tunnel-updated",
						Description:    name + " tunnel updated",
						MTU:            "1410",
						IPKey:          102,
						IPv4CIDR:       "192.0.2.22/31",
						EndpointRemote: "198.51.100.2",
						IPVersion:      4,
					}),
					testAccCheckRouterSNATAPI("netactuate_router_vrf_snat_rule.test", testAccRouterNATWant{
						Protocol:         "TCP",
						Description:      name + " snat updated",
						MatchNetwork:     "192.0.2.0/24",
						MatchPortStart:   1010,
						MatchPortEnd:     1015,
						TranslateNetwork: "198.51.100.11/32",
						TranslateStart:   2010,
						TranslateEnd:     2015,
					}),
					testAccCheckRouterDNATAPI("netactuate_router_vrf_dnat_rule.test", testAccRouterNATWant{
						Protocol:         "UDP",
						Description:      name + " dnat updated",
						MatchNetwork:     "198.51.100.21/32",
						MatchPortStart:   3010,
						MatchPortEnd:     3015,
						TranslateNetwork: "192.0.2.21/32",
						TranslateStart:   4010,
						TranslateEnd:     4015,
					}),
				),
			},
			{
				Config: routerOnlyConfig,
				Check:  ids.checkRemovedFromAPI,
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					ids.capture,
					testAccCheckRouterSNATAPI("netactuate_router_vrf_snat_rule.test", testAccRouterNATWant{
						Protocol:         "TCP",
						Description:      name + " snat updated",
						MatchNetwork:     "192.0.2.0/24",
						MatchPortStart:   1010,
						MatchPortEnd:     1015,
						TranslateNetwork: "198.51.100.11/32",
						TranslateStart:   2010,
						TranslateEnd:     2015,
					}),
				),
			},
			{
				PreConfig: func() {
					if err := ids.deleteRouterOutOfBand(); err != nil {
						t.Fatalf("out of band router delete: %v", err)
					}
					testAccWaitGoneFromRouterRoutingAPI(t, "router tunnel NAT resources", ids.goneFromAPI)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGone("netactuate_router.test"),
					testAccCheckGone("netactuate_router_vrf.test"),
					testAccCheckGone("netactuate_router_vrf_tunnel.test"),
					testAccCheckGone("netactuate_router_vrf_snat_rule.test"),
					testAccCheckGone("netactuate_router_vrf_dnat_rule.test"),
				),
			},
		},
	})
}

func TestAccNetactuateRouterIPSec_importUpdateRemoveAndOutOfBandDelete(t *testing.T) {
	name := testAccName("router-ipsec")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	initialConfig := testAccRouterIPSecConfig(name, locationID, plan, false)
	updateConfig := testAccRouterIPSecConfig(name, locationID, plan, true)
	routerOnlyConfig := testAccRouterServiceRouterOnlyConfig(name, locationID, plan)
	ids := &testAccRouterIPSecIDs{}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_PLAN",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRouterConnectivityDestroy,
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_router", "netactuate_router.test"),
					ids.capture,
					testAccCheckRouterIPSecAPI("netactuate_router_ipsec.test", testAccRouterIPSecWant{
						AutoRenegotiation: true,
						KeyExchange:       2,
						IKELifetime:       28800,
						DHGroup:           14,
						IKEEncryption:     "aes256",
						IKEHash:           "sha256",
						IKEPRF:            "prfsha256",
						ESPLifetime:       3600,
						ESPEncryption:     "aes256",
						ESPHash:           "sha256",
					}),
					testAccCheckRouterIPSecPeerAPI("netactuate_router_vrf_ipsec_peer.test", testAccRouterIPSecPeerWant{
						Name:                 name + "-peer",
						Description:          name + " peer initial",
						RemoteID:             "198.51.100.10",
						DoInitiateConnection: true,
						PeerAddress:          "198.51.100.10",
						OverlayIPv4:          "192.0.2.40/31",
					}),
					resource.TestCheckResourceAttr("netactuate_router_vrf_ipsec_peer.test", "psk_secret", "nah-disposable-ipsec-psk-initial"),
				),
			},
			{
				ResourceName:      "netactuate_router_ipsec.test",
				ImportState:       true,
				ImportStateIdFunc: testAccRouterRoutingImportID("netactuate_router_ipsec.test"),
				ImportStateVerify: true,
			},
			{
				ResourceName:            "netactuate_router_vrf_ipsec_peer.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccRouterRoutingImportID("netactuate_router_vrf_ipsec_peer.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"psk_secret"},
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					ids.capture,
					testAccCheckRouterIPSecAPI("netactuate_router_ipsec.test", testAccRouterIPSecWant{
						AutoRenegotiation: false,
						KeyExchange:       1,
						IKELifetime:       14400,
						DHGroup:           15,
						IKEEncryption:     "aes128",
						IKEHash:           "sha1",
						IKEPRF:            "prfsha1",
						ESPLifetime:       1800,
						ESPEncryption:     "aes128",
						ESPHash:           "sha1",
					}),
					testAccCheckRouterIPSecPeerAPI("netactuate_router_vrf_ipsec_peer.test", testAccRouterIPSecPeerWant{
						Name:                 name + "-peer-updated",
						Description:          name + " peer updated",
						RemoteID:             "198.51.100.11",
						DoInitiateConnection: true,
						PeerAddress:          "198.51.100.11",
						OverlayIPv4:          "192.0.2.42/31",
					}),
					// vAPI3 currently builds pskSecret into the peer read payload from
					// live config, but this test intentionally asserts the disposable
					// PSK only in Terraform state. Live acceptance results should
					// confirm whether that API read-back is a secret exposure defect.
					resource.TestCheckResourceAttr("netactuate_router_vrf_ipsec_peer.test", "psk_secret", "nah-disposable-ipsec-psk-updated"),
				),
			},
			{
				Config: routerOnlyConfig,
				Check:  ids.checkRemovedFromAPI,
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					ids.capture,
					testAccCheckRouterIPSecPeerAPI("netactuate_router_vrf_ipsec_peer.test", testAccRouterIPSecPeerWant{
						Name:                 name + "-peer-updated",
						Description:          name + " peer updated",
						RemoteID:             "198.51.100.11",
						DoInitiateConnection: true,
						PeerAddress:          "198.51.100.11",
						OverlayIPv4:          "192.0.2.42/31",
					}),
				),
			},
			{
				PreConfig: func() {
					if err := ids.deleteRouterOutOfBand(); err != nil {
						t.Fatalf("out of band router delete: %v", err)
					}
					testAccWaitGoneFromRouterRoutingAPI(t, "router IPSec resources", ids.goneFromAPI)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGone("netactuate_router.test"),
					testAccCheckGone("netactuate_router_ipsec.test"),
					testAccCheckGone("netactuate_router_vrf.test"),
					testAccCheckGone("netactuate_router_vrf_ipsec_peer.test"),
				),
			},
		},
	})
}

type testAccRouterInterfaceServicesIDs struct {
	RouterID      int
	VRFID         int
	DummyID       int
	LoopbackID    int
	WireguardID   int
	WireguardPeer int
}

type testAccRouterTunnelNATIDs struct {
	RouterID   int
	VRFID      int
	Interface  int
	TunnelID   int
	SNATRuleID int
	DNATRuleID int
}

type testAccRouterIPSecIDs struct {
	RouterID int
	VRFID    int
	PeerID   int
}

type testAccRouterInterfaceWant struct {
	Type          string
	Name          string
	Description   string
	IPv4CIDR      string
	WireguardPort int
}

type testAccWireguardPeerWant struct {
	Name        string
	Description string
	AllowedIPs  []string
}

type testAccRouterDHCPWant struct {
	Enabled              bool
	Subnet               string
	LeaseTimeout         int
	DoPingCheck          bool
	DefaultRouterAddress string
	ClientDomainName     string
	RangeFirst           string
	RangeLast            string
	DomainNameServers    []string
	NTPServers           []string
	StaticRouteNetwork   string
	StaticRouteNextHop   string
}

type testAccRouterTunnelWant struct {
	Name           string
	Description    string
	MTU            string
	IPKey          int
	IPv4CIDR       string
	EndpointRemote string
	IPVersion      int
}

type testAccRouterNATWant struct {
	Protocol         string
	Description      string
	MatchNetwork     string
	MatchPortStart   int
	MatchPortEnd     int
	TranslateNetwork string
	TranslateStart   int
	TranslateEnd     int
}

type testAccRouterIPSecWant struct {
	AutoRenegotiation bool
	KeyExchange       int
	IKELifetime       int
	DHGroup           int
	IKEEncryption     string
	IKEHash           string
	IKEPRF            string
	ESPLifetime       int
	ESPEncryption     string
	ESPHash           string
}

type testAccRouterIPSecPeerWant struct {
	Name                 string
	Description          string
	RemoteID             string
	DoInitiateConnection bool
	PeerAddress          string
	OverlayIPv4          string
}

func (ids *testAccRouterInterfaceServicesIDs) capture(s *terraform.State) error {
	routerID, err := testAccID(s, "netactuate_router.test")
	if err != nil {
		return err
	}
	_, vrfID, err := testAccCompositeID(s, "netactuate_router_vrf.test")
	if err != nil {
		return err
	}
	_, _, dummyID, err := testAccTripleID(s, "netactuate_router_vrf_interface.dummy")
	if err != nil {
		return err
	}
	_, _, loopbackID, err := testAccTripleID(s, "netactuate_router_vrf_interface.loopback")
	if err != nil {
		return err
	}
	_, _, wireguardID, err := testAccTripleID(s, "netactuate_router_vrf_interface.wireguard")
	if err != nil {
		return err
	}
	_, _, _, peerID, err := testAccQuadID(s, "netactuate_router_vrf_interface_wireguard_peer.test")
	if err != nil {
		return err
	}

	ids.RouterID = routerID
	ids.VRFID = vrfID
	ids.DummyID = dummyID
	ids.LoopbackID = loopbackID
	ids.WireguardID = wireguardID
	ids.WireguardPeer = peerID
	return nil
}

func (ids *testAccRouterTunnelNATIDs) capture(s *terraform.State) error {
	routerID, err := testAccID(s, "netactuate_router.test")
	if err != nil {
		return err
	}
	_, vrfID, err := testAccCompositeID(s, "netactuate_router_vrf.test")
	if err != nil {
		return err
	}
	_, _, interfaceID, err := testAccTripleID(s, "netactuate_router_vrf_interface.dummy")
	if err != nil {
		return err
	}
	_, _, tunnelID, err := testAccTripleID(s, "netactuate_router_vrf_tunnel.test")
	if err != nil {
		return err
	}
	_, _, snatRuleID, err := testAccTripleID(s, "netactuate_router_vrf_snat_rule.test")
	if err != nil {
		return err
	}
	_, _, dnatRuleID, err := testAccTripleID(s, "netactuate_router_vrf_dnat_rule.test")
	if err != nil {
		return err
	}

	ids.RouterID = routerID
	ids.VRFID = vrfID
	ids.Interface = interfaceID
	ids.TunnelID = tunnelID
	ids.SNATRuleID = snatRuleID
	ids.DNATRuleID = dnatRuleID
	return nil
}

func (ids *testAccRouterIPSecIDs) capture(s *terraform.State) error {
	routerID, err := testAccID(s, "netactuate_router.test")
	if err != nil {
		return err
	}
	_, vrfID, err := testAccCompositeID(s, "netactuate_router_vrf.test")
	if err != nil {
		return err
	}
	_, _, peerID, err := testAccTripleID(s, "netactuate_router_vrf_ipsec_peer.test")
	if err != nil {
		return err
	}

	ids.RouterID = routerID
	ids.VRFID = vrfID
	ids.PeerID = peerID
	return nil
}

func (ids *testAccRouterInterfaceServicesIDs) checkRemovedFromAPI(s *terraform.State) error {
	if !ids.goneFromAPI() {
		return fmt.Errorf("router interface service resources still present after Terraform removal")
	}
	return nil
}

func (ids *testAccRouterTunnelNATIDs) checkRemovedFromAPI(s *terraform.State) error {
	if !ids.goneFromAPI() {
		return fmt.Errorf("router tunnel NAT resources still present after Terraform removal")
	}
	return nil
}

func (ids *testAccRouterIPSecIDs) checkRemovedFromAPI(s *terraform.State) error {
	if !ids.goneFromAPI() {
		return fmt.Errorf("router IPSec resources still present after Terraform removal")
	}
	return nil
}

func (ids *testAccRouterInterfaceServicesIDs) deleteRouterOutOfBand() error {
	return deleteRouterOutOfBand(ids.RouterID)
}

func (ids *testAccRouterTunnelNATIDs) deleteRouterOutOfBand() error {
	return deleteRouterOutOfBand(ids.RouterID)
}

func (ids *testAccRouterIPSecIDs) deleteRouterOutOfBand() error {
	return deleteRouterOutOfBand(ids.RouterID)
}

func deleteRouterOutOfBand(routerID int) error {
	if routerID == 0 {
		return fmt.Errorf("router id was not captured")
	}
	err := testAccClients().V3.DeleteRouter(routerID)
	if err != nil && !testAccRouterRoutingNotFound(err) {
		return err
	}
	return nil
}

func (ids *testAccRouterInterfaceServicesIDs) goneFromAPI() bool {
	clients := testAccClients()
	if ids.RouterID == 0 {
		return true
	}
	if _, err := clients.V3.GetRouter(ids.RouterID); err != nil && testAccRouterRoutingNotFound(err) {
		return true
	}
	if ids.WireguardPeer != 0 {
		if _, err := clients.V3.GetRouterVRFInterfaceWireguardPeer(ids.RouterID, ids.VRFID, ids.WireguardID, ids.WireguardPeer); err == nil || !testAccRouterRoutingNotFound(err) {
			return false
		}
	}
	for _, interfaceID := range []int{ids.DummyID, ids.LoopbackID, ids.WireguardID} {
		if interfaceID == 0 {
			continue
		}
		if _, err := clients.V3.GetRouterVRFInterface(ids.RouterID, ids.VRFID, interfaceID); err == nil || !testAccRouterRoutingNotFound(err) {
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

func (ids *testAccRouterTunnelNATIDs) goneFromAPI() bool {
	clients := testAccClients()
	if ids.RouterID == 0 {
		return true
	}
	if _, err := clients.V3.GetRouter(ids.RouterID); err != nil && testAccRouterRoutingNotFound(err) {
		return true
	}
	if ids.SNATRuleID != 0 {
		if _, err := clients.V3.GetRouterVRFSNATRule(ids.RouterID, ids.VRFID, ids.SNATRuleID); err == nil || !testAccRouterRoutingNotFound(err) {
			return false
		}
	}
	if ids.DNATRuleID != 0 {
		if _, err := clients.V3.GetRouterVRFDNATRule(ids.RouterID, ids.VRFID, ids.DNATRuleID); err == nil || !testAccRouterRoutingNotFound(err) {
			return false
		}
	}
	if ids.TunnelID != 0 {
		if _, err := clients.V3.GetRouterVRFTunnel(ids.RouterID, ids.VRFID, ids.TunnelID); err == nil || !testAccRouterRoutingNotFound(err) {
			return false
		}
	}
	if ids.Interface != 0 {
		if _, err := clients.V3.GetRouterVRFInterface(ids.RouterID, ids.VRFID, ids.Interface); err == nil || !testAccRouterRoutingNotFound(err) {
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

func (ids *testAccRouterIPSecIDs) goneFromAPI() bool {
	clients := testAccClients()
	if ids.RouterID == 0 {
		return true
	}
	if _, err := clients.V3.GetRouter(ids.RouterID); err != nil && testAccRouterRoutingNotFound(err) {
		return true
	}
	if ids.PeerID != 0 {
		if _, err := clients.V3.GetRouterVRFIPSecPeer(ids.RouterID, ids.VRFID, ids.PeerID); err == nil || !testAccRouterRoutingNotFound(err) {
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

func testAccCheckRouterInterfaceAPI(address string, want testAccRouterInterfaceWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, interfaceID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		got, err := testAccClients().V3.GetRouterVRFInterface(routerID, vrfID, interfaceID)
		if err != nil {
			return err
		}
		if got.Type != want.Type || got.Name != want.Name {
			return fmt.Errorf("interface type/name: expected %q/%q, got %q/%q", want.Type, want.Name, got.Type, got.Name)
		}
		if stringValue(got.Description) != want.Description {
			return fmt.Errorf("interface description: expected %q, got %#v", want.Description, got.Description)
		}
		if stringValue(got.IPv4CIDR) != want.IPv4CIDR {
			return fmt.Errorf("interface ipv4_cidr: expected %q, got %#v", want.IPv4CIDR, got.IPv4CIDR)
		}
		if want.WireguardPort != 0 && (got.WireguardPort == nil || *got.WireguardPort != want.WireguardPort) {
			return fmt.Errorf("wireguard port: expected %d, got %#v", want.WireguardPort, got.WireguardPort)
		}
		return nil
	}
}

func testAccCheckWireguardPeerAPI(address string, want testAccWireguardPeerWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, interfaceID, peerID, err := testAccQuadID(s, address)
		if err != nil {
			return err
		}
		got, err := testAccClients().V3.GetRouterVRFInterfaceWireguardPeer(routerID, vrfID, interfaceID, peerID)
		if err != nil {
			return err
		}
		if stringValue(got.Name) != want.Name {
			return fmt.Errorf("wireguard peer name: expected %q, got %#v", want.Name, got.Name)
		}
		if stringValue(got.Description) != want.Description {
			return fmt.Errorf("wireguard peer description: expected %q, got %#v", want.Description, got.Description)
		}
		for _, network := range want.AllowedIPs {
			if !wireguardAllowedIPsContain(got.AllowedIPs, network) {
				return fmt.Errorf("wireguard peer allowed_ips missing %q: %#v", network, got.AllowedIPs)
			}
		}
		return nil
	}
}

func testAccCheckRouterDHCPAPI(address string, want testAccRouterDHCPWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, err := testAccCompositeID(s, address)
		if err != nil {
			return err
		}
		got, err := testAccClients().V3.GetRouterVRFDHCP(routerID, vrfID)
		if err != nil {
			return err
		}
		if got.Enabled != want.Enabled || got.Subnet != want.Subnet || got.LeaseTimeout != want.LeaseTimeout || got.DoPingCheck != want.DoPingCheck {
			return fmt.Errorf("DHCP scalar mismatch: %#v", got)
		}
		if got.DefaultRouterAddress != want.DefaultRouterAddress || got.ClientDomainName != want.ClientDomainName {
			return fmt.Errorf("DHCP options mismatch: %#v", got)
		}
		if got.Range == nil || got.Range.FirstAddress != want.RangeFirst || got.Range.LastAddress != want.RangeLast {
			return fmt.Errorf("DHCP range: expected %s-%s, got %#v", want.RangeFirst, want.RangeLast, got.Range)
		}
		for _, address := range want.DomainNameServers {
			if !dhcpServersContain(got.DomainNameServers, address) {
				return fmt.Errorf("DHCP domain_name_servers missing %q: %#v", address, got.DomainNameServers)
			}
		}
		for _, address := range want.NTPServers {
			if !dhcpServersContain(got.NTPServers, address) {
				return fmt.Errorf("DHCP ntp_servers missing %q: %#v", address, got.NTPServers)
			}
		}
		if !dhcpStaticRoutesContain(got.StaticRoutes, want.StaticRouteNetwork, want.StaticRouteNextHop) {
			return fmt.Errorf("DHCP static_routes missing %s via %s: %#v", want.StaticRouteNetwork, want.StaticRouteNextHop, got.StaticRoutes)
		}
		return nil
	}
}

func testAccCheckRouterNTPAPI(address string, enabled bool, upstreams []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		got, err := testAccClients().V3.GetRouterNTPConfig(routerID)
		if err != nil {
			return err
		}
		if got.Enabled != enabled {
			return fmt.Errorf("NTP enabled: expected %t, got %t", enabled, got.Enabled)
		}
		for _, domain := range upstreams {
			if !ntpUpstreamsContain(got.Upstreams, domain) {
				return fmt.Errorf("NTP upstreams missing %q: %#v", domain, got.Upstreams)
			}
		}
		return nil
	}
}

func testAccCheckRouterTunnelAPI(address string, want testAccRouterTunnelWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, tunnelID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		got, err := testAccClients().V3.GetRouterVRFTunnel(routerID, vrfID, tunnelID)
		if err != nil {
			return err
		}
		if got.Name != want.Name || stringValue(got.Description) != want.Description {
			return fmt.Errorf("tunnel name/description mismatch: %#v", got)
		}
		if got.MTU != want.MTU || got.IPKey != want.IPKey || got.IPVersion != want.IPVersion {
			return fmt.Errorf("tunnel scalar mismatch: %#v", got)
		}
		if stringValue(got.IPv4CIDR) != want.IPv4CIDR || got.EndpointAddress.Remote != want.EndpointRemote {
			return fmt.Errorf("tunnel address mismatch: %#v", got)
		}
		return nil
	}
}

func testAccCheckRouterSNATAPI(address string, want testAccRouterNATWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, ruleID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		got, err := testAccClients().V3.GetRouterVRFSNATRule(routerID, vrfID, ruleID)
		if err != nil {
			return err
		}
		if got.Match == nil || got.Match.Port == nil {
			return fmt.Errorf("SNAT match missing: %#v", got.Match)
		}
		if got.Translation == nil || got.Translation.Port == nil {
			return fmt.Errorf("SNAT translation missing: %#v", got.Translation)
		}
		return checkRouterNAT(
			"SNAT",
			got.Protocol,
			got.Description,
			got.Match.Network,
			got.Match.Port.Start,
			got.Match.Port.End,
			got.Translation.Network,
			got.Translation.Port.Start,
			got.Translation.Port.End,
			want,
		)
	}
}

func testAccCheckRouterDNATAPI(address string, want testAccRouterNATWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, ruleID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		got, err := testAccClients().V3.GetRouterVRFDNATRule(routerID, vrfID, ruleID)
		if err != nil {
			return err
		}
		if got.Match == nil || got.Match.Port == nil {
			return fmt.Errorf("DNAT match missing: %#v", got.Match)
		}
		if got.Translation == nil || got.Translation.Port == nil {
			return fmt.Errorf("DNAT translation missing: %#v", got.Translation)
		}
		return checkRouterNAT(
			"DNAT",
			got.Protocol,
			got.Description,
			got.Match.Network,
			got.Match.Port.Start,
			got.Match.Port.End,
			got.Translation.Network,
			got.Translation.Port.Start,
			got.Translation.Port.End,
			want,
		)
	}
}

func testAccCheckRouterIPSecAPI(address string, want testAccRouterIPSecWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, err := testAccID(s, address)
		if err != nil {
			return err
		}
		got, err := testAccClients().V3.GetRouterIPSecConfig(routerID)
		if err != nil {
			return err
		}
		if got.IKEGroup.DoAutoRenegotiation != want.AutoRenegotiation || got.IKEGroup.KeyExchangeVersion != want.KeyExchange ||
			got.IKEGroup.LifetimeSeconds != want.IKELifetime || got.IKEGroup.DHGroupNumber != want.DHGroup ||
			got.IKEGroup.Encryption != want.IKEEncryption || got.IKEGroup.Hash != want.IKEHash || got.IKEGroup.PRF != want.IKEPRF {
			return fmt.Errorf("IKE group mismatch: %#v", got.IKEGroup)
		}
		if got.ESPGroup.LifetimeSeconds != want.ESPLifetime || got.ESPGroup.Encryption != want.ESPEncryption || got.ESPGroup.Hash != want.ESPHash {
			return fmt.Errorf("ESP group mismatch: %#v", got.ESPGroup)
		}
		return nil
	}
}

func testAccCheckRouterIPSecPeerAPI(address string, want testAccRouterIPSecPeerWant) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routerID, vrfID, peerID, err := testAccTripleID(s, address)
		if err != nil {
			return err
		}
		got, err := testAccClients().V3.GetRouterVRFIPSecPeer(routerID, vrfID, peerID)
		if err != nil {
			return err
		}
		if got.Name != want.Name || stringValue(got.Description) != want.Description || got.RemoteID != want.RemoteID {
			return fmt.Errorf("IPSec peer identity mismatch: %#v", got)
		}
		if got.DoInitiateConnection != want.DoInitiateConnection || got.PeerAddress != want.PeerAddress {
			return fmt.Errorf("IPSec peer connection mismatch: %#v", got)
		}
		if stringValue(got.OverlayNetwork.IPv4) != want.OverlayIPv4 {
			return fmt.Errorf("IPSec peer overlay IPv4: expected %q, got %#v", want.OverlayIPv4, got.OverlayNetwork.IPv4)
		}
		return nil
	}
}

func testAccCheckRouterNTPDisabled(routerID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		got, err := testAccClients().V3.GetRouterNTPConfig(routerID)
		if err != nil {
			return err
		}
		if got.Enabled {
			return fmt.Errorf("NTP still enabled after Terraform removal: %#v", got)
		}
		return nil
	}
}

func testAccCheckRouterDHCPDisabled(routerID, vrfID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		got, err := testAccClients().V3.GetRouterVRFDHCP(routerID, vrfID)
		if err != nil {
			return err
		}
		if got.Enabled {
			return fmt.Errorf("DHCP still enabled after Terraform removal: %#v", got)
		}
		return nil
	}
}

func testAccCheckRouterConnectivityDestroy(s *terraform.State) error {
	return testAccCheckRouterDestroy(s)
}

func testAccQuadID(s *terraform.State, address string) (int, int, int, int, error) {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return 0, 0, 0, 0, fmt.Errorf("not found: %s", address)
	}
	parts := strings.SplitN(rs.Primary.ID, "/", 4)
	if len(parts) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("%s has invalid quad ID %q", address, rs.Primary.ID)
	}
	first, second, third, err := parseTripleIDParts(address, parts[:3])
	if err != nil {
		return 0, 0, 0, 0, err
	}
	fourth, err := parsePositiveIDPart(address, "fourth", parts[3])
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return first, second, third, fourth, nil
}

func checkRouterNAT(kind, protocol, description, matchNetwork string, matchStart, matchEnd int, translationNetwork string, translationStart, translationEnd int, want testAccRouterNATWant) error {
	if protocol != want.Protocol || description != want.Description {
		return fmt.Errorf("%s identity: expected %s/%q, got %s/%q", kind, want.Protocol, want.Description, protocol, description)
	}
	if matchNetwork != want.MatchNetwork || matchStart != want.MatchPortStart || matchEnd != want.MatchPortEnd {
		return fmt.Errorf("%s match mismatch: %s %d-%d", kind, matchNetwork, matchStart, matchEnd)
	}
	if translationNetwork != want.TranslateNetwork || translationStart != want.TranslateStart || translationEnd != want.TranslateEnd {
		return fmt.Errorf("%s translation mismatch: %s %d-%d", kind, translationNetwork, translationStart, translationEnd)
	}
	return nil
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func wireguardAllowedIPsContain(ips []gona.WireguardPeerAllowedIP, want string) bool {
	for _, ip := range ips {
		if ip.Network == want {
			return true
		}
	}
	return false
}

func dhcpServersContain(servers []gona.RouterDHCPServer, want string) bool {
	for _, server := range servers {
		if server.Address == want {
			return true
		}
	}
	return false
}

func dhcpStaticRoutesContain(routes []gona.RouterDHCPStaticRoute, network, nextHop string) bool {
	for _, route := range routes {
		if route.Network == network && route.NextHop == nextHop {
			return true
		}
	}
	return false
}

func ntpUpstreamsContain(upstreams []gona.RouterNTPUpstream, want string) bool {
	for _, upstream := range upstreams {
		if upstream.Domain == want {
			return true
		}
	}
	return false
}

func testAccRouterServiceRouterOnlyConfig(name string, locationID int, plan string) string {
	return fmt.Sprintf(`
%s

resource "netactuate_router" "test" {
  name        = %[2]q
  description = %[3]q
  location_id = %[4]d
  plan        = %[5]q
}
`, testAccProviderConfig(), name, name+" router", locationID, plan)
}

func testAccRouterInterfaceServicesConfig(name string, locationID int, plan string, updated bool) string {
	dummyName := name + "-dummy"
	dummyDescription := name + " dummy initial"
	dummyCIDR := "192.0.2.1/32"
	// A second dummy rather than a loopback. The API advertises "loopback" in its interface type
	// enum, but the product removed loopback because dummy serves the same purpose, and the
	// device refuses to create one: "Loopback interface must be named lo", which holds even when
	// it is named lo. Until the enum and the device agree, loopback is not a type a customer can
	// use, so testing it would assert a capability that does not exist.
	loopbackName := name + "-dummy2"
	loopbackDescription := name + " loopback initial"
	// DHCP binds to this interface, and the platform requires the interface to hold an address
	// inside the DHCP subnet. A /32 cannot contain a range, so this one carries a real prefix in
	// 198.51.100.0/24 and stays put across the update: the DHCP block below serves that subnet.
	loopbackCIDR := "198.51.100.1/24"
	wireguardName := name + "-wireguard"
	wireguardDescription := name + " wireguard initial"
	wireguardCIDR := "192.0.2.3/32"
	wireguardPort := 51820
	peerName := name + "-wg-peer"
	peerDescription := name + " wireguard peer initial"
	peerAllowedIP := "198.51.100.0/24"
	leaseTimeout := 3600
	doPingCheck := "true"
	clientDomain := "example.test"
	rangeFirst := "198.51.100.10"
	rangeLast := "198.51.100.20"
	dnsServer := "198.51.100.53"
	ntpServer := "198.51.100.123"
	ntpDomain := "time.cloudflare.com"

	if updated {
		dummyName = name + "-dummy-updated"
		dummyDescription = name + " dummy updated"
		dummyCIDR = "192.0.2.11/32"
		loopbackDescription = name + " loopback updated"
		wireguardName = name + "-wireguard-updated"
		wireguardDescription = name + " wireguard updated"
		wireguardCIDR = "192.0.2.13/32"
		wireguardPort = 51821
		leaseTimeout = 7200
		doPingCheck = "false"
		clientDomain = "updated.example.test"
		rangeFirst = "198.51.100.30"
		rangeLast = "198.51.100.40"
		dnsServer = "198.51.100.54"
		ntpServer = "198.51.100.124"
		ntpDomain = "time.google.com"
	}

	return fmt.Sprintf(`
%s

resource "netactuate_router" "test" {
  name        = %[2]q
  description = %[3]q
  location_id = %[4]d
  plan        = %[5]q
}

resource "netactuate_router_vrf" "test" {
  router_id   = netactuate_router.test.router_id
  name        = %[6]q
  description = %[7]q
}

resource "netactuate_router_vrf_interface" "dummy" {
  router_id   = netactuate_router.test.router_id
  vrf_id      = netactuate_router_vrf.test.vrf_id
  type        = "dummy"
  name        = %[8]q
  description = %[9]q
  ipv4_cidr   = %[10]q
}

resource "netactuate_router_vrf_interface" "loopback" {
  router_id   = netactuate_router.test.router_id
  vrf_id      = netactuate_router_vrf.test.vrf_id
  type        = "dummy"
  name        = %[11]q
  description = %[12]q
  ipv4_cidr   = %[13]q
}

resource "netactuate_router_vrf_interface" "wireguard" {
  router_id      = netactuate_router.test.router_id
  vrf_id         = netactuate_router_vrf.test.vrf_id
  type           = "wireguard"
  name           = %[14]q
  description    = %[15]q
  ipv4_cidr      = %[16]q
  wireguard_port = %[17]d
}

# Ethernet interfaces require ethernet_hardware_id issued by NetActuate
# support, so this lane cannot create one. Dummy, loopback and wireguard are
# the testable cloudRouterInterface variants here.
resource "netactuate_router_vrf_interface_wireguard_peer" "test" {
  router_id    = netactuate_router.test.router_id
  vrf_id       = netactuate_router_vrf.test.vrf_id
  interface_id = netactuate_router_vrf_interface.wireguard.interface_id
  name         = %[18]q
  description  = %[19]q

  allowed_ips {
    network = %[20]q
  }
}

resource "netactuate_router_vrf_dhcp" "test" {
  router_id              = netactuate_router.test.router_id
  vrf_id                 = netactuate_router_vrf.test.vrf_id
  enabled                = true
  interface_id           = netactuate_router_vrf_interface.loopback.interface_id
  subnet                 = "198.51.100.0/24"
  lease_timeout          = %[21]d
  do_ping_check          = %[22]s
  default_router_address = "198.51.100.1"
  client_domain_name     = %[23]q

  range {
    first_address = %[24]q
    last_address  = %[25]q
  }

  domain_name_servers {
    address = %[26]q
  }

  ntp_servers {
    address = %[27]q
  }

  static_routes {
    network  = "203.0.113.0/24"
    next_hop = "198.51.100.1"
  }
}

resource "netactuate_router_ntp" "test" {
  router_id    = netactuate_router.test.router_id
  enabled      = true
  interface_id = netactuate_router_vrf_interface.loopback.interface_id

  upstreams {
    domain = %[28]q
  }
}
`, testAccProviderConfig(), name, name+" router", locationID, plan, name+" vrf", name+" vrf",
		dummyName, dummyDescription, dummyCIDR, loopbackName, loopbackDescription, loopbackCIDR,
		wireguardName, wireguardDescription, wireguardCIDR, wireguardPort, peerName, peerDescription,
		peerAllowedIP, leaseTimeout, doPingCheck, clientDomain, rangeFirst, rangeLast, dnsServer,
		ntpServer, ntpDomain)
}

func testAccRouterTunnelNATConfig(name string, locationID int, plan string, updated bool) string {
	tunnelName := name + "-tunnel"
	tunnelDescription := name + " tunnel initial"
	tunnelMTU := 1400
	tunnelKey := 101
	tunnelCIDR := "192.0.2.20/31"
	tunnelRemote := "198.51.100.1"
	snatDescription := name + " snat initial"
	snatMatchStart := 1000
	snatMatchEnd := 1005
	snatTranslation := "198.51.100.10/32"
	snatTranslateStart := 2000
	snatTranslateEnd := 2005
	dnatDescription := name + " dnat initial"
	dnatMatch := "198.51.100.20/32"
	dnatMatchStart := 3000
	dnatMatchEnd := 3005
	dnatTranslation := "192.0.2.20/32"
	dnatTranslateStart := 4000
	dnatTranslateEnd := 4005

	if updated {
		tunnelName = name + "-tunnel-updated"
		tunnelDescription = name + " tunnel updated"
		tunnelMTU = 1410
		tunnelKey = 102
		tunnelCIDR = "192.0.2.22/31"
		tunnelRemote = "198.51.100.2"
		snatDescription = name + " snat updated"
		snatMatchStart = 1010
		snatMatchEnd = 1015
		snatTranslation = "198.51.100.11/32"
		snatTranslateStart = 2010
		snatTranslateEnd = 2015
		dnatDescription = name + " dnat updated"
		dnatMatch = "198.51.100.21/32"
		dnatMatchStart = 3010
		dnatMatchEnd = 3015
		dnatTranslation = "192.0.2.21/32"
		dnatTranslateStart = 4010
		dnatTranslateEnd = 4015
	}

	return fmt.Sprintf(`
%s

resource "netactuate_router" "test" {
  name        = %[2]q
  description = %[3]q
  location_id = %[4]d
  plan        = %[5]q
}

resource "netactuate_router_vrf" "test" {
  router_id   = netactuate_router.test.router_id
  name        = %[6]q
  description = %[7]q
}

resource "netactuate_router_vrf_interface" "dummy" {
  router_id   = netactuate_router.test.router_id
  vrf_id      = netactuate_router_vrf.test.vrf_id
  type        = "dummy"
  name        = %[8]q
  description = %[9]q
  ipv4_cidr   = "192.0.2.30/32"
}

resource "netactuate_router_vrf_tunnel" "test" {
  router_id               = netactuate_router.test.router_id
  vrf_id                  = netactuate_router_vrf.test.vrf_id
  name                    = %[10]q
  description             = %[11]q
  mtu                     = %[12]d
  ip_key                  = %[13]d
  ipv4_cidr               = %[14]q
  endpoint_address_remote = %[15]q
}

resource "netactuate_router_vrf_snat_rule" "test" {
  router_id              = netactuate_router.test.router_id
  vrf_id                 = netactuate_router_vrf.test.vrf_id
  ip_version             = 4
  protocol               = "TCP"
  description            = %[16]q
  match_interface_id     = netactuate_router_vrf_interface.dummy.interface_id
  match_network          = "192.0.2.0/24"
  match_port_start       = %[17]d
  match_port_end         = %[18]d
  translation_network    = %[19]q
  translation_port_start = %[20]d
  translation_port_end   = %[21]d
}

resource "netactuate_router_vrf_dnat_rule" "test" {
  router_id              = netactuate_router.test.router_id
  vrf_id                 = netactuate_router_vrf.test.vrf_id
  ip_version             = 4
  protocol               = "UDP"
  description            = %[22]q
  match_interface_id     = netactuate_router_vrf_interface.dummy.interface_id
  match_network          = %[23]q
  match_port_start       = %[24]d
  match_port_end         = %[25]d
  translation_network    = %[26]q
  translation_port_start = %[27]d
  translation_port_end   = %[28]d
}
`, testAccProviderConfig(), name, name+" router", locationID, plan, name+" vrf", name+" vrf",
		name+"-dummy", name+" dummy", tunnelName, tunnelDescription, tunnelMTU, tunnelKey,
		tunnelCIDR, tunnelRemote, snatDescription, snatMatchStart, snatMatchEnd, snatTranslation,
		snatTranslateStart, snatTranslateEnd, dnatDescription, dnatMatch, dnatMatchStart,
		dnatMatchEnd, dnatTranslation, dnatTranslateStart, dnatTranslateEnd)
}

func testAccRouterIPSecConfig(name string, locationID int, plan string, updated bool) string {
	autoRenegotiation := "true"
	keyExchange := 2
	ikeLifetime := 28800
	dhGroup := 14
	ikeEncryption := "aes256"
	ikeHash := "sha256"
	ikePRF := "prfsha256"
	espLifetime := 3600
	espEncryption := "aes256"
	espHash := "sha256"
	peerName := name + "-peer"
	peerDescription := name + " peer initial"
	remoteID := "198.51.100.10"
	peerAddress := "198.51.100.10"
	overlayIPv4 := "192.0.2.40/31"
	psk := "nah-disposable-ipsec-psk-initial"

	if updated {
		autoRenegotiation = "false"
		keyExchange = 1
		ikeLifetime = 14400
		dhGroup = 15
		ikeEncryption = "aes128"
		ikeHash = "sha1"
		ikePRF = "prfsha1"
		espLifetime = 1800
		espEncryption = "aes128"
		espHash = "sha1"
		peerName = name + "-peer-updated"
		peerDescription = name + " peer updated"
		remoteID = "198.51.100.11"
		peerAddress = "198.51.100.11"
		overlayIPv4 = "192.0.2.42/31"
		psk = "nah-disposable-ipsec-psk-updated"
	}

	return fmt.Sprintf(`
%s

resource "netactuate_router" "test" {
  name        = %[2]q
  description = %[3]q
  location_id = %[4]d
  plan        = %[5]q
}

resource "netactuate_router_ipsec" "test" {
  router_id                 = netactuate_router.test.router_id
  ike_do_auto_renegotiation = %[6]s
  ike_key_exchange_version  = %[7]d
  ike_lifetime_seconds      = %[8]d
  ike_dh_group_number       = %[9]d
  ike_encryption            = %[10]q
  ike_hash                  = %[11]q
  ike_prf                   = %[12]q
  esp_lifetime_seconds      = %[13]d
  esp_encryption            = %[14]q
  esp_hash                  = %[15]q
}

resource "netactuate_router_vrf" "test" {
  router_id   = netactuate_router.test.router_id
  name        = %[16]q
  description = %[17]q
}

resource "netactuate_router_vrf_ipsec_peer" "test" {
  router_id                = netactuate_router.test.router_id
  vrf_id                   = netactuate_router_vrf.test.vrf_id
  name                     = %[18]q
  description              = %[19]q
  remote_id                = %[20]q
  psk_secret               = %[21]q
  do_initiate_connection   = true
  peer_address             = %[22]q
  overlay_ipv4             = %[23]q
  depends_on               = [netactuate_router_ipsec.test]
}
`, testAccProviderConfig(), name, name+" router", locationID, plan, autoRenegotiation, keyExchange,
		ikeLifetime, dhGroup, ikeEncryption, ikeHash, ikePRF, espLifetime, espEncryption, espHash,
		name+" vrf", name+" vrf", peerName, peerDescription, remoteID, psk, peerAddress, overlayIPv4)
}
