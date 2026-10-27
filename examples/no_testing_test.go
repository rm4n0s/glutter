//go:build !js

package examples

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// testingPackage reports whether an import path is a package that only a test binary may link: `testing` and
// everything under it (`testing/fstest`, `testing/iotest`, ...), `net/http/httptest`, and the widget-testing package
// `gluttertest`.
func testingPackage(path string) bool {
	return path == "testing" || strings.HasPrefix(path, "testing/") ||
		path == "net/http/httptest" ||
		path == "gluttertest" || strings.HasSuffix(path, "/gluttertest")
}

// testingImports reads the output of `go list -deps -f '{{.ImportPath}} {{.Imports}}'` and returns, for every package
// that imports a test-only package, a line that names both.
func testingImports(list string) []string {
	var out []string
	for line := range strings.SplitSeq(list, "\n") {
		importPath, imports, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		imports = strings.Trim(imports, "[]")
		for imp := range strings.FieldsSeq(imports) {
			if testingPackage(imp) {
				out = append(out, importPath+" imports "+imp)
			}
		}
	}
	return out
}

// TestNoExampleLinksTesting checks what step 29.16 asks: no example links the `testing` package, or any other package
// that only a test binary needs, on any target. `testing` costs size in every program, and Go registers its flags
// only in a test binary, so the cost is silent: this test is what notices it. Each target is one `go list -deps`
// of all the examples that run on it; a failure names the package that imports `testing`, so that it is found at
// once. The list needs neither cgo nor the NDK.
func TestNoExampleLinksTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("lists the dependencies of every example for five targets; skipped by -short")
	}
	var web, all []string
	for _, name := range allExamples(t) {
		all = append(all, "./examples/"+name)
	}
	for _, name := range webExamples(t) {
		web = append(web, "./examples/"+name)
	}
	if len(all) < 11 || len(web) < 9 {
		t.Fatalf("found %d examples and %d for the web, want at least 11 and 9", len(all), len(web))
	}
	tests := []struct {
		name     string
		goos     string
		goarch   string
		packages []string
	}{
		{"linux/amd64", "linux", "amd64", all},
		{"android/arm64", "android", "arm64", web},
		{"js/wasm", "js", "wasm", web},
		{"windows/amd64", "windows", "amd64", all},
		{"windows/arm64", "windows", "arm64", all},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			args := append([]string{"list", "-deps", "-f", "{{.ImportPath}} {{.Imports}}"}, tt.packages...)
			cmd := exec.CommandContext(ctx, "go", args...)
			cmd.Dir = ".."
			cmd.Env = append(os.Environ(), "GOOS="+tt.goos, "GOARCH="+tt.goarch, "CGO_ENABLED=0")
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("go list -deps for %s: %v\n%s", tt.name, err, stderr.String())
			}
			for _, line := range testingImports(stdout.String()) {
				t.Errorf("an example links a test-only package on %s: %s", tt.name, line)
			}
		})
	}
}

// TestTestingImports checks the filter of TestNoExampleLinksTesting on lists made by hand.
func TestTestingImports(t *testing.T) {
	tests := []struct {
		name string
		list string
		want []string
	}{
		{"nothing", "fmt [errors io]\nmain [fmt os]\n", nil},
		{"testing", "dl [reflect testing]\n", []string{"dl imports testing"}},
		{"testing subpackage", "a [testing/fstest]\n", []string{"a imports testing/fstest"}},
		{"httptest", "a [net/http net/http/httptest]\n", []string{"a imports net/http/httptest"}},
		{"gluttertest", "a [github.com/rm4n0s/glutter/gluttertest]\n", []string{"a imports github.com/rm4n0s/glutter/gluttertest"}},
		{"similar names are fine", "a [mytesting contesting/x net/http]\n", nil},
		{"two packages", "a [testing]\nb [testing]\n", []string{"a imports testing", "b imports testing"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testingImports(tt.list)
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
