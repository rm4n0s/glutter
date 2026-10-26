package perfstats

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rm4n0s/errors"
)

// Table is a titled table of text cells: rendered as aligned columns or as a Markdown table.
//
// Flutter: no counterpart
type Table struct {
	Title  string
	Header []string
	Rows   [][]string
}

// Doc is a report as key/value lines and tables, which both formats print.
//
// Flutter: no counterpart
type Doc struct {
	Head   [][2]string
	Tables []Table
}

// Ms formats a duration as milliseconds with one decimal.
//
// Flutter: no counterpart
func Ms(d time.Duration) string { return fmt.Sprintf("%.1f", float64(d)/float64(time.Millisecond)) }

// Ms2 formats a duration as milliseconds with two decimals.
//
// Flutter: no counterpart
func Ms2(d time.Duration) string { return fmt.Sprintf("%.2f", float64(d)/float64(time.Millisecond)) }

// KBToMB formats kilobytes as megabytes with one decimal.
//
// Flutter: no counterpart
func KBToMB(kb int) string { return fmt.Sprintf("%.1f", float64(kb)/1024) }

// BytesToMB formats bytes as megabytes with two decimals.
//
// Flutter: no counterpart
func BytesToMB(bytes int64) string { return fmt.Sprintf("%.2f", float64(bytes)/(1<<20)) }

// Render writes the document as "text" (aligned columns) or "markdown" (for docs/PERFORMANCE.md);
// program is the name in its first line ("glutter-android perf"). It fails with PerfWriteFailed when w does.
//
// Flutter: no counterpart
func (doc Doc) Render(w io.Writer, program, format string) error {
	var b strings.Builder
	md := format == "markdown"
	if md {
		b.WriteString("### " + program + "\n\n")
		for _, kv := range doc.Head {
			fmt.Fprintf(&b, "- **%s**: %s\n", kv[0], kv[1])
		}
	} else {
		b.WriteString(program + "\n")
		for _, kv := range doc.Head {
			fmt.Fprintf(&b, "%-12s %s\n", kv[0]+":", kv[1])
		}
	}
	for _, t := range doc.Tables {
		b.WriteString("\n")
		if md {
			b.WriteString("#### " + t.Title + "\n\n")
			b.WriteString("| " + strings.Join(t.Header, " | ") + " |\n")
			b.WriteString("|" + strings.Repeat("---|", len(t.Header)) + "\n")
			for _, row := range t.Rows {
				b.WriteString("| " + strings.Join(row, " | ") + " |\n")
			}
			continue
		}
		b.WriteString(t.Title + "\n")
		tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
		for _, row := range append([][]string{t.Header}, t.Rows...) {
			// A strings.Builder never fails to write.
			_, _ = io.WriteString(tw, "  "+strings.Join(row, "\t")+"\n")
		}
		_ = tw.Flush() // the same
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return errors.NewErr(TagPerfWriteFailed, err)
	}
	return nil
}

// TagPerfWriteFailed is the tag of the error for a report that cannot be written to its writer.
const TagPerfWriteFailed = "PerfWriteFailed"
