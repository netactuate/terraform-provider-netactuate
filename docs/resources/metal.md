# netactuate_metal

Manages a NetActuate bare metal server. Bare metal servers provide dedicated physical hardware at a PoP location, suitable for high-performance workloads such as game servers, DDoS scrubbing, and high-throughput proxies.

## Example Usage

```hcl
resource "netactuate_sshkey" "default" {
  name       = "my-key"
  public_key = file("~/.ssh/id_ed25519.pub")
}

resource "netactuate_metal" "edge" {
  hostname   = "edge01.example.com"
  location   = "SJC"
  device_id  = 42
  profile    = 10
  ssh_key_id = netactuate_sshkey.default.id
}

output "ipv4" {
  value = netactuate_metal.edge.primary_ipv4
}
```

## Argument Reference

### Required

- `hostname` (String) — Fully-qualified hostname for the server.
- `device_id` (Number) — The bare metal device ID to provision.
- `profile` (Number) — The build profile ID to apply.

### Required (one of `location`/`location_id`)

- `location` (String) — Location IATA code, e.g. `"SJC"`.
- `location_id` (Number) — Numeric location ID.

### Required (exactly one of `password`, `ssh_key_id`, `ssh_key`)

- `password` (String, Sensitive) — Root password for the server.
- `ssh_key_id` (Number) — ID of an existing `netactuate_sshkey` resource.
- `ssh_key` (String) — SSH public key content.

### Optional

- `build_script` (String) — Post-build script content to run after provisioning.
- `disklayout` (Number) — Disk layout ID to use during provisioning.

### Computed

- `primary_ipv4` (String) — Primary public IPv4 address.
- `primary_ipv6` (String) — Primary public IPv6 address.

## Notes

- Bare metal provisioning can take up to 45 minutes. The provider waits for the build job to complete before returning.
- Changing `profile`, `build_script`, `hostname`, or `disklayout` triggers a rebuild (re-provisioning) of the server.

## Import

```
terraform import netactuate_metal.edge <mbpkgid>
```
