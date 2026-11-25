---
page_title: "netactuate_nke_cluster_api_allow_list Data Source - netactuate"
subcategory: ""
description: |-
  Lists customer and platform API allow-list entries.
---

# netactuate_nke_cluster_api_allow_list (Data Source)

Lists customer and platform API allow-list entries.

Requires vapi3 permission `nke.cluster.read`. vapi3 flag: `enable-nke-api-access`.

## Example Usage

```terraform
data "netactuate_nke_cluster_api_allow_list" "example" {
  cluster_id = 12345
}
```

## Schema

### Required

- `cluster_id` (Number) Parent ID.

### Read-Only

- `id` (String) Parent ID.
- `entries` (List of Object) Each row exposes `allow_list_entry_id`, `network`, `description`, `is_platform`, `owner`, and `created_on`. Platform rows have `is_platform = true` and cannot be deleted.

Data sources are read at refresh and are not imported.
