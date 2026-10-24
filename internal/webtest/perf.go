package webtest

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/perfstats"
	"github.com/rm4n0s/glutter/internal/platform/web"
)

// PerfOptions are the parameters of Perf.
//
// Flutter: no counterpart
type PerfOptions struct {
	// Dir is the build to measure (the directory webtool.Build wrote).
	Dir string
	// Browser is the browser to open it in.
	Browser Browser
	// BrowserPath is the program of the browser; found when empty.
	BrowserPath string
	// Query is added to the URL's query. GLUTTER_FRAME_STATS=1 is always added.
	Query url.Values
	// Width and Height are the size of the viewport the app sees in CSS pixels; 1536x864 (a
	// 1920x1080 display at a ratio of 1.25, SPEC §15) when zero.
	Width, Height int
	// ScaleFactor is the device pixel ratio of the browser; 1.25 when zero.
	ScaleFactor float64
	// Loads is how many times the page is loaded to time the start: the first with the browser's
	// cache empty, the others with app.wasm in it. Four when zero.
	Loads int
	// Settle is how long the app runs after the last load before the measurement starts; three
	// seconds when zero.
	Settle time.Duration
	// Measure is how long the frames, the CPU time and the calls of requestAnimationFrame are
	// measured; ten seconds when zero.
	Measure time.Duration
	// Clicks, when set, make the run a measurement of the response to clicks instead of the steady
	// state: after the start, a pointer move, a press and a release (as the mouse does) are
	// dispatched at each position (CSS pixels from the top left of the view), ClickGap apart, and
	// the frames that follow each are timed (PerfResult.Clicks).
	Clicks [][2]float64
	// ClickGap is the time between two clicks; a second and a half when zero.
	ClickGap time.Duration
	// Blank measures a page that has no app at all, for the CPU time that the browser itself
	// takes (the baseline that the idle CPU of an app is read against): there are no loads and no
	// frames.
	Blank bool
	// Timeout is how long the whole run may take; three minutes when zero.
	Timeout time.Duration
	// Output receives the browser's standard output and error.
	Output io.Writer
}

// PerfFrames is one "frames" line of the engine's statistics (GLUTTER_FRAME_STATS=1): the frames
// of one second.
//
// Flutter: no counterpart
type PerfFrames = perfstats.Frames

// PerfResult is what Perf measured.
//
// Flutter: no counterpart
type PerfResult struct {
	// Renderer is the renderer and device that the first frame says.
	Renderer string
	// FirstFrames are the times from the start of the page to the first frame of the app, one per
	// load: the first with an empty cache. SinceRun are the times from the start of Run to the
	// first frame, as the engine logs them.
	FirstFrames, SinceRun []time.Duration
	// Frames are the statistics lines of the measured window.
	Frames []PerfFrames
	// RAFCalls is the number of calls of requestAnimationFrame in the window, and TimerCalls the
	// number of calls of setTimeout and setInterval (a timer armed: the Go runtime wakes with
	// setTimeout), or -1 when the page could not tell.
	RAFCalls, TimerCalls int
	// CPU is the CPU time of the browser (every process of its group, and of its X server) in
	// the window, which lasted Window.
	CPU, Window time.Duration
	// Intervals are the times between the starts of two callbacks of requestAnimationFrame in the
	// window, which is the cadence of the frames the browser asked for.
	Intervals []time.Duration
	// Slow are the frames over a 60th of a second that the engine logged on their own, in order.
	Slow []PerfSlow
	// Clicks are the responses to the clicks of PerfOptions.Clicks, in order.
	Clicks []PerfClick
}

// PerfSlow is a "slow frame" line of the engine.
//
// Flutter: no counterpart
type PerfSlow struct{ Build, Raster time.Duration }

// PerfClick is what followed one click.
//
// Flutter: no counterpart
type PerfClick struct {
	X, Y float64
	// Latency is the time from the release of the mouse button to the end of the first frame that
	// began after it: the frame's callback ends when the Go frame (build, layout, paint and the
	// renderer's calls) has run; the GPU's work after it is not counted.
	Latency time.Duration
	// Frames are the durations of the callbacks of the first frames after the release (at most
	// five), Max the longest callback and Gap the longest time between the ends of two frames, in
	// the second after the release.
	Frames   []time.Duration
	Max, Gap time.Duration
}

