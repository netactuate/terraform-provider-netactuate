resource "netactuate_router_vrf_interface" "example" {
  name      = "example"
  router_id = 12345
  type      = "dummy"
  vrf_id    = 12345
}
