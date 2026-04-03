# netactuate_router_ntp

Configures the NTP service on a cloud router. The router can act as an NTP server for downstream clients and optionally be restricted to a specific interface.

## Example Usage

```hcl
resource "netactuate_router" "main" {
  name     = "prod-router"
  location = "SJC"
  plan     = "VR1x1x25"
}

resource "netactuate_router_ntp" "ntp" {
  router_id = netactuate_router.main.id
  enabled   = true

  upstreams {
    domain = "time.cloudflare.com"
  }

  upstreams {
    domain = "time.google.com"
  }
}
```

### Restrict NTP to a specific interface

```hcl
resource "netactuate_router_ntp" "ntp" {
  router_id    = netactuate_router.main.id
  enabled      = true
  interface_id = netactuate_router_vrf_interface.lan.interface_id

  upstreams {
    domain = "time.cloudflare.com"
  }
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.
- `enabled` (Boolean) — Whether the NTP service is active.
- `upstreams` (Block List, min 1) — Upstream NTP servers. Each block requires:
  - `domain` (String) — Hostname of the NTP server, e.g. `"time.cloudflare.com"`.

### Optional

- `interface_id` (Number) — Restrict the NTP service to this interface. If omitted, NTP is available on all interfaces.

## Import

```
terraform import netactuate_router_ntp.ntp <router_id>
```