// Perf opens the build in a headless browser, times its start in several loads, lets it settle,
// and then measures for a while: the engine's frame statistics, the calls of
// requestAnimationFrame and the CPU time of the browser. The app is run with
// GLUTTER_FRAME_STATS=1. It starts the server and the browser and stops both before it returns;
// it fails with WebAppFailed when the program ends with an error, BrowserExited when the browser
// ends first, BrowserTimeout when the run takes too long, and BrowserCPUUnavailable where the CPU
// time of the browser cannot be read.
//
// Flutter: no counterpart (flutter run --profile, the timeline)
func Perf(ctx context.Context, o PerfOptions) (PerfResult, error) {
	width, height := o.Width, o.Height
	if width == 0 || height == 0 {
		width, height = 1536, 864
	}
	scale := o.ScaleFactor
	if scale == 0 {
		scale = 1.25
	}
	loads := o.Loads
	if loads == 0 {
		loads = 4
	}
	settle, measure := o.Settle, o.Measure
	if settle == 0 {
		settle = 3 * time.Second
	}
	if measure == 0 {
		measure = 10 * time.Second
	}
	timeout := o.Timeout
	if timeout == 0 {
		timeout = 3 * time.Minute
	}
	clickGap := o.ClickGap
	if clickGap == 0 {
		clickGap = 1500 * time.Millisecond
	}
	srv := NewCachingServer(o.Dir)
	defer srv.Close()
	var page []byte
	if o.Blank {
		// The blank page makes the same posts as the frame page of an app does, so that the CPU time of
		// the baseline has the cost of the measurement in it too.
		page = []byte("<!doctype html><title>blank</title><body style=\"margin:0\"><script>setInterval(function(){" +
			"fetch(\"" + web.ResultsPath + "\",{method:\"POST\",headers:{\"Content-Type\":\"application/json\"}," +
			"body:JSON.stringify({kind:\"raf\",data:{count:-1,timers:-1}})}).catch(function(){});},2000);</script></body>")
	} else {
		index, err := os.ReadFile(filepath.Join(o.Dir, "index.html"))
		if err != nil {
			return PerfResult{}, errors.NewErr(tagWebPerfFailed, err).SetMetadata("what", "reading index.html of the build")
		}
		srv.AddPage(PerfAppPath, InjectRAFCounter(index))
		q := url.Values{}
		for k, v := range o.Query {
			q[k] = v
		}
		q.Set("GLUTTER_FRAME_STATS", "1")
		q.Set("glutter_report", web.ResultsPath)
		q.Set("glutter_log_every", "1000")
		page = PerfFramePage(PerfAppPath+"?"+q.Encode(), width, height, loads, o.Clicks, clickGap)
	}
	srv.AddPage(FramePath, page)
	proc, err := Launch(o.Browser, srv.URL+FramePath, LaunchOptions{
		Path: o.BrowserPath, Width: max(width, 500) + 40, Height: height + 200, ScaleFactor: scale,
		Lifetime: timeout + 30*time.Second, Output: o.Output,
	})
	if err != nil {
		return PerfResult{}, err
	}
	defer proc.Stop()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var res PerfResult
	res.RAFCalls, res.TimerCalls = -1, -1
	if !o.Blank {
		if err := awaitStarts(ctx, srv, proc, &res); err != nil {
			return res, err
		}
	}
	if len(o.Clicks) > 0 && !o.Blank {
		err := awaitClicks(ctx, srv, proc, &res)
		return res, err
	}
	// The collector reads what the page posts from here on, so that its queue does not fill while the
	// app settles, but keeps only what arrives inside the window.
	c := &perfCollector{res: &res, raf0: -1, raf1: -1, timer0: -1, timer1: -1}
	collectCtx, stopCollecting := context.WithCancel(ctx)
	defer stopCollecting()
	failed := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.run(collectCtx, srv, proc, failed)
	}()
	// done stops the collector and waits for it: it is the only writer of the result until then.
	done := func(err error) (PerfResult, error) {
		stopCollecting()
		wg.Wait()
		if err == nil && c.raf0 >= 0 {
			res.RAFCalls, res.TimerCalls = c.raf1-c.raf0, c.timer1-c.timer0
		}
		return res, err
	}
	if err := sleepOrEnd(ctx, settle, proc, failed); err != nil {
		return done(err)
	}
	cpu0, err := proc.CPUTime()
	if err != nil {
		return done(err)
	}
	t0 := time.Now()
	c.record(true)
	if err := sleepOrEnd(ctx, measure, proc, failed); err != nil {
		return done(err)
	}
	c.record(false)
	window := time.Since(t0)
	cpu1, err := proc.CPUTime()
	if err != nil {
		return done(err)
	}
	res.Window, res.CPU = window, cpu1-cpu0
	return done(nil)
}

