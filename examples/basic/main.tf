provider "netactuate" {
  # Optional: API key can be set here, rather than as an environment variable
  //api_key = "NETACTUATE_API_KEY"
}

# Define SSH Key to use for server login
resource "netactuate_sshkey" "sshkey" {
  name = "default_key"
  key  = "ssh-ed25519 REDACTED_SSH_KEY user@email.test"
}

# Define locations, OS, and instance sizing
resource "netactuate_server" "server" {
  hostname    = "terraform.example.com"
  plan        = "VR1x1x25"
  location    = "SJC"
  image       = "Ubuntu 22.04 (20221110)"
  ssh_key_id  = netactuate_sshkey.sshkey.id
  package_billing_contract_id = PROVIDED_CODE
}

