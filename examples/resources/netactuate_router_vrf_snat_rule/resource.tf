resource "netactuate_router_vrf_snat_rule" "example" {
  ip_version          = 4
  match_interface_id  = 12345
  match_network       = "192.0.2.0/24"
  protocol            = "TCP"
  router_id           = 12345
  translation_network = "192.0.2.0/24"
  vrf_id              = 12345
}
