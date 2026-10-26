package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/internal/perfstats"
)

// cacheState is the state of the shader cache at a cold start.
type cacheState string

const (
	// cacheEmpty is a start after the shader cache of the prefix was emptied: the shaders are compiled.
	cacheEmpty cacheState = "empty"
	// cacheWarm is a start with the cache that the earlier runs filled: the shaders are read from it.
	cacheWarm cacheState = "warm"
)

// perfReport is what `glutter-windows perf` measured.
type perfReport struct {
	Package string
	Release bool
	// Env are the entries the app was given by -env and -renderer, KEY=value.
	Env      []string
	Renderer string
	Sizes    []sizeRow
	Starts   []startRun
	Idle     idleResult
	Scenario scenarioResult
	Memory   []memSample
}

// sizeRow is the size of the .exe of one architecture.
type sizeRow struct {
	Arch  string
	Bytes int64
}

// startRun is one cold start.
type startRun struct {
	Cache cacheState
	// Wall is the time the whole run took, Wine and the X server included, from the start of the command
	// to its end (the app saved its first frame and ended).
	Wall   time.Duration
	Frame  firstFrame
	Stages startupStages
}

// procCPU is the CPU time of one Wine process over the idle window.
type procCPU struct {
	PID     int
	Name    string
	IsApp   bool
	Ticks   uint64
	Percent float64
}

// idleResult is the idle window: the CPU time of each Wine process of the prefix over it, and the
// frames the engine logged in it.
type idleResult struct {
	Window time.Duration
	Procs  []procCPU
	Frames int
	// AppTicks, WineTicks, AppPercent and WinePercent sum the app's process and all of the prefix's
	// (the app included); sum computes them from Procs.
	AppTicks, WineTicks     uint64
	AppPercent, WinePercent float64
}

// sum fills the totals from the processes.
func (r *idleResult) sum() {
	r.AppTicks, r.WineTicks, r.AppPercent, r.WinePercent = 0, 0, 0, 0
	for _, p := range r.Procs {
		r.WineTicks += p.Ticks
		r.WinePercent += p.Percent
		if p.IsApp {
			r.AppTicks += p.Ticks
			r.AppPercent += p.Percent
		}
	}
}

// scenarioResult is the frame window.
type scenarioResult struct {
	Ran     bool
	Window  time.Duration
	Frames  []perfstats.Frames
	Summary perfstats.FrameSummary
}

// procMem is the memory of one Wine process, in kilobytes.
type procMem struct {
	PID                 int
	Name                string
	IsApp               bool
	RSSKB, HWMKB, PSSKB int
	HasPSS              bool
}

// memSample is the memory of the Wine processes at one moment.
type memSample struct {
	Label string
	Procs []procMem
}

// totals are the app's own memory and the sum over all the prefix's processes. RSS counts a shared page
// once for each process that maps it, PSS shares it out: the PSS sum is the memory the prefix takes.
func (s memSample) totals() (app procMem, wine procMem) {
	for _, p := range s.Procs {
		wine.RSSKB += p.RSSKB
		wine.PSSKB += p.PSSKB
		wine.HWMKB += p.HWMKB
		wine.HasPSS = wine.HasPSS || p.HasPSS
		if p.IsApp {
			app = p
		}
	}
	return app, wine
}

// cacheOrder lists the states in the order of the report.
var cacheOrder = []cacheState{cacheEmpty, cacheWarm}

// document turns the report into the tables that both formats print.
func (r *perfReport) document() perfstats.Doc {
	build := "debug"
	if r.Release {
		build = "release (-tags glutter_release, stripped)"
	}
	doc := perfstats.Doc{Head: [][2]string{{"app", r.Package}, {"build", build},
		{"where", "Wine on Linux, Direct3D 12 through vkd3d on the host's Vulkan (lavapipe): not a GPU on Windows"}}}
	if r.Renderer != "" {
		doc.Head = append(doc.Head, [2]string{"renderer", r.Renderer})
	}
	if len(r.Env) > 0 {
		doc.Head = append(doc.Head, [2]string{"environment", strings.Join(r.Env, " ")})
	}

	sizes := perfstats.Table{Title: ".exe size", Header: []string{"architecture", "bytes", "MiB"}}
	for _, s := range r.Sizes {
		sizes.Rows = append(sizes.Rows, []string{s.Arch, fmt.Sprint(s.Bytes), perfstats.BytesToMB(s.Bytes)})
	}
	doc.Tables = append(doc.Tables, sizes)

	if len(r.Starts) > 0 {
		doc.Tables = append(doc.Tables, r.startTable())
	}
	if r.Idle.Window > 0 {
		doc.Tables = append(doc.Tables, r.idleTables()...)
	}
	if r.Scenario.Ran {
		doc.Tables = append(doc.Tables, r.frameTables()...)
	}
	if len(r.Memory) > 0 {
		doc.Tables = append(doc.Tables, r.memoryTables()...)
	}
	return doc
}

