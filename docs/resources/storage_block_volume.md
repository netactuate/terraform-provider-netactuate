# netactuate_storage_block_volume

Manages a block storage volume. A block volume is a pre-provisioned RBD image within a Ceph cluster, addressable as a single block device.

## Example Usage

```hcl
resource "netactuate_storage_block_volume" "app_data" {
  label    = "app-data-vol"
  location = "SJC"
  capacity = 50
}

output "block_volume_id" {
  value = netactuate_storage_block_volume.app_data.block_volume_id
}

output "image_name" {
  value = netactuate_storage_block_volume.app_data.image_name
}

output "endpoints" {
  value = netactuate_storage_block_volume.app_data.endpoints
}
```

## Argument Reference

### Required

- `label` (String) — Display label for the block volume.

### Required (one of `location`/`location_id`)

- `location` (String) — Location name, e.g. `"SJC"`. Forces recreation.
- `location_id` (Number) — Numeric location ID. Forces recreation.

### Optional

- `capacity` (Number) — Provisioned volume size in GB. Valid range: 1–1000.

### Computed

- `block_volume_id` (Number) — The API-assigned volume ID.
- `ready` (Boolean) — Whether the volume is provisioned and ready.
- `assigned_on` (String) — ISO 8601 timestamp of provisioning.
- `location_name` (String) — Full location name.
- `total_capacity_gb` (Number) — Currently provisioned capacity in GB.
- `endpoints` (List of String) — Ceph monitor endpoints (host:port).
- `storage_pool` (String) — The Ceph pool name. In NetActuate block-volume workflows this is typically `global-block-pool`.
- `storage_namespace` (String) — The Ceph namespace.
- `storage_cluster_id` (String) — The Ceph cluster UUID.
- `image_name` (String) — The RBD image name for this volume.
- `user_key` (String, Sensitive) — Ceph client user key for RBD operations.
- `secret_key` (String, Sensitive) — Ceph client secret key for RBD operations.

## Import

```
terraform import netactuate_storage_block_volume.app_data <block_volume_id>
```
