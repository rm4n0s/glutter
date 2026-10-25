package d3d12

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

func TestDXGIFormat(t *testing.T) {
	tests := []struct {
		f    explicit.TextureFormat
		want uint32
	}{
		{explicit.TextureFormatRGBA8Unorm, dxgiFormatRGBA8},
		{explicit.TextureFormatBGRA8Unorm, dxgiFormatBGRA8},
		{explicit.TextureFormatR8Unorm, dxgiFormatR8},
		{explicit.TextureFormatUnspecified, dxgiFormatUnknown},
	}
	for _, tt := range tests {
		if got := dxgiFormat(tt.f); got != tt.want {
			t.Errorf("dxgiFormat(%d) = %d, want %d", tt.f, got, tt.want)
		}
	}
}

func TestD3DFilter(t *testing.T) {
	n, l := explicit.FilterNearest, explicit.FilterLinear
	tests := []struct {
		name          string
		min, mag, mip explicit.FilterMode
		want          uint32
	}{
		{"all unspecified is point", explicit.FilterUnspecified, explicit.FilterUnspecified, explicit.FilterUnspecified, filterMinMagMipPoint},
		{"all nearest", n, n, n, filterMinMagMipPoint},
		{"mip linear", n, n, l, filterMipLinear},
		{"mag linear", n, l, n, filterMagLinear},
		{"min linear", l, n, n, filterMinLinear},
		{"min and mag linear", l, l, n, filterMinLinear | filterMagLinear},
		{"all linear", l, l, l, filterMinMagMipLinear},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d3dFilter(tt.min, tt.mag, tt.mip); got != tt.want {
				t.Errorf("d3dFilter = %#x, want %#x", got, tt.want)
			}
		})
	}
}

func TestD3DAddress(t *testing.T) {
	tests := []struct {
		m    explicit.AddressMode
		want uint32
	}{
		{explicit.AddressUnspecified, addressModeClamp},
		{explicit.AddressClamp, addressModeClamp},
		{explicit.AddressRepeat, addressModeWrap},
		{explicit.AddressMirror, addressModeMirror},
	}
	for _, tt := range tests {
		if got := d3dAddress(tt.m); got != tt.want {
			t.Errorf("d3dAddress(%d) = %d, want %d", tt.m, got, tt.want)
		}
	}
}

func TestSamplerDesc(t *testing.T) {
	d := samplerDesc(explicit.SamplerDesc{MinFilter: explicit.FilterLinear, MagFilter: explicit.FilterLinear, AddressU: explicit.AddressRepeat})
	if d.Filter != filterMinLinear|filterMagLinear || d.MaxLOD != 0 {
		t.Errorf("without a mip filter only level 0 is read: filter %#x, MaxLOD %v", d.Filter, d.MaxLOD)
	}
	if d.AddressU != addressModeWrap || d.AddressW != addressModeWrap || d.AddressV != addressModeClamp {
		t.Errorf("addresses = %d, %d, %d", d.AddressU, d.AddressV, d.AddressW)
	}
	d = samplerDesc(explicit.SamplerDesc{MinFilter: explicit.FilterLinear, MagFilter: explicit.FilterLinear, MipmapFilter: explicit.FilterLinear})
	if d.Filter != filterMinMagMipLinear || d.MaxLOD != float32Max || d.MaxAnisotropy != 1 {
		t.Errorf("with a mip filter every level is read: filter %#x, MaxLOD %v", d.Filter, d.MaxLOD)
	}
}
