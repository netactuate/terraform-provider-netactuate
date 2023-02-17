output "server_details" {
  value = {
    for server in netactuate_server.map:
    server.hostname => { "ipv4" = server.primary_ipv4, "ipv6" = server.primary_ipv6 }
  }
}
