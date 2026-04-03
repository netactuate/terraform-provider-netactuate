# netactuate_firewall_set

Manages a firewall set — a named collection of firewall rules that can be attached to one or more virtual machines. Rules are managed with `netactuate_firewall_rule` and VMs are attached via `netactuate_firewall_set_vm`.

## Example Usage

```hcl
resource "netactuate_firewall_set" "web" {
  name        = "web-servers"
  description = "Firewall rules for web-tier VMs"
  enabled     = true
}

resource "netactuate_firewall_rule" "allow_https" {
  firewall_set_id        = netactuate_firewall_set.web.id
  ip_version             = "IPv4"
  action                 = "ACCEPT"
  protocol               = "tcp"
  destination_port_start = 443
  destination_port_end   = 443
  enabled                = true
  admin_comment          = "Allow HTTPS inbound"
}

resource "netactuate_firewall_rule" "allow_http" {
  firewall_set_id        = netactuate_firewall_set.web.id
  ip_version             = "IPv4"
  action                 = "ACCEPT"
  protocol               = "tcp"
  destination_port_start = 80
  destination_port_end   = 80
  enabled                = true
}

resource "netactuate_firewall_set_vm" "web_vm" {
  firewall_set_id = netactuate_firewall_set.web.id
  mbpkgid         = 12345
}

output "firewall_set_id" {
  value = netactuate_firewall_set.web.id
}
```

## Argument Reference

### Required

- `name` (String) — The name of the firewall set.

### Optional

- `description` (String) — Description of the firewall set.
- `enabled` (Boolean) — Whether the firewall set rules are active. Default: `true`.

### Computed

- `id` (String) — The API-assigned firewall set ID.

## Import

```
terraform import netactuate_firewall_set.web <firewall_set_id>
```
