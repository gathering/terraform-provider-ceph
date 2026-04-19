//go:build tools

package tools

import (
	// keep terraform-plugin-docs in go.mod for `go generate`
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
