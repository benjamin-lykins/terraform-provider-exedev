package main

import (
	"context"
	"flag"
	"log"

	"github.com/benjamin-lykins/terraform-provider-exedev/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// ProviderAddr is the registry address used by Terraform to identify this provider.
const ProviderAddr = "registry.terraform.io/benjamin-lykins/exedev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: ProviderAddr,
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New, opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
