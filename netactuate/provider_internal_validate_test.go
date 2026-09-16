package netactuate

import "testing"

// TestProviderInternalValidate runs the SDK's own schema validation over the whole provider.
//
// go build, go vet and a compile-only test pass all succeed on an invalid schema, so neither
// catches defects of this kind:
//
//   - A data source declared top level "id" as TypeInt. Terraform reserves top level id
//     as a string, so every read panicked the provider.
//   - A computed-only nested block set MaxItems, which InternalValidate rejects because
//     there is nothing for the practitioner to configure on a computed-only field.
//
// Both are caught here in milliseconds, with no credentials and no network, so a schema mistake
// now fails at `go test` instead of after a live run has built infrastructure.
func TestProviderInternalValidate(t *testing.T) {
	p := Provider()
	for _, name := range []string{
		"netactuate_access_control_subnet",
		"netactuate_bgp_group",
		"netactuate_bgp_group_firewall_set",
		"netactuate_bgp_prefix_purchase",
		"netactuate_dedicated_server_buy",
		"netactuate_dedicated_server_buy_build",
		"netactuate_dedicated_server_deployment",
		"netactuate_dedicated_server_ipv4_reverse",
		"netactuate_dedicated_server_power_status",
		"netactuate_oidc_client_bare_metal_allow_list",
		"netactuate_router_ipsec",
		"netactuate_router_ntp",
		"netactuate_router_prefix_list",
		"netactuate_router_routing_view",
		"netactuate_router_static_route",
		"netactuate_router_vrf",
		"netactuate_router_vrf_bgp",
		"netactuate_router_vrf_bgp_neighbor",
		"netactuate_router_vrf_dhcp",
		"netactuate_router_vrf_dnat_rule",
		"netactuate_router_vrf_interface",
		"netactuate_router_vrf_interface_wireguard_peer",
		"netactuate_router_vrf_ipsec_peer",
		"netactuate_router_vrf_snat_rule",
		"netactuate_router_vrf_tunnel",
		"netactuate_vpc_backend_template_backends",
		"netactuate_vpc_gateway_standby",
		"netactuate_vpc_nameservers",
		"netactuate_firewall_set_detach_all_vms",
		"netactuate_firewall_set_rule_order",
		"netactuate_firewall_set_vm_detach",
		"netactuate_nke_access_urls",
		"netactuate_nke_worker_node",
		"netactuate_usage_contract",
		"netactuate_cloud_ipv4_reverse_dns",
		"netactuate_cloud_ipv6_reverse_dns",
		"netactuate_cloud_floating_ipv4",
		"netactuate_cloud_floating_ipv4_vm_grant",
		"netactuate_server_options",
		"netactuate_cloud_firewall_set_binding",
	} {
		if p.ResourcesMap[name] == nil {
			t.Fatalf("provider is missing resource schema %s", name)
		}
	}
	for _, name := range []string{
		"netactuate_access_control_subnets",
		"netactuate_access_control_subnet",
		"netactuate_account_agreements",
		"netactuate_bgp_asn",
		"netactuate_bgp_asns",
		"netactuate_bgp_dashboard",
		"netactuate_bgp_groups",
		"netactuate_bgp_prefix",
		"netactuate_bgp_prefixes",
		"netactuate_bgp_summary",
		"netactuate_cloud_base_images",
		"netactuate_cloud_deploy_sizes",
		"netactuate_cloud_extras",
		"netactuate_cloud_floating_ipv4_vms",
		"netactuate_cloud_iplimits",
		"netactuate_cloud_kernels",
		"netactuate_cloud_location",
		"netactuate_cloud_location_vlans",
		"netactuate_cloud_networking_locations",
		"netactuate_cloud_plan_id",
		"netactuate_cloud_pool",
		"netactuate_cloud_private_images",
		"netactuate_cloud_routing_meshes",
		"netactuate_cloud_routing_router_interfaces",
		"netactuate_cloud_routing_router_routing_overview",
		"netactuate_cloud_routing_router_vrf_bgp_neighbors",
		"netactuate_cloud_routing_router_vrf_interfaces",
		"netactuate_cloud_routing_router_vrf_tunnels",
		"netactuate_cloud_routing_router_vrfs",
		"netactuate_cloud_routing_routers",
		"netactuate_cloud_scaling_options",
		"netactuate_cloud_storage_locations",
		"netactuate_cloud_vlan",
		"netactuate_colocation_package",
		"netactuate_colocation_services",
		"netactuate_colocation_service",
		"netactuate_ddos_rule",
		"netactuate_dedicated_device_os_profiles",
		"netactuate_dedicated_devices",
		"netactuate_dedicated_locations",
		"netactuate_dedicated_plans",
		"netactuate_dedicated_servers",
		"netactuate_http_loadbalancer_groups",
		"netactuate_images_provisioning_jobs_count",
		"netactuate_firewall_external_ipsets",
		"netactuate_firewall_manage_enabled",
		"netactuate_firewall_set_available_vms",
		"netactuate_firewall_set_related_vms",
		"netactuate_iptransit_ips",
		"netactuate_iptransit_ports",
		"netactuate_iptransit_services",
		"netactuate_iptransit_service",
		"netactuate_location",
		"netactuate_oidc_client_bare_metal_servers",
		"netactuate_platform_change_log_entry",
		"netactuate_platform_datacenters",
		"netactuate_platform_looking_glass",
		"netactuate_platform_looking_glass_init",
		"netactuate_platform_maintenance_info",
		"netactuate_secret_values",
		"netactuate_services",
		"netactuate_server_bgp_sessions_raw",
		"netactuate_server_deployment_info",
		"netactuate_server_ipv4",
		"netactuate_server_ipv6",
		"netactuate_server_networkips",
		"netactuate_server_summary",
		"netactuate_server_vnc_status",
		"netactuate_ssl_certificates",
		"netactuate_storage_block_namespaces",
		"netactuate_storage_buckets",
		"netactuate_storage_object_stores",
		"netactuate_storage_types",
		"netactuate_tag_logs",
		"netactuate_tag_resources",
		"netactuate_transit_package",
		"netactuate_transport_ports",
		"netactuate_transport_services",
		"netactuate_transport_service",
		"netactuate_current_server",
		"netactuate_unprovisioned_packages",
		"netactuate_vpc_ip_reservations",
		"netactuate_vpc_locations",
		"netactuate_vpc_ssh",
		"netactuate_vpcs",
		"netactuate_virtual_server_contract",
	} {
		if p.DataSourcesMap[name] == nil {
			t.Fatalf("provider is missing data source schema %s", name)
		}
	}
	if err := p.InternalValidate(); err != nil {
		t.Fatalf("provider schema failed the SDK's own validation: %v", err)
	}
}
