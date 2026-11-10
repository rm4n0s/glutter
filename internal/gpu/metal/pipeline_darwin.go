//go:build darwin

package metal

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// languageVersion24 is MTLLanguageVersion2_4: (2 << 16) | 4.
const languageVersion24 uint64 = 2<<16 | 4

// The options of a pipeline state (MTLPipelineOption): fail instead of compiling when the binary archive
// does not hold the pipeline.
const pipelineFailOnArchiveMiss uint64 = 1 << 2

// library is the MTLLibrary that Metal compiles from the MSL of a shader, with the functions that were taken
// from it.
type library struct {
	d      *Device
	obj    objc.ID
	shader *explicit.Shader
	funcs  map[string]objc.ID
}

// library returns the library of a shader, compiling its MSL at the first pipeline that needs it
// (newLibraryWithSource:options:error: needs no Xcode on the user's Mac, SPEC §29.5). It fails with
// MetalShaderCompileFailed, with the compiler's output.
func (d *Device) library(sh *explicit.Shader) (*library, error) {
	if l, ok := d.libs[sh]; ok {
		return l, nil
	}
	if sh.MSL == "" {
		return nil, errors.New(TagMetalShaderCompileFailed, "the shader has no MSL: run glutter-gen shaders", "shader", sh.Name)
	}
	var obj objc.ID
	var failure string
	objcx.WithPool(func() {
		opts := newObject("MTLCompileOptions")
		defer release(opts)
		call(opts, "setLanguageVersion:", languageVersion24)
		// Strict IEEE arithmetic keeps the results of the shaders what the software renderer computes.
		call(opts, "setFastMathEnabled:", false)
		var e objc.ID
		obj = call(d.dev, "newLibraryWithSource:options:error:", objcx.NewString(sh.MSL), opts, ptr(&e))
		if obj == 0 {
			failure = description(e)
		}
	})
	if obj == 0 {
		return nil, errors.New(TagMetalShaderCompileFailed, "the Metal compiler refused the shader", "shader", sh.Name, "output", failure)
	}
	d.stats.Libraries++
	l := &library{d: d, obj: obj, shader: sh, funcs: map[string]objc.ID{}}
	d.libs[sh] = l
	return l, nil
}

// function returns the function of the entry point, which the library keeps.
func (l *library) function(name string) (objc.ID, error) {
	if f, ok := l.funcs[name]; ok {
		return f, nil
	}
	f := call(l.obj, "newFunctionWithName:", objcx.NewString(name))
	if f == 0 {
		return 0, errors.New(TagMetalPipelineFailed, "the compiled shader has no function with the name of the entry point", "shader", l.shader.Name, "entry", name)
	}
	l.funcs[name] = f
	return f, nil
}

func (l *library) release() {
	for _, f := range l.funcs {
		release(f)
	}
	release(l.obj)
}

// archive is the MTLBinaryArchive that keeps the pipelines in the shader cache (SPEC §9.7, §29.5): a file in
// the cache directory for each device, loaded at start and saved at Close when a pipeline was added. A
// pipeline that the archive has is not compiled, which makes the second start of an app fast. A nil archive
// does nothing, as when there is no cache.
type archive struct {
	d      *Device
	obj    objc.ID
	path   string
	loaded bool // the archive was read from a file, so it may hold the pipeline
	dirty  bool
}

