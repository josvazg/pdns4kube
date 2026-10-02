package zone

import (
	"example.com/pdns4kube/internal/pdns"
	v1 "example.com/pdns4kube/v1"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	constate "github.com/crd2go/constate"
)

func NewPDNSZoneReconciler(
	predicates []predicate.Predicate, pdnsClient *pdns.Client) (*constate.Reconciler[v1.DNSZone], error) {
	zh := NewZoneHandler(predicates, pdnsClient)

	return constate.NewStateReconciler(zh), nil
}
