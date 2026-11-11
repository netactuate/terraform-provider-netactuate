terraform {
  required_providers {
    netactuate = {
      source  = "registry.terraform.io/netactuate/netactuate"
      version = ">= 0.4.10"
    }
  }
}

provider "netactuate" {} # reads NETACTUATE_API_KEY
