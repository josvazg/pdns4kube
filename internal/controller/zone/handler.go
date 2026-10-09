package zone

import (
	"context"
	"fmt"

	constate "github.com/crd2go/constate"
	"github.com/josvazg/pdns4kube/internal/pdns"
	v1 "github.com/josvazg/pdns4kube/v1"
	crt "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ZoneHandler struct {
	constate.VersionDispatcher[v1.DNSZone]
	// PDNSZoneClient performs zone mutations against the PowerDNS API.
	PDNSZoneClient  PDNSZoneClient
	client          client.Client
	predicates      []predicate.Predicate
	v0000015Handler *V000015Handler
}

// PDNSZoneClient is the subset of the PowerDNS client used by the zone
// handlers.
type PDNSZoneClient interface {
	CreateZone(ctx context.Context, z pdns.Zone) error
	UpdateZone(ctx context.Context, zoneID string, z pdns.Zone) error
	DeleteZone(ctx context.Context, zoneID string) error
}

func NewZoneHandler(predicates []predicate.Predicate, pdnsClient PDNSZoneClient) (*ZoneHandler, error) {
	v0000015Handler, err := NewV000015Handler(pdnsClient)
	if err != nil {
		return nil, fmt.Errorf("v0_0_15 handler: %w", err)
	}
	zh := &ZoneHandler{
		predicates:      predicates,
		PDNSZoneClient:  pdnsClient,
		v0000015Handler: v0000015Handler,
	}
	zh.VersionDispatcher = *constate.NewVersionDispatcher(zh.HandlerSelector)
	return zh, nil
}

func (zh *ZoneHandler) HandlerSelector(ctx context.Context, obj *v1.DNSZone) (constate.StateHandler[v1.DNSZone], error) {
	if obj.Spec.V0_0_15 != nil {
		return zh.v0000015Handler, nil
	}
	return nil, fmt.Errorf("no valid resource spec version specified")
}

func (zh *ZoneHandler) For() (client.Object, builder.Predicates) {
	obj := &v1.DNSZone{}
	return obj, builder.WithPredicates(zh.predicates...)
}

func (zh *ZoneHandler) SetupWithManager(
	mgr crt.Manager,
	rec reconcile.Reconciler,
	defaultOptions controller.Options) error {
	zh.client = mgr.GetClient()
	return crt.NewControllerManagedBy(mgr).Named(
		"DNSZone").For(zh.For()).WithOptions(defaultOptions).Complete(rec)
}
