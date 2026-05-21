### Cluster

output "cluster_id" {
  description = "NKE cluster ID"
  value       = netactuate_nke_cluster.pop.cluster_id
}

output "cluster_api_url" {
  description = "Kubernetes API server URL"
  value       = netactuate_nke_cluster.pop.api_url
}

output "cluster_status" {
  description = "Cluster status"
  value       = netactuate_nke_cluster.pop.status
}

output "kubeconfig_path" {
  description = "Path to the written kubeconfig file"
  value       = "${path.module}/kubeconfig.yaml"
}

output "pod_network" {
  description = "Pod network CIDR"
  value       = netactuate_nke_cluster.pop.pod_network
}

output "service_network" {
  description = "Service network CIDR"
  value       = netactuate_nke_cluster.pop.service_network
}

output "worker_nodes" {
  description = "Worker node list"
  value       = terraform_data.worker_nodes.output
}

### Ceph RBD storage

output "rbd_monitors" {
  description = "Ceph monitor endpoints — use in the CSI ConfigMap"
  value       = netactuate_storage_block_namespace.rbd.endpoints
}

output "rbd_pool" {
  description = "Ceph RBD pool name — use in the CSI StorageClass"
  value       = local.rbd_pool_name
}

output "rbd_namespace" {
  description = "Ceph RBD namespace"
  value       = netactuate_storage_block_namespace.rbd.storage_namespace
}

output "rbd_cluster_id" {
  description = "Ceph cluster ID — use as clusterID in the CSI ConfigMap"
  value       = netactuate_storage_block_namespace.rbd.storage_cluster_id
}

output "rbd_user_id" {
  description = "Ceph user ID for tenant namespace authentication"
  value       = netactuate_storage_block_namespace.rbd.user_key
  sensitive   = true
}

output "rbd_user_key" {
  description = "Ceph user key for tenant namespace authentication"
  value       = netactuate_storage_block_namespace.rbd.secret_key
  sensitive   = true
}

output "rbd_namespace_id" {
  description = "Block namespace ID (used as tenant ID in generated Kubernetes objects)"
  value       = netactuate_storage_block_namespace.rbd.block_namespace_id
}

### Ceph CSI ConfigMap
# Apply with: terraform output -raw ceph_csi_config | kubectl apply -f -
# See: https://netactuate.com/docs/infrastructure/storage/storage-rbd

output "ceph_csi_config" {
  description = "JSON config for the ceph-csi-config ConfigMap data.config field"
  value = jsonencode([{
    clusterID = netactuate_storage_block_namespace.rbd.storage_cluster_id
    monitors  = netactuate_storage_block_namespace.rbd.endpoints
    rbd = {
      radosNamespace = netactuate_storage_block_namespace.rbd.storage_namespace
    }
  }])
}

output "ceph_csi_values" {
  description = "Resolved values to substitute into the NetActuate Ceph CSI manifest template"
  sensitive   = true
  value = {
    cluster_id         = netactuate_storage_block_namespace.rbd.storage_cluster_id
    monitors           = netactuate_storage_block_namespace.rbd.endpoints
    pool               = local.rbd_pool_name
    rados_namespace    = netactuate_storage_block_namespace.rbd.storage_namespace
    user_id            = netactuate_storage_block_namespace.rbd.user_key
    user_key           = netactuate_storage_block_namespace.rbd.secret_key
    tenant_namespace   = local.tenant_namespace
    csi_secret_name    = local.csi_secret_name
    storage_class_name = local.storage_class_name
    pvc_name           = local.pvc_name
    pod_name           = local.pod_name
    pvc_size           = var.test_pvc_size
  }
}

output "csi_driver_manifest_template_path" {
  description = "Path to the CSI driver template file in this example"
  value       = "${path.module}/ceph-csi-rbd-driver-template.yaml"
}

output "csi_driver_manifest" {
  description = "Rendered Ceph CSI driver manifest with live cluster/monitor/namespace values"
  value = replace(
    replace(
      replace(
        file("${path.module}/ceph-csi-rbd-driver-template.yaml"),
        "<CLUSTER_ID>",
        netactuate_storage_block_namespace.rbd.storage_cluster_id
      ),
      "<MON_ADDR>",
      netactuate_storage_block_namespace.rbd.endpoints[0]
    ),
    "<RADOS_NAMESPACE>",
    netactuate_storage_block_namespace.rbd.storage_namespace
  )
}

output "csi_driver_manifest_apply_command" {
  description = "Command to render and apply the Ceph CSI driver manifest before tenant storage objects"
  value       = "terraform output -raw csi_driver_manifest > ceph-csi-rbd-driver-${netactuate_storage_block_namespace.rbd.block_namespace_id}.yaml && kubectl apply -f ceph-csi-rbd-driver-${netactuate_storage_block_namespace.rbd.block_namespace_id}.yaml"
}

output "tenant_storage_manifest_template_path" {
  description = "Path to the tenant storage manifest template file in this example"
  value       = "${path.module}/ceph-csi-rbd-tenant-template.yaml.tmpl"
}

output "tenant_storage_manifest" {
  description = "Rendered tenant storage manifest (Secret, StorageClass, PVC, Pod) aligned to NetActuate RBD CSI docs"
  sensitive   = true
  value = templatefile("${path.module}/ceph-csi-rbd-tenant-template.yaml.tmpl", {
    tenant_namespace   = local.tenant_namespace
    csi_secret_name    = local.csi_secret_name
    storage_class_name = local.storage_class_name
    cluster_id         = netactuate_storage_block_namespace.rbd.storage_cluster_id
    pool               = local.rbd_pool_name
    rbd_user_id        = netactuate_storage_block_namespace.rbd.user_key
    rbd_user_key       = netactuate_storage_block_namespace.rbd.secret_key
    rados_namespace    = netactuate_storage_block_namespace.rbd.storage_namespace
    pvc_name           = local.pvc_name
    pvc_size           = var.test_pvc_size
    pod_name           = local.pod_name
  })
}

output "tenant_storage_manifest_apply_command" {
  description = "Command to render and apply tenant storage objects after CSI driver install"
  value       = "terraform output -raw tenant_storage_manifest > ceph-csi-rbd-tenant-${netactuate_storage_block_namespace.rbd.block_namespace_id}.yaml && kubectl apply -f ceph-csi-rbd-tenant-${netactuate_storage_block_namespace.rbd.block_namespace_id}.yaml"
}

output "secret_list_id" {
  description = "Secret list ID containing RBD/CSI connection and auth values"
  value       = netactuate_secret_list.k8s_rbd.id
}

### Ansible vars

output "ansible_k8s_vars" {
  description = "Structured vars for an Ansible Kubernetes role"
  sensitive   = true
  value = {
    kubeconfig  = "${path.module}/kubeconfig.yaml"
    cluster_api = netactuate_nke_cluster.pop.api_url

    rbd_monitors   = netactuate_storage_block_namespace.rbd.endpoints
    rbd_pool       = local.rbd_pool_name
    rbd_namespace  = netactuate_storage_block_namespace.rbd.storage_namespace
    rbd_cluster_id = netactuate_storage_block_namespace.rbd.storage_cluster_id
    rbd_user_id    = netactuate_storage_block_namespace.rbd.user_key
    rbd_user_key   = netactuate_storage_block_namespace.rbd.secret_key

    tenant_namespace   = local.tenant_namespace
    csi_secret_name    = local.csi_secret_name
    storage_class_name = local.storage_class_name
    pvc_name           = local.pvc_name
    pod_name           = local.pod_name
    pvc_size           = var.test_pvc_size
  }
}
