//go:build glutter_browser

package web_test

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// The audio test builds internal/platform/web/audiotest, whose driver opens devices of the platform's audio service and posts what
// happened (SPEC §28.3, §28.8). The browsers get the settings that a real user would have given by hand: the page may start sound
// without a gesture, and the microphone is a fake one that is allowed (or the prompt is refused). A gesture that the browser trusts
// cannot be made by a headless test, so the test of the gesture shows that the device waits and that a gesture reaches the context;
// that the sound then starts is the one thing that stays a check by hand (docs/MANUAL-CHECKS.md).

var (
	audioOnce sync.Once
	audioDir  string
	audioErr  error
)

func audioSite(t *testing.T) string {
	t.Helper()
	audioOnce.Do(func() {
		if audioDir, audioErr = os.MkdirTemp("", "glutter-web-audiotest-"); audioErr != nil {
			return
		}
		_, audioErr = webtool.Build(context.Background(), webtool.BuildOptions{
			Debug:   true,
			Package: "./internal/platform/web/audiotest", Dir: webtest.ModuleRoot(t), OutDir: audioDir,
		})
	})
	if audioErr != nil {
		t.Fatalf("building the audio test app for the web: %v", audioErr)
	}
	return audioDir
}

func cleanupAudioSite() {
	if audioDir != "" {
		_ = os.RemoveAll(audioDir) // a temporary directory: a leftover is harmless
	}
}

// audioObservations mirrors what audiotest posts.
type audioObservations struct {
	Mode              string   `json:"mode"`
	Isolated          bool     `json:"isolated"`
	OpenTag           string   `json:"openTag"`
	Rate              int      `json:"rate"`
	Channels          int      `json:"channels"`
	Shared            bool     `json:"shared"`
	States            []string `json:"states"`
	Peak              float64  `json:"peak"`
	Crossings         int      `json:"crossings"`
	WindowFrames      int      `json:"windowFrames"`
	FramesPerSecond   float64  `json:"framesPerSecond"`
	Underruns         float64  `json:"underruns"`
	Overruns          float64  `json:"overruns"`
	PausedPeak        float64  `json:"pausedPeak"`
	PausedFrames      float64  `json:"pausedFrames"`
	ResumedPeak       float64  `json:"resumedPeak"`
	ResumedFrames     float64  `json:"resumedFrames"`
	StressUnderruns   float64  `json:"stressUnderruns"`
	StressPeak        float64  `json:"stressPeak"`
	CloseError        string   `json:"closeError"`
	ClosedOrGone      bool     `json:"closedOrGone"`
	ContextState      string   `json:"contextState"`
	ResumeBefore      int      `json:"resumeBefore"`
	ResumeAfter       int      `json:"resumeAfter"`
	FramesWhileWaited int64    `json:"framesWhileWaited"`
	CaptureFrames     int64    `json:"captureFrames"`
	CaptureNonZero    bool     `json:"captureNonZero"`
}

// audioSettings are the flags that Chromium is given (webtest.ExtraFlagsEnv).
type audioSettings string

const (
	// autoplay lets the page start sound without a gesture.
	autoplay audioSettings = "--autoplay-policy=no-user-gesture-required"
	// fakeMicrophone is a microphone that makes a beep every second, and that the user allowed.
	fakeMicrophone audioSettings = "--autoplay-policy=no-user-gesture-required --use-fake-device-for-media-stream --use-fake-ui-for-media-stream"
	// refusedMicrophone is a user who answers the prompt for the microphone with "block".
	refusedMicrophone audioSettings = "--autoplay-policy=no-user-gesture-required --use-fake-device-for-media-stream --deny-permission-prompts"
	// nothing is a browser with its defaults.
	nothing audioSettings = ""
)

