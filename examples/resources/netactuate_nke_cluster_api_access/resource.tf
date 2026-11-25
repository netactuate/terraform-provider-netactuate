resource "netactuate_nke_cluster_api_access" "example" {
  cluster_id               = 12345
  public_enabled           = true
  private_endpoint_enabled = true
}