// startTable has a row for each cold start and the minimum, the median and the maximum of each state
// of the shader cache.
func (r *perfReport) startTable() perfstats.Table {
	t := perfstats.Table{Title: "Cold start (the engine's \"first frame presented since_run\", and the stages of GLUTTER_FRAME_STATS=1)",
		Header: []string{"shader cache", "start", "first frame ms", "device ms", "shaders ms", "shaders built / from cache", "rest of the first frame ms", "whole run ms (Wine included)"}}
	for _, state := range cacheOrder {
		var frame, wall []time.Duration
		n := 0
		for _, s := range r.Starts {
			if s.Cache != state {
				continue
			}
			n++
			cells := "n/a"
			if s.Stages.HasShaders {
				cells = fmt.Sprintf("%d / %d", s.Stages.Built, s.Stages.FromCache)
			}
			t.Rows = append(t.Rows, []string{string(state), fmt.Sprint(n), perfstats.Ms(s.Frame.SinceRun), perfstats.Ms(s.Stages.Device),
				perfstats.Ms(s.Stages.Shaders), cells, perfstats.Ms(s.Stages.Rest), perfstats.Ms(s.Wall)})
			frame, wall = append(frame, s.Frame.SinceRun), append(wall, s.Wall)
		}
		if n == 0 {
			continue
		}
		fs, ws := perfstats.SummarizeDurations(frame), perfstats.SummarizeDurations(wall)
		for _, row := range []struct {
			name string
			get  func(perfstats.DurStats) time.Duration
		}{
			{"min", func(d perfstats.DurStats) time.Duration { return d.Min }},
			{"median", func(d perfstats.DurStats) time.Duration { return d.Median }},
			{"max", func(d perfstats.DurStats) time.Duration { return d.Max }},
		} {
			t.Rows = append(t.Rows, []string{string(state), row.name, perfstats.Ms(row.get(fs)), "", "", "", "", perfstats.Ms(row.get(ws))})
		}
	}
	return t
}

// idleTables are the totals of the idle window and the CPU time of each process.
func (r *perfReport) idleTables() []perfstats.Table {
	sum := perfstats.Table{Title: "Idle CPU (no input; utime+stime of the Wine processes over the window)",
		Header: []string{"window s", "app ticks", "app CPU % of one core", "all Wine processes ticks", "all Wine processes CPU %", "frames logged in the window"},
		Rows: [][]string{{fmt.Sprintf("%.1f", r.Idle.Window.Seconds()), fmt.Sprint(r.Idle.AppTicks), fmt.Sprintf("%.2f", r.Idle.AppPercent),
			fmt.Sprint(r.Idle.WineTicks), fmt.Sprintf("%.2f", r.Idle.WinePercent), fmt.Sprint(r.Idle.Frames)}}}
	per := perfstats.Table{Title: "Idle CPU of each Wine process", Header: []string{"pid", "process", "ticks", "CPU % of one core"}}
	for _, p := range r.Idle.Procs {
		name := p.Name
		if p.IsApp {
			name += " (the app)"
		}
		per.Rows = append(per.Rows, []string{fmt.Sprint(p.PID), name, fmt.Sprint(p.Ticks), fmt.Sprintf("%.2f", p.Percent)})
	}
	return []perfstats.Table{sum, per}
}

