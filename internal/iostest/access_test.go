//go:build glutter_ios

package iostest_test

import (
	"context"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/iostest"
	"github.com/rm4n0s/glutter/internal/iostest/serviceslines"
	"github.com/rm4n0s/glutter/internal/platform/ios"
)

// waitLineAfter waits for a line of the app that contains the text and comes after the first n lines.
func waitLineAfter(t *testing.T, app *iostest.Launched, n int, text string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out := app.Output()
		for i := n; i < len(out); i++ {
			if strings.Contains(out[i], text) {
				return out[i]
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("no line with %q after the first %d lines:\n%s", text, n, strings.Join(app.Output(), "\n"))
	return ""
}

// TestSettingsFollowTheSystem changes the dark mode and the Dynamic Type size of the Simulator while the app runs, with `simctl ui`, and
// reads the settings that the app is told: the brightness, the text scale of the category (the body size over 17), the scaled size of a font
// (which grows by less than the scale for a large category: UIFontMetrics), and the switches (SPEC §30.7).
func TestSettingsFollowTheSystem(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./internal/iostest/serviceapp")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	// The device is shared with the other tests: its interface goes back to what it was.
	t.Cleanup(func() {
		c, stop := context.WithTimeout(context.Background(), time.Minute)
		defer stop()
		_ = h.Simctl.UI(c, dev.UDID, "appearance", "light")
		_ = h.Simctl.UI(c, dev.UDID, "content_size", "large")
	})
	if err := h.Simctl.UI(ctx, dev.UDID, "appearance", "light"); err != nil {
		t.Fatalf("appearance: %v", err)
	}
	if err := h.Simctl.UI(ctx, dev.UDID, "content_size", "large"); err != nil {
		t.Fatalf("content_size: %v", err)
	}
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, serviceslines.StepsEnv+"=settings")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	line, err := app.WaitFor(serviceslines.Settings, time.Minute)
	if err != nil {
		t.Fatalf("the first settings: %v", err)
	}
	if !strings.Contains(line, "brightness=light scale=1.0000 font17=17.00") || !strings.Contains(line, "bold=false") || !strings.Contains(line, "voiceover=false") {
		t.Errorf("the settings at the start: %q", line)
	}
	n := len(app.Output())
	if err := h.Simctl.UI(ctx, dev.UDID, "appearance", "dark"); err != nil {
		t.Fatalf("dark: %v", err)
	}
	if line := waitLineAfter(t, app, n, "brightness=dark", time.Minute); !strings.Contains(line, "scale=1.0000") {
		t.Errorf("dark mode: %q", line)
	}
	n = len(app.Output())
	if err := h.Simctl.UI(ctx, dev.UDID, "content_size", "accessibility-extra-extra-extra-large"); err != nil {
		t.Fatalf("content_size: %v", err)
	}
	line = waitLineAfter(t, app, n, "scale=3.1", time.Minute)
	scale, font := numberAfter(t, line, "scale="), numberAfter(t, line, "font17=")
	if math.Abs(scale-53.0/17) > 0.001 {
		t.Errorf("the scale at the largest size is %v, want %v", scale, 53.0/17)
	}
	if font < 17*1.5 || font > 17*scale+0.5 {
		t.Errorf("the scaled 17 point font is %v, want between 25.5 and %v", font, 17*scale)
	}
	n = len(app.Output())
	if err := h.Simctl.UI(ctx, dev.UDID, "content_size", "extra-small"); err != nil {
		t.Fatalf("content_size: %v", err)
	}
	line = waitLineAfter(t, app, n, "scale=0.8", time.Minute)
	if scale := numberAfter(t, line, "scale="); math.Abs(scale-14.0/17) > 0.001 {
		t.Errorf("the scale at the smallest size is %v, want %v", scale, 14.0/17)
	}
}

var numberRE = regexp.MustCompile(`-?\d+(\.\d+)?`)

// numberAfter is the number that follows the key in the line.
func numberAfter(t *testing.T, line, key string) float64 {
	t.Helper()
	i := strings.Index(line, key)
	if i < 0 {
		t.Fatalf("no %q in %q", key, line)
	}
	v, err := strconv.ParseFloat(numberRE.FindString(line[i+len(key):]), 64)
	if err != nil {
		t.Fatalf("the number after %q in %q: %v", key, line, err)
	}
	return v
}

// a11yElement is one "ios a11y element" line.
type a11yElement struct {
	label, value, hint string
	traits             uint64
	x, y, w, h         float64
}

// a11yElements are the elements that the app logged since the first n lines, one list for each "ios a11y elements" line.
func a11yElements(t *testing.T, app *iostest.Launched) [][]a11yElement {
	t.Helper()
	var lists [][]a11yElement
	for _, l := range app.Output() {
		switch {
		case strings.Contains(l, ios.A11yElementsLogLine):
			lists = append(lists, nil)
		case strings.Contains(l, ios.A11yElementLogLine) && len(lists) > 0:
			var e a11yElement
			e.label, e.value, e.hint = quotedAfter(l, "label="), quotedAfter(l, "value="), quotedAfter(l, "hint=")
			e.traits = uint64(numberAfter(t, l, "traits="))
			e.x, e.y, e.w, e.h = numberAfter(t, l, " x="), numberAfter(t, l, " y="), numberAfter(t, l, " w="), numberAfter(t, l, " h=")
			lists[len(lists)-1] = append(lists[len(lists)-1], e)
		}
	}
	return lists
}

// quotedAfter is the value of the key in a slog text line: a quoted string, or the word that follows the key when slog did not need quotes.
func quotedAfter(line, key string) string {
	i := strings.Index(line, " "+key)
	if i < 0 {
		return ""
	}
	rest := line[i+1+len(key):]
	if !strings.HasPrefix(rest, `"`) {
		if sp := strings.IndexByte(rest, ' '); sp >= 0 {
			return rest[:sp]
		}
		return rest
	}
	rest = rest[1:]
	var out []byte
	for j := 0; j < len(rest); j++ {
		switch {
		case rest[j] == '\\' && j+1 < len(rest):
			j++
			if rest[j] == 'n' {
				out = append(out, '\n')
			} else {
				out = append(out, rest[j])
			}
		case rest[j] == '"':
			return string(out)
		default:
			out = append(out, rest[j])
		}
	}
	return string(out)
}

func find(es []a11yElement, label string) (a11yElement, bool) {
	for _, e := range es {
		if e.label == label {
			return e, true
		}
	}
	return a11yElement{}, false
}

// UIKit's trait values (UIAccessibilityTraits): button 1, static text 64, header 65536.
const (
	traitButton     = 1
	traitSelected   = 8
	traitStaticText = 64
)

// TestAccessibilityTreeOfTheCounter reads the accessibility elements of the counter back from the view (what VoiceOver gets from
// accessibilityElements): the labels of its texts and of the button with their traits and frames on the screen, and then activates the
// button as VoiceOver's double tap does, and reads that the counter went up (SPEC §30.7).
func TestAccessibilityTreeOfTheCounter(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./examples/counter")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, "GLUTTER_DEV_EVENTS=wait:1500ms;a11y;a11y-activate:Increment;wait:800ms;a11y")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if _, err := app.WaitFor(ios.A11yActivateLogLine, time.Minute); err != nil {
		t.Fatalf("the activation: %v", err)
	}
	waitLineAfter(t, app, 0, "ios a11y elements", time.Minute)
	deadline := time.Now().Add(time.Minute)
	for len(a11yElements(t, app)) < 2 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	lists := a11yElements(t, app)
	if len(lists) < 2 {
		t.Fatalf("the app listed its elements %d times, want 2:\n%s", len(lists), strings.Join(app.Output(), "\n"))
	}
	before, after := lists[0], lists[1]
	fab, ok := find(before, "Increment")
	if !ok || fab.traits&traitButton == 0 {
		t.Errorf("the button: %+v (found %v) in %+v", fab, ok, before)
	}
	if fab.w <= 0 || fab.h <= 0 || fab.x < 0 || fab.y < 0 {
		t.Errorf("the button's frame on the screen: %+v", fab)
	}
	if e, ok := find(before, "You have pushed the button this many times:"); !ok || e.traits&traitStaticText == 0 {
		t.Errorf("the text: %+v (found %v)", e, ok)
	}
	if _, ok := find(before, "0"); !ok {
		t.Errorf("the counter's 0 is not an element: %+v", before)
	}
	act := ""
	for _, l := range app.Output() {
		if strings.Contains(l, ios.A11yActivateLogLine) {
			act = l
		}
	}
	if !strings.Contains(act, "found=true") || !strings.Contains(act, "activated=true") {
		t.Errorf("the activation: %q", act)
	}
	if _, ok := find(after, "1"); !ok {
		t.Errorf("after the activation the counter shows no 1: %+v", after)
	}
	if _, ok := find(after, "0"); ok {
		t.Errorf("after the activation the counter still has its 0: %+v", after)
	}
}

