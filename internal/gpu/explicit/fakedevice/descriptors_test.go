package fakedevice

import (
	"fmt"
	"image"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

var allFormats = []explicit.TextureFormat{
	explicit.TextureFormatRGBA8Unorm, explicit.TextureFormatBGRA8Unorm, explicit.TextureFormatR8Unorm,
	explicit.TextureFormatRGBA16Float, explicit.TextureFormatRGBA32Float, explicit.TextureFormatR32Float,
	explicit.TextureFormatR32Uint, explicit.TextureFormatRG32Uint,
	explicit.TextureFormatDepth32Float, explicit.TextureFormatDepth24PlusStencil8,
}

var allUsages = []explicit.TextureUsage{
	explicit.TextureSampled, explicit.TextureRenderTarget, explicit.TextureCopySource, explicit.TextureStorage,
}

// expectInvalid runs do and checks that it panics with the tag, through the function of the route.
func expectInvalid(t *testing.T, tag, route string, do func()) *errors.Error {
	t.Helper()
	err := panicOf(do)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected a panic with *errors.Error, got %v", err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Fatalf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tag, route)
	}
	return appErr
}

func TestEveryFormatWithEveryUsage(t *testing.T) {
	skipUnlessDebug(t)
	for _, f := range allFormats {
		for _, u := range allUsages {
			t.Run(fmt.Sprintf("format %d usage %d", f, u), func(t *testing.T) {
				d := New()
				desc := explicit.TextureDesc{Width: 4, Height: 4, Format: f, Usage: u}
				if u&f.AllowedUsage() == u {
					if err := panicOf(func() { _, _ = d.NewTexture(desc) }); err != nil {
						t.Fatalf("an allowed usage panicked: %v", err)
					}
					return
				}
				appErr := expectInvalid(t, "GPUDescriptorInvalid", "Device.NewTexture->Limits.CheckTexture->invalid.GPUDescriptorInvalid", func() { _, _ = d.NewTexture(desc) })
				if appErr.Metadata["what"] != "texture usage for the format" {
					t.Errorf("what = %v", appErr.Metadata["what"])
				}
			})
		}
	}
}

func TestTheUsagesOfTheFormats(t *testing.T) {
	tests := []struct {
		format explicit.TextureFormat
		want   explicit.TextureUsage
		bpp    int
	}{
		{explicit.TextureFormatRGBA8Unorm, explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource | explicit.TextureStorage, 4},
		{explicit.TextureFormatBGRA8Unorm, explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource, 4},
		{explicit.TextureFormatR8Unorm, explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource, 1},
		{explicit.TextureFormatRGBA16Float, explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource | explicit.TextureStorage, 8},
		{explicit.TextureFormatRGBA32Float, explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource, 16},
		{explicit.TextureFormatR32Float, explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource | explicit.TextureStorage, 4},
		{explicit.TextureFormatR32Uint, explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource | explicit.TextureStorage, 4},
		{explicit.TextureFormatRG32Uint, explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource, 8},
		{explicit.TextureFormatDepth32Float, explicit.TextureRenderTarget | explicit.TextureCopySource, 4},
		{explicit.TextureFormatDepth24PlusStencil8, explicit.TextureRenderTarget, 4},
		{explicit.TextureFormatUnspecified, 0, 0},
	}
	for _, tt := range tests {
		if got := tt.format.AllowedUsage(); got != tt.want {
			t.Errorf("format %d allows %d, want %d", tt.format, got, tt.want)
		}
		if got := tt.format.BytesPerPixel(); got != tt.bpp {
			t.Errorf("format %d takes %d bytes, want %d", tt.format, got, tt.bpp)
		}
	}
	features := []struct {
		format explicit.TextureFormat
		with   bool
		want   explicit.SampleType
	}{
		{explicit.TextureFormatRGBA8Unorm, false, explicit.SampleTypeFloat},
		{explicit.TextureFormatRGBA16Float, false, explicit.SampleTypeFloat},
		{explicit.TextureFormatRGBA32Float, false, explicit.SampleTypeUnfilterableFloat},
		{explicit.TextureFormatRGBA32Float, true, explicit.SampleTypeFloat},
		{explicit.TextureFormatR32Float, false, explicit.SampleTypeUnfilterableFloat},
		{explicit.TextureFormatR32Float, true, explicit.SampleTypeFloat},
		{explicit.TextureFormatR32Uint, true, explicit.SampleTypeUint},
		{explicit.TextureFormatRG32Uint, false, explicit.SampleTypeUint},
	}
	for _, tt := range features {
		if got := tt.format.SampleType(explicit.Features{Float32Filterable: tt.with}); got != tt.want {
			t.Errorf("format %d with float32-filterable %v samples as %d, want %d", tt.format, tt.with, got, tt.want)
		}
	}
}