// openArchive opens the archive file of the device, or makes an empty archive; nil without a cache directory
// or when Metal has no archives (before macOS 11).
func openArchive(d *Device) *archive {
	if d.opts.ShaderCacheDir == "" || objc.GetClass("MTLBinaryArchiveDescriptor") == 0 {
		return nil
	}
	key := strconv.FormatUint(send[uint64](d.dev, "registryID"), 16)
	a := &archive{d: d, path: filepath.Join(d.opts.ShaderCacheDir, "metal-"+key+".archive")}
	objcx.WithPool(func() {
		desc := newObject("MTLBinaryArchiveDescriptor")
		defer release(desc)
		var e objc.ID
		if _, err := os.Stat(a.path); err == nil {
			call(desc, "setUrl:", objcx.NewFileURL(a.path))
			a.obj = call(d.dev, "newBinaryArchiveWithDescriptor:error:", desc, ptr(&e))
			if a.obj != 0 {
				a.loaded = true
				return
			}
			// An archive of another OS build or a damaged file is not an error: it is made again.
			d.log.Info("the pipeline archive was not read; it is made again", "path", a.path, "output", description(e))
			call(desc, "setUrl:", objc.ID(0))
		}
		a.obj = call(d.dev, "newBinaryArchiveWithDescriptor:error:", desc, ptr(&e))
		if a.obj == 0 {
			d.log.Warn("Metal made no pipeline archive", "output", description(e))
		}
	})
	if a.obj == 0 {
		return nil
	}
	return a
}

// list is the array of archives that a pipeline descriptor looks in.
func (a *archive) list() objc.ID {
	if a == nil || a.obj == 0 {
		return 0
	}
	return objcx.NewArray(a.obj)
}

// save writes the archive when a pipeline was added since it was read.
func (a *archive) save() {
	if a == nil || a.obj == 0 || !a.dirty {
		return
	}
	if err := os.MkdirAll(filepath.Dir(a.path), 0o755); err != nil {
		a.d.log.Warn("the pipeline archive directory cannot be made", "path", a.path, "err", err)
		return
	}
	tmp := a.path + ".tmp"
	var e objc.ID
	objcx.WithPool(func() {
		if !sendBool(a.obj, "serializeToURL:error:", objcx.NewFileURL(tmp), ptr(&e)) {
			// An archive that cannot be written out would fail at every start: it is dropped, and the next start makes it again.
			a.d.log.Warn("the pipeline archive was not saved, and is removed so that the next start makes it again", "path", a.path, "output", description(e))
			_ = os.Remove(a.path)
			tmp = ""
		}
	})
	if tmp == "" {
		return
	}
	if err := os.Rename(tmp, a.path); err != nil {
		a.d.log.Warn("the pipeline archive was not saved", "path", a.path, "err", err)
		_ = os.Remove(tmp)
		return
	}
	a.dirty = false
}

func (a *archive) close() {
	if a != nil {
		release(a.obj)
		a.obj = 0
	}
}

// pipeline is a render pipeline: the state object, the depth-stencil state, the fixed-function state that an
// encoder sets with them, and the bindings of the two entry points.
type pipeline struct {
	d     *Device
	desc  explicit.PipelineDesc
	state objc.ID
	depth objc.ID // 0 without a depth-stencil format
	table *bindingTable

	prim  uint64
	cull  uint64
	front uint64
	bias  float32
	slope float32
	clamp float32
}

var _ explicit.Pipeline = (*pipeline)(nil)

func (p *pipeline) Desc() explicit.PipelineDesc { return p.desc }

// Destroy implements explicit.Pipeline: a command buffer that uses the pipeline keeps it.
func (p *pipeline) Destroy() {
	release(p.state)
	release(p.depth)
	p.state, p.depth = 0, 0
}

