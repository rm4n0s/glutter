//go:build js && wasm

package webgpu

import (
	"image"
	"syscall/js"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// Device is a WebGPU device and its queue: an explicit.Device (SPEC §9.5). It is used on the UI
// goroutine only, except for the notes of the browser's callbacks, which are guarded.
//
// Flutter: no counterpart
type Device struct {
	p      *Provider
	id     int
	dev    js.Value
	queue  js.Value
	info   string
	limits explicit.Limits
	loss   lossState

	modules    map[string]js.Value // the WGSL modules compiled at startup, by shader name
	open       *encoder            // the frame being recorded
	after      []func()            // what waits for the frame being recorded to be submitted
	scratch    js.Value            // a Uint8Array that data crosses into JavaScript in, reused
	scratchLen int
	dynamic    js.Value // a Uint32Array of the dynamic offsets of a bind group, reused
}

var _ explicit.Device = (*Device)(nil)

// newDevice makes the Device over what a request for a device gave.
func (p *Provider) newDevice(a *acquired) *Device {
	d := &Device{p: p, dev: a.device, queue: a.device.Get("queue"), info: a.description, limits: explicit.DefaultLimits(),
		modules: map[string]js.Value{}}
	for _, s := range explicit.Shaders() {
		d.modules[s.Name] = a.modules.Get(s.Name)
	}
	d.dynamic = js.Global().Get("Uint32Array").New(16)
	d.id = p.register(d)
	p.helper.Call("watch", d.dev, d.id)
	if a.fallback {
		p.log.Info("the WebGPU adapter is a fallback adapter (software)", "adapter", a.description)
	}
	p.log.Info("WebGPU device", "adapter", a.description)
	return d
}

// Description names the backend and the adapter: "webgpu: google swiftshader".
//
// Flutter: no counterpart
func (d *Device) Description() string {
	if d.info == "" {
		return "webgpu"
	}
	return "webgpu: " + d.info
}

// Limits implements explicit.Device.
func (d *Device) Limits() explicit.Limits { return d.limits }

// noteValidation records an error of the browser and logs it.
func (d *Device) noteValidation(message string) {
	d.p.noteValidation(message)
}

// usable returns the error of a call on a device that was lost or closed.
func (d *Device) usable() error {
	if err := d.loss.err(); err != nil {
		return err
	}
	return nil
}

// tryCall runs fn, which calls into the browser, and returns what it threw as WebGPUCallFailed.
func tryCall(call string, fn func() js.Value) (v js.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			if je, ok := r.(js.Error); ok {
				err = errCallFailed(call, je)
				return
			}
			panic(r)
		}
	}()
	return fn(), nil
}

// retire runs fn when the frame being recorded was submitted (the commands that name a resource
// must be submitted before it is destroyed), at once when none is open: the work that was
// submitted before keeps what it uses (WebGPU's destroy does not touch it).
func (d *Device) retire(fn func()) {
	if d.open != nil {
		d.after = append(d.after, fn)
		return
	}
	fn()
}

// Close destroys the device and everything made with it. It can be called twice, on a lost device
// too.
//
// Flutter: no counterpart
func (d *Device) Close() error {
	if !d.loss.markClosed() {
		return nil
	}
	d.p.forget(d.id)
	_, _ = tryCall("destroy", func() js.Value { return d.dev.Call("destroy") })
	d.open, d.after = nil, nil
	return nil
}

// ensureScratch makes the reused array at least n bytes.
func (d *Device) ensureScratch(n int) {
	if d.scratchLen >= n {
		return
	}
	size := max(n, 2*d.scratchLen, 64<<10)
	d.scratch = js.Global().Get("Uint8Array").New(size)
	d.scratchLen = size
}

// buffer is a GPUBuffer.
type buffer struct {
	d         *Device
	buf       js.Value
	size      int
	destroyed bool
}

var _ explicit.Buffer = (*buffer)(nil)

// Size implements explicit.Buffer.
func (b *buffer) Size() int { return b.size }

// Destroy implements explicit.Buffer: the buffer goes when the frame being recorded is submitted.
func (b *buffer) Destroy() {
	if b.destroyed {
		return
	}
	b.destroyed = true
	b.d.retire(func() { _, _ = tryCall("destroy", func() js.Value { return b.buf.Call("destroy") }) })
}

