//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestReadErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{not a list"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		path string
	}{
		{"a file that is not there", filepath.Join(dir, "missing.json")},
		{"a file that is not JSON", bad},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := read(tt.path)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tagHLSLCheckInput || !appErr.HasRoute("read.HLSLCheckInput") {
				t.Errorf("got %s via %s", appErr.Tag, appErr.Route())
			}
		})
	}
}

func TestReadAList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "in.json")
	data := `[{"name":"a","hlsl":"x","entries":[{"name":"vs_main","target":"vs_5_1"}]}]`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	shaders, err := read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(shaders) != 1 || shaders[0].Name != "a" || len(shaders[0].Entries) != 1 || shaders[0].Entries[0].Target != "vs_5_1" {
		t.Errorf("read = %+v", shaders)
	}
}

func TestOneLine(t *testing.T) {
	err := errors.New(tagTestCompileFailed, "failed", "messages", "shader.hlsl(3,5): error X3004\nshader.hlsl(4,1): error X3000")
	if got, want := oneLine(err), "shader.hlsl(3,5): error X3004 | shader.hlsl(4,1): error X3000"; got != want {
		t.Errorf("oneLine = %q, want %q", got, want)
	}
}

// tagTestCompileFailed is the tag of the compile error that the test makes up.
const tagTestCompileFailed = "D3D12ShaderCompileFailed"
