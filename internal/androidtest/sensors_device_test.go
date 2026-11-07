//go:build glutter_android

package androidtest

import (
	"context"
	"math"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/androidtool"
)

// The motion sensors of Android (SPEC §28.4, §28.8, step 31.7) on the emulator, whose virtual sensors a test sets with the console
// (adb emu sensor set). examples/dev/android-sensors opens the accelerometer and the gyroscope through package sensors and logs the
// latest readings twice a second.

var sensorLine = regexp.MustCompile(`sensors (accelerometer|gyroscope) x=(-?[\d.]+) y=(-?[\d.]+) z=(-?[\d.]+) readings=(\d+) frames=(\d+) dropped=(\d+)`)

// sensorStats is one "sensors <kind>" line of the log.
type sensorStats struct {
	kind                      string
	x, y, z                   float64
	readings, frames, dropped int
}

// lastSensor is the last line of the log for a kind.
func lastSensor(log, kind string) (sensorStats, bool) {
	var out sensorStats
	found := false
	for _, m := range sensorLine.FindAllStringSubmatch(log, -1) {
		if m[1] != kind {
			continue
		}
		f := func(i int) float64 { v, _ := strconv.ParseFloat(m[i], 64); return v }
		n := func(i int) int { v, _ := strconv.Atoi(m[i]); return v }
		out, found = sensorStats{kind: kind, x: f(2), y: f(3), z: f(4), readings: n(5), frames: n(6), dropped: n(7)}, true
	}
	return out, found
}

func sensorApp(ctx context.Context, t *testing.T, e *Emulator) string {
	t.Helper()
	appID := installAppWith(ctx, t, e, "./examples/dev/android-sensors", androidtool.BuildOptions{})
	t.Cleanup(func() { _, _ = e.ADB.Output(context.Background(), "shell", "am", "force-stop", appID) })
	if err := e.Launch(ctx, appID, map[string]string{"GLUTTER_RENDERER": "soft"}); err != nil {
		t.Fatal(Describe(err))
	}
	return appID
}

// setSensor sets a virtual sensor of the emulator: "acceleration" (m/s²) or "gyroscope" (rad/s).
func setSensor(ctx context.Context, t *testing.T, e *Emulator, name string, x, y, z float64) {
	t.Helper()
	v := strconv.FormatFloat(x, 'f', -1, 64) + ":" + strconv.FormatFloat(y, 'f', -1, 64) + ":" + strconv.FormatFloat(z, 'f', -1, 64)
	if out, err := e.ADB.Output(ctx, "emu", "sensor", "set", name, v); err != nil {
		t.Fatalf("adb emu sensor set %s %s: %v\n%s", name, v, err, out)
	}
}

// waitSensor waits until the log's last line for the kind has the values (within 0.05), and returns it.
func waitSensor(ctx context.Context, t *testing.T, e *Emulator, appID, kind string, x, y, z float64) sensorStats {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	var last sensorStats
	for {
		log, err := e.Log(ctx, appID)
		if err != nil {
			t.Fatal(Describe(err))
		}
		if s, ok := lastSensor(log, kind); ok {
			last = s
			if math.Abs(s.x-x) < 0.05 && math.Abs(s.y-y) < 0.05 && math.Abs(s.z-z) < 0.05 {
				return s
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("the %s never read %v %v %v; the last line was %+v\nlog:\n%s", kind, x, y, z, last, log)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func TestSensorsFollowTheEmulatorsVirtualSensors(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	appID := sensorApp(ctx, t, e)
	for _, kind := range []string{"accelerometer", "gyroscope"} {
		if _, err := e.WaitLog(ctx, appID, "sensors open kind="+kind, 2*time.Minute); err != nil {
			t.Fatal(Describe(err))
		}
	}

	// The axes: a phone lying flat reads gravity on z, one held upright on y, one on its side on x; the signs are Android's, which
	// are the toolkit's. A gyroscope turning about each axis shows on that axis.
	tests := []struct {
		name    string
		sensor  string
		kind    string
		x, y, z float64
	}{
		{"flat on its back", "acceleration", "accelerometer", 0, 0, 9.81},
		{"upright", "acceleration", "accelerometer", 0, 9.81, 0},
		{"on its side, the right edge down", "acceleration", "accelerometer", 9.81, 0, 0},
		{"tilted", "acceleration", "accelerometer", 3, 4, 5},
		{"turning about x", "gyroscope", "gyroscope", 1.5, 0, 0},
		{"turning about the three axes", "gyroscope", "gyroscope", 1, -2, 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setSensor(ctx, t, e, tt.sensor, tt.x, tt.y, tt.z)
			s := waitSensor(ctx, t, e, appID, tt.kind, tt.x, tt.y, tt.z)
			t.Logf("%+v", s)
		})
	}

	// The readings keep coming, in batches of a frame: no more calls than readings, and none thrown away.
	a, _ := lastSensor(mustLogOf(ctx, t, e, appID), "accelerometer")
	time.Sleep(3 * time.Second)
	b, _ := lastSensor(mustLogOf(ctx, t, e, appID), "accelerometer")
	if b.readings <= a.readings {
		t.Errorf("the accelerometer gave no more readings in 3 s: %d then %d", a.readings, b.readings)
	}
	if b.frames > b.readings || b.dropped != 0 {
		t.Errorf("%d readings in %d calls, %d dropped", b.readings, b.frames, b.dropped)
	}
	if log := mustLogOf(ctx, t, e, appID); regexp.MustCompile(`sensors failed`).MatchString(log) {
		t.Errorf("a sensor failed to open:\n%s", log)
	}
}

func mustLogOf(ctx context.Context, t *testing.T, e *Emulator, appID string) string {
	t.Helper()
	log, err := e.Log(ctx, appID)
	if err != nil {
		t.Fatal(Describe(err))
	}
	return log
}
