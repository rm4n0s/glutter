package d3d12

import "github.com/rm4n0s/glutter/internal/gpu/explicit"

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