func TestTextureKindsMipLevelsAndLayers(t *testing.T) {
	skipUnlessDebug(t)
	rgba := explicit.TextureFormatRGBA8Unorm
	tests := []struct {
		name string
		desc explicit.TextureDesc
		what string // "" means the description is valid
	}{
		{"a 2D texture", explicit.TextureDesc{Width: 16, Height: 8, Format: rgba, MipLevels: 5}, ""},
		{"a 2D texture with one level too many", explicit.TextureDesc{Width: 16, Height: 8, Format: rgba, MipLevels: 6}, "mip levels"},
		{"a 2D texture with layers", explicit.TextureDesc{Width: 16, Height: 8, Format: rgba, DepthOrLayers: 2}, "layers of a 2D texture"},
		{"an array of 256 layers", explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 4, Height: 4, DepthOrLayers: 256, Format: rgba}, ""},
		{"an array of 257 layers", explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 4, Height: 4, DepthOrLayers: 257, Format: rgba}, "texture array layers"},
		{"an array with mip levels", explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 8, Height: 8, DepthOrLayers: 3, Format: rgba, MipLevels: 4}, ""},
		{"a cube", explicit.TextureDesc{Kind: explicit.TextureKindCube, Width: 8, Height: 8, Format: rgba, MipLevels: 4}, ""},
		{"a cube with six layers", explicit.TextureDesc{Kind: explicit.TextureKindCube, Width: 8, Height: 8, DepthOrLayers: 6, Format: rgba}, ""},
		{"a cube with five layers", explicit.TextureDesc{Kind: explicit.TextureKindCube, Width: 8, Height: 8, DepthOrLayers: 5, Format: rgba}, "layers of a cube"},
		{"a cube that is not square", explicit.TextureDesc{Kind: explicit.TextureKindCube, Width: 8, Height: 4, Format: rgba}, "cube faces are square"},
		{"a 3D texture", explicit.TextureDesc{Kind: explicit.TextureKind3D, Width: 2048, Height: 2048, DepthOrLayers: 2048, Format: rgba, Usage: explicit.TextureSampled}, ""},
		{"a 3D texture too deep", explicit.TextureDesc{Kind: explicit.TextureKind3D, Width: 8, Height: 8, DepthOrLayers: 2049, Format: rgba}, "texture depth"},
		{"a 3D texture too wide", explicit.TextureDesc{Kind: explicit.TextureKind3D, Width: 2049, Height: 8, DepthOrLayers: 8, Format: rgba}, "texture width"},
		{"a 3D texture drawn into", explicit.TextureDesc{Kind: explicit.TextureKind3D, Width: 8, Height: 8, DepthOrLayers: 8, Format: rgba, Usage: explicit.TextureRenderTarget}, "a 3D texture cannot be drawn into"},
		{"a 3D depth texture", explicit.TextureDesc{Kind: explicit.TextureKind3D, Width: 8, Height: 8, DepthOrLayers: 8, Format: explicit.TextureFormatDepth32Float}, "a 3D texture has no depth format"},
		{"4 samples", explicit.TextureDesc{Width: 8, Height: 8, Format: rgba, SampleCount: 4, Usage: explicit.TextureRenderTarget}, ""},
		{"2 samples", explicit.TextureDesc{Width: 8, Height: 8, Format: rgba, SampleCount: 2, Usage: explicit.TextureRenderTarget}, "sample count"},
		{"8 samples", explicit.TextureDesc{Width: 8, Height: 8, Format: rgba, SampleCount: 8, Usage: explicit.TextureRenderTarget}, "sample count"},
		{"4 samples with mip levels", explicit.TextureDesc{Width: 8, Height: 8, Format: rgba, SampleCount: 4, MipLevels: 2, Usage: explicit.TextureRenderTarget}, "a multisampled texture is one level of a 2D texture"},
		{"4 samples on an array", explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 8, Height: 8, DepthOrLayers: 2, Format: rgba, SampleCount: 4, Usage: explicit.TextureRenderTarget}, "a multisampled texture is one level of a 2D texture"},
		{"4 samples that is not drawn into", explicit.TextureDesc{Width: 8, Height: 8, Format: rgba, SampleCount: 4, Usage: explicit.TextureSampled}, "a multisampled texture is drawn into"},
		{"4 samples that is also sampled", explicit.TextureDesc{Width: 8, Height: 8, Format: rgba, SampleCount: 4, Usage: explicit.TextureRenderTarget | explicit.TextureSampled}, "a multisampled texture is only drawn into and resolved"},
		{"a depth texture with 4 samples", explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatDepth32Float, SampleCount: 4, Usage: explicit.TextureRenderTarget}, ""},
		{"a storage texture", explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatR32Uint, Usage: explicit.TextureStorage | explicit.TextureSampled}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			do := func() { _, _ = d.NewTexture(tt.desc) }
			if tt.what == "" {
				if err := panicOf(do); err != nil {
					t.Fatalf("a valid description panicked: %v", err)
				}
				return
			}
			err := panicOf(do)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected a panic with *errors.Error, got %v", err)
			}
			switch appErr.Tag {
			case "GPUDescriptorInvalid":
				if appErr.Metadata["what"] != tt.what {
					t.Errorf("what = %v, want %q (route %s)", appErr.Metadata["what"], tt.what, appErr.Route())
				}
				if !appErr.HasRoute("Device.NewTexture->Limits.CheckTexture->invalid.GPUDescriptorInvalid") {
					t.Errorf("unexpected failure path: %s", appErr.Route())
				}
			case "GPULimitExceeded":
				if appErr.Metadata["limit"] != tt.what {
					t.Errorf("limit = %v, want %q (route %s)", appErr.Metadata["limit"], tt.what, appErr.Route())
				}
				if !appErr.HasRoute("Device.NewTexture->Limits.CheckTexture->exceeded.GPULimitExceeded") {
					t.Errorf("unexpected failure path: %s", appErr.Route())
				}
			default:
				t.Errorf("unexpected tag %s via %s", appErr.Tag, appErr.Route())
			}
		})
	}
}

