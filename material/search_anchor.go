// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	searchViewOpenDuration  = 600 * time.Millisecond
	searchAnchorFadeDuraton = 150 * time.Millisecond
)

var (
	searchViewFadeOnInterval        = animation.NewInterval(0.0, 1.0/2)
	searchViewIconsFadeOnInterval   = animation.NewInterval(1.0/6, 2.0/6)
	searchViewDividerFadeOnInterval = animation.NewInterval(0.0, 1.0/6)
	searchViewListFadeOnInterval    = animation.NewInterval(133.0/600, 233.0/600)
)

// SearchAnchorChildBuilder is the type of the function that builds the widget that
// opens the search view, typically a SearchBar or an icon button.
//
// Flutter: material/search_anchor.dart SearchAnchorChildBuilder
type SearchAnchorChildBuilder func(bc widgets.BuildContext, controller *SearchController) widgets.Widget

// SuggestionsBuilder is the type of the function that builds the suggestions for the
// text of the controller. It calls deliver with them, right away or later (on the
// UI thread, or through the binding's Post from another goroutine).
//
// Deviation: Flutter's builder returns a FutureOr<Iterable<Widget>>; Go has no
// futures, so the suggestions are delivered through a callback (SPEC §12.10).
//
// Flutter: material/search_anchor.dart SuggestionsBuilder
type SuggestionsBuilder func(bc widgets.BuildContext, controller *SearchController, deliver func(suggestions []widgets.Widget))

// ViewBuilder is the type of the function that builds the content of the search view
// from the suggestions, instead of a list.
//
// Flutter: material/search_anchor.dart ViewBuilder
type ViewBuilder func(suggestions []widgets.Widget) widgets.Widget

// SearchAnchorOpts are the optional parameters of NewSearchAnchor.
//
// Flutter: material/search_anchor.dart SearchAnchor.new
type SearchAnchorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// IsFullScreen says whether the search view is full screen; the default is true
	// on iOS, Android and Fuchsia, and false on the desktop platforms.
	IsFullScreen foundation.Opt[bool]
	// SearchController controls the search view and the text of its field; the
	// default is the controller of the anchor.
	SearchController *SearchController
	// ViewBuilder builds the content of the view from the suggestions; the default
	// is a list.
	ViewBuilder ViewBuilder
	// ViewLeading is the widget before the text field of the view; the default is a
	// back button.
	ViewLeading widgets.Widget
	// ViewTrailing are the widgets after the text field of the view; the default is a
	// button that clears the text.
	ViewTrailing []widgets.Widget
	// ViewHintText is the hint of the text field of the view.
	ViewHintText foundation.Opt[string]
	// ViewBackgroundColor is the background color of the view.
	ViewBackgroundColor foundation.Opt[ui.Color]
	// ViewElevation is the elevation of the view.
	ViewElevation foundation.Opt[float64]
	// ViewSurfaceTintColor is the color of the surface tint of the view.
	ViewSurfaceTintColor foundation.Opt[ui.Color]
	// ViewSide is the border of the view; it is not used by a full screen view.
	ViewSide foundation.Opt[painting.BorderSide]
	// ViewShape is the shape of the view; it is not used by a full screen view.
	ViewShape painting.OutlinedBorder
	// ViewBarPadding is the padding around the text field of the view.
	ViewBarPadding painting.EdgeInsetsGeometry
	// HeaderHeight is the height of the text field of the view.
	HeaderHeight foundation.Opt[float64]
	// HeaderTextStyle is the style of the text that the user types in the view.
	HeaderTextStyle *painting.TextStyle
	// HeaderHintStyle is the style of the hint of the view.
	HeaderHintStyle *painting.TextStyle
	// DividerColor is the color of the divider under the text field of the view.
	DividerColor foundation.Opt[ui.Color]
	// ViewConstraints are the size constraints of the view.
	ViewConstraints foundation.Opt[rendering.BoxConstraints]
	// ViewPadding is the padding around the view; it is not used by a full screen
	// view.
	ViewPadding painting.EdgeInsetsGeometry
	// ShrinkWrap says whether the view shrinks to the height of its suggestions.
	ShrinkWrap foundation.Opt[bool]
	// TextCapitalization is how the input method capitalizes the text.
	TextCapitalization services.TextCapitalization
	// ViewOnChanged is called when the text of the view changes.
	ViewOnChanged func(text string)
	// ViewOnSubmitted is called when the user submits the text of the view.
	ViewOnSubmitted func(text string)
	// ViewOnClose is called when the view closes.
	ViewOnClose func()
	// ViewOnOpen is called when the view opens.
	ViewOnOpen func()
	// TextInputAction is the action key of the keyboard.
	TextInputAction services.TextInputAction
	// KeyboardType is the type of the keyboard.
	KeyboardType foundation.Opt[services.TextInputType]
	// Disabled makes the anchor not open the view, and fade.
	Disabled bool
	// SmartDashesType says whether the dashes are replaced.
	SmartDashesType services.SmartDashesType
	// SmartQuotesType says whether the quotes are replaced.
	SmartQuotesType services.SmartQuotesType
}

