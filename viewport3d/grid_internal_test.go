package viewport3d

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

func TestTheGridResolvesItsDefaults(t *testing.T) {
	dark, light := ui.NewColor(0xFF22272E), ui.NewColor(0xFFF4F4F4)
	red := ui.NewColor(0xFFFF0000)
	tests := []struct {
		name       string
		style      GridStyle
		background ui.Color
		check      func(t *testing.T, r resolvedGrid)
	}{
		{"not shown", GridStyle{}, dark, func(t *testing.T, r resolvedGrid) {
			if r.show {
				t.Errorf("the zero style shows a grid")
			}
		}},
		{"the defaults", GridStyle{Show: true}, dark, func(t *testing.T, r resolvedGrid) {
			if r.normal != vecmath.Vec3Y || r.spacing != 1 || r.majorEvery != 10 || r.width != 1 || r.fade != 200 {
				t.Errorf("normal %v, spacing %v, major %v, width %v, fade %v", r.normal, r.spacing, r.majorEvery, r.width, r.fade)
			}
			if r.minor.A >= r.major.A || r.minor.A <= 0 {
				t.Errorf("the minor lines are %v opaque and the major ones %v: the major are the stronger", r.minor.A, r.major.A)
			}
			if r.minor.ComputeLuminance() < 0.5 {
				t.Errorf("the lines on a dark background are dark: %v", r.minor)
			}
		}},
		{"on a light background the lines are dark", GridStyle{Show: true}, light, func(t *testing.T, r resolvedGrid) {
			if r.minor.ComputeLuminance() > 0.5 || r.major.ComputeLuminance() > 0.5 {
				t.Errorf("the lines on a light background are light: %v %v", r.minor, r.major)
			}
		}},
		{"the fade follows the spacing", GridStyle{Show: true, Spacing: 0.25}, dark, func(t *testing.T, r resolvedGrid) {
			if r.fade != 50 {
				t.Errorf("fade = %v, want 200 spacings = 50", r.fade)
			}
		}},
		{"a normal that is not a unit vector, and no normal at all", GridStyle{Show: true, Normal: vecmath.Vec3{Z: 5}}, dark, func(t *testing.T, r resolvedGrid) {
			if r.normal != vecmath.Vec3Z {
				t.Errorf("normal = %v", r.normal)
			}
		}},
		{"numbers that cannot be used", GridStyle{Show: true, Spacing: -3, MajorEvery: -1, LineWidth: -2, FadeDistance: math.NaN(), Normal: vecmath.Vec3{X: math.NaN()}}, dark, func(t *testing.T, r resolvedGrid) {
			if r.spacing != 1 || r.majorEvery != 10 || r.width != 1 || r.fade != 200 || r.normal != vecmath.Vec3Y {
				t.Errorf("%+v", r)
			}
		}},
		{"what is given", GridStyle{Show: true, Spacing: 2, MajorEvery: 4, LineWidth: 3, FadeDistance: 9, MinorColor: foundation.Some(red), MajorColor: foundation.Some(red)}, dark,
			func(t *testing.T, r resolvedGrid) {
				if r.spacing != 2 || r.majorEvery != 4 || r.width != 3 || r.fade != 9 || r.minor != red || r.major != red {
					t.Errorf("%+v", r)
				}
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.check(t, tt.style.resolve(tt.background)) })
	}
}

func TestTheAxesOfTheGridAreOrthonormalAndInThePlane(t *testing.T) {
	tests := []struct {
		name   string
		normal vecmath.Vec3
		u      vecmath.Vec3
	}{
		{"y", vecmath.Vec3Y, vecmath.Vec3X},
		{"-y", vecmath.Vec3Y.Neg(), vecmath.Vec3X},
		{"z", vecmath.Vec3Z, vecmath.Vec3X},
		{"x", vecmath.Vec3X, vecmath.Vec3Y},
		{"a slanted plane", vecmath.Vec3{X: 1, Y: 2, Z: 3}.Normalize(), vecmath.Vec3{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, v := gridAxes(tt.normal)
			if math.Abs(u.Length()-1) > 1e-12 || math.Abs(v.Length()-1) > 1e-12 || math.Abs(u.Dot(v)) > 1e-12 || math.Abs(u.Dot(tt.normal)) > 1e-12 || math.Abs(v.Dot(tt.normal)) > 1e-12 {
				t.Errorf("u %v, v %v are not orthonormal axes of the plane of %v", u, v, tt.normal)
			}
			if tt.u != (vecmath.Vec3{}) && !nearVec(u, tt.u, 1e-12) {
				t.Errorf("u = %v, want the world's axis %v", u, tt.u)
			}
			// Right handed: u x v is the normal.
			if !nearVec(u.Cross(v), tt.normal, 1e-12) {
				t.Errorf("u x v = %v, want the normal %v", u.Cross(v), tt.normal)
			}
		})
	}
}

