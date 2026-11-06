package scenetest

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/viewport3d"
)

// The scenes of the benchmarks of SPEC §27.10: a part of about a million triangles, and ten thousand items that share a mesh or
// each have their own.

// Terraces makes a surface of n x n cells, two triangles each (2*n*n in all), whose height steps by block cells: a stepped
// surface has the edges of a real part (a crease at each step) and a smooth normal on each tread. n = 708 makes 1,002,528
// triangles.
func Terraces(n, block int) *viewport3d.Mesh {
	h := func(x, y int) float32 {
		bx, by := x/block, y/block
		return float32(math.Sin(float64(bx)*0.7)*math.Cos(float64(by)*0.5)) * 4
	}
	var data viewport3d.MeshData
	stride := n + 1
	data.Positions = make([]float32, 0, stride*stride*3)
	data.Normals = make([]float32, 0, stride*stride*3)
	data.Indices32 = make([]uint32, 0, n*n*6)
	for y := 0; y <= n; y++ {
		for x := 0; x <= n; x++ {
			// A vertex on the border of two blocks takes the height of the block it is the corner of (the cell to its upper right),
			// so a tread is flat and a riser is one cell wide, steep.
			data.Positions = append(data.Positions, float32(x)*0.1, float32(y)*0.1, h(min(x, n-1), min(y, n-1)))
			data.Normals = append(data.Normals, 0, 0, 1)
		}
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			a := uint32(y*stride + x)
			b, c, d := a+1, a+uint32(stride), a+uint32(stride)+1
			data.Indices32 = append(data.Indices32, a, b, d, a, d, c)
		}
	}
	return mustMesh(data)
}

// TerraceScene is the scene of the benchmark of a million triangles: one item that holds the surface of Terraces(708, 12), drawn
// with its edges, and the camera that sees all of it from above at an angle.
func TerraceScene(w, h int) (*viewport3d.Scene, *viewport3d.Camera) {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(Terraces(708, 12), viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: ui.NewColor(0xFFB8C4D0)})))
	scene.SetStyle(viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeShadedWithEdges})
	return scene, Isometric(scene, w, h)
}

// InstancedScene is the scene of ten thousand items that share a mesh and a material (one draw), in a square of 100 x 100.
func InstancedScene(w, h int) (*viewport3d.Scene, *viewport3d.Camera) {
	scene := viewport3d.NewScene()
	mesh, mat := Box(0.8, 0.8, 0.8), viewport3d.NewPhongMaterial()
	items := make([]*viewport3d.Item, 0, 10000)
	for i := 0; i < 10000; i++ {
		items = append(items, viewport3d.NewItem(mesh, mat, viewport3d.ItemOpts{Transform: translate(float64(i%100), float64(i/100), 0)}))
	}
	scene.Add(items...)
	return scene, Isometric(scene, w, h)
}

// SeparateScene is the scene of ten thousand items that each have a mesh of their own (a box of its own size), so that no two
// can be one draw.
func SeparateScene(w, h int) (*viewport3d.Scene, *viewport3d.Camera) {
	scene := viewport3d.NewScene()
	mat := viewport3d.NewPhongMaterial()
	items := make([]*viewport3d.Item, 0, 10000)
	for i := 0; i < 10000; i++ {
		size := 0.6 + float32(i%7)*0.02
		items = append(items, viewport3d.NewItem(Box(size, size, size), mat, viewport3d.ItemOpts{Transform: translate(float64(i%100), float64(i/100), 0)}))
	}
	scene.Add(items...)
	return scene, Isometric(scene, w, h)
}
