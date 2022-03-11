provider "netactuate" {}

resource "netactuate_server" "server" {
  hostname = "vm1.com"
  plan = "VR1x1x25"
  package_billing = "usage"
  package_billing_contract_id = "6d0037798723523"
  //location_id = 3 // SJC - San Jose, CA
  location = "SJC - San Jose, CA" // 3
  image_id = 5726 // Ubuntu 20.04.3 LTS x64
  //image = "Ubuntu 20.04.3 LTS x64" // 5726
  //password = "password1"
  ssh_key_id = netactuate_sshkey.sshkey.id
  //user_data = file("init.sh")
  //user_data_base64 = "IyEvYmluL3NoCmVjaG8gIkhlbGxvIFdvcmxkIiB8IHRlZSAvaGVsbG8ubG9nCg=="
}

resource "netactuate_sshkey" "sshkey" {
  name = "default_key"
  key = file("${path.module}/ssh/id_rsa.pub")
}

resource "netactuate_bgp_sessions" "bgp_sessions" {
  mbpkgid = netactuate_server.server.id
  group_id = 12345
  ipv6 = true
}
