# netactuate_nke_worker_nodes (Data Source)

Returns the list of worker nodes in an NKE cluster. Useful for inventory, monitoring integration, or dynamic configuration that depends on individual node details.

## Example Usage

```hcl
resource "netactuate_nke_cluster" "app" {
  name          = "prod"
  version       = "1.33.4"
  location      = "SJC"
  plan          = "VR1x1x25"
  contract_id   = 301
  minimum_nodes = 2
  maximum_nodes = 5
  do_autoscaling = true
}

data "netactuate_nke_worker_nodes" "app_nodes" {
  cluster_id = netactuate_nke_cluster.app.cluster_id
}

output "worker_node_names" {
  value = [for n in data.netactuate_nke_worker_nodes.app_nodes.worker_nodes : n.name]
}

output "worker_node_count" {
  value = length(data.netactuate_nke_worker_nodes.app_nodes.worker_nodes)
}
```

## Argument Reference

- `cluster_id` (Number, required) — The ID of the NKE cluster.

## Attribute Reference

- `worker_nodes` (List of Object) — List of worker nodes. Each object contains:
  - `worker_node_id` (String) — The worker node ID.
  - `name` (String) — The hostname of the worker node.
  - `status_ready` (Boolean) — Whether the node is in a ready state.
  - `package_id` (Number) — The plan/package ID of the node.
  - `package_name` (String) — The plan/package name of the node.
  - `location_id` (Number) — The location ID of the node.
  - `location_name` (String) — The location name of the node.
