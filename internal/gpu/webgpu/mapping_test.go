package webgpu

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

func TestUsageFlags(t *testing.T) {
	bufs := []struct {
		name string
		in   explicit.BufferUsage
		want int
	}{
		{"nothing is still writable", 0, bufferCopyDst},
		{"vertex", explicit.BufferVertex, bufferCopyDst | bufferVertex},
		{"index", explicit.BufferIndex, bufferCopyDst | bufferIndex},
		{"uniform", explicit.BufferUniform, bufferCopyDst | bufferUniform},
		{"vertex and uniform", explicit.BufferVertex | explicit.BufferUniform, bufferCopyDst | bufferVertex | bufferUniform},
	}
	for _, tt := range bufs {
		if got := bufferUsage(tt.in); got != tt.want {
			t.Errorf("bufferUsage %s = %#x, want %#x", tt.name, got, tt.want)
		}
	}
	texs := []struct {
		name string
		in   explicit.TextureUsage
		want int
	}{
		{"nothing is still writable", 0, textureCopyDst},
		{"sampled", explicit.TextureSampled, textureCopyDst | textureBinding},
		{"a render target can be copied from", explicit.TextureRenderTarget, textureCopyDst | textureTarget | textureCopySrc},
		{"copy source", explicit.TextureCopySource, textureCopyDst | textureCopySrc},
		{"the frame texture", explicit.TextureRenderTarget | explicit.TextureSampled | explicit.TextureCopySource,
			textureCopyDst | textureTarget | textureCopySrc | textureBinding},
	}
	for _, tt := range texs {
		if got := textureUsage(tt.in); got != tt.want {
			t.Errorf("textureUsage %s = %#x, want %#x", tt.name, got, tt.want)
		}
	}
}

func TestFormatNames(t *testing.T) {
	tests := []struct {
		f    explicit.TextureFormat
		name string
	}{
		{explicit.TextureFormatRGBA8Unorm, "rgba8unorm"},
		{explicit.TextureFormatBGRA8Unorm, "bgra8unorm"},
		{explicit.TextureFormatR8Unorm, "r8unorm"},
		{explicit.TextureFormatUnspecified, ""},
	}
	for _, tt := range tests {
		if got := formatName(tt.f); got != tt.name {
			t.Errorf("formatName(%d) = %q, want %q", tt.f, got, tt.name)
		}
		if got := formatOf(tt.name); got != tt.f {
			t.Errorf("formatOf(%q) = %d, want %d", tt.name, got, tt.f)
		}
	}
	if got := formatOf("rgba16float"); got != explicit.TextureFormatUnspecified {
		t.Errorf("formatOf of a format the renderer does not draw into = %d", got)
	}
}

func TestSamplerNames(t *testing.T) {
	if filterName(explicit.FilterUnspecified) != "nearest" || filterName(explicit.FilterNearest) != "nearest" || filterName(explicit.FilterLinear) != "linear" {
		t.Error("filterName")
	}
	for m, want := range map[explicit.AddressMode]string{
		explicit.AddressUnspecified: "clamp-to-edge", explicit.AddressClamp: "clamp-to-edge",
		explicit.AddressRepeat: "repeat", explicit.AddressMirror: "mirror-repeat",
	} {
		if got := addressName(m); got != want {
			t.Errorf("addressName(%d) = %q, want %q", m, got, want)
		}
	}
	if samplerLodMax(explicit.SamplerDesc{}) != 0 || samplerLodMax(explicit.SamplerDesc{MipmapFilter: explicit.FilterLinear}) != 32 {
		t.Error("a sampler with no mipmap filter reads level 0 only, one with a filter every level")
	}
}

func TestVertexFormatNames(t *testing.T) {
	for f, want := range map[explicit.VertexFormat]string{
		explicit.VertexFloat32: "float32", explicit.VertexFloat32x2: "float32x2", explicit.VertexFloat32x3: "float32x3",
		explicit.VertexFloat32x4: "float32x4", explicit.VertexUnorm8x4: "unorm8x4", explicit.VertexFormatUnspecified: "float32x4",
	} {
		if got := vertexFormatName(f); got != want {
			t.Errorf("vertexFormatName(%d) = %q, want %q", f, got, want)
		}
	}
}

func TestBlendFactors(t *testing.T) {
	tests := []struct {
		name     string
		in       explicit.BlendState
		src, dst string
	}{
		{"source over", explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendOne, DstFactor: explicit.BlendOneMinusSrcAlpha}, "one", "one-minus-src-alpha"},
		{"unspecified source is one, destination zero", explicit.BlendState{Enabled: true}, "one", "zero"},
		{"destination over", explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendOneMinusDstAlpha, DstFactor: explicit.BlendOne}, "one-minus-dst-alpha", "one"},
		{"destination out", explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendZero, DstFactor: explicit.BlendOneMinusSrcAlpha}, "zero", "one-minus-src-alpha"},
		{"alphas", explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendSrcAlpha, DstFactor: explicit.BlendDstAlpha}, "src-alpha", "dst-alpha"},
	}
	for _, tt := range tests {
		src, dst := blendFactors(tt.in)
		if src != tt.src || dst != tt.dst {
			t.Errorf("%s: got %s, %s; want %s, %s", tt.name, src, dst, tt.src, tt.dst)
		}
	}
}

func TestAlignUp(t *testing.T) {
	for _, tt := range [][3]int{{0, 4, 0}, {1, 4, 4}, {4, 4, 4}, {5, 4, 8}, {7, 1, 7}, {300, 256, 512}} {
		if got := alignUp(tt[0], tt[1]); got != tt[2] {
			t.Errorf("alignUp(%d, %d) = %d, want %d", tt[0], tt[1], got, tt[2])
		}
	}
}