// SearchAnchor is the anchor of a search view: a widget that opens a view with
// a text field and the suggestions for the text when the user taps it. The view
// grows from the anchor, or fills the screen.
//
// Flutter: material/search_anchor.dart SearchAnchor
type SearchAnchor struct {
	widgets.StatefulWidget
	builder            SearchAnchorChildBuilder
	suggestionsBuilder SuggestionsBuilder
	o                  SearchAnchorOpts
}

// NewSearchAnchor creates an anchor out of the widget that the builder makes and
// the builder of the suggestions.
//
// Flutter: material/search_anchor.dart SearchAnchor.new
func NewSearchAnchor(builder SearchAnchorChildBuilder, suggestionsBuilder SuggestionsBuilder, opts ...SearchAnchorOpts) *SearchAnchor {
	o := firstOpts(opts)
	return &SearchAnchor{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), builder: builder, suggestionsBuilder: suggestionsBuilder, o: o}
}

// SearchAnchorBarOpts are the optional parameters of NewSearchAnchorBar.
//
// Flutter: material/search_anchor.dart SearchAnchor.bar
type SearchAnchorBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BarLeading is the widget before the text field of the bar; the default is a
	// search icon.
	BarLeading widgets.Widget
	// BarTrailing are the widgets after the text field of the bar.
	BarTrailing []widgets.Widget
	// BarHintText is the hint of the bar, and of the view when it has none.
	BarHintText foundation.Opt[string]
	// OnTap is called when the user taps the bar.
	OnTap func()
	// OnSubmitted is called when the user submits the text.
	OnSubmitted func(text string)
	// OnChanged is called when the text of the view changes.
	OnChanged func(text string)
	// OnClose is called when the view closes.
	OnClose func()
	// OnOpen is called when the view opens.
	OnOpen func()
	// BarElevation is the elevation of the bar, in each state.
	BarElevation widgets.WidgetStateProperty[foundation.Opt[float64]]
	// BarBackgroundColor is the background color of the bar, in each state.
	BarBackgroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// BarOverlayColor is the overlay color of the bar, in each state.
	BarOverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// BarSide is the border of the bar, in each state.
	BarSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// BarShape is the shape of the bar, in each state.
	BarShape widgets.WidgetStateProperty[painting.OutlinedBorder]
	// BarPadding is the padding of the bar, in each state; the default is 16 on
	// each side.
	BarPadding widgets.WidgetStateProperty[painting.EdgeInsetsGeometry]
	// ViewBarPadding is the padding around the text field of the view.
	ViewBarPadding painting.EdgeInsetsGeometry
	// BarTextStyle is the style of the text of the bar, in each state.
	BarTextStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// BarHintStyle is the style of the hint of the bar, in each state.
	BarHintStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// ViewBuilder builds the content of the view from the suggestions.
	ViewBuilder ViewBuilder
	// ViewLeading is the widget before the text field of the view.
	ViewLeading widgets.Widget
	// ViewTrailing are the widgets after the text field of the view.
	ViewTrailing []widgets.Widget
	// ViewHintText is the hint of the view.
	ViewHintText foundation.Opt[string]
	// ViewBackgroundColor is the background color of the view.
	ViewBackgroundColor foundation.Opt[ui.Color]
	// ViewElevation is the elevation of the view.
	ViewElevation foundation.Opt[float64]
	// ViewSide is the border of the view.
	ViewSide foundation.Opt[painting.BorderSide]
	// ViewShape is the shape of the view.
	ViewShape painting.OutlinedBorder
	// ViewHeaderHeight is the height of the text field of the view.
	ViewHeaderHeight foundation.Opt[float64]
	// ViewHeaderTextStyle is the style of the text of the view.
	ViewHeaderTextStyle *painting.TextStyle
	// ViewHeaderHintStyle is the style of the hint of the view.
	ViewHeaderHintStyle *painting.TextStyle
	// DividerColor is the color of the divider of the view.
	DividerColor foundation.Opt[ui.Color]
	// Constraints are the size constraints of the bar.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// ViewConstraints are the size constraints of the view.
	ViewConstraints foundation.Opt[rendering.BoxConstraints]
	// ViewPadding is the padding around the view.
	ViewPadding painting.EdgeInsetsGeometry
	// ShrinkWrap says whether the view shrinks to the height of its suggestions.
	ShrinkWrap foundation.Opt[bool]
	// IsFullScreen says whether the view is full screen.
	IsFullScreen foundation.Opt[bool]
	// SearchController controls the view and the text of its field.
	SearchController *SearchController
	// TextCapitalization is how the input method capitalizes the text.
	TextCapitalization services.TextCapitalization
	// TextInputAction is the action key of the keyboard.
	TextInputAction services.TextInputAction
	// KeyboardType is the type of the keyboard.
	KeyboardType foundation.Opt[services.TextInputType]
	// ScrollPadding is the space kept around the caret when the field scrolls.
	ScrollPadding foundation.Opt[painting.EdgeInsets]
	// ContextMenuBuilder builds the context menu of the text field.
	ContextMenuBuilder widgets.EditableTextContextMenuBuilder
	// Disabled makes the bar not open the view, and fade.
	Disabled bool
	// SmartDashesType says whether the dashes are replaced.
	SmartDashesType services.SmartDashesType
	// SmartQuotesType says whether the quotes are replaced.
	SmartQuotesType services.SmartQuotesType
}

