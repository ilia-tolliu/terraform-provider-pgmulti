package main

import (
	"context"
	"flag"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/ilia-tolliu/terraform-provider-pgmulti/internal/provider"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address:         "registry.terraform.io/ilia.tolliu/pgmulti",
		ProtocolVersion: 6,
		Debug:           debug,
	}

	providerserver.Serve(context.Background(), provider.NewPgmulti, opts)
}
