package viewport3d

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

func nearVec(a, b vecmath.Vec3, tol float64) bool { return a.Sub(b).Length() <= tol }

func TestTheCellsOfTheCubeGoToTwentySixViews(t *testing.T) {
	seen := map[[3]int]cubeCell{}
	kinds := [3]int{}
	for face := range cubeFaces {
		for ia := -1; ia <= 1; ia++ {
			for ib := -1; ib <= 1; ib++ {
				c := cubeCell{face: face, ia: ia, ib: ib, valid: true}
				d := c.direction()
				key := [3]int{int(math.Round(d.X)), int(math.Round(d.Y)), int(math.Round(d.Z))}
				// The cell of a face goes to the direction of the face's normal; an edge or a corner to the sum of the faces' normals.
				if _, dup := seen[key]; dup && c.kind() == 0 {
					t.Errorf("two faces go to %v", key)
				}
				seen[key] = c
				kinds[c.kind()]++
				axes := 0
				for _, v := range key {
					if v != 0 {
						axes++
					}
				}
				if axes != c.kind()+1 {
					t.Errorf("cell %+v goes to %v, which is on %d axes", c, key, axes)
				}
			}
		}
	}
	if len(seen) != 26 {
		t.Errorf("the cells go to %d directions, want the 26 of a cube (6 faces, 12 edges, 8 corners)", len(seen))
	}
	// Each cell of a face is one of nine: 6 centers, 24 edge strips (each edge is two strips) and 24 corner squares (each corner is
	// three).
	if kinds != [3]int{6, 24, 24} {
		t.Errorf("kinds of cells %v, want [6 24 24]", kinds)
	}
}

func TestTheViewDirectionOrientationIsTheStandardViewsWhenTheDirectionIsTheirs(t *testing.T) {
	for _, up := range []vecmath.Vec3{vecmath.Vec3Y, vecmath.Vec3Z} {
		cam := NewCamera(CameraOpts{Up: up})
		right := cam.up.Cross(cam.front)
		tests := []struct {
			view StandardView
			dir  vecmath.Vec3
		}{
			{StandardViewFront, cam.front},
			{StandardViewBack, cam.front.Neg()},
			{StandardViewRight, right},
			{StandardViewLeft, right.Neg()},
			{StandardViewTop, cam.up},
			{StandardViewBottom, cam.up.Neg()},
			{StandardViewIsometric, cam.front.Add(right).Add(cam.up)},
		}
		for _, tt := range tests {
			want, got := cam.StandardViewOrientation(tt.view), cam.ViewDirectionOrientation(tt.dir.Scale(3))
			// The same orientation when it turns the same vectors the same way (a quaternion and its negative are one).
			for _, v := range []vecmath.Vec3{vecmath.Vec3X, vecmath.Vec3Y, vecmath.Vec3Z} {
				if !nearVec(want.Rotate(v), got.Rotate(v), 1e-9) {
					t.Errorf("up %v, %v: the axis %v goes to %v, want %v", up, tt.view, v, got.Rotate(v), want.Rotate(v))
				}
			}
		}
	}
	cam := NewCamera()
	if got := cam.ViewDirectionOrientation(vecmath.Vec3{}); got != cam.Orientation() {
		t.Errorf("a direction with no length gave %v, want the camera's own orientation", got)
	}
	// The viewer is in the direction asked, and up is up in the view for any direction that is not along the up axis.
	got := cam.ViewDirectionOrientation(vecmath.Vec3{X: 1, Y: 0.4, Z: 1})
	back := got.Rotate(vecmath.Vec3Z)
	if !nearVec(back, vecmath.Vec3{X: 1, Y: 0.4, Z: 1}.Normalize(), 1e-9) {
		t.Errorf("the viewer is at %v", back)
	}
	if up := got.Rotate(vecmath.Vec3Y); up.Y <= 0 || math.Abs(got.Rotate(vecmath.Vec3X).Y) > 1e-9 {
		t.Errorf("the view's up is %v and its right has a vertical part", up)
	}
}

func TestTheCubeUnderAPointOfThePicture(t *testing.T) {
	const size = 104.0
	half := size * cubeHalf
	c := ui.Offset{Dx: size / 2, Dy: size / 2}
	front := NewCamera()
	front.SetStandardView(StandardViewFront)
	iso := NewCamera()
	iso.SetStandardView(StandardViewIsometric)
	at := func(a, b float64) ui.Offset { return ui.Offset{Dx: c.Dx + a*half, Dy: c.Dy - b*half} }
	tests := []struct {
		name  string
		cam   *Camera
		point ui.Offset
		want  cubeCell
		valid bool
	}{
		{"the middle of the front face", front, at(0, 0), cubeCell{face: cubeFront}, true},
		{"its right strip, the edge with the right face", front, at(0.85, 0), cubeCell{face: cubeFront, ia: 1}, true},
		{"its top strip", front, at(0, 0.85), cubeCell{face: cubeFront, ib: 1}, true},
		{"its top right corner", front, at(0.85, 0.85), cubeCell{face: cubeFront, ia: 1, ib: 1}, true},
		{"its bottom left corner", front, at(-0.85, -0.85), cubeCell{face: cubeFront, ia: -1, ib: -1}, true},
		{"just inside the strip", front, at(0.55, 0), cubeCell{face: cubeFront}, true},
		{"just outside the cube", front, at(1.05, 0), cubeCell{}, false},
		{"far outside", front, ui.Offset{Dx: -20, Dy: 300}, cubeCell{}, false},
		{"the middle of the cube seen from a corner is a corner of the nearest face", iso, c, cubeCell{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newCubeGeometry(tt.cam, size)
			got := g.cellAt(tt.point)
			if got.valid != tt.valid {
				t.Fatalf("cell %+v, valid %v want %v", got, got.valid, tt.valid)
			}
			if tt.valid && tt.cam == front && got != (cubeCell{face: tt.want.face, ia: tt.want.ia, ib: tt.want.ib, valid: true}) {
				t.Errorf("cell %+v, want %+v", got, tt.want)
			}
		})
	}
	// Seen from the corner, three faces are seen, each by a third of the cube: the point at the middle is on the corner of all
	// three, and each face is hit by a point inside it.
	g := newCubeGeometry(iso, size)
	faces := map[int]bool{}
	for _, f := range g.visible() {
		faces[f.index] = true
		inside := f.at(0, 0)
		if got := g.cellAt(inside); !got.valid || got.face != f.index || got.kind() != 0 {
			t.Errorf("the middle of face %d is the cell %+v", f.index, got)
		}
	}
	if len(faces) != 3 || !faces[cubeFront] || !faces[cubeRight] || !faces[cubeTop] {
		t.Errorf("the faces seen from the corner are %v, want the front, the right and the top", faces)
	}
}

func TestOnlyTheFacesThatLookAtTheViewerAreSeen(t *testing.T) {
	tests := []struct {
		name string
		view StandardView
		want []int
	}{
		{"front", StandardViewFront, []int{cubeFront}},
		{"back", StandardViewBack, []int{cubeBack}},
		{"left", StandardViewLeft, []int{cubeLeft}},
		{"top", StandardViewTop, []int{cubeTop}},
		{"bottom", StandardViewBottom, []int{cubeBottom}},
	}
	for _, tt := range tests {
		cam := NewCamera()
		cam.SetStandardView(tt.view)
		g := newCubeGeometry(cam, 104)
		var got []int
		for _, f := range g.visible() {
			got = append(got, f.index)
		}
		if len(got) != len(tt.want) || got[0] != tt.want[0] {
			t.Errorf("%s: faces %v, want %v", tt.name, got, tt.want)
		}
	}
}
