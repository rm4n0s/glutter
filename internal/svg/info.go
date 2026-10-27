package svg

import "github.com/rm4n0s/glutter/ui"

// ScaleDependent says whether the picture that the document records depends on the scale it is recorded at: it has a
// pattern (whose tile is rasterized at the device scale), a filter (whose buffer is), a stroke that does not scale, or
// an SVG image (which may have any of these). A picture of a document that is not scale dependent is the same at every
// scale, so a widget records it once and scales it; the others are recorded at the scale they are drawn at.
//
// SVG 1.1: pservers.html#Patterns, filters.html
func (d *Document) ScaleDependent() bool { return d.scaleDependent }

// ApproximateBytes estimates the memory that the document holds, in bytes: its nodes, paths, resources and images.
// It is meant to decide how many documents a cache keeps, not to account for memory exactly.
//
// SVG 1.1: no counterpart
func (d *Document) ApproximateBytes() int {
	s := &sizer{seen: map[any]bool{}}
	s.n = 512 + len(d.Title) + len(d.Desc)
	s.node(d.Root)
	return s.n
}

// The costs of the pieces of a tree, from the sizes of their structs and what they point to.
const (
	bytesNode      = 320
	bytesStyle     = 480
	bytesResource  = 240
	bytesSegment   = 40
	bytesPrimitive = 280
	bytesSpan      = 200
	bytesVertex    = 48
)

// sizer adds up the memory of a tree. It counts what several nodes share (a style, a gradient) once.
type sizer struct {
	n    int
	seen map[any]bool
}

// once is true the first time it sees the pointer.
func (s *sizer) once(p any) bool {
	if s.seen[p] {
		return false
	}
	s.seen[p] = true
	return true
}

func (s *sizer) path(p *ui.Path) {
	if p == nil {
		return
	}
	s.n += 64
	p.ForEachSegment(func(ui.PathSegment) { s.n += bytesSegment })
}

func (s *sizer) nodes(list []*Node) {
	s.n += 8 * len(list)
	for _, c := range list {
		s.node(c)
	}
}

func (s *sizer) node(n *Node) {
	if n == nil || !s.once(n) {
		return
	}
	s.n += bytesNode + len(n.Name) + len(n.ID)
	s.style(n.Style)
	s.path(n.Path)
	s.n += bytesVertex * len(n.Vertices)
	s.nodes(n.Children)
	if n.Image != nil {
		s.n += 96 + len(n.Image.Data) + len(n.Image.MIME)
	}
	if n.Text != nil {
		s.span(n.Text)
	}
	s.clip(n.ClipPath)
	s.mask(n.Mask)
	for _, f := range n.Filters {
		s.filter(f)
	}
}

func (s *sizer) style(st *Style) {
	if st == nil || !s.once(st) {
		return
	}
	s.n += bytesStyle + 8*len(st.DashArray) + 16*len(st.FontFamily)
	s.paint(st.Fill)
	s.paint(st.Stroke)
	s.marker(st.MarkerStart)
	s.marker(st.MarkerMid)
	s.marker(st.MarkerEnd)
}

func (s *sizer) paint(p Paint) {
	switch srv := p.Server.(type) {
	case *Gradient:
		if srv != nil && s.once(srv) {
			s.n += bytesResource + 24*len(srv.Stops)
		}
	case *Pattern:
		if srv != nil && s.once(srv) {
			s.n += bytesResource
			s.nodes(srv.Children)
		}
	}
}

func (s *sizer) clip(c *ClipPath) {
	for ; c != nil && s.once(c); c = c.ClipPath {
		s.n += bytesResource
		s.nodes(c.Children)
	}
}

func (s *sizer) mask(m *Mask) {
	for ; m != nil && s.once(m); m = m.Mask {
		s.n += bytesResource
		s.nodes(m.Children)
	}
}

func (s *sizer) marker(m *Marker) {
	if m == nil || !s.once(m) {
		return
	}
	s.n += bytesResource
	s.nodes(m.Children)
}

func (s *sizer) filter(f *Filter) {
	if f == nil || !s.once(f) {
		return
	}
	s.n += bytesResource
	for _, p := range f.Primitives {
		s.primitive(p)
	}
}

func (s *sizer) primitive(p *FilterPrimitive) {
	if p == nil {
		return
	}
	s.n += bytesPrimitive
	for k, v := range p.Attrs {
		s.n += 32 + len(k) + len(v)
	}
	for _, c := range p.Children {
		s.primitive(c)
	}
}

func (s *sizer) span(t *TextSpan) {
	if t == nil {
		return
	}
	s.n += bytesSpan + 8*(len(t.X)+len(t.Y)+len(t.Dx)+len(t.Dy)+len(t.Rotate))
	s.style(t.Style)
	s.path(t.PathData)
	for _, it := range t.Items {
		s.n += 24 + len(it.Text)
		s.span(it.Span)
	}
}
