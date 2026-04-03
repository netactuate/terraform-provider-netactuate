# netactuate_storage_bucket

Manages an S3-compatible object storage bucket. Provides S3 API credentials and endpoints on creation.

## Example Usage

```hcl
data "netactuate_storage_locations" "available" {}

resource "netactuate_storage_bucket" "assets" {
  label               = "prod-assets"
  location            = "SJC"
  capacity            = 100
  private             = true
  enable_auto_scaling = true
}

output "bucket_endpoint" {
  value = netactuate_storage_bucket.assets.endpoints[0]
}

output "bucket_access_key" {
  value     = netactuate_storage_bucket.assets.access_key
  sensitive = true
}

output "bucket_secret_key" {
  value     = netactuate_storage_bucket.assets.secret_key
  sensitive = true
}
```

## Argument Reference

### Required

- `label` (String) — Display label for the bucket.

### Required (one of `location`/`location_id`)

- `location` (String) — Location name, e.g. `"SJC"`. Use `data.netactuate_storage_locations` to list available locations. Forces recreation.
- `location_id` (Number) — Numeric location ID. Forces recreation.

### Optional

- `capacity` (Number) — Provisioned storage in GB. Valid range: 1–1000.
- `enable_auto_scaling` (Boolean) — Automatically expand capacity to match usage peaks. Default: `false`.
- `private` (Boolean) — If `true`, the bucket has no public read access. Default: `false`.

### Computed

- `bucket_id` (Number) — The API-assigned bucket ID.
- `ready` (Boolean) — Whether the bucket is provisioned and ready.
- `assigned_on` (String) — ISO 8601 timestamp of when the bucket was provisioned.
- `location_name` (String) — Full location name.
- `total_capacity_gb` (Number) — Currently provisioned capacity in GB.
- `auto_scaling` (Boolean) — Current auto-scaling state.
- `endpoints` (List of String) — S3-compatible HTTPS endpoint URLs.
- `access_key` (String, Sensitive) — S3 access key ID.
- `secret_key` (String, Sensitive) — S3 secret access key.
- `user_key` (String, Sensitive) — S3 user key.

## Import

```
terraform import netactuate_storage_bucket.assets <bucket_id>
```
