provider "netactuate" {
  # API key can also be set via NETACTUATE_API_KEY environment variable
  api_key    = "NETACTUATE_API_KEY"
  api_url    = "vAPI2_URL"  #vAPI2 URL
  api_url_v3 = "vAPI3_URL"  #vAPI3 URL
}


data "netactuate_storage_locations" "available" {}

output "available_storage_locations" {
  value = data.netactuate_storage_locations.available.locations
}

# Bucket
resource "netactuate_storage_bucket" "example" {
  label              = "Terraform test bucket 2"
  location           = "DEVRDU - Raleigh, NC"
  capacity           = 2
  private            = false
  enable_auto_scaling = false
}

output "bucket_id" {
  value = netactuate_storage_bucket.example.bucket_id
}

output "bucket_endpoints" {
  value = netactuate_storage_bucket.example.endpoints
}

output "bucket_access_key" {
  value     = netactuate_storage_bucket.example.access_key
  sensitive = true
}

# Object Store
resource "netactuate_storage_object_store" "example" {
  label              = "Terraform test object store 2"
  location           = "DEVRDU - Raleigh, NC"
  capacity           = 2
  enable_auto_scaling = false
}

output "object_store_id" {
  value = netactuate_storage_object_store.example.object_store_id
}

output "object_store_endpoints" {
  value = netactuate_storage_object_store.example.endpoints
}

# Dynamic Block
resource "netactuate_storage_block_namespace" "example" {
  label              = "Terraform dynamic block 2"
  location           = "DEVRDU - Raleigh, NC"
  capacity           = 2
  enable_auto_scaling = false
}

output "block_namespace_id" {
  value = netactuate_storage_block_namespace.example.block_namespace_id
}

output "block_namespace_pool" {
  value = netactuate_storage_block_namespace.example.storage_pool
}

output "block_namespace_endpoints" {
  value = netactuate_storage_block_namespace.example.endpoints
}

# Block Volume
resource "netactuate_storage_block_volume" "example" {
  label       = "Terraform block volume 2"
  location    = "DEVRDU - Raleigh, NC"
  capacity    = 2
}

output "block_volume_id" {
  value = netactuate_storage_block_volume.example.block_volume_id
}

output "block_volume_image" {
  value = netactuate_storage_block_volume.example.image_name
}

output "block_volume_endpoints" {
  value = netactuate_storage_block_volume.example.endpoints
}
