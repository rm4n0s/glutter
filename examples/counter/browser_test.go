//go:build glutter_browser

package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
	m "github.com/rm4n0s/glutter/material"
)

// The browser tests build the counter for js/wasm once, serve it, and open it in headless
// Chromium and Firefox (SPEC §17 "Browser tests"). The engine reads its first idle frame back
// and posts it (proof method D), and the tests compare it with what the soft renderer draws
// in the widget tester.

var (
	buildOnce sync.Once
	siteDir   string
	buildErr  error
)

// site builds the counter on the first call and returns the directory of the build.
func site(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		siteDir, buildErr = os.MkdirTemp("", "glutter-counter-web-")
		if buildErr != nil {
			return
		}
		_, buildErr = webtool.Build(context.Background(), webtool.BuildOptions{
			Package: "./examples/counter", Dir: webtest.ModuleRoot(t), OutDir: siteDir,
		})
	})
	if buildErr != nil {
		t.Fatalf("building the counter for the web: %v", buildErr)
	}
	return siteDir
}

func TestMain(tm *testing.M) {
	code := tm.Run()
	if siteDir != "" {
		_ = os.RemoveAll(siteDir) // a temporary directory: a leftover is harmless
	}
	os.Exit(code)
}

// shot opens the counter, with three taps already made, in the browser and returns the frame
// it posts.
func shot(t *testing.T, b webtest.Browser, o webtest.ShotOptions) *image.RGBA {
	t.Helper()
	if o.Query == nil {
		o.Query = url.Values{}
	}
	o.Query.Set("GLUTTER_DEV_TAPS", "3")
	return shotAsIs(t, b, o)
}

// shotAsIs opens the counter in the browser with the options as they are and returns the frame it posts.
func shotAsIs(t *testing.T, b webtest.Browser, o webtest.ShotOptions) *image.RGBA {
	t.Helper()
	o.Dir, o.Browser = site(t), b
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	data, err := webtest.Shot(ctx, o)
	if err != nil {
		t.Fatalf("Shot in %s: %v", b, err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("the screenshot is not a PNG: %v", err)
	}
	rgba := image.NewRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba
}

// reference draws the counter after three taps in the widget tester at the size and ratio:
// what the soft renderer makes of it.
func reference(t *testing.T, width, height, dpr float64) *image.RGBA {
	t.Helper()
	useLinux(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: m.Size{Width: width, Height: height}, DevicePixelRatio: dpr})
	ts.PumpWidget(NewMyApp())
	for range 3 {
		ts.Tap(gluttertest.ByType[*m.FloatingActionButton]())
		ts.Pump(50 * time.Millisecond)
	}
	ts.Pump(time.Second)
	return ts.Screenshot()
}

// compare fails the test when the frame differs from the reference, and saves the frame next to
// the goldens.
func compare(t *testing.T, name string, want, got *image.RGBA) {
	t.Helper()
	res := goldens.Compare(want, got, goldens.DefaultTolerance)
	if res.OK() {
		return
	}
	path := filepath.Join("testdata", "goldens", name+".browser.actual.png")
	if err := goldens.WritePNG(path, got); err != nil {
		t.Errorf("saving the frame: %v", err)
	}
	t.Errorf("%s: the browser's frame differs from the reference (size mismatch %v, %d pixels, largest difference %d); the frame is saved as %s",
		name, res.SizeMismatch, res.Pixels, res.MaxDiff, path)
}

// TestBrowserFirstFrameMatchesTheGolden: the first idle frame of the counter in the browser, 480x640,
// is the soft golden of the counter.
func TestBrowserFirstFrameMatchesTheGolden(t *testing.T) {
	want, err := goldens.ReadPNG(filepath.Join("testdata", "goldens", "counter.png"))
	if err != nil {
		t.Fatal(err)
	}
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		got := shot(t, b, webtest.ShotOptions{Width: 480, Height: 640})
		compare(t, "counter-"+string(b), want, got)
	})
}

// TestBrowserResizeGivesAFrameOfTheNewSize: the window gets smaller after the first frame, and
// the frame the app draws next has the new size and the layout for it. The frame the app sees
// is its page's viewport, which the harness changes by resizing the frame the app is in.
func TestBrowserResizeGivesAFrameOfTheNewSize(t *testing.T) {
	want := reference(t, 360, 500, 1)
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		got := shot(t, b, webtest.ShotOptions{
			Width: 480, Height: 640,
			Query:   url.Values{"GLUTTER_SCREENSHOT_AFTER": {"6s"}},
			Resizes: []webtest.Resize{{After: time.Second, Width: 360, Height: 500}},
		})
		if got.Bounds().Dx() != 360 || got.Bounds().Dy() != 500 {
			t.Fatalf("the frame is %v, want 360x500", got.Bounds().Size())
		}
		compare(t, "counter-resized-"+string(b), want, got)
	})
}

// TestBrowserDevicePixelRatio: on a screen with a device pixel ratio of 2 the frame is twice the
// size in each direction and is what the soft renderer draws at that ratio.
func TestBrowserDevicePixelRatio(t *testing.T) {
	want := reference(t, 480, 640, 2)
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		got := shot(t, b, webtest.ShotOptions{Width: 480, Height: 640, ScaleFactor: 2})
		if got.Bounds().Dx() != 960 || got.Bounds().Dy() != 1280 {
			t.Fatalf("the frame is %v, want 960x1280", got.Bounds().Size())
		}
		compare(t, "counter-dpr2-"+string(b), want, got)
	})
}

// TestBrowserThreeTapsOnTheButtonShowThree: three taps on the floating action button, made by the
// debug-only script of synthetic pointer events (GLUTTER_DEV_EVENTS, replayed after the first frame),
// go through the web platform, the engine and the framework: the counter shows 3, as in the golden of
// the counter after three taps made in the widget tester.
func TestBrowserThreeTapsOnTheButtonShowThree(t *testing.T) {
	want, err := goldens.ReadPNG(filepath.Join("testdata", "goldens", "counter.png"))
	if err != nil {
		t.Fatal(err)
	}
	useLinux(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: m.Size{Width: 480, Height: 640}})
	ts.PumpWidget(NewMyApp())
	center := ts.Center(gluttertest.ByType[*m.FloatingActionButton]())
	tap := fmt.Sprintf("tap:%g,%g;wait:300ms;", center.Dx, center.Dy)
	// The mouse leaves the button at the end, so that it does not show its hover highlight.
	script := "wait:300ms;" + strings.Repeat(tap, 3) + "move:5,5"
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		got := shotAsIs(t, b, webtest.ShotOptions{
			Width: 480, Height: 640,
			Query: url.Values{"GLUTTER_DEV_EVENTS": {script}, "GLUTTER_SCREENSHOT_AFTER": {"6s"}},
		})
		compare(t, "counter-taps-"+string(b), want, got)
	})
}
