terraform {
  required_providers {
    netactuate = {
      source = "netactuate/netactuate"
    }
    external = {
      source = "hashicorp/external"
    }
  }
}

provider "netactuate" {
  api_key = var.api_key
}
