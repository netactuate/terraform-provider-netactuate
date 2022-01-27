provider "netactuate" {}

resource "netactuate_server" "server" {
  hostname = "vm1.com"
  plan = "VR1x1x25"
  package_billing = "123fe"
  package_billing_contract_id = "6d0037798723523"
  //location_id = 3 // SJC - San Jose, CA
  location = "SJC - San Jose, CA" // 3
  image_id = 5726 // Ubuntu 20.04.3 LTS x64
  //image = "Ubuntu 20.04.3 LTS x64" // 5726
  //password = "password1"
  ssh_key_id = netactuate_sshkey.sshkey.id
}

resource "netactuate_sshkey" "sshkey" {
  name = "default_key"
  key = file("${path.module}/ssh/id_rsa.pub")
}
