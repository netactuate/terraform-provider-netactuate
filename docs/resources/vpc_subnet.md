---
page_title: "netactuate_vpc_subnet Resource - netactuate"
subcategory: ""
description: |-
  Creates an additional IPv4 subnet in a VPC.
---

# netactuate_vpc_subnet (Resource)

Creates an additional IPv4 subnet in a VPC.

Requires vapi3 permission `vpc.manage` for writes and `vpc.read` for reads. vapi3 flag: `enable-vpc-subnets`.

All arguments require replacement when changed. Subnet 0 is the default and cannot be imported or deleted by this resource. Create waits for readiness when reported. Delete waits until GET returns 404 or deletedOn. Delete is refused while the subnet is in use.

## Example Usage

```terraform
resource "netactuate_vpc_subnet" "example" {
  vpc_id = 12345
  network = "172.31.251.0/24"
  label = "app"
}
```

## Schema

### Required

- `vpc_id` (Number) VPC ID.
- `network` (String) IPv4 CIDR, /16 to /28 in RFC1918 or 100.64.0.0/10. Server validates overlap and capacity.
- `label` (String) Subnet label, 1 to 255 characters.

### Optional


### Read-Only

- `subnet_id` (Number) Subnet ID.
- `prefix_length` (Number) Prefix length.
- `is_default` (Boolean) Whether this is the default subnet.
- `state` (String) Value reported by the server.
- `state_message` (String) Value reported by the server.
- `created_on` (String) Value reported by the server.
- `ready_on` (String) Value reported by the server.
- `delete_requested_on` (String) Value reported by the server.
- `deleted_on` (String) Value reported by the server.
- `id` (String) Resource import ID.

### Timeouts

The `timeouts` block accepts `create`, `update`, and `delete`, defaulting to `30m`. Each wait uses the corresponding configured timeout. Synchronous requests do not poll.

## Import

Import ID: `<vpcId>/<subnetId>`.

```shell
terraform import netactuate_vpc_subnet.example 12345/67890
```
