//go:build windows

package d3d12

import (
	"runtime"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// pipeline is a pipeline state object with the root signature it was made for.
type pipeline struct {
	d         *Device
	desc      explicit.PipelineDesc
	root      *rootSig
	pso       win32.Object
	destroyed bool
}

var _ explicit.Pipeline = (*pipeline)(nil)

// Desc implements explicit.Pipeline.
func (p *pipeline) Desc() explicit.PipelineDesc { return p.desc }

// Destroy implements explicit.Pipeline: the pipeline state is released when the GPU is done with the
// frames that may use it. The root signature belongs to the device.
func (p *pipeline) Destroy() {
	if p.destroyed {
		return
	}
	p.destroyed = true
	d := p.d
	delete(d.live, p)
	d.pipes.live--
	d.retire(func() { d.disown(p.pso) })
}

// pipelineStats counts the pipelines the device made, for the startup report and the tests.
type pipelineStats struct {
	live, created, fromLibrary int
	took                       time.Duration
}

// PipelineStats is what the device did to make its pipelines.
//
// Flutter: no counterpart
type PipelineStats struct {
	// Created is the number of pipeline states made by CreateGraphicsPipelineState, FromLibrary the
	// number loaded from the pipeline library, and Took the time of both, with the shaders.
	Created, FromLibrary int
	Took                 time.Duration
}

// PipelineStats says how the pipelines of the device were made.
//
// Flutter: no counterpart
func (d *Device) PipelineStats() PipelineStats {
	return PipelineStats{Created: d.pipes.created, FromLibrary: d.pipes.fromLibrary, Took: d.pipes.took}
}

// CompilerStats says how the shaders of the device were made: compiled, taken from the cache and
// refused (SPEC §9.7). It is the zero value before the first pipeline.
//
// Flutter: no counterpart
func (d *Device) CompilerStats() CompilerStats {
	if d.compiler == nil {
		return CompilerStats{}
	}
	return d.compiler.Stats()
}

// dxbcKey names the bytecode of one entry point of a shader.
type dxbcKey struct {
	shader *explicit.Shader
	entry  string
}

// shaderCompiler loads d3dcompiler_47.dll when the first pipeline needs it (SPEC §9.8 "Loading"). Its
// errors are the ones of win32, returned as they are.
func (d *Device) shaderCompiler() (*Compiler, error) {
	if d.compiler != nil {
		return d.compiler, nil
	}
	dll, err := LoadCompilerDLL()
	if err != nil {
		return nil, err
	}
	d.compiler = NewCompiler(dll, d.cache)
	return d.compiler, nil
}

// bytecode returns the DXBC of an entry point, through the cache of step 28.7 and remembered for
// the pipelines that share the entry point (the blends of one shader).
func (d *Device) bytecode(sh *explicit.Shader, entry, target string) ([]byte, error) {
	k := dxbcKey{sh, entry}
	if b, ok := d.dxbc[k]; ok {
		return b, nil
	}
	c, err := d.shaderCompiler()
	if err != nil {
		return nil, err
	}
	res, err := c.Compile(sh.HLSL, entry, target, CompileFlags(d.getenv))
	if err != nil {
		return nil, err
	}
	d.dxbc[k] = res.DXBC
	return res.DXBC, nil
}

// NewPipeline implements explicit.Device: the root signature from the bindings of the two entry
// points (kept by their shape), the bytecode through the shader cache, and the pipeline state object
// (SPEC §9.8 "Pipelines"). The renderer asks for a pipeline when a frame first needs it, which is when
// it is made; with a pipeline library it is loaded when the library has it. It fails with
// D3D12PipelineInvalid for a description that is not a pipeline, D3D12ShaderCompileFailed (a bug of
// the generator) and the errors of the compiler's library, and GPUOutOfMemory, GPUDeviceLost and
// D3D12CallFailed.
func (d *Device) NewPipeline(desc explicit.PipelineDesc) (explicit.Pipeline, error) {
	d.limits.CheckPipeline(desc)
	if desc.Shader == nil || desc.Shader.HLSL == "" {
		return nil, errors.New(tagD3D12PipelineInvalid, "a pipeline needs a shader with HLSL")
	}
	vs, ok1 := desc.Shader.Entry(desc.VertexEntry)
	fs, ok2 := desc.Shader.Entry(desc.FragmentEntry)
	if !ok1 || !ok2 {
		return nil, errors.New(tagD3D12PipelineInvalid, "the shader lacks the entry point",
			"shader", desc.Shader.Name, "vertex", desc.VertexEntry, "fragment", desc.FragmentEntry)
	}
	if dxgiFormat(desc.Target) == dxgiFormatUnknown {
		return nil, errors.New(tagD3D12PipelineInvalid, "a pipeline needs the format of its target", "shader", desc.Shader.Name)
	}
	plan, err := planRoot(vs, fs, d.limits)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	defer func() { d.pipes.took += time.Since(start) }()
	rs, err := d.rootSignature(plan)
	if err != nil {
		return nil, err
	}
	vsCode, err := d.bytecode(desc.Shader, desc.VertexEntry, TargetVertex)
	if err != nil {
		return nil, err
	}
	psCode, err := d.bytecode(desc.Shader, desc.FragmentEntry, TargetPixel)
	if err != nil {
		return nil, err
	}
	pso, err := d.makePipelineState(desc, rs, vsCode, psCode)
	if err != nil {
		return nil, err
	}
	p := &pipeline{d: d, desc: desc, root: rs, pso: pso}
	d.pipes.live++
	d.live[p] = func() { p.destroyed = true; d.pipes.live--; d.disown(p.pso) }
	return p, nil
}

// makePipelineState makes the pipeline state object: the shader, the vertex layout, the blend state of
// the one render target, the target format and no depth or stencil; from the pipeline library when it
// has the pipeline.
func (d *Device) makePipelineState(desc explicit.PipelineDesc, rs *rootSig, vsCode, psCode []byte) (win32.Object, error) {
	els := inputElements(desc.VertexBuffers)
	elDescs := make([]d3d12InputElementDesc, len(els))
	for i, e := range els {
		elDescs[i] = d3d12InputElementDesc{
			SemanticName: uintptr(unsafe.Pointer(&semanticLOC[0])), SemanticIndex: e.Location, Format: e.Format, InputSlot: e.Slot,
			AlignedByteOffset: e.Offset, InputSlotClass: e.Class, InstanceDataStepRate: e.StepRate,
		}
	}
	pd := d3d12GraphicsPipelineStateDesc{
		RootSignature: rs.obj.Ptr(),
		VS:            d3d12ShaderBytecode{Code: uintptr(unsafe.Pointer(&vsCode[0])), Length: uintptr(len(vsCode))},
		PS:            d3d12ShaderBytecode{Code: uintptr(unsafe.Pointer(&psCode[0])), Length: uintptr(len(psCode))},
		BlendState:    blendDesc(desc.Blend),
		SampleMask:    defaultSampleMask,
		RasterizerState: d3d12RasterizerDesc{
			FillMode: fillModeSolid, CullMode: cullModeNone, DepthClipEnable: 1,
		},
		DepthStencilState: d3d12DepthStencilDesc{
			DepthWriteMask: depthWriteMaskZero, DepthFunc: comparisonAlways,
			FrontFace: d3d12DepthStencilOpDesc{StencilFailOp: stencilOpKeep, StencilDepthFailOp: stencilOpKeep, StencilPassOp: stencilOpKeep, StencilFunc: comparisonAlways},
			BackFace:  d3d12DepthStencilOpDesc{StencilFailOp: stencilOpKeep, StencilDepthFailOp: stencilOpKeep, StencilPassOp: stencilOpKeep, StencilFunc: comparisonAlways},
		},
		PrimitiveTopologyType: topologyTypeTriangle,
		NumRenderTargets:      1,
		SampleDesc:            dxgiSampleDesc{Count: 1},
	}
	pd.RTVFormats[0] = dxgiFormat(desc.Target)
	if len(elDescs) > 0 {
		pd.InputLayout = d3d12InputLayoutDesc{Elements: uintptr(unsafe.Pointer(&elDescs[0])), NumElements: uint32(len(elDescs))}
	}
	defer runtime.KeepAlive(elDescs)
	defer runtime.KeepAlive(vsCode)
	defer runtime.KeepAlive(psCode)

	lib := d.pipelineLibrary()
	var name []uint16
	if !lib.IsNil() {
		var err error
		if name, err = win32.UTF16FromString(pipelineName(desc, rs.key, vsCode, psCode)); err != nil {
			return win32.Object{}, err
		}
		var out uintptr
		hr := win32.FromResult(lib.Call(libraryLoadGraphicsPipeline, uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&pd)),
			uintptr(unsafe.Pointer(&iidPipelineState)), uintptr(unsafe.Pointer(&out))))
		if hr.Succeeded() {
			d.pipes.fromLibrary++
			return d.own(win32.ObjectFromPtr(out)), nil
		}
		// Not in the library (E_INVALIDARG), or a blob it cannot use: the pipeline is made and stored.
	}
	var out uintptr
	if err := d.hr(d.dev, "ID3D12Device::CreateGraphicsPipelineState", deviceCreateGraphicsPipeline,
		uintptr(unsafe.Pointer(&pd)), uintptr(unsafe.Pointer(&iidPipelineState)), uintptr(unsafe.Pointer(&out))); err != nil {
		return win32.Object{}, err
	}
	d.pipes.created++
	pso := d.own(win32.ObjectFromPtr(out))
	if !lib.IsNil() {
		if hr := win32.FromResult(lib.Call(libraryStorePipeline, uintptr(unsafe.Pointer(&name[0])), pso.Ptr())); hr.Succeeded() {
			d.library.dirty = true
		}
	}
	return pso, nil
}

