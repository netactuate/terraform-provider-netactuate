---
page_title: "netactuate_nke_cluster_api_access Resource - netactuate"
subcategory: ""
description: |-
  Manages public and private NKE API endpoint settings.
---

# netactuate_nke_cluster_api_access (Resource)

Manages public and private NKE API endpoint settings.

Requires vapi3 permission `nke.cluster.manage` for writes, `nke.cluster.read` for reads. vapi3 flag: `enable-nke-api-access` and `enable-nke-private-endpoint`.

Create and update PATCH the API and wait until `state` is no longer `applying` and `private_endpoint_state` is no longer `provisioning`. Destroy restores public access and disables the private endpoint.

## Example Usage

```terraform
resource "netactuate_nke_cluster_api_access" "example" {
  cluster_id               = 12345
  public_enabled           = true
  private_endpoint_enabled = true
}
```

## Schema

### Required

- `cluster_id` (Number) Parent cluster ID. Changes replace this resource.

### Optional

- `public_enabled` (Boolean) Enable the public endpoint. Defaults to true.
- `private_endpoint_enabled` (Boolean) Enable the private endpoint. Defaults to false.

### Read-Only

- `id` (String) Resource import ID.
- `private_endpoint_address` (String) Private endpoint IPv4 address.
- `private_dns_name` (String) Published private DNS name.
- `private_endpoint_state` (String) Private endpoint state.
- `state` (String) API access reconciliation state.

### Timeouts

The `timeouts` block accepts `create`, `update`, and `delete`, each defaulting to `30m`. Operations that poll use the corresponding configured timeout. Reservation and allow-list requests complete synchronously. Private network deletion only removes state.

## Import

Import ID: `<clusterId>`.

```shell
terraform import netactuate_nke_cluster_api_access.example 12345
```
