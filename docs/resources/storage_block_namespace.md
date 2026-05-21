# netactuate_storage_block_namespace

Manages a block storage namespace. A block namespace is a Ceph-backed pool that provides RBD (RADOS Block Device) endpoints. Used for attaching persistent block storage to virtual machines or containers via the Ceph protocol.

## Example Usage

```hcl
resource "netactuate_storage_block_namespace" "db_storage" {
  label               = "prod-db-storage"
  location            = "SJC"
  capacity            = 200
  enable_auto_scaling = false
}

output "block_namespace_id" {
  value = netactuate_storage_block_namespace.db_storage.block_namespace_id
}

output "block_endpoints" {
  value = netactuate_storage_block_namespace.db_storage.endpoints
}

output "storage_pool" {
  value = netactuate_storage_block_namespace.db_storage.storage_pool
}
```

## Argument Reference

### Required

- `label` (String) — Display label for the block namespace.

### Required (one of `location`/`location_id`)

- `location` (String) — Location name, e.g. `"SJC"`. Forces recreation.
- `location_id` (Number) — Numeric location ID. Forces recreation.

### Optional

- `capacity` (Number) — Provisioned storage in GB.
- `enable_auto_scaling` (Boolean) — Automatically expand capacity. Default: `false`.

### Computed

- `block_namespace_id` (Number) — The API-assigned namespace ID.
- `ready` (Boolean) — Whether the namespace is provisioned and ready.
- `assigned_on` (String) — ISO 8601 timestamp of provisioning.
- `location_name` (String) — Full location name.
- `total_capacity_gb` (Number) — Currently provisioned capacity in GB.
- `auto_scaling` (Boolean) — Current auto-scaling state.
- `endpoints` (List of String) — Ceph monitor endpoints (host:port).
- `storage_pool` (String) — The Ceph pool name. In NetActuate block-namespace workflows this is typically `global-block-pool`.
- `storage_namespace` (String) — The Ceph namespace within the pool.
- `storage_cluster_id` (String) — The Ceph cluster UUID.
- `user_key` (String, Sensitive) — Ceph client user key for namespace-backed RBD operations.
- `secret_key` (String, Sensitive) — Ceph client secret key for namespace-backed RBD operations.

## Import

```
terraform import netactuate_storage_block_namespace.db_storage <block_namespace_id>
```
