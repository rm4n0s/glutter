//go:build glutter_ios

package iostest_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/iostest"
	"github.com/rm4n0s/glutter/internal/iostest/serviceslines"
)

// TestAudioStreamPlaysAndKeepsTime opens a playback stream of 48 kHz stereo on the Simulator, whose audio goes to the Mac's output, and
// counts the frames that the device asked the app for in the time that it ran: the stream keeps time (SPEC §30.9).
func TestAudioStreamPlaysAndKeepsTime(t *testing.T) {
	h := iostest.New(t)
	app, _ := runServices(t, h, "audio")
	line, err := app.WaitFor(serviceslines.Audio+" frames=", time.Minute)
	if err != nil {
		t.Fatalf("the audio line: %v", err)
	}
	frames, elapsed := numberAfter(t, line, "frames="), numberAfter(t, line, "elapsed=")
	if numberAfter(t, line, "rate=") != 48000 || numberAfter(t, line, "devices=") != 2 {
		t.Errorf("the stream's format or the device list: %q", line)
	}
	want := 48000 * elapsed / 1000
	if math.Abs(frames-want) > want*0.25 {
		t.Errorf("the stream asked for %v frames in %v ms, want about %v", frames, elapsed, want)
	}
	if frames < 48000/2 {
		t.Errorf("the stream asked for only %v frames", frames)
	}
}

// TestAudioInterruptionPausesAndResumes drives the audio session's interruption notification of the system while a stream runs: the stream
// is told that it paused, and when the interruption ends and the system says to resume, that it runs again (SPEC §30.9).
func TestAudioInterruptionPausesAndResumes(t *testing.T) {
	h := iostest.New(t)
	app, _ := runServices(t, h, "audio;interrupt:began;interrupt:ended")
	var states []string
	for _, l := range app.Output() {
		if i := strings.Index(l, serviceslines.State+" "); i >= 0 {
			states = append(states, strings.TrimSpace(l[i+len(serviceslines.State):]))
		}
	}
	if len(states) != 2 || states[0] != "paused" || states[1] != "running" {
		t.Errorf("the stream's states = %v, want paused then running", states)
	}
}

// TestSensorsAreUnavailableInTheSimulator opens the accelerometer and the gyroscope: the Simulator has neither, and Core Motion says so with
// the tag SensorUnavailable (SPEC §30.9).
func TestSensorsAreUnavailableInTheSimulator(t *testing.T) {
	h := iostest.New(t)
	app, _ := runServices(t, h, "sensor:accel;sensor:gyro")
	for _, step := range []string{"sensor:accel", "sensor:gyro"} {
		if r, ok := stepResult(app, step); !ok || r != "available=false err=SensorUnavailable" {
			t.Errorf("%s: %q, %v", step, r, ok)
		}
	}
}

// TestPadServiceStarts starts the pad service in the Simulator: it starts without an error and says which mapping table it is for.
func TestPadServiceStarts(t *testing.T) {
	h := iostest.New(t)
	app, _ := runServices(t, h, "pads")
	if r, ok := stepResult(app, "pads"); !ok || r != "mapping=iOS err=nil" {
		t.Errorf("pads: %q, %v", r, ok)
	}
	// The Simulator may have a virtual controller; whatever pad it reports is a standard one, with a name.
	for _, l := range app.Output() {
		if strings.Contains(l, "pad added") && !strings.Contains(l, "standard=true") {
			t.Errorf("a pad that is not standard: %q", l)
		}
	}
}
