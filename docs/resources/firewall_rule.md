# netactuate_firewall_rule

Manages an individual firewall rule within a `netactuate_firewall_set`. Rules can match on IP version, protocol, source/destination CIDRs, and port ranges.

## Example Usage

### Allow HTTPS from anywhere

```hcl
resource "netactuate_firewall_rule" "allow_https" {
  firewall_set_id        = netactuate_firewall_set.web.id
  ip_version             = "IPv4"
  action                 = "ACCEPT"
  protocol               = "tcp"
  destination_port_start = 443
  destination_port_end   = 443
  enabled                = true
  admin_comment          = "Allow HTTPS inbound"
}
```

### Allow SSH from a specific source

```hcl
resource "netactuate_firewall_rule" "allow_ssh" {
  firewall_set_id        = netactuate_firewall_set.web.id
  ip_version             = "IPv4"
  action                 = "ACCEPT"
  protocol               = "tcp"
  source_net             = ["203.0.113.0/24"]
  destination_port_start = 22
  destination_port_end   = 22
  enabled                = true
  admin_comment          = "Office SSH access"
}
```

### Drop ICMP

```hcl
resource "netactuate_firewall_rule" "drop_icmp" {
  firewall_set_id = netactuate_firewall_set.web.id
  ip_version      = "IPv4"
  action          = "DROP"
  protocol        = "icmp"
  icmp_type       = "echo-request"
  enabled         = true
}
```

## Argument Reference

### Required

- `firewall_set_id` (String) — The ID of the parent firewall set. Forces recreation.
- `ip_version` (String) — IP version: `"IPv4"` or `"IPv6"`.
- `action` (String) — Rule action: `"ACCEPT"` or `"DROP"`.
- `enabled` (Boolean) — Whether this rule is active.

### Optional

- `direction` (String) — Traffic direction. Always `"IN"` per backend.
- `protocol` (String) — Protocol: `"tcp"`, `"udp"`, or `"icmp"`.
- `icmp_type` (String) — ICMP type when `protocol = "icmp"`: `"echo-request"` or `"echo-reply"`.
- `source_port_start` (Number) — Start of source port range. Valid range: 0–65535.
- `source_port_end` (Number) — End of source port range. Valid range: 0–65535.
- `destination_port_start` (Number) — Start of destination port range. Valid range: 0–65535.
- `destination_port_end` (Number) — End of destination port range. Valid range: 0–65535.
- `source_net` (List of String) — Source CIDRs to match.
- `destination_net` (List of String) — Destination CIDRs to match.
- `admin_comment` (String) — Comment describing the rule's purpose.
- `rule_priority` (Number) — Priority order. Auto-assigned if not set.
- `sync_after_publish` (Boolean) — Sync rules to attached VMs after publishing.

### Computed

- `rule_id` (String) — The API-assigned rule ID.

## Import

```
terraform import netactuate_firewall_rule.allow_https <firewall_set_id>/<rule_id>
```
