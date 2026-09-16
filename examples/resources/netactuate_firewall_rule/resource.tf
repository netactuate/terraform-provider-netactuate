resource "netactuate_firewall_rule" "example" {
  action          = "ACCEPT"
  enabled         = true
  firewall_set_id = 12345
  ip_version      = "IPv4"
}
