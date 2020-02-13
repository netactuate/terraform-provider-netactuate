package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/netactuate/terraform-provider-netactuate/netactuate"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: netactuate.Provider})
}
