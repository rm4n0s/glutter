//go:build glutter_android

package androidtest

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

// backApp installs examples/dev/android-back, which logs the lifecycle, the back button and the back
// gesture, and starts it with the renderer soft, the page to start on (GLUTTER_DEMO) and the dev events
// script. It returns once the app has presented its first frame.
func backApp(ctx context.Context, t *testing.T, e *Emulator, demo, script string) string {
	t.Helper()
	appID := installApp(ctx, t, e, "./examples/dev/android-back")
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	extras := map[string]string{"GLUTTER_RENDERER": "soft", "GLUTTER_DEMO": demo}
	if script != "" {
		extras["GLUTTER_DEV_EVENTS"] = script
	}
	if err := e.Launch(ctx, appID, extras); err != nil {
		t.Fatal(Describe(err))
	}
	if _, err := e.WaitLog(ctx, appID, "first frame presented", time.Minute); err != nil {
		t.Fatal(Describe(err))
	}
	return appID
}

// bringBack starts the launcher activity of the app again, as the launcher icon does.
func bringBack(ctx context.Context, t *testing.T, e *Emulator, appID string) {
	t.Helper()
	shell(ctx, t, e, "monkey", "-p", appID, "-c", "android.intent.category.LAUNCHER", "1")
}

// lines are the lines of the log that contain want, in order.
func lines(log, want string) []string {
	var out []string
	for _, l := range strings.Split(log, "\n") {
		if strings.Contains(l, want) {
			out = append(out, l)
		}
	}
	return out
}

const (
	lcPaused  = "lifecycle state=AppLifecycleStatePaused"
	lcResumed = "lifecycle state=AppLifecycleStateResumed"
)

// TestHomeAndTheReturnToTheAppRunTheLifecycle presses Home, which stops the activity (the app is
// paused, through inactive and hidden), and starts the app again from the launcher (resumed).
func TestHomeAndTheReturnToTheAppRunTheLifecycle(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := backApp(ctx, t, e, "", "")
		if _, err := e.WaitLog(ctx, appID, lcResumed, 30*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
		shell(ctx, t, e, "input", "keyevent", "KEYCODE_HOME")
		log, err := e.WaitLog(ctx, appID, lcPaused, 30*time.Second)
		if err != nil {
			t.Fatal(Describe(err))
		}
		for _, want := range []string{"lifecycle state=AppLifecycleStateInactive", "lifecycle state=AppLifecycleStateHidden"} {
			if len(lines(log, want)) == 0 {
				t.Errorf("the log has no %q:\n%s", want, strings.Join(lines(log, "lifecycle"), "\n"))
			}
		}
		before := len(lines(log, lcResumed))
		bringBack(ctx, t, e, appID)
		for end := time.Now().Add(30 * time.Second); ; time.Sleep(300 * time.Millisecond) {
			log, err = e.Log(ctx, appID)
			if err != nil {
				t.Fatal(Describe(err))
			}
			if len(lines(log, lcResumed)) > before {
				break
			}
			if time.Now().After(end) {
				t.Fatalf("the app was not resumed again:\n%s", strings.Join(lines(log, "lifecycle"), "\n"))
			}
		}
	})
}

// TestBackPopsTheRouteOfTheNavigator sends Back through the activity as the system does (the callback
// of the activity, or its onBackPressed) and sees the navigator pop the second page.
func TestBackPopsTheRouteOfTheNavigator(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := backApp(ctx, t, e, "second", "wait:2s;back:pop")
		if _, err := e.WaitLog(ctx, appID, "navigator event=pop route=/second", 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
		pid := pidOf(ctx, t, e, appID)
		if pid == 0 {
			t.Error("the app has no process")
		}
	})
}

// TestTheBackKeyPopsTheRouteOfTheNavigator presses the back key of the system, which Android turns into
// the back of the activity.
func TestTheBackKeyPopsTheRouteOfTheNavigator(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := backApp(ctx, t, e, "second", "")
		time.Sleep(2 * time.Second)
		shell(ctx, t, e, "input", "keyevent", "KEYCODE_BACK")
		if _, err := e.WaitLog(ctx, appID, "navigator event=pop route=/second", 40*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
	})
}

// TestAPopScopeThatCannotPopKeepsThePage has Back asked on a page under a PopScope with canPop false:
// the scope hears of it and the page stays.
func TestAPopScopeThatCannotPopKeepsThePage(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := backApp(ctx, t, e, "locked", "wait:2s;back:pop")
		log, err := e.WaitLog(ctx, appID, "pop scope page=locked didPop=false", 40*time.Second)
		if err != nil {
			t.Fatal(Describe(err))
		}
		time.Sleep(time.Second)
		log, err = e.Log(ctx, appID)
		if err != nil {
			t.Fatal(Describe(err))
		}
		if got := lines(log, "navigator event=pop"); len(got) != 0 {
			t.Errorf("the locked page was popped: %v", got)
		}
		if pidOf(ctx, t, e, appID) == 0 {
			t.Error("the app has no process")
		}
	})
}

