### Edge cache server
# Services running on this VM: nginx (reverse proxy + cache on 80/443), SSH for
# management, s3cmd for asset synchronization from the S3 bucket below.

resource "netactuate_server" "edge" {
  hostname                    = "edge01.${lower(var.location)}.example.com"
  plan                        = var.server_plan
  location                    = var.location
  image                       = var.server_image
  package_billing_contract_id = var.billing_contract_id
  ssh_key                     = var.ssh_public_key
  tags                        = "pop, edge-cache, ${var.location}"
}

### S3 object storage bucket
# Private bucket for static assets served via nginx on the edge VM.

resource "netactuate_storage_bucket" "assets" {
  label    = "pop-${lower(var.location)}-assets"
  location = var.location
  capacity = var.bucket_capacity
  private  = true
}

### Block volume for nginx cache disk

resource "netactuate_storage_block_volume" "cache" {
  label    = "pop-${lower(var.location)}-cache"
  location = var.location
  capacity = 50
}

locals {
  s3_bucket_name        = "default"
  rbd_pool_name         = "global-block-pool"
  rbd_monitors_csv      = join(",", netactuate_storage_block_volume.cache.endpoints)
  secret_list_base_name = "pop-${lower(var.location)}-secrets"
  secret_list_name      = trimspace(var.secret_list_name_suffix) != "" ? "${local.secret_list_base_name}-${trimspace(var.secret_list_name_suffix)}" : local.secret_list_base_name
  secret_key_prefix     = replace(netactuate_secret_list.pop.name, "-", "_")
  secret_keys = {
    s3_access_key  = "${local.secret_key_prefix}_s3_access_key"
    s3_secret_key  = "${local.secret_key_prefix}_s3_secret_key"
    s3_endpoint    = "${local.secret_key_prefix}_s3_endpoint"
    s3_bucket      = "${local.secret_key_prefix}_s3_bucket"
    rbd_user_key   = "${local.secret_key_prefix}_rbd_user_key"
    rbd_secret_key = "${local.secret_key_prefix}_rbd_secret_key"
    rbd_monitors   = "${local.secret_key_prefix}_rbd_monitors"
    rbd_pool       = "${local.secret_key_prefix}_rbd_pool"
    rbd_namespace  = "${local.secret_key_prefix}_rbd_namespace"
    rbd_image      = "${local.secret_key_prefix}_rbd_image"
  }
}

### Secret list for app credentials

resource "netactuate_secret_list" "pop" {
  name = local.secret_list_name
}

resource "netactuate_secret_list_value" "s3_access_key" {
  secret_list_id = netactuate_secret_list.pop.id
  secret_key     = local.secret_keys.s3_access_key
  secret_value   = netactuate_storage_bucket.assets.access_key
}

resource "netactuate_secret_list_value" "s3_secret_key" {
  secret_list_id = netactuate_secret_list.pop.id
  secret_key     = local.secret_keys.s3_secret_key
  secret_value   = netactuate_storage_bucket.assets.secret_key
}

resource "netactuate_secret_list_value" "s3_endpoint" {
  secret_list_id = netactuate_secret_list.pop.id
  secret_key     = local.secret_keys.s3_endpoint
  secret_value   = netactuate_storage_bucket.assets.endpoints[0]
}

resource "netactuate_secret_list_value" "s3_bucket" {
  secret_list_id = netactuate_secret_list.pop.id
  secret_key     = local.secret_keys.s3_bucket
  secret_value   = local.s3_bucket_name
}

resource "netactuate_secret_list_value" "rbd_user_key" {
  secret_list_id = netactuate_secret_list.pop.id
  secret_key     = local.secret_keys.rbd_user_key
  secret_value   = netactuate_storage_block_volume.cache.user_key
}

resource "netactuate_secret_list_value" "rbd_secret_key" {
  secret_list_id = netactuate_secret_list.pop.id
  secret_key     = local.secret_keys.rbd_secret_key
  secret_value   = netactuate_storage_block_volume.cache.secret_key
}

