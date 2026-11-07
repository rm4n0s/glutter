package viewport3d_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func TestNewPBRMaterialDefaults(t *testing.T) {
	m := viewport3d.NewPBRMaterial()
	if m.Color() != ui.NewColor(0xFFFFFFFF) || m.Metallic() != 0 || m.Roughness() != 0.5 || m.AlphaMode() != viewport3d.AlphaModeOpaque || m.DoubleSided() || m.Transparent() {
		t.Errorf("the defaults are %v %v %v %v %v %v", m.Color(), m.Metallic(), m.Roughness(), m.AlphaMode(), m.DoubleSided(), m.Transparent())
	}
	if m.Emissive() != [3]float64{} || m.Texture(viewport3d.PBRSlotNormal).Texture != nil {
		t.Errorf("emissive %v, normal texture %v", m.Emissive(), m.Texture(viewport3d.PBRSlotNormal))
	}
	// A color that is not opaque is not a transparent material in the opaque mode (glTF ignores the alpha there): only the blend mode is.
	if viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: ui.NewColor(0x80FF0000)}).Transparent() {
		t.Error("a half transparent color in the opaque mode is transparent")
	}
	if !viewport3d.NewPBRMaterial(viewport3d.PBROpts{AlphaMode: viewport3d.AlphaModeBlend}).Transparent() {
		t.Error("the blend mode is not transparent")
	}
	if got := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Metallic: foundation.Some(1.0), Roughness: foundation.Some(0.0)}); got.Metallic() != 1 || got.Roughness() != 0 {
		t.Errorf("explicit values: %v %v", got.Metallic(), got.Roughness())
	}
}

func TestPBRMaterialSettersNotifyOnChange(t *testing.T) {
	m := viewport3d.NewPBRMaterial()
	calls := 0
	m.AddListener(func() { calls++ })
	other := viewport3d.PBRTexture{TexCoord: 1}
	tests := []struct {
		name string
		do   func()
		want int
	}{
		{"a color", func() { m.SetColor(ui.NewColor(0xFF112233)) }, 1},
		{"the same color", func() { m.SetColor(ui.NewColor(0xFF112233)) }, 0},
		{"metallic", func() { m.SetMetallic(0.5) }, 1},
		{"the same metallic", func() { m.SetMetallic(0.5) }, 0},
		{"roughness", func() { m.SetRoughness(0.1) }, 1},
		{"the same roughness", func() { m.SetRoughness(0.1) }, 0},
		{"emissive", func() { m.SetEmissive([3]float64{1, 1, 1}) }, 1},
		{"the same emissive", func() { m.SetEmissive([3]float64{1, 1, 1}) }, 0},
		{"a texture", func() { m.SetTexture(viewport3d.PBRSlotEmissive, other) }, 1},
		{"the same texture", func() { m.SetTexture(viewport3d.PBRSlotEmissive, other) }, 0},
		{"an alpha mode", func() { m.SetAlphaMode(viewport3d.AlphaModeMask) }, 1},
		{"the same alpha mode", func() { m.SetAlphaMode(viewport3d.AlphaModeMask) }, 0},
		{"double sided", func() { m.SetDoubleSided(true) }, 1},
		{"the same double sided", func() { m.SetDoubleSided(true) }, 0},
	}
	for _, tt := range tests {
		calls = 0
		tt.do()
		if calls != tt.want {
			t.Errorf("%s: %d notices, want %d", tt.name, calls, tt.want)
		}
	}
	if m.Texture(viewport3d.PBRSlotEmissive).TexCoord != 1 {
		t.Error("the texture was not kept")
	}
}

// TestNewEnvironmentChecksItsCube checks the panic of NewEnvironment, with its tag and its route.
func TestNewEnvironmentChecksItsCube(t *testing.T) {
	rig := newDrawRig(t, viewport3d.NewScene(), frontCamera())
	dev := rig.frame.Device
	make1 := func(kind gpu.TextureKind, layers int, format gpu.TextureFormat, usage gpu.TextureUsage) *gpu.Texture {
		tex, err := dev.NewTexture(gpu.TextureDesc{Kind: kind, Size: gpu.Extent3D{Width: 4, Height: 4, DepthOrArrayLayers: layers}, Format: format, Usage: usage})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(tex.Release)
		return tex
	}
	good := make1(gpu.TextureKindCube, 6, gpu.TextureFormatRGBA16Float, gpu.TextureUsageTextureBinding)
	srgb := make1(gpu.TextureKindCube, 6, gpu.TextureFormatRGBA8Unorm, gpu.TextureUsageTextureBinding)
	flat := make1(gpu.TextureKind2D, 1, gpu.TextureFormatRGBA8Unorm, gpu.TextureUsageTextureBinding)
	unsampled := make1(gpu.TextureKindCube, 6, gpu.TextureFormatRGBA8Unorm, gpu.TextureUsageCopySrc)
	r32 := make1(gpu.TextureKindCube, 6, gpu.TextureFormatR32Float, gpu.TextureUsageTextureBinding)
	if viewport3d.NewEnvironment(good) == nil || viewport3d.NewEnvironment(srgb) == nil {
		t.Fatal("a cube of a format that is read gave no environment")
	}
	tests := []struct {
		name string
		cube *gpu.Texture
	}{
		{"no texture", nil},
		{"a flat texture", flat},
		{"a cube that cannot be sampled", unsampled},
		{"a cube of another format", r32},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skipUnlessDebug(t)
			wantTag(t, panicOf(func() { viewport3d.NewEnvironment(tt.cube) }), viewport3d.TagEnvironmentInvalid, "NewEnvironment.EnvironmentInvalid")
		})
	}
}