// NewSearchAnchorBar creates an anchor that is a SearchBar: tapping it, or typing
// in it, opens the view, whose text field takes over.
//
// Flutter: material/search_anchor.dart SearchAnchor.bar
func NewSearchAnchorBar(suggestionsBuilder SuggestionsBuilder, opts ...SearchAnchorBarOpts) *SearchAnchor {
	b := firstOpts(opts)
	viewHint := b.ViewHintText
	if viewHint.IsNone() {
		viewHint = b.BarHintText
	}
	o := SearchAnchorOpts{
		Key: b.Key, IsFullScreen: b.IsFullScreen, SearchController: b.SearchController, ViewBuilder: b.ViewBuilder, ViewLeading: b.ViewLeading,
		ViewTrailing: b.ViewTrailing, ViewHintText: viewHint, ViewBackgroundColor: b.ViewBackgroundColor, ViewElevation: b.ViewElevation,
		ViewSide: b.ViewSide, ViewShape: b.ViewShape, ViewBarPadding: b.ViewBarPadding, HeaderHeight: b.ViewHeaderHeight,
		HeaderTextStyle: b.ViewHeaderTextStyle, HeaderHintStyle: b.ViewHeaderHintStyle, DividerColor: b.DividerColor,
		ViewConstraints: b.ViewConstraints, ViewPadding: b.ViewPadding, ShrinkWrap: b.ShrinkWrap, TextCapitalization: b.TextCapitalization,
		ViewOnChanged: b.OnChanged, ViewOnSubmitted: b.OnSubmitted, ViewOnClose: b.OnClose, ViewOnOpen: b.OnOpen,
		TextInputAction: b.TextInputAction, KeyboardType: b.KeyboardType, Disabled: b.Disabled,
		SmartDashesType: b.SmartDashesType, SmartQuotesType: b.SmartQuotesType,
	}
	builder := func(_ widgets.BuildContext, controller *SearchController) widgets.Widget {
		padding := b.BarPadding
		if padding == nil {
			padding = widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](painting.NewEdgeInsetsSymmetric(0, 16.0))
		}
		leading := b.BarLeading
		if leading == nil {
			leading = widgets.NewIcon(Icons.Search())
		}
		return NewSearchBar(SearchBarOpts{
			Constraints: b.Constraints, Controller: controller.TextEditingController,
			OnTap:       func() { controller.OpenView(); callIfSet(b.OnTap) },
			OnChanged:   func(string) { controller.OpenView() },
			OnSubmitted: b.OnSubmitted, HintText: b.BarHintText, HintStyle: b.BarHintStyle, TextStyle: b.BarTextStyle,
			Elevation: b.BarElevation, BackgroundColor: b.BarBackgroundColor, OverlayColor: b.BarOverlayColor, Side: b.BarSide, Shape: b.BarShape,
			Padding: padding, Leading: leading, Trailing: b.BarTrailing, TextCapitalization: b.TextCapitalization,
			TextInputAction: b.TextInputAction, KeyboardType: b.KeyboardType, ScrollPadding: b.ScrollPadding,
			ContextMenuBuilder: b.ContextMenuBuilder, SmartDashesType: b.SmartDashesType, SmartQuotesType: b.SmartQuotesType,
		})
	}
	return &SearchAnchor{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), builder: builder, suggestionsBuilder: suggestionsBuilder, o: o}
}

