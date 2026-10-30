//go:build linux && glutter_display

package vulkan

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"
	"log/slog"
	"math"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/ui"
)

// These tests need Vulkan but no display: they run on Mesa's lavapipe, with the
// validation layer on, and fail on any validation error (SPEC §9.4, §17).

// lavapipe points the Vulkan loader at lavapipe's driver file, or skips the test.
func lavapipe(t *testing.T) {
	t.Helper()
	icd := lavapipeICD()
	if icd == "" {
		t.Skip("Mesa's lavapipe is not installed (sudo dnf install mesa-vulkan-drivers)")
	}
	if _, err := os.Stat("/usr/share/vulkan/explicit_layer.d/VkLayer_khronos_validation.json"); err != nil {
		t.Skip("the validation layer is not installed (sudo dnf install vulkan-validation-layers)")
	}
	t.Setenv("VK_DRIVER_FILES", icd)
}

// lavapipeICD is lavapipe's driver file for the architecture of this test, or "" when
// it is not installed. A machine with Mesa's 32-bit drivers too has lvp_icd.i686.json,
// which sorts first and which a 64-bit process cannot load.
func lavapipeICD() string {
	arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64", "386": "i686"}[runtime.GOARCH]
	file := "/usr/share/vulkan/icd.d/lvp_icd." + arch + ".json"
	if _, err := os.Stat(file); err != nil {
		return ""
	}
	return file
}

