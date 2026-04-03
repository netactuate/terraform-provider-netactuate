# netactuate_nke_kubeconfig (Data Source)

Fetches a kubeconfig for an NKE cluster. The kubeconfig contains a time-limited token for authenticating to the Kubernetes API.

> **Security note:** The `kubeconfig` output contains credentials. Always mark it `sensitive = true` and avoid storing it in plaintext state backends.

## Example Usage

```hcl
resource "netactuate_nke_cluster" "app" {
  name          = "prod"
  version       = "1.33.4"
  location      = "SJC"
  plan          = "VR1x1x25"
  contract_id   = 301
  minimum_nodes = 1
  maximum_nodes = 3
}

data "netactuate_nke_kubeconfig" "app" {
  cluster_id         = netactuate_nke_cluster.app.cluster_id
  expiration_seconds = 86400  # 24 hours
}

# Write to a local file for use with kubectl
resource "local_file" "kubeconfig" {
  filename        = "${path.module}/kubeconfig.yaml"
  content         = data.netactuate_nke_kubeconfig.app.kubeconfig
  file_permission = "0600"
}

output "kubeconfig" {
  value     = data.netactuate_nke_kubeconfig.app.kubeconfig
  sensitive = true
}
```

## Argument Reference

- `cluster_id` (Number, required) — The ID of the NKE cluster.
- `expiration_seconds` (Number, optional) — Token lifetime in seconds. Default: `31536000` (1 year). Maximum: `3153600000`.

## Attribute Reference

- `kubeconfig` (String, Sensitive) — The kubeconfig YAML content for use with `kubectl` or Kubernetes providers.
