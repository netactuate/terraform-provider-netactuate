# NKE cluster in a VPC, exposed without the load balancer add-on: a floating IP and a VPC network
# load balancer sends TCP port 80 to a NodePort Service on the worker nodes. See README.md.

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
  label               = "nke-nodeport-example"
  description         = "VPC for the NKE NodePort example"
  location            = var.location
  enable_default_snat = true # NKE workers need outbound access to bootstrap
}

resource "netactuate_nke_cluster" "main" {
  name          = "nke-nodeport-example"
  location      = var.location
  version       = "1.35.1" # see data.netactuate_nke_versions
  plan          = "VR2x2x25"
  contract_id   = var.contract_id
  minimum_nodes = 2
  maximum_nodes = 2
  vpc_id        = netactuate_vpc.main.vpc_id
}

# Issued once and kept in state; replaced by the next apply after it expires.
resource "netactuate_nke_kubeconfig" "main" {
  cluster_id         = netactuate_nke_cluster.main.cluster_id
  expiration_seconds = var.kubeconfig_expiration_seconds
}

# Each worker node's private IP in the VPC: the load balancer's backends.
data "netactuate_nke_worker_private_ips" "main" {
  cluster_id = netactuate_nke_cluster.main.cluster_id
}

resource "netactuate_vpc_floating_ip" "web" {
  vpc_id     = netactuate_vpc.main.vpc_id
  ip_version = 4
}

resource "netactuate_network_loadbalancer_group" "web" {
  network_loadbalancer_id = netactuate_vpc.main.network_loadbalancer_id
  name                    = "web-80"
  ip_version              = 4
  algorithm               = "round-robin"
  match_address           = netactuate_vpc_floating_ip.web.address

  health_check {
    enabled  = true
    method   = "TCP"
    interval = 10
    retries  = 3
    delay    = 5
    timeout  = 5
  }

  rule {
    protocol      = "TCP"
    port_match    = 80
    port_internal = 30080 # the Service's nodePort in app.yaml
  }

  dynamic "backend" {
    for_each = data.netactuate_nke_worker_private_ips.main.private_ips
    content {
      name             = backend.key
      internal_address = backend.value
    }
  }
}

output "kubeconfig" {
  value     = netactuate_nke_kubeconfig.main.kubeconfig
  sensitive = true
}

output "url" {
  value = "http://${netactuate_vpc_floating_ip.web.address}"
}