// frameTables are the summary of the frame window and the statistics of every second.
func (r *perfReport) frameTables() []perfstats.Table {
	s := r.Scenario.Summary
	sum := perfstats.Table{Title: fmt.Sprintf("Frame times (GLUTTER_FRAME_STATS=1, %.0f s after the first frame)", r.Scenario.Window.Seconds()),
		Header: []string{"metric", "value"}}
	if s.Seconds == 0 {
		sum.Rows = [][]string{{"frames", "none logged: the app drew no frame in the window"}}
		return []perfstats.Table{sum}
	}
	sum.Rows = [][]string{
		{"seconds with frames", fmt.Sprint(s.Seconds)},
		{"frames", fmt.Sprint(s.Frames)},
		{"FPS mean (min, max)", fmt.Sprintf("%.1f (%d, %d)", s.MeanFPS, s.MinFPS, s.MaxFPS)},
		{"build ms avg / worst", perfstats.Ms2(s.BuildAvg) + " / " + perfstats.Ms2(s.BuildMax)},
		{"raster ms avg / worst", perfstats.Ms2(s.RasterAvg) + " / " + perfstats.Ms2(s.RasterMax)},
		{"worst second, build+raster avg ms", perfstats.Ms2(s.WorstSecond)},
		{"frames over 16 ms", fmt.Sprintf("%d (%.1f%%)", s.Over16, s.Over16Percent)},
		{"Go heap MB, max / last", fmt.Sprintf("%d / %d", s.HeapMaxMB, s.HeapLastMB)},
		{"Go sys MB, last", fmt.Sprint(s.SysLastMB)},
	}
	per := perfstats.Table{Title: "Frame times per second", Header: []string{"second", "FPS", "build avg ms", "build max ms", "raster avg ms", "raster max ms", "over 16 ms", "Go heap MB"}}
	for i, f := range r.Scenario.Frames {
		per.Rows = append(per.Rows, []string{fmt.Sprint(i + 1), fmt.Sprint(f.FPS), perfstats.Ms2(f.BuildAvg), perfstats.Ms2(f.BuildMax),
			perfstats.Ms2(f.RasterAvg), perfstats.Ms2(f.RasterMax), fmt.Sprint(f.Over16), fmt.Sprint(f.GoHeapMB)})
	}
	return []perfstats.Table{sum, per}
}

// memoryTables are the memory of the app and of the whole prefix at each moment, and of each process
// at the last moment.
func (r *perfReport) memoryTables() []perfstats.Table {
	mem := perfstats.Table{Title: "Memory (MB; PSS shares a page between the processes that map it)",
		Header: []string{"when", "app VmRSS", "app VmHWM", "app PSS", "all Wine processes PSS", "all Wine processes VmRSS"}}
	for _, s := range r.Memory {
		app, wine := s.totals()
		pss := "n/a"
		if app.HasPSS {
			pss = perfstats.KBToMB(app.PSSKB)
		}
		winePSS := "n/a"
		if wine.HasPSS {
			winePSS = perfstats.KBToMB(wine.PSSKB)
		}
		mem.Rows = append(mem.Rows, []string{s.Label, perfstats.KBToMB(app.RSSKB), perfstats.KBToMB(app.HWMKB), pss, winePSS, perfstats.KBToMB(wine.RSSKB)})
	}
	last := r.Memory[len(r.Memory)-1]
	per := perfstats.Table{Title: "Memory of each Wine process " + strings.ToLower(last.Label) + " (MB)", Header: []string{"pid", "process", "VmRSS", "VmHWM", "PSS"}}
	procs := append([]procMem(nil), last.Procs...)
	sort.Slice(procs, func(i, j int) bool { return procs[i].PSSKB > procs[j].PSSKB })
	for _, p := range procs {
		name := p.Name
		if p.IsApp {
			name += " (the app)"
		}
		pss := "n/a"
		if p.HasPSS {
			pss = perfstats.KBToMB(p.PSSKB)
		}
		per.Rows = append(per.Rows, []string{fmt.Sprint(p.PID), name, perfstats.KBToMB(p.RSSKB), perfstats.KBToMB(p.HWMKB), pss})
	}
	return []perfstats.Table{mem, per}
}

// render writes the report as "text" (aligned columns) or "markdown" (for docs/PERFORMANCE.md). It
// fails with PerfWriteFailed (internal/perfstats) when w does.
func (r *perfReport) render(w io.Writer, format string) error {
	return r.document().Render(w, "glutter-windows perf", format)
}
