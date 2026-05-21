variable "api_key" {
  description = "NetActuate API key (or set NETACTUATE_API_KEY env var)"
  type        = string
  sensitive   = true
  default     = null
}

variable "billing_contract_id" {
  description = "Billing contract ID provided by NetActuate"
  type        = string
}

variable "location" {
  description = "Location code for deployment, e.g. 'SJC'"
  type        = string
  default     = "SJC"
}

variable "router_plan" {
  description = "Plan/package name for the cloud router"
  type        = string
  default     = "VR1x1x25"
}

variable "local_asn" {
  description = "Your BGP ASN as a string, e.g. '65001'"
  type        = string
}

variable "anycast_prefix" {
  description = "Example IPv4 prefix value used in generated routing/config outputs, e.g. '203.0.113.0/24'"
  type        = string
}

variable "upstream_peer_ip" {
  description = "Example peer IP value used in generated routing/config outputs"
  type        = string
}

variable "upstream_peer_asn" {
  description = "Example peer ASN value used in generated routing/config outputs"
  type        = number
}

variable "vpc_cidr" {
  description = "VPC CIDR from example 03: `terraform output vpc_cidr`. Used for the static route."
  type        = string
  default     = "10.10.0.0/24"
}

variable "vpc_bastion_ip" {
  description = "Bastion IP from example 03: `terraform output bastion_ipv4`. Used as the static route next hop."
  type        = string
}

variable "wireguard_port" {
  description = "WireGuard listen port on the router"
  type        = number
  default     = 51820
}

variable "wireguard_tunnel_cidr" {
  description = "IPv4 CIDR for the router end of the WireGuard tunnel, e.g. '10.200.0.1/30'. The VM peer should use the next address (10.200.0.2/30)."
  type        = string
  default     = "10.200.0.1/30"
}

variable "wireguard_remote_public_key" {
  description = "WireGuard public key of the edge VM (example 01). After setting up WireGuard on the VM per its wireguard_setup_guide output: `sudo cat /etc/wireguard/public.key`"
  type        = string
}

variable "wireguard_remote_endpoint" {
  description = "WireGuard endpoint of the edge VM from example 01: `terraform output wireguard_endpoint`. Format: IP:port."
  type        = string
}

variable "wireguard_allowed_ips" {
  description = "CIDRs routed through the WireGuard tunnel to the edge VM side. Include the tunnel subnet (e.g. 10.200.0.0/30) so BGP over the tunnel works."
  type        = list(string)
  default     = ["10.0.0.0/8"]
}