// eachAudioBrowser runs fn as a subtest for Chromium only: SPEC §28.8 tests the audio of the web in Chromium, whose headless mode
// plays to a fake output; Firefox needs a sound server of its own, so its sound is a manual check (docs/MANUAL-CHECKS.md).
func eachAudioBrowser(t *testing.T, fn func(t *testing.T, b webtest.Browser)) {
	t.Helper()
	t.Run(string(webtest.Chromium), func(t *testing.T) {
		if _, err := webtest.Locate(webtest.Chromium, nil); err != nil {
			t.Fatal(err)
		}
		fn(t, webtest.Chromium)
	})
}

// runAudio runs a mode of audiotest in the browser and returns what it posted. isolated says whether the page that embeds the app is
// cross-origin isolated.
func runAudio(t *testing.T, b webtest.Browser, mode string, isolated bool, set audioSettings) audioObservations {
	t.Helper()
	dir := audioSite(t)
	t.Setenv(webtest.ExtraFlagsEnv, string(set))
	srv := webtest.NewServer(dir)
	defer srv.Close()
	srv.SetIsolated(isolated)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	res, err := webtest.Await(ctx, webtest.ShotOptions{
		Dir: dir, Server: srv, Browser: b, Width: 480, Height: 640,
		Query: url.Values{"GLUTTER_AUDIO_MODE": {mode}},
	}, "audio")
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			t.Fatalf("Await in %s: %v\nmetadata: %v", b, err, appErr.Metadata)
		}
		t.Fatalf("Await in %s: %v", b, err)
	}
	var o audioObservations
	if err := json.Unmarshal(res.Data, &o); err != nil {
		t.Fatalf("the result is not the observations: %v\n%s", err, res.Data)
	}
	t.Logf("%+v", o)
	return o
}

