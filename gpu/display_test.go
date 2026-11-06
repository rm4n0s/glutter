//go:build linux && glutter_display

package gpu_test

import (
	"encoding/binary"
	"image/color"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
	"github.com/rm4n0s/glutter/internal/gpu/vulkan"
	"github.com/rm4n0s/glutter/internal/gpuhost"
)

// These tests need Vulkan but no display: they run on Mesa's lavapipe, with the validation layer on, and fail on any
// validation error (SPEC §9.4, §17).

// openLavapipe opens lavapipe with validation, and checks at the end of the test that the layer reported nothing.
func openLavapipe(t *testing.T) *vulkan.Device {
	t.Helper()
	arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64", "386": "i686"}[runtime.GOARCH]
	icd := "/usr/share/vulkan/icd.d/lvp_icd." + arch + ".json"
	if _, err := os.Stat(icd); err != nil {
		t.Skip("Mesa's lavapipe is not installed (sudo dnf install mesa-vulkan-drivers)")
	}
	if _, err := os.Stat("/usr/share/vulkan/explicit_layer.d/VkLayer_khronos_validation.json"); err != nil {
		t.Skip("the validation layer is not installed (sudo dnf install vulkan-validation-layers)")
	}
	t.Setenv("VK_DRIVER_FILES", icd)
	d, err := vulkan.Open(vulkan.Options{AllowCPU: true, Validation: true, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
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

// lavapipeApp is an app whose GPU device is lavapipe.
func lavapipeApp(t *testing.T) (*gpuhost.App, *gpu.Device) {
	t.Helper()
	app := gpuhost.NewBinding(t, openLavapipe(t))
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		t.Fatal(err)
	}
	return app, d
}

// TestTriangleThroughGPU draws a triangle that covers a texture through package gpu and reads it back.
func TestTriangleThroughGPU(t *testing.T) {
	app, d := lavapipeApp(t)
	m := must(d.NewShaderModule(gpuhost.ShaderOf(devicetest.ShaderFill)))
	pipe := must(d.NewRenderPipeline(gpu.RenderPipelineDesc{
		Vertex:   gpu.VertexState{Module: m, EntryPoint: "vs_full"},
		Fragment: gpu.FragmentState{Module: m, EntryPoint: "fs_float", Targets: []gpu.ColorTargetState{{Format: gpu.TextureFormatRGBA8Unorm}}},
	}))
	const size = 16
	tex := must(d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: size, Height: size}, Format: gpu.TextureFormatRGBA8Unorm,
		Usage: gpu.TextureUsageRenderAttachment | gpu.TextureUsageCopySrc}))
	rowBytes := d.Limits().BytesPerRowAlignment
	buf := must(d.NewBuffer(gpu.BufferDesc{Size: rowBytes * size, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst}))

	enc := d.NewCommandEncoder()
	pass := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: tex.View(), ClearValue: gpu.Color{A: 1}}}})
	pass.SetPipeline(pipe)
	pass.Draw(3, 1, 0, 0)
	pass.End()
	enc.CopyTextureToBuffer(gpu.TextureCopy{Texture: tex}, gpu.BufferCopy{Buffer: buf, Layout: gpu.DataLayout{BytesPerRow: rowBytes}}, gpu.Extent3D{Width: size, Height: size, DepthOrArrayLayers: 1})
	if err := d.Queue().Submit(enc.Finish()); err != nil {
		t.Fatal(err)
	}
	var got []byte
	var gotErr error
	calls := 0
	buf.Read(0, rowBytes*size, func(data []byte, err error) { got, gotErr, calls = data, err, calls+1 })
	if calls != 0 {
		t.Fatal("done ran inside the call")
	}
	app.RunPosted()
	if calls != 1 || gotErr != nil {
		t.Fatalf("done ran %d times with %v", calls, gotErr)
	}
	want := color.RGBA{R: 64, G: 128, B: 191, A: 255}
	for _, p := range [][2]int{{0, 0}, {size - 1, 0}, {0, size - 1}, {size - 1, size - 1}, {8, 8}} {
		i := p[1]*rowBytes + p[0]*4
		have := color.RGBA{R: got[i], G: got[i+1], B: got[i+2], A: got[i+3]}
		if abs(int(have.R)-int(want.R)) > 1 || abs(int(have.G)-int(want.G)) > 1 || abs(int(have.B)-int(want.B)) > 1 || have.A != want.A {
			t.Errorf("pixel %v = %v, want %v", p, have, want)
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// TestComputeThroughGPU squares numbers with a compute shader, with a uniform buffer and storage buffers.
func TestComputeThroughGPU(t *testing.T) {
	app, d := lavapipeApp(t)
	m := must(d.NewShaderModule(gpuhost.ShaderOf(devicetest.ShaderCompute)))
	pipe := must(d.NewComputePipeline(gpu.ComputePipelineDesc{Module: m, EntryPoint: "cs_square"}))
	const n = 1000
	in := make([]byte, n*4)
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint32(in[i*4:], uint32(i))
	}
	input := must(d.NewBuffer(gpu.BufferDesc{Size: n * 4, Usage: gpu.BufferUsageStorage | gpu.BufferUsageCopyDst}))
	output := must(d.NewBuffer(gpu.BufferDesc{Size: n * 4, Usage: gpu.BufferUsageStorage | gpu.BufferUsageCopySrc}))
	params := must(d.NewBuffer(gpu.BufferDesc{Size: 256, Usage: gpu.BufferUsageUniform | gpu.BufferUsageCopyDst}))
	d.Queue().WriteBuffer(input, 0, in)
	p := make([]byte, 16)
	binary.LittleEndian.PutUint32(p, n)
	d.Queue().WriteBuffer(params, 0, p)
	g := must(d.NewBindGroup(gpu.BindGroupDesc{Layout: pipe.BindGroupLayout(0), Entries: []gpu.BindGroupEntry{
		{Binding: 0, Buffer: params, Size: 16}, {Binding: 1, Buffer: input}, {Binding: 2, Buffer: output}}}))
	enc := d.NewCommandEncoder()
	pass := enc.BeginComputePass(gpu.ComputePassDesc{})
	pass.SetPipeline(pipe)
	pass.SetBindGroup(0, g, 0)
	pass.DispatchWorkgroups((n+63)/64, 1, 1)
	pass.End()
	if err := d.Queue().Submit(enc.Finish()); err != nil {
		t.Fatal(err)
	}
	var got []byte
	output.Read(0, n*4, func(data []byte, err error) {
		if err != nil {
			t.Error(err)
		}
		got = data
	})
	app.RunPosted()
	if len(got) != n*4 {
		t.Fatalf("read %d bytes", len(got))
	}
	for i := 0; i < n; i++ {
		if v := binary.LittleEndian.Uint32(got[i*4:]); v != uint32(i*i) {
			t.Fatalf("output[%d] = %d, want %d", i, v, i*i)
		}
	}
}

// TestLossAndLeaksOnAGPU loses the device of the app and ends it, with a buffer that was not released.
func TestLossAndLeaksOnAGPU(t *testing.T) {
	app, d := lavapipeApp(t)
	buf := must(d.NewBuffer(gpu.BufferDesc{Label: "leaky", Size: 64, Usage: gpu.BufferUsageVertex}))
	lost := 0
	d.OnLost(func(error) { lost++ })
	app.Host.Replace(openLavapipe(t))
	app.RunPosted()
	if lost != 1 || !d.IsLost() {
		t.Errorf("OnLost ran %d times, lost %v", lost, d.IsLost())
	}
	buf.Release() // the old device is gone: nothing to free
	fresh, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil || fresh == d {
		t.Fatalf("DeviceOfBinding after the loss = %p, %v", fresh, err)
	}
	b := must(fresh.NewBuffer(gpu.BufferDesc{Label: "leaky", Size: 64, Usage: gpu.BufferUsageVertex}))
	_ = b
	app.Host.ClearLogs()
	app.Host.End()
	if !app.Host.LogContains("a GPU resource was not released") {
		t.Errorf("the leak was not logged:\n%s", app.Host.Logs())
	}
}
