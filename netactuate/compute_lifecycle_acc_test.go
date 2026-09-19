//go:build acctest

package netactuate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateServerLifecycle(t *testing.T) {
	name := testAccName("s")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	createConfig := testAccComputeServerConfig(name, locationID, imageID, plan, contractID, password, false)
	modifiedConfig := testAccComputeServerConfig(name, locationID, imageID, plan, contractID, password, true)

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
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_server.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_server", "netactuate_server.test"),
			testAccCheckComputeServerAPI("netactuate_server.test", name+".e.invalid", plan, locationID, imageID),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_server", "netactuate_server.test"),
			testAccCheckComputeServerAPI("netactuate_server.test", name+"-u.e.invalid", plan, locationID, imageID),
		), testAccLifecycleOptions{ImportStateVerifyIgnore: []string{"password", "allow_downsize_reboot", "package_billing", "params", "tags"}}),
	})
}

func TestAccNetactuateServerOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("so")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")

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
		Steps: []resource.TestStep{{
			Config: testAccComputeServerConfig(name, locationID, imageID, plan, contractID, password, false),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_server", "netactuate_server.test"),
				testAccCheckComputeServerAPI("netactuate_server.test", name+".e.invalid", plan, locationID, imageID),
				testAccDeleteServerOutOfBand("netactuate_server.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateServerNICLifecycle(t *testing.T) {
	name := testAccName("n")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	vlanID := testAccNICVLANID(t)
	createConfig := testAccServerNICConfig(name+".e.invalid", locationID, imageID, plan, contractID, password, vlanID, nil)
	modifiedConfig := testAccServerNICConfig(name+".e.invalid", locationID, imageID, plan, contractID, password, vlanID, intPtr(2))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_IMAGE_ID",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
				"NETACTUATE_ACC_SERVER_PASSWORD",
			)
			testAccNICVLANID(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckServerNICAndServerDestroy,
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_server_nic.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_server", "netactuate_server.test"),
			resource.TestCheckResourceAttrPair("netactuate_server_nic.test", "mbpkgid", "netactuate_server.test", "id"),
			resource.TestCheckResourceAttr("netactuate_server_nic.test", "customer_vlan_id", fmt.Sprintf("%d", vlanID)),
			testAccCheckServerNICAPI("netactuate_server_nic.test", vlanID, true),
		), resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("netactuate_server_nic.test", "customer_vlan_id", fmt.Sprintf("%d", vlanID)),
			resource.TestCheckResourceAttr("netactuate_server_nic.test", "attach_order", "2"),
			testAccCheckServerNICAPI("netactuate_server_nic.test", vlanID, true),
		), testAccLifecycleOptions{ImportStateIdFunc: testAccComputeLifecycleImportID("netactuate_server_nic.test")}),
	})
}

func TestAccNetactuateServerNICOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("no")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	vlanID := testAccNICVLANID(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_IMAGE_ID",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CONTRACT_ID",
				"NETACTUATE_ACC_SERVER_PASSWORD",
			)
			testAccNICVLANID(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckServerNICAndServerDestroy,
		Steps: []resource.TestStep{{
			Config: testAccServerNICConfig(name+".e.invalid", locationID, imageID, plan, contractID, password, vlanID, nil),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_server", "netactuate_server.test"),
				testAccCheckServerNICAPI("netactuate_server_nic.test", vlanID, true),
				testAccDeleteServerNICOutOfBand("netactuate_server_nic.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateServerOptionsLifecycle(t *testing.T) {
	name := testAccName("o")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	createConfig := testAccComputeServerOptionsConfig(name, locationID, imageID, plan, contractID, password, false)
	modifiedConfig := testAccComputeServerOptionsConfig(name, locationID, imageID, plan, contractID, password, true)

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
		Steps: testAccLifecycleSteps(createConfig, modifiedConfig, "netactuate_server_options.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_server", "netactuate_server.test"),
			testAccCheckServerOptionsAPI("netactuate_server_options.test", name+"-o.e.invalid"),
		), resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_server", "netactuate_server.test"),
			testAccCheckServerOptionsAPI("netactuate_server_options.test", name+"-u.e.invalid"),
		), testAccLifecycleOptions{ImportStateVerifyIgnore: []string{"raw_json"}}),
	})
}

