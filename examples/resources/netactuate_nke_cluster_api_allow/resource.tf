resource "netactuate_nke_cluster_api_allow" "example" {
  cluster_id = 12345
  cidr       = "192.0.2.0/24"
  label      = "office"
}
