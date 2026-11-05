package d3d12

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// The mappings from the descriptions of the explicit renderer to the values of Direct3D 12 and DXGI.

// dxgiFormat maps a texture format; an unspecified format is DXGI_FORMAT_UNKNOWN.
func dxgiFormat(f explicit.TextureFormat) uint32 {
	switch f {
	case explicit.TextureFormatRGBA8Unorm:
		return dxgiFormatRGBA8
	case explicit.TextureFormatBGRA8Unorm:
		return dxgiFormatBGRA8
	case explicit.TextureFormatR8Unorm:
		return dxgiFormatR8
	case explicit.TextureFormatRGBA16Float:
		return dxgiFormatRGBA16F
	case explicit.TextureFormatRGBA32Float:
		return dxgiFormatRGBA32
	case explicit.TextureFormatR32Float:
		return dxgiFormatR32
	case explicit.TextureFormatR32Uint:
		return dxgiFormatR32U
	case explicit.TextureFormatRG32Uint:
		return dxgiFormatRG32U
	case explicit.TextureFormatDepth32Float:
		return dxgiFormatD32F
	case explicit.TextureFormatDepth24PlusStencil8:
		return dxgiFormatD24S8
	}
	return dxgiFormatUnknown
}

// dxgiCopyFormat is the format of the footprint of a copy between a texture and a buffer: the format
// of the texture, but for the depth plane of Depth32Float, which a copy reads as R32_FLOAT.
func dxgiCopyFormat(f explicit.TextureFormat) uint32 {
	if f == explicit.TextureFormatDepth32Float {
		return dxgiFormatR32
	}
	return dxgiFormat(f)
}

// d3dFilter maps the three filters of a sampler to D3D12_FILTER: the minification filter, the
// magnification filter and the mip filter, each linear or not (FilterUnspecified is nearest).
func d3dFilter(min, mag, mip explicit.FilterMode) uint32 {
	var f uint32
	if min == explicit.FilterLinear {
		f |= filterMinLinear
	}
	if mag == explicit.FilterLinear {
		f |= filterMagLinear
	}
	if mip == explicit.FilterLinear {
		f |= filterMipLinear
	}
	return f
}

// d3dAddress maps an address mode; AddressUnspecified is clamp.
func d3dAddress(m explicit.AddressMode) uint32 {
	switch m {
	case explicit.AddressRepeat:
		return addressModeWrap
	case explicit.AddressMirror:
		return addressModeMirror
	}
	return addressModeClamp
}

// samplerDesc maps a sampler description (SPEC §9.3): the mip filter picks between levels, and
// FilterUnspecified uses level 0 only, with a maximum LOD of 0.
func samplerDesc(d explicit.SamplerDesc) d3d12SamplerDesc {
	maxLOD := float32(0)
	if d.MipmapFilter != explicit.FilterUnspecified {
		maxLOD = float32Max
	}
	return d3d12SamplerDesc{
		Filter:   d3dFilter(d.MinFilter, d.MagFilter, d.MipmapFilter),
		AddressU: d3dAddress(d.AddressU), AddressV: d3dAddress(d.AddressV), AddressW: d3dAddress(d.AddressW),
		MaxAnisotropy: 1, ComparisonFunc: comparisonNever, MaxLOD: maxLOD,
	}
}

// d3dBlend maps a blend factor. As in the other backends, the unspecified source factor is one and the
// unspecified destination factor is zero (explicit.BlendState).
func d3dBlend(f explicit.BlendFactor, destination bool) uint32 {
	switch f {
	case explicit.BlendZero:
		return blendZero
	case explicit.BlendOne:
		return blendOne
	case explicit.BlendSrcAlpha:
		return blendSrcAlpha
	case explicit.BlendOneMinusSrcAlpha:
		return blendInvSrcAlpha
	case explicit.BlendDstAlpha:
		return blendDestAlpha
	case explicit.BlendOneMinusDstAlpha:
		return blendInvDestAlpha
	}
	if destination {
		return blendZero
	}
	return blendOne
}

// blendDesc maps the blend state of a pipeline to the blend state of its one render target: with
// blending off the source replaces the destination; with it on, the factors are used for the color and
// the alpha alike, with the add operation (SPEC §9.3).
func blendDesc(b explicit.BlendState) d3d12BlendDesc {
	return blendDescOf([]explicit.ColorTargetState{{Blend: b}})
}

