provider "netactuate" {
  # Optional: API key can be set here, rather than as an environment variable
  //api_key = "NETACTUATE_API_KEY"
  //api_url = "https://netactuate.alternate.url"
}

# Define SSH Key to use for server login
resource "netactuate_sshkey" "sshkey" {
  name = "default_key"
  key  = var.ssh_public_key
}

# Define locations, OS, and instance sizing
resource "netactuate_server" "server" {
  hostname                    = "terraform.example.com"
  plan                        = "VR1x1x25"
  location                    = "SJC"
  image                       = "Ubuntu 22.04 (20221110)"
  ssh_key_id                  = netactuate_sshkey.sshkey.id
  package_billing_contract_id = PROVIDED_CODE

  # Alternate and optional parameters; see documentation:
  //location_id = 3
  //image_id = 5762
  //user_data = file("init.sh")
  //user_data_base64 = "IyEvYmluL3NoCmVjaG8gIkhlbGxvIFdvcmxkIiB8IHRlZSAvaGVsbG8ubG9nCg=="
  //password = var.server_password
}

# Define a BGP group to provision sessions for this instance
resource "netactuate_bgp_sessions" "bgp_sessions" {
  mbpkgid  = netactuate_server.server.id
  group_id = PROVIDED_CODE
  ipv6     = true
}

variable "ssh_public_key" {
  description = "SSH public key to authorize on the example resources."
  type        = string
}
