package gltf_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/gltf"
)

// The samples are CC0 models of the Khronos Group's glTF-Sample-Assets, with their licenses in testdata/sample-assets/LICENSE.txt.

func loadSample(t testing.TB, name string, rec *records) *model.Model {
	t.Helper()
	opts := gltf.Options{Logger: discard()}
	if rec != nil {
		opts.Logger = slog.New(rec)
	}
	m, err := gltf.Load(context.Background(), os.DirFS("testdata/sample-assets"), name, opts)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return m
}

func TestSamples(t *testing.T) {
	tests := []struct {
		file                                              string
		meshes, triangles, materials, textures, images    int
		nodes, skins, animations, scenes, cameras, lights int
	}{
		{"Triangle/glTF/Triangle.gltf", 1, 1, 0, 0, 0, 1, 0, 0, 1, 0, 0},
		{"Triangle/glTF-Embedded/Triangle.gltf", 1, 1, 0, 0, 0, 1, 0, 0, 1, 0, 0},
		{"TriangleWithoutIndices/glTF/TriangleWithoutIndices.gltf", 1, 1, 0, 0, 0, 1, 0, 0, 1, 0, 0},
		{"AnimatedTriangle/glTF/AnimatedTriangle.gltf", 1, 1, 0, 0, 0, 1, 0, 1, 1, 0, 0},
		{"SimpleMeshes/glTF/SimpleMeshes.gltf", 1, 2, 0, 0, 0, 2, 0, 0, 1, 0, 0},
		{"SimpleSkin/glTF/SimpleSkin.gltf", 1, 8, 0, 0, 0, 2, 1, 1, 1, 0, 0},
		{"SimpleMorph/glTF/SimpleMorph.gltf", 1, 1, 0, 0, 0, 1, 0, 1, 1, 0, 0},
		{"SimpleTexture/glTF/SimpleTexture.gltf", 1, 2, 1, 1, 1, 1, 0, 0, 1, 0, 0},
		{"Cameras/glTF/Cameras.gltf", 1, 2, 0, 0, 0, 3, 0, 0, 1, 2, 0},
		{"MultipleScenes/glTF/MultipleScenes.gltf", 2, 2, 0, 0, 0, 1, 0, 0, 2, 0, 0},
		{"MeshPrimitiveModes/glTF/MeshPrimitiveModes.gltf", 7, 16, 0, 0, 0, 7, 0, 0, 1, 0, 0},
		{"BoxVertexColors/glTF-Binary/BoxVertexColors.glb", 1, 12, 0, 0, 0, 1, 0, 0, 1, 0, 0},
		{"AnimatedMorphCube/glTF-Binary/AnimatedMorphCube.glb", 1, 12, 1, 0, 0, 1, 0, 1, 1, 0, 0},
		{"AnimatedMorphCube/glTF-Quantized/AnimatedMorphCube.gltf", 1, 12, 1, 0, 0, 1, 0, 1, 1, 0, 0},
		{"InterpolationTest/glTF-Binary/InterpolationTest.glb", 2, 110, 2, 1, 1, 10, 0, 9, 1, 0, 0},
		{"TextureTransformTest/glTF/TextureTransformTest.gltf", 9, 24, 9, 5, 5, 6, 0, 0, 1, 0, 0},
		{"MeshoptCubeTest/glTF/MeshoptCubeTest.gltf", 35, 320, 10, 10, 10, 35, 0, 1, 1, 0, 0},
		{"MeshoptCubeTest/glTF-Meshopt/MeshoptCubeTest.gltf", 35, 320, 10, 10, 10, 35, 0, 1, 1, 0, 0},
		{"SimpleInstancing/glTF-Binary/SimpleInstancing.glb", 1, 12, 0, 0, 0, 1, 0, 0, 1, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			m := loadSample(t, tt.file, nil)
			got := []int{len(m.Meshes), m.TriangleCount(), len(m.Materials), len(m.Textures), len(m.Images), len(m.Nodes), len(m.Skins), len(m.Animations), len(m.Scenes), len(m.Cameras), len(m.Lights)}
			want := []int{tt.meshes, tt.triangles, tt.materials, tt.textures, tt.images, tt.nodes, tt.skins, tt.animations, tt.scenes, tt.cameras, tt.lights}
			if !slices.Equal(got, want) {
				t.Errorf("counts (meshes, triangles, materials, textures, images, nodes, skins, animations, scenes, cameras, lights) are %v, want %v", got, want)
			}
		})
	}
}

