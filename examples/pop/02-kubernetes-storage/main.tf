### Available Kubernetes versions

data "netactuate_nke_versions" "available" {}

locals {
  rbd_pool_name      = "global-block-pool"
  tenant_id          = tostring(netactuate_storage_block_namespace.rbd.block_namespace_id)
  tenant_namespace   = "namespace-${local.tenant_id}"
  csi_secret_name    = "csi-rbd-secret"
  storage_class_name = "ceph-rbd-sc-${local.tenant_id}"
  pvc_name           = "test-volume-${local.tenant_id}"
  pod_name           = "test-pod-${local.tenant_id}"
  rbd_monitors_csv   = join(",", netactuate_storage_block_namespace.rbd.endpoints)
}

### HA Kubernetes cluster
# 3-replica control plane (high availability), 3–4 worker nodes with autoscaling.
# Services: application pods, nginx-ingress, cert-manager, Prometheus/Grafana,
# and persistent volumes backed by the Ceph RBD namespace below.

resource "netactuate_nke_cluster" "pop" {
  name           = "pop-${lower(var.location)}"
  version        = var.kubernetes_version
  replicas       = 3
  minimum_nodes  = 3
  maximum_nodes  = 4
  do_autoscaling = true

  location    = var.location
  plan        = var.node_plan
  contract_id = var.billing_contract_id
}

### Write kubeconfig to local file
# Refreshed once per year; removed on destroy.

data "netactuate_nke_kubeconfig" "kube" {
  cluster_id = netactuate_nke_cluster.pop.cluster_id
  depends_on = [netactuate_nke_cluster.pop]
}

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

### Worker node snapshot
# Captured once at first apply; autoscaling events don't trigger re-apply.

data "netactuate_nke_worker_nodes" "nodes" {
  cluster_id = netactuate_nke_cluster.pop.cluster_id
  depends_on = [netactuate_nke_cluster.pop]
}

resource "terraform_data" "worker_nodes" {
  input = data.netactuate_nke_worker_nodes.nodes.worker_nodes

  lifecycle {
    ignore_changes = [input]
  }
}

### Ceph RBD block storage namespace
# Provides a Ceph RBD pool for Kubernetes persistent volumes via the CSI driver.
# See: https://netactuate.com/docs/infrastructure/storage/storage-rbd

resource "netactuate_storage_block_namespace" "rbd" {
  label    = "pop-${lower(var.location)}-k8s-rbd"
  location = var.location
  capacity = var.block_storage_capacity
}

### Secret list for Ceph CSI tenant auth

resource "netactuate_secret_list" "k8s_rbd" {
  name = "pop-${lower(var.location)}-k8s-rbd-secrets"
}

resource "netactuate_secret_list_value" "rbd_monitors" {
  secret_list_id = netactuate_secret_list.k8s_rbd.id
  secret_key     = "rbd_monitors_${local.tenant_id}"
  secret_value   = local.rbd_monitors_csv
}

resource "netactuate_secret_list_value" "rbd_pool" {
  secret_list_id = netactuate_secret_list.k8s_rbd.id
  secret_key     = "rbd_pool_${local.tenant_id}"
  secret_value   = local.rbd_pool_name
}

resource "netactuate_secret_list_value" "rbd_namespace" {
  secret_list_id = netactuate_secret_list.k8s_rbd.id
  secret_key     = "rbd_namespace_${local.tenant_id}"
  secret_value   = netactuate_storage_block_namespace.rbd.storage_namespace
}

resource "netactuate_secret_list_value" "rbd_cluster_id" {
  secret_list_id = netactuate_secret_list.k8s_rbd.id
  secret_key     = "rbd_cluster_id_${local.tenant_id}"
  secret_value   = netactuate_storage_block_namespace.rbd.storage_cluster_id
}

resource "netactuate_secret_list_value" "rbd_user_id" {
  secret_list_id = netactuate_secret_list.k8s_rbd.id
  secret_key     = "rbd_user_id_${local.tenant_id}"
  secret_value   = netactuate_storage_block_namespace.rbd.user_key
}

resource "netactuate_secret_list_value" "rbd_user_key" {
  secret_list_id = netactuate_secret_list.k8s_rbd.id
  secret_key     = "rbd_user_key_${local.tenant_id}"
  secret_value   = netactuate_storage_block_namespace.rbd.secret_key
}

resource "netactuate_secret_list_value" "tenant_namespace" {
  secret_list_id = netactuate_secret_list.k8s_rbd.id
  secret_key     = "tenant_namespace_${local.tenant_id}"
  secret_value   = local.tenant_namespace
}

resource "netactuate_secret_list_value" "storage_class_name" {
  secret_list_id = netactuate_secret_list.k8s_rbd.id
  secret_key     = "storage_class_name_${local.tenant_id}"
  secret_value   = local.storage_class_name
}
