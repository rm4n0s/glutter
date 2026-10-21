package explicit_test

import (
	"image/color"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/ui"
)

var window = gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 1}

// newRenderer makes a renderer over a fake device; opened counts how many devices it
// opened.
func newRenderer(t *testing.T, opts ...explicit.RendererOpts) (*explicit.Renderer, *fakedevice.Device, *int) {
	t.Helper()
	dev := fakedevice.New()
	dev.Name = "test GPU"
	dev.CompleteReadsOnSubmit = true
	opened := 0
	r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { opened++; return dev, nil }, opts...)
	return r, dev, &opened
}

func TestDrawStagesAFrame(t *testing.T) {
	r, dev, opened := newRenderer(t, explicit.RendererOpts{Background: ui.NewColor(0xFF6750A4), HasBackground: true})
	img, err := r.Draw(window, 40, 30, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if *opened != 1 {
		t.Errorf("the device was opened %d times", *opened)
	}
	want := []string{
		"NewSampler", "NewSwapchain", "Configure", "NewTexture", // the first frame: the sampler, the window's swapchain, the frame texture
		"Begin", "Acquire", "NewPipeline", "BeginPass", "EndPass", // the frame texture is cleared to the background
		"NewBindGroup", "BeginPass", "SetPipeline", "SetBindGroup", "Draw", "EndPass", // and blitted into the swapchain image
		"ReadTexture", "Submit", "Present",
	}
	if got := dev.Ops(); !reflect.DeepEqual(got, want) {
		t.Errorf("ops\n got %v\nwant %v", got, want)
	}
	// The capture is the frame texture, top row first, in the background color.
	if img == nil || img.Rect.Dx() != 40 || img.Rect.Dy() != 30 {
		t.Fatalf("capture = %v", img)
	}
	if got := img.RGBAAt(5, 5); got != (color.RGBA{0x67, 0x50, 0xA4, 0xFF}) {
		t.Errorf("pixel = %v, want #6750A4", got)
	}
	var draws, passes []string
	for _, c := range dev.Calls() {
		switch c.Op {
		case "Draw":
			draws = append(draws, c.Args)
		case "BeginPass":
			passes = append(passes, c.Args)
		}
	}
	if len(draws) != 1 || !strings.HasPrefix(draws[0], "vertices=3 instances=1") {
		t.Errorf("draws = %v, want one blit of 3 vertices", draws)
	}
	if len(passes) != 2 {
		t.Errorf("passes = %v", passes)
	}
	if got := r.Describe(); got != "fake: test GPU" {
		t.Errorf("Describe = %q", got)
	}
}

func TestSecondFrameReusesEverything(t *testing.T) {
	r, dev, opened := newRenderer(t)
	if _, err := r.Draw(window, 16, 16, nil, false); err != nil {
		t.Fatal(err)
	}
	first := len(dev.Calls())
	if _, err := r.Draw(window, 16, 16, nil, false); err != nil {
		t.Fatal(err)
	}
	var second []string
	for _, c := range dev.Calls()[first:] {
		second = append(second, c.Op)
	}
	// No new texture, pipeline, sampler or swapchain; the configure is a call that the swapchain
	// answers without doing anything.
	for _, op := range []string{"NewTexture", "NewPipeline", "NewSampler", "NewSwapchain"} {
		if slices.Contains(second, op) {
			t.Errorf("the second frame made a resource: %s (%v)", op, second)
		}
	}
	if *opened != 1 {
		t.Errorf("the device was opened %d times", *opened)
	}
}

func TestResizeMakesANewFrameTexture(t *testing.T) {
	r, dev, _ := newRenderer(t)
	for _, size := range [][2]int{{16, 16}, {32, 24}} {
		if _, err := r.Draw(window, size[0], size[1], nil, false); err != nil {
			t.Fatal(err)
		}
	}
	var configures, textures []string
	for _, c := range dev.Calls() {
		switch c.Op {
		case "Configure":
			configures = append(configures, c.Args)
		case "NewTexture":
			textures = append(textures, c.Args)
		}
	}
	if !reflect.DeepEqual(configures, []string{"16x16", "32x24"}) {
		t.Errorf("configures = %v", configures)
	}
	if len(textures) != 2 || !strings.HasPrefix(textures[1], "32x24") {
		t.Errorf("textures = %v, want one for each size", textures)
	}
}

func TestOutOfDateConfiguresAgainAndRetriesOnce(t *testing.T) {
	r, dev, _ := newRenderer(t)
	dev.AcquireErrs = []error{errors.New(explicit.SwapchainOutOfDate, "old")}
	if _, err := r.Draw(window, 16, 16, nil, false); err != nil {
		t.Fatal(err)
	}
	var configures int
	for _, op := range dev.Ops() {
		if op == "Configure" {
			configures++
		}
	}
	if configures != 2 {
		t.Errorf("Configure ran %d times, want the first and one after the out-of-date", configures)
	}
	if got := dev.Ops()[len(dev.Ops())-1]; got != "Present" {
		t.Errorf("the frame ends with %s, want Present", got)
	}

	// Out of date twice in a row is an error, and the frame that was begun is closed.
	r2, dev2, _ := newRenderer(t)
	dev2.AcquireErrs = []error{errors.New(explicit.SwapchainOutOfDate, "old"), errors.New(explicit.SwapchainOutOfDate, "still old")}
	_, err := r2.Draw(window, 16, 16, nil, false)
	if !explicit.IsSwapchainOutOfDate(err) {
		t.Fatalf("got %v, want the out-of-date error", err)
	}
	ops := dev2.Ops()
	if ops[len(ops)-1] != "Submit" || slices.Contains(ops, "Present") {
		t.Errorf("a failed frame must end with Submit and no Present: %v", ops)
	}
}

func TestOtherAcquireErrorsAreReturned(t *testing.T) {
	r, dev, _ := newRenderer(t)
	dev.AcquireErrs = []error{errors.New("VulkanSwapchainFailed", "no image")}
	_, err := r.Draw(window, 16, 16, nil, false)
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "VulkanSwapchainFailed" {
		t.Errorf("got %v, want VulkanSwapchainFailed", err)
	}
}

func TestPresentDamage(t *testing.T) {
	present := func(r *explicit.Renderer, dev *fakedevice.Device) string {
		t.Helper()
		if _, err := r.Draw(window, 128, 128, ui.NewSceneBuilder().Build(), false); err != nil {
			t.Fatal(err)
		}
		calls := dev.Calls()
		for i := len(calls) - 1; i >= 0; i-- {
			if calls[i].Op == "Present" {
				return calls[i].Args
			}
		}
		t.Fatal("no Present")
		return ""
	}
	t.Run("without damage the frame is presented whole", func(t *testing.T) {
		r, dev, _ := newRenderer(t)
		present(r, dev)
		if got := present(r, dev); got != "damage=[]" {
			t.Errorf("Present(%s), want none", got)
		}
	})
	t.Run("the first frame is whole, an unchanged one is a pixel", func(t *testing.T) {
		r, dev, _ := newRenderer(t, explicit.RendererOpts{Damage: true})
		if got := present(r, dev); got != "damage=[]" {
			t.Errorf("first frame: Present(%s), want the whole window", got)
		}
		if got := present(r, dev); got != "damage=[(0,0)-(1,1)]" {
			t.Errorf("unchanged frame: Present(%s), want a pixel", got)
		}
	})
	t.Run("an invalidated window is presented whole again", func(t *testing.T) {
		r, dev, _ := newRenderer(t, explicit.RendererOpts{Damage: true})
		present(r, dev)
		present(r, dev)
		r.Invalidate(window)
		if got := present(r, dev); got != "damage=[]" {
			t.Errorf("Present(%s), want the whole window", got)
		}
		r.Invalidate(gpu.SurfaceHandle{Window: 99}) // a window that was never drawn is ignored
	})
}

func TestSeveralWindowsShareTheDevice(t *testing.T) {
	r, dev, opened := newRenderer(t)
	other := gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 2}
	for _, h := range []gpu.SurfaceHandle{window, other, window} {
		if _, err := r.Draw(h, 16, 16, nil, false); err != nil {
			t.Fatal(err)
		}
	}
	count := map[string]int{}
	for _, op := range dev.Ops() {
		count[op]++
	}
	if *opened != 1 || count["NewSwapchain"] != 2 || count["NewTexture"] != 2 || count["NewPipeline"] != 1 || count["Present"] != 3 {
		t.Errorf("opened %d, swapchains %d, frame textures %d, blit pipelines %d, presents %d; want 1, 2, 2, 1, 3",
			*opened, count["NewSwapchain"], count["NewTexture"], count["NewPipeline"], count["Present"])
	}
	r.Release(other)
	r.Release(other) // twice is safe
	if count := countOp(dev, "DestroySwapchain"); count != 1 {
		t.Errorf("%d swapchains destroyed after releasing one window", count)
	}
}

