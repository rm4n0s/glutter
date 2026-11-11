package cupertino

import (
	"math"
	"reflect"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoMenuItemOpts are the optional parameters of NewCupertinoMenuItem.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.new
type CupertinoMenuItemOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Subtitle is a widget below the child.
	Subtitle widgets.Widget
	// Leading is a widget before the child, such as an icon.
	Leading widgets.Widget
	// LeadingWidth is the width of the room for the leading widget.
	LeadingWidth foundation.Opt[float64]
	// LeadingMidpointAlignment is where the middle of the leading widget is, in the room for it.
	LeadingMidpointAlignment painting.AlignmentGeometry
	// Trailing is a widget after the child.
	Trailing widgets.Widget
	// TrailingWidth is the width of the room for the trailing widget.
	TrailingWidth foundation.Opt[float64]
	// TrailingMidpointAlignment is where the middle of the trailing widget is, in the room for it.
	TrailingMidpointAlignment painting.AlignmentGeometry
	// Padding is the padding around the content; the default makes the item as high as iOS does.
	Padding painting.EdgeInsetsGeometry
	// Constraints are the constraints of the content.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Autofocus asks for the focus when the item is built.
	Autofocus bool
	// FocusNode is the focus node of the item.
	FocusNode *widgets.FocusNode
	// OnFocusChange is called when the item gains or loses the focus.
	OnFocusChange func(focused bool)
	// OnHover is called when a mouse enters or leaves the item.
	OnHover func(hovered bool)
	// OnPressed is called when the item is activated; nil disables the item.
	OnPressed func()
	// Decoration is the background of the item for each state of it; the default is CupertinoMenuItemDefaultDecoration.
	Decoration widgets.WidgetStateProperty[*painting.BoxDecoration]
	// MouseCursor is the cursor for each state of the item; the default is the click cursor on the web.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// Behavior is how the item is hit by a pointer; the default is HitTestBehaviorOpaque.
	Behavior rendering.HitTestBehavior
	// NotRequestCloseOnActivate keeps the menu open when the item is activated.
	NotRequestCloseOnActivate bool
	// NotRequestFocusOnHover stops a hover from giving the focus to the item.
	NotRequestFocusOnHover bool
	// IsDestructiveAction makes the text red.
	IsDestructiveAction bool
}

// CupertinoMenuItem is an item of a CupertinoMenuAnchor: a label with an optional subtitle, leading and trailing
// widget.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem
type CupertinoMenuItem struct {
	widgets.StatelessWidget
	child widgets.Widget
	o     CupertinoMenuItemOpts
}

// NewCupertinoMenuItem creates an item that shows the child.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.new
func NewCupertinoMenuItem(child widgets.Widget, opts ...CupertinoMenuItemOpts) *CupertinoMenuItem {
	o := firstOf(opts)
	if o.Behavior == rendering.HitTestBehaviorUnspecified {
		o.Behavior = rendering.HitTestBehaviorOpaque
	}
	return &CupertinoMenuItem{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// Child is the label of the item.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.child
func (w *CupertinoMenuItem) Child() widgets.Widget { return w.child }

// Subtitle is the widget below the label.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.subtitle
func (w *CupertinoMenuItem) Subtitle() widgets.Widget { return w.o.Subtitle }

// Leading is the widget before the label.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.leading
func (w *CupertinoMenuItem) Leading() widgets.Widget { return w.o.Leading }

// Trailing is the widget after the label.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.trailing
func (w *CupertinoMenuItem) Trailing() widgets.Widget { return w.o.Trailing }

// OnPressed is called when the item is activated.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.onPressed
func (w *CupertinoMenuItem) OnPressed() func() { return w.o.OnPressed }

// IsDestructiveAction is whether the text is red.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.isDestructiveAction
func (w *CupertinoMenuItem) IsDestructiveAction() bool { return w.o.IsDestructiveAction }

// HasLeading is whether the item has a leading widget.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.hasLeading
func (w *CupertinoMenuItem) HasLeading(widgets.BuildContext) bool { return w.o.Leading != nil }

// IsDivider is false.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.isDivider
func (w *CupertinoMenuItem) IsDivider() bool { return false }

// CupertinoMenuItemDefaultDecoration is the default background of an item for the states of it: a gray veil that is
// darker when the item is dragged over or pressed, and lighter when it is focused or hovered. The colors are resolved
// in the context.
//
// Deviation: Flutter's CupertinoMenuItem.kDefaultDecoration is a constant that holds CupertinoDynamicColors, which a
// BoxDecoration of Glutter cannot hold (SPEC §12.10).
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.kDefaultDecoration
func CupertinoMenuItemDefaultDecoration(bc widgets.BuildContext) widgets.WidgetStateProperty[*painting.BoxDecoration] {
	veil := func(lightAlpha float64) *painting.BoxDecoration {
		return painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(CupertinoDynamicColorResolve(
			NewCupertinoDynamicColorWithBrightness(ui.NewColorFromRGBO(50, 50, 50, lightAlpha), ui.NewColorFromRGBO(255, 255, 255, lightAlpha)), bc))})
	}
	return widgets.NewWidgetStateMapper(widgets.WidgetStateMap[*painting.BoxDecoration]{
		{Key: widgets.WidgetStateDragged, Value: veil(0.1)},
		{Key: widgets.WidgetStatePressed, Value: veil(0.1)},
		{Key: widgets.WidgetStateFocused, Value: veil(0.075)},
		{Key: widgets.WidgetStateHovered, Value: veil(0.05)},
		{Key: widgets.WidgetStateAny, Value: painting.NewBoxDecoration()},
	})
}

