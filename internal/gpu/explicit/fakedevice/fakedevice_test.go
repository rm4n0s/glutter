package fakedevice

import (
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/ui"
)

func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("the design limits are checked in debug builds only")
	}
}

// panicOf runs fn and returns the error it panicked with, or nil.
func panicOf(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(error)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	fn()
	return nil
}

// shaderWith is a shader whose fragment entry point reads that many resources of each
// kind, all in the group.
func shaderWith(group, textures, samplers, uniforms int) *explicit.Shader {
	var bs []explicit.ShaderBinding
	n := 0
	add := func(kind explicit.BindingKind, count int) {
		for i := 0; i < count; i++ {
			bs = append(bs, explicit.ShaderBinding{Name: "b", Group: group, Binding: n, Kind: kind})
			n++
		}
	}
	add(explicit.BindingSampledTexture, textures)
	add(explicit.BindingSampler, samplers)
	add(explicit.BindingUniformBuffer, uniforms)
	return &explicit.Shader{Name: "synthetic", Entries: []explicit.EntryPoint{
		{Name: "vs", Stage: explicit.ShaderStageVertex},
		{Name: "fs", Stage: explicit.ShaderStageFragment, Bindings: bs},
	}}
}

func layout(buffers, attributesEach int) []explicit.VertexBufferLayout {
	var out []explicit.VertexBufferLayout
	for i := 0; i < buffers; i++ {
		l := explicit.VertexBufferLayout{Stride: 16}
		for j := 0; j < attributesEach; j++ {
			l.Attributes = append(l.Attributes, explicit.VertexAttribute{Location: len(out)*attributesEach + j, Format: explicit.VertexFloat32x4})
		}
		out = append(out, l)
	}
	return out
}

