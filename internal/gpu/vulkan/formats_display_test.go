//go:build linux && glutter_display

package vulkan

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
)

func TestFeaturesOfLavapipe(t *testing.T) {
	d := openLavapipe(t)
	if !d.Features().Float32Filterable {
		t.Error("lavapipe filters float32 textures")
	}
	if !d.independentBlend {
		t.Error("lavapipe has independentBlend")
	}
	if d.depthStencil != formatD24UnormS8Uint && d.depthStencil != formatD32SfloatS8Uint {
		t.Errorf("the depth-stencil format is %d", d.depthStencil)
	}
}

func TestTexturesOfEveryKindAreMadeAndDescribed(t *testing.T) {
	d := openLavapipe(t)
	rgba := explicit.TextureFormatRGBA8Unorm
	use := explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource
	tests := []struct {
		name         string
		desc         explicit.TextureDesc
		layers       int
		viewLevelW   int // the width of Level(1), 0 for none
		wantViewType uint32
	}{
		{"2D", explicit.TextureDesc{Width: 16, Height: 8, Format: rgba, MipLevels: 3, Usage: use}, 1, 8, imageViewType2D},
		{"2D array", explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 16, Height: 16, DepthOrLayers: 5, Format: rgba, MipLevels: 3, Usage: use}, 5, 8, imageViewType2DArray},
		{"cube", explicit.TextureDesc{Kind: explicit.TextureKindCube, Width: 16, Height: 16, Format: rgba, MipLevels: 3, Usage: use}, 6, 8, imageViewTypeCube},
		{"3D", explicit.TextureDesc{Kind: explicit.TextureKind3D, Width: 16, Height: 16, DepthOrLayers: 8, Format: rgba, MipLevels: 3, Usage: explicit.TextureSampled}, 1, 8, imageViewType3D},
		{"depth32float", explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatDepth32Float, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource}, 1, 0, imageViewType2D},
		{"depth24plus-stencil8", explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatDepth24PlusStencil8, Usage: explicit.TextureRenderTarget}, 1, 0, imageViewType2D},
		{"multisampled", explicit.TextureDesc{Width: 8, Height: 8, Format: rgba, SampleCount: 4, Usage: explicit.TextureRenderTarget}, 1, 0, imageViewType2D},
		{"storage", explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatR32Uint, Usage: explicit.TextureStorage | explicit.TextureSampled}, 1, 0, imageViewType2D},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tex, err := d.NewTexture(tt.desc)
			if err != nil {
				t.Fatal(err)
			}
			defer tex.Destroy()
			got := tex.Desc()
			if got != tt.desc.Normalized() {
				t.Errorf("Desc = %+v, want %+v", got, tt.desc.Normalized())
			}
			vt := tex.(*texture)
			if vt.desc.Layers() != tt.layers {
				t.Errorf("layers = %d, want %d", vt.desc.Layers(), tt.layers)
			}
			if tt.viewLevelW > 0 {
				lv := tex.Level(1)
				if lv.Desc().Width != tt.viewLevelW || lv.Desc().MipLevels != 1 || lv.Desc().Kind != got.Kind {
					t.Errorf("Level(1) = %+v", lv.Desc())
				}
			}
		})
	}
}

