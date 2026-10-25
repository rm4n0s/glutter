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
	}
	return dxgiFormatUnknown
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
		AddressU: d3dAddress(d.AddressU), AddressV: d3dAddress(d.AddressV), AddressW: d3dAddress(d.AddressU),
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
	rt := d3d12RenderTargetBlendDesc{
		SrcBlend: blendOne, DestBlend: blendZero, BlendOp: blendOpAdd,
		SrcBlendAlpha: blendOne, DestBlendAlpha: blendZero, BlendOpAlpha: blendOpAdd,
		LogicOp: logicOpNoop, RenderTargetWriteMask: colorWriteEnableAll,
	}
	if b.Enabled {
		rt.BlendEnable = 1
		rt.SrcBlend, rt.SrcBlendAlpha = d3dBlend(b.SrcFactor, false), d3dBlend(b.SrcFactor, false)
		rt.DestBlend, rt.DestBlendAlpha = d3dBlend(b.DstFactor, true), d3dBlend(b.DstFactor, true)
	}
	return d3d12BlendDesc{RenderTarget: [8]d3d12RenderTargetBlendDesc{rt}}
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
	part("glutter d3d12 pso 1")
	part(rootKey)
	part(fmt.Sprintf("%d|%+v|%v|%v", desc.Target, desc.Blend, desc.VertexEntry, desc.FragmentEntry))
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
