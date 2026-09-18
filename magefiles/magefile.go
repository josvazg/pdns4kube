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

// GenGo runs crd2go and controller-gen to generate the Go types from the CRDs
func GenGo() error {
	fmt.Println("Generating Go types from CRDs...")
	err := sh.RunV("go", "tool", "crd2go",
                "--input", "config/crd/bases/pdns.example.io.yaml",
		"--output", "v1")
	if err != nil {
		return fmt.Errorf("CRD2Go failed: %w", err)
	}
        return sh.RunV("go", "tool", "controller-gen", "object", "paths=./v1/...")
}

