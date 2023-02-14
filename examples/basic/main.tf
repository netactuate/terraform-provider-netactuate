provider "netactuate" {
  api_key = "NETACTUATE_API_KEY"
}

resource "netactuate_sshkey" "sshkey" {
  name = "default_key"
  key  = "ssh-ed25519 REDACTED_SSH_KEY user@email.test"
}

resource "netactuate_server" "server" {
  hostname    = "terraform.example.com"
  plan        = "VR1x1x25"
  location    = "SJC - San Jose, CA" // 3
  image_id    = 5726                 // Ubuntu 20.04.3 LTS x64
  ssh_key_id  = netactuate_sshkey.sshkey.id
  package_billing_contract_id = PROVIDED_CODE
}

//  Other Variables you can use inside the netactuate_server resource block:
//
//  package_billing             = "usage"
//  package_billing_contract_id = "6d0037798723523"
//  location_id = 3 // SJC - San Jose, CA
//  image = "Ubuntu 20.04.3 LTS x64" // 5726
//  password = "password1"
//  ssh_key = file("${path.module}/ssh/id_rsa.pub")
//  user_data = file("init.sh")
