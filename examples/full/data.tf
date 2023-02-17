data "netactuate_server" "server" {
  id = netactuate_server.server.id

  depends_on = [netactuate_bgp_sessions.bgp_sessions]
}

data "netactuate_sshkey" "ssh_key" {
  id = netactuate_sshkey.sshkey.id
}

data "netactuate_bgp_sessions" "bgp_sessions" {
  mbpkgid = netactuate_server.server.id

  depends_on = [netactuate_bgp_sessions.bgp_sessions]
}
