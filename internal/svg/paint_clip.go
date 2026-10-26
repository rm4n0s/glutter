package svg

import (
	"github.com/rm4n0s/glutter/ui"
)

// applyClip narrows the canvas's clip to the clip path of a node (and to the clip paths those refer to). It reports
// false if the node is not to be drawn at all: a clip path in objectBoundingBox units on a node that has no area.
//
// SVG 1.1: masking.html#EstablishingANewClippingPath
func (p *painter) applyClip(cp *ClipPath, n *Node) bool {
	var bbox ui.Rect
	var hasBBox, measured bool
	for ; cp != nil; cp = cp.ClipPath {
		if cp.Units == UnitsObjectBoundingBox && !measured {
			bbox, hasBBox = objectBBox(n)
			measured = true
		}
		path, ok := clipGeometry(cp, bbox, hasBBox)
		if !ok {
			return false
		}
		p.c.ClipPath(path)
	}
	return true
}

// clipGeometry returns the area that a clip path lets through, in the user space of the element it clips: the union
// of its children's shapes, each with its clip-rule, its transform and its own clip-path. The bounding box is the
// element's, for the objectBoundingBox units.
//
// SVG 1.1: masking.html#EstablishingANewClippingPath
func clipGeometry(cp *ClipPath, bbox ui.Rect, hasBBox bool) (*ui.Path, bool) {
	m := cp.Transform
	if cp.Units == UnitsObjectBoundingBox {
		if !hasBBox {
			return nil, false
		}
		m = m.Mul(Affine{A: bbox.Width(), D: bbox.Height(), E: bbox.Left, F: bbox.Top})
	}
	var paths []*ui.Path
	for _, c := range cp.Children {
		collectClip(c, m, bbox, hasBBox, &paths)
	}
	switch len(paths) {
	case 0:
		return ui.NewPath(), true
	case 1:
		return paths[0], true
	}
	// The union of the shapes: each keeps its own fill rule, which one path could not.
	union := paths[0]
	for _, q := range paths[1:] {
		union = ui.PathCombine(ui.PathOperationUnion, union, q)
	}
	return union, true
}

// collectClip adds the paths of a child of a clip path (a shape, or a use of one: a group) to the list, transformed by
// m, with the node's own clip-path applied.
func collectClip(n *Node, m Affine, bbox ui.Rect, hasBBox bool, out *[]*ui.Path) {
	m = m.Mul(n.Transform)
	switch n.Kind {
	case NodeShape:
		if n.Path == nil || n.Path.IsEmpty() || !n.Style.Visible {
			return
		}
		path := n.Path.Transform(m.Matrix4())
		if n.ClipPath != nil {
			// The clip-path of a child is in the child's own user space.
			cb, ok := objectBBox(n)
			for cp := n.ClipPath; cp != nil; cp = cp.ClipPath {
				g, good := clipGeometry(cp, cb, ok)
				if !good {
					return
				}
				path = ui.PathCombine(ui.PathOperationIntersect, path, g.Transform(m.Matrix4()))
			}
		}
		*out = append(*out, path)
	case NodeText:
		// 29.3: the outlines of text clip too.
	default:
		for _, c := range n.Children {
			collectClip(c, m, bbox, hasBBox, out)
		}
	}
}