// TestTheEnvironmentIsMadeOnceAndAgainWhenItChanges checks the compute passes of the environment: the default one is made by a pass
// for the sky and one for the prefiltering, the app's cube by the second only, none of them is made again by a frame that changes
// nothing, and a change of the environment makes the new one.
func TestTheEnvironmentIsMadeOnceAndAgainWhenItChanges(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Sphere(0.5, 12, 6), viewport3d.NewPBRMaterial()))
	rig := newDrawRig(t, scene, frontCamera())
	computes := func(calls []string) int { return countPrefix(calls, "BeginComputePass(") }
	cube, err := rig.frame.Device.NewTexture(gpu.TextureDesc{Kind: gpu.TextureKindCube, Size: gpu.Extent3D{Width: 8, Height: 8, DepthOrArrayLayers: 6}, MipLevelCount: 4,
		Format: gpu.TextureFormatRGBA16Float, Usage: gpu.TextureUsageTextureBinding})
	if err != nil {
		t.Fatal(err)
	}
	defer cube.Release()
	if n := computes(rig.redraw()); n != 0 {
		t.Errorf("a frame that changes nothing made %d compute passes", n)
	}
	scene.SetStyle(viewport3d.SceneStyle{Environment: viewport3d.NewEnvironment(cube)})
	if n := computes(rig.redraw()); n != 1 {
		t.Errorf("the app's environment was made by %d compute passes, want 1", n)
	}
	if n := computes(rig.redraw()); n != 0 {
		t.Errorf("the next frame made %d compute passes", n)
	}
	scene.SetStyle(viewport3d.SceneStyle{})
	if n := computes(rig.redraw()); n != 2 {
		t.Errorf("the default environment was made by %d compute passes, want 2", n)
	}
	// A style that changes only the exposure does not make it again.
	scene.SetStyle(viewport3d.SceneStyle{Exposure: foundation.Some(2.0)})
	if n := computes(rig.redraw()); n != 0 {
		t.Errorf("a new exposure made %d compute passes", n)
	}
}

// TestTheMeshChecksThePBRAttributes checks MeshInvalid for the attributes of the PBR material that do not fit the vertices.
func TestTheMeshChecksThePBRAttributes(t *testing.T) {
	pos := []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}
	valid := func() viewport3d.MeshData {
		return viewport3d.MeshData{Positions: pos, Indices16: []uint16{0, 1, 2}}
	}
	tests := []struct {
		name  string
		mod   func(d *viewport3d.MeshData)
		valid bool
	}{
		{"all the attributes", func(d *viewport3d.MeshData) {
			d.Tangents, d.TexCoords1 = make([]float32, 12), make([]float32, 6)
			d.Joints, d.Weights = make([]uint16, 12), make([]float32, 12)
			d.MorphTargets = []viewport3d.MorphTarget{{Positions: make([]float32, 9), Normals: make([]float32, 9)}, {Positions: make([]float32, 9)}, {}}
		}, true},
		{"tangents of three components", func(d *viewport3d.MeshData) { d.Tangents = make([]float32, 9) }, false},
		{"a second set of texture coordinates for two vertices", func(d *viewport3d.MeshData) { d.TexCoords1 = make([]float32, 4) }, false},
		{"joints without weights", func(d *viewport3d.MeshData) { d.Joints = make([]uint16, 12) }, false},
		{"weights without joints", func(d *viewport3d.MeshData) { d.Weights = make([]float32, 12) }, false},
		{"joints of three", func(d *viewport3d.MeshData) { d.Joints, d.Weights = make([]uint16, 9), make([]float32, 12) }, false},
		{"a morph target of two vertices", func(d *viewport3d.MeshData) {
			d.MorphTargets = []viewport3d.MorphTarget{{Positions: make([]float32, 6)}}
		}, false},
		{"a morph target with short normals", func(d *viewport3d.MeshData) {
			d.MorphTargets = []viewport3d.MorphTarget{{Normals: make([]float32, 3)}}
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := valid()
			tt.mod(&d)
			m, err := viewport3d.NewMesh(d)
			if tt.valid {
				if err != nil {
					t.Fatal(err)
				}
				if !m.Skinned() || m.MorphTargetCount() != 3 {
					t.Errorf("Skinned %v, MorphTargetCount %d", m.Skinned(), m.MorphTargetCount())
				}
				return
			}
			wantTag(t, err, viewport3d.TagMeshInvalid, "NewMesh->Mesh.validate.MeshInvalid")
		})
	}
	plain, err := viewport3d.NewMesh(valid())
	if err != nil {
		t.Fatal(err)
	}
	if plain.Skinned() || plain.MorphTargetCount() != 0 {
		t.Errorf("a mesh with neither: Skinned %v, MorphTargetCount %d", plain.Skinned(), plain.MorphTargetCount())
	}
}

