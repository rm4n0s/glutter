package viewport3d_test

import (
	"context"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/viewport3d"
)

// cloudMesh is a mesh of points, with a color for each when the colors are given.
func cloudMesh(t *testing.T, colors []byte, indices []uint16) *viewport3d.Mesh {
	t.Helper()
	m, err := viewport3d.NewMesh(viewport3d.MeshData{
		Primitive: viewport3d.PrimitivePoints, Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1}, Colors: colors, Indices16: indices,
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestPointMaterialVertexColors(t *testing.T) {
	m := viewport3d.NewPointMaterial()
	if m.VertexColors() {
		t.Error("a point material takes the colors of the points by default")
	}
	if !viewport3d.NewPointMaterial(viewport3d.PointOpts{VertexColors: true}).VertexColors() {
		t.Error("the option is not kept")
	}
	changes := 0
	m.AddListener(func() { changes++ })
	m.SetVertexColors(true)
	m.SetVertexColors(true)
	m.SetVertexColors(false)
	if changes != 2 || m.VertexColors() {
		t.Errorf("%d changes, colors %v: two changes and off", changes, m.VertexColors())
	}
}

// TestPointsWithColorsAreDrawnWithTheirColors checks that the colors are a buffer of their own that the draw binds, when the material
// asks for them and the mesh has them, and that a draw without them binds none.
func TestPointsWithColorsAreDrawnWithTheirColors(t *testing.T) {
	colors := []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 255, 255, 255, 255}
	tests := []struct {
		name string
		mesh *viewport3d.Mesh
		mat  *viewport3d.PointMaterial
		// buffers is the number of vertex buffers of the draw of the points: the positions, the corners and the colors.
		buffers int
	}{
		{"colors asked and there", cloudMesh(t, colors, nil), viewport3d.NewPointMaterial(viewport3d.PointOpts{VertexColors: true}), 3},
		{"colors asked and indexed", cloudMesh(t, colors, []uint16{3, 1, 2}), viewport3d.NewPointMaterial(viewport3d.PointOpts{VertexColors: true}), 3},
		{"colors not asked", cloudMesh(t, colors, nil), viewport3d.NewPointMaterial(), 2},
		{"colors asked and none there", cloudMesh(t, nil, nil), viewport3d.NewPointMaterial(viewport3d.PointOpts{VertexColors: true}), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := viewport3d.NewScene()
			scene.Add(viewport3d.NewItem(tt.mesh, tt.mat))
			rig := newDrawRig(t, scene, scenetest.Isometric(scene, 200, 100))
			calls := rig.redraw()
			var buffers []string
			for i, c := range calls {
				if c == "SetPipeline(shader=point)" {
					for _, d := range calls[i+1:] {
						if strings.HasPrefix(d, "Draw(") {
							break
						}
						if strings.HasPrefix(d, "SetVertexBuffer(") {
							buffers = append(buffers, d)
						}
					}
				}
			}
			if len(buffers) != tt.buffers {
				t.Errorf("the points draw with %d vertex buffers, want %d: %v", len(buffers), tt.buffers, buffers)
			}
			if st := rig.r.Stats(); st.Draws == 0 {
				t.Errorf("nothing was drawn: %+v", st)
			}
		})
	}
}

func TestUploadShowsTheColorsOfAPointCloud(t *testing.T) {
	cloud := func(colors []byte) *model.Model {
		mesh := &model.Mesh{Primitive: model.PrimitivePoints, Positions: []float32{0, 0, 0, 1, 0, 0}, Indices: []uint32{0, 1}, Colors: colors}
		node := model.NewNode("cloud")
		node.Meshes = []*model.Mesh{mesh}
		return &model.Model{Nodes: []*model.Node{node}, Meshes: []*model.Mesh{mesh}}
	}
	tests := []struct {
		name   string
		colors []byte
		want   bool
	}{{"a cloud with colors", []byte{255, 0, 0, 255, 0, 255, 0, 255}, true}, {"a cloud with none", nil, false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rig := newDrawRig(t, viewport3d.NewScene(), viewport3d.NewCamera())
			u, err := viewport3d.Upload(context.Background(), rig.frame.Device, cloud(tt.colors), viewport3d.UploadOpts{})
			if err != nil {
				t.Fatal(err)
			}
			defer u.Release()
			pm, ok := u.Parts[0].Item.Material().(*viewport3d.PointMaterial)
			if !ok || pm.VertexColors() != tt.want {
				t.Errorf("material %T, vertex colors %v, want %v", u.Parts[0].Item.Material(), ok && pm.VertexColors(), tt.want)
			}
		})
	}
}

// TestLeaksPointCloud opens a cloud of points with colors, as a PLY file gives it, draws it and closes it, 100 times: the buffers of
// the positions and of the colors are let go of with the mesh.
func TestLeaksPointCloud(t *testing.T) {
	scene := viewport3d.NewScene()
	rig := newDrawRigOpts(t, scene, frontCamera(), gpu.TextureFormatDepth32Float, gluttertest.Opts{GPU: true, DiscardScenes: true})
	rec := rig.rec
	rec.Device().CompleteReadsOnSubmit, rec.Device().Quiet = true, true
	dev := rig.frame.Device
	var colors []byte
	var positions []float32
	var indices []uint32
	for i := 0; i < 500; i++ {
		positions = append(positions, float32(i%20)*0.1, float32(i/20)*0.1, 0)
		colors = append(colors, byte(i), byte(i/2), byte(i/4), 255)
		indices = append(indices, uint32(i))
	}
	m := &model.Model{}
	mesh := &model.Mesh{Primitive: model.PrimitivePoints, Positions: positions, Colors: colors, Indices: indices}
	node := model.NewNode("cloud")
	node.Meshes = []*model.Mesh{mesh}
	m.Nodes, m.Meshes = []*model.Node{node}, []*model.Mesh{mesh}
	openClose := func() {
		u, err := viewport3d.Upload(context.Background(), dev, m, viewport3d.UploadOpts{})
		if err != nil {
			t.Fatal(err)
		}
		scene.Add(u.Items()...)
		rig.redraw()
		rec.CompleteReads()
		if st := rig.r.Stats(); st.Draws == 0 {
			t.Fatalf("the cloud drew nothing: %+v", st)
		}
		scene.Remove(u.Items()...)
		u.Release()
		rig.redraw()
		rec.CompleteReads()
		rig.ts.ClearLogs()
		rec.ClearLogs()
	}
	openClose()
	base := dev.Resources().Total()
	baseLive := rec.Device().Live()
	leaktest.Run(t, "viewport3d/point-cloud", func() func() { return openClose },
		leaktest.Opts{N: 100, Warmup: 10, Counters: leakCounters(rig.ts, rec)})
	if got := dev.Resources().Total(); got != base {
		t.Errorf("a cloud opened and closed 400 times left %d GPU resources alive (%d before)", got, base)
	}
	if got := rec.Device().Live(); got != baseLive {
		t.Errorf("the device holds %+v after the cloud was closed 400 times, %+v before", got, baseLive)
	}
}
