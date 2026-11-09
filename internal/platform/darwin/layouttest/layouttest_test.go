package layouttest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

type rightRect struct {
	Left int32     `c:"left"`
	Top  int32     `c:"top"`
	Name *uint16   `c:"name"`
	Tip  [8]uint16 `c:"tip"`
}

type wrongRect struct { // Left is wider than the header says, so Top moves
	Left int64     `c:"left"`
	Name *uint16   `c:"name"`
	Tip  [8]uint16 `c:"tip"`
	Top  int32     `c:"top"`
}

type withSkipped struct {
	A int32 `c:"a"`
	_ int32
	B int64 `c:"-"`
	C int32 `c:"c"`
}

const fixtureHeader = `
#include <stdint.h>
typedef struct {
  int32_t left;
  int32_t top;
  uint16_t *name;
  uint16_t tip[8];
} FixtureRect;
#define FIXTURE_FAIL 0x80004005
`

func TestStructOf(t *testing.T) {
	got := StructOf("FixtureRect", rightRect{})
	want := Struct{CName: "FixtureRect", Size: 32, Fields: []Field{
		{"left", 0, 4}, {"top", 4, 4}, {"name", 8, 8}, {"tip", 16, 16},
	}}
	if got.CName != want.CName || got.Size != want.Size || len(got.Fields) != len(want.Fields) {
		t.Fatalf("StructOf = %+v, want %+v", got, want)
	}
	for i := range want.Fields {
		if got.Fields[i] != want.Fields[i] {
			t.Errorf("field %d = %+v, want %+v", i, got.Fields[i], want.Fields[i])
		}
	}
	skipped := StructOf("S", withSkipped{})
	if len(skipped.Fields) != 2 || skipped.Fields[0].CName != "a" || skipped.Fields[1].CName != "c" {
		t.Errorf("padding and c:\"-\" fields were not skipped: %+v", skipped.Fields)
	}
}

func TestStructOfPanics(t *testing.T) {
	tests := []struct {
		name string
		v    any
	}{
		{"not a struct", 3},
		{"nil", nil},
		{"a field without a c tag", struct{ A int32 }{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("StructOf did not panic")
				}
			}()
			StructOf("X", tt.v)
		})
	}
}

func TestSource(t *testing.T) {
	spec := Spec{
		Headers: []string{"CoreGraphics/CoreGraphics.h"},
		Defines: []string{"FOO 1"},
		Structs: []Struct{{CName: "CGPoint", Size: 16, Fields: []Field{{"x", 0, 8}, {"y", 8, 8}}}},
		Consts:  []Const{{"NSUTF8StringEncoding", 4}},
	}
	src := spec.Source()
	for _, want := range []string{
		"#define FOO 1\n",
		"#import <CoreGraphics/CoreGraphics.h>\n",
		`_Static_assert(sizeof(CGPoint) == 16, "CGPoint: size");`,
		`_Static_assert(offsetof(CGPoint, y) == 8, "CGPoint.y: offset");`,
		`_Static_assert(sizeof(((CGPoint *)0)->x) == 8, "CGPoint.x: size");`,
		`_Static_assert((long long)(NSUTF8StringEncoding) == 4LL, "NSUTF8StringEncoding: value");`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("the source does not contain %q:\n%s", want, src)
		}
	}
}

func TestParseOutput(t *testing.T) {
	tests := []struct {
		name       string
		out        string
		wantFailed []string
		wantHeader string
	}{
		{"nothing", "", nil, ""},
		{"two assertions", "l.m:5:16: error: static assertion failed due to requirement 'sizeof(CGRect) == 8': CGRect: size\n    5 | _Static_assert(...);\nl.m:6:16: error: static assertion failed due to requirement 'offsetof(CGRect, size) == 4': CGRect.size: offset\n", []string{"CGRect: size", "CGRect.size: offset"}, ""},
		{"a missing header", "l.m:2:9: fatal error: 'AppKit/Nothing.h' file not found\n", nil, "AppKit/Nothing.h"},
		{"another error", "l.m:5:1: error: unknown type name 'FOO'\n", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, header := parseOutput([]byte(tt.out))
			if strings.Join(failed, "|") != strings.Join(tt.wantFailed, "|") || header != tt.wantHeader {
				t.Errorf("parseOutput = %v, %q; want %v, %q", failed, header, tt.wantFailed, tt.wantHeader)
			}
		})
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		c    Checker
		want string
	}{
		{"the defaults", Checker{}, "xcrun --sdk macosx clang -fsyntax-only -x objective-c -std=gnu11 f.m"},
		{"an iOS SDK with a target", Checker{SDK: "iphonesimulator", Args: []string{"-target", "arm64-apple-ios15.0-simulator"}},
			"xcrun --sdk iphonesimulator clang -fsyntax-only -x objective-c -std=gnu11 -target arm64-apple-ios15.0-simulator f.m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strings.Join(tt.c.command("xcrun", "f.m"), " "); got != tt.want {
				t.Errorf("command = %q, want %q", got, tt.want)
			}
		})
	}
}

// The tag of the error that a fake runner makes.
const tagRunnerBroke = "RunnerBroke"

