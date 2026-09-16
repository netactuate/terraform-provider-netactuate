resource "netactuate_router_vrf_ipsec_peer" "example" {
  name         = "example"
  psk_secret   = "change-me-example-value"
  remote_id    = "example"
  router_id    = 12345
  vrf_id       = 12345
  overlay_ipv4 = "192.0.2.1/30"
  peer_address = "192.0.2.2"
}