// NewPipeline implements explicit.Device. It fails with MetalShaderCompileFailed when the compiler refuses the
// shader and MetalPipelineFailed when Metal cannot make the state.
func (d *Device) NewPipeline(desc explicit.PipelineDesc) (explicit.Pipeline, error) {
	d.limits.CheckPipeline(desc)
	if err := d.checkLost(); err != nil {
		return nil, err
	}
	table, err := newBindingTable(desc.Shader, entryStage{desc.VertexEntry, stageVertex}, entryStage{desc.FragmentEntry, stageFragment})
	if err != nil {
		return nil, err
	}
	lib, err := d.library(desc.Shader)
	if err != nil {
		return nil, err
	}
	vs, err := lib.function(desc.VertexEntry)
	if err != nil {
		return nil, err
	}
	var fs objc.ID
	if desc.FragmentEntry != "" {
		if fs, err = lib.function(desc.FragmentEntry); err != nil {
			return nil, err
		}
	}
	p := &pipeline{d: d, desc: desc, table: table, prim: primitiveType(desc.Topology), cull: cullMode(desc.CullMode), front: winding(desc.FrontFace)}
	ds := desc.DepthStencil
	p.bias, p.slope, p.clamp = float32(ds.DepthBias), ds.DepthBiasSlopeScale, ds.DepthBiasClamp
	var failure string
	objcx.WithPool(func() {
		rd := newObject("MTLRenderPipelineDescriptor")
		defer release(rd)
		call(rd, "setVertexFunction:", vs)
		if fs != 0 {
			call(rd, "setFragmentFunction:", fs)
		}
		if len(desc.VertexBuffers) > 0 {
			call(rd, "setVertexDescriptor:", vertexDescriptor(desc.VertexBuffers))
		}
		for i, t := range desc.Targets() {
			if t.Format == explicit.TextureFormatUnspecified {
				continue
			}
			att := call(call(rd, "colorAttachments"), "objectAtIndexedSubscript:", uint64(i))
			call(att, "setPixelFormat:", pixelFormat(t.Format))
			bl := blendFor(t.Blend)
			call(att, "setBlendingEnabled:", bl.enabled)
			call(att, "setSourceRGBBlendFactor:", bl.srcColor)
			call(att, "setDestinationRGBBlendFactor:", bl.dstColor)
			call(att, "setSourceAlphaBlendFactor:", bl.srcAlpha)
			call(att, "setDestinationAlphaBlendFactor:", bl.dstAlpha)
			call(att, "setRgbBlendOperation:", blendOpAdd)
			call(att, "setAlphaBlendOperation:", blendOpAdd)
			call(att, "setWriteMask:", writeMask(t.WriteMask))
		}
		if ds.Format != explicit.TextureFormatUnspecified {
			call(rd, "setDepthAttachmentPixelFormat:", pixelFormat(ds.Format))
			if ds.Format.HasStencil() {
				call(rd, "setStencilAttachmentPixelFormat:", pixelFormat(ds.Format))
			}
		}
		call(rd, "setRasterSampleCount:", uint64(desc.Samples()))
		call(rd, "setInputPrimitiveTopology:", topologyClass(desc.Topology))
		p.state, failure = d.makeRenderState(rd, desc)
		if p.state != 0 && ds.Format != explicit.TextureFormatUnspecified {
			p.depth = d.depthStencilState(ds)
		}
	})
	if p.state == 0 {
		return nil, errors.New(TagMetalPipelineFailed, "Metal made no render pipeline", "shader", desc.Shader.Name,
			"vertex", desc.VertexEntry, "fragment", desc.FragmentEntry, "output", failure)
	}
	return p, nil
}

// makeRenderState makes the pipeline state of a descriptor, from the binary archive when it has the pipeline,
// compiling it otherwise (and adding it to the archive). It returns the output of Metal for a failure.
func (d *Device) makeRenderState(rd objc.ID, desc explicit.PipelineDesc) (objc.ID, string) {
	var e objc.ID
	// A pipeline without a fragment function (a shadow pass) cannot be serialized: the archiver refuses it ("expecting
	// 'fragment' stage in pipeline"), and one such pipeline would lose the whole archive at save. It is compiled at each start,
	// which is cheap.
	archived := d.archive != nil && desc.FragmentEntry != ""
	if a := d.archive; archived {
		call(rd, "setBinaryArchives:", a.list())
		if a.loaded {
			if st := call(d.dev, "newRenderPipelineStateWithDescriptor:options:reflection:error:", rd, pipelineFailOnArchiveMiss, uintptr(0), ptr(&e)); st != 0 {
				d.stats.PipelinesFromCache++
				return st, ""
			}
		}
	}
	st := call(d.dev, "newRenderPipelineStateWithDescriptor:error:", rd, ptr(&e))
	if st == 0 {
		return 0, description(e)
	}
	d.stats.PipelinesCompiled++
	if a := d.archive; archived {
		var ae objc.ID
		if sendBool(a.obj, "addRenderPipelineFunctionsWithDescriptor:error:", rd, ptr(&ae)) {
			a.dirty = true
		} else {
			d.log.Debug("the pipeline was not added to the archive", "shader", desc.Shader.Name, "output", description(ae))
		}
	}
	return st, ""
}

