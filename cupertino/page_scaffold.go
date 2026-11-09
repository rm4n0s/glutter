package cupertino

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ObstructingPreferredSizeWidget is a PreferredSizeWidget that can say whether it hides what is below it:
// the navigation bar of a CupertinoPageScaffold, which is translucent when its background is not opaque.
//
// Flutter: cupertino/page_scaffold.dart ObstructingPreferredSizeWidget
type ObstructingPreferredSizeWidget interface {
	widgets.PreferredSizeWidget
	// ShouldFullyObstruct is whether the widget fully hides the content below it: the scaffold then removes
	// the area of the bar from the padding that the content sees, and does not draw under it.
	//
	// Flutter: cupertino/page_scaffold.dart ObstructingPreferredSizeWidget.shouldFullyObstruct
	ShouldFullyObstruct(bc widgets.BuildContext) bool
}

// CupertinoPageScaffoldOpts are the optional parameters of NewCupertinoPageScaffold.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffold.new
type CupertinoPageScaffoldOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// NavigationBar is the navigation bar at the top of the page; it is under the status bar.
	NavigationBar ObstructingPreferredSizeWidget
	// BackgroundColor is the color of the page; the default is the scaffold background color of the theme.
	BackgroundColor ui.ColorValue
	// NotResizeToAvoidBottomInset says that the content does not shrink when the keyboard is up (Flutter's
	// resizeToAvoidBottomInset is true by default).
	NotResizeToAvoidBottomInset bool
}

// CupertinoPageScaffold is the basic layout for an iOS page: a navigation bar at the top and the content under
// it, with the content padded to stay clear of the bar and of the keyboard. A tap on the status bar scrolls the
// primary scroll controller of the page to its top.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffold
type CupertinoPageScaffold struct {
	widgets.StatefulWidget
	child widgets.Widget
	o     CupertinoPageScaffoldOpts
}

// NewCupertinoPageScaffold creates a page of the child.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffold.new
func NewCupertinoPageScaffold(child widgets.Widget, opts ...CupertinoPageScaffoldOpts) *CupertinoPageScaffold {
	o := firstOf(opts)
	return &CupertinoPageScaffold{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child, o}
}

// Child is the content of the page.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffold.child
func (w *CupertinoPageScaffold) Child() widgets.Widget { return w.child }

// NavigationBar is the bar at the top of the page, or nil.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffold.navigationBar
func (w *CupertinoPageScaffold) NavigationBar() ObstructingPreferredSizeWidget {
	return w.o.NavigationBar
}

// BackgroundColor is the color of the page, or nil for the theme's.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffold.backgroundColor
func (w *CupertinoPageScaffold) BackgroundColor() ui.ColorValue { return w.o.BackgroundColor }

// ResizeToAvoidBottomInset is whether the content shrinks when the keyboard is up.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffold.resizeToAvoidBottomInset
func (w *CupertinoPageScaffold) ResizeToAvoidBottomInset() bool {
	return !w.o.NotResizeToAvoidBottomInset
}

// CreateState creates the state.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffold.createState
func (w *CupertinoPageScaffold) CreateState() widgets.State { return &cupertinoPageScaffoldState{} }

type cupertinoPageScaffoldState struct {
	widgets.StateOf[*CupertinoPageScaffold]
	widgets.WidgetsBindingObserverBase
	statusBarKey *widgets.GlobalKey[widgets.State]
}

func (s *cupertinoPageScaffoldState) InitState() {
	s.statusBarKey = widgets.NewGlobalKey[widgets.State]()
	widgets.WidgetsBindingOf(s.Context()).AddObserver(s)
}

func (s *cupertinoPageScaffoldState) Deactivate() {
	widgets.WidgetsBindingOf(s.Context()).RemoveObserver(s)
}

func (s *cupertinoPageScaffoldState) Activate() {
	widgets.WidgetsBindingOf(s.Context()).AddObserver(s)
}

