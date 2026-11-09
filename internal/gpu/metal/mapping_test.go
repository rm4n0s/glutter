package metal

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// The numbers in these tables are the values of the headers of Metal (MTLPixelFormat.h and the others), not
// the constants of mapping.go, so that a wrong constant fails.

func TestPixelFormat(t *testing.T) {
	tests := []struct {
		f    explicit.TextureFormat
		want uint64
	}{
		{explicit.TextureFormatUnspecified, 0},
		{explicit.TextureFormatR8Unorm, 10},
		{explicit.TextureFormatR32Uint, 53},
		{explicit.TextureFormatR32Float, 55},
		{explicit.TextureFormatRGBA8Unorm, 70},
		{explicit.TextureFormatBGRA8Unorm, 80},
		{explicit.TextureFormatRG32Uint, 103},
		{explicit.TextureFormatRGBA16Float, 115},
		{explicit.TextureFormatRGBA32Float, 125},
		{explicit.TextureFormatDepth32Float, 252},
		{explicit.TextureFormatDepth24PlusStencil8, 260},
	}
	for _, tt := range tests {
		if got := pixelFormat(tt.f); got != tt.want {
			t.Errorf("pixelFormat(%d) = %d, want %d", tt.f, got, tt.want)
		}
	}
}

func TestEveryFormatHasAPixelFormat(t *testing.T) {
	for f := explicit.TextureFormatRGBA8Unorm; f <= explicit.TextureFormatDepth24PlusStencil8; f++ {
		if pixelFormat(f) == pixelFormatInvalid {
			t.Errorf("format %d has no pixel format", f)
		}
	}
}

func TestTextureTypeAndUsage(t *testing.T) {
	types := []struct {
		kind    explicit.TextureKind
		samples int
		want    uint64
	}{
		{explicit.TextureKindUnspecified, 1, 2},
		{explicit.TextureKind2D, 1, 2},
		{explicit.TextureKind2DArray, 1, 3},
		{explicit.TextureKind2D, 4, 4},
		{explicit.TextureKindCube, 1, 5},
		{explicit.TextureKind3D, 1, 7},
	}
	for _, tt := range types {
		if got := textureType(tt.kind, tt.samples); got != tt.want {
			t.Errorf("textureType(%d, %d) = %d, want %d", tt.kind, tt.samples, got, tt.want)
		}
	}
	usages := []struct {
		u    explicit.TextureUsage
		want uint64
	}{
		{explicit.TextureSampled, 1},
		{explicit.TextureRenderTarget, 4},
		{explicit.TextureSampled | explicit.TextureRenderTarget, 5},
		{explicit.TextureStorage, 3},
		{explicit.TextureCopySource, 0},
	}
	for _, tt := range usages {
		if got := textureUsage(tt.u); got != tt.want {
			t.Errorf("textureUsage(%d) = %d, want %d", tt.u, got, tt.want)
		}
	}
}

func TestResourceOptions(t *testing.T) {
	tests := []struct {
		storage uint64
		wc      bool
		want    uint64
	}{
		{storageShared, false, 0},
		{storageShared, true, 1},
		{storageManaged, false, 0x10},
		{storageManaged, true, 0x11},
		{storagePrivate, false, 0x20},
	}
	for _, tt := range tests {
		if got := resourceOptions(tt.storage, tt.wc); got != tt.want {
			t.Errorf("resourceOptions(%d, %v) = %#x, want %#x", tt.storage, tt.wc, got, tt.want)
		}
	}
}

func TestSamplerMapping(t *testing.T) {
	if minMagFilter(explicit.FilterUnspecified) != 0 || minMagFilter(explicit.FilterNearest) != 0 || minMagFilter(explicit.FilterLinear) != 1 {
		t.Error("the min and mag filters are wrong")
	}
	if mipFilter(explicit.FilterUnspecified) != 0 || mipFilter(explicit.FilterNearest) != 1 || mipFilter(explicit.FilterLinear) != 2 {
		t.Error("the mip filters are wrong: the unspecified one must read level 0 only")
	}
	for m, want := range map[explicit.AddressMode]uint64{
		explicit.AddressUnspecified: 0, explicit.AddressClamp: 0, explicit.AddressRepeat: 2, explicit.AddressMirror: 3,
	} {
		if got := addressMode(m); got != want {
			t.Errorf("addressMode(%d) = %d, want %d", m, got, want)
		}
	}
}

