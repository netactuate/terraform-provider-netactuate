---
page_title: "netactuate_vpc_security_group_enforcement Resource - netactuate"
subcategory: ""
description: |-
  Manages security group enforcement for a VPC.
---

# netactuate_vpc_security_group_enforcement (Resource)

Manages security group enforcement for a VPC.

Requires vapi3 permission `cloud.security-group.manage` for writes and `cloud.security-group.read` for reads. vapi3 flag: `enable-cloud-security-group-enforcement and enable-cloud-security-groups`.

Destroy disables enforcement by sending enabled:false. It does not delete the VPC.

Enforcement applies to IPv4. While it is on, IPv6 traffic to servers in this VPC is blocked.

## Example Usage

```terraform
resource "netactuate_vpc_security_group_enforcement" "example" {
  vpc_id = 12345
  enabled = true
}
```

## Schema

### Required

- `vpc_id` (Number) VPC ID. Requires replacement when changed.

### Optional

- `enabled` (Boolean) Defaults to true. Updated in place.

### Read-Only

- `enabled_on` (String) Time enforcement was enabled, or an empty string when disabled.
- `id` (String) Resource import ID.

### Timeouts

The `timeouts` block accepts `create`, `update`, and `delete`, defaulting to `30m`. Each wait uses the corresponding configured timeout. Synchronous requests do not poll.

## Import

Import ID: `<vpcId>`.

```shell
terraform import netactuate_vpc_security_group_enforcement.example 12345
```
