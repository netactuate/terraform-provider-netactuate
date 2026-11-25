---
page_title: "netactuate_security_groups Data Source - netactuate"
subcategory: ""
description: |-
  Lists account security groups.
---

# netactuate_security_groups (Data Source)

Lists account security groups.

Requires vapi3 permission `cloud.security-group.read`. vapi3 flag: `enable-cloud-security-groups`.

## Example Usage

```terraform
data "netactuate_security_groups" "example" {
}
```

## Schema

### Read-Only

- `id` (String) Data source ID.
- `security_groups` (List of Object) Rows with: `security_group_id`, `name`, `description`, `is_default`, `default_for_vpc_id`, `member_count`, `created_on`, `modified_on`.
