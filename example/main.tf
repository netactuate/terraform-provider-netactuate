provider "netactuate" {}

resource "netactuate_server" "server" {
  hostname = "vm1.com"
  plan = "VR1x1x25"
  location_id = 3 //SJC - San Jose, CA
  image_id = 5348 //Ubuntu 18.04.2 LTS x64 (HVM\/PV)
  //password = "password1"
  ssh_key_id = netactuate_sshkey.sshkey.id
}

resource "netactuate_sshkey" "sshkey" {
  name = "default_key"
  key = file("${path.module}/ssh/id_rsa.pub")
}
