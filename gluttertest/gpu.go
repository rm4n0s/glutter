package gluttertest

import (
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/internal/gpuhost"
	"github.com/rm4n0s/glutter/internal/renderhost"
)

// GPURecorder is the GPU device of a tester made with Opts.GPU: a recording device that draws nothing and keeps
// the log of what the app asked of it, so that the widget tests of an app that uses package gpu (and, later,
// gpu.View and viewport3d) run with no GPU. A test checks what was encoded, not pixels. gpu.DeviceOf gives the app
// this device, from the start (a real renderer opens its device at the first frame).
//
// Flutter: no counterpart
type GPURecorder struct {
	*gpuhost.Host
	fake     *fakedevice.Device
	features explicit.Features
}

func newGPURecorder() *GPURecorder {
	r := &GPURecorder{fake: fakedevice.New()}
	r.fake.Name = "gluttertest"
	r.Host = gpuhost.New(r.fake)
	return r
}

// Device returns the recording device of the moment. After Lose it is the new one.
//
// Flutter: no counterpart
func (r *GPURecorder) Device() *fakedevice.Device { return r.fake }

// SupportTimestamps makes the device report Features.TimestampQuery with the period of one tick in nanoseconds.
// Call it before the app asks for the device.
//
// Flutter: no counterpart
func (r *GPURecorder) SupportTimestamps(periodNanoseconds float64) {
	r.features = explicit.Features{TimestampQuery: true, TimestampPeriod: periodNanoseconds}
	r.fake.Supported = r.features
}

// Calls returns the calls the device received, oldest first, as "Op(args)".
//
// Flutter: no counterpart
func (r *GPURecorder) Calls() []string {
	calls := r.fake.Calls()
	out := make([]string, len(calls))
	for i, c := range calls {
		out[i] = c.String()
	}
	return out
}

// Ops returns the names of the calls, without their arguments.
//
// Flutter: no counterpart
func (r *GPURecorder) Ops() []string { return r.fake.Ops() }

// Reset forgets the calls, so that a test checks the ones that follow.
//
// Flutter: no counterpart
func (r *GPURecorder) Reset() { r.fake.Reset() }

// CompleteReads finishes the reads and the submissions that wait for the GPU, as a GPU that caught up does: the
// functions of Buffer.Read and Queue.OnSubmittedWorkDone run on the next Pump.
//
// Flutter: no counterpart
func (r *GPURecorder) CompleteReads() { r.fake.CompleteReads() }

// Lose loses the device, as a driver reset does, and opens a new one at once, as the renderer does (SPEC §9.4,
// §9.5). The app's gpu.Device is lost (its OnLost functions run on the next Pump) and gpu.DeviceOf gives the new one.
//
// Flutter: no counterpart
func (r *GPURecorder) Lose() {
	fresh := fakedevice.New()
	fresh.Name = r.fake.Name
	fresh.Supported = r.features
	r.fake = fresh
	r.Replace(fresh)
}

// EndApp ends the app as the engine does when its windows are gone: the device is closed, which makes the gpu
// package report the resources that were not released, and the watchers are told. The tester does it when the
// test ends.
//
// Flutter: no counterpart
func (r *GPURecorder) EndApp() { r.End() }

// gpuHost is the fake host with a GPU device.
type gpuHost struct {
	*renderhost.Host
	*GPURecorder
}

// gpuSpellHost is the fake host with a spell checker and a GPU device.
type gpuSpellHost struct {
	*spellCheckHost
	*GPURecorder
}

var (
	_ explicit.DeviceHost = (*gpuHost)(nil)
	_ explicit.DeviceHost = (*gpuSpellHost)(nil)
)

// GPUDevice returns the recording GPU device of the tester; nil when Opts.GPU was not set. Without it the app has
// no GPU device, as with the software renderer: gpu.DeviceOf fails with GPUUnavailable.
//
// Flutter: no counterpart
func (t *Tester) GPUDevice() *GPURecorder { return t.gpu }
