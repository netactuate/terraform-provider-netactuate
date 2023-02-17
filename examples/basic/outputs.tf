output "ipv4" {
  value       = netactuate_server.server.primary_ipv4
  description = "The primary IPv4 address of the VM."
}

output "ipv6" {
  value       = netactuate_server.server.primary_ipv6
  description = "The primary IPv6 address of the VM."
}
