//go:build js && wasm

package main

import (
	"context"
	"syscall/js"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/gpu"
	igpu "github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
	"github.com/rm4n0s/glutter/internal/gpu/webgpu"
	"github.com/rm4n0s/glutter/internal/gpuhost"
)

// gpuAPIResult is the result of the mode gpuapi.
type gpuAPIResult struct {
	// Error says what failed before the checks could run.
	Error string `json:"error"`
	// Triangle is the pixel that a triangle drawn through package gpu left in the texture, read back.
	Triangle [4]int `json:"triangle"`
	// OnLost is how many times the OnLost function ran after GPUDevice.destroy(); AfterLoss are the tags of the
	// calls on the lost device (a buffer, a submission, a read).
	OnLost    int      `json:"onLost"`
	AfterLoss []string `json:"afterLoss"`
	// NewDevice is whether the device that replaced the lost one drew the triangle again.
	NewDevice [4]int `json:"newDevice"`
	// Leaked is whether the buffer that was not released was logged when the app ended.
	Leaked     bool     `json:"leaked"`
	Validation []string `json:"validation"`
}

// cleanups runs what the program registers, as a test does when it ends.
type cleanups []func()

func (c *cleanups) Helper()           {}
func (c *cleanups) Cleanup(fn func()) { *c = append(*c, fn) }

// pump runs what package gpu posted to the UI thread until done says it can stop, or the time is out. The
// reads of WebGPU finish in tasks of the browser, so the program waits for them in a loop.
func pump(app *gpuhost.App, done func() bool) bool {
	for i := 0; i < 600; i++ {
		app.RunPosted()
		if done() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// drawTriangle draws a triangle that covers a 16 by 16 texture through package gpu, reads it back, and returns
// the pixel (0, 0).
func drawTriangle(app *gpuhost.App, d *gpu.Device) ([4]int, error) {
	m, err := d.NewShaderModule(gpuhost.ShaderOf(devicetest.ShaderFill))
	if err != nil {
		return [4]int{}, err
	}
	defer m.Release()
	pipe, err := d.NewRenderPipeline(gpu.RenderPipelineDesc{
		Vertex:   gpu.VertexState{Module: m, EntryPoint: "vs_full"},
		Fragment: gpu.FragmentState{Module: m, EntryPoint: "fs_float", Targets: []gpu.ColorTargetState{{Format: gpu.TextureFormatRGBA8Unorm}}},
	})
	if err != nil {
		return [4]int{}, err
	}
	defer pipe.Release()
	const size = 16
	tex, err := d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: size, Height: size}, Format: gpu.TextureFormatRGBA8Unorm,
		Usage: gpu.TextureUsageRenderAttachment | gpu.TextureUsageCopySrc})
	if err != nil {
		return [4]int{}, err
	}
	defer tex.Release()
	rowBytes := d.Limits().BytesPerRowAlignment
	buf, err := d.NewBuffer(gpu.BufferDesc{Size: rowBytes * size, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst})
	if err != nil {
		return [4]int{}, err
	}
	defer buf.Release()
	enc := d.NewCommandEncoder()
	pass := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: tex.View(), ClearValue: gpu.Color{A: 1}}}})
	pass.SetPipeline(pipe)
	pass.Draw(3, 1, 0, 0)
	pass.End()
	enc.CopyTextureToBuffer(gpu.TextureCopy{Texture: tex}, gpu.BufferCopy{Buffer: buf, Layout: gpu.DataLayout{BytesPerRow: rowBytes}}, gpu.Extent3D{Width: size, Height: size, DepthOrArrayLayers: 1})
	if err := d.Queue().Submit(enc.Finish()); err != nil {
		return [4]int{}, err
	}
	var got []byte
	var readErr error
	finished := false
	buf.Read(0, rowBytes*size, func(data []byte, err error) { got, readErr, finished = data, err, true })
	if !pump(app, func() bool { return finished }) {
		return [4]int{}, errors.New(tagGPUWebTestReadTimeout, "the read of the texture did not finish in 30 seconds")
	}
	if readErr != nil {
		return [4]int{}, readErr
	}
	return [4]int{int(got[0]), int(got[1]), int(got[2]), int(got[3])}, nil
}

// gpuAPI runs package gpu over the WebGPU device: a triangle read back, the loss of the device under it
// (GPUDevice.destroy()), what the calls on the lost device return, a new device, and a leak at the end.
func gpuAPI() gpuAPIResult {
	var res gpuAPIResult
	eval(patches)
	patch("recordDevices")
	newCanvas(0, 64, 64)
	ready := make(chan struct{}, 8)
	prov, err := webgpu.Acquire(context.Background(), webgpu.Options{
		Timeout:       time.Minute,
		OnDeviceReady: func(igpu.SurfaceHandle) { ready <- struct{}{} },
	})
	if err != nil {
		res.Error = "Acquire: " + describe(err)
		return res
	}
	dev, err := prov.Open(handleOf(0))
	if err != nil {
		res.Error = "Open: " + describe(err)
		return res
	}
	var tb cleanups
	app := gpuhost.NewBinding(&tb, dev)
	defer func() {
		for i := len(tb) - 1; i >= 0; i-- {
			tb[i]()
		}
	}()
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		res.Error = "DeviceOfBinding: " + describe(err)
		return res
	}
	if res.Triangle, err = drawTriangle(app, d); err != nil {
		res.Error = "the triangle: " + describe(err)
		return res
	}

	// The device is destroyed under the app. The owner of the device (the renderer, here the test) tells package
	// gpu, and a new device comes 400 ms late.
	d.OnLost(func(error) { res.OnLost++ })
	leaky, err := d.NewBuffer(gpu.BufferDesc{Label: "leaky", Size: 64, Usage: gpu.BufferUsageVertex})
	if err != nil {
		res.Error = "the buffer: " + describe(err)
		return res
	}
	js.Global().Set("__delay", 400)
	destroyLatest()
	app.Host.Lose()
	app.RunPosted()
	_, err = d.NewBuffer(gpu.BufferDesc{Size: 4, Usage: gpu.BufferUsageVertex})
	res.AfterLoss = append(res.AfterLoss, tagOf(err).Tag)
	res.AfterLoss = append(res.AfterLoss, tagOf(d.Queue().Submit()).Tag)
	leaky.Release() // the device is gone: nothing to free
	select {
	case <-ready:
	case <-time.After(30 * time.Second):
		res.Error = "the new device did not come"
		return res
	}
	fresh, err := prov.Open(handleOf(0))
	if err != nil {
		res.Error = "Open after the loss: " + describe(err)
		return res
	}
	app.Host.Open(fresh)
	nd, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		res.Error = "DeviceOfBinding after the loss: " + describe(err)
		return res
	}
	if res.NewDevice, err = drawTriangle(app, nd); err != nil {
		res.Error = "the triangle on the new device: " + describe(err)
		return res
	}
	if _, err := nd.NewBuffer(gpu.BufferDesc{Label: "leaky again", Size: 64, Usage: gpu.BufferUsageVertex}); err != nil {
		res.Error = "the buffer on the new device: " + describe(err)
		return res
	}
	app.Host.ClearLogs()
	app.Host.End()
	res.Leaked = app.Host.LogContains("leaky again")
	res.Validation = prov.ValidationErrors()
	return res
}
