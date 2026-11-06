package gpu_test

import (
	"bytes"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

func TestOtherGoroutinesPanic(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the thread is checked in debug builds only")
	}
	tests := []struct {
		name string
		do   func(h *harness)
	}{
		{"DeviceOfBinding", func(h *harness) { _, _ = gpu.DeviceOfBinding(h.ts.Binding) }},
		{"OnDevice", func(h *harness) { gpu.OnDevice(h.ts.Binding, func(*gpu.Device) {}) }},
		{"NewBuffer", func(h *harness) { _, _ = h.d.NewBuffer(gpu.BufferDesc{Size: 4, Usage: gpu.BufferUsageVertex}) }},
		{"NewTexture", func(h *harness) { _, _ = h.d.NewTexture(gpu.TextureDesc{}) }},
		{"NewCommandEncoder", func(h *harness) { h.d.NewCommandEncoder() }},
		{"Release", func(h *harness) {
			b := must(h.d.NewBuffer(gpu.BufferDesc{Size: 4, Usage: gpu.BufferUsageVertex}))
			done := make(chan struct{})
			go func() {
				defer close(done)
				defer func() { h.t.Logf("recovered: %v", recover()) }()
				b.Release()
			}()
			<-done
		}},
		{"Submit", func(h *harness) { _ = h.d.Queue().Submit() }},
		{"OnLost", func(h *harness) { h.d.OnLost(func(error) {}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			var got error
			done := make(chan struct{})
			if tt.name == "Release" {
				// Release starts its own goroutine, the others are started here.
				tt.do(h)
				return
			}
			go func() {
				defer close(done)
				got = panicOf(func() { tt.do(h) })
			}()
			<-done
			if got == nil {
				t.Fatal("no panic")
			}
			wantTag(t, got, gpu.TagGPUWrongThread, "")
		})
	}
}

func TestReleasedResourcesAreCounted(t *testing.T) {
	h := newHarness(t)
	d := h.d
	if n := d.Resources().Total(); n != 0 {
		t.Fatalf("a new device has %d resources", n)
	}
	b := must(d.NewBuffer(gpu.BufferDesc{Size: 16, Usage: gpu.BufferUsageVertex}))
	tex := must(d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 4, Height: 4}, Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding}))
	s := must(d.NewSampler(gpu.SamplerDesc{}))
	m := h.module()
	p := must(d.NewRenderPipeline(h.pipelineDesc(m, nil)))
	c := must(d.NewComputePipeline(gpu.ComputePipelineDesc{Module: m, EntryPoint: "cs"}))
	bg := must(d.NewBindGroup(gpu.BindGroupDesc{Layout: p.BindGroupLayout(1), Entries: []gpu.BindGroupEntry{{Binding: 0, TextureView: tex.View()}, {Binding: 1, Sampler: s}}}))
	want := gpu.ResourceCounts{Buffers: 1, Textures: 1, Samplers: 1, ShaderModules: 1, RenderPipelines: 1, ComputePipelines: 1, BindGroups: 1}
	if got := d.Resources(); got != want {
		t.Fatalf("Resources = %+v, want %+v", got, want)
	}
	live := h.rec.Device().Live()
	if live.Buffers != 1 || live.Textures != 1 || live.Samplers != 1 || live.Pipelines < 1 {
		t.Errorf("the backend has %+v", live)
	}
	for _, r := range []interface{ Release() }{b, tex, s, p, c, bg, m} {
		r.Release()
		r.Release() // twice is fine
	}
	if got := d.Resources(); got != (gpu.ResourceCounts{}) {
		t.Errorf("after the releases Resources = %+v", got)
	}
	live = h.rec.Device().Live()
	if live.Buffers != 0 || live.Textures != 0 || live.Samplers != 0 || live.Pipelines != 0 {
		t.Errorf("the backend still has %+v", live)
	}
	h.ts.Pump()
	if h.rec.LogContains("not released") {
		t.Errorf("log: %s", h.rec.Logs())
	}
}

func TestResourcesNotReleasedAreLoggedWhenTheAppEnds(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the leaks are reported in debug builds only")
	}
	h := newHarness(t)
	h.rec.ClearLogs()
	must(h.d.NewBuffer(gpu.BufferDesc{Label: "leaky vertices", Size: 16, Usage: gpu.BufferUsageVertex}))
	kept := must(h.d.NewBuffer(gpu.BufferDesc{Label: "kept", Size: 16, Usage: gpu.BufferUsageVertex}))
	kept.Release()
	h.rec.EndApp()
	logs := h.rec.Logs()
	for _, want := range []string{"a GPU resource was not released", gpu.TagGPUResourceLeaked, "leaky vertices", "lifetime_test.go"} {
		if !bytes.Contains([]byte(logs), []byte(want)) {
			t.Errorf("the log lacks %q:\n%s", want, logs)
		}
	}
	if bytes.Contains([]byte(logs), []byte("kept")) {
		t.Errorf("a released buffer is reported:\n%s", logs)
	}
	if !h.d.IsLost() {
		t.Errorf("the device of an app that ended works")
	}
}

