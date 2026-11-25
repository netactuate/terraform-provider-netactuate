resource "netactuate_vpc_attachment" "example" {
  vpc_id      = 12345
  type        = "customer-router"
  address     = "10.0.0.2"
  description = "customer router"
  accept      = ["192.0.2.0/24"]
  advertise   = ["10.0.0.0/24"]
}
