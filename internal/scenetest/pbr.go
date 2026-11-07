package scenetest

import (
	"context"
	"embed"
	"io/fs"
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/gltf"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// samples are CC0 models of Khronos' glTF-Sample-Assets, with their license (testdata/samples/LICENSE.txt).
//
//go:embed testdata/samples
var samples embed.FS

// frontCamera looks at the scene from the front, so that it fills the picture.
func frontCamera(scene *viewport3d.Scene, w, h int) *viewport3d.Camera {
	cam := viewport3d.NewCamera(viewport3d.CameraOpts{Up: vecmath.Vec3Y})
	cam.SetViewportSize(ui.Size{Width: float64(w), Height: float64(h)})
	cam.SetStandardView(viewport3d.StandardViewFront)
	cam.SetBounds(scene.Bounds())
	cam.FitTo(scene.Bounds())
	return cam
}

// loadSample reads a sample model and makes its items, scaled and moved so that it fills a cube of the size around the point.
func loadSample(d *gpu.Device, file string, at vecmath.Vec3, size float64) (*model.Model, *viewport3d.Uploaded) {
	fsys, err := fs.Sub(samples, "testdata/samples")
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	m, err := gltf.Load(ctx, fsys, file)
	if err != nil {
		panic(err)
	}
	b := m.Bounds()
	ext := b.Max.Sub(b.Min)
	k := size / math.Max(math.Max(ext.X, ext.Y), math.Max(ext.Z, 1e-9))
	to := vecmath.NewMat4Translation(at).Mul(vecmath.NewMat4Scale(vecmath.Vec3{X: k, Y: k, Z: k})).Mul(vecmath.NewMat4Translation(b.Center().Neg()))
	u, err := viewport3d.Upload(ctx, d, m, viewport3d.UploadOpts{Transform: foundation.Some(to)})
	if err != nil {
		panic(err)
	}
	return m, u
}

// quad makes a square of the side facing +z, with texture coordinates (v down) and, when asked, tangents along +x.
func quad(side float32, tangents bool) *viewport3d.Mesh {
	h := side / 2
	d := viewport3d.MeshData{
		Positions: []float32{-h, -h, 0, h, -h, 0, h, h, 0, -h, h, 0},
		Normals:   []float32{0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1},
		TexCoords: []float32{0, 1, 1, 1, 1, 0, 0, 0},
		Indices16: []uint16{0, 1, 2, 0, 2, 3},
	}
	if tangents {
		d.Tangents = []float32{1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1}
	}
	return mustMesh(d)
}

// domeNormalMap makes a texture of a grid of round bumps, as a normal map (the x of a normal in red, its y in green, up in the
// image).
func domeNormalMap(d *gpu.Device) (*gpu.Texture, func()) {
	const n, cell = 64, 16
	px := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			// The position in the cell from -1 to 1, with y up in the image.
			cx := (float64(x%cell)+0.5)/cell*2 - 1
			cy := 1 - (float64(y%cell)+0.5)/cell*2
			r2 := cx*cx + cy*cy
			nx, ny, nz := 0.0, 0.0, 1.0
			if r2 < 0.8 {
				// The slope of a dome of height sqrt(1 - r2): the normal leans away from the center.
				nx, ny, nz = cx, cy, math.Sqrt(1-r2)*0.9+0.1
				l := math.Sqrt(nx*nx + ny*ny + nz*nz)
				nx, ny, nz = nx/l, ny/l, nz/l
			}
			copy(px[(y*n+x)*4:], []byte{byte(math.Round((nx*0.5 + 0.5) * 255)), byte(math.Round((ny*0.5 + 0.5) * 255)), byte(math.Round((nz*0.5 + 0.5) * 255)), 255})
		}
	}
	t, err := d.NewTexture(gpu.TextureDesc{Label: "normal map", Size: gpu.Extent3D{Width: n, Height: n, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatRGBA8Unorm,
		Usage: gpu.TextureUsageTextureBinding})
	if err != nil {
		panic(err)
	}
	d.Queue().WriteTexture(gpu.TextureCopy{Texture: t}, px, gpu.DataLayout{BytesPerRow: 4 * n}, gpu.Extent3D{Width: n, Height: n, DepthOrArrayLayers: 1})
	return t, t.Release
}

// alphaTexture makes a texture of a checker of opaque and clear squares.
func alphaTexture(d *gpu.Device) (*gpu.Texture, func()) {
	const n = 8
	px := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			a := byte(0)
			if (x/2+y/2)%2 == 0 {
				a = 255
			}
			copy(px[(y*n+x)*4:], []byte{230, 190, 60, a})
		}
	}
	t, err := d.NewTexture(gpu.TextureDesc{Label: "alpha checker", Size: gpu.Extent3D{Width: n, Height: n, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatRGBA8Unorm,
		Usage: gpu.TextureUsageTextureBinding})
	if err != nil {
		panic(err)
	}
	d.Queue().WriteTexture(gpu.TextureCopy{Texture: t}, px, gpu.DataLayout{BytesPerRow: 4 * n}, gpu.Extent3D{Width: n, Height: n, DepthOrArrayLayers: 1})
	return t, t.Release
}

