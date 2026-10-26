package perfstats

import (
	stdErrors "errors"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

func TestDocRender(t *testing.T) {
	doc := Doc{
		Head:   [][2]string{{"app", "./examples/counter"}, {"build", "release"}},
		Tables: []Table{{Title: "Sizes", Header: []string{"arch", "MB"}, Rows: [][]string{{"amd64", "37.0"}, {"arm64", "35.0"}}}},
	}
	tests := []struct {
		format string
		want   string
	}{
		{"text", "prog\napp:         ./examples/counter\nbuild:       release\n\nSizes\n  arch   MB\n  amd64  37.0\n  arm64  35.0\n"},
		{"markdown", "### prog\n\n- **app**: ./examples/counter\n- **build**: release\n\n#### Sizes\n\n| arch | MB |\n|---|---|\n| amd64 | 37.0 |\n| arm64 | 35.0 |\n"},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var b strings.Builder
			if err := doc.Render(&b, "prog", tt.format); err != nil {
				t.Fatal(err)
			}
			if b.String() != tt.want {
				t.Errorf("Render =\n%q\nwant\n%q", b.String(), tt.want)
			}
		})
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, stdErrors.New("closed pipe") }

func TestDocRenderFailure(t *testing.T) {
	err := Doc{}.Render(failingWriter{}, "prog", "text")
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("Doc.Render."+TagPerfWriteFailed) {
		t.Errorf("got %v, want PerfWriteFailed from Doc.Render", err)
	}
}

func TestFormats(t *testing.T) {
	if got := Ms(1500 * time.Microsecond); got != "1.5" {
		t.Errorf("Ms = %q", got)
	}
	if got := Ms2(1234 * time.Microsecond); got != "1.23" {
		t.Errorf("Ms2 = %q", got)
	}
	if got := KBToMB(1536); got != "1.5" {
		t.Errorf("KBToMB = %q", got)
	}
	if got := BytesToMB(3 << 19); got != "1.50" {
		t.Errorf("BytesToMB = %q", got)
	}
}
