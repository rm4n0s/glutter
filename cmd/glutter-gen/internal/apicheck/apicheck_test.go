package apicheck

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func writePackage(t *testing.T, src string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "pkg.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func keysOf(vs []Violation) []string {
	var out []string
	for _, v := range vs {
		out = append(out, v.Key())
	}
	sort.Strings(out)
	return out
}

func TestCheckDir(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "a conforming package",
			src: `package pkg

// Padding is a widget.
//
// Flutter: widgets/basic.dart Padding
type Padding struct{}

// PaddingOpts are the options.
//
// Flutter: widgets/basic.dart Padding
type PaddingOpts struct{}

// NewPadding creates one.
//
// Flutter: widgets/basic.dart Padding
func NewPadding(padding int, child Widget, opts ...PaddingOpts) *Padding { return nil }

// Widget is a widget.
//
// Glutter-specific: the interface of all widgets
type Widget interface{}

// NewCurve takes an optional positional parameter.
//
// Flutter: animation/curves.dart Curve
func NewCurve(a float64, tension ...float64) *Padding { return nil }

// Colors is a read-only value.
//
// Flutter: material/colors.dart Colors
var Colors = Padding{}
`,
			want: nil,
		},
		{
			name: "the options value is not variadic",
			src: `package pkg

// X is a thing.
//
// Flutter: no counterpart
type X struct{}

// XOpts are the options.
//
// Flutter: no counterpart
type XOpts struct{}

// NewX creates one.
//
// Flutter: no counterpart
func NewX(opts XOpts) *X { return nil }
`,
			want: []string{RuleCtorShape + " pkg.NewX"},
		},
		{
			name: "child and children before another required parameter",
			src: `package pkg

// X is a thing.
//
// Flutter: no counterpart
type X struct{}

// NewX creates one.
//
// Flutter: no counterpart
func NewX(child Widget, size int) *X { return nil }

// NewY creates one.
//
// Flutter: no counterpart
func NewY(children []Widget, size int, opts ...YOpts) *X { return nil }

// NewZ has the child last.
//
// Flutter: no counterpart
func NewZ(size int, child Widget) *X { return nil }
`,
			want: []string{RuleCtorOrder + " pkg.NewX", RuleCtorOrder + " pkg.NewY"},
		},
		{
			name: "too many required parameters",
			src: `package pkg

// NewX creates one.
//
// Flutter: no counterpart
func NewX(a, b, c, d, e, f, g int) *int { return nil }

// NewY creates one.
//
// Flutter: no counterpart
func NewY(a, b, c, d, e, f int) *int { return nil }
`,
			want: []string{RuleCtorSize + " pkg.NewX"},
		},
		{
			name: "doc comments",
			src: `package pkg

// A has no reference.
type A struct{}

type B struct{}

// C has a reference in the middle.
//
// Flutter: no counterpart
//
// And then goes on.
type C struct{}

// D has the reference in the last line of its paragraph.
// Flutter: material/d.dart D
type D struct{}

// E is in a group.
type (
	// F has one.
	//
	// Flutter: no counterpart
	F struct{}
	// G has none.
	G struct{}
)

// Func has none.
func Func() {}

type unexported struct{}
`,
			want: []string{RuleDocRef + " pkg.A", RuleDocRef + " pkg.B", RuleDocRef + " pkg.C", RuleDocRef + " pkg.G", RuleDocRef + " pkg.Func"},
		},
		{
			name: "enum zero values",
			src: `package pkg

// Align is an enum with a good zero.
//
// Flutter: no counterpart
type Align int

const (
	// AlignUnspecified is the default.
	//
	// Flutter: no counterpart
	AlignUnspecified Align = iota
	// AlignStart is the start.
	AlignStart
)

// Mode is an enum with a named zero.
//
// Flutter: no counterpart
type Mode uint8

const (
	// ModeA is first.
	ModeA Mode = iota
	// ModeB is second.
	ModeB
)

// Flags are bit flags, so there is no named zero.
//
// Flutter: no counterpart
type Flags int

const (
	FlagA Flags = 1 << iota
	FlagB
)

// Alone has a single value.
//
// Flutter: no counterpart
type Alone int

const AloneOnly Alone = iota
`,
			want: []string{RuleEnumZero + " pkg.Mode"},
		},
		{
			name: "package variables",
			src: `package pkg

// Table is a map.
//
// Flutter: no counterpart
var Table = map[int]int{1: 2}

// Names is a slice.
//
// Flutter: no counterpart
var Names = []string{"a"}

// Flag is a bool.
//
// Flutter: no counterpart
var Flag = false

// Hook is a function.
//
// Flutter: no counterpart
var Hook func()

// Value is a struct that nobody assigns to.
//
// Flutter: no counterpart
var Value = struct{ A int }{1}

// Counter is a struct that the package assigns to.
//
// Flutter: no counterpart
var Counter = struct{ A int }{1}

func init() { Counter = struct{ A int }{2} }

var unexported = map[int]int{}
`,
			want: []string{RuleMutableVar + " pkg.Counter", RuleMutableVar + " pkg.Flag", RuleMutableVar + " pkg.Hook", RuleMutableVar + " pkg.Names", RuleMutableVar + " pkg.Table"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writePackage(t, tt.src)
			got, err := CheckDir(root, "pkg")
			if err != nil {
				t.Fatal(err)
			}
			want := append([]string(nil), tt.want...)
			sort.Strings(want)
			if g := keysOf(got); strings.Join(g, "\n") != strings.Join(want, "\n") {
				t.Errorf("violations:\n%s\nwant:\n%s", strings.Join(g, "\n"), strings.Join(want, "\n"))
			}
		})
	}
}

