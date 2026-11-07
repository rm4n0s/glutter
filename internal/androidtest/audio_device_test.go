//go:build glutter_android

package androidtest

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/androidtool"
)

// The audio of Android (SPEC §28.3, §28.8, step 31.6) on the emulator with audio hardware (glutter-37-audio: the emulator runs
// with -no-audio, so the sound goes to a null device on the host). examples/dev/android-audio plays a tone through the public package
// and through a device of the platform, and logs twice a second.

var statsLine = regexp.MustCompile(`audio stats public=(\d+) platform=(\d+) frame=(-?\d+) nanos=(-?\d+) ts=(true|false) xruns=(\d+) buffer=(\d+) burst=(\d+)`)

// stats is one "audio stats" line of the log.
type stats struct {
	public, platform, frame, nanos int64
	ts                             bool
	xruns, buffer, burst           int
}

// statsOf reads every "audio stats" line of the log.
func statsOf(log string) []stats {
	var out []stats
	for _, m := range statsLine.FindAllStringSubmatch(log, -1) {
		n := func(i int) int64 { v, _ := strconv.ParseInt(m[i], 10, 64); return v }
		out = append(out, stats{public: n(1), platform: n(2), frame: n(3), nanos: n(4), ts: m[5] == "true", xruns: int(n(6)), buffer: int(n(7)), burst: int(n(8))})
	}
	return out
}

// audioApp installs examples/dev/android-audio with the permission to record, with the renderer soft, and starts it.
func audioApp(ctx context.Context, t *testing.T, e *Emulator, extras map[string]string, grant bool) string {
	t.Helper()
	appID := installAppWith(ctx, t, e, "./examples/dev/android-audio", androidtool.BuildOptions{Permissions: []string{"android.permission.RECORD_AUDIO"}})
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	if grant {
		shell(ctx, t, e, "pm", "grant", appID, "android.permission.RECORD_AUDIO")
	} else {
		// The permission may be granted from an earlier run of this app.
		_, _ = e.ADB.Output(ctx, "shell", "pm", "revoke", appID, "android.permission.RECORD_AUDIO")
	}
	all := map[string]string{"GLUTTER_RENDERER": "soft"}
	for k, v := range extras {
		all[k] = v
	}
	if err := e.Launch(ctx, appID, all); err != nil {
		t.Fatal(Describe(err))
	}
	return appID
}

