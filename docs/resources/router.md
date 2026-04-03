# netactuate_router

Manages a NetActuate Cloud Router. A Cloud Router is a virtual routing appliance provisioned in a NetActuate data centre. It is the top-level resource for all routing, NAT, BGP, and VPN configuration.

## Example Usage

```hcl
resource "netactuate_router" "main" {
  name        = "prod-router"
  description = "Production cloud router"
  location    = "SJC"
  plan        = "VR1x1x25"
}

output "router_id" {
  value = netactuate_router.main.id
}

output "router_ip" {
  value = netactuate_router.main.ipv4_address
}
```

## Argument Reference

### Required (one of `plan`/`package_id`)

- `plan` (String) — Plan name for the router, e.g. `"VR1x1x25"`. Resolved to `package_id` on create. Exactly one of `plan` or `package_id` must be set. Forces recreation.
- `package_id` (Number) — Numeric package ID. Exactly one of `plan` or `package_id` must be set. Forces recreation.

### Required (one of `location`/`location_id`)

- `location` (String) — Location short code or full name, e.g. `"SJC"` or `"SJC - San Jose, CA"`. Exactly one of `location` or `location_id` must be set. Forces recreation.
- `location_id` (Number) — Numeric location ID. Exactly one of `location` or `location_id` must be set. Forces recreation.

### Optional

- `name` (String) — A display name for the router.
- `description` (String) — A description for the router.

### Computed

- `router_id` (Number) — The API-assigned router ID.
- `ipv4_address` (String) — The public IPv4 address of the router.
- `has_default_vrf` (Boolean) — Whether the router has a default VRF.
- `default_vrf_id` (Number) — The ID of the default VRF. Used as `vrf_id` in sub-resources when targeting the default routing table.
- `mesh_id` (Number) — Mesh ID if the router is part of a Magic Mesh.
- `status` (String) — Current status, e.g. `"online"`.
- `version` (Number) — Configuration version, incremented on each change.
- `updated_on` (String) — ISO 8601 timestamp of the last configuration change.
- `can_join_magic_mesh` (Boolean) — Whether this router is eligible to join a Magic Mesh.

## Import

Cloud routers are not importable via `terraform import` (no importer block). Use the Terraform state to track existing routers.

## Notes

- The provider waits up to 10 minutes for the router to reach `ready` status after creation.
- `ipv4_address` may transiently appear as a decimal integer immediately after creation. This self-corrects on the next `terraform plan`.
