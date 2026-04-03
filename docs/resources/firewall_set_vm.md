# netactuate_firewall_set_vm

Attaches a virtual machine to a `netactuate_firewall_set`, applying the set's rules to that VM's traffic.

## Example Usage

```hcl
resource "netactuate_server" "web" {
  hostname           = "web01"
  plan               = "VR1x1x25"
  location           = "SJC"
  image              = "Ubuntu 22.04 LTS (20240417)"
  contract_id        = 301
}

resource "netactuate_firewall_set" "web" {
  name    = "web-servers"
  enabled = true
}

resource "netactuate_firewall_rule" "allow_https" {
  firewall_set_id        = netactuate_firewall_set.web.id
  ip_version             = "IPv4"
  action                 = "ACCEPT"
  protocol               = "tcp"
  destination_port_start = 443
  destination_port_end   = 443
  enabled                = true
}

resource "netactuate_firewall_set_vm" "web_attachment" {
  firewall_set_id = netactuate_firewall_set.web.id
  mbpkgid         = netactuate_server.web.id
}
```

## Argument Reference

### Required

- `firewall_set_id` (String) — The ID of the firewall set to attach. Forces recreation.
- `mbpkgid` (Number) — The VM package ID to attach. Forces recreation.

### Optional

- `interface_id` (Number) — The network interface index on the VM. Default: `0`. Forces recreation.
- `set_priority` (Number) — Priority of this firewall set for the VM when multiple sets are attached. Default: `0`. Forces recreation.

### Computed

- `hostname` (String) — The hostname of the attached VM.
- `location` (String) — The location name of the VM.
- `iata_code` (String) — The IATA airport code of the VM location.

## Import

```
terraform import netactuate_firewall_set_vm.web_attachment <firewall_set_id>/<mbpkgid>
```
