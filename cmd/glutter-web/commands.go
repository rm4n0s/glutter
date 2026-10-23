package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// buildFlags are the flags the commands share.
type buildFlags struct {
	release bool
	host    string
}

func (b *buildFlags) add(fs *flag.FlagSet) {
	fs.BoolVar(&b.release, "release", false, "build with -tags glutter_release and -ldflags=-s -w")
	fs.StringVar(&b.host, "host", "", "CSS selector of the element the view fills (default: the whole page)")
}

// parse parses the flags and returns the one package that is left.
func parse(fs *flag.FlagSet, args []string) (string, error) {
	if err := fs.Parse(args); err != nil {
		return "", errors.NewErr(tagWebUsage, err)
	}
	if fs.NArg() != 1 {
		return "", errors.New(tagWebUsage, "one package is needed, such as ./examples/counter\n"+usage)
	}
	return fs.Arg(0), nil
}

// runBuild is `glutter-web build`.
func runBuild(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("o", "web-build", "the directory to write the build to")
	var bf buildFlags
	bf.add(fs)
	pkg, err := parse(fs, args)
	if err != nil {
		return err
	}
	res, err := webtool.Build(ctx, webtool.BuildOptions{Package: pkg, OutDir: *out, Release: bf.release, HostElement: bf.host})
	if err != nil {
		return err
	}
	return say(stdout, "built %s into %s: app.wasm %s, app.wasm.gz %s\n", pkg, res.Dir, size(res.WasmSize), size(res.GzipSize))
}

// runServe is `glutter-web serve`.
func runServe(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", ":8080", "the address to listen on")
	var bf buildFlags
	bf.add(fs)
	pkg, err := parse(fs, args)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "glutter-web-")
	if err != nil {
		return errors.NewErr(tagWebServeFailed, err)
	}
	defer func() { _ = os.RemoveAll(dir) }() // a temporary directory: a leftover is harmless
	if _, err := webtool.Build(ctx, webtool.BuildOptions{Package: pkg, OutDir: dir, Release: bf.release, HostElement: bf.host}); err != nil {
		return err
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return errors.NewErr(tagWebServeFailed, err).SetMetadata("addr", *addr)
	}
	srv := &http.Server{Handler: webtool.Handler(dir), ReadHeaderTimeout: 10 * time.Second}
	if err := say(stdout, "serving %s at http://%s/ (Ctrl-C stops it)\n", pkg, displayAddr(ln.Addr())); err != nil {
		_ = ln.Close()
		return err
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			return errors.NewErr(tagWebServeFailed, err)
		}
		return nil
	case err := <-errc:
		return errors.NewErr(tagWebServeFailed, err)
	}
}

// displayAddr is the address of the listener as a browser can open it: a listener on every
// address is reached through localhost.
func displayAddr(a net.Addr) string {
	host, port, err := net.SplitHostPort(a.String())
	if err != nil {
		return a.String()
	}
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "localhost"
	}
	return net.JoinHostPort(host, port)
}

// queryFlag collects the -query flags: each is a query string.
type queryFlag url.Values

func (q *queryFlag) String() string { return url.Values(*q).Encode() }

func (q *queryFlag) Set(s string) error {
	v, err := url.ParseQuery(strings.TrimPrefix(s, "?"))
	if err != nil {
		return errors.NewErr(tagWebUsage, err).SetMetadata("query", s)
	}
	if *q == nil {
		*q = queryFlag{}
	}
	for k, vs := range v {
		(*q)[k] = append((*q)[k], vs...)
	}
	return nil
}

// runShot is `glutter-web shot`.
func runShot(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("shot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	browserName := fs.String("browser", "", "chromium or firefox")
	renderer := fs.String("renderer", "", "the renderer: GLUTTER_RENDERER (auto, webgpu, webgl or soft)")
	out := fs.String("o", "", "the PNG file to write")
	width := fs.Int("width", 480, "the width of the browser window in CSS pixels")
	height := fs.Int("height", 640, "the height of the browser window in CSS pixels")
	timeout := fs.Duration("timeout", time.Minute, "how long to wait for the screenshot")
	events := fs.String("events", "", "a script of synthetic input events that a debug build replays after its first frame (GLUTTER_DEV_EVENTS), such as \"wait:500ms;wheel:240,320,0,300\"")
	after := fs.Duration("after", 0, "how long after the first idle frame to take the screenshot (GLUTTER_SCREENSHOT_AFTER), so that the events have had their effect")
	var query queryFlag
	fs.Var(&query, "query", "a query string added to the URL, such as GLUTTER_DEV_TAPS=3 (may be repeated)")
	var bf buildFlags
	bf.add(fs)
	pkg, err := parse(fs, args)
	if err != nil {
		return err
	}
	if *browserName == "" || *out == "" {
		return errors.New(tagWebUsage, "-browser and -o are needed\n"+usage)
	}
	browser, err := webtest.ParseBrowser(*browserName)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "glutter-web-")
	if err != nil {
		return errors.NewErr(tagWebServeFailed, err)
	}
	defer func() { _ = os.RemoveAll(dir) }() // a temporary directory: a leftover is harmless
	if _, err := webtool.Build(ctx, webtool.BuildOptions{Package: pkg, OutDir: dir, Release: bf.release, HostElement: bf.host}); err != nil {
		return err
	}
	q := url.Values(query)
	if q == nil {
		q = url.Values{}
	}
	if *renderer != "" {
		q.Set("GLUTTER_RENDERER", *renderer)
	}
	if *events != "" {
		q.Set("GLUTTER_DEV_EVENTS", *events)
	}
	if *after > 0 {
		q.Set("GLUTTER_SCREENSHOT_AFTER", after.String())
	}
	png, err := webtest.Shot(ctx, webtest.ShotOptions{Dir: dir, Browser: browser, Query: q, Width: *width, Height: *height, Timeout: *timeout})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return errors.NewErr(tagWebWriteFailed, err).SetMetadata("path", *out)
	}
	if err := os.WriteFile(*out, png, 0o644); err != nil {
		return errors.NewErr(tagWebWriteFailed, err).SetMetadata("path", *out)
	}
	return say(stdout, "saved %s (%d bytes)\n", *out, len(png))
}

