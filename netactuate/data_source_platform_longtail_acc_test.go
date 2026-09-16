//go:build acctest

package netactuate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/netactuate/gona/gona"
)

func TestAccNetactuatePlatformAndLongTailDataSources_apiRead(t *testing.T) {
	location := testAccEnvString(t, "NETACTUATE_ACC_PLATFORM_LOCATION")
	lgAction := testAccEnvString(t, "NETACTUATE_ACC_LOOKING_GLASS_ACTION")
	lgTarget := testAccEnvString(t, "NETACTUATE_ACC_LOOKING_GLASS_TARGET")
	lgLocation := testAccEnvString(t, "NETACTUATE_ACC_LOOKING_GLASS_LOCATION")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_PLATFORM_LOCATION",
				"NETACTUATE_ACC_LOOKING_GLASS_ACTION",
				"NETACTUATE_ACC_LOOKING_GLASS_TARGET",
				"NETACTUATE_ACC_LOOKING_GLASS_LOCATION",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccNoopCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlatformAndLongTailDataSourcesConfig(location, lgAction, lgTarget, lgLocation),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountAgreementsDataSource("data.netactuate_account_agreements.test"),
					testAccCheckLocationDataSource("data.netactuate_location.test"),
					testAccCheckServicesDataSource("data.netactuate_services.test"),
					testAccCheckUsageContractDataSource("data.netactuate_usage_contract.test"),
					testAccCheckCurrentServerDataSource("data.netactuate_current_server.test"),
					testAccCheckRawJSONDataSource("data.netactuate_images_provisioning_jobs_count.test", func() ([]byte, error) {
						return testAccClients().V2.GetImagesProvisioningJobsCount()
					}),
					testAccCheckRawJSONDataSource("data.netactuate_unprovisioned_packages.test", func() ([]byte, error) {
						return testAccClients().V2.GetUnprovisionedPackages()
					}),
					testAccCheckPlatformDatacentersDataSource("data.netactuate_platform_datacenters.test", location),
					testAccCheckRawJSONDataSource("data.netactuate_platform_looking_glass_init.test", func() ([]byte, error) {
						result, err := testAccClients().V2.GetPlatformLookingGlassInit()
						if err != nil {
							return nil, err
						}
						return result.Raw, nil
					}),
					testAccCheckRawJSONDataSource("data.netactuate_platform_looking_glass.test", func() ([]byte, error) {
						full := 0
						result, err := testAccClients().V2.ExecutePlatformLookingGlass(gona.PlatformLookingGlassExecuteOptions{
							Action:   &lgAction,
							Target:   &lgTarget,
							Location: &lgLocation,
							Full:     &full,
						})
						if err != nil {
							return nil, err
						}
						return result.Raw, nil
					}),
				),
			},
		},
	})
}

func TestAccNetactuateIDBackedDataSources_apiRead(t *testing.T) {
	ddosRuleID := testAccEnvInt(t, "NETACTUATE_ACC_DDOS_RULE_ID")
	changeLogID := testAccEnvInt(t, "NETACTUATE_ACC_PLATFORM_CHANGE_LOG_ID")
	maintenanceID := testAccEnvInt(t, "NETACTUATE_ACC_PLATFORM_MAINTENANCE_ID")
	transitPackageID := testAccEnvInt(t, "NETACTUATE_ACC_TRANSIT_MBPKGID")
	virtualServerID := testAccEnvInt(t, "NETACTUATE_ACC_VM_MBPKGID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_DDOS_RULE_ID",
				"NETACTUATE_ACC_PLATFORM_CHANGE_LOG_ID",
				"NETACTUATE_ACC_PLATFORM_MAINTENANCE_ID",
				"NETACTUATE_ACC_TRANSIT_MBPKGID",
				"NETACTUATE_ACC_VM_MBPKGID",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccNoopCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIDBackedDataSourcesConfig(ddosRuleID, changeLogID, maintenanceID, transitPackageID, virtualServerID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDDoSRuleDataSource("data.netactuate_ddos_rule.test", ddosRuleID),
					testAccCheckPlatformChangeLogEntryDataSource("data.netactuate_platform_change_log_entry.test", changeLogID),
					testAccCheckRawJSONDataSource("data.netactuate_platform_maintenance_info.test", func() ([]byte, error) {
						result, err := testAccClients().V2.GetPlatformMaintenanceInfo(maintenanceID)
						if err != nil {
							return nil, err
						}
						return result.Raw, nil
					}),
					testAccCheckTransitPackageDataSource("data.netactuate_transit_package.test", transitPackageID),
					testAccCheckVirtualServerContractDataSource("data.netactuate_virtual_server_contract.test", virtualServerID),
				),
			},
		},
	})
}

func TestAccNetactuateSecretValuesDataSource_apiReadCreatedValue(t *testing.T) {
	name := testAccName("secret-values")
	key := testAccName("secret-key")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSecretListValueDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretValuesDataSourceConfig(name, key),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_secret_list", "netactuate_secret_list.test"),
					testAccTrackFromState("netactuate_secret_list_value", "netactuate_secret_list_value.test"),
					testAccCheckSecretValuesDataSource("data.netactuate_secret_values.test", "netactuate_secret_list_value.test"),
				),
			},
		},
	})
}