// HandleStatusBarTap scrolls the primary scroll controller to the top when the status bar of this page is
// what was tapped (and not a page above it).
//
// Flutter: cupertino/page_scaffold.dart _CupertinoPageScaffoldState.handleStatusBarTap
func (s *cupertinoPageScaffoldState) HandleStatusBarTap() {
	controller := widgets.PrimaryScrollControllerMaybeOf(s.Context())
	if controller != nil && controller.HasClients() && hitTestableAtOrigin(s.statusBarKey) {
		controller.AnimateTo(0, 500*time.Millisecond, animation.Curves.LinearToEaseOut)
	}
}

func (s *cupertinoPageScaffoldState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	paddedContent := w.child
	backgroundColor := CupertinoThemeOf(bc).ScaffoldBackgroundColor().EffectiveColor()
	if w.o.BackgroundColor != nil {
		backgroundColor = CupertinoDynamicColorResolve(w.o.BackgroundColor, bc)
	}
	existing := widgets.MediaQueryOf(bc)
	resizeToAvoidBottomInset := w.ResizeToAvoidBottomInset()
	if bar := w.o.NavigationBar; bar != nil {
		topPadding := bar.PreferredSize().Height + existing.Padding.Top
		bottomPadding := 0.0
		newViewInsets := existing.ViewInsets
		if resizeToAvoidBottomInset {
			bottomPadding = existing.ViewInsets.Bottom
			newViewInsets = existing.ViewInsets.CopyWith(painting.EdgeInsetsCopy{Bottom: foundation.Some(0.0)})
		}
		if bar.ShouldFullyObstruct(bc) {
			// If the navigation bar is opaquely obstructing, directly shift the main content down. If the
			// navigation bar is translucent, let the main content draw behind it.
			paddedContent = widgets.NewMediaQuery(
				existing.RemovePadding(widgets.RemoveSides{Top: true}).CopyWith(widgets.MediaQueryDataCopyOpts{ViewInsets: foundation.Some(newViewInsets)}),
				widgets.NewPadding(painting.EdgeInsets{Top: topPadding, Bottom: bottomPadding}, paddedContent))
		} else {
			// If the navigation bar is translucent, the main content is not shifted: its padding says how much
			// is under the bar.
			paddedContent = widgets.NewMediaQuery(
				existing.CopyWith(widgets.MediaQueryDataCopyOpts{Padding: foundation.Some(existing.Padding.CopyWith(painting.EdgeInsetsCopy{Top: foundation.Some(topPadding)})), ViewInsets: foundation.Some(newViewInsets)}),
				widgets.NewPadding(painting.EdgeInsets{Bottom: bottomPadding}, paddedContent))
		}
	} else if resizeToAvoidBottomInset {
		// If there is no navigation bar, still may need to add padding in order to support resizeToAvoidBottomInset.
		paddedContent = widgets.NewMediaQuery(
			existing.CopyWith(widgets.MediaQueryDataCopyOpts{ViewInsets: foundation.Some(existing.ViewInsets.CopyWith(painting.EdgeInsetsCopy{Bottom: foundation.Some(0.0)}))}),
			widgets.NewPadding(painting.EdgeInsets{Bottom: existing.ViewInsets.Bottom}, paddedContent))
	}
	children := []widgets.Widget{paddedContent}
	if bar := w.o.NavigationBar; bar != nil {
		children = append(children, widgets.NewPositioned(widgets.NewMediaQueryWithNoTextScaling(bar),
			widgets.PositionedOpts{Top: foundation.Some(0.0), Left: foundation.Some(0.0), Right: foundation.Some(0.0)}))
	}
	// The status bar: a strip at the top that tells whether a tap on the status bar is for this page.
	children = append(children, widgets.NewPositioned(newHitTestableAtOrigin(s.statusBarKey),
		widgets.PositionedOpts{Top: foundation.Some(0.0), Left: foundation.Some(0.0), Right: foundation.Some(0.0), Height: foundation.Some(existing.Padding.Top)}))
	return widgets.NewScrollNotificationObserver(widgets.NewDecoratedBox(
		painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(backgroundColor)}),
		NewCupertinoPageScaffoldBackgroundColor(backgroundColor, widgets.NewStack(children))))
}

