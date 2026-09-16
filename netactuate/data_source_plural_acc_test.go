//go:build acctest

package netactuate

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetactuateServersDataSource_listsServerShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_servers" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_servers.test", "servers", map[string]testAccPluralFieldCheck{
					"mbpkgid": positiveIntField,
					"fqdn":    nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateLocationsDataSource_listsLocationShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_locations" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_locations.test", "locations", map[string]testAccPluralFieldCheck{
					"id":   positiveIntField,
					"name": nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuatePlansDataSource_listsPlanShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_plans" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_plans.test", "plans", map[string]testAccPluralFieldCheck{
					"plan_id": positiveIntField,
					"plan":    nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateSizesDataSource_listsSizeShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_sizes" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_sizes.test", "sizes", map[string]testAccPluralFieldCheck{
					"plan_id": positiveIntField,
					"plan":    nonEmptyStringField,
					"cpu":     positiveIntField,
					"port":    nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateVLANsDataSource_listsVLANShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_vlans" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_vlans.test", "vlans", map[string]testAccPluralFieldCheck{
					"id":                           positiveIntField,
					"mbid":                         positiveIntField,
					"private":                      integerField,
					"allow_sriov":                  integerField,
					"display_name":                 nonEmptyStringField,
					"provisioned_locations.#":      nonNegativeIntField,
					"provisioned_locations.0.name": nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateCloudFloatingIPsDataSource_listsFloatingIPv4Shape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_cloud_floating_ips" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_cloud_floating_ips.test", "floating_ips", map[string]testAccPluralFieldCheck{
					"floating_ipv4_id": positiveIntField,
					"assigned_on":      nonEmptyStringField,
					"address":          nonEmptyStringField,
					"vlan_id":          positiveIntField,
					"location.0.id":    positiveIntField,
					"location.0.name":  nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateServerNICsDataSource_readsEmptyObservedList(t *testing.T) {
	mbpkgID := testAccEnvInt(t, "NETACTUATE_ACC_SERVER_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(fmt.Sprintf(`
data "netactuate_server_nics" "test" {
  mbpkgid = %d
}
`, mbpkgID)),
				// Empty is a valid state on the test account, not a failed read.
				Check: testAccCheckAttributePresent("data.netactuate_server_nics.test", "nics.#"),
			},
		},
	})
}

func TestAccNetactuateAccountLimitsDataSource_listsLimitShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_account_limits" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_account_limits.test", "limits", map[string]testAccPluralFieldCheck{
					"resource":        nonEmptyStringField,
					"used":            integerField,
					"max":             positiveIntField,
					"allowed_plans.#": nonNegativeIntField,
				}),
			},
		},
	})
}

func TestAccNetactuateContractUsageDataSource_readsContractShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_contract_usage" "test" {}
`),
				// contract_usage_id, not id. Terraform's own id is the fixed string this data
				// source sets, so asserting on it would prove only that the provider ran.
				Check: testAccCheckAttributePresent("data.netactuate_contract_usage.test", "contract_usage_id"),
			},
		},
	})
}

func TestAccNetactuateMetricNamesDataSource_listsMetricShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_metric_names" "test" {}
`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluralListShape("data.netactuate_metric_names.test", "metrics", map[string]testAccPluralFieldCheck{
						"metric":  nonInternalMetricField,
						"service": nonEmptyStringField,
					}),
					testAccCheckNoInternalMetricNames("data.netactuate_metric_names.test"),
				),
			},
		},
	})
}

func TestAccNetactuateStatisticsDataSource_readsMetricShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_statistics" "test" {
  metrics = ["vmStatus"]
}
`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluralListShape("data.netactuate_statistics.test", "results", map[string]testAccPluralFieldCheck{
						"metric": vmStatusField,
					}),
				),
			},
		},
	})
}

func TestAccNetactuateNetworkingStatisticsDataSource_readsMetricShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_networking_statistics" "test" {
  metrics = ["vmStatus"]
}
`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluralListShape("data.netactuate_networking_statistics.test", "results", map[string]testAccPluralFieldCheck{
						"metric": vmStatusField,
					}),
				),
			},
		},
	})
}

func TestAccNetactuateAnycastStatisticsDataSource_readsMetricShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_anycast_statistics" "test" {
  metrics = ["vmStatus"]
}
`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluralListShape("data.netactuate_anycast_statistics.test", "results", map[string]testAccPluralFieldCheck{
						"metric": vmStatusField,
					}),
				),
			},
		},
	})
}

