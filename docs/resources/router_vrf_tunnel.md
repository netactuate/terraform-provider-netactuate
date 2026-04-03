# netactuate_router_vrf_tunnel

Manages a GRE tunnel interface on a VRF. GRE tunnels create point-to-point Layer 3 paths between the cloud router and a remote endpoint, typically used with static routes or BGP over an IP underlay.

## Example Usage

```hcl
resource "netactuate_router_vrf_tunnel" "gre0" {
  router_id              = netactuate_router.main.id
  vrf_id                 = netactuate_router.main.default_vrf_id
  name                   = "gre0"
  description            = "GRE tunnel to branch office"
  ip_key                 = 42
  mtu                    = 1476
  ipv4_cidr              = "169.254.0.1/30"
  endpoint_address_remote = "203.0.113.50"
}

# Route branch traffic over the tunnel
resource "netactuate_router_static_route" "branch" {
  router_id = netactuate_router.main.id
  vrf_id    = netactuate_router.main.default_vrf_id
  network   = "10.50.0.0/16"
  tunnel_id = netactuate_router_vrf_tunnel.gre0.tunnel_id
}

output "tunnel_source_ip" {
  value = netactuate_router_vrf_tunnel.gre0.endpoint_address_source
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.
- `vrf_id` (Number) — The ID of the VRF. Forces recreation.
- `name` (String) — A name for the tunnel interface.
- `ip_key` (Number) — The GRE key to associate with this tunnel. Used to differentiate multiple GRE tunnels to the same remote.
- `mtu` (Number) — The MTU for the tunnel interface. Valid range: 68–16000. Typical GRE over IPv4 value is `1476`.
- `endpoint_address_remote` (String) — The remote endpoint IP address (tunnel destination).

### Optional

- `description` (String) — A description for the tunnel.
- `ipv4_cidr` (String) — IPv4 address in CIDR notation to assign to the tunnel interface.
- `ipv6_cidr` (String) — IPv6 address in CIDR notation to assign to the tunnel interface.

### Computed

- `tunnel_id` (Number) — The API-assigned tunnel ID. Referenced in `netactuate_router_static_route`.
- `endpoint_address_source` (String) — The source IP address used by the cloud router for this tunnel.
- `ip_version` (Number) — The IP version of the tunnel (derived from endpoint addresses).

## Import

```
terraform import netactuate_router_vrf_tunnel.gre0 <router_id>/<vrf_id>/<tunnel_id>
```