// CupertinoPageScaffoldBackgroundColor is the inherited widget through which a CupertinoPageScaffold gives
// its background color to the widgets below it (the navigation bar draws its large title over it).
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffoldBackgroundColor
type CupertinoPageScaffoldBackgroundColor struct {
	widgets.InheritedWidget
	color ui.Color
}

// NewCupertinoPageScaffoldBackgroundColor creates the widget.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffoldBackgroundColor.new
func NewCupertinoPageScaffoldBackgroundColor(color ui.Color, child widgets.Widget, opts ...widgets.WidgetOpts) *CupertinoPageScaffoldBackgroundColor {
	return &CupertinoPageScaffoldBackgroundColor{widgets.NewInheritedWidgetBase(child, firstOf(opts)), color}
}

// Color is the background color of the scaffold.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffoldBackgroundColor.color
func (w *CupertinoPageScaffoldBackgroundColor) Color() ui.Color { return w.color }

// UpdateShouldNotify is whether the color changed.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffoldBackgroundColor.updateShouldNotify
func (w *CupertinoPageScaffoldBackgroundColor) UpdateShouldNotify(old widgets.Widget) bool {
	return w.color != old.(*CupertinoPageScaffoldBackgroundColor).color
}

// CupertinoPageScaffoldBackgroundColorMaybeOf is the background color of the closest scaffold above the
// context, or none. The context depends on it.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffoldBackgroundColor.maybeOf
func CupertinoPageScaffoldBackgroundColorMaybeOf(bc widgets.BuildContext) foundation.Opt[ui.Color] {
	if w, ok := widgets.DependOn[*CupertinoPageScaffoldBackgroundColor](bc); ok {
		return foundation.Some(w.color)
	}
	return foundation.None[ui.Color]()
}

// DebugFillProperties adds the color.
//
// Flutter: cupertino/page_scaffold.dart CupertinoPageScaffoldBackgroundColor.debugFillProperties
func (w *CupertinoPageScaffoldBackgroundColor) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(painting.NewColorProperty("page scaffold background color", foundation.Some(w.color)))
}

// hitTestableAtOrigin is whether the strip of the status bar of a page is what a hit test at the top left of the
// view reaches: a page above this one, or a sheet, would be in the way.
//
// Flutter: cupertino/page_scaffold.dart _HitTestableAtOrigin.hitTestableAtOrigin
func hitTestableAtOrigin(key *widgets.GlobalKey[widgets.State]) bool {
	bc := key.CurrentContext()
	if bc == nil {
		if foundation.DebugMode {
			panic(errors.New(tagCupertinoStatusBarNotMounted, "the BuildContext associated with the key of the status bar is not mounted"))
		}
		return false
	}
	renderObject := bc.(widgets.Element).RenderObject()
	result := gestures.NewHitTestResult()
	widgets.WidgetsBindingOf(bc).Renderer().HitTestInView(result, ui.Offset{}, widgets.ViewOf(bc).ViewID())
	for _, entry := range result.Path() {
		if entry.Target() == renderObject {
			return true
		}
	}
	return false
}

// hitTestableAtOriginWidget is the strip of the status bar: a translucent meta data box that fills the space.
type hitTestableAtOriginWidget struct {
	widgets.StatelessWidget
	key *widgets.GlobalKey[widgets.State]
}

func newHitTestableAtOrigin(key *widgets.GlobalKey[widgets.State]) *hitTestableAtOriginWidget {
	return &hitTestableAtOriginWidget{widgets.NewStatelessWidgetBase(), key}
}

func (w *hitTestableAtOriginWidget) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewMetaData(widgets.NewSizedBoxExpand(nil), widgets.MetaDataOpts{Key: w.key, Behavior: rendering.HitTestBehaviorTranslucent})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoStatusBarNotMounted = "CupertinoStatusBarNotMounted"
)