// NewBuffer implements explicit.Device. The browser's buffers are sized in multiples of four
// bytes (queue.writeBuffer wants it), so the buffer is rounded up; Size is what was asked for.
func (d *Device) NewBuffer(size int, usage explicit.BufferUsage) (explicit.Buffer, error) {
	if err := d.usable(); err != nil {
		return nil, err
	}
	n := max(alignUp(size, 4), 4)
	v, err := tryCall("createBuffer", func() js.Value {
		return d.dev.Call("createBuffer", map[string]any{"size": n, "usage": bufferUsage(usage)})
	})
	if err != nil {
		return nil, err
	}
	return &buffer{d: d, buf: v, size: size}, nil
}

// WriteBuffer implements explicit.Device: the bytes cross into JavaScript in one copy into the
// reused array, and the queue writes them in order with the frames submitted before and after.
func (d *Device) WriteBuffer(b explicit.Buffer, offset int, data []byte) {
	vb, ok := b.(*buffer)
	if !ok || vb.d != d {
		panic(errForeign("buffer"))
	}
	if offset < 0 || offset+len(data) > vb.size {
		panic(errWriteOutOfRange(offset, len(data), vb.size))
	}
	if len(data) == 0 || d.loss.isLost() {
		return
	}
	n := alignUp(len(data), 4)
	d.ensureScratch(n)
	js.CopyBytesToJS(d.scratch, data)
	d.queue.Call("writeBuffer", vb.buf, offset, d.scratch, 0, n)
}

// texture is a GPUTexture, or a view of one level of a GPUTexture.
type texture struct {
	d         *Device
	desc      explicit.TextureDesc
	tex       js.Value
	view      js.Value // every level, for sampling (the level for a texture made by Level)
	rtView    js.Value // level 0 only, when the view above has more levels
	external  bool     // the swapchain's texture: it goes at the end of the task
	destroyed bool
	// base is the mip level of the GPUTexture that this texture starts at.
	base   int
	parent *texture
	levels []*texture
}

var _ explicit.Texture = (*texture)(nil)

// Desc implements explicit.Texture.
func (t *texture) Desc() explicit.TextureDesc { return t.desc }

// Level implements explicit.Texture: a texture over one level, with a view of its own.
func (t *texture) Level(i int) explicit.Texture {
	if i < 0 || i >= t.desc.MipLevels {
		panic(errLevelInvalid(i, t.desc.MipLevels))
	}
	if t.desc.MipLevels == 1 || t.parent != nil {
		return t
	}
	if t.levels == nil {
		t.levels = make([]*texture, t.desc.MipLevels)
	}
	if lv := t.levels[i]; lv != nil {
		return lv
	}
	desc := t.desc
	desc.Width, desc.Height, desc.MipLevels = max(desc.Width>>i, 1), max(desc.Height>>i, 1), 1
	view := t.tex.Call("createView", map[string]any{"baseMipLevel": i, "mipLevelCount": 1})
	lv := &texture{d: t.d, desc: desc, tex: t.tex, view: view, base: i, parent: t}
	t.levels[i] = lv
	return lv
}

// targetView is the view that a pass draws into: one level.
func (t *texture) targetView() js.Value {
	if t.rtView.Truthy() {
		return t.rtView
	}
	return t.view
}

// Destroy implements explicit.Texture.
func (t *texture) Destroy() {
	if t.destroyed || t.parent != nil || t.external {
		return
	}
	t.destroyed = true
	t.d.retire(func() { _, _ = tryCall("destroy", func() js.Value { return t.tex.Call("destroy") }) })
}

