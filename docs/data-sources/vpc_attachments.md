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

The computed `bgp` block is empty when BGP is disabled. Otherwise it contains `peer_asn`, `max_prefix`, `has_password`, `gateway_asn`, `gateway_address`, `peer_address`, `state`, `state_message` and `applied_on`. No password is returned. BGP requires vapi3 flag `enable-nke-bgp` and the account list `enable-nke-bgp-accounts`; reading attachments requires `vpc.read`.