// openLavapipe opens lavapipe with validation, and checks at the end of the test that
// the layer reported nothing.
func openLavapipe(t *testing.T) *Device {
	t.Helper()
	lavapipe(t)
	d, err := Open(Options{AllowCPU: true, Validation: true, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if errs := d.ValidationErrors(); len(errs) > 0 {
			t.Errorf("the validation layer reported %d errors:\n%s", len(errs), strings.Join(errs, "\n"))
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}

func colorOf(r, g, b, a float32) ui.Color { return ui.Color{R: r, G: g, B: b, A: a} }

// target makes a render target texture that can be read back.
func target(t *testing.T, d *Device, w, h int, format explicit.TextureFormat) explicit.Texture {
	t.Helper()
	tex, err := d.NewTexture(explicit.TextureDesc{Width: w, Height: h, Format: format,
		Usage: explicit.TextureRenderTarget | explicit.TextureSampled | explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	return tex
}

// readBack reads the whole texture with a frame of its own.
func readBack(t *testing.T, d *Device, tex explicit.Texture, enc explicit.Encoder) *image.RGBA {
	t.Helper()
	var got *image.RGBA
	called := 0
	desc := tex.Desc()
	enc.ReadTexture(tex, image.Rect(0, 0, desc.Width, desc.Height), func(img *image.RGBA, err error) {
		if err != nil {
			t.Error(err)
		}
		got = img
		called++
	})
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
	if called != 1 || got == nil {
		t.Fatalf("done ran %d times", called)
	}
	return got
}

func wantPixel(t *testing.T, img *image.RGBA, x, y int, want color.RGBA) {
	t.Helper()
	if got := img.RGBAAt(x, y); got != want {
		t.Errorf("pixel (%d,%d) = %v, want %v", x, y, got, want)
	}
}

func TestOpenChoosesLavapipe(t *testing.T) {
	d := openLavapipe(t)
	if !strings.Contains(strings.ToLower(d.DeviceName()), "llvmpipe") {
		t.Errorf("device = %q, want llvmpipe", d.DeviceName())
	}
	if d.Description() != "vulkan: "+d.DeviceName() {
		t.Errorf("Description = %q", d.Description())
	}
	if d.Limits() != explicit.DefaultLimits() {
		t.Errorf("Limits = %+v", d.Limits())
	}
}

func TestClearAndReadBack(t *testing.T) {
	d := openLavapipe(t)
	for _, format := range []explicit.TextureFormat{explicit.TextureFormatRGBA8Unorm, explicit.TextureFormatBGRA8Unorm} {
		tex := target(t, d, 8, 4, format)
		enc, err := d.Begin()
		if err != nil {
			t.Fatal(err)
		}
		enc.BeginPass(tex, explicit.LoadClear, colorOf(1, 0.5, 0, 1)).End()
		img := readBack(t, d, tex, enc)
		if img.Rect.Dx() != 8 || img.Rect.Dy() != 4 {
			t.Fatalf("read %v", img.Rect)
		}
		// 0.5 is 128 as a unorm byte (127.5 rounds up on every GPU that follows the spec).
		for _, p := range []image.Point{{0, 0}, {7, 3}, {3, 2}} {
			wantPixel(t, img, p.X, p.Y, color.RGBA{255, 128, 0, 255})
		}
		tex.Destroy()
	}
}

// solidPipeline makes the pipeline of the solid rectangles over the format.
func solidPipeline(t *testing.T, d *Device, format explicit.TextureFormat, blend explicit.BlendState) explicit.Pipeline {
	t.Helper()
	p, err := d.NewPipeline(explicit.PipelineDesc{
		Shader: explicit.ShaderSolid, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: format, Blend: blend,
		VertexBuffers: []explicit.VertexBufferLayout{{Stride: 32, PerInstance: true, Attributes: []explicit.VertexAttribute{
			{Location: 0, Format: explicit.VertexFloat32x4, Offset: 0},
			{Location: 1, Format: explicit.VertexFloat32x4, Offset: 16},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func floats(vs ...float32) []byte {
	b := make([]byte, 4*len(vs))
	for i, v := range vs {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(v))
	}
	return b
}

func TestDrawSolidRectangles(t *testing.T) {
	d := openLavapipe(t)
	const w, h = 32, 16
	tex := target(t, d, w, h, explicit.TextureFormatRGBA8Unorm)
	pipe := solidPipeline(t, d, explicit.TextureFormatRGBA8Unorm, explicit.BlendState{})

	// The uniform buffer has the surface and the origin; the second frame binds it at
	// an offset that is a multiple of 256.
	uniforms, err := d.NewBuffer(512, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	d.WriteBuffer(uniforms, 0, floats(w, h, 0, 0))
	d.WriteBuffer(uniforms, 256, floats(w, h, 4, 2)) // a layer whose top left is at (4,2) of the surface
	instances, err := d.NewBuffer(64, explicit.BufferVertex)
	if err != nil {
		t.Fatal(err)
	}
	// Opaque red at (2,1) 10x6, and opaque blue at (6,4) 10x6: they overlap.
	d.WriteBuffer(instances, 0, floats(2, 1, 10, 6, 1, 0, 0, 1, 6, 4, 10, 6, 0, 0, 1, 1))
	group, err := d.NewBindGroup(pipe, 0, []explicit.Binding{{Binding: 0, Buffer: uniforms, Size: 16}})
	if err != nil {
		t.Fatal(err)
	}

	enc, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	pass := enc.BeginPass(tex, explicit.LoadClear, colorOf(0, 1, 0, 1))
	pass.SetPipeline(pipe)
	pass.SetBindGroup(0, group, 0)
	pass.SetVertexBuffer(0, instances, 0)
	pass.Draw(6, 2, 0, 0)
	pass.End()
	img := readBack(t, d, tex, enc)

	red, blue, green := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}, color.RGBA{0, 255, 0, 255}
	for _, c := range []struct {
		x, y int
		want color.RGBA
	}{
		{0, 0, green}, {1, 1, green}, {2, 1, red}, {12, 1, green}, {2, 7, green},
		{5, 3, red}, {11, 3, red}, {6, 4, blue}, {11, 6, blue}, {15, 9, blue}, {16, 4, green}, {6, 10, green}, // blue draws over red
		{20, 12, green}, {31, 15, green},
	} {
		wantPixel(t, img, c.x, c.y, c.want)
	}

	// A second frame with the second uniform offset: the same rectangles move by the
	// origin, (-4,-2).
	group2, err := d.NewBindGroup(pipe, 0, []explicit.Binding{{Binding: 0, Buffer: uniforms, Size: 16}})
	if err != nil {
		t.Fatal(err)
	}
	enc, err = d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	pass = enc.BeginPass(tex, explicit.LoadClear, colorOf(0, 1, 0, 1))
	pass.SetPipeline(pipe)
	pass.SetBindGroup(0, group2, 256)
	pass.SetVertexBuffer(0, instances, 0)
	pass.SetScissor(image.Rect(0, 0, 8, 16)) // only the left part
	pass.Draw(6, 2, 0, 0)
	pass.End()
	img = readBack(t, d, tex, enc)
	wantPixel(t, img, 0, 0, red)   // (2,1) - (4,2) = (-2,-1): the rectangle covers (0,0)
	wantPixel(t, img, 3, 3, blue)  // blue at (6,4) - (4,2) = (2,2), covers (3,3)
	wantPixel(t, img, 8, 3, green) // outside the scissor
	wantPixel(t, img, 3, 12, green)
}

func TestBlendOverTheFrame(t *testing.T) {
	d := openLavapipe(t)
	tex := target(t, d, 4, 4, explicit.TextureFormatRGBA8Unorm)
	pipe := solidPipeline(t, d, explicit.TextureFormatRGBA8Unorm, explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendOne, DstFactor: explicit.BlendOneMinusSrcAlpha})
	uniforms, _ := d.NewBuffer(256, explicit.BufferUniform)
	d.WriteBuffer(uniforms, 0, floats(4, 4, 0, 0))
	instances, _ := d.NewBuffer(32, explicit.BufferVertex)
	// Premultiplied 50% white over opaque black.
	d.WriteBuffer(instances, 0, floats(0, 0, 4, 4, 0.5, 0.5, 0.5, 0.5))
	group, err := d.NewBindGroup(pipe, 0, []explicit.Binding{{Binding: 0, Buffer: uniforms, Size: 16}})
	if err != nil {
		t.Fatal(err)
	}
	enc, _ := d.Begin()
	pass := enc.BeginPass(tex, explicit.LoadClear, colorOf(0, 0, 0, 1))
	pass.SetPipeline(pipe)
	pass.SetBindGroup(0, group, 0)
	pass.SetVertexBuffer(0, instances, 0)
	pass.Draw(6, 1, 0, 0)
	pass.End()
	img := readBack(t, d, tex, enc)
	wantPixel(t, img, 1, 1, color.RGBA{128, 128, 128, 255})
}

func TestBlitCopiesAnUploadedTexture(t *testing.T) {
	d := openLavapipe(t)
	// A 4x2 source: the top row is red, the bottom row is blue, with a row pitch of 20
	// bytes.
	src, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 2, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	row := func(c color.RGBA) []byte {
		var b []byte
		for i := 0; i < 4; i++ {
			b = append(b, c.R, c.G, c.B, c.A)
		}
		return append(b, 0xEE, 0xEE, 0xEE, 0xEE) // padding of the row pitch
	}
	d.WriteTexture(src, 0, image.Rect(0, 0, 4, 2), append(row(color.RGBA{255, 0, 0, 255}), row(color.RGBA{0, 0, 255, 255})...), 20)

	sampler, err := d.NewSampler(explicit.SamplerDesc{MinFilter: explicit.FilterNearest, MagFilter: explicit.FilterNearest})
	if err != nil {
		t.Fatal(err)
	}
	pipe, err := d.NewPipeline(explicit.PipelineDesc{Shader: explicit.ShaderBlit, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: explicit.TextureFormatBGRA8Unorm})
	if err != nil {
		t.Fatal(err)
	}
	// The Blit uniform: the rotation in quarter turns, one entry for each, 256 bytes apart.
	rotations := rotationBuffer(t, d)
	dst := target(t, d, 8, 4, explicit.TextureFormatBGRA8Unorm)
	red, blue := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}
	tests := []struct {
		name     string
		rotation uint32
		// the pixels at the top left, top right, bottom left and bottom right
		corners [4]color.RGBA
	}{
		{"upright", 0, [4]color.RGBA{red, red, blue, blue}},
		{"one turn clockwise", 1, [4]color.RGBA{blue, red, blue, red}},
		{"a half turn", 2, [4]color.RGBA{blue, blue, red, red}},
		{"three turns", 3, [4]color.RGBA{red, blue, red, blue}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, _ := d.Begin()
			// A bind group lives for the frame it is made in.
			group, err := d.NewBindGroup(pipe, 0, []explicit.Binding{{Binding: 0, Texture: src}, {Binding: 1, Sampler: sampler}, {Binding: 2, Buffer: rotations, Size: 16}})
			if err != nil {
				t.Fatal(err)
			}
			pass := enc.BeginPass(dst, explicit.LoadClear, colorOf(0, 0, 0, 1))
			pass.SetPipeline(pipe)
			pass.SetBindGroup(0, group, tt.rotation*256)
			pass.Draw(3, 1, 0, 0)
			pass.End()
			img := readBack(t, d, dst, enc)
			wantPixel(t, img, 0, 0, tt.corners[0]) // upright: the top row is red
			wantPixel(t, img, 7, 0, tt.corners[1])
			wantPixel(t, img, 0, 3, tt.corners[2])
			wantPixel(t, img, 7, 3, tt.corners[3])
		})
	}
}

// rotationBuffer is the buffer of the Blit uniform of the renderer: the rotations 0 to 3, a uniform
// offset apart (256).
func rotationBuffer(t *testing.T, d *Device) explicit.Buffer {
	t.Helper()
	b, err := d.NewBuffer(4*256, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 4*256)
	for i := 0; i < 4; i++ {
		data[i*256] = byte(i)
	}
	d.WriteBuffer(b, 0, data)
	return b
}

func TestCopyTexture(t *testing.T) {
	d := openLavapipe(t)
	a := target(t, d, 8, 8, explicit.TextureFormatRGBA8Unorm)
	b := target(t, d, 8, 8, explicit.TextureFormatRGBA8Unorm)
	enc, _ := d.Begin()
	enc.BeginPass(a, explicit.LoadClear, colorOf(1, 1, 0, 1)).End()
	enc.BeginPass(b, explicit.LoadClear, colorOf(0, 0, 0, 1)).End()
	enc.CopyTexture(a, image.Rect(0, 0, 3, 2), b, image.Pt(4, 5))
	img := readBack(t, d, b, enc)
	yellow, black := color.RGBA{255, 255, 0, 255}, color.RGBA{0, 0, 0, 255}
	wantPixel(t, img, 4, 5, yellow)
	wantPixel(t, img, 6, 6, yellow)
	wantPixel(t, img, 7, 5, black)
	wantPixel(t, img, 3, 5, black)
	wantPixel(t, img, 4, 7, black)
}

func TestManyFramesReuseTheTwoSlots(t *testing.T) {
	d := openLavapipe(t)
	tex := target(t, d, 4, 4, explicit.TextureFormatRGBA8Unorm)
	pipe := solidPipeline(t, d, explicit.TextureFormatRGBA8Unorm, explicit.BlendState{})
	uniforms, _ := d.NewBuffer(256, explicit.BufferUniform)
	d.WriteBuffer(uniforms, 0, floats(4, 4, 0, 0))
	instances, _ := d.NewBuffer(32, explicit.BufferVertex)
	d.WriteBuffer(instances, 0, floats(0, 0, 4, 4, 1, 1, 1, 1))
	for frame := 0; frame < 9; frame++ {
		// Bind groups come from the descriptor pool of the frame, and are made for
		// each frame; a texture upload and a destroyed buffer wait for the GPU.
		group, err := d.NewBindGroup(pipe, 0, []explicit.Binding{{Binding: 0, Buffer: uniforms, Size: 16}})
		if err != nil {
			t.Fatalf("frame %d: %v", frame, err)
		}
		scratch, _ := d.NewBuffer(1024, explicit.BufferVertex)
		enc, err := d.Begin()
		if err != nil {
			t.Fatalf("frame %d: %v", frame, err)
		}
		pass := enc.BeginPass(tex, explicit.LoadClear, colorOf(float32(frame%2), 0, 0, 1))
		pass.SetPipeline(pipe)
		pass.SetBindGroup(0, group, 0)
		pass.SetVertexBuffer(0, instances, 0)
		pass.Draw(6, 1, 0, 0)
		pass.End()
		scratch.Destroy()
		if frame == 8 {
			img := readBack(t, d, tex, enc)
			wantPixel(t, img, 2, 2, color.RGBA{255, 255, 255, 255})
			continue
		}
		if err := enc.Submit(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestManyBindGroupsInOneFrame(t *testing.T) {
	d := openLavapipe(t)
	pipe := solidPipeline(t, d, explicit.TextureFormatRGBA8Unorm, explicit.BlendState{})
	uniforms, _ := d.NewBuffer(256, explicit.BufferUniform)
	for i := 0; i < poolSets+10; i++ {
		if _, err := d.NewBindGroup(pipe, 0, []explicit.Binding{{Binding: 0, Buffer: uniforms, Size: 16}}); err != nil {
			t.Fatalf("bind group %d: %v", i, err)
		}
	}
}

func TestUploadLargerThanTheStagingRing(t *testing.T) {
	d := openLavapipe(t)
	const w, h = 2048, 2048 // 16 MiB, twice the share of a slot
	tex, err := d.NewTexture(explicit.TextureDesc{Width: w, Height: h, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, w*h*4)
	for i := range data {
		data[i] = byte(i / 4 % 251)
	}
	d.WriteTexture(tex, 0, image.Rect(0, 0, w, h), data, w*4)
	enc, _ := d.Begin()
	var got *image.RGBA
	enc.ReadTexture(tex, image.Rect(100, 100, 104, 101), func(img *image.RGBA, err error) {
		if err != nil {
			t.Error(err)
		}
		got = img
	})
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
	want := byte((100*w + 100) % 251)
	if got == nil || got.RGBAAt(0, 0).R != want || got.RGBAAt(3, 0).R != want+3 {
		t.Errorf("pixels = %v, want R %d, %d", got, want, want+3)
	}
}

func TestMipLevelsAndSamplerKinds(t *testing.T) {
	d := openLavapipe(t)
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 16, Height: 16, Format: explicit.TextureFormatR8Unorm, MipLevels: 5,
		Usage: explicit.TextureSampled | explicit.TextureRenderTarget})
	if err != nil {
		t.Fatal(err)
	}
	d.WriteTexture(tex, 1, image.Rect(0, 0, 8, 8), make([]byte, 64), 8)
	for _, desc := range []explicit.SamplerDesc{
		{}, {MinFilter: explicit.FilterLinear, MagFilter: explicit.FilterLinear, MipmapFilter: explicit.FilterLinear, AddressU: explicit.AddressRepeat, AddressV: explicit.AddressMirror},
	} {
		s, err := d.NewSampler(desc)
		if err != nil {
			t.Fatal(err)
		}
		s.Destroy()
	}
	enc, _ := d.Begin()
	enc.BeginPass(tex, explicit.LoadClear, colorOf(0, 0, 0, 1)).End() // a render target with more levels draws into level 0
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
}

func TestResourcesAreDestroyedOnce(t *testing.T) {
	d := openLavapipe(t)
	b, _ := d.NewBuffer(64, explicit.BufferVertex)
	tex := target(t, d, 4, 4, explicit.TextureFormatRGBA8Unorm)
	s, _ := d.NewSampler(explicit.SamplerDesc{})
	p := solidPipeline(t, d, explicit.TextureFormatRGBA8Unorm, explicit.BlendState{})
	for i := 0; i < 2; i++ {
		b.Destroy()
		tex.Destroy()
		s.Destroy()
		p.Destroy()
	}
	if len(d.live) != 0 {
		t.Errorf("%d resources are still live after Destroy", len(d.live))
	}
	// What is left when the device is closed is destroyed by Close, with no validation
	// error for an object that was not destroyed.
	_, _ = d.NewBuffer(64, explicit.BufferVertex)
	_ = target(t, d, 4, 4, explicit.TextureFormatRGBA8Unorm)
}

func TestErrorsOfTheDevice(t *testing.T) {
	d := openLavapipe(t)
	t.Run("a second frame before the first is submitted", func(t *testing.T) {
		enc, err := d.Begin()
		if err != nil {
			t.Fatal(err)
		}
		_, err = d.Begin()
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagVulkanFrameOpen || !appErr.HasRoute("Device.Begin.VulkanFrameOpen") {
			t.Errorf("got %v, want VulkanFrameOpen via Device.Begin", err)
		}
		if err := enc.Submit(); err != nil {
			t.Fatal(err)
		}
		err = enc.Submit()
		appErr, ok = errors.FromError(err)
		if !ok || appErr.Tag != tagVulkanFrameNotOpen || !appErr.HasRoute("encoder.Submit.VulkanFrameNotOpen") {
			t.Errorf("got %v, want VulkanFrameNotOpen via encoder.Submit", err)
		}
	})
	t.Run("a pass before the previous one ended", func(t *testing.T) {
		tex := target(t, d, 4, 4, explicit.TextureFormatRGBA8Unorm)
		enc, _ := d.Begin()
		pass := enc.BeginPass(tex, explicit.LoadClear, colorOf(0, 0, 0, 1))
		err := panicError(func() { enc.BeginPass(tex, explicit.LoadClear, colorOf(0, 0, 0, 1)) })
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagVulkanPassNotEnded || !appErr.HasRoute("encoder.BeginPass.VulkanPassNotEnded") {
			t.Errorf("got %v, want VulkanPassNotEnded via encoder.BeginPass", err)
		}
		pass.End()
		if err := enc.Submit(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("a resource of another device", func(t *testing.T) {
		other := &texture{d: &Device{}}
		enc, _ := d.Begin()
		err := panicError(func() { enc.BeginPass(other, explicit.LoadClear, colorOf(0, 0, 0, 1)) })
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagVulkanForeignResource || !appErr.HasRoute("encoder.BeginPass.VulkanForeignResource") {
			t.Errorf("got %v, want VulkanForeignResource via encoder.BeginPass", err)
		}
		if err := enc.Submit(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("a write outside the buffer", func(t *testing.T) {
		b, _ := d.NewBuffer(16, explicit.BufferVertex)
		err := panicError(func() { d.WriteBuffer(b, 8, make([]byte, 16)) })
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagVulkanWriteOutOfRange || !appErr.HasRoute("Device.WriteBuffer.VulkanWriteOutOfRange") {
			t.Errorf("got %v, want VulkanWriteOutOfRange via Device.WriteBuffer", err)
		}
	})
	t.Run("invalid descriptions", func(t *testing.T) {
		tests := []struct {
			name string
			do   func() error
			tag  string
		}{
			{"a texture without a format", func() error { _, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4}); return err }, "VulkanTextureInvalid"},
			{"a pipeline without a shader", func() error { _, err := d.NewPipeline(explicit.PipelineDesc{}); return err }, "VulkanPipelineInvalid"},
			{"a pipeline with an entry point that is not there", func() error {
				_, err := d.NewPipeline(explicit.PipelineDesc{Shader: explicit.ShaderBlit, VertexEntry: "nope", FragmentEntry: "fs_main"})
				return err
			}, "VulkanPipelineInvalid"},
			{"a bind group of a group the pipeline lacks", func() error {
				p := solidPipeline(t, d, explicit.TextureFormatRGBA8Unorm, explicit.BlendState{})
				_, err := d.NewBindGroup(p, 3, nil)
				return err
			}, "VulkanBindGroupInvalid"},
			{"a bind group with a binding the shader lacks", func() error {
				p := solidPipeline(t, d, explicit.TextureFormatRGBA8Unorm, explicit.BlendState{})
				_, err := d.NewBindGroup(p, 0, []explicit.Binding{{Binding: 9}})
				return err
			}, "VulkanBindGroupInvalid"},
			{"a swapchain before step 20.2", func() error { _, err := d.NewSwapchain(explicitHandle()); return err }, "VulkanSurfaceFailed"},
		}
		for _, tt := range tests {
			err := tt.do()
			if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tt.tag {
				t.Errorf("%s: got %v, want %s", tt.name, err, tt.tag)
			}
		}
	})
}

func TestLimitsPanicOnTheRealDevice(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the design limits are checked in debug builds only")
	}
	d := openLavapipe(t)
	err := panicError(func() {
		_, _ = d.NewTexture(explicit.TextureDesc{Width: 9000, Height: 1, Format: explicit.TextureFormatRGBA8Unorm})
	})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "GPULimitExceeded" || !appErr.HasRoute("Limits.CheckTexture->exceeded.GPULimitExceeded") {
		t.Errorf("got %v, want GPULimitExceeded via Limits.CheckTexture", err)
	}
}

func TestOpenErrorsWithTheDriver(t *testing.T) {
	lavapipe(t)
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	t.Run("a CPU device is not used unless Vulkan is asked for", func(t *testing.T) {
		_, err := Open(Options{Logger: quiet})
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagVulkanNoDevice || !appErr.HasRoute("Open->openDevice->chooseDevice.VulkanNoDevice") {
			t.Errorf("got %v, want VulkanNoDevice via Open->openDevice->chooseDevice", err)
		}
	})
	t.Run("GLUTTER_VULKAN_DEVICE names no device", func(t *testing.T) {
		env := map[string]string{deviceEnvVar: "no such gpu"}
		_, err := Open(Options{Logger: quiet, AllowCPU: true, Getenv: func(k string) string { return env[k] }})
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagVulkanNoDevice {
			t.Errorf("got %v, want VulkanNoDevice", err)
		}
	})
	t.Run("GLUTTER_VULKAN_DEVICE names the device", func(t *testing.T) {
		env := map[string]string{deviceEnvVar: "LLVMPIPE", validationEnvVar: "1"}
		d, err := Open(Options{Logger: quiet, AllowCPU: true, Getenv: func(k string) string { return env[k] }})
		if err != nil {
			t.Fatal(err)
		}
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Close can be called twice", func(t *testing.T) {
		d, err := Open(Options{Logger: quiet, AllowCPU: true})
		if err != nil {
			t.Fatal(err)
		}
		if err := d.Close(); err != nil || d.Close() != nil {
			t.Error("Close failed")
		}
	})
}

// panicError runs fn and returns the error it panicked with, or nil.
func panicError(fn func()) (err error) {
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

func explicitHandle() gpu.SurfaceHandle { return gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake} }

func TestValidationMessagesAreSeen(t *testing.T) {
	d := openLavapipe(t)
	if d.inst.messenger == 0 {
		t.Fatal("validation is on but there is no debug messenger")
	}
	// A buffer of size 0 breaks a rule of the specification; the layer says so through the
	// messenger, which records it. The test then clears it, so that the check at the end
	// of the test does not fail.
	a := d.arena
	defer a.rewind(a.mark())
	info := put(a, bufferCreateInfo{sType: sTypeBufferCreateInfo, size: 0, usage: bufferUsageVertex})
	out := a.out()
	if r := d.f.createBuffer(d.dev, info, 0, out); r == resultSuccess {
		d.f.destroyBuffer(d.dev, u64(out), 0)
	}
	errs := d.ValidationErrors()
	if len(errs) == 0 || !strings.Contains(errs[0], "size") {
		t.Fatalf("the layer's messages = %v, want one about the size", errs)
	}
	d.inst.mu.Lock()
	d.inst.errors = nil
	d.inst.mu.Unlock()
}

func TestNoMessengerWithoutValidation(t *testing.T) {
	lavapipe(t)
	d, err := Open(Options{AllowCPU: true, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	if d.inst.messenger != 0 || len(debugTargetsOf(d.inst)) != 0 {
		t.Error("a debug messenger was made without validation")
	}
}

// debugTargetsOf returns the registry entries of the instance.
func debugTargetsOf(i *instance) []uintptr {
	debugTargets.Lock()
	defer debugTargets.Unlock()
	var ids []uintptr
	for id, inst := range debugTargets.m {
		if inst == i {
			ids = append(ids, id)
		}
	}
	return ids
}

func TestMessengerIsUnregisteredAtClose(t *testing.T) {
	lavapipe(t)
	d, err := Open(Options{AllowCPU: true, Validation: true, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	inst := d.inst
	if len(debugTargetsOf(inst)) != 1 {
		t.Fatal("the instance is not registered for the callback")
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if len(debugTargetsOf(inst)) != 0 {
		t.Error("the instance stays registered after Close")
	}
}

// TestLevelsAreDrawnIntoAndSampled makes the mip levels of a texture as the renderer does:
// each level is drawn from the one above it, through the textures that Level returns, and the
// levels are read back, each from its own texture. The validation layer is on.
func TestLevelsAreDrawnIntoAndSampled(t *testing.T) {
	d := openLavapipe(t)
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, MipLevels: 4,
		Usage: explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	// Level 0: the left half red, the right half blue.
	pix := make([]byte, 8*4*4)
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			c := []byte{255, 0, 0, 255}
			if x >= 4 {
				c = []byte{0, 0, 255, 255}
			}
			copy(pix[4*(y*8+x):], c)
		}
	}
	d.WriteTexture(tex, 0, image.Rect(0, 0, 8, 4), pix, 32)
	sampler, err := d.NewSampler(explicit.SamplerDesc{MinFilter: explicit.FilterLinear, MagFilter: explicit.FilterLinear})
	if err != nil {
		t.Fatal(err)
	}
	pipe, err := d.NewPipeline(explicit.PipelineDesc{Shader: explicit.ShaderBlit, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: explicit.TextureFormatRGBA8Unorm})
	if err != nil {
		t.Fatal(err)
	}
	// The levels have the size of the level, and one level.
	for i, want := range []image.Point{{8, 4}, {4, 2}, {2, 1}, {1, 1}} {
		desc := tex.Level(i).Desc()
		if got := image.Pt(desc.Width, desc.Height); got != want || desc.MipLevels != 1 {
			t.Errorf("level %d is %v with %d levels, want %v with 1", i, got, desc.MipLevels, want)
		}
	}
	if tex.Level(1) != tex.Level(1) {
		t.Error("Level made another texture for the same level")
	}
	rotations := rotationBuffer(t, d)
	enc, _ := d.Begin()
	for level := 1; level < 4; level++ {
		group, err := d.NewBindGroup(pipe, 0, []explicit.Binding{{Binding: 0, Texture: tex.Level(level - 1)}, {Binding: 1, Sampler: sampler}, {Binding: 2, Buffer: rotations, Size: 16}})
		if err != nil {
			t.Fatal(err)
		}
		pass := enc.BeginPass(tex.Level(level), explicit.LoadClear, colorOf(0, 0, 0, 0))
		pass.SetPipeline(pipe)
		pass.SetBindGroup(0, group, 0)
		pass.Draw(3, 1, 0, 0)
		pass.End()
	}
	var level1, level3 *image.RGBA
	enc.ReadTexture(tex.Level(1), image.Rect(0, 0, 4, 2), func(img *image.RGBA, err error) {
		if err != nil {
			t.Error(err)
		}
		level1 = img
	})
	enc.ReadTexture(tex.Level(3), image.Rect(0, 0, 1, 1), func(img *image.RGBA, err error) {
		if err != nil {
			t.Error(err)
		}
		level3 = img
	})
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
	if level1 == nil || level3 == nil {
		t.Fatal("the levels were not read back")
	}
	red, blue := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}
	wantPixel(t, level1, 0, 0, red)
	wantPixel(t, level1, 1, 1, red)
	wantPixel(t, level1, 2, 0, blue)
	wantPixel(t, level1, 3, 1, blue)
	// The last level is the average of the two halves.
	if got := level3.RGBAAt(0, 0); got.A != 255 || got.R < 120 || got.R > 135 || got.B < 120 || got.B > 135 || got.G != 0 {
		t.Errorf("the last level is %v, want about half red and half blue", got)
	}
	if errs := d.ValidationErrors(); len(errs) > 0 {
		t.Errorf("the validation layer reported %d errors:\n%v", len(errs), errs)
	}
	// A level that does not exist is a programmer error; a texture of one level is its own level.
	flat := target(t, d, 4, 4, explicit.TextureFormatRGBA8Unorm)
	if flat.Level(0) != flat {
		t.Error("Level(0) of a texture with one level is not the texture")
	}
	for _, i := range []int{-1, 4} {
		err := panicError(func() { tex.Level(i) })
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagVulkanLevelInvalid || !appErr.HasRoute("texture.Level.VulkanLevelInvalid") {
			t.Errorf("Level(%d): got %v, want VulkanLevelInvalid", i, err)
		}
	}
}
