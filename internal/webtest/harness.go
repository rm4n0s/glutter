package webtest

import (
	"os"
	"path/filepath"
	"testing"
)

// ModuleRoot finds the directory of go.mod above the working directory of the test, which the
// packages of the apps to build are relative to.
//
// Flutter: no counterpart
func ModuleRoot(tb testing.TB) string {
	tb.Helper()
	dir, err := os.Getwd()
	if err != nil {
		tb.Fatalf("no working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatal("go.mod was not found above the test's directory")
		}
		dir = parent
	}
}

// EachBrowser runs fn as a subtest for Chromium and for Firefox. A browser that is not
// installed fails its subtest with the error of Locate: the browser tests exist to run in both
// (SPEC §17), so a missing one is not a pass.
//
// Flutter: no counterpart
func EachBrowser(t *testing.T, fn func(t *testing.T, b Browser)) {
	t.Helper()
	for _, b := range []Browser{Chromium, Firefox} {
		t.Run(string(b), func(t *testing.T) {
			if _, err := Locate(b, nil); err != nil {
				t.Fatal(err)
			}
			fn(t, b)
		})
	}
}