func TestSampleSimpleSkin(t *testing.T) {
	m := loadSample(t, "SimpleSkin/glTF/SimpleSkin.gltf", nil)
	skin := m.Skins[0]
	if len(skin.Joints) != 2 || len(skin.InverseBindMatrices) != 2 || skin.InverseBindMatrices[1][13] != -1 {
		t.Errorf("skin %+v", skin)
	}
	mesh := m.Meshes[0]
	if mesh.VertexCount() != 10 || len(mesh.Joints) != 40 || len(mesh.Weights) != 40 {
		t.Errorf("mesh has %d vertices, %d joints, %d weights", mesh.VertexCount(), len(mesh.Joints), len(mesh.Weights))
	}
	anim := m.Animations[0]
	if len(anim.Channels) != 1 || anim.Channels[0].Path != model.PathRotation || anim.Channels[0].Node != skin.Joints[1] || anim.Duration() != 5.5 {
		t.Errorf("animation %+v duration %v", anim.Channels, anim.Duration())
	}
}

func TestSampleMorphs(t *testing.T) {
	m := loadSample(t, "SimpleMorph/glTF/SimpleMorph.gltf", nil)
	mesh := m.Meshes[0]
	if len(mesh.MorphTargets) != 2 || len(mesh.MorphTargets[0].Positions) != 9 || !slices.Equal(mesh.MorphWeights, []float32{0.5, 0.5}) {
		t.Errorf("morph targets %+v weights %v", mesh.MorphTargets, mesh.MorphWeights)
	}
	if c := m.Animations[0].Channels[0]; c.Path != model.PathWeights || c.Width() != 2 || c.Interpolation != model.InterpolationLinear {
		t.Errorf("channel %+v", c)
	}

	// the quantized variant has the same cube as the binary one, from short integers
	q := loadSample(t, "AnimatedMorphCube/glTF-Quantized/AnimatedMorphCube.gltf", nil)
	f := loadSample(t, "AnimatedMorphCube/glTF-Binary/AnimatedMorphCube.glb", nil)
	if len(q.Meshes[0].MorphTargets) != 2 || len(f.Meshes[0].MorphTargets) != 2 {
		t.Fatalf("targets %d and %d", len(q.Meshes[0].MorphTargets), len(f.Meshes[0].MorphTargets))
	}
	if q.TriangleCount() != f.TriangleCount() {
		t.Errorf("the quantized cube has %d triangles, the other %d", q.TriangleCount(), f.TriangleCount())
	}
	if qa := q.Meshes[0].Positions; len(qa) == 0 {
		t.Errorf("the quantized cube has no positions")
	}
}

func TestSampleInterpolation(t *testing.T) {
	m := loadSample(t, "InterpolationTest/glTF-Binary/InterpolationTest.glb", nil)
	seen := map[model.Interpolation]int{}
	paths := map[model.AnimationPath]int{}
	for _, a := range m.Animations {
		for _, c := range a.Channels {
			seen[c.Interpolation]++
			paths[c.Path]++
			if c.Interpolation == model.InterpolationCubicSpline && len(c.Values) != 3*len(c.Times)*c.Width() {
				t.Errorf("a cubic spline has %d values for %d times of %d values", len(c.Values), len(c.Times), c.Width())
			}
		}
	}
	if seen[model.InterpolationLinear] == 0 || seen[model.InterpolationStep] == 0 || seen[model.InterpolationCubicSpline] == 0 {
		t.Errorf("interpolations %v", seen)
	}
	if paths[model.PathTranslation] == 0 || paths[model.PathRotation] == 0 || paths[model.PathScale] == 0 {
		t.Errorf("paths %v", paths)
	}
}

func TestSamplePrimitiveModes(t *testing.T) {
	m := loadSample(t, "MeshPrimitiveModes/glTF/MeshPrimitiveModes.gltf", nil)
	counts := map[model.Primitive]int{}
	for _, mesh := range m.Meshes {
		counts[mesh.Primitive]++
		for _, i := range mesh.Indices {
			if int(i) >= mesh.VertexCount() {
				t.Fatalf("an index past the vertices")
			}
		}
	}
	if counts[model.PrimitivePoints] != 1 || counts[model.PrimitiveLines] != 3 || counts[model.PrimitiveTriangles] != 3 {
		t.Errorf("primitives %v", counts)
	}
}

