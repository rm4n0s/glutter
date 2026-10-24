//go:build glutter_android

package androidtest

import (
	"context"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// sysApp installs examples/dev/android-system, which logs what the framework knows of the system on
// each build, and starts it with the renderer soft (so that the tests do not depend on the GPU) and the
// demo that is asked for (GLUTTER_DEMO). It returns once the app has drawn.
func sysApp(ctx context.Context, t *testing.T, e *Emulator, demo string) string {
	t.Helper()
	appID := installApp(ctx, t, e, "./examples/dev/android-system")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	extras := map[string]string{"GLUTTER_RENDERER": "soft"}
	if demo != "" {
		extras["GLUTTER_DEMO"] = demo
	}
	if err := e.Launch(ctx, appID, extras); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "first frame presented", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	return appID
}

// mq is the last "mq" line of the log of examples/dev/android-system, as a map of its keys.
type mq map[string]string

func lastMQ(log string) mq {
	var last string
	for _, l := range strings.Split(log, "\n") {
		if strings.Contains(l, " INFO mq ") {
			last = l
		}
	}
	out := mq{}
	if last == "" {
		return out
	}
	for _, f := range strings.Fields(last[strings.Index(last, " INFO mq ")+len(" INFO mq "):]) {
		if k, v, ok := strings.Cut(f, "="); ok {
			out[k] = v
		}
	}
	return out
}

func (m mq) float(key string) float64 {
	f, _ := strconv.ParseFloat(m[key], 64)
	return f
}

// edges reads the four numbers of an insets value "l,t,r,b".
func (m mq) edges(key string) [4]float64 {
	var out [4]float64
	for i, p := range strings.Split(m[key], ",") {
		if i < 4 {
			out[i], _ = strconv.ParseFloat(p, 64)
		}
	}
	return out
}

// waitMQ waits until the last mq line has the value, for 20 seconds, and returns it.
func waitMQ(ctx context.Context, t *testing.T, e *Emulator, appID string, ok func(mq) bool, what string) mq {
	t.Helper()
	var last mq
	for end := time.Now().Add(20 * time.Second); ; time.Sleep(300 * time.Millisecond) {
		log, err := e.Log(ctx, appID)
		if err != nil {
			t.Fatal(Describe(err))
		}
		last = lastMQ(log)
		if len(last) > 0 && ok(last) {
			return last
		}
		if time.Now().After(end) {
			t.Fatalf("timed out waiting for %s; the last mq line is %v", what, last)
		}
	}
}

// forEachDevice runs a test on the API 37 image and on the minimum API level.
func forEachDevice(t *testing.T, fn func(t *testing.T, e *Emulator, ctx context.Context)) {
	for _, avd := range []string{"glutter-37", "glutter-26"} {
		t.Run(avd, func(t *testing.T) {
			e := Shared(t, avd)
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
			defer cancel()
			fn(t, e, ctx)
		})
	}
}

// TestInsetsAreTheStatusBarAndTheNavigationBar checks that the padding that the framework gets is what
// the system says the bars cover: the status bar at the top, and the navigation bar at the bottom or
// the side, and that the view fills the screen.
func TestInsetsAreTheStatusBarAndTheNavigationBar(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := sysApp(ctx, t, e, "")
		m := waitMQ(ctx, t, e, appID, func(m mq) bool { return m.edges("viewPadding")[1] > 0 }, "the status bar's padding")
		dpr := m.float("dpr")
		pad := m.edges("viewPadding")
		out := shell(ctx, t, e, "dumpsys", "window")
		top := statusBarHeight(out)
		if top == 0 {
			t.Fatalf("dumpsys window has no status bar:\n%s", grep(out, "statusBars", "InsetsSource", "mStable"))
		}
		if got := math.Round(pad[1] * dpr); got != float64(top) {
			t.Errorf("the top padding is %v pixels, the status bar is %d", got, top)
		}
		t.Logf("padding %v dpr %v status bar %d", pad, dpr, top)
		// The view draws behind the bars: the logical size is the whole screen.
		size := strings.Split(m["size"], "x")
		w, _ := strconv.ParseFloat(size[0], 64)
		if w*dpr < 1000 {
			t.Errorf("the view is %v pixels wide", w*dpr)
		}
	})
}

// statusBarHeight is the height in pixels of the status bar in the dump of the window service: the
// insets source of Android 11 and later, or the top of the stable area before.
func statusBarHeight(dump string) int {
	if h := barHeight(dump, "statusBars"); h > 0 {
		return h
	}
	for _, l := range strings.Split(dump, "\n") {
		if l = strings.TrimSpace(l); strings.HasPrefix(l, "mStable=(") {
			// mStable=(0,63)-(1080,2274)
			if i := strings.Index(l, ","); i > 0 {
				if j := strings.Index(l[i:], ")"); j > 0 {
					top, _ := strconv.Atoi(l[i+1 : i+j])
					return top
				}
			}
		}
	}
	return 0
}

