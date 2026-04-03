# netactuate_router_ipsec

Configures the global IPSec policy (IKE and ESP parameters) for a cloud router. These settings apply to all IPSec peers created with `netactuate_router_vrf_ipsec_peer`. Only one IPSec config exists per router.

## Example Usage

```hcl
resource "netactuate_router_ipsec" "policy" {
  router_id = netactuate_router.main.id

  ike_key_exchange_version   = 2
  ike_lifetime_seconds       = 28800
  ike_dh_group_number        = 14
  ike_encryption             = "aes256"
  ike_hash                   = "sha256"
  ike_do_auto_renegotiation  = true

  esp_lifetime_seconds = 3600
  esp_encryption       = "aes256"
  esp_hash             = "sha256"
}
```

## Argument Reference

### Required

- `router_id` (Number) — The ID of the cloud router. Forces recreation.

### Optional

All IKE and ESP fields are optional. The API applies defaults when fields are omitted.

- `ike_do_auto_renegotiation` (Boolean) — Auto-reconnect on peer loss.
- `ike_key_exchange_version` (Number) — IKE version: `1` or `2`.
- `ike_lifetime_seconds` (Number) — IKE SA lifetime in seconds. Valid range: 0–86400.
- `ike_dh_group_number` (Number) — Diffie-Hellman group number (e.g. `14` for 2048-bit MODP).
- `ike_encryption` (String) — IKE encryption algorithm, e.g. `"aes256"`, `"aes128"`.
- `ike_hash` (String) — IKE integrity/hash algorithm, e.g. `"sha256"`, `"sha1"`.
- `ike_prf` (String) — Pseudo-random function for IKE.
- `esp_lifetime_seconds` (Number) — ESP SA lifetime in seconds. Valid range: 30–86400.
- `esp_encryption` (String) — ESP encryption algorithm, e.g. `"aes256"`.
- `esp_hash` (String) — ESP integrity algorithm, e.g. `"sha256"`.

## Import

```
terraform import netactuate_router_ipsec.policy <router_id>
```