// vertexDescriptor makes the MTLVertexDescriptor of the vertex buffers of a pipeline: the buffer of slot s is
// at the index vertexBufferIndex(s), and the attribute of location n is attribute(n).
func vertexDescriptor(vbs []explicit.VertexBufferLayout) objc.ID {
	vd := call(class("MTLVertexDescriptor"), "vertexDescriptor")
	attrs, layouts := call(vd, "attributes"), call(vd, "layouts")
	for slot, vb := range vbs {
		l := call(layouts, "objectAtIndexedSubscript:", vertexBufferIndex(slot))
		call(l, "setStride:", uint64(vb.Stride))
		step := stepPerVertex
		if vb.PerInstance {
			step = stepPerInstance
		}
		call(l, "setStepFunction:", step)
		call(l, "setStepRate:", uint64(1))
		for _, a := range vb.Attributes {
			at := call(attrs, "objectAtIndexedSubscript:", uint64(a.Location))
			call(at, "setFormat:", vertexFormat(a.Format))
			call(at, "setOffset:", uint64(a.Offset))
			call(at, "setBufferIndex:", vertexBufferIndex(slot))
		}
	}
	return vd
}

// depthStencilState makes the MTLDepthStencilState of a pipeline: with a depth-stencil format the depth test
// always runs (the comparison "always" without the write is the same as no test, as in WebGPU), and the
// stencil test runs when the format has a stencil.
func (d *Device) depthStencilState(ds explicit.DepthStencilState) objc.ID {
	dd := newObject("MTLDepthStencilDescriptor")
	defer release(dd)
	call(dd, "setDepthCompareFunction:", compareFunction(ds.DepthCompare))
	call(dd, "setDepthWriteEnabled:", ds.DepthWrite)
	if ds.Format.HasStencil() {
		face := func(setter string, f explicit.StencilFaceState) {
			sd := newObject("MTLStencilDescriptor")
			defer release(sd)
			call(sd, "setStencilCompareFunction:", compareFunction(f.Compare))
			call(sd, "setStencilFailureOperation:", stencilOperation(f.FailOp))
			call(sd, "setDepthFailureOperation:", stencilOperation(f.DepthFailOp))
			call(sd, "setDepthStencilPassOperation:", stencilOperation(f.PassOp))
			call(sd, "setReadMask:", stencilMask(ds.StencilReadMask))
			call(sd, "setWriteMask:", stencilMask(ds.StencilWriteMask))
			call(dd, setter, sd)
		}
		face("setFrontFaceStencil:", ds.StencilFront)
		face("setBackFaceStencil:", ds.StencilBack)
	}
	return call(d.dev, "newDepthStencilStateWithDescriptor:", dd)
}

// computePipeline is a compute pipeline: the state object, the workgroup size of the entry point and its
// bindings.
type computePipeline struct {
	d     *Device
	desc  explicit.ComputePipelineDesc
	state objc.ID
	table *bindingTable
	group [3]int
}

var _ explicit.ComputePipeline = (*computePipeline)(nil)

func (p *computePipeline) Desc() explicit.ComputePipelineDesc { return p.desc }

