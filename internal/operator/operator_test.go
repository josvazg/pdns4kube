package operator

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
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
			name: "missing kubeconfig",
			env: map[string]string{
				"KUBECONFIG": filepath.Join(t.TempDir(), "does-not-exist.yaml"),
			},
			wantErr: "get config",
		},
		{
			name: "unknown flag in otherwise valid args",
			args: []string{"-metrics-bind-address", ":9090", "-nope"},
			env: map[string]string{
				"KUBECONFIG": filepath.Join(t.TempDir(), "does-not-exist.yaml"),
			},
			wantErr: "parse flags",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string { return tt.env[key] }
			err := Run(context.Background(), tt.args, getenv)
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
