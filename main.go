package main

import (
	"context"

	"github.com/gathering/terraform-provider-ceph/ceph"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name ceph

func main() {
	if err := providerserver.Serve(context.Background(), ceph.New(), providerserver.ServeOpts{
		Address: "registry.terraform.io/gathering/ceph",
	}); err != nil {
		panic(err)
	}
}