// Destroy implements explicit.ComputePipeline.
func (p *computePipeline) Destroy() { release(p.state); p.state = 0 }

// NewComputePipeline implements explicit.Device.
func (d *Device) NewComputePipeline(desc explicit.ComputePipelineDesc) (explicit.ComputePipeline, error) {
	d.limits.CheckComputePipeline(desc)
	if err := d.checkLost(); err != nil {
		return nil, err
	}
	entry, ok := desc.Shader.Entry(desc.Entry)
	if !ok || entry.Stage != explicit.ShaderStageCompute {
		return nil, errors.New(TagMetalPipelineFailed, "the shader has no compute entry point with that name", "shader", desc.Shader.Name, "entry", desc.Entry)
	}
	table, err := newBindingTable(desc.Shader, entryStage{desc.Entry, stageCompute})
	if err != nil {
		return nil, err
	}
	lib, err := d.library(desc.Shader)
	if err != nil {
		return nil, err
	}
	fn, err := lib.function(desc.Entry)
	if err != nil {
		return nil, err
	}
	p := &computePipeline{d: d, desc: desc, table: table, group: entry.WorkgroupSize}
	var failure string
	objcx.WithPool(func() {
		cd := newObject("MTLComputePipelineDescriptor")
		defer release(cd)
		call(cd, "setComputeFunction:", fn)
		p.state, failure = d.makeComputeState(cd, desc)
	})
	if p.state == 0 {
		return nil, errors.New(TagMetalPipelineFailed, "Metal made no compute pipeline", "shader", desc.Shader.Name, "entry", desc.Entry, "output", failure)
	}
	return p, nil
}

// makeComputeState is makeRenderState for a compute pipeline.
func (d *Device) makeComputeState(cd objc.ID, desc explicit.ComputePipelineDesc) (objc.ID, string) {
	var e objc.ID
	if a := d.archive; a != nil {
		call(cd, "setBinaryArchives:", a.list())
		if a.loaded {
			if st := call(d.dev, "newComputePipelineStateWithDescriptor:options:reflection:error:", cd, pipelineFailOnArchiveMiss, uintptr(0), ptr(&e)); st != 0 {
				d.stats.PipelinesFromCache++
				return st, ""
			}
		}
	}
	st := call(d.dev, "newComputePipelineStateWithDescriptor:options:reflection:error:", cd, uint64(0), uintptr(0), ptr(&e))
	if st == 0 {
		return 0, description(e)
	}
	d.stats.PipelinesCompiled++
	if a := d.archive; a != nil {
		var ae objc.ID
		if sendBool(a.obj, "addComputePipelineFunctionsWithDescriptor:error:", cd, ptr(&ae)) {
			a.dirty = true
		} else {
			d.log.Debug("the pipeline was not added to the archive", "shader", desc.Shader.Name, "output", description(ae))
		}
	}
	return st, ""
}

// NewQuerySet implements explicit.Device: Metal's counter sample buffers are not used, so there are no
// timestamp queries (Features.TimestampQuery is false). It fails with GPUQueryUnsupported.
func (d *Device) NewQuerySet(count int) (explicit.QuerySet, error) {
	return nil, errors.New(explicit.TagGPUQueryUnsupported, "the Metal backend has no timestamp queries", "count", count)
}

// CompileShader compiles the MSL of a shader with Metal's compiler and finds the functions of its entry points,
// as the first pipeline that uses the shader does, and returns the error that pipeline would get
// (MetalShaderCompileFailed, MetalPipelineFailed). The tests of the packages that have generated shaders (the 3D view,
// the examples) use it to check that every MSL is accepted (SPEC §29.5).
//
// Flutter: no counterpart
func (d *Device) CompileShader(name, msl string, entries ...string) error {
	lib, err := d.library(&explicit.Shader{Name: name, MSL: msl})
	if err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := lib.function(e); err != nil {
			return err
		}
	}
	return nil
}