func callIfSet(f func()) {
	if f != nil {
		f()
	}
}

// CreateState creates the state.
//
// Flutter: material/search_anchor.dart SearchAnchor.createState
func (w *SearchAnchor) CreateState() widgets.State { return &searchAnchorState{anchorIsVisible: true} }

// SearchController controls a SearchAnchor: it opens and closes the view, and it
// is the controller of the text of its field.
//
// Deviation: Flutter's SearchController is a TextEditingController; Go has no
// inheritance, so it holds one, and its methods (Text, SetText, Clear, AddListener
// and the others) are the ones of the embedded controller. Pass
// controller.TextEditingController where a *widgets.TextEditingController is
// needed (SPEC §12.10).
//
// Flutter: material/search_anchor.dart SearchController
type SearchController struct {
	*widgets.TextEditingController
	anchor *searchAnchorState
}

// NewSearchController creates a controller with no text.
//
// Flutter: material/search_anchor.dart SearchController.new
func NewSearchController() *SearchController {
	return &SearchController{TextEditingController: widgets.NewTextEditingController()}
}

func (c *SearchController) debugCheckAttached() {
	if foundation.DebugMode && c.anchor == nil {
		panic(errors.New(tagSearchControllerNotAttached, "the SearchController is not attached to any SearchAnchor"))
	}
}

// IsAttached says whether an anchor uses the controller. In debug builds IsOpen,
// OpenView and CloseView panic with the tag SearchControllerNotAttached when it
// is not.
//
// Flutter: material/search_anchor.dart SearchController.isAttached
func (c *SearchController) IsAttached() bool { return c.anchor != nil }

// IsOpen says whether the view is open.
//
// Flutter: material/search_anchor.dart SearchController.isOpen
func (c *SearchController) IsOpen() bool {
	c.debugCheckAttached()
	return c.anchor != nil && c.anchor.viewIsOpen()
}

// OpenView opens the view.
//
// Flutter: material/search_anchor.dart SearchController.openView
func (c *SearchController) OpenView() {
	c.debugCheckAttached()
	if c.anchor != nil {
		c.anchor.openView()
	}
}

// CloseView closes the view, and, when selectedText is set, makes it the text of
// the controller.
//
// Flutter: material/search_anchor.dart SearchController.closeView
func (c *SearchController) CloseView(selectedText foundation.Opt[string]) {
	c.debugCheckAttached()
	if c.anchor != nil {
		c.anchor.closeView(selectedText)
	}
}

type searchAnchorState struct {
	widgets.StateOf[*SearchAnchor]
	screenSize         foundation.Opt[ui.Size]
	anchorIsVisible    bool
	anchorKey          *widgets.GlobalKey[widgets.State]
	internalController *SearchController
	route              *searchViewRoute
}

