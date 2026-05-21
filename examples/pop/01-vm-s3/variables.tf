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
  description = "SSH public key content to install on the server"
  type        = string
}

variable "server_plan" {
  description = "Server plan/package name"
  type        = string
  default     = "VR1x1x25"
}

variable "server_image" {
  description = "OS image name for the server"
  type        = string
  default     = "Ubuntu 24.04 LTS (20240423)"
}

variable "bucket_capacity" {
  description = "S3 bucket capacity in GB (required when autoscaling is disabled)"
  type        = number
  default     = 100
}

variable "management_cidr" {
  description = "CIDR allowed to SSH into the server. Restrict this in production (e.g. '203.0.113.5/32')"
  type        = string
  default     = "0.0.0.0/0"
}

variable "bgp_group_id" {
  description = "BGP group ID for anycast sessions (optional). Leave null to skip BGP configuration."
  type        = number
  default     = null
}

variable "wireguard_port" {
  description = "UDP port for the WireGuard tunnel (used in the firewall rule). Must match the port configured on the VM."
  type        = number
  default     = 51820
}

variable "secret_list_name_suffix" {
  description = "Optional suffix appended to the secret list name for uniqueness across repeated test runs (example: run01)."
  type        = string
  default     = ""
}