// perfCollector keeps the logs and the counts of requestAnimationFrame that the page posts while
// the measurement window is open.
type perfCollector struct {
	mu         sync.Mutex
	recording  bool
	res        *PerfResult
	raf0, raf1 int
	// timer0 and timer1 are the first and the last count of timers armed in the window.
	timer0, timer1 int
}

func (c *perfCollector) record(on bool) {
	c.mu.Lock()
	c.recording = on
	c.mu.Unlock()
}

// run reads the results until ctx ends; a program that ends or a browser that ends goes to failed.
func (c *perfCollector) run(ctx context.Context, srv *Server, proc *Process, failed chan<- error) {
	for {
		select {
		case r := <-srv.results:
			if r.Kind == "exit" {
				failed <- errors.New(tagWebAppFailed, "the app ended during the measurement", r.exitMetadata()...)
				return
			}
			c.take(r)
		case <-proc.Done():
			failed <- errors.New(tagBrowserExited, "the browser ended during the measurement", proc.exitMetadata()...)
			return
		case <-ctx.Done():
			return
		}
	}
}

// take keeps a result if the window is open.
func (c *perfCollector) take(r Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.recording {
		return
	}
	switch r.Kind {
	case "log":
		c.res.addLog(r.Log)
	case "raf":
		if d, ok := rafData(r); ok {
			if c.raf0 < 0 {
				c.raf0, c.timer0 = d.Count, d.Timers
			}
			c.raf1, c.timer1 = d.Count, d.Timers
			for _, v := range d.Intervals {
				c.res.Intervals = append(c.res.Intervals, time.Duration(v*float64(time.Millisecond)))
			}
		}
	}
}

// PerfAppPath is where Perf serves the app's page with the counter of requestAnimationFrame.
//
// Flutter: no counterpart
const PerfAppPath = "/__glutter/perf-app.html"

// InjectRAFCounter returns the page with a script that counts the calls of requestAnimationFrame,
// before any script of the page runs, and offers the count as window.glutterRafCount(); the calls of
// setTimeout and setInterval are counted too (window.glutterTimerCount()), and the times between the starts of
// two callbacks of requestAnimationFrame are kept until window.glutterIntervals() takes them. The page is
// served from another path than the build's, so it also gets a base address that keeps its
// relative files (wasm_exec.js, app.wasm) where they are.
//
// Flutter: no counterpart
func InjectRAFCounter(index []byte) []byte {
	const script = "<base href=\"/\"><script>(function(){var n=0,t=0,o=window.requestAnimationFrame.bind(window)," +
		"st=window.setTimeout.bind(window),si=window.setInterval.bind(window);" +
		"var fr=window.glutterFrames=[],iv=[],last=0;" +
		"window.requestAnimationFrame=function(f){n++;return o(function(ts){var a=performance.now();" +
		"if(last&&iv.length<10000)iv.push(a-last);last=a;" +
		"try{return f(ts)}finally{if(fr.length<100000)fr.push([a,performance.now()])}})};" +
		"window.setTimeout=function(){t++;return st.apply(null,arguments)};" +
		"window.setInterval=function(){t++;return si.apply(null,arguments)};" +
		"window.glutterRafCount=function(){return n};window.glutterTimerCount=function(){return t};" +
		"window.glutterIntervals=function(){var r=iv;iv=[];return r}})();</script>"
	s := string(index)
	if i := strings.Index(s, "<head>"); i >= 0 {
		i += len("<head>")
		return []byte(s[:i] + script + s[i:])
	}
	return []byte(script + s)
}