func TestLimitsAreChecked(t *testing.T) {
	skipUnlessDebug(t)
	rt := explicit.TextureRenderTarget
	tests := []struct {
		name  string
		do    func(d *Device)
		limit string // the metadata "limit" of the error; "" means no panic
	}{
		{"a texture at the largest size", func(d *Device) {
			_, _ = d.NewTexture(explicit.TextureDesc{Width: 8192, Height: 8192, Format: explicit.TextureFormatR8Unorm, Usage: rt})
		}, ""},
		{"a texture too wide", func(d *Device) {
			_, _ = d.NewTexture(explicit.TextureDesc{Width: 8193, Height: 1, Format: explicit.TextureFormatRGBA8Unorm})
		}, "texture width"},
		{"a texture too tall", func(d *Device) {
			_, _ = d.NewTexture(explicit.TextureDesc{Width: 1, Height: 8193, Format: explicit.TextureFormatRGBA8Unorm})
		}, "texture height"},
		{"a swapchain image too large", func(d *Device) {
			sc, _ := d.NewSwapchain(gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake})
			_ = sc.Configure(9000, 100)
			_, _ = sc.Acquire()
		}, "texture width"},
		{"8 vertex buffers", func(d *Device) { _, _ = d.NewPipeline(explicit.PipelineDesc{VertexBuffers: layout(8, 2)}) }, ""},
		{"9 vertex buffers", func(d *Device) { _, _ = d.NewPipeline(explicit.PipelineDesc{VertexBuffers: layout(9, 1)}) }, "vertex buffers"},
		{"16 vertex attributes", func(d *Device) { _, _ = d.NewPipeline(explicit.PipelineDesc{VertexBuffers: layout(4, 4)}) }, ""},
		{"17 vertex attributes", func(d *Device) { _, _ = d.NewPipeline(explicit.PipelineDesc{VertexBuffers: layout(1, 17)}) }, "vertex attributes"},
		{"16 textures, 16 samplers and 12 uniforms in a stage", func(d *Device) {
			_, _ = d.NewPipeline(explicit.PipelineDesc{Shader: shaderWith(0, 16, 16, 12), FragmentEntry: "fs"})
		}, ""},
		{"17 sampled textures in a stage", func(d *Device) {
			_, _ = d.NewPipeline(explicit.PipelineDesc{Shader: shaderWith(0, 17, 0, 0), FragmentEntry: "fs"})
		}, "sampled textures per stage"},
		{"17 samplers in a stage", func(d *Device) {
			_, _ = d.NewPipeline(explicit.PipelineDesc{Shader: shaderWith(0, 0, 17, 0), FragmentEntry: "fs"})
		}, "samplers per stage"},
		{"13 uniform buffers in a stage", func(d *Device) {
			_, _ = d.NewPipeline(explicit.PipelineDesc{Shader: shaderWith(0, 0, 0, 13), FragmentEntry: "fs"})
		}, "uniform buffers per stage"},
		{"a shader that reads the fifth bind group", func(d *Device) {
			_, _ = d.NewPipeline(explicit.PipelineDesc{Shader: shaderWith(4, 1, 0, 0), FragmentEntry: "fs"})
		}, "bind group index"},
		{"the fourth bind group", func(d *Device) { _, _ = d.NewBindGroup(nil, 3, nil) }, ""},
		{"the fifth bind group", func(d *Device) { _, _ = d.NewBindGroup(nil, 4, nil) }, "bind group index"},
		{"SetBindGroup with the fifth group", func(d *Device) { newPass(d).SetBindGroup(4, nil) }, "bind group index"},
		{"a uniform binding of 64 KiB", func(d *Device) {
			b, _ := d.NewBuffer(64<<10, explicit.BufferUniform)
			_, _ = d.NewBindGroup(nil, 0, []explicit.Binding{{Binding: 0, Buffer: b}})
		}, ""},
		{"a uniform binding over 64 KiB, to the end of the buffer", func(d *Device) {
			b, _ := d.NewBuffer(64<<10+1, explicit.BufferUniform)
			_, _ = d.NewBindGroup(nil, 0, []explicit.Binding{{Binding: 0, Buffer: b}})
		}, "uniform binding size"},
		{"a uniform binding over 64 KiB, with a size", func(d *Device) {
			b, _ := d.NewBuffer(1<<20, explicit.BufferUniform)
			_, _ = d.NewBindGroup(nil, 0, []explicit.Binding{{Binding: 0, Buffer: b, Offset: 256, Size: 64<<10 + 4}})
		}, "uniform binding size"},
		{"dynamic offsets that are multiples of 256", func(d *Device) { newPass(d).SetBindGroup(0, nil, 0, 256, 1024) }, ""},
		{"a dynamic offset that is not", func(d *Device) { newPass(d).SetBindGroup(0, nil, 0, 100) }, "dynamic uniform offset"},
		{"the eighth vertex buffer slot", func(d *Device) { newPass(d).SetVertexBuffer(7, nil, 0) }, ""},
		{"the ninth vertex buffer slot", func(d *Device) { newPass(d).SetVertexBuffer(8, nil, 0) }, "vertex buffer slot"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := panicOf(func() { tt.do(New()) })
			if tt.limit == "" {
				if err != nil {
					t.Fatalf("panicked: %v", err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected a panic with *errors.Error, got %v", err)
			}
			if appErr.Tag != "GPULimitExceeded" || !appErr.HasRoute("exceeded.GPULimitExceeded") && !appErr.HasRoute("misaligned.GPULimitExceeded") {
				t.Errorf("got %s via %s, want GPULimitExceeded via exceeded or misaligned", appErr.Tag, appErr.Route())
			}
			if appErr.Metadata["limit"] != tt.limit {
				t.Errorf("the limit is %v, want %q", appErr.Metadata["limit"], tt.limit)
			}
		})
	}
}

// newPass makes a pass on a texture of the device.
func newPass(d *Device) explicit.Pass {
	tex, _ := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget})
	enc, _ := d.Begin()
	return enc.BeginPass(tex, explicit.LoadClear, ui.Color{})
}

func TestTheSpecifiedLimitsAreTheDefaults(t *testing.T) {
	want := explicit.Limits{
		MaxBindGroups: 4, MaxSampledTexturesPerStage: 16, MaxSamplersPerStage: 16, MaxUniformBuffersPerStage: 12,
		UniformOffsetAlignment: 256, MaxUniformBindingSize: 65536, MaxTextureSize: 8192,
		MaxVertexBuffers: 8, MaxVertexAttributes: 16, CopyRowPitchAlignment: 256,
	}
	if got := New().Limits(); got != want {
		t.Errorf("Limits = %+v, want %+v", got, want)
	}
}

