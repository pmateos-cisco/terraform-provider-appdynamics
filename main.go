// Command terraform-provider-appdynamics is a Terraform provider for the
// Splunk AppDynamics Alert and Respond APIs (health rules, policies, actions,
// schedules).
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/pmateos-cisco/terraform-provider-appdynamics/internal/provider"
)

// version is set by the release build via -ldflags "-X main.version=...";
// it stays "dev" for local builds.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/pmateos-cisco/appdynamics",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
