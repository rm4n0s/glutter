package main

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/perfstats"
)

// perfReport is what `glutter-android perf` measured.
type perfReport struct {
	Package string
	Release bool
	// Env are the extras the app was given by -env and -renderer, KEY=value.
	Env      []string
	Device   deviceInfo
	Renderer string
	Sizes    []sizeRow
	Starts   []startRun
	Idle     idleResult
	Scenario scenarioResult
	Memory   []memSample
}

// deviceInfo describes the device from its system properties.
type deviceInfo struct {
	Serial, Model, Android, SDK, ABIList, ABI string
}

// sizeRow is the size of one APK: Label is the ABI, or the ABIs joined with "+"; Libs the size of
// libapp.so inside it by the name of the ABI.
type sizeRow struct {
	Label string
	ABIs  []string
	APK   int64
	Libs  map[string]int64
}

// startRun is one cold start.
type startRun struct {
	Total, Wait, This time.Duration
	LaunchState       string
	Frame             firstFrame
}

// idleResult is the idle window: Ticks of utime+stime over Window, and the frames the engine
// logged in it.
type idleResult struct {
	Window     time.Duration
	Ticks      uint64
	CPUPercent float64
	Frames     int
}

// scenarioResult is the frame window.
type scenarioResult struct {
	Ran     bool
	Window  time.Duration
	Frames  []perfstats.Frames
	Summary frameSummary
}

// memSample is the memory of the app at one moment, in kilobytes.
type memSample struct {
	Label            string
	PSSKB, RSSKB     int
	HasRSS           bool
	VmRSSKB, VmHWMKB int
}

// table is a titled table of text cells: rendered as aligned columns or as a Markdown table.
type table struct {
	title  string
	header []string
	rows   [][]string
}

// reportDoc is the report as key/value lines and tables, which both formats print.
type reportDoc struct {
	head   [][2]string
	tables []table
}

func ms(d time.Duration) string  { return fmt.Sprintf("%.1f", float64(d)/float64(time.Millisecond)) }
func ms2(d time.Duration) string { return fmt.Sprintf("%.2f", float64(d)/float64(time.Millisecond)) }
func mb(bytes int64) string      { return fmt.Sprintf("%.2f", float64(bytes)/(1<<20)) }
func kbMB(kb int) string         { return fmt.Sprintf("%.1f", float64(kb)/1024) }

// document turns the report into the tables that both formats print.
func (r *perfReport) document() reportDoc {
	build := "debug"
	if r.Release {
		build = "release (-tags glutter_release, stripped)"
	}
	dev := r.Device.Serial
	if r.Device.Model != "" {
		dev += " " + r.Device.Model
	}
	dev += fmt.Sprintf(", Android %s (API %s), runs %s", r.Device.Android, r.Device.SDK, r.Device.ABI)
	doc := reportDoc{head: [][2]string{{"app", r.Package}, {"build", build}, {"device", dev}}}
	if r.Renderer != "" {
		doc.head = append(doc.head, [2]string{"renderer", r.Renderer})
	}
	if len(r.Env) > 0 {
		doc.head = append(doc.head, [2]string{"environment", strings.Join(r.Env, " ")})
	}

	sizes := table{title: "APK size", header: []string{"ABIs", "APK bytes", "APK MB", "libapp.so bytes (per ABI)"}}
	for _, s := range r.Sizes {
		var libs []string
		for _, abi := range s.ABIs {
			libs = append(libs, fmt.Sprintf("%s %d", abi, s.Libs[abi]))
		}
		sizes.rows = append(sizes.rows, []string{s.Label, fmt.Sprint(s.APK), mb(s.APK), strings.Join(libs, ", ")})
	}
	doc.tables = append(doc.tables, sizes)

	if len(r.Starts) > 0 {
		doc.tables = append(doc.tables, r.startTable())
	}
	doc.tables = append(doc.tables, table{
		title:  "Idle CPU (no input)",
		header: []string{"window s", "CPU ticks (utime+stime)", "CPU % of one core", "frames logged in the window"},
		rows:   [][]string{{fmt.Sprintf("%.1f", r.Idle.Window.Seconds()), fmt.Sprint(r.Idle.Ticks), fmt.Sprintf("%.2f", r.Idle.CPUPercent), fmt.Sprint(r.Idle.Frames)}},
	})
	if r.Scenario.Ran {
		doc.tables = append(doc.tables, r.frameTables()...)
	}
	if len(r.Memory) > 0 {
		mem := table{title: "Memory (MB)", header: []string{"when", "dumpsys TOTAL PSS", "dumpsys TOTAL RSS", "VmRSS", "VmHWM"}}
		for _, s := range r.Memory {
			rss := "n/a"
			if s.HasRSS {
				rss = kbMB(s.RSSKB)
			}
			mem.rows = append(mem.rows, []string{s.Label, kbMB(s.PSSKB), rss, kbMB(s.VmRSSKB), kbMB(s.VmHWMKB)})
		}
		doc.tables = append(doc.tables, mem)
	}
	return doc
}