func TestNormalizedDescription(t *testing.T) {
	tests := []struct {
		name          string
		desc          explicit.TextureDesc
		kind          explicit.TextureKind
		layers, depth int
	}{
		{"zero is a 2D texture", explicit.TextureDesc{}, explicit.TextureKind2D, 1, 1},
		{"a cube has 6 layers", explicit.TextureDesc{Kind: explicit.TextureKindCube}, explicit.TextureKindCube, 6, 1},
		{"an array keeps its layers", explicit.TextureDesc{Kind: explicit.TextureKind2DArray, DepthOrLayers: 7}, explicit.TextureKind2DArray, 7, 1},
		{"a volume keeps its depth", explicit.TextureDesc{Kind: explicit.TextureKind3D, DepthOrLayers: 9}, explicit.TextureKind3D, 1, 9},
	}
	for _, tt := range tests {
		n := tt.desc.Normalized()
		if n.Kind != tt.kind || tt.desc.Layers() != tt.layers || tt.desc.Depth() != tt.depth || n.MipLevels != 1 || n.SampleCount != 1 {
			t.Errorf("%s: %+v layers=%d depth=%d", tt.name, n, tt.desc.Layers(), tt.desc.Depth())
		}
	}
	if got := explicit.MipLevelsOf(1, 1); got != 1 {
		t.Errorf("MipLevelsOf(1, 1) = %d", got)
	}
	if got := explicit.MipLevelsOf(8192, 16); got != 14 {
		t.Errorf("MipLevelsOf(8192, 16) = %d", got)
	}
}

