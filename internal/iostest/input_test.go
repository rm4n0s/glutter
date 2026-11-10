//go:build glutter_ios

package iostest_test

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/iostest"
	"github.com/rm4n0s/glutter/internal/iostest/inputlines"
	"github.com/rm4n0s/glutter/internal/iostest/scenelines"
)

// TestScriptedTouchesReachTheFramework runs the input app with a script of a tap and a drag (GLUTTER_DEV_EVENTS: the headless Simulator
// has no way to send a real touch, so the script gives its touches to the view's conversion, the one that real touches take): the framework
// gets a tap down from a touch at the place, and a pan that goes as far as the drag did (SPEC §30.4, §30.11).
func TestScriptedTouchesReachTheFramework(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./internal/iostest/inputapp")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, "GLUTTER_DEV_EVENTS=tap:100,200;wait:200ms;drag:50,300,50,100,160ms")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	line, err := app.WaitFor(inputlines.Tap, time.Minute)
	if err != nil {
		t.Fatalf("the tap: %v", err)
	}
	if !strings.HasSuffix(line, "100 200 PointerDeviceKindTouch") {
		t.Errorf("tap line = %q, want the place 100,200 from a touch", line)
	}
	line, err = app.WaitFor(inputlines.Pan, time.Minute)
	if err != nil {
		t.Fatalf("the pan: %v", err)
	}
	var dx, dy int
	if _, err := fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(line[strings.Index(line, inputlines.Pan):], inputlines.Pan)), "%d %d", &dx, &dy); err != nil {
		t.Fatalf("the pan line %q: %v", line, err)
	}
	// The pan starts after the framework's slop, so it is a little shorter than the 200 points of the drag.
	if dy > -150 || dy < -200 || dx < -2 || dx > 2 {
		t.Errorf("the pan went (%d, %d), want about (0, -180)", dx, dy)
	}
}

// TestScriptedKeysReachTheFramework runs the input app with a script of keys of a hardware keyboard: the framework gets the physical and
// the logical key of each, the character of a letter (upper case with Shift), the modifier keys pressed before and released after, and the
// release of each key.
func TestScriptedKeysReachTheFramework(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./internal/iostest/inputapp")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, "GLUTTER_DEV_EVENTS=key:KeyA;key:KeyA,shift;key:ArrowLeft;key:KeyC,meta;key:Enter")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if _, err := app.WaitFor(inputlines.Key+` up Enter`, time.Minute); err != nil {
		t.Fatalf("the last key: %v", err)
	}
	var keys []string
	for _, l := range app.Output() {
		if i := strings.Index(l, inputlines.Key); i >= 0 {
			keys = append(keys, strings.TrimSpace(strings.TrimPrefix(l[i:], inputlines.Key)))
		}
	}
	want := []string{
		`down Key A Key A "a"`, `up Key A Key A ""`,
		`down Shift Left Shift Left ""`, `down Key A Key A "A"`, `up Key A Key A ""`, `up Shift Left Shift Left ""`,
		`down Arrow Left Arrow Left ""`, `up Arrow Left Arrow Left ""`,
		`down Meta Left Meta Left ""`, `down Key C Key C "c"`, `up Key C Key C ""`, `up Meta Left Meta Left ""`,
		`down Enter Enter "\r"`, `up Enter Enter ""`,
	}
	if strings.Join(keys, "\n") != strings.Join(want, "\n") {
		t.Errorf("keys:\n%s\nwant:\n%s", strings.Join(keys, "\n"), strings.Join(want, "\n"))
	}
}

// differs counts the share of the pixels of two frames of the same size that differ.
func differs(a, b *image.RGBA) float64 {
	if a.Bounds() != b.Bounds() {
		return 1
	}
	n := 0
	for i := 0; i < len(a.Pix); i += 4 {
		if a.Pix[i] != b.Pix[i] || a.Pix[i+1] != b.Pix[i+1] || a.Pix[i+2] != b.Pix[i+2] {
			n++
		}
	}
	return float64(n) / float64(len(a.Pix)/4)
}

// TestTapOnTheCounterCounts taps the counter's button with a script and compares the frame with that of a counter that starts at 1: the
// tap reached the button and the counter shows one.
func TestTapOnTheCounterCounts(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./examples/counter")
	_, tapped := capture(t, h, dev, res.Info.ID, "soft", "GLUTTER_DEV_EVENTS=tap:358,795")
	_, started := capture(t, h, dev, res.Info.ID, "soft", "GLUTTER_DEV_TAPS=1")
	_, zero := capture(t, h, dev, res.Info.ID, "soft")
	if d := differs(tapped, started); d > 0.001 {
		t.Errorf("the frame after the tap differs from the frame of a counter at 1 in %.2f%% of the pixels", d*100)
	}
	if d := differs(tapped, zero); d < 0.0005 {
		t.Errorf("the tap changed nothing: %.3f%% of the pixels differ from the counter at 0", d*100)
	}
}

// TestDragScrollsTheList drags a finger up over examples/scrolling's list with a script: the list moved.
func TestDragScrollsTheList(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./examples/scrolling")
	_, still := capture(t, h, dev, res.Info.ID, "soft")
	_, moved := capture(t, h, dev, res.Info.ID, "soft", "GLUTTER_DEV_EVENTS=drag:200,700,200,300,200ms")
	if d := differs(still, moved); d < 0.005 {
		t.Errorf("only %.2f%% of the pixels differ after the drag: the list did not scroll", d*100)
	}
}

