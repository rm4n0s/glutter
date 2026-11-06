package scenetest

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// Size of the pictures of the cases, in pixels.
const (
	W = 400
	H = 300
)

func color32(v uint32) ui.Color { return ui.NewColor(v) }

func translate(x, y, z float64) foundation.Opt[vecmath.Mat4] {
	return foundation.Some(vecmath.NewMat4Translation(vecmath.Vec3{X: x, Y: y, Z: z}))
}

// Isometric returns a perspective camera that shows the whole scene from the corner of the front, right and top.
func Isometric(scene *viewport3d.Scene, w, h int) *viewport3d.Camera {
	cam := viewport3d.NewCamera(viewport3d.CameraOpts{Up: vecmath.Vec3Y})
	cam.SetViewportSize(ui.Size{Width: float64(w), Height: float64(h)})
	cam.SetStandardView(viewport3d.StandardViewIsometric)
	cam.SetBounds(scene.Bounds())
	cam.FitTo(scene.Bounds())
	return cam
}

// topDown returns an orthographic camera that looks down -z at the origin, with the view h/unit units tall, so that a unit is
// pixelsPerUnit pixels.
func topDown(h int, pixelsPerUnit float64) *viewport3d.Camera {
	cam := viewport3d.NewCamera(viewport3d.CameraOpts{Projection: viewport3d.ProjectionKindOrthographic})
	half := float64(h) / pixelsPerUnit / 2
	cam.SetDistance(half / math.Tan(cam.FieldOfView()/2))
	cam.SetViewportSize(ui.Size{Width: float64(W), Height: float64(h)})
	cam.SetBounds(vecmath.NewBox3(vecmath.Vec3{X: -4, Y: -4, Z: -1}, vecmath.Vec3{X: 4, Y: 4, Z: 1}))
	return cam
}

// sectionCase is the part cut by section planes (each cuts away what its normal points to), with the caps and the outlines.
func sectionCase(name string, mode viewport3d.DisplayMode, planes ...vecmath.Plane) Case {
	return Case{Name: name, Width: W, Height: H, Depth: gpu.TextureFormatDepth24PlusStencil8, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
		s := viewport3d.NewScene()
		s.SetStyle(viewport3d.SceneStyle{DisplayMode: mode})
		s.Add(Part()...)
		s.SetSectionPlanes(planes...)
		return s, PartCamera(s, W, H), nil
	}}
}

// PartCamera is Isometric turned a little, so that the three faces of a box are not lit alike by the light at the camera (an
// isometric view sees them all at the same angle).
func PartCamera(scene *viewport3d.Scene, w, h int) *viewport3d.Camera {
	cam := Isometric(scene, w, h)
	cam.Orbit(0.35, -0.12)
	cam.FitTo(scene.Bounds())
	return cam
}

// Part is a part of the display mode cases: a box (8 shared vertices, no normals: flat), a cylinder through it and a sphere.
func Part() []*viewport3d.Item {
	steel := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFFB8C4D0)})
	blue := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFF4A78C8)})
	orange := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFFD8873A), Shininess: 64})
	return []*viewport3d.Item{
		viewport3d.NewItem(BoxNoNormals(2.4, 1, 1.4), steel),
		viewport3d.NewItem(Cylinder(0.4, 1.8, 28), blue, viewport3d.ItemOpts{Transform: translate(0.5, 0.5, 0)}),
		viewport3d.NewItem(Sphere(0.45, 24, 14), orange, viewport3d.ItemOpts{Transform: translate(-0.7, 0.9, 0.1)}),
	}
}

func partCase(name string, mode viewport3d.DisplayMode) Case {
	return Case{Name: name, Width: W, Height: H, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
		s := viewport3d.NewScene()
		s.SetStyle(viewport3d.SceneStyle{DisplayMode: mode})
		s.Add(Part()...)
		return s, PartCamera(s, W, H), nil
	}}
}

// checker makes an 8 by 8 texture of 4 by 4 texel squares, and its release.
func checker(d *gpu.Device) (*gpu.Texture, func()) {
	t, err := d.NewTexture(gpu.TextureDesc{Label: "checker", Size: gpu.Extent3D{Width: 8, Height: 8, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatRGBA8Unorm,
		Usage: gpu.TextureUsageTextureBinding})
	if err != nil {
		panic(err)
	}
	px := make([]byte, 8*8*4)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			v := byte(255)
			if (x/2+y/2)%2 == 0 {
				v = 70
			}
			copy(px[(y*8+x)*4:], []byte{v, v, 255, 255})
		}
	}
	d.Queue().WriteTexture(gpu.TextureCopy{Texture: t}, px, gpu.DataLayout{BytesPerRow: 32}, gpu.Extent3D{Width: 8, Height: 8, DepthOrArrayLayers: 1})
	return t, t.Release
}