var (
	menuItemTextColor         = NewCupertinoDynamicColorWithBrightness(ui.NewColorFromRGBO(0, 0, 0, 0.96), ui.NewColorFromRGBO(255, 255, 255, 0.96))
	menuItemSubtitleTextColor = NewCupertinoDynamicColorWithBrightness(ui.NewColorFromRGBO(0, 0, 0, 0.55), ui.NewColorFromRGBO(255, 255, 255, 0.4))
)

const (
	menuItemDefaultMaxLines          = 2
	menuItemDefaultLargeTextMaxLines = 100
)

func (w *CupertinoMenuItem) resolveDefaultTextStyle(bc widgets.BuildContext, textScaler painting.TextScaler) *painting.TextStyle {
	var color ui.ColorValue
	switch {
	case w.o.OnPressed == nil:
		color = CupertinoColors.SystemGrey
	case w.o.IsDestructiveAction:
		color = CupertinoColors.SystemRed
	default:
		color = menuItemTextColor
	}
	return dynamicTypeBody.resolveTextStyle(textScaler).CopyWith(painting.TextStyleCopy{
		FontSize: foundation.Some(17.0), Color: foundation.Some(CupertinoDynamicColorResolve(color, bc))})
}

func (w *CupertinoMenuItem) resolveDefaultSubtitleStyle(bc widgets.BuildContext, textScaler painting.TextScaler) *painting.TextStyle {
	brightness, _ := CupertinoThemeMaybeBrightnessOf(bc)
	paint := ui.NewPaint()
	if _, ok := CupertinoThemeMaybeBrightnessOf(bc); ok && brightness == ui.BrightnessDark {
		paint.SetBlendMode(ui.BlendModePlus)
	} else {
		paint.SetBlendMode(ui.BlendModeHardLight)
	}
	paint.SetColor(CupertinoDynamicColorResolve(menuItemSubtitleTextColor, bc))
	return dynamicTypeSubhead.resolveTextStyle(textScaler).CopyWith(painting.TextStyleCopy{
		FontSize: foundation.Some(15.0), TextBaseline: ui.TextBaselineAlphabetic, Foreground: paint})
}

func (w *CupertinoMenuItem) handleSelect(bc widgets.BuildContext) {
	if !w.o.NotRequestCloseOnActivate {
		if c := widgets.MenuControllerMaybeOf(bc); c != nil {
			c.Close()
		}
	}
	if w.o.OnPressed != nil {
		w.o.OnPressed()
	}
}