func TestOnDevice(t *testing.T) {
	t.Run("the device that exists, from the event loop", func(t *testing.T) {
		h := newHarness(t)
		var got []*gpu.Device
		gpu.OnDevice(h.ts.Binding, func(d *gpu.Device) { got = append(got, d) })
		if len(got) != 0 {
			t.Fatalf("OnDevice called fn inside the call")
		}
		h.ts.Pump()
		if len(got) != 1 || got[0] != h.d {
			t.Errorf("got %v, want the device %p", got, h.d)
		}
	})
	t.Run("a cancelled function is not called", func(t *testing.T) {
		h := newHarness(t)
		called := false
		cancel := gpu.OnDevice(h.ts.Binding, func(*gpu.Device) { called = true })
		cancel()
		h.ts.Pump()
		if called {
			t.Errorf("fn was called after cancel")
		}
	})
	t.Run("a new device after a loss", func(t *testing.T) {
		h := newHarness(t)
		var got []*gpu.Device
		gpu.OnDevice(h.ts.Binding, func(d *gpu.Device) { got = append(got, d) })
		h.ts.Pump()
		h.rec.Lose()
		h.ts.Pump()
		if len(got) != 2 || got[0] == got[1] || got[0] != h.d {
			t.Fatalf("got %d devices, the first the old one: %v", len(got), len(got) > 0 && got[0] == h.d)
		}
		if got[1].IsLost() {
			t.Errorf("the new device is lost")
		}
	})
	t.Run("without a GPU fn is not called", func(t *testing.T) {
		ts := newPlainTester(t)
		called := false
		gpu.OnDevice(ts.Binding, func(*gpu.Device) { called = true })
		ts.Pump()
		if called {
			t.Errorf("fn was called")
		}
	})
}

func TestDeviceLoss(t *testing.T) {
	h := newHarness(t)
	old := h.d
	b := must(old.NewBuffer(gpu.BufferDesc{Size: 16, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst}))
	var lost []error
	old.OnLost(func(err error) { lost = append(lost, err) })
	removed := false
	remove := old.OnLost(func(error) { removed = true })
	remove()
	h.rec.Lose()
	if !old.IsLost() {
		t.Fatal("the old device works")
	}
	if len(lost) != 0 {
		t.Fatalf("OnLost ran inside the loss")
	}
	h.ts.Pump()
	if len(lost) != 1 || removed {
		t.Fatalf("OnLost ran %d times (removed one: %v), want once", len(lost), removed)
	}
	wantTag(t, lost[0], gpu.TagGPUDeviceLost, "")
	if !gpu.IsDeviceLost(lost[0]) {
		t.Errorf("IsDeviceLost = false")
	}
	h.ts.Pump()
	if len(lost) != 1 {
		t.Errorf("OnLost ran %d times after another pump", len(lost))
	}

	// What returns an error says the device is lost; the rest does nothing.
	_, err := old.NewBuffer(gpu.BufferDesc{Size: 4, Usage: gpu.BufferUsageVertex})
	wantTag(t, err, gpu.TagGPUDeviceLost, "Device.lose.GPUDeviceLost")
	_, err = old.NewTexture(gpu.TextureDesc{})
	wantTag(t, err, gpu.TagGPUDeviceLost, "")
	_, err = old.NewSampler(gpu.SamplerDesc{})
	wantTag(t, err, gpu.TagGPUDeviceLost, "")
	_, err = old.NewShaderModule(testShader)
	wantTag(t, err, gpu.TagGPUDeviceLost, "")
	_, err = old.NewQuerySet(gpu.QuerySetDesc{Count: 1})
	wantTag(t, err, gpu.TagGPUDeviceLost, "")
	wantTag(t, old.Queue().Submit(), gpu.TagGPUDeviceLost, "Device.lose.GPUDeviceLost")
	var readErr error
	b.Read(0, 16, func(_ []byte, err error) { readErr = err })
	h.ts.Pump()
	wantTag(t, readErr, gpu.TagGPUDeviceLost, "")
	old.Queue().WriteBuffer(b, 0, make([]byte, 4))
	called := false
	old.Queue().OnSubmittedWorkDone(func() { called = true })
	h.ts.Pump()
	if !called {
		t.Errorf("OnSubmittedWorkDone was not called on a lost device")
	}
	b.Release() // does nothing to the backend, which is gone
	again := false
	old.OnLost(func(err error) { again = gpu.IsDeviceLost(err) })
	h.ts.Pump()
	if !again {
		t.Errorf("OnLost on a lost device was not called")
	}

	// The renderer made a new device.
	fresh, err := gpu.DeviceOfBinding(h.ts.Binding)
	if err != nil || fresh == old || fresh.IsLost() {
		t.Fatalf("DeviceOfBinding after the loss = %p, %v; old %p", fresh, err, old)
	}
	if _, err := fresh.NewBuffer(gpu.BufferDesc{Size: 4, Usage: gpu.BufferUsageVertex}); err != nil {
		t.Errorf("NewBuffer on the new device: %v", err)
	}
	if h.rec.LogContains("not released") && !foundation.DebugMode {
		t.Errorf("log: %s", h.rec.Logs())
	}
}

