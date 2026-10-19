package widgets

import (
	"github.com/rm4n0s/glutter/rendering"
)

// CompositedTransformTarget is a widget that a CompositedTransformFollower can
// follow: it provides the anchor that is linked by the LayerLink.
//
// Deviation: CompositedTransformFollower belongs to the step of the context
// menus (11.5) (SPEC §12.10).
//
// Flutter: widgets/basic.dart CompositedTransformTarget
type CompositedTransformTarget struct {
	SingleChildRenderObjectWidget
	link *rendering.LayerLink
}

// NewCompositedTransformTarget creates the widget; the link is what followers
// use to find the target.
//
// Flutter: widgets/basic.dart CompositedTransformTarget.new
func NewCompositedTransformTarget(link *rendering.LayerLink, child Widget, opts ...WidgetOpts) *CompositedTransformTarget {
	return &CompositedTransformTarget{NewSingleChildRenderObjectWidgetBase(child, firstOpts(opts)), link}
}

// Link is the link that connects this target to its followers.
//
// Flutter: widgets/basic.dart CompositedTransformTarget.link
func (w *CompositedTransformTarget) Link() *rendering.LayerLink { return w.link }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart CompositedTransformTarget.createRenderObject
func (w *CompositedTransformTarget) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderLeaderLayer(w.link, nil)
}

// UpdateRenderObject sets the link.
//
// Flutter: widgets/basic.dart CompositedTransformTarget.updateRenderObject
func (w *CompositedTransformTarget) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderLeaderLayer).SetLink(w.link)
}
