terraform {
  required_providers {
    netactuate = {
      source  = "netactuate/netactuate"
      version = "~> 0.4"
    }
  }
}

provider "netactuate" {
  # Reads NETACTUATE_API_KEY from the environment when api_key is omitted.
}