func (s *searchAnchorState) viewIsOpen() bool { return !s.anchorIsVisible }

func (s *searchAnchorState) searchController() *SearchController {
	if c := s.Widget().o.SearchController; c != nil {
		return c
	}
	if s.internalController == nil {
		s.internalController = NewSearchController()
	}
	return s.internalController
}

func (s *searchAnchorState) InitState() {
	s.anchorKey = widgets.NewGlobalKey[widgets.State]()
	s.searchController().anchor = s
}

func (s *searchAnchorState) DidChangeDependencies() {
	updated := widgets.MediaQuerySizeOf(s.Context())
	if old, ok := s.screenSize.Get(); ok && old != updated {
		if s.searchController().IsOpen() && !s.showFullScreenView() {
			s.closeView(foundation.None[string]())
		}
	}
	s.screenSize = foundation.Some(updated)
}

func (s *searchAnchorState) DidUpdateWidget(old *SearchAnchor) {
	if old.o.SearchController != s.Widget().o.SearchController {
		if old.o.SearchController != nil && old.o.SearchController.anchor == s {
			old.o.SearchController.anchor = nil
		}
		s.searchController().anchor = s
	}
}

func (s *searchAnchorState) detach(c *SearchController) {
	if c != nil && c.anchor == s {
		c.anchor = nil
	}
}

func (s *searchAnchorState) Dispose() {
	external := s.Widget().o.SearchController
	s.detach(external)
	s.detach(s.internalController)
	if s.route != nil && s.route.Navigator() != nil {
		s.route.dismiss(external == nil)
		if external != nil && s.internalController != nil {
			s.internalController.Dispose()
		}
	} else if s.internalController != nil {
		s.internalController.Dispose()
	}
}

func (s *searchAnchorState) openView() {
	navigator := widgets.NavigatorOf(s.Context())
	w := s.Widget()
	r := &searchViewRoute{
		o: w.o, showFullScreenView: s.showFullScreenView(), toggleVisibility: s.toggleVisibility,
		textDirection: widgets.DirectionalityOf(s.Context()), anchorKey: s.anchorKey, searchController: s.searchController(),
		suggestionsBuilder: w.suggestionsBuilder, capturedThemes: widgets.CaptureThemes(s.Context(), navigator.Context()),
	}
	widgets.InitPopupRouteBase(&r.PopupRouteBase, r, widgets.ModalRouteOpts{})
	s.route = r
	navigator.Push(r)
}

func (s *searchAnchorState) closeView(selectedText foundation.Opt[string]) {
	if text, ok := selectedText.Get(); ok {
		s.searchController().SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: text}))
	}
	widgets.NavigatorOf(s.Context()).Pop()
}

func (s *searchAnchorState) toggleVisibility() bool {
	s.SetState(func() { s.anchorIsVisible = !s.anchorIsVisible })
	return s.anchorIsVisible
}

func (s *searchAnchorState) showFullScreenView() bool {
	if v, ok := s.Widget().o.IsFullScreen.Get(); ok {
		return v
	}
	switch ThemeOf(s.Context()).Platform {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
		return true
	}
	return false
}

func (s *searchAnchorState) opacity() float64 {
	if !s.Widget().o.Disabled {
		if s.anchorIsVisible {
			return 1.0
		}
		return 0.0
	}
	return searchBarDisabledOpacity
}

func (s *searchAnchorState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	return widgets.NewAnimatedOpacity(s.opacity(), searchAnchorFadeDuraton,
		widgets.NewIgnorePointer(
			widgets.NewGestureDetector(w.builder(bc, s.searchController()), widgets.GestureDetectorOpts{OnTap: s.openView}),
			widgets.IgnorePointerOpts{Ignoring: foundation.Some(w.o.Disabled)}),
		widgets.AnimatedOpacityOpts{Key: s.anchorKey})
}

