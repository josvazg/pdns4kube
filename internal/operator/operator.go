// Package operator contains the pdns4kube operator entrypoint logic,
// factored out of cmd/main.go so it can be unit tested.
package operator

import (
	"context"
	"flag"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"example.com/pdns4kube/internal/controller/zone"
	"example.com/pdns4kube/internal/pdns"
	v1 "example.com/pdns4kube/v1"
)

type operatorCfg struct {
	proveAddr            string
	metricsAddr          string
	enableLeaderElection bool
	pdnsAPIURL           string
	pdnsAPIKey           string
	zopts                zap.Options
}

// Run executes the operator with the given arguments and environment
// lookup. getenv is used for environment driven configuration; when nil
// it falls back to os.Getenv.
func Run(ctx context.Context, args []string, getenv func(string) string) error {
	if getenv == nil {
		getenv = os.Getenv
	}
	opConfig, err := parseFlagsConfig(args, getenv)
	if err != nil {
		return fmt.Errorf("failed to parse config args: %w", err)
	}

	mgr, err := createManager(opConfig)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	if err := registerReconcilers(mgr, opConfig); err != nil {
		return fmt.Errorf("failed to register reconcilers: %w", err)
	}

	if err := setupChecks(mgr); err != nil {
		return fmt.Errorf("failed to setup operator checks: %w", err)
	}

	setupLog := ctrl.Log.WithName("setup")
	setupLog.Info("starting manager")
	return mgr.Start(ctx)
}

func parseFlagsConfig(args []string, getenv func(string) string) (*operatorCfg, error) {
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
		return nil, fmt.Errorf("parse flags: %w", err)
	}
	return &operatorCfg{
		proveAddr:            probeAddr,
		metricsAddr:          metricsAddr,
		enableLeaderElection: enableLeaderElection,
		pdnsAPIURL:           getenv("PDNS_API_URL"),
		pdnsAPIKey:           getenv("PDNS_API_KEY"),
		zopts:                opts,
	}, nil
}

func createManager(opConfig *operatorCfg) (manager.Manager, error) {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opConfig.zopts)))

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}
	return ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: opConfig.metricsAddr},
		HealthProbeBindAddress: opConfig.proveAddr,
		LeaderElection:         opConfig.enableLeaderElection,
		LeaderElectionID:       "pdns4kube.pdns.example.io",
	})
}

func setupChecks(mgr manager.Manager) error {
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("setup health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("setup ready check: %w", err)
	}
	return nil
}

func registerReconcilers(mgr manager.Manager, opConfig *operatorCfg) error {
	if opConfig.pdnsAPIURL == "" {
		return fmt.Errorf("PDNS_API_URL is required")
	}
	if opConfig.pdnsAPIKey == "" {
		return fmt.Errorf("PDNS_API_KEY is required")
	}
	pdnsClient := pdns.NewClient(opConfig.pdnsAPIURL, opConfig.pdnsAPIKey, nil)
	reconciler, err := zone.NewPDNSZoneReconciler(zone.Predicates(), pdnsClient)
	if err != nil {
		return fmt.Errorf("error creating DNSZone controller: %w", err)
	}
	if err := reconciler.SetupWithManager(mgr, controller.Options{}); err != nil {
		return fmt.Errorf("failed setup DNSZone controller: %w", err)
	}
	return nil
}