// awaitClicks waits for the page to post the responses to its clicks, and reads the frame statistics
// that arrive meanwhile.
func awaitClicks(ctx context.Context, srv *Server, proc *Process, res *PerfResult) error {
	for {
		select {
		case r := <-srv.results:
			switch r.Kind {
			case "log":
				res.addLog(r.Log)
			case "clicks":
				var d []struct {
					X, Y    float64
					Latency float64   `json:"latency"`
					Frames  []float64 `json:"frames"`
					Max     float64   `json:"max"`
					Gap     float64   `json:"gap"`
				}
				if err := json.Unmarshal(r.Data, &d); err != nil {
					return errors.NewErr(tagWebPerfFailed, err).SetMetadata("what", "the responses to the clicks")
				}
				dur := func(v float64) time.Duration { return time.Duration(v * float64(time.Millisecond)) }
				for _, c := range d {
					pc := PerfClick{X: c.X, Y: c.Y, Latency: dur(c.Latency), Max: dur(c.Max), Gap: dur(c.Gap)}
					for _, f := range c.Frames {
						pc.Frames = append(pc.Frames, dur(f))
					}
					res.Clicks = append(res.Clicks, pc)
				}
				return nil
			case "exit":
				return errors.New(tagWebAppFailed, "the app ended during the measurement", r.exitMetadata()...)
			}
		case <-proc.Done():
			return errors.New(tagBrowserExited, "the browser ended during the measurement", proc.exitMetadata()...)
		case <-ctx.Done():
			return errors.NewErr(tagBrowserTimeout, ctx.Err()).SetMetadata("waiting for", "the responses to the clicks")
		}
	}
}