func colorTarget(t *testing.T, d *Device, f explicit.TextureFormat, samples int) explicit.Texture {
	t.Helper()
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 16, Height: 16, Format: f, SampleCount: samples, Usage: explicit.TextureRenderTarget})
	if err != nil {
		t.Fatal(err)
	}
	return tex
}

func TestRenderPassAttachments(t *testing.T) {
	skipUnlessDebug(t)
	rgba, f32 := explicit.TextureFormatRGBA8Unorm, explicit.TextureFormatRGBA32Float
	tests := []struct {
		name  string
		build func(t *testing.T, d *Device) explicit.PassDesc
		tag   string
		what  string // the metadata "limit" or "what"; "" means the pass is valid
	}{
		{"four color attachments", func(t *testing.T, d *Device) explicit.PassDesc {
			c := explicit.ColorAttachment{Target: colorTarget(t, d, rgba, 1)}
			return explicit.PassDesc{Color: []explicit.ColorAttachment{c, c, c, c}}
		}, "", ""},
		{"five color attachments", func(t *testing.T, d *Device) explicit.PassDesc {
			c := explicit.ColorAttachment{Target: colorTarget(t, d, rgba, 1)}
			return explicit.PassDesc{Color: []explicit.ColorAttachment{c, c, c, c, c}}
		}, "GPULimitExceeded", "color attachments"},
		{"no attachment", func(t *testing.T, d *Device) explicit.PassDesc { return explicit.PassDesc{} }, "GPUDescriptorInvalid", "a pass has an attachment"},
		{"two RGBA32Float attachments, 32 bytes", func(t *testing.T, d *Device) explicit.PassDesc {
			c := explicit.ColorAttachment{Target: colorTarget(t, d, f32, 1)}
			return explicit.PassDesc{Color: []explicit.ColorAttachment{c, c}}
		}, "", ""},
		{"three RGBA32Float attachments, 48 bytes", func(t *testing.T, d *Device) explicit.PassDesc {
			c := explicit.ColorAttachment{Target: colorTarget(t, d, f32, 1)}
			return explicit.PassDesc{Color: []explicit.ColorAttachment{c, c, c}}
		}, "GPULimitExceeded", "color attachment bytes per sample"},
		{"attachments of two sizes", func(t *testing.T, d *Device) explicit.PassDesc {
			small, _ := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 16, Format: rgba, Usage: explicit.TextureRenderTarget})
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: colorTarget(t, d, rgba, 1)}, {Target: small}}}
		}, "GPUDescriptorInvalid", "the attachments have one size"},
		{"attachments of two sample counts", func(t *testing.T, d *Device) explicit.PassDesc {
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: colorTarget(t, d, rgba, 1)}, {Target: colorTarget(t, d, rgba, 4)}}}
		}, "GPUDescriptorInvalid", "the attachments have one sample count"},
		{"a texture that is not a render target", func(t *testing.T, d *Device) explicit.PassDesc {
			tex, _ := d.NewTexture(explicit.TextureDesc{Width: 16, Height: 16, Format: rgba, Usage: explicit.TextureSampled})
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: tex}}}
		}, "GPUDescriptorInvalid", "a color attachment is a render target"},
		{"a depth format as a color attachment", func(t *testing.T, d *Device) explicit.PassDesc {
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: colorTarget(t, d, explicit.TextureFormatDepth32Float, 1)}}}
		}, "GPUDescriptorInvalid", "a color attachment has a color format"},
		{"a color format as the depth attachment", func(t *testing.T, d *Device) explicit.PassDesc {
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: colorTarget(t, d, rgba, 1)}}, Depth: &explicit.DepthAttachment{Target: colorTarget(t, d, rgba, 1)}}
		}, "GPUDescriptorInvalid", "the depth attachment has a depth format"},
		{"a depth attachment of the size of the color", func(t *testing.T, d *Device) explicit.PassDesc {
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: colorTarget(t, d, rgba, 1)}}, Depth: &explicit.DepthAttachment{Target: colorTarget(t, d, explicit.TextureFormatDepth32Float, 1)}}
		}, "", ""},
		{"a depth-only pass", func(t *testing.T, d *Device) explicit.PassDesc {
			return explicit.PassDesc{Depth: &explicit.DepthAttachment{Target: colorTarget(t, d, explicit.TextureFormatDepth24PlusStencil8, 1)}}
		}, "", ""},
		{"a depth attachment with another sample count", func(t *testing.T, d *Device) explicit.PassDesc {
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: colorTarget(t, d, rgba, 4)}}, Depth: &explicit.DepthAttachment{Target: colorTarget(t, d, explicit.TextureFormatDepth32Float, 1)}}
		}, "GPUDescriptorInvalid", "the attachments have one sample count"},
		{"a resolve target of the right size", func(t *testing.T, d *Device) explicit.PassDesc {
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: colorTarget(t, d, rgba, 4), Resolve: colorTarget(t, d, rgba, 1)}}}
		}, "", ""},
		{"a resolve target of the wrong size", func(t *testing.T, d *Device) explicit.PassDesc {
			small, _ := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 16, Format: rgba, Usage: explicit.TextureRenderTarget})
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: colorTarget(t, d, rgba, 4), Resolve: small}}}
		}, "GPUDescriptorInvalid", "a resolve target has the size of its attachment"},
		{"a resolve target of the wrong format", func(t *testing.T, d *Device) explicit.PassDesc {
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: colorTarget(t, d, rgba, 4), Resolve: colorTarget(t, d, explicit.TextureFormatBGRA8Unorm, 1)}}}
		}, "GPUDescriptorInvalid", "a resolve target has one sample and the format of its attachment"},
		{"a resolve target that is multisampled", func(t *testing.T, d *Device) explicit.PassDesc {
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: colorTarget(t, d, rgba, 4), Resolve: colorTarget(t, d, rgba, 4)}}}
		}, "GPUDescriptorInvalid", "a resolve target has one sample and the format of its attachment"},
		{"a resolve target for an attachment of one sample", func(t *testing.T, d *Device) explicit.PassDesc {
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: colorTarget(t, d, rgba, 1), Resolve: colorTarget(t, d, rgba, 1)}}}
		}, "GPUDescriptorInvalid", "only a multisampled attachment is resolved"},
		{"an attachment that is a whole array", func(t *testing.T, d *Device) explicit.PassDesc {
			tex, _ := d.NewTexture(explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 16, Height: 16, DepthOrLayers: 3, Format: rgba, Usage: explicit.TextureRenderTarget})
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: tex}}}
		}, "GPUDescriptorInvalid", "a color attachment is a 2D texture (use Layer for a layer of an array or a cube)"},
		{"an attachment that is a layer of an array", func(t *testing.T, d *Device) explicit.PassDesc {
			tex, _ := d.NewTexture(explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 16, Height: 16, DepthOrLayers: 3, Format: rgba, Usage: explicit.TextureRenderTarget})
			return explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: tex.Layer(0, 2)}}}
		}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			desc := tt.build(t, d)
			enc, _ := d.Begin()
			do := func() { enc.BeginRenderPass(desc) }
			if tt.tag == "" {
				if err := panicOf(do); err != nil {
					t.Fatalf("a valid pass panicked: %v", err)
				}
				return
			}
			appErr := expectInvalid(t, tt.tag, "encoder.BeginRenderPass->Limits.CheckPass", do)
			key := "what"
			if tt.tag == "GPULimitExceeded" {
				key = "limit"
			}
			if appErr.Metadata[key] != tt.what {
				t.Errorf("%s = %v, want %q (route %s)", key, appErr.Metadata[key], tt.what, appErr.Route())
			}
		})
	}
}

