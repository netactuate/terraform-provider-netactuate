data "netactuate_server" "server" {
  id = netactuate_server.server.id
}

data "netactuate_sshkey" "ssh_key" {
  id = netactuate_sshkey.sshkey.id
}

