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


# The cloud router is in a magic mesh and only allows the default VRF configuration.
#
# resource "netactuate_router_vrf" "example" {
#   router_id = netactuate_router.example.id
#   name = "Example Terraform Router VPF"
#   description = "Example Terraform Router VPF Description"
# }

resource "netactuate_router_vrf_interface" "example" {
  router_id = netactuate_router.example.id
  vrf_id    = netactuate_router.example.default_vrf_id
  type = "dummy"
  name = "Example Terraform Router VRF Interface"
  description = "Example Terraform Router VRF Interface Description"
  ipv4_cidr = "192.168.0.1/24"
}

resource "netactuate_router_vrf_bgp" "example" {
  router_id = netactuate_router.example.id
  vrf_id = netactuate_router.example.default_vrf_id
  local_asn = "65002"
  networks {
    subnet = "168.192.0.0/16"
  }
}

resource "netactuate_router_vrf_bgp_neighbor" "example" {
  router_id = netactuate_router.example.id
  vrf_id = netactuate_router.example.default_vrf_id
  name = "Example Terraform Router VRF BGP Neighbor"
  description = "Example Terraform Router VRF BGP Neighbor Description"
  address = "192.168.1.1"
  ipv4_enabled = true
  ipv6_enabled = true
  remote_asn = 65001
}

resource "netactuate_router_vrf_snat_rule" "example" {
  router_id = netactuate_router.example.id
  vrf_id = netactuate_router.example.default_vrf_id
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
  vrf_id = netactuate_router.example.default_vrf_id
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

resource "netactuate_router_vrf_tunnel" "example" {
  router_id = netactuate_router.example.id
  vrf_id = netactuate_router.example.default_vrf_id
  ip_key = 676769
  name = "Example Terraform Router VRF Tunnel"
  description = "Example Terraform Router VRF Tunnel Description"
  mtu = 16000
  ipv4_cidr = "192.168.0.1/24"
  endpoint_address_remote = "192.168.1.1"
}

resource "netactuate_router_prefix_list" "example" {
  router_id  = netactuate_router.example.id
  name       = "Example Prefix List"
  ip_version = 4
  description = "Allow internal networks"

  rule {
    action = "permit"
    prefix = "10.0.0.0/8"
  }

  rule {
    action = "permit"
    prefix = "172.16.0.0/12"
  }

  rule {
    action = "deny"
    prefix = "0.0.0.0/0"
  }
}

resource "netactuate_router_prefix_list" "example_ipv6" {
  router_id   = netactuate_router.example.id
  name        = "Example Prefix List IPv6"
  ip_version  = 6
  description = "Allow internal IPv6 networks"

  rule {
    action = "permit"
    prefix = "fd00::/8"
  }

  rule {
    action = "deny"
    prefix = "::/0"
  }
}

resource "netactuate_router_static_route" "example" {
  router_id    = netactuate_router.example.id
  vrf_id       = netactuate_router.example.default_vrf_id
  network      = "10.0.0.0/24"
  # To route via next-hop IP, use:
  # next_hop    = "192.168.0.254"
  # distance    = 10
  # To route via interface, use:
  interface_id = netactuate_router_vrf_interface.example.interface_id
  description  = "Example static route via interface"
}

resource "netactuate_magic_mesh" "example" {
  name        = "Example Magic Mesh updated"
  description = "Mesh connecting multiple cloud routers"
}

resource "netactuate_magic_mesh_router" "example" {
  mesh_id   = netactuate_magic_mesh.example.id
  router_id = netactuate_router.example.router_id
}

resource "netactuate_router_vrf_dhcp" "example" {
  router_id = netactuate_router.example.id
  vrf_id = netactuate_router.example.default_vrf_id
  enabled = true
  interface_id = netactuate_router_vrf_interface.example.id
  subnet = "192.168.0.0/24"
  lease_timeout = 86400
  do_ping_check = true
  default_router_address = "198.51.100.42"
  client_domain_name = "test-26022026.netactuate.com"
  range {
    first_address = "192.168.0.1"
    last_address = "192.168.0.2"
  }
  domain_name_servers {
    address = "198.51.100.42"
  }
  ntp_servers {
    address = "198.51.100.42"
  }
  static_routes {
    network = "192.168.0.0/24"
    next_hop = "198.51.100.42"
  }
}