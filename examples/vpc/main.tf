provider "netactuate" {
  # API key can also be set via NETACTUATE_API_KEY environment variable
  api_key    = "NETACTUATE_API_KEY"
  api_url    = "vAPI2_URL"  #vAPI2 URL
  api_url_v3 = "vAPI3_URL"  #vAPI3 URL
}

# Create a VPC with a private IPv4 network and default SNAT rule
resource "netactuate_vpc" "example" {
  label       = "TERRAFORM VPC Test"
  description = "test"
  location    = "DEVRDU - Raleigh, NC"

  # Private network CIDR within the VPC
  network_ipv4 = "192.168.16.0/20"

  # DNS nameservers announced via DHCP
  nameservers_ipv4 = ["104.225.32.1", "104.225.32.2"]

  # Enable default SNAT rule for outbound internet connectivity
  enable_default_snat = true

  # Bastion SSH port (optional, defaults to API default)
  bastion_port = 80
}

# Outputs for the VPC
output "vpc_id" {
  value = netactuate_vpc.example.vpc_id
}

output "vpc_status" {
  value = netactuate_vpc.example.status
}

output "vpc_bastion_ipv4" {
  value = netactuate_vpc.example.bastion_ipv4
}

output "vpc_bastion_ipv6" {
  value = netactuate_vpc.example.bastion_ipv6
}

output "vpc_bastion_port" {
  value = netactuate_vpc.example.bastion_port
}

output "vpc_bastion_enabled" {
  value = netactuate_vpc.example.bastion_enabled
}

# Define SSH Key to use for server login
resource "netactuate_sshkey" "sshkey" {
  name = "test@test-pc"
  key  = "ssh-rsa SSH_KEY test@test-pc"
}


# Deploy a server into the VPC
resource "netactuate_server" "vpc_server" {
  hostname    = "vpc-vm.example.com"
  plan        = "PLAN" #PLAN_NAME
  location    = "DEVRDU - Raleigh, NC"
  image       = "Ubuntu 24.04 LTS (20240423)"
  ssh_key_id  = netactuate_sshkey.sshkey.id

  package_billing_contract_id = 1

  # Attach this server to the VPC
  vpc_id = netactuate_vpc.example.vpc_id
}

# Forward port 80 from the VPC's public IPv4 to the private server
resource "netactuate_vpc_gateway_dnat_rule" "web" {
  vpc_id      = netactuate_vpc.example.vpc_id
  ip_version  = 4
  protocol    = "TCP"
  description = "Forward HTTP (IPv4)"

  match_address    = netactuate_vpc.example.bastion_ipv4
  match_port_start = 80
  match_port_end   = 80

  translation_address    = "192.168.16.10"
  translation_port_start = 80
  translation_port_end   = 80
}

# Forward port 80 from the VPC's public IPv6 to the private server
resource "netactuate_vpc_gateway_dnat_rule" "web_v6" {
  vpc_id      = netactuate_vpc.example.vpc_id
  ip_version  = 6
  protocol    = "TCP"
  description = "Forward HTTP (IPv6)"

  match_address    = netactuate_vpc.example.bastion_ipv6
  match_port_start = 80
  match_port_end   = 80

  translation_address    = "fd00::10"
  translation_port_start = 80
  translation_port_end   = 80
}

# SNAT rule: allow the VPC IPv4 subnet to reach the internet (placed first)
resource "netactuate_vpc_gateway_snat_rule" "outbound" {
  vpc_id     = netactuate_vpc.example.vpc_id
  ip_version = 4
  protocol   = "TCP"
  description = "Test SNAT rule"
  translation_port_start    = 10000
  translation_port_end      = 20000
  match_internal_cidr         = "192.168.16.0/20"
  translation_address_start   = netactuate_vpc.example.bastion_ipv4

  priority_location = "start"
}

# SNAT rule: allow the VPC IPv6 subnet to reach the internet
resource "netactuate_vpc_gateway_snat_rule" "outbound_v6" {
  vpc_id      = netactuate_vpc.example.vpc_id
  ip_version  = 6
  protocol    = "TCP"
  match_internal_cidr  = "fd00::/6"
  description = "Outbound SNAT (IPv6)"

  priority_location = "start"
}

# SNAT rule: secondary outbound rule for a different subnet (placed after the first)
resource "netactuate_vpc_gateway_snat_rule" "outbound_secondary" {
  vpc_id     = netactuate_vpc.example.vpc_id
  ip_version = 4
  protocol   = "TCP"
  description = "Secondary SNAT rule for management subnet"
  translation_port_start    = 20001
  translation_port_end      = 30000
  match_internal_cidr         = "192.168.17.0/24"
  translation_address_start   = netactuate_vpc.example.bastion_ipv4

  priority_after_rule_id = netactuate_vpc_gateway_snat_rule.outbound.rule_id
}

