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

func TestBlendDesc(t *testing.T) {
	e := explicit.BlendState{Enabled: true}
	tests := []struct {
		name string
		in   explicit.BlendState
		want d3d12RenderTargetBlendDesc
	}{
		{"off replaces the destination", explicit.BlendState{}, d3d12RenderTargetBlendDesc{
			SrcBlend: blendOne, DestBlend: blendZero, BlendOp: blendOpAdd, SrcBlendAlpha: blendOne, DestBlendAlpha: blendZero, BlendOpAlpha: blendOpAdd,
			LogicOp: logicOpNoop, RenderTargetWriteMask: colorWriteEnableAll}},
		{"off ignores the factors", explicit.BlendState{SrcFactor: explicit.BlendSrcAlpha, DstFactor: explicit.BlendDstAlpha}, d3d12RenderTargetBlendDesc{
			SrcBlend: blendOne, DestBlend: blendZero, BlendOp: blendOpAdd, SrcBlendAlpha: blendOne, DestBlendAlpha: blendZero, BlendOpAlpha: blendOpAdd,
			LogicOp: logicOpNoop, RenderTargetWriteMask: colorWriteEnableAll}},
		{"unspecified factors: source one, destination zero", e, d3d12RenderTargetBlendDesc{
			BlendEnable: 1, SrcBlend: blendOne, DestBlend: blendZero, BlendOp: blendOpAdd, SrcBlendAlpha: blendOne, DestBlendAlpha: blendZero, BlendOpAlpha: blendOpAdd,
			LogicOp: logicOpNoop, RenderTargetWriteMask: colorWriteEnableAll}},
		{"source over", explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendOne, DstFactor: explicit.BlendOneMinusSrcAlpha}, d3d12RenderTargetBlendDesc{
			BlendEnable: 1, SrcBlend: blendOne, DestBlend: blendInvSrcAlpha, BlendOp: blendOpAdd, SrcBlendAlpha: blendOne, DestBlendAlpha: blendInvSrcAlpha, BlendOpAlpha: blendOpAdd,
			LogicOp: logicOpNoop, RenderTargetWriteMask: colorWriteEnableAll}},
		{"destination over", explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendOneMinusDstAlpha, DstFactor: explicit.BlendOne}, d3d12RenderTargetBlendDesc{
			BlendEnable: 1, SrcBlend: blendInvDestAlpha, DestBlend: blendOne, BlendOp: blendOpAdd, SrcBlendAlpha: blendInvDestAlpha, DestBlendAlpha: blendOne, BlendOpAlpha: blendOpAdd,
			LogicOp: logicOpNoop, RenderTargetWriteMask: colorWriteEnableAll}},
		{"destination out", explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendZero, DstFactor: explicit.BlendOneMinusSrcAlpha}, d3d12RenderTargetBlendDesc{
			BlendEnable: 1, SrcBlend: blendZero, DestBlend: blendInvSrcAlpha, BlendOp: blendOpAdd, SrcBlendAlpha: blendZero, DestBlendAlpha: blendInvSrcAlpha, BlendOpAlpha: blendOpAdd,
			LogicOp: logicOpNoop, RenderTargetWriteMask: colorWriteEnableAll}},
		{"alphas", explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendSrcAlpha, DstFactor: explicit.BlendDstAlpha}, d3d12RenderTargetBlendDesc{
			BlendEnable: 1, SrcBlend: blendSrcAlpha, DestBlend: blendDestAlpha, BlendOp: blendOpAdd, SrcBlendAlpha: blendSrcAlpha, DestBlendAlpha: blendDestAlpha, BlendOpAlpha: blendOpAdd,
			LogicOp: logicOpNoop, RenderTargetWriteMask: colorWriteEnableAll}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blendDesc(tt.in)
			if got.RenderTarget[0] != tt.want {
				t.Errorf("render target 0 = %+v\nwant %+v", got.RenderTarget[0], tt.want)
			}
			if got.AlphaToCoverageEnable != 0 || got.IndependentBlendEnable != 0 {
				t.Errorf("blend desc %+v", got)
			}
		})
	}
}

func TestDXGIVertexFormat(t *testing.T) {
	tests := []struct {
		f    explicit.VertexFormat
		want uint32
	}{
		{explicit.VertexFloat32, dxgiFormatR32},
		{explicit.VertexFloat32x2, dxgiFormatRG32},
		{explicit.VertexFloat32x3, dxgiFormatRGB32},
		{explicit.VertexFloat32x4, dxgiFormatRGBA32},
		{explicit.VertexUnorm8x4, dxgiFormatRGBA8},
		{explicit.VertexFormatUnspecified, dxgiFormatRGBA32},
	}
	for _, tt := range tests {
		if got := dxgiVertexFormat(tt.f); got != tt.want {
			t.Errorf("dxgiVertexFormat(%d) = %d, want %d", tt.f, got, tt.want)
		}
	}
}

