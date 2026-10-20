package material

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// FloatingActionButtonMargin is the margin between a floating action button
// and the edges of the scaffold.
//
// Flutter: material/floating_action_button_location.dart kFloatingActionButtonMargin
const FloatingActionButtonMargin = 16.0

// ScaffoldOpts are the optional parameters of NewScaffold.
//
// Flutter: material/scaffold.dart Scaffold.new
type ScaffoldOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AppBar is the app bar, displayed at the top of the scaffold. It must
	// implement widgets.PreferredSizeWidget, as AppBar does.
	AppBar widgets.Widget
	// Body is the primary content of the scaffold, displayed below the app bar.
	Body widgets.Widget
	// FloatingActionButton is displayed above the body, at the end of the bottom
	// of the scaffold (Flutter's FloatingActionButtonLocation.endFloat).
	FloatingActionButton widgets.Widget
	// BackgroundColor is the color of the Material of the scaffold; the default
	// is the scaffold background color of the theme.
	BackgroundColor foundation.Opt[ui.Color]
	// NoResizeToAvoidBottomInset says that the body does not resize when the
	// bottom view insets (the on-screen keyboard) appear (Flutter's
	// resizeToAvoidBottomInset: false).
	NoResizeToAvoidBottomInset bool
	// NoPrimary says that the scaffold is not at the top of the screen, so the
	// app bar does not pad itself with the top padding (Flutter's primary: false).
	NoPrimary bool
}

// Scaffold implements the basic Material Design visual layout structure: an
// app bar, a body and a floating action button. This is the minimum of the
// Material 3 scaffold that the counter example needs.
//
// TODO(step 13.1): the drawers, the bottom navigation bar, the bottom sheets,
// the persistent footer buttons, the ScaffoldMessenger and SnackBar, the
// material banner, the floating action button locations, animator and
// transition, ScaffoldState and Scaffold.of, extendBody and
// extendBodyBehindAppBar, and the scroll notification observer.
//
// Flutter: material/scaffold.dart Scaffold
type Scaffold struct {
	widgets.StatefulWidget
	o ScaffoldOpts
}

