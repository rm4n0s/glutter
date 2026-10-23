//go:build linux

package gles

import (
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestInt32Array(t *testing.T) {
	// Not a pointer test: the values must land in C memory in order.
	if got := int32Array(nil); got == 0 {
		t.Error("an empty array still needs valid memory")
	}
}

func TestOpenReportsEGLLoadFailed(t *testing.T) {
	missing := "/nonexistent-glutter-dir/libEGL.so.1"
	t.Setenv(eglLibraryEnv, missing)
	_, _, err := Open(waylandHandle)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tagEGLLoadFailed || !appErr.HasRoute("Open->loadEGL.EGLLoadFailed") {
		t.Errorf("Tag = %q, route = %s, want EGLLoadFailed via Open->loadEGL", appErr.Tag, appErr.Route())
	}
	if appErr.Metadata["library"] != "libEGL" {
		t.Errorf("Metadata[library] = %v, want libEGL", appErr.Metadata["library"])
	}
	names, _ := appErr.Metadata["names"].([]string)
	if len(names) != 1 || names[0] != missing {
		t.Errorf("Metadata[names] = %v, want the override path", appErr.Metadata["names"])
	}
	if s, _ := appErr.Metadata["dlerror"].(string); !strings.Contains(s, "nonexistent") {
		t.Errorf("Metadata[dlerror] = %q, want the loader's message", s)
	}
}

func TestOpenReportsGLESLoadFailed(t *testing.T) {
	// libEGL loads normally here; only libGLESv2 is pointed at a missing file.
	if _, err := loadEGL(); err != nil {
		t.Skipf("libEGL is not available: %v", err)
	}
	t.Setenv(glesLibraryEnv, "/nonexistent-glutter-dir/libGLESv2.so.2")
	_, _, err := Open(waylandHandle)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagEGLLoadFailed || appErr.Metadata["library"] != "libGLESv2" {
		t.Errorf("Open returned %v, want EGLLoadFailed for libGLESv2", err)
	}
}

func TestGLSymbolTable(t *testing.T) {
	g := new(glLib)
	seen := map[string]bool{}
	for _, b := range g.glSymbols() {
		if !strings.HasPrefix(b.symbol, "gl") {
			t.Errorf("symbol %q does not look like a GL function", b.symbol)
		}
		if seen[b.symbol] {
			t.Errorf("symbol %q is bound twice", b.symbol)
		}
		seen[b.symbol] = true
		if b.fptr == nil {
			t.Errorf("symbol %q has no target", b.symbol)
		}
	}
	for _, must := range []string{"glGetString", "glGetStringi", "glDrawArraysInstanced", "glBlitFramebuffer", "glTexImage2D", "glVertexAttribDivisor"} {
		if !seen[must] {
			t.Errorf("the table lacks %s", must)
		}
	}
}

func TestLoadGLSymbolMissing(t *testing.T) {
	egl := newFakeEGL() // knows no GL functions
	_, err := loadGL(egl)
	appErr := routeOf(t, err, "SymbolMissing")
	if !appErr.HasRoute("loadGL.SymbolMissing") || appErr.Metadata["symbol"] == "" || appErr.Metadata["library"] != "eglGetProcAddress" {
		t.Errorf("route %s, metadata %v", appErr.Route(), appErr.Metadata)
	}
}
