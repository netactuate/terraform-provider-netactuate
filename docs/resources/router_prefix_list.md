# netactuate_router_prefix_list

Manages a prefix list on a cloud router. Prefix lists are reusable sets of network prefix rules used to filter BGP import/export policies on `netactuate_router_vrf_bgp_neighbor`.

## Example Usage

```hcl
resource "netactuate_router" "main" {
  name     = "prod-router"
  location = "SJC"
  plan     = "VR1x1x25"
}

# Allow RFC1918 private ranges
resource "netactuate_router_prefix_list" "allow_private" {
  router_id   = netactuate_router.main.id
  name        = "allow-private-v4"
  ip_version  = 4
  description = "Permit RFC1918 address space"

  rule {
    action = "permit"
    prefix = "10.0.0.0/8"
  }

  rule {
    action = "permit"
    prefix = "172.16.0.0/12"
  }

  rule {
    action = "permit"
    prefix = "192.168.0.0/16"
  }
}

# Deny everything else
resource "netactuate_router_prefix_list" "deny_all" {
  router_id  = netactuate_router.main.id
  name       = "deny-all-v4"
  ip_version = 4

  rule {
    action = "deny"
    prefix = "0.0.0.0/0"
  }
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.
- `name` (String) — A name for the prefix list, referenced in BGP neighbor import/export rules.
- `ip_version` (Number) — `4` for IPv4 or `6` for IPv6. Forces recreation.
- `rule` (Block List, min 1) — Ordered list of prefix match rules. Each block requires:
  - `action` (String, required) — `"permit"` or `"deny"`.
  - `prefix` (String, required) — Network prefix in CIDR notation, e.g. `"10.0.0.0/8"`.

### Optional

- `description` (String) — A description for the prefix list.

### Computed

- `prefix_list_id` (Number) — The API-assigned prefix list ID. Referenced in `netactuate_router_vrf_bgp_neighbor` import/export rules.

## Notes

- Rule order in the Terraform config is significant. Rules are evaluated top-to-bottom.
- Reordering `rule` blocks in your config no longer causes a spurious diff (EEN-971 fix).

## Import

```
terraform import netactuate_router_prefix_list.allow_private <router_id>/<prefix_list_id>
```
