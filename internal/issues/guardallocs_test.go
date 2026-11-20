package issues

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

// goodGuardAllocsReport is the report of a release build that has the fix.
func goodGuardAllocsReport() GuardAllocsReport {
	return GuardAllocsReport{Cases: []GuardAllocsCase{
		{Name: GuardAllocsLayout, Allocs: 0, Context: "during PerformLayout()"},
		{Name: GuardAllocsResize, Allocs: 0, Context: "during PerformResize()"},
		{Name: GuardAllocsPaint, Allocs: 0, Context: "during paint()"},
		{Name: GuardAllocsDispatch, Allocs: 1, Context: "while dispatching a pointer event to guardTarget#1a2b3"},
	}}
}

func TestGuardAllocsProblems(t *testing.T) {
	tests := []struct {
		name   string
		change func(r *GuardAllocsReport)
		want   string // a part of the one problem; empty for none
	}{
		{"the fixed release build", func(*GuardAllocsReport) {}, ""},
		{"the fixed debug build", func(r *GuardAllocsReport) {
			r.Debug, r.Cases[0].Allocs, r.Cases[1].Allocs = true, 2, 2
		}, ""},
		{"a few allocations of other goroutines", func(r *GuardAllocsReport) { r.Cases[2].Allocs = 0.005 }, ""},
		{"a case is missing", func(r *GuardAllocsReport) { r.Cases = r.Cases[:3] }, "has 3 cases"},
		{"another case", func(r *GuardAllocsReport) { r.Cases[1].Name = "measure" }, `case 1 is "measure", want "resize"`},
		{"the string of each layout, the issue", func(r *GuardAllocsReport) { r.Cases[0].Allocs = 1 }, "layout: 1.000 allocations for each child, want at most 0"},
		{"the strings of each resize and its layout", func(r *GuardAllocsReport) { r.Cases[1].Allocs = 2 }, "resize: 2.000 allocations"},
		{"the string of each paint", func(r *GuardAllocsReport) { r.Cases[2].Allocs = 1 }, "paint: 1.000 allocations"},
		{"the description of each target", func(r *GuardAllocsReport) { r.Cases[3].Allocs = 11 }, "dispatch: 11.000 allocations for each child, want at most 1"},
		{"the issue in a debug build", func(r *GuardAllocsReport) {
			r.Debug, r.Cases[0].Allocs, r.Cases[1].Allocs = true, 3, 2
		}, "layout: 3.000 allocations for each child, want at most 2"},
		{"a release build with a debug build's allocations", func(r *GuardAllocsReport) { r.Cases[1].Allocs = 2 }, "want at most 0"},
		{"a panic not reported", func(r *GuardAllocsReport) { r.Cases[2].Context = "" }, `paint: a panic was reported with the context "", want "during paint()"`},
		{"the context of another method", func(r *GuardAllocsReport) { r.Cases[1].Context = "during PerformLayout()" }, `want "during PerformResize()"`},
		{"a target not named", func(r *GuardAllocsReport) { r.Cases[3].Context = "while dispatching a pointer event" }, "dispatch: a panic was reported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := goodGuardAllocsReport()
			tt.change(&r)
			got := r.Problems()
			if tt.want == "" {
				if len(got) != 0 {
					t.Fatalf("problems of a good report: %q", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tt.want) {
				t.Fatalf("problems: %q, want one with %q", got, tt.want)
			}
		})
	}
}

func TestGuardAllocsReportIsReadBackFromItsLines(t *testing.T) {
	tests := []struct {
		name   string
		report GuardAllocsReport
	}{
		{"a good report", goodGuardAllocsReport()},
		{"a bad one of a debug build", func() GuardAllocsReport {
			r := goodGuardAllocsReport()
			r.Debug, r.Cases[0].Allocs, r.Cases[3].Allocs, r.Cases[2].Context = true, 3.25, 11, ""
			return r
		}()},
		{"no case", GuardAllocsReport{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The lines among the other output of an app, each after a prefix of the log.
			lines := []string{"renderer ready", ""}
			for _, l := range tt.report.Lines() {
				lines = append(lines, "2026-11-20 I/app: "+l, "another line")
			}
			got, err := ParseGuardAllocs(lines)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.report) {
				t.Errorf("read back\n%+v\nwant\n%+v", got, tt.report)
			}
		})
	}
}

func TestParseGuardAllocsRejectsALineThatDoesNotRead(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"cut short", `guard-allocs: case "layout" allocs`},
		{"a number that is not one", `guard-allocs: case "paint" allocs none context "during paint()"`},
		{"a build that is not a bool", `guard-allocs: debug maybe`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseGuardAllocs([]string{tt.line})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("ParseGuardAllocs." + TagIssueReportInvalid) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["line"] != tt.line {
				t.Errorf("metadata: %v", appErr.Metadata)
			}
		})
	}
}

// TestLaysOutAndPaintsWithoutBuildingAString runs the reproduction of Fishball's issue of step
// 52.1 (guard-allocs) in the test process: the layout, resize and paint of a render object, and the
// dispatch of a pointer event to a target, allocate nothing for the recovery point around them, and
// a panic in each is still reported with the context that names it. It is the runner of the
// platforms whose test process is the platform: Linux here, Windows under Wine
// (scripts/wine-exec.sh) and WebAssembly under Node (scripts/test-wasm.sh); Android and the browsers
// run the same reproduction in an app (internal/androidtest, internal/gpu/canvas2d). Before the
// step a release build made 1 allocation for each layout and paint, 2 for each resize and 11 for
// each target of a dispatch.
func TestLaysOutAndPaintsWithoutBuildingAString(t *testing.T) {
	rep := RunGuardAllocs()
	for _, line := range rep.Lines() {
		t.Log(line)
	}
	for _, problem := range rep.Problems() {
		t.Error(problem)
	}
}
