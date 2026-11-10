//go:build glutter_ios

package iostest_test

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/iostest"
	"github.com/rm4n0s/glutter/internal/iostest/serviceslines"
)

// runServices runs the services app with the steps and returns it when it has printed its last line.
func runServices(t *testing.T, h *iostest.Harness, steps string) (*iostest.Launched, string) {
	t.Helper()
	dev, res := installed(t, h, "./internal/iostest/serviceapp")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, serviceslines.StepsEnv+"="+steps)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("the app's output:\n%s", strings.Join(app.Output(), "\n"))
		}
	})
	if _, err := app.WaitFor(serviceslines.Done, 2*time.Minute); err != nil {
		t.Fatalf("the steps did not end: %v", err)
	}
	return app, dev.UDID
}

// stepLines are the results of the steps that the app printed, in order, each as "<step> <result>".
func stepLines(app *iostest.Launched) []string {
	var out []string
	for _, l := range app.Output() {
		if i := strings.Index(l, serviceslines.Step); i >= 0 {
			out = append(out, strings.TrimSpace(l[i+len(serviceslines.Step):]))
		}
	}
	return out
}

// stepResult is the result of the first step with that name, and false when it did not run.
func stepResult(app *iostest.Launched, step string) (string, bool) {
	for _, l := range stepLines(app) {
		if strings.HasPrefix(l, step+" ") {
			return strings.TrimPrefix(l, step+" "), true
		}
	}
	return "", false
}

// chromeState is one "ios chrome" line of the platform: what UIKit had of the system UI.
type chromeState struct {
	style                    int
	statusHidden, homeHidden bool
	orientations             int
}

var chromeRE = regexp.MustCompile(`statusBarStyle=(\d+) statusBarHidden=(\w+) homeIndicatorHidden=(\w+) orientations=(\d+)`)

func chromeStates(t *testing.T, app *iostest.Launched) []chromeState {
	t.Helper()
	var out []chromeState
	for _, l := range app.Output() {
		if m := chromeRE.FindStringSubmatch(l); m != nil && strings.Contains(l, "ios chrome") {
			style, _ := strconv.Atoi(m[1])
			orient, _ := strconv.Atoi(m[4])
			out = append(out, chromeState{style, m[2] == "true", m[3] == "true", orient})
		}
	}
	return out
}

// TestSystemUIFollowsTheApp asks for a dark and a light status bar, hidden bars and edge to edge, and the landscape orientations and
// every orientation, and reads what UIKit has after each (the status bar manager of the scene, the view controller): the style of the
// status bar is light content for a dark bar and dark content for a light one, the bars hide and come back, and the view controller
// supports the orientations that the app asked for (SPEC §30.3).
func TestSystemUIFollowsTheApp(t *testing.T) {
	h := iostest.New(t)
	app, _ := runServices(t, h, "report;style:dark;report;style:light;report;mode:hidden;report;mode:edge;report;orient:landscape;report;orient:all;report")
	got := chromeStates(t, app)
	if len(got) != 7 {
		t.Fatalf("the app reported %d states, want 7:\n%s", len(got), strings.Join(app.Output(), "\n"))
	}
	want := []struct {
		name  string
		check func(c chromeState) bool
	}{
		{"every orientation to begin with", func(c chromeState) bool { return c.orientations == 30 && !c.statusHidden }},
		{"a dark status bar has light content", func(c chromeState) bool { return c.style == 1 }},
		{"a light status bar has dark content", func(c chromeState) bool { return c.style == 3 }},
		{"hidden bars", func(c chromeState) bool { return c.statusHidden && c.homeHidden }},
		{"edge to edge shows them again", func(c chromeState) bool { return !c.statusHidden && !c.homeHidden }},
		{"landscape", func(c chromeState) bool { return c.orientations == 24 }},
		{"every orientation again", func(c chromeState) bool { return c.orientations == 30 }},
	}
	for i, w := range want {
		if !w.check(got[i]) {
			t.Errorf("%s: state %d is %+v", w.name, i, got[i])
		}
	}
}