// coloredBox is a box whose eight vertices are the corners of the color cube.
func coloredBox() *viewport3d.Mesh {
	m, err := viewport3d.NewMesh(viewport3d.MeshData{
		Positions: []float32{-.5, -.5, -.5, .5, -.5, -.5, .5, .5, -.5, -.5, .5, -.5, -.5, -.5, .5, .5, -.5, .5, .5, .5, .5, -.5, .5, .5},
		Colors:    []byte{255, 0, 0, 255, 255, 255, 0, 255, 0, 255, 0, 255, 0, 255, 255, 255, 255, 0, 255, 255, 255, 255, 255, 255, 0, 0, 255, 255, 255, 0, 255, 255},
		Indices16: []uint16{4, 5, 6, 4, 6, 7, 1, 0, 3, 1, 3, 2, 5, 1, 2, 5, 2, 6, 0, 4, 7, 0, 7, 3, 7, 6, 2, 7, 2, 3, 0, 1, 5, 0, 5, 4},
	})
	if err != nil {
		panic(err)
	}
	return m
}

// Cases are the scenes of the display tests. Each is drawn with 4 samples unless it says otherwise.
func Cases() []Case {
	cases := []Case{
		partCase("part-shaded", viewport3d.DisplayModeShaded),
		partCase("part-edges", viewport3d.DisplayModeShadedWithEdges),
		partCase("part-wireframe", viewport3d.DisplayModeWireframe),
		partCase("part-hidden", viewport3d.DisplayModeHiddenLine),
		{Name: "materials", Width: 520, Height: 160, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			tex, release := checker(d)
			s := viewport3d.NewScene()
			s.Add(
				viewport3d.NewItem(Box(1, 1, 1), viewport3d.NewUnlitMaterial(viewport3d.UnlitOpts{Color: color32(0xFFE04040)}), viewport3d.ItemOpts{Transform: translate(-2.4, 0, 0)}),
				viewport3d.NewItem(BoxNoNormals(1, 1, 1), viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFF50B060)}), viewport3d.ItemOpts{Transform: translate(-1.2, 0, 0)}),
				viewport3d.NewItem(Sphere(0.55, 32, 20), viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFFC8C8D8), Specular: color32(0xFFFFFFFF), Shininess: 80}), viewport3d.ItemOpts{Transform: translate(0, 0, 0)}),
				viewport3d.NewItem(coloredBox(), viewport3d.NewUnlitMaterial(viewport3d.UnlitOpts{Color: color32(0xFFFFFFFF), VertexColors: true}), viewport3d.ItemOpts{Transform: translate(1.2, 0, 0)}),
				viewport3d.NewItem(Box(1, 1, 1), viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFFFFFFFF), Texture: tex}), viewport3d.ItemOpts{Transform: translate(2.4, 0, 0)}),
			)
			cam := viewport3d.NewCamera(viewport3d.CameraOpts{Orientation: vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 1, Y: 1}, 0.45)})
			cam.SetViewportSize(ui.Size{Width: 520, Height: 160})
			cam.SetBounds(s.Bounds())
			cam.FitTo(s.Bounds())
			return s, cam, release
		}},
		{Name: "lights", Width: W, Height: H, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			s := viewport3d.NewScene()
			s.SetStyle(viewport3d.SceneStyle{
				Ambient: foundation.Some(0.15), HeadLight: foundation.Some(0.25),
				Lights: []viewport3d.DirectionalLight{
					{Direction: vecmath.Vec3{X: 1, Y: 0.5, Z: 0.5}, Color: color32(0xFFFF6040), Intensity: 0.8},
					{Direction: vecmath.Vec3{X: -1, Y: 0.3, Z: 0.5}, Color: color32(0xFF4060FF), Intensity: 0.8},
				},
			})
			s.Add(
				viewport3d.NewItem(Sphere(1, 48, 32), viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFFE0E0E0), Shininess: 40})),
				viewport3d.NewItem(BoxNoNormals(4, 0.1, 4), viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFFA0A0A0)}), viewport3d.ItemOpts{Transform: translate(0, -1.05, 0)}),
			)
			return s, Isometric(s, W, H), nil
		}},
		{Name: "lines", Width: W, Height: H, Samples: 4, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			s := viewport3d.NewScene()
			var items []*viewport3d.Item
			add := func(m *viewport3d.Mesh, mat viewport3d.Material) { items = append(items, viewport3d.NewItem(m, mat)) }
			ink := color32(0xFFF0F0F0)
			// Widths 1, 2, 4 and 8 pixels.
			for i, w := range []float64{1, 2, 4, 8} {
				y := float32(1.8 - 0.25*float64(i))
				add(Segments([3]float32{-3.6, y, 0}, [3]float32{-0.4, y, 0}), viewport3d.NewLineMaterial(viewport3d.LineOpts{Color: ink, Width: w}))
			}
			// Dashed, dotted and dash-dot, 2 pixels wide.
			for i, p := range [][]float64{{10, 6}, {2, 4}, {14, 4, 2, 4}, {6}} {
				y := float32(0.7 - 0.25*float64(i))
				add(Segments([3]float32{-3.6, y, 0}, [3]float32{-0.4, y, 0}), viewport3d.NewLineMaterial(viewport3d.LineOpts{Color: color32(0xFFFFC040), Width: 2, Stipple: p}))
			}
			// A polyline of many short segments keeps its dash pattern running.
			var curve [][3]float32
			for i := 0; i <= 40; i++ {
				a := float64(i) / 40 * 2 * math.Pi
				curve = append(curve, [3]float32{float32(1.9 + 1.3*math.Cos(a)), float32(1.1 + 1.0*math.Sin(a)), 0})
			}
			add(Polyline(curve...), viewport3d.NewLineMaterial(viewport3d.LineOpts{Color: color32(0xFF60D0FF), Width: 2, Stipple: []float64{8, 5}}))
			// Slanted lines show the anti-aliasing.
			for i := 0; i < 5; i++ {
				x := float32(-3.5 + 0.35*float64(i))
				add(Segments([3]float32{x, -0.5, 0}, [3]float32{x + 0.2, -2.0, 0}), viewport3d.NewLineMaterial(viewport3d.LineOpts{Color: color32(0xFF90FF90), Width: 1}))
			}
			add(Segments([3]float32{0.2, -0.6, 0}, [3]float32{3.6, -2.2, 0}, [3]float32{0.2, -2.2, 0}, [3]float32{3.6, -0.6, 0}), viewport3d.NewLineMaterial(viewport3d.LineOpts{Color: color32(0xFFFF8080), Width: 3}))
			s.Add(items...)
			return s, topDown(H, 60), nil
		}},
		{Name: "points", Width: W, Height: H, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			s := viewport3d.NewScene()
			for i, size := range []float64{2, 4, 8, 16, 24} {
				y := float32(1.6 - 0.7*float64(i))
				var pts [][3]float32
				for j := 0; j < 6; j++ {
					pts = append(pts, [3]float32{float32(-3.6 + 0.6*float64(j)), y, 0})
				}
				s.Add(viewport3d.NewItem(Points(pts...), viewport3d.NewPointMaterial(viewport3d.PointOpts{Color: color32(0xFFFFD060), Size: size})))
				var sq [][3]float32
				for j := 0; j < 6; j++ {
					sq = append(sq, [3]float32{float32(0.4 + 0.6*float64(j)), y, 0})
				}
				s.Add(viewport3d.NewItem(Points(sq...), viewport3d.NewPointMaterial(viewport3d.PointOpts{Color: color32(0xFF60C0FF), Size: size, Shape: viewport3d.PointShapeSquare})))
			}
			return s, topDown(H, 60), nil
		}},
		{Name: "transparent", Width: W, Height: H, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			s := viewport3d.NewScene()
			opaque := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFFC8C8C8)})
			red := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: ui.NewColorFromARGB(140, 230, 50, 50)})
			blue := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: ui.NewColorFromARGB(140, 50, 90, 230)})
			// Added in the order that is wrong for painting: the transparent items must be sorted.
			s.Add(
				viewport3d.NewItem(Sphere(0.8, 32, 20), red, viewport3d.ItemOpts{Transform: translate(-0.3, 0, 1.4)}),
				viewport3d.NewItem(Box(1.4, 1.4, 1.4), blue, viewport3d.ItemOpts{Transform: translate(0.4, 0, -1.6)}),
				viewport3d.NewItem(BoxNoNormals(2, 1.6, 0.4), opaque),
			)
			cam := viewport3d.NewCamera(viewport3d.CameraOpts{Orientation: vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 0.2, Y: 1}, 0.5)})
			cam.SetViewportSize(ui.Size{Width: W, Height: H})
			cam.SetBounds(s.Bounds())
			cam.FitTo(s.Bounds())
			return s, cam, nil
		}},
		{Name: "instances", Width: 640, Height: 480, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			s := viewport3d.NewScene()
			mesh := Box(0.18, 0.18, 0.18)
			mat := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFF6FA8DC)})
			items := make([]*viewport3d.Item, 0, 10000)
			for i := 0; i < 100; i++ {
				for j := 0; j < 100; j++ {
					h := 0.3 + 0.3*math.Sin(float64(i)/7)*math.Cos(float64(j)/9)
					m := vecmath.NewMat4TRS(vecmath.Vec3{X: float64(i)*0.2 - 10, Y: 0.09 * h * 8, Z: float64(j)*0.2 - 10}, vecmath.QuatIdentity(), vecmath.Vec3{X: 1, Y: h * 8, Z: 1})
					items = append(items, viewport3d.NewItem(mesh, mat, viewport3d.ItemOpts{Transform: foundation.Some(m)}))
				}
			}
			s.Add(items...)
			return s, Isometric(s, 640, 480), nil
		}},
		sectionCase("section", viewport3d.DisplayModeShaded, vecmath.Plane{Normal: vecmath.Vec3X, D: -0.3}),
		sectionCase("section-two", viewport3d.DisplayModeShadedWithEdges, vecmath.Plane{Normal: vecmath.Vec3X, D: -0.3}, vecmath.Plane{Normal: vecmath.Vec3Y, D: -0.6}),
		{Name: "section-torus", Width: W, Height: H, Depth: gpu.TextureFormatDepth24PlusStencil8, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			s := viewport3d.NewScene()
			s.SetStyle(viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeShadedWithEdges})
			s.Add(viewport3d.NewItem(Torus(1, 0.4, 48, 20), viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFFD8873A)})))
			// The plane through the axis of the torus: the cut is two rings.
			s.SetSectionPlanes(vecmath.Plane{Normal: vecmath.Vec3Z})
			cam := viewport3d.NewCamera(viewport3d.CameraOpts{Up: vecmath.Vec3Y})
			cam.SetViewportSize(ui.Size{Width: W, Height: H})
			cam.SetStandardView(viewport3d.StandardViewFront)
			cam.Orbit(0.5, 0.45)
			cam.SetBounds(s.Bounds())
			cam.FitTo(s.Bounds())
			return s, cam, nil
		}},
		{Name: "selection", Width: W, Height: H, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			s := viewport3d.NewScene()
			s.SetStyle(viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeShadedWithEdges})
			items := Part()
			s.Add(items...)
			// A line and a point are selected too: they are drawn in the highlight color.
			line := viewport3d.NewItem(Segments([3]float32{-1.2, -0.5, 0.8}, [3]float32{1.2, -0.5, 0.8}), viewport3d.NewLineMaterial(viewport3d.LineOpts{Color: color32(0xFFE0E0E0), Width: 2}))
			dot := viewport3d.NewItem(Points([3]float32{1.2, -0.5, 0.8}), viewport3d.NewPointMaterial(viewport3d.PointOpts{Size: 8}))
			s.Add(line, dot)
			s.Select(items[1], items[2], line, dot)
			return s, PartCamera(s, W, H), nil
		}},
		PartAt("far-a", FarOrigin, 0),
		PartAt("far-b", FarOrigin, 0.001),
		{Name: "ortho", Width: W, Height: H, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
			s := viewport3d.NewScene()
			s.SetStyle(viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeShadedWithEdges})
			s.Add(Part()...)
			cam := PartCamera(s, W, H)
			cam.SetKind(viewport3d.ProjectionKindOrthographic)
			return s, cam, nil
		}},
	}
	return cases
}

