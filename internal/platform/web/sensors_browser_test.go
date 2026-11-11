//go:build glutter_browser

package web_test

import (
	"context"
	"encoding/json"
	"math"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// The sensors test builds internal/platform/web/sensortest, whose driver opens sensors of package sensors, sends DeviceMotionEvents
// of its own making to the page, and posts what the app saw (SPEC §28.4, §28.8). The motion of a real phone is the manual check.

var (
	sensorOnce sync.Once
	sensorDir  string
	sensorErr  error
)

func sensorSite(t *testing.T) string {
	t.Helper()
	sensorOnce.Do(func() {
		if sensorDir, sensorErr = os.MkdirTemp("", "glutter-web-sensortest-"); sensorErr != nil {
			return
		}
		_, sensorErr = webtool.Build(context.Background(), webtool.BuildOptions{
			Debug:   true,
			Package: "./internal/platform/web/sensortest", Dir: webtest.ModuleRoot(t), OutDir: sensorDir,
		})
	})
	if sensorErr != nil {
		t.Fatalf("building the sensors test app for the web: %v", sensorErr)
	}
	return sensorDir
}

func cleanupSensorSite() {
	if sensorDir != "" {
		_ = os.RemoveAll(sensorDir) // a temporary directory: a leftover is harmless
	}
}

// sensorObservations mirrors what sensortest posts.
type sensorObservations struct {
	Mode                  string     `json:"mode"`
	Stage                 string     `json:"stage"`
	Available             bool       `json:"available"`
	HasAccel              bool       `json:"hasAccel"`
	HasGyro               bool       `json:"hasGyro"`
	AccelOpenTag          string     `json:"accelOpenTag"`
	AccelOpenRoute        string     `json:"accelOpenRoute"`
	GyroOpenTag           string     `json:"gyroOpenTag"`
	AccelRate             float64    `json:"accelRate"`
	AccelLatest           [3]float64 `json:"accelLatest"`
	GyroLatest            [3]float64 `json:"gyroLatest"`
	AccelBatches          []int      `json:"accelBatches"`
	GyroBatches           []int      `json:"gyroBatches"`
	AccelLast3            [3]float64 `json:"accelLast3"`
	GyroCountNoRot        int        `json:"gyroCountNoRot"`
	AccelCountNoRot       int        `json:"accelCountNoRot"`
	GyroAfterClose        int        `json:"gyroAfterClose"`
	AccelAfterClose       int        `json:"accelAfterClose"`
	CloseErrors           []string   `json:"closeErrors"`
	Vibrations            []int      `json:"vibrations"`
	AskedBeforeGesture    int        `json:"askedBeforeGesture"`
	RejectedBeforeGesture int        `json:"rejectedBeforeGesture"`
	AskedAfterGesture     int        `json:"askedAfterGesture"`
	AskedAfterSecond      int        `json:"askedAfterSecond"`
	OpenedBeforeGesture   bool       `json:"openedBeforeGesture"`
	SecondOpenTag         string     `json:"secondOpenTag"`
	GotReading            bool       `json:"gotReading"`
}

func runSensors(t *testing.T, b webtest.Browser, query url.Values) sensorObservations {
	t.Helper()
	dir := sensorSite(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	res, err := webtest.Await(ctx, webtest.ShotOptions{Dir: dir, Browser: b, Width: 480, Height: 640, Query: query}, "sensors")
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			t.Fatalf("Await in %s: %v\nmetadata: %v", b, err, appErr.Metadata)
		}
		t.Fatalf("Await in %s: %v", b, err)
	}
	var o sensorObservations
	if err := json.Unmarshal(res.Data, &o); err != nil {
		t.Fatalf("the result is not the observations: %v\n%s", err, res.Data)
	}
	t.Logf("%+v", o)
	return o
}

func near(a, b [3]float64) bool {
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1e-9 {
			return false
		}
	}
	return true
}

// TestBrowserSensors: DeviceMotionEvents sent to the page reach the sensors of package sensors with the units and the axes of the
// toolkit, are batched once a frame, stop at Close, and the haptics vibrate with the lengths of the kinds.
func TestBrowserSensors(t *testing.T) {
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		o := runSensors(t, b, url.Values{"GLUTTER_SENSOR_MODE": {"motion"}})
		if o.Stage != "done" {
			t.Fatalf("the driver stopped at %q", o.Stage)
		}
		if !o.Available || !o.HasAccel || !o.HasGyro || o.AccelOpenTag != "" || o.GyroOpenTag != "" {
			t.Fatalf("the sensors did not open: %+v", o)
		}
		if !near(o.AccelLatest, [3]float64{0.5, -9.8, 1.5}) {
			t.Errorf("acceleration = %v, want 0.5 -9.8 1.5 (m/s², the browser's axes are the toolkit's)", o.AccelLatest)
		}
		// alpha 90, beta 180, gamma -45 degrees a second: x is beta, y is gamma, z is alpha.
		if !near(o.GyroLatest, [3]float64{math.Pi, -math.Pi / 4, math.Pi / 2}) {
			t.Errorf("rotation = %v, want pi, -pi/4, pi/2 (rad/s)", o.GyroLatest)
		}
		if !reflect.DeepEqual(o.AccelBatches, []int{1, 3}) || !reflect.DeepEqual(o.GyroBatches, []int{1, 3}) {
			t.Errorf("batches: accelerometer %v, gyroscope %v; want one reading, then the three that came together", o.AccelBatches, o.GyroBatches)
		}
		if !near(o.AccelLast3, [3]float64{3, 3, 3}) {
			t.Errorf("the last reading of the batch is %v", o.AccelLast3)
		}
		if o.AccelCountNoRot != 1 || o.GyroCountNoRot != 0 {
			t.Errorf("an event with no rotation gave the accelerometer %d readings and the gyroscope %d, want 1 and 0", o.AccelCountNoRot, o.GyroCountNoRot)
		}
		if o.GyroAfterClose != 0 || o.AccelAfterClose != 1 {
			t.Errorf("after the gyroscope was closed it got %d readings and the accelerometer %d, want 0 and 1", o.GyroAfterClose, o.AccelAfterClose)
		}
		if len(o.CloseErrors) != 0 {
			t.Errorf("Close failed: %v", o.CloseErrors)
		}
		if !reflect.DeepEqual(o.Vibrations, []int{50, 10, 20, 40, 5}) {
			t.Errorf("navigator.vibrate was called with %v, want 50 10 20 40 5 (vibrate, light, medium, heavy, selection)", o.Vibrations)
		}
	})
}

