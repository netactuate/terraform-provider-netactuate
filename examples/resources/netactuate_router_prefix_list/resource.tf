resource "netactuate_router_prefix_list" "example" {
  ip_version = 4
  name       = "example"
  router_id  = 12345
  rule {
    action = "ACCEPT"
    prefix = "192.0.2.0/24"
  }
}