// TestAnItemTakesAPBRMaterialForTrianglesOnly checks NewItem: the material of triangles goes with a mesh of triangles.
func TestAnItemTakesAPBRMaterialForTrianglesOnly(t *testing.T) {
	skipUnlessDebug(t)
	pbr := viewport3d.NewPBRMaterial()
	wantTag(t, panicOf(func() { viewport3d.NewItem(scenetest.Segments([3]float32{}, [3]float32{1, 1, 1}), pbr) }), viewport3d.TagItemInvalid, "NewItem")
	wantTag(t, panicOf(func() { viewport3d.NewItem(scenetest.Points([3]float32{}), pbr) }), viewport3d.TagItemInvalid, "NewItem")
	if viewport3d.NewItem(scenetest.Box(1, 1, 1), pbr) == nil {
		t.Error("no item")
	}
}

// TestAnItemsSkinAndMorphWeights checks the accessors and that the scene listens to the skin while an item of it has the skin.
func TestAnItemsSkinAndMorphWeights(t *testing.T) {
	skin := viewport3d.NewSkin([]vecmath.Mat4{vecmath.Mat4Identity()})
	item := viewport3d.NewItem(scenetest.SkinnedBar(), viewport3d.NewPBRMaterial())
	scene := viewport3d.NewScene()
	notices := 0
	scene.AddListener(func() { notices++ })
	scene.Add(item)
	item.SetSkin(skin)
	if item.Skin() != skin {
		t.Fatal("the skin was not kept")
	}
	notices = 0
	skin.SetJointWorld(0, vecmath.NewMat4Translation(vecmath.Vec3{X: 1}))
	if notices != 1 {
		t.Errorf("the scene was told %d times of a joint, want 1", notices)
	}
	// Another item with the same skin: still one notice for a change.
	other := viewport3d.NewItem(scenetest.SkinnedBar(), viewport3d.NewPBRMaterial())
	other.SetSkin(skin)
	scene.Add(other)
	notices = 0
	skin.SetJointWorld(0, vecmath.Mat4Identity())
	if notices != 1 {
		t.Errorf("the scene was told %d times of a joint of a skin of two items, want 1", notices)
	}
	scene.Remove(item, other)
	notices = 0
	skin.SetJointWorld(0, vecmath.NewMat4Translation(vecmath.Vec3{X: 5}))
	if notices != 0 {
		t.Errorf("the scene was told %d times of a joint of a skin that it no longer has", notices)
	}
	// Adding an item that has a skin already listens too.
	scene.Add(item)
	notices = 0
	skin.SetJointWorld(0, vecmath.Mat4Identity())
	if notices != 1 {
		t.Errorf("the scene was told %d times of a joint after the item came back, want 1", notices)
	}
	item.SetMorphWeights([]float32{0.25, 0.5})
	w := item.MorphWeights()
	w[0] = 9
	if got := item.MorphWeights(); len(got) != 2 || got[0] != 0.25 {
		t.Errorf("the weights are %v: the accessor gives a copy", got)
	}
	if viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPBRMaterial()).MorphWeights() != nil {
		t.Error("an item with no weights has some")
	}
}

// TestTheStyleResolvesTheShadowAndTheToneMapping checks the defaults of the new parts of the style and their names.
func TestTheStyleResolvesTheShadowAndTheToneMapping(t *testing.T) {
	for tm, name := range map[viewport3d.ToneMapping]string{viewport3d.ToneMappingUnspecified: "PBR Neutral", viewport3d.ToneMappingNeutral: "PBR Neutral",
		viewport3d.ToneMappingACES: "ACES", viewport3d.ToneMappingNone: "none"} {
		if tm.String() != name {
			t.Errorf("%d is %q, want %q", tm, tm.String(), name)
		}
	}
	// A frame whose style has a shadow map of an absurd size still draws, with the size kept in range.
	scene := viewport3d.NewScene()
	scene.SetStyle(viewport3d.SceneStyle{Lights: sun(), Shadow: viewport3d.ShadowStyle{Enabled: true, Size: 1 << 20, Strength: foundation.Some(7.0), Light: -4}})
	scene.Add(viewport3d.NewItem(scenetest.Sphere(0.5, 12, 6), viewport3d.NewPBRMaterial()))
	rig := newDrawRig(t, scene, frontCamera())
	calls := rig.redraw()
	if !strings.Contains(strings.Join(pipelinesOf(calls), ","), "shadow") {
		t.Errorf("no shadow pass: %v", pipelinesOf(calls))
	}
}