// waitStats waits until the log has n lines of stats that satisfy ok (counted from the last one) and returns them.
func waitStats(ctx context.Context, t *testing.T, e *Emulator, appID string, timeout time.Duration, ok func(last stats) bool) []stats {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		log, err := e.Log(ctx, appID)
		if err != nil {
			t.Fatal(Describe(err))
		}
		if all := statsOf(log); len(all) > 0 && ok(all[len(all)-1]) {
			return all
		}
		if time.Now().After(deadline) {
			t.Fatalf("the stats never came as expected:\n%s", log)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func TestAudioCallbacksRunAndTheTimestampsAdvance(t *testing.T) {
	e := Shared(t, "glutter-37-audio")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	appID := audioApp(ctx, t, e, nil, false)
	first := waitStats(ctx, t, e, appID, 2*time.Minute, func(s stats) bool { return s.ts && s.platform > 0 && s.public > 0 })
	f := first[len(first)-1]
	time.Sleep(2 * time.Second)
	all := statsOf(mustLog(ctx, t, e, appID))
	last := all[len(all)-1]
	if last.frame <= f.frame || last.nanos <= f.nanos {
		t.Errorf("the timestamp did not advance: %+v then %+v", f, last)
	}
	if last.platform <= f.platform || last.public <= f.public {
		t.Errorf("the callbacks did not go on: %+v then %+v", f, last)
	}
	// The hardware plays about as many frames as the time that passed says (a timestamp is frames and the time at which they were).
	if dt := float64(last.nanos-f.nanos) / 1e9; dt > 0.5 {
		rate := float64(last.frame-f.frame) / dt
		if rate < 40000 || rate > 56000 {
			t.Errorf("the stream plays %.0f frames a second", rate)
		}
	}
	if last.burst <= 0 || last.buffer < last.burst {
		t.Errorf("the buffer is %d frames and the burst %d", last.buffer, last.burst)
	}
	log := mustLog(ctx, t, e, appID)
	for _, bad := range []string{"audio playback failed", "audio platform device failed"} {
		if strings.Contains(log, bad) {
			t.Errorf("%q is in the log:\n%s", bad, log)
		}
	}
}

func mustLog(ctx context.Context, t *testing.T, e *Emulator, appID string) string {
	t.Helper()
	log, err := e.Log(ctx, appID)
	if err != nil {
		t.Fatal(Describe(err))
	}
	return log
}

func TestAudioPausesWhenTheAppIsHiddenAndGoesOn(t *testing.T) {
	e := Shared(t, "glutter-37-audio")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	appID := audioApp(ctx, t, e, nil, false)
	waitStats(ctx, t, e, appID, 2*time.Minute, func(s stats) bool { return s.ts && s.platform > 0 && s.public > 0 })

	// Home: the activity stops, and the devices that pause in the background (the default of the public package on Android) with it.
	shell(ctx, t, e, "input", "keyevent", "KEYCODE_HOME")
	if _, err := e.WaitLog(ctx, appID, "audio state state=paused", 30*time.Second); err != nil {
		t.Fatal(Describe(err))
	}
	time.Sleep(time.Second)
	all := statsOf(mustLog(ctx, t, e, appID))
	a := all[len(all)-1]
	time.Sleep(2 * time.Second)
	all = statsOf(mustLog(ctx, t, e, appID))
	b := all[len(all)-1]
	if b.public != a.public || b.platform != a.platform {
		t.Errorf("the callbacks went on while the app was hidden: %+v then %+v", a, b)
	}

	// Back to the app: the same process, the same devices.
	pid, err := e.ADB.Pid(ctx, appID)
	if err != nil || pid == 0 {
		t.Fatalf("the app has no process: %v", err)
	}
	shell(ctx, t, e, "am", "start", "-n", appID+"/"+androidtool.ActivityClass)
	if _, err := e.WaitLog(ctx, appID, "audio state state=running", 30*time.Second); err != nil {
		t.Fatal(Describe(err))
	}
	waitStats(ctx, t, e, appID, time.Minute, func(s stats) bool { return s.public > b.public && s.platform > b.platform })
	if again, err := e.ADB.Pid(ctx, appID); err != nil || again != pid {
		t.Errorf("the app was started again: pid %d then %d (%v)", pid, again, err)
	}
	if log := mustLog(ctx, t, e, appID); strings.Contains(log, "state=lost") {
		t.Errorf("a device was lost:\n%s", log)
	}
}

// captureFrames is the last "audio capture frames=N" of the log, or -1.
func captureFrames(log string) int {
	re := regexp.MustCompile(`audio capture frames=(\d+)`)
	all := re.FindAllStringSubmatch(log, -1)
	if len(all) == 0 {
		return -1
	}
	n, _ := strconv.Atoi(all[len(all)-1][1])
	return n
}

func TestAudioCaptureWithThePermission(t *testing.T) {
	e := Shared(t, "glutter-37-audio")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	appID := audioApp(ctx, t, e, map[string]string{"GLUTTER_AUDIO_CAPTURE": "1"}, true)
	deadline := time.Now().Add(2 * time.Minute)
	for {
		log := mustLog(ctx, t, e, appID)
		if strings.Contains(log, "audio capture failed") {
			t.Fatalf("the recording failed although the permission was granted:\n%s", log)
		}
		if captureFrames(log) > 24000 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the recording gave no frames:\n%s", log)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// tapDialogButton taps the button of the permission dialog whose text is one of the labels. It reads the screen with uiautomator.
func tapDialogButton(ctx context.Context, t *testing.T, e *Emulator, labels ...string) {
	t.Helper()
	node := regexp.MustCompile(`<node [^>]*text="([^"]*)"[^>]*bounds="\[(\d+),(\d+)\]\[(\d+),(\d+)\]"`)
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		xml, err := e.ADB.Output(ctx, "shell", "uiautomator dump /sdcard/ui.xml >/dev/null; cat /sdcard/ui.xml")
		if err == nil {
			for _, m := range node.FindAllStringSubmatch(xml, -1) {
				for _, label := range labels {
					if strings.EqualFold(strings.ReplaceAll(m[1], "’", "'"), label) {
						n := func(i int) int { v, _ := strconv.Atoi(m[i]); return v }
						shell(ctx, t, e, "input", "tap", strconv.Itoa((n(2)+n(4))/2), strconv.Itoa((n(3)+n(5))/2))
						return
					}
				}
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("no button of the dialog says one of %v", labels)
}

func TestAudioCaptureAsksTheUserAndTakesTheRefusal(t *testing.T) {
	e := Shared(t, "glutter-37-audio")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	appID := audioApp(ctx, t, e, map[string]string{"GLUTTER_AUDIO_CAPTURE": "1"}, false)
	// The dialog is the system's: the test finds its button on the screen (the Java side's lines are under another tag).
	tapDialogButton(ctx, t, e, "Don't allow", "Deny")
	log, err := e.WaitLog(ctx, appID, "audio capture failed err=AudioCapturePermissionDenied", time.Minute)
	if err != nil {
		t.Fatal(Describe(err))
	}
	if strings.Contains(log, "audio playback failed") {
		t.Errorf("playback failed too:\n%s", log)
	}
	// Playback goes on without the permission.
	waitStats(ctx, t, e, appID, time.Minute, func(s stats) bool { return s.public > 0 })
}