// TestClipboardRoundTrip copies text from the app and reads it back, and reads what the app copied with simctl (which has its own view of
// the Simulator's pasteboard): the copy reaches UIPasteboard and the paste returns what it holds (SPEC §30.6).
func TestClipboardRoundTrip(t *testing.T) {
	h := iostest.New(t)
	app, udid := runServices(t, h, "copy:Hello from Glutter;paste;copy:second;paste")
	lines := stepLines(app)
	var copies, pastes []string
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, "copy:"):
			copies = append(copies, l)
		case strings.HasPrefix(l, "paste "):
			pastes = append(pastes, l)
		}
	}
	if len(copies) != 2 || !strings.HasSuffix(copies[0], "err=nil") || !strings.HasSuffix(copies[1], "err=nil") {
		t.Errorf("the copies: %q", copies)
	}
	if len(pastes) != 2 || !strings.Contains(pastes[0], `ok=true text="Hello from Glutter" err=nil`) || !strings.Contains(pastes[1], `ok=true text="second" err=nil`) {
		t.Errorf("the pastes: %q", pastes)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	// The Simulator's pasteboard reaches simctl a moment after the app wrote it, so the read waits for it.
	var out string
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(500 * time.Millisecond) {
		var err error
		if out, err = h.Simctl.Pbpaste(ctx, udid); err != nil {
			t.Fatalf("pbpaste: %v", err)
		}
		if strings.TrimSpace(out) == "second" || time.Now().After(deadline) {
			break
		}
	}
	if strings.TrimSpace(out) != "second" {
		t.Errorf("simctl pbpaste = %q, want the last text the app copied", out)
	}
}

// TestSpellCheckFindsMisspelledWords asks UITextChecker for the misspelled words of a sentence, with the spans in bytes and the first
// guess, and for a language it has no dictionary for (SPEC §30.6).
func TestSpellCheckFindsMisspelledWords(t *testing.T) {
	h := iostest.New(t)
	app, _ := runServices(t, h, "spell:en-US:This is a mispelled wrod today;spell:en-US:All words here are right;spell:xx-YY:anything")
	lines := stepLines(app)
	var spells []string
	for _, l := range lines {
		if strings.HasPrefix(l, "spell:") {
			spells = append(spells, l)
		}
	}
	if len(spells) != 3 {
		t.Fatalf("the spell steps: %q", spells)
	}
	// "This is a " is 10 bytes, "mispelled" 9, a blank, "wrod" 4.
	if !strings.Contains(spells[0], "spans=10,19,misspelled;20,24,") || !strings.HasSuffix(spells[0], "err=nil") {
		t.Errorf("the misspelled text: %q", spells[0])
	}
	if !strings.Contains(spells[1], "spans= err=nil") {
		t.Errorf("a correct text: %q", spells[1])
	}
	if !strings.HasSuffix(spells[2], "err=SpellCheckLanguageUnavailable") {
		t.Errorf("an unknown language: %q", spells[2])
	}
}

// TestHapticsSoundsAndOpenURI gives every haptic feedback and system sound (the Simulator has no Taptic Engine and no way to say they
// were made: the app must go on), refuses a script URI, and fails to open a scheme that no app handles.
func TestHapticsSoundsAndOpenURI(t *testing.T) {
	h := iostest.New(t)
	app, _ := runServices(t, h, "haptic;sound;open:javascript:alert(1);open:nosuchscheme-glutter://x")
	for _, step := range []string{"haptic", "sound"} {
		if r, ok := stepResult(app, step); !ok || r != "done" {
			t.Errorf("%s: %q, %v", step, r, ok)
		}
	}
	if r, _ := stepResult(app, "open:javascript:alert(1)"); r != "err=OpenURISchemeBlocked" {
		t.Errorf("a script URI: %q", r)
	}
	if r, _ := stepResult(app, "open:nosuchscheme-glutter://x"); r != "err=OpenURIUnavailable" {
		t.Errorf("a scheme without an app: %q", r)
	}
	if app.Ended() {
		t.Error("the app ended")
	}
}

// TestNotificationAsksForThePermission shows a notification: the system's permission prompt cannot be answered in a headless Simulator, so
// the call ends with its context as NotificationsUnavailable, or succeeds if the permission was granted before.
func TestNotificationAsksForThePermission(t *testing.T) {
	h := iostest.New(t)
	app, _ := runServices(t, h, "notify")
	r, ok := stepResult(app, "notify")
	if !ok || (r != "err=nil" && r != "err=NotificationsUnavailable") {
		t.Errorf("notify: %q, %v", r, ok)
	}
}

// TestPbcopyOfAMissingDeviceFails is the route of the error of Simctl.Pbcopy: simctl says no for a device that does not exist.
func TestPbcopyOfAMissingDeviceFails(t *testing.T) {
	h := iostest.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := h.Simctl.Pbcopy(ctx, "00000000-0000-0000-0000-000000000000", "x")
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("Simctl.Pbcopy.IOSSimctlFailed") {
		t.Errorf("error %v, want IOSSimctlFailed from Simctl.Pbcopy", err)
	}
}
