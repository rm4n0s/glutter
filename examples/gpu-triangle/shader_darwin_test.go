//go:build darwin && !ios && glutter_display

package main

import (
	"io"
	"log/slog"
	"runtime"
	"testing"

	"github.com/rm4n0s/glutter/examples/gpu-triangle/shaders"
	"github.com/rm4n0s/glutter/internal/gpu/metal"
)

// TestTheShaderCompilesOnMetal compiles the MSL that glutter-gen gpushaders made for the example with Metal's own
// compiler (SPEC §29.5). It skips on a Mac without a Metal device.
func TestTheShaderCompilesOnMetal(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	d, err := metal.Open(metal.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Skipf("no Metal device: %v", err)
	}
	defer func() { _ = d.Close() }()
	var entries []string
	for _, e := range shaders.Cube.Entries {
		entries = append(entries, e.Name)
	}
	if err := d.CompileShader(shaders.Cube.Name, shaders.Cube.MSL, entries...); err != nil {
		t.Fatal(err)
	}
}
