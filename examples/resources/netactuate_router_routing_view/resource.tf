resource "netactuate_router_routing_view" "example" {
  router_id = 12345
  view {
    ip_version = 4
    name       = "example"
  }
  vrf_id = 12345
}