func TestPipelineState(t *testing.T) {
	skipUnlessDebug(t)
	rgba := explicit.TextureFormatRGBA8Unorm
	tests := []struct {
		name string
		desc explicit.PipelineDesc
		tag  string
		what string
	}{
		{"the 2D renderer's pipeline", explicit.PipelineDesc{Target: rgba}, "", ""},
		{"four color targets", explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: rgba}, {Format: rgba}, {Format: rgba}, {Format: rgba}}}, "", ""},
		{"five color targets", explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: rgba}, {Format: rgba}, {Format: rgba}, {Format: rgba}, {Format: rgba}}}, "GPULimitExceeded", "color targets"},
		{"three RGBA32Float targets", explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: explicit.TextureFormatRGBA32Float}, {Format: explicit.TextureFormatRGBA32Float}, {Format: explicit.TextureFormatRGBA32Float}}}, "GPULimitExceeded", "color target bytes per sample"},
		{"a depth format as a color target", explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: explicit.TextureFormatDepth32Float}}}, "GPUDescriptorInvalid", "a color target has a renderable color format"},
		{"Target and ColorTargets together", explicit.PipelineDesc{Target: rgba, ColorTargets: []explicit.ColorTargetState{{Format: rgba}}}, "GPUDescriptorInvalid", "ColorTargets replace Target and Blend"},
		{"every topology", explicit.PipelineDesc{Target: rgba, Topology: explicit.TopologyPointList}, "", ""},
		{"an unknown topology", explicit.PipelineDesc{Target: rgba, Topology: 9}, "GPUDescriptorInvalid", "topology"},
		{"4 samples", explicit.PipelineDesc{Target: rgba, SampleCount: 4}, "", ""},
		{"3 samples", explicit.PipelineDesc{Target: rgba, SampleCount: 3}, "GPUDescriptorInvalid", "sample count"},
		{"depth test and write", explicit.PipelineDesc{Target: rgba, DepthStencil: explicit.DepthStencilState{Format: explicit.TextureFormatDepth32Float, DepthWrite: true, DepthCompare: explicit.CompareGreaterEqual}}, "", ""},
		{"a depth state on a color format", explicit.PipelineDesc{Target: rgba, DepthStencil: explicit.DepthStencilState{Format: rgba}}, "GPUDescriptorInvalid", "the depth-stencil state has a depth format"},
		{"depth write without a depth format", explicit.PipelineDesc{Target: rgba, DepthStencil: explicit.DepthStencilState{DepthWrite: true}}, "GPUDescriptorInvalid", "a depth state needs a depth format"},
		{"a depth bias on triangles", explicit.PipelineDesc{Target: rgba, DepthStencil: explicit.DepthStencilState{Format: explicit.TextureFormatDepth32Float, DepthBias: 2, DepthBiasSlopeScale: 1}}, "", ""},
		{"a depth bias on lines", explicit.PipelineDesc{Target: rgba, Topology: explicit.TopologyLineList, DepthStencil: explicit.DepthStencilState{Format: explicit.TextureFormatDepth32Float, DepthBias: 2}}, "GPUDescriptorInvalid", "only triangles have a depth bias"},
		{"a vertex format that fits", explicit.PipelineDesc{Target: rgba, VertexBuffers: []explicit.VertexBufferLayout{{Stride: 12, Attributes: []explicit.VertexAttribute{{Location: 0, Format: explicit.VertexFloat32x3}}}}}, "", ""},
		{"a vertex attribute past the stride", explicit.PipelineDesc{Target: rgba, VertexBuffers: []explicit.VertexBufferLayout{{Stride: 12, Attributes: []explicit.VertexAttribute{{Location: 0, Offset: 4, Format: explicit.VertexFloat32x3}}}}}, "GPUDescriptorInvalid", "vertex attribute inside the stride"},
		{"an unspecified vertex format", explicit.PipelineDesc{Target: rgba, VertexBuffers: []explicit.VertexBufferLayout{{Stride: 12, Attributes: []explicit.VertexAttribute{{Location: 0}}}}}, "GPUDescriptorInvalid", "vertex format"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			do := func() { _, _ = d.NewPipeline(tt.desc) }
			if tt.tag == "" {
				if err := panicOf(do); err != nil {
					t.Fatalf("a valid pipeline panicked: %v", err)
				}
				return
			}
			appErr := expectInvalid(t, tt.tag, "Device.NewPipeline->Limits.CheckPipeline", do)
			key := "what"
			if tt.tag == "GPULimitExceeded" {
				key = "limit"
			}
			if appErr.Metadata[key] != tt.what {
				t.Errorf("%s = %v, want %q (route %s)", key, appErr.Metadata[key], tt.what, appErr.Route())
			}
		})
	}
}