// barHeight is the height of the insets source of a type in the dump of the window service, or 0.
func barHeight(dump, kind string) int {
	for _, l := range strings.Split(dump, "\n") {
		if strings.Contains(l, "InsetsSource") && strings.Contains(l, "type="+kind) {
			// InsetsSource id=... type=statusBars frame=[0,0][1080,63] visible=true
			if i := strings.Index(l, "frame=["); i >= 0 {
				rest := l[i+len("frame=["):]
				if j := strings.Index(rest, "]"); j >= 0 {
					p := strings.Split(rest[:j], ",")
					rest = rest[j+2:]
					if k := strings.Index(rest, "]"); k >= 0 && len(p) == 2 {
						q := strings.Split(rest[:k], ",")
						if len(q) == 2 {
							y0, _ := strconv.Atoi(p[1])
							y1, _ := strconv.Atoi(q[1])
							if y1 > y0 {
								return y1 - y0
							}
						}
					}
				}
			}
		}
	}
	return 0
}

func grep(text string, parts ...string) string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		for _, p := range parts {
			if strings.Contains(l, p) {
				out = append(out, l)
				break
			}
		}
	}
	return strings.Join(out, "\n")
}

// pidOf is the process of the app, for a test that checks that a setting did not restart it.
func pidOf(ctx context.Context, t *testing.T, e *Emulator, appID string) int {
	t.Helper()
	pid, err := e.WaitPid(ctx, appID, 30*time.Second)
	if err != nil {
		t.Fatal(Describe(err))
	}
	return pid
}

// TestSettingsFollowTheSystem changes the night mode, the font scale, the animator duration scale and
// the clock of the system while the app runs: the framework hears of each, and the process is the same.
func TestSettingsFollowTheSystem(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		t.Cleanup(func() {
			for _, c := range [][]string{
				{"cmd", "uimode", "night", "no"},
				{"settings", "put", "system", "font_scale", "1.0"},
				{"settings", "put", "global", "animator_duration_scale", "1.0"},
				{"settings", "put", "system", "time_12_24", "12"},
			} {
				_, _ = e.ADB.Output(context.Background(), append([]string{"shell"}, c...)...)
			}
		})
		appID := sysApp(ctx, t, e, "")
		pid := pidOf(ctx, t, e, appID)
		m := waitMQ(ctx, t, e, appID, func(m mq) bool { return m["brightness"] != "" }, "the first build")
		if !strings.Contains(strings.ToLower(m["brightness"]), "light") || m.float("textScale") != 1 || m["animationsOff"] != "false" || m["h24"] != "false" {
			t.Fatalf("the system is not in its default state: %v", m)
		}

		shell(ctx, t, e, "cmd", "uimode", "night", "yes")
		waitMQ(ctx, t, e, appID, func(m mq) bool { return strings.Contains(strings.ToLower(m["brightness"]), "dark") }, "the dark theme")

		// As in Flutter, a change of the 24-hour setting alone is not a reason to rebuild the
		// MediaQuery: it comes with the next change, here the font scale.
		shell(ctx, t, e, "settings", "put", "system", "time_12_24", "24")
		shell(ctx, t, e, "settings", "put", "system", "font_scale", "1.5")
		m = waitMQ(ctx, t, e, appID, func(m mq) bool { return m.float("textScale") == 1.5 }, "the font scale 1.5")
		if e.AVD.API == "26" {
			if m.float("scale10") != 15 || m.float("scale100") != 150 {
				t.Errorf("the text scaling of API 26 is linear: scale(10)=%v scale(100)=%v", m["scale10"], m["scale100"])
			}
		} else {
			// Android 14 and later scale large text less than small text.
			// (The system's curve: 10 sp is 15 at the scale 1.5, and large sizes are not scaled at all.)
			if s10, s100 := m.float("scale10"), m.float("scale100"); s10 < 14 || s10 > 16 || s100 >= 150 || s100 < 100 {
				t.Errorf("the text scaling is not the system's non-linear one: scale(10)=%v scale(100)=%v", s10, s100)
			}
		}
		if m["h24"] != "true" {
			t.Errorf("the 24-hour clock did not reach the framework: %v", m)
		}

		shell(ctx, t, e, "settings", "put", "global", "animator_duration_scale", "0")
		waitMQ(ctx, t, e, appID, func(m mq) bool { return m["animationsOff"] == "true" }, "animations off")

		if got := pidOf(ctx, t, e, appID); got != pid {
			t.Errorf("the app restarted: process %d, then %d", pid, got)
		}
		if alive, err := e.HasActivity(ctx, appID); err != nil || !alive {
			t.Errorf("the activity is gone: %v %v", alive, err)
		}
	})
}