func TestAccNetactuateTagResourcesAndLogsDataSources_apiReadCreatedTag(t *testing.T) {
	name := testAccName("tag-details")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckTagsDestroyed(name),
		Steps: []resource.TestStep{
			{
				Config: testAccTagDetailsDataSourcesConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTagResourcesDataSource("data.netactuate_tag_resources.test", "netactuate_tag.test"),
					testAccCheckTagLogsDataSource("data.netactuate_tag_logs.test", "netactuate_tag.test"),
				),
			},
		},
	})
}

func TestAccNetactuateVPCsDataSource_apiReadCreatedVPC(t *testing.T) {
	name := testAccName("vpcs-ds")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVPCsDataSourceConfig(name, locationID),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccCheckVPCsDataSource("data.netactuate_vpcs.test", "netactuate_vpc.test"),
				),
			},
		},
	})
}

func TestAccNetactuateHTTPLoadbalancerGroupsDataSource_apiReadCreatedGroup(t *testing.T) {
	name := testAccName("httplb-ds")
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPLBGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPLBGroupsDataSourceConfig(name, locationID),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_vpc", "netactuate_vpc.test"),
					testAccTrackHTTPLBGroupFromState("netactuate_http_loadbalancer_group.test"),
					testAccCheckHTTPLoadbalancerGroupsDataSource("data.netactuate_http_loadbalancer_groups.test", "netactuate_vpc.test", "netactuate_http_loadbalancer_group.test"),
				),
			},
		},
	})
}

func TestAccNetactuateOIDCClientBareMetalAllowList_createImportDataSourceAndOutOfBandDelete(t *testing.T) {
	label := testAccName("oidc-bm-allow")
	mbpkgid := testAccEnvInt(t, "NETACTUATE_ACC_BARE_METAL_MBPKGID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_BARE_METAL_MBPKGID") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckOIDCBareMetalAllowListAndClientDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOIDCClientBareMetalAllowListConfig(label, mbpkgid),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackOIDCClient("netactuate_oidc_client.test"),
					resource.TestCheckResourceAttr("netactuate_oidc_client_bare_metal_allow_list.test", "mbpkgid", strconv.Itoa(mbpkgid)),
					testAccCheckOIDCBareMetalAllowListAPI("netactuate_oidc_client.test", mbpkgid, true),
					testAccCheckOIDCBareMetalServersDataSource("data.netactuate_oidc_client_bare_metal_servers.test", "netactuate_oidc_client.test"),
				),
			},
			{
				ResourceName:      "netactuate_oidc_client_bare_metal_allow_list.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				PreConfig: func() {
					clientID := testAccOIDCClientIDByLabel(t, label)
					if err := testAccClients().V3.RemoveOIDCClientBareMetalServer(clientID, mbpkgid); err != nil {
						t.Fatalf("out of band bare metal allow list delete: %v", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckGone("netactuate_oidc_client_bare_metal_allow_list.test"),
			},
		},
	})
}

func TestAccNetactuateSSLCertificatesDataSource_apiRead(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccNoopCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
data "netactuate_ssl_certificates" "test" {}
`,
				Check: testAccCheckSSLCertificatesDataSource("data.netactuate_ssl_certificates.test"),
			},
		},
	})
}

func testAccNoopCheckDestroy(s *terraform.State) error {
	return nil
}

func testAccPlatformAndLongTailDataSourcesConfig(location, lgAction, lgTarget, lgLocation string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
data "netactuate_account_agreements" "test" {}
data "netactuate_location" "test" {}
data "netactuate_services" "test" {}
data "netactuate_usage_contract" "test" {}
data "netactuate_current_server" "test" {}
data "netactuate_images_provisioning_jobs_count" "test" {}
data "netactuate_unprovisioned_packages" "test" {}

data "netactuate_platform_datacenters" "test" {
  location = %q
}

data "netactuate_platform_looking_glass_init" "test" {}

data "netactuate_platform_looking_glass" "test" {
  action   = %q
  target   = %q
  location = %q
  full     = 0
}
`, location, lgAction, lgTarget, lgLocation)
}

