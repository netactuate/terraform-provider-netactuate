# netactuate_magic_mesh

Manages a NetActuate Magic Mesh — a managed overlay network that automatically interconnects cloud routers across multiple locations with BGP-based full-mesh routing. Routers are added to the mesh with `netactuate_magic_mesh_router`.

## Example Usage

```hcl
resource "netactuate_magic_mesh" "global" {
  name        = "global-backbone"
  description = "Multi-region router mesh"
}

# Routers in SJC and AMS
resource "netactuate_router" "sjc" {
  name     = "sjc-router"
  location = "SJC"
  plan     = "VR1x1x25"
}

resource "netactuate_router" "ams" {
  name     = "ams-router"
  location = "AMS"
  plan     = "VR1x1x25"
}

resource "netactuate_magic_mesh_router" "sjc" {
  mesh_id   = netactuate_magic_mesh.global.mesh_id
  router_id = netactuate_router.sjc.id
}

resource "netactuate_magic_mesh_router" "ams" {
  mesh_id   = netactuate_magic_mesh.global.mesh_id
  router_id = netactuate_router.ams.id
}

output "mesh_id" {
  value = netactuate_magic_mesh.global.mesh_id
}
```

## Argument Reference

### Required

- `name` (String) — A name for the Magic Mesh.

### Optional

- `description` (String) — A description for the Magic Mesh.

### Computed

- `mesh_id` (Number) — The API-assigned mesh ID.

## Import

```
terraform import netactuate_magic_mesh.global <mesh_id>
```
