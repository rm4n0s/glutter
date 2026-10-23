package webtest

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/web"
)

// Await opens the page of the build in a headless browser, like Shot, but waits for a result of
// the program (a Result posted to web.ResultsPath with the Kind) instead of a screenshot: a test
// program that dispatches its own events and reports what it saw. GLUTTER_SCREENSHOT is not set.
// It starts the server and the browser and stops both before it returns. It fails with
// WebAppFailed when the program ends before it posted the result (the error carries what it
// wrote), BrowserExited when the browser ends first, and BrowserTimeout when the result does not
// come in time.
//
// Flutter: no counterpart
func Await(ctx context.Context, o ShotOptions, kind string) (Result, error) {
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
	q.Set("glutter_report", web.ResultsPath)
	width, height := o.Width, o.Height
	if width == 0 || height == 0 {
		width, height = 480, 640
	}
	srv.AddPage(FramePath, FramePage(page+"?"+q.Encode(), width, height, o.Resizes))
	proc, err := Launch(o.Browser, srv.URL+FramePath, LaunchOptions{
		Path: o.BrowserPath, Width: max(width, 500) + 40, Height: height + 200, ScaleFactor: o.ScaleFactor, Output: o.Output,
	})
	if err != nil {
		return Result{}, err
	}
	defer proc.Stop()
	timeout := o.Timeout
	if timeout == 0 {
		timeout = time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case res := <-srv.results:
			if res.Kind == kind {
				return res, nil
			}
			if res.Kind == "exit" {
				return Result{}, errors.New(tagWebAppFailed, "the app ended before it reported "+strconv.Quote(kind),
					"exit code", strconv.Itoa(res.ExitCode), "message", res.Message, "shown", res.Shown, "log", strings.TrimSpace(res.Log))
			}
		case <-proc.Done():
			md := []any{}
			if proc.Err != nil {
				md = append(md, "error", proc.Err.Error())
			}
			return Result{}, errors.New(tagBrowserExited, "the browser ended before the result came", md...)
		case <-ctx.Done():
			return Result{}, errors.NewErr(tagBrowserTimeout, ctx.Err()).SetMetadata("waiting for", "the result "+kind)
		}
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBrowserExited  = "BrowserExited"
	tagBrowserTimeout = "BrowserTimeout"
	tagWebAppFailed   = "WebAppFailed"
)