// PerfFramePage makes the page that runs the app (src, a URL on the same server) in a frame of
// width x height CSS pixels, loads it loads times one after the other (the frame's address
// changes, so that each load is a new page), reads when each one drew its first frame from
// window.glutterTimings, which the loader sets, and posts {kind: "starts", data: [...]} when all
// have; then, when there are clicks, it dispatches them at the canvas of the app, gap apart, and posts
// {kind: "clicks", data: [...]} with the frames that followed each (window.glutterFrames, which the counter
// of InjectRAFCounter keeps: the start and end of each requestAnimationFrame callback); else it posts {kind: "raf", data: {count, timers}} every two seconds.
//
// Flutter: no counterpart
func PerfFramePage(src string, width, height, loads int, clicks [][2]float64, gap time.Duration) []byte {
	srcJSON, _ := json.Marshal(src) // a string always marshals
	safe := strings.NewReplacer("<", "\\u003c", ">", "\\u003e", "&", "\\u0026").Replace(string(srcJSON))
	const script = `
var f=document.getElementById("app"),src=%SRC%,loads=%LOADS%,clicks=%CLICKS%,gap=%GAP%,n=0,starts=[];
function runClicks(){
  var w=f.contentWindow,cv=w.document.querySelector("canvas[data-glutter-view]"),r=cv.getBoundingClientRect(),i=0,out=[];
  function ev(type,c,buttons,pressure){return new w.PointerEvent(type,{bubbles:true,cancelable:true,composed:true,clientX:r.left+c[0],clientY:r.top+c[1],
    pointerType:"mouse",pointerId:1,isPrimary:true,button:0,buttons:buttons,pressure:pressure});}
  function next(){
    if(i>=clicks.length){post("clicks",out);return;}
    var c=clicks[i++];
    cv.dispatchEvent(ev("pointermove",c,0,0));
    setTimeout(function(){cv.dispatchEvent(ev("pointerdown",c,1,0.5));
      setTimeout(function(){
        var up=w.performance.now();cv.dispatchEvent(ev("pointerup",c,0,0));
        setTimeout(function(){
          var fr=w.glutterFrames.filter(function(x){return x[0]>=up&&x[0]<up+1000;}),res={x:c[0],y:c[1],latency:-1,frames:[],max:0,gap:0};
          if(fr.length){res.latency=fr[0][1]-up;res.frames=fr.slice(0,5).map(function(x){return x[1]-x[0];});
            res.max=Math.max.apply(null,fr.map(function(x){return x[1]-x[0];}));
            for(var k=1;k<fr.length;k++)res.gap=Math.max(res.gap,fr[k][1]-fr[k-1][1]);}
          out.push(res);next();
        },gap);
      },50);
    },100);
  }
  next();
}
function post(kind,data){fetch("%RESULTS%",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({kind:kind,data:data})}).catch(function(){});}
function start(){f.src=src+"&load="+n;var t=setInterval(function(){
  var w=f.contentWindow,tm;try{tm=w&&w.glutterTimings;}catch(e){}
  if(!tm||!tm.firstFrame)return;
  clearInterval(t);starts.push({instantiated:tm.instantiated,firstFrame:tm.firstFrame});n++;
  if(n<loads){setTimeout(start,500);return;}
  post("starts",starts);
  if(clicks.length){setTimeout(runClicks,1500);return;}
  setInterval(function(){var c=-1,t=-1,v=[];try{c=f.contentWindow.glutterRafCount();t=f.contentWindow.glutterTimerCount();v=f.contentWindow.glutterIntervals();}catch(e){}post("raf",{count:c,timers:t,intervals:v});},2000);
},20);}
start();
`
	clicksJSON, _ := json.Marshal(clicks) // numbers always marshal
	if clicks == nil {
		clicksJSON = []byte("[]")
	}
	body := strings.NewReplacer("%SRC%", safe, "%LOADS%", strconv.Itoa(loads), "%CLICKS%", string(clicksJSON),
		"%GAP%", strconv.Itoa(int(gap.Milliseconds())), "%RESULTS%", web.ResultsPath).Replace(script)
	return []byte("<!doctype html>\n<html><head><meta charset=\"utf-8\"><title>glutter perf</title>\n" +
		"<style>html,body{margin:0;background:#808080}iframe{border:0;display:block;width:" + strconv.Itoa(width) + "px;height:" +
		strconv.Itoa(height) + "px}</style></head>\n<body><iframe id=\"app\"></iframe>\n<script>" + body + "</script></body></html>\n")
}

// startsData is the data of the "starts" result.
type startsData []struct {
	Instantiated float64 `json:"instantiated"`
	FirstFrame   float64 `json:"firstFrame"`
}

// awaitStarts waits for the "starts" result, taking the logs that arrive meanwhile for the
// renderer's name and the engine's own times.
func awaitStarts(ctx context.Context, srv *Server, proc *Process, res *PerfResult) error {
	for {
		select {
		case r := <-srv.results:
			switch r.Kind {
			case "log":
				res.noteStart(r.Log)
			case "starts":
				var d startsData
				if err := json.Unmarshal(r.Data, &d); err != nil {
					return errors.NewErr(tagWebPerfFailed, err).SetMetadata("what", "the times of the starts")
				}
				for _, s := range d {
					res.FirstFrames = append(res.FirstFrames, time.Duration(s.FirstFrame*float64(time.Millisecond)))
				}
				// The loader posts the output once a second, so the line of the last load's first frame
				// may still be on its way: take the logs until there is one for each load.
				late := time.After(2500 * time.Millisecond)
				for len(res.SinceRun) < len(res.FirstFrames) {
					select {
					case r := <-srv.results:
						if r.Kind == "log" {
							res.noteStart(r.Log)
						}
					case <-late:
						return nil
					case <-proc.Done():
						return nil
					case <-ctx.Done():
						return nil
					}
				}
				return nil
			case "exit":
				return errors.New(tagWebAppFailed, "the app ended during the measurement", r.exitMetadata()...)
			}
		case <-proc.Done():
			return errors.New(tagBrowserExited, "the browser ended during the measurement", proc.exitMetadata()...)
		case <-ctx.Done():
			return errors.NewErr(tagBrowserTimeout, ctx.Err()).SetMetadata("waiting for", "the app's first frames")
		}
	}
}