// NewTexture implements explicit.Device.
func (d *Device) NewTexture(desc explicit.TextureDesc) (explicit.Texture, error) {
	d.limits.CheckTexture(desc)
	if err := d.usable(); err != nil {
		return nil, err
	}
	if desc.MipLevels < 1 {
		desc.MipLevels = 1
	}
	if desc.Width < 1 || desc.Height < 1 || desc.Format.BytesPerPixel() == 0 {
		return nil, errTextureInvalid(desc.Width, desc.Height, int(desc.Format))
	}
	var t *texture
	_, err := tryCall("createTexture", func() js.Value {
		v := d.dev.Call("createTexture", map[string]any{
			"size": []any{desc.Width, desc.Height}, "format": formatName(desc.Format),
			"mipLevelCount": desc.MipLevels, "usage": textureUsage(desc.Usage),
		})
		t = &texture{d: d, desc: desc, tex: v, view: v.Call("createView")}
		if desc.Usage&explicit.TextureRenderTarget != 0 && desc.MipLevels > 1 {
			t.rtView = v.Call("createView", map[string]any{"baseMipLevel": 0, "mipLevelCount": 1})
		}
		return v
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

// WriteTexture implements explicit.Device: one copy into the reused array and one queue call.
func (d *Device) WriteTexture(t explicit.Texture, level int, r image.Rectangle, data []byte, stride int) {
	vt, ok := t.(*texture)
	if !ok || vt.d != d {
		panic(errForeign("texture"))
	}
	if r.Empty() || d.loss.isLost() {
		return
	}
	d.ensureScratch(len(data))
	js.CopyBytesToJS(d.scratch, data)
	d.queue.Call("writeTexture",
		map[string]any{"texture": vt.tex, "mipLevel": vt.base + level, "origin": []any{r.Min.X, r.Min.Y}},
		d.scratch,
		map[string]any{"offset": 0, "bytesPerRow": stride, "rowsPerImage": r.Dy()},
		[]any{r.Dx(), r.Dy()})
}

// sampler is a GPUSampler.
type sampler struct {
	s js.Value
	d *Device
}

var _ explicit.Sampler = (*sampler)(nil)

// Destroy implements explicit.Sampler: samplers have nothing to release before the garbage
// collector takes them.
func (s *sampler) Destroy() {}

// NewSampler implements explicit.Device.
func (d *Device) NewSampler(desc explicit.SamplerDesc) (explicit.Sampler, error) {
	if err := d.usable(); err != nil {
		return nil, err
	}
	v, err := tryCall("createSampler", func() js.Value {
		return d.dev.Call("createSampler", map[string]any{
			"magFilter": filterName(desc.MagFilter), "minFilter": filterName(desc.MinFilter),
			"mipmapFilter": filterName(desc.MipmapFilter),
			"addressModeU": addressName(desc.AddressU), "addressModeV": addressName(desc.AddressV),
			"addressModeW": addressName(desc.AddressU),
			"lodMinClamp":  0, "lodMaxClamp": samplerLodMax(desc),
		})
	})
	if err != nil {
		return nil, err
	}
	return &sampler{s: v, d: d}, nil
}

// pipeline is a GPURenderPipeline with the layouts of its bind groups.
type pipeline struct {
	d       *Device
	desc    explicit.PipelineDesc
	pipe    js.Value
	layouts []js.Value
	kinds   []map[int]explicit.BindingKind
}

var _ explicit.Pipeline = (*pipeline)(nil)

// Desc implements explicit.Pipeline.
func (p *pipeline) Desc() explicit.PipelineDesc { return p.desc }

// Destroy implements explicit.Pipeline: pipelines have nothing to release before the garbage
// collector takes them.
func (p *pipeline) Destroy() {}

// layoutEntry is the JavaScript description of one binding of a bind group layout.
func layoutEntry(b bindingPlan) map[string]any {
	e := map[string]any{"binding": b.Binding, "visibility": b.Visibility}
	switch b.Kind {
	case explicit.BindingSampledTexture:
		e["texture"] = map[string]any{"sampleType": "float", "viewDimension": "2d"}
	case explicit.BindingSampler:
		e["sampler"] = map[string]any{"type": "filtering"}
	default:
		e["buffer"] = map[string]any{"type": "uniform", "hasDynamicOffset": true}
	}
	return e
}

// NewPipeline implements explicit.Device: the bind group layouts come from the bindings of the two
// entry points, with dynamic offsets for the uniforms (SPEC §9.3).
func (d *Device) NewPipeline(desc explicit.PipelineDesc) (explicit.Pipeline, error) {
	d.limits.CheckPipeline(desc)
	if err := d.usable(); err != nil {
		return nil, err
	}
	if desc.Shader == nil {
		return nil, errPipelineInvalid("a pipeline needs a shader")
	}
	vs, ok1 := desc.Shader.Entry(desc.VertexEntry)
	fs, ok2 := desc.Shader.Entry(desc.FragmentEntry)
	if !ok1 || !ok2 {
		return nil, errPipelineInvalid("the shader lacks the entry point",
			"shader", desc.Shader.Name, "vertex", desc.VertexEntry, "fragment", desc.FragmentEntry)
	}
	module, ok := d.modules[desc.Shader.Name]
	if !ok || !module.Truthy() {
		return nil, errPipelineInvalid("the shader is not one of the renderer's", "shader", desc.Shader.Name)
	}
	plan := planLayouts(vs, fs)
	p := &pipeline{d: d, desc: desc, kinds: bindingKinds(plan)}
	_, err := tryCall("createRenderPipeline", func() js.Value {
		var layouts []any
		for _, g := range plan {
			entries := make([]any, len(g))
			for i, b := range g {
				entries[i] = layoutEntry(b)
			}
			l := d.dev.Call("createBindGroupLayout", map[string]any{"entries": entries})
			p.layouts = append(p.layouts, l)
			layouts = append(layouts, l)
		}
		pl := d.dev.Call("createPipelineLayout", map[string]any{"bindGroupLayouts": layouts})
		buffers := make([]any, len(desc.VertexBuffers))
		for i, vb := range desc.VertexBuffers {
			step := "vertex"
			if vb.PerInstance {
				step = "instance"
			}
			attrs := make([]any, len(vb.Attributes))
			for j, at := range vb.Attributes {
				attrs[j] = map[string]any{"shaderLocation": at.Location, "format": vertexFormatName(at.Format), "offset": at.Offset}
			}
			buffers[i] = map[string]any{"arrayStride": vb.Stride, "stepMode": step, "attributes": attrs}
		}
		target := map[string]any{"format": formatName(desc.Target)}
		if desc.Blend.Enabled {
			src, dst := blendFactors(desc.Blend)
			part := map[string]any{"srcFactor": src, "dstFactor": dst, "operation": "add"}
			target["blend"] = map[string]any{"color": part, "alpha": part}
		}
		p.pipe = d.dev.Call("createRenderPipeline", map[string]any{
			"label":     desc.Shader.Name,
			"layout":    pl,
			"vertex":    map[string]any{"module": module, "entryPoint": desc.VertexEntry, "buffers": buffers},
			"fragment":  map[string]any{"module": module, "entryPoint": desc.FragmentEntry, "targets": []any{target}},
			"primitive": map[string]any{"topology": "triangle-list", "cullMode": "none"},
		})
		return p.pipe
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// bindGroup is a GPUBindGroup.
type bindGroup struct{ g js.Value }

var _ explicit.BindGroup = (*bindGroup)(nil)

// Destroy implements explicit.BindGroup: nothing to release before the garbage collector takes it.
func (b *bindGroup) Destroy() {}

// NewBindGroup implements explicit.Device.
func (d *Device) NewBindGroup(p explicit.Pipeline, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	d.limits.CheckBindGroup(group, entries)
	vp, ok := p.(*pipeline)
	if !ok || vp.d != d {
		panic(errForeign("pipeline"))
	}
	if err := d.usable(); err != nil {
		return nil, err
	}
	if group < 0 || group >= len(vp.layouts) {
		return nil, errBindGroupInvalid("the pipeline has no such bind group", "group", group, "groups", len(vp.layouts))
	}
	resources := make([]any, 0, len(entries))
	for _, e := range entries {
		kind, ok := vp.kinds[group][e.Binding]
		if !ok {
			return nil, errBindGroupInvalid("the shader has no such binding", "group", group, "binding", e.Binding)
		}
		var resource any
		switch kind {
		case explicit.BindingUniformBuffer:
			b, ok := e.Buffer.(*buffer)
			if !ok || b.d != d {
				panic(errForeign("buffer"))
			}
			size := e.Size
			if size == 0 {
				size = b.size - e.Offset
			}
			resource = map[string]any{"buffer": b.buf, "offset": e.Offset, "size": size}
		case explicit.BindingSampledTexture:
			t, ok := e.Texture.(*texture)
			if !ok || t.d != d {
				panic(errForeign("texture"))
			}
			resource = t.view
		case explicit.BindingSampler:
			s, ok := e.Sampler.(*sampler)
			if !ok || s.d != d {
				panic(errForeign("sampler"))
			}
			resource = s.s
		}
		resources = append(resources, map[string]any{"binding": e.Binding, "resource": resource})
	}
	v, err := tryCall("createBindGroup", func() js.Value {
		return d.dev.Call("createBindGroup", map[string]any{"layout": vp.layouts[group], "entries": resources})
	})
	if err != nil {
		return nil, err
	}
	return &bindGroup{g: v}, nil
}

// NewSwapchain implements explicit.Device: the canvas of the view and its webgpu context.
func (d *Device) NewSwapchain(h gpu.SurfaceHandle) (explicit.Swapchain, error) {
	if err := d.usable(); err != nil {
		return nil, err
	}
	return d.newSwapchain(h)
}
