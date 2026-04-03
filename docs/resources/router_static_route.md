# netactuate_router_static_route

Manages a static route in a VRF on a cloud router. Routes traffic to a next-hop IP address, interface, GRE tunnel, or IPSec peer.

## Example Usage

### Route to next-hop IP

```hcl
resource "netactuate_router_static_route" "default" {
  router_id   = netactuate_router.main.id
  vrf_id      = netactuate_router.main.default_vrf_id
  network     = "0.0.0.0/0"
  next_hop    = "203.0.113.1"
  description = "Default route via upstream gateway"
  distance    = 1
}
```

### Route via tunnel interface

```hcl
resource "netactuate_router_static_route" "via_tunnel" {
  router_id    = netactuate_router.main.id
  vrf_id       = netactuate_router.main.default_vrf_id
  network      = "10.50.0.0/16"
  tunnel_id    = netactuate_router_vrf_tunnel.gre0.tunnel_id
  description  = "Route branch office traffic over GRE"
}
```

### Route via IPSec peer

```hcl
resource "netactuate_router_static_route" "via_ipsec" {
  router_id     = netactuate_router.main.id
  vrf_id        = netactuate_router.main.default_vrf_id
  network       = "172.16.0.0/12"
  ipsec_peer_id = netactuate_router_vrf_ipsec_peer.branch.ipsec_peer_id
  description   = "Route to remote site over IPSec"
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.
- `vrf_id` (Number) — The ID of the VRF. Forces recreation.
- `network` (String) — Destination network in CIDR notation, e.g. `"10.0.0.0/24"`.

### Optional

At least one of `next_hop`, `interface_id`, `tunnel_id`, or `ipsec_peer_id` should be set to define the route target.

- `description` (String) — A description for the route.
- `distance` (Number) — Administrative distance. Valid range: 1–255.
- `next_hop` (String) — Next-hop IPv4 or IPv6 address.
- `interface_id` (Number) — Route via the specified interface.
- `tunnel_id` (Number) — Route via the specified GRE tunnel.
- `ipsec_peer_id` (Number) — Route via the specified IPSec peer.

### Computed

- `route_id` (Number) — The API-assigned static route ID.

## Import

Static routes do not support `terraform import`. Use `terraform state` to manage existing routes.
