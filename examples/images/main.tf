provider "netactuate" {
  # API key can also be set via NETACTUATE_API_KEY environment variable
  api_key    = "NETACTUATE_API_KEY"
  api_url    = "vAPI2_URL"  #vAPI2 URL
}

# Create a my image from VM
resource "netactuate_image" "example" {
  name              = "My Terraform Ubuntu Image test"
  description       = "Ubuntu 24.04 with custom configuration test"
  server_id         = 1174
  keep_ssh_userdirs = false
}

data "netactuate_image" "example" {
  id = netactuate_image.example.id
}

output "image_id" {
  value = netactuate_image.example.id
}

output "image_name" {
  value = data.netactuate_image.example.name
}
