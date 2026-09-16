resource "netactuate_vpc_gateway_dnat_rule" "example" {
  ip_version          = 4
  translation_address = "192.0.2.10"
  vpc_id              = 12345
}