// blendDescOf maps the blend states and write masks of the color targets of a pipeline; with more
// than one target, each has its own state (IndependentBlendEnable).
func blendDescOf(targets []explicit.ColorTargetState) d3d12BlendDesc {
	var out d3d12BlendDesc
	if len(targets) > 1 {
		out.IndependentBlendEnable = 1
	}
	for i, t := range targets {
		rt := d3d12RenderTargetBlendDesc{
			SrcBlend: blendOne, DestBlend: blendZero, BlendOp: blendOpAdd,
			SrcBlendAlpha: blendOne, DestBlendAlpha: blendZero, BlendOpAlpha: blendOpAdd,
			LogicOp: logicOpNoop, RenderTargetWriteMask: uint8(t.WriteMask.Bits()),
		}
		if t.Blend.Enabled {
			rt.BlendEnable = 1
			rt.SrcBlend, rt.SrcBlendAlpha = d3dBlend(t.Blend.SrcFactor, false), d3dBlend(t.Blend.SrcFactor, false)
			rt.DestBlend, rt.DestBlendAlpha = d3dBlend(t.Blend.DstFactor, true), d3dBlend(t.Blend.DstFactor, true)
		}
		out.RenderTarget[i] = rt
	}
	return out
}

// primitiveTopologyType maps a topology to the type that a pipeline state is made for.
func primitiveTopologyType(t explicit.PrimitiveTopology) uint32 {
	switch t {
	case explicit.TopologyLineList, explicit.TopologyLineStrip:
		return topologyTypeLine
	case explicit.TopologyPointList:
		return topologyTypePoint
	}
	return topologyTypeTriangle
}

// primitiveTopology maps a topology to the D3D_PRIMITIVE_TOPOLOGY the input assembler is set to.
func primitiveTopology(t explicit.PrimitiveTopology) uint32 {
	switch t {
	case explicit.TopologyTriangleStrip:
		return topologyTriangleStrip
	case explicit.TopologyLineList:
		return topologyLineList
	case explicit.TopologyLineStrip:
		return topologyLineStrip
	case explicit.TopologyPointList:
		return topologyPointList
	}
	return topologyTriangleList
}

// d3dCull maps a cull mode.
func d3dCull(c explicit.CullMode) uint32 {
	switch c {
	case explicit.CullFront:
		return cullModeFront
	case explicit.CullBack:
		return cullModeBack
	}
	return cullModeNone
}

// d3dCompare maps a comparison; the unspecified one is always.
func d3dCompare(c explicit.CompareFunc) uint32 {
	switch c {
	case explicit.CompareNever:
		return comparisonNever
	case explicit.CompareLess:
		return comparisonLess
	case explicit.CompareEqual:
		return comparisonEqual
	case explicit.CompareLessEqual:
		return comparisonLessEqual
	case explicit.CompareGreater:
		return comparisonGreater
	case explicit.CompareNotEqual:
		return comparisonNotEqual
	case explicit.CompareGreaterEqual:
		return comparisonGreaterEqual
	}
	return comparisonAlways
}

// d3dStencilOp maps a stencil operation.
func d3dStencilOp(o explicit.StencilOp) uint32 {
	switch o {
	case explicit.StencilZero:
		return stencilOpZero
	case explicit.StencilReplace:
		return stencilOpReplace
	case explicit.StencilInvert:
		return stencilOpInvert
	case explicit.StencilIncrementClamp:
		return stencilOpIncrSat
	case explicit.StencilDecrementClamp:
		return stencilOpDecrSat
	case explicit.StencilIncrementWrap:
		return stencilOpIncr
	case explicit.StencilDecrementWrap:
		return stencilOpDecr
	}
	return stencilOpKeep
}

// stencilFace maps the stencil test of one kind of faces.
func stencilFace(f explicit.StencilFaceState) d3d12DepthStencilOpDesc {
	return d3d12DepthStencilOpDesc{
		StencilFailOp: d3dStencilOp(f.FailOp), StencilDepthFailOp: d3dStencilOp(f.DepthFailOp),
		StencilPassOp: d3dStencilOp(f.PassOp), StencilFunc: d3dCompare(f.Compare),
	}
}

// stencilByteMask maps a stencil mask of the description: 0 means every bit, and Direct3D 12 has 8.
func stencilByteMask(m uint32) uint8 {
	if m == 0 {
		return 0xFF
	}
	return uint8(m)
}

