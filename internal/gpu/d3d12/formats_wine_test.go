//go:build windows && glutter_wine

package d3d12

import (
	"image"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
	"github.com/rm4n0s/glutter/ui"
)

// The tests of step 30.1 for what the scenes (scenes_wine_test.go) do not reach: the descriptions of
// the textures of every kind, the views, the checks of a bind group, and the tags of the new errors.

func TestWineTexturesOfEveryKindAreMadeAndDescribed(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	rgba := explicit.TextureFormatRGBA8Unorm
	use := explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource
	tests := []struct {
		name       string
		desc       explicit.TextureDesc
		layers     int
		subs       int // the subresources of the resource: mips × layers
		viewLevelW int // the width of Level(1), 0 for none
	}{
		{"2D", explicit.TextureDesc{Width: 16, Height: 8, Format: rgba, MipLevels: 3, Usage: use}, 1, 3, 8},
		{"2D array", explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 16, Height: 16, DepthOrLayers: 5, Format: rgba, MipLevels: 3, Usage: use}, 5, 15, 8},
		{"cube", explicit.TextureDesc{Kind: explicit.TextureKindCube, Width: 16, Height: 16, Format: rgba, MipLevels: 3, Usage: use}, 6, 18, 8},
		{"3D", explicit.TextureDesc{Kind: explicit.TextureKind3D, Width: 16, Height: 16, DepthOrLayers: 8, Format: rgba, MipLevels: 3, Usage: explicit.TextureSampled}, 1, 3, 8},
		{"depth32float", explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatDepth32Float, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource}, 1, 1, 0},
		{"depth24plus-stencil8", explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatDepth24PlusStencil8, Usage: explicit.TextureRenderTarget}, 1, 1, 0},
		{"multisampled", explicit.TextureDesc{Width: 8, Height: 8, Format: rgba, SampleCount: 4, Usage: explicit.TextureRenderTarget}, 1, 1, 0},
		{"storage", explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatR32Uint, Usage: explicit.TextureStorage | explicit.TextureSampled}, 1, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tex, err := d.NewTexture(tt.desc)
			if err != nil {
				t.Fatal(err)
			}
			if got := tex.Desc(); got != tt.desc.Normalized() {
				t.Errorf("Desc = %+v, want %+v", got, tt.desc.Normalized())
			}
			dt := tex.(*texture)
			if dt.desc.Layers() != tt.layers || dt.subresourceCount() != tt.subs {
				t.Errorf("layers = %d and subresources = %d, want %d and %d", dt.desc.Layers(), dt.subresourceCount(), tt.layers, tt.subs)
			}
			if tt.viewLevelW > 0 {
				lv := tex.Level(1)
				if lv.Desc().Width != tt.viewLevelW || lv.Desc().MipLevels != 1 || lv.Desc().Kind != tex.Desc().Kind {
					t.Errorf("Level(1) = %+v", lv.Desc())
				}
				if lv.(*texture).sub() != 1 {
					t.Errorf("the first subresource of Level(1) is %d", lv.(*texture).sub())
				}
			}
			tex.Destroy()
		})
	}
}

func TestWineLayersAreViewsOfTheSameResource(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	rgba := explicit.TextureFormatRGBA8Unorm
	array, err := d.NewTexture(explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 8, Height: 8, DepthOrLayers: 3, Format: rgba, MipLevels: 2,
		Usage: explicit.TextureSampled | explicit.TextureRenderTarget})
	if err != nil {
		t.Fatal(err)
	}
	defer array.Destroy()
	l := array.Layer(1, 2).(*texture)
	a := array.(*texture)
	if l.res != a.res || l.base != 1 || l.baseLayer != 2 || l.desc.Width != 4 || l.desc.Kind != explicit.TextureKind2D || l.sub() != 1+2*2 {
		t.Errorf("the layer is %+v of the resource, base %d, layer %d, subresource %d", l.desc, l.base, l.baseLayer, l.sub())
	}
	if array.Layer(1, 2) != array.Layer(1, 2) {
		t.Error("the view of a layer is made once")
	}
	if l.rtv < 0 || l.srv >= 0 {
		t.Errorf("a layer has a view to draw into (%d) and none to sample (%d)", l.rtv, l.srv)
	}
	appErr := expectPanic(t, func() { array.Level(1).Layer(0, 0) })
	if appErr.Tag != tagD3D12LevelInvalid || !appErr.HasRoute("texture.Layer.D3D12LevelInvalid") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
}

func TestWineATextureBoundToTheWrongBindingFails(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
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
	defer cube.Destroy()
	defer array.Destroy()
	defer uint2D.Destroy()
	// A device that cannot filter float32 textures, whatever this adapter can.
	was := d.features.Float32Filterable
	d.features.Float32Filterable = false
	t.Cleanup(func() { d.features.Float32Filterable = was })
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
			if !ok || appErr.Tag != tagD3D12TextureBindingMismatch || !appErr.HasRoute("Device.NewBindGroup->Device.makeBindGroup.D3D12TextureBindingMismatch") {
				t.Fatalf("got %v, want D3D12TextureBindingMismatch via Device.NewBindGroup", err)
			}
		})
	}
}

