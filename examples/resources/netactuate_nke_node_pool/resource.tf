resource "netactuate_nke_node_pool" "example" {
  cluster_id = 12345
  name       = "nah-workers"
  type       = "managed"
  package_id = 42
  autoscaling {
    enabled       = false
    minimum_nodes = 0
    maximum_nodes = 3
  }
  desired_nodes = 1
  labels = {
    workload = "batch"
  }
  taints {
    key    = "dedicated"
    value  = "batch"
    effect = "NoSchedule"
  }
  timeouts {
    create = "30m"
    update = "30m"
    delete = "30m"
  }
}