// depthStencilDesc maps the depth and stencil state of a pipeline. Without a depth-stencil format the
// tests are off; with one, the depth test always runs (the comparison "always" without the write is the
// same as no test, as in WebGPU) and the stencil test runs when the format has a stencil.
func depthStencilDesc(ds explicit.DepthStencilState) d3d12DepthStencilDesc {
	out := d3d12DepthStencilDesc{
		DepthFunc: comparisonAlways,
		FrontFace: stencilFace(explicit.StencilFaceState{}), BackFace: stencilFace(explicit.StencilFaceState{}),
		StencilReadMask: 0xFF, StencilWriteMask: 0xFF,
	}
	if ds.Format == explicit.TextureFormatUnspecified {
		return out
	}
	out.DepthEnable = 1
	if ds.DepthWrite {
		out.DepthWriteMask = depthWriteMaskAll
	}
	out.DepthFunc = d3dCompare(ds.DepthCompare)
	if ds.Format.HasStencil() {
		out.StencilEnable = 1
		out.StencilReadMask, out.StencilWriteMask = stencilByteMask(ds.StencilReadMask), stencilByteMask(ds.StencilWriteMask)
		out.FrontFace, out.BackFace = stencilFace(ds.StencilFront), stencilFace(ds.StencilBack)
	}
	return out
}

// rasterizerDesc maps the cull mode, the winding and the depth bias of a pipeline. The clip space is
// WebGPU's, which is Direct3D's: y up, depth from 0 to 1, so a counter-clockwise triangle on the screen is
// a front face when FrontCounterClockwise is set (SPEC §27.1).
func rasterizerDesc(desc explicit.PipelineDesc) d3d12RasterizerDesc {
	r := d3d12RasterizerDesc{
		FillMode: fillModeSolid, CullMode: d3dCull(desc.CullMode), DepthClipEnable: 1,
		DepthBias: desc.DepthStencil.DepthBias, DepthBiasClamp: desc.DepthStencil.DepthBiasClamp,
		SlopeScaledDepthBias: desc.DepthStencil.DepthBiasSlopeScale,
	}
	if desc.FrontFace == explicit.FrontFaceCCW {
		r.FrontCounterClockwise = 1
	}
	return r
}

// dxgiVertexFormat maps the type of a vertex attribute; the unspecified format is four floats, as in the
// Vulkan backend.
func dxgiVertexFormat(f explicit.VertexFormat) uint32 {
	switch f {
	case explicit.VertexFloat32:
		return dxgiFormatR32
	case explicit.VertexFloat32x2:
		return dxgiFormatRG32
	case explicit.VertexFloat32x3:
		return dxgiFormatRGB32
	case explicit.VertexUnorm8x4:
		return dxgiFormatRGBA8
	case explicit.VertexUint32:
		return dxgiFormatR32U
	case explicit.VertexUint32x2:
		return dxgiFormatRG32U
	case explicit.VertexUint32x3:
		return dxgiFormatRGB32U
	case explicit.VertexUint32x4:
		return dxgiFormatRGBA32U
	case explicit.VertexSint32:
		return dxgiFormatR32S
	case explicit.VertexSint32x2:
		return dxgiFormatRG32S
	case explicit.VertexSint32x3:
		return dxgiFormatRGB32S
	case explicit.VertexSint32x4:
		return dxgiFormatRGBA32S
	case explicit.VertexUint8x2:
		return dxgiFormatRG8U
	case explicit.VertexUint8x4:
		return dxgiFormatRGBA8U
	case explicit.VertexSint8x2:
		return dxgiFormatRG8S
	case explicit.VertexSint8x4:
		return dxgiFormatRGBA8S
	case explicit.VertexUnorm8x2:
		return dxgiFormatRG8N
	case explicit.VertexSnorm8x2:
		return dxgiFormatRG8SN
	case explicit.VertexSnorm8x4:
		return dxgiFormatRGBA8SN
	case explicit.VertexUint16x2:
		return dxgiFormatRG16U
	case explicit.VertexUint16x4:
		return dxgiFormatRGBA16U
	case explicit.VertexSint16x2:
		return dxgiFormatRG16S
	case explicit.VertexSint16x4:
		return dxgiFormatRGBA16S
	case explicit.VertexUnorm16x2:
		return dxgiFormatRG16N
	case explicit.VertexUnorm16x4:
		return dxgiFormatRGBA16N
	case explicit.VertexSnorm16x2:
		return dxgiFormatRG16SN
	case explicit.VertexSnorm16x4:
		return dxgiFormatRGBA16SN
	case explicit.VertexFloat16x2:
		return dxgiFormatRG16F
	case explicit.VertexFloat16x4:
		return dxgiFormatRGBA16F
	}
	return dxgiFormatRGBA32
}

