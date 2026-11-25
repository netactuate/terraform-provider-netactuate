resource "netactuate_nke_autoscaler_addon" "example" {
  cluster_id = 12345
  config {
    scale_down_enabled               = true
    scale_down_unneeded_time         = "10m"
    scale_down_delay_after_add       = "10m"
    scale_down_utilization_threshold = 0.5
    max_node_provision_time          = "15m"
    expander                        = "least-waste"
    skip_nodes_with_local_storage    = true
    balance_similar_node_groups      = false
    max_nodes_total                  = 20
  }
}
