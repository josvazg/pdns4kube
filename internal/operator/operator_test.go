package operator

import (
	"context"
	"errors"
	"strings"
	"testing"

	"k8s.io/client-go/rest"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		wantErr string
	}{
		{
			name:    "unknown flag",
			args:    []string{"-unknown"},
			wantErr: "parse flags",
		},
		{
			name:    "bad flag value",
			args:    []string{"-leader-elect=maybe"},
			wantErr: "parse flags",
		},
		{
			name:    "get config failure",
			wantErr: "get config",
		},
		{
			name:    "unknown flag in otherwise valid args",
			args:    []string{"-metrics-bind-address", ":9090", "-nope"},
			wantErr: "parse flags",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string { return tt.env[key] }
			// All test cases exit before the manager starts, so inject a
			// getConfig that always fails; flag parse errors take precedence
			// over it.
			getConfig := func() (*rest.Config, error) {
				return nil, errors.New("injected config failure")
			}
			err := run(context.Background(), tt.args, getenv, getConfig)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Run() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Run() error = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Run() error = %q, want error containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}