// TestBrowserAudioPlaysATone: a device of the platform plays 440 Hz into the worklet, whose output a tap reads back: the right
// pitch and level, every frame taken, no underrun, silence while paused, sound again after the resume. It is run with the ring in
// shared memory and with the frames sent by messages.
func TestBrowserAudioPlaysATone(t *testing.T) {
	eachAudioBrowser(t, func(t *testing.T, b webtest.Browser) {
		for _, isolated := range []bool{true, false} {
			name := "messages"
			if isolated {
				name = "shared memory"
			}
			t.Run(name, func(t *testing.T) {
				o := runAudio(t, b, "play", isolated, autoplay)
				if o.OpenTag != "" {
					t.Fatalf("Open failed with %s", o.OpenTag)
				}
				if o.Isolated != isolated || o.Shared != isolated {
					t.Errorf("the page is isolated %v and the ring is shared %v, want both %v", o.Isolated, o.Shared, isolated)
				}
				if o.Rate < 8000 || o.Channels < 1 {
					t.Fatalf("the device has the format %d Hz, %d channels", o.Rate, o.Channels)
				}
				if !contains(o.States, "running") {
					t.Errorf("states %v lack running", o.States)
				}
				rate := float64(o.Rate)
				if o.FramesPerSecond < 0.9*rate || o.FramesPerSecond > 1.1*rate {
					t.Errorf("the worklet took %.0f frames a second, want about %d", o.FramesPerSecond, o.Rate)
				}
				if o.Peak < 0.45 || o.Peak > 0.55 {
					t.Errorf("the peak of the tone is %.3f, want 0.5", o.Peak)
				}
				// Two sign changes for each cycle: a window of n frames holds n*440/rate cycles.
				want := int(2 * 440 * float64(o.WindowFrames) / rate)
				if o.Crossings < want-2 || o.Crossings > want+2 {
					t.Errorf("the tone changes sign %d times in %d frames, want %d (a pure 440 Hz sine)", o.Crossings, o.WindowFrames, want)
				}
				if o.Underruns != 0 {
					t.Errorf("the ring ran dry %v times", o.Underruns)
				}
				if o.Overruns > 0 { // -1: a ring of messages has no such count
					t.Errorf("the ring was overrun %v times", o.Overruns)
				}
				if o.PausedFrames > 256 {
					t.Errorf("the worklet took %v frames while the device was paused", o.PausedFrames)
				}
				if o.PausedPeak > 0.001 {
					t.Errorf("the tap read a peak of %.4f while the device was paused", o.PausedPeak)
				}
				if o.ResumedFrames < 0.35*rate {
					t.Errorf("the worklet took %v frames in half a second after the resume, want about %.0f", o.ResumedFrames, 0.5*rate)
				}
				if o.ResumedPeak < 0.4 {
					t.Errorf("the peak after the resume is %.3f, want 0.5", o.ResumedPeak)
				}
				if o.StressUnderruns != 0 {
					t.Errorf("the sound had %v gaps while the page was busy for 50 ms at a time: the buffer of 100 ms must carry it", o.StressUnderruns)
				}
				if o.StressPeak < 0.4 {
					t.Errorf("the tone after the busy frames has the peak %.3f, want 0.5", o.StressPeak)
				}
				if o.CloseError != "" || !o.ClosedOrGone {
					t.Errorf("close: error %q, closed or gone %v", o.CloseError, o.ClosedOrGone)
				}
			})
		}
	})
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// TestBrowserAudioWaitsForAGesture: in a browser with its default autoplay rule the device opens at once and waits (Suspended), the
// handler is asked for no more than the ring takes, and a gesture makes the page ask the context to resume.
func TestBrowserAudioWaitsForAGesture(t *testing.T) {
	eachAudioBrowser(t, func(t *testing.T, b webtest.Browser) {
		o := runAudio(t, b, "gesture", true, nothing)
		if o.OpenTag != "" {
			t.Fatalf("Open failed with %s: it must not wait for the gesture", o.OpenTag)
		}
		if o.ContextState != "suspended" {
			t.Skipf("this %s lets a page start sound without a gesture (the context is %s), so there is nothing to wait for", b, o.ContextState)
		}
		if !contains(o.States, "suspended") {
			t.Errorf("states %v lack suspended", o.States)
		}
		if o.ResumeAfter <= o.ResumeBefore {
			t.Errorf("a gesture did not make the page resume the context: %d calls before, %d after", o.ResumeBefore, o.ResumeAfter)
		}
		if limit := int64(o.Rate) * 3 / 2; o.FramesWhileWaited > limit {
			t.Errorf("the handler was asked for %d frames while the sound could not start, want at most the buffer", o.FramesWhileWaited)
		}
	})
}

// TestBrowserAudioCapture: a recording of the fake microphone delivers about a second of frames each second.
func TestBrowserAudioCapture(t *testing.T) {
	eachAudioBrowser(t, func(t *testing.T, b webtest.Browser) {
		o := runAudio(t, b, "capture", true, fakeMicrophone)
		if o.OpenTag != "" {
			t.Fatalf("Open of the recording failed with %s", o.OpenTag)
		}
		if min := int64(o.Rate) * 3 / 2; o.CaptureFrames < min || o.CaptureFrames > int64(o.Rate)*3 {
			t.Errorf("the recording gave %d frames in two seconds at %d Hz", o.CaptureFrames, o.Rate)
		}
		if !o.CaptureNonZero {
			t.Error("the recording is silent: the fake microphone beeps every second")
		}
		if o.CloseError != "" {
			t.Errorf("close: %s", o.CloseError)
		}
	})
}

// TestBrowserAudioCaptureRefused: a user who blocks the microphone gives AudioCapturePermissionDenied.
func TestBrowserAudioCaptureRefused(t *testing.T) {
	eachAudioBrowser(t, func(t *testing.T, b webtest.Browser) {
		o := runAudio(t, b, "capture", true, refusedMicrophone)
		if o.OpenTag != platform.TagAudioCapturePermissionDenied {
			t.Errorf("Open gave the tag %q, want %s", o.OpenTag, platform.TagAudioCapturePermissionDenied)
		}
		if strings.Contains(o.OpenTag, "NoAudio") {
			t.Error("the page has no audio service")
		}
	})
}
