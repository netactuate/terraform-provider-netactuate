---
page_title: "netactuate_vpc_server_attachment Resource - netactuate"
subcategory: ""
description: |-
  Attaches an existing server to a VPC subnet.
---

# netactuate_vpc_server_attachment (Resource)

Attaches an existing server to a VPC subnet.

Requires vapi3 permission `vpc.manage` for writes and `vpc.read` for reads. vapi3 flag: `enable-vpc-server-attach`.

All arguments require replacement when changed. The server must be in the same location, installed, not built in a VPC and not NKE managed. The server is not restarted. Destroy detaches the server. Detaching a native member returns the server error text.

## Example Usage

```terraform
resource "netactuate_vpc_server_attachment" "example" {
  vpc_id = 12345
  mbpkgid = 67890
  subnet_id = 0
}
```

## Schema

### Required

- `vpc_id` (Number) VPC ID.
- `mbpkgid` (Number) Server package ID.

### Optional

- `subnet_id` (Number) Defaults to 0, the default subnet.

### Read-Only

- `nic_id` (Number) Attached NIC ID, or zero for a native member.
- `state` (String) Attachment state.
- `address` (String) Private address reported by the API.
- `membership` (String) native or attached.
- `id` (String) Resource import ID.

### Timeouts

The `timeouts` block accepts `create`, `update`, and `delete`, defaulting to `30m`. Each wait uses the corresponding configured timeout. Synchronous requests do not poll.

## Import

Import ID: `<vpcId>/<mbpkgid>`.

```shell
terraform import netactuate_vpc_server_attachment.example 12345/67890
```
