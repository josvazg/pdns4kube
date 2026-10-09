package zone

import (
	"github.com/josvazg/pdns4kube/internal/pdns"
	v1 "github.com/josvazg/pdns4kube/v1"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	constate "github.com/crd2go/constate"
)

func NewPDNSZoneReconciler(
	predicates []predicate.Predicate, pdnsClient *pdns.Client) (*constate.Reconciler[v1.DNSZone], error) {
	zh, err := NewZoneHandler(predicates, pdnsClient)
	if err != nil {
		return nil, err
	}

	return constate.NewStateReconciler(zh), nil
}
