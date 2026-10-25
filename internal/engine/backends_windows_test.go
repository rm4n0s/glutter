//go:build windows

package engine

import (
	"context"
	"testing"
)

func TestConnectDisplayChoosesTheBackendOfWindows(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		env     string
		tag     string
		route   string
	}{
		{"wayland is not on Windows", "wayland", "", "BackendUnavailable", "Run->connectDisplay.BackendUnavailable"},
		{"x11 is not on Windows", "x11", "", "BackendUnavailable", "Run->connectDisplay.BackendUnavailable"},
		{"an unknown name", "metal", "", "BackendUnknown", "Run->connectDisplay.BackendUnknown"},
		{"an unknown name in the environment", "", "metal", "BackendUnknown", "Run->connectDisplay.BackendUnknown"},
		{"wayland in the environment", "", "wayland", "BackendUnavailable", "Run->connectDisplay.BackendUnavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := map[string]string{backendEnv: tt.env}
			cfg := Config{Logger: quiet(), Backend: tt.backend, Getenv: func(k string) string { return env[k] }}
			assertRoute(t, Run(context.Background(), cfg), tt.tag, tt.route)
		})
	}
}
