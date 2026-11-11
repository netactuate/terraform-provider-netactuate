data "netactuate_nke_lb_services" "example" {
  cluster_id = netactuate_nke_cluster.example.cluster_id
}

output "lb_addresses" {
  value = { for s in data.netactuate_nke_lb_services.example.services : "${s.kind}/${s.namespace}/${s.name}" => s.vip }
}
