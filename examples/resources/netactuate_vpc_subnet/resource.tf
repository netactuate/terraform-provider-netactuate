resource "netactuate_vpc_subnet" "example" {
  vpc_id = 12345
  network = "172.31.251.0/24"
  label = "app"
}
