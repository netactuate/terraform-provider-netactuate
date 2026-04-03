# netactuate_vpc_gateway_dnat_rule

Manages a Destination NAT rule on the VPC gateway. Forwards inbound traffic from a public address/port to a private host inside the VPC.

## Example Usage

### Forward port 443 to internal HTTPS server

```hcl
resource "netactuate_vpc_gateway_dnat_rule" "https" {
  vpc_id               = netactuate_vpc.main.vpc_id
  ip_version           = 4
  protocol             = "TCP"
  description          = "Forward HTTPS to internal server"
  match_port_start     = 443
  match_port_end       = 443
  translation_address  = "10.0.0.10"
  translation_port_start = 443
  translation_port_end   = 443
  priority_location    = "end"
}
```

### Forward a port range

```hcl
resource "netactuate_vpc_gateway_dnat_rule" "game_ports" {
  vpc_id               = netactuate_vpc.main.vpc_id
  ip_version           = 4
  protocol             = "UDP"
  description          = "Game server port range"
  match_port_start     = 27015
  match_port_end       = 27020
  translation_address  = "10.0.0.20"
  translation_port_start = 27015
  translation_port_end   = 27020
  priority_location    = "end"
}
```

## Argument Reference

### Required

- `vpc_id` (Number) — The ID of the parent VPC. Forces recreation.
- `ip_version` (Number) — IP version: `4` or `6`. Forces recreation.
- `translation_address` (String) — The private IP inside the VPC to forward traffic to.

### Optional

- `protocol` (String) — Protocol: `"TCP"`, `"UDP"`, or `"ICMP"`.
- `description` (String) — Description of the rule.
- `match_address` (String) — Public IP to match on inbound traffic.
- `match_port_start` (Number) — Start of the match port range.
- `match_port_end` (Number) — End of the match port range.
- `translation_port_start` (Number) — Start of the translated port range.
- `translation_port_end` (Number) — End of the translated port range.
- `priority_location` (String) — Rule placement: `"start"` or `"end"`.
- `priority_after_rule_id` (Number) — Insert after this DNAT rule ID.

### Computed

- `rule_id` (Number) — The API-assigned DNAT rule ID.

## Import

```
terraform import netactuate_vpc_gateway_dnat_rule.https <vpc_id>/<rule_id>
```
