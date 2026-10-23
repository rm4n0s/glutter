//go:build !js

package engine

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestBuildsHaveTheirOwnBackends lists what an app imports on each target (SPEC §24, "Build
// tags"): Linux has the Wayland and X11 backends, Android has none of the desktop backends (only
// the purego loader dl, which the gles and vulkan renderers share), and the web has neither
// (SPEC §6.3 rules 2 and 7). It needs no NDK: go list does not compile.
func TestBuildsHaveTheirOwnBackends(t *testing.T) {
	if testing.Short() {
		t.Skip("runs go list")
	}
	const prefix = "github.com/rm4n0s/glutter/internal/platform/"
	tests := []struct {
		name         string
		goos, goarch string
		want         []string // import paths below internal/platform/ that must be there
		none         []string // prefixes below internal/platform/ that must not be there
		onlyDL       bool     // internal/platform/linux/ may only hold dl
	}{
		{name: "linux", goos: "linux", goarch: "amd64", want: []string{"linux", "linux/wayland", "linux/x11", "linux/fontconfig"}, none: []string{"web", "android"}},
		{name: "js", goos: "js", goarch: "wasm", want: []string{"web"}, none: []string{"linux/", "android"}},
		{name: "android/arm64", goos: "android", goarch: "arm64", none: []string{"web"}, onlyDL: true},
		{name: "android/amd64", goos: "android", goarch: "amd64", none: []string{"web"}, onlyDL: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "list", "-e", "-deps", "-f", "{{.ImportPath}}", "../../examples/counter")
			cmd.Env = append(os.Environ(), "GOOS="+tt.goos, "GOARCH="+tt.goarch, "CGO_ENABLED=0")
			if tt.goos == "android" {
				// Android needs cgo (D21); go list only reads the files, so it needs no NDK.
				cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
			}
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("go list: %v", err)
			}
			have := map[string]bool{}
			if !strings.Contains(string(out), "github.com/rm4n0s/glutter/internal/engine\n") {
				t.Fatalf("%s: go list did not list the engine:\n%s", tt.name, out)
			}
			for _, p := range strings.Fields(string(out)) {
				if rest, ok := strings.CutPrefix(p, prefix); ok {
					have[rest] = true
				}
			}
			for _, w := range tt.want {
				if !have[w] {
					t.Errorf("%s: internal/platform/%s is not imported", tt.name, w)
				}
			}
			for p := range have {
				for _, n := range tt.none {
					if p == n || strings.HasPrefix(p, n) {
						t.Errorf("%s: internal/platform/%s is imported", tt.name, p)
					}
				}
				if tt.onlyDL && strings.HasPrefix(p, "linux") && p != "linux/dl" {
					t.Errorf("%s: internal/platform/%s is imported; only linux/dl is shared", tt.name, p)
				}
			}
		})
	}
}
