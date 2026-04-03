provider "netactuate" {
  # API key can also be set via NETACTUATE_API_KEY environment variable
  api_key    = "NETACTUATE_API_KEY"
  api_url    = "vAPI2_URL"  #vAPI2 URL
  api_url_v3 = "vAPI3_URL"  #vAPI3 URL
}

# Look up available Kubernetes versions
data "netactuate_nke_versions" "available" {}

output "available_versions" {
  value = data.netactuate_nke_versions.available.versions
}

# Create an NKE (managed Kubernetes) cluster
resource "netactuate_nke_cluster" "example" {
  name    = "test-terraform-k8s-cluster"
  version = "1.35.0"   # or reference data.netactuate_nke_versions.available.versions[0]

  # Node configuration
  replicas      = 2
  minimum_nodes = 2
  maximum_nodes = 3

  # Autoscaling (optional)
  do_autoscaling = false

  # Control plane options (require recreation if changed)
  kubernetes_dashboard = true
  do_dual_stack        = false

  # Location: use name or location_id = <int>
  location = "DEVRDU - Raleigh, NC"

  # Plan/package name (same as plan in netactuate_server)
  plan = "VR2x2x25"

  # Billing contract ID (required for some accounts)
  contract_id = 1

  # Optional: assign tags by ID
  tag_ids = [3]
}

# Outputs for the cluster
output "cluster_id" {
  value = netactuate_nke_cluster.example.cluster_id
}

output "cluster_status" {
  value = netactuate_nke_cluster.example.status
}

output "cluster_api_url" {
  value = netactuate_nke_cluster.example.api_url
}

output "cluster_prometheus_url" {
  value = netactuate_nke_cluster.example.prometheus_url
}

output "cluster_pod_network" {
  value = netactuate_nke_cluster.example.pod_network
}

output "cluster_service_network" {
  value = netactuate_nke_cluster.example.service_network
}

# Kubeconfig: separate data source — each read creates a new access token
data "netactuate_nke_kubeconfig" "kube" {
  cluster_id = netactuate_nke_cluster.example.cluster_id
  depends_on = [netactuate_nke_cluster.example]
}

# Write kubeconfig to local file — refreshes once per year, removed on destroy
resource "terraform_data" "kubeconfig_file" {
  input            = data.netactuate_nke_kubeconfig.kube.kubeconfig
  triggers_replace = formatdate("YYYY", plantimestamp())

  provisioner "local-exec" {
    command     = "printf '%s' \"$KUBECONFIG_CONTENT\" > kubeconfig.yaml"
    environment = { KUBECONFIG_CONTENT = self.input }
    working_dir = path.module
  }

  provisioner "local-exec" {
    when        = destroy
    command     = "rm -f kubeconfig.yaml"
    working_dir = path.module
  }

  lifecycle {
    ignore_changes = [input]
  }
}

# Worker nodes — captured once at first apply; ignore_changes suppresses constant
# diffs from autoscaling events (nodes added/removed without user action needed)
data "netactuate_nke_worker_nodes" "nodes" {
  cluster_id = netactuate_nke_cluster.example.cluster_id
  depends_on = [netactuate_nke_cluster.example]
}

resource "terraform_data" "worker_nodes" {
  input = data.netactuate_nke_worker_nodes.nodes.worker_nodes

  lifecycle {
    ignore_changes = [input]
  }
}

output "worker_nodes" {
  value = terraform_data.worker_nodes.output
}
