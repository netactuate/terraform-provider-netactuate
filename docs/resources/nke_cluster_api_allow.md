---
page_title: "netactuate_nke_cluster_api_allow Resource - netactuate"
subcategory: ""
description: |-
  Manages a customer source on an NKE cluster API allow-list.
---

# netactuate_nke_cluster_api_allow (Resource)

Manages a customer source on an NKE cluster API allow-list.

Requires vapi3 permission `nke.cluster.manage` for writes, `nke.cluster.read` for reads. vapi3 flag: `enable-nke-api-access`.

All configured attributes require replacement when changed. Platform entries cannot be deleted. With no customer sources the public API accepts any source address. An address supplied without a prefix is returned as a /32.

## Example Usage

```terraform
resource "netactuate_nke_cluster_api_allow" "example" {
  cluster_id = 12345
  cidr       = "192.0.2.0/24"
  label      = "office"
}
```

## Schema

### Required

- `cluster_id` (Number) Parent cluster ID.
- `cidr` (String) IPv4 address or CIDR. Sent as API `network`.

### Optional

- `label` (String) Optional description, at most 255 characters. Sent as API `description`.

### Read-Only

- `id` (String) Resource import ID.

### Timeouts

The `timeouts` block accepts `create`, `update`, and `delete`, each defaulting to `30m`. Operations that poll use the corresponding configured timeout. Reservation and allow-list requests complete synchronously. Private network deletion only removes state.

## Import

Import ID: `<clusterId>/<allowListEntryId>`.

```shell
terraform import netactuate_nke_cluster_api_allow.example 12345/customer-67890
```
