package svg

import (
	"github.com/rm4n0s/glutter/ui"
)

// maxMarkers is the most markers that one shape draws; a path with more vertices draws the markers of the first
// ones. It keeps a path of a million vertices from recording a million copies of the marker.
const maxMarkers = 10_000

// markerRole says which marker property a vertex uses.
type markerRole uint8

const (
	markerStart markerRole = iota
	markerMid
	markerEnd
)

// drawMarkers draws the markers of a shape at its vertices: marker-start at the first, marker-end at the last and
// marker-mid at the others.
//
// SVG 1.1: painting.html#MarkerProperties
func (p *painter) drawMarkers(n *Node, alpha float64) {
	st := n.Style
	verts := n.Vertices
	if len(verts) < 2 || (st.MarkerStart == nil && st.MarkerMid == nil && st.MarkerEnd == nil) {
		return
	}
	drawn := 0
	for i, v := range verts {
		role, m := markerMid, st.MarkerMid
		switch i {
		case 0:
			role, m = markerStart, st.MarkerStart
		case len(verts) - 1:
			role, m = markerEnd, st.MarkerEnd
		}
		if m == nil {
			continue
		}
		if drawn++; drawn > maxMarkers {
			return
		}
		p.drawMarker(m, v, role, st.StrokeWidth, alpha)
	}
}

// drawMarker draws one marker at a vertex. The marker is turned by orient, scaled by the stroke width of the shape
// (markerUnits="strokeWidth"), and shifted so that its reference point lies on the vertex; its content is fitted in
// markerWidth x markerHeight by the viewBox and clipped to that viewport unless overflow says otherwise.
//
// SVG 1.1: painting.html#MarkerElement
func (p *painter) drawMarker(m *Marker, v Vertex, role markerRole, strokeWidth, alpha float64) {
	if p.active[m] || len(m.Children) == 0 {
		return
	}
	p.active[m] = true
	defer delete(p.active, m)
	angle := m.Angle
	switch m.Orient {
	case OrientAuto:
		angle = v.bisect()
	case OrientAutoStartReverse:
		angle = v.bisect()
		if role == markerStart {
			angle += 180
		}
	}
	p.save()
	defer p.restore()
	p.concat(Translate(v.X, v.Y).Mul(Rotate(angle)))
	if m.StrokeWidthUnits {
		p.concat(Scale(strokeWidth, strokeWidth))
	}
	content := Identity
	oldW, oldH := p.vw, p.vh
	defer func() { p.vw, p.vh = oldW, oldH }()
	p.vw, p.vh = m.Width, m.Height
	if m.ViewBox != nil {
		content = ViewBoxTransform(*m.ViewBox, m.PreserveAspectRatio, m.Width, m.Height)
		p.vw, p.vh = m.ViewBox.Width, m.ViewBox.Height
	}
	rx, ry := content.Apply(m.RefX, m.RefY)
	p.concat(Translate(-rx, -ry))
	if !m.Overflow {
		p.c.ClipRect(ui.NewRectFromLTWH(0, 0, m.Width, m.Height))
	}
	p.concat(content)
	for _, c := range m.Children {
		p.drawNode(c, alpha)
	}
}
