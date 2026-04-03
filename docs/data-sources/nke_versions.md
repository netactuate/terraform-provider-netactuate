# netactuate_nke_versions (Data Source)

Returns the list of Kubernetes versions currently available for NKE cluster creation. Use this to discover valid values for the `version` argument of `netactuate_nke_cluster`.

## Example Usage

```hcl
data "netactuate_nke_versions" "available" {}

output "available_k8s_versions" {
  value = data.netactuate_nke_versions.available.versions
}

# Use the latest version automatically
locals {
  latest_k8s_version = data.netactuate_nke_versions.available.versions[0]
}

resource "netactuate_nke_cluster" "app" {
  name        = "prod"
  version     = local.latest_k8s_version
  location    = "SJC"
  plan        = "VR1x1x25"
  contract_id = 301
  minimum_nodes = 1
  maximum_nodes = 1
}
```

## Attribute Reference

- `versions` (List of String) — Available Kubernetes version strings in descending order, e.g. `["1.35.1", "1.34.3", "1.33.4"]`.
