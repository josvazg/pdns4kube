package main

import (
	"fmt"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"

	"example.com/pdns4kube/internal/operator"
)

func main() {
	ctx := ctrl.SetupSignalHandler()
	if err := operator.Run(ctx, os.Args[1:], os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