// Build builds the item.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuItem.build
func (w *CupertinoMenuItem) Build(bc widgets.BuildContext) widgets.Widget {
	textScaler, ok := widgets.MediaQueryMaybeTextScalerOf(bc)
	if !ok {
		textScaler = painting.NewTextScalerLinear(1)
	}
	defaultTextStyle := w.resolveDefaultTextStyle(bc, textScaler)
	largeText := menuLargeTextModeEnabled(bc)
	var leading, trailing, subtitle widgets.Widget
	if w.o.Leading != nil {
		leading = widgets.NewDefaultTextStyleMerge(widgets.NewIconThemeMerge(
			widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(15.0), Weight: foundation.Some(600.0), ApplyTextScaling: foundation.Some(true)}), w.o.Leading),
			widgets.DefaultTextStyleMergeOpts{Style: painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(15.0), FontWeight: ui.FontWeightW600})})
	}
	if w.o.Trailing != nil && !largeText {
		trailing = widgets.NewDefaultTextStyleMerge(widgets.NewIconThemeMerge(
			widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(21.0), ApplyTextScaling: foundation.Some(true)}), w.o.Trailing),
			widgets.DefaultTextStyleMergeOpts{Style: painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(21.0)})})
	}
	if w.o.Subtitle != nil {
		subtitle = widgets.NewDefaultTextStyleMerge(w.o.Subtitle, widgets.DefaultTextStyleMergeOpts{Style: w.resolveDefaultSubtitleStyle(bc, textScaler)})
	}
	maxLines := menuItemDefaultMaxLines
	if largeText {
		maxLines = menuItemDefaultLargeTextMaxLines
	}
	cursor := w.o.MouseCursor
	if cursor == nil {
		cursor = widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) services.MouseCursor {
			if !states.Has(widgets.WidgetStateDisabled) && foundation.IsWeb {
				return services.SystemMouseCursors.Click
			}
			return services.MouseCursorDefer
		})
	}
	decoration := w.o.Decoration
	if decoration == nil {
		decoration = CupertinoMenuItemDefaultDecoration(bc)
	}
	var onPressed func()
	if w.o.OnPressed != nil {
		onPressed = func() { w.handleSelect(bc) }
	}
	color := defaultTextStyle.Color
	return widgets.NewMediaQueryWithClampedTextScaling(
		&menuItemInteractionHandler{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}),
			onHover: w.o.OnHover, onPressed: onPressed, onFocusChange: w.o.OnFocusChange, focusNode: w.o.FocusNode, autofocus: w.o.Autofocus,
			requestFocusOnHover: !w.o.NotRequestFocusOnHover, behavior: w.o.Behavior, mouseCursor: cursor, decoration: decoration,
			child: widgets.NewDefaultTextStyleMerge(widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: color}),
				&menuItemLabel{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), padding: w.o.Padding, constraints: w.o.Constraints,
					trailing: trailing, leading: leading, leadingMidpointAlignment: w.o.LeadingMidpointAlignment,
					trailingMidpointAlignment: w.o.TrailingMidpointAlignment, leadingWidth: w.o.LeadingWidth, trailingWidth: w.o.TrailingWidth,
					subtitle: subtitle, child: widgets.NewDefaultTextStyleMerge(w.child, widgets.DefaultTextStyleMergeOpts{Style: defaultTextStyle})}),
				widgets.DefaultTextStyleMergeOpts{MaxLines: foundation.Some(maxLines), Overflow: painting.TextOverflowEllipsis,
					SoftWrap: foundation.Some(true), Style: painting.NewTextStyle(painting.TextStyleOpts{Color: color})})},
		widgets.MediaQueryClampedTextScalingOpts{MinScaleFactor: menuMinimumTextScaleFactor, MaxScaleFactor: menuMaximumTextScaleFactor})
}

const (
	menuItemDefaultHorizontalWidth = 16.0
	menuItemLeadingWidthSlope      = -311.0 / 1000
	menuItemLeadingWidthIntercept  = 10.0
	menuItemLeadingMidSlope        = 118.0 / 1000000
	menuItemLeadingMidIntercept    = 73.0 / 125
	menuItemTrailingWidthSlope     = 1.0 / 10
	menuItemTrailingWidthIntercept = 22.0
	menuItemFirstBaselineSlope     = 14.0 / 11
	menuItemLastBaselineSlope      = 71.0 / 100
)

