//go:build js && wasm

package webgpu

import (
	"image"
	"syscall/js"

	"github.com/rm4n0s/errors"
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
	// features are the optional features the device was made with.
	features explicit.Features
	loss     lossState

	modules    map[string]js.Value           // the WGSL modules compiled at startup, by shader name
	extra      map[*explicit.Shader]js.Value // the modules of shaders that are not the renderer's, made when first used
	open       *encoder                      // the frame being recorded
	after      []func()                      // what waits for the frame being recorded to be submitted
	scratch    js.Value                      // a Uint8Array that data crosses into JavaScript in, reused
	scratchLen int
	dynamic    js.Value // a Uint32Array of the dynamic offsets of a bind group, reused
}

var _ explicit.Device = (*Device)(nil)

// newDevice makes the Device over what a request for a device gave.
func (p *Provider) newDevice(a *acquired) *Device {
	d := &Device{p: p, dev: a.device, queue: a.device.Get("queue"), info: a.description, limits: explicit.DefaultLimits(),
		features: explicit.Features{Float32Filterable: a.float32Filterable},
		modules:  map[string]js.Value{}, extra: map[*explicit.Shader]js.Value{}}
	for _, s := range explicit.Shaders() {
		d.modules[s.Name] = a.modules.Get(s.Name)
	}
	if a.timestampQuery {
		d.features.TimestampQuery, d.features.TimestampPeriod = true, 1 // WebGPU's timestamps are nanoseconds
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

// Features implements explicit.Device: float32-filterable and timestamp-query when the adapter has them.
func (d *Device) Features() explicit.Features { return d.features }

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
				err = errors.NewErr(tagWebGPUCallFailed, je).SetMetadata("call", call)
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
	usage     explicit.BufferUsage
	destroyed bool
}

var _ explicit.Buffer = (*buffer)(nil)

// Size implements explicit.Buffer.
func (b *buffer) Size() int { return b.size }

// Usage implements explicit.Buffer.
func (b *buffer) Usage() explicit.BufferUsage { return b.usage }

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
	d.limits.CheckBuffer(size, usage)
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
	return &buffer{d: d, buf: v, size: size, usage: usage}, nil
}

// WriteBuffer implements explicit.Device: the bytes cross into JavaScript in one copy into the
// reused array, and the queue writes them in order with the frames submitted before and after.
func (d *Device) WriteBuffer(b explicit.Buffer, offset int, data []byte) {
	vb, ok := b.(*buffer)
	if !ok || vb.d != d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "buffer"))
	}
	if offset < 0 || offset+len(data) > vb.size {
		panic(errors.New(tagWebGPUWriteOutOfRange, "the write is outside the buffer", "offset", offset, "bytes", len(data), "size", vb.size))
	}
	if len(data) == 0 || d.loss.isLost() {
		return
	}
	n := alignUp(len(data), 4)
	d.ensureScratch(n)
	js.CopyBytesToJS(d.scratch, data)
	d.queue.Call("writeBuffer", vb.buf, offset, d.scratch, 0, n)
}

// texture is a GPUTexture, or a view of one level or one layer of a GPUTexture.
type texture struct {
	d         *Device
	desc      explicit.TextureDesc
	tex       js.Value
	view      js.Value // every level and layer, for sampling (the level or the layer for a texture made by Level or Layer)
	rtView    js.Value // level 0, layer 0 only, when the view above has more
	external  bool     // the swapchain's texture: it goes at the end of the task
	destroyed bool
	// base is the mip level of the GPUTexture that this texture starts at, and baseLayer the array
	// layer.
	base      int
	baseLayer int
	parent    *texture
	levels    []*texture
	layers    map[[2]int]*texture
}

var _ explicit.Texture = (*texture)(nil)

// Desc implements explicit.Texture.
func (t *texture) Desc() explicit.TextureDesc { return t.desc }

