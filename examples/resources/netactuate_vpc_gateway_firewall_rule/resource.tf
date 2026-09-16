resource "netactuate_vpc_gateway_firewall_rule" "example" {
  direction  = "inbound"
  ip_version = 4
  vpc_id     = 12345
}
