package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"striveworks.us/terraform-provider-k3s/internal/provider"
)

var version string = "0.2.3"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.opentofu.org/striveworks/k3s",
		Debug:   debug,
	})

	if err != nil {
		log.Fatal(err.Error())
	}
}