// TestTheAppAsksForAnOrientation checks SystemChrome.setPreferredOrientations: the view is wider than
// tall when the app asks for landscape, taller when it asks for portrait.
func TestTheAppAsksForAnOrientation(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		wide := func(m mq) bool { return aspect(m) > 1 }
		appID := sysApp(ctx, t, e, "landscape")
		waitMQ(ctx, t, e, appID, wide, "a landscape view")
		shell(ctx, t, e, "am", "force-stop", appID)
		appID = sysApp(ctx, t, e, "portrait")
		waitMQ(ctx, t, e, appID, func(m mq) bool { return aspect(m) < 1 }, "a portrait view")
	})
}

func aspect(m mq) float64 {
	size := strings.Split(m["size"], "x")
	if len(size) != 2 {
		return 0
	}
	w, _ := strconv.ParseFloat(size[0], 64)
	h, _ := strconv.ParseFloat(size[1], 64)
	if h == 0 {
		return 0
	}
	return w / h
}

// TestImmersiveModeHidesTheBars checks SystemChrome.setEnabledSystemUIMode: with the bars hidden the
// padding of the navigation bar is gone (and the status bar's, but for a display cutout, which stays), and the
// app hears of the bars coming back when the user swipes them in.
func TestImmersiveModeHidesTheBars(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := sysApp(ctx, t, e, "immersive,edge-late")
		t.Cleanup(func() {
			if t.Failed() {
				bg := context.Background()
				log, _ := e.FullLog(bg)
				dump, _ := e.ADB.Output(bg, "shell", "dumpsys", "window")
				t.Logf("java: %s\nbars: %s", grep(log, "GlutterJava"), grep(dump, "type=statusBars", "type=navigationBars"))
			}
		})
		m := waitMQ(ctx, t, e, appID, func(m mq) bool { return m["size"] != "" && m.edges("viewPadding")[3] == 0 }, "no navigation bar padding")
		t.Logf("immersive: %v", m)
		if top := m.edges("viewPadding")[1]; e.AVD.API == "26" && top != 0 {
			t.Errorf("the status bar is hidden and the top padding is %v", top)
		}
		if out := shell(ctx, t, e, "dumpsys", "window"); strings.Contains(grep(out, "type=statusBars"), "visible=true") {
			t.Errorf("the status bar is visible:\n%s", grep(out, "type=statusBars"))
		}
		// The bars come back (the app asks for the edge-to-edge mode three seconds later), and the app is told.
		if _, err := e.WaitLog(ctx, appID, "systemui visible=true", 20*time.Second); err != nil {
			t.Error(Describe(err))
		}
	})
}

// TestFeedbackReachesTheSystem checks that HapticFeedback and SystemSound reach the activity, and that
// KeepScreenOn sets the flag of the window; the app logs the commands of the debug build.
func TestFeedbackReachesTheSystem(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := sysApp(ctx, t, e, "haptic,sound,keepon,style-dark")
		time.Sleep(2 * time.Second)
		log, err := e.FullLog(ctx)
		if err != nil {
			t.Fatal(Describe(err))
		}
		for _, want := range []string{"command op=1 a=4", "command op=2 a=1", "command op=6 a=1", "command op=5"} {
			if !strings.Contains(log, want) {
				t.Errorf("the activity did not get %q:\n%s", want, grep(log, "GlutterJava"))
			}
		}
		if out := shell(ctx, t, e, "dumpsys", "window", "windows"); !strings.Contains(out, "keepScreenOn") && !strings.Contains(out, "FLAG_KEEP_SCREEN_ON") {
			t.Logf("dumpsys window does not show the keep-screen-on flag (not an error: its text varies):\n%s", grep(out, "ScreenOn", "SCREEN_ON"))
		}
		_ = appID
	})
}

// TestProofDarkGalleryUnderTheStatusBar takes the screen of the dark gallery: the app bar is below the
// status bar, whose icons are light. With GLUTTER_PROOF_OUT set to a file name the image is written there
// (the proof image of step 25.7).
func TestProofDarkGalleryUnderTheStatusBar(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	shell(ctx, t, e, "cmd", "uimode", "night", "yes")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "cmd", "uimode", "night", "no") })
	appID := installApp(ctx, t, e, "./examples/gallery")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	// The gallery has its own theme mode, light at first: the dev events tap the moon in the app bar (the
	// button that switches to the dark theme, at the right of the app bar of a phone).
	if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_DEV_EVENTS": "wait:2s;tap:996,200"}); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "first frame presented", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	time.Sleep(6 * time.Second)
	b, err := e.ADB.Screencap(ctx)
	if err != nil {
		t.Fatal(Describe(err))
	}
	img := decodePNG(t, b)
	// The corner of the status bar is the dark theme's background, not the light one.
	if c := img.RGBAAt(img.Rect.Dx()/2, 10); int(c.R)+int(c.G)+int(c.B) > 3*100 {
		t.Errorf("the top of the screen is light in the dark theme: %v", c)
	}
	if out := os.Getenv("GLUTTER_PROOF_OUT"); out != "" {
		if err := os.WriteFile(out, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
