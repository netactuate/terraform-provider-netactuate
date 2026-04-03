# netactuate_storage_locations (Data Source)

Returns the list of locations where NetActuate storage services (buckets, object stores, block namespaces, block volumes) are available.

## Example Usage

```hcl
data "netactuate_storage_locations" "available" {}

output "storage_locations" {
  value = data.netactuate_storage_locations.available.locations
}

# Use a specific location by name lookup
locals {
  sjc_location_id = [
    for l in data.netactuate_storage_locations.available.locations :
    l.id if l.name == "SJC - San Jose, CA"
  ][0]
}
```

## Attribute Reference

- `locations` (List of Object) — Available storage locations. Each object contains:
  - `id` (Number) — The numeric location ID. Pass this to `location_id` on storage resources.
  - `name` (String) — The full location name, e.g. `"SJC - San Jose, CA"`.