func TestAccNetactuateServerOptionsOutOfBandDeleteIdempotent(t *testing.T) {
	name := testAccName("oo")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")

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
		Steps: []resource.TestStep{{
			Config: testAccComputeServerOptionsConfig(name, locationID, imageID, plan, contractID, password, false),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_server", "netactuate_server.test"),
				testAccCheckServerOptionsAPI("netactuate_server_options.test", name+"-o.e.invalid"),
				testAccDeleteServerOutOfBand("netactuate_server.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccNetactuateTagAssignmentLifecycle(t *testing.T) {
	tagName := testAccName("ta")
	serverName := testAccName("tas") + ".e.invalid"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")
	config := testAccComputeTagAssignmentConfig(tagName, serverName, locationID, imageID, plan, contractID, password)

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
		CheckDestroy:      testAccCheckServerDestroyAndTags(tagName),
		// netactuate_tag_assignment has no mutable fields, so this lifecycle has no modify step.
		Steps: testAccComputeNoModifyLifecycleSteps(config, "netactuate_tag_assignment.test", resource.ComposeTestCheckFunc(
			testAccTrackFromState("netactuate_server", "netactuate_server.test"),
			testAccCheckResourceHasTag("netactuate_server.test", tagName, true),
		), testAccLifecycleOptions{ImportStateIdFunc: testAccComputeLifecycleImportID("netactuate_tag_assignment.test")}),
	})
}

func TestAccNetactuateTagAssignmentOutOfBandDeleteIdempotent(t *testing.T) {
	tagName := testAccName("tao")
	serverName := testAccName("taos") + ".e.invalid"
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	contractID := testAccEnvString(t, "NETACTUATE_ACC_CONTRACT_ID")
	password := testAccEnvString(t, "NETACTUATE_ACC_SERVER_PASSWORD")

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
		CheckDestroy:      testAccCheckServerDestroyAndTags(tagName),
		Steps: []resource.TestStep{{
			Config: testAccComputeTagAssignmentConfig(tagName, serverName, locationID, imageID, plan, contractID, password),
			Check: resource.ComposeTestCheckFunc(
				testAccTrackFromState("netactuate_server", "netactuate_server.test"),
				testAccCheckResourceHasTag("netactuate_server.test", tagName, true),
				testAccDeleteTagAssignmentOutOfBand("netactuate_tag.test", "netactuate_server.test"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func testAccComputeNoModifyLifecycleSteps(config, resourceAddress string, check resource.TestCheckFunc, opts testAccLifecycleOptions) []resource.TestStep {
	return []resource.TestStep{
		{
			Config: config,
			Check:  check,
		},
		{
			Config:             config,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
		{
			ResourceName:            resourceAddress,
			ImportState:             true,
			ImportStateIdFunc:       opts.ImportStateIdFunc,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: opts.ImportStateVerifyIgnore,
		},
		{
			Config:             config,
			PlanOnly:           true,
			ExpectNonEmptyPlan: false,
		},
	}
}

func testAccComputeLifecycleImportID(address string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return testAccStateID(s, address)
	}
}

func testAccComputeServerConfig(name string, locationID, imageID int, plan, contractID, password string, updated bool) string {
	hostname := name + ".e.invalid"
	if updated {
		hostname = name + "-u.e.invalid"
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
    ignore_changes = [password]
  }
}
`, hostname, plan, locationID, imageID, password, contractID)
}

func testAccComputeServerOptionsConfig(name string, locationID, imageID int, plan, contractID, password string, updated bool) string {
	fqdn := name + "-o.e.invalid"
	if updated {
		fqdn = name + "-u.e.invalid"
	}
	return testAccServerNICConfigNoNIC(name+".e.invalid", locationID, imageID, plan, contractID, password) + fmt.Sprintf(`
data "netactuate_cloud_kernels" "test" {}

resource "netactuate_server_options" "test" {
  mbpkgid   = netactuate_server.test.id
  fqdn      = %q
  kernel_id = data.netactuate_cloud_kernels.test.kernels[0].kernel_id
}
`, fqdn)
}

func testAccComputeTagAssignmentConfig(tagName, serverName string, locationID, imageID int, plan, contractID, password string) string {
	return testAccServerNICConfigNoNIC(serverName, locationID, imageID, plan, contractID, password) + fmt.Sprintf(`
resource "netactuate_tag" "test" {
  name = %q
}

resource "netactuate_tag_assignment" "test" {
  tag_id        = netactuate_tag.test.id
  resource_name = %q
  identifier    = tonumber(netactuate_server.test.id)
}
`, tagName, resourceNameVirtualServer)
}

func testAccCheckComputeServerAPI(address, hostname, plan string, locationID, imageID int) resource.TestCheckFunc {
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
			return fmt.Errorf("API server hostname = %q, want %q", server.Name, hostname)
		}
		if !strings.EqualFold(server.Package, plan) {
			return fmt.Errorf("API server plan = %q, want %q", server.Package, plan)
		}
		if server.LocationID != locationID {
			return fmt.Errorf("API server location_id = %d, want %d", server.LocationID, locationID)
		}
		if server.OSID != imageID {
			return fmt.Errorf("API server image_id = %d, want %d", server.OSID, imageID)
		}
		return nil
	}
}

func testAccDeleteServerOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccID(s, address)
		if err != nil {
			return err
		}
		clients := testAccClients()
		jobID, err := clients.V2.DeleteServer(id, true)
		if err != nil {
			return err
		}
		if d := wait4JobStatus("delete", jobID, clients.V2); d != nil && d.HasError() {
			return fmt.Errorf("waiting for server %d delete job: %v", id, d)
		}
		return clients.V2.UnlinkServer(id)
	}
}

func testAccDeleteServerNICOutOfBand(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		mbpkgID, nicID, err := parseServerNICID(rs.Primary.ID)
		if err != nil {
			return err
		}
		return testAccClients().V2.DetachServerNIC(mbpkgID, nicID)
	}
}

func testAccDeleteTagAssignmentOutOfBand(tagAddress, serverAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tagID, err := testAccID(s, tagAddress)
		if err != nil {
			return err
		}
		identifier, err := testAccID(s, serverAddress)
		if err != nil {
			return err
		}
		return testAccClients().V2.RemoveTagResource(tagID, resourceNameVirtualServer, identifier)
	}
}
