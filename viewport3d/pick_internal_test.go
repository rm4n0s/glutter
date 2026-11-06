package viewport3d

import (
	"encoding/binary"
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

func TestThePickMatrixTakesTheRectangleToTheWholeClipSpace(t *testing.T) {
	tests := []struct {
		name string
		rect image.Rectangle
		w, h int
	}{
		{"the whole frame", image.Rect(0, 0, 200, 100), 200, 100},
		{"the top left corner", image.Rect(0, 0, 10, 10), 200, 100},
		{"the bottom right corner", image.Rect(190, 90, 200, 100), 200, 100},
		{"the middle", image.Rect(95, 45, 106, 56), 200, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := pickMatrix(tt.rect, tt.w, tt.h)
			// The corners of the rectangle in the clip space of the whole frame, x right and y up.
			clip := func(px, py float64) (float64, float64) { return 2*px/float64(tt.w) - 1, 1 - 2*py/float64(tt.h) }
			x0, y0 := clip(float64(tt.rect.Min.X), float64(tt.rect.Min.Y))
			x1, y1 := clip(float64(tt.rect.Max.X), float64(tt.rect.Max.Y))
			for _, c := range []struct{ x, y, wantX, wantY float64 }{{x0, y0, -1, 1}, {x1, y1, 1, -1}, {x0, y1, -1, -1}, {x1, y0, 1, 1}} {
				p := m.TransformPoint(vecmath.Vec3{X: c.x, Y: c.y})
				if math.Abs(p.X-c.wantX) > 1e-12 || math.Abs(p.Y-c.wantY) > 1e-12 {
					t.Errorf("(%v, %v) goes to (%v, %v), want (%v, %v)", c.x, c.y, p.X, p.Y, c.wantX, c.wantY)
				}
			}
		})
	}
}

func TestThePickRectangleIsClippedToTheFrameAndFollowsTheScale(t *testing.T) {
	tests := []struct {
		name string
		q    pickRequest
		dpr  float64
		want image.Rectangle
	}{
		{"a point, radius 2", pickRequest{single: true, at: ui.Offset{Dx: 50.5, Dy: 40.2}, radius: 2}, 1, image.Rect(48, 38, 53, 43)},
		{"a point at twice the scale", pickRequest{single: true, at: ui.Offset{Dx: 50.5, Dy: 40.2}, radius: 2}, 2, image.Rect(97, 76, 106, 85)},
		{"a point at the corner is clipped", pickRequest{single: true, at: ui.Offset{Dx: 0.5, Dy: 99.5}, radius: 2}, 1, image.Rect(0, 97, 3, 100)},
		{"a point outside", pickRequest{single: true, at: ui.Offset{Dx: -50, Dy: -50}, radius: 2}, 1, image.Rectangle{}},
		{"a rectangle", pickRequest{rect: ui.NewRectFromLTRB(10.2, 20.7, 30.1, 40), radius: 0}, 1, image.Rect(10, 20, 31, 40)},
		{"a rectangle past the frame", pickRequest{rect: ui.NewRectFromLTRB(150, 50, 400, 400)}, 1, image.Rect(150, 50, 200, 100)},
		{"a rectangle outside", pickRequest{rect: ui.NewRectFromLTRB(300, 300, 400, 400)}, 1, image.Rectangle{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := int(200*tt.dpr), int(100*tt.dpr)
			got := pickRect(&tt.q, w, h, tt.dpr)
			if got != tt.want && (!got.Empty() || !tt.want.Empty()) {
				t.Errorf("rectangle %v, want %v", got, tt.want)
			}
		})
	}
}

// pickBytes makes the bytes a pick pass leaves: the numbers of the items and primitives of each texel (row bytes apart), and the depth.
type pickBytes struct {
	ids, depth       []byte
	idsRow, depthRow int
}

func newPickBytes(w, h int) *pickBytes {
	p := &pickBytes{idsRow: (w*8 + 255) / 256 * 256, depthRow: (w*4 + 255) / 256 * 256}
	p.ids, p.depth = make([]byte, p.idsRow*h), make([]byte, p.depthRow*h)
	return p
}

func (p *pickBytes) set(x, y int, item, prim uint32, depth float32) {
	binary.LittleEndian.PutUint32(p.ids[y*p.idsRow+8*x:], item)
	binary.LittleEndian.PutUint32(p.ids[y*p.idsRow+8*x+4:], prim)
	binary.LittleEndian.PutUint32(p.depth[y*p.depthRow+4*x:], math.Float32bits(depth))
}