# Firewall rule: allow inbound HTTP traffic from a specific network (IPv4)
resource "netactuate_vpc_gateway_firewall_rule" "allow_http_ipv4" {
  vpc_id     = netactuate_vpc.example.vpc_id
  ip_version = 4
  direction  = "inbound"
  protocol   = "TCP"
  description = "Allow HTTP from 172.16.0.0/12"
  network    = "172.16.0.0/12"
  port_start = 80
  port_end   = 81
}

# Firewall rule: allow inbound HTTP traffic (IPv6)
resource "netactuate_vpc_gateway_firewall_rule" "allow_http_ipv6" {
  vpc_id      = netactuate_vpc.example.vpc_id
  ip_version  = 6
  direction   = "inbound"
  protocol    = "TCP"
  description = "Allow HTTP over IPv6"
  network     = "::/0"
  port_start  = 80
  port_end    = 80
}

# Firewall rule: allow inbound SSH traffic (IPv6)
resource "netactuate_vpc_gateway_firewall_rule" "allow_ssh_ipv6" {
  vpc_id      = netactuate_vpc.example.vpc_id
  ip_version  = 6
  direction   = "inbound"
  protocol    = "TCP"
  description = "Allow SSH over IPv6"
  network     = "::/0"
  port_start  = 22
  port_end    = 22
}

# Floating IP: additional IPv4 address with custom PTR
resource "netactuate_vpc_floating_ip" "extra_ipv4" {
  vpc_id     = netactuate_vpc.example.vpc_id
  ip_version = 4
  ptr        = "test.terraform.example.com"
}

# Floating IP: additional IPv6 address with custom PTR
resource "netactuate_vpc_floating_ip" "extra_ipv6" {
  vpc_id     = netactuate_vpc.example.vpc_id
  ip_version = 6
  ptr        = "test.terraform.example.com"
}

output "extra_ipv4_address" {
  value = netactuate_vpc_floating_ip.extra_ipv4.address
}

output "extra_ipv6_address" {
  value = netactuate_vpc_floating_ip.extra_ipv6.address
}

# Backend template: group of backend hosts for load balancing
resource "netactuate_vpc_backend_template" "web_backends" {
  vpc_id      = netactuate_vpc.example.vpc_id
  name        = "test.terraform"
  description = "Backend2 pool for web servers"

  backend_host {
    name    = "web12"
    address = "192.168.16.10"
  }

  backend_host {
    name    = "web22"
    address = "192.168.16.11"
  }

  backend_host {
    name    = "web32"
    address = "192.168.16.12"
  }
}

# Network load balancer group with health checks and forwarding rules
resource "netactuate_network_loadbalancer_group" "web_lb" {
  network_loadbalancer_id = netactuate_vpc.example.network_loadbalancer_id
  name                    = "test-terraform-lb-group"
  description             = "Web server load balancer group"
  ip_version              = 4
  algorithm               = "least-connections"
  match_address           = netactuate_vpc.example.bastion_ipv4

  health_check {
    enabled  = true
    method   = "Ping"
    interval = 10
    retries  = 3
    delay    = 5
    timeout  = 5
  }

  rule {
    protocol      = "UDP"
    port_match    = 80
    port_internal = 80
  }

  backend {
    name             = "web1"
    internal_address = "192.168.16.10"
  }

  backend {
    name             = "web2"
    internal_address = "192.168.16.11"
  }
}

# SSL Certificate
resource "netactuate_ssl_certificate" "example" {
  name        = "example-cert2"
  description = "Test SSL certificate"
  certificate = file("cert.pem")
  private_key = file("key.pem")
}

# HTTP load balancer group with domain rules and SSL
resource "netactuate_http_loadbalancer_group" "web_https" {
  http_loadbalancer_id    = netactuate_vpc.example.http_loadbalancer_id
  name                    = "test http balancer"
  description             = "Test http balancer"
  algorithm               = "least-connections"
  sticky_sessions_enabled = true
  ssl_to_backend_enabled  = false
  internal_port           = 80
  match_address           = netactuate_vpc.example.bastion_ipv4
  match_ports             = "443"

  health_check_active_enabled  = true
  health_check_active_interval = 300
  health_check_active_retries  = 3
  health_check_active_delay    = 300
  health_check_active_path     = "/health"
  health_check_passive_enabled = true

  rule {
    match_domain           = "example.com"
    match_path             = "/"
    ssl_enabled            = true
    https_redirect_enabled = true
  }

  backend {
    name             = "web1"
    internal_address = "192.168.16.10"
  }

  backend {
    name             = "web2"
    internal_address = "192.168.16.11"
  }
backend {
    name             = "web3"
    internal_address = "192.168.16.13"
  }
}

# Enable an existing account-level SSH key for this VPC
resource "netactuate_vpc_ssh_key" "bastion_key" {
  vpc_id     = netactuate_vpc.example.vpc_id
  ssh_key_id = netactuate_sshkey.sshkey.id
  enabled    = true
}
