---
page_title: "netactuate_vpc_ip_reservation Resource - netactuate"
subcategory: ""
description: |-
  Reserves a customer IPv4 address in a VPC.
---

# netactuate_vpc_ip_reservation (Resource)

Reserves a customer IPv4 address in a VPC.

Requires vapi3 permission `vpc.ip.manage` for create and delete, `vpc.read` for reads. vapi3 flag: `enable-vpc-ip-reservations` and `enable-vpc-internal-lb`.

Deletion is refused while `in_use` is true. Release the load balancer first.

## Example Usage

```terraform
resource "netactuate_vpc_ip_reservation" "example" {
  vpc_id      = 12345
  description = "application VIP"
  # Omit address to allocate the next free address.
  address     = "10.0.0.200"
}
```

## Schema

### Required

- `vpc_id` (Number) Parent VPC ID. Changes replace the reservation.
- `description` (String) Required description, 1 to 255 characters. Changes replace the reservation.

### Optional

- `address` (String) Requested IPv4 address, or the allocated address when omitted. Changes replace the reservation.

### Read-Only

- `id` (String) Resource import ID.
- `in_use` (Boolean) Whether a load balancer uses the reservation.
- `reserved_on` (String) Reservation timestamp.

### Timeouts

The `timeouts` block accepts `create`, `update`, and `delete`, each defaulting to `30m`. Operations that poll use the corresponding configured timeout. Reservation and allow-list requests complete synchronously. Private network deletion only removes state.

## Import

Import ID: `<vpcId>/<ipReservationId>`.

```shell
terraform import netactuate_vpc_ip_reservation.example 12345/67890
```