// TestTheBackGestureRunsInTheFrameworkOnPredictiveBackAndPopsBelow has a back gesture scripted. With
// predictive back (Android 14 and later) the framework hears of its start, its progress and its commit
// (the example's observer takes the gesture, so a commit that nobody else takes is not a pop); below,
// there is no gesture, only a plain pop of the second page.
func TestTheBackGestureRunsInTheFrameworkOnPredictiveBackAndPopsBelow(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		api := strings.TrimSpace(shell(ctx, t, e, "getprop", "ro.build.version.sdk"))
		if len(api) == 2 && api >= "34" {
			appID := backApp(ctx, t, e, "second,take", "wait:2s;back:gesture")
			log, err := e.WaitLog(ctx, appID, "back event=commit", 40*time.Second)
			if err != nil {
				t.Fatal(Describe(err))
			}
			starts, updates := lines(log, "back event=start"), lines(log, "back event=update")
			if len(starts) != 1 || len(updates) == 0 || !strings.Contains(starts[0], "edge=SwipeEdge.left") {
				t.Errorf("API %s: the gesture was\n%s", api, strings.Join(lines(log, "back event="), "\n"))
			}
			return
		}
		appID := backApp(ctx, t, e, "second", "wait:2s;back:gesture")
		log, err := e.WaitLog(ctx, appID, "navigator event=pop route=/second", 40*time.Second)
		if err != nil {
			t.Fatal(Describe(err))
		}
		if got := lines(log, "back event=start"); len(got) != 0 {
			t.Errorf("API %s has a gesture start: %v", api, got)
		}
	})
}

// TestACancelledBackGestureKeepsThePage has the gesture started and then cancelled: the framework hears
// the cancel and the page stays. Below predictive back there is no gesture, so there is nothing to see.
func TestACancelledBackGestureKeepsThePage(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	appID := backApp(ctx, t, e, "second,take", "wait:2s;back:cancel")
	if _, err := e.WaitLog(ctx, appID, "back event=cancel", 40*time.Second); err != nil {
		t.Fatal(Describe(err))
	}
	time.Sleep(time.Second)
	log, err := e.Log(ctx, appID)
	if err != nil {
		t.Fatal(Describe(err))
	}
	if got := lines(log, "navigator event=pop"); len(got) != 0 {
		t.Errorf("a cancelled gesture popped the page: %v", got)
	}
}

// TestAKilledAppStartsAgainFromMain stops the backgrounded process as the system does when it needs
// the memory, and starts the app again: the program runs from its main function once more, in a new
// process.
func TestAKilledAppStartsAgainFromMain(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := backApp(ctx, t, e, "", "")
		first := pidOf(ctx, t, e, appID)
		shell(ctx, t, e, "input", "keyevent", "KEYCODE_HOME")
		if _, err := e.WaitLog(ctx, appID, lcPaused, 30*time.Second); err != nil {
			t.Fatal(Describe(err))
		}
		// "am kill" kills only a process that the system has put in its cached state; Android 8 does not
		// always do that for a stopped activity, and then the process gets the signal that the system's
		// low memory killer sends (the app is debuggable, so run-as can send it).
		for try := 0; ; try++ {
			if try < 3 {
				_, _ = e.ADB.Output(ctx, "shell", "am", "kill", appID)
			} else {
				_, _ = e.ADB.Output(ctx, "shell", "run-as", appID, "kill", "-9", strconv.Itoa(first))
			}
			time.Sleep(2 * time.Second)
			if pid, err := e.ADB.Pid(ctx, appID); err == nil && pid == 0 {
				break
			}
			if try > 6 {
				t.Fatalf("the process %d was not killed", first)
			}
		}
		if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_RENDERER": "soft"}); err != nil {
			t.Fatal(Describe(err))
		}
		if _, err := e.WaitLog(ctx, appID, "first frame presented", time.Minute); err != nil {
			t.Fatal(Describe(err))
		}
		if second := pidOf(ctx, t, e, appID); second == first {
			t.Errorf("the process is still %d", first)
		}
	})
}

// TestBackOnTheFirstRouteLeavesTheApp presses Back where the navigator has nothing to pop: the framework
// does not handle it, so Android does what it does for the root activity of a task, which is to leave
// the app (the activity is stopped, or finished and the process ends).
func TestBackOnTheFirstRouteLeavesTheApp(t *testing.T) {
	forEachDevice(t, func(t *testing.T, e *Emulator, ctx context.Context) {
		appID := backApp(ctx, t, e, "", "")
		time.Sleep(2 * time.Second)
		shell(ctx, t, e, "input", "keyevent", "KEYCODE_BACK")
		for end := time.Now().Add(40 * time.Second); ; time.Sleep(500 * time.Millisecond) {
			log, err := e.Log(ctx, appID)
			if err != nil {
				t.Fatal(Describe(err))
			}
			pid, err := e.ADB.Pid(ctx, appID)
			if err != nil {
				t.Fatal(Describe(err))
			}
			if len(lines(log, lcPaused)) > 0 || pid == 0 {
				t.Logf("left the app; process %d\n%s", pid, strings.Join(lines(log, "lifecycle"), "\n"))
				return
			}
			if time.Now().After(end) {
				t.Fatalf("the app is still shown:\n%s", strings.Join(lines(log, "lifecycle"), "\n"))
			}
		}
	})
}
