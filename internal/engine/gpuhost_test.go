package engine

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
)

func TestGPUHubStates(t *testing.T) {
	dev := fakedevice.New()
	tests := []struct {
		name   string
		chosen bool
		picked string
		events []explicit.DeviceEvent
		want   explicit.HostState
	}{
		{"before the first frame", false, "", nil, explicit.HostPending},
		{"an explicit renderer that opened its device", true, "vulkan", []explicit.DeviceEvent{{Kind: explicit.DeviceOpened, Device: dev}}, explicit.HostReady},
		{"an explicit renderer that lost its device", true, "d3d12", []explicit.DeviceEvent{{Kind: explicit.DeviceOpened, Device: dev}, {Kind: explicit.DeviceLost, Device: dev}}, explicit.HostPending},
		{"WebGPU after its device was lost", true, "webgpu", []explicit.DeviceEvent{{Kind: explicit.DeviceOpened, Device: dev}, {Kind: explicit.DeviceLost, Device: dev}}, explicit.HostPending},
		{"GLES", true, "gles", nil, explicit.HostUnavailable},
		{"WebGL", true, "webgl", nil, explicit.HostUnavailable},
		{"the software renderer", true, "soft", nil, explicit.HostUnavailable},
		{"the engine went on from Vulkan to GLES", true, "gles", []explicit.DeviceEvent{{Kind: explicit.DeviceOpened, Device: dev}, {Kind: explicit.DeviceClosed, Device: dev}}, explicit.HostUnavailable},
		{"a closed device of another renderer is not the current one", true, "vulkan", []explicit.DeviceEvent{{Kind: explicit.DeviceOpened, Device: dev}, {Kind: explicit.DeviceClosed, Device: fakedevice.New()}}, explicit.HostReady},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &gpuHub{renderer: func() (bool, string) { return tt.chosen, tt.picked }}
			for _, e := range tt.events {
				g.notify(e)
			}
			got, state := g.device()
			if state != tt.want {
				t.Fatalf("state = %v, want %v", state, tt.want)
			}
			if (state == explicit.HostReady) != (got != nil) {
				t.Errorf("device = %v in state %v", got, state)
			}
		})
	}
	t.Run("a hub without a renderer function waits", func(t *testing.T) {
		if _, state := (&gpuHub{}).device(); state != explicit.HostPending {
			t.Errorf("state = %v", state)
		}
	})
}

func TestGPUHubWatchers(t *testing.T) {
	g := &gpuHub{}
	var seen []string
	stopA := g.watch(func(e explicit.DeviceEvent) { seen = append(seen, "a:"+e.Kind.String()) })
	stopB := g.watch(func(e explicit.DeviceEvent) { seen = append(seen, "b:"+e.Kind.String()) })
	dev := fakedevice.New()
	g.notify(explicit.DeviceEvent{Kind: explicit.DeviceOpened, Device: dev})
	stopA()
	stopA() // twice is harmless
	g.notify(explicit.DeviceEvent{Kind: explicit.DeviceLost, Device: dev})
	stopB()
	g.notify(explicit.DeviceEvent{Kind: explicit.HostEnded})
	want := []string{"a:opened", "b:opened", "b:lost"}
	if !reflect.DeepEqual(seen, want) {
		t.Errorf("seen %v, want %v", seen, want)
	}
	// A watcher may stop itself, or another, while it is told.
	var stop func()
	calls := 0
	stop = g.watch(func(explicit.DeviceEvent) { calls++; stop() })
	g.notify(explicit.DeviceEvent{Kind: explicit.DeviceOpened, Device: dev})
	g.notify(explicit.DeviceEvent{Kind: explicit.DeviceOpened, Device: dev})
	if calls != 1 {
		t.Errorf("a watcher that stopped itself was called %d times", calls)
	}
}

func TestHandlerIsTheDeviceHost(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	h := &handler{log: log}
	h.gpu.renderer = h.chosenRenderer
	var host explicit.DeviceHost = h
	if host.GPULogger() != log {
		t.Error("GPULogger is not the engine's log")
	}
	if _, state := host.GPUDevice(); state != explicit.HostPending {
		t.Errorf("state before the first frame = %v", state)
	}
	h.r = &chooser{cur: nil}
	if _, state := host.GPUDevice(); state != explicit.HostPending {
		t.Errorf("state of a chooser that has not chosen = %v", state)
	}
	h.r = &chooser{cur: &fakeRenderer{}, name: "soft"}
	if _, state := host.GPUDevice(); state != explicit.HostUnavailable {
		t.Errorf("state under the software renderer = %v", state)
	}
	var seen explicit.DeviceEventKind
	stop := host.WatchGPUDevice(func(e explicit.DeviceEvent) { seen = e.Kind })
	h.release()
	stop()
	if seen != explicit.HostEnded {
		t.Errorf("the end of the app told %v, want HostEnded", seen)
	}
}