// menuItemLabel lays out the leading widget, the label with its subtitle and the trailing widget of an item.
//
// Flutter: cupertino/menu_anchor.dart _CupertinoMenuItemLabel
type menuItemLabel struct {
	widgets.StatelessWidget
	leading, trailing, child, subtitle widgets.Widget
	leadingWidth, trailingWidth        foundation.Opt[float64]
	leadingMidpointAlignment           painting.AlignmentGeometry
	trailingMidpointAlignment          painting.AlignmentGeometry
	padding                            painting.EdgeInsetsGeometry
	constraints                        foundation.Opt[rendering.BoxConstraints]
}

func (w *menuItemLabel) Build(bc widgets.BuildContext) widgets.Widget {
	textDirection, ok := widgets.MaybeDirectionalityOf(bc)
	if !ok {
		textDirection = ui.TextDirectionLTR
	}
	textScaler, ok := widgets.MediaQueryMaybeTextScalerOf(bc)
	if !ok {
		textScaler = painting.NewTextScalerLinear(1)
	}
	pixelRatio := 1.0
	if r, ok := widgets.MediaQueryMaybeDevicePixelRatioOf(bc); ok {
		pixelRatio = r
	}
	body := dynamicTypeBody.resolveTextStyle(textScaler)
	lineHeight := body.FontSize.Value() * body.Height.Value()
	hasLeadingOfAnchor, _ := CupertinoMenuAnchorMaybeHasLeadingOf(bc)
	showLeadingWidget := w.leading != nil || hasLeadingOfAnchor
	to := 1 / pixelRatio
	minimumHeight := menuRoundToDivisible(lineHeight*menuItemFirstBaselineSlope, to) + menuRoundToDivisible(lineHeight*menuItemLastBaselineSlope, to)
	constraints := w.constraints.OrElse(rendering.BoxConstraints{MinHeight: minimumHeight, MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)})
	padding := w.padding
	if padding == nil {
		padding = painting.NewEdgeInsetsSymmetric(0, math.Max(0, minimumHeight-lineHeight)/2)
	}
	units := menuNormalizeTextScale(textScaler)
	leadingWidth := w.leadingWidth.OrElse(menuItemDefaultHorizontalWidth)
	if !w.leadingWidth.IsSome() && showLeadingWidget {
		leadingWidth = menuRoundToDivisible(menuItemLeadingWidthSlope*units+menuItemLeadingWidthIntercept+lineHeight, to)
	}
	trailingWidth := w.trailingWidth.OrElse(menuItemDefaultHorizontalWidth)
	if !w.trailingWidth.IsSome() && w.trailing != nil {
		trailingWidth = menuRoundToDivisible(menuItemTrailingWidthSlope*units+menuItemTrailingWidthIntercept+lineHeight, to)
	}
	var stack []widgets.Widget
	if showLeadingWidget {
		alignment := w.leadingMidpointAlignment
		if alignment == nil {
			ratio := menuItemLeadingMidSlope*units + menuItemLeadingMidIntercept
			alignment = painting.AlignmentDirectional{Start: ratio*2 - 1}
		}
		stack = append(stack, widgets.NewPositionedDirectional(newMenuAlignMidpoint(alignment, w.leading), widgets.PositionedDirectionalOpts{
			Start: foundation.Some(0.0), Top: foundation.Some(0.0), Bottom: foundation.Some(0.0), Width: foundation.Some(leadingWidth)}))
	}
	var content widgets.Widget
	if w.subtitle == nil {
		content = widgets.NewAlign(w.child, widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart})
	} else {
		content = widgets.NewColumn([]widgets.Widget{w.child, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(1.0)}), w.subtitle},
			widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch, MainAxisAlignment: rendering.MainAxisAlignmentCenter})
	}
	stack = append(stack, widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(leadingWidth, 0, trailingWidth, 0), content))
	if w.trailing != nil {
		alignment := w.trailingMidpointAlignment
		if alignment == nil {
			offset := trailingWidth/2 + 6
			ratio := (trailingWidth - offset) / trailingWidth
			alignment = painting.AlignmentDirectional{Start: ratio*2 - 1}
		}
		stack = append(stack, widgets.NewPositionedDirectional(newMenuAlignMidpoint(alignment, w.trailing), widgets.PositionedDirectionalOpts{
			End: foundation.Some(0.0), Top: foundation.Some(0.0), Bottom: foundation.Some(0.0), Width: foundation.Some(trailingWidth)}))
	}
	_ = textDirection
	return widgets.NewConstrainedBox(constraints, widgets.NewPadding(padding, widgets.NewStack(stack)))
}