func testAccIDBackedDataSourcesConfig(ddosRuleID, changeLogID, maintenanceID, transitPackageID, virtualServerID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
data "netactuate_ddos_rule" "test" {
  rule_id = %d
}

data "netactuate_platform_change_log_entry" "test" {
  change_log_id = %d
}

data "netactuate_platform_maintenance_info" "test" {
  maintenance_id = %d
}

data "netactuate_transit_package" "test" {
  mbpkgid = %d
}

data "netactuate_virtual_server_contract" "test" {
  mbpkgid = %d
}
`, ddosRuleID, changeLogID, maintenanceID, transitPackageID, virtualServerID)
}

func testAccSecretValuesDataSourceConfig(name, key string) string {
	return testAccSecretListValueConfig(name, key, testAccSecretValueInitial) + `
data "netactuate_secret_values" "test" {
  depends_on = [netactuate_secret_list_value.test]
}
`
}

func testAccTagDetailsDataSourcesConfig(name string) string {
	return testAccTagConfig(name, "tag detail data source") + `
data "netactuate_tag_resources" "test" {
  tag_id     = netactuate_tag.test.id
  depends_on = [netactuate_tag.test]
}

data "netactuate_tag_logs" "test" {
  tag_id     = netactuate_tag.test.id
  depends_on = [netactuate_tag.test]
}
`
}

func testAccVPCsDataSourceConfig(name string, locationID int) string {
	return testAccVPCConfigID(name, locationID) + `
data "netactuate_vpcs" "test" {
  depends_on = [netactuate_vpc.test]
}
`
}

func testAccHTTPLBGroupsDataSourceConfig(name string, locationID int) string {
	return testAccHTTPLBGroupConfig(name, locationID, false, false) + `
data "netactuate_http_loadbalancer_groups" "test" {
  http_loadbalancer_id = netactuate_vpc.test.http_loadbalancer_id
  depends_on           = [netactuate_http_loadbalancer_group.test]
}
`
}

func testAccOIDCClientBareMetalAllowListConfig(label string, mbpkgid int) string {
	return testAccOIDCClientConfig(label, "client for bare metal allow list", 300, "https://api.example.com/oidc-audience", true, true) + fmt.Sprintf(`
resource "netactuate_oidc_client_bare_metal_allow_list" "test" {
  oidc_client_id = netactuate_oidc_client.test.oidc_client_id
  mbpkgid        = %d
}

data "netactuate_oidc_client_bare_metal_servers" "test" {
  oidc_client_id = netactuate_oidc_client.test.oidc_client_id
  depends_on     = [netactuate_oidc_client_bare_metal_allow_list.test]
}
`, mbpkgid)
}

func testAccCheckRawJSONDataSource(address string, read func() ([]byte, error)) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		raw, err := read()
		if err != nil {
			return err
		}
		want, err := compactSurfaceJSON(json.RawMessage(raw))
		if err != nil {
			return err
		}
		got := rs.Primary.Attributes["raw_json"]
		if got == "" {
			return fmt.Errorf("%s raw_json is empty", address)
		}
		gotCompact, err := compactSurfaceJSON(json.RawMessage(got))
		if err != nil {
			return fmt.Errorf("%s raw_json is not valid JSON: %w", address, err)
		}
		if gotCompact != want {
			return fmt.Errorf("%s raw_json does not match API response", address)
		}
		return nil
	}
}

func testAccCheckAccountAgreementsDataSource(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		agreements, err := testAccClients().V2.ListAccountAgreements()
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["agreements.#"]; got != strconv.Itoa(len(agreements)) {
			return fmt.Errorf("%s agreement count = %q, want API count %d", address, got, len(agreements))
		}
		if len(agreements) > 0 && rs.Primary.Attributes["agreements.0.name"] != agreements[0].Name {
			return fmt.Errorf("%s first agreement name does not match API", address)
		}
		return nil
	}
}

func testAccCheckLocationDataSource(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		location, err := testAccClients().V2.GetLocationByCurrentIP()
		if err != nil {
			return err
		}
		if rs.Primary.Attributes["ip"] != location.IP {
			return fmt.Errorf("%s ip does not match API", address)
		}
		if rs.Primary.Attributes["location"] != location.Location {
			return fmt.Errorf("%s location does not match API", address)
		}
		return nil
	}
}

func testAccCheckServicesDataSource(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		services, err := testAccClients().V2.GetServices()
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["services.#"]; got != strconv.Itoa(len(services)) {
			return fmt.Errorf("%s service count = %q, want API count %d", address, got, len(services))
		}
		return nil
	}
}

func testAccCheckUsageContractDataSource(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		usage, err := testAccClients().V2.GetContractUsage()
		if err != nil {
			return err
		}
		return testAccCheckContractUsageAttrs(address, rs, usage, "contract_usage_id")
	}
}

func testAccCheckCurrentServerDataSource(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		server, err := testAccClients().V2.GetCurrentServer()
		if err != nil {
			return err
		}
		if rs.Primary.Attributes["mbpkgid"] != strconv.Itoa(server.ID) {
			return fmt.Errorf("%s mbpkgid does not match API", address)
		}
		if rs.Primary.Attributes["hostname"] != server.Name {
			return fmt.Errorf("%s hostname does not match API", address)
		}
		return nil
	}
}

func testAccCheckPlatformDatacentersDataSource(address, location string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		datacenters, err := testAccClients().V2.GetPlatformDatacenters(location)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["datacenters.#"]; got != strconv.Itoa(len(datacenters)) {
			return fmt.Errorf("%s datacenter count = %q, want API count %d", address, got, len(datacenters))
		}
		if len(datacenters) > 0 && rs.Primary.Attributes["datacenters.0.datacenter_id"] != strconv.Itoa(datacenters[0].ID) {
			return fmt.Errorf("%s first datacenter id does not match API", address)
		}
		return nil
	}
}

func testAccCheckDDoSRuleDataSource(address string, ruleID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		rule, err := testAccClients().V2.GetDDoSRule(ruleID)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["rules.#"]; got != "1" {
			return fmt.Errorf("%s rule count = %q, want 1", address, got)
		}
		if rs.Primary.Attributes["rules.0.rule_id"] != strconv.Itoa(rule.RuleID) {
			return fmt.Errorf("%s rule_id does not match API", address)
		}
		if rs.Primary.Attributes["rules.0.rule_name"] != rule.RuleName {
			return fmt.Errorf("%s rule_name does not match API", address)
		}
		return nil
	}
}

func testAccCheckPlatformChangeLogEntryDataSource(address string, changeLogID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		entry, err := testAccClients().V2.GetPlatformChangeLogEntry(changeLogID)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["entry.#"]; got != "1" {
			return fmt.Errorf("%s entry count = %q, want 1", address, got)
		}
		if rs.Primary.Attributes["entry.0.change_log_id"] != entry.ChangeLogID {
			return fmt.Errorf("%s change_log_id does not match API", address)
		}
		if rs.Primary.Attributes["entry.0.title"] != entry.Title {
			return fmt.Errorf("%s title does not match API", address)
		}
		return nil
	}
}

func testAccCheckTransitPackageDataSource(address string, mbpkgid int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		pkg, err := testAccClients().V2.GetTransitPackage(mbpkgid)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["packages.#"]; got != "1" {
			return fmt.Errorf("%s package count = %q, want 1", address, got)
		}
		if rs.Primary.Attributes["packages.0.mbpkgid"] != strconv.Itoa(pkg.MBPkgID) {
			return fmt.Errorf("%s mbpkgid does not match API", address)
		}
		if rs.Primary.Attributes["packages.0.fqdn"] != pkg.FQDN {
			return fmt.Errorf("%s fqdn does not match API", address)
		}
		return nil
	}
}

func testAccCheckVirtualServerContractDataSource(address string, mbpkgid int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		contract, err := testAccClients().V2.GetVirtualServerContract(mbpkgid)
		if err != nil {
			return err
		}
		return testAccCheckContractUsageAttrs(address, rs, contract, "contract_id")
	}
}

func testAccCheckSecretValuesDataSource(address, valueAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		valueID, err := testAccID(s, valueAddress)
		if err != nil {
			return err
		}
		values, err := testAccClients().V2.GetAllSecretValues()
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["values.#"]; got != strconv.Itoa(len(values)) {
			return fmt.Errorf("%s value count = %q, want API count %d", address, got, len(values))
		}
		for i, value := range values {
			if value.ID == valueID {
				prefix := fmt.Sprintf("values.%d", i)
				if rs.Primary.Attributes[prefix+".secret_key"] != value.SecretKey {
					return fmt.Errorf("%s %s.secret_key does not match API", address, prefix)
				}
				return nil
			}
		}
		return fmt.Errorf("%s did not include created secret value %d", address, valueID)
	}
}

func testAccCheckTagResourcesDataSource(address, tagAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		tagID, err := testAccID(s, tagAddress)
		if err != nil {
			return err
		}
		resources, err := testAccClients().V2.GetTagResources(tagID)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["resources.#"]; got != strconv.Itoa(len(resources)) {
			return fmt.Errorf("%s resource count = %q, want API count %d", address, got, len(resources))
		}
		return nil
	}
}

func testAccCheckTagLogsDataSource(address, tagAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		tagID, err := testAccID(s, tagAddress)
		if err != nil {
			return err
		}
		logs, err := testAccClients().V2.GetTagLogs(tagID)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["logs.#"]; got != strconv.Itoa(len(logs)) {
			return fmt.Errorf("%s log count = %q, want API count %d", address, got, len(logs))
		}
		if len(logs) > 0 && rs.Primary.Attributes["logs.0.tag_id"] != strconv.Itoa(logs[0].TagID) {
			return fmt.Errorf("%s first log tag_id does not match API", address)
		}
		return nil
	}
}

func testAccCheckVPCsDataSource(address, vpcAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		vpcID, err := testAccID(s, vpcAddress)
		if err != nil {
			return err
		}
		count, err := strconv.Atoi(rs.Primary.Attributes["vpcs.#"])
		if err != nil {
			return fmt.Errorf("%s VPC count is invalid: %w", address, err)
		}
		for i := 0; i < count; i++ {
			if rs.Primary.Attributes[fmt.Sprintf("vpcs.%d.vpc_id", i)] == strconv.Itoa(vpcID) {
				return nil
			}
		}
		return fmt.Errorf("%s did not include created VPC %d", address, vpcID)
	}
}

func testAccCheckHTTPLoadbalancerGroupsDataSource(address, vpcAddress, groupAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		vpc, err := testAccResource(s, vpcAddress)
		if err != nil {
			return err
		}
		lbID, err := strconv.Atoi(vpc.Primary.Attributes["http_loadbalancer_id"])
		if err != nil {
			return err
		}
		_, groupID, err := testAccCompositeID(s, groupAddress)
		if err != nil {
			return err
		}
		groups, err := testAccClients().V3.GetHTTPLBGroups(lbID)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["groups.#"]; got != strconv.Itoa(len(groups)) {
			return fmt.Errorf("%s group count = %q, want API count %d", address, got, len(groups))
		}
		for i, group := range groups {
			if group.HTTPGroupID == groupID {
				prefix := fmt.Sprintf("groups.%d", i)
				if rs.Primary.Attributes[prefix+".name"] != group.Name {
					return fmt.Errorf("%s %s.name does not match API", address, prefix)
				}
				if rs.Primary.Attributes[prefix+".match_ports"] != group.Match.Ports {
					return fmt.Errorf("%s %s.match_ports does not match API", address, prefix)
				}
				return nil
			}
		}
		return fmt.Errorf("%s did not include created HTTP load balancer group %d", address, groupID)
	}
}

func testAccCheckOIDCBareMetalServersDataSource(address, clientAddress string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		clientID, err := testAccID(s, clientAddress)
		if err != nil {
			return err
		}
		servers, err := testAccClients().V3.GetOIDCClientBareMetalServers(clientID)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["servers.#"]; got != strconv.Itoa(len(servers)) {
			return fmt.Errorf("%s server count = %q, want API count %d", address, got, len(servers))
		}
		if len(servers) > 0 && rs.Primary.Attributes["servers.0.mbpkgid"] != strconv.Itoa(servers[0].MBPkgID) {
			return fmt.Errorf("%s first mbpkgid does not match API", address)
		}
		return nil
	}
}

func testAccCheckSSLCertificatesDataSource(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		certs, err := testAccClients().V3.GetSSLCertificates()
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["certificates.#"]; got != strconv.Itoa(len(certs)) {
			return fmt.Errorf("%s certificate count = %q, want API count %d", address, got, len(certs))
		}
		if len(certs) > 0 && rs.Primary.Attributes["certificates.0.ssl_certificate_id"] != strconv.Itoa(certs[0].SSLCertificateID) {
			return fmt.Errorf("%s first certificate id does not match API", address)
		}
		return nil
	}
}

func testAccCheckOIDCBareMetalAllowListAPI(clientAddress string, mbpkgid int, want bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clientID, err := testAccID(s, clientAddress)
		if err != nil {
			return err
		}
		servers, err := testAccClients().V3.GetOIDCClientBareMetalServers(clientID)
		if err != nil {
			return err
		}
		for _, server := range servers {
			if server.MBPkgID == mbpkgid {
				if want {
					return nil
				}
				return fmt.Errorf("OIDC client %d still allows bare metal package %d", clientID, mbpkgid)
			}
		}
		if want {
			return fmt.Errorf("OIDC client %d does not allow bare metal package %d", clientID, mbpkgid)
		}
		return nil
	}
}

func testAccCheckOIDCBareMetalAllowListAndClientDestroy(s *terraform.State) error {
	if err := testAccCheckOIDCBareMetalAllowListDestroy(s); err != nil {
		return err
	}
	return testAccCheckOIDCClientDestroy("netactuate_oidc_client.test")(s)
}

func testAccCheckOIDCBareMetalAllowListDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "netactuate_oidc_client_bare_metal_allow_list" {
			continue
		}
		clientID, mbpkgid, err := parseTwoPartID(rs.Primary.ID, "clientId", "mbpkgid")
		if err != nil {
			return err
		}
		servers, err := testAccClients().V3.GetOIDCClientBareMetalServers(clientID)
		if err != nil {
			if gona.IsV3NotFound(err) {
				continue
			}
			return err
		}
		for _, server := range servers {
			if server.MBPkgID == mbpkgid {
				return fmt.Errorf("OIDC client %d still allows bare metal package %d", clientID, mbpkgid)
			}
		}
	}
	return nil
}

func testAccOIDCClientIDByLabel(t *testing.T, label string) int {
	t.Helper()
	clients, err := testAccClients().V3.GetOIDCClients()
	if err != nil {
		t.Fatalf("list OIDC clients: %v", err)
	}
	for _, client := range clients {
		if client.Label == label {
			return client.ClientID
		}
	}
	t.Fatalf("OIDC client with label %q not found", label)
	return 0
}

func testAccCheckContractUsageAttrs(address string, rs *terraform.ResourceState, usage gona.ContractUsage, idAttr string) error {
	if usage.ID != nil && rs.Primary.Attributes[idAttr] != strconv.Itoa(*usage.ID) {
		return fmt.Errorf("%s %s does not match API", address, idAttr)
	}
	if usage.ContractType != nil && rs.Primary.Attributes["contract_type"] != *usage.ContractType {
		return fmt.Errorf("%s contract_type does not match API", address)
	}
	return nil
}

func testAccResource(s *terraform.State, address string) (*terraform.ResourceState, error) {
	rs, ok := s.RootModule().Resources[address]
	if !ok {
		return nil, fmt.Errorf("not found: %s", address)
	}
	if rs.Primary == nil {
		return nil, fmt.Errorf("%s has no primary state", address)
	}
	return rs, nil
}

func testAccRequireNonEmptyString(value, what string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is empty", what)
	}
	return nil
}

func TestAccNetactuateComputeImageServerRoutingDataSources_apiRead(t *testing.T) {
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_LOCATION_ID")
	locationName := testAccEnvString(t, "NETACTUATE_ACC_LOCATION_NAME")
	imageID := testAccEnvInt(t, "NETACTUATE_ACC_IMAGE_ID")
	plan := testAccEnvString(t, "NETACTUATE_ACC_PLAN")
	poolID := testAccEnvInt(t, "NETACTUATE_ACC_CLOUD_POOL_ID")
	vlanID := testAccEnvInt(t, "NETACTUATE_ACC_CUSTOMER_VLAN_ID")
	floatingIPv4ID := testAccEnvInt(t, "NETACTUATE_ACC_FLOATING_IPV4_ID")
	serverID := testAccEnvInt(t, "NETACTUATE_ACC_SERVER_ID")
	routerID := testAccEnvInt(t, "NETACTUATE_ACC_ROUTER_ID")
	vrfID := testAccEnvInt(t, "NETACTUATE_ACC_ROUTER_VRF_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t,
				"NETACTUATE_ACC_LOCATION_ID",
				"NETACTUATE_ACC_LOCATION_NAME",
				"NETACTUATE_ACC_IMAGE_ID",
				"NETACTUATE_ACC_PLAN",
				"NETACTUATE_ACC_CLOUD_POOL_ID",
				"NETACTUATE_ACC_CUSTOMER_VLAN_ID",
				"NETACTUATE_ACC_FLOATING_IPV4_ID",
				"NETACTUATE_ACC_SERVER_ID",
				"NETACTUATE_ACC_ROUTER_ID",
				"NETACTUATE_ACC_ROUTER_VRF_ID",
			)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccNoopCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeImageServerRoutingDataSourcesConfig(locationID, locationName, plan, poolID, vlanID, floatingIPv4ID, serverID, routerID, vrfID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudBaseImagesDataSource("data.netactuate_cloud_base_images.test", imageID),
					testAccCheckCloudPrivateImagesDataSource("data.netactuate_cloud_private_images.test"),
					testAccCheckCloudKernelsDataSource("data.netactuate_cloud_kernels.test"),
					testAccCheckCloudLocationDataSource("data.netactuate_cloud_location.test", locationID),
					testAccCheckCloudPoolDataSource("data.netactuate_cloud_pool.test", poolID),
					testAccCheckCloudVLANDataSource("data.netactuate_cloud_vlan.test", vlanID),
					testAccCheckCloudLocationVLANsDataSource("data.netactuate_cloud_location_vlans.test", locationID),
					testAccCheckCloudNetworkingLocationsDataSource("data.netactuate_cloud_networking_locations.test"),
					testAccCheckCloudFloatingIPv4VMsDataSource("data.netactuate_cloud_floating_ipv4_vms.test", floatingIPv4ID),
					testAccCheckCloudDeploySizesDataSource("data.netactuate_cloud_deploy_sizes.test", locationName),
					testAccCheckServerIPFamilyDataSource("data.netactuate_server_ipv4.test", serverID, "ipv4"),
					testAccCheckServerIPFamilyDataSource("data.netactuate_server_ipv6.test", serverID, "ipv6"),
					testAccCheckCloudRoutingMeshesDataSource("data.netactuate_cloud_routing_meshes.test"),
					testAccCheckCloudRoutingRoutersDataSource("data.netactuate_cloud_routing_routers.test"),
					testAccCheckCloudRoutingRouterInterfacesDataSource("data.netactuate_cloud_routing_router_interfaces.test", routerID),
					testAccCheckCloudRoutingRouterVRFsDataSource("data.netactuate_cloud_routing_router_vrfs.test", routerID),
					testAccCheckCloudRoutingRouterVRFListDataSource("data.netactuate_cloud_routing_router_vrf_bgp_neighbors.test", "neighbors", routerID, vrfID, func(c *ProviderClients) (int, error) {
						values, err := c.V3.ListRouterVRFBGPNeighbors(routerID, vrfID)
						return len(values), err
					}),
					testAccCheckCloudRoutingRouterVRFListDataSource("data.netactuate_cloud_routing_router_vrf_interfaces.test", "interfaces", routerID, vrfID, func(c *ProviderClients) (int, error) {
						values, err := c.V3.ListRouterVRFInterfaces(routerID, vrfID)
						return len(values), err
					}),
					testAccCheckCloudRoutingRouterVRFListDataSource("data.netactuate_cloud_routing_router_vrf_tunnels.test", "tunnels", routerID, vrfID, func(c *ProviderClients) (int, error) {
						values, err := c.V3.ListRouterVRFTunnels(routerID, vrfID)
						return len(values), err
					}),
					testAccCheckCloudRoutingOverviewDataSource("data.netactuate_cloud_routing_router_routing_overview.test", routerID, vrfID),
					testAccCheckRawJSONDataSource("data.netactuate_cloud_plan_id.test", func() ([]byte, error) { return testAccClients().V2.GetPlanID(plan) }),
					testAccCheckRawJSONDataSource("data.netactuate_cloud_extras.test", func() ([]byte, error) { return testAccClients().V2.GetCloudExtras(serverID) }),
					testAccCheckRawJSONDataSource("data.netactuate_cloud_iplimits.test", func() ([]byte, error) { return testAccClients().V2.GetIPLimits(serverID) }),
					testAccCheckRawJSONDataSource("data.netactuate_server_deployment_info.test", func() ([]byte, error) {
						return testAccClients().V2.GetServerDeploymentInfo("")
					}),
					testAccCheckRawJSONDataSource("data.netactuate_cloud_storage_locations.test", func() ([]byte, error) {
						return testAccClients().V2.GetStorageLocations(nil)
					}),
					testAccCheckRawJSONDataSource("data.netactuate_cloud_scaling_options.test", func() ([]byte, error) {
						includeCurrent := true
						minCPUs, minRAM := 1, 1
						return testAccClients().V2.GetScalingOptions(serverID, &gona.ScalingOptionsRequest{IncludeCurrentPlan: &includeCurrent, MinCPUs: &minCPUs, MinRAM: &minRAM})
					}),
					testAccCheckRawJSONDataSource("data.netactuate_server_vnc_status.test", func() ([]byte, error) {
						return testAccClients().V2.GetServerVNCStatus(serverID)
					}),
					testAccCheckRawJSONDataSource("data.netactuate_server_networkips.test", func() ([]byte, error) {
						return testAccClients().V2.GetServerNetworkIPs(serverID)
					}),
					testAccCheckRawJSONDataSource("data.netactuate_server_bgp_sessions_raw.test", func() ([]byte, error) {
						return testAccClients().V2.GetServerBGPSessions(serverID, "")
					}),
					testAccCheckRawJSONDataSource("data.netactuate_server_summary.test", func() ([]byte, error) {
						return testAccClients().V2.GetServerSummary(serverID)
					}),
				),
			},
		},
	})
}

func testAccComputeImageServerRoutingDataSourcesConfig(locationID int, locationName, plan string, poolID, vlanID, floatingIPv4ID, serverID, routerID, vrfID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
data "netactuate_cloud_base_images" "test" {}
data "netactuate_cloud_private_images" "test" {}
data "netactuate_cloud_kernels" "test" {}
data "netactuate_cloud_networking_locations" "test" {}
data "netactuate_server_deployment_info" "test" {}
data "netactuate_cloud_storage_locations" "test" {}
data "netactuate_cloud_routing_meshes" "test" {}
data "netactuate_cloud_routing_routers" "test" {}
data "netactuate_cloud_location" "test" { location_id = %d }
data "netactuate_cloud_pool" "test" { cloud_pool_id = %d }
data "netactuate_cloud_vlan" "test" { customer_vlan_id = %d }
data "netactuate_cloud_location_vlans" "test" { location_id = %d }
data "netactuate_cloud_floating_ipv4_vms" "test" { floating_ipv4_id = %d }
data "netactuate_cloud_deploy_sizes" "test" {
  location = %q
  min_cpu  = 1
  min_ram  = 1
}
data "netactuate_cloud_plan_id" "test" { plan_name = %q }
data "netactuate_cloud_extras" "test" { mbpkgid = %d }
data "netactuate_cloud_iplimits" "test" { mbpkgid = %d }
data "netactuate_cloud_scaling_options" "test" {
  mbpkgid              = %d
  include_current_plan = true
  min_cpus             = 1
  min_ram              = 1
}
data "netactuate_server_vnc_status" "test" { mbpkgid = %d }
data "netactuate_server_ipv4" "test" { mbpkgid = %d }
data "netactuate_server_ipv6" "test" { mbpkgid = %d }
data "netactuate_server_networkips" "test" { mbpkgid = %d }
data "netactuate_server_bgp_sessions_raw" "test" { mbpkgid = %d }
data "netactuate_server_summary" "test" { mbpkgid = %d }
data "netactuate_cloud_routing_router_interfaces" "test" { router_id = %d }
data "netactuate_cloud_routing_router_vrfs" "test" { router_id = %d }
data "netactuate_cloud_routing_router_vrf_bgp_neighbors" "test" {
  router_id = %d
  vrf_id    = %d
}
data "netactuate_cloud_routing_router_vrf_interfaces" "test" {
  router_id = %d
  vrf_id    = %d
}
data "netactuate_cloud_routing_router_vrf_tunnels" "test" {
  router_id = %d
  vrf_id    = %d
}
data "netactuate_cloud_routing_router_routing_overview" "test" {
  router_id = %d
  vrf_id    = %d
}
`, locationID, poolID, vlanID, locationID, floatingIPv4ID, locationName, plan, serverID, serverID, serverID, serverID, serverID, serverID, serverID, serverID, serverID, routerID, routerID, routerID, vrfID, routerID, vrfID, routerID, vrfID, routerID, vrfID)
}