// Level implements explicit.Texture: a texture over one level, with a view of its own. A 2D array,
// cube or 3D texture keeps its kind.
func (t *texture) Level(i int) explicit.Texture {
	if i < 0 || i >= t.desc.MipLevels {
		panic(errors.New(tagWebGPULevelInvalid, "the texture has no such mip level", "level", i, "levels", t.desc.MipLevels))
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
	if desc.Kind == explicit.TextureKind3D {
		desc.DepthOrLayers = max(desc.DepthOrLayers>>i, 1)
	}
	view := t.tex.Call("createView", viewDescriptor(t.desc.Kind, i, 1, 0, t.desc.Layers()))
	lv := &texture{d: t.d, desc: desc, tex: t.tex, view: view, base: i, parent: t}
	t.levels[i] = lv
	return lv
}

// viewDescriptor is the descriptor of a view of a texture of the kind: its dimension, levels and
// layers (a 3D texture has no layers).
func viewDescriptor(kind explicit.TextureKind, baseLevel, levels, baseLayer, layers int) map[string]any {
	v := map[string]any{"dimension": dimensionName(kind), "baseMipLevel": baseLevel, "mipLevelCount": levels}
	if kind != explicit.TextureKind3D {
		v["baseArrayLayer"], v["arrayLayerCount"] = baseLayer, layers
	}
	return v
}

// Layer implements explicit.Texture: a 2D view of one layer of one level.
func (t *texture) Layer(level, layer int) explicit.Texture {
	t.d.limits.CheckLayer(t.desc, level, layer)
	if t.desc.Kind == explicit.TextureKind2D {
		return t.Level(level)
	}
	if t.parent != nil {
		panic(errors.New(tagWebGPULevelInvalid, "Layer is called on the texture, not on a view of a level", "level", level, "layer", layer))
	}
	key := [2]int{level, layer}
	if lv := t.layers[key]; lv != nil {
		return lv
	}
	desc := t.desc
	desc.Kind, desc.DepthOrLayers, desc.MipLevels = explicit.TextureKind2D, 1, 1
	desc.Width, desc.Height = max(desc.Width>>level, 1), max(desc.Height>>level, 1)
	view := t.tex.Call("createView", viewDescriptor(explicit.TextureKind2D, level, 1, layer, 1))
	lv := &texture{d: t.d, desc: desc, tex: t.tex, view: view, base: level, baseLayer: layer, parent: t}
	if t.layers == nil {
		t.layers = map[[2]int]*texture{}
	}
	t.layers[key] = lv
	return lv
}

// targetView is the view that a pass draws into: one level of one layer.
func (t *texture) targetView() js.Value {
	if t.rtView.Truthy() {
		return t.rtView
	}
	return t.view
}

// aspect is the GPUTextureAspect that a copy of the texture reads: the depth of a depth format.
func (t *texture) aspect() string {
	if t.desc.Format.IsDepthStencil() {
		return "depth-only"
	}
	return "all"
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
	desc = desc.Normalized()
	if desc.Width < 1 || desc.Height < 1 || desc.DepthOrLayers < 1 || desc.Format.BytesPerPixel() == 0 {
		return nil, errors.New(tagWebGPUTextureInvalid, "a texture needs a size and a format", "width", desc.Width, "height", desc.Height, "format", int(desc.Format))
	}
	if desc.Usage&^desc.Format.AllowedUsage() != 0 || desc.Kind < explicit.TextureKind2D || desc.Kind > explicit.TextureKind3D {
		return nil, errors.New(tagWebGPUTextureInvalid, "the format does not allow the usage, or the kind is unknown",
			"format", int(desc.Format), "usage", int(desc.Usage), "kind", int(desc.Kind))
	}
	var t *texture
	_, err := tryCall("createTexture", func() js.Value {
		v := d.dev.Call("createTexture", map[string]any{
			"size": []any{desc.Width, desc.Height, desc.DepthOrLayers}, "dimension": textureDimensionName(desc.Kind), "format": formatName(desc.Format),
			"mipLevelCount": desc.MipLevels, "sampleCount": desc.SampleCount, "usage": textureUsage(desc.Usage),
		})
		t = &texture{d: d, desc: desc, tex: v, view: v.Call("createView", viewDescriptor(desc.Kind, 0, desc.MipLevels, 0, desc.Layers()))}
		if desc.Usage&explicit.TextureRenderTarget != 0 && desc.MipLevels > 1 && desc.Kind == explicit.TextureKind2D {
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
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "texture"))
	}
	if r.Empty() || d.loss.isLost() {
		return
	}
	d.ensureScratch(len(data))
	js.CopyBytesToJS(d.scratch, data)
	d.queue.Call("writeTexture",
		map[string]any{"texture": vt.tex, "mipLevel": vt.base + level, "origin": []any{r.Min.X, r.Min.Y, vt.baseLayer}},
		d.scratch,
		map[string]any{"offset": 0, "bytesPerRow": stride, "rowsPerImage": r.Dy()},
		[]any{r.Dx(), r.Dy(), 1})
}

// WriteTextureRegion implements explicit.Device: one copy into the reused array and one queue call.
func (d *Device) WriteTextureRegion(t explicit.Texture, region explicit.TextureRegion, data []byte, bytesPerRow, rowsPerImage int) {
	vt, ok := t.(*texture)
	if !ok || vt.d != d {
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "texture"))
	}
	if region.Size[0] <= 0 || region.Size[1] <= 0 || region.Size[2] <= 0 || d.loss.isLost() {
		return
	}
	d.ensureScratch(len(data))
	js.CopyBytesToJS(d.scratch, data)
	d.queue.Call("writeTexture",
		map[string]any{"texture": vt.tex, "mipLevel": vt.base + region.Level, "origin": []any{region.Origin[0], region.Origin[1], vt.baseLayer + region.Origin[2]}},
		d.scratch,
		map[string]any{"offset": 0, "bytesPerRow": bytesPerRow, "rowsPerImage": rowsPerImage},
		[]any{region.Size[0], region.Size[1], region.Size[2]})
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
			"addressModeW": addressName(desc.AddressW),
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
	d        *Device
	desc     explicit.PipelineDesc
	pipe     js.Value
	layouts  []js.Value
	bindings []map[int]bindingPlan
	// vertex is the vertex entry point, for the rule about the first vertex and instance.
	vertex explicit.EntryPoint
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
		e["texture"] = map[string]any{"sampleType": sampleTypeName(b.Sample), "viewDimension": bindingDimensionName(b.Dimension)}
	case explicit.BindingSampler:
		e["sampler"] = map[string]any{"type": "filtering"}
	case explicit.BindingStorageBufferReadOnly:
		e["buffer"] = map[string]any{"type": "read-only-storage"}
	case explicit.BindingStorageBuffer:
		e["buffer"] = map[string]any{"type": "storage"}
	case explicit.BindingStorageTexture:
		e["storageTexture"] = map[string]any{"access": "write-only", "format": formatName(b.StorageFormat), "viewDimension": bindingDimensionName(b.Dimension)}
	default:
		e["buffer"] = map[string]any{"type": "uniform", "hasDynamicOffset": true}
	}
	return e
}

