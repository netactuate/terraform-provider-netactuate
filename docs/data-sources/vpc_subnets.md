---
page_title: "netactuate_vpc_subnets Data Source - netactuate"
subcategory: ""
description: |-
  Lists VPC subnets, including the default subnet with subnet_id 0.
---

# netactuate_vpc_subnets (Data Source)

Lists VPC subnets, including the default subnet with subnet_id 0.

Requires vapi3 permission `vpc.read`. vapi3 flag: `enable-vpc-subnets`.

## Example Usage

```terraform
data "netactuate_vpc_subnets" "example" {
  vpc_id = 12345
}
```

## Schema

### Required

- `vpc_id` (Number) VPC ID.

### Read-Only

- `id` (String) Data source ID.
- `subnets` (List of Object) Rows with: `subnet_id`, `network`, `prefix_length`, `label`, `is_default`, `state`, `state_message`, `created_on`, `ready_on`, `delete_requested_on`, `deleted_on`.

GET still lists the default subnet when the flag is off.
