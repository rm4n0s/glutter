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
		{"storage", explicit.BufferStorage, bufferCopyDst | bufferStorage},
		{"the arguments of an indirect draw that a shader wrote", explicit.BufferStorage | explicit.BufferIndirect, bufferCopyDst | bufferStorage | bufferIndirect},
		{"a buffer that is copied and read", explicit.BufferCopySource | explicit.BufferCopyDestination, bufferCopyDst | bufferCopySrc},
		{"the timestamps", explicit.BufferQueryResolve | explicit.BufferCopySource, bufferCopyDst | bufferQueryResolve | bufferCopySrc},
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
		{"storage", explicit.TextureStorage, textureCopyDst | textureStorage},
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
	for f, want := range map[explicit.TextureFormat]string{
		explicit.TextureFormatRGBA16Float: "rgba16float", explicit.TextureFormatRGBA32Float: "rgba32float",
		explicit.TextureFormatR32Float: "r32float", explicit.TextureFormatR32Uint: "r32uint", explicit.TextureFormatRG32Uint: "rg32uint",
		explicit.TextureFormatDepth32Float: "depth32float", explicit.TextureFormatDepth24PlusStencil8: "depth24plus-stencil8",
	} {
		if got := formatName(f); got != want {
			t.Errorf("formatName(%d) = %q, want %q", f, got, want)
		}
		if got := formatOf(want); got != explicit.TextureFormatUnspecified {
			t.Errorf("formatOf(%q) = %d: a canvas has only the three formats", want, got)
		}
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

func TestEveryVertexFormatHasAName(t *testing.T) {
	seen := map[string]explicit.VertexFormat{}
	for f := explicit.VertexFloat32; f <= explicit.VertexFloat16x4; f++ {
		name := vertexFormatName(f)
		if other, dup := seen[name]; dup {
			t.Errorf("formats %d and %d are both %q", other, f, name)
		}
		seen[name] = f
		if f.Size() == 0 {
			t.Errorf("format %d (%s) has no size", f, name)
		}
	}
	for f, want := range map[explicit.VertexFormat]string{
		explicit.VertexUint32x3: "uint32x3", explicit.VertexSint32: "sint32", explicit.VertexSnorm16x2: "snorm16x2",
		explicit.VertexFloat16x4: "float16x4", explicit.VertexUint8x4: "uint8x4", explicit.VertexSnorm8x2: "snorm8x2",
	} {
		if got := vertexFormatName(f); got != want {
			t.Errorf("vertexFormatName(%d) = %q, want %q", f, got, want)
		}
	}
}

func TestPrimitiveAndDepthNames(t *testing.T) {
	for topology, want := range map[explicit.PrimitiveTopology]string{
		explicit.TopologyTriangleList: "triangle-list", explicit.TopologyTriangleStrip: "triangle-strip",
		explicit.TopologyLineList: "line-list", explicit.TopologyLineStrip: "line-strip", explicit.TopologyPointList: "point-list",
	} {
		if got := topologyName(topology); got != want {
			t.Errorf("topologyName(%d) = %q, want %q", topology, got, want)
		}
	}
	if cullName(explicit.CullNone) != "none" || cullName(explicit.CullFront) != "front" || cullName(explicit.CullBack) != "back" {
		t.Error("cullName")
	}
	if frontFaceName(explicit.FrontFaceCCW) != "ccw" || frontFaceName(explicit.FrontFaceCW) != "cw" {
		t.Error("frontFaceName")
	}
	for c, want := range map[explicit.CompareFunc]string{
		explicit.CompareUnspecified: "always", explicit.CompareNever: "never", explicit.CompareLess: "less", explicit.CompareEqual: "equal",
		explicit.CompareLessEqual: "less-equal", explicit.CompareGreater: "greater", explicit.CompareNotEqual: "not-equal",
		explicit.CompareGreaterEqual: "greater-equal", explicit.CompareAlways: "always",
	} {
		if got := compareName(c); got != want {
			t.Errorf("compareName(%d) = %q, want %q", c, got, want)
		}
	}
	for o, want := range map[explicit.StencilOp]string{
		explicit.StencilKeep: "keep", explicit.StencilZero: "zero", explicit.StencilReplace: "replace", explicit.StencilInvert: "invert",
		explicit.StencilIncrementClamp: "increment-clamp", explicit.StencilDecrementClamp: "decrement-clamp",
		explicit.StencilIncrementWrap: "increment-wrap", explicit.StencilDecrementWrap: "decrement-wrap",
	} {
		if got := stencilOpName(o); got != want {
			t.Errorf("stencilOpName(%d) = %q, want %q", o, got, want)
		}
	}
	if loadOpName(explicit.LoadClear) != "clear" || loadOpName(explicit.LoadLoad) != "load" || loadOpName(explicit.LoadDontCare) != "clear" {
		t.Error("loadOpName")
	}
	if storeOpName(explicit.StoreStore) != "store" || storeOpName(explicit.StoreDiscard) != "discard" {
		t.Error("storeOpName")
	}
	for m, want := range map[explicit.ColorWriteMask]int{
		explicit.ColorWriteAll: 15, explicit.ColorWriteNone: 0, explicit.ColorWriteRed: 1,
		explicit.ColorWriteRed | explicit.ColorWriteAlpha: 9, explicit.ColorWriteGreen | explicit.ColorWriteBlue: 6,
	} {
		if got := writeMaskBits(m); got != want {
			t.Errorf("writeMaskBits(%d) = %d, want %d", m, got, want)
		}
	}
	if stencilMask(0) != 0xFFFFFFFF || stencilMask(0x0f) != 15 {
		t.Error("stencilMask")
	}
}

func TestTextureKindsAndSampleTypes(t *testing.T) {
	for kind, want := range map[explicit.TextureKind][2]string{
		explicit.TextureKindUnspecified: {"2d", "2d"}, explicit.TextureKind2D: {"2d", "2d"}, explicit.TextureKind2DArray: {"2d-array", "2d"},
		explicit.TextureKindCube: {"cube", "2d"}, explicit.TextureKind3D: {"3d", "3d"},
	} {
		if got := dimensionName(kind); got != want[0] {
			t.Errorf("dimensionName(%d) = %q, want %q", kind, got, want[0])
		}
		if got := textureDimensionName(kind); got != want[1] {
			t.Errorf("textureDimensionName(%d) = %q, want %q", kind, got, want[1])
		}
		if got := bindingDimensionName(kind.Dimension()); got != want[0] {
			t.Errorf("the binding dimension of kind %d = %q, want %q", kind, got, want[0])
		}
	}
	for s, want := range map[explicit.SampleType]string{
		explicit.SampleTypeFloat: "float", explicit.SampleTypeUnfilterableFloat: "unfilterable-float", explicit.SampleTypeUint: "uint",
	} {
		if got := sampleTypeName(s); got != want {
			t.Errorf("sampleTypeName(%d) = %q, want %q", s, got, want)
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
