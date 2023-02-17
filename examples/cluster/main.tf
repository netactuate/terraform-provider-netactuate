provider "netactuate" {
  # Optional: API key can be set here, rather than as an environment variable
  //api_key = "NETACTUATE_API_KEY"
}

# Define SSH Key to use for server login
resource "netactuate_sshkey" "sshkey" {
  name = "default_key"
  key  = "ssh-ed25519 SHORT_SSH_KEY_REDACTED email@email.test"
}

# Define locations, OS, and instance sizing ( fqdn = { location name , OS_image , VM sizing } )
locals {
  server_settings = {
    "server-1.terraform.test"  = { location = "LGA", image = "Ubuntu 22.04 (20221110)", plan = "VR1x1x25" },
    "server-2.terraform.test"  = { location = "AMS", image = "Ubuntu 22.04 (20221110)", plan = "VR2x2x25" },
    "server-3.terraform.test"  = { location = "LAX", image = "Ubuntu 22.04 (20221110)", plan = "VR1x1x25" }
  }
}

# Define server creation values
resource "netactuate_server" "map" {
  for_each   = local.server_settings
  hostname   = each.key
  plan       = each.value.plan
  location   = each.value.location
  image      = each.value.image
  ssh_key_id = netactuate_sshkey.sshkey.id
  package_billing_contract_id = PROVIDED_CODE
}