func TestSampleCamerasAndScenes(t *testing.T) {
	m := loadSample(t, "Cameras/glTF/Cameras.gltf", nil)
	var persp, ortho int
	for _, c := range m.Cameras {
		if c.Orthographic {
			ortho++
		} else {
			persp++
		}
	}
	if persp != 1 || ortho != 1 {
		t.Errorf("cameras: %d perspective, %d orthographic", persp, ortho)
	}
	s := loadSample(t, "MultipleScenes/glTF/MultipleScenes.gltf", nil)
	// the file names its second scene as the default
	if s.DefaultScene == nil || s.DefaultScene != s.Scenes[1] || len(s.Nodes) != 1 || s.Nodes[0] != s.Scenes[1].Nodes[0] {
		t.Errorf("default scene %+v", s.DefaultScene)
	}
}

func TestSampleTextures(t *testing.T) {
	m := loadSample(t, "SimpleTexture/glTF/SimpleTexture.gltf", nil)
	img := m.Images[0]
	if img.Type != "image/png" || img.URI == "" || !bytes.HasPrefix(img.Data, []byte("\x89PNG")) {
		t.Errorf("image %q %q %d bytes", img.URI, img.Type, len(img.Data))
	}
	if m.Materials[0].BaseColorImage != img || m.Meshes[0].TexCoords == nil {
		t.Errorf("the material has the image and the mesh has texture coordinates")
	}

	tt := loadSample(t, "TextureTransformTest/glTF/TextureTransformTest.gltf", nil)
	var transforms int
	for _, mat := range tt.Materials {
		if mat.BaseColorTexture != nil && mat.BaseColorTexture.Transform != nil {
			transforms++
		}
	}
	if transforms == 0 {
		t.Errorf("no material has a texture transform")
	}
}

func TestSampleColorsAndInstancing(t *testing.T) {
	m := loadSample(t, "BoxVertexColors/glTF-Binary/BoxVertexColors.glb", nil)
	if mesh := m.Meshes[0]; len(mesh.Colors) != 4*mesh.VertexCount() {
		t.Errorf("%d bytes of colors for %d vertices", len(mesh.Colors), mesh.VertexCount())
	}
	rec := &records{}
	loadSample(t, "SimpleInstancing/glTF-Binary/SimpleInstancing.glb", rec)
	if len(rec.msgs) != 1 {
		t.Errorf("messages %v: the extension of instances is left out and noted once", rec.msgs)
	}
}

// TestMeshoptSampleEqualsFallback decodes the compressed model and the one with the data uncompressed, and compares them.
func TestMeshoptSampleEqualsFallback(t *testing.T) {
	plain := loadSample(t, "MeshoptCubeTest/glTF/MeshoptCubeTest.gltf", nil)
	packed := loadSample(t, "MeshoptCubeTest/glTF-Meshopt/MeshoptCubeTest.gltf", nil)
	if len(plain.Meshes) != len(packed.Meshes) {
		t.Fatalf("meshes %d and %d", len(plain.Meshes), len(packed.Meshes))
	}
	for i, a := range plain.Meshes {
		b := packed.Meshes[i]
		if !slices.Equal(a.Positions, b.Positions) || !slices.Equal(a.Normals, b.Normals) || !slices.Equal(a.TexCoords, b.TexCoords) ||
			!slices.Equal(a.Tangents, b.Tangents) || !slices.Equal(a.Indices, b.Indices) || !slices.Equal(a.Colors, b.Colors) {
			t.Errorf("mesh %d (%s) differs between the compressed model and its fallback", i, a.Name)
		}
		if len(a.Positions) == 0 {
			t.Errorf("mesh %d has no positions", i)
		}
	}
}

func TestSamplesLoadWithEveryReader(t *testing.T) {
	// a model that is read from a reader, with its files from the file system of the options
	root := os.DirFS("testdata/sample-assets/Triangle/glTF")
	f, err := root.Open("Triangle.gltf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	m, err := gltf.Decode(context.Background(), f, gltf.Options{FS: root, Dir: ".", Name: "tri", Logger: discard()})
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "tri" || m.TriangleCount() != 1 {
		t.Errorf("model %q", m.Name)
	}
}