func TestInputElements(t *testing.T) {
	vbs := []explicit.VertexBufferLayout{
		{Stride: 32, PerInstance: true, Attributes: []explicit.VertexAttribute{
			{Location: 0, Format: explicit.VertexFloat32x4, Offset: 0}, {Location: 1, Format: explicit.VertexFloat32x4, Offset: 16}}},
		{Stride: 8, Attributes: []explicit.VertexAttribute{{Location: 3, Format: explicit.VertexFloat32x2, Offset: 4}}},
		{Stride: 4, Attributes: []explicit.VertexAttribute{{Location: 2, Format: explicit.VertexUnorm8x4}}},
	}
	want := []inputElement{
		{Location: 0, Format: dxgiFormatRGBA32, Slot: 0, Offset: 0, Class: inputPerInstance, StepRate: 1},
		{Location: 1, Format: dxgiFormatRGBA32, Slot: 0, Offset: 16, Class: inputPerInstance, StepRate: 1},
		{Location: 3, Format: dxgiFormatRG32, Slot: 1, Offset: 4, Class: inputPerVertex, StepRate: 0},
		{Location: 2, Format: dxgiFormatRGBA8, Slot: 2, Offset: 0, Class: inputPerVertex, StepRate: 0},
	}
	got := inputElements(vbs)
	if len(got) != len(want) {
		t.Fatalf("%d elements, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("element %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if len(inputElements(nil)) != 0 {
		t.Error("no vertex buffers, no elements")
	}
	if string(semanticLOC[:3]) != "LOC" || semanticLOC[3] != 0 {
		t.Errorf("semanticLOC = %v", semanticLOC)
	}
}

func TestLoadActionOf(t *testing.T) {
	tests := []struct {
		op   explicit.LoadOp
		want loadAction
	}{
		{explicit.LoadClear, loadActionClear},
		{explicit.LoadLoad, loadActionKeep},
		{explicit.LoadDontCare, loadActionDiscard},
	}
	for _, tt := range tests {
		if got := loadActionOf(tt.op); got != tt.want {
			t.Errorf("loadActionOf(%d) = %d, want %d", tt.op, got, tt.want)
		}
	}
}

func TestPipelineNameAndLibraryKey(t *testing.T) {
	base := explicit.PipelineDesc{VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: explicit.TextureFormatRGBA8Unorm,
		VertexBuffers: []explicit.VertexBufferLayout{{Stride: 16, Attributes: []explicit.VertexAttribute{{Location: 0, Format: explicit.VertexFloat32x4}}}}}
	name := func(d explicit.PipelineDesc, root string, vs, ps string) string {
		return pipelineName(d, root, []byte(vs), []byte(ps))
	}
	ref := name(base, "b0//", "vs", "ps")
	if ref != name(base, "b0//", "vs", "ps") {
		t.Error("the name of one pipeline is not stable")
	}
	changed := func(f func(d *explicit.PipelineDesc)) explicit.PipelineDesc {
		d := base
		d.VertexBuffers = append([]explicit.VertexBufferLayout(nil), base.VertexBuffers...)
		f(&d)
		return d
	}
	tests := []struct{ name, got string }{
		{"the root signature", name(base, "b0b1//", "vs", "ps")},
		{"the vertex bytecode", name(base, "b0//", "VS", "ps")},
		{"the pixel bytecode", name(base, "b0//", "vs", "PS")},
		{"the bytecode split differently", name(base, "b0//", "vsp", "s")},
		{"the target", name(changed(func(d *explicit.PipelineDesc) { d.Target = explicit.TextureFormatBGRA8Unorm }), "b0//", "vs", "ps")},
		{"the blend", name(changed(func(d *explicit.PipelineDesc) { d.Blend = explicit.BlendState{Enabled: true} }), "b0//", "vs", "ps")},
		{"the stride", name(changed(func(d *explicit.PipelineDesc) { d.VertexBuffers[0].Stride = 32 }), "b0//", "vs", "ps")},
		{"the instance step", name(changed(func(d *explicit.PipelineDesc) { d.VertexBuffers[0].PerInstance = true }), "b0//", "vs", "ps")},
		{"an attribute", name(changed(func(d *explicit.PipelineDesc) {
			d.VertexBuffers[0].Attributes = []explicit.VertexAttribute{{Location: 1, Format: explicit.VertexFloat32x4}}
		}), "b0//", "vs", "ps")},
		{"the entry point", name(changed(func(d *explicit.PipelineDesc) { d.FragmentEntry = "fs_other" }), "b0//", "vs", "ps")},
	}
	seen := map[string]string{ref: "the reference"}
	for _, tt := range tests {
		if tt.got == ref {
			t.Errorf("changing %s does not change the name", tt.name)
		}
		if other, dup := seen[tt.got]; dup {
			t.Errorf("changing %s gives the name that %s gives", tt.name, other)
		}
		seen[tt.got] = tt.name
	}
	k := LibraryKey(0x10DE, 0x1D01, 2, 0x000A0000)
	for _, k2 := range []string{LibraryKey(0x1002, 0x1D01, 2, 0x000A0000), LibraryKey(0x10DE, 0x1D02, 2, 0x000A0000), LibraryKey(0x10DE, 0x1D01, 3, 0x000A0000), LibraryKey(0x10DE, 0x1D01, 2, 0x000A0001)} {
		if k2 == k {
			t.Errorf("a different adapter or driver has the key %q", k)
		}
	}
	if want := "d3d12-pso-library-1-10de-1d01-02-00000000000a0000"; k != want {
		t.Errorf("LibraryKey = %q, want %q", k, want)
	}
}

func TestLibraryRefusal(t *testing.T) {
	tests := []struct {
		code uint32
		want string
	}{
		{d3d12ErrorAdapterNotFound, "the blob is for another adapter"},
		{d3d12ErrorDriverVersionMismatch, "the blob is for another driver version"},
		{0x80070057, "the blob is not a pipeline library"},
		{0x80004005, "the library refused the blob"},
	}
	for _, tt := range tests {
		if got := libraryRefusal(tt.code); got != tt.want {
			t.Errorf("libraryRefusal(%#x) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
