resource "netactuate_dns_record" "example" {
  content = "192.0.2.10"
  type    = "A"
  zone_id = 12345
}