// TestScriptedTextReachesTheField runs the field app with a script that types text through the text input view of UIKit (insertText:,
// and the setMarkedText:selectedRange: of an input method that composes Japanese and commits it): the framework's field gets the text,
// the composing states on the way and the commit, an emoji of two UTF-16 units included (SPEC §30.5, §30.11).
func TestScriptedTextReachesTheField(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./internal/iostest/fieldapp")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, "GLUTTER_DEV_EVENTS=wait:500ms;text:Hi 😀;compose:にほ,にほん,日本")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if _, err := app.WaitFor(inputlines.Field+` "Hi 😀日本"`, time.Minute); err != nil {
		t.Fatalf("the committed text: %v; output:\n%s", err, strings.Join(app.Output(), "\n"))
	}
	var fields []string
	for _, l := range app.Output() {
		if i := strings.Index(l, inputlines.Field); i >= 0 {
			fields = append(fields, strings.TrimSpace(strings.TrimPrefix(l[i:], inputlines.Field)))
		}
	}
	for _, want := range []string{`"H"`, `"Hi"`, `"Hi 😀"`, `"Hi 😀にほ"`, `"Hi 😀にほん"`, `"Hi 😀日本"`} {
		found := false
		for _, f := range fields {
			found = found || f == want
		}
		if !found {
			t.Errorf("the field never had the text %s; it had %v", want, fields)
		}
	}
}

// TestKeyboardInsetsFollowTheKeyboard runs the scene app with a script that brings the keyboard up 300 points and takes it down: UIKit's
// notification gives the view an inset of 300 points in pixels, animated, and the inset goes back to nothing (SPEC §30.3).
func TestKeyboardInsetsFollowTheKeyboard(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./internal/iostest/sceneapp")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, "GLUTTER_DEV_EVENTS=wait:300ms;keyboard:300;wait:700ms;keyboardhide")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	// The scale of the phone says what 300 points are.
	first, err := app.WaitFor(scenelines.Metrics, time.Minute)
	if err != nil {
		t.Fatalf("the first metrics: %v", err)
	}
	var scale float64
	if i := strings.Index(first, "scale="); i < 0 {
		t.Fatalf("no scale in %q", first)
	} else if _, err := fmt.Sscanf(first[i:], "scale=%g", &scale); err != nil {
		t.Fatalf("the scale of %q: %v", first, err)
	}
	full := fmt.Sprintf("insets=0,0,0,%g", 300*scale)
	if _, err := app.WaitFor(full, time.Minute); err != nil {
		t.Fatalf("the full inset %s: %v", full, err)
	}
	// It got there by steps, and went back to nothing afterwards.
	deadline := time.Now().Add(time.Minute)
	last := ""
	for time.Now().Before(deadline) {
		out := app.Output()
		for i := len(out) - 1; i >= 0; i-- {
			if strings.Contains(out[i], scenelines.Metrics) {
				last = out[i]
				break
			}
		}
		if strings.HasSuffix(last, "insets=0,0,0,0") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !strings.HasSuffix(last, "insets=0,0,0,0") {
		t.Fatalf("the inset did not go back to nothing; the last metrics are %q", last)
	}
	steps := 0
	for _, l := range app.Output() {
		if strings.Contains(l, scenelines.Metrics) && strings.Contains(l, "insets=0,0,0,") && !strings.HasSuffix(l, "insets=0,0,0,0") && !strings.HasSuffix(l, full) {
			steps++
		}
	}
	if steps < 3 {
		t.Errorf("the inset went up and down in %d steps between nothing and the full inset, want an animation of several", steps)
	}
}

// TestSoftKeyboardGrowsTheInsets runs the field app, whose field takes the focus: the soft keyboard of the Simulator comes up (UIKit's own,
// shown by the text input view's becomeFirstResponder), and the bottom inset that the framework gets is the keyboard's height (SPEC
// §30.3, §30.5).
func TestSoftKeyboardGrowsTheInsets(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./internal/iostest/fieldapp")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if _, err := app.WaitFor(inputlines.Insets+" 0", time.Minute); err != nil {
		t.Fatalf("the first inset: %v", err)
	}
	deadline := time.Now().Add(time.Minute)
	best := 0
	for time.Now().Before(deadline) && best < 200 {
		for _, l := range app.Output() {
			var n int
			if i := strings.Index(l, inputlines.Insets); i >= 0 {
				if _, err := fmt.Sscanf(strings.TrimSpace(l[i+len(inputlines.Insets):]), "%d", &n); err == nil && n > best {
					best = n
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	// A phone's soft keyboard is about 300 points high (more on a Pro Max, less with a compact one): over 200 is the keyboard.
	if best < 200 {
		if dir := os.Getenv("GLUTTER_IOS_PROOF_DIR"); dir != "" {
			_ = h.Simctl.Screenshot(ctx, dev.UDID, filepath.Join(dir, "keyboard-failed.png"))
		}
		t.Errorf("the largest inset was %d points: the keyboard did not come up; output:\n%s", best, strings.Join(app.Output(), "\n"))
	}
}
