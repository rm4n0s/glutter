package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/internal/perfstats"
)

// perfReport is what `perf` measured.
type perfReport struct {
	Package  string
	Release  bool
	Env      []string
	Machine  string
	Renderer string
	Archs    []string
	Size     int64
	Thin     map[string]int64
	// Empty is the first start, with an empty shader cache, and Warm the others.
	Empty, Warm []startResult
	Window      windowResult
}

// document turns the report into the tables that both formats print.
func (r perfReport) document() perfstats.Doc {
	build := "debug"
	if r.Release {
		build = "release (-tags glutter_release, stripped)"
	}
	doc := perfstats.Doc{Head: [][2]string{{"app", r.Package}, {"build", build}, {"where", r.Machine}}}
	if r.Renderer != "" {
		doc.Head = append(doc.Head, [2]string{"renderer", r.Renderer})
	}
	if len(r.Env) > 0 {
		doc.Head = append(doc.Head, [2]string{"environment", strings.Join(r.Env, " ")})
	}
	size := perfstats.Table{Title: "Executable", Header: []string{"file", "MB"}}
	size.Rows = append(size.Rows, []string{strings.Join(r.Archs, "+"), perfstats.BytesToMB(r.Size)})
	arches := make([]string, 0, len(r.Thin))
	for a := range r.Thin {
		arches = append(arches, a)
	}
	sort.Strings(arches)
	if len(arches) > 1 {
		for _, a := range arches {
			size.Rows = append(size.Rows, []string{"slice " + a, perfstats.BytesToMB(r.Thin[a])})
		}
	}
	start := perfstats.Table{Title: "Start to the first frame", Header: []string{"shader cache", "starts", "since Run median (ms)", "since process median (ms)", "whole run median (ms)", "worst since Run (ms)"}}
	for _, g := range []struct {
		name string
		s    []startResult
	}{{"empty", r.Empty}, {"warm", r.Warm}} {
		if len(g.s) == 0 {
			continue
		}
		var run, proc, wall []time.Duration
		for _, s := range g.s {
			run, proc, wall = append(run, s.First.SinceRun), append(proc, s.First.SinceProcess), append(wall, s.Wall)
		}
		sinceProcess := "-" // the engine did not know when the process started
		if m := perfstats.Median(proc); m > 0 {
			sinceProcess = perfstats.Ms(m)
		}
		start.Rows = append(start.Rows, []string{g.name, fmt.Sprint(len(g.s)), perfstats.Ms(perfstats.Median(run)), sinceProcess,
			perfstats.Ms(perfstats.Median(wall)), perfstats.Ms(perfstats.SummarizeDurations(run).Max)})
	}
	idle := perfstats.Table{Title: fmt.Sprintf("Window of %d s", r.Window.Seconds), Header: []string{"what", "value"}, Rows: [][]string{
		{"CPU of the process (percent of one core)", fmt.Sprintf("%.1f", r.Window.CPU)},
		{"resident memory at the end (MB)", perfstats.KBToMB(r.Window.RSSKB)},
	}}
	doc.Tables = append(doc.Tables, size, start, idle)
	if r.Window.Count > 0 {
		f := r.Window.Frames
		doc.Tables = append(doc.Tables, perfstats.Table{Title: "Frames (the seconds that had a frame)", Header: []string{"what", "value"}, Rows: [][]string{
			{"seconds, frames", fmt.Sprintf("%d, %d", f.Seconds, f.Frames)},
			{"frames a second: mean, lowest, highest", fmt.Sprintf("%.1f, %d, %d", f.MeanFPS, f.MinFPS, f.MaxFPS)},
			{"build: average, worst (ms)", perfstats.Ms2(f.BuildAvg) + ", " + perfstats.Ms2(f.BuildMax)},
			{"raster: average, worst (ms)", perfstats.Ms2(f.RasterAvg) + ", " + perfstats.Ms2(f.RasterMax)},
			{"frames over 16 ms", fmt.Sprintf("%d (%.1f%%)", f.Over16, f.Over16Percent)},
			{"Go heap: highest, last; from the system (MB)", fmt.Sprintf("%d, %d; %d", f.HeapMaxMB, f.HeapLastMB, f.SysLastMB)},
		}})
	}
	return doc
}
