//go:build glutter_ios

package iostest_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/iostest"
	"github.com/rm4n0s/glutter/internal/iostest/behaviorlines"
)

const behaviorPackage = "./internal/iostest/behaviorapp"

// launchBehavior installs the behavior app and starts it in the mode, with the script of events.
func launchBehavior(t *testing.T, mode, events string) *iostest.Launched {
	t.Helper()
	h := iostest.New(t)
	dev, res := installed(t, h, behaviorPackage)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)
	app, err := h.Launch(ctx, dev.UDID, res.Info.ID, behaviorlines.ModeEnv+"="+mode, "GLUTTER_DEV_EVENTS="+events)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	return app
}

// offsets are the scroll offsets that the app printed, in order.
func offsets(app *iostest.Launched) []float64 {
	var out []float64
	for _, l := range app.Output() {
		if i := strings.Index(l, behaviorlines.Offset+" "); i >= 0 {
			if v, err := strconv.ParseFloat(strings.TrimSpace(l[i+len(behaviorlines.Offset):]), 64); err == nil {
				out = append(out, v)
			}
		}
	}
	return out
}

// TestListBouncesAtTheTop drags a finger down over a list that is at its start: with the physics of iOS the list goes past its start (a
// negative offset: the bounce) and settles back at 0 when the finger is up (SPEC §30.10, §31: BouncingScrollPhysics).
func TestListBouncesAtTheTop(t *testing.T) {
	app := launchBehavior(t, "scroll", "wait:1s;drag:200,300,200,700,300ms")
	if _, err := app.WaitFor(behaviorlines.Offset+" -", 2*time.Minute); err != nil {
		t.Fatalf("the list did not go past its start: %v", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		o := offsets(app)
		neg := false
		for i, v := range o {
			if v < 0 {
				neg = true
			}
			if neg && i == len(o)-1 && v == 0 {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("the list did not settle back at 0: %v", o)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// TestListScrollsWithTheFinger drags the list up by 400 points: it scrolled by a few hundred points.
func TestListScrollsWithTheFinger(t *testing.T) {
	app := launchBehavior(t, "scroll", "wait:1s;drag:200,700,200,300,200ms")
	if _, err := app.WaitFor(behaviorlines.Offset+" 3", 2*time.Minute); err != nil {
		t.Fatalf("the list did not scroll up by a few hundred points: %v", err)
	}
}

// TestBackSwipeFromTheLeftEdgePopsThePage starts with a page over the home page and swipes from the left edge to the right, as the back
// gesture of iOS: the page is popped (SPEC §30.10: the Cupertino page transition with the back swipe).
func TestBackSwipeFromTheLeftEdgePopsThePage(t *testing.T) {
	app := launchBehavior(t, "back", "wait:2s;drag:2,400,330,400,300ms")
	if _, err := app.WaitFor(behaviorlines.Pushed, 2*time.Minute); err != nil {
		t.Fatalf("the page was not pushed: %v", err)
	}
	if _, err := app.WaitFor(behaviorlines.Popped, time.Minute); err != nil {
		t.Fatalf("the swipe from the left edge did not pop the page: %v", err)
	}
}

// TestTheMiddleOfThePageDoesNotGoBack drags from the middle of the page, not from the edge: nothing is popped.
func TestTheMiddleOfThePageDoesNotGoBack(t *testing.T) {
	app := launchBehavior(t, "back", "wait:2s;drag:150,400,330,400,300ms;wait:2s")
	if _, err := app.WaitFor(behaviorlines.Pushed, 2*time.Minute); err != nil {
		t.Fatalf("the page was not pushed: %v", err)
	}
	time.Sleep(6 * time.Second)
	if n := count(app, behaviorlines.Popped); n != 0 {
		t.Errorf("a drag from the middle popped the page")
	}
}

// selections are the selections that the field printed, in order, as "<base> <extent>".
func selections(app *iostest.Launched) []string {
	var out []string
	for _, l := range app.Output() {
		if i := strings.Index(l, behaviorlines.Selection+" "); i >= 0 {
			out = append(out, strings.TrimSpace(l[i+len(behaviorlines.Selection):]))
		}
	}
	return out
}

// TestDoubleTapSelectsAWord taps twice on the first word of the field's text: the word is selected (SPEC §30.10: the Cupertino text
// field's selection behavior on iOS).
func TestDoubleTapSelectsAWord(t *testing.T) {
	app := launchBehavior(t, "select", "wait:1s;tap:50,100;tap:50,100")
	if _, err := app.WaitFor(behaviorlines.Selection+" 0 6", time.Minute); err != nil {
		t.Fatalf("the word was not selected: %v (selections %v)", err, selections(app))
	}
}

// TestLongPressSelectsAWordAndTheDragExtendsIt presses the field's first word and holds it while the finger moves along the text, slowly
// enough that the press is a long one: the word is selected and the selection grows with the finger, as on iOS (the framework shows the
// magnifier while the finger is down, which only a person can see: MANUAL-CHECKS 35.9).
func TestLongPressSelectsAWordAndTheDragExtendsIt(t *testing.T) {
	app := launchBehavior(t, "select", "wait:1s;drag:40,100,120,100,3000ms")
	if _, err := app.WaitFor(behaviorlines.Selection+" 0 6", time.Minute); err != nil {
		t.Fatalf("the long press did not select the word: %v (selections %v)", err, selections(app))
	}
	time.Sleep(4 * time.Second)
	got := selections(app)
	if len(got) < 3 {
		t.Fatalf("the selection changed %d times (%v), want several", len(got), got)
	}
	var base, extent int
	if _, err := fmt.Sscanf(got[len(got)-1], "%d %d", &base, &extent); err != nil || base != 0 || extent <= 6 {
		t.Errorf("the last selection is %q, want the word and more of the text", got[len(got)-1])
	}
}

// TestSelectionShowsHandlesAndTheToolbar draws the field's first frame after a double tap and compares it with the frame of the field
// with no selection: the selection's highlight, its two handles and the Cupertino toolbar over it make a visible difference.
func TestSelectionShowsHandlesAndTheToolbar(t *testing.T) {
	h := iostest.New(t)
	dev, res := installed(t, h, behaviorPackage)
	_, plain := capture(t, h, dev, res.Info.ID, "soft", behaviorlines.ModeEnv+"=select")
	_, selected := capture(t, h, dev, res.Info.ID, "soft", behaviorlines.ModeEnv+"=select", "GLUTTER_DEV_EVENTS=wait:1s;tap:50,100;tap:50,100;wait:1s")
	if d := differs(plain, selected); d < 0.005 {
		t.Errorf("only %.3f%% of the pixels differ after the double tap: no handles or toolbar are drawn", d*100)
	}
}