// NewScaffold creates a scaffold. In debug builds it panics with the tag
// ScaffoldAppBarNotPreferredSize when the app bar does not implement
// widgets.PreferredSizeWidget.
//
// Flutter: material/scaffold.dart Scaffold.new
func NewScaffold(opts ...ScaffoldOpts) *Scaffold {
	o := firstOpts(opts)
	if foundation.DebugMode && o.AppBar != nil {
		if _, ok := o.AppBar.(widgets.PreferredSizeWidget); !ok {
			panic(errors.New("ScaffoldAppBarNotPreferredSize", "the app bar of a Scaffold must have a preferred size (implement widgets.PreferredSizeWidget)",
				"type", foundation.RuntimeTypeName(o.AppBar)))
		}
	}
	return &Scaffold{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// AppBar is the app bar.
//
// Flutter: material/scaffold.dart Scaffold.appBar
func (w *Scaffold) AppBar() widgets.Widget { return w.o.AppBar }

// Body is the primary content of the scaffold.
//
// Flutter: material/scaffold.dart Scaffold.body
func (w *Scaffold) Body() widgets.Widget { return w.o.Body }

// FloatingActionButton is the button displayed above the body.
//
// Flutter: material/scaffold.dart Scaffold.floatingActionButton
func (w *Scaffold) FloatingActionButton() widgets.Widget { return w.o.FloatingActionButton }

// CreateState creates the state.
//
// Flutter: material/scaffold.dart Scaffold.createState
func (w *Scaffold) CreateState() widgets.State { return &scaffoldState{} }

type scaffoldState struct {
	widgets.StateOf[*Scaffold]
}

// scaffoldSlot is the id of a child of the layout of a scaffold.
//
// Flutter: material/scaffold.dart _ScaffoldSlot
type scaffoldSlot int

const (
	scaffoldSlotBody scaffoldSlot = iota
	scaffoldSlotAppBar
	scaffoldSlotFloatingActionButton
)

// Build builds the scaffold.
//
// Flutter: material/scaffold.dart ScaffoldState.build
func (s *scaffoldState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	theme := ThemeOf(bc)
	mq := widgets.MediaQueryOf(bc)

	// Wraps the child of a slot in the media query with the padding removed that
	// the other slots take.
	addSlot := func(children []widgets.Widget, slot scaffoldSlot, child widgets.Widget, sides widgets.RemoveSides) []widgets.Widget {
		if child == nil {
			return children
		}
		return append(children, widgets.NewLayoutId(slot, widgets.NewMediaQueryRemovePadding(bc, sides, child)))
	}

	var children []widgets.Widget
	children = addSlot(children, scaffoldSlotBody, o.Body, widgets.RemoveSides{Top: o.AppBar != nil})

	if o.AppBar != nil {
		preferred := o.AppBar.(widgets.PreferredSizeWidget).PreferredSize()
		topPadding := 0.0
		if !o.NoPrimary {
			topPadding = mq.Padding.Top
		}
		maxHeight := preferred.Height + topPadding
		children = addSlot(children, scaffoldSlotAppBar,
			widgets.NewConstrainedBox(rendering.BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: maxHeight}, o.AppBar),
			widgets.RemoveSides{Bottom: true})
	}

	children = addSlot(children, scaffoldSlotFloatingActionButton, o.FloatingActionButton,
		widgets.RemoveSides{Left: true, Top: true, Right: true, Bottom: true})

	bottomInset := 0.0
	if !o.NoResizeToAvoidBottomInset {
		bottomInset = mq.ViewInsets.Bottom
	}
	layout := &scaffoldLayout{
		MultiChildLayoutDelegateBase: rendering.NewMultiChildLayoutDelegateBase(nil),
		minInsets:                    mq.Padding.CopyWith(painting.EdgeInsetsCopy{Bottom: foundation.Some(bottomInset)}),
		minViewPadding:               mq.ViewPadding.CopyWith(painting.EdgeInsetsCopy{Bottom: foundation.Some(bottomInset)}),
		textDirection:                widgets.DirectionalityOf(bc),
	}

	return NewMaterial(
		widgets.NewCustomMultiChildLayout(layout, children),
		MaterialOpts{Color: foundation.Some(o.BackgroundColor.OrElse(theme.ScaffoldBackgroundColor))},
	)
}

// scaffoldLayout positions the slots of a scaffold: the app bar at the top, the
// body below it and the floating action button at the end of the bottom.
//
// Flutter: material/scaffold.dart _ScaffoldLayout
type scaffoldLayout struct {
	rendering.MultiChildLayoutDelegateBase
	minInsets      painting.EdgeInsets
	minViewPadding painting.EdgeInsets
	textDirection  ui.TextDirection
}

func (d *scaffoldLayout) PerformLayout(size ui.Size) {
	looseConstraints := rendering.NewBoxConstraintsLoose(size)
	fullWidthConstraints := looseConstraints.Tighten(rendering.TightenOpts{Width: foundation.Some(size.Width)})
	bottom := size.Height
	contentTop := 0.0

	if d.HasChild(scaffoldSlotAppBar) {
		appBarHeight := d.LayoutChild(scaffoldSlotAppBar, fullWidthConstraints).Height
		contentTop = appBarHeight
		d.PositionChild(scaffoldSlotAppBar, ui.Offset{})
	}

	contentBottom := math.Max(0.0, bottom-d.minInsets.Bottom)

	if d.HasChild(scaffoldSlotBody) {
		bodyMaxHeight := math.Max(0.0, contentBottom-contentTop)
		d.LayoutChild(scaffoldSlotBody, rendering.BoxConstraints{MaxWidth: fullWidthConstraints.MaxWidth, MaxHeight: bodyMaxHeight})
		d.PositionChild(scaffoldSlotBody, ui.Offset{Dx: 0, Dy: contentTop})
	}

	if d.HasChild(scaffoldSlotFloatingActionButton) {
		fabSize := d.LayoutChild(scaffoldSlotFloatingActionButton, looseConstraints)
		// FloatingActionButtonLocation.endFloat.
		var x float64
		if d.textDirection == ui.TextDirectionRTL {
			x = FloatingActionButtonMargin + d.minInsets.Left
		} else {
			x = size.Width - FloatingActionButtonMargin - d.minInsets.Right - fabSize.Width
		}
		bottomContentHeight := size.Height - contentBottom
		safeMargin := math.Max(FloatingActionButtonMargin, d.minViewPadding.Bottom-bottomContentHeight+FloatingActionButtonMargin)
		y := contentBottom - fabSize.Height - safeMargin
		d.PositionChild(scaffoldSlotFloatingActionButton, ui.Offset{Dx: x, Dy: y})
	}
}

func (d *scaffoldLayout) ShouldRelayout(old rendering.MultiChildLayoutDelegate) bool {
	o := old.(*scaffoldLayout)
	return o.minInsets != d.minInsets || o.minViewPadding != d.minViewPadding || o.textDirection != d.textDirection
}