func testAccCheckDataSourceListCount(s *terraform.State, address, attr string, want int) error {
	rs, err := testAccResource(s, address)
	if err != nil {
		return err
	}
	if got := rs.Primary.Attributes[attr+".#"]; got != strconv.Itoa(want) {
		return fmt.Errorf("%s %s count = %q, want API count %d", address, attr, got, want)
	}
	return nil
}

func testAccCheckDataSourceListHasInt(s *terraform.State, address, attr, field string, want int) error {
	rs, err := testAccResource(s, address)
	if err != nil {
		return err
	}
	count, err := strconv.Atoi(rs.Primary.Attributes[attr+".#"])
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if got := rs.Primary.Attributes[fmt.Sprintf("%s.%d.%s", attr, i, field)]; got == strconv.Itoa(want) {
			return nil
		}
	}
	return fmt.Errorf("%s %s has no item with %s = %d", address, attr, field, want)
}

func testAccCheckCloudBaseImagesDataSource(address string, imageID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		images, err := testAccClients().V2.GetBaseImages()
		if err != nil {
			return err
		}
		if err := testAccCheckDataSourceListCount(s, address, "base_images", len(images)); err != nil {
			return err
		}
		return testAccCheckDataSourceListHasInt(s, address, "base_images", "image_id", imageID)
	}
}