// jobOf is the pick job that Draw would make for a request, for a camera.
func jobOf(t *testing.T, cam *Camera, q pickRequest, w, h int, items ...*Item) *pickJob {
	t.Helper()
	proj := cam.ProjectionMatrix()
	invProj, ok := proj.Inverse()
	if !ok {
		t.Fatal("the projection has no inverse")
	}
	rot := cam.RelativeViewMatrix()
	invRot, ok := rot.Inverse()
	if !ok {
		t.Fatal("the view has no inverse")
	}
	return &pickJob{req: q, rect: pickRect(&q, w, h, 1), invProj: invProj, invRot: invRot, eye: cam.Eye(), width: w, height: h, dpr: 1, items: items}
}

func TestThePickedPointIsMadeFromTheDepthInFloat64(t *testing.T) {
	far := vecmath.Vec3{X: 50000, Y: -30000, Z: 70000}
	for _, kc := range []struct {
		name string
		kind ProjectionKind
	}{{"perspective", ProjectionKindPerspective}, {"orthographic", ProjectionKindOrthographic}} {
		kind := kc.kind
		for _, tt := range []struct {
			name   string
			target vecmath.Vec3
		}{{"near the origin", vecmath.Vec3{}}, {"50 km from the origin", far}} {
			t.Run(kc.name+" "+tt.name, func(t *testing.T) {
				const w, h = 200, 100
				cam := NewCamera(CameraOpts{Projection: kind, Target: tt.target, Distance: 10})
				cam.SetViewportSize(ui.Size{Width: w, Height: h})
				// A point of the world, the pixel that shows it, and its depth: the pick finds the point of the pixel's center at that
				// depth, which the camera works out in its own way.
				p := tt.target.Add(vecmath.Vec3{X: 1.3, Y: -0.7, Z: 2})
				ndc, ok := cam.ProjectNDC(p)
				if !ok {
					t.Fatal("the point is behind the camera")
				}
				px, py := int((ndc.X+1)/2*w), int((1-ndc.Y)/2*h)
				center := ui.Offset{Dx: float64(px) + 0.5, Dy: float64(py) + 0.5}
				want, ok := cam.Unproject(center, ndc.Z)
				if !ok {
					t.Fatal("no point at that depth")
				}
				lines := &Item{mesh: &Mesh{data: MeshData{Primitive: PrimitiveLines}}, pickID: 42}
				job := jobOf(t, cam, pickRequest{single: true, at: center, radius: 2}, w, h, lines)
				bytes := newPickBytes(job.rect.Dx(), job.rect.Dy())
				bytes.set(px-job.rect.Min.X, py-job.rect.Min.Y, 1, 7, float32(ndc.Z))
				got := job.single(bytes.ids, bytes.depth, bytes.idsRow, bytes.depthRow)
				if !got.Hit || got.Item != lines || got.PickID != 42 || got.Primitive != PrimitiveLines || got.Index != 7 {
					t.Fatalf("result %+v", got)
				}
				// Float32 holds the depth to 6 or 7 digits; the world point is as good as that, at 10 units.
				if d := got.Point.Sub(want).Length(); d > 2e-3 {
					t.Errorf("point %v, want %v (off by %g)", got.Point, want, d)
				}
				if got.Normal != (vecmath.Vec3{}) {
					t.Errorf("a line has the normal %v", got.Normal)
				}
				if got.At != center {
					t.Errorf("at %v, want %v", got.At, center)
				}
			})
		}
	}
}

