---
page_title: "netactuate_nke_cluster_private_network Resource - netactuate"
subcategory: ""
description: |-
  Adds a VPC private network to an existing one node or larger Edge cluster.
---

# netactuate_nke_cluster_private_network (Resource)

Adds a VPC private network to an existing one node or larger Edge cluster.

Requires vapi3 permission `nke.cluster.manage` for create, `nke.cluster.read` for reads. vapi3 flag: `enable-nke-edge-retrofit`.

Create waits for `applied` or `roll_required`. The latter produces a warning and records the node IDs; roll those workers separately. The API refuses clusters already in a VPC or already retrofitted. Destroy removes Terraform state only and emits a warning: the API has no detach operation. Replacing this resource does not detach the old network, so a new create against that cluster will be refused. Import an existing attachment to manage it.

## Example Usage

```terraform
resource "netactuate_nke_cluster_private_network" "example" {
  cluster_id = 12345
  vpc_id     = 67890
}
```

## Schema

### Required

- `cluster_id` (Number) Existing Edge cluster ID. Changes replace the resource.
- `vpc_id` (Number) VPC in the cluster location. Changes replace the resource.

### Read-Only

- `id` (String) Resource import ID.
- `state` (String) `in_progress`, `applied`, or `roll_required`.
- `nodes_requiring_roll` (List of Number) Worker node IDs requiring a roll.

### Timeouts

The `timeouts` block accepts `create`, `update`, and `delete`, each defaulting to `30m`. Operations that poll use the corresponding configured timeout. Reservation and allow-list requests complete synchronously. Private network deletion only removes state.

## Import

Import ID: `<clusterId>`.

```shell
terraform import netactuate_nke_cluster_private_network.example 12345
```
