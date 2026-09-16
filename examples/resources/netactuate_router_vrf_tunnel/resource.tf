resource "netactuate_router_vrf_tunnel" "example" {
  endpoint_address_remote = "192.0.2.10"
  ip_key                  = 1000
  mtu                     = 1500
  name                    = "example"
  router_id               = 12345
  vrf_id                  = 12345
  ipv4_cidr               = "192.0.2.1/30"
}