func TestLayersOfArraysCubesAndVolumes(t *testing.T) {
	skipUnlessDebug(t)
	d := New()
	rgba := explicit.TextureFormatRGBA8Unorm
	cube, _ := d.NewTexture(explicit.TextureDesc{Kind: explicit.TextureKindCube, Width: 8, Height: 8, Format: rgba, MipLevels: 4, Usage: explicit.TextureSampled | explicit.TextureRenderTarget})
	array, _ := d.NewTexture(explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 8, Height: 8, DepthOrLayers: 3, Format: rgba, Usage: explicit.TextureRenderTarget})
	volume, _ := d.NewTexture(explicit.TextureDesc{Kind: explicit.TextureKind3D, Width: 8, Height: 8, DepthOrLayers: 4, Format: rgba, Usage: explicit.TextureSampled})
	flat, _ := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: rgba})

	face := cube.Layer(2, 5)
	if got := face.Desc(); got.Kind != explicit.TextureKind2D || got.Width != 2 || got.Height != 2 || got.MipLevels != 1 {
		t.Errorf("the face of a cube at level 2: %+v", got)
	}
	if cube.Layer(2, 5) != face {
		t.Error("Layer does not return the same view twice")
	}
	if flat.Layer(0, 0) != flat {
		t.Error("the layer of a flat texture is the texture")
	}
	for _, tt := range []struct {
		name string
		do   func()
		what string
	}{
		{"a layer past the array", func() { array.Layer(0, 3) }, "a layer the texture has"},
		{"a negative layer", func() { array.Layer(0, -1) }, "a layer the texture has"},
		{"the seventh face", func() { cube.Layer(0, 6) }, "a layer the texture has"},
		{"a level past the chain", func() { cube.Layer(4, 0) }, "layer of a mip level the texture has"},
		{"a layer of a volume", func() { volume.Layer(0, 0) }, "a 3D texture has no layers"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			appErr := expectInvalid(t, "GPUDescriptorInvalid", "texture.Layer->Limits.CheckLayer->invalid.GPUDescriptorInvalid", tt.do)
			if appErr.Metadata["what"] != tt.what {
				t.Errorf("what = %v, want %q", appErr.Metadata["what"], tt.what)
			}
		})
	}
}

