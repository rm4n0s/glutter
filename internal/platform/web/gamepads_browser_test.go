//go:build glutter_browser

package web_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// The gamepad test builds internal/platform/web/padtest, a small app that replaces navigator.getGamepads with a script of its
// own and posts what the framework did with the pad (SPEC §28.8): a pad with the standard mapping is plugged in, its D-pad moves
// the focus between two Material buttons and its A button activates the one that has it, the sticks and triggers reach the
// state of the pad, rumble plays the "dual-rumble" effect, and the page reads the pads only while one is connected.

var (
	padOnce sync.Once
	padDir  string
	padErr  error
)

func padSite(t *testing.T) string {
	t.Helper()
	padOnce.Do(func() {
		if padDir, padErr = os.MkdirTemp("", "glutter-web-padtest-"); padErr != nil {
			return
		}
		_, padErr = webtool.Build(context.Background(), webtool.BuildOptions{
			Package: "./internal/platform/web/padtest", Dir: webtest.ModuleRoot(t), OutDir: padDir,
		})
	})
	if padErr != nil {
		t.Fatalf("building the pad test app for the web: %v", padErr)
	}
	return padDir
}

func cleanupPadSite() {
	if padDir != "" {
		_ = os.RemoveAll(padDir) // a temporary directory: a leftover is harmless
	}
}

// padObservations mirrors what padtest posts.
type padObservations struct {
	ReadsBeforeConnect  int      `json:"readsBeforeConnect"`
	Pads                []string `json:"pads"`
	Kinds               []string `json:"kinds"`
	ReadsWhileConnected int      `json:"readsWhileConnected"`
	FocusStart          int      `json:"focusStart"`
	CountAfterA         [2]int   `json:"countAfterA"`
	FocusAfterDown      int      `json:"focusAfterDown"`
	CountAfterDownA     [2]int   `json:"countAfterDownA"`
	AxisLeftX           float64  `json:"axisLeftX"`
	TriggerRight        float64  `json:"triggerRight"`
	CanRumble           bool     `json:"canRumble"`
	Effects             []string `json:"effects"`
	Resets              int      `json:"resets"`
	ThrowTag            string   `json:"throwTag"`
	NoActuatorTag       string   `json:"noActuatorTag"`
	PadsAfterUnplug     int      `json:"padsAfterUnplug"`
	ReadsAfterUnplug    int      `json:"readsAfterUnplug"`
}

// TestBrowserGamepads: a replaced navigator.getGamepads drives a Material button through focus, in Chromium and Firefox.
func TestBrowserGamepads(t *testing.T) {
	dir := padSite(t)
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		res, err := webtest.Await(ctx, webtest.ShotOptions{Dir: dir, Browser: b, Width: 480, Height: 640}, "pads")
		if err != nil {
			t.Fatalf("Await in %s: %v", b, err)
		}
		var o padObservations
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
		if o.ReadsBeforeConnect > 2 {
			t.Errorf("the page read the pads %d times before one was plugged in, want at most the read made at the start", o.ReadsBeforeConnect)
		}
		check("pads after the plug", strings.Join(o.Pads, ","), "Xbox 360 Controller")
		check("kind of the pad", strings.Join(o.Kinds, ","), "Xbox 360")
		if o.ReadsWhileConnected < 5 {
			t.Errorf("the page read the pads %d times in half a second with a pad plugged in, want about one for each frame", o.ReadsWhileConnected)
		}
		check("button focused at the start", o.FocusStart, 0)
		check("counts after A", o.CountAfterA, [2]int{1, 0})
		check("button focused after the D-pad down", o.FocusAfterDown, 1)
		check("counts after the D-pad down and A", o.CountAfterDownA, [2]int{1, 1})
		if o.AxisLeftX < 0.499 || o.AxisLeftX > 0.501 {
			t.Errorf("left stick x = %v, want 0.5", o.AxisLeftX)
		}
		check("right trigger", o.TriggerRight, 1.0)
		check("the pad can rumble", o.CanRumble, true)
		check("effects played", strings.Join(o.Effects, ";"), "dual-rumble 1000 0.50 0.25")
		check("resets of the actuator", o.Resets, 1)
		check("tag of a rumble whose actuator throws", o.ThrowTag, "WebRumbleFailed")
		check("tag of a rumble of a pad with no actuator", o.NoActuatorTag, "GamepadRumbleUnavailable")
		check("pads after the unplug", o.PadsAfterUnplug, 0)
		check("reads after the unplug", o.ReadsAfterUnplug, 0)
	})
}
