output "server" {
  value = data.netactuate_server.server
}

output "ssh_key" {
  value = data.netactuate_sshkey.ssh_key
}

output "bgp_sessions" {
  value = data.netactuate_bgp_sessions.bgp_sessions
}