// runPerf is `glutter-web perf`: scripts/perf-web.sh runs it (SPEC §15, §23.8).
func runPerf(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("perf", flag.ContinueOnError)
	fs.SetOutput(stderr)
	browserName := fs.String("browser", "", "chromium or firefox")
	renderer := fs.String("renderer", "", "the renderer: GLUTTER_RENDERER (auto, webgpu, webgl or soft)")
	width := fs.Int("width", 1536, "the width of the app's viewport in CSS pixels")
	height := fs.Int("height", 864, "the height of the app's viewport in CSS pixels")
	scale := fs.Float64("scale", 1.25, "the device pixel ratio")
	loads := fs.Int("loads", 4, "how many times the page is loaded to time the start (the first with an empty cache)")
	settle := fs.Duration("settle", 3*time.Second, "how long the app runs after the last load before the measurement")
	measure := fs.Duration("measure", 10*time.Second, "how long the frames, the CPU time and requestAnimationFrame are measured")
	clicks := fs.String("clicks", "", "measure the response to clicks instead of the steady state: positions x,y separated by \";\" in CSS pixels, such as \"100,218;100,306\"")
	clickGap := fs.Duration("click-gap", 1500*time.Millisecond, "the time between two clicks")
	blank := fs.Bool("blank", false, "measure a page with no app, for the CPU time that the browser takes by itself (no package)")
	var query queryFlag
	fs.Var(&query, "query", "a query string added to the URL, such as GLUTTER_GALLERY_PAGE=data (may be repeated)")
	var bf buildFlags
	bf.add(fs)
	if err := fs.Parse(args); err != nil {
		return errors.NewErr(tagWebUsage, err)
	}
	if *browserName == "" || (!*blank && fs.NArg() != 1) || (*blank && fs.NArg() != 0) {
		return errors.New(tagWebUsage, "-browser and one package (none with -blank) are needed\n"+usage)
	}
	browser, err := webtest.ParseBrowser(*browserName)
	if err != nil {
		return err
	}
	dir := ""
	if !*blank {
		if dir, err = os.MkdirTemp("", "glutter-web-"); err != nil {
			return errors.NewErr(tagWebServeFailed, err)
		}
		defer func() { _ = os.RemoveAll(dir) }() // a temporary directory: a leftover is harmless
		if _, err := webtool.Build(ctx, webtool.BuildOptions{Package: fs.Arg(0), OutDir: dir, Release: bf.release, HostElement: bf.host}); err != nil {
			return err
		}
	}
	q := url.Values(query)
	if q == nil {
		q = url.Values{}
	}
	if *renderer != "" {
		q.Set("GLUTTER_RENDERER", *renderer)
	}
	points, err := parseClicks(*clicks)
	if err != nil {
		return err
	}
	res, err := webtest.Perf(ctx, webtest.PerfOptions{
		Clicks: points, ClickGap: *clickGap,
		Dir: dir, Browser: browser, Query: q, Width: *width, Height: *height, ScaleFactor: *scale,
		Loads: *loads, Settle: *settle, Measure: *measure, Blank: *blank,
	})
	if err != nil {
		return err
	}
	return say(stdout, "%s", res.Summary())
}

// parseClicks reads "x,y;x,y".
func parseClicks(s string) ([][2]float64, error) {
	var out [][2]float64
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		xs, ys, ok := strings.Cut(part, ",")
		x, errX := strconv.ParseFloat(strings.TrimSpace(xs), 64)
		y, errY := strconv.ParseFloat(strings.TrimSpace(ys), 64)
		if !ok || errX != nil || errY != nil {
			return nil, errors.New(tagWebUsage, "a click is x,y in CSS pixels", "value", part)
		}
		out = append(out, [2]float64{x, y})
	}
	return out, nil
}

// say prints to w.
func say(w io.Writer, format string, args ...any) error {
	if _, err := fmt.Fprintf(w, format, args...); err != nil {
		return errors.NewErr(tagWebUsage, err)
	}
	return nil
}

// size formats a size in bytes in megabytes.
func size(n int64) string { return fmt.Sprintf("%.1f MB", float64(n)/(1<<20)) }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWebServeFailed = "WebServeFailed"
	tagWebUsage       = "WebUsage"
	tagWebWriteFailed = "WebWriteFailed"
)
