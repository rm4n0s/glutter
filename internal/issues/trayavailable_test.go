package issues

import (
	stdErrors "errors"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestTrayAvailableProblems(t *testing.T) {
	tests := []struct {
		name   string
		report TrayAvailableReport
		want   []string // a part of each problem, in order; none for a good report
	}{
		{"Linux with a watcher", TrayAvailableReport{"linux-watcher", TrayAnswerYes, TrayAnswerYes, TrayShown}, nil},
		{"Linux without one", TrayAvailableReport{"linux-no-watcher", TrayAnswerNo, TrayAnswerNo, TrayUnavailable}, nil},
		{"the web", TrayAvailableReport{"web", TrayAnswerNo, TrayAnswerNo, TrayUnavailable}, nil},
		{"a runner that expects nothing", TrayAvailableReport{Platform: "any", Answer: TrayAnswerYes, Tray: TrayShown}, nil},
		{"no report", TrayAvailableReport{}, []string{"no report"}},
		{"the issue: no way to ask", TrayAvailableReport{"linux-no-watcher", TrayAnswerNo, "error DesktopCallOnUIThread", TrayUnavailable},
			[]string{"TrayAvailable failed on linux-no-watcher: error DesktopCallOnUIThread"}},
		{"a wrong answer", TrayAvailableReport{"android", TrayAnswerNo, TrayAnswerYes, TrayUnavailable}, []string{
			"TrayAvailable answered yes on android, want no",
			"TrayAvailable answered yes, but the tray reported TrayUnavailable",
		}},
		{"a no and a tray that was shown", TrayAvailableReport{"windows", TrayAnswerYes, TrayAnswerNo, TrayShown}, []string{
			"answered no on windows, want yes",
			"TrayAvailable answered no, but the tray was shown",
		}},
		{"a tray error that does not match the tag", TrayAvailableReport{"linux-no-watcher", TrayAnswerNo, TrayAnswerNo, "error TrayRegisterFailed"},
			[]string{`the tray reported something other than TrayUnavailable: "error TrayRegisterFailed"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.report.Problems()
			if len(got) != len(tt.want) {
				t.Fatalf("problems = %q, want %d like %q", got, len(tt.want), tt.want)
			}
			for i := range got {
				if !strings.Contains(got[i], tt.want[i]) {
					t.Errorf("problem %d = %q, want a part %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestTrayAnswerAndOutcome(t *testing.T) {
	tagged := errors.New(tagTrayRegisterFailed, "refused")
	plain := stdErrors.New("no such thing")
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"a yes", TrayAnswerOf(true, nil), TrayAnswerYes},
		{"a no", TrayAnswerOf(false, nil), TrayAnswerNo},
		{"an error of the module", TrayAnswerOf(false, tagged), "error TrayRegisterFailed"},
		{"another error", TrayAnswerOf(true, plain), "error no_such_thing"},
		{"a tray that was shown", TrayOutcomeOf(nil, false), TrayShown},
		{"a tray that is unavailable", TrayOutcomeOf(tagged, true), TrayUnavailable},
		{"another tray error", TrayOutcomeOf(tagged, false), "error TrayRegisterFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestTrayAvailableReportIsReadBackFromItsLines(t *testing.T) {
	tests := []TrayAvailableReport{
		{"linux-watcher", TrayAnswerYes, TrayAnswerYes, TrayShown},
		{"web", "", "error DesktopWithoutBinding", "error TrayRegisterFailed"},
	}
	for _, rep := range tests {
		lines := append([]string{"I/Glutter: something else", "app: " + TrayAvailableName + " starts"}, rep.Lines()...)
		got, err := ParseTrayAvailable(lines)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, rep) {
			t.Errorf("read back %+v from %q, want %+v", got, lines, rep)
		}
	}
	if none, err := ParseTrayAvailable([]string{"nothing of the report"}); err != nil || none != (TrayAvailableReport{}) {
		t.Errorf("no report read back as %+v, %v", none, err)
	}
}

func TestParseTrayAvailableRejectsALineThatDoesNotRead(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"cut short", "tray-available: platform web want no"},
		{"another word", "tray-available: platform web want no reply no tray unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTrayAvailable([]string{tt.line})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("ParseTrayAvailable." + TagIssueReportInvalid) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["line"] != tt.line {
				t.Errorf("metadata: %v", appErr.Metadata)
			}
		})
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTrayRegisterFailed = "TrayRegisterFailed"
)