resource "netactuate_secret_list_value" "rbd_monitors" {
  secret_list_id = netactuate_secret_list.pop.id
  secret_key     = local.secret_keys.rbd_monitors
  secret_value   = local.rbd_monitors_csv
}

resource "netactuate_secret_list_value" "rbd_pool" {
  secret_list_id = netactuate_secret_list.pop.id
  secret_key     = local.secret_keys.rbd_pool
  secret_value   = local.rbd_pool_name
}

resource "netactuate_secret_list_value" "rbd_namespace" {
  secret_list_id = netactuate_secret_list.pop.id
  secret_key     = local.secret_keys.rbd_namespace
  secret_value   = netactuate_storage_block_volume.cache.storage_namespace
}

resource "netactuate_secret_list_value" "rbd_image" {
  secret_list_id = netactuate_secret_list.pop.id
  secret_key     = local.secret_keys.rbd_image
  secret_value   = netactuate_storage_block_volume.cache.image_name
}

### Hypervisor-level firewall
# Applied at the NIC level on the hypervisor — no VM rebuild required.

resource "netactuate_firewall_set" "edge" {
  name        = "pop-${lower(var.location)}-edge"
  description = "Firewall for edge cache VM at ${var.location}"
  enabled     = true
}

# Allow HTTP from anywhere
resource "netactuate_firewall_rule" "allow_http" {
  firewall_set_id        = netactuate_firewall_set.edge.id
  ip_version             = "IPv4"
  action                 = "ACCEPT"
  enabled                = true
  rule_priority          = 10
  protocol               = "tcp"
  destination_port_start = 80
  destination_port_end   = 80
}

# Allow HTTPS from anywhere
resource "netactuate_firewall_rule" "allow_https" {
  firewall_set_id        = netactuate_firewall_set.edge.id
  ip_version             = "IPv4"
  action                 = "ACCEPT"
  enabled                = true
  rule_priority          = 20
  protocol               = "tcp"
  destination_port_start = 443
  destination_port_end   = 443
}

# Allow SSH from management CIDR only
resource "netactuate_firewall_rule" "allow_ssh" {
  firewall_set_id        = netactuate_firewall_set.edge.id
  ip_version             = "IPv4"
  action                 = "ACCEPT"
  enabled                = true
  rule_priority          = 30
  protocol               = "tcp"
  source_net             = [var.management_cidr]
  destination_port_start = 22
  destination_port_end   = 22
}

# Allow ICMP (ping) from anywhere for monitoring
resource "netactuate_firewall_rule" "allow_icmp" {
  firewall_set_id = netactuate_firewall_set.edge.id
  ip_version      = "IPv4"
  action          = "ACCEPT"
  enabled         = true
  rule_priority   = 40
  protocol        = "icmp"
}

# Allow WireGuard inbound — the cloud router (example 04) connects to this port
resource "netactuate_firewall_rule" "allow_wireguard" {
  firewall_set_id        = netactuate_firewall_set.edge.id
  ip_version             = "IPv4"
  action                 = "ACCEPT"
  enabled                = true
  rule_priority          = 45
  protocol               = "udp"
  destination_port_start = var.wireguard_port
  destination_port_end   = var.wireguard_port
}

# Default deny — drop everything else inbound
resource "netactuate_firewall_rule" "deny_all" {
  firewall_set_id = netactuate_firewall_set.edge.id
  ip_version      = "IPv4"
  action          = "DROP"
  enabled         = true
  rule_priority   = 1000
}

### Attach firewall to VM
# Pure API call to the hypervisor NIC — the server is NOT rebuilt.

resource "netactuate_firewall_set_vm" "edge" {
  firewall_set_id = netactuate_firewall_set.edge.id
  mbpkgid         = netactuate_server.edge.id

  depends_on = [netactuate_server.edge]
}

### anycast BGP session (optional)
# Skip this block by leaving var.bgp_group_id = null.

resource "netactuate_bgp_sessions" "edge" {
  count    = var.bgp_group_id != null ? 1 : 0
  mbpkgid  = netactuate_server.edge.id
  group_id = var.bgp_group_id
}
