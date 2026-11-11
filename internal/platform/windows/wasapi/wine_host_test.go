//go:build linux && glutter_wine && glutter_audio

package wasapi_test

// The test of the host: a Windows program (wine_child_test.go) plays through WASAPI under Wine, whose audio goes to the private
// server of scripts/headless-audio.sh, and the host records the monitor of the sink and checks the sound, and what happens when
// the default device changes or a device goes away (SPEC §28.8). Run it with
//
//	scripts/headless-audio.sh -- go test -tags 'glutter_wine glutter_audio' -p 1 ./internal/platform/windows/wasapi

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/pulse"
	"github.com/rm4n0s/glutter/internal/winetest"
)

const (
	toneHz = 440.0
	rate   = 48000
)

func requireHostServer(t *testing.T) {
	t.Helper()
	if os.Getenv("GLUTTER_AUDIO_SERVER_RUNTIME") == "" {
		t.Skip("no private audio server: run this with scripts/headless-audio.sh -- go test ...")
	}
}

// recorder keeps the left channel of a stereo capture.
type recorder struct {
	mu      sync.Mutex
	samples []float32
}

func (r *recorder) Fill([]float32) {}
func (r *recorder) Capture(in []float32) {
	r.mu.Lock()
	for i := 0; i+2 <= len(in); i += 2 {
		r.samples = append(r.samples, in[i])
	}
	r.mu.Unlock()
}
func (r *recorder) StateChanged(platform.AudioState) {}

// take returns what was recorded since the last call.
func (r *recorder) take() []float32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.samples
	r.samples = nil
	return out
}

// toneOf fits a sine of toneHz and returns its amplitude and the share of the energy that it does not explain.
func toneOf(samples []float32) (amp, residual float64) {
	var i, q, energy float64
	for n, v := range samples {
		ph := 2 * math.Pi * toneHz * float64(n) / rate
		i += float64(v) * math.Cos(ph)
		q += float64(v) * math.Sin(ph)
		energy += float64(v) * float64(v)
	}
	n := float64(len(samples))
	if n == 0 || energy == 0 {
		return 0, 1
	}
	amp = 2 * math.Hypot(i, q) / n
	return amp, (energy - amp*amp/2*n) / energy
}

