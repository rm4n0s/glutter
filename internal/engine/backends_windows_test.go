//go:build windows

package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/d3d12"
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

// The directory of the shader cache on Windows is glutter\shaders under %LOCALAPPDATA%; XDG_CACHE_HOME
// and HOME, which Wine passes through from the host, are not read (SPEC §9.7).
func TestDefaultShaderCacheDirIsUnderLocalAppData(t *testing.T) {
	local := filepath.Join("C:\\Users", "u", "AppData", "Local")
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"under LocalAppData", map[string]string{"LocalAppData": local}, filepath.Join(local, "glutter", "shaders")},
		{"XDG_CACHE_HOME and HOME are not read", map[string]string{"LocalAppData": local, "XDG_CACHE_HOME": "/x", "HOME": "/h"}, filepath.Join(local, "glutter", "shaders")},
		{"only HOME", map[string]string{"HOME": "/h"}, ""},
		{"turned off", map[string]string{"GLUTTER_SHADER_CACHE": "0", "LocalAppData": local}, ""},
		{"nothing", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultShaderCacheDir(func(k string) string { return tt.env[k] }); got != tt.want {
				t.Errorf("defaultShaderCacheDir = %q, want %q", got, tt.want)
			}
		})
	}
}

// With the real environment: os.UserCacheDir of Windows is %LOCALAPPDATA%, and the engine's directory
// is under it.
func TestDefaultShaderCacheDirAgreesWithUserCacheDir(t *testing.T) {
	base, err := os.UserCacheDir()
	if err != nil {
		t.Skipf("no LocalAppData in this environment: %v", err)
	}
	if got, want := defaultShaderCacheDir(os.Getenv), filepath.Join(base, "glutter", "shaders"); got != want {
		t.Errorf("defaultShaderCacheDir = %q, os.UserCacheDir gives %q", got, want)
	}
}

// The engine does not import package d3d12 in the files that the other hosts build, so it names the variable
// of the adapter itself (windows_candidates.go); here the two are compared.
func TestD3D12AdapterEnvIsTheDevicesOwn(t *testing.T) {
	if d3d12AdapterEnv != d3d12.AdapterEnvVar {
		t.Errorf("the engine reads %q, the device %q", d3d12AdapterEnv, d3d12.AdapterEnvVar)
	}
}

// The renderers of Windows are the three of SPEC §9.6, in that order.
func TestWindowsRendererCandidates(t *testing.T) {
	var names []string
	for _, c := range rendererCandidates() {
		names = append(names, c.label())
	}
	if got := strings.Join(names, ","); got != "d3d12,d3d12 (warp),soft" {
		t.Errorf("candidates %q, want d3d12,d3d12 (warp),soft", got)
	}
}