// searchViewRoute is the route of the search view.
//
// Flutter: material/search_anchor.dart _SearchViewRoute
type searchViewRoute struct {
	widgets.PopupRouteBase
	o                       SearchAnchorOpts
	showFullScreenView      bool
	toggleVisibility        func() bool
	textDirection           ui.TextDirection
	anchorKey               *widgets.GlobalKey[widgets.State]
	searchController        *SearchController
	suggestionsBuilder      SuggestionsBuilder
	capturedThemes          *widgets.CapturedThemes
	curvedAnimation         *animation.CurvedAnimation
	viewFadeOnIntervalCurve *animation.CurvedAnimation
	willDisposeController   bool
	viewDefaults            SearchViewThemeData
	viewTheme               SearchViewThemeData
	rectTween               *animation.Tween[foundation.Opt[ui.Rect]]
}

// BarrierColor is none: the barrier is transparent.
func (r *searchViewRoute) BarrierColor() foundation.Opt[ui.Color] { return foundation.None[ui.Color]() }

// BarrierDismissible is true: tapping outside closes the view.
func (r *searchViewRoute) BarrierDismissible() bool { return true }

// BarrierLabel is "Dismiss".
func (r *searchViewRoute) BarrierLabel() string { return "Dismiss" }

// TransitionDuration is 600 ms.
func (r *searchViewRoute) TransitionDuration() time.Duration { return searchViewOpenDuration }

// ReverseTransitionDuration is 600 ms.
func (r *searchViewRoute) ReverseTransitionDuration() time.Duration { return searchViewOpenDuration }

// getRect is the rectangle of the anchor in the navigator, if it is in the tree.
//
// Flutter: material/search_anchor.dart _SearchViewRoute.getRect
func (r *searchViewRoute) getRect() foundation.Opt[ui.Rect] {
	ctx := r.anchorKey.CurrentContext()
	if ctx == nil {
		return foundation.None[ui.Rect]()
	}
	box, ok := ctx.FindRenderObject().(rendering.Box)
	if !ok {
		return foundation.None[ui.Rect]()
	}
	navigator := widgets.NavigatorOf(ctx)
	location := box.LocalToGlobal(ui.Offset{}, navigator.Context().FindRenderObject())
	return foundation.Some(location.And(box.Size()))
}

// DidPush reads the defaults and the theme of the view, works out where the view
// goes, hides the anchor and starts the transition.
//
// Flutter: material/search_anchor.dart _SearchViewRoute.didPush
func (r *searchViewRoute) DidPush() *scheduler.TickerFuture {
	ctx := r.anchorKey.CurrentContext()
	r.updateViewConfig(ctx)
	r.updateTweens(ctx)
	r.toggleVisibility()
	callIfSet(r.o.ViewOnOpen)
	return r.PopupRouteBase.DidPush()
}

// DidPop works out where the view ends, shows the anchor and starts the transition.
//
// Flutter: material/search_anchor.dart _SearchViewRoute.didPop
func (r *searchViewRoute) DidPop(result any) bool {
	ctx := r.anchorKey.CurrentContext()
	r.updateTweens(ctx)
	r.toggleVisibility()
	callIfSet(r.o.ViewOnClose)
	widgets.WidgetsBindingOf(ctx).Scheduler().AddPostFrameCallback(func(time.Duration) {
		if c := r.anchorKey.CurrentContext(); c != nil {
			widgets.FocusScopeOf(c).Unfocus()
		}
	})
	return r.PopupRouteBase.DidPop(result)
}

func (r *searchViewRoute) dismiss(disposeController bool) {
	r.willDisposeController = disposeController
	if r.IsActive() {
		r.Navigator().RemoveRoute(r)
	}
}

// Dispose releases the animations and, when it was asked to, the controller.
//
// Flutter: material/search_anchor.dart _SearchViewRoute.dispose
func (r *searchViewRoute) Dispose() {
	if r.curvedAnimation != nil {
		r.curvedAnimation.Dispose()
	}
	if r.viewFadeOnIntervalCurve != nil {
		r.viewFadeOnIntervalCurve.Dispose()
	}
	if r.willDisposeController {
		r.searchController.Dispose()
	}
	r.PopupRouteBase.Dispose()
}

