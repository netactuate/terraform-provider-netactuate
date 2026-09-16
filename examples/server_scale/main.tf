provider "netactuate" {
  # API key can also be set via NETACTUATE_API_KEY environment variable
  api_key = "NETACTUATE_API_KEY"
  api_url = "vAPI2_URL" #vAPI2 URL
}

resource "netactuate_sshkey" "sshkey" {
  name = "scale-example"
  key  = var.ssh_public_key
}

resource "netactuate_server" "scalable" {
  hostname                    = "scale-test.example.com"
  location                    = "DEVRDU"
  image                       = "Ubuntu 24.04 LTS (20240423)"
  ssh_key_id                  = netactuate_sshkey.sshkey.id
  package_billing_contract_id = "1"

  plan = "VR1x1x25"
  # plan = "VR2x2x25"   # deploy VR1x1x25 and uncomment this for scale

  # Upgrades (larger plan, no reboot) apply automatically. A downsize/reboot is
  # rejected during apply before the scale API call unless you opt in. Default is false.
  # allow_downsize_reboot = true
}

output "server_id" {
  value = netactuate_server.scalable.id
}

output "server_ip" {
  value = netactuate_server.scalable.primary_ipv4
}

variable "ssh_public_key" {
  description = "SSH public key to authorize on the example resources."
  type        = string
}
