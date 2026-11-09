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

// DefaultCompiler is the program that runs the SDK's clang: xcrun finds the SDK of the Command Line
// Tools or of Xcode.
//
// Flutter: no counterpart
const DefaultCompiler = "xcrun"

// InstallCommand installs the compiler and the SDK headers.
//
// Flutter: no counterpart
const InstallCommand = "xcode-select --install"

// Checker compiles a Spec. The zero value uses the real compiler; the fields let a test use a fake
// runner or another SDK.
//
// Flutter: no counterpart
type Checker struct {
	// Compiler is the program to run; empty means DefaultCompiler.
	Compiler string
	// SDK is the name of the SDK that xcrun picks ("macosx", "iphoneos", "iphonesimulator"); empty
	// means "macosx".
	SDK string
	// Args are more arguments for clang, such as the -target of an iOS build.
	Args []string
	// LookPath finds the compiler; nil means exec.LookPath.
	LookPath func(string) (string, error)
	// Run runs the compiler with its arguments and returns its output (standard output and error
	// together) and its exit code; the error is for a compiler that cannot be started, and is returned
	// by Check as it is (it must be tagged). nil runs the program with exec.
	Run func(ctx context.Context, argv []string) (output []byte, exitCode int, err error)
}

// Check writes the C file of spec to a temporary directory and compiles it with -fsyntax-only. It
// returns nil when every assertion holds, and fails with:
//   - DarwinLayoutCompilerMissing when xcrun is not installed (metadata "compiler" and "install");
//   - DarwinLayoutHeaderMissing when a header is not found (metadata "header" and "install");
//   - DarwinLayoutMismatch when a Go mirror differs from the header (metadata "failed", the messages of
//     the assertions that failed, which name the structure and the field, and "output");
//   - DarwinLayoutCompileFailed for any other compiler error, such as a type or member that the header
//     does not have (metadata "output", "exit code" and "command");
//   - DarwinLayoutCompilerRunFailed when the compiler cannot be started;
//   - DarwinLayoutFileFailed when the C file cannot be written.
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
		return errors.NewErr(tagDarwinLayoutCompilerMissing, err).SetMetadata("compiler", compiler, "install", InstallCommand)
	}
	dir, err := os.MkdirTemp("", "glutter-layout-")
	if err != nil {
		return errors.NewErr(tagDarwinLayoutFileFailed, err)
	}
	defer func() { _ = os.RemoveAll(dir) }() // a temporary directory; nothing depends on it
	file := filepath.Join(dir, "layout.m")
	if err := os.WriteFile(file, []byte(spec.Source()), 0o600); err != nil {
		return errors.NewErr(tagDarwinLayoutFileFailed, err).SetMetadata("file", file)
	}
	argv := c.command(path, file)
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
		return errors.New(tagDarwinLayoutMismatch, "a Go mirror does not match the SDK headers: "+strings.Join(failed, "; "),
			"failed", failed, "output", string(out))
	case missing != "":
		return errors.New(tagDarwinLayoutHeaderMissing, "the compiler does not find the header "+missing,
			"header", missing, "install", InstallCommand, "output", string(out))
	}
	return errors.New(tagDarwinLayoutCompileFailed, "the compiler failed on the layout file",
		"exit code", code, "output", string(out), "command", strings.Join(argv, " "))
}

// command is the command line that compiles the file: xcrun --sdk <sdk> clang -fsyntax-only, the
// language, the extra arguments and the file.
func (c Checker) command(xcrun, file string) []string {
	sdk := c.SDK
	if sdk == "" {
		sdk = "macosx"
	}
	argv := []string{xcrun, "--sdk", sdk, "clang", "-fsyntax-only", "-x", "objective-c", "-std=gnu11"}
	argv = append(argv, c.Args...)
	return append(argv, file)
}

// runCommand runs argv and returns its combined output and exit code. It fails with
// DarwinLayoutCompilerRunFailed when the program cannot be started.
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
	return nil, -1, errors.NewErr(tagDarwinLayoutCompilerRunFailed, err).SetMetadata("command", strings.Join(argv, " "))
}

var (
	// clang writes: static assertion failed due to requirement 'sizeof(CGRect) == 16': CGRect: size
	failedAssertion = regexp.MustCompile(`(?m)static assertion failed due to requirement '.*': (.+)$`)
	// clang writes: fatal error: 'AppKit/AppKit.h' file not found
	missingHeader = regexp.MustCompile(`fatal error: '([^']+)' file not found`)
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

// Check runs the default Checker over spec in a test. A missing compiler or header skips the test with
// the install command (so does every host that has no xcrun); any other failure fails it, with the
// compiler's own messages.
//
// Flutter: no counterpart
func Check(t testing.TB, spec Spec) {
	t.Helper()
	err := Checker{}.Check(context.Background(), spec)
	if err == nil {
		return
	}
	if appErr, ok := errors.FromError(err); ok && (appErr.Tag == tagDarwinLayoutCompilerMissing || appErr.Tag == tagDarwinLayoutHeaderMissing) {
		t.Skipf("%s: %v (install it with: %s)", appErr.Tag, appErr.Message, InstallCommand)
	}
	t.Fatalf("%v", err)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDarwinLayoutCompilerMissing   = "DarwinLayoutCompilerMissing"
	tagDarwinLayoutHeaderMissing     = "DarwinLayoutHeaderMissing"
	tagDarwinLayoutMismatch          = "DarwinLayoutMismatch"
	tagDarwinLayoutCompileFailed     = "DarwinLayoutCompileFailed"
	tagDarwinLayoutCompilerRunFailed = "DarwinLayoutCompilerRunFailed"
	tagDarwinLayoutFileFailed        = "DarwinLayoutFileFailed"
)