func TestBlendFor(t *testing.T) {
	tests := []struct {
		name string
		in   explicit.BlendState
		want blendOf
	}{
		{"off replaces", explicit.BlendState{}, blendOf{false, 1, 0, 1, 0}},
		{"off ignores the factors", explicit.BlendState{SrcFactor: explicit.BlendSrcAlpha}, blendOf{false, 1, 0, 1, 0}},
		{"source over", explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendOne, DstFactor: explicit.BlendOneMinusSrcAlpha}, blendOf{true, 1, 5, 1, 5}},
		{"unspecified factors are one and zero", explicit.BlendState{Enabled: true}, blendOf{true, 1, 0, 1, 0}},
		{"destination alpha", explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendDstAlpha, DstFactor: explicit.BlendOneMinusDstAlpha}, blendOf{true, 8, 9, 8, 9}},
		{"zero source", explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendZero, DstFactor: explicit.BlendSrcAlpha}, blendOf{true, 0, 4, 0, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := blendFor(tt.in); got != tt.want {
				t.Errorf("blendFor = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestWriteMaskReversesTheBits(t *testing.T) {
	tests := []struct {
		m    explicit.ColorWriteMask
		want uint64
	}{
		{explicit.ColorWriteAll, 15},
		{explicit.ColorWriteNone, 0},
		{explicit.ColorWriteRed, 8},
		{explicit.ColorWriteGreen, 4},
		{explicit.ColorWriteBlue, 2},
		{explicit.ColorWriteAlpha, 1},
		{explicit.ColorWriteRed | explicit.ColorWriteAlpha, 9},
	}
	for _, tt := range tests {
		if got := writeMask(tt.m); got != tt.want {
			t.Errorf("writeMask(%d) = %d, want %d", tt.m, got, tt.want)
		}
	}
}

func TestTopologyAndFaces(t *testing.T) {
	tops := []struct {
		t       explicit.PrimitiveTopology
		prim    uint64
		classOf uint64
	}{
		{explicit.TopologyTriangleList, 3, 3},
		{explicit.TopologyTriangleStrip, 4, 3},
		{explicit.TopologyLineList, 1, 2},
		{explicit.TopologyLineStrip, 2, 2},
		{explicit.TopologyPointList, 0, 1},
	}
	for _, tt := range tops {
		if primitiveType(tt.t) != tt.prim || topologyClass(tt.t) != tt.classOf {
			t.Errorf("topology %d: type %d class %d, want %d and %d", tt.t, primitiveType(tt.t), topologyClass(tt.t), tt.prim, tt.classOf)
		}
	}
	if cullMode(explicit.CullNone) != 0 || cullMode(explicit.CullFront) != 1 || cullMode(explicit.CullBack) != 2 {
		t.Error("the cull modes are wrong")
	}
	if winding(explicit.FrontFaceCCW) != 1 || winding(explicit.FrontFaceCW) != 0 {
		t.Error("the windings are wrong")
	}
}

func TestCompareAndStencil(t *testing.T) {
	cmps := map[explicit.CompareFunc]uint64{
		explicit.CompareUnspecified: 7, explicit.CompareNever: 0, explicit.CompareLess: 1, explicit.CompareEqual: 2,
		explicit.CompareLessEqual: 3, explicit.CompareGreater: 4, explicit.CompareNotEqual: 5,
		explicit.CompareGreaterEqual: 6, explicit.CompareAlways: 7,
	}
	for c, want := range cmps {
		if got := compareFunction(c); got != want {
			t.Errorf("compareFunction(%d) = %d, want %d", c, got, want)
		}
	}
	ops := map[explicit.StencilOp]uint64{
		explicit.StencilKeep: 0, explicit.StencilZero: 1, explicit.StencilReplace: 2, explicit.StencilIncrementClamp: 3,
		explicit.StencilDecrementClamp: 4, explicit.StencilInvert: 5, explicit.StencilIncrementWrap: 6, explicit.StencilDecrementWrap: 7,
	}
	for o, want := range ops {
		if got := stencilOperation(o); got != want {
			t.Errorf("stencilOperation(%d) = %d, want %d", o, got, want)
		}
	}
	if stencilMask(0) != 0xFF || stencilMask(0x0F) != 0x0F || stencilMask(0x1FF) != 0xFF {
		t.Error("the stencil masks are wrong")
	}
}

func TestLoadAndStoreActions(t *testing.T) {
	if loadAction(explicit.LoadClear) != 2 || loadAction(explicit.LoadLoad) != 1 || loadAction(explicit.LoadDontCare) != 0 {
		t.Error("the load actions are wrong")
	}
	tests := []struct {
		op      explicit.StoreOp
		resolve bool
		want    uint64
	}{
		{explicit.StoreStore, false, 1},
		{explicit.StoreDiscard, false, 0},
		{explicit.StoreStore, true, 3},
		{explicit.StoreDiscard, true, 2},
	}
	for _, tt := range tests {
		if got := storeAction(tt.op, tt.resolve); got != tt.want {
			t.Errorf("storeAction(%d, %v) = %d, want %d", tt.op, tt.resolve, got, tt.want)
		}
	}
}

// vertexFormats are the values of MTLVertexFormat for each format of explicit.
var vertexFormats = map[explicit.VertexFormat]uint64{
	explicit.VertexFormatUnspecified: 31, // four floats
	explicit.VertexFloat32:           28, explicit.VertexFloat32x2: 29, explicit.VertexFloat32x3: 30, explicit.VertexFloat32x4: 31,
	explicit.VertexUnorm8x4: 9,
	explicit.VertexUint32:   36, explicit.VertexUint32x2: 37, explicit.VertexUint32x3: 38, explicit.VertexUint32x4: 39,
	explicit.VertexSint32: 32, explicit.VertexSint32x2: 33, explicit.VertexSint32x3: 34, explicit.VertexSint32x4: 35,
	explicit.VertexUint8x2: 1, explicit.VertexUint8x4: 3, explicit.VertexSint8x2: 4, explicit.VertexSint8x4: 6,
	explicit.VertexUnorm8x2: 7, explicit.VertexSnorm8x2: 10, explicit.VertexSnorm8x4: 12,
	explicit.VertexUint16x2: 13, explicit.VertexUint16x4: 15, explicit.VertexSint16x2: 16, explicit.VertexSint16x4: 18,
	explicit.VertexUnorm16x2: 19, explicit.VertexUnorm16x4: 21, explicit.VertexSnorm16x2: 22, explicit.VertexSnorm16x4: 24,
	explicit.VertexFloat16x2: 25, explicit.VertexFloat16x4: 27,
}

func TestVertexFormat(t *testing.T) {
	for f, want := range vertexFormats {
		if got := vertexFormat(f); got != want {
			t.Errorf("vertexFormat(%d) = %d, want %d", f, got, want)
		}
	}
	// Every format of explicit up to the last one is in the table.
	for f := explicit.VertexFloat32; f <= explicit.VertexFloat16x4; f++ {
		if _, ok := vertexFormats[f]; !ok {
			t.Errorf("format %d is not in the test table", f)
		}
	}
}

func TestVertexBufferIndexIsAboveTheShaderBuffers(t *testing.T) {
	if vertexBufferIndex(0) != 16 || vertexBufferIndex(7) != 23 {
		t.Errorf("slots 0 and 7 are at %d and %d, want 16 and 23", vertexBufferIndex(0), vertexBufferIndex(7))
	}
}