func (r *searchViewRoute) updateViewConfig(bc widgets.BuildContext) {
	r.viewDefaults = searchViewDefaultsM3(bc, r.showFullScreenView)
	r.viewTheme = SearchViewThemeOf(bc)
}

// updateTweens works out the rectangle of the view: it starts at the anchor and
// ends at a place that is in the screen.
//
// Flutter: material/search_anchor.dart _SearchViewRoute.updateTweens
func (r *searchViewRoute) updateTweens(bc widgets.BuildContext) {
	navigatorBox := widgets.NavigatorOf(bc).Context().FindRenderObject().(rendering.Box)
	screenSize := navigatorBox.Size()
	anchorRect := r.getRect().OrElse(ui.Rect{})
	effective := firstOpt(r.o.ViewConstraints, r.viewTheme.Constraints, r.viewDefaults.Constraints).Value()
	if r.rectTween == nil {
		r.rectTween = animation.NewRectTween(foundation.None[ui.Rect](), foundation.None[ui.Rect]())
	}
	r.rectTween.Begin = foundation.Some(anchorRect)

	viewWidth := ui.ClampDouble(anchorRect.Width(), effective.MinWidth, effective.MaxWidth)
	viewHeight := ui.ClampDouble(screenSize.Height*2/3, effective.MinHeight, effective.MaxHeight)
	viewTopToScreenBottom := screenSize.Height - anchorRect.Top

	var topLeft ui.Offset
	if r.textDirection == ui.TextDirectionRTL {
		topLeft = ui.Offset{Dx: math.Max(anchorRect.Right-viewWidth, 0.0), Dy: anchorRect.Top}
		if anchorRect.Right < viewWidth {
			topLeft = ui.Offset{Dx: 0.0, Dy: topLeft.Dy}
		}
	} else {
		topLeft = anchorRect.TopLeft()
		if screenSize.Width-anchorRect.Left < viewWidth {
			topLeft = ui.Offset{Dx: screenSize.Width - math.Min(viewWidth, screenSize.Width), Dy: topLeft.Dy}
		}
	}
	if viewTopToScreenBottom < viewHeight {
		topLeft = ui.Offset{Dx: topLeft.Dx, Dy: screenSize.Height - math.Min(viewHeight, screenSize.Height)}
	}
	end := topLeft.And(ui.Size{Width: viewWidth, Height: viewHeight})
	if r.showFullScreenView {
		end = ui.Offset{}.And(screenSize)
	}
	r.rectTween.End = foundation.Some(end)
}

// BuildPage builds the view in the rectangle that the animation gives.
//
// Flutter: material/search_anchor.dart _SearchViewRoute.buildPage
func (r *searchViewRoute) BuildPage(_ widgets.BuildContext, anim, _ animation.Animation[float64]) widgets.Widget {
	return widgets.NewDirectionality(r.textDirection, widgets.NewAnimatedBuilder(anim, func(bc widgets.BuildContext, _ widgets.Widget) widgets.Widget {
		if r.curvedAnimation == nil {
			r.curvedAnimation = animation.NewCurvedAnimation(anim, animation.Curves.EaseInOutCubicEmphasized, animation.Curves.EaseInOutCubicEmphasized.Flipped())
		}
		viewRect := r.rectTween.Evaluate(r.curvedAnimation).Value()
		topPadding := 0.0
		if r.showFullScreenView {
			topPadding = ui.LerpDouble(0.0, widgets.MediaQueryPaddingOf(bc).Top, r.curvedAnimation.Value())
		}
		if r.viewFadeOnIntervalCurve == nil {
			r.viewFadeOnIntervalCurve = animation.NewCurvedAnimation(anim, searchViewFadeOnInterval, searchViewFadeOnInterval.Flipped())
		}
		return widgets.NewFadeTransition(r.viewFadeOnIntervalCurve, r.capturedThemes.Wrap(&searchViewContent{
			StatefulWidget: widgets.NewStatefulWidgetBase(), route: r, animation: r.curvedAnimation, topPadding: topPadding,
			viewMaxWidth: r.rectTween.End.Value().Width(), viewRect: viewRect,
		}))
	}, nil))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSearchControllerNotAttached = "SearchControllerNotAttached"
)
