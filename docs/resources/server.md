# netactuate_server

Manages a NetActuate cloud virtual machine (server). Servers are deployed into a location and billed according to a plan and billing contract. SSH keys, cloud-init, and VPC placement are all configurable at creation time.

## Example Usage

### Basic server with SSH key

```hcl
resource "netactuate_sshkey" "default" {
  name       = "my-key"
  public_key = file("~/.ssh/id_ed25519.pub")
}

resource "netactuate_server" "web" {
  hostname                     = "web01.example.com"
  plan                         = "VR1x1x25"
  location                     = "SJC"
  image                        = "Ubuntu 24.04 LTS (20240423)"
  package_billing_contract_id  = "12345"
  ssh_key_id                   = netactuate_sshkey.default.id
}

output "ipv4" {
  value = netactuate_server.web.primary_ipv4
}
```

### Server inside a VPC

```hcl
resource "netactuate_server" "backend" {
  hostname                    = "backend01.internal"
  plan                        = "VR1x1x25"
  location                    = "SJC"
  image                       = "Ubuntu 24.04 LTS (20240423)"
  package_billing_contract_id = "12345"
  vpc_id                      = netactuate_vpc.main.vpc_id
  ssh_key_id                  = netactuate_sshkey.default.id
  tags                        = "backend, production"
}
```

## Argument Reference

### Required

- `hostname` (String) — Fully-qualified hostname for the server.
- `plan` (String) — Plan/package name for the server (e.g. `"VR1x1x25"`).

### Required (one of `location`/`location_id`)

- `location` (String) — Location name, e.g. `"SJC"`.
- `location_id` (Number) — Numeric location ID.

### Required (one of `image`/`image_id`)

- `image` (String) — OS image name, e.g. `"Ubuntu 24.04 LTS (20240423)"`.
- `image_id` (Number) — Numeric image ID.

### Optional

- `package_billing_contract_id` (String) — Billing contract ID for this server's plan.
- `package_billing` (String) — Billing type override.
- `package_billing_opt_in` (String) — Billing opt-in override.
- `ssh_key` (String) — SSH public key content to install on the server.
- `ssh_key_id` (Number) — ID of an existing `netactuate_sshkey` resource to install.
- `password` (String, Sensitive) — Root password for the server.
- `cloud_config` (String) — Cloud-init config in YAML format (cloud-config).
- `user_data` (String) — Arbitrary user data string passed to the server.
- `user_data_base64` (String) — Base64-encoded user data.
- `allow_downsize_reboot` (Boolean) — Single opt-in for disruptive scaling. Defaults to `false`: no-reboot upgrades happen automatically, but a downsize (lower mem/cpu/disk) or any reboot-requiring scale is rejected during `terraform apply` before the scale API call. Set `true` to permit downsize+reboot.
- `vpc_id` (Number) — VPC ID to deploy this server into. Forces recreation.
- `cloud_pool_id` (Number) — Cloud pool ID to provision the server from. Forces recreation.
- `params` (String) — JSON-encoded string of additional API parameters (e.g. `no_install`, `iso_url`, `customer_vlan_id`, `firewall_set_list`, `disks`, `nics`).
- `tags` (String) — Tag name or comma-separated tag names to assign to the server, e.g. `"kube"` or `"kube, sjc, cluster"`. Tags are created automatically if they do not exist. When configured, Terraform is authoritative and forces the portal/API tag set to match. When omitted, tags are unmanaged. See [Tagging](../public/tagging.md).

### Computed

- `id` (String) — The server's package ID (`mbpkgid`).
- `primary_ipv4` (String) — Primary public IPv4 address. For VPC-attached servers, this may be empty.
- `primary_ipv6` (String) — Primary public IPv6 address. For VPC-attached servers, this may be empty.
- `vpc_reserved_network` (String) — Private IP address reserved for this server within its VPC. Empty for non-VPC servers.
- `private_ip` (String) — Alias of `vpc_reserved_network`. Empty for non-VPC servers.

## Notes

- Changing `plan` scales the server in place. Plan names are `VR{MEM}x{CPU}x{DISK}`. A **larger** plan (no reboot needed) scales automatically; a **downgrade** (any of mem/cpu/disk smaller) or a non-`VR` plan name is rejected at `terraform apply` (before any API call) unless `allow_downsize_reboot = true`. `terraform refresh` and `terraform plan` always work and surface the drift; the policy gate fires only at apply, so the server is never stopped or rebooted by a refused attempt.
- `vpc_id` and `cloud_pool_id` are set at provision time and require recreation to change.
- `params` is an escape hatch for API features not yet exposed as first-class schema fields.

## Import

```
terraform import netactuate_server.web <mbpkgid>
```
