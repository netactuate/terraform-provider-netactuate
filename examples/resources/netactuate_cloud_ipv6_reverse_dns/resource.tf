resource "netactuate_cloud_ipv6_reverse_dns" "example" {
  address_id = 12345
  mbpkgid    = 12345
  reverse    = "host.example.com"
}
