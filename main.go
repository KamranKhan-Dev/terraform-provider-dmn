package main

import (
	"context"
	"flag"
	"log"

	"github.com/KamranKhan-Dev/terraform-provider-dmn/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is set via -ldflags at release time.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/KamranKhan-Dev/dmn",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
