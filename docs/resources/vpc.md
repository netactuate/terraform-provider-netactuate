# netactuate_vpc

Manages a NetActuate Virtual Private Cloud (VPC). A VPC provides an isolated private network with a bastion gateway, optional DHCP, firewall, and NAT capabilities. Virtual machines can be placed inside a VPC for private connectivity.

## Example Usage

### Basic VPC with outbound internet access

```hcl
resource "netactuate_vpc" "main" {
  label               = "prod-vpc"
  description         = "Production VPC"
  location            = "SJC"
  network_ipv4        = "10.0.0.0/24"
  enable_default_snat = true
}

output "vpc_id" {
  value = netactuate_vpc.main.vpc_id
}

output "bastion_ip" {
  value = netactuate_vpc.main.bastion_ipv4
}
```

### VPC with firewall and custom nameservers

```hcl
resource "netactuate_vpc" "secured" {
  label               = "secured-vpc"
  description         = "VPC with inbound firewall"
  location            = "SJC"
  network_ipv4        = "10.10.0.0/24"
  enable_default_snat = true

  nameservers_ipv4 = ["1.1.1.1", "8.8.8.8"]

  firewall_ipv4_inbound  = true
  firewall_ipv4_outbound = false
}
```

## Argument Reference

### Required

- `label` (String) — Display name for the VPC (max 32 chars).
- `description` (String) — Description for the VPC (max 255 chars).

### Required (one of `location`/`location_id`)

- `location` (String) — Location name, e.g. `"SJC"`. Forces recreation.
- `location_id` (Number) — Numeric location ID. Forces recreation.

### Optional

- `network_ipv4` (String) — IPv4 CIDR block for the VPC private network, e.g. `"10.0.0.0/24"`. Forces recreation.
- `network_ipv6` (String) — IPv6 CIDR block for the VPC private network. Forces recreation.
- `nameservers_ipv4` (List of String) — IPv4 DNS servers for DHCP.
- `nameservers_ipv6` (List of String) — IPv6 DNS servers for DHCP.
- `enable_default_snat` (Boolean) — If `true`, a default SNAT rule is created allowing outbound internet access. Forces recreation.
- `firewall_ipv4_inbound` (Boolean) — Enable inbound IPv4 firewall.
- `firewall_ipv4_outbound` (Boolean) — Enable outbound IPv4 firewall.
- `firewall_ipv6_inbound` (Boolean) — Enable inbound IPv6 firewall.
- `firewall_ipv6_outbound` (Boolean) — Enable outbound IPv6 firewall.
- `bastion_port` (Number) — SSH port for the bastion gateway.

### Computed

- `vpc_id` (Number) — The API-assigned VPC ID.
- `status` (String) — Current VPC status.
- `bastion_ipv4` (String) — Public IPv4 address of the bastion gateway.
- `bastion_ipv6` (String) — Public IPv6 address of the bastion gateway.
- `bastion_enabled` (Boolean) — Whether the bastion is active.
- `network_loadbalancer_id` (Number) — The network load balancer ID for this VPC.
- `http_loadbalancer_id` (Number) — The HTTP load balancer ID for this VPC.

## Notes

- The provider waits up to 10 minutes for the VPC to become ready after creation. VPCs can take 10–15 minutes to provision.
- `enable_default_snat` is only settable at creation time.

## Import

```
terraform import netactuate_vpc.main <vpc_id>
```
