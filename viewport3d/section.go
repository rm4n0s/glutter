package viewport3d

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

// MaxSectionPlanes is how many section planes a scene has.
//
// Flutter: no counterpart
const MaxSectionPlanes = 6

// TagSectionInvalid is the tag of the panic of SetSectionPlanes for more planes than MaxSectionPlanes or a plane whose normal
// has no length. Programmer errors panic in debug builds (SPEC §14); a release build keeps the first planes it can use.
//
// Flutter: no counterpart
const TagSectionInvalid = "SectionInvalid"

// SectionPlanes are the planes that cut the scene (SPEC §27.8, "Section planes").
//
// Flutter: no counterpart
func (s *Scene) SectionPlanes() []vecmath.Plane { return s.planes }

// SetSectionPlanes sets the planes that cut the scene, at most MaxSectionPlanes, in world coordinates in float64. A plane cuts
// away what is on the side its normal points to (the points where Plane.DistanceTo is positive), in the faces, the lines and the
// points; where the solid meshes are cut, a cap closes them and an outline marks the cut (SceneStyle.Section). A normal that is
// not a unit vector is made one. Each frame the planes are made relative to the eye, in float64, like the models (D30). It panics
// with SectionInvalid in debug builds for too many planes or a normal with no length.
//
// Flutter: no counterpart
func (s *Scene) SetSectionPlanes(planes ...vecmath.Plane) {
	if foundation.DebugMode {
		if len(planes) > MaxSectionPlanes {
			panic(errors.New(TagSectionInvalid, "a scene has at most six section planes", "planes", len(planes)))
		}
		for i, p := range planes {
			if p.Normal.LengthSq() == 0 || !p.Normal.IsFinite() || !isFinite(p.D) {
				panic(errors.New(TagSectionInvalid, "a section plane has a normal with a length and a finite distance", "plane", i))
			}
		}
	}
	next := make([]vecmath.Plane, 0, len(planes))
	for _, p := range planes {
		n := p.Normal.Length()
		if len(next) == MaxSectionPlanes || n == 0 || !p.Normal.IsFinite() || !isFinite(p.D) {
			continue
		}
		next = append(next, vecmath.Plane{Normal: p.Normal.Scale(1 / n), D: p.D / n})
	}
	s.planes = next
	s.NotifyListeners()
}

func isFinite(f float64) bool { return f-f == 0 }

// putSectionPlanes writes the planes into the frame's uniform data, in the view space of the eye: the normal turned by the
// rotation of the view, and the distance moved to the eye (D30).
func putSectionPlanes(b []byte, planes []vecmath.Plane, rot *vecmath.Mat4, eye vecmath.Vec3) {
	for i, p := range planes {
		n := rot.TransformDirection(p.Normal)
		d := p.D + p.Normal.Dot(eye)
		putVec4(b, frameSectionOffset+16*i, float32(n.X), float32(n.Y), float32(n.Z), float32(d))
	}
	for i := len(planes); i < MaxSectionPlanes; i++ {
		putVec4(b, frameSectionOffset+16*i, 0, 0, 0, 0)
	}
	putVec4(b, frameSectionOffset+16*MaxSectionPlanes, float32(len(planes)), 0, 0, 0)
}

// SectionStyle is how a cut looks (SceneStyle.Section).
//
// Flutter: no counterpart
type SectionStyle struct {
	// NoCap leaves the solid meshes open where a plane cuts them; by default a cap closes each one.
	NoCap bool
	// CapColor is the color of the caps; the default is the color of the cut item's material.
	CapColor foundation.Opt[ui.Color]
	// NoOutline leaves out the outline of the cut.
	NoOutline bool
	// OutlineColor is the color of the outline; the default is the color of the edges.
	OutlineColor foundation.Opt[ui.Color]
	// OutlineWidth is the width of the outline in logical pixels; the default is twice the width of the edges.
	OutlineWidth float64
}

// resolvedSection is a SectionStyle with its defaults filled in; capColor and outlineColor are none while the defaults stand.
type resolvedSection struct {
	cap, outline bool
	capColor     foundation.Opt[ui.Color]
	outlineColor ui.Color
	outlineWidth float64
}

// resolve fills in the defaults of the style, for the edge color and width the scene has.
func (s SectionStyle) resolve(edgeColor ui.Color, edgeWidth float64) resolvedSection {
	r := resolvedSection{cap: !s.NoCap, outline: !s.NoOutline, capColor: s.CapColor, outlineColor: s.OutlineColor.OrElse(edgeColor), outlineWidth: s.OutlineWidth}
	if r.outlineWidth <= 0 {
		r.outlineWidth = 2 * edgeWidth
	}
	return r
}

// Closed says whether a mesh of triangles is a solid: every edge between its triangles (vertices at the same position count as
// one) belongs to exactly two of them. A closed mesh gets a cap where a section plane cuts it. It is worked out once and kept.
//
// Flutter: no counterpart
func (m *Mesh) Closed() bool {
	if m.closed == closedUnknown {
		m.closed = closedNo
		if m.data.Primitive == PrimitiveTriangles {
			entries, _ := triangleEdges(m)
			closed := len(entries) > 0
			for i := 0; i < len(entries) && closed; {
				j := i + 1
				for j < len(entries) && entries[j].key == entries[i].key {
					j++
				}
				closed = j-i == 2
				i = j
			}
			if closed {
				m.closed = closedYes
			}
		}
	}
	return m.closed == closedYes
}

// The states of Mesh.closed.
const (
	closedUnknown int8 = iota
	closedYes
	closedNo
)