func TestCheckDirSkipsTests(t *testing.T) {
	root := writePackage(t, "package pkg\n")
	if err := os.WriteFile(filepath.Join(root, "pkg", "pkg_test.go"), []byte("package pkg\n\nvar Bad = map[int]int{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := CheckDir(root, "pkg")
	if err != nil || len(got) != 0 {
		t.Errorf("violations = %v, err = %v", got, err)
	}
}

func TestViolationFormat(t *testing.T) {
	root := writePackage(t, "package pkg\n\n// A has no reference.\ntype A struct{}\n")
	got, err := CheckDir(root, "pkg")
	if err != nil || len(got) != 1 {
		t.Fatalf("violations = %v, err = %v", got, err)
	}
	if want := "pkg/pkg.go:4: [doc-ref] A: "; !strings.HasPrefix(got[0].String(), want) {
		t.Errorf("String() = %q, want the prefix %q", got[0].String(), want)
	}
}

func TestRun(t *testing.T) {
	dirty := "package pkg\n\n// A has no reference.\ntype A struct{}\n"
	tests := []struct {
		name      string
		src       string
		allow     string
		wantRoute string // the Tag at the end of the route
		wantOut   string
	}{
		{"clean package", "package pkg\n", "", "", "0 violations, 0 accepted"},
		{"violation", dirty, "", "ApiCheckViolations", "[doc-ref] A"},
		{"accepted by the allow file", dirty, "doc-ref pkg.A # why\n", "", "0 violations, 1 accepted"},
		{"stale line of the allow file", "package pkg\n", "doc-ref pkg.A\n", "ApiCheckViolations", "the allow file lists a violation that is gone"},
		{"bad line of the allow file", "package pkg\n", "doc-ref\n", "ApiCheckAllowSyntax", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writePackage(t, tt.src)
			args := []string{"-root", root, "-pkgs", "pkg"}
			if tt.allow != "" {
				path := filepath.Join(t.TempDir(), "allow")
				if err := os.WriteFile(path, []byte(tt.allow), 0o644); err != nil {
					t.Fatal(err)
				}
				args = append(args, "-allow", path)
			}
			var out bytes.Buffer
			err := run(args, &out)
			if tt.wantRoute == "" {
				if err != nil {
					t.Fatalf("err = %v", err)
				}
			} else {
				appErr, ok := errors.FromError(err)
				if !ok {
					t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
				}
				if !appErr.HasRoute(tt.wantRoute) {
					t.Errorf("unexpected failure path: %s", appErr.Route())
				}
			}
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("output %q does not contain %q", out.String(), tt.wantOut)
			}
		})
	}
}

func TestRunFailures(t *testing.T) {
	root := writePackage(t, "package pkg\n")
	if err := os.MkdirAll(filepath.Join(root, "broken"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken", "b.go"), []byte("package broken\nfunc ("), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		args []string
		tag  string
	}{
		{"unknown flag", []string{"-nope"}, "ApiCheckBadFlags"},
		{"missing allow file", []string{"-root", root, "-pkgs", "pkg", "-allow", filepath.Join(root, "missing")}, "ApiCheckAllowUnreadable"},
		{"source that does not parse", []string{"-root", root, "-pkgs", "broken"}, "ApiCheckParseFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.args, &bytes.Buffer{})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.tag) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

// TestTheModuleKeepsTheConventions runs the check on the module itself: an API that breaks a
// rule fails here, unless cmd/glutter-gen/apicheck.allow accepts it.
func TestTheModuleKeepsTheConventions(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	allow, err := ReadAllow(filepath.Join(root, "cmd", "glutter-gen", "apicheck.allow"))
	if err != nil {
		t.Fatal(err)
	}
	used := map[string]bool{}
	for _, pkg := range DefaultPackages {
		vs, err := CheckDir(root, pkg)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range vs {
			if allow[v.Key()] {
				used[v.Key()] = true
				continue
			}
			t.Errorf("%s", v)
		}
	}
	for key := range allow {
		if !used[key] {
			t.Errorf("the allow file lists %q, which is not a violation any more", key)
		}
	}
}
