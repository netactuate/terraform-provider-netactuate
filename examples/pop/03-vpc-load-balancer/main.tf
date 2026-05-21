### SSH key

resource "netactuate_sshkey" "pop" {
  name = "pop-${var.location}"
  key  = var.ssh_public_key
}

locals {
  # Minimal backend bootstrap so LB health checks and data-plane tests work
  # immediately after apply.
  backend_http_user_data = <<-EOT
    #cloud-config
    package_update: true
    packages:
      - nginx
    write_files:
      - path: /var/www/html/health
        permissions: "0644"
        content: |
          ok
    runcmd:
      - systemctl enable nginx
      - bash -lc 'echo "pop ${var.location} backend $(hostname)" > /var/www/html/index.html'
      - systemctl restart nginx
  EOT
}

### VPC
# Private network with bastion gateway. SNAT provides outbound internet access
# for the backend VMs. Inbound firewalls are enabled so we can attach rules below.

resource "netactuate_vpc" "pop" {
  label                 = "pop-${lower(var.location)}"
  description           = "PoP VPC at ${var.location} — private backend + load balancer"
  location              = var.location
  network_ipv4          = var.vpc_network_ipv4
  enable_default_snat   = true
  firewall_ipv4_inbound = true
  firewall_ipv6_inbound = true
  bastion_port          = 22
}

### Enable SSH key on VPC bastion

resource "netactuate_vpc_ssh_key" "pop" {
  vpc_id     = netactuate_vpc.pop.vpc_id
  ssh_key_id = netactuate_sshkey.pop.id
}

### Backend VMs
# Application servers inside the VPC. Traffic from the internet never reaches
# them directly — only via the load balancer.

resource "netactuate_server" "backend1" {
  hostname                    = "backend01.${lower(var.location)}.internal"
  plan                        = var.backend_plan
  location                    = var.location
  image                       = var.backend_image
  package_billing_contract_id = var.billing_contract_id
  vpc_id                      = netactuate_vpc.pop.vpc_id
  ssh_key_id                  = netactuate_sshkey.pop.id
  user_data                   = var.bootstrap_backend_http_service ? local.backend_http_user_data : null
  tags                        = "pop, backend, ${var.location}"
}

resource "netactuate_server" "backend2" {
  hostname                    = "backend02.${lower(var.location)}.internal"
  plan                        = var.backend_plan
  location                    = var.location
  image                       = var.backend_image
  package_billing_contract_id = var.billing_contract_id
  vpc_id                      = netactuate_vpc.pop.vpc_id
  ssh_key_id                  = netactuate_sshkey.pop.id
  user_data                   = var.bootstrap_backend_http_service ? local.backend_http_user_data : null
  tags                        = "pop, backend, ${var.location}"
}

# Resolve VPC-internal VM addresses for backend/LB wiring.
# In VPC mode, cloud/server `ip` is null; `vpc_reserved_network` is the usable
# backend address for VPC load balancer groups.
### Backend pool

resource "netactuate_vpc_backend_template" "app" {
  vpc_id      = netactuate_vpc.pop.vpc_id
  name        = "app-backends"
  description = "Application server pool"

  backend_host {
    name    = "backend1"
    address = netactuate_server.backend1.vpc_reserved_network
  }

  backend_host {
    name    = "backend2"
    address = netactuate_server.backend2.vpc_reserved_network
  }
}

### Stable public floating IPs

resource "netactuate_vpc_floating_ip" "pub" {
  vpc_id     = netactuate_vpc.pop.vpc_id
  ip_version = 4
  ptr        = "pop-${lower(var.location)}.example.com"
}

resource "netactuate_vpc_floating_ip" "pub_v6" {
  vpc_id     = netactuate_vpc.pop.vpc_id
  ip_version = 6
  ptr        = "pop-${lower(var.location)}.example.com"
}

### Network (TCP) load balancer group
# Handles raw TCP port 80 traffic directly to backend VMs.

resource "netactuate_network_loadbalancer_group" "http" {
  network_loadbalancer_id = netactuate_vpc.pop.network_loadbalancer_id
  name                    = "pop-tcp-80"
  description             = "TCP port 80 — direct pass-through to backends"
  ip_version              = 4
  algorithm               = "least-connections"
  match_address           = netactuate_vpc_floating_ip.pub.address

  health_check {
    enabled  = true
    method   = "Ping"
    interval = 10
    retries  = 3
    delay    = 5
    timeout  = 5
  }

  rule {
    protocol      = "TCP"
    port_match    = 80
    port_internal = 80
  }

  backend {
    name             = "backend1"
    internal_address = netactuate_server.backend1.vpc_reserved_network
  }

  backend {
    name             = "backend2"
    internal_address = netactuate_server.backend2.vpc_reserved_network
  }
}

### VPC gateway firewall rules
# These rules control what enters the VPC through the bastion gateway.

# Allow HTTP inbound
resource "netactuate_vpc_gateway_firewall_rule" "allow_http" {
  vpc_id      = netactuate_vpc.pop.vpc_id
  ip_version  = 4
  direction   = "inbound"
  protocol    = "TCP"
  description = "Allow HTTP"
  port_start  = 80
  port_end    = 80
}

# Allow SSH to bastion from management CIDR only
resource "netactuate_vpc_gateway_firewall_rule" "allow_ssh" {
  vpc_id      = netactuate_vpc.pop.vpc_id
  ip_version  = 4
  direction   = "inbound"
  protocol    = "TCP"
  description = "Allow SSH to bastion from management CIDR"
  network     = var.management_cidr
  port_start  = 22
  port_end    = 22
}

# Allow ICMP (ping) for monitoring
resource "netactuate_vpc_gateway_firewall_rule" "allow_icmp" {
  vpc_id      = netactuate_vpc.pop.vpc_id
  ip_version  = 4
  direction   = "inbound"
  protocol    = "ICMP"
  description = "Allow ICMP for monitoring"
}

### SNAT rule: backend subnet outbound

resource "netactuate_vpc_gateway_snat_rule" "backends" {
  vpc_id                    = netactuate_vpc.pop.vpc_id
  ip_version                = 4
  protocol                  = "TCP"
  description               = "SNAT backend subnet outbound"
  match_internal_cidr       = var.vpc_network_ipv4
  translation_address_start = netactuate_vpc_floating_ip.pub.address
  translation_address_end   = netactuate_vpc_floating_ip.pub.address
}
