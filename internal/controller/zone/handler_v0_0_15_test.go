// Copyright 2026 Jose Vazquez
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zone

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	constate "github.com/crd2go/constate"
	"github.com/josvazg/pdns4kube/internal/pdns"
	v1 "github.com/josvazg/pdns4kube/v1"
)

func TestV000015Handler(t *testing.T) {
	ctx := context.Background()

	t.Run("HandleInitial creates zone", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{}
		h := newTestHandler(t, pdnsClient)

		res, err := h.HandleInitial(ctx, testDNSZone())
		if err != nil {
			t.Fatalf("HandleInitial() error = %v", err)
		}
		if res.NextState != constate.StateCreated {
			t.Errorf("NextState = %v, want %v", res.NextState, constate.StateCreated)
		}
		if len(pdnsClient.created) != 1 {
			t.Fatalf("created calls = %d, want 1", len(pdnsClient.created))
		}
		want := pdns.Zone{
			Name:        "example.org.",
			Kind:        "Native",
			Nameservers: []string{"ns1.example.org.", "ns2.example.org."},
		}
		if !slices.Equal(pdnsClient.created[0].Nameservers, want.Nameservers) ||
			pdnsClient.created[0].Name != want.Name || pdnsClient.created[0].Kind != want.Kind {
			t.Errorf("created zone = %+v, want %+v", pdnsClient.created[0], want)
		}
	})

	t.Run("HandleInitial missing entry", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{}
		h := newTestHandler(t, pdnsClient)

		res, err := h.HandleInitial(ctx, &v1.DNSZone{})
		if wantErr := "v0_0_15 spec entry is required"; err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error = %v, want substring %q", err, wantErr)
		}
		if res.NextState != constate.StateInitial {
			t.Errorf("NextState = %v, want %v", res.NextState, constate.StateInitial)
		}
		if len(pdnsClient.created) != 0 {
			t.Errorf("created calls = %d, want 0", len(pdnsClient.created))
		}
	})

	t.Run("HandleInitial create error", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{createErr: errors.New("pdns failure")}
		h := newTestHandler(t, pdnsClient)

		res, err := h.HandleInitial(ctx, testDNSZone())
		if wantErr := "create zone: pdns failure"; err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error = %v, want substring %q", err, wantErr)
		}
		if res.NextState != constate.StateInitial {
			t.Errorf("NextState = %v, want %v", res.NextState, constate.StateInitial)
		}
	})

	t.Run("HandleCreated updates zone", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{}
		h := newTestHandler(t, pdnsClient)

		res, err := h.HandleCreated(ctx, testDNSZone())
		if err != nil {
			t.Fatalf("HandleCreated() error = %v", err)
		}
		if res.NextState != constate.StateUpdated {
			t.Errorf("NextState = %v, want %v", res.NextState, constate.StateUpdated)
		}
		if len(pdnsClient.updated) != 1 {
			t.Fatalf("updated calls = %d, want 1", len(pdnsClient.updated))
		}
		if got := pdnsClient.updated[0].id; got != "example.org." {
			t.Errorf("zone ID = %q, want %q", got, "example.org.")
		}
		want := pdns.Zone{
			Name:        "example.org.",
			Kind:        "Native",
			Nameservers: []string{"ns1.example.org.", "ns2.example.org."},
		}
		got := pdnsClient.updated[0].zone
		if !slices.Equal(got.Nameservers, want.Nameservers) ||
			got.Name != want.Name || got.Kind != want.Kind {
			t.Errorf("updated zone = %+v, want %+v", got, want)
		}
	})

	t.Run("HandleUpdated updates zone", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{}
		h := newTestHandler(t, pdnsClient)

		res, err := h.HandleUpdated(ctx, testDNSZone())
		if err != nil {
			t.Fatalf("HandleUpdated() error = %v", err)
		}
		if res.NextState != constate.StateUpdated {
			t.Errorf("NextState = %v, want %v", res.NextState, constate.StateUpdated)
		}
		if len(pdnsClient.updated) != 1 {
			t.Fatalf("updated calls = %d, want 1", len(pdnsClient.updated))
		}
		if got := pdnsClient.updated[0].id; got != "example.org." {
			t.Errorf("zone ID = %q, want %q", got, "example.org.")
		}
		want := pdns.Zone{
			Name:        "example.org.",
			Kind:        "Native",
			Nameservers: []string{"ns1.example.org.", "ns2.example.org."},
		}
		got := pdnsClient.updated[0].zone
		if !slices.Equal(got.Nameservers, want.Nameservers) ||
			got.Name != want.Name || got.Kind != want.Kind {
			t.Errorf("updated zone = %+v, want %+v", got, want)
		}
	})

	t.Run("HandleCreated missing entry", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{}
		h := newTestHandler(t, pdnsClient)

		_, err := h.HandleCreated(ctx, &v1.DNSZone{})
		if wantErr := "v0_0_15 spec entry is required"; err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error = %v, want substring %q", err, wantErr)
		}
	})

	t.Run("HandleUpdated missing entry", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{}
		h := newTestHandler(t, pdnsClient)

		_, err := h.HandleUpdated(ctx, &v1.DNSZone{})
		if wantErr := "v0_0_15 spec entry is required"; err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error = %v, want substring %q", err, wantErr)
		}
	})

	t.Run("HandleCreated update error", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{updateErr: errors.New("pdns failure")}
		h := newTestHandler(t, pdnsClient)

		_, err := h.HandleCreated(ctx, testDNSZone())
		if wantErr := "update zone: pdns failure"; err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error = %v, want substring %q", err, wantErr)
		}
	})

	t.Run("HandleUpdated update error", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{updateErr: errors.New("pdns failure")}
		h := newTestHandler(t, pdnsClient)

		_, err := h.HandleUpdated(ctx, testDNSZone())
		if wantErr := "update zone: pdns failure"; err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error = %v, want substring %q", err, wantErr)
		}
	})

	t.Run("HandleDeletionRequested deletes zone", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{}
		h := newTestHandler(t, pdnsClient)

		res, err := h.HandleDeletionRequested(ctx, testDNSZone())
		if err != nil {
			t.Fatalf("HandleDeletionRequested() error = %v", err)
		}
		if res.NextState != constate.StateDeleted {
			t.Errorf("NextState = %v, want %v", res.NextState, constate.StateDeleted)
		}
		if got := pdnsClient.deleted; len(got) != 1 || got[0] != "example.org." {
			t.Errorf("deleted = %v, want [example.org.]", got)
		}
	})

	t.Run("HandleDeletionRequested missing entry", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{}
		h := newTestHandler(t, pdnsClient)

		res, err := h.HandleDeletionRequested(ctx, &v1.DNSZone{})
		if wantErr := "v0_0_15 spec entry is required"; err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error = %v, want substring %q", err, wantErr)
		}
		if res.NextState != constate.StateDeletionRequested {
			t.Errorf("NextState = %v, want %v", res.NextState, constate.StateDeletionRequested)
		}
		if len(pdnsClient.deleted) != 0 {
			t.Errorf("deleted calls = %d, want 0", len(pdnsClient.deleted))
		}
	})

	t.Run("HandleDeletionRequested delete error", func(t *testing.T) {
		pdnsClient := &fakePDNSZoneClient{deleteErr: errors.New("pdns failure")}
		h := newTestHandler(t, pdnsClient)

		_, err := h.HandleDeletionRequested(ctx, testDNSZone())
		if wantErr := "delete zone: pdns failure"; err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error = %v, want substring %q", err, wantErr)
		}
	})
}

