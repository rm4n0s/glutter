package viewport3d

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

// shadowTestMesh is a triangle.
func shadowTestMesh(t *testing.T) *Mesh {
	t.Helper()
	m, err := NewMesh(MeshData{Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}, Indices16: []uint16{0, 1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func f32(b []byte, off int) float32 {
	return math.Float32frombits(uint32(b[off]) | uint32(b[off+1])<<8 | uint32(b[off+2])<<16 | uint32(b[off+3])<<24)
}

// TestTheTextureTransformIsTheMatrixOfKHRTextureTransform checks the matrix that the shader gets against model.TextureTransform.Apply,
// which is the extension's own formula, on a few points.
func TestTheTextureTransformIsTheMatrixOfKHRTextureTransform(t *testing.T) {
	tests := []struct {
		name string
		tr   *TextureTransform
		mt   model.TextureTransform
	}{
		{"none", nil, model.TextureTransform{Scale: [2]float32{1, 1}}},
		{"a zero scale is 1", &TextureTransform{}, model.TextureTransform{Scale: [2]float32{1, 1}}},
		{"an offset", &TextureTransform{Offset: [2]float32{0.25, -0.5}}, model.TextureTransform{Offset: [2]float32{0.25, -0.5}, Scale: [2]float32{1, 1}}},
		{"a scale", &TextureTransform{Scale: [2]float32{2, 3}}, model.TextureTransform{Scale: [2]float32{2, 3}}},
		{"a rotation", &TextureTransform{Rotation: 0.7, Scale: [2]float32{1, 1}}, model.TextureTransform{Rotation: 0.7, Scale: [2]float32{1, 1}}},
		{"all three", &TextureTransform{Offset: [2]float32{0.1, 0.2}, Rotation: -1.2, Scale: [2]float32{2, 0.5}}, model.TextureTransform{Offset: [2]float32{0.1, 0.2}, Rotation: -1.2, Scale: [2]float32{2, 0.5}}},
	}
	for _, tt := range tests {
		m := tt.tr.matrix()
		for _, p := range [][2]float32{{0, 0}, {1, 0}, {0, 1}, {0.3, 0.8}} {
			wu, wv := tt.mt.Apply(p[0], p[1])
			gu, gv := m[0]*p[0]+m[1]*p[1]+m[2], m[3]*p[0]+m[4]*p[1]+m[5]
			if math.Abs(float64(gu-wu)) > 1e-5 || math.Abs(float64(gv-wv)) > 1e-5 {
				t.Errorf("%s: %v goes to (%v, %v), want (%v, %v)", tt.name, p, gu, gv, wu, wv)
			}
		}
	}
}

// TestPBRUniforms checks the layout of the material's uniform data against what pbr.wgsl reads.
func TestPBRUniforms(t *testing.T) {
	m := NewPBRMaterial(PBROpts{
		Color: ui.Color{A: 0.5, R: 0.5, G: 1, B: 0}, Metallic: foundation.Some(0.25), Roughness: foundation.Some(0.75), Emissive: [3]float64{1, 2, 3},
		NormalScale: foundation.Some(2.0), OcclusionStrength: foundation.Some(0.5), AlphaMode: AlphaModeMask, AlphaCutoff: foundation.Some(0.3), VertexColors: true, Unlit: true,
	})
	m.tex[PBRSlotNormal] = PBRTexture{TexCoord: 1, Transform: &TextureTransform{Offset: [2]float32{0.5, 0}, Scale: [2]float32{2, 2}}}
	var data [matUniformMax]byte
	if n := pbrUniforms(m, &data); n != pbrUniformSize {
		t.Fatalf("the size is %d, want %d", n, pbrUniformSize)
	}
	want := map[int]float32{
		0: srgbToLinear(0.5), 4: srgbToLinear(1), 8: 0, 12: 0.5, // the color, linear, with the alpha as it is
		16: 0.25, 20: 0.75, 24: 2, 28: 0.5, // metallic, roughness, normal scale, occlusion strength
		32: 1, 36: 2, 40: 3, // emissive
		48: 1, 52: 0.3, 56: 3, // the alpha mode, the cutoff, the bits: unlit (1) and vertex colors (2)
		// the normal slot: the first row (m00, m01, tx, set) and the second (m10, m11, ty, present)
		64 + 32*2: 2, 64 + 32*2 + 4: 0, 64 + 32*2 + 8: 0.5, 64 + 32*2 + 12: 1,
		64 + 32*2 + 16: 0, 64 + 32*2 + 20: 2, 64 + 32*2 + 24: 0, 64 + 32*2 + 28: 0, // no texture: present is 0
		// the base color slot has the identity transform
		64: 1, 64 + 4: 0, 64 + 16 + 4: 1,
	}
	for off, w := range want {
		if got := f32(data[:], off); math.Abs(float64(got-w)) > 1e-5 {
			t.Errorf("the float at byte %d is %v, want %v", off, got, w)
		}
	}
}

// TestSRGBTransfer checks the two conversions against each other and a known value.
func TestSRGBTransfer(t *testing.T) {
	for _, v := range []float32{0, 0.01, 0.04, 0.2, 0.5, 0.8, 1} {
		if got := linearToSRGB(srgbToLinear(v)); math.Abs(float64(got-v)) > 1e-5 {
			t.Errorf("round trip of %v is %v", v, got)
		}
	}
	// 0.5 in sRGB is 0.2140 in linear light.
	if got := srgbToLinear(0.5); math.Abs(float64(got)-0.21404) > 1e-4 {
		t.Errorf("srgbToLinear(0.5) = %v", got)
	}
}

// TestFitShadowIsStable checks the fit of the shadow map: it does not depend on the camera, its center is on the texels of the map and
// its size changes by steps, so that a small movement of an item does not change it.
func TestFitShadowIsStable(t *testing.T) {
	scene := NewScene()
	scene.SetStyle(SceneStyle{Lights: []DirectionalLight{{Direction: vecmath.Vec3{X: 1, Y: 2, Z: 0.5}}}, Shadow: ShadowStyle{Enabled: true}})
	scale := vecmath.NewMat4Scale(vecmath.Vec3{X: 1.1, Y: 1.1, Z: 1.1}) // not at the edge of a step of the radius
	a := NewItem(shadowTestMesh(t), NewPBRMaterial(), ItemOpts{Transform: foundation.Some(scale)})
	scene.Add(a)
	r := NewRenderer(scene, NewCamera())
	style := &scene.rs
	first, ok := r.fitShadow(style, 2048)
	if !ok {
		t.Fatal("no fit")
	}
	// A small move of the item inside the same step of radius and texel gives a map that is the same up to a texel.
	a.SetTransform(vecmath.NewMat4Translation(vecmath.Vec3{X: 0.001}).Mul(scale))
	second, _ := r.fitShadow(style, 2048)
	if second.radius != first.radius {
		t.Errorf("the radius changed from %v to %v for a move of 0.001", first.radius, second.radius)
	}
	texel := 2 * first.radius / 2048
	if d := second.center.Sub(first.center).Length(); d > texel*1.5 {
		t.Errorf("the center moved by %v, more than a texel and a half (%v)", d, texel*1.5)
	}
	// The light looks at the center, whatever the camera does.
	ndc, _ := first.lightWorld.ProjectPoint(first.center)
	if math.Abs(ndc.X) > 1e-9 || math.Abs(ndc.Y) > 1e-9 || math.Abs(ndc.Z-0.5) > 1e-9 {
		t.Errorf("the center is at %v in the light's clip space, want (0, 0, 0.5)", ndc)
	}
	// Nothing to shadow: no light, a light that does not exist, an empty scene.
	scene.SetStyle(SceneStyle{Shadow: ShadowStyle{Enabled: true}})
	if _, ok := r.fitShadow(&scene.rs, 2048); ok {
		t.Error("a fit with no light")
	}
	scene.SetStyle(SceneStyle{Lights: []DirectionalLight{{Direction: vecmath.Vec3{}}}, Shadow: ShadowStyle{Enabled: true}})
	if _, ok := r.fitShadow(&scene.rs, 2048); ok {
		t.Error("a fit with a light with no direction")
	}
}
