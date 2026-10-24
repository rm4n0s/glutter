//go:build glutter_android

package androidtest

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
)

// inputApp installs examples/dev/input (it logs every key and pointer event the framework gets, and
// sets the mouse cursor) and starts it with the renderer soft, so that the tests do not depend on
// the GPU; it returns once the app has its metrics.
func inputApp(ctx context.Context, t *testing.T, e *Emulator, pkg string, extras map[string]string) string {
	t.Helper()
	appID := installApp(ctx, t, e, pkg)
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	all := map[string]string{"GLUTTER_RENDERER": "soft"}
	for k, v := range extras {
		all[k] = v
	}
	if err := e.Launch(ctx, appID, all); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "renderer chosen", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	time.Sleep(time.Second)
	return appID
}

// pointerLines are the "pointer" lines of the log of examples/dev/input, one string per line.
func pointerLines(log string) []string {
	var out []string
	for _, l := range strings.Split(log, "\n") {
		if strings.Contains(l, "INFO pointer ") {
			out = append(out, l)
		}
	}
	return out
}

func has(lines []string, parts ...string) bool {
	for _, l := range lines {
		ok := true
		for _, p := range parts {
			ok = ok && strings.Contains(l, p)
		}
		if ok {
			return true
		}
	}
	return false
}

// index is the position of the first line that has all the parts, or -1.
func index(lines []string, parts ...string) int {
	for i, l := range lines {
		ok := true
		for _, p := range parts {
			ok = ok && strings.Contains(l, p)
		}
		if ok {
			return i
		}
	}
	return -1
}

// ptr is a "pointer" line of examples/dev/input read back.
type ptr struct {
	change, kind string
	x, y         float64
}

var pointerLine = regexp.MustCompile(`INFO pointer change=PointerChange(\w+) kind=PointerDeviceKind(\w+) device=\d+ x=([\d.]+) y=([\d.]+)`)

// pointers reads the pointer lines of the log of examples/dev/input.
func pointers(log string) []ptr {
	var out []ptr
	for _, l := range strings.Split(log, "\n") {
		if m := pointerLine.FindStringSubmatch(l); m != nil {
			x, _ := strconv.ParseFloat(m[3], 64)
			y, _ := strconv.ParseFloat(m[4], 64)
			out = append(out, ptr{m[1], m[2], x, y})
		}
	}
	return out
}

// near says whether the pointer is at the place, within the few pixels that the system moves a
// touch by (the injected events of `adb shell input` are not exactly where they are asked).
func (p ptr) near(x, y float64) bool { return abs(p.x-x) <= 12 && abs(p.y-y) <= 12 }

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// TestTouchTapAndSwipeAreDelivered sends a tap and a swipe with `adb shell input`: the framework gets
// the touch as Flutter's embedding gives it (an add before the down, the up, and the remove after
// it), at the place that was touched, and a swipe has moves between its down and its up.
func TestTouchTapAndSwipeAreDelivered(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	appID := inputApp(ctx, t, e, "./examples/dev/input", nil)

	shell(ctx, t, e, "input", "tap", "540", "1200")
	log, err := e.WaitLog(ctx, appID, "change=PointerChangeRemove kind=PointerDeviceKindTouch", 20*time.Second)
	if err != nil {
		t.Fatal(Describe(err))
	}
	seen := pointers(log)
	var order []string
	for _, p := range seen {
		if p.kind == "Touch" {
			order = append(order, p.change)
			if !p.near(540, 1200) {
				t.Errorf("%s is at (%.1f, %.1f), not at (540, 1200)", p.change, p.x, p.y)
			}
		}
	}
	// A tap may have a move between the down and the up (the system resamples the touch).
	if got := strings.Join(slicesDropAll(order, "Move"), " "); got != "Add Down Up Remove" {
		t.Errorf("the tap is %q, want Add Down Up Remove", got)
	}

	shell(ctx, t, e, "input", "swipe", "540", "1800", "540", "600", "400")
	deadline := time.Now().Add(20 * time.Second)
	var swipe []ptr
	for {
		log, _ = e.Log(ctx, appID)
		seen = pointers(log)
		swipe = nil
		for i, p := range seen {
			if p.kind == "Touch" && p.change == "Down" && p.near(540, 1800) {
				swipe = seen[i:]
			}
		}
		if len(swipe) > 0 && swipe[len(swipe)-1].change == "Remove" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the swipe never ended:\n%v", seen)
		}
		time.Sleep(300 * time.Millisecond)
	}
	moves := 0
	for _, p := range swipe {
		if p.change == "Move" {
			moves++
		}
	}
	up := swipe[len(swipe)-2]
	if up.change != "Up" || abs(up.x-540) > 12 || abs(up.y-600) > 40 || moves < 3 {
		t.Errorf("the swipe has %d moves and ends with %+v, want 3 or more moves and an up near (540, 600)", moves, up)
	}
}

