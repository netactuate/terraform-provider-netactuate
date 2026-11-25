---
page_title: "netactuate_security_group Resource - netactuate"
subcategory: ""
description: |-
  Creates an account security group.
---

# netactuate_security_group (Resource)

Creates an account security group.

Requires vapi3 permission `cloud.security-group.manage` for writes and `cloud.security-group.read` for reads. vapi3 flag: `enable-cloud-security-groups`.

Default groups are read only. Delete is refused while a group has members or is referenced by a rule.

## Example Usage

```terraform
resource "netactuate_security_group" "example" {
  name = "web"
  description = "Web tier"
}
```

## Schema

### Required

- `name` (String) Group name, 1 to 64 characters. Updated in place.

### Optional

- `description` (String) Description, up to 255 characters. Updated in place.

### Read-Only

- `security_group_id` (Number) Group ID.
- `is_default` (Boolean) Whether this is a VPC default group.
- `member_count` (Number) Current member count.
- `id` (String) Resource import ID.

### Timeouts

The `timeouts` block accepts `create`, `update`, and `delete`, defaulting to `30m`. Each wait uses the corresponding configured timeout. Synchronous requests do not poll.

## Import

Import ID: `<securityGroupId>`.

```shell
terraform import netactuate_security_group.example 12345
```
