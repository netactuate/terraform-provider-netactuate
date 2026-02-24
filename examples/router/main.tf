provider "netactuate" {
  api_key = "NETACTUATE_API_KEY"
  api_url    = "VAPI2_URL"
  api_url_v3 = "VAPI3_URL"
}

resource "netactuate_router" "example" {
  name = "Example Terraform Router"
  description = "Example Terraform Router Description"
  location    = "DEVRDU - Raleigh, NC"
  package_id = 857
}

resource "netactuate_router_vrf" "example" {
  router_id = netactuate_router.example.id
  name = "Example Terraform Router VPF"
  description = "Example Terraform Router VPF Description"
}

resource "netactuate_router_vrf_interface" "example" {
  router_id = netactuate_router.example.id
  vrf_id = netactuate_router_vrf.example.id
  type = "dummy"
  name = "Example Terraform Router VRF Interface"
  description = "Example Terraform Router VRF Interface Description"
  ipv4_cidr = "192.168.0.1/24"
}

resource "netactuate_router_vrf_bgp" "example" {
  router_id = netactuate_router.example.id
  vrf_id = netactuate_router_vrf.example.id
  local_asn = "65002"
  networks {
    subnet = "168.192.0.0/16"
  }
}

resource "netactuate_router_vrf_bgp_neighbor" "example" {
  router_id = netactuate_router.example.id
  vrf_id = netactuate_router_vrf.example.id
  name = "Example Terraform Router VRF BGP Neighbor"
  description = "Example Terraform Router VRF BGP Neighbor Description"
  address = "192.168.1.1"
  ipv4_enabled = true
  ipv6_enabled = true
  remote_asn = 65001
}

resource "netactuate_router_vrf_snat_rule" "example" {
  router_id = netactuate_router.example.id
  vrf_id = netactuate_router_vrf.example.id
  ip_version = 4
  protocol = "TCP"
  description = "Example Terraform Router VRF SNAT Rule Description"
  match_interface_id = netactuate_router_vrf_interface.example.id
  match_network = "192.168.1.0/24"
  match_port_start = 30000
  match_port_end = 31000
  translation_network = "192.168.2.0/24"
  translation_port_start = 30000
  translation_port_end = 31000
  priority_location = "end"
}

resource "netactuate_router_vrf_dnat_rule" "example" {
  router_id = netactuate_router.example.id
  vrf_id = netactuate_router_vrf.example.id
  ip_version = 4
  protocol = "TCP"
  description = "Example Terraform Router VRF DNAT Rule Description"
  match_interface_id = netactuate_router_vrf_interface.example.id
  match_network = "192.168.1.0/24"
  match_port_start = 30000
  match_port_end = 31000
  translation_network = "192.168.2.0/24"
  translation_port_start = 30000
  translation_port_end = 31000
  priority_location = "end"
}