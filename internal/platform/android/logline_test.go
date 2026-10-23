package android

import (
	"reflect"
	"testing"
)

func TestLinePriority(t *testing.T) {
	tests := []struct {
		line string
		want int
	}{
		{`time=2026-10-23T10:00:00.000+02:00 level=INFO msg=hello`, logInfo},
		{`time=2026-10-23T10:00:00.000+02:00 level=WARN msg="a warning"`, logWarn},
		{`time=2026-10-23T10:00:00.000+02:00 level=ERROR msg=boom`, logError},
		{`time=2026-10-23T10:00:00.000+02:00 level=DEBUG msg=x`, logDebug},
		{`{"time":"t","level":"ERROR","msg":"boom"}`, logError},
		{`{"time":"t","level":"WARN","msg":"w"}`, logWarn},
		{`2026/10/23 10:00:00 WARN android-hello: a warning line`, logWarn},
		{`2026/10/23 10:00:00 ERROR android-hello: an error line`, logError},
		{`2026/10/23 10:00:00 INFO android-hello: hello`, logInfo},
		{`2026/10/23 10:00:00 DEBUG x`, logDebug},
		{`WARN something`, logWarn},
		{`just a line printed with fmt.Println`, logInfo},
		{`a line that says ERROR in the middle of a sentence`, logInfo},
		{``, logInfo},
	}
	for _, tt := range tests {
		if got := linePriority(tt.line); got != tt.want {
			t.Errorf("linePriority(%q) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		in    string
		lines []string
		rest  string
	}{
		{"a\nb\nc", []string{"a", "b"}, "c"},
		{"a\r\nb\r\n", []string{"a", "b"}, ""},
		{"no newline", nil, "no newline"},
		{"\n\n", []string{"", ""}, ""},
		{"", nil, ""},
	}
	for _, tt := range tests {
		lines, rest := splitLines(tt.in)
		if !reflect.DeepEqual(lines, tt.lines) || rest != tt.rest {
			t.Errorf("splitLines(%q) = %q, %q; want %q, %q", tt.in, lines, rest, tt.lines, tt.rest)
		}
	}
}