func TestCheckWithAFakeCompiler(t *testing.T) {
	found := func(string) (string, error) { return "/usr/bin/xcrun", nil }
	run := func(out string, code int) func(context.Context, []string) ([]byte, int, error) {
		return func(context.Context, []string) ([]byte, int, error) { return []byte(out), code, nil }
	}
	tests := []struct {
		name      string
		c         Checker
		wantTag   string // "" means success
		wantRoute string
		wantKey   string
	}{
		{"every assertion holds", Checker{LookPath: found, Run: run("", 0)}, "", "", ""},
		{"a mismatch names the structure", Checker{LookPath: found, Run: run("x: error: static assertion failed due to requirement 'sizeof(CGRect) == 8': CGRect: size\n", 1)},
			tagDarwinLayoutMismatch, "Checker.Check." + tagDarwinLayoutMismatch, "failed"},
		{"a header that is not there", Checker{LookPath: found, Run: run("x: fatal error: 'Nope/Nope.h' file not found\n", 1)},
			tagDarwinLayoutHeaderMissing, "Checker.Check." + tagDarwinLayoutHeaderMissing, "header"},
		{"another compiler error", Checker{LookPath: found, Run: run("x: error: unknown type name 'Foo'\n", 1)},
			tagDarwinLayoutCompileFailed, "Checker.Check." + tagDarwinLayoutCompileFailed, "command"},
		{"no xcrun", Checker{LookPath: func(string) (string, error) { return "", os.ErrNotExist }},
			tagDarwinLayoutCompilerMissing, "Checker.Check." + tagDarwinLayoutCompilerMissing, "install"},
		{"a compiler that cannot be started", Checker{LookPath: found, Run: func(context.Context, []string) ([]byte, int, error) {
			return nil, -1, errors.New(tagRunnerBroke, "cannot start")
		}}, tagRunnerBroke, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.Check(context.Background(), Spec{})
			if tt.wantTag == "" {
				if err != nil {
					t.Fatalf("Check: %v", err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tt.wantTag {
				t.Fatalf("got %v, want an error tagged %s", err, tt.wantTag)
			}
			if tt.wantRoute != "" && !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("unexpected failure path: %s, want it to contain %s", appErr.Route(), tt.wantRoute)
			}
			if tt.wantKey != "" {
				if _, has := appErr.Metadata[tt.wantKey]; !has {
					t.Errorf("metadata %v has no %q", appErr.Metadata, tt.wantKey)
				}
			}
		})
	}
}

// TestRealCompilerChecksAFixture runs the SDK's clang over a fixture header, so the harness is checked without
// the SDK's own types: the right mirror passes, the wrong ones fail and name the structure. It is skipped on
// a machine without xcrun.
func TestRealCompilerChecksAFixture(t *testing.T) {
	if _, err := exec.LookPath(DefaultCompiler); err != nil {
		t.Skipf("no %s on this machine (%s)", DefaultCompiler, InstallCommand)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fixture.h"), []byte(fixtureHeader), 0o600); err != nil {
		t.Fatal(err)
	}
	checker := Checker{Args: []string{"-I" + dir}}
	base := Spec{Headers: []string{"fixture.h"}}
	with := func(f func(*Spec)) Spec {
		s := base
		f(&s)
		return s
	}
	tests := []struct {
		name       string
		spec       Spec
		wantFailed []string // nil: the check passes
	}{
		{"the right mirror", with(func(s *Spec) {
			s.Structs = []Struct{StructOf("FixtureRect", rightRect{})}
			s.Consts = []Const{{"FIXTURE_FAIL", 0x80004005}}
		}), nil},
		{"a wrong size and offset", with(func(s *Spec) { s.Structs = []Struct{StructOf("FixtureRect", wrongRect{})} }),
			[]string{"FixtureRect: size", "FixtureRect.left: size", "FixtureRect.top: offset"}},
		{"a wrong constant", with(func(s *Spec) { s.Consts = []Const{{"FIXTURE_FAIL", 0x80004006}} }), []string{"FIXTURE_FAIL: value"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.Check(context.Background(), tt.spec)
			if tt.wantFailed == nil {
				if err != nil {
					t.Fatalf("the right mirror failed: %v", err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tagDarwinLayoutMismatch {
				t.Fatalf("got %v, want DarwinLayoutMismatch", err)
			}
			failed, _ := appErr.Metadata["failed"].([]string)
			for _, want := range tt.wantFailed {
				found := false
				for _, f := range failed {
					found = found || f == want
				}
				if !found {
					t.Errorf("failed = %v, want %q among them", failed, want)
				}
			}
		})
	}
	t.Run("a member the header does not have", func(t *testing.T) {
		err := checker.Check(context.Background(), with(func(s *Spec) {
			s.Structs = []Struct{{CName: "FixtureRect", Size: 32, Fields: []Field{{CName: "nothing", Size: 4}}}}
		}))
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagDarwinLayoutCompileFailed {
			t.Errorf("got %v, want DarwinLayoutCompileFailed", err)
		}
	})
	t.Run("a header that is not there", func(t *testing.T) {
		err := checker.Check(context.Background(), Spec{Headers: []string{"nothing_here.h"}})
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagDarwinLayoutHeaderMissing {
			t.Errorf("got %v, want DarwinLayoutHeaderMissing", err)
		}
	})
}
