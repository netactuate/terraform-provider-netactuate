# netactuate_vpc_gateway_firewall_rule

Manages a firewall rule on the VPC gateway. Controls which traffic is permitted or denied at the VPC boundary. Requires the relevant firewall direction to be enabled on the parent `netactuate_vpc`.

## Example Usage

### Allow inbound SSH from a specific CIDR

```hcl
resource "netactuate_vpc" "main" {
  label                         = "prod-vpc"
  description                   = "Production"
  location                      = "SJC"
  firewall_inbound_ipv4_enabled = true
}

resource "netactuate_vpc_gateway_firewall_rule" "allow_ssh" {
  vpc_id      = netactuate_vpc.main.vpc_id
  ip_version  = 4
  direction   = "inbound"
  protocol    = "TCP"
  description = "Allow SSH from office"
  network     = "203.0.113.0/24"
  port_start  = 22
  port_end    = 22
}

resource "netactuate_vpc_gateway_firewall_rule" "allow_https" {
  vpc_id      = netactuate_vpc.main.vpc_id
  ip_version  = 4
  direction   = "inbound"
  protocol    = "TCP"
  description = "Allow HTTPS from anywhere"
  port_start  = 443
  port_end    = 443
}
```

## Argument Reference

### Required

- `vpc_id` (Number) — The ID of the parent VPC. Forces recreation.
- `ip_version` (Number) — IP version: `4` or `6`. Forces recreation.
- `direction` (String) — Traffic direction: `"inbound"` or `"outbound"`.

### Optional

- `protocol` (String) — Protocol: `"TCP"`, `"UDP"`, or `"ICMP"`.
- `description` (String) — Description of the rule.
- `network` (String) — Source/destination network CIDR to match, e.g. `"203.0.113.0/24"`.
- `port_start` (Number) — Start of the port range.
- `port_end` (Number) — End of the port range.

### Computed

- `rule_id` (Number) — The API-assigned firewall rule ID.

## Import

```
terraform import netactuate_vpc_gateway_firewall_rule.allow_ssh <vpc_id>/<rule_id>
```
