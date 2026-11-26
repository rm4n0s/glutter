//go:build glutter_android

package androidtest

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/issues"
)

// TestAndroidHoversOnlyWhenThePointerMoves is the issue "A hover for a motion event that did not move
// the pointer" (step 58.1) on the emulator: the dev events script of examples/dev/input (which logs
// every pointer data the framework gets) moves the mouse to the places of the reproduction, most of
// them where it already is, as hover moves of a mouse that the view gets from its own dispatch (`adb
// shell input` makes no hover: its mouse has no hover move, and the system's input reader makes no
// event for a motion that does not move), and the framework gets a hover only where the mouse went.
// Android runs its pointer data through the engine's converter (internal/platform/pointerconv),
// which drops a hover whose place did not change, as Flutter's does.
func TestAndroidHoversOnlyWhenThePointerMoves(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	steps := issues.StillHoverSteps()
	script := []string{"wait:3s", hoverStep(steps[0])}
	for _, s := range steps {
		script = append(script, hoverStep(s))
	}
	appID := inputApp(ctx, t, e, "./examples/dev/input", map[string]string{"GLUTTER_DEV_EVENTS": strings.Join(script, ";")})

	last := steps[len(steps)-1]
	log, err := e.WaitLog(ctx, appID, "x="+strconv.Itoa(int(last.X))+" y="+strconv.Itoa(int(last.Y)), 30*time.Second)
	if err != nil {
		t.Logf("the pointer lines:\n%s", strings.Join(pointerLines(log), "\n"))
		t.Fatal(Describe(err))
	}
	time.Sleep(2 * time.Second) // the rest of the script
	if log, err = e.Log(ctx, appID); err != nil {
		t.Fatal(Describe(err))
	}
	rep := stillHoverReportOfLog(t, log)
	for _, p := range rep.Problems() {
		t.Error(p)
	}
	if t.Failed() {
		t.Logf("the report:\n%s\nthe pointer lines:\n%s", strings.Join(rep.Lines(), "\n"), strings.Join(pointerLines(log), "\n"))
	}
}

// hoverStep is the step of the dev events script that moves the mouse to the place.
func hoverStep(p issues.StillHoverPoint) string {
	return "hover:" + strconv.Itoa(int(p.X)) + "," + strconv.Itoa(int(p.Y))
}

// stillHoverReportOfLog makes the report of the reproduction from the pointer lines of the log of
// examples/dev/input: its hovers of the mouse, with their places and deltas.
func stillHoverReportOfLog(t *testing.T, log string) issues.StillHoverReport {
	t.Helper()
	var rep issues.StillHoverReport
	for _, l := range pointerLines(log) {
		if !strings.Contains(l, "change=PointerChangeHover kind=PointerDeviceKindMouse") || strings.Contains(l, "scroll") {
			continue
		}
		var h issues.StillHoverEvent
		for _, f := range strings.Fields(l) {
			k, v, ok := strings.Cut(f, "=")
			if !ok {
				continue
			}
			n, err := strconv.ParseFloat(v, 64)
			if err != nil {
				continue
			}
			switch k {
			case "x":
				h.X = n
			case "y":
				h.Y = n
			case "dx":
				h.DX = n
			case "dy":
				h.DY = n
			}
		}
		rep.Hovers = append(rep.Hovers, h)
	}
	return rep
}
