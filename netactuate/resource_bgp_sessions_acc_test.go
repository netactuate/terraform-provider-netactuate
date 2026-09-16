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

var testAccBGPSessionsServerID int

func TestAccNetactuateBGPSessions_importPlanDataSourceAndOutOfBandDelete(t *testing.T) {
	name := testAccName("bgp")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	groupID := testAccEnvInt(t, "NETACTUATE_ACC_BGP_GROUP_ID")
	serverConfig := testAccBGPSessionsServerConfig(name, locationID, imageID, plan, password)
	config := testAccBGPSessionsConfig(name, locationID, imageID, plan, password, groupID, false, false)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_IMAGE_ID",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_SERVER_PASSWORD",
				"NETACTUATE_ACC_BGP_GROUP_ID",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckBGPSessionsAndServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: serverConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_server", "netactuate_server.test"),
					testAccRememberID("netactuate_server.test", &testAccBGPSessionsServerID),
					testAccCheckBGPSessionsServerFromAPI("netactuate_server.test", name+".example.invalid", plan, locationID, imageID),
					testAccRequireEmptyBGPSessionsFromState("netactuate_server.test"),
				),
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_server", "netactuate_server.test"),
					testAccRememberID("netactuate_server.test", &testAccBGPSessionsServerID),
					testAccCheckBGPSessionsFromAPI("netactuate_bgp_sessions.test", groupID, false, false),
					testAccCheckBGPSessionsDataSourceFromAPI("data.netactuate_bgp_sessions.test", "netactuate_server.test", groupID),
				),
			},
			{
				ResourceName:      "netactuate_bgp_sessions.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
			{
				PreConfig: func() {
					mbpkgid := testAccBGPSessionsServerID
					if mbpkgid == 0 {
						t.Fatal("no tracked BGP server ID")
					}
					if err := testAccDeleteBGPSessionsForEmptyTestPackage(mbpkgid, groupID); err != nil {
						t.Fatalf("out of band BGP session delete: %v", err)
					}
					testAccWaitBGPSessionsCleared(t, mbpkgid)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_bgp_sessions.test"),
			},
		},
	})
}

func testAccCheckBGPSessionsAndServerDestroy(s *terraform.State) error {
	if err := testAccCheckBGPSessionsDestroy(s); err != nil {
		return err
	}
	return testAccCheckServerDestroy(s)
}

func testAccRequireEmptyBGPSessionsFromState(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		mbpkgid, err := testAccID(s, address)
		if err != nil {
			return err
		}
		sessions, err := testAccClients().V2.GetBGPSessions(mbpkgid)
		if err != nil {
			return fmt.Errorf("checking BGP sessions for test server mbpkgid %d: %w", mbpkgid, err)
		}
		if len(sessions) != 0 {
			return fmt.Errorf("test server mbpkgid %d already has %d BGP session(s); refusing to continue", mbpkgid, len(sessions))
		}
		return nil
	}
}

func testAccCheckBGPSessionsDestroy(s *terraform.State) error {
	clients := testAccClients()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_bgp_sessions" {
			continue
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		sessions, err := clients.V2.GetBGPSessions(id)
		if err != nil {
			return err
		}
		if len(sessions) != 0 {
			return fmt.Errorf("BGP sessions still exist for mbpkgid %d: %d", id, len(sessions))
		}
	}
	return nil
}

func testAccCheckBGPSessionsFromAPI(address string, expectedGroupID int, expectedIPv6, expectedRedundant bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		mbpkgid, err := testAccID(s, address)
		if err != nil {
			return err
		}
		return testAccWaitBGPSessions(mbpkgid, expectedGroupID, expectedIPv6, expectedRedundant)
	}
}

func testAccCheckBGPSessionsDataSourceFromAPI(address, serverAddress string, expectedGroupID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		mbpkgid, err := testAccID(s, serverAddress)
		if err != nil {
			return err
		}
		sessions, err := testAccClients().V2.GetBGPSessions(mbpkgid)
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			return fmt.Errorf("BGP sessions API returned no sessions for mbpkgid %d", mbpkgid)
		}
		if got := rs.Primary.Attributes["sessions.#"]; got != strconv.Itoa(len(sessions)) {
			return fmt.Errorf("%s state session count = %q, want API count %d", address, got, len(sessions))
		}
		for i, session := range sessions {
			prefix := fmt.Sprintf("sessions.%d", i)
			if got := rs.Primary.Attributes[prefix+".id"]; got != strconv.Itoa(session.ID) {
				return fmt.Errorf("%s state %s.id = %q, want API id %d", address, prefix, got, session.ID)
			}
			if got := rs.Primary.Attributes[prefix+".group_id"]; got != strconv.Itoa(expectedGroupID) {
				return fmt.Errorf("%s state %s.group_id = %q, want %d", address, prefix, got, expectedGroupID)
			}
			if got := rs.Primary.Attributes[prefix+".customer_peer_ip"]; got != session.CustomerIP {
				return fmt.Errorf("%s state %s.customer_peer_ip = %q, want API value %q", address, prefix, got, session.CustomerIP)
			}
			if got := rs.Primary.Attributes[prefix+".provider_peer_ip"]; got != session.ProviderPeerIP {
				return fmt.Errorf("%s state %s.provider_peer_ip = %q, want API value %q", address, prefix, got, session.ProviderPeerIP)
			}
		}
		return nil
	}
}

