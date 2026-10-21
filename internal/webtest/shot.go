package webtest

import (
	"context"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/web"
)

// ShotOptions are the parameters of Shot.
//
// Flutter: no counterpart
type ShotOptions struct {
	// Dir is the build to serve (the directory webtool.Build wrote).
	Dir string
	// Browser is the browser to open it in.
	Browser Browser
	// BrowserPath is the program of the browser; found when empty.
	BrowserPath string
	// Page is the path of the page of the app, "/index.html" when empty.
	Page string
	// Query is added to the URL's query: GLUTTER_* parameters become the program's
	// environment (SPEC §23.1). GLUTTER_SCREENSHOT=1 is always added, which makes the
	// engine post its first idle frame (SPEC §23.8).
	Query url.Values
	// Width and Height are the size of the viewport the app sees, in CSS pixels; 480x640
	// when zero. The app runs in a frame of that size (FramePage).
	Width, Height int
	// ScaleFactor is the device pixel ratio of the browser; 1 when zero.
	ScaleFactor float64
	// Resizes change the size of the frame some time after the page loads.
	Resizes []Resize
	// Timeout is how long to wait for the screenshot; one minute when zero.
	Timeout time.Duration
	// Output receives the browser's standard output and error.
	Output io.Writer
	// Server, when set, is used instead of a new one for Dir: a test that has added pages to
	// it. Shot does not close it.
	Server *Server
}

// Shot opens the page of the build in a headless browser and returns the PNG that the
// engine posts after its first idle frame (proof method D, SPEC §23.8). It starts the
// server and the browser and stops both before it returns. It fails with WebAppFailed when
// the program ends with an error (the error carries what it wrote), BrowserExited when the
// browser ends first, and BrowserTimeout when the screenshot does not come in time; the
// errors of Launch come back unchanged.
//
// Flutter: no counterpart (flutter drive --screenshot)
func Shot(ctx context.Context, o ShotOptions) ([]byte, error) {
	srv := o.Server
	if srv == nil {
		srv = NewServer(o.Dir)
		defer srv.Close()
	}
	page := o.Page
	if page == "" {
		page = "/index.html"
	}
	q := url.Values{}
	for k, v := range o.Query {
		q[k] = v
	}
	q.Set("GLUTTER_SCREENSHOT", "1")
	q.Set("glutter_report", web.ResultsPath)
	width, height := o.Width, o.Height
	if width == 0 || height == 0 {
		width, height = 480, 640
	}
	srv.AddPage(FramePath, FramePage(page+"?"+q.Encode(), width, height, o.Resizes))
	// The window is bigger than the frame: browsers keep a minimum width and show toolbars.
	proc, err := Launch(o.Browser, srv.URL+FramePath, LaunchOptions{
		Path: o.BrowserPath, Width: max(width, 500) + 40, Height: height + 200, ScaleFactor: o.ScaleFactor, Output: o.Output,
	})
	if err != nil {
		return nil, err
	}
	defer proc.Stop()
	timeout := o.Timeout
	if timeout == 0 {
		timeout = time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return awaitScreenshot(ctx, srv, proc.Done(), func() error { return proc.Err })
}

// awaitScreenshot waits for the PNG, for the program to say it failed, for the browser to end,
// or for ctx; a PNG that is there wins over everything else.
func awaitScreenshot(ctx context.Context, srv *Server, browserDone <-chan struct{}, browserErr func() error) ([]byte, error) {
	for {
		select {
		case png := <-srv.shots:
			return png, nil
		default:
		}
		select {
		case png := <-srv.shots:
			return png, nil
		case res := <-srv.results:
			if res.Kind != "exit" {
				continue
			}
			select {
			case png := <-srv.shots: // posted just before the exit
				return png, nil
			default:
			}
			if res.ExitCode != 0 || res.Message != "" {
				return nil, errors.New("WebAppFailed", "the app ended with an error",
					"exit code", strconv.Itoa(res.ExitCode), "message", res.Message, "log", strings.TrimSpace(res.Log))
			}
			return nil, errors.New("WebAppFailed", "the app ended without taking a screenshot", "log", strings.TrimSpace(res.Log))
		case <-browserDone:
			select {
			case png := <-srv.shots:
				return png, nil
			default:
			}
			md := []any{}
			if err := browserErr(); err != nil {
				md = append(md, "error", err.Error())
			}
			return nil, errors.New("BrowserExited", "the browser ended before the screenshot came", md...)
		case <-ctx.Done():
			return nil, errors.NewErr("BrowserTimeout", ctx.Err()).SetMetadata("waiting for", "a screenshot")
		}
	}
}