// slicesDropAll returns the strings of s that are not drop.
func slicesDropAll(s []string, drop string) []string {
	var out []string
	for _, v := range s {
		if v != drop {
			out = append(out, v)
		}
	}
	return out
}

// TestMouseStylusAndScroll sends a mouse click, the scroll wheel and a stylus tap. `adb shell input`
// has no button state for the mouse, and Flutter's embedding takes a mouse down with no buttons from
// a mouse source for the trackpad scrolling of a Chromebook (a pan and zoom): that is what the click
// gives; the wheel scrolls, and the stylus touches as a stylus.
func TestMouseStylusAndScroll(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	appID := inputApp(ctx, t, e, "./examples/dev/input", nil)

	shell(ctx, t, e, "input", "mouse", "tap", "300", "500")
	log, err := e.WaitLog(ctx, appID, "change=PointerChangePanZoomEnd kind=PointerDeviceKindTrackpad", 20*time.Second)
	if err != nil {
		t.Fatal(Describe(err))
	}
	lines := pointerLines(log)
	if a, b := index(lines, "change=PointerChangePanZoomStart kind=PointerDeviceKindTrackpad"), index(lines, "change=PointerChangePanZoomEnd kind=PointerDeviceKindTrackpad"); a < 0 || b < a {
		t.Errorf("the click without buttons is not a pan and zoom of the trackpad:\n%s", strings.Join(lines, "\n"))
	}

	shell(ctx, t, e, "input", "mouse", "scroll", "300", "500", "--axis", "VSCROLL,-3")
	log, err = e.WaitLog(ctx, appID, "scrollY=", 20*time.Second)
	if err != nil {
		t.Fatal(Describe(err))
	}
	if !regexp.MustCompile(`kind=PointerDeviceKindMouse .*scrollY=-?[1-9]`).MatchString(strings.Join(pointerLines(log), "\n")) {
		t.Errorf("the wheel did not scroll:\n%s", strings.Join(pointerLines(log), "\n"))
	}

	shell(ctx, t, e, "input", "stylus", "tap", "300", "900")
	log, err = e.WaitLog(ctx, appID, "kind=PointerDeviceKindStylus", 20*time.Second)
	if err != nil {
		t.Fatal(Describe(err))
	}
	if !has(pointerLines(log), "change=PointerChangeDown kind=PointerDeviceKindStylus") {
		t.Errorf("the stylus did not touch:\n%s", strings.Join(pointerLines(log), "\n"))
	}
}

// TestKeysAreDelivered sends keys with `adb shell input keyevent`: the framework gets Flutter's key
// data, the physical and the logical key.
func TestKeysAreDelivered(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	appID := inputApp(ctx, t, e, "./examples/dev/input", nil)

	shell(ctx, t, e, "input", "keyevent", "KEYCODE_K")
	log, err := e.WaitLog(ctx, appID, "type=\"Key Up\"", 20*time.Second)
	if err != nil {
		t.Fatal(Describe(err))
	}
	var keys []string
	for _, l := range strings.Split(log, "\n") {
		if strings.Contains(l, "INFO key ") {
			keys = append(keys, l)
		}
	}
	// KeyK: the logical key 0x6b ('k'); the key that adb injects has no scan code, so the physical key
	// is the one of the Android plane with the key code (0x1100000000 | 39), as in Flutter.
	down := index(keys, `type="Key Down"`, "physical=0x1100000027", "logical=0x6b", `character="\"k\""`)
	up := index(keys, `type="Key Up"`, "physical=0x1100000027", "logical=0x6b")
	if down < 0 || up < down {
		t.Errorf("K is not a down and an up of KeyK:\n%s", strings.Join(keys, "\n"))
	}
}

