//go:build mage

package main

import (
	"fmt"

	"github.com/magefile/mage/sh"
)

// GenCRD runs openapi2crd to generate the Kubernetes Custom Resource Definitions.
func GenCRD() error {
	fmt.Println("Generating CRDs from OpenAPI spec...")
	return sh.RunV(
		"go", "tool", "openapi2crd",
		"--config", "openapi2crd.yaml",
		"--output", "config/crd/bases/pdns.example.io.yaml",
		"--force",
	)
}

