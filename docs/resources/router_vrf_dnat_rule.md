# netactuate_router_vrf_dnat_rule

Manages a Destination NAT (DNAT) rule on a VRF. DNAT rules translate the destination address of inbound traffic — typically used to forward traffic from a public IP/port to a private host.

## Example Usage

### Port forward HTTP to internal server

```hcl
resource "netactuate_router_vrf_dnat_rule" "http_forward" {
  depends_on = [netactuate_router_vrf_interface.dummy]

  router_id             = netactuate_router.main.id
  vrf_id                = netactuate_router.main.default_vrf_id
  ip_version            = 4
  protocol              = "TCP"
  description           = "Forward port 80 to internal web server"
  match_interface_id    = netactuate_router_vrf_interface.dummy.interface_id
  match_network         = "0.0.0.0/0"
  match_port_start      = 80
  translation_network   = "192.168.10.5/32"
  translation_port_start = 80
  priority_location     = "end"
}
```

### Single-port forward (no port range)

```hcl
resource "netactuate_router_vrf_dnat_rule" "ssh_forward" {
  router_id             = netactuate_router.main.id
  vrf_id                = netactuate_router.main.default_vrf_id
  ip_version            = 4
  protocol              = "TCP"
  description           = "Forward external port 2222 to internal SSH"
  match_interface_id    = netactuate_router_vrf_interface.dummy.interface_id
  match_network         = "0.0.0.0/0"
  match_port_start      = 2222
  translation_network   = "192.168.10.10/32"
  translation_port_start = 22
  priority_location     = "end"
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.
- `vrf_id` (Number) — The ID of the VRF. Forces recreation.
- `ip_version` (Number) — IP version for the rule: `4` or `6`.
- `protocol` (String) — Protocol to match: `"TCP"`, `"UDP"`, or `"ICMP"`.
- `match_interface_id` (Number) — The ingress interface on which to match traffic.
- `match_network` (String) — Source network to match in CIDR notation. Use `"0.0.0.0/0"` to match any source.
- `translation_network` (String) — The translated (internal) destination network in CIDR notation.

### Optional

- `description` (String) — A human-readable description of the rule.
- `match_port_start` (Number) — Start of destination port range to match. Valid range: 1–32000. When set, `match_port_end` must equal `match_port_start` or be omitted for a single-port match.
- `match_port_end` (Number) — End of destination port range to match. Valid range: 1–32000.
- `translation_port_start` (Number) — Start of the translated port range. Valid range: 1–32000.
- `translation_port_end` (Number) — End of the translated port range. Valid range: 1–32000.
- `priority_location` (String) — Placement of the rule: `"start"`, `"end"`, or `"after"`.
- `priority_after_dnat_rule_id` (Number) — Insert this rule after the given DNAT rule ID. Used when `priority_location = "after"`.

### Computed

- `dnat_rule_id` (Number) — The API-assigned DNAT rule ID.

## Notes

- Port values above 32000 are rejected by the API.
- For a single-port forward, set only `match_port_start` (omit `match_port_end`). The API rejects rules where `match_port_start == match_port_end`.
- `depends_on` the interface resource is recommended to ensure the interface exists before the rule references its ID.

## Import

DNAT rules do not support `terraform import`. Use `terraform state` to manage existing rules.