func TestThePickTakesTheNearestPixelThatHasAnItem(t *testing.T) {
	const w, h = 200, 100
	cam := NewCamera()
	cam.SetViewportSize(ui.Size{Width: w, Height: h})
	a, b := &Item{mesh: &Mesh{data: MeshData{Primitive: PrimitivePoints}}}, &Item{mesh: &Mesh{data: MeshData{Primitive: PrimitivePoints}}}
	at := ui.Offset{Dx: 100.5, Dy: 50.5}
	tests := []struct {
		name      string
		set       func(p *pickBytes, cx, cy int)
		wantHit   bool
		wantItem  *Item
		wantIndex int
	}{
		{"nothing", func(p *pickBytes, cx, cy int) {}, false, nil, 0},
		{"the center", func(p *pickBytes, cx, cy int) { p.set(cx, cy, 1, 3, 0.5) }, true, a, 3},
		{"the nearer of two", func(p *pickBytes, cx, cy int) { p.set(cx+2, cy, 1, 4, 0.5); p.set(cx, cy+1, 2, 9, 0.2) }, true, b, 9},
		{"the nearer in front of the farther at the same distance", func(p *pickBytes, cx, cy int) { p.set(cx+1, cy, 1, 4, 0.2); p.set(cx-1, cy, 2, 5, 0.6) }, true, b, 5},
		{"an item that is not in the pass is left out", func(p *pickBytes, cx, cy int) { p.set(cx, cy, 9, 1, 0.5); p.set(cx+2, cy, 1, 6, 0.5) }, true, a, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := jobOf(t, cam, pickRequest{single: true, at: at, radius: 2}, w, h, a, b)
			bytes := newPickBytes(job.rect.Dx(), job.rect.Dy())
			tt.set(bytes, int(at.Dx)-job.rect.Min.X, int(at.Dy)-job.rect.Min.Y)
			got := job.single(bytes.ids, bytes.depth, bytes.idsRow, bytes.depthRow)
			if got.Hit != tt.wantHit || got.Item != tt.wantItem || got.Index != tt.wantIndex {
				t.Errorf("hit %v item %p index %d; want %v %p %d", got.Hit, got.Item, got.Index, tt.wantHit, tt.wantItem, tt.wantIndex)
			}
		})
	}
}

func TestAPickWithoutDataIsNoHit(t *testing.T) {
	cam := NewCamera()
	cam.SetViewportSize(ui.Size{Width: 200, Height: 100})
	job := jobOf(t, cam, pickRequest{single: true, at: ui.Offset{Dx: 500, Dy: 500}, radius: 2}, 200, 100)
	if got := job.single(nil, nil, 0, 0); got.Hit {
		t.Errorf("hit %+v", got)
	}
	if got := job.many(nil, nil, 0); len(got.Items) != 0 {
		t.Errorf("items %v", got.Items)
	}
}

func TestAPickedRectangleHasEachVisibleItemOnceInTheOrderOfTheScene(t *testing.T) {
	cam := NewCamera()
	cam.SetViewportSize(ui.Size{Width: 200, Height: 100})
	a, b, c := &Item{}, &Item{}, &Item{}
	job := jobOf(t, cam, pickRequest{rect: ui.NewRectFromLTRB(10, 10, 20, 20)}, 200, 100, a, b, c)
	bytes := newPickBytes(job.rect.Dx(), job.rect.Dy())
	bytes.set(1, 1, 3, 0, 0.5)
	bytes.set(2, 2, 3, 1, 0.5)
	bytes.set(5, 5, 1, 0, 0.5)
	bytes.set(6, 6, 7, 0, 0.5)
	got := job.many(bytes.ids, bytes.depth, bytes.idsRow)
	if len(got.Items) != 2 || got.Items[0] != a || got.Items[1] != c {
		t.Errorf("items %v, want the first and the third once each", got.Items)
	}
}

func TestThePickedNormalFacesTheCamera(t *testing.T) {
	const w, h = 200, 100
	tri, err := NewMesh(MeshData{Positions: []float32{-5, -5, 0, 5, -5, 0, 0, 5, 0}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		z    float64
	}{{"seen from the front", 10}, {"seen from the back", -10}} {
		t.Run(tt.name, func(t *testing.T) {
			cam := NewCamera(CameraOpts{Distance: 10})
			if tt.z < 0 {
				cam.SetOrientation(vecmath.NewQuatAxisAngle(vecmath.Vec3Y, math.Pi))
			}
			cam.SetViewportSize(ui.Size{Width: w, Height: h})
			item := NewItem(tri, NewPhongMaterial())
			center := ui.Offset{Dx: 100.5, Dy: 50.5}
			ndc, _ := cam.ProjectNDC(vecmath.Vec3{})
			job := jobOf(t, cam, pickRequest{single: true, at: center, radius: 2}, w, h, item)
			bytes := newPickBytes(job.rect.Dx(), job.rect.Dy())
			bytes.set(int(center.Dx)-job.rect.Min.X, int(center.Dy)-job.rect.Min.Y, 1, 0, float32(ndc.Z))
			got := job.single(bytes.ids, bytes.depth, bytes.idsRow, bytes.depthRow)
			want := vecmath.Vec3{Z: math.Copysign(1, tt.z)}
			if !got.Hit || got.Normal.Sub(want).Length() > 1e-9 {
				t.Errorf("normal %v, want %v (result %+v)", got.Normal, want, got)
			}
		})
	}
}
