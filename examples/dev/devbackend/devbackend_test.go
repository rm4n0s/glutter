package devbackend

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestName(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    string
		wantTag string
	}{
		{"nothing set", nil, "wayland", ""},
		{"a Wayland session", map[string]string{"WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":0"}, "wayland", ""},
		{"an X session", map[string]string{"DISPLAY": ":0"}, "x11", ""},
		{"asked for X11", map[string]string{BackendEnv: "x11", "WAYLAND_DISPLAY": "wayland-0"}, "x11", ""},
		{"asked for Wayland", map[string]string{BackendEnv: "wayland", "DISPLAY": ":0"}, "wayland", ""},
		{"something else", map[string]string{BackendEnv: "quartz"}, "", "DevBackendUnknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Name(func(k string) string { return tt.env[k] })
			if tt.wantTag != "" {
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tt.wantTag || !appErr.HasRoute("Name.DevBackendUnknown") {
					t.Fatalf("error = %v, want %s via Name", err, tt.wantTag)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("Name = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestNewRefusesShmOnX11(t *testing.T) {
	t.Setenv(BackendEnv, "x11")
	_, err := New(true)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "DevBackendUnsupported" || !appErr.HasRoute("New.DevBackendUnsupported") {
		t.Fatalf("New(true) = %v, want DevBackendUnsupported via New", err)
	}
}
