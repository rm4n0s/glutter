package cupertino

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoFocusHalo is a rounded rectangular outline that shows around a part of the tree when a widget
// inside it has the focus: the focus ring of Cupertino.
//
// Flutter: cupertino/cupertino_focus_halo.dart CupertinoFocusHalo
type CupertinoFocusHalo struct {
	widgets.StatefulWidget
	child        widgets.Widget
	borderRadius painting.BorderRadiusGeometry
	superellipse bool
	rect         bool
}

// NewCupertinoFocusHaloWithRect creates a rectangular halo around the child.
//
// Flutter: cupertino/cupertino_focus_halo.dart CupertinoFocusHalo.withRect
func NewCupertinoFocusHaloWithRect(child widgets.Widget, opts ...widgets.WidgetOpts) *CupertinoFocusHalo {
	return &CupertinoFocusHalo{widgets.NewStatefulWidgetBase(firstOf(opts)), child, painting.BorderRadiusZero, false, true}
}

// NewCupertinoFocusHaloWithRRect creates a halo with rounded corners around the child.
//
// Flutter: cupertino/cupertino_focus_halo.dart CupertinoFocusHalo.withRRect
func NewCupertinoFocusHaloWithRRect(borderRadius painting.BorderRadiusGeometry, child widgets.Widget, opts ...widgets.WidgetOpts) *CupertinoFocusHalo {
	return &CupertinoFocusHalo{widgets.NewStatefulWidgetBase(firstOf(opts)), child, borderRadius, false, false}
}

// NewCupertinoFocusHaloWithRoundedSuperellipse creates a halo with the continuous corners of iOS around the
// child.
//
// Flutter: cupertino/cupertino_focus_halo.dart CupertinoFocusHalo.withRoundedSuperellipse
func NewCupertinoFocusHaloWithRoundedSuperellipse(borderRadius painting.BorderRadiusGeometry, child widgets.Widget, opts ...widgets.WidgetOpts) *CupertinoFocusHalo {
	return &CupertinoFocusHalo{widgets.NewStatefulWidgetBase(firstOf(opts)), child, borderRadius, true, false}
}

// Child is the widget that the halo surrounds.
//
// Flutter: cupertino/cupertino_focus_halo.dart CupertinoFocusHalo.child
func (w *CupertinoFocusHalo) Child() widgets.Widget { return w.child }

// CreateState creates the state.
//
// Flutter: cupertino/cupertino_focus_halo.dart CupertinoFocusHalo.createState
func (w *CupertinoFocusHalo) CreateState() widgets.State { return &cupertinoFocusHaloState{} }

type cupertinoFocusHaloState struct {
	widgets.StateOf[*CupertinoFocusHalo]
	childHasFocus bool
}

// cupertinoFocusOutlineColor is the color of the outline of a focused widget over the color: the color with the
// opacity, the lightness and the saturation of the focus of Cupertino.
func cupertinoFocusOutlineColor(c ui.Color) ui.Color {
	return painting.NewHSLColorFromColor(c.WithOpacity(CupertinoFocusColorOpacity)).
		WithLightness(CupertinoFocusColorBrightness).WithSaturation(CupertinoFocusColorSaturation).ToColor()
}

func (s *cupertinoFocusHaloState) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	side := painting.BorderSideNone
	if s.childHasFocus {
		side = painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(cupertinoFocusOutlineColor(CupertinoColors.ActiveBlue.Color)), Width: foundation.Some(3.5)})
	}
	var shape painting.ShapeBorder
	switch {
	case w.superellipse:
		shape = painting.NewRoundedSuperellipseBorder(painting.RoundedSuperellipseBorderOpts{Side: side, BorderRadius: w.borderRadius})
	default:
		shape = painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{Side: side, BorderRadius: w.borderRadius})
	}
	return widgets.NewFocus(widgets.NewDecoratedBox(painting.NewShapeDecoration(shape), w.child,
		widgets.DecoratedBoxOpts{Position: rendering.DecorationPositionForeground}),
		widgets.FocusOpts{CanRequestFocus: foundation.Some(false), SkipTraversal: foundation.Some(true), ExcludeSemantics: true,
			OnFocusChange: func(hasFocus bool) { s.SetState(func() { s.childHasFocus = hasFocus }) }})
}
