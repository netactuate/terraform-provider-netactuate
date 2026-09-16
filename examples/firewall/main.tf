provider "netactuate" {
  # API key can also be set via NETACTUATE_API_KEY environment variable
  api_key = "NETACTUATE_API_KEY"
  api_url = "vAPI2_URL" #vAPI2 URL
}


resource "netactuate_sshkey" "sshkey" {
  name = "terraform-firewall-test2"
  key  = var.ssh_public_key
}

resource "netactuate_server" "vm" {
  hostname                    = "fw-test.example.com"
  plan                        = "VR1x1x25"
  location                    = "DEVRDU"
  image                       = "Ubuntu 24.04 LTS (20240423)"
  ssh_key_id                  = netactuate_sshkey.sshkey.id
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

variable "ssh_public_key" {
  description = "SSH public key to authorize on the example resources."
  type        = string
}
