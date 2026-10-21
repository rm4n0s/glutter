package webtool

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a wasm program")
	}
	out := filepath.Join(t.TempDir(), "site")
	res, err := Build(context.Background(), BuildOptions{Package: "./testdata/hello", OutDir: out, HostElement: "#stage", Body: `<div id="stage"></div>`})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, name := range []string{"index.html", "wasm_exec.js", "app.wasm", "app.wasm.gz"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("the build lacks %s: %v", name, err)
		}
	}
	wasm, err := os.ReadFile(filepath.Join(out, "app.wasm"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(wasm, []byte("\x00asm")) {
		t.Error("app.wasm is not a WebAssembly module")
	}
	gz, err := os.ReadFile(filepath.Join(out, "app.wasm.gz"))
	if err != nil {
		t.Fatal(err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		t.Fatal(err)
	}
	unzipped, err := io.ReadAll(zr)
	if err != nil || !bytes.Equal(unzipped, wasm) {
		t.Errorf("app.wasm.gz does not hold app.wasm (err %v)", err)
	}
	if res.WasmSize != int64(len(wasm)) || res.GzipSize != int64(len(gz)) || res.Dir != out {
		t.Errorf("result %+v, want sizes %d and %d in %s", res, len(wasm), len(gz), out)
	}
	// wasm_exec.js is the one of the Go that built the program.
	goroot, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(goroot)), "lib", "wasm", "wasm_exec.js"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(out, "wasm_exec.js"))
	if err != nil || !bytes.Equal(got, want) {
		t.Errorf("wasm_exec.js is not the one of GOROOT (err %v)", err)
	}
	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<title>hello</title>", `{"hostElement":"#stage"}`, `<div id="stage">`} {
		if !strings.Contains(string(index), want) {
			t.Errorf("index.html lacks %q", want)
		}
	}
}

func TestBuildRelease(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a wasm program")
	}
	dir := t.TempDir()
	plain, err := Build(context.Background(), BuildOptions{Package: "./testdata/hello", OutDir: filepath.Join(dir, "plain")})
	if err != nil {
		t.Fatal(err)
	}
	release, err := Build(context.Background(), BuildOptions{Package: "./testdata/hello", OutDir: filepath.Join(dir, "release"), Release: true})
	if err != nil {
		t.Fatal(err)
	}
	if release.WasmSize >= plain.WasmSize {
		t.Errorf("the release build (%d bytes) is not smaller than the plain one (%d): -s -w did not apply", release.WasmSize, plain.WasmSize)
	}
}

func TestBuildFailures(t *testing.T) {
	tests := []struct {
		name string
		opts BuildOptions
		tag  string
		fn   string
		meta string
	}{
		{"no package", BuildOptions{OutDir: t.TempDir()}, "WebBuildFailed", "Build", ""},
		{"does not compile", BuildOptions{Package: "./testdata/broken", OutDir: t.TempDir()}, "WebBuildFailed", "Build", "undefined"},
		{"no such package", BuildOptions{Package: "./testdata/nothing", OutDir: t.TempDir()}, "WebBuildFailed", "Build", "nothing"},
		{"output directory is a file", BuildOptions{Package: "./testdata/hello", OutDir: filepath.Join(fileIn(t), "sub")}, "WebWriteFailed", "Build", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Build(context.Background(), tc.opts)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tc.tag || !appErr.HasRoute(tc.fn+"."+tc.tag) {
				t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tc.tag, tc.fn)
			}
			if out, _ := appErr.Metadata["output"].(string); tc.meta != "" && !strings.Contains(out, tc.meta) {
				t.Errorf("the output of the go command %q does not mention %q", out, tc.meta)
			}
		})
	}
}

// fileIn makes a plain file and returns its path.
func fileIn(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}
