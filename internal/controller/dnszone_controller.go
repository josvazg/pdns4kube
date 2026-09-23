package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "example.com/pdns4kube/v1"
)

// DNSZoneReconciler reconciles a DNSZone object.
type DNSZoneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pdns.example.io,resources=dnszones,verbs=get;list;watch
// +kubebuilder:rbac:groups=pdns.example.io,resources=dnszones/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pdns.example.io,resources=dnszones/finalizers,verbs=update

// Reconcile reconciles DNSZone objects.
func (r *DNSZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var zone v1.DNSZone
	if err := r.Get(ctx, req.NamespacedName, &zone); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// TODO: implement DNSZone reconciliation logic (sync zone state with PowerDNS).
	logger.Info("reconciling DNSZone", "name", req.NamespacedName)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DNSZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.DNSZone{}).
		Named("dnszone").
		Complete(r)
}