// pipelineLibrary is the ID3D12PipelineLibrary of the device, where the device has one (SPEC §9.8
// "Pipelines"): it keeps the pipeline states between runs in a blob of the shader cache.
type pipelineLibrary struct {
	obj    win32.Object
	tried  bool
	key    string
	loaded int  // the size of the blob that was read from the cache
	dirty  bool // a pipeline was stored since the last save
}

// IsNil says whether the device has no library.
func (l *pipelineLibrary) IsNil() bool { return l.obj.IsNil() }

func (l *pipelineLibrary) destroy(d *Device) {
	d.disown(l.obj)
	l.obj = win32.Object{}
}

// pipelineLibrary returns the library, making it at the first pipeline: ID3D12Device1::CreatePipelineLibrary
// with the blob that the cache has for this adapter and driver. A blob that the driver refuses
// (D3D12_ERROR_DRIVER_VERSION_MISMATCH, D3D12_ERROR_ADAPTER_NOT_FOUND, E_INVALIDARG) is dropped and
// the library starts empty. Where the call is not implemented (Wine's vkd3d at this version) or the device
// has no ID3D12Device1, there is no library and the pipelines are made again at each start.
func (d *Device) pipelineLibrary() win32.Object {
	l := &d.library
	if l.tried {
		return l.obj
	}
	l.tried = true
	if d.noLib {
		return l.obj
	}
	create := d.makeLibrary
	if create == nil {
		dev1, err := d.dev.QueryInterface(&iidDevice1)
		if err != nil {
			d.log.Debug("the device has no ID3D12Device1; pipelines are made again at each start", "err", err)
			return l.obj
		}
		d.own(dev1)
		defer d.disown(dev1)
		create = func(blob []byte) (win32.Object, win32.HRESULT) {
			var out, p uintptr
			if len(blob) > 0 {
				p = uintptr(unsafe.Pointer(&blob[0]))
			}
			hr := win32.FromResult(dev1.Call(device1CreatePipelineLibrary, p, uintptr(len(blob)),
				uintptr(unsafe.Pointer(&iidPipelineLibrary)), uintptr(unsafe.Pointer(&out))))
			runtime.KeepAlive(blob)
			return win32.ObjectFromPtr(out), hr
		}
	}
	l.key = LibraryKey(d.info.VendorID, d.info.DeviceID, d.info.Revision, d.driverVersion())
	var blob []byte
	if d.cache != nil {
		if b, ok := d.cache.Load(l.key); ok {
			blob = b
		}
	}
	try := func(blob []byte) win32.HRESULT {
		obj, hr := create(blob)
		if hr.Succeeded() {
			l.obj = d.own(obj)
		}
		return hr
	}
	hr := try(blob)
	if hr.Failed() && len(blob) > 0 {
		d.log.Debug("the pipeline library of the cache was refused and is dropped", "hresult", hr.String(), "bytes", len(blob), "why", libraryRefusal(hr.Code()))
		hr = try(nil)
	} else if hr.Succeeded() {
		l.loaded = len(blob)
	}
	if hr.Failed() {
		d.log.Debug("there is no pipeline library on this device; pipelines are made again at each start", "hresult", hr.String())
	}
	return l.obj
}