// SkinnedBar is a square bar along x, from 0 to 2, with a skin of two joints (the first at x = 0, the second at x = 1) and the
// weights blended between x = 0.75 and x = 1.25.
func SkinnedBar() *viewport3d.Mesh {
	var d viewport3d.MeshData
	const rings = 17
	for i := 0; i < rings; i++ {
		x := float32(i) * 0.125
		w1 := min(max((x-0.75)/0.5, 0), 1)
		for k := 0; k < 4; k++ {
			ang := float64(k)*math.Pi/2 + math.Pi/4
			ny, nz := float32(math.Cos(ang)), float32(math.Sin(ang))
			d.Positions = append(d.Positions, x, 0.2*ny, 0.2*nz)
			d.Normals = append(d.Normals, 0, ny, nz)
			d.Joints = append(d.Joints, 0, 1, 0, 0)
			d.Weights = append(d.Weights, 1-w1, w1, 0, 0)
		}
	}
	for i := 0; i < rings-1; i++ {
		for k := 0; k < 4; k++ {
			a, b := uint16(i*4+k), uint16(i*4+(k+1)%4)
			c, e := uint16((i+1)*4+(k+1)%4), uint16((i+1)*4+k)
			d.Indices16 = append(d.Indices16, a, b, e, b, c, e)
		}
	}
	return mustMesh(d)
}

// skinCase is the bar bent at the second joint by the angle.
func skinCase(name string, angle float64) Case {
	return Case{Name: name, Width: 400, Height: 240, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
		s := viewport3d.NewScene()
		skin := viewport3d.NewSkin([]vecmath.Mat4{vecmath.Mat4Identity(), vecmath.NewMat4Translation(vecmath.Vec3{X: -1})})
		skin.SetJointWorld(1, vecmath.NewMat4Translation(vecmath.Vec3{X: 1}).Mul(vecmath.NewMat4RotationZ(angle)))
		item := viewport3d.NewItem(SkinnedBar(), viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: color32(0xFF4A78C8), Metallic: foundation.Some(0.1), Roughness: foundation.Some(0.4)}))
		item.SetSkin(skin)
		// The rest pose and a marker give the camera the same box in every frame.
		marker := viewport3d.NewItem(Sphere(0.1, 12, 8), viewport3d.NewPBRMaterial(), viewport3d.ItemOpts{Transform: translate(1, 1.2, 0)})
		s.Add(item, marker)
		return s, frontCamera(s, 400, 240), nil
	}}
}

// MorphSphere is a sphere of the radius that has one morph target: the change to a squashed ellipsoid.
func MorphSphere(r float32, segments, rings int) *viewport3d.Mesh {
	data := Sphere(r, segments, rings).Data()
	target := viewport3d.MorphTarget{Positions: make([]float32, len(data.Positions))}
	for i := 0; i < len(data.Positions); i += 3 {
		target.Positions[i], target.Positions[i+1] = 0.5*data.Positions[i], -0.45*data.Positions[i+1]
	}
	data.MorphTargets = []viewport3d.MorphTarget{target}
	return mustMesh(data)
}

// morphCase is three spheres that are moved towards a squashed ellipsoid by 0, a half and all of it.
func morphCase() Case {
	return Case{Name: "pbr-morph", Width: 480, Height: 200, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
		s := viewport3d.NewScene()
		mesh := MorphSphere(0.45, 40, 24)
		mat := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: color32(0xFFC8503C), Roughness: foundation.Some(0.35)})
		for i, w := range []float32{0, 0.5, 1} {
			it := viewport3d.NewItem(mesh, mat, viewport3d.ItemOpts{Transform: translate(float64(i)*1.4-1.4, 0, 0)})
			it.SetMorphWeights([]float32{w})
			s.Add(it)
		}
		return s, frontCamera(s, 480, 200), nil
	}}
}