// semanticLOC is the name of the semantics that naga writes for the locations of the vertex inputs:
// location N is LOCN (SPEC §9.8 "Shaders"). The input layout points to it, so it is NUL-terminated.
var semanticLOC = [4]byte{'L', 'O', 'C', 0}

// inputElement is one element of the input layout of a pipeline: the attribute of a vertex buffer,
// read as the semantic LOC<Location>.
type inputElement struct {
	Location, Format, Slot, Offset, Class, StepRate uint32
}

// inputElements turns the vertex buffer layouts of a pipeline into the elements of its input layout,
// buffer by buffer, in the order of the attributes. A buffer that advances per instance steps once for
// each instance.
func inputElements(vbs []explicit.VertexBufferLayout) []inputElement {
	var els []inputElement
	for slot, vb := range vbs {
		class, step := inputPerVertex, uint32(0)
		if vb.PerInstance {
			class, step = inputPerInstance, 1
		}
		for _, a := range vb.Attributes {
			els = append(els, inputElement{
				Location: uint32(a.Location), Format: dxgiVertexFormat(a.Format), Slot: uint32(slot), Offset: uint32(a.Offset),
				Class: class, StepRate: step,
			})
		}
	}
	return els
}

// loadAction is what a pass does with its target before it draws.
type loadAction uint8

const (
	// loadActionClear fills the target with the clear color (ClearRenderTargetView).
	loadActionClear loadAction = iota
	// loadActionKeep keeps the target's contents.
	loadActionKeep
	// loadActionDiscard tells the driver that the contents are not needed (DiscardResource), for a pass
	// that draws every pixel (SPEC §9.3).
	loadActionDiscard
)

// loadActionOf maps the load operation of a pass.
func loadActionOf(op explicit.LoadOp) loadAction {
	switch op {
	case explicit.LoadLoad:
		return loadActionKeep
	case explicit.LoadDontCare:
		return loadActionDiscard
	}
	return loadActionClear
}

// pipelineName is the name under which a pipeline library keeps the pipeline, and the key of the
// pipelines in memory: the hash of everything that makes the pipeline what it is, the shape of the root
// signature, the description of the pipeline, and the bytecode of both stages (so that a changed shader
// never finds the pipeline of the old one).
func pipelineName(desc explicit.PipelineDesc, rootKey string, vs, ps []byte) string {
	h := sha256.New()
	part := func(s string) {
		var n [8]byte
		binary.LittleEndian.PutUint64(n[:], uint64(len(s)))
		h.Write(n[:])
		h.Write([]byte(s))
	}
	part("glutter d3d12 pso 2")
	part(rootKey)
	part(fmt.Sprintf("%d|%+v|%v|%v", desc.Target, desc.Blend, desc.VertexEntry, desc.FragmentEntry))
	part(fmt.Sprintf("%+v|%d|%d|%d|%+v|%d", desc.ColorTargets, desc.Topology, desc.CullMode, desc.FrontFace, desc.DepthStencil, desc.Samples()))
	for _, vb := range desc.VertexBuffers {
		part(fmt.Sprintf("%d,%v,%v", vb.Stride, vb.PerInstance, vb.Attributes))
	}
	part(string(vs))
	part(string(ps))
	return "pso-" + hex.EncodeToString(h.Sum(nil)[:16])
}

// LibraryKey is the key under which the shader cache keeps the serialized pipeline library of an
// adapter: its vendor, device, revision and the version of its driver, so that a new driver or another
// adapter never reads a blob that is not for it (SPEC §9.8 "Pipelines").
//
// Flutter: no counterpart
func LibraryKey(vendor, device, revision uint32, driver uint64) string {
	return fmt.Sprintf("d3d12-pso-library-1-%04x-%04x-%02x-%016x", vendor, device, revision, driver)
}

// libraryRefusal says why a pipeline library did not take the blob it was given, for the log: the
// blob belongs to another adapter or another driver version, or it is not a library at all.
func libraryRefusal(code uint32) string {
	switch code {
	case d3d12ErrorAdapterNotFound:
		return "the blob is for another adapter"
	case d3d12ErrorDriverVersionMismatch:
		return "the blob is for another driver version"
	case 0x80070057: // E_INVALIDARG
		return "the blob is not a pipeline library"
	}
	return "the library refused the blob"
}
