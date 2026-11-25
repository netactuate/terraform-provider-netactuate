---
page_title: "netactuate_security_group_member Resource - netactuate"
subcategory: ""
description: |-
  Adds a server or node pool to a security group.
---

# netactuate_security_group_member (Resource)

Adds a server or node pool to a security group.

Requires vapi3 permission `cloud.security-group.manage` for writes and `cloud.security-group.read` for reads. vapi3 flag: `enable-cloud-security-groups`.

Every argument requires replacement when changed. Removing a member does not detach its server from the VPC.

Enforcement applies to IPv4. While it is on, IPv6 traffic to servers in this VPC is blocked.

## Example Usage

```terraform
resource "netactuate_security_group_member" "example" {
  security_group_id = 12345
  vpc_id = 23456
  mbpkgid = 34567
}
```

## Schema

### Required

- `security_group_id` (Number) Parent group ID.
- `vpc_id` (Number) VPC ID.

### Optional

- `mbpkgid` (Number) Server package ID. Exactly one of mbpkgid or node_pool_id.
- `node_pool_id` (Number) Node pool ID. Exactly one of mbpkgid or node_pool_id.

### Read-Only

- `enforcement_state` (String) Enforcement state reported by the server.
- `id` (String) Resource import ID.

### Timeouts

The `timeouts` block accepts `create`, `update`, and `delete`, defaulting to `30m`. Each wait uses the corresponding configured timeout. Synchronous requests do not poll.

## Import

Import ID: `<securityGroupId>/<memberId>`.

```shell
terraform import netactuate_security_group_member.example 12345/67890
```
