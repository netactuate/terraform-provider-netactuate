provider "netactuate" {
  # API key can also be set via NETACTUATE_API_KEY environment variable
  api_key    = "NETACTUATE_API_KEY"
  api_url    = "vAPI2_URL"  #vAPI2 URL
}


resource "netactuate_sshkey" "sshkey" {
  name = "terraform-firewall-test2"
  key  = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCtRyTqrahdFrNLiNh+f6U29nSDdUWe9FIIO2JXE+SDk3Job7Z10LU8iNJ8p1cyTPW2aWIN5R6Nk1fmJYsAVVtibUaIcybGh6sYAh8HbYhdo4FhyMjzSzhvNoY7VjkqEEiwKpcGSSuyAU+M8okYePkz7ECkC52OhBXzGbsGiXreJeePJttyjWEEkAt4N71BsiEPWllD1K13ZnX0qJDvpEW79bx3CsEHHD2iM2FzaSuzudQ8eyUvJqR6D1e604ZWV4nGIlGA8HUZOAG7vfn5+u7/154vqdKurcQPOuVwrzUVUUE3a9ITohj7aHQRQSBSUlOddQ9Ks9h+6c7qWGT7VUNacUsCpEcVCHJJ6D4XrSYtRLrHYGrINczJirSInHE3ZMz3cjepsyi8gcZkepyOqJOnALQLhHGaYNK9J+ubeK77J6Tfja7hX/OvaPHNXj6ruHh2R3/DtWv832Ad4ytpBizcZ1X3VAQ6nSwgv/P+V+P1Voo95gvmAANSCI/JBijE5YY= test@test"
}

resource "netactuate_server" "vm" {
  hostname                    = "fw-test.example.com"
  plan                        = "VR1x1x25"
  location                    = "DEVRDU"
  image       = "Ubuntu 24.04 LTS (20240423)"
  ssh_key_id   = netactuate_sshkey.sshkey.id
  package_billing_contract_id = 1
}

resource "netactuate_firewall_set" "example" {
  name        = "Terraform Firewall Test"
  description = "Test set"
  enabled     = true
}

resource "netactuate_firewall_rule" "allow_ssh" {
  firewall_set_id        = netactuate_firewall_set.example.id
  ip_version             = "IPv4"
  action                 = "ACCEPT"
  protocol               = "tcp"
  destination_port_start = 22
  destination_port_end   = 22
  enabled                = true
  admin_comment          = "Allow SSH"
}

resource "netactuate_firewall_rule" "allow_http" {
  firewall_set_id        = netactuate_firewall_set.example.id
  ip_version             = "IPv4"
  action                 = "ACCEPT"
  protocol               = "tcp"
  destination_port_start = 80
  destination_port_end   = 443
  enabled                = true
  admin_comment          = "Allow HTTP/HTTPS"
}

resource "netactuate_firewall_rule" "allow_icmp" {
  firewall_set_id = netactuate_firewall_set.example.id
  ip_version      = "IPv4"
  action          = "ACCEPT"
  protocol        = "icmp"
  icmp_type       = "echo-request"
  enabled         = true
  admin_comment   = "Allow Ping"
}

resource "netactuate_firewall_rule" "allow_ssh_v6" {
  firewall_set_id        = netactuate_firewall_set.example.id
  ip_version             = "IPv6"
  action                 = "ACCEPT"
  protocol               = "tcp"
  destination_port_start = 22
  destination_port_end   = 22
  enabled                = true
  rule_priority          = 102
  admin_comment          = "Allow SSH over IPv6"
}

resource "netactuate_firewall_rule" "drop_all_v4" {
  firewall_set_id = netactuate_firewall_set.example.id
  ip_version      = "IPv4"
  action          = "DROP"
  enabled         = true
  rule_priority   = 100
  admin_comment   = "Drop all other IPv4 traffic"
}

resource "netactuate_firewall_rule" "drop_all_v6" {
  firewall_set_id = netactuate_firewall_set.example.id
  ip_version      = "IPv6"
  action          = "DROP"
  enabled         = true
  rule_priority   = 101
  admin_comment   = "Drop all other IPv6 traffic"
}

# Attach the firewall set to the VM
resource "netactuate_firewall_set_vm" "example_vm" {
  firewall_set_id = netactuate_firewall_set.example.id
  mbpkgid         = netactuate_server.vm.id
}


# To detach manually without destroying other resources:
#   terraform destroy -target=netactuate_firewall_set_vm.example_vm
