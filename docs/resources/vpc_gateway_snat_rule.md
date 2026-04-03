# netactuate_vpc_gateway_snat_rule

Manages a Source NAT rule on the VPC gateway. Controls how outbound traffic from inside the VPC is translated when leaving through the bastion.

## Example Usage

### Masquerade a specific subnet out the gateway

```hcl
resource "netactuate_vpc_gateway_snat_rule" "outbound" {
  vpc_id               = netactuate_vpc.main.vpc_id
  ip_version           = 4
  protocol             = "TCP"
  description          = "Masquerade internal subnet outbound"
  match_internal_cidr  = "10.0.0.0/24"
  translation_port_start = 1024
  translation_port_end   = 65535
  priority_location    = "end"
}
```

### Default SNAT (all internal traffic)

```hcl
resource "netactuate_vpc_gateway_snat_rule" "default_snat" {
  vpc_id          = netactuate_vpc.main.vpc_id
  ip_version      = 4
  description     = "Default outbound SNAT"
  priority_location = "end"
}
```

## Argument Reference

### Required

- `vpc_id` (Number) — The ID of the parent VPC. Forces recreation.
- `ip_version` (Number) — IP version: `4` or `6`. Forces recreation.

### Optional

- `protocol` (String) — Protocol to match: `"TCP"`, `"UDP"`, or `"ICMP"`.
- `description` (String) — Human-readable description.
- `match_internal_cidr` (String) — Internal VPC CIDR to match. Defaults to `0.0.0.0/0` or `::/0` (match all).
- `translation_address_start` (String) — Lower bound of the public IP translation range. Defaults to the bastion IP.
- `translation_address_end` (String) — Upper bound of the public IP translation range. Defaults to the bastion IP.
- `translation_port_start` (Number) — Start of the translated port range.
- `translation_port_end` (Number) — End of the translated port range.
- `priority_location` (String) — Rule placement: `"start"` or `"end"`.
- `priority_after_rule_id` (Number) — Insert after this SNAT rule ID.

### Computed

- `rule_id` (Number) — The API-assigned SNAT rule ID.

## Import

```
terraform import netactuate_vpc_gateway_snat_rule.outbound <vpc_id>/<rule_id>
```
