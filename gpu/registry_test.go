package gpu_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

func (h *harness) sampledTexture(format gpu.TextureFormat) *gpu.Texture {
	h.t.Helper()
	return must(h.d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 16, Height: 16}, Format: format,
		Usage: gpu.TextureUsageTextureBinding | gpu.TextureUsageRenderAttachment}))
}

func TestRegisteringTextures(t *testing.T) {
	h := newHarness(t)
	a, b := h.sampledTexture(gpu.TextureFormatRGBA8Unorm), h.sampledTexture(gpu.TextureFormatRGBA16Float)
	idA, idB := h.d.RegisterTexture(a), h.d.RegisterTexture(b)
	if idA < 1 || idB < 1 || idA == idB {
		t.Fatalf("ids = %d, %d: want two different positive numbers", idA, idB)
	}
	if again := h.d.RegisterTexture(a); again == idA {
		t.Errorf("registering a texture twice gave the same id")
	}
	h.d.UnregisterTexture(idA)
	h.d.UnregisterTexture(idA)  // twice is fine
	h.d.UnregisterTexture(9999) // so is an id that was never registered
}

func TestReleasingATextureUnregistersIt(t *testing.T) {
	h := newHarness(t)
	tex := h.sampledTexture(gpu.TextureFormatRGBA8Unorm)
	reg := h.rec.GPUTextures()
	id := h.d.RegisterTexture(tex)
	if reg.Version(id) == 0 || reg.Len() != 1 {
		t.Fatalf("the registry has %d textures, version %d", reg.Len(), reg.Version(id))
	}
	tex.Release()
	if reg.Len() != 0 || reg.Version(id) != 0 {
		t.Errorf("after Release the registry has %d textures, version %d", reg.Len(), reg.Version(id))
	}
}

func TestMarkingAFrameAvailableSchedulesAFrameAndChangesTheVersion(t *testing.T) {
	h := newHarness(t)
	id := h.d.RegisterTexture(h.sampledTexture(gpu.TextureFormatRGBA8Unorm))
	reg := h.rec.GPUTextures()
	h.ts.PumpWidget(NewCenter(NewTexture(id)))
	h.ts.Pump()
	if h.ts.Binding.Scheduler().HasScheduledFrame() {
		t.Fatal("a frame is scheduled already")
	}
	before := reg.Version(id)
	h.d.MarkTextureFrameAvailable(id)
	if reg.Version(id) == before {
		t.Errorf("the version stayed %d", before)
	}
	if !h.ts.Binding.Scheduler().HasScheduledFrame() {
		t.Errorf("no frame was scheduled")
	}
	h.ts.Pump()
	// An id that is not registered schedules nothing.
	h.d.MarkTextureFrameAvailable(12345)
	if h.ts.Binding.Scheduler().HasScheduledFrame() {
		t.Errorf("a frame was scheduled for an id that is not registered")
	}
}

func TestATextureWidgetPutsTheTextureInTheScene(t *testing.T) {
	h := newHarness(t)
	id := h.d.RegisterTexture(h.sampledTexture(gpu.TextureFormatRGBA8Unorm))
	h.ts.PumpWidget(NewCenter(NewSizedBox(NewTexture(id, TextureOpts{FilterQuality: ui.FilterQualityMedium}), SizedBoxOpts{Width: foundation.Some(120.0), Height: foundation.Some(80.0)})))
	scenes := h.ts.Scenes()
	if len(scenes) == 0 {
		t.Fatal("no scene was rendered")
	}
	var found *ui.TextureLayer
	var at ui.Offset // where the layer's corner is, with the transforms above it
	var walk func(l ui.Layer, m ui.Matrix4)
	walk = func(l ui.Layer, m ui.Matrix4) {
		if tl, ok := l.(*ui.TextureLayer); ok {
			found, at = tl, m.TransformPoint(tl.Offset)
		}
		if tr, ok := l.(*ui.TransformLayer); ok {
			m.MultiplyBy(&tr.Matrix)
		}
		for _, c := range l.Children() {
			walk(c, m)
		}
	}
	walk(scenes[len(scenes)-1].Root(), *ui.NewMatrix4Identity())
	if found == nil {
		t.Fatal("the scene has no texture layer")
	}
	if found.TextureID != id || found.Width != 120 || found.Height != 80 || found.FilterQuality != ui.FilterQualityMedium {
		t.Errorf("texture layer = %+v", *found)
	}
	// The widget is centered in the 800 x 600 view.
	if at != (ui.Offset{Dx: 340, Dy: 260}) {
		t.Errorf("the layer is at %v, want (340, 260), centered", at)
	}
}

func TestRegistryAfterTheDeviceIsLost(t *testing.T) {
	h := newHarness(t)
	tex := h.sampledTexture(gpu.TextureFormatRGBA8Unorm)
	id := h.d.RegisterTexture(tex)
	reg := h.rec.GPUTextures()
	h.rec.Lose()
	if reg.Len() != 0 {
		t.Errorf("the registry still has %d textures of the lost device", reg.Len())
	}
	if got := h.d.RegisterTexture(tex); got != 0 {
		t.Errorf("a lost device registered a texture: %d", got)
	}
	h.d.UnregisterTexture(id)
	h.d.MarkTextureFrameAvailable(id)
	if h.ts.Binding.Scheduler().HasScheduledFrame() {
		t.Errorf("a lost device scheduled a frame")
	}
}

func TestRegisteringAnUnshowableTexture(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		desc gpu.TextureDesc
	}{
		{"a texture that shaders cannot sample", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageRenderAttachment}},
		{"a format the composite does not read", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: gpu.TextureFormatR8Unorm, Usage: gpu.TextureUsageTextureBinding}},
		{"a cube", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Kind: gpu.TextureKindCube, Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			tex := must(h.d.NewTexture(tt.desc))
			err := panicOf(func() { h.d.RegisterTexture(tex) })
			if err == nil {
				t.Fatal("no panic")
			}
			wantTag(t, err, gpu.TagGPUEncodingInvalid, "Device.RegisterTexture->misuse.GPUEncodingInvalid")
		})
	}
	t.Run("a texture of another device", func(t *testing.T) {
		h, other := newHarness(t), newHarness(t)
		tex := other.sampledTexture(gpu.TextureFormatRGBA8Unorm)
		err := panicOf(func() { h.d.RegisterTexture(tex) })
		wantTag(t, err, gpu.TagGPUEncodingInvalid, "")
	})
}
