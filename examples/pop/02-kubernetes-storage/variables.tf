variable "api_key" {
  description = "NetActuate API key (or set NETACTUATE_API_KEY env var)"
  type        = string
  sensitive   = true
  default     = null
}

variable "billing_contract_id" {
  description = "Billing contract ID provided by NetActuate (NKE-specific contract ID, must be usage-type)"
  type        = number
}

variable "location" {
  description = "Location code for deployment, e.g. 'SJC'"
  type        = string
  default     = "SJC"
}

variable "node_plan" {
  description = "Plan/package name for NKE worker nodes"
  type        = string
  default     = "VR2x2x25"
}

variable "kubernetes_version" {
  description = "Kubernetes version to deploy. Use data.netactuate_nke_versions to list available versions."
  type        = string
  default     = "1.35.0"
}

variable "block_storage_capacity" {
  description = "RBD block storage namespace capacity in GB"
  type        = number
  default     = 100
}

variable "test_pvc_size" {
  description = "Requested size for the generated test PVC manifest"
  type        = string
  default     = "10Gi"
}
