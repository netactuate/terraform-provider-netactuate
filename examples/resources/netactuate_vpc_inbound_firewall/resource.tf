resource "netactuate_vpc_gateway_firewall_rule" "http" {
  vpc_id      = 12345
  ip_version  = 4
  direction   = "inbound"
  protocol    = "TCP"
  description = "Allow HTTP"
  port_start  = 80
  port_end    = 80
}

resource "netactuate_vpc_inbound_firewall" "example" {
  vpc_id = 12345
  ipv4   = true

  depends_on = [netactuate_vpc_gateway_firewall_rule.http]
}