// moduleOf returns the GPUShaderModule of a shader: the renderer's were compiled at startup; a shader
// that is not one of them (a test's, an app's) is compiled when it is first used, from its WGSL.
func (d *Device) moduleOf(s *explicit.Shader) (js.Value, error) {
	if module, ok := d.modules[s.Name]; ok && module.Truthy() {
		return module, nil
	}
	if s.WGSL == "" {
		return js.Value{}, errors.New(tagWebGPUPipelineInvalid, "the shader is not one of the renderer's and has no WGSL", "shader", s.Name)
	}
	if module, ok := d.extra[s]; ok {
		return module, nil
	}
	v, err := tryCall("createShaderModule", func() js.Value {
		return d.dev.Call("createShaderModule", map[string]any{"label": s.Name, "code": s.WGSL})
	})
	if err != nil {
		return js.Value{}, err
	}
	d.extra[s] = v
	return v, nil
}

// NewPipeline implements explicit.Device: the bind group layouts come from the bindings of the two
// entry points, with dynamic offsets for the uniforms (SPEC §9.3).
func (d *Device) NewPipeline(desc explicit.PipelineDesc) (explicit.Pipeline, error) {
	d.limits.CheckPipeline(desc)
	if err := d.usable(); err != nil {
		return nil, err
	}
	if desc.Shader == nil {
		return nil, errors.New(tagWebGPUPipelineInvalid, "a pipeline needs a shader")
	}
	vs, ok1 := desc.Shader.Entry(desc.VertexEntry)
	fs, ok2 := desc.Shader.Entry(desc.FragmentEntry)
	if !ok1 || !ok2 {
		return nil, errors.New(tagWebGPUPipelineInvalid, "the shader lacks the entry point",
			"shader", desc.Shader.Name, "vertex", desc.VertexEntry, "fragment", desc.FragmentEntry)
	}
	module, err := d.moduleOf(desc.Shader)
	if err != nil {
		return nil, err
	}
	plan := planLayouts(vs, fs)
	p := &pipeline{d: d, desc: desc, bindings: bindingsByNumber(plan), vertex: vs}
	_, err = tryCall("createRenderPipeline", func() js.Value {
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
		targets := make([]any, 0, len(desc.Targets()))
		for _, ct := range desc.Targets() {
			target := map[string]any{"format": formatName(ct.Format), "writeMask": writeMaskBits(ct.WriteMask)}
			if ct.Blend.Enabled {
				src, dst := blendFactors(ct.Blend)
				part := map[string]any{"srcFactor": src, "dstFactor": dst, "operation": "add"}
				target["blend"] = map[string]any{"color": part, "alpha": part}
			}
			targets = append(targets, target)
		}
		state := map[string]any{
			"label":     desc.Shader.Name,
			"layout":    pl,
			"vertex":    map[string]any{"module": module, "entryPoint": desc.VertexEntry, "buffers": buffers},
			"fragment":  map[string]any{"module": module, "entryPoint": desc.FragmentEntry, "targets": targets},
			"primitive": map[string]any{"topology": topologyName(desc.Topology), "cullMode": cullName(desc.CullMode), "frontFace": frontFaceName(desc.FrontFace)},
		}
		if n := desc.Samples(); n > 1 {
			state["multisample"] = map[string]any{"count": n}
		}
		if ds := desc.DepthStencil; ds.Format != explicit.TextureFormatUnspecified {
			face := func(f explicit.StencilFaceState) map[string]any {
				return map[string]any{"compare": compareName(f.Compare), "failOp": stencilOpName(f.FailOp),
					"depthFailOp": stencilOpName(f.DepthFailOp), "passOp": stencilOpName(f.PassOp)}
			}
			depth := map[string]any{
				"format": formatName(ds.Format), "depthWriteEnabled": ds.DepthWrite, "depthCompare": compareName(ds.DepthCompare),
				"depthBias": int(ds.DepthBias), "depthBiasSlopeScale": float64(ds.DepthBiasSlopeScale), "depthBiasClamp": float64(ds.DepthBiasClamp),
			}
			if ds.Format.HasStencil() {
				depth["stencilFront"], depth["stencilBack"] = face(ds.StencilFront), face(ds.StencilBack)
				depth["stencilReadMask"], depth["stencilWriteMask"] = stencilMask(ds.StencilReadMask), stencilMask(ds.StencilWriteMask)
			}
			state["depthStencil"] = depth
		}
		p.pipe = d.dev.Call("createRenderPipeline", state)
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
		panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "pipeline"))
	}
	return d.makeBindGroup(vp.layouts, vp.bindings, group, entries)
}

