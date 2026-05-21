### Router

output "router_id" {
  description = "Cloud router ID"
  value       = netactuate_router.pop.router_id
}

output "router_ipv4" {
  description = "Router management IPv4 address"
  value       = netactuate_router.pop.ipv4_address
}

output "router_status" {
  description = "Router status"
  value       = netactuate_router.pop.status
}

output "default_vrf_id" {
  description = "Default VRF ID — used to reference this router in other resources"
  value       = netactuate_router.pop.default_vrf_id
}

### BGP

output "local_asn" {
  description = "Local BGP ASN"
  value       = var.local_asn
}

output "anycast_prefix" {
  description = "Configured IPv4 prefix value used in generated outputs"
  value       = var.anycast_prefix
}

output "prefix_list_id" {
  description = "Prefix list ID controlling BGP export of the configured prefix"
  value       = netactuate_router_prefix_list.anycast_export.prefix_list_id
}

output "bgp_neighbor_id" {
  description = "Edge VM BGP neighbor ID (over WireGuard)"
  value       = netactuate_router_vrf_bgp_neighbor.upstream.neighbor_id
}

output "loopback_ipv4" {
  description = "Loopback IP used as the router-side BGP source address"
  value       = netactuate_router_vrf_interface.lo.ipv4_cidr
}

### WireGuard
# Give wireguard_endpoint to the edge VM (example 01) admin to configure their
# side of the tunnel. The router's own WireGuard public key must be obtained
# via the router management interface.

output "wireguard_endpoint" {
  description = "WireGuard endpoint for this router — give this to the edge VM (example 01)"
  value       = "${netactuate_router.pop.ipv4_address}:${var.wireguard_port}"
}

output "wireguard_interface_id" {
  description = "WireGuard interface ID"
  value       = netactuate_router_vrf_interface.wg.interface_id
}

output "wireguard_public_key" {
  description = "Router's WireGuard public key — use this as the peer public key on the edge VM (example 01)"
  value       = netactuate_router_vrf_interface.wg.wireguard_public_key
}

output "wireguard_vm_conf" {
  description = "Complete /etc/wireguard/wg0.conf for the edge VM (example 01) — fill in PrivateKey then: sudo systemctl enable --now wg-quick@wg0"
  value       = <<-EOT
    [Interface]
    # PrivateKey = $(sudo cat /etc/wireguard/private.key)
    Address    = ${cidrhost(var.wireguard_tunnel_cidr, 2)}/${split("/", var.wireguard_tunnel_cidr)[1]}
    ListenPort = ${var.wireguard_port}

    [Peer]
    PublicKey           = ${netactuate_router_vrf_interface.wg.wireguard_public_key}
    Endpoint            = ${netactuate_router.pop.ipv4_address}:${var.wireguard_port}
    AllowedIPs          = ${join(", ", var.wireguard_allowed_ips)}
    PersistentKeepalive = 25
  EOT
}

output "bird_conf" {
  description = "Complete /etc/bird/bird.conf for the edge VM (example 01) — apply with: sudo birdc configure"
  value       = <<-EOT
    log syslog all;
    router id ${split(":", var.wireguard_remote_endpoint)[0]};

    protocol device { }

    protocol kernel {
      ipv4 { export where source = RTS_BGP; };
    }

    protocol direct {
      ipv4;
    }

    protocol static {
      ipv4;
      route ${var.anycast_prefix} blackhole;
    }

    protocol bgp cloud_router {
      description "eBGP to cloud router AS${var.local_asn}";
      local ${cidrhost(var.wireguard_tunnel_cidr, 2)} as ${var.upstream_peer_asn};
      neighbor ${cidrhost(var.wireguard_tunnel_cidr, 1)} as ${var.local_asn};
      hold time 90;

      ipv4 {
        import all;
        export where source = RTS_STATIC;
      };
    }
  EOT
}

### Connectivity tests

output "connectivity_tests" {
  description = "Commands to verify end-to-end connectivity across all examples"
  value       = <<-EOT
    # Test VPC load balancer (example 03) — reachable from anywhere:
    curl http://${var.vpc_bastion_ip}

    # Test edge VM HTTP (example 01) — reachable from anywhere:
    curl http://${split(":", var.wireguard_remote_endpoint)[0]}

    # Test router management reachability:
    ping ${netactuate_router.pop.ipv4_address}

    # From edge VM (example 01), via WireGuard, test VPC subnet route:
    # curl http://<backend_private_ip>   # e.g. 10.10.0.4 (from example 03 backend_private_ips output)
  EOT
}

### Ansible router vars

output "ansible_router_vars" {
  description = "Structured vars for network automation"
  value = {
    router_ip      = netactuate_router.pop.ipv4_address
    local_asn      = var.local_asn
    anycast_prefix = var.anycast_prefix
    wg_endpoint    = "${netactuate_router.pop.ipv4_address}:${var.wireguard_port}"
    wg_peer_ip     = var.upstream_peer_ip
    vpc_cidr       = var.vpc_cidr
  }
}
