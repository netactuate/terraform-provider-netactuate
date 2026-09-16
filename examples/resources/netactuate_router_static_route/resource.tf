resource "netactuate_router_static_route" "example" {
  network   = "192.0.2.0/24"
  router_id = 12345
  vrf_id    = 12345
}