func TestWriteAndReadTheRegionsOfLayersAndVolumes(t *testing.T) {
	d := New()
	rgba := explicit.TextureFormatRGBA8Unorm
	array, _ := d.NewTexture(explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 2, Height: 2, DepthOrLayers: 3, Format: rgba, Usage: explicit.TextureSampled | explicit.TextureCopySource})
	layers := []byte{
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
	}
	d.WriteTextureRegion(array, explicit.TextureRegion{Origin: [3]int{0, 0, 1}, Size: [3]int{2, 2, 2}}, layers, 8, 2)
	enc, _ := d.Begin()
	var got [3][]byte
	for i := range got {
		enc.ReadTextureBytes(array.Layer(0, i), image2x2, func(data []byte, rowBytes int, err error) {
			if err != nil || rowBytes != 8 {
				t.Errorf("layer read: rowBytes=%d err=%v", rowBytes, err)
			}
			got[i] = data
		})
	}
	_ = enc.Submit()
	d.CompleteReads()
	for i, want := range []byte{0, 1, 2} {
		if len(got[i]) != 16 || got[i][0] != want || got[i][15] != want {
			t.Errorf("layer %d reads %v, want all %d", i, got[i], want)
		}
	}
}

func TestClearingWritesTheFormatsBytes(t *testing.T) {
	d := New()
	tests := []struct {
		format explicit.TextureFormat
		clear  [4]float64
		want   []byte
	}{
		{explicit.TextureFormatRGBA8Unorm, [4]float64{1, 0.5, 0, 1}, []byte{255, 128, 0, 255}},
		{explicit.TextureFormatBGRA8Unorm, [4]float64{1, 0.5, 0, 1}, []byte{0, 128, 255, 255}},
		{explicit.TextureFormatR32Uint, [4]float64{7, 0, 0, 0}, []byte{7, 0, 0, 0}},
		{explicit.TextureFormatRG32Uint, [4]float64{1, 2, 0, 0}, []byte{1, 0, 0, 0, 2, 0, 0, 0}},
		{explicit.TextureFormatR32Float, [4]float64{1, 0, 0, 0}, []byte{0, 0, 0x80, 0x3f}},
		{explicit.TextureFormatRGBA16Float, [4]float64{1, 0.5, 2, -1}, []byte{0, 0x3c, 0, 0x38, 0, 0x40, 0, 0xbc}},
		{explicit.TextureFormatRGBA32Float, [4]float64{1, 0, 0, 0}, []byte{0, 0, 0x80, 0x3f, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	}
	for _, tt := range tests {
		tex, _ := d.NewTexture(explicit.TextureDesc{Width: 2, Height: 1, Format: tt.format, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
		enc, _ := d.Begin()
		enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: tex, Clear: tt.clear}}}).End()
		var got []byte
		enc.ReadTextureBytes(tex, image.Rect(0, 0, 1, 1), func(data []byte, _ int, _ error) { got = data })
		_ = enc.Submit()
		d.CompleteReads()
		if string(got) != string(tt.want) {
			t.Errorf("format %d cleared to %v reads %v, want %v", tt.format, tt.clear, got, tt.want)
		}
	}
}

