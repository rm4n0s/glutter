//go:build darwin && !ios

package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestRunHasNoMacOSBackendYet(t *testing.T) {
	err := Run(context.Background(), Config{Logger: quiet()})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != "BackendUnavailable" || !appErr.HasRoute("Run->connectDisplay.BackendUnavailable") {
		t.Errorf("got %s via %s, want BackendUnavailable via Run->connectDisplay", appErr.Tag, appErr.Route())
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
