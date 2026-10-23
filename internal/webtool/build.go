package webtool

import (
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/rm4n0s/errors"
)

// BuildOptions are the parameters of Build.
//
// Flutter: no counterpart
type BuildOptions struct {
	// Package is the main package to build, as `go build` takes it: "./examples/counter".
	Package string
	// OutDir is the directory to write the build to; it is created.
	OutDir string
	// Release builds with -tags glutter_release and -ldflags=-s -w (SPEC §23.1).
	Release bool
	// Dir is the directory to run the go command in, which Package is relative to. The
	// default is the current directory.
	Dir string
	// HostElement is the CSS selector of the element the view fills; empty fills the page.
	HostElement string
	// Body is HTML for the body of index.html, before the loader: the host element when
	// HostElement is set. Empty by default.
	Body string
	// Title is the title of index.html; the base name of Package when empty.
	Title string
	// Env is added to the environment of the go command.
	Env []string
}

// BuildResult says what Build wrote, all in OutDir.
//
// Flutter: no counterpart
type BuildResult struct {
	// Dir is the directory of the build.
	Dir string
	// WasmSize and GzipSize are the sizes of app.wasm and app.wasm.gz in bytes.
	WasmSize, GzipSize int64
}

// Build builds the package for js/wasm with the standard Go toolchain (TinyGo is not
// supported, D18) and writes the directory that any static web server can serve
// (SPEC §23.1): index.html, wasm_exec.js (copied from $(go env GOROOT)/lib/wasm, which must
// match the Go that builds), app.wasm and app.wasm.gz. It fails with WebBuildFailed when the
// go command fails (its output is in the error's metadata) and with WebWriteFailed when a
// file cannot be written.
//
// Flutter: no counterpart (flutter build web)
func Build(ctx context.Context, o BuildOptions) (*BuildResult, error) {
	if o.Package == "" {
		return nil, errors.New(tagWebBuildFailed, "there is no package to build")
	}
	if err := os.MkdirAll(o.OutDir, 0o755); err != nil {
		return nil, errors.NewErr(tagWebWriteFailed, err).SetMetadata("path", o.OutDir)
	}
	env := append(os.Environ(), "GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0")
	env = append(env, o.Env...)
	wasm := filepath.Join(o.OutDir, "app.wasm")
	args := []string{"build", "-o", wasm}
	if o.Release {
		args = append(args, "-tags", "glutter_release", "-ldflags=-s -w")
	}
	args = append(args, o.Package)
	if _, err := runGo(ctx, o.Dir, env, args...); err != nil {
		return nil, err
	}
	goroot, err := runGo(ctx, o.Dir, env, "env", "GOROOT")
	if err != nil {
		return nil, err
	}
	execJS, err := os.ReadFile(filepath.Join(strings.TrimSpace(goroot), "lib", "wasm", "wasm_exec.js"))
	if err != nil {
		return nil, errors.NewErr(tagWebWriteFailed, err).SetMetadata("what", "wasm_exec.js of GOROOT")
	}
	if err := os.WriteFile(filepath.Join(o.OutDir, "wasm_exec.js"), execJS, 0o644); err != nil {
		return nil, errors.NewErr(tagWebWriteFailed, err).SetMetadata("path", filepath.Join(o.OutDir, "wasm_exec.js"))
	}
	title := o.Title
	if title == "" {
		title = path.Base(filepath.ToSlash(o.Package))
	}
	index, err := IndexHTML(Page{Title: title, HostElement: o.HostElement, Body: o.Body})
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(o.OutDir, "index.html"), index, 0o644); err != nil {
		return nil, errors.NewErr(tagWebWriteFailed, err).SetMetadata("path", filepath.Join(o.OutDir, "index.html"))
	}
	wasmBytes, err := os.ReadFile(wasm)
	if err != nil {
		return nil, errors.NewErr(tagWebWriteFailed, err).SetMetadata("path", wasm)
	}
	var zipped bytes.Buffer
	zw, err := gzip.NewWriterLevel(&zipped, gzip.BestCompression)
	if err != nil {
		return nil, errors.NewErr(tagWebWriteFailed, err).SetMetadata("path", wasm+".gz")
	}
	if _, err := zw.Write(wasmBytes); err != nil {
		return nil, errors.NewErr(tagWebWriteFailed, err).SetMetadata("path", wasm+".gz")
	}
	if err := zw.Close(); err != nil {
		return nil, errors.NewErr(tagWebWriteFailed, err).SetMetadata("path", wasm+".gz")
	}
	if err := os.WriteFile(wasm+".gz", zipped.Bytes(), 0o644); err != nil {
		return nil, errors.NewErr(tagWebWriteFailed, err).SetMetadata("path", wasm+".gz")
	}
	return &BuildResult{Dir: o.OutDir, WasmSize: int64(len(wasmBytes)), GzipSize: int64(zipped.Len())}, nil
}

// runGo runs the go command and returns its combined output. It fails with
// WebBuildFailed, and the output of the command is in the metadata "output".
func runGo(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir, cmd.Env = dir, env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.NewErr(tagWebBuildFailed, err).SetMetadata("command", "go "+strings.Join(args, " "), "output", string(out))
	}
	return string(out), nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWebBuildFailed = "WebBuildFailed"
	tagWebWriteFailed = "WebWriteFailed"
)
