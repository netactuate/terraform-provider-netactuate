# NKE cluster in a VPC with the VPC load balancer add-on.
# Services that set loadBalancerClass "netactuate.com/vpc-lb" (app.yaml) and Ingresses that set
# ingressClassName "netactuate" (ingress.yaml) each get a public floating IP. See README.md.

variable "location" {
  description = "Location for the VPC and the cluster"
  type        = string
  default     = "SJC"
}

variable "contract_id" {
  description = "Your NKE usage contract ID"
  type        = number
}

variable "kubeconfig_expiration_seconds" {
  description = "How long the kubeconfig token stays valid, in seconds (default 7 days)"
  type        = number
  default     = 604800
}

resource "netactuate_vpc" "main" {
  label               = "nke-lb-example"
  description         = "VPC for the NKE load balancer example"
  location            = var.location
  enable_default_snat = true # NKE workers need outbound access to bootstrap
}

resource "netactuate_nke_cluster" "main" {
  name          = "nke-lb-example"
  location      = var.location
  version       = "1.35.1" # see data.netactuate_nke_versions
  plan          = "VR2x2x25"
  contract_id   = var.contract_id
  minimum_nodes = 2
  maximum_nodes = 2
  vpc_id        = netactuate_vpc.main.vpc_id
}

resource "netactuate_nke_vpc_lb_addon" "main" {
  cluster_id = netactuate_nke_cluster.main.cluster_id
}

# Issued once and kept in state; replaced by the next apply after it expires.
resource "netactuate_nke_kubeconfig" "main" {
  cluster_id         = netactuate_nke_cluster.main.cluster_id
  expiration_seconds = var.kubeconfig_expiration_seconds
}

# The Services and Ingresses the add-on manages, with their floating IPs.
data "netactuate_nke_lb_services" "main" {
  cluster_id = netactuate_nke_cluster.main.cluster_id
  depends_on = [netactuate_nke_vpc_lb_addon.main]
}

output "lb_addresses" {
  value = { for s in data.netactuate_nke_lb_services.main.services : "${s.kind}/${s.namespace}/${s.name}" => s.vip }
}

output "kubeconfig" {
  value     = netactuate_nke_kubeconfig.main.kubeconfig
  sensitive = true
}
