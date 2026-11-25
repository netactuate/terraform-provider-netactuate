resource "netactuate_vpc_ip_reservation" "example" {
  vpc_id      = 12345
  description = "application VIP"
  # Omit address to allocate the next free address.
  address     = "10.0.0.200"
}
