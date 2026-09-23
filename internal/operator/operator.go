// Package operator contains the pdns4kube operator entrypoint logic,
// factored out of cmd/main.go so it can be unit tested.
package operator

import (
	"context"
	"flag"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"example.com/pdns4kube/internal/controller"
	v1 "example.com/pdns4kube/v1"
)

// Run executes the operator with the given arguments and environment
// lookup. getenv is reserved for future configuration (e.g. environment
// driven overrides) and is intentionally unused for now.
func Run(ctx context.Context, args []string, getenv func(string) string) error {
	_ = getenv // reserved for future config

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	fs := flag.NewFlagSet("pdns4kube", flag.ContinueOnError)
	fs.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	fs.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	fs.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(fs)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	setupLog := ctrl.Log.WithName("setup")

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("get config: %w", err)
	}
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "pdns4kube.pdns.example.io",
	})
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	if err := (&controller.DNSZoneReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setup DNSZone controller: %w", err)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("setup health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("setup ready check: %w", err)
	}

	setupLog.Info("starting manager")
	return mgr.Start(ctx)
}