func TestATextureBoundToTheWrongBindingFails(t *testing.T) {
	d := openLavapipe(t)
	rgba := explicit.TextureFormatRGBA8Unorm
	use := explicit.TextureSampled | explicit.TextureRenderTarget
	flat, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: rgba, Usage: use})
	if err != nil {
		t.Fatal(err)
	}
	defer flat.Destroy()
	float32s, err := d.NewTexture(explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 8, Height: 8, DepthOrLayers: 2, Format: explicit.TextureFormatRGBA32Float, Usage: use})
	if err != nil {
		t.Fatal(err)
	}
	defer float32s.Destroy()
	ints, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatR32Uint, Usage: use})
	if err != nil {
		t.Fatal(err)
	}
	defer ints.Destroy()
	smp, _ := d.NewSampler(explicit.SamplerDesc{})
	defer smp.Destroy()
	pipeline := func(fragment string) explicit.Pipeline {
		p, err := d.NewPipeline(explicit.PipelineDesc{Shader: devicetest.ShaderSample, VertexEntry: "vs_uv", FragmentEntry: fragment,
			ColorTargets: []explicit.ColorTargetState{{Format: rgba}}})
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	cube, array, uint2D := pipeline("fs_cube"), pipeline("fs_array"), pipeline("fs_uint")
	// A device that cannot filter float32 textures; lavapipe can.
	d.features.Float32Filterable = false
	t.Cleanup(func() { d.features.Float32Filterable = true })
	tests := []struct {
		name    string
		p       explicit.Pipeline
		binding int
		tex     explicit.Texture
	}{
		{"a 2D texture where the shader reads a cube", cube, 0, flat},
		{"a float32 array where the shader filters floats", array, 1, float32s},
		{"a float texture where the shader reads integers", uint2D, 4, flat},
		{"an integer texture where the shader reads a cube", cube, 0, ints},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := d.NewBindGroup(tt.p, 1, []explicit.Binding{{Binding: tt.binding, Texture: tt.tex}})
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tagVulkanTextureBindingMismatch || !appErr.HasRoute("Device.NewBindGroup.VulkanTextureBindingMismatch") {
				t.Fatalf("got %v, want VulkanTextureBindingMismatch via Device.NewBindGroup", err)
			}
		})
	}
}

func TestDifferentBlendStatesNeedIndependentBlend(t *testing.T) {
	d := openLavapipe(t)
	rgba := explicit.TextureFormatRGBA8Unorm
	desc := explicit.PipelineDesc{Shader: devicetest.ShaderFill, VertexEntry: "vs_full", FragmentEntry: "fs_four",
		ColorTargets: []explicit.ColorTargetState{{Format: rgba}, {Format: rgba, Blend: explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendSrcAlpha}},
			{Format: explicit.TextureFormatR32Uint}, {Format: explicit.TextureFormatRGBA16Float}}}
	d.independentBlend = false
	t.Cleanup(func() { d.independentBlend = true })
	_, err := d.NewPipeline(desc)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagVulkanPipelineInvalid || !appErr.HasRoute("Device.NewPipeline.VulkanPipelineInvalid") {
		t.Fatalf("got %v, want VulkanPipelineInvalid via Device.NewPipeline", err)
	}
	same := explicit.PipelineDesc{Shader: devicetest.ShaderFill, VertexEntry: "vs_full", FragmentEntry: "fs_float", ColorTargets: []explicit.ColorTargetState{{Format: rgba}, {Format: rgba}}}
	if p, err := d.NewPipeline(same); err != nil {
		t.Errorf("two targets with the same state: %v", err)
	} else {
		p.Destroy()
	}
}

func TestLayersAreViewsOfTheSameImage(t *testing.T) {
	d := openLavapipe(t)
	rgba := explicit.TextureFormatRGBA8Unorm
	array, err := d.NewTexture(explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 8, Height: 8, DepthOrLayers: 3, Format: rgba, MipLevels: 2,
		Usage: explicit.TextureSampled | explicit.TextureRenderTarget})
	if err != nil {
		t.Fatal(err)
	}
	defer array.Destroy()
	l := array.Layer(1, 2).(*texture)
	a := array.(*texture)
	if l.image != a.image || l.base != 1 || l.baseLayer != 2 || l.desc.Width != 4 || l.desc.Kind != explicit.TextureKind2D {
		t.Errorf("the layer is %+v of the image %d", l.desc, l.image)
	}
	if array.Layer(1, 2) != array.Layer(1, 2) {
		t.Error("the view of a layer is made once")
	}
}
