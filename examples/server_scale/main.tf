provider "netactuate" {
  # API key can also be set via NETACTUATE_API_KEY environment variable
  api_key    = "NETACTUATE_API_KEY"
  api_url    = "vAPI2_URL"  #vAPI2 URL
}

resource "netactuate_sshkey" "sshkey" {
  name = "scale-example"
  key  = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCtRyTqrahdFrNLiNh+f6U29nSDdUWe9FIIO2JXE+SDk3Job7Z10LU8iNJ8p1cyTPW2aWIN5R6Nk1fmJYsAVVtibUaIcybGh6sYAh8HbYhdo4FhyMjzSzhvNoY7VjkqEEiwKpcGSSuyAU+M8okYePkz7ECkC52OhBXzGbsGiXreJeePJttyjWEEkAt4N71BsiEPWllD1K13ZnX0qJDvpEW79bx3CsEHHD2iM2FzaSuzudQ8eyUvJqR6D1e604ZWV4nGIlGA8HUZOAG7vfn5+u7/154vqdKurcQPOuVwrzUVUUE3a9ITohj7aHQRQSBSUlOddQ9Ks9h+6c7qWGT7VUNacUsCpEcVCHJJ6D4XrSYtRLrHYGrINczJirSInHE3ZMz3cjepsyi8gcZkepyOqJOnALQLhHGaYNK9J+ubeK77J6Tfja7hX/OvaPHNXj6ruHh2R3/DtWv832Ad4ytpBizcZ1X3VAQ6nSwgv/P+V+P1Voo95gvmAANSZZ/JBijE5YY= scale-example@test"
}

resource "netactuate_server" "scalable" {
  hostname    = "scale-test.example.com"
  location    = "DEVRDU"
  image       = "Ubuntu 24.04 LTS (20240423)"
  ssh_key_id  = netactuate_sshkey.sshkey.id
  package_billing_contract_id = "1"

  plan = "VR1x1x25"
  # plan = "VR2x2x25"   # deploy VR1x1x25 and uncomment this for scale

  # Allow the API to reboot the server during scaling. Required for RAM downscaling. Default is true.
  # allow_reboot = true
}

output "server_id" {
  value = netactuate_server.scalable.id
}

output "server_ip" {
  value = netactuate_server.scalable.primary_ipv4
}
