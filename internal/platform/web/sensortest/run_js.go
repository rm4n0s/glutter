//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"syscall/js"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/web"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/sensors"
)

// observations is what the program posts as the result "sensors". A field that a mode does not fill stays empty.
type observations struct {
	Mode string `json:"mode"`
	// Stage is the last step that the driver reached; a driver that hangs posts what it has, with the stage, after 30 s.
	Stage string `json:"stage"`
	// Has says whether the System believes the device has the sensors, Available that the platform has the service.
	Available      bool    `json:"available"`
	HasAccel       bool    `json:"hasAccel"`
	HasGyro        bool    `json:"hasGyro"`
	AccelOpenTag   string  `json:"accelOpenTag"`
	AccelOpenRoute string  `json:"accelOpenRoute"`
	GyroOpenTag    string  `json:"gyroOpenTag"`
	AccelRate      float64 `json:"accelRate"`
	GyroRateAfter  float64 `json:"gyroRateAfter"`

	// motion: the latest readings after the first event, the sizes of the batches that the listeners got (a batch is what
	// arrived since the last frame), the frames in which they were called, what the gyroscope had after an event with no
	// rotation, and what each sensor had after it was closed.
	AccelLatest     [3]float64 `json:"accelLatest"`
	GyroLatest      [3]float64 `json:"gyroLatest"`
	AccelTimestamp  float64    `json:"accelTimestampMs"`
	AccelBatches    []int      `json:"accelBatches"`
	GyroBatches     []int      `json:"gyroBatches"`
	AccelLast3      [3]float64 `json:"accelLast3"`
	GyroCountNoRot  int        `json:"gyroCountNoRot"`
	AccelCountNoRot int        `json:"accelCountNoRot"`
	GyroAfterClose  int        `json:"gyroAfterClose"`
	AccelAfterClose int        `json:"accelAfterClose"`
	CloseErrors     []string   `json:"closeErrors"`
	Vibrations      []int      `json:"vibrations"`

	// permission: the calls of requestPermission before and after a gesture, and after a second Open.
	AskedBeforeGesture    int    `json:"askedBeforeGesture"`
	RejectedBeforeGesture int    `json:"rejectedBeforeGesture"`
	AskedAfterGesture     int    `json:"askedAfterGesture"`
	AskedAfterSecond      int    `json:"askedAfterSecond"`
	OpenedBeforeGesture   bool   `json:"openedBeforeGesture"`
	SecondOpenTag         string `json:"secondOpenTag"`
	GotReading            bool   `json:"gotReading"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	p := newProbe()
	go func() {
		o := &observations{Mode: os.Getenv("GLUTTER_SENSOR_MODE")}
		var once sync.Once
		post := func() { once.Do(func() { postResult("sensors", *o) }) }
		time.AfterFunc(30*time.Second, func() {
			if o.Stage != "done" {
				o.Stage += " (hung)"
			}
			post()
		})
		observe(p, o)
		o.Stage = "done"
		post()
	}()
	return m.RunApp(newSensorApp(p), m.RunAppOpts{Window: m.WindowConfig{Title: "Sensors test"}})
}

// routeOf is the route of the error, "" for none.
func routeOf(err error) string {
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Route()
	}
	return ""
}

func tagOf(err error) string {
	if err == nil {
		return ""
	}
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Tag
	}
	return "none"
}

// installScripts defines what the driver calls in the page: the sending of a DeviceMotionEvent, a recorder in place of
// navigator.vibrate, and the fake requestPermission of iOS.
func installScripts() {
	js.Global().Get("Function").New(`
		globalThis.motion = function (accel, rot, interval) {
			var init = {interval: interval};
			if (accel) init.accelerationIncludingGravity = {x: accel[0], y: accel[1], z: accel[2]};
			if (rot) init.rotationRate = {alpha: rot[0], beta: rot[1], gamma: rot[2]};
			window.dispatchEvent(new DeviceMotionEvent("devicemotion", init));
		};
		globalThis.vibrations = [];
		navigator.vibrate = function (p) { globalThis.vibrations.push(p); return true; };
		globalThis.permissionCalls = 0;
		globalThis.permissionRejects = 0;
		globalThis.gestureActive = false;
		globalThis.fakePermission = function (answer, needsGesture) {
			DeviceMotionEvent.requestPermission = function () {
				if (needsGesture && !globalThis.gestureActive) {
					globalThis.permissionRejects++;
					return Promise.reject(new DOMException("needs a gesture", "NotAllowedError"));
				}
				globalThis.permissionCalls++;
				return Promise.resolve(answer);
			};
		};
	`).Invoke()
}

// counter collects what the listeners of a sensor got.
type counter struct {
	mu      sync.Mutex
	batches []int
	last    sensors.Reading
	total   int
}

func (c *counter) add(r []sensors.Reading) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.batches = append(c.batches, len(r))
	c.total += len(r)
	c.last = r[len(r)-1]
}

func (c *counter) snapshot() (batches []int, total int, last sensors.Reading) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]int(nil), c.batches...), c.total, c.last
}

func send(accel, rot []float64, interval float64) {
	args := []any{nil, nil, interval}
	if accel != nil {
		args[0] = toAny(accel)
	}
	if rot != nil {
		args[1] = toAny(rot)
	}
	js.Global().Call("motion", args...)
}

func toAny(v []float64) any {
	out := make([]any, len(v))
	for i, x := range v {
		out[i] = x
	}
	return out
}

func frames(n int) { time.Sleep(time.Duration(n) * 40 * time.Millisecond) }

// waitTotal waits until the listener of the counter has had n readings in all (the frame that brings them takes as long as it
// takes), and a little longer, so that a reading that must not come has the time to show.
func waitTotal(c *counter, n int) {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, total, _ := c.snapshot(); total >= n {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	frames(3)
}

// observe waits for the app and runs the mode. It runs on a goroutine of its own: Open may wait for the user.
func observe(p *probe, o *observations) {
	<-p.ready
	installScripts()
	sys := sensors.ForBinding(p.binding)
	o.Available = sys.Available()
	switch o.Mode {
	case "none":
		o.Stage = "none"
		js.Global().Set("DeviceMotionEvent", js.Undefined())
		o.HasAccel, o.HasGyro = sys.Has(sensors.Accelerometer), sys.Has(sensors.Gyroscope)
		_, err := sys.Open(context.Background(), sensors.Accelerometer)
		o.AccelOpenTag, o.AccelOpenRoute = tagOf(err), routeOf(err)
	case "motion":
		motion(o, p, sys)
	case "permission":
		permission(o, sys)
	}
}

func motion(o *observations, p *probe, sys *sensors.System) {
	o.HasAccel, o.HasGyro = sys.Has(sensors.Accelerometer), sys.Has(sensors.Gyroscope)
	ctx := context.Background()
	accel, err := sys.Open(ctx, sensors.Accelerometer)
	o.AccelOpenTag = tagOf(err)
	gyro, err2 := sys.Open(ctx, sensors.Gyroscope)
	o.GyroOpenTag = tagOf(err2)
	if err != nil || err2 != nil {
		return
	}
	var a, g counter
	accel.AddListener(a.add)
	gyro.AddListener(g.add)

	o.Stage = "first event"
	// One event: a phone lying flat on its back, turning slowly.
	send([]float64{0.5, -9.8, 1.5}, []float64{90, 180, -45}, 16)
	waitTotal(&a, 1)
	if r, ok := accel.Latest(); ok {
		o.AccelLatest, o.AccelTimestamp = r.Values, float64(r.Timestamp)/float64(time.Millisecond)
	}
	if r, ok := gyro.Latest(); ok {
		o.GyroLatest = r.Values
	}
	o.AccelRate, o.GyroRateAfter = accel.Rate(), gyro.Rate()

	o.Stage = "three events"
	// Three events in a row, before any frame can run: one batch of three for each sensor.
	send([]float64{1, 1, 1}, []float64{1, 1, 1}, 16)
	send([]float64{2, 2, 2}, []float64{2, 2, 2}, 16)
	send([]float64{3, 3, 3}, []float64{3, 3, 3}, 16)
	waitTotal(&a, 4)
	var last3 sensors.Reading
	o.AccelBatches, _, last3 = a.snapshot()
	o.AccelLast3 = last3.Values
	o.GyroBatches, _, _ = g.snapshot()

	o.Stage = "no rotation"
	// An event with only the acceleration (a device with no gyroscope): the gyroscope hears nothing.
	_, ag0, _ := a.snapshot()
	_, gg0, _ := g.snapshot()
	send([]float64{4, 4, 4}, nil, 16)
	waitTotal(&a, ag0+1)
	_, ag1, _ := a.snapshot()
	_, gg1, _ := g.snapshot()
	o.AccelCountNoRot, o.GyroCountNoRot = ag1-ag0, gg1-gg0

	o.Stage = "closing the gyroscope"
	// The sensors are closed one after the other: the one that is closed hears nothing more, the other goes on.
	if err := gyro.Close(); err != nil {
		o.CloseErrors = append(o.CloseErrors, tagOf(err))
	}
	_, gg2, _ := g.snapshot()
	_, ag2, _ := a.snapshot()
	send([]float64{5, 5, 5}, []float64{5, 5, 5}, 16)
	waitTotal(&a, ag2+1)
	_, gg3, _ := g.snapshot()
	_, ag3, _ := a.snapshot()
	o.GyroAfterClose, o.AccelAfterClose = gg3-gg2, ag3-ag2
	o.Stage = "closing the accelerometer"
	if err := accel.Close(); err != nil {
		o.CloseErrors = append(o.CloseErrors, tagOf(err))
	}
	send([]float64{6, 6, 6}, []float64{6, 6, 6}, 16)
	frames(3)

	o.Stage = "haptics"
	// Haptics: every kind, once.
	h := p.binding.HapticFeedback()
	h.Vibrate()
	h.LightImpact()
	h.MediumImpact()
	h.HeavyImpact()
	h.SelectionClick()
	frames(2)
	v := js.Global().Get("vibrations")
	for i := 0; i < v.Length(); i++ {
		o.Vibrations = append(o.Vibrations, v.Index(i).Int())
	}
}

func permission(o *observations, sys *sensors.System) {
	answer := os.Getenv("GLUTTER_SENSOR_ANSWER")
	g := js.Global()
	needsGesture := os.Getenv("GLUTTER_SENSOR_GESTURE") != "0"
	g.Call("fakePermission", answer, needsGesture)
	type result struct {
		s   *sensors.Sensor
		err error
	}
	done := make(chan result, 1)
	go func() {
		s, err := sys.Open(context.Background(), sensors.Accelerometer)
		done <- result{s, err}
	}()
	time.Sleep(500 * time.Millisecond)
	o.AskedBeforeGesture = g.Get("permissionCalls").Int()
	o.RejectedBeforeGesture = g.Get("permissionRejects").Int()
	var r result
	select {
	case r = <-done:
		o.OpenedBeforeGesture = true
	default:
		// A gesture: the page asks again, in it, and the answer comes.
		g.Set("gestureActive", true)
		g.Get("document").Call("dispatchEvent", g.Get("Event").New("pointerdown"))
		g.Set("gestureActive", false)
		r = <-done
	}
	o.AccelOpenTag, o.AccelOpenRoute = tagOf(r.err), routeOf(r.err)
	o.AskedAfterGesture = g.Get("permissionCalls").Int()
	if r.err != nil {
		return
	}
	var c counter
	r.s.AddListener(c.add)
	send([]float64{0, 0, 9.8}, nil, 16)
	waitTotal(&c, 1)
	_, total, _ := c.snapshot()
	o.GotReading = total == 1

	// A second Open does not ask again.
	s2, err := sys.Open(context.Background(), sensors.Gyroscope)
	o.SecondOpenTag = tagOf(err)
	o.AskedAfterSecond = g.Get("permissionCalls").Int()
	if s2 != nil {
		_ = s2.Close() // the program is ending: nothing to report
	}
	_ = r.s.Close() // the program is ending: nothing to report
}

// postResult posts a result to the test server; the answer is not waited for.
func postResult(kind string, data any) {
	body, err := json.Marshal(map[string]any{"kind": kind, "data": data})
	if err != nil {
		panic(err)
	}
	init := map[string]any{"method": "POST", "headers": map[string]any{"Content-Type": "application/json"}, "body": string(body)}
	js.Global().Call("fetch", web.ResultsPath, init)
}
