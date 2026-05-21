### Cloud router
# All resources below use the default_vrf_id — no custom VRFs needed.

resource "netactuate_router" "pop" {
  name        = "pop-${lower(var.location)}"
  description = "PoP cloud router at ${var.location}"
  location    = var.location
  plan        = var.router_plan
}

### NTP synchronization
# Accurate time is important for BGP hold timers and WireGuard handshakes.

resource "netactuate_router_ntp" "pop" {
  router_id = netactuate_router.pop.id
  enabled   = true

  upstreams {
    domain = "0.pool.ntp.org"
  }
  upstreams {
    domain = "1.pool.ntp.org"
  }
}

### Loopback interface
# Holds a stable source IP for the router-side BGP session to the edge VM.
# Using a loopback keeps the session source stable if other interfaces flap.

resource "netactuate_router_vrf_interface" "lo" {
  router_id   = netactuate_router.pop.id
  vrf_id      = netactuate_router.pop.default_vrf_id
  type        = "dummy"
  name        = "lo0"
  description = "Loopback — BGP source"
  ipv4_cidr   = "${cidrhost(var.anycast_prefix, 1)}/32"

  depends_on = [netactuate_router.pop]
}

### WireGuard interface
# Encrypted tunnel between this cloud router and the edge VM.
# This tunnel carries both private routing traffic and the BGP peering session.

resource "netactuate_router_vrf_interface" "wg" {
  router_id      = netactuate_router.pop.id
  vrf_id         = netactuate_router.pop.default_vrf_id
  type           = "wireguard"
  name           = "wg0"
  description    = "WireGuard tunnel to edge VM"
  ipv4_cidr      = var.wireguard_tunnel_cidr
  wireguard_port = var.wireguard_port

  depends_on = [netactuate_router.pop]
}

### WireGuard peer (edge VM)
# Add the edge VM as a peer. Provide its public key and remote endpoint.
# The allowed_ips list restricts which traffic traverses the tunnel.

resource "netactuate_router_vrf_interface_wireguard_peer" "central_dc" {
  router_id    = netactuate_router.pop.id
  vrf_id       = netactuate_router.pop.default_vrf_id
  interface_id = netactuate_router_vrf_interface.wg.interface_id
  name         = "central-dc"
  description  = "WireGuard peer — edge VM"
  remote       = var.wireguard_remote_endpoint
  public_key   = var.wireguard_remote_public_key

  dynamic "allowed_ips" {
    for_each = var.wireguard_allowed_ips
    content {
      network = allowed_ips.value
    }
  }
}

### BGP prefix list
# Export only the configured service prefix to the edge VM BGP peer.

resource "netactuate_router_prefix_list" "anycast_export" {
  router_id   = netactuate_router.pop.id
  name        = "anycast-export"
  ip_version  = 4
  description = "Permit only the configured service prefix for BGP export"

  rule {
    action = "permit"
    prefix = var.anycast_prefix
  }
}

### BGP configuration
# Uses default_vrf_id only. local_asn must be a quoted string.

resource "netactuate_router_vrf_bgp" "pop" {
  router_id = netactuate_router.pop.id
  vrf_id    = netactuate_router.pop.default_vrf_id
  local_asn = var.local_asn

  networks {
    subnet = var.anycast_prefix
  }

  depends_on = [netactuate_router.pop]
}

### BGP neighbor over WireGuard (edge VM)

resource "netactuate_router_vrf_bgp_neighbor" "upstream" {
  router_id        = netactuate_router.pop.id
  vrf_id           = netactuate_router.pop.default_vrf_id
  name             = "edge-vm-wg"
  description      = "Edge VM BGP peer over WireGuard"
  address          = var.upstream_peer_ip
  remote_asn       = var.upstream_peer_asn
  do_next_hop_self = true
  ipv4_enabled     = true
  ipv6_enabled     = false

  export_default_drop = true
  export_rules {
    prefix_list_id = netactuate_router_prefix_list.anycast_export.prefix_list_id
    action         = "permit"
  }

  depends_on = [netactuate_router_vrf_bgp.pop]
}

### Static route to VPC
# Routes VPC backend traffic (10.10.0.0/24) through the VPC bastion gateway,
# enabling the router to reach the backend VMs from example 03.

resource "netactuate_router_static_route" "vpc" {
  router_id   = netactuate_router.pop.id
  vrf_id      = netactuate_router.pop.default_vrf_id
  network     = var.vpc_cidr
  next_hop    = var.vpc_bastion_ip
  description = "Route to VPC backend subnet via bastion"
}
