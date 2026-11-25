---
page_title: "netactuate_vpc_servers Data Source - netactuate"
subcategory: ""
description: |-
  Lists native and attached servers in a VPC.
---

# netactuate_vpc_servers (Data Source)

Lists native and attached servers in a VPC.

Requires vapi3 permission `vpc.read`. vapi3 flag: `enable-vpc-server-attach`.

## Example Usage

```terraform
data "netactuate_vpc_servers" "example" {
  vpc_id = 12345
}
```

## Schema

### Required

- `vpc_id` (Number) VPC ID.

### Read-Only

- `id` (String) Data source ID.
- `servers` (List of Object) Rows with: `mbpkgid`, `fqdn`, `membership`, `subnet_id`, `nic_id`, `state`, `address`.

GET still lists native members when the flag is off.
