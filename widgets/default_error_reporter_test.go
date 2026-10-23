package widgets_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/widgets"
)

// logEntry is a line of the JSON log, with the error as the attribute "err".
type logEntry struct {
	Level string          `json:"level"`
	Msg   string          `json:"msg"`
	Err   json.RawMessage `json:"err"`
}

func reportOnce(t *testing.T, err error) logEntry {
	t.Helper()
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, nil))
	widgets.DefaultErrorReporter(logger)(err)
	var entry logEntry
	if jerr := json.Unmarshal(out.Bytes(), &entry); jerr != nil {
		t.Fatalf("the log line is not JSON: %v\n%s", jerr, out.String())
	}
	return entry
}

func failWithMetadata() error {
	return errors.New(tagReporterTestFailed, "the reporter test failed", "widget", "Padding", "depth", 3)
}

func TestDefaultErrorReporterLogsTheErrorAsErr(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
		wantTag string
	}{
		{"tagged error", failWithMetadata(), "glutter: ReporterTestFailed: the reporter test failed", "ReporterTestFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := reportOnce(t, tt.err)
			if entry.Level != "ERROR" || entry.Msg != tt.wantMsg {
				t.Errorf("level %q, message %q; want ERROR, %q", entry.Level, entry.Msg, tt.wantMsg)
			}
			var logged errors.ErrorJson
			if err := json.Unmarshal(entry.Err, &logged); err != nil {
				t.Fatal(err)
			}
			if logged.Tag != tt.wantTag {
				t.Errorf("Tag = %q, want %q", logged.Tag, tt.wantTag)
			}
			if got := logged.Metadata["widget"]; got != "Padding" {
				t.Errorf("metadata widget = %v", got)
			}
			if got := logged.Metadata["depth"]; got != float64(3) {
				t.Errorf("metadata depth = %v (%T), want 3 as a JSON number", got, got)
			}
			if !logged.HasRoute("failWithMetadata.ReporterTestFailed") {
				t.Errorf("the logged route is %q", logged.Route())
			}
			replayed := errors.FromJson(logged)
			if replayed.Tag != logged.Tag || !replayed.HasRoute("failWithMetadata.ReporterTestFailed") {
				t.Errorf("the replayed error is %s via %s", replayed.Tag, replayed.Route())
			}
		})
	}
}

func TestDefaultErrorReporterLogsAnUntaggedError(t *testing.T) {
	var out bytes.Buffer
	widgets.DefaultErrorReporter(slog.New(slog.NewJSONHandler(&out, nil)))(plainFailure{})
	line := out.String()
	for _, want := range []string{`"level":"ERROR"`, `"msg":"glutter: plain failure"`, `"err":"plain failure"`} {
		if !strings.Contains(line, want) {
			t.Errorf("the log line %s does not contain %s", line, want)
		}
	}
}

type plainFailure struct{}

func (plainFailure) Error() string { return "plain failure" }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagReporterTestFailed = "ReporterTestFailed"