// TestAKeyThatAShortcutHandlesIsTakenAndBackReachesTheActivity: in a Material page whose Shortcuts
// handle K, K runs the shortcut each time; in an app that handles no keys, Back, which Android
// delivers to the activity (the back callbacks of step 25.9 come later), closes the activity.
func TestAKeyThatAShortcutHandlesIsTakenAndBackReachesTheActivity(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	appID := inputApp(ctx, t, e, "./examples/dev/android-input", nil)
	shell(ctx, t, e, "input", "keyevent", "KEYCODE_K")
	if _, err := e.WaitLog(ctx, appID, "shortcut ran count=1", 20*time.Second); err != nil {
		t.Fatal(Describe(err))
	}
	shell(ctx, t, e, "input", "keyevent", "KEYCODE_K")
	if _, err := e.WaitLog(ctx, appID, "shortcut ran count=2", 20*time.Second); err != nil {
		t.Fatal(Describe(err))
	}
	time.Sleep(2 * time.Second)
	if alive, err := e.HasActivity(ctx, appID); err != nil || !alive {
		t.Errorf("K, which a shortcut handled, closed the activity (alive %v, %v)", alive, err)
	}

	appID = inputApp(ctx, t, e, "./examples/dev/input", nil)
	shell(ctx, t, e, "input", "keyevent", "KEYCODE_BACK")
	deadline := time.Now().Add(20 * time.Second)
	for {
		alive, err := e.HasActivity(ctx, appID)
		if err != nil {
			t.Fatal(Describe(err))
		}
		if !alive {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Back, which no widget handles, did not reach the activity: it is still there")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// TestDevEventsTapLikeTheUser runs the counter with the dev events script of three taps on the
// button and with three taps of `adb shell input`: both show 3, and the screens agree.
func TestDevEventsTapLikeTheUser(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	const fab = "964"
	tap := "tap:964,2284"
	script := tap + ";wait:400ms;" + tap + ";wait:400ms;" + tap

	screen := func(extras map[string]string, adbTaps bool) *image.RGBA {
		appID := inputApp(ctx, t, e, "./examples/counter", extras)
		if adbTaps {
			for range 3 {
				shell(ctx, t, e, "input", "tap", fab, "2284")
				time.Sleep(400 * time.Millisecond)
			}
		}
		time.Sleep(4 * time.Second)
		img := screenWithContent(ctx, t, e)
		_, _ = e.ADB.Output(ctx, "shell", "am", "force-stop", appID)
		return img
	}
	none := screen(nil, false)
	byScript := screen(map[string]string{"GLUTTER_DEV_EVENTS": script}, false)
	byAdb := screen(nil, true)
	if dir := os.Getenv("GLUTTER_ANDROIDTEST_DUMP"); dir != "" {
		for name, img := range map[string]*image.RGBA{"none": none, "adb": byAdb, "script": byScript} {
			_ = goldens.WritePNG(filepath.Join(dir, "devev-"+name+".png"), img)
		}
	}
	band := image.Rect(0, 1000, 1080, 1400) // the text and the count
	if gputest.DiffRect(none, byAdb, band).Over8 == 0 {
		t.Error("three taps of adb did not change the screen")
	}
	gputest.Tolerance{Over8Permille: 1}.Check(t, "dev events against adb taps", gputest.DiffRect(byAdb, byScript, band))
}

// TestASwipeScrollsTheScrollingExample sends a swipe to examples/scrolling and the page moves.
func TestASwipeScrollsTheScrollingExample(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	appID := inputApp(ctx, t, e, "./examples/scrolling", nil)
	grab := func() *image.RGBA {
		return screenWithContent(ctx, t, e)
	}
	before := grab()
	shell(ctx, t, e, "input", "swipe", "540", "1800", "540", "600", "300")
	time.Sleep(3 * time.Second)
	after := grab()
	band := image.Rect(0, 200, 1080, 2200)
	if d := gputest.DiffRect(before, after, band); d.Over32 < 20000 {
		t.Errorf("the page did not scroll: only %d pixels changed by more than 32 levels", d.Over32)
	}
	if alive, err := e.HasActivity(ctx, appID); err != nil || !alive {
		t.Errorf("the app is gone after the swipe: %v %v", alive, err)
	}
}

// screenWithContent takes the screen once something other than black is on it: the emulator shows
// a presented frame some time after the app presented it, on Vulkan sometimes seconds.
func screenWithContent(ctx context.Context, t *testing.T, e *Emulator) *image.RGBA {
	t.Helper()
	var img *image.RGBA
	for end := time.Now().Add(15 * time.Second); ; time.Sleep(500 * time.Millisecond) {
		b, err := e.ADB.Screencap(ctx)
		if err != nil {
			t.Fatal(Describe(err))
		}
		img = decodePNG(t, b)
		light := 0
		for i := 0; i < len(img.Pix); i += 4 {
			if int(img.Pix[i])+int(img.Pix[i+1])+int(img.Pix[i+2]) > 96 {
				light++
			}
		}
		if light > len(img.Pix)/4/100 || time.Now().After(end) {
			return img
		}
	}
}
