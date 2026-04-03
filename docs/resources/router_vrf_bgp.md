# netactuate_router_vrf_bgp

Configures the BGP daemon on a VRF. Sets the local ASN and the networks to advertise. BGP neighbors are managed separately with `netactuate_router_vrf_bgp_neighbor`.

## Example Usage

```hcl
resource "netactuate_router_vrf_bgp" "bgp" {
  router_id = netactuate_router.main.id
  vrf_id    = netactuate_router.main.default_vrf_id
  local_asn = 65001

  networks {
    subnet = "203.0.113.0/24"
  }

  networks {
    subnet = "198.51.100.0/24"
  }
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.
- `vrf_id` (Number) — The ID of the VRF. Forces recreation.
- `local_asn` (Number) — Your local BGP ASN. Valid range: 1–4294967294.
- `networks` (Block List, min 1) — Networks to announce over BGP sessions. Each block:
  - `subnet` (String, required) — Network prefix in CIDR notation, e.g. `"203.0.113.0/24"`.

## Import

```
terraform import netactuate_router_vrf_bgp.bgp <router_id>/<vrf_id>
```