// menuAlignMidpoint aligns the middle of the child at the point of the alignment, and keeps the child inside.
//
// Flutter: cupertino/menu_anchor.dart _AlignMidpoint
type menuAlignMidpoint struct {
	widgets.SingleChildRenderObjectWidget
	alignment painting.AlignmentGeometry
}

func newMenuAlignMidpoint(alignment painting.AlignmentGeometry, child widgets.Widget) *menuAlignMidpoint {
	return &menuAlignMidpoint{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child), alignment: alignment}
}

func (w *menuAlignMidpoint) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	td, _ := widgets.MaybeDirectionalityOf(bc)
	r := &renderMenuAlignMidpoint{}
	rendering.InitRenderAligningShiftedBox(&r.RenderAligningShiftedBox, r, w.alignment, td, nil)
	return r
}

func (w *menuAlignMidpoint) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderMenuAlignMidpoint)
	td, _ := widgets.MaybeDirectionalityOf(bc)
	r.SetAlignment(w.alignment)
	r.SetTextDirection(td)
}

// renderMenuAlignMidpoint is the render object of menuAlignMidpoint.
//
// Flutter: cupertino/menu_anchor.dart _RenderAlignMidpoint
type renderMenuAlignMidpoint struct {
	rendering.RenderPositionedBox
}

func (r *renderMenuAlignMidpoint) PerformLayout() {
	r.RenderPositionedBox.PerformLayout()
	child := r.Child()
	if child == nil {
		return
	}
	size := r.Size()
	offset := r.ResolvedAlignment().AlongSize(size).Sub(ui.Offset{Dx: child.Size().Width / 2, Dy: child.Size().Height / 2})
	dx := ui.ClampDouble(offset.Dx, 0.0, size.Width-child.Size().Width)
	dy := ui.ClampDouble(offset.Dy, 0.0, size.Height-child.Size().Height)
	rendering.BoxParentDataOf(child).Offset = ui.Offset{Dx: dx, Dy: dy}
}

// menuItemInteractionHandler is the focus, hover, press and swipe handling of an item, with the states that its
// appearance follows.
//
// Flutter: cupertino/menu_anchor.dart _CupertinoMenuItemInteractionHandler
type menuItemInteractionHandler struct {
	widgets.StatefulWidget
	onHover             func(bool)
	onPressed           func()
	onFocusChange       func(bool)
	focusNode           *widgets.FocusNode
	autofocus           bool
	requestFocusOnHover bool
	behavior            rendering.HitTestBehavior
	mouseCursor         widgets.WidgetStateProperty[services.MouseCursor]
	decoration          widgets.WidgetStateProperty[*painting.BoxDecoration]
	child               widgets.Widget
}

func (w *menuItemInteractionHandler) CreateState() widgets.State {
	return &menuItemInteractionHandlerState{}
}

type menuItemInteractionHandlerState struct {
	widgets.StateOf[*menuItemInteractionHandler]
	actions          widgets.ActionMap
	gestures         map[reflect.Type]widgets.GestureRecognizerFactory
	gestureSettings  foundation.Opt[gestures.DeviceGestureSettings]
	internalFocus    *widgets.FocusNode
	statesController *widgets.WidgetStatesController
}

func (s *menuItemInteractionHandlerState) focusNode() *widgets.FocusNode {
	if n := s.Widget().focusNode; n != nil {
		return n
	}
	return s.internalFocus
}

func (s *menuItemInteractionHandlerState) has(state widgets.WidgetState) bool {
	return s.statesController.Value().Has(state)
}

func (s *menuItemInteractionHandlerState) isEnabled() bool {
	return !s.has(widgets.WidgetStateDisabled)
}
func (s *menuItemInteractionHandlerState) isHovered() bool { return s.has(widgets.WidgetStateHovered) }

