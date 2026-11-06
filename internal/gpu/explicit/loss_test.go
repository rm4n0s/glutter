package explicit_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/ui"
)

// lossRig is a renderer over a fake device that can report a loss, with a clock the test moves.
type lossRig struct {
	r    *explicit.Renderer
	devs []*fakedevice.Device
	now  time.Time
}

func newLossRig(t *testing.T) *lossRig {
	t.Helper()
	rig := &lossRig{now: time.Unix(1000, 0)}
	rig.r = explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) {
		d := fakedevice.New()
		d.CompleteReadsOnSubmit = true
		rig.devs = append(rig.devs, d)
		return d, nil
	}, explicit.RendererOpts{Now: func() time.Time { return rig.now }})
	return rig
}

func (g *lossRig) last() *fakedevice.Device { return g.devs[len(g.devs)-1] }

func (g *lossRig) lose() {
	g.last().Err = errors.New(tagVulkanDeviceLost, "the Vulkan device was lost", "call", "vkQueueSubmit")
}

func TestDeviceLostMakesEverythingAgainOnce(t *testing.T) {
	scene := sceneOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000)) })
	g := newLossRig(t)
	if _, err := g.r.Draw(window, 32, 32, scene, true); err != nil {
		t.Fatal(err)
	}
	// The first loss: a new device is opened, the old one closed, and the frame is drawn again.
	g.now = g.now.Add(time.Minute)
	g.lose()
	img, err := g.r.Draw(window, 32, 32, scene, true)
	if err != nil || img == nil {
		t.Fatalf("the frame after a first loss: %v, %v", img, err)
	}
	if len(g.devs) != 2 || !g.devs[0].Closed() || g.devs[1].Closed() {
		t.Fatalf("%d devices; first closed %v", len(g.devs), g.devs[0].Closed())
	}
	if n := len(calls(g.devs[1], "NewSwapchain")); n != 1 {
		t.Errorf("the new device made %d swapchains, want the window's one", n)
	}
	if st := g.r.LastFrame(); st.Instances != 1 || st.Skipped != 0 {
		t.Errorf("stats %+v: the frame is drawn whole on the new device", st)
	}
	// A second loss within ten seconds ends the app with the error.
	g.now = g.now.Add(9 * time.Second)
	g.lose()
	_, err = g.r.Draw(window, 32, 32, scene, true)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagVulkanDeviceLost || !appErr.HasRoute(tagVulkanDeviceLost) {
		t.Fatalf("got %v, want VulkanDeviceLost", err)
	}
	if len(g.devs) != 2 {
		t.Errorf("a second loss within ten seconds opened a device: %d", len(g.devs))
	}
}

func TestDeviceLostAfterTheWindowIsRecovered(t *testing.T) {
	scene := sceneOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000)) })
	g := newLossRig(t)
	if _, err := g.r.Draw(window, 32, 32, scene, false); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		g.now = g.now.Add(11 * time.Second)
		g.lose()
		if _, err := g.r.Draw(window, 32, 32, scene, false); err != nil {
			t.Fatalf("loss %d: %v", i+1, err)
		}
	}
	if len(g.devs) != 4 {
		t.Errorf("%d devices for three losses spaced over ten seconds", len(g.devs))
	}
}

func TestOtherErrorsAreNotDeviceLoss(t *testing.T) {
	scene := sceneOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000)) })
	g := newLossRig(t)
	if _, err := g.r.Draw(window, 32, 32, scene, false); err != nil {
		t.Fatal(err)
	}
	g.last().Err = errors.New(tagGPUOutOfMemory, "out of memory")
	_, err := g.r.Draw(window, 32, 32, scene, false)
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagGPUOutOfMemory {
		t.Fatalf("got %v, want GPUOutOfMemory", err)
	}
	if len(g.devs) != 1 {
		t.Errorf("an out-of-memory error replaced the device")
	}
}

func TestDeviceLostBeforeTheDeviceOpensIsAnError(t *testing.T) {
	r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) {
		return nil, errors.New(tagVulkanDeviceLost, "lost while opening")
	})
	_, err := r.Draw(window, 8, 8, nil, false)
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagVulkanDeviceLost {
		t.Fatalf("got %v", err)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGPUOutOfMemory   = "GPUOutOfMemory"
	tagVulkanDeviceLost = "VulkanDeviceLost"
)

// TestDeviceEventsTellTheOpenTheLossAndTheClose: OnDevice hears each device, in order, before the device is
// closed (SPEC §27.2: package gpu lets go of what it made with a device that is lost).
func TestDeviceEventsTellTheOpenTheLossAndTheClose(t *testing.T) {
	scene := sceneOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000)) })
	type seen struct {
		kind       explicit.DeviceEventKind
		dev        explicit.Device
		closedThen bool
		err        error
	}
	var events []seen
	rig := &lossRig{now: time.Unix(1000, 0)}
	rig.r = explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) {
		d := fakedevice.New()
		d.CompleteReadsOnSubmit = true
		rig.devs = append(rig.devs, d)
		return d, nil
	}, explicit.RendererOpts{Now: func() time.Time { return rig.now }, OnDevice: func(e explicit.DeviceEvent) {
		events = append(events, seen{e.Kind, e.Device, e.Device.(*fakedevice.Device).Closed(), e.Err})
	}})
	if _, err := rig.r.Draw(window, 32, 32, scene, false); err != nil {
		t.Fatal(err)
	}
	rig.now = rig.now.Add(time.Minute)
	rig.lose()
	if _, err := rig.r.Draw(window, 32, 32, scene, false); err != nil {
		t.Fatal(err)
	}
	rig.r.Close()
	want := []struct {
		kind explicit.DeviceEventKind
		dev  int
		err  bool
	}{{explicit.DeviceOpened, 0, false}, {explicit.DeviceLost, 0, true}, {explicit.DeviceOpened, 1, false}, {explicit.DeviceClosed, 1, false}}
	if len(events) != len(want) {
		t.Fatalf("%d events %v, want %d", len(events), events, len(want))
	}
	for i, w := range want {
		e := events[i]
		if e.kind != w.kind || e.dev != explicit.Device(rig.devs[w.dev]) || (e.err != nil) != w.err {
			t.Errorf("event %d = %v on device %v with error %v; want %v on device %d, error %v", i, e.kind, e.dev, e.err, w.kind, w.dev, w.err)
		}
		if e.closedThen {
			t.Errorf("event %d (%v) came after the device was closed", i, e.kind)
		}
	}
	// A renderer that never opened a device tells nothing when it closes.
	events = nil
	idle := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { return fakedevice.New(), nil },
		explicit.RendererOpts{OnDevice: func(e explicit.DeviceEvent) { events = append(events, seen{kind: e.Kind}) }})
	idle.Close()
	if len(events) != 0 {
		t.Errorf("a renderer without a device told %v", events)
	}
}