// TestBrowserSensorsAskTheUserAtAGesture: where the browser has requestPermission and wants it asked in a gesture (Safari on iOS),
// the first Open asks at once, is rejected, waits for a gesture, asks again in it, and takes the answer; a refusal is
// SensorPermissionDenied. A browser that grants without a gesture (Chrome) opens the sensor at once, with one question.
func TestBrowserSensorsAskTheUserAtAGesture(t *testing.T) {
	eachBrowserFor := func(t *testing.T, fn func(t *testing.T, b webtest.Browser)) { webtest.EachBrowser(t, fn) }
	eachBrowserFor(t, func(t *testing.T, b webtest.Browser) {
		ask := func(answer, gesture string) sensorObservations {
			return runSensors(t, b, url.Values{"GLUTTER_SENSOR_MODE": {"permission"}, "GLUTTER_SENSOR_ANSWER": {answer}, "GLUTTER_SENSOR_GESTURE": {gesture}})
		}
		t.Run("granted at a gesture", func(t *testing.T) {
			o := ask("granted", "1")
			if o.Stage != "done" {
				t.Fatalf("the driver stopped at %q", o.Stage)
			}
			if o.AskedBeforeGesture != 0 || o.RejectedBeforeGesture != 1 || o.OpenedBeforeGesture {
				t.Errorf("before any gesture: %d questions taken, %d rejected, opened %v; want 0, 1 (the try at Open) and not opened",
					o.AskedBeforeGesture, o.RejectedBeforeGesture, o.OpenedBeforeGesture)
			}
			if o.AccelOpenTag != "" || o.AskedAfterGesture != 1 || !o.GotReading {
				t.Errorf("after the gesture: open tag %q, asked %d times, reading %v", o.AccelOpenTag, o.AskedAfterGesture, o.GotReading)
			}
			if o.SecondOpenTag != "" || o.AskedAfterSecond != 1 {
				t.Errorf("a second open: tag %q, asked %d times in all, want no second question", o.SecondOpenTag, o.AskedAfterSecond)
			}
		})
		t.Run("denied at a gesture", func(t *testing.T) {
			o := ask("denied", "1")
			if o.Stage != "done" {
				t.Fatalf("the driver stopped at %q", o.Stage)
			}
			if o.AskedBeforeGesture != 0 || o.OpenedBeforeGesture {
				t.Errorf("%d questions taken and opened %v before any gesture", o.AskedBeforeGesture, o.OpenedBeforeGesture)
			}
			if o.AccelOpenTag != platform.TagSensorPermissionDenied || o.AskedAfterGesture != 1 {
				t.Errorf("open tag %q after %d questions, want %s after one", o.AccelOpenTag, o.AskedAfterGesture, platform.TagSensorPermissionDenied)
			}
			if !strings.Contains(o.AccelOpenRoute, "webSensors.Open.SensorPermissionDenied") {
				t.Errorf("unexpected failure path: %s", o.AccelOpenRoute)
			}
		})
		t.Run("granted without a gesture", func(t *testing.T) {
			o := ask("granted", "0")
			if o.Stage != "done" {
				t.Fatalf("the driver stopped at %q", o.Stage)
			}
			if !o.OpenedBeforeGesture || o.AskedBeforeGesture != 1 || o.AccelOpenTag != "" || !o.GotReading {
				t.Errorf("opened without a gesture %v after %d questions, tag %q, reading %v", o.OpenedBeforeGesture, o.AskedBeforeGesture, o.AccelOpenTag, o.GotReading)
			}
		})
	})
}

// TestBrowserSensorsWithoutMotionEvents: a browser with no DeviceMotionEvent has no sensors, and Open says so.
func TestBrowserSensorsWithoutMotionEvents(t *testing.T) {
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		o := runSensors(t, b, url.Values{"GLUTTER_SENSOR_MODE": {"none"}})
		if !o.Available || o.HasAccel || o.HasGyro || o.AccelOpenTag != platform.TagSensorUnavailable {
			t.Errorf("without DeviceMotionEvent: available %v, has %v and %v, open tag %q", o.Available, o.HasAccel, o.HasGyro, o.AccelOpenTag)
		}
		if !strings.Contains(o.AccelOpenRoute, "webSensors.Open.SensorUnavailable") {
			t.Errorf("unexpected failure path: %s", o.AccelOpenRoute)
		}
	})
}
