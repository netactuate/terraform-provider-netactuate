resource "netactuate_dedicated_server_ipv4_reverse" "example" {
  ipv4_id = 12345
  mbpkgid = 12345
  reverse = "host.example.com"
}