func testAccCheckCloudPrivateImagesDataSource(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		images, err := testAccClients().V2.GetPrivateImages()
		if err != nil {
			return err
		}
		return testAccCheckDataSourceListCount(s, address, "private_images", len(images))
	}
}

func testAccCheckCloudKernelsDataSource(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		kernels, err := testAccClients().V2.GetKernels()
		if err != nil {
			return err
		}
		if err := testAccCheckDataSourceListCount(s, address, "kernels", len(kernels)); err != nil {
			return err
		}
		if len(kernels) == 0 {
			return fmt.Errorf("kernels API returned no kernels")
		}
		return testAccCheckDataSourceListHasInt(s, address, "kernels", "kernel_id", kernels[0].ID)
	}
}

func testAccCheckCloudLocationDataSource(address string, locationID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		location, err := testAccClients().V2.GetCloudLocation(locationID)
		if err != nil {
			return err
		}
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		return testAccCheckAttrs(rs.Primary.Attributes, "", map[string]string{"name": location.Name, "location": location.Location, "city": location.City, "country": location.Country})
	}
}

func testAccCheckCloudPoolDataSource(address string, poolID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		pool, err := testAccClients().V2.GetCloudPool(poolID)
		if err != nil {
			return err
		}
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		return testAccCheckAttrs(rs.Primary.Attributes, "", map[string]string{"name": pool.Name, "private": strconv.Itoa(pool.Private)})
	}
}

