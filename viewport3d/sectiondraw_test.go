package viewport3d_test

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// The places of the planes in the frame's uniform data (the Frame of shaders/common.wgsl.inc): six planes of four floats, then their
// number.
const (
	framePlanes = 272
	frameCount  = 368
)

// uniformsOf returns the last write of the uniform buffer of the frame: the largest one, the instances being smaller.
func (d *drawRig) uniformsOf() []byte {
	d.t.Helper()
	var best []byte
	for _, w := range d.rec.Device().Writes {
		if len(w.Data) > len(best) {
			best = w.Data
		}
	}
	if len(best) < frameCount+4 {
		d.t.Fatalf("no uniform data of the frame: %d bytes", len(best))
	}
	return best
}

func floatAt(b []byte, off int) float64 {
	return float64(math.Float32frombits(binary.LittleEndian.Uint32(b[off:])))
}

func TestTheSectionPlanesReachTheShadersRelativeToTheEye(t *testing.T) {
	far := scenetest.FarOrigin
	tests := []struct {
		name   string
		target vecmath.Vec3
		planes []vecmath.Plane
	}{
		{"a plane through the part", vecmath.Vec3{}, []vecmath.Plane{vecmath.NewPlaneFromPointNormal(vecmath.Vec3{X: 0.3}, vecmath.Vec3X)}},
		{"the plane of the target, seen from 10 units", vecmath.Vec3{}, []vecmath.Plane{vecmath.NewPlaneFromPointNormal(vecmath.Vec3{}, vecmath.Vec3Z)}},
		{"two planes", vecmath.Vec3{}, []vecmath.Plane{vecmath.NewPlaneFromPointNormal(vecmath.Vec3{Y: 0.1}, vecmath.Vec3Y), vecmath.NewPlaneFromPointNormal(vecmath.Vec3{Z: -0.2}, vecmath.Vec3Z.Neg())}},
		{"a plane 50 km from the origin", far, []vecmath.Plane{vecmath.NewPlaneFromPointNormal(far.Add(vecmath.Vec3{X: 0.002}), vecmath.Vec3X)}},
		{"no plane", vecmath.Vec3{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := viewport3d.NewScene()
			scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial(), viewport3d.ItemOpts{Transform: foundation.Some(vecmath.NewMat4Translation(tt.target))}))
			scene.SetSectionPlanes(tt.planes...)
			cam := frontCamera()
			cam.SetTarget(tt.target)
			cam.SetDistance(10)
			rig := newDrawRigDepth(t, scene, cam, gpu.TextureFormatDepth24PlusStencil8)
			rig.redraw()
			u := rig.uniformsOf()
			if got := floatAt(u, frameCount); got != float64(len(tt.planes)) {
				t.Errorf("%v planes in the frame, want %d", got, len(tt.planes))
			}
			rot := cam.RelativeViewMatrix()
			eye := cam.Eye()
			for i, p := range tt.planes {
				n := rot.TransformDirection(p.Normal)
				d := p.D + p.Normal.Dot(eye)
				got := [4]float64{floatAt(u, framePlanes+16*i), floatAt(u, framePlanes+16*i+4), floatAt(u, framePlanes+16*i+8), floatAt(u, framePlanes+16*i+12)}
				want := [4]float64{n.X, n.Y, n.Z, d}
				for k := range got {
					if math.Abs(got[k]-want[k]) > 1e-6 {
						t.Errorf("plane %d = %v, want %v", i, got, want)
						break
					}
				}
			}
		})
	}
}

func TestTheCapsAndTheOutlineAreDrawnForTheClosedItemsThePlanesCut(t *testing.T) {
	openData := scenetest.Box(1, 1, 1).Data()
	openData.Indices16 = openData.Indices16[:len(openData.Indices16)-6]
	open, err := viewport3d.NewMesh(openData)
	if err != nil {
		t.Fatal(err)
	}
	cut := vecmath.NewPlaneFromPointNormal(vecmath.Vec3{X: 0.1}, vecmath.Vec3X)
	tests := []struct {
		name               string
		mesh               *viewport3d.Mesh
		style              viewport3d.SceneStyle
		depth              gpu.TextureFormat
		planes             []vecmath.Plane
		wantCaps, wantOutl int
	}{
		{"a closed item", scenetest.Box(1, 1, 1), viewport3d.SceneStyle{}, gpu.TextureFormatDepth24PlusStencil8, []vecmath.Plane{cut}, 1, 1},
		{"two planes", scenetest.Box(1, 1, 1), viewport3d.SceneStyle{}, gpu.TextureFormatDepth24PlusStencil8, []vecmath.Plane{cut, vecmath.NewPlaneFromPointNormal(vecmath.Vec3{Y: 0.1}, vecmath.Vec3Y)}, 2, 2},
		{"no cap in the style", scenetest.Box(1, 1, 1), viewport3d.SceneStyle{Section: viewport3d.SectionStyle{NoCap: true}}, gpu.TextureFormatDepth32Float, []vecmath.Plane{cut}, 0, 1},
		{"no outline in the style", scenetest.Box(1, 1, 1), viewport3d.SceneStyle{Section: viewport3d.SectionStyle{NoOutline: true}}, gpu.TextureFormatDepth24PlusStencil8, []vecmath.Plane{cut}, 1, 0},
		{"an open item has no cap", open, viewport3d.SceneStyle{}, gpu.TextureFormatDepth24PlusStencil8, []vecmath.Plane{cut}, 0, 1},
		{"no stencil in the depth target", scenetest.Box(1, 1, 1), viewport3d.SceneStyle{}, gpu.TextureFormatDepth32Float, []vecmath.Plane{cut}, 0, 1},
		{"no plane", scenetest.Box(1, 1, 1), viewport3d.SceneStyle{}, gpu.TextureFormatDepth24PlusStencil8, nil, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := viewport3d.NewScene()
			scene.SetStyle(tt.style)
			scene.Add(viewport3d.NewItem(tt.mesh, viewport3d.NewPhongMaterial()))
			scene.SetSectionPlanes(tt.planes...)
			rig := newDrawRigDepth(t, scene, frontCamera(), tt.depth)
			st := rig.r.Stats()
			if st.CapDraws != tt.wantCaps || st.OutlineDraws != tt.wantOutl {
				t.Errorf("cap draws %d, outline draws %d; want %d, %d", st.CapDraws, st.OutlineDraws, tt.wantCaps, tt.wantOutl)
			}
		})
	}
}

func TestAViewAddsAStencilToItsDepthTargetWhileThePlanesCutIt(t *testing.T) {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()))
	ts, rec := newViewTester(t, scene)
	hasStencil := func() bool {
		stencil := false
		for _, c := range rec.Calls() {
			if strings.HasPrefix(c, "NewTexture(") && strings.Contains(c, "format=") && textureFormatOf(c) == explicitDepthStencil {
				stencil = true
			}
		}
		return stencil
	}
	if hasStencil() {
		t.Fatalf("a scene with no planes has a stencil")
	}
	rec.Reset()
	scene.SetSectionPlanes(vecmath.NewPlaneFromPointNormal(vecmath.Vec3{}, vecmath.Vec3X))
	ts.Pump(frame)
	if !hasStencil() {
		t.Errorf("a scene with a plane has no stencil in its depth target")
	}
	rec.Reset()
	scene.SetSectionPlanes()
	ts.Pump(frame)
	if hasStencil() {
		t.Errorf("a scene without planes still draws into a stencil target")
	}
}
