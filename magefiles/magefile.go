//go:build mage

package main

import (
	"fmt"
	"strings"

	"github.com/magefile/mage/mg"
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

// KindUp creates the kind cluster named pdns4kube if it is not already running.
func KindUp() error {
	out, err := sh.Output("kind", "get", "clusters")
	if err != nil {
		return fmt.Errorf("kind get clusters failed: %w", err)
	}
	if strings.Contains(out, "pdns4kube") {
		fmt.Println("cluster pdns4kube already running")
		return nil
	}
	return sh.RunV("kind", "create", "cluster", "--name", "pdns4kube")
}

// KindDown deletes the kind cluster named pdns4kube.
func KindDown() error {
	return sh.RunV("kind", "delete", "cluster", "--name", "pdns4kube")
}

// InstallCRD applies the generated CRDs to the current cluster.
func InstallCRD() error {
	fmt.Println("Installing CRDs...")
	return sh.RunV("kubectl", "apply", "-f", "config/crd/bases/pdns.example.io.yaml")
}

// Run runs the operator locally, outside of the cluster, starting the kind cluster first if needed
// and installing the CRDs before running.
func Run() error {
	mg.SerialDeps(KindUp, InstallCRD)
	fmt.Println("Running operator locally...")
	return sh.RunV("go", "run", "./cmd")
}