func floatAtBytes(b []byte, off int) float64 {
	return float64(math.Float32frombits(binary.LittleEndian.Uint32(b[off:])))
}

func TestTheGridRecordIsRelativeToTheEyeInFloat64(t *testing.T) {
	far := vecmath.Vec3{X: 50000, Y: -30000, Z: 70000}
	tests := []struct {
		name   string
		origin vecmath.Vec3
		eye    vecmath.Vec3
		normal vecmath.Vec3
	}{
		{"near the origin", vecmath.Vec3{Y: -0.5}, vecmath.Vec3{X: 3, Y: 4, Z: 5}, vecmath.Vec3Y},
		{"50 km from it", far, far.Add(vecmath.Vec3{X: 3.3, Y: 4.1, Z: -5.7}), vecmath.Vec3Y},
		{"a slanted plane far away", far, far.Add(vecmath.Vec3{X: -2, Y: 6, Z: 1}), vecmath.Vec3{X: 1, Y: 1, Z: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := GridStyle{Show: true, Origin: tt.origin, Normal: tt.normal, Spacing: 0.5, MajorEvery: 5}.resolve(ui.NewColor(0xFF000000))
			// A camera that looks from the eye at its target.
			cam := NewCamera(CameraOpts{Target: tt.eye.Add(vecmath.Vec3{X: -0.3, Y: -0.4, Z: -2.5}), Distance: 2.5})
			cam.SetOrientation(vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 0.3, Y: 1, Z: 0.1}, 0.7))
			eye := cam.Eye()
			rot := cam.RelativeViewMatrix()
			rec := make([]byte, 112)
			putGrid(rec, 0, &g, &rot, eye, 2)
			n := vecmath.Vec3{X: floatAtBytes(rec, 0), Y: floatAtBytes(rec, 4), Z: floatAtBytes(rec, 8)}
			d := floatAtBytes(rec, 12)
			uv := vecmath.Vec3{X: floatAtBytes(rec, 16), Y: floatAtBytes(rec, 20), Z: floatAtBytes(rec, 24)}
			vv := vecmath.Vec3{X: floatAtBytes(rec, 32), Y: floatAtBytes(rec, 36), Z: floatAtBytes(rec, 40)}
			eu, ev := floatAtBytes(rec, 28), floatAtBytes(rec, 44)
			if w := floatAtBytes(rec, 96); w != 2 {
				t.Errorf("the width is %v pixels, want 1 logical pixel at twice the scale = 2", w)
			}
			if floatAtBytes(rec, 48) != 0.5 || floatAtBytes(rec, 52) != 5 {
				t.Errorf("spacing %v, major %v", floatAtBytes(rec, 48), floatAtBytes(rec, 52))
			}
			period := g.period()
			if math.Abs(eu) > period+1e-6 || math.Abs(ev) > period+1e-6 {
				t.Errorf("the eye's coordinates %v, %v are not reduced to the period %v", eu, ev, period)
			}
			// Points of the plane near the eye, which the shader works out from the view space, must have the coordinates that they
			// have in the plane, up to a whole number of periods.
			for _, off := range []vecmath.Vec3{{X: 0.4, Z: -1}, {X: -1.3, Z: 0.2}, {X: 2.2, Z: 3.1}} {
				// A point of the plane: the one nearest to the eye plus offsets along the axes.
				foot := eye.Sub(g.normal.Scale(g.normal.Dot(eye.Sub(g.origin))))
				w := foot.Add(g.u.Scale(off.X)).Add(g.v.Scale(off.Z))
				p := rot.TransformDirection(w.Sub(eye))
				if dist := n.Dot(p) + d; math.Abs(dist) > 2e-3 {
					t.Errorf("a point of the plane is %v off it in the record's view space", dist)
				}
				gotU, gotV := p.Dot(uv)+eu, p.Dot(vv)+ev
				wantU, wantV := w.Sub(g.origin).Dot(g.u), w.Sub(g.origin).Dot(g.v)
				if !wholePeriods(gotU-wantU, period, 5e-3) || !wholePeriods(gotV-wantV, period, 5e-3) {
					t.Errorf("the shader's coordinates (%v, %v) differ from the plane's (%v, %v) by more than whole periods of %v", gotU, gotV, wantU, wantV, period)
				}
			}
		})
	}
}

func wholePeriods(diff, period, tol float64) bool {
	k := math.Round(diff / period)
	return math.Abs(diff-k*period) <= tol
}