func (s *menuItemInteractionHandlerState) set(state widgets.WidgetState, v bool) {
	s.statesController.Update(state, v)
}

func (s *menuItemInteractionHandlerState) InitState() {
	s.statesController = widgets.NewWidgetStatesController()
	s.actions = widgets.ActionMap{
		reflect.TypeFor[widgets.ActivateIntent]():       widgets.NewCallbackAction(func(widgets.ActivateIntent) any { s.handleActivation(); return nil }),
		reflect.TypeFor[widgets.ButtonActivateIntent](): widgets.NewCallbackAction(func(widgets.ButtonActivateIntent) any { s.handleActivation(); return nil }),
	}
	if s.Widget().focusNode == nil {
		s.internalFocus = widgets.NewFocusNode()
	}
	s.set(widgets.WidgetStateDisabled, s.Widget().onPressed == nil)
	s.set(widgets.WidgetStateFocused, s.focusNode().HasPrimaryFocus())
}

func (s *menuItemInteractionHandlerState) DidUpdateWidget(old *menuItemInteractionHandler) {
	w := s.Widget()
	if w.focusNode != old.focusNode {
		if w.focusNode != nil {
			if s.internalFocus != nil {
				s.internalFocus.Dispose()
			}
			s.internalFocus = nil
		} else {
			s.internalFocus = widgets.NewFocusNode()
		}
		s.set(widgets.WidgetStateFocused, s.focusNode().HasPrimaryFocus())
	}
	if (w.onPressed == nil) != (old.onPressed == nil) {
		if w.onPressed == nil {
			for _, st := range []widgets.WidgetState{widgets.WidgetStateHovered, widgets.WidgetStatePressed, widgets.WidgetStateDragged, widgets.WidgetStateFocused} {
				s.set(st, false)
			}
			s.set(widgets.WidgetStateDisabled, true)
		} else {
			s.set(widgets.WidgetStateDisabled, false)
		}
	}
}

func (s *menuItemInteractionHandlerState) Dispose() {
	s.statesController.Dispose()
	if s.internalFocus != nil {
		s.internalFocus.Dispose()
		s.internalFocus = nil
	}
}

func (s *menuItemInteractionHandlerState) handleFocusChange(bool) {
	focused := s.focusNode().HasPrimaryFocus()
	s.set(widgets.WidgetStateFocused, focused)
	if f := s.Widget().onFocusChange; f != nil {
		f(focused)
	}
}

func (s *menuItemInteractionHandlerState) handleActivation() {
	s.set(widgets.WidgetStateDragged, false)
	s.set(widgets.WidgetStatePressed, false)
	if f := s.Widget().onPressed; f != nil {
		f()
	}
}

func (s *menuItemInteractionHandlerState) handleTapUp(gestures.TapUpDetails) {
	s.set(widgets.WidgetStatePressed, false)
	if f := s.Widget().onPressed; f != nil {
		f()
	}
}

func (s *menuItemInteractionHandlerState) handlePointerExit(*gestures.PointerExitEvent) {
	if s.isHovered() {
		s.set(widgets.WidgetStateHovered, false)
		s.set(widgets.WidgetStateFocused, false)
		if f := s.Widget().onHover; f != nil {
			f(false)
		}
	}
}

func (s *menuItemInteractionHandlerState) handlePointerHover(*gestures.PointerHoverEvent) {
	if s.isHovered() {
		return
	}
	s.set(widgets.WidgetStateHovered, true)
	if f := s.Widget().onHover; f != nil {
		f(true)
	}
	if s.Widget().requestFocusOnHover {
		s.focusNode().RequestFocus()
		widgets.FocusTraversalGroupOf(s.Context()).InvalidateScopeData(widgets.FocusScopeOf(s.Context()))
	}
}

func (s *menuItemInteractionHandlerState) handleDismissMenu() {
	widgets.ActionsInvoke(s.Context(), widgets.DismissIntent{})
}

func (s *menuItemInteractionHandlerState) handleSwipeEnter() {
	if !s.isEnabled() {
		return
	}
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformAndroid:
		widgets.WidgetsBindingOf(s.Context()).HapticFeedback().SelectionClick()
	}
	s.set(widgets.WidgetStateDragged, true)
}