// TestAccessibilityTreeOfTheGallery reads the elements of the gallery's first page back from the view: its title and the heading of the
// page are there, there are many more, and every frame is on the screen of the phone (SPEC §30.7).
func TestAccessibilityTreeOfTheGallery(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, "./examples/gallery")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, "GLUTTER_DEV_EVENTS=wait:3s;a11y")
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	waitLineAfter(t, app, 0, ios.A11yElementsLogLine, 2*time.Minute)
	time.Sleep(time.Second)
	lists := a11yElements(t, app)
	if len(lists) != 1 {
		t.Fatalf("the elements were listed %d times", len(lists))
	}
	es := lists[0]
	if len(es) < 10 {
		t.Fatalf("the gallery has %d elements, want at least 10: %+v", len(es), es)
	}
	if _, ok := find(es, "Glutter gallery"); !ok {
		t.Errorf("no element labeled Glutter gallery in %+v", es)
	}
	// The heading and the text under it are one element: a text with its description is read as one.
	heading := false
	for _, e := range es {
		heading = heading || strings.HasPrefix(e.label, "Material 3 in Glutter")
	}
	if !heading {
		t.Errorf("no element for the heading of the page in %+v", es)
	}
	// The selected segment of the mode is a selected button (traits 1 and 8).
	if e, ok := find(es, "Light"); !ok || e.traits != traitButton|traitSelected {
		t.Errorf("the selected segment: %+v (found %v)", e, ok)
	}
	for _, e := range es {
		if e.w <= 0 || e.h <= 0 || e.x < -1 || e.x+e.w > 500 {
			t.Errorf("the frame of %q is off the screen: %+v", e.label, e)
		}
	}
}
