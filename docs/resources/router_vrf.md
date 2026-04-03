# netactuate_router_vrf

Manages an additional VRF (Virtual Routing and Forwarding instance) on a cloud router. Each router has a default VRF (its `default_vrf_id`); this resource creates additional isolated routing tables.

> **Note:** Routers enrolled in a `netactuate_magic_mesh_router` can only use the default VRF and cannot have additional VRFs created via this resource.

## Example Usage

```hcl
resource "netactuate_router" "main" {
  name     = "prod-router"
  location = "SJC"
  plan     = "VR1x1x25"
}

resource "netactuate_router_vrf" "tenant_a" {
  router_id   = netactuate_router.main.id
  name        = "tenant-a"
  description = "Isolated VRF for tenant A"
}

output "tenant_a_vrf_id" {
  value = netactuate_router_vrf.tenant_a.vrf_id
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.

### Optional

- `name` (String) — A name for the VRF.
- `description` (String) — A description for the VRF.

### Computed

- `vrf_id` (Number) — The API-assigned VRF ID. Used as `vrf_id` in all VRF sub-resources.

## Import

```
terraform import netactuate_router_vrf.tenant_a <router_id>/<vrf_id>
```
