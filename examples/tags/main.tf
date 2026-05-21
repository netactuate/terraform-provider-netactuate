# Tagging example.
#
# Auth: export NETACTUATE_API_KEY (or set provider api_key).
# The provider exposes one server-level tags field. It accepts a single tag
# or a comma-separated list and forces the portal/API tag set to match.

variable "contract_id" {
  type    = string
  default = "301"
}

variable "ssh_public_key" {
  type        = string
  description = "Public SSH key material (e.g. file(\"~/.ssh/id_ed25519.pub\"))."
}

provider "netactuate" {}

resource "netactuate_sshkey" "default" {
  name = "tags-example-key"
  key  = var.ssh_public_key
}

resource "netactuate_server" "api" {
  hostname                    = "api01.example.com"
  plan                        = "SSD 2"
  location                    = "SJC"
  image                       = "Ubuntu 22.04 LTS"
  ssh_key_id                  = netactuate_sshkey.default.id
  package_billing_contract_id = var.contract_id

  tags = "kube, sjc, cluster"
}
