resource "netactuate_http_loadbalancer_group" "example" {
  algorithm = "round_robin"
  backend {
    internal_address = "192.0.2.10"
    name             = "example"
  }
  http_loadbalancer_id = 12345
  internal_port        = 80
  match_address        = "192.0.2.10"
  match_ports          = "80"
  name                 = "example"
  rule {
    match_domain = "example.com"
    match_path   = "/"
  }
}