// makeBindGroup makes the GPUBindGroup of a group of a pipeline, a render or a compute pipeline, from
// the layouts and the bindings it was made with.
func (d *Device) makeBindGroup(layouts []js.Value, bindings []map[int]bindingPlan, group int, entries []explicit.Binding) (explicit.BindGroup, error) {
	if err := d.usable(); err != nil {
		return nil, err
	}
	if group < 0 || group >= len(layouts) {
		return nil, errors.New(tagWebGPUBindGroupInvalid, "the pipeline has no such bind group", "group", group, "groups", len(layouts))
	}
	resources := make([]any, 0, len(entries))
	for _, e := range entries {
		plan, ok := bindings[group][e.Binding]
		if !ok {
			return nil, errors.New(tagWebGPUBindGroupInvalid, "the shader has no such binding", "group", group, "binding", e.Binding)
		}
		d.limits.CheckBinding(plan.shaderBinding(), e)
		var resource any
		switch plan.Kind {
		case explicit.BindingUniformBuffer, explicit.BindingStorageBufferReadOnly, explicit.BindingStorageBuffer:
			b, ok := e.Buffer.(*buffer)
			if !ok || b.d != d {
				panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "buffer"))
			}
			size := e.Size
			if size == 0 {
				size = b.size - e.Offset
			}
			resource = map[string]any{"buffer": b.buf, "offset": e.Offset, "size": size}
		case explicit.BindingSampledTexture:
			t, ok := e.Texture.(*texture)
			if !ok || t.d != d {
				panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "texture"))
			}
			if t.desc.Kind.Dimension() != plan.Dimension || !t.desc.Format.SampleType(d.features).Fits(plan.Sample) {
				return nil, errors.New(tagWebGPUTextureBindingMismatch, "the texture is not the dimension or sample type that the shader declares",
					"group", group, "binding", e.Binding, "kind", int(t.desc.Kind), "dimension", int(plan.Dimension), "format", int(t.desc.Format), "sample", int(plan.Sample))
			}
			resource = t.view
		case explicit.BindingStorageTexture:
			t, ok := e.Texture.(*texture)
			if !ok || t.d != d {
				panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "texture"))
			}
			if t.desc.Kind.Dimension() != plan.Dimension {
				return nil, errors.New(tagWebGPUTextureBindingMismatch, "the texture is not the dimension that the shader declares",
					"group", group, "binding", e.Binding, "kind", int(t.desc.Kind), "dimension", int(plan.Dimension))
			}
			// A storage texture is one mip level: the texture's own view has all of them.
			if t.desc.MipLevels > 1 {
				t = t.Level(0).(*texture)
			}
			resource = t.view
		case explicit.BindingSampler:
			s, ok := e.Sampler.(*sampler)
			if !ok || s.d != d {
				panic(errors.New(tagWebGPUForeignResource, "the resource was not made by this WebGPU device", "resource", "sampler"))
			}
			resource = s.s
		}
		resources = append(resources, map[string]any{"binding": e.Binding, "resource": resource})
	}
	v, err := tryCall("createBindGroup", func() js.Value {
		return d.dev.Call("createBindGroup", map[string]any{"layout": layouts[group], "entries": resources})
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

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWebGPUBindGroupInvalid       = "WebGPUBindGroupInvalid"
	tagWebGPUCallFailed             = "WebGPUCallFailed"
	tagWebGPUForeignResource        = "WebGPUForeignResource"
	tagWebGPULevelInvalid           = "WebGPULevelInvalid"
	tagWebGPUPipelineInvalid        = "WebGPUPipelineInvalid"
	tagWebGPUTextureBindingMismatch = "WebGPUTextureBindingMismatch"
	tagWebGPUTextureInvalid         = "WebGPUTextureInvalid"
	tagWebGPUWriteOutOfRange        = "WebGPUWriteOutOfRange"
)

// shaderBinding is the binding of the shader that the plan was made from, as far as the checks of a
// bind group entry need it.
func (b bindingPlan) shaderBinding() explicit.ShaderBinding {
	return explicit.ShaderBinding{Binding: b.Binding, Kind: b.Kind, Dimension: b.Dimension, Sample: b.Sample, StorageFormat: b.StorageFormat}
}
