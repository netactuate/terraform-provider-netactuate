resource "netactuate_cloud_ipv4_reverse_dns" "example" {
  address_id = 12345
  mbpkgid    = 12345
  reverse    = "host.example.com"
}
