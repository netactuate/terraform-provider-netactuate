# netactuate_storage_object_store

Manages an S3-compatible object store. Similar to a storage bucket but with a distinct product profile. Provides S3 credentials and endpoints on creation.

## Example Usage

```hcl
resource "netactuate_storage_object_store" "backups" {
  label               = "prod-backups"
  location            = "SJC"
  capacity            = 500
  enable_auto_scaling = false
}

output "object_store_endpoint" {
  value = netactuate_storage_object_store.backups.endpoints[0]
}

output "object_store_access_key" {
  value     = netactuate_storage_object_store.backups.access_key
  sensitive = true
}
```

## Argument Reference

### Required

- `label` (String) — Display label for the object store.

### Required (one of `location`/`location_id`)

- `location` (String) — Location name, e.g. `"SJC"`. Forces recreation.
- `location_id` (Number) — Numeric location ID. Forces recreation.

### Optional

- `capacity` (Number) — Provisioned storage in GB. Valid range: 1–1000.
- `enable_auto_scaling` (Boolean) — Automatically expand capacity to match usage peaks. Default: `false`.

### Computed

- `object_store_id` (Number) — The API-assigned object store ID.
- `ready` (Boolean) — Whether the object store is provisioned and ready.
- `assigned_on` (String) — ISO 8601 timestamp of provisioning.
- `location_name` (String) — Full location name.
- `total_capacity_gb` (Number) — Currently provisioned capacity in GB.
- `auto_scaling` (Boolean) — Current auto-scaling state.
- `endpoints` (List of String) — S3-compatible HTTPS endpoint URLs.
- `access_key` (String, Sensitive) — S3 access key ID.
- `secret_key` (String, Sensitive) — S3 secret access key.
- `user_key` (String, Sensitive) — S3 user key.

## Import

```
terraform import netactuate_storage_object_store.backups <object_store_id>
```