func TestAccNetactuatePlatformStatusDataSource_listsStatusShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_platform_status" "test" {}
`),
				Check: testAccCheckPlatformStatusShape("data.netactuate_platform_status.test"),
			},
		},
	})
}

func TestAccNetactuatePlatformChangeLogDataSource_listsChangeLogShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_platform_change_log" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_platform_change_log.test", "entries", map[string]testAccPluralFieldCheck{
					"change_log_id":     nonEmptyStringField,
					"title":             nonEmptyStringField,
					"short_description": stringField,
					"status":            nonEmptyStringField,
					"entry_json":        nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuatePlatformIncidentsDataSource_readsActiveShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_platform_incidents" "test" {
  location = "3"
}
`),
				// Empty is a valid state on the test account, not a failed read.
				Check: testAccCheckAttributePresent("data.netactuate_platform_incidents.test", "active.#"),
			},
		},
	})
}

func TestAccNetactuatePlatformIncidentHistoryDataSource_listsHistoryShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_platform_incident_history" "test" {
  location = "3"
}
`),
				Check: testAccCheckPluralListShape("data.netactuate_platform_incident_history.test", "historic", platformHistoryFieldChecks()),
			},
		},
	})
}

func TestAccNetactuatePlatformMaintenanceDataSource_readsActiveShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_platform_maintenance" "test" {
  location = "3"
}
`),
				// Empty is a valid state on the test account, not a failed read.
				Check: testAccCheckAttributePresent("data.netactuate_platform_maintenance.test", "active.#"),
			},
		},
	})
}

func TestAccNetactuatePlatformMaintenanceHistoryDataSource_listsHistoryShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_platform_maintenance_history" "test" {
  location = "3"
}
`),
				Check: testAccCheckPluralListShape("data.netactuate_platform_maintenance_history.test", "historic", platformHistoryFieldChecks()),
			},
		},
	})
}

func TestAccNetactuateDDoSAttacksDataSource_listsAttackShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_ddos_attacks" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_ddos_attacks.test", "attacks", ddosAttackFieldChecks()),
			},
		},
	})
}

func TestAccNetactuateDDoSActiveAttacksDataSource_readsEmptyObservedList(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_ddos_active_attacks" "test" {}
`),
				// Empty is a valid state on the test account, not a failed read.
				Check: testAccCheckAttributePresent("data.netactuate_ddos_active_attacks.test", "attacks.#"),
			},
		},
	})
}

func TestAccNetactuateDDoSDashboardDataSource_readsDashboardShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_ddos_dashboard" "test" {
  period        = 86400
  include_ended = true
  limit         = 10
}
`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTypedAttribute("data.netactuate_ddos_dashboard.test", "total_attacks", nonNegativeIntField),
					testAccCheckTypedAttribute("data.netactuate_ddos_dashboard.test", "active_rules", nonNegativeIntField),
					testAccCheckTypedAttribute("data.netactuate_ddos_dashboard.test", "longest_attack_seconds", nonNegativeIntField),
					testAccCheckTypedAttribute("data.netactuate_ddos_dashboard.test", "period", positiveIntField),
					testAccCheckAttributePresent("data.netactuate_ddos_dashboard.test", "top_attacks.#"),
				),
			},
		},
	})
}

func TestAccNetactuateDDoSRulesDataSource_listsRuleShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_ddos_rules" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_ddos_rules.test", "rules", map[string]testAccPluralFieldCheck{
					"rule_id":                    positiveIntField,
					"rule_name":                  nonEmptyStringField,
					"description":                stringField,
					"prefixes.#":                 positiveIntField,
					"prefixes.0.prefix_id":       positiveIntField,
					"prefixes.0.prefix":          nonEmptyStringField,
					"prefixes.0.prefix_type":     nonEmptyStringField,
					"prefixes.0.description":     stringField,
					"prefixes.0.allowed_pps":     nonNegativeIntField,
					"rules.#":                    positiveIntField,
					"rules.0.name":               nonEmptyStringField,
					"rules.0.action_type":        nonEmptyStringField,
					"rules.0.run_order":          nonNegativeIntField,
					"rules.0.action_name":        nonEmptyStringField,
					"rules.0.action_description": stringField,
				}),
			},
		},
	})
}

func TestAccNetactuatePackagesDataSource_listsPackageShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_packages" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_packages.test", "packages", map[string]testAccPluralFieldCheck{
					"mbpkgid": positiveIntField,
					"name":    nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateBillingPackagesDataSource_listsBillingPackageShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_billing_packages" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_billing_packages.test", "billing_packages", map[string]testAccPluralFieldCheck{
					"billing_package_id": positiveIntField,
					"name":               nonEmptyStringField,
					"amount":             stringField,
					"nextduedate":        stringField,
				}),
			},
		},
	})
}

func TestAccNetactuateColocationPackagesDataSource_listsPackageShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_colocation_packages" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_colocation_packages.test", "packages", map[string]testAccPluralFieldCheck{
					"mbpkgid": positiveIntField,
				}),
			},
		},
	})
}

func TestAccNetactuateColocationPackageDataSource_readsPackageShape(t *testing.T) {
	mbpkgID := testAccEnvInt(t, "NETACTUATE_ACC_COLOCATION_PACKAGE_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_COLOCATION_PACKAGE_ID") },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(fmt.Sprintf(`
