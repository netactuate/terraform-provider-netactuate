---
page_title: "netactuate_vpc_attachments Data Source - netactuate"
subcategory: ""
description: |-
  Lists VPC attachments and their active prefixes.
---

# netactuate_vpc_attachments (Data Source)

Lists VPC attachments and their active prefixes.

Requires vapi3 permission `vpc.read`. vapi3 flag: `enable-vpc-attachment-cloud-router` for cloud-router creation only; customer-router reads are unflagged.

## Example Usage

```terraform
data "netactuate_vpc_attachments" "example" {
  vpc_id = 12345
}
```

## Schema

### Required

- `vpc_id` (Number) Parent ID.

### Read-Only

- `id` (String) Parent ID.
- `attachments` (List of Object) Each row exposes `attachment_id`, `type`, `address`, `cloud_router_id`, `vrf_id`, `description`, `accept`, `advertise`, `state`, and `state_message`. Prefix sets come from the individual attachment GET route.

Data sources are read at refresh and are not imported.
