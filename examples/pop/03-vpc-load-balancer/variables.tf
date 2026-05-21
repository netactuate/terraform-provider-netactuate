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

variable "ssh_public_key" {
  description = "SSH public key content for server and bastion access"
  type        = string
}

variable "backend_plan" {
  description = "Server plan/package name for backend VMs"
  type        = string
  default     = "VR1x1x25"
}

variable "backend_image" {
  description = "OS image name for backend VMs"
  type        = string
  default     = "Ubuntu 24.04 LTS (20240423)"
}

variable "bastion_ssh_user" {
  description = "SSH username for the VPC bastion jump host"
  type        = string
  default     = "jumpuser"
}

variable "backend_ssh_user" {
  description = "SSH username for backend VMs"
  type        = string
  default     = "ubuntu"
}

variable "bootstrap_backend_http_service" {
  description = "Bootstrap nginx + /health endpoint on backend VMs via cloud-init."
  type        = bool
  default     = true
}

variable "vpc_network_ipv4" {
  description = "IPv4 CIDR for the VPC backend subnet. Pass this value into example 04 (vpc_cidr)."
  type        = string
  default     = "10.10.0.0/24"
}

variable "management_cidr" {
  description = "CIDR allowed to reach the bastion SSH port. Restrict this in production."
  type        = string
  default     = "0.0.0.0/0"
}

