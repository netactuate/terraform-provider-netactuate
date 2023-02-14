provider "netactuate" {
  api_key = "NETACTUATE_API_KEY"
//  api_url = "https://netactuate.provided.url"
}

# Define SSH Key to use for server login
resource "netactuate_sshkey" "sshkey" {
  name = "default_key"
  key  = "ssh-ed25519 SHORT_SSH_KEY_REDACTED email@email.test"
}

# Define locations, OS, and VM Sizing ( fqdn = { location name , OS_image , VM sizing } )
locals {
  server_settings = {
    "server-1.terraform.test"  = { location = "IAD - Reston, VA", image_id = 5762, plan = "VR1x1x25" },
    "server-2.terraform.test"  = { location = "CHI - Chicago, IL", image_id = 5762, plan = "VR2x2x25" },
    "server-3.terraform.test"  = { location = "RDU - Raleigh, NC", image_id = 5762, plan = "VR1x1x25" }
  }
}

# Define server creation values
resource "netactuate_server" "map" {
  for_each      = local.server_settings
  hostname                    = each.key
  plan                        = each.value.plan
  location = each.value.location
  image_id = each.value.image_id
  ssh_key_id = netactuate_sshkey.sshkey.id
  package_billing_contract_id = PROVIDED_CODE

//  user_data_base64 = "IyEvYmluL3NoCmVjaG8gIkhlbGxvIFdvcmxkIiB8IHRlZSAvaGVsbG8ubG9nCg=="
//  user_data = file("init.sh")
}