// startTable has a row for the first start (the coldest), one for each later start, and the
// minimum, the median and the maximum over all of them.
func (r *perfReport) startTable() table {
	t := table{title: fmt.Sprintf("Cold start (%d starts; am start -W, and the engine's first frame)", len(r.Starts)),
		header: []string{"start", "TotalTime ms", "WaitTime ms", "engine first frame (since_run) ms"}}
	var total, wait, frame []time.Duration
	for i, s := range r.Starts {
		name := fmt.Sprintf("%d", i+1)
		if i == 0 {
			name += " (first)"
		}
		t.rows = append(t.rows, []string{name, ms(s.Total), ms(s.Wait), ms(s.Frame.SinceRun)})
		total, wait, frame = append(total, s.Total), append(wait, s.Wait), append(frame, s.Frame.SinceRun)
	}
	ts, ws, fs := summarizeDurations(total), summarizeDurations(wait), summarizeDurations(frame)
	for _, row := range []struct {
		name string
		get  func(durStats) time.Duration
	}{
		{"min", func(d durStats) time.Duration { return d.Min }},
		{"median", func(d durStats) time.Duration { return d.Median }},
		{"max", func(d durStats) time.Duration { return d.Max }},
	} {
		t.rows = append(t.rows, []string{row.name, ms(row.get(ts)), ms(row.get(ws)), ms(row.get(fs))})
	}
	return t
}

// frameTables are the summary of the frame window and the statistics of every second.
func (r *perfReport) frameTables() []table {
	s := r.Scenario.Summary
	sum := table{title: fmt.Sprintf("Frame times (GLUTTER_FRAME_STATS=1, %.0f s after the first frame)", r.Scenario.Window.Seconds()),
		header: []string{"metric", "value"}}
	if s.Seconds == 0 {
		sum.rows = [][]string{{"frames", "none logged: the app drew no frame in the window"}}
		return []table{sum}
	}
	sum.rows = [][]string{
		{"seconds with frames", fmt.Sprint(s.Seconds)},
		{"frames", fmt.Sprint(s.Frames)},
		{"FPS mean (min, max)", fmt.Sprintf("%.1f (%d, %d)", s.MeanFPS, s.MinFPS, s.MaxFPS)},
		{"build ms avg / worst", ms2(s.BuildAvg) + " / " + ms2(s.BuildMax)},
		{"raster ms avg / worst", ms2(s.RasterAvg) + " / " + ms2(s.RasterMax)},
		{"worst second, build+raster avg ms", ms2(s.WorstSecond)},
		{"frames over 16 ms", fmt.Sprintf("%d (%.1f%%)", s.Over16, s.Over16Percent)},
		{"Go heap MB, max / last", fmt.Sprintf("%d / %d", s.HeapMaxMB, s.HeapLastMB)},
		{"Go sys MB, last", fmt.Sprint(s.SysLastMB)},
	}
	per := table{title: "Frame times per second", header: []string{"second", "FPS", "build avg ms", "build max ms", "raster avg ms", "raster max ms", "over 16 ms", "Go heap MB"}}
	for i, f := range r.Scenario.Frames {
		per.rows = append(per.rows, []string{fmt.Sprint(i + 1), fmt.Sprint(f.FPS), ms2(f.BuildAvg), ms2(f.BuildMax), ms2(f.RasterAvg), ms2(f.RasterMax), fmt.Sprint(f.Over16), fmt.Sprint(f.GoHeapMB)})
	}
	return []table{sum, per}
}

// render writes the report as "text" (aligned columns) or "markdown" (for docs/PERFORMANCE.md).
// It fails with AndroidPerfWriteFailed when w does.
func (r *perfReport) render(w io.Writer, format string) error {
	doc := r.document()
	var b strings.Builder
	md := format == "markdown"
	if md {
		b.WriteString("### glutter-android perf\n\n")
		for _, kv := range doc.head {
			fmt.Fprintf(&b, "- **%s**: %s\n", kv[0], kv[1])
		}
	} else {
		b.WriteString("glutter-android perf\n")
		for _, kv := range doc.head {
			fmt.Fprintf(&b, "%-12s %s\n", kv[0]+":", kv[1])
		}
	}
	for _, t := range doc.tables {
		b.WriteString("\n")
		if md {
			b.WriteString("#### " + t.title + "\n\n")
			b.WriteString("| " + strings.Join(t.header, " | ") + " |\n")
			b.WriteString("|" + strings.Repeat("---|", len(t.header)) + "\n")
			for _, row := range t.rows {
				b.WriteString("| " + strings.Join(row, " | ") + " |\n")
			}
			continue
		}
		b.WriteString(t.title + "\n")
		tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
		for _, row := range append([][]string{t.header}, t.rows...) {
			if _, err := io.WriteString(tw, "  "+strings.Join(row, "\t")+"\n"); err != nil {
				return errors.NewErr(tagAndroidPerfWriteFailed, err)
			}
		}
		if err := tw.Flush(); err != nil {
			return errors.NewErr(tagAndroidPerfWriteFailed, err)
		}
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return errors.NewErr(tagAndroidPerfWriteFailed, err)
	}
	return nil
}
