package main

import (
	"github.com/EfficientIP-Labs/terraform-provider-solidserver/solidserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: solidserver.Provider,
	})
}
