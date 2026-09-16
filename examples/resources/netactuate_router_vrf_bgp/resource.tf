resource "netactuate_router_vrf_bgp" "example" {
  local_asn = 64512
  networks {
    subnet = "192.0.2.0/24"
  }
  router_id = 12345
  vrf_id    = 12345
}
