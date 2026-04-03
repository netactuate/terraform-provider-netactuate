# netactuate_router_vrf_dhcp

Configures a DHCP server on a VRF interface of a cloud router. The router hands out IP leases, DNS servers, NTP servers, and static routes to DHCP clients on the attached subnet.

## Example Usage

```hcl
resource "netactuate_router_vrf_interface" "lan" {
  router_id = netactuate_router.main.id
  vrf_id    = netactuate_router.main.default_vrf_id
  type      = "dummy"
  name      = "lan0"
  ipv4_cidr = "10.100.0.1/24"
}

resource "netactuate_router_vrf_dhcp" "lan_dhcp" {
  router_id    = netactuate_router.main.id
  vrf_id       = netactuate_router.main.default_vrf_id
  enabled      = true
  interface_id = netactuate_router_vrf_interface.lan.interface_id
  subnet       = "10.100.0.0/24"

  default_router_address = "10.100.0.1"
  lease_timeout          = 86400
  do_ping_check          = true
  client_domain_name     = "internal.example.com"

  range {
    first_address = "10.100.0.100"
    last_address  = "10.100.0.200"
  }

  domain_name_servers {
    address = "1.1.1.1"
  }

  domain_name_servers {
    address = "8.8.8.8"
  }

  ntp_servers {
    address = "10.100.0.1"
  }

  static_routes {
    network  = "192.168.0.0/16"
    next_hop = "10.100.0.1"
  }
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.
- `vrf_id` (Number) — The ID of the VRF. Forces recreation.
- `enabled` (Boolean) — Whether the DHCP service is active.
- `interface_id` (Number) — The interface on which the DHCP service listens.
- `subnet` (String) — The subnet CIDR for DHCP address assignment, e.g. `"10.100.0.0/24"`.

### Optional

- `default_router_address` (String) — Default gateway IP given to DHCP clients.
- `lease_timeout` (Number) — Lease duration in seconds. Must be ≥ 1. Default: `86400` (24 hours).
- `do_ping_check` (Boolean) — If `true`, the router pings an address before assigning it to prevent conflicts. Default: `true`.
- `client_domain_name` (String) — DNS search domain pushed to clients. Omit if not needed.
- `range` (Block, optional) — The leasable address range within the subnet:
  - `first_address` (String, required) — First address in the range.
  - `last_address` (String, required) — Last address in the range.
- `domain_name_servers` (Block List) — DNS servers pushed to clients. Each block:
  - `address` (String, required) — DNS server IP.
- `ntp_servers` (Block List) — NTP servers pushed to clients. Each block:
  - `address` (String, required) — NTP server IP.
- `static_routes` (Block List) — Static routes pushed to clients via DHCP option 121. Each block:
  - `network` (String, required) — Destination network in CIDR notation.
  - `next_hop` (String, required) — Next-hop IP for this route.

## Import

```
terraform import netactuate_router_vrf_dhcp.lan_dhcp <router_id>/<vrf_id>
```
