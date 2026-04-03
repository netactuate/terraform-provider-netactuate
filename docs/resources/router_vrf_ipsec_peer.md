# netactuate_router_vrf_ipsec_peer

Manages an IPSec VPN peer on a VRF. Uses the global IPSec policy defined in `netactuate_router_ipsec`. Each peer represents one site-to-site tunnel.

## Example Usage

### Initiate connection to remote site

```hcl
resource "netactuate_router_ipsec" "policy" {
  router_id                = netactuate_router.main.id
  ike_key_exchange_version = 2
  ike_encryption           = "aes256"
  ike_hash                 = "sha256"
  esp_encryption           = "aes256"
  esp_hash                 = "sha256"
}

resource "netactuate_router_vrf_ipsec_peer" "branch" {
  depends_on = [netactuate_router_ipsec.policy]

  router_id              = netactuate_router.main.id
  vrf_id                 = netactuate_router.main.default_vrf_id
  name                   = "branch-office"
  description            = "IPSec tunnel to branch office"
  remote_id              = "branch.example.com"
  psk_secret             = var.ipsec_psk
  do_initiate_connection = true
  peer_address           = "203.0.113.100"
  overlay_ipv4_cidr      = "169.254.100.0/30"
}

# Route branch traffic over IPSec
resource "netactuate_router_static_route" "branch" {
  router_id     = netactuate_router.main.id
  vrf_id        = netactuate_router.main.default_vrf_id
  network       = "10.50.0.0/16"
  ipsec_peer_id = netactuate_router_vrf_ipsec_peer.branch.ipsec_peer_id
}
```

### Passive (respond-only) peer

```hcl
resource "netactuate_router_vrf_ipsec_peer" "passive" {
  router_id              = netactuate_router.main.id
  vrf_id                 = netactuate_router.main.default_vrf_id
  name                   = "remote-initiator"
  remote_id              = "remote.example.com"
  psk_secret             = var.ipsec_psk
  do_initiate_connection = false
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.
- `vrf_id` (Number) — The ID of the VRF. Forces recreation.
- `name` (String) — A name for the IPSec peer.
- `remote_id` (String) — The remote peer identifier (1–64 chars), e.g. a hostname or IP.
- `psk_secret` (String, Sensitive) — The pre-shared key (1–256 chars).

### Optional

- `description` (String) — A description for the peer.
- `do_initiate_connection` (Boolean) — If `true`, the router initiates the tunnel. If `false`, the router waits for the remote to connect (passive mode).
- `peer_address` (String) — Remote router IPv4 address. Required when `do_initiate_connection = true`. Cannot be set in passive mode.
- `overlay_ipv4_cidr` (String) — IPv4 CIDR for the tunnel overlay network.
- `overlay_ipv6_cidr` (String) — IPv6 CIDR for the tunnel overlay network.

### Computed

- `ipsec_peer_id` (Number) — The API-assigned IPSec peer ID. Referenced in `netactuate_router_static_route`.
- `local_id` (String) — The local router identifier used in IKE negotiation.

## Import

```
terraform import netactuate_router_vrf_ipsec_peer.branch <router_id>/<vrf_id>/<ipsec_peer_id>
```
