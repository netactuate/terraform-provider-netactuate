terraform {
  required_providers {
    netactuate = {
      source = "netactuate/netactuate"
    }
  }
}

provider "netactuate" {
  api_key = var.api_key
}