func TestPassMustEndBeforeTheNext(t *testing.T) {
	d := New()
	tex, _ := d.NewTexture(explicit.TextureDesc{Width: 2, Height: 2, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget})
	enc, _ := d.Begin()
	enc.BeginPass(tex, explicit.LoadClear, ui.Color{})
	err := panicOf(func() { enc.BeginPass(tex, explicit.LoadLoad, ui.Color{}) })
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "FakePassNotEnded" || !appErr.HasRoute("encoder.BeginPass.FakePassNotEnded") {
		t.Errorf("got %v, want FakePassNotEnded via encoder.BeginPass", err)
	}
}

func TestRecordsTheCallsOfAFrame(t *testing.T) {
	d := New()
	buf, _ := d.NewBuffer(64, explicit.BufferVertex)
	d.WriteBuffer(buf, 16, []byte{1, 2, 3})
	tex, _ := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
	pl, _ := d.NewPipeline(explicit.PipelineDesc{Shader: explicit.ShaderSolid, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: explicit.TextureFormatRGBA8Unorm})
	enc, _ := d.Begin()
	pass := enc.BeginPass(tex, explicit.LoadClear, ui.Color{})
	pass.SetPipeline(pl)
	pass.SetVertexBuffer(0, buf, 0)
	pass.SetScissor(image.Rect(0, 0, 2, 2))
	pass.Draw(6, 3, 0, 0)
	pass.End()
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
	want := []string{"NewBuffer", "WriteBuffer", "NewTexture", "NewPipeline", "Begin", "BeginPass", "SetPipeline", "SetVertexBuffer", "SetScissor", "Draw", "EndPass", "Submit"}
	if got := d.Ops(); !reflect.DeepEqual(got, want) {
		t.Errorf("ops\n got %v\nwant %v", got, want)
	}
	if got := d.Calls()[9].String(); got != "Draw(vertices=6 instances=3 firstVertex=0 firstInstance=0)" {
		t.Errorf("the Draw call is %q", got)
	}
	if b := Bytes(buf); b[16] != 1 || b[18] != 3 || len(b) != 64 {
		t.Errorf("the buffer holds %v", b)
	}
	if err := d.Close(); err != nil || !d.Closed() {
		t.Errorf("Close = %v, closed %v", err, d.Closed())
	}
}

func TestReadTextureIsDeliveredAfterTheFrame(t *testing.T) {
	d := New()
	tex, _ := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 3, Format: explicit.TextureFormatBGRA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
	enc, _ := d.Begin()
	enc.BeginPass(tex, explicit.LoadClear, ui.NewColorFromRGBO(10, 20, 30, 1)).End()
	var got *image.RGBA
	calls := 0
	enc.ReadTexture(tex, image.Rect(1, 1, 3, 2), func(img *image.RGBA, err error) {
		if err != nil {
			t.Error(err)
		}
		got, calls = img, calls+1
	})
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("done ran inside the frame that asked")
	}
	d.CompleteReads()
	if calls != 1 || got == nil || got.Rect.Dx() != 2 || got.Rect.Dy() != 1 {
		t.Fatalf("done ran %d times with %v", calls, got)
	}
	if c := got.RGBAAt(0, 0); c != (color.RGBA{10, 20, 30, 255}) {
		t.Errorf("pixel %v, want the clear color (the BGRA texture reads back as RGBA)", c)
	}
	d.CompleteReads()
	if calls != 1 {
		t.Error("done ran twice")
	}
}

func TestWriteAndCopyTexture(t *testing.T) {
	d := New()
	src, _ := d.NewTexture(explicit.TextureDesc{Width: 2, Height: 2, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureSampled | explicit.TextureCopySource})
	dst, _ := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
	// Two rows of 2 pixels, with a row pitch of 12 bytes.
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 0, 0, 0, 0, 9, 10, 11, 12, 13, 14, 15, 16}
	d.WriteTexture(src, 0, image.Rect(0, 0, 2, 2), data, 12)
	enc, _ := d.Begin()
	enc.CopyTexture(src, image.Rect(0, 0, 2, 2), dst, image.Pt(1, 2))
	var got *image.RGBA
	enc.ReadTexture(dst, image.Rect(0, 0, 4, 4), func(img *image.RGBA, _ error) { got = img })
	_ = enc.Submit()
	d.CompleteReads()
	for _, c := range []struct {
		x, y int
		want color.RGBA
	}{{1, 2, color.RGBA{1, 2, 3, 4}}, {2, 2, color.RGBA{5, 6, 7, 8}}, {1, 3, color.RGBA{9, 10, 11, 12}}, {2, 3, color.RGBA{13, 14, 15, 16}}, {0, 0, color.RGBA{}}} {
		if g := got.RGBAAt(c.x, c.y); g != c.want {
			t.Errorf("pixel (%d,%d) = %v, want %v", c.x, c.y, g, c.want)
		}
	}
}

