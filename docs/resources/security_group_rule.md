---
page_title: "netactuate_security_group_rule Resource - netactuate"
subcategory: ""
description: |-
  Creates an immutable inbound security group rule.
---

# netactuate_security_group_rule (Resource)

Creates an immutable inbound security group rule.

Requires vapi3 permission `cloud.security-group.manage` for writes and `cloud.security-group.read` for reads. vapi3 flag: `enable-cloud-security-groups`.

Every argument requires replacement when changed. Rules have no update API. A group source is sent as securityGroupId in the request body.

## Example Usage

```terraform
resource "netactuate_security_group_rule" "example" {
  security_group_id = 12345
  ip_version = 4
  protocol = "tcp"
  port_start = 443
  port_end = 443
  network = "192.0.2.0/24"
}
```

## Schema

### Required

- `security_group_id` (Number) Parent group ID.
- `ip_version` (Number) 4 or 6.

### Optional

- `direction` (String) Defaults to inbound, the only supported direction.
- `protocol` (String) Protocol validated by the server.
- `port_start` (Number) First port, 1 to 65535. Requires port_end.
- `port_end` (Number) Last port, 1 to 65535. Requires port_start.
- `network` (String) Source CIDR. Exactly one of network or source_security_group_id.
- `source_security_group_id` (Number) Source group ID. Exactly one of network or source_security_group_id.
- `description` (String) Rule description.

### Read-Only

- `id` (String) Resource import ID.

### Timeouts

The `timeouts` block accepts `create`, `update`, and `delete`, defaulting to `30m`. Each wait uses the corresponding configured timeout. Synchronous requests do not poll.

## Import

Import ID: `<securityGroupId>/<ruleId>`.

```shell
terraform import netactuate_security_group_rule.example 12345/67890
```
