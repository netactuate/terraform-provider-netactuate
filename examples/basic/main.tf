provider "netactuate" {
  //api_key = "REDACTED_API_KEY"
}

resource "netactuate_sshkey" "sshkey" {
  name = "default_key"
  key  = "ssh-ed25519 key-data..."
}

resource "netactuate_server" "server" {
  hostname                    = "terraform.example.com"
  plan                        = "VR1x1x25"
  package_billing             = "usage"
  package_billing_contract_id = "6d0037798723523"
  //location_id = 3 // SJC - San Jose, CA
  location = "SJC - San Jose, CA" // 3
  image_id = 5726                 // Ubuntu 20.04.3 LTS x64
  //image = "Ubuntu 20.04.3 LTS x64" // 5726
  //password = "password1"
  //ssh_key = file("${path.module}/ssh/id_rsa.pub")
  ssh_key_id = netactuate_sshkey.sshkey.id
  //user_data = file("init.sh")
}