func TestErrIsReturnedOnce(t *testing.T) {
	d := New()
	d.Err = errors.New("GPUOutOfMemory", "no memory")
	if _, err := d.NewBuffer(8, explicit.BufferVertex); err == nil || !strings.Contains(err.Error(), "no memory") {
		t.Errorf("NewBuffer = %v, want the error that was set", err)
	}
	if _, err := d.NewBuffer(8, explicit.BufferVertex); err != nil {
		t.Errorf("the error was returned twice: %v", err)
	}
}

func TestSwapchain(t *testing.T) {
	d := New()
	sc, err := d.NewSwapchain(gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake})
	if err != nil {
		t.Fatal(err)
	}
	if err := sc.Configure(30, 20); err != nil {
		t.Fatal(err)
	}
	tex, err := sc.Acquire()
	if err != nil || tex.Desc().Width != 30 || tex.Desc().Height != 20 || tex.Desc().Format != explicit.TextureFormatBGRA8Unorm {
		t.Fatalf("Acquire = %v, %v", tex, err)
	}
	if err := sc.Present([]image.Rectangle{image.Rect(0, 0, 5, 5)}); err != nil {
		t.Fatal(err)
	}
	sc.Destroy()
	want := []string{"NewSwapchain", "Configure", "Acquire", "Present", "DestroySwapchain"}
	if got := d.Ops(); !reflect.DeepEqual(got, want) {
		t.Errorf("ops %v, want %v", got, want)
	}
}

func TestLevelsAndBufferWrites(t *testing.T) {
	d := New()
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, MipLevels: 3, Usage: explicit.TextureSampled | explicit.TextureRenderTarget})
	if err != nil {
		t.Fatal(err)
	}
	// Each level is a texture of its size with one level, the same one each time; level 0 is
	// the texture's own pixels.
	for i, want := range []image.Point{{8, 4}, {4, 2}, {2, 1}} {
		lv := tex.Level(i)
		if got := image.Pt(lv.Desc().Width, lv.Desc().Height); got != want || (i > 0 && lv.Desc().MipLevels != 1) {
			t.Errorf("level %d is %v with %d levels", i, got, lv.Desc().MipLevels)
		}
		if tex.Level(i) != lv {
			t.Errorf("level %d is another texture each time", i)
		}
	}
	if tex.Level(0) != tex {
		t.Error("level 0 is not the texture")
	}
	// A level has pixels of its own, written with the level number.
	d.WriteTexture(tex, 1, image.Rect(0, 0, 4, 2), []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 5, 6, 7, 8, 5, 6, 7, 8, 5, 6, 7, 8, 5, 6, 7, 8}, 16)
	if got := tex.Level(1).(*texture).read(image.Rect(0, 0, 4, 2)).RGBAAt(1, 1); got != (color.RGBA{5, 6, 7, 8}) {
		t.Errorf("level 1 pixel %v", got)
	}
	if got := tex.(*texture).read(image.Rect(0, 0, 8, 4)).RGBAAt(1, 1); got != (color.RGBA{}) {
		t.Errorf("level 0 changed: %v", got)
	}
	for _, i := range []int{-1, 3} {
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err, _ = r.(error)
				}
			}()
			tex.Level(i)
			return nil
		}()
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "FakeLevelInvalid" || !appErr.HasRoute("texture.Level.FakeLevelInvalid") {
			t.Errorf("Level(%d): got %v, want FakeLevelInvalid", i, err)
		}
	}

	// The writes of buffers are kept with the usage of the buffer.
	b, _ := d.NewBuffer(16, explicit.BufferUniform)
	d.WriteBuffer(b, 4, []byte{9, 8})
	last := d.Writes[len(d.Writes)-1]
	if last.Offset != 4 || !reflect.DeepEqual(last.Data, []byte{9, 8}) || last.Usage != explicit.BufferUniform {
		t.Errorf("write %+v", last)
	}
}