data "netactuate_colocation_package" "test" {
  mbpkgid = %d
}
`, mbpkgID)),
				Check: testAccCheckPluralListShape("data.netactuate_colocation_package.test", "packages", map[string]testAccPluralFieldCheck{
					"mbpkgid": positiveIntField,
				}),
			},
		},
	})
}

func TestAccNetactuateColocationServicesDataSource_listsServiceShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_colocation_services" "test" {}
`),
				Check: testAccCheckAttributePresent("data.netactuate_colocation_services.test", "services.#"),
			},
		},
	})
}

func TestAccNetactuateColocationServiceDataSource_readsServiceShape(t *testing.T) {
	serviceID := testAccEnvInt(t, "NETACTUATE_ACC_COLOCATION_SERVICE_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_COLOCATION_SERVICE_ID") },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(fmt.Sprintf(`
data "netactuate_colocation_service" "test" {
  service_id = %d
}
`, serviceID)),
				Check: testAccCheckPluralListShape("data.netactuate_colocation_service.test", "service", map[string]testAccPluralFieldCheck{
					"service_id": positiveIntField,
					"name":       nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateTransitPackagesDataSource_listsPackageShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_transit_packages" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_transit_packages.test", "packages", map[string]testAccPluralFieldCheck{
					"mbpkgid": positiveIntField,
				}),
			},
		},
	})
}

func TestAccNetactuateCloudPoolsDataSource_listsCloudPoolShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_cloud_pools" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_cloud_pools.test", "cloud_pools", map[string]testAccPluralFieldCheck{
					"cloud_pool_id":       positiveIntField,
					"name":                nonEmptyStringField,
					"hard_capabilities.#": positiveIntField,
					"soft_capabilities.#": positiveIntField,
					"default_ram_price":   stringField,
					"default_cpu_price":   stringField,
					"default_disk_price":  stringField,
				}),
			},
		},
	})
}