// colorCube makes a cube texture of six flat colors (the faces +x, -x, +y, -y, +z, -z), with its mip levels.
func colorCube(d *gpu.Device, colors [6][3]byte) (*gpu.Texture, func()) {
	const size, levels = 16, 5
	t, err := d.NewTexture(gpu.TextureDesc{Label: "color cube", Kind: gpu.TextureKindCube, Size: gpu.Extent3D{Width: size, Height: size, DepthOrArrayLayers: 6},
		MipLevelCount: levels, Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding})
	if err != nil {
		panic(err)
	}
	for l := 0; l < levels; l++ {
		n := size >> l
		for f := 0; f < 6; f++ {
			px := make([]byte, n*n*4)
			for i := 0; i < n*n; i++ {
				copy(px[4*i:], []byte{colors[f][0], colors[f][1], colors[f][2], 255})
			}
			d.Queue().WriteTexture(gpu.TextureCopy{Texture: t, MipLevel: l, Origin: gpu.Origin3D{Z: f}}, px, gpu.DataLayout{BytesPerRow: 4 * n}, gpu.Extent3D{Width: n, Height: n, DepthOrArrayLayers: 1})
		}
	}
	return t, t.Release
}

// sampleSkinCase is the skin of Khronos' SimpleSkin at the fraction of its animation.
func sampleSkinCase(name string, fraction float32) Case {
	return Case{Name: name, Width: 300, Height: 300, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
		s := viewport3d.NewScene()
		m, u := loadSample(d, "SimpleSkin/glTF/SimpleSkin.gltf", vecmath.Vec3{}, 3)
		pose := viewport3d.NewPose()
		anim := viewport3d.NewAnimation(m.Animations[0])
		anim.Sample(fraction*anim.Duration(), pose)
		u.SetPose(pose)
		s.Add(u.Items()...)
		return s, frontCamera(s, 300, 300), u.Release
	}}
}

