resource "netactuate_network_loadbalancer_group" "example" {
  algorithm = "round_robin"
  backend {
    internal_address = "192.0.2.10"
    name             = "example"
  }
  health_check {
    delay    = 80
    enabled  = true
    interval = 80
    method   = "tcp"
    retries  = 1
    timeout  = 80
  }
  ip_version              = 4
  match_address           = "192.0.2.10"
  name                    = "example"
  network_loadbalancer_id = 12345
  rule {
    port_internal = 80
    port_match    = 80
    protocol      = "TCP"
  }
}
