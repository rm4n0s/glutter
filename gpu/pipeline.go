package gpu

import (
	"sort"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// pipelineRef is what render and compute pipelines have in common for the bind groups that they use.
type pipelineRef interface {
	// layout returns the bind group layout of group i, which is empty for a group the shader does not use.
	layout(i int) *BindGroupLayout
	// groupCount is the number of groups the shader uses: the highest group number plus 1.
	groupCount() int
	// newGroup makes the explicit bind group of group i from entries.
	newGroup(dev explicit.Device, i int, entries []explicit.Binding) (explicit.BindGroup, error)
}

// BindGroupLayout says which resources a bind group of a pipeline holds: the bindings that the shaders of the
// pipeline declare in one group (WebGPU's GPUBindGroupLayout). A pipeline makes its layouts, from the WGSL:
// RenderPipeline.BindGroupLayout and ComputePipeline.BindGroupLayout. A layout belongs to its pipeline and goes
// with it.
//
// Flutter: no counterpart
type BindGroupLayout struct {
	owner pipelineRef
	group int
	// bindings are the bindings of the group, by binding number.
	bindings []ShaderBinding
}

// Group returns the number of the group in the shader.
//
// Flutter: no counterpart
func (l *BindGroupLayout) Group() int { return l.group }

// Bindings returns the bindings of the group, by binding number. Each is what the WGSL declares.
//
// Flutter: no counterpart
func (l *BindGroupLayout) Bindings() []ShaderBinding {
	return append([]ShaderBinding(nil), l.bindings...)
}

// binding returns the binding number n of the group.
func (l *BindGroupLayout) binding(n int) (ShaderBinding, bool) {
	i := sort.Search(len(l.bindings), func(i int) bool { return l.bindings[i].Binding >= n })
	if i < len(l.bindings) && l.bindings[i].Binding == n {
		return l.bindings[i], true
	}
	return ShaderBinding{}, false
}

// dynamicCount is the number of dynamic offsets that SetBindGroup takes for the group: one for each uniform buffer.
func (l *BindGroupLayout) dynamicCount() int {
	n := 0
	for _, b := range l.bindings {
		if b.Kind == BindingUniformBuffer {
			n++
		}
	}
	return n
}

// sameShape says whether a bind group made for the layout fits a pipeline that has other as the layout of the
// group: the bindings are the same, with the same kinds.
func (l *BindGroupLayout) sameShape(other *BindGroupLayout) bool {
	if l == other {
		return true
	}
	if l.group != other.group || len(l.bindings) != len(other.bindings) {
		return false
	}
	for i, a := range l.bindings {
		b := other.bindings[i]
		if a.Binding != b.Binding || a.Kind != b.Kind || a.Dimension != b.Dimension || a.Sample != b.Sample || a.StorageFormat != b.StorageFormat {
			return false
		}
	}
	return true
}

// mergeBindings gathers the bindings of the entry points by group. A binding that two entry points declare must be
// the same resource: when it is not, conflict is that binding.
func mergeBindings(entries ...EntryPoint) (groups [][]ShaderBinding, conflict *ShaderBinding) {
	type slot struct{ group, binding int }
	merged := map[slot]ShaderBinding{}
	top := -1
	for _, e := range entries {
		for _, b := range e.Bindings {
			s := slot{b.Group, b.Binding}
			if old, ok := merged[s]; ok {
				if old.Kind != b.Kind || old.Dimension != b.Dimension || old.Sample != b.Sample || old.StorageFormat != b.StorageFormat {
					c := b
					return nil, &c
				}
				b.Size = max(b.Size, old.Size)
			}
			merged[s] = b
			top = max(top, b.Group)
		}
	}
	groups = make([][]ShaderBinding, top+1)
	for _, b := range merged {
		groups[b.Group] = append(groups[b.Group], b)
	}
	for _, g := range groups {
		sort.Slice(g, func(i, j int) bool { return g[i].Binding < g[j].Binding })
	}
	return groups, nil
}

// makeLayouts makes the layouts of the groups for a pipeline.
func makeLayouts(owner pipelineRef, groups [][]ShaderBinding) []*BindGroupLayout {
	out := make([]*BindGroupLayout, len(groups))
	for i, g := range groups {
		out[i] = &BindGroupLayout{owner: owner, group: i, bindings: g}
	}
	return out
}

// layoutAt returns the layout of group i of a pipeline whose layouts are layouts.
func layoutAt(owner pipelineRef, layouts []*BindGroupLayout, i int) *BindGroupLayout {
	if i >= 0 && i < len(layouts) {
		return layouts[i]
	}
	return &BindGroupLayout{owner: owner, group: i}
}

// passesBackendError says whether an error of the backend is returned as it is by a pipeline constructor. The ones
// that are not about the device or its memory are about the shader.
func passesBackendError(err error) bool {
	appErr, ok := errors.FromError(err)
	if !ok {
		return false
	}
	switch appErr.Tag {
	case TagGPUOutOfMemory, TagGPUDeviceLost, explicit.TagVulkanDeviceLost, explicit.GPUDevicePending:
		return true
	}
	return false
}

// vertexClass is how a shader reads a vertex attribute.
type vertexClass int

const (
	classFloat vertexClass = iota + 1
	classSint
	classUint
)

func (f VertexFormat) class() vertexClass {
	switch f {
	case VertexFormatUint32, VertexFormatUint32x2, VertexFormatUint32x3, VertexFormatUint32x4, VertexFormatUint8x2, VertexFormatUint8x4,
		VertexFormatUint16x2, VertexFormatUint16x4:
		return classUint
	case VertexFormatSint32, VertexFormatSint32x2, VertexFormatSint32x3, VertexFormatSint32x4, VertexFormatSint8x2, VertexFormatSint8x4,
		VertexFormatSint16x2, VertexFormatSint16x4:
		return classSint
	case VertexFormatUnspecified:
		return 0
	}
	return classFloat
}

func (t ShaderType) class() vertexClass {
	switch t {
	case ShaderTypeFloat32, ShaderTypeFloat32x2, ShaderTypeFloat32x3, ShaderTypeFloat32x4:
		return classFloat
	case ShaderTypeSint32, ShaderTypeSint32x2, ShaderTypeSint32x3, ShaderTypeSint32x4:
		return classSint
	case ShaderTypeUint32, ShaderTypeUint32x2, ShaderTypeUint32x3, ShaderTypeUint32x4:
		return classUint
	}
	return 0
}

// RenderPipeline is the state of a draw: the vertex and fragment functions of a shader, the layout of the vertex
// buffers, the targets with their blending, the primitives and the depth and stencil tests (WebGPU's
// GPURenderPipeline). A pass draws with one set by SetPipeline.
//
// Flutter: no counterpart
type RenderPipeline struct {
	resource
	exp    explicit.Pipeline
	vs, fs EntryPoint
	// buffers are the vertex buffer layouts, by slot.
	buffers []VertexBufferLayout
	targets []ColorTargetState
	// depth is the format of the depth-stencil attachment; TextureFormatUnspecified for none.
	depth   TextureFormat
	samples int
	layouts []*BindGroupLayout
}

func (p *RenderPipeline) layout(i int) *BindGroupLayout { return layoutAt(p, p.layouts, i) }
func (p *RenderPipeline) groupCount() int               { return len(p.layouts) }
func (p *RenderPipeline) newGroup(dev explicit.Device, i int, entries []explicit.Binding) (explicit.BindGroup, error) {
	return dev.NewBindGroup(p.exp, i, entries)
}

// BindGroupLayout returns the layout of bind group i of the pipeline: the bindings the vertex and fragment functions
// declare in `@group(i)`. A group the shader does not use has a layout with no bindings. i is from 0 to
// Limits.MaxBindGroups - 1.
//
// Flutter: no counterpart
func (p *RenderPipeline) BindGroupLayout(i int) *BindGroupLayout {
	p.d.assertThread("RenderPipeline.BindGroupLayout")
	p.usable("RenderPipeline.BindGroupLayout")
	misuse(i >= 0 && i < p.d.limits.MaxBindGroups, "RenderPipeline.BindGroupLayout", "the group is below Limits.MaxBindGroups", "group", i)
	return p.layout(i)
}

// validateVertex checks the vertex buffers and the inputs of the vertex function. It returns the field and the
// reason of the first fault, or the empty field.
func (d *Device) validateVertex(buffers []VertexBufferLayout, vs EntryPoint) (field, reason string, value int, limit bool) {
	if len(buffers) > d.limits.MaxVertexBuffers {
		return "Vertex.Buffers", "the pipeline has more vertex buffers than Limits.MaxVertexBuffers", len(buffers), true
	}
	attributes := 0
	locations := map[int]VertexFormat{}
	for _, b := range buffers {
		attributes += len(b.Attributes)
		if b.ArrayStride < 1 || b.ArrayStride%4 != 0 {
			return "Vertex.Buffers.ArrayStride", "the stride of a vertex buffer is a positive multiple of 4", b.ArrayStride, false
		}
		for _, a := range b.Attributes {
			size := explicit.VertexFormat(a.Format).Size()
			switch {
			case a.Format.class() == 0 || size == 0:
				return "Vertex.Buffers.Attributes.Format", "a vertex attribute has a format", int(a.Format), false
			case a.Offset < 0 || a.Offset+size > b.ArrayStride:
				return "Vertex.Buffers.Attributes.Offset", "a vertex attribute is inside the stride of its buffer", a.Offset, false
			case a.ShaderLocation < 0 || a.ShaderLocation >= d.limits.MaxVertexAttributes:
				return "Vertex.Buffers.Attributes.ShaderLocation", "the location of a vertex attribute is below Limits.MaxVertexAttributes", a.ShaderLocation, true
			}
			if _, dup := locations[a.ShaderLocation]; dup {
				return "Vertex.Buffers.Attributes.ShaderLocation", "two vertex attributes have one location", a.ShaderLocation, false
			}
			locations[a.ShaderLocation] = a.Format
		}
	}
	if attributes > d.limits.MaxVertexAttributes {
		return "Vertex.Buffers", "the pipeline has more vertex attributes than Limits.MaxVertexAttributes", attributes, true
	}
	for _, in := range vs.Inputs {
		f, ok := locations[in.Location]
		if !ok {
			return "Vertex.Buffers", "the vertex function reads a location that no vertex attribute has", in.Location, false
		}
		if want := in.Type.class(); want != 0 && f.class() != want {
			return "Vertex.Buffers.Attributes.Format", "a vertex attribute has the type of the input of the vertex function", in.Location, false
		}
	}
	return "", "", 0, false
}

// validateTargets checks the color targets and the depth-stencil state.
func (d *Device) validateTargets(desc RenderPipelineDesc) (field, reason string, value int, limit bool) {
	targets := desc.Fragment.Targets
	if len(targets) < 1 {
		return "Fragment.Targets", "a pipeline has at least one color target", 0, false
	}
	if len(targets) > d.limits.MaxColorAttachments {
		return "Fragment.Targets", "the pipeline has more color targets than Limits.MaxColorAttachments", len(targets), true
	}
	bytes := 0
	for _, t := range targets {
		f := explicit.TextureFormat(t.Format)
		if !t.Format.valid() || f.AllowedUsage()&explicit.TextureRenderTarget == 0 || f.IsDepthStencil() {
			return "Fragment.Targets.Format", "a color target has a color format that can be drawn into", int(t.Format), false
		}
		bytes += f.RenderTargetPixelBytes()
		if t.Blend.Enabled && (t.Blend.SrcFactor < BlendFactorUnspecified || t.Blend.SrcFactor > BlendFactorOneMinusDstAlpha ||
			t.Blend.DstFactor < BlendFactorUnspecified || t.Blend.DstFactor > BlendFactorOneMinusDstAlpha) {
			return "Fragment.Targets.Blend", "a blend factor is one of the BlendFactor values", int(t.Blend.SrcFactor), false
		}
		if t.WriteMask < 0 || t.WriteMask > 31 {
			return "Fragment.Targets.WriteMask", "a write mask is made of the ColorWriteMask values", int(t.WriteMask), false
		}
	}
	if bytes > d.limits.MaxColorAttachmentBytesPerSample {
		return "Fragment.Targets", "the targets take more bytes per sample than Limits.MaxColorAttachmentBytesPerSample", bytes, true
	}
	p := desc.Primitive
	if p.Topology < PrimitiveTopologyTriangleList || p.Topology > PrimitiveTopologyPointList {
		return "Primitive.Topology", "the topology is one of the PrimitiveTopology values", int(p.Topology), false
	}
	if p.CullMode < CullModeNone || p.CullMode > CullModeBack {
		return "Primitive.CullMode", "the cull mode is one of the CullMode values", int(p.CullMode), false
	}
	if p.FrontFace < FrontFaceCCW || p.FrontFace > FrontFaceCW {
		return "Primitive.FrontFace", "the front face is one of the FrontFace values", int(p.FrontFace), false
	}
	if n := desc.Multisample.Count; n != 0 && n != 1 && n != 4 {
		return "Multisample.Count", "a pipeline has 1 or 4 samples", n, false
	}
	if ds := desc.DepthStencil; ds != nil {
		if !ds.Format.valid() || !ds.Format.isDepthStencil() {
			return "DepthStencil.Format", "the depth-stencil state has a depth format", int(ds.Format), false
		}
		if ds.DepthCompare < CompareFunctionUnspecified || ds.DepthCompare > CompareFunctionAlways {
			return "DepthStencil.DepthCompare", "the depth comparison is one of the CompareFunction values", int(ds.DepthCompare), false
		}
		if (ds.DepthBias != 0 || ds.DepthBiasSlopeScale != 0 || ds.DepthBiasClamp != 0) &&
			p.Topology != PrimitiveTopologyTriangleList && p.Topology != PrimitiveTopologyTriangleStrip {
			return "DepthStencil.DepthBias", "only triangles have a depth bias", int(p.Topology), false
		}
	}
	return "", "", 0, false
}

// NewRenderPipeline makes a pipeline (WebGPU's GPUDevice.createRenderPipeline). It fails with GPUInvalidDescriptor
// and GPULimitExceeded for a description that cannot be made (the metadata "field", "value" and the reason in the
// message say which part: an entry point the shader lacks, a vertex input with no attribute or of another type, a
// target that cannot be drawn into, a binding that the two functions declare differently), with GPUShaderCompileFailed
// when the backend cannot compile the shader (its error is the cause), with GPUDeviceLost on a lost device, and with
// GPUOutOfMemory.
//
// Flutter: no counterpart
func (d *Device) NewRenderPipeline(desc RenderPipelineDesc) (*RenderPipeline, error) {
	d.assertThread("NewRenderPipeline")
	if d.lostErr != nil {
		return nil, d.lostErr
	}
	vm, fm := desc.Vertex.Module, desc.Fragment.Module
	if vm == nil || fm == nil || vm.d != d || fm.d != d {
		return nil, errors.New(TagGPUInvalidDescriptor, "a pipeline has a vertex module and a fragment module made by this device", "field", "Vertex.Module", "label", desc.Label)
	}
	vm.usable("NewRenderPipeline")
	fm.usable("NewRenderPipeline")
	vs, ok := vm.shader.Entry(desc.Vertex.EntryPoint)
	if !ok || vs.Stage != ShaderStageVertex {
		return nil, errors.New(TagGPUInvalidDescriptor, "the shader has a vertex entry point of this name", "field", "Vertex.EntryPoint", "value", desc.Vertex.EntryPoint, "label", desc.Label)
	}
	fs, ok := fm.shader.Entry(desc.Fragment.EntryPoint)
	if !ok || fs.Stage != ShaderStageFragment {
		return nil, errors.New(TagGPUInvalidDescriptor, "the shader has a fragment entry point of this name", "field", "Fragment.EntryPoint", "value", desc.Fragment.EntryPoint, "label", desc.Label)
	}
	for _, e := range []EntryPoint{vs, fs} {
		for _, b := range e.Bindings {
			if b.Kind == BindingStorageBuffer || b.Kind == BindingStorageTexture {
				return nil, errors.New(TagGPUInvalidDescriptor, "only a compute entry point writes storage", "field", "EntryPoint", "entry", e.Name, "binding", b.Name, "label", desc.Label)
			}
		}
	}
	if field, reason, value, limit := d.validateVertex(desc.Vertex.Buffers, vs); field != "" {
		if limit {
			return nil, errors.New(TagGPULimitExceeded, reason, "field", field, "value", value, "label", desc.Label)
		}
		return nil, errors.New(TagGPUInvalidDescriptor, reason, "field", field, "value", value, "label", desc.Label)
	}
	if field, reason, value, limit := d.validateTargets(desc); field != "" {
		if limit {
			return nil, errors.New(TagGPULimitExceeded, reason, "field", field, "value", value, "label", desc.Label)
		}
		return nil, errors.New(TagGPUInvalidDescriptor, reason, "field", field, "value", value, "label", desc.Label)
	}
	groups, conflict := mergeBindings(vs, fs)
	if conflict != nil {
		return nil, errors.New(TagGPUInvalidDescriptor, "the vertex and the fragment function declare a binding differently", "field", "EntryPoint",
			"binding", conflict.Name, "group", conflict.Group, "number", conflict.Binding, "label", desc.Label)
	}
	if len(groups) > d.limits.MaxBindGroups {
		return nil, errors.New(TagGPULimitExceeded, "the shader uses more bind groups than Limits.MaxBindGroups", "field", "EntryPoint", "value", len(groups), "max", d.limits.MaxBindGroups, "label", desc.Label)
	}
	samples := max(desc.Multisample.Count, 1)
	pd := explicit.PipelineDesc{Shader: vm.exp, VertexEntry: vs.Name, FragmentEntry: fs.Name, SampleCount: samples,
		Topology: explicit.PrimitiveTopology(desc.Primitive.Topology), CullMode: explicit.CullMode(desc.Primitive.CullMode), FrontFace: explicit.FrontFace(desc.Primitive.FrontFace)}
	if fm.shader != vm.shader {
		// The two functions are in one shader for the backends.
		return nil, errors.New(TagGPUInvalidDescriptor, "the vertex and the fragment function are in one shader", "field", "Fragment.Module", "label", desc.Label)
	}
	for _, b := range desc.Vertex.Buffers {
		vb := explicit.VertexBufferLayout{Stride: b.ArrayStride, PerInstance: b.PerInstance}
		for _, a := range b.Attributes {
			vb.Attributes = append(vb.Attributes, explicit.VertexAttribute{Location: a.ShaderLocation, Format: explicit.VertexFormat(a.Format), Offset: a.Offset})
		}
		pd.VertexBuffers = append(pd.VertexBuffers, vb)
	}
	targets := append([]ColorTargetState(nil), desc.Fragment.Targets...)
	for _, t := range targets {
		pd.ColorTargets = append(pd.ColorTargets, explicit.ColorTargetState{Format: explicit.TextureFormat(t.Format),
			Blend:     explicit.BlendState{Enabled: t.Blend.Enabled, SrcFactor: explicit.BlendFactor(t.Blend.SrcFactor), DstFactor: explicit.BlendFactor(t.Blend.DstFactor)},
			WriteMask: explicit.ColorWriteMask(t.WriteMask)})
	}
	depth := TextureFormatUnspecified
	if ds := desc.DepthStencil; ds != nil {
		depth = ds.Format
		pd.DepthStencil = explicit.DepthStencilState{Format: explicit.TextureFormat(ds.Format), DepthWrite: ds.DepthWriteEnabled, DepthCompare: explicit.CompareFunc(ds.DepthCompare),
			StencilFront: explicit.StencilFaceState{Compare: explicit.CompareFunc(ds.StencilFront.Compare), FailOp: explicit.StencilOp(ds.StencilFront.FailOp),
				DepthFailOp: explicit.StencilOp(ds.StencilFront.DepthFailOp), PassOp: explicit.StencilOp(ds.StencilFront.PassOp)},
			StencilBack: explicit.StencilFaceState{Compare: explicit.CompareFunc(ds.StencilBack.Compare), FailOp: explicit.StencilOp(ds.StencilBack.FailOp),
				DepthFailOp: explicit.StencilOp(ds.StencilBack.DepthFailOp), PassOp: explicit.StencilOp(ds.StencilBack.PassOp)},
			StencilReadMask: ds.StencilReadMask, StencilWriteMask: ds.StencilWriteMask,
			DepthBias: ds.DepthBias, DepthBiasSlopeScale: ds.DepthBiasSlopeScale, DepthBiasClamp: ds.DepthBiasClamp}
	}
	exp, err := d.dev.NewPipeline(pd)
	if err != nil {
		if passesBackendError(err) {
			return nil, err
		}
		return nil, errors.NewErr(TagGPUShaderCompileFailed, err).SetMetadata("label", desc.Label, "shader", vm.shader.Name)
	}
	p := &RenderPipeline{resource: d.track(kindRenderPipeline, desc.Label, 1), exp: exp, vs: vs, fs: fs, targets: targets, depth: depth, samples: samples}
	for _, b := range desc.Vertex.Buffers {
		b.Attributes = append([]VertexAttribute(nil), b.Attributes...)
		p.buffers = append(p.buffers, b)
	}
	p.layouts = makeLayouts(p, groups)
	return p, nil
}

// Release frees the pipeline, when the work that uses it is done. It may be called twice.
//
// Flutter: no counterpart
func (p *RenderPipeline) Release() {
	p.d.assertThread("RenderPipeline.Release")
	if !p.untrack() {
		return
	}
	if p.d.lostErr == nil {
		p.exp.Destroy()
	}
}

// ComputePipeline is a compute function of a shader (WebGPU's GPUComputePipeline). A compute pass runs one set
// by SetPipeline.
//
// Flutter: no counterpart
type ComputePipeline struct {
	resource
	exp     explicit.ComputePipeline
	entry   EntryPoint
	layouts []*BindGroupLayout
}

func (p *ComputePipeline) layout(i int) *BindGroupLayout { return layoutAt(p, p.layouts, i) }
func (p *ComputePipeline) groupCount() int               { return len(p.layouts) }
func (p *ComputePipeline) newGroup(dev explicit.Device, i int, entries []explicit.Binding) (explicit.BindGroup, error) {
	return dev.NewComputeBindGroup(p.exp, i, entries)
}

// BindGroupLayout returns the layout of bind group i of the pipeline, as RenderPipeline.BindGroupLayout does.
//
// Flutter: no counterpart
func (p *ComputePipeline) BindGroupLayout(i int) *BindGroupLayout {
	p.d.assertThread("ComputePipeline.BindGroupLayout")
	p.usable("ComputePipeline.BindGroupLayout")
	misuse(i >= 0 && i < p.d.limits.MaxBindGroups, "ComputePipeline.BindGroupLayout", "the group is below Limits.MaxBindGroups", "group", i)
	return p.layout(i)
}

// WorkgroupSize returns the @workgroup_size of the compute function.
//
// Flutter: no counterpart
func (p *ComputePipeline) WorkgroupSize() [3]int { return p.entry.WorkgroupSize }

// NewComputePipeline makes a compute pipeline (WebGPU's GPUDevice.createComputePipeline). It fails with
// GPUInvalidDescriptor for an entry point the shader lacks or that is not a compute function, and with
// GPULimitExceeded for a @workgroup_size or a count of resources over the limits; the other failures are those of
// NewRenderPipeline.
//
// Flutter: no counterpart
func (d *Device) NewComputePipeline(desc ComputePipelineDesc) (*ComputePipeline, error) {
	d.assertThread("NewComputePipeline")
	if d.lostErr != nil {
		return nil, d.lostErr
	}
	m := desc.Module
	if m == nil || m.d != d {
		return nil, errors.New(TagGPUInvalidDescriptor, "a compute pipeline has a module made by this device", "field", "Module", "label", desc.Label)
	}
	m.usable("NewComputePipeline")
	e, ok := m.shader.Entry(desc.EntryPoint)
	if !ok || e.Stage != ShaderStageCompute {
		return nil, errors.New(TagGPUInvalidDescriptor, "the shader has a compute entry point of this name", "field", "EntryPoint", "value", desc.EntryPoint, "label", desc.Label)
	}
	w := e.WorkgroupSize
	for i, n := range w {
		if n < 1 {
			return nil, errors.New(TagGPUInvalidDescriptor, "a compute function has a @workgroup_size", "field", "EntryPoint", "value", n, "label", desc.Label)
		}
		if n > d.limits.MaxComputeWorkgroupSize[i] {
			return nil, errors.New(TagGPULimitExceeded, "the @workgroup_size is over Limits.MaxComputeWorkgroupSize", "field", "EntryPoint", "value", n, "max", d.limits.MaxComputeWorkgroupSize[i], "label", desc.Label)
		}
	}
	if n := w[0] * w[1] * w[2]; n > d.limits.MaxComputeInvocationsPerWorkgroup {
		return nil, errors.New(TagGPULimitExceeded, "the @workgroup_size is over Limits.MaxComputeInvocationsPerWorkgroup", "field", "EntryPoint", "value", n, "max", d.limits.MaxComputeInvocationsPerWorkgroup, "label", desc.Label)
	}
	groups, _ := mergeBindings(e)
	if len(groups) > d.limits.MaxBindGroups {
		return nil, errors.New(TagGPULimitExceeded, "the shader uses more bind groups than Limits.MaxBindGroups", "field", "EntryPoint", "value", len(groups), "max", d.limits.MaxBindGroups, "label", desc.Label)
	}
	exp, err := d.dev.NewComputePipeline(explicit.ComputePipelineDesc{Shader: m.exp, Entry: e.Name})
	if err != nil {
		if passesBackendError(err) {
			return nil, err
		}
		return nil, errors.NewErr(TagGPUShaderCompileFailed, err).SetMetadata("label", desc.Label, "shader", m.shader.Name)
	}
	p := &ComputePipeline{resource: d.track(kindComputePipeline, desc.Label, 1), exp: exp, entry: e}
	p.layouts = makeLayouts(p, groups)
	return p, nil
}

// Release frees the pipeline, when the work that uses it is done. It may be called twice.
//
// Flutter: no counterpart
func (p *ComputePipeline) Release() {
	p.d.assertThread("ComputePipeline.Release")
	if !p.untrack() {
		return
	}
	if p.d.lostErr == nil {
		p.exp.Destroy()
	}
}
