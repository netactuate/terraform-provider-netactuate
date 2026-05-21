# netactuate_router_vrf_bgp_neighbor

Manages a BGP neighbor (peer) in a VRF. Supports import and export routing policies using `netactuate_router_prefix_list` rules, MD5 authentication, eBGP multihop, and address-family control.

## Example Usage

### Basic eBGP peer with route filtering

```hcl
resource "netactuate_router_prefix_list" "allow_private" {
  router_id  = netactuate_router.main.id
  name       = "allow-private-v4"
  ip_version = 4

  rule {
    action = "permit"
    prefix = "10.0.0.0/8"
  }
}

resource "netactuate_router_vrf_bgp_neighbor" "upstream" {
  router_id        = netactuate_router.main.id
  vrf_id           = netactuate_router.main.default_vrf_id
  name             = "upstream-peer"
  description      = "Transit provider"
  address          = "203.0.113.1"
  remote_asn       = 64512
  do_next_hop_self = true

  ipv4_enabled = true
  ipv6_enabled = false

  import_default_drop = true
  import_rules {
    prefix_list_id       = netactuate_router_prefix_list.allow_private.prefix_list_id
    action               = "permit"
    set_local_preference = 200
  }

  export_default_drop = false
}
```

### iBGP peer with MD5 and multihop

```hcl
resource "netactuate_router_vrf_bgp_neighbor" "rr" {
  router_id      = netactuate_router.main.id
  vrf_id         = netactuate_router.main.default_vrf_id
  name           = "route-reflector"
  address        = "10.0.0.1"
  remote_asn     = 65001
  md5_secret     = var.bgp_md5_secret
  ebgp_multihop  = 3
  source_address = "10.0.0.2"
  ipv4_enabled   = true
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.
- `vrf_id` (Number) — The ID of the VRF. Forces recreation.
- `address` (String) — The IP address of the BGP peer.
- `remote_asn` (Number) — The peer's ASN. Valid range: 1–4294967294.

### Optional

- `name` (String) — A name for this neighbor.
- `description` (String) — A description for this neighbor.
- `is_shutdown` (Boolean) — Administratively shut down this neighbor without removing it.
- `do_as_override` (Boolean) — Enable AS override on outbound updates. Useful when multiple peers share the same ASN.
- `do_next_hop_self` (Boolean) — Enable `next-hop-self` on outbound updates. Recommended for eBGP.
- `source_address` (String) — Source IP for establishing the BGP session (update-source).
- `ipv4_enabled` (Boolean) — Enable the IPv4 unicast address family.
- `ipv6_enabled` (Boolean) — Enable the IPv6 unicast address family.
- `ebgp_multihop` (Number) — eBGP multihop TTL. Valid range: 1–255.
- `md5_secret` (String, Sensitive) — MD5 authentication password. Valid length: 1–128 chars.
- `import_default_drop` (Boolean) — Drop all imported routes that do not match an `import_rules` entry.
- `import_rules` (Block List) — Ordered import routing policy. Each block:
  - `prefix_list_id` (Number, required) — The prefix list to match against.
  - `action` (String, required) — `"permit"`, `"deny"`, or `"next"`.
  - `set_local_preference` (Number, optional) — Set local preference on matching routes (≥ 0).
- `export_default_drop` (Boolean) — Drop all exported routes that do not match an `export_rules` entry.
- `export_rules` (Block List) — Ordered export routing policy. Each block:
  - `prefix_list_id` (Number, required) — The prefix list to match against.
  - `action` (String, required) — `"permit"`, `"deny"`, or `"next"`.
  - `prepend_last_asn` (Number, optional) — Number of times to prepend the last ASN on export. Valid range: 1–10.

### Computed

- `neighbor_id` (Number) — The API-assigned neighbor ID.

## Import

```
terraform import netactuate_router_vrf_bgp_neighbor.upstream <router_id>/<vrf_id>/<neighbor_id>
```