func testAccCheckCloudVLANDataSource(address string, vlanID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vlan, err := testAccClients().V2.GetCustomerVLAN(vlanID)
		if err != nil {
			return err
		}
		if err := testAccCheckDataSourceListCount(s, address, "vlan", 1); err != nil {
			return err
		}
		return testAccCheckDataSourceListHasInt(s, address, "vlan", "customer_vlan_id", vlan.ID)
	}
}

func testAccCheckCloudLocationVLANsDataSource(address string, locationID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vlans, err := testAccClients().V2.ListCustomerVLANsAtLocation(locationID)
		if err != nil {
			return err
		}
		return testAccCheckDataSourceListCount(s, address, "vlans", len(vlans))
	}
}

func testAccCheckCloudNetworkingLocationsDataSource(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		locations, err := testAccClients().V3.ListCloudNetworkingLocations()
		if err != nil {
			return err
		}
		return testAccCheckDataSourceListCount(s, address, "locations", len(locations))
	}
}

func testAccCheckCloudFloatingIPv4VMsDataSource(address string, floatingIPv4ID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vms, err := testAccClients().V3.ListCloudFloatingIPv4VMs(floatingIPv4ID)
		if err != nil {
			return err
		}
		return testAccCheckDataSourceListCount(s, address, "vms", len(vms))
	}
}

