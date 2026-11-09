//go:build darwin && !ios

package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestBackendNamesOnMacOS(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		tag     string
	}{
		{"wayland is not on macOS", "wayland", "BackendUnavailable"},
		{"x11 is not on macOS", "x11", "BackendUnavailable"},
		{"an unknown backend", "quartz", "BackendUnknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run(context.Background(), Config{Logger: quiet(), Backend: tt.backend})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute("Run->connectDisplay."+tt.tag) {
				t.Errorf("got %s via %s, want %s via Run->connectDisplay", appErr.Tag, appErr.Route(), tt.tag)
			}
		})
	}
}

func TestMacOSRenderersAreMetalThenSoft(t *testing.T) {
	cands := rendererCandidates()
	if len(cands) != 2 || cands[0].Name != "metal" || cands[1].Name != "soft" {
		t.Fatalf("candidates = %v, want metal then soft", cands)
	}
	for _, name := range []string{"gles", "vulkan", "d3d12", "webgpu", "webgl"} {
		_, err := newChooser(&Config{}, quiet(), name, cands)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "RendererUnavailable" || !appErr.HasRoute("newChooser.RendererUnavailable") {
			t.Errorf("%s: err = %v, want RendererUnavailable", name, err)
		}
	}
	for _, name := range []string{"metal", "soft", "auto", ""} {
		if _, err := newChooser(&Config{}, quiet(), name, cands); err != nil {
			t.Errorf("%q: %v", name, err)
		}
	}
	_, err := newChooser(&Config{}, quiet(), "bogus", cands)
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "RendererUnknown" {
		t.Errorf("bogus: err = %v, want RendererUnknown", err)
	}
}

func TestDefaultShaderCacheDirOnMacOS(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"the caches of the user", map[string]string{"HOME": "/Users/ada"}, filepath.Join("/Users/ada", "Library", "Caches", "glutter", "shaders")},
		{"XDG_CACHE_HOME is not used on macOS", map[string]string{"HOME": "/Users/ada", "XDG_CACHE_HOME": "/elsewhere"}, filepath.Join("/Users/ada", "Library", "Caches", "glutter", "shaders")},
		{"the cache is off", map[string]string{"HOME": "/Users/ada", "GLUTTER_SHADER_CACHE": "0"}, ""},
		{"no home", map[string]string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultShaderCacheDir(func(k string) string { return tt.env[k] }); got != tt.want {
				t.Errorf("defaultShaderCacheDir = %q, want %q", got, tt.want)
			}
		})
	}
}