func TestWineALayerCannotBeSampled(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	rgba := explicit.TextureFormatRGBA8Unorm
	array, err := d.NewTexture(explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 8, Height: 8, DepthOrLayers: 2, Format: rgba,
		Usage: explicit.TextureSampled | explicit.TextureRenderTarget})
	if err != nil {
		t.Fatal(err)
	}
	defer array.Destroy()
	p, err := d.NewPipeline(explicit.PipelineDesc{Shader: devicetest.ShaderSample, VertexEntry: "vs_uv", FragmentEntry: "fs_array",
		ColorTargets: []explicit.ColorTargetState{{Format: rgba}}})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Destroy()
	_, err = d.NewBindGroup(p, 1, []explicit.Binding{{Binding: 1, Texture: array.Layer(0, 1)}})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagD3D12BindGroupInvalid || !appErr.HasRoute("Device.NewBindGroup->Device.makeBindGroup.D3D12BindGroupInvalid") {
		t.Fatalf("got %v, want D3D12BindGroupInvalid via Device.NewBindGroup", err)
	}
}

func TestWineWriteTextureRegionErrors(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	other, _ := openTestDevice(t, Options{})
	rgba := explicit.TextureFormatRGBA8Unorm
	array, err := d.NewTexture(explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 4, Height: 4, DepthOrLayers: 2, Format: rgba, Usage: explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	defer array.Destroy()
	foreign, err := other.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: rgba, Usage: explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	defer foreign.Destroy()
	data := make([]byte, 4*4*4*2)
	region := func(level int, o, size [3]int) explicit.TextureRegion {
		return explicit.TextureRegion{Level: level, Origin: o, Size: size}
	}
	tests := []struct {
		name string
		tex  explicit.Texture
		r    explicit.TextureRegion
		data []byte
		tag  string
		path string
	}{
		{"past the layers", array, region(0, [3]int{0, 0, 1}, [3]int{4, 4, 2}), data, tagD3D12WriteOutOfRange, "Device.WriteTextureRegion.D3D12WriteOutOfRange"},
		{"past the width", array, region(0, [3]int{1, 0, 0}, [3]int{4, 4, 1}), data, tagD3D12WriteOutOfRange, "Device.WriteTextureRegion.D3D12WriteOutOfRange"},
		{"a level the texture lacks", array, region(1, [3]int{0, 0, 0}, [3]int{1, 1, 1}), data, tagD3D12LevelInvalid, "Device.WriteTextureRegion.D3D12LevelInvalid"},
		{"too little data", array, region(0, [3]int{0, 0, 0}, [3]int{4, 4, 2}), data[:50], tagD3D12WriteOutOfRange, "Device.WriteTextureRegion.D3D12WriteOutOfRange"},
		{"a texture of another device", foreign, region(0, [3]int{0, 0, 0}, [3]int{1, 1, 1}), data, tagD3D12ForeignResource, "Device.WriteTextureRegion->Device.own2texture.D3D12ForeignResource"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := expectPanic(t, func() { d.WriteTextureRegion(tt.tex, tt.r, tt.data, 16, 4) })
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.path) {
				t.Errorf("%s via %s", appErr.Tag, appErr.Route())
			}
		})
	}
}

func TestWineReadTextureBytesErrors(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	defer tex.Destroy()
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	appErr := expectPanic(t, func() {
		e.ReadTextureBytes(tex, image.Rect(0, 0, 5, 1), func([]byte, int, error) {})
	})
	if appErr.Tag != tagD3D12RectOutOfRange || !appErr.HasRoute("encoder.ReadTextureBytes.D3D12RectOutOfRange") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
	pass := e.BeginPass(tex, explicit.LoadClear, ui.Color{})
	appErr = expectPanic(t, func() { e.ReadTextureBytes(tex, image.Rect(0, 0, 1, 1), func([]byte, int, error) {}) })
	if appErr.Tag != tagD3D12PassNotEnded || !appErr.HasRoute("encoder.ReadTextureBytes.D3D12PassNotEnded") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
	pass.End()
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
}

func TestWineAPassNeedsItsAttachmentsToBeRenderTargets(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	rgba := explicit.TextureFormatRGBA8Unorm
	plain, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: rgba, Usage: explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Destroy()
	depth, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatDepth32Float, Usage: explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	defer depth.Destroy()
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	colored, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: rgba, Usage: explicit.TextureRenderTarget})
	if err != nil {
		t.Fatal(err)
	}
	defer colored.Destroy()
	for name, desc := range map[string]explicit.PassDesc{
		"a depth attachment that is not a render target": {Color: []explicit.ColorAttachment{{Target: colored}}, Depth: &explicit.DepthAttachment{Target: depth}},
		"a color attachment that is not a render target": {Color: []explicit.ColorAttachment{{Target: plain}}},
		"no attachment": {},
	} {
		appErr := expectPanic(t, func() { e.BeginRenderPass(desc) })
		if appErr.Tag != tagD3D12TextureInvalid || !appErr.HasRoute("encoder.BeginRenderPass.D3D12TextureInvalid") {
			t.Errorf("%s: %s via %s", name, appErr.Tag, appErr.Route())
		}
	}
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
}