func testAccCheckCloudDeploySizesDataSource(address, locationName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		minCPU, minRAM := 1, 1
		sizes, err := testAccClients().V2.GetDeploySizes(locationName, &gona.DeploySizesRequest{MinCPU: &minCPU, MinRAM: &minRAM})
		if err != nil {
			return err
		}
		return testAccCheckDataSourceListCount(s, address, "sizes", len(sizes))
	}
}

func testAccCheckServerIPFamilyDataSource(address string, serverID int, family string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var addresses []gona.ServerIPAddress
		var err error
		if family == "ipv4" {
			addresses, err = testAccClients().V2.GetServerIPv4(serverID)
		} else {
			addresses, err = testAccClients().V2.GetServerIPv6(serverID)
		}
		if err != nil {
			return err
		}
		return testAccCheckDataSourceListCount(s, address, family, len(addresses))
	}
}

func testAccCheckCloudRoutingMeshesDataSource(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		meshes, err := testAccClients().V3.ListMagicMeshes()
		if err != nil {
			return err
		}
		return testAccCheckDataSourceListCount(s, address, "meshes", len(meshes))
	}
}

func testAccCheckCloudRoutingRoutersDataSource(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		routers, err := testAccClients().V3.ListRouters()
		if err != nil {
			return err
		}
		return testAccCheckDataSourceListCount(s, address, "routers", len(routers))
	}
}

