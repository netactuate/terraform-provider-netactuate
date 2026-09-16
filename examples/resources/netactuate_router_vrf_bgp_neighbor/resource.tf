resource "netactuate_router_vrf_bgp_neighbor" "example" {
  address    = "192.0.2.10"
  remote_asn = 64512
  router_id  = 12345
  vrf_id     = 12345
}
