package errcheck

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// fixtureModule writes a module with the path of Glutter, so that the checker treats its
// packages as the module's, and with the dependency on rm4n0s/errors of the real module.
func fixtureModule(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := filepath.Join("..", "..", "..", "..")
	goMod, err := os.ReadFile(filepath.Join(repo, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	goSum, err := os.ReadFile(filepath.Join(repo, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	var version, goDirective string
	for _, line := range strings.Split(string(goMod), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "go "):
			goDirective = line
		case strings.HasPrefix(line, "github.com/rm4n0s/errors "):
			version = strings.Fields(line)[1]
		}
	}
	if version == "" || goDirective == "" {
		t.Fatalf("go.mod of the repository has no go directive or no rm4n0s/errors: %q", goMod)
	}
	root := t.TempDir()
	write := func(name, content string) {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module "+modulePath+"\n\n"+goDirective+"\n\nrequire github.com/rm4n0s/errors "+version+"\n")
	write("go.sum", string(goSum))
	for name, content := range files {
		write(name, content)
	}
	return root
}

// check returns the violations as "rule Func" lines, without the tag-literal ones unless
// literals is true.
func check(t *testing.T, files map[string]string, literals bool) []string {
	t.Helper()
	root := fixtureModule(t, files)
	vs, err := CheckModule(context.Background(), root, []Pass{{GOOS: "linux", GOARCH: "amd64"}})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, v := range vs {
		if v.Rule == RuleTagLiteral && !literals {
			continue
		}
		out = append(out, v.Rule+" "+v.Func)
	}
	sort.Strings(out)
	return out
}

const violations = `package fix

import (
	stdErrors "errors"
	"fmt"
	"os"
	"strings"

	"github.com/rm4n0s/errors"
)

var sentinel = errors.New("Sentinel", "bad")

var fine = &errors.Error{Tag: "Fine"}

var _ = stdErrors.Join

func helper() error { return errors.New("Helper", "x") }

func panicHelper() { panic(errors.New("Panic", "x")) }

func chainNew() error { return errors.New("ChainNew", "x").SetMetadata("a", 1) }

func chainTwice(err error) error {
	return errors.NewErr("ChainTwice", err).SetMetadata("a", 1).SetMetadata("b", 2)
}

func oddPairs(n int) error {
	if n > 0 {
		return errors.New("OddPairs", "x", "a")
	}
	return nil
}

func numberKey(n int) error {
	if n > 0 {
		return errors.New("NumberKey", "x", 1, "a")
	}
	return nil
}

func format() error { return fmt.Errorf("x") }

func untagged(path string) error {
	_, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return nil
}

func inner(path string) error {
	_, err := os.ReadFile(path)
	if err != nil {
		return errors.NewErr("Inner", err)
	}
	return nil
}

func rewrap(path string) error {
	if err := inner(path); err != nil {
		return errors.NewErr("Rewrap", err)
	}
	return nil
}

func rewrapf(path string) error {
	if err := inner(path); err != nil {
		return errors.NewErrf("Rewrapf", "context: %w", err)
	}
	return nil
}

func branch(err error) bool {
	appErr, ok := errors.FromError(err)
	if !ok {
		return strings.Contains(err.Error(), "boom")
	}
	switch appErr.Tag {
	case "A":
		return true
	}
	return appErr.Tag == "B" || appErr.Message == "m"
}

func logged(err *errors.Error) any { return err.ToJson() }
`

const clean = `package fix

import (
	stdErrors "errors"
	"os"

	"github.com/rm4n0s/errors"
)

const (
	tagReadFailed = "ReadFailed"
	tagTooBig     = "TooBig"
)

var sentinel = &errors.Error{Tag: tagTooBig}

var _ = stdErrors.Join

func read(path string) error {
	if _, err := os.ReadFile(path); err != nil {
		return errors.NewErr(tagReadFailed, err).SetMetadata("path", path, "mode", "r")
	}
	return nil
}

func size(n int) error {
	if n > 10 {
		return errors.New(tagTooBig, "too big", "n", n)
	}
	return nil
}

func load(path string, n int) error {
	if err := read(path); err != nil {
		return err
	}
	return size(n)
}

func tooBig(err error) bool {
	appErr, ok := errors.FromError(err)
	return (ok && appErr.Tag == tagTooBig) || errors.Is(err, sentinel)
}

func addLater(err *errors.Error, id int) *errors.Error { return err.SetMetadata("id", id) }

func validate(n int) error {
	if n < 0 {
		return errors.New(tagTooBig, "negative", "n", n)
	}
	return nil
}
`

func TestRules(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{
			name:  "every rule fires once, in the function that holds the code",
			files: map[string]string{"fix/fix.go": violations},
			want: []string{
				"error-helper helper",
				"error-helper panicHelper",
				"error-rewrap rewrap",
				"error-rewrap rewrapf",
				"error-tojson logged",
				"error-untagged format",
				"error-untagged untagged",
				"fmt-errorf format",
				"metadata-chain chainNew",
				"metadata-chain chainTwice",
				"metadata-pairs numberKey",
				"metadata-pairs oddPairs",
				"sentinel-ctor -",
				"tag-branch branch",
				"tag-branch branch",
				"tag-branch branch",
				"tag-branch branch",
			},
		},
		{
			name:  "code that follows the rules has no violation",
			files: map[string]string{"fix/fix.go": clean},
		},
		{
			name: "the standard errors package needs an alias",
			files: map[string]string{"fix/fix.go": `package fix

import "errors"

var _ = errors.Join
`},
			want: []string{"std-errors -"},
		},
		{
			name: "a test file may compare a tag or a message with a string, and call ToJson",
			files: map[string]string{
				"fix/fix.go": clean,
				"fix/fix_test.go": `package fix

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestReplay(t *testing.T) {
	err := errors.New(tagTooBig, "x")
	if err.ToJson().Tag != tagTooBig || err.Tag != "TooBig" || err.Message != "x" {
		t.Fatal("tag")
	}
}
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := check(t, tt.files, false)
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("violations:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

func TestTagLiteral(t *testing.T) {
	got := check(t, map[string]string{"fix/fix.go": `package fix

import "github.com/rm4n0s/errors"

const tagOK = "OK"

func a(n int) error {
	if n > 0 {
		return errors.New("Literal", "x")
	}
	return nil
}

func b(n int) error {
	if n > 0 {
		return errors.New(tagOK, "x")
	}
	return nil
}

func c(n int) error {
	if n > 0 {
		return errors.NewErrf(prefix()+"Dynamic", "x")
	}
	return nil
}

func prefix() string { return "p" }
`}, true)
	want := []string{"tag-literal a"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("violations = %v, want %v", got, want)
	}
}

func TestRunReportsAndAccepts(t *testing.T) {
	root := fixtureModule(t, map[string]string{"fix/fix.go": violations})
	allowDir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(allowDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	tests := []struct {
		name      string
		args      []string
		wantTag   string
		wantRoute string
		wantOut   string
	}{
		{"violations", []string{"-root", root}, "ErrCheckViolations", "run.ErrCheckViolations", "[error-helper] helper"},
		{"stale line of the allow file", []string{"-root", root, "-allow", write("stale", "error-helper nowhere.go:nothing\n")}, "ErrCheckViolations", "run.ErrCheckViolations", "the allow file lists a violation that is gone"},
		{"unknown flag", []string{"-nope"}, "ErrCheckBadFlags", "run.ErrCheckBadFlags", ""},
		{"missing allow file", []string{"-root", root, "-allow", filepath.Join(allowDir, "missing")}, "ErrCheckAllowUnreadable", "ReadAllow.ErrCheckAllowUnreadable", ""},
		{"bad line of the allow file", []string{"-root", root, "-allow", write("bad", "error-helper\n")}, "ErrCheckAllowSyntax", "ReadAllow.ErrCheckAllowSyntax", ""},
		{"root that does not exist", []string{"-root", filepath.Join(allowDir, "nowhere")}, "ErrCheckListFailed", "checkPass.ErrCheckListFailed", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := run(tt.args, &out)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != tt.wantTag {
				t.Errorf("Tag = %q, want %q", appErr.Tag, tt.wantTag)
			}
			if !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("output %q does not contain %q", out.String(), tt.wantOut)
			}
		})
	}

	t.Run("the allow file accepts a violation by rule, file and function", func(t *testing.T) {
		vs, err := CheckModule(context.Background(), root, []Pass{{GOOS: "linux", GOARCH: "amd64"}})
		if err != nil {
			t.Fatal(err)
		}
		var lines []string
		for _, v := range vs {
			lines = append(lines, v.Key()+" # "+v.Message)
		}
		allow := write("all", strings.Join(lines, "\n")+"\n")
		var out bytes.Buffer
		if err := run([]string{"-root", root, "-allow", allow}, &out); err != nil {
			t.Fatalf("run failed: %v\n%s", err, out.String())
		}
		if !strings.Contains(out.String(), "0 violations") {
			t.Errorf("output = %q", out.String())
		}
	})
}

func TestViolationFormat(t *testing.T) {
	v := Violation{Rule: RuleErrorHelper, File: "a/b.go", Func: "T.f", Line: 7, Message: "m"}
	if got, want := v.Key(), "error-helper a/b.go:T.f"; got != want {
		t.Errorf("Key = %q, want %q", got, want)
	}
	if got, want := v.String(), "a/b.go:7: [error-helper] T.f: m"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}

// TestTheModuleKeepsTheErrorRules runs the check on the module itself: code that breaks a
// rule of rm4n0s/errors fails here, unless cmd/glutter-gen/errcheck.allow accepts it.
func TestTheModuleKeepsTheErrorRules(t *testing.T) {
	if testing.Short() {
		t.Skip("type-checks the whole module")
	}
	root := filepath.Join("..", "..", "..", "..")
	allow, err := ReadAllow(filepath.Join(root, "cmd", "glutter-gen", "errcheck.allow"))
	if err != nil {
		t.Fatal(err)
	}
	vs, err := CheckModule(context.Background(), root, DefaultPasses)
	if err != nil {
		t.Fatal(err)
	}
	used := map[string]bool{}
	for _, v := range vs {
		if allow[v.Key()] {
			used[v.Key()] = true
			continue
		}
		t.Errorf("%s", v)
	}
	for key := range allow {
		if !used[key] {
			t.Errorf("the allow file lists %q, which is not a violation any more", key)
		}
	}
}