func countOp(dev *fakedevice.Device, op string) int {
	n := 0
	for _, o := range dev.Ops() {
		if o == op {
			n++
		}
	}
	return n
}

func TestCloseDestroysEverythingOnce(t *testing.T) {
	r, dev, _ := newRenderer(t)
	if _, err := r.Draw(window, 16, 16, nil, false); err != nil {
		t.Fatal(err)
	}
	r.Close()
	r.Close()
	if !dev.Closed() || countOp(dev, "DestroySwapchain") != 1 || countOp(dev, "Close") != 1 {
		t.Errorf("closed %v, swapchains destroyed %d, Close calls %d", dev.Closed(), countOp(dev, "DestroySwapchain"), countOp(dev, "Close"))
	}
	if _, err := r.Draw(window, 16, 16, nil, false); err == nil {
		t.Error("Draw after Close succeeded")
	} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "RendererClosed" || !appErr.HasRoute("Renderer.Draw.RendererClosed") {
		t.Errorf("got %v, want RendererClosed via Renderer.Draw", err)
	}
}

func TestDrawErrors(t *testing.T) {
	t.Run("a size that is not positive", func(t *testing.T) {
		r, _, _ := newRenderer(t)
		_, err := r.Draw(window, 0, 10, nil, false)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "RendererSizeInvalid" || !appErr.HasRoute("Renderer.Draw.RendererSizeInvalid") {
			t.Errorf("got %v, want RendererSizeInvalid via Renderer.Draw", err)
		}
	})
	t.Run("the device cannot be opened", func(t *testing.T) {
		want := errors.New("VulkanNoDevice", "no device")
		r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { return nil, want })
		_, err := r.Draw(window, 8, 8, nil, false)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "VulkanNoDevice" || !appErr.HasRoute("VulkanNoDevice") {
			t.Errorf("got %v, want the error of open, unchanged", err)
		}
		r.Close() // the renderer has no device: closing is safe
	})
	t.Run("the swapchain cannot be made", func(t *testing.T) {
		r, dev, _ := newRenderer(t)
		// NewSampler is the first call, then NewSwapchain: the error is for the second.
		if _, err := r.Draw(window, 8, 8, nil, false); err != nil {
			t.Fatal(err)
		}
		dev.Err = errors.New("VulkanSurfaceFailed", "no surface")
		_, err := r.Draw(gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 5}, 8, 8, nil, false)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "VulkanSurfaceFailed" {
			t.Errorf("got %v, want VulkanSurfaceFailed", err)
		}
	})
	t.Run("a device that does not deliver the capture", func(t *testing.T) {
		r, dev, _ := newRenderer(t)
		dev.CompleteReadsOnSubmit = false
		_, err := r.Draw(window, 8, 8, nil, true)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "RendererCaptureFailed" || !appErr.HasRoute("Renderer.frame.RendererCaptureFailed") {
			t.Errorf("got %v, want RendererCaptureFailed via Renderer.frame", err)
		}
	})
	t.Run("more than one options value", func(t *testing.T) {
		defer func() {
			if appErr, ok := errors.FromError(recoverError(recover())); !ok || appErr.Tag != "MultipleOptions" || !appErr.HasRoute("NewRenderer.MultipleOptions") {
				t.Errorf("expected MultipleOptions via NewRenderer, got %v", appErr)
			}
		}()
		explicit.NewRenderer(nil, explicit.RendererOpts{}, explicit.RendererOpts{})
	})
}

func recoverError(r any) error {
	if e, ok := r.(error); ok {
		return e
	}
	return nil
}