func TestThePipelineAndPassStateIsRecorded(t *testing.T) {
	d := New()
	rgba := explicit.TextureFormatRGBA8Unorm
	_, _ = d.NewPipeline(explicit.PipelineDesc{Target: rgba})
	_, _ = d.NewPipeline(explicit.PipelineDesc{
		ColorTargets: []explicit.ColorTargetState{{Format: rgba}, {Format: explicit.TextureFormatR32Uint}},
		Topology:     explicit.TopologyLineStrip, CullMode: explicit.CullBack, FrontFace: explicit.FrontFaceCW, SampleCount: 4,
		DepthStencil: explicit.DepthStencilState{Format: explicit.TextureFormatDepth32Float, DepthWrite: true, DepthCompare: explicit.CompareLess},
	})
	calls := d.Calls()
	if got := calls[0].Args; got != "shader=<nil> vertex= fragment= buffers=0 target=1" {
		t.Errorf("the 2D pipeline is logged as %q", got)
	}
	for _, want := range []string{"targets=", "topology=3", "cull=2", "frontface=1", "depthstencil=", "samples=4"} {
		if !contains(calls[1].Args, want) {
			t.Errorf("the 3D pipeline's log %q lacks %q", calls[1].Args, want)
		}
	}
	tex := colorTarget(t, d, rgba, 1)
	depth := colorTarget(t, d, explicit.TextureFormatDepth32Float, 1)
	enc, _ := d.Begin()
	pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: tex, Store: explicit.StoreDiscard}}, Depth: &explicit.DepthAttachment{Target: depth, ClearDepth: 1}})
	pass.SetStencilReference(5)
	pass.End()
	ops := d.Ops()
	if want := []string{"BeginRenderPass", "SetStencilReference", "EndPass"}; !reflectEqual(ops[len(ops)-3:], want) {
		t.Errorf("the last calls are %v, want %v", ops[len(ops)-3:], want)
	}
	if got := d.Calls()[len(ops)-3].Args; !contains(got, "store=1") || !contains(got, "depth={") {
		t.Errorf("the pass is logged as %q", got)
	}
}
