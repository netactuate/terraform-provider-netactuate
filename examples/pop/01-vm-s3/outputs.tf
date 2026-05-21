### Server

output "server_ipv4" {
  description = "Primary public IPv4 address of the edge VM"
  value       = netactuate_server.edge.primary_ipv4
}

output "server_ipv6" {
  description = "Primary public IPv6 address of the edge VM"
  value       = netactuate_server.edge.primary_ipv6
}

output "server_hostname" {
  description = "Hostname of the edge VM"
  value       = netactuate_server.edge.hostname
}

### S3 storage

output "s3_endpoint" {
  description = "Primary S3 endpoint URL"
  value       = netactuate_storage_bucket.assets.endpoints[0]
}

output "s3_access_key" {
  description = "S3 access key for the assets bucket"
  value       = netactuate_storage_bucket.assets.access_key
  sensitive   = true
}

output "s3_secret_key" {
  description = "S3 secret key for the assets bucket"
  value       = netactuate_storage_bucket.assets.secret_key
  sensitive   = true
}

### s3cmd configuration
# Write this to ~/.s3cfg on the edge VM to configure s3cmd.
# See: https://netactuate.com/docs/infrastructure/storage/storage-s3

output "s3cfg" {
  description = "Ready-to-write ~/.s3cfg configuration file for s3cmd"
  sensitive   = true
  value       = <<-EOT
    [default]
    access_key = ${netactuate_storage_bucket.assets.access_key}
    secret_key = ${netactuate_storage_bucket.assets.secret_key}
    host_base = ${trimprefix(netactuate_storage_bucket.assets.endpoints[0], "https://")}
    host_bucket = ${trimprefix(netactuate_storage_bucket.assets.endpoints[0], "https://")}/%(bucket)s
    use_https  = True
    signature_v2 = True
    check_ssl_certificate = False
    check_ssl_hostname = False
  EOT
}

### Block volume

output "cache_volume_endpoints" {
  description = "Block storage endpoints for the nginx cache volume"
  value       = netactuate_storage_block_volume.cache.endpoints
}

output "cache_volume_image" {
  description = "Block volume image name (for mounting via rbd)"
  value       = netactuate_storage_block_volume.cache.image_name
}

output "cache_block_volume_id" {
  description = "Block volume ID"
  value       = netactuate_storage_block_volume.cache.block_volume_id
}

output "cache_storage_pool" {
  description = "Ceph pool for the block volume"
  value       = local.rbd_pool_name
}

output "cache_storage_namespace" {
  description = "Ceph namespace for the block volume"
  value       = netactuate_storage_block_volume.cache.storage_namespace
}

output "cache_storage_cluster_id" {
  description = "Ceph cluster UUID for the block volume"
  value       = netactuate_storage_block_volume.cache.storage_cluster_id
}

output "cache_rbd_user_key" {
  description = "Ceph client user key for this block volume"
  value       = netactuate_storage_block_volume.cache.user_key
  sensitive   = true
}

output "cache_rbd_secret_key" {
  description = "Ceph client secret key for this block volume"
  value       = netactuate_storage_block_volume.cache.secret_key
  sensitive   = true
}

output "cache_ceph_conf" {
  description = "Minimal ceph.conf for RBD client configuration"
  value       = <<-EOT
    [global]
    mon_host = ${join(",", netactuate_storage_block_volume.cache.endpoints)}
  EOT
}

output "cache_rbd_ls_command" {
  description = "Ready-to-run RBD command to list images in the volume pool/namespace"
  value       = "rbd --mon-host '${join(",", netactuate_storage_block_volume.cache.endpoints)}' --id '${netactuate_storage_block_volume.cache.user_key}' --key '${netactuate_storage_block_volume.cache.secret_key}' --pool '${local.rbd_pool_name}' --namespace '${netactuate_storage_block_volume.cache.storage_namespace}' ls"
  sensitive   = true
}