func TestZoneFromEntry(t *testing.T) {
	h := newTestHandler(t, &fakePDNSZoneClient{})

	t.Run("translates valid CR fields", func(t *testing.T) {
		obj := testDNSZone()

		got, err := h.zoneFromEntry(obj)
		if err != nil {
			t.Fatalf("zoneFromEntry() error = %v", err)
		}
		want := pdns.Zone{
			Name:        "example.org.",
			Kind:        "Native",
			Nameservers: []string{"ns1.example.org.", "ns2.example.org."},
		}
		if !slices.Equal(got.Nameservers, want.Nameservers) ||
			got.Name != want.Name || got.Kind != want.Kind {
			t.Errorf("zone = %+v, want %+v", got, want)
		}
	})

	t.Run("absent optional nameservers", func(t *testing.T) {
		obj := testDNSZone()
		obj.Spec.V0_0_15.Entry.Nameservers = nil

		got, err := h.zoneFromEntry(obj)
		if err != nil {
			t.Fatalf("zoneFromEntry() error = %v", err)
		}
		if got.Name != "example.org." || got.Kind != "Native" {
			t.Errorf("zone = %+v, want name %q kind %q", got, "example.org.", "Native")
		}
		if len(got.Nameservers) != 0 {
			t.Errorf("nameservers = %v, want empty", got.Nameservers)
		}
	})

	t.Run("missing versioned entry", func(t *testing.T) {
		obj := &v1.DNSZone{}

		_, err := h.zoneFromEntry(obj)
		if wantErr := "v0_0_15 spec entry is required"; err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error = %v, want substring %q", err, wantErr)
		}
	})

	t.Run("malformed versioned entry", func(t *testing.T) {
		obj := &v1.DNSZone{
			Spec: v1.DNSZoneSpec{
				V0_0_15: &v1.DNSZoneSpecV0_0_15{},
			},
		}

		_, err := h.zoneFromEntry(obj)
		if wantErr := "v0_0_15 spec entry is required"; err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error = %v, want substring %q", err, wantErr)
		}
	})

	t.Run("translation error for mismatched GVK", func(t *testing.T) {
		obj := testDNSZone()
		obj.APIVersion = "other.example.io/v1"
		obj.Kind = "DNSZone"

		_, err := h.zoneFromEntry(obj)
		if err == nil {
			t.Fatal("zoneFromEntry() error = nil, want translation error")
		}
		if wantErr := "translate v0_0_15 entry"; !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error = %v, want substring %q", err, wantErr)
		}
	})
}

