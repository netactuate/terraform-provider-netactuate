### VPC

output "vpc_id" {
  description = "VPC ID"
  value       = netactuate_vpc.pop.vpc_id
}

output "bastion_ipv4" {
  description = "Bastion gateway IPv4 address — use as SSH jump host"
  value       = netactuate_vpc.pop.bastion_ipv4
}

output "bastion_ipv6" {
  description = "Bastion gateway IPv6 address"
  value       = netactuate_vpc.pop.bastion_ipv6
}

output "bastion_port" {
  description = "Bastion SSH port"
  value       = netactuate_vpc.pop.bastion_port
}

output "vpc_cidr" {
  description = "VPC CIDR. Use this as vpc_cidr input in example 04."
  value       = var.vpc_network_ipv4
}

output "router_handoff" {
  description = "Values to pass into example 04-cloud-router."
  value = {
    location       = var.location
    vpc_cidr       = var.vpc_network_ipv4
    vpc_bastion_ip = netactuate_vpc.pop.bastion_ipv4
    lb_floating_ip = netactuate_vpc_floating_ip.pub.address
  }
}

### Public entry points

output "floating_ipv4" {
  description = "Public anycast IPv4 — point your DNS A record here"
  value       = netactuate_vpc_floating_ip.pub.address
}

output "floating_ipv6" {
  description = "Reserved public IPv6 address. Add explicit IPv6 LB listener/rules before publishing AAAA."
  value       = netactuate_vpc_floating_ip.pub_v6.address
}

output "http_test_url" {
  description = "HTTP test URL for the TCP load balancer — curl this to verify backends are reachable"
  value       = "http://${netactuate_vpc_floating_ip.pub.address}"
}

### Backend servers

output "backend_private_ips" {
  description = "Private IP addresses of backend VMs"
  value = [
    netactuate_server.backend1.vpc_reserved_network,
    netactuate_server.backend2.vpc_reserved_network,
  ]
}

output "backend_hostnames" {
  description = "Hostnames of backend VMs"
  value = [
    netactuate_server.backend1.hostname,
    netactuate_server.backend2.hostname,
  ]
}

### Ansible SSH proxy
# Paste this value into ansible.cfg as: ssh_common_args = <value>
# Backends are only reachable via the bastion jump host.

output "ansible_ssh_proxy" {
  description = "SSH ProxyJump argument for reaching backend VMs via bastion"
  value       = "-J ${var.bastion_ssh_user}@${netactuate_vpc.pop.bastion_ipv4}:${netactuate_vpc.pop.bastion_port}"
}

output "ssh_backend_example" {
  description = "Example SSH command for backend access through bastion"
  value       = "ssh -J ${var.bastion_ssh_user}@${netactuate_vpc.pop.bastion_ipv4}:${netactuate_vpc.pop.bastion_port} ${var.backend_ssh_user}@${netactuate_server.backend1.vpc_reserved_network}"
}

### Ansible inventory
# terraform output -json ansible_inventory | jq > inventory/pop_<location>.json

output "ansible_inventory" {
  description = "Structured Ansible inventory — backends behind bastion jump host"
  sensitive   = true
  value = {
    backends = {
      hosts = {
        backend1 = {
          ansible_host = netactuate_server.backend1.vpc_reserved_network
          ansible_user = var.backend_ssh_user
        }
        backend2 = {
          ansible_host = netactuate_server.backend2.vpc_reserved_network
          ansible_user = var.backend_ssh_user
        }
      }
      vars = {
        ansible_ssh_common_args = "-J ${var.bastion_ssh_user}@${netactuate_vpc.pop.bastion_ipv4}:${netactuate_vpc.pop.bastion_port}"
        lb_public_ip            = netactuate_vpc_floating_ip.pub.address
        vpc_network             = var.vpc_network_ipv4
      }
    }
  }
}