// pbrCases are the scenes of the PBR material (SPEC §28.5).
func pbrCases() []Case {
	return []Case{
		{Name: "pbr-environment", Width: 480, Height: 200, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			// Metal spheres in an environment of six flat colors: +x red, -x blue, +y white, -y brown, +z green, -z yellow.
			cube, release := colorCube(d, [6][3]byte{{230, 60, 50}, {50, 80, 230}, {240, 240, 240}, {90, 60, 30}, {60, 200, 80}, {240, 220, 60}})
			s := viewport3d.NewScene()
			s.SetStyle(viewport3d.SceneStyle{Environment: viewport3d.NewEnvironment(cube), HeadLight: foundation.Some(0.0)})
			mesh := Sphere(0.55, 40, 24)
			for i, rough := range []float64{0.08, 0.35, 0.7} {
				m := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: color32(0xFFE8E8E8), Metallic: foundation.Some(1.0), Roughness: foundation.Some(rough)})
				s.Add(viewport3d.NewItem(mesh, m, viewport3d.ItemOpts{Transform: translate(float64(i)*1.4-1.4, 0, 0)}))
			}
			return s, frontCamera(s, 480, 200), release
		}},
		sampleSkinCase("pbr-sample-skin-a", 0),
		sampleSkinCase("pbr-sample-skin-b", 0.75),
		{Name: "pbr-spheres", Width: 520, Height: 320, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			// Spheres by metalness (rows) and roughness (columns), in the default environment.
			s := viewport3d.NewScene()
			mesh := Sphere(0.42, 40, 24)
			for row, metallic := range []float64{0, 0.5, 1} {
				for col, rough := range []float64{0.05, 0.25, 0.5, 0.75, 1} {
					m := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: color32(0xFFD8A060), Metallic: foundation.Some(metallic), Roughness: foundation.Some(rough)})
					s.Add(viewport3d.NewItem(mesh, m, viewport3d.ItemOpts{Transform: translate(float64(col)-2, 1-float64(row), 0)}))
				}
			}
			return s, frontCamera(s, 520, 320), nil
		}},
		{Name: "pbr-normal-maps", Width: 480, Height: 260, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			// The same bumps with the frame of the mesh's tangents (left) and with the frame from derivatives (right), lit from the
			// side so that the slopes show.
			tex, release := domeNormalMap(d)
			s := viewport3d.NewScene()
			s.SetStyle(viewport3d.SceneStyle{HeadLight: foundation.Some(0.0), EnvironmentIntensity: foundation.Some(0.3),
				Lights: []viewport3d.DirectionalLight{{Direction: vecmath.Vec3{X: 0.8, Y: 0.5, Z: 0.5}, Intensity: 1.6}}})
			for i, tangents := range []bool{true, false} {
				m := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: color32(0xFFB0B8C0), Roughness: foundation.Some(0.5),
					NormalTexture: viewport3d.PBRTexture{Texture: tex}})
				s.Add(viewport3d.NewItem(quad(2, tangents), m, viewport3d.ItemOpts{Transform: translate(float64(i)*2.2-1.1, 0, 0)}))
			}
			return s, frontCamera(s, 480, 260), release
		}},
		skinCase("pbr-skin-a", 0),
		skinCase("pbr-skin-b", 1.1),
		morphCase(),
		{Name: "pbr-shadow", Width: 440, Height: 300, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			// A box and a sphere on a plane, with the shadow of the light that comes from the right and above.
			s := viewport3d.NewScene()
			s.SetStyle(viewport3d.SceneStyle{HeadLight: foundation.Some(0.1), EnvironmentIntensity: foundation.Some(0.35),
				Lights: []viewport3d.DirectionalLight{{Direction: vecmath.Vec3{X: -0.6, Y: 1, Z: 0.35}, Intensity: 1.7}},
				Shadow: viewport3d.ShadowStyle{Enabled: true}})
			ground := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: color32(0xFF8C948C), Roughness: foundation.Some(0.9)})
			s.Add(viewport3d.NewItem(Box(4.4, 0.1, 4.4), ground, viewport3d.ItemOpts{Transform: translate(0, -0.55, 0)}))
			s.Add(viewport3d.NewItem(Box(1, 1, 1), viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: color32(0xFFC84A3C), Roughness: foundation.Some(0.5)}), viewport3d.ItemOpts{Transform: translate(-0.6, 0, 0)}))
			s.Add(viewport3d.NewItem(Sphere(0.45, 32, 20), viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: color32(0xFF4A78C8), Roughness: foundation.Some(0.3)}), viewport3d.ItemOpts{Transform: translate(1, -0.1, 0.5)}))
			cam := Isometric(s, 440, 300)
			cam.Orbit(0.2, -0.1)
			cam.FitTo(s.Bounds())
			return s, cam, nil
		}},
		{Name: "pbr-alpha", Width: 480, Height: 240, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			// A masked quad with a checker of clear squares, a blended sphere over a box, and a plane seen from behind (double sided).
			tex, release := alphaTexture(d)
			s := viewport3d.NewScene()
			masked := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: color32(0xFFFFFFFF), Roughness: foundation.Some(0.6), AlphaMode: viewport3d.AlphaModeMask, DoubleSided: true,
				BaseColorTexture: viewport3d.PBRTexture{Texture: tex}})
			s.Add(viewport3d.NewItem(quad(1.6, false), masked, viewport3d.ItemOpts{Transform: translate(-1.8, 0, 0)}))
			s.Add(viewport3d.NewItem(Box(1, 1, 1), viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: color32(0xFFC84A3C)}), viewport3d.ItemOpts{Transform: translate(0, 0, -0.3)}))
			blend := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: ui.NewColorFromARGB(140, 70, 140, 220), Roughness: foundation.Some(0.2), AlphaMode: viewport3d.AlphaModeBlend})
			s.Add(viewport3d.NewItem(Sphere(0.6, 32, 20), blend, viewport3d.ItemOpts{Transform: translate(0.1, 0.1, 0.5)}))
			back := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Color: color32(0xFF60B070), Roughness: foundation.Some(0.5), DoubleSided: true})
			turned := vecmath.NewMat4Translation(vecmath.Vec3{X: 1.8}).Mul(vecmath.NewMat4RotationY(math.Pi * 0.8))
			s.Add(viewport3d.NewItem(quad(1.6, false), back, viewport3d.ItemOpts{Transform: foundation.Some(turned)}))
			return s, frontCamera(s, 480, 240), release
		}},
		{Name: "pbr-samples", Width: 600, Height: 200, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			// CC0 models: a textured quad, the texture transforms, vertex colors, a morphing cube and a skin, each in its own cube.
			s := viewport3d.NewScene()
			var ups []*viewport3d.Uploaded
			add := func(file string, x float64, pose func(m *model.Model, u *viewport3d.Uploaded)) {
				m, u := loadSample(d, file, vecmath.Vec3{X: x}, 1.8)
				if pose != nil {
					pose(m, u)
				}
				s.Add(u.Items()...)
				ups = append(ups, u)
			}
			add("SimpleTexture/glTF/SimpleTexture.gltf", -4, nil)
			add("TextureTransformTest/glTF/TextureTransformTest.gltf", -2, nil)
			add("BoxVertexColors/glTF-Binary/BoxVertexColors.glb", 0, nil)
			add("AnimatedMorphCube/glTF-Binary/AnimatedMorphCube.glb", 2, func(m *model.Model, u *viewport3d.Uploaded) {
				pose := viewport3d.NewPose()
				viewport3d.NewAnimation(m.Animations[0]).Sample(0.5*m.Animations[0].Duration(), pose)
				u.SetPose(pose)
			})
			add("SimpleSkin/glTF/SimpleSkin.gltf", 4, func(m *model.Model, u *viewport3d.Uploaded) {
				pose := viewport3d.NewPose()
				viewport3d.NewAnimation(m.Animations[0]).Sample(0.75*m.Animations[0].Duration(), pose)
				u.SetPose(pose)
			})
			cam := frontCamera(s, 600, 200)
			return s, cam, func() {
				for _, u := range ups {
					u.Release()
				}
			}
		}},
	}
}
