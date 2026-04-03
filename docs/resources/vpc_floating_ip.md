# netactuate_vpc_floating_ip

Manages a floating IP address associated with a VPC. Floating IPs are additional public addresses that can be assigned to VMs inside the VPC.

## Example Usage

```hcl
resource "netactuate_vpc" "main" {
  label       = "prod-vpc"
  description = "Production VPC"
  location    = "SJC"
}

resource "netactuate_vpc_floating_ip" "web" {
  vpc_id     = netactuate_vpc.main.vpc_id
  ip_version = 4
  ptr        = "web.example.com"
}

output "floating_ip_address" {
  value = netactuate_vpc_floating_ip.web.address
}
```

## Argument Reference

### Required

- `vpc_id` (Number) — The ID of the parent VPC. Forces recreation.
- `ip_version` (Number) — IP version: `4` or `6`. Forces recreation.

### Optional

- `ptr` (String) — Reverse DNS (PTR) record for the floating IP.

### Computed

- `floating_ip_id` (Number) — The API-assigned floating IP ID.
- `address` (String) — The assigned public IP address.
- `is_primary` (Boolean) — Whether this is the primary floating IP for its IP version on the VPC.

## Import

```
terraform import netactuate_vpc_floating_ip.web <vpc_id>/<floating_ip_id>
```