func pactl(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("pactl", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("pactl %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// child is a Windows program under Wine.
type child struct {
	dir    string
	done   chan struct{}
	output string
	status int
	err    error
}

// startChild builds the test binary for Windows, starts it with the role and waits until it plays.
func startChild(t *testing.T, role string, env ...string) *child {
	t.Helper()
	h, err := winetest.New("")
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(t.TempDir(), "wasapi.test.exe")
	build := exec.Command("go", "test", "-c", "-tags", "glutter_wine glutter_audio debug_mode", "-o", exe, "./internal/platform/windows/wasapi")
	build.Dir = h.Root
	build.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go test -c: %v\n%s", err, out)
	}
	c := &child{dir: t.TempDir(), done: make(chan struct{})}
	go func() {
		defer close(c.done)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		res, err := h.Run(ctx, exe, winetest.RunOptions{
			Args:    []string{"-test.run", "^TestWineChild$", "-test.v", "-test.count=1"},
			Env:     append([]string{"GLUTTER_WASAPI_CHILD=" + role, "GLUTTER_WASAPI_SYNC=Z:" + c.dir}, env...),
			Timeout: 4 * time.Minute,
		})
		c.output, c.status, c.err = res.Output, res.ExitCode, err
	}()
	t.Cleanup(func() {
		c.stop()
		if left := winetest.Leftovers(h.Root); len(left) > 0 {
			t.Errorf("processes left running:\n%s", strings.Join(left, "\n"))
		}
	})
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if _, err := os.Stat(filepath.Join(c.dir, "ready")); err == nil {
			return c
		}
		select {
		case <-c.done:
			t.Fatalf("the program ended before it played: %v, status %d\n%s", c.err, c.status, c.output)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("the program did not start to play in two minutes")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// stop tells the program to end, waits for it and returns the states it saw.
func (c *child) stop() []string {
	if err := os.WriteFile(filepath.Join(c.dir, "stop"), nil, 0o600); err != nil {
		return nil
	}
	<-c.done
	b, err := os.ReadFile(filepath.Join(c.dir, "states"))
	if err != nil || len(b) == 0 {
		return nil
	}
	return strings.Split(string(b), "\n")
}

// monitor records a monitor of the server.
func monitor(t *testing.T, svc *pulse.Service, name string) *recorder {
	t.Helper()
	r := &recorder{}
	dev, err := svc.Open(context.Background(), platform.AudioCapture, platform.AudioOpenOptions{ID: name, Format: platform.AudioFormat{Rate: rate, Channels: 2}}, r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dev.Close() })
	return r
}

func TestWineToneIsHeardOnTheMonitor(t *testing.T) {
	requireHostServer(t)
	svc := pulse.New(pulse.Options{AppName: "wasapi-host-test"})
	rec := monitor(t, svc, "glutter_sink.monitor")
	c := startChild(t, "tone")
	rec.take()
	time.Sleep(1500 * time.Millisecond)
	got := rec.take()
	if len(got) < rate {
		t.Fatalf("%d frames recorded in a second and a half", len(got))
	}
	// The first part may hold the start of the stream; the tone is checked on the rest.
	got = got[rate/4:]
	amp, residual := toneOf(got)
	t.Logf("amplitude %.3f, residual %.5f over %d frames", amp, residual, len(got))
	if math.Abs(amp-0.5) > 0.03 {
		t.Errorf("the amplitude is %.3f, want 0.5", amp)
	}
	if residual > 0.01 {
		t.Errorf("the tone is not pure: %.4f of the energy is something else", residual)
	}
	jumps := 0
	limit := float32(1.5 * 0.5 * 2 * math.Pi * toneHz / rate)
	for i := 1; i < len(got); i++ {
		if d := got[i] - got[i-1]; d > limit || -d > limit {
			jumps++
		}
	}
	if jumps != 0 {
		t.Errorf("%d jumps in the tone", jumps)
	}
	c.stop()
	if c.status != 0 || c.err != nil {
		t.Errorf("the program: %v, status %d\n%s", c.err, c.status, c.output)
	}
}

// secondSink loads a second null sink into the server and unloads it at the end of the test. unload ends it earlier.
func secondSink(t *testing.T) (unload func()) {
	t.Helper()
	module := pactl(t, "load-module", "module-null-sink", "sink_name=glutter_second", "sink_properties=device.description=Glutter-second")
	unloaded := false
	unload = func() {
		if !unloaded {
			unloaded = true
			pactl(t, "unload-module", module)
		}
	}
	t.Cleanup(unload)
	return unload
}

// eventuallyHears waits until the recording has a chunk of 100 ms in which the tone is as loud as it should be.
func eventuallyHears(t *testing.T, what string, rec *recorder, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		got := rec.take()
		if len(got) >= rate/10 {
			if amp, _ := toneOf(got); amp > 0.4 {
				return
			}
		}
	}
	t.Fatalf("%s: the tone was not heard in %v", what, timeout)
}

// silent says whether the recording of the last 200 ms is silent.
func silent(rec *recorder) bool {
	time.Sleep(200 * time.Millisecond)
	for _, v := range rec.take() {
		if v > 0.01 || v < -0.01 {
			return false
		}
	}
	return true
}

// sinkIndex is the index that the server gives a sink, as a string.
func sinkIndex(t *testing.T, name string) string {
	t.Helper()
	for _, line := range strings.Split(pactl(t, "list", "short", "sinks"), "\n") {
		if f := strings.Fields(line); len(f) > 1 && f[1] == name {
			return f[0]
		}
	}
	t.Fatalf("no sink %s", name)
	return ""
}

// sinkOfTheInput is the index of the sink that the one playing stream is on.
func sinkOfTheInput(t *testing.T) string {
	t.Helper()
	lines := strings.Split(pactl(t, "list", "short", "sink-inputs"), "\n")
	if len(lines) != 1 {
		t.Fatalf("%d streams play, want one: %v", len(lines), lines)
	}
	return strings.Fields(lines[0])[1]
}

func TestWineFollowsTheDefaultDevice(t *testing.T) {
	requireHostServer(t)
	secondSink(t)
	svc := pulse.New(pulse.Options{AppName: "wasapi-host-test"})
	first := monitor(t, svc, "glutter_sink.monitor")
	second := monitor(t, svc, "glutter_second.monitor")
	pactl(t, "set-default-sink", "glutter_sink")
	t.Cleanup(func() { pactl(t, "set-default-sink", "glutter_sink") })
	c := startChild(t, "tone")
	eventuallyHears(t, "on the first sink", first, 10*time.Second)
	if !silent(second) {
		t.Fatal("the second sink hears the tone before the default changed")
	}
	pactl(t, "set-default-sink", "glutter_second")
	eventuallyHears(t, "on the second sink after the default changed", second, 15*time.Second)
	// The monitor of a sink that nothing plays on keeps the last buffer it had, so the stream's own sink is what is checked.
	if got, want := sinkOfTheInput(t), sinkIndex(t, "glutter_second"); got != want {
		t.Errorf("the stream plays on sink %s, want %s", got, want)
	}
	pactl(t, "set-default-sink", "glutter_sink")
	eventuallyHears(t, "on the first sink again", first, 15*time.Second)
	if got, want := sinkOfTheInput(t), sinkIndex(t, "glutter_sink"); got != want {
		t.Errorf("the stream plays on sink %s, want %s", got, want)
	}
	states := c.stop()
	if len(states) != 0 {
		t.Errorf("the device saw the states %v while it followed the default", states)
	}
	t.Logf("the program's output:\n%s", c.output)
}

func TestWineLosesADeviceThatGoesAway(t *testing.T) {
	requireHostServer(t)
	unload := secondSink(t)
	svc := pulse.New(pulse.Options{AppName: "wasapi-host-test"})
	second := monitor(t, svc, "glutter_second.monitor")
	c := startChild(t, "tone", "GLUTTER_WASAPI_DEVICE=Glutter-second")
	eventuallyHears(t, "on the second sink", second, 10*time.Second)
	unload()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-c.done:
			t.Fatalf("the program ended: %v, status %d\n%s", c.err, c.status, c.output)
		default:
		}
		time.Sleep(200 * time.Millisecond)
		if strings.Contains(c.output, "lost") {
			break
		}
		// The program writes the states when it is told to stop; look at the log in the meantime.
		break
	}
	time.Sleep(3 * time.Second)
	states := c.stop()
	t.Logf("the program's output:\n%s", c.output)
	if len(states) != 1 || states[0] != "lost" {
		t.Errorf("the device saw the states %v, want one Lost", states)
	}
}
