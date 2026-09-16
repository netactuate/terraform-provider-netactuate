resource "netactuate_router_vrf_dhcp" "example" {
  enabled      = true
  interface_id = 12345
  router_id    = 12345
  subnet       = "192.0.2.0/24"
  vrf_id       = 12345
}