func TestAccNetactuateCloudCapacityDataSource_listsCapacityShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_cloud_capacity" "test" {
  # These come from /cloud/pools and the acceptance location, not from generated fixtures.
  cloud_pool_id = 1
  location_id   = 3
}
`),
				Check: testAccCheckPluralListShape("data.netactuate_cloud_capacity.test", "capacity", map[string]testAccPluralFieldCheck{
					"package_id":    positiveIntField,
					"package_name":  nonEmptyStringField,
					"package_cpu":   positiveIntField,
					"package_ram":   positiveIntField,
					"package_disk":  positiveIntField,
					"monthly_price": floatField,
					"available":     nonNegativeIntField,
				}),
			},
		},
	})
}

// The account holds no external IP sets, and the endpoint returns an empty list rather than an
// error. Emptiness is a valid state, so this asserts the read succeeded and the shape holds for
// any set that does come back.
func TestAccNetactuateFirewallExternalIPSetsDataSource_listsIPSetShapeWhenPresent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_firewall_external_ipsets" "test" {}
`),
				Check: testAccCheckPluralListShapeWhenPresent("data.netactuate_firewall_external_ipsets.test", "ipsets", map[string]testAccPluralFieldCheck{
					"external_ipset_id": positiveIntField,
					"name":              nonEmptyStringField,
					"raw_json":          nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateFirewallManageEnabledDataSource_readsEnabledShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_firewall_manage_enabled" "test" {}
`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTypedAttribute("data.netactuate_firewall_manage_enabled.test", "enabled", boolField),
					testAccCheckTypedAttribute("data.netactuate_firewall_manage_enabled.test", "raw_json", nonEmptyStringField),
				),
			},
		},
	})
}

func TestAccNetactuateFirewallSetAvailableVMsDataSource_readsAvailableList(t *testing.T) {
	name := testAccName("fw-available-vms")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFirewallSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallSetAvailableVMsDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_firewall_set", "netactuate_firewall_set.test"),
					testAccCheckAttributePresent("data.netactuate_firewall_set_available_vms.test", "vms.#"),
				),
			},
		},
	})
}

func TestAccNetactuateFirewallSetRelatedVMsDataSource_readsRelatedList(t *testing.T) {
	serverID := testAccEnvInt(t, "NETACTUATE_ACC_SERVER_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_SERVER_ID") },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(fmt.Sprintf(`
data "netactuate_firewall_set_related_vms" "test" {
  mbpkgid                     = %d
  disable_interface_id_filter = true
}
`, serverID)),
				Check: testAccCheckAttributePresent("data.netactuate_firewall_set_related_vms.test", "vms.#"),
			},
		},
	})
}

func TestAccNetactuateDedicatedCapacityDataSource_listsDedicatedCapacityShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_dedicated_capacity" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_dedicated_capacity.test", "dedicated_capacity", map[string]testAccPluralFieldCheck{
					"device_id":       positiveIntField,
					"location_id":     positiveIntField,
					"looking_glass":   nonEmptyStringField,
					"name":            nonEmptyStringField,
					"pub_description": stringField,
				}),
			},
		},
	})
}

func TestAccNetactuateDedicatedDevicesDataSource_listsDeviceShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_dedicated_devices" "test" {
  per_page = 10
}
`),
				Check: testAccCheckPluralListShape("data.netactuate_dedicated_devices.test", "devices", map[string]testAccPluralFieldCheck{
					"device_id": positiveIntField,
					"raw_json":  nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateDedicatedLocationsDataSource_listsLocationShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_dedicated_locations" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_dedicated_locations.test", "locations", map[string]testAccPluralFieldCheck{
					"short_name":      nonEmptyStringField,
					"pub_description": nonEmptyStringField,
					"location_id":     positiveIntField,
				}),
			},
		},
	})
}

