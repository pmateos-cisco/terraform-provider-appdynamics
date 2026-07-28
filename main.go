// Command terraform-provider-appdynamics is a Terraform provider for the
// Splunk AppDynamics Alert and Respond APIs (health rules, policies, actions,
// schedules).
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/pmateos/terraform-provider-appdynamics/internal/provider"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(), providerserver.ServeOpts{
		Address: "registry.terraform.io/pmateos/appdynamics",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
