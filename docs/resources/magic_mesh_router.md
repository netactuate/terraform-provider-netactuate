# netactuate_magic_mesh_router

Adds a cloud router to a `netactuate_magic_mesh`. Once enrolled, the router participates in the Magic Mesh BGP overlay and can exchange routes with all other routers in the mesh.

> **Note:** Routers enrolled in a Magic Mesh can only use their default VRF. Additional VRFs created with `netactuate_router_vrf` are not supported while a router is in a mesh.

## Example Usage

```hcl
resource "netactuate_magic_mesh" "backbone" {
  name = "backbone"
}

resource "netactuate_router" "sjc" {
  name     = "sjc-router"
  location = "SJC"
  plan     = "VR1x1x25"
}

resource "netactuate_router" "lhr" {
  name     = "lhr-router"
  location = "LHR"
  plan     = "VR1x1x25"
}

resource "netactuate_magic_mesh_router" "sjc" {
  mesh_id   = netactuate_magic_mesh.backbone.mesh_id
  router_id = netactuate_router.sjc.id
}

resource "netactuate_magic_mesh_router" "lhr" {
  mesh_id   = netactuate_magic_mesh.backbone.mesh_id
  router_id = netactuate_router.lhr.id
}
```

## Argument Reference

### Required

- `mesh_id` (Number) — The ID of the Magic Mesh. Forces recreation.
- `router_id` (Number) — The ID of the cloud router to enroll. Only routers where `can_join_magic_mesh = true` are eligible. Forces recreation.

## Import

```
terraform import netactuate_magic_mesh_router.sjc <mesh_id>/<router_id>
```
