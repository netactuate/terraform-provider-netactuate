resource "netactuate_nke_cluster" "example" {
  contract_id   = 12345
  maximum_nodes = 1
  minimum_nodes = 1
  name          = "example"
  plan          = "vc-1"
  version       = "1.29"
}
