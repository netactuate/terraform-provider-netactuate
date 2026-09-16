resource "netactuate_router_ntp" "example" {
  enabled   = true
  router_id = 12345
  upstreams {
    domain = "example"
  }
}
