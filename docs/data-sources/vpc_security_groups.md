---
page_title: "netactuate_vpc_security_groups Data Source - netactuate"
subcategory: ""
description: |-
  Lists groups with live members in the VPC plus its default group.
---

# netactuate_vpc_security_groups (Data Source)

Lists groups with live members in the VPC plus its default group.

Requires vapi3 permission `vpc.read`. vapi3 flag: `enable-cloud-security-groups`.

## Example Usage

```terraform
data "netactuate_vpc_security_groups" "example" {
  vpc_id = 12345
}
```

## Schema

### Required

- `vpc_id` (Number) VPC ID.

### Read-Only

- `id` (String) Data source ID.
- `security_groups` (List of Object) Rows with: `security_group_id`, `name`, `description`, `is_default`, `default_for_vpc_id`, `member_count`, `created_on`, `modified_on`.