func TestAccNetactuateDedicatedDeviceOSProfilesDataSource_listsProfileShape(t *testing.T) {
	deviceID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_DEVICE_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DEDICATED_DEVICE_ID") },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(fmt.Sprintf(`
data "netactuate_dedicated_device_os_profiles" "test" {
  device_id  = %d
  is_buyable = true
}
`, deviceID)),
				Check: testAccCheckPluralListShape("data.netactuate_dedicated_device_os_profiles.test", "os_profiles", map[string]testAccPluralFieldCheck{
					"os_id": positiveIntField,
					"name":  nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateDedicatedPlansDataSource_listsPlanShape(t *testing.T) {
	locationID := testAccEnvInt(t, "NETACTUATE_ACC_DEDICATED_LOCATION_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_DEDICATED_LOCATION_ID") },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(fmt.Sprintf(`
data "netactuate_dedicated_plans" "test" {
  location_id = %d
}
`, locationID)),
				Check: testAccCheckPluralListShape("data.netactuate_dedicated_plans.test", "plans", map[string]testAccPluralFieldCheck{
					"plan_id":  positiveIntField,
					"raw_json": nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateDedicatedServersDataSource_listsServerShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_dedicated_servers" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_dedicated_servers.test", "servers", map[string]testAccPluralFieldCheck{
					"dedicated_server_id": positiveIntField,
					"mbpkgid":             positiveIntField,
					"hostname":            nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateOSImagesDataSource_listsOSImageShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_os_images" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_os_images.test", "os_images", map[string]testAccPluralFieldCheck{
					"id": positiveIntField,
					"os": nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateTagsDataSource_listsTagShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_tags" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_tags.test", "tags", map[string]testAccPluralFieldCheck{
					"id":   positiveIntField,
					"name": nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateMyImagesDataSource_listsImageShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_my_images" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_my_images.test", "images", map[string]testAccPluralFieldCheck{
					"id": positiveIntField,
					"os": nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateSSHKeysDataSource_listsSSHKeyShape(t *testing.T) {
	name := testAccName("sshkeys-list")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSSHKeysDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_sshkey", "netactuate_sshkey.test"),
					testAccCheckPluralListShape("data.netactuate_sshkeys.test", "sshkeys", map[string]testAccPluralFieldCheck{
						"id":      positiveIntField,
						"ssh_key": nonEmptyStringField,
					}),
				),
			},
		},
	})
}

func TestAccNetactuateSecretListsDataSource_listsSecretListShape(t *testing.T) {
	name := testAccName("secret-lists-list")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSecretListValueDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretListsDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_secret_list", "netactuate_secret_list.test"),
					testAccCheckPluralListShape("data.netactuate_secret_lists.test", "secret_lists", map[string]testAccPluralFieldCheck{
						"id":   positiveIntField,
						"name": nonEmptyStringField,
					}),
				),
			},
		},
	})
}

func TestAccNetactuateSecretListValuesDataSource_listsSecretListValueShape(t *testing.T) {
	name := testAccName("secret-list-values-list")
	key := testAccName("secret-key-list")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSecretListValueDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretListValuesDataSourceConfig(name, key),
				Check: resource.ComposeTestCheckFunc(
					testAccTrackFromState("netactuate_secret_list", "netactuate_secret_list.test"),
					testAccTrackFromState("netactuate_secret_list_value", "netactuate_secret_list_value.test"),
					testAccCheckPluralListShape("data.netactuate_secret_list_values.test", "values", map[string]testAccPluralFieldCheck{
						"id":             positiveIntField,
						"secret_list_id": positiveIntField,
						"secret_key":     nonEmptyStringField,
					}),
				),
			},
		},
	})
}

func TestAccNetactuateStorageBlockVolumesDataSource_listsBlockVolumeShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_storage_block_volumes" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_storage_block_volumes.test", "block_volumes", map[string]testAccPluralFieldCheck{
					"metadata.0.block_volume_id": positiveIntField,
					"metadata.0.label":           nonEmptyStringField,
				}),
			},
		},
	})
}

type testAccPluralFieldCheck func(string) error

// testAccCheckPluralListShapeWhenPresent checks the same field shapes as
// testAccCheckPluralListShape, but treats an empty list as a valid read rather than a failure.
// Use it where the account legitimately holds none of the object: an empty list then proves the
// read succeeded, and the shape is still asserted the moment anything comes back.
func testAccCheckPluralListShapeWhenPresent(address, listAttr string, checks map[string]testAccPluralFieldCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		if got, ok := rs.Primary.Attributes[listAttr+".#"]; !ok {
			return fmt.Errorf("%s state has no %s attribute", address, listAttr)
		} else if got == "" || got == "0" {
			return nil
		}
		return testAccCheckPluralListShape(address, listAttr, checks)(s)
	}
}

func testAccCheckPluralListShape(address, listAttr string, checks map[string]testAccPluralFieldCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		got := rs.Primary.Attributes[listAttr+".#"]
		if got == "" || got == "0" {
			return fmt.Errorf("%s state %s is empty", address, listAttr)
		}
		count, err := strconv.Atoi(got)
		if err != nil {
			return fmt.Errorf("%s state %s count is not an integer: %w", address, listAttr, err)
		}
		for i := 0; i < count; i++ {
			itemOK := true
			for field, check := range checks {
				key := fmt.Sprintf("%s.%d.%s", listAttr, i, field)
				value, ok := rs.Primary.Attributes[key]
				if !ok {
					itemOK = false
					break
				}
				if err := check(value); err != nil {
					itemOK = false
					break
				}
			}
			if itemOK {
				return nil
			}
		}
		return fmt.Errorf("%s state %s has no item matching the expected shape", address, listAttr)
	}
}

func testAccCheckAttributePresent(address, attr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		if _, ok := rs.Primary.Attributes[attr]; !ok {
			return fmt.Errorf("%s state missing %s", address, attr)
		}
		return nil
	}
}

func testAccCheckTypedAttribute(address, attr string, check testAccPluralFieldCheck) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		value, ok := rs.Primary.Attributes[attr]
		if !ok {
			return fmt.Errorf("%s state missing %s", address, attr)
		}
		if err := check(value); err != nil {
			return fmt.Errorf("%s state %s is invalid: %w", address, attr, err)
		}
		return nil
	}
}

func testAccCheckNoInternalMetricNames(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		got := rs.Primary.Attributes["metrics.#"]
		if got == "" || got == "0" {
			return fmt.Errorf("%s state metrics is empty", address)
		}
		count, err := strconv.Atoi(got)
		if err != nil {
			return fmt.Errorf("%s state metrics count is not an integer: %w", address, err)
		}
		for i := 0; i < count; i++ {
			metric := rs.Primary.Attributes[fmt.Sprintf("metrics.%d.metric", i)]
			if strings.HasPrefix(metric, "__") {
				return fmt.Errorf("%s state metrics includes internal key %q", address, metric)
			}
		}
		return nil
	}
}

func testAccCheckPlatformStatusShape(address string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("not found: %s", address)
		}
		if err := positiveIntField(rs.Primary.Attributes["services.#"]); err != nil {
			return fmt.Errorf("%s state services is empty or invalid: %w", address, err)
		}
		if err := nonEmptyStringField(rs.Primary.Attributes["services.0.service"]); err != nil {
			return fmt.Errorf("%s state first service is invalid: %w", address, err)
		}
		if err := nonEmptyStringField(rs.Primary.Attributes["services.0.component_id"]); err != nil {
			return fmt.Errorf("%s state first component_id is invalid: %w", address, err)
		}
		if err := positiveIntField(rs.Primary.Attributes["services.0.locations.#"]); err != nil {
			return fmt.Errorf("%s state first service locations is empty or invalid: %w", address, err)
		}
		for _, key := range []string{
			"services.0.locations.0.location",
			"services.0.locations.0.container_id",
			"services.0.locations.0.status",
			"services.0.locations.0.last_updated",
		} {
			if err := nonEmptyStringField(rs.Primary.Attributes[key]); err != nil {
				return fmt.Errorf("%s state %s is invalid: %w", address, key, err)
			}
		}
		return nil
	}
}

func platformHistoryFieldChecks() map[string]testAccPluralFieldCheck {
	return map[string]testAccPluralFieldCheck{
		"event_id":     nonEmptyStringField,
		"type":         nonEmptyStringField,
		"name":         nonEmptyStringField,
		"status":       nonEmptyStringField,
		"start_time":   nonEmptyStringField,
		"end_time":     nonEmptyStringField,
		"components.#": nonNegativeIntField,
		"containers.#": nonNegativeIntField,
	}
}

func ddosAttackFieldChecks() map[string]testAccPluralFieldCheck {
	return map[string]testAccPluralFieldCheck{
		"attack_id":    positiveIntField,
		"date_start":   nonEmptyStringField,
		"date_end":     stringField,
		"status":       integerField,
		"ip":           nonEmptyStringField,
		"prefix":       nonEmptyStringField,
		"direction":    nonEmptyStringField,
		"pps":          nonNegativeIntField,
		"rule_id":      positiveIntField,
		"rule_type":    nonEmptyStringField,
		"ban_duration": nonNegativeIntField,
		"rule_name":    nonEmptyStringField,
	}
}

func positiveIntField(value string) error {
	i, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if i <= 0 {
		return fmt.Errorf("value must be positive")
	}
	return nil
}

func nonNegativeIntField(value string) error {
	i, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if i < 0 {
		return fmt.Errorf("value must be non-negative")
	}
	return nil
}

func integerField(value string) error {
	_, err := strconv.Atoi(value)
	return err
}

func floatField(value string) error {
	_, err := strconv.ParseFloat(value, 64)
	return err
}

func boolField(value string) error {
	if value != "true" && value != "false" {
		return fmt.Errorf("value must be boolean")
	}
	return nil
}

func stringField(value string) error {
	return nil
}

func nonEmptyStringField(value string) error {
	if value == "" {
		return fmt.Errorf("value must not be empty")
	}
	return nil
}

func nonInternalMetricField(value string) error {
	if err := nonEmptyStringField(value); err != nil {
		return err
	}
	if strings.HasPrefix(value, "__") {
		return fmt.Errorf("metric must not begin with double underscores")
	}
	return nil
}

func vmStatusField(value string) error {
	if value != "vmStatus" {
		return fmt.Errorf("metric = %q, want vmStatus", value)
	}
	return nil
}

func testAccPluralDataSourceConfig(body string) string {
	return testAccProviderConfig() + body
}

func testAccSSHKeysDataSourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_sshkey" "test" {
  name = %q
  key  = %q
}

data "netactuate_sshkeys" "test" {
  depends_on = [netactuate_sshkey.test]
}
`, name, testAccSSHPublicKey)
}

func testAccSecretListsDataSourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_secret_list" "test" {
  name = %q
}

data "netactuate_secret_lists" "test" {
  depends_on = [netactuate_secret_list.test]
}
`, name)
}

func testAccSecretListValuesDataSourceConfig(name, key string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_secret_list" "test" {
  name = %q
}

resource "netactuate_secret_list_value" "test" {
  secret_list_id = netactuate_secret_list.test.id
  secret_key     = %q
  secret_value   = %q
}

data "netactuate_secret_list_values" "test" {
  secret_list_id = netactuate_secret_list.test.id
  depends_on     = [netactuate_secret_list_value.test]
}
`, name, key, testAccSecretValueInitial)
}

func testAccFirewallSetAvailableVMsDataSourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "netactuate_firewall_set" "test" {
  name        = %q
  description = %q
  enabled     = true
}

data "netactuate_firewall_set_available_vms" "test" {
  firewall_set_id               = netactuate_firewall_set.test.id
  check_vpc                     = false
  disable_interface_id_filter   = true
  depends_on                    = [netactuate_firewall_set.test]
}
`, name, name)
}

func TestAccNetactuateBootProfilesDataSource_listsBootProfileShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_boot_profiles" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_boot_profiles.test", "boot_profiles", map[string]testAccPluralFieldCheck{
					"boot_profile_id": positiveIntField,
					"name":            nonEmptyStringField,
				}),
			},
		},
	})
}

// The disk list is empty for every server on this account, so this asserts the read succeeded
// and the attribute is present rather than that anything came back. Emptiness is a valid
// state, not a failed read.
func TestAccNetactuateServerDisksDataSource_readsEmptyObservedList(t *testing.T) {
	mbpkgID := testAccEnvInt(t, "NETACTUATE_ACC_SERVER_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t, "NETACTUATE_ACC_SERVER_ID") },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(fmt.Sprintf(`