func (s *menuItemInteractionHandlerState) handleSwipeExit() {
	if s.Mounted() {
		s.set(widgets.WidgetStateDragged, false)
	}
}

func (s *menuItemInteractionHandlerState) handleSwipeCompleted() {
	if s.Mounted() && s.isEnabled() {
		s.handleActivation()
	}
}

func (s *menuItemInteractionHandlerState) buildStatefulAppearance(bc widgets.BuildContext, states widgets.WidgetStates, child widgets.Widget) widgets.Widget {
	w := s.Widget()
	cursor := w.mouseCursor.Resolve(states)
	decoration := w.decoration.Resolve(states)
	hasBackground := decoration.Color.IsSome() || decoration.Gradient != nil
	blend := decoration.BackgroundBlendMode
	if !foundation.IsWeb && hasBackground && blend == ui.BlendModeUnspecified {
		if brightness, ok := CupertinoThemeMaybeBrightnessOf(bc); ok && brightness == ui.BrightnessLight {
			blend = ui.BlendModeMultiply
		} else {
			blend = ui.BlendModePlus
		}
	}
	var onHover func(*gestures.PointerHoverEvent)
	var onExit func(*gestures.PointerExitEvent)
	if s.isEnabled() {
		onHover, onExit = s.handlePointerHover, s.handlePointerExit
	}
	return widgets.NewMouseRegion(widgets.NewDecoratedBox(decoration.CopyWith(painting.BoxDecorationCopy{BackgroundBlendMode: blend}), child),
		widgets.MouseRegionOpts{OnHover: onHover, OnExit: onExit, HitTestBehavior: rendering.HitTestBehaviorDeferToChild, Cursor: cursor})
}

func (s *menuItemInteractionHandlerState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	settings := foundation.None[gestures.DeviceGestureSettings]()
	if g, ok := widgets.MediaQueryMaybeGestureSettingsOf(bc); ok {
		settings = foundation.Some(g)
	}
	if s.gestureSettings != settings {
		s.gestureSettings = settings
		s.gestures = nil
	}
	if s.gestures == nil {
		binding := widgets.WidgetsBindingOf(bc).Gestures()
		s.gestures = map[reflect.Type]widgets.GestureRecognizerFactory{
			reflect.TypeFor[*gestures.TapGestureRecognizer](): widgets.NewGestureRecognizerFactoryWithHandlers(
				func() *gestures.TapGestureRecognizer {
					return gestures.NewTapGestureRecognizer(binding, gestures.TapGestureRecognizerOpts{DebugOwner: s})
				},
				func(r *gestures.TapGestureRecognizer) {
					r.OnTapDown = func(gestures.TapDownDetails) { s.set(widgets.WidgetStatePressed, true) }
					r.OnTapUp = s.handleTapUp
					r.OnTapCancel = func() { s.set(widgets.WidgetStatePressed, false) }
					r.GestureSettings = s.gestureSettings
				}),
		}
	}
	enabled := s.isEnabled()
	var onDismiss foundation.VoidCallback
	actions := widgets.ActionMap{}
	recognizers := map[reflect.Type]widgets.GestureRecognizerFactory{}
	if enabled {
		onDismiss, actions, recognizers = s.handleDismissMenu, s.actions, s.gestures
	}
	return widgets.NewMergeSemantics(widgets.NewSemanticsFromProperties(&semantics.SemanticsProperties{Enabled: foundation.Some(enabled), OnDismiss: onDismiss},
		widgets.NewActions(actions, widgets.NewFocus(
			newMenuSwipeTarget(s.handleSwipeEnter, s.handleSwipeExit, s.handleSwipeCompleted,
				widgets.NewValueListenableBuilder[widgets.WidgetStates](s.statesController, s.buildStatefulAppearance,
					widgets.NewRawGestureDetector(w.child, widgets.RawGestureDetectorOpts{Behavior: w.behavior, Gestures: recognizers}))),
			widgets.FocusOpts{Autofocus: enabled && w.autofocus, FocusNode: s.focusNode(), CanRequestFocus: foundation.Some(enabled),
				SkipTraversal: foundation.Some(!enabled), OnFocusChange: s.handleFocusChange}))))
}