func testAccCheckCloudRoutingRouterInterfacesDataSource(address string, routerID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		raw, err := testAccClients().V3.ListRouterConfigInterfaces(routerID)
		if err != nil {
			return err
		}
		want, err := compactSurfaceJSON(raw)
		if err != nil {
			return err
		}
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["interfaces"]; got != want {
			return fmt.Errorf("%s interfaces does not match API", address)
		}
		return nil
	}
}

func testAccCheckCloudRoutingRouterVRFsDataSource(address string, routerID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vrfs, err := testAccClients().V3.ListRouterVRFs(routerID)
		if err != nil {
			return err
		}
		return testAccCheckDataSourceListCount(s, address, "vrfs", len(vrfs))
	}
}

func testAccCheckCloudRoutingRouterVRFListDataSource(address, listAttr string, routerID, vrfID int, read func(*ProviderClients) (int, error)) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		count, err := read(testAccClients())
		if err != nil {
			return err
		}
		return testAccCheckDataSourceListCount(s, address, listAttr, count)
	}
}

func testAccCheckCloudRoutingOverviewDataSource(address string, routerID, vrfID int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		raw, err := testAccClients().V3.GetRouterRoutingOverview(routerID, vrfID)
		if err != nil {
			return err
		}
		want, err := compactSurfaceJSON(raw)
		if err != nil {
			return err
		}
		rs, err := testAccResource(s, address)
		if err != nil {
			return err
		}
		if got := rs.Primary.Attributes["overview"]; got != want {
			return fmt.Errorf("%s overview does not match API", address)
		}
		return nil
	}
}

func testAccCheckAttrs(attrs map[string]string, prefix string, checks map[string]string) error {
	for key, want := range checks {
		if got := attrs[prefix+key]; got != want {
			return fmt.Errorf("%s%s = %q, want API value %q", prefix, key, got, want)
		}
	}
	return nil
}
