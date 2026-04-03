# netactuate_nke_cluster

Manages a NetActuate Kubernetes Engine (NKE) cluster. NKE provides managed Kubernetes with autoscaling, high availability control planes, and optional addons.

## Example Usage

### Single-node cluster

```hcl
data "netactuate_nke_versions" "available" {}

resource "netactuate_nke_cluster" "app" {
  name          = "prod-cluster"
  version       = "1.33.4"
  location      = "SJC"
  plan          = "VR1x1x25"
  contract_id   = 301
  minimum_nodes = 1
  maximum_nodes = 1
}

output "kubeconfig_cluster_id" {
  value = netactuate_nke_cluster.app.cluster_id
}

output "api_url" {
  value = netactuate_nke_cluster.app.api_url
}
```

### HA cluster with autoscaling and dashboard

```hcl
resource "netactuate_nke_cluster" "ha" {
  name                 = "ha-cluster"
  version              = "1.33.4"
  location             = "SJC"
  plan                 = "VR2x2x25"
  contract_id          = 301
  replicas             = 3
  minimum_nodes        = 2
  maximum_nodes        = 10
  do_autoscaling       = true
  kubernetes_dashboard = true
  do_dual_stack        = false
}
```

### Fetch kubeconfig after creation

```hcl
data "netactuate_nke_kubeconfig" "app" {
  cluster_id         = netactuate_nke_cluster.app.cluster_id
  expiration_seconds = 86400
}

output "kubeconfig" {
  value     = data.netactuate_nke_kubeconfig.app.kubeconfig
  sensitive = true
}
```

## Argument Reference

### Required

- `name` (String) — The cluster name.
- `version` (String) — Kubernetes version. Must be a value returned by `data.netactuate_nke_versions`. The provider validates this at plan time.
- `plan` (String) — Worker node plan name, e.g. `"VR1x1x25"`.
- `contract_id` (Number) — NKE billing contract ID. Must be a usage-type contract. Forces recreation.
- `minimum_nodes` (Number) — Minimum number of worker nodes (autoscaling lower bound).
- `maximum_nodes` (Number) — Maximum number of worker nodes (autoscaling upper bound).

### Required (one of `location`/`location_id`)

- `location` (String) — Location name, e.g. `"SJC"`. Forces recreation.
- `location_id` (Number) — Numeric location ID. Forces recreation.

### Optional

- `replicas` (Number) — Number of control plane replicas. Default: `1`. Set to `3` for high availability. Forces recreation.
- `do_autoscaling` (Boolean) — Enable node autoscaling. Default: `false`.
- `kubernetes_dashboard` (Boolean) — Install the Kubernetes dashboard addon. Default: `false`. Forces recreation.
- `do_dual_stack` (Boolean) — Enable IPv4+IPv6 dual-stack networking. Default: `false`. Forces recreation.
- `tag_ids` (Set of Number) — Tag IDs to assign to the cluster. Note: the API does not return tag assignments so drift from out-of-band changes may not be detected.

### Computed

- `cluster_id` (Number) — The API-assigned cluster ID.
- `package_id` (Number) — The resolved package ID for the worker node plan.
- `high_availability` (Boolean) — `true` when `replicas > 1`.
- `status` (String) — Cluster status, e.g. `"Healthy"`, `"Initializing"`.
- `api_url` (String) — Kubernetes API server endpoint.
- `prometheus_url` (String) — Prometheus metrics endpoint.
- `kubernetes_dashboard_url` (String) — Dashboard URL (empty if not installed).
- `pod_network` (String) — Pod network CIDR.
- `service_network` (String) — Service network CIDR.

## Import

```
terraform import netactuate_nke_cluster.app <cluster_id>
```

## Notes

- The provider waits up to 15 minutes for a new cluster to reach `Healthy` status.
- `version` is validated against the live list of available versions at plan time. Use `data.netactuate_nke_versions` to discover valid values.
- Upgrading `version` is an in-place update (no recreation). Downgrading is not supported.