func testAccCheckBGPSessionsServerFromAPI(address, hostname, plan string, locationID, imageID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		server, err := testAccClients().V2.GetServer(id)
		if err != nil {
			return err
		}
		if server.Name != hostname {
			return fmt.Errorf("%s API hostname = %q, want %q", address, server.Name, hostname)
		}
		if server.Package != plan {
			return fmt.Errorf("%s API plan = %q, want %q", address, server.Package, plan)
		}
		if server.LocationID != locationID {
			return fmt.Errorf("%s API location_id = %d, want %d", address, server.LocationID, locationID)
		}
		if server.OSID != imageID {
			return fmt.Errorf("%s API image_id = %d, want %d", address, server.OSID, imageID)
		}
		return nil
	}
}

func testAccWaitBGPSessions(mbpkgid, expectedGroupID int, expectedIPv6, expectedRedundant bool) error {
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		sessions, err := testAccClients().V2.GetBGPSessions(mbpkgid)
		if err != nil {
			lastErr = err
		} else if err := testAccAssertBGPSessions(sessions, expectedGroupID, expectedIPv6, expectedRedundant); err != nil {
			lastErr = err
		} else {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for BGP sessions on mbpkgid %d: %v", mbpkgid, lastErr)
}

func testAccAssertBGPSessions(sessions []*gona.BGPSession, expectedGroupID int, expectedIPv6, expectedRedundant bool) error {
	if len(sessions) == 0 {
		return fmt.Errorf("API returned no BGP sessions")
	}
	v4Count, v6Count := 0, 0
	for _, session := range sessions {
		if session.GroupID != expectedGroupID {
			return fmt.Errorf("BGP session %d API group_id = %d, want %d", session.ID, session.GroupID, expectedGroupID)
		}
		if session.CustomerIP == "" || session.ProviderPeerIP == "" {
			return fmt.Errorf("BGP session %d API peer IPs are incomplete", session.ID)
		}
		if session.IsProviderIPTypeV4() {
			v4Count++
		} else {
			v6Count++
		}
	}
	if expectedIPv6 && v6Count == 0 {
		return fmt.Errorf("API returned no IPv6 BGP sessions")
	}
	if !expectedIPv6 && v6Count != 0 {
		return fmt.Errorf("API returned %d IPv6 BGP sessions, want none", v6Count)
	}
	if !expectedRedundant && (v4Count > 1 || v6Count > 1) {
		return fmt.Errorf("API returned redundant BGP sessions, want non-redundant")
	}
	return nil
}

func testAccDeleteBGPSessionsForEmptyTestPackage(mbpkgid, expectedGroupID int) error {
	clients := testAccClients()
	server, err := clients.V2.GetServer(mbpkgid)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(server.Name, "nah-") {
		return fmt.Errorf("refusing to delete BGP sessions for non-acceptance server %d named %q", mbpkgid, server.Name)
	}
	sessions, err := clients.V2.GetBGPSessions(mbpkgid)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if session.GroupID != expectedGroupID {
			return fmt.Errorf("refusing to delete BGP session %d with group_id %d, want test group_id %d", session.ID, session.GroupID, expectedGroupID)
		}
		if err := clients.V2.DeleteBGPSession(session.ID); err != nil {
			return err
		}
	}
	return nil
}

func testAccWaitBGPSessionsCleared(t *testing.T, mbpkgid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		sessions, err := testAccClients().V2.GetBGPSessions(mbpkgid)
		if err != nil {
			lastErr = err
		} else if len(sessions) == 0 {
			return
		} else {
			lastErr = fmt.Errorf("%d BGP session(s) remain", len(sessions))
		}
		time.Sleep(5 * time.Second)
	}
	t.Fatalf("timed out waiting for BGP sessions on mbpkgid %d to clear: %v", mbpkgid, lastErr)
}

func testAccBGPSessionsServerConfig(name string, locationID, imageID int, plan, password string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_server" "test" {
  hostname               = %q
  plan                   = %q
  location_id            = %d
  image_id               = %d
  password               = %q
  package_billing        = "package"
  package_billing_opt_in = "yes"

  lifecycle {
    ignore_changes = [password]
  }
}
`, name+".example.invalid", plan, locationID, imageID, password)
}

func testAccBGPSessionsConfig(name string, locationID, imageID int, plan, password string, groupID int, ipv6, redundant bool) string {
	return testAccBGPSessionsServerConfig(name, locationID, imageID, plan, password) + fmt.Sprintf(`
resource "netactuate_bgp_sessions" "test" {
  mbpkgid   = netactuate_server.test.id
  group_id  = %d
  ipv6      = %t
  redundant = %t
}

data "netactuate_bgp_sessions" "test" {
  mbpkgid = netactuate_bgp_sessions.test.mbpkgid
}
`, groupID, ipv6, redundant)
}