// driverVersion is the version of the adapter's user-mode driver (IDXGIAdapter::CheckInterfaceSupport),
// part of the key of the pipeline library; 0 when the adapter does not say.
func (d *Device) driverVersion() uint64 {
	if d.adapter.IsNil() {
		return 0
	}
	var v uint64
	if hr := win32.FromResult(d.adapter.Call(adapterCheckInterfaceSupport, uintptr(unsafe.Pointer(&iidDXGIDevice)), uintptr(unsafe.Pointer(&v)))); hr.Failed() {
		return 0
	}
	return v
}

// HasPipelineLibrary says whether the device has an ID3D12PipelineLibrary: it is made at the first
// pipeline, and is missing where CreatePipelineLibrary is not implemented (Wine's vkd3d) and on a device
// without ID3D12Device1. The startup report of the engine says it (SPEC §9.7).
//
// Flutter: no counterpart
func (d *Device) HasPipelineLibrary() bool { return !d.library.IsNil() }

// SavePipelineCache writes the pipeline library into the shader cache when it grew since the last save
// (SPEC §9.7). The renderer calls it after the first frame, and Close again. A device without a library
// or a cache does nothing.
//
// Flutter: no counterpart
func (d *Device) SavePipelineCache() {
	l := &d.library
	if l.obj.IsNil() || d.cache == nil || !l.dirty {
		return
	}
	n := int(l.obj.Call(librarySerializedSize))
	if n <= 0 {
		return
	}
	buf := make([]byte, n)
	if hr := win32.FromResult(l.obj.Call(librarySerialize, uintptr(unsafe.Pointer(&buf[0])), uintptr(n))); hr.Failed() {
		d.log.Debug("serializing the pipeline library failed", "hresult", hr.String())
		return
	}
	d.cache.Store(l.key, buf)
	l.dirty = false
}
