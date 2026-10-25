//go:build windows && glutter_wine

package d3d12

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"sync"
	"testing"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// f32s is the bytes of the floats.
func f32s(v ...float32) []byte {
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(f))
	}
	return b
}

// solidRig is what the tests that draw rectangles with the solid pipeline share: a target, the
// pipeline, a ring of two Frame uniforms (the surface 16x16 at offset 0 and 32x32 at 256), and a bind
// group of it.
type solidRig struct {
	d      *Device
	target explicit.Texture
	pipe   explicit.Pipeline
	ring   explicit.Buffer
	inst   explicit.Buffer
}

func newSolidRig(t *testing.T, d *Device, w, h int) *solidRig {
	t.Helper()
	target, err := d.NewTexture(explicit.TextureDesc{Width: w, Height: h, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource | explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	pipe, err := d.NewPipeline(solidDesc(explicit.BlendState{}))
	if err != nil {
		t.Fatalf("NewPipeline: %+v", err)
	}
	ring, err := d.NewBuffer(512, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	d.WriteBuffer(ring, 0, f32s(16, 16, 0, 0))
	d.WriteBuffer(ring, 256, f32s(32, 32, 0, 0))
	inst, err := d.NewBuffer(256, explicit.BufferVertex)
	if err != nil {
		t.Fatal(err)
	}
	return &solidRig{d: d, target: target, pipe: pipe, ring: ring, inst: inst}
}

// rect is an instance of the solid pipeline: x, y, width, height, and the premultiplied color.
func rect(x, y, w, h float32, r, g, b, a float32) []byte { return f32s(x, y, w, h, r, g, b, a) }

func (s *solidRig) group(t *testing.T) explicit.BindGroup {
	t.Helper()
	bg, err := s.d.NewBindGroup(s.pipe, 0, []explicit.Binding{{Binding: 0, Buffer: s.ring, Size: 16}})
	if err != nil {
		t.Fatalf("NewBindGroup: %+v", err)
	}
	return bg
}

func pixel(img *image.RGBA, x, y int) color.RGBA { return img.RGBAAt(x, y) }

func TestWineDrawsInstancesWithDynamicOffsetsAndVertexOffsets(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	s := newSolidRig(t, d, 16, 16)
	d.WriteBuffer(s.inst, 0, rect(2, 2, 4, 4, 1, 0, 0, 1))
	d.WriteBuffer(s.inst, 32, rect(8, 8, 4, 4, 0, 1, 0, 1))
	frame(t, d, func(e explicit.Encoder) {
		bg := s.group(t)
		p := e.BeginPass(s.target, explicit.LoadClear, ui.Color{})
		p.SetPipeline(s.pipe)
		p.SetBindGroup(0, bg, 0)
		p.SetVertexBuffer(0, s.inst, 0)
		p.Draw(6, 2, 0, 0) // both instances at the scale of the surface 16
		p.End()
	})
	img := readTexture(t, d, s.target, image.Rect(0, 0, 16, 16))
	for _, tt := range []struct {
		x, y int
		want color.RGBA
	}{
		{2, 2, color.RGBA{255, 0, 0, 255}}, {5, 5, color.RGBA{255, 0, 0, 255}}, {6, 6, color.RGBA{}},
		{8, 8, color.RGBA{0, 255, 0, 255}}, {11, 11, color.RGBA{0, 255, 0, 255}}, {12, 12, color.RGBA{}}, {0, 0, color.RGBA{}}, {15, 15, color.RGBA{}},
	} {
		if got := pixel(img, tt.x, tt.y); got != tt.want {
			t.Errorf("pixel (%d,%d) = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
	// The second uniform (a surface of 32) and an offset into the vertex buffer: only the second instance,
	// at half the size and half the place.
	frame(t, d, func(e explicit.Encoder) {
		bg := s.group(t)
		p := e.BeginPass(s.target, explicit.LoadClear, ui.Color{})
		p.SetPipeline(s.pipe)
		p.SetBindGroup(0, bg, 256)
		p.SetVertexBuffer(0, s.inst, 32)
		p.Draw(6, 1, 0, 0)
		p.End()
	})
	img = readTexture(t, d, s.target, image.Rect(0, 0, 16, 16))
	for _, tt := range []struct {
		x, y int
		want color.RGBA
	}{{3, 3, color.RGBA{}}, {4, 4, color.RGBA{0, 255, 0, 255}}, {5, 5, color.RGBA{0, 255, 0, 255}}, {6, 6, color.RGBA{}}, {2, 2, color.RGBA{}}} {
		if got := pixel(img, tt.x, tt.y); got != tt.want {
			t.Errorf("second frame: pixel (%d,%d) = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
}

func TestWineScissorLoadOpsAndDiscard(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	s := newSolidRig(t, d, 16, 16)
	d.WriteBuffer(s.inst, 0, rect(0, 0, 16, 16, 0, 0, 1, 1))
	d.WriteBuffer(s.inst, 32, rect(0, 0, 16, 16, 1, 1, 0, 1))
	draw := func(e explicit.Encoder, load explicit.LoadOp, clear ui.Color, instance int, scissor image.Rectangle) {
		bg := s.group(t)
		p := e.BeginPass(s.target, load, clear)
		p.SetPipeline(s.pipe)
		p.SetBindGroup(0, bg, 0)
		p.SetVertexBuffer(0, s.inst, 32*instance)
		if !scissor.Empty() {
			p.SetScissor(scissor)
		}
		p.Draw(6, 1, 0, 0)
		p.End()
	}
	// A scissor limits the draw, and a scissor larger than the target is clipped to it.
	frame(t, d, func(e explicit.Encoder) {
		draw(e, explicit.LoadClear, ui.Color{R: 1, G: 0, B: 0, A: 1}, 0, image.Rect(4, 4, 8, 8))
	})
	img := readTexture(t, d, s.target, image.Rect(0, 0, 16, 16))
	for _, tt := range []struct {
		x, y int
		want color.RGBA
	}{{4, 4, color.RGBA{0, 0, 255, 255}}, {7, 7, color.RGBA{0, 0, 255, 255}}, {8, 8, color.RGBA{255, 0, 0, 255}}, {3, 3, color.RGBA{255, 0, 0, 255}}, {15, 15, color.RGBA{255, 0, 0, 255}}} {
		if got := pixel(img, tt.x, tt.y); got != tt.want {
			t.Errorf("scissor: pixel (%d,%d) = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
	// LoadLoad keeps what was there and draws on it.
	frame(t, d, func(e explicit.Encoder) { draw(e, explicit.LoadLoad, ui.Color{}, 1, image.Rect(0, 0, 2, 2)) })
	img = readTexture(t, d, s.target, image.Rect(0, 0, 16, 16))
	for _, tt := range []struct {
		x, y int
		want color.RGBA
	}{{0, 0, color.RGBA{255, 255, 0, 255}}, {1, 1, color.RGBA{255, 255, 0, 255}}, {2, 2, color.RGBA{255, 0, 0, 255}}, {5, 5, color.RGBA{0, 0, 255, 255}}} {
		if got := pixel(img, tt.x, tt.y); got != tt.want {
			t.Errorf("load: pixel (%d,%d) = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
	// LoadDontCare: the pass draws every pixel, so the result is that draw whatever was there before.
	frame(t, d, func(e explicit.Encoder) {
		draw(e, explicit.LoadDontCare, ui.Color{R: 1, G: 1, B: 1, A: 1}, 0, image.Rectangle{})
	})
	img = readTexture(t, d, s.target, image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if got := pixel(img, x, y); got != (color.RGBA{0, 0, 255, 255}) {
				t.Fatalf("don't care: pixel (%d,%d) = %v, want the blue of the draw", x, y, got)
			}
		}
	}
}

func TestWineDrawIndexed(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	s := newSolidRig(t, d, 16, 16)
	d.WriteBuffer(s.inst, 0, rect(0, 0, 16, 16, 1, 0, 0, 1))
	ib, err := d.NewBuffer(64, explicit.BufferIndex)
	if err != nil {
		t.Fatal(err)
	}
	// Index data from a byte offset of 4: the six corners in another order, then the first triangle.
	idx := make([]byte, 64)
	for i, v := range []uint16{3, 4, 5, 0, 1, 2, 0, 1, 2} {
		binary.LittleEndian.PutUint16(idx[4+2*i:], v)
	}
	d.WriteBuffer(ib, 0, idx)
	run := func(first, count int) *image.RGBA {
		frame(t, d, func(e explicit.Encoder) {
			bg := s.group(t)
			p := e.BeginPass(s.target, explicit.LoadClear, ui.Color{})
			p.SetPipeline(s.pipe)
			p.SetBindGroup(0, bg, 0)
			p.SetVertexBuffer(0, s.inst, 0)
			p.SetIndexBuffer(ib, 4)
			p.DrawIndexed(count, 1, first, 0, 0)
			p.End()
		})
		return readTexture(t, d, s.target, image.Rect(0, 0, 16, 16))
	}
	red := color.RGBA{255, 0, 0, 255}
	img := run(0, 6)
	for _, pt := range [][2]int{{1, 1}, {14, 1}, {1, 14}, {14, 14}, {8, 8}} {
		if got := pixel(img, pt[0], pt[1]); got != red {
			t.Errorf("both triangles: pixel %v = %v", pt, got)
		}
	}
	// Only the first triangle (0,0) (1,0) (0,1): the upper left half.
	img = run(6, 3)
	if got := pixel(img, 2, 2); got != red {
		t.Errorf("first triangle: pixel (2,2) = %v", got)
	}
	if got := pixel(img, 13, 13); got != (color.RGBA{}) {
		t.Errorf("first triangle: pixel (13,13) = %v, want clear", got)
	}
}

func TestWineBlendStateOfAPipeline(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	s := newSolidRig(t, d, 16, 16)
	over, err := d.NewPipeline(solidDesc(explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendOne, DstFactor: explicit.BlendOneMinusSrcAlpha}))
	if err != nil {
		t.Fatal(err)
	}
	d.WriteBuffer(s.inst, 0, rect(0, 0, 16, 16, 0, 0, 0.5, 0.5)) // premultiplied half blue
	frame(t, d, func(e explicit.Encoder) {
		bg := s.group(t)
		p := e.BeginPass(s.target, explicit.LoadClear, ui.Color{R: 1, G: 1, B: 1, A: 1})
		p.SetPipeline(over)
		p.SetBindGroup(0, bg, 0)
		p.SetVertexBuffer(0, s.inst, 0)
		p.Draw(6, 1, 0, 0)
		p.End()
	})
	img := readTexture(t, d, s.target, image.Rect(0, 0, 16, 16))
	got := pixel(img, 8, 8)
	want := color.RGBA{128, 128, 255, 255} // 0.5 blue over white
	for _, d := range []int{int(got.R) - int(want.R), int(got.G) - int(want.G), int(got.B) - int(want.B), int(got.A) - int(want.A)} {
		if d < -1 || d > 1 {
			t.Fatalf("source over: pixel = %v, want %v", got, want)
		}
	}
}

// TestWineSamplingATextureInAPass draws the blit of a texture (a sampler, a texture and a uniform
// with the pre-rotation) into a BGRA target and reads the pixels back: the tables of the descriptors,
// the barrier to PIXEL_SHADER_RESOURCE and the root arguments all have to be right.
func TestWineSamplingATextureInAPass(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	src, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	data := pattern(8, 8, 4, 8*4, 17)
	for i := 3; i < len(data); i += 4 {
		data[i] = 255
	}
	d.WriteTexture(src, 0, image.Rect(0, 0, 8, 8), data, 8*4)
	dst, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatBGRA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	pipe, err := d.NewPipeline(blitDesc())
	if err != nil {
		t.Fatal(err)
	}
	smp, err := d.NewSampler(explicit.SamplerDesc{MinFilter: explicit.FilterNearest, MagFilter: explicit.FilterNearest})
	if err != nil {
		t.Fatal(err)
	}
	rot, err := d.NewBuffer(4*256, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	for r := 0; r < 4; r++ {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(r))
		d.WriteBuffer(rot, r*256, b)
	}
	frame(t, d, func(e explicit.Encoder) {
		bg, err := d.NewBindGroup(pipe, 0, []explicit.Binding{{Binding: 0, Texture: src}, {Binding: 1, Sampler: smp}, {Binding: 2, Buffer: rot, Size: 16}})
		if err != nil {
			t.Fatal(err)
		}
		p := e.BeginPass(dst, explicit.LoadDontCare, ui.Color{})
		p.SetPipeline(pipe)
		p.SetBindGroup(0, bg, 0)
		p.Draw(3, 1, 0, 0)
		p.End()
	})
	samePixels(t, "the blit of a texture", readTexture(t, d, dst, image.Rect(0, 0, 8, 8)), expected(explicit.TextureFormatRGBA8Unorm, data, 8, 8, 8*4))
}

// A bind group with a binding left out holds a null texture and the default sampler.
func TestWineBindGroupWithBindingsLeftOut(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	pipe, err := d.NewPipeline(blitDesc())
	if err != nil {
		t.Fatal(err)
	}
	rot, err := d.NewBuffer(256, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatBGRA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	frame(t, d, func(e explicit.Encoder) {
		bg, err := d.NewBindGroup(pipe, 0, []explicit.Binding{{Binding: 2, Buffer: rot, Size: 16}})
		if err != nil {
			t.Fatal(err)
		}
		p := e.BeginPass(dst, explicit.LoadClear, ui.Color{R: 1, A: 1})
		p.SetPipeline(pipe)
		p.SetBindGroup(0, bg, 0)
		p.Draw(3, 1, 0, 0)
		p.End()
	})
	img := readTexture(t, d, dst, image.Rect(0, 0, 4, 4))
	if got := pixel(img, 1, 1); got.A != 0 || got.R != 0 {
		t.Errorf("a null texture reads as zero, and the blit drew %v", got)
	}
}

// ---- errors ----

func TestWineBindGroupErrors(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	blit, err := d.NewPipeline(blitDesc())
	if err != nil {
		t.Fatal(err)
	}
	ub, err := d.NewBuffer(1024, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	notSampled, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget})
	if err != nil {
		t.Fatal(err)
	}
	smp, err := d.NewSampler(explicit.SamplerDesc{})
	if err != nil {
		t.Fatal(err)
	}
	uni := explicit.Binding{Binding: 2, Buffer: ub, Size: 16}
	tests := []struct {
		name    string
		group   int
		entries []explicit.Binding
		route   string
	}{
		{"a group the pipeline lacks", 3, []explicit.Binding{uni}, "Device.NewBindGroup.D3D12BindGroupInvalid"},
		{"a binding the shader lacks", 0, []explicit.Binding{uni, {Binding: 7, Sampler: smp}}, "Device.NewBindGroup.D3D12BindGroupInvalid"},
		{"a uniform without its entry", 0, []explicit.Binding{{Binding: 1, Sampler: smp}}, "Device.NewBindGroup.D3D12BindGroupInvalid"},
		{"a uniform without its buffer", 0, []explicit.Binding{{Binding: 2}}, "Device.NewBindGroup.D3D12BindGroupInvalid"},
		{"an offset that is not a multiple of 256", 0, []explicit.Binding{{Binding: 2, Buffer: ub, Offset: 100}}, "Device.NewBindGroup.D3D12BindGroupInvalid"},
		{"an offset past the end", 0, []explicit.Binding{{Binding: 2, Buffer: ub, Offset: 2048}}, "Device.NewBindGroup.D3D12BindGroupInvalid"},
		{"a texture binding without a texture", 0, []explicit.Binding{uni, {Binding: 0}}, "Device.NewBindGroup.D3D12BindGroupInvalid"},
		{"a texture that is not sampled", 0, []explicit.Binding{uni, {Binding: 0, Texture: notSampled}}, "Device.NewBindGroup.D3D12BindGroupInvalid"},
		{"a sampler binding without a sampler", 0, []explicit.Binding{uni, {Binding: 1}}, "Device.NewBindGroup.D3D12BindGroupInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bg, err := d.NewBindGroup(blit, tt.group, tt.entries)
			if bg != nil {
				t.Errorf("a bind group was made")
			}
			_ = wantRoute(t, err, tagD3D12BindGroupInvalid, tt.route)
		})
	}
	// A resource of another device.
	other, _ := openTestDevice(t, Options{})
	foreign, err := other.NewBuffer(256, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	appErr := expectPanic(t, func() { _, _ = d.NewBindGroup(blit, 0, []explicit.Binding{{Binding: 2, Buffer: foreign}}) })
	if appErr.Tag != tagD3D12ForeignResource || !appErr.HasRoute("Device.NewBindGroup.D3D12ForeignResource") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
	appErr = expectPanic(t, func() { _, _ = d.NewBindGroup(nil, 0, nil) })
	if appErr.Tag != tagD3D12ForeignResource {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
}

func TestWinePassCommandErrors(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	s := newSolidRig(t, d, 8, 8)
	blit, err := d.NewPipeline(blitDesc())
	if err != nil {
		t.Fatal(err)
	}
	smp, err := d.NewSampler(explicit.SamplerDesc{})
	if err != nil {
		t.Fatal(err)
	}
	rot, err := d.NewBuffer(256, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	other, _ := openTestDevice(t, Options{})
	otherBuf, err := other.NewBuffer(256, explicit.BufferVertex)
	if err != nil {
		t.Fatal(err)
	}
	// A bind group of the blit, for the frames that follow.
	var stale explicit.BindGroup
	run := func(fn func(p explicit.Pass, bg explicit.BindGroup)) {
		e, err := d.Begin()
		if err != nil {
			t.Fatal(err)
		}
		bg, err := d.NewBindGroup(blit, 0, []explicit.Binding{{Binding: 0, Texture: s.target}, {Binding: 1, Sampler: smp}, {Binding: 2, Buffer: rot, Size: 16}})
		if err != nil {
			t.Fatal(err)
		}
		other, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatBGRA8Unorm, Usage: explicit.TextureRenderTarget})
		if err != nil {
			t.Fatal(err)
		}
		defer other.Destroy()
		p := e.BeginPass(other, explicit.LoadClear, ui.Color{})
		fn(p, bg)
		p.End()
		if err := e.Submit(); err != nil {
			t.Fatal(err)
		}
		stale = bg
	}
	wantPanic := func(what, route string, tag string, fn func()) {
		t.Helper()
		appErr := expectPanic(t, fn)
		if appErr.Tag != tag || !appErr.HasRoute(route) {
			t.Errorf("%s: %s via %s, want %s via %s", what, appErr.Tag, appErr.Route(), tag, route)
		}
	}
	run(func(p explicit.Pass, bg explicit.BindGroup) {
		wantPanic("Draw before SetPipeline", "pass.Draw->pass.flush.D3D12PipelineInvalid", tagD3D12PipelineInvalid, func() { p.Draw(3, 1, 0, 0) })
		wantPanic("DrawIndexed before SetPipeline", "pass.DrawIndexed->pass.flush.D3D12PipelineInvalid", tagD3D12PipelineInvalid, func() { p.DrawIndexed(3, 1, 0, 0, 0) })
		wantPanic("SetBindGroup before SetPipeline", "pass.SetBindGroup.D3D12BindGroupInvalid", tagD3D12BindGroupInvalid, func() { p.SetBindGroup(0, bg, 0) })
		wantPanic("a pipeline of another device", "pass.SetPipeline.D3D12ForeignResource", tagD3D12ForeignResource, func() { p.SetPipeline(nil) })
		wantPanic("a bind group of another device", "pass.SetBindGroup.D3D12ForeignResource", tagD3D12ForeignResource, func() { p.SetBindGroup(0, nil) })
		wantPanic("a buffer of another device", "pass.SetVertexBuffer.D3D12ForeignResource", tagD3D12ForeignResource, func() { p.SetVertexBuffer(0, otherBuf, 0) })
		wantPanic("an index buffer of another device", "pass.SetIndexBuffer.D3D12ForeignResource", tagD3D12ForeignResource, func() { p.SetIndexBuffer(otherBuf, 0) })
		wantPanic("a vertex buffer offset past the end", "pass.SetVertexBuffer.D3D12BindGroupInvalid", tagD3D12BindGroupInvalid, func() { p.SetVertexBuffer(0, s.inst, 1<<20) })
		p.SetPipeline(blit)
		wantPanic("a group the pipeline lacks", "pass.SetBindGroup.D3D12BindGroupInvalid", tagD3D12BindGroupInvalid, func() { p.SetBindGroup(3, bg, 0) })
		wantPanic("the wrong number of dynamic offsets", "pass.SetBindGroup.D3D12BindGroupInvalid", tagD3D12BindGroupInvalid, func() { p.SetBindGroup(0, bg) })
		wantPanic("too many dynamic offsets", "pass.SetBindGroup.D3D12BindGroupInvalid", tagD3D12BindGroupInvalid, func() { p.SetBindGroup(0, bg, 0, 0) })
		wantPanic("a bind group for another group", "pass.SetBindGroup.D3D12BindGroupInvalid", tagD3D12BindGroupInvalid, func() { p.SetBindGroup(1, bg, 0) })
		p.SetBindGroup(0, bg, 0) // and the right way works
	})
	// A bind group of an earlier frame: its slot was reused or is another one.
	run(func(p explicit.Pass, bg explicit.BindGroup) {
		p.SetPipeline(blit)
		wantPanic("a bind group of the frame before", "pass.SetBindGroup.D3D12BindGroupInvalid", tagD3D12BindGroupInvalid, func() { p.SetBindGroup(0, stale, 0) })
	})
	run(func(p explicit.Pass, bg explicit.BindGroup) {
		p.SetPipeline(blit)
		wantPanic("a bind group of two frames before", "pass.SetBindGroup.D3D12BindGroupInvalid", tagD3D12BindGroupInvalid, func() { p.SetBindGroup(0, stale, 0) })
	})
}

func TestWineSamplingThePassTargetIsAStateConflict(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	blit, err := d.NewPipeline(blitDesc())
	if err != nil {
		t.Fatal(err)
	}
	smp, err := d.NewSampler(explicit.SamplerDesc{})
	if err != nil {
		t.Fatal(err)
	}
	rot, err := d.NewBuffer(256, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatBGRA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	bg, err := d.NewBindGroup(blit, 0, []explicit.Binding{{Binding: 0, Texture: tex}, {Binding: 1, Sampler: smp}, {Binding: 2, Buffer: rot, Size: 16}})
	if err != nil {
		t.Fatal(err)
	}
	p := e.BeginPass(tex, explicit.LoadClear, ui.Color{})
	p.SetPipeline(blit)
	appErr := expectPanic(t, func() { p.SetBindGroup(0, bg, 0) })
	if appErr.Tag != tagD3D12StateConflict || !appErr.HasRoute("pass.SetBindGroup->stateTracker[...].need.D3D12StateConflict") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
	p.End()
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
}

// ---- the encoder over a fake COM layer ----

// recordedCall is one call on the command list: the method, its arguments, and for a barrier the
// transitions it carries.
type recordedCall struct {
	index    int
	args     [6]uintptr
	barriers []recordedBarrier
}

type recordedBarrier struct {
	res           uintptr
	sub           uint32
	before, after resState
}

// listProxy is an ID3D12GraphicsCommandList made in Go over the real one: every method records its
// call and forwards it, so the frame really runs on the GPU. The test reads what the encoder did, in
// order.
type listProxy struct {
	real  win32.Object
	mu    sync.Mutex
	calls []recordedCall
}

var (
	proxyOnce sync.Once
	proxyHost *win32.Host
	proxyType *win32.InterfaceType
	proxyErr  error
)

// listMethods is how many methods ID3D12GraphicsCommandList has after IUnknown (index 3 to 59).
const listMethods = 57

// makeListProxy wraps the real list. The Host and the interface type are made once for the process.
func makeListProxy(t *testing.T, real win32.Object) (win32.Object, *listProxy) {
	t.Helper()
	proxyOnce.Do(func() {
		proxyHost, proxyErr = win32.NewHost(nil)
		if proxyErr != nil {
			return
		}
		methods := make([]win32.MethodDef, listMethods)
		for i := range methods {
			index := i + 3
			methods[i] = win32.MethodDef{Name: fmt.Sprintf("method%d", index), Args: 6, Fn: func(impl any, a []uintptr) uintptr {
				p := impl.(*listProxy)
				c := recordedCall{index: index}
				copy(c.args[:], a)
				if index == listResourceBarrier {
					n := int(a[0])
					raw := win32.Bytes(a[1], n*int(unsafeSizeofBarrier))
					for k := 0; k < n; k++ {
						b := raw[k*int(unsafeSizeofBarrier):]
						c.barriers = append(c.barriers, recordedBarrier{
							res: uintptr(binary.LittleEndian.Uint64(b[8:])), sub: binary.LittleEndian.Uint32(b[16:]),
							before: resState(binary.LittleEndian.Uint32(b[20:])), after: resState(binary.LittleEndian.Uint32(b[24:])),
						})
					}
				}
				p.mu.Lock()
				p.calls = append(p.calls, c)
				p.mu.Unlock()
				return p.real.Call(index, a[0], a[1], a[2], a[3], a[4], a[5])
			}}
		}
		proxyType, proxyErr = proxyHost.NewInterface(win32.InterfaceDef{Name: "ID3D12GraphicsCommandList", IID: iidGraphicsCommandList, Methods: methods})
	})
	if proxyErr != nil {
		t.Fatal(proxyErr)
	}
	p := &listProxy{real: real}
	obj, err := proxyHost.NewObject(p, proxyType)
	if err != nil {
		t.Fatal(err)
	}
	return obj, p
}

// unsafeSizeofBarrier is the size of D3D12_RESOURCE_BARRIER.
const unsafeSizeofBarrier = 32

// queueProxy is an ID3D12CommandQueue made in Go over the real one: ExecuteCommandLists gets the
// proxies of the command lists, which it swaps for the lists behind them; everything else is forwarded.
type queueProxy struct {
	real  win32.Object
	lists map[uintptr]uintptr // the pointer of a list proxy to the real list
}

var queueType *win32.InterfaceType

// queueMethods is how many methods ID3D12CommandQueue has after IUnknown (index 3 to 18).
const queueMethods = 16

func makeQueueProxy(t *testing.T, real win32.Object, lists map[uintptr]uintptr) win32.Object {
	t.Helper()
	// The Host is the one that makeListProxy made: proxyLists makes the lists first.
	if queueType == nil {
		methods := make([]win32.MethodDef, queueMethods)
		for i := range methods {
			index := i + 3
			methods[i] = win32.MethodDef{Name: fmt.Sprintf("queue%d", index), Args: 6, Fn: func(impl any, a []uintptr) uintptr {
				q := impl.(*queueProxy)
				if index == queueExecuteCommandLists {
					n := int(a[0])
					raw := win32.Bytes(a[1], n*8)
					swapped := make([]uintptr, n)
					for k := range swapped {
						swapped[k] = uintptr(binary.LittleEndian.Uint64(raw[8*k:]))
						if r, ok := q.lists[swapped[k]]; ok {
							swapped[k] = r
						}
					}
					return q.real.Call(index, a[0], uintptr(unsafe.Pointer(&swapped[0])))
				}
				return q.real.Call(index, a[0], a[1], a[2], a[3], a[4], a[5])
			}}
		}
		var err error
		if queueType, err = proxyHost.NewInterface(win32.InterfaceDef{Name: "ID3D12CommandQueue", IID: iidCommandQueue, Methods: methods}); err != nil {
			t.Fatal(err)
		}
	}
	obj, err := proxyHost.NewObject(&queueProxy{real: real, lists: lists}, queueType)
	if err != nil {
		t.Fatal(err)
	}
	return obj
}

// proxyLists puts a proxy over the command list of each frame slot, and puts the real lists back when
// the test ends (the device releases what is in its slots).
func proxyLists(t *testing.T, d *Device) []*listProxy {
	t.Helper()
	var proxies []*listProxy
	reals := make([]win32.Object, len(d.slots))
	lists := map[uintptr]uintptr{}
	for i := range d.slots {
		reals[i] = d.slots[i].list
		obj, p := makeListProxy(t, reals[i])
		d.slots[i].list = obj
		lists[obj.Ptr()] = reals[i].Ptr()
		proxies = append(proxies, p)
	}
	realQueue := d.queue
	d.queue = makeQueueProxy(t, realQueue, lists)
	t.Cleanup(func() {
		for i := range d.slots {
			d.slots[i].list.Release() // the proxy
			d.slots[i].list = reals[i]
		}
		d.queue.Release()
		d.queue = realQueue
	})
	return proxies
}

// take returns the calls recorded so far and forgets them.
func (p *listProxy) take() []recordedCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	c := p.calls
	p.calls = nil
	return c
}

var listNames = map[int]string{
	listClose: "Close", listReset: "Reset", listDrawInstanced: "DrawInstanced", listDrawIndexedInstanced: "DrawIndexedInstanced",
	listCopyTextureRegion: "CopyTextureRegion", listIASetPrimitiveTopo: "IASetPrimitiveTopology", listRSSetViewports: "RSSetViewports",
	listRSSetScissorRects: "RSSetScissorRects", listSetPipelineState: "SetPipelineState", listResourceBarrier: "ResourceBarrier",
	listSetDescriptorHeaps: "SetDescriptorHeaps", listSetRootSignature: "SetGraphicsRootSignature", listSetRootTable: "SetGraphicsRootDescriptorTable",
	listSetRootCBV: "SetGraphicsRootConstantBufferView", listIASetIndexBuffer: "IASetIndexBuffer", listIASetVertexBuffers: "IASetVertexBuffers",
	listOMSetRenderTargets: "OMSetRenderTargets", listClearRenderTargetView: "ClearRenderTargetView", listDiscardResource: "DiscardResource",
}

// describe writes the calls as lines; names maps the resources of the barriers to the names the test
// gave them. Reset and Close are left out.
func describe(calls []recordedCall, names map[uintptr]string) []string {
	var out []string
	for _, c := range calls {
		name, ok := listNames[c.index]
		if !ok {
			name = fmt.Sprintf("method%d", c.index)
		}
		switch c.index {
		case listReset, listClose:
			continue
		case listResourceBarrier:
			var bs []string
			for _, b := range c.barriers {
				n := names[b.res]
				if n == "" {
					n = fmt.Sprintf("%#x", b.res)
				}
				bs = append(bs, fmt.Sprintf("%s.%d %v->%v", n, b.sub, b.before, b.after))
			}
			out = append(out, name+" "+strings.Join(bs, ", "))
		case listSetDescriptorHeaps:
			out = append(out, fmt.Sprintf("%s %d", name, c.args[0]))
		case listSetRootTable:
			out = append(out, fmt.Sprintf("%s p%d", name, c.args[0]))
		case listSetRootCBV:
			out = append(out, fmt.Sprintf("%s p%d", name, c.args[0]))
		case listDrawInstanced:
			out = append(out, fmt.Sprintf("%s %d %d %d %d", name, c.args[0], c.args[1], c.args[2], c.args[3]))
		default:
			out = append(out, name)
		}
	}
	return out
}

func equalLines(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls\n got:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

// TestWineEncoderOrderOfBarriersHeapsAndRootArguments records a frame that uploads, draws a texture
// into a target, copies the target and reads it back, and checks the order of the barriers, the heaps
// and the root arguments (SPEC §9.8 "States", "Descriptors", "Binding model").
func TestWineEncoderOrderOfBarriersHeapsAndRootArguments(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	proxies := proxyLists(t, d)
	src, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	dst, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatBGRA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	copyDst, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatBGRA8Unorm, Usage: explicit.TextureCopySource | explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	pipe, err := d.NewPipeline(blitDesc())
	if err != nil {
		t.Fatal(err)
	}
	smp, err := d.NewSampler(explicit.SamplerDesc{})
	if err != nil {
		t.Fatal(err)
	}
	rot, err := d.NewBuffer(512, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	names := map[uintptr]string{src.(*texture).res.Ptr(): "src", dst.(*texture).res.Ptr(): "dst", copyDst.(*texture).res.Ptr(): "copy"}
	d.WriteTexture(src, 0, image.Rect(0, 0, 4, 4), pattern(4, 4, 4, 16, 3), 16)

	slot := d.next
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	bg, err := d.NewBindGroup(pipe, 0, []explicit.Binding{{Binding: 0, Texture: src}, {Binding: 1, Sampler: smp}, {Binding: 2, Buffer: rot, Size: 16}})
	if err != nil {
		t.Fatal(err)
	}
	p := e.BeginPass(dst, explicit.LoadClear, ui.Color{})
	p.SetPipeline(pipe)
	p.SetBindGroup(0, bg, 256)
	p.Draw(3, 1, 0, 0)
	p.End()
	e.CopyTexture(dst, image.Rect(0, 0, 4, 4), copyDst, image.Point{})
	var read *image.RGBA
	e.ReadTexture(copyDst, image.Rect(0, 0, 4, 4), func(img *image.RGBA, err error) {
		if err != nil {
			t.Error(err)
		}
		read = img
	})
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
	if read == nil {
		t.Fatal("the read gave nothing")
	}
	// Parameters of the blit: 0 is the uniform b2, 1 the texture table, 2 the sampler table.
	want := []string{
		"ResourceBarrier src.0 COMMON->COPY_DEST", "CopyTextureRegion", // the upload, recorded at the start of the frame
		"ResourceBarrier dst.0 COMMON->RENDER_TARGET", "ClearRenderTargetView", "OMSetRenderTargets", "RSSetViewports", "RSSetScissorRects", "IASetPrimitiveTopology",
		"SetGraphicsRootSignature", "SetPipelineState",
		"ResourceBarrier src.0 COPY_DEST->PIXEL_SHADER_RESOURCE",
		"SetDescriptorHeaps 2", "SetGraphicsRootDescriptorTable p1", "SetGraphicsRootDescriptorTable p2", "SetGraphicsRootConstantBufferView p0",
		"DrawInstanced 3 1 0 0",
		"ResourceBarrier dst.0 RENDER_TARGET->COPY_SOURCE, copy.0 COMMON->COPY_DEST", "CopyTextureRegion",
		"ResourceBarrier copy.0 COPY_DEST->COPY_SOURCE", "CopyTextureRegion",
	}
	equalLines(t, describe(proxies[slot].take(), names), want)
}

// TestWineEncoderSwitchesHeapsWhenABlockIsAdded makes the blocks of the heaps small, binds groups of
// the layer pipeline from the first block, then one from the second: the heaps are set again, and the
// groups bound before are copied into the second block and their tables set again.
func TestWineEncoderSwitchesHeapsWhenABlockIsAdded(t *testing.T) {
	d, _ := openTestDevice(t, Options{viewBlock: 6, samplerBlock: 2})
	proxies := proxyLists(t, d)
	layer, err := d.NewPipeline(explicit.PipelineDesc{Shader: explicit.ShaderLayer, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: explicit.TextureFormatRGBA8Unorm,
		VertexBuffers: []explicit.VertexBufferLayout{{Stride: 16, PerInstance: true, Attributes: []explicit.VertexAttribute{{Location: 0, Format: explicit.VertexFloat32x4}}}}})
	if err != nil {
		t.Fatal(err)
	}
	mk := func(w int) explicit.Texture {
		tex, err := d.NewTexture(explicit.TextureDesc{Width: w, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureSampled})
		if err != nil {
			t.Fatal(err)
		}
		d.WriteTexture(tex, 0, image.Rect(0, 0, w, 4), pattern(w, 4, 4, w*4, 1), w*4)
		return tex
	}
	t1, t2, t3, t4, t5 := mk(1), mk(2), mk(3), mk(4), mk(5)
	target, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget})
	if err != nil {
		t.Fatal(err)
	}
	ub, err := d.NewBuffer(4096, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	// The layer pipeline: group 0 has three uniforms (parameters 0 to 2), group 1 the clip mask (3), group 2
	// two textures (4), group 3 two textures (5); there are no samplers.
	slot := d.next
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	group := func(g int, tex ...explicit.Texture) explicit.BindGroup {
		var es []explicit.Binding
		if g == 0 {
			es = []explicit.Binding{{Binding: 0, Buffer: ub, Size: 16}, {Binding: 1, Buffer: ub, Size: 16}, {Binding: 2, Buffer: ub, Size: 16}}
		}
		for i, x := range tex {
			es = append(es, explicit.Binding{Binding: i, Texture: x})
		}
		bg, err := d.NewBindGroup(layer, g, es)
		if err != nil {
			t.Fatalf("group %d: %+v", g, err)
		}
		return bg
	}
	g0 := group(0)
	g1a := group(1, t1)    // block 0: 1 of 6
	g2 := group(2, t2, t3) // block 0: 3 of 6
	g3 := group(3, t4, t5) // block 0: 5 of 6
	g1b := group(1, t4)    // block 0: 6 of 6 (a table of the same textures as another group's would be shared)
	p := e.BeginPass(target, explicit.LoadClear, ui.Color{})
	p.SetPipeline(layer)
	p.SetBindGroup(0, g0, 0, 256, 512)
	p.SetBindGroup(1, g1a)
	p.SetBindGroup(2, g2)
	p.SetBindGroup(3, g3)
	first := describe(proxies[slot].take(), nil)
	p.SetBindGroup(1, g1b) // still block 0
	g1c := group(1, t2)    // block 1: the block was full
	if d.slots[slot].views.alloc.blocks() != 2 {
		t.Fatalf("%d view blocks, want 2", d.slots[slot].views.alloc.blocks())
	}
	proxies[slot].take()
	p.SetBindGroup(1, g1c) // the heaps change, g2 and g3 are copied into block 1 and bound again
	second := describe(proxies[slot].take(), nil)
	p.End()
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
	count := func(lines []string, prefix string) (n int) {
		for _, l := range lines {
			if strings.HasPrefix(l, prefix) {
				n++
			}
		}
		return n
	}
	t.Logf("first:\n  %s\nsecond:\n  %s", strings.Join(first, "\n  "), strings.Join(second, "\n  "))
	if got := count(first, "SetDescriptorHeaps"); got != 1 {
		t.Errorf("the first groups set the heaps %d times, want once (the group of the same block does not set them)", got)
	}
	if got := count(second, "SetDescriptorHeaps"); got != 1 {
		t.Errorf("the group of the second block set the heaps %d times, want once", got)
	}
	// After the switch every group with a table has its table set again: groups 1, 2 and 3 are
	// parameters 3, 4 and 5 in this order (g1b before it only set its own).
	var tables []string
	seen := false
	for _, l := range second {
		if strings.HasPrefix(l, "SetDescriptorHeaps") {
			seen = true
			continue
		}
		if seen && strings.HasPrefix(l, "SetGraphicsRootDescriptorTable") {
			tables = append(tables, l)
		}
	}
	equalLines(t, tables, []string{"SetGraphicsRootDescriptorTable p3", "SetGraphicsRootDescriptorTable p4", "SetGraphicsRootDescriptorTable p5"})
	if g2.(*bindGroup).vBlock != 1 || g3.(*bindGroup).vBlock != 1 || g1c.(*bindGroup).vBlock != 1 {
		t.Errorf("blocks after the switch: g2 %d, g3 %d, g1c %d, want 1", g2.(*bindGroup).vBlock, g3.(*bindGroup).vBlock, g1c.(*bindGroup).vBlock)
	}
}

// A bind group made between two frames is for the next frame, as the Vulkan descriptor sets are.
func TestWineBindGroupMadeBetweenFramesIsForTheNextFrame(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	s := newSolidRig(t, d, 16, 16)
	d.WriteBuffer(s.inst, 0, rect(0, 0, 16, 16, 0, 1, 0, 1))
	bg := s.group(t) // no frame is open
	frame(t, d, func(e explicit.Encoder) {
		p := e.BeginPass(s.target, explicit.LoadClear, ui.Color{})
		p.SetPipeline(s.pipe)
		p.SetBindGroup(0, bg, 0)
		p.SetVertexBuffer(0, s.inst, 0)
		p.Draw(6, 1, 0, 0)
		p.End()
	})
	if got := pixel(readTexture(t, d, s.target, image.Rect(0, 0, 16, 16)), 8, 8); got != (color.RGBA{0, 255, 0, 255}) {
		t.Errorf("pixel = %v, want green", got)
	}
}
