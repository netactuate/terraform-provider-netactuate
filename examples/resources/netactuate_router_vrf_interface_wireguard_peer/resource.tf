resource "netactuate_router_vrf_interface_wireguard_peer" "example" {
  allowed_ips {
    network = "192.0.2.0/24"
  }
  interface_id = 12345
  router_id    = 12345
  vrf_id       = 12345
}
