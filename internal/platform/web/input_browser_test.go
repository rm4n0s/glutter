//go:build glutter_browser

package web_test

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// The input tests build internal/platform/web/inputtest, a small app that dispatches synthetic DOM
// events at its own canvas and posts what it saw (SPEC §23.4): taps on a button, a key that a
// Shortcuts widget handles, wheel events over a ListView, the cursor, and the browser's context menu.

var (
	inputOnce sync.Once
	inputDir  string
	inputErr  error
)

func inputSite(t *testing.T) string {
	t.Helper()
	inputOnce.Do(func() {
		if inputDir, inputErr = os.MkdirTemp("", "glutter-web-inputtest-"); inputErr != nil {
			return
		}
		_, inputErr = webtool.Build(context.Background(), webtool.BuildOptions{
			Package: "./internal/platform/web/inputtest", Dir: webtest.ModuleRoot(t), OutDir: inputDir,
		})
	})
	if inputErr != nil {
		t.Fatalf("building the input test app for the web: %v", inputErr)
	}
	return inputDir
}

func cleanupInputSite() {
	if inputDir != "" {
		_ = os.RemoveAll(inputDir) // a temporary directory: a leftover is harmless
	}
}

// observations mirrors what inputtest posts.
type observations struct {
	TabIndex               string   `json:"tabIndex"`
	Downs                  []string `json:"downs"`
	Count                  int      `json:"count"`
	Cursor                 string   `json:"cursor"`
	CursorAway             string   `json:"cursorAway"`
	KeyHandledPrevented    bool     `json:"keyHandledPrevented"`
	KeyUnhandledPrevented  bool     `json:"keyUnhandledPrevented"`
	KeyCount               int      `json:"keyCount"`
	WheelList              float64  `json:"wheelList"`
	WheelListPrevented     bool     `json:"wheelListPrevented"`
	WheelBandPrevented     bool     `json:"wheelBandPrevented"`
	WheelBack              float64  `json:"wheelBack"`
	WheelBoundaryPrevented bool     `json:"wheelBoundaryPrevented"`
	TimerKeyPrevented      bool     `json:"timerKeyPrevented"`
	TimerWheelPrevented    bool     `json:"timerWheelPrevented"`
	TimerKeyCount          int      `json:"timerKeyCount"`
	TimerOffset            float64  `json:"timerOffset"`
	MenuDisabledPrevented  bool     `json:"menuDisabledPrevented"`
	MenuEnabledPrevented   bool     `json:"menuEnabledPrevented"`
	MenuEnabledFlag        bool     `json:"menuEnabledFlag"`
	MenuDisabledFlag       bool     `json:"menuDisabledFlag"`
}

// TestBrowserInput: in Chromium and Firefox, synthetic pointer, key, wheel and contextmenu events
// at the canvas reach the framework as the real ones would, and the answers of the framework reach
// the events: three taps on a button count 3, the key that a Shortcuts widget handles has its
// default prevented and one that nothing handles does not, wheel events scroll a ListView and are
// prevented there but not over a part that does not scroll or at the end of the list, the button
// asks for the pointing hand, and the browser's context menu can be disabled.
func TestBrowserInput(t *testing.T) {
	dir := inputSite(t)
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		res, err := webtest.Await(ctx, webtest.ShotOptions{Dir: dir, Browser: b, Width: 480, Height: 640}, "inputs")
		if err != nil {
			t.Fatalf("Await in %s: %v", b, err)
		}
		var o observations
		if err := json.Unmarshal(res.Data, &o); err != nil {
			t.Fatalf("the result is not the observations: %v\n%s", err, res.Data)
		}
		t.Logf("%+v", o)
		check := func(name string, got, want any) {
			t.Helper()
			if got != want {
				t.Errorf("%s = %v, want %v", name, got, want)
			}
		}
		check("the tabindex of the host", o.TabIndex, "0")
		check("pointer-down events the framework saw", len(o.Downs), 3)
		check("taps on the button", o.Count, 3)
		check("cursor over the button", o.Cursor, "pointer")
		check("cursor over the list", o.CursorAway, "auto")
		check("default of the K key (handled by Shortcuts) prevented", o.KeyHandledPrevented, true)
		check("default of the Q key (unhandled) prevented", o.KeyUnhandledPrevented, false)
		check("invocations of the shortcut", o.KeyCount, 1)
		check("offset of the list after a wheel of 300", o.WheelList, 300.0)
		check("default of the wheel over the list prevented", o.WheelListPrevented, true)
		check("default of the wheel over the band that does not scroll prevented", o.WheelBandPrevented, false)
		check("offset of the list scrolled back", o.WheelBack, 0.0)
		check("default of the wheel at the top of the list prevented", o.WheelBoundaryPrevented, false)
		// Events the browser itself delivers, from its timer: the listeners wait for the UI goroutine.
		check("default of the K key from the timer prevented", o.TimerKeyPrevented, true)
		check("default of the wheel from the timer prevented", o.TimerWheelPrevented, true)
		check("invocations of the shortcut after the timer's key", o.TimerKeyCount, 2)
		check("offset of the list after the timer's wheel", o.TimerOffset, 100.0)
		check("context menu enabled to begin with", o.MenuEnabledFlag, true)
		check("context menu disabled by the app", o.MenuDisabledFlag, true)
		check("default of the contextmenu event prevented while disabled", o.MenuDisabledPrevented, true)
		check("default of the contextmenu event prevented once enabled", o.MenuEnabledPrevented, false)
	})
}

// TestBrowserAwaitTimesOut: a result that the program never posts is a BrowserTimeout, and the
// browser is stopped.
func TestBrowserAwaitTimesOut(t *testing.T) {
	dir := inputSite(t)
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		_, err := webtest.Await(ctx, webtest.ShotOptions{Dir: dir, Browser: b, Timeout: 3 * time.Second}, "never")
		appErr, ok := errors.FromError(err)
		if !ok || !appErr.HasRoute("Await.BrowserTimeout") {
			t.Fatalf("got %v, want a BrowserTimeout from Await", err)
		}
	})
}
