resource "netactuate_security_group_rule" "example" {
  security_group_id = 12345
  ip_version = 4
  protocol = "tcp"
  port_start = 443
  port_end = 443
  network = "192.0.2.0/24"
}
