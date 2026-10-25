package layouttest

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// DefaultCompiler is the mingw-w64 compiler for 64-bit Windows (Fedora: mingw64-gcc).
//
// Flutter: no counterpart
const DefaultCompiler = "x86_64-w64-mingw32-gcc"

// InstallCommand installs the compiler and the headers on Fedora.
//
// Flutter: no counterpart
const InstallCommand = "sudo dnf install mingw64-gcc mingw64-headers"

// Checker compiles a Spec. The zero value uses the real compiler; the fields let a test use a
// fake runner or another compiler.
//
// Flutter: no counterpart
type Checker struct {
	// Compiler is the program to run; empty means DefaultCompiler.
	Compiler string
	// LookPath finds the compiler; nil means exec.LookPath.
	LookPath func(string) (string, error)
	// Run runs the compiler with its arguments and returns its output (standard output and error
	// together) and its exit code; the error is for a compiler that cannot be started, and is
	// returned by Check as it is (it must be tagged). nil runs the program with exec.
	Run func(ctx context.Context, argv []string) (output []byte, exitCode int, err error)
}

// Check writes the C file of spec to a temporary directory and compiles it with -fsyntax-only. It
// returns nil when every assertion holds, and fails with:
//   - LayoutCompilerMissing when the compiler is not installed (metadata "compiler" and "install");
//   - LayoutHeaderMissing when a header is not found (metadata "header" and "install");
//   - LayoutMismatch when a Go mirror differs from the header (metadata "failed", the messages of
//     the assertions that failed, which name the structure and the field, and "output");
//   - LayoutCompileFailed for any other compiler error, such as a type or member that the header
//     does not have (metadata "output", "exit code" and "command");
//   - LayoutCompilerRunFailed when the compiler cannot be started;
//   - LayoutFileFailed when the C file cannot be written.
//
// Flutter: no counterpart
func (c Checker) Check(ctx context.Context, spec Spec) error {
	compiler := c.Compiler
	if compiler == "" {
		compiler = DefaultCompiler
	}
	lookPath := c.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	path, err := lookPath(compiler)
	if err != nil {
		return errors.NewErr(TagLayoutCompilerMissing, err).SetMetadata("compiler", compiler, "install", InstallCommand)
	}
	dir, err := os.MkdirTemp("", "glutter-layout-")
	if err != nil {
		return errors.NewErr(TagLayoutFileFailed, err)
	}
	defer func() { _ = os.RemoveAll(dir) }() // a temporary directory; nothing depends on it
	file := filepath.Join(dir, "layout.c")
	if err := os.WriteFile(file, []byte(spec.Source()), 0o600); err != nil {
		return errors.NewErr(TagLayoutFileFailed, err).SetMetadata("file", file)
	}
	argv := Command(path, spec, file)
	run := c.Run
	if run == nil {
		run = runCommand
	}
	out, code, err := run(ctx, argv)
	if err != nil {
		return err
	}
	if code == 0 {
		return nil
	}
	failed, missing := parseOutput(out)
	switch {
	case len(failed) > 0:
		return errors.New(TagLayoutMismatch, "a Go mirror does not match the C headers: "+strings.Join(failed, "; "),
			"failed", failed, "output", string(out))
	case missing != "":
		return errors.New(TagLayoutHeaderMissing, "the compiler does not find the header "+missing,
			"header", missing, "install", InstallCommand, "output", string(out))
	}
	return errors.New(TagLayoutCompileFailed, "the compiler failed on the layout file",
		"exit code", code, "output", string(out), "command", strings.Join(argv, " "))
}

// Command is the command line that compiles the C file of spec: the compiler, -fsyntax-only, the
// defines (COBJMACROS first), the include directories and the file.
//
// Flutter: no counterpart
func Command(compiler string, spec Spec, file string) []string {
	argv := []string{compiler, "-fsyntax-only", "-x", "c", "-std=gnu11", "-DCOBJMACROS", "-D_WIN32_WINNT=0x0A00"}
	for _, d := range spec.Defines {
		argv = append(argv, "-D"+d)
	}
	for _, dir := range spec.IncludeDirs {
		argv = append(argv, "-I"+dir)
	}
	return append(argv, file)
}

// runCommand runs argv and returns its combined output and exit code. It fails with
// LayoutCompilerRunFailed when the program cannot be started.
func runCommand(ctx context.Context, argv []string) ([]byte, int, error) {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return out.Bytes(), 0, nil
	case errors.As(err, &exit):
		return out.Bytes(), exit.ExitCode(), nil
	}
	return nil, -1, errors.NewErr(TagLayoutCompilerRunFailed, err).SetMetadata("command", strings.Join(argv, " "))
}

var (
	failedAssertion = regexp.MustCompile(`static assertion failed: "([^"]*)"`)
	missingHeader   = regexp.MustCompile(`fatal error: ([^:\s]+): No such file or directory`)
)

// parseOutput reads the compiler's messages: the messages of the assertions that failed, and the
// header it did not find.
func parseOutput(out []byte) (failed []string, header string) {
	for _, m := range failedAssertion.FindAllSubmatch(out, -1) {
		failed = append(failed, string(m[1]))
	}
	if m := missingHeader.FindSubmatch(out); m != nil {
		header = string(m[1])
	}
	return failed, header
}

// Check runs the default Checker over spec in a test. A missing compiler or header skips the test
// with the install command; any other failure fails it, with the compiler's own messages.
//
// Flutter: no counterpart
func Check(t testing.TB, spec Spec) {
	t.Helper()
	err := Checker{}.Check(context.Background(), spec)
	if err == nil {
		return
	}
	if appErr, ok := errors.FromError(err); ok && (appErr.Tag == TagLayoutCompilerMissing || appErr.Tag == TagLayoutHeaderMissing) {
		t.Skipf("%s: %v (install it with: %s)", appErr.Tag, appErr.Message, InstallCommand)
	}
	t.Fatalf("%v", err)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagLayoutCompilerMissing is the tag of the error for a mingw-w64 compiler that is not installed.
	TagLayoutCompilerMissing = "LayoutCompilerMissing"
	// TagLayoutHeaderMissing is the tag of the error for a header the compiler does not find.
	TagLayoutHeaderMissing = "LayoutHeaderMissing"
	// TagLayoutMismatch is the tag of the error for a Go mirror that differs from the C header.
	TagLayoutMismatch = "LayoutMismatch"
	// TagLayoutCompileFailed is the tag of the error for any other compiler failure.
	TagLayoutCompileFailed = "LayoutCompileFailed"
	// TagLayoutCompilerRunFailed is the tag of the error for a compiler that cannot be started.
	TagLayoutCompilerRunFailed = "LayoutCompilerRunFailed"
	// TagLayoutFileFailed is the tag of the error for a C file that cannot be written.
	TagLayoutFileFailed = "LayoutFileFailed"
)
