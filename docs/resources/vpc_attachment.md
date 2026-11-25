---
page_title: "netactuate_vpc_attachment Resource - netactuate"
subcategory: ""
description: |-
  Connects a VPC to a customer router or Cloud Router.
---

# netactuate_vpc_attachment (Resource)

Connects a VPC to a customer router or Cloud Router.

Requires vapi3 permission `vpc.attachment.manage` for writes, `vpc.read` for reads. vapi3 flag: `enable-vpc-attachment-cloud-router` for cloud-router only. Customer-router is unflagged.

All attributes except `accept` and `advertise` require replacement when changed. Prefix changes use the prefixes API in place, adding prefixes before removing old ones unless the 32-prefix limit requires removing an obsolete prefix first. Reads obtain active prefixes from the attachment GET route. Create and update wait for `ready`. Delete waits until GET returns 404. The server validates prefix overlap and router location.

## Example Usage

```terraform
resource "netactuate_vpc_attachment" "example" {
  vpc_id      = 12345
  type        = "customer-router"
  address     = "10.0.0.2"
  description = "customer router"
  accept      = ["192.0.2.0/24"]
  advertise   = ["10.0.0.0/24"]
}
```

## Schema

### Required

- `vpc_id` (Number) Parent VPC ID.
- `type` (String) `customer-router` or `cloud-router`.
- `accept` (Set of String) One to 32 accepted CIDR prefixes.

### Optional

- `advertise` (Set of String) Zero to 32 advertised CIDR prefixes.
- `address` (String) Customer router address. Computed when allocated by the API.
- `cloud_router_id` (Number) Cloud Router ID, sent as `routerId`. Required by the API for cloud-router; forbidden for customer-router.
- `vrf_id` (Number) Cloud Router VRF, sent as `vrfId`.
- `description` (String) Description, at most 255 characters.

### Read-Only

- `id` (String) Resource import ID.
- `state` (String) Attachment state.
- `state_message` (String) Attachment state detail.

### Timeouts

The `timeouts` block accepts `create`, `update`, and `delete`, each defaulting to `30m`. Operations that poll use the corresponding configured timeout. Reservation and allow-list requests complete synchronously. Private network deletion only removes state.

## Import

Import ID: `<vpcId>/<attachmentId>`.

```shell
terraform import netactuate_vpc_attachment.example 12345/67890
```
