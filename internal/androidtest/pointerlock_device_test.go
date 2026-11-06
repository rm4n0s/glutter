//go:build glutter_android

package androidtest

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestPointerCaptureHoldsThePointer locks the pointer of examples/dev/input at the first event of the
// mouse (GLUTTER_DEV_LOCK=1) with `adb shell input mouse`, and checks that the system gave the app the
// pointer capture (View.requestPointerCapture, SPEC §27.6) and that the motion that the view gets as a
// captured pointer event reaches the framework as the delta of a pointer that stays where it was.
//
// `adb shell input` makes the events of a mouse with a place, which the system does not turn into the
// relative events of a captured mouse, so the relative motion comes from the dev events script
// (relmove), which calls View.onCapturedPointerEvent as the system does for a real mouse.
func TestPointerCaptureHoldsThePointer(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	appID := inputApp(ctx, t, e, "./examples/dev/input", map[string]string{
		"GLUTTER_DEV_LOCK":   "1",
		"GLUTTER_DEV_EVENTS": "wait:12s;relmove:7,-3;relmove:5,5",
	})

	shell(ctx, t, e, "input", "mouse", "tap", "300", "500")
	if _, err := e.WaitLog(ctx, appID, "INFO lock result=locked", 20*time.Second); err != nil {
		t.Fatal(Describe(err))
	}
	dump := shell(ctx, t, e, "dumpsys", "input")
	if !regexp.MustCompile(`Current Window with Pointer Capture: \S+ ` + regexp.QuoteMeta(appID) + `/`).MatchString(dump) {
		t.Errorf("the app's window has no pointer capture:\n%s", grepLines(dump, "Pointer Capture"))
	}

	log, err := e.WaitLog(ctx, appID, "dx=5 dy=5", 40*time.Second)
	if err != nil {
		t.Fatal(Describe(err))
	}
	lines := pointerLines(log)
	motion := regexp.MustCompile(`change=PointerChange(Hover|Move) kind=PointerDeviceKindMouse device=\d+ x=300 y=500 buttons=0 dx=(-?[\d.]+) dy=(-?[\d.]+)`)
	var got []string
	for _, l := range lines {
		if m := motion.FindStringSubmatch(l); m != nil {
			got = append(got, m[2]+","+m[3])
		}
	}
	if want := "7,-3 5,5"; strings.Join(got, " ") != want {
		t.Errorf("the locked pointer's motion was %q, want %q (at 300,500):\n%s", strings.Join(got, " "), want, strings.Join(lines, "\n"))
	}
}

func grepLines(s string, parts ...string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		for _, p := range parts {
			if strings.Contains(l, p) {
				out = append(out, l)
				break
			}
		}
	}
	return strings.Join(out, "\n")
}
