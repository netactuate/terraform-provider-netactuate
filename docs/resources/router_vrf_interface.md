# netactuate_router_vrf_interface

Manages a network interface attached to a VRF on a cloud router. Interfaces are the attachment points for routing, NAT rules, DHCP, and tunnels.

## Example Usage

### Dummy interface (loopback-style, no physical attachment)

```hcl
resource "netactuate_router_vrf_interface" "dummy" {
  router_id   = netactuate_router.main.id
  vrf_id      = netactuate_router.main.default_vrf_id
  type        = "dummy"
  name        = "dummy0"
  description = "Outbound masquerade interface"
  ipv4_cidr   = "192.168.10.1/24"
}
```

### WireGuard interface

```hcl
resource "netactuate_router_vrf_interface" "wg0" {
  router_id      = netactuate_router.main.id
  vrf_id         = netactuate_router.main.default_vrf_id
  type           = "wireguard"
  name           = "wg0"
  description    = "WireGuard tunnel interface"
  ipv4_cidr      = "10.10.0.1/30"
  wireguard_port = 51820
}
```

### Ethernet interface (requires hardware ID from NetActuate support)

```hcl
resource "netactuate_router_vrf_interface" "eth0" {
  router_id           = netactuate_router.main.id
  vrf_id              = netactuate_router.main.default_vrf_id
  type                = "ethernet"
  name                = "eth0"
  ethernet_hardware_id = "hw-abc123"
  ipv4_cidr           = "203.0.113.1/29"
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.
- `vrf_id` (Number) — The ID of the VRF. Forces recreation.
- `type` (String) — Interface type. Valid values: `dummy`, `loopback`, `wireguard`, `ethernet`.
- `name` (String) — The interface name (used in routing configs and NAT rules).

### Optional

- `description` (String) — A description for the interface.
- `ipv4_cidr` (String) — IPv4 address in CIDR notation, e.g. `"192.168.10.1/24"`.
- `ipv6_cidr` (String) — IPv6 address in CIDR notation.
- `ethernet_hardware_id` (String) — Required for `type = "ethernet"`. The hardware ID provided by NetActuate support.
- `wireguard_port` (Number) — Listen port for `type = "wireguard"`. Valid range: 1–65535.

### Computed

- `interface_id` (Number) — The API-assigned interface ID. Referenced by NAT rules, DHCP, and static routes.

## Import

```
terraform import netactuate_router_vrf_interface.dummy <router_id>/<vrf_id>/<interface_id>
```