// sleepOrEnd waits for d, and fails if the program or the browser ends (failed) or ctx does first.
func sleepOrEnd(ctx context.Context, d time.Duration, proc *Process, failed <-chan error) error {
	select {
	case <-time.After(d):
		return nil
	case err := <-failed:
		return err
	case <-proc.Done():
		return errors.New(tagBrowserExited, "the browser ended during the measurement", proc.exitMetadata()...)
	case <-ctx.Done():
		return errors.NewErr(tagBrowserTimeout, ctx.Err()).SetMetadata("waiting for", "the app to settle")
	}
}

// exitMetadata is the metadata of a WebAppFailed error: how the program ended.
func (r Result) exitMetadata() []any {
	return []any{"exit code", strconv.Itoa(r.ExitCode), "message", r.Message, "shown", r.Shown, "log", strings.TrimSpace(r.Log)}
}

// exitMetadata is the metadata of a BrowserExited error: why the browser ended, when it is known.
func (p *Process) exitMetadata() []any {
	if p.Err != nil {
		return []any{"error", p.Err.Error()}
	}
	return nil
}

// rafData reads a "raf" result.
func rafData(r Result) (d struct {
	Count     int       `json:"count"`
	Timers    int       `json:"timers"`
	Intervals []float64 `json:"intervals"`
}, ok bool) {
	if err := json.Unmarshal(r.Data, &d); err != nil || d.Count < 0 {
		return d, false
	}
	return d, true
}

// noteStart reads the "first frame presented" lines of the logs that came while the page loaded:
// the renderer and the time since Run began, for each load.
func (r *PerfResult) noteStart(log string) {
	for _, line := range strings.Split(log, "\n") {
		if !strings.Contains(line, "first frame presented") {
			continue
		}
		kv := ParseLogLine(line)
		if v := kv["renderer"]; v != "" {
			r.Renderer = v
		}
		if d, err := time.ParseDuration(kv["since_run"]); err == nil {
			r.SinceRun = append(r.SinceRun, d)
		}
	}
}

// addLog reads the "frames" lines of a log into the statistics.
func (x *PerfResult) addLog(log string) {
	for _, line := range strings.Split(log, "\n") {
		if strings.Contains(line, " slow frame ") {
			kv := ParseLogLine(line)
			b, errB := time.ParseDuration(kv["build"])
			r, errR := time.ParseDuration(kv["raster"])
			if errB == nil && errR == nil {
				x.Slow = append(x.Slow, PerfSlow{Build: b, Raster: r})
			}
			continue
		}
		if !strings.Contains(line, " frames ") {
			continue
		}
		if f, ok := ParseFramesLine(line); ok {
			x.Frames = append(x.Frames, f)
		}
	}
}

// ParseLogLine splits a log line of the engine (slog's text format, key=value with values in
// quotes when they hold blanks) into its pairs.
//
// Flutter: no counterpart
func ParseLogLine(line string) map[string]string { return perfstats.ParseLogLine(line) }

// ParseFramesLine reads a "frames" line of the engine's statistics; ok is false for a line that
// is not one or has a value that is not a number or a duration.
//
// Flutter: no counterpart
func ParseFramesLine(line string) (PerfFrames, bool) { return perfstats.ParseFramesLine(line) }

// Median is the middle value of the durations (the upper one for an even count), or zero for none.
//
// Flutter: no counterpart
func Median(d []time.Duration) time.Duration { return perfstats.Median(d) }