// newTestHandler builds a V000015Handler through the constructor.
func newTestHandler(t *testing.T, pdnsClient PDNSZoneClient) *V000015Handler {
	t.Helper()
	h, err := NewV000015Handler(pdnsClient)
	if err != nil {
		t.Fatalf("NewV000015Handler() error = %v", err)
	}
	return h
}

// fakePDNSZoneClient records PDNS calls made by the handler.
type fakePDNSZoneClient struct {
	created   []pdns.Zone
	updated   []updateCall
	deleted   []string
	createErr error
	updateErr error
	deleteErr error
}

type updateCall struct {
	id   string
	zone pdns.Zone
}

func (f *fakePDNSZoneClient) CreateZone(ctx context.Context, z pdns.Zone) error {
	f.created = append(f.created, z)
	return f.createErr
}

func (f *fakePDNSZoneClient) UpdateZone(ctx context.Context, zoneID string, z pdns.Zone) error {
	f.updated = append(f.updated, updateCall{id: zoneID, zone: z})
	return f.updateErr
}

func (f *fakePDNSZoneClient) DeleteZone(ctx context.Context, zoneID string) error {
	f.deleted = append(f.deleted, zoneID)
	return f.deleteErr
}

// testDNSZone returns a DNSZone with a v0_0_15 entry for handler tests.
func testDNSZone() *v1.DNSZone {
	return &v1.DNSZone{
		Spec: v1.DNSZoneSpec{
			V0_0_15: &v1.DNSZoneSpecV0_0_15{
				Entry: &v1.Entry{
					Name:        "example.org.",
					Kind:        "Native",
					Nameservers: &[]string{"ns1.example.org.", "ns2.example.org."},
				},
			},
		},
	}
}