data "netactuate_server_disks" "test" {
  mbpkgid = %d
}
`, mbpkgID)),
				Check: testAccCheckAttributePresent("data.netactuate_server_disks.test", "disks.#"),
			},
		},
	})
}

func TestAccNetactuateDedicatedOSProfilesDataSource_listsProfileShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_dedicated_os_profiles" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_dedicated_os_profiles.test", "os_profiles", map[string]testAccPluralFieldCheck{
					"os_id": positiveIntField,
					"name":  nonEmptyStringField,
				}),
			},
		},
	})
}

func TestAccNetactuateDedicatedRescueOSDataSource_listsProfileShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_dedicated_rescue_os" "test" {}
`),
				Check: testAccCheckPluralListShape("data.netactuate_dedicated_rescue_os.test", "os_profiles", map[string]testAccPluralFieldCheck{
					"os_id": positiveIntField,
					"name":  nonEmptyStringField,
				}),
			},
		},
	})
}

// os_id 3 is confirmed to return layouts. It comes from netactuate_dedicated_os_profiles rather
// than being a magic number, and an invalid id answers 422 rather than an empty list.
func TestAccNetactuateDedicatedDiskLayoutsDataSource_listsLayoutShape(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPluralDataSourceConfig(`
data "netactuate_dedicated_disk_layouts" "test" {
  os_id = 3
}
`),
				Check: testAccCheckPluralListShape("data.netactuate_dedicated_disk_layouts.test", "disk_layouts", map[string]testAccPluralFieldCheck{
					"layout_id": positiveIntField,
					"name":      nonEmptyStringField,
				}),
			},
		},
	})
}
