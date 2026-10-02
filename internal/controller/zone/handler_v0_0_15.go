package zone

import (
	"context"
	"fmt"
	"log"

	"example.com/pdns4kube/internal/pdns"
	v1 "example.com/pdns4kube/v1"
	constate "github.com/crd2go/constate"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type V000015Handler struct {
	constate.FallbackHandler[v1.DNSZone]
	// pdns performs zone mutations against the PowerDNS API.
	pdns PDNSZoneClient
}

func (h *V000015Handler) For() (client.Object, builder.Predicates) {
	return nil, builder.Predicates{}
}

// zoneFromEntry converts the v0_0_15 spec entry into a pdns.Zone request.
func zoneFromEntry(obj *v1.DNSZone) (pdns.Zone, error) {
	if obj.Spec.V0_0_15 == nil || obj.Spec.V0_0_15.Entry == nil {
		return pdns.Zone{}, fmt.Errorf("v0_0_15 spec entry is required")
	}
	entry := obj.Spec.V0_0_15.Entry
	z := pdns.Zone{
		Name: entry.Name,
		Kind: entry.Kind,
	}
	if entry.Nameservers != nil {
		z.Nameservers = *entry.Nameservers
	}
	return z, nil
}

func (h *V000015Handler) HandleInitial(ctx context.Context, obj *v1.DNSZone) (constate.Result, error) {
	z, err := zoneFromEntry(obj)
	if err != nil {
		return constate.ErrorState(constate.StateInitial, err)
	}
	if err := h.pdns.CreateZone(ctx, z); err != nil {
		return constate.ErrorState(constate.StateInitial, fmt.Errorf("create zone: %w", err))
	}
	return constate.NextState(constate.StateCreated, "zone created in pdns")
}

func (h *V000015Handler) HandleCreated(ctx context.Context, obj *v1.DNSZone) (constate.Result, error) {
	return h.upsert(ctx, obj)
}

func (h *V000015Handler) HandleUpdated(ctx context.Context, obj *v1.DNSZone) (constate.Result, error) {
	return h.upsert(ctx, obj)
}

func (h *V000015Handler) HandleDeletionRequested(ctx context.Context, obj *v1.DNSZone) (constate.Result, error) {
	if obj.Spec.V0_0_15 == nil || obj.Spec.V0_0_15.Entry == nil {
		err := fmt.Errorf("v0_0_15 spec entry is required")
		return constate.ErrorState(constate.StateDeletionRequested, err)
	}
	if err := h.pdns.DeleteZone(ctx, obj.Spec.V0_0_15.Entry.Name); err != nil {
		return constate.ErrorState(constate.StateDeletionRequested, fmt.Errorf("delete zone: %w", err))
	}
	return constate.NextState(constate.StateDeleted, "zone deleted from pdns")
}

func (h *V000015Handler) upsert(ctx context.Context, obj *v1.DNSZone) (constate.Result, error) {
	z, err := zoneFromEntry(obj)
	if err != nil {
		return constate.ErrorState(constate.StateCreated, err)
	}
	log.Printf("updatign with %v", z)
	if err := h.pdns.UpdateZone(ctx, z.Name, z); err != nil {
		return constate.ErrorState(constate.StateCreated, fmt.Errorf("update zone: %w", err))
	}
	return constate.NextState(constate.StateUpdated, "zone updated in pdns")
}