func TestBufferReadAndWrite(t *testing.T) {
	t.Run("a read gives a copy, after the GPU finished", func(t *testing.T) {
		h := newHarness(t)
		b := must(h.d.NewBuffer(gpu.BufferDesc{Size: 16, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst}))
		h.d.Queue().WriteBuffer(b, 0, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
		var got []byte
		var gotErr error
		calls := 0
		b.Read(4, 8, func(data []byte, err error) { got, gotErr, calls = data, err, calls+1 })
		if calls != 0 {
			t.Fatal("done ran inside the call")
		}
		h.ts.Pump()
		if calls != 0 {
			t.Fatal("done ran before the GPU finished")
		}
		h.rec.CompleteReads()
		h.ts.Pump()
		if calls != 1 || gotErr != nil || !bytes.Equal(got, []byte{5, 6, 7, 8, 9, 10, 11, 12}) {
			t.Errorf("done ran %d times with %v, %v", calls, got, gotErr)
		}
	})
	t.Run("a read that failed", func(t *testing.T) {
		h := newHarness(t)
		b := must(h.d.NewBuffer(gpu.BufferDesc{Size: 16, Usage: gpu.BufferUsageCopySrc}))
		h.rec.Device().Err = errors.New(tagTestBackendFailed, "the backend failed")
		var gotErr error
		b.Read(0, 16, func(_ []byte, err error) { gotErr = err })
		h.ts.Pump()
		wantTag(t, gotErr, gpu.TagGPUReadFailed, "")
	})
	t.Run("a write while the GPU uses the buffer waits for the next submission", func(t *testing.T) {
		h := newHarness(t)
		b := must(h.d.NewBuffer(gpu.BufferDesc{Size: 16, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst}))
		other := must(h.d.NewBuffer(gpu.BufferDesc{Size: 16, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst}))
		enc := h.d.NewCommandEncoder()
		enc.CopyBufferToBuffer(b, 0, other, 0, 16)
		if err := h.d.Queue().Submit(enc.Finish()); err != nil {
			t.Fatal(err)
		}
		h.rec.Reset()
		h.d.Queue().WriteBuffer(b, 0, []byte{9, 9, 9, 9})
		writes := h.rec.Device().Writes
		if len(writes) != 1 || writes[0].Usage != explicit.BufferCopySource {
			t.Errorf("the write went to %+v, want a staging buffer", writes)
		}
		h.rec.CompleteReads()
		// The next submission copies the staged bytes first.
		h.rec.Reset()
		if err := h.d.Queue().Submit(h.d.NewCommandEncoder().Finish()); err != nil {
			t.Fatal(err)
		}
		ops := h.rec.Ops()
		if len(ops) < 2 || ops[0] != "Begin" || ops[1] != "CopyBuffer" {
			t.Errorf("ops = %v, want Begin and then the CopyBuffer of the staged write", ops)
		}
	})
	t.Run("a read of a buffer submitted ahead of the write sees the earlier bytes", func(t *testing.T) {
		h := newHarness(t)
		b := must(h.d.NewBuffer(gpu.BufferDesc{Size: 4, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst}))
		h.d.Queue().WriteBuffer(b, 0, []byte{1, 1, 1, 1})
		var first []byte
		b.Read(0, 4, func(data []byte, _ error) { first = data })
		h.d.Queue().WriteBuffer(b, 0, []byte{2, 2, 2, 2})
		h.rec.CompleteReads()
		h.ts.Pump()
		if !bytes.Equal(first, []byte{1, 1, 1, 1}) {
			t.Errorf("the read saw %v, want the bytes written before it", first)
		}
	})
}

func TestSubmitReturnsTheErrorOfTheBackend(t *testing.T) {
	h := newHarness(t)
	h.rec.Device().Err = errors.New(tagTestBackendFailed, "the backend failed")
	err := h.d.Queue().Submit(h.d.NewCommandEncoder().Finish())
	wantTag(t, err, tagTestBackendFailed, "")
}

func TestOnSubmittedWorkDone(t *testing.T) {
	h := newHarness(t)
	calls := 0
	h.d.Queue().OnSubmittedWorkDone(func() { calls++ })
	if calls != 0 {
		t.Fatal("fn ran inside the call")
	}
	h.ts.Pump()
	if calls != 0 {
		t.Fatal("fn ran before the GPU finished")
	}
	h.rec.CompleteReads()
	h.ts.Pump()
	if calls != 1 {
		t.Errorf("fn ran %d times", calls)
	}
}

// tagTestBackendFailed is the tag of the error that the tests make the fake device return.
const tagTestBackendFailed = "TestBackendFailed"