// Summary is the result as lines of "key value", for scripts (scripts/perf-web.sh reads them).
// Times are in milliseconds.
//
// Flutter: no counterpart
func (r PerfResult) Summary() string {
	var b strings.Builder
	line := func(key string, v any) { b.WriteString(key + " " + toText(v) + "\n") }
	if r.Renderer != "" {
		line("renderer", r.Renderer)
	}
	if len(r.FirstFrames) > 0 {
		line("first_frame_cold_ms", ms(r.FirstFrames[0]))
		if len(r.FirstFrames) > 1 {
			line("first_frame_warm_ms", ms(Median(r.FirstFrames[1:])))
			line("first_frame_warm_loads", len(r.FirstFrames)-1)
		}
	}
	if len(r.SinceRun) > 1 {
		line("since_run_warm_ms", ms(Median(r.SinceRun[1:])))
	}
	for i, c := range r.Clicks {
		var fr []string
		for _, f := range c.Frames {
			fr = append(fr, strconv.FormatFloat(ms(f), 'f', 1, 64))
		}
		b.WriteString("click " + strconv.Itoa(i) + " " + strconv.FormatFloat(c.X, 'f', 0, 64) + "," + strconv.FormatFloat(c.Y, 'f', 0, 64) +
			" latency_ms " + toText(ms(c.Latency)) + " max_frame_ms " + toText(ms(c.Max)) + " max_gap_ms " + toText(ms(c.Gap)) +
			" frames_ms " + strings.Join(fr, ",") + "\n")
	}
	for _, f := range r.Slow {
		b.WriteString("slow build_ms " + toText(ms(f.Build)) + " raster_ms " + toText(ms(f.Raster)) + "\n")
	}
	line("window_s", ms(r.Window)/1000)
	line("cpu_ms", ms(r.CPU))
	if r.Window > 0 {
		line("cpu_percent", 100*float64(r.CPU)/float64(r.Window))
	}
	line("raf_calls", r.RAFCalls)
	line("timer_calls", r.TimerCalls)
	if n := len(r.Intervals); n > 0 {
		sorted := append([]time.Duration(nil), r.Intervals...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		var sum time.Duration
		over60, over30 := 0, 0
		for _, d := range sorted {
			sum += d
			if d > time.Second/60 {
				over60++
			}
			if d > time.Second/30 {
				over30++
			}
		}
		line("raf_intervals", n)
		line("raf_interval_mean_ms", ms(sum)/float64(n))
		line("raf_interval_median_ms", ms(sorted[n/2]))
		line("raf_interval_p99_ms", ms(sorted[min(n-1, n*99/100)]))
		line("raf_interval_max_ms", ms(sorted[n-1]))
		line("raf_intervals_over_16.7ms", over60)
		line("raf_intervals_over_33.3ms", over30)
	}
	line("frame_lines", len(r.Frames))
	if n := len(r.Frames); n > 0 {
		var fps, over, heap, sys int
		var buildAvg, rasterAvg, buildMax, rasterMax, worstSum time.Duration
		for _, f := range r.Frames {
			fps += f.FPS
			over += f.Over16
			buildAvg += f.BuildAvg
			rasterAvg += f.RasterAvg
			buildMax, rasterMax = max(buildMax, f.BuildMax), max(rasterMax, f.RasterMax)
			worstSum = max(worstSum, f.BuildAvg+f.RasterAvg)
			heap, sys = f.GoHeapMB, f.GoSysMB
		}
		line("fps", float64(fps)/float64(n))
		line("over_16ms", over)
		line("build_avg_ms", ms(buildAvg)/float64(n))
		line("raster_avg_ms", ms(rasterAvg)/float64(n))
		line("ui_avg_ms", (ms(buildAvg)+ms(rasterAvg))/float64(n))
		line("ui_worst_second_avg_ms", ms(worstSum))
		line("build_max_ms", ms(buildMax))
		line("raster_max_ms", ms(rasterMax))
		line("go_heap_mb", heap)
		line("go_sys_mb", sys)
	}
	return b.String()
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func toText(v any) string {
	switch v := v.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', 2, 64)
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	}
	return ""
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWebPerfFailed = "WebPerfFailed"
)