output "cache_rbd_map_command" {
  description = "Ready-to-run RBD command template to map the cache image"
  value       = "rbd --mon-host '${join(",", netactuate_storage_block_volume.cache.endpoints)}' --id '${netactuate_storage_block_volume.cache.user_key}' --key '${netactuate_storage_block_volume.cache.secret_key}' --pool '${local.rbd_pool_name}' --namespace '${netactuate_storage_block_volume.cache.storage_namespace}' map '${netactuate_storage_block_volume.cache.image_name}'"
  sensitive   = true
}

### Firewall

output "firewall_set_id" {
  description = "Firewall set ID (informational)"
  value       = netactuate_firewall_set.edge.id
}

output "firewall_rule_ids" {
  description = "Map of firewall rule names to their IDs"
  value = {
    allow_http      = netactuate_firewall_rule.allow_http.id
    allow_https     = netactuate_firewall_rule.allow_https.id
    allow_ssh       = netactuate_firewall_rule.allow_ssh.id
    allow_icmp      = netactuate_firewall_rule.allow_icmp.id
    allow_wireguard = netactuate_firewall_rule.allow_wireguard.id
    deny_all        = netactuate_firewall_rule.deny_all.id
  }
}

### WireGuard handoff
# Use these outputs when filling in example 04's terraform.tfvars.

output "wireguard_endpoint" {
  description = "WireGuard endpoint for this VM — paste as wireguard_remote_endpoint in example 04"
  value       = "${netactuate_server.edge.primary_ipv4}:${var.wireguard_port}"
}

output "wireguard_setup_guide" {
  description = "Commands to install WireGuard and generate keys on the edge VM — run before applying example 04"
  value       = <<-EOT
    # 1. SSH to VM
    ssh ubuntu@${netactuate_server.edge.primary_ipv4}

    # 2. Install WireGuard and generate keypair
    sudo apt update && sudo apt install -y wireguard bird2
    wg genkey | sudo tee /etc/wireguard/private.key | wg pubkey | sudo tee /etc/wireguard/public.key
    sudo chmod 600 /etc/wireguard/private.key

    # 3. Copy this public key — paste as wireguard_remote_public_key in example 04's tfvars
    sudo cat /etc/wireguard/public.key

    # 4. Apply example 04, then use its wireguard_vm_conf and bird_conf outputs to finish setup.
  EOT
}

### Secret store

output "secret_list_id" {
  description = "Secret list ID — reference via API to retrieve credentials without exposing them in pipelines"
  value       = netactuate_secret_list.pop.id
}

output "secret_list_name" {
  description = "Secret list name used for this deployment"
  value       = netactuate_secret_list.pop.name
}

output "secret_list_keys" {
  description = "Actual secret keys written to the secret list (namespaced for account-wide uniqueness)"
  value       = local.secret_keys
}

### Ansible host vars
# Pipe this to: terraform output -json ansible_host_vars | jq > host_vars/edge01.json

output "ansible_host_vars" {
  description = "Structured vars for Ansible host_vars — contains all credentials needed to configure the VM"
  sensitive   = true
  value = {
    ansible_host = netactuate_server.edge.primary_ipv4
    ansible_user = "ubuntu"

    s3_endpoint     = netactuate_storage_bucket.assets.endpoints[0]
    s3_bucket       = "default"
    s3_bucket_label = netactuate_storage_bucket.assets.label
    s3_access_key   = netactuate_storage_bucket.assets.access_key
    s3_secret_key   = netactuate_storage_bucket.assets.secret_key

    cache_volume_endpoints   = netactuate_storage_block_volume.cache.endpoints
    cache_volume_image       = netactuate_storage_block_volume.cache.image_name
    cache_block_volume_id    = netactuate_storage_block_volume.cache.block_volume_id
    cache_storage_pool       = local.rbd_pool_name
    cache_storage_namespace  = netactuate_storage_block_volume.cache.storage_namespace
    cache_storage_cluster_id = netactuate_storage_block_volume.cache.storage_cluster_id
    cache_rbd_user_key       = netactuate_storage_block_volume.cache.user_key
    cache_rbd_secret_key     = netactuate_storage_block_volume.cache.secret_key
  }
}