// FarOrigin is where the part of the far cases is: 50 km from the origin in each axis.
var FarOrigin = vecmath.Vec3{X: 50000, Y: -50000, Z: 30000}

// PartAt is a box at origin, moved by dx along x, seen from 3 units away. With FarOrigin it is 50 km from the world's origin:
// the pictures of far-a and far-b differ by what 1 mm is in pixels, and are those of the same part at the world's origin
// (SPEC §27.8, D30).
func PartAt(name string, origin vecmath.Vec3, dx float64) Case {
	return Case{Name: name, Width: W, Height: H, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
		s := viewport3d.NewScene()
		s.SetStyle(viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeShadedWithEdges})
		steel := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: color32(0xFFB8C4D0)})
		// The mesh's origin is the part's place, and its vertices are float32 offsets from it.
		data := BoxNoNormals(1, 1, 1).Data()
		data.Origin = origin
		box := mustMesh(data)
		s.Add(viewport3d.NewItem(box, steel, viewport3d.ItemOpts{Transform: translate(dx, 0, 0)}))
		cam := viewport3d.NewCamera(viewport3d.CameraOpts{Target: origin, Distance: 3,
			Orientation: vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 1, Y: 1}, 0.5)})
		cam.SetViewportSize(ui.Size{Width: W, Height: H})
		cam.SetBounds(s.Bounds())
		return s, cam, nil
	}}
}
