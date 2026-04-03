# netactuate_router_vrf_snat_rule

Manages a Source NAT (SNAT) rule on a VRF. SNAT rules translate the source address of outbound traffic — typically used to masquerade a private subnet behind a public IP or to allow traffic out via a specific interface.

## Example Usage

### Masquerade all RFC1918 traffic out a dummy interface

```hcl
resource "netactuate_router_vrf_interface" "dummy" {
  router_id = netactuate_router.main.id
  vrf_id    = netactuate_router.main.default_vrf_id
  type      = "dummy"
  name      = "dummy0"
  ipv4_cidr = "192.168.10.1/24"
}

resource "netactuate_router_vrf_snat_rule" "masquerade" {
  depends_on = [netactuate_router_vrf_interface.dummy]

  router_id           = netactuate_router.main.id
  vrf_id              = netactuate_router.main.default_vrf_id
  ip_version          = 4
  protocol            = "TCP"
  description         = "Masquerade RFC1918 out dummy0"
  match_interface_id  = netactuate_router_vrf_interface.dummy.interface_id
  match_network       = "192.168.10.0/24"
  match_port_start    = 1024
  match_port_end      = 32000
  translation_network      = "0.0.0.0/0"
  translation_port_start   = 1024
  translation_port_end     = 32000
  priority_location        = "end"
}
```

### SNAT without port range (ICMP)

```hcl
resource "netactuate_router_vrf_snat_rule" "icmp_snat" {
  router_id          = netactuate_router.main.id
  vrf_id             = netactuate_router.main.default_vrf_id
  ip_version         = 4
  protocol           = "ICMP"
  match_interface_id = netactuate_router_vrf_interface.dummy.interface_id
  match_network      = "10.0.0.0/8"
  translation_network = "0.0.0.0/0"
  priority_location  = "end"
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.
- `vrf_id` (Number) — The ID of the VRF. Forces recreation.
- `ip_version` (Number) — IP version for the rule: `4` or `6`.
- `protocol` (String) — Protocol to match: `"TCP"`, `"UDP"`, or `"ICMP"`.
- `match_interface_id` (Number) — The interface through which the outbound traffic exits.
- `match_network` (String) — Source network to match in CIDR notation.
- `translation_network` (String) — Destination/translated network. Use `"0.0.0.0/0"` to masquerade to the interface address.

### Optional

- `description` (String) — A human-readable description of the rule.
- `match_port_start` (Number) — Start of source port range to match. Valid range: 1–32000. Required when `match_port_end` is set (TCP/UDP only).
- `match_port_end` (Number) — End of source port range to match. Valid range: 1–32000. Required when `match_port_start` is set.
- `translation_port_start` (Number) — Start of translated port range. Valid range: 1–32000. Required when `match_port_start`/`match_port_end` are set.
- `translation_port_end` (Number) — End of translated port range. Valid range: 1–32000.
- `priority_location` (String) — Placement of the rule: `"start"`, `"end"`, or `"after"`.
- `priority_after_snat_rule_id` (Number) — Insert this rule after the given SNAT rule ID. Used when `priority_location = "after"`.

### Computed

- `snat_rule_id` (Number) — The API-assigned SNAT rule ID.

## Notes

- Port ranges are only applicable to `TCP` and `UDP` protocols. Do not set port fields when using `ICMP`.
- When port ranges are set for TCP/UDP, both `match_port_start`/`match_port_end` and `translation_port_start`/`translation_port_end` must be provided together.
- Port values above 32000 are rejected by the API.
- `depends_on` the interface resource is recommended to ensure the interface exists before the rule references its ID.

## Import

SNAT rules do not support `terraform import`. Use `terraform state` to manage existing rules.
