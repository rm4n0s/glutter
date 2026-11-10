package cupertino

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	pickerDefaultDiameterRatio      = 1.07
	pickerDefaultPerspective        = 0.003
	pickerSqueeze                   = 1.45
	pickerOverAndUnderCenterOpacity = 0.447
	pickerTapToScrollDuration       = 300 * time.Millisecond
)

// CupertinoPickerOpts are the optional parameters of NewCupertinoPicker and NewCupertinoPickerBuilder.
//
// Flutter: cupertino/picker.dart CupertinoPicker.new
type CupertinoPickerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// DiameterRatio is the ratio of the diameter of the wheel to the height of the picker; the default is 1.07.
	DiameterRatio foundation.Opt[float64]
	// BackgroundColor is the background of the picker.
	BackgroundColor ui.ColorValue
	// OffAxisFraction is how far the wheel is off the center, to look like a ball seen from the side.
	OffAxisFraction float64
	// UseMagnifier magnifies the item in the center.
	UseMagnifier bool
	// Magnification is how much the center item is magnified; the default is 1.
	Magnification foundation.Opt[float64]
	// ScrollController is the controller of the wheel, which reads and sets the selected item.
	ScrollController *widgets.FixedExtentScrollController
	// Squeeze is how tightly the items are packed on the wheel; the default is 1.45.
	Squeeze foundation.Opt[float64]
	// ChangeReportingBehavior is when OnSelectedItemChanged is called; the default is on every scroll update.
	ChangeReportingBehavior widgets.ChangeReportingBehavior
	// SelectionOverlay is drawn over the center item; the default is CupertinoPickerDefaultSelectionOverlay.
	SelectionOverlay widgets.Widget
	// NoSelectionOverlay draws no overlay over the center item.
	NoSelectionOverlay bool
	// Looping makes the list of children wrap around (NewCupertinoPicker only).
	Looping bool
	// ChildCount is the number of items (NewCupertinoPickerBuilder only); none is an infinite list.
	ChildCount foundation.Opt[int]
}

// CupertinoPicker is an iOS style wheel picker: a list of items on a cylinder that snaps to its items.
//
// Flutter: cupertino/picker.dart CupertinoPicker
type CupertinoPicker struct {
	widgets.StatefulWidget
	itemExtent            float64
	onSelectedItemChanged func(int)
	childDelegate         widgets.ListWheelChildDelegate
	o                     CupertinoPickerOpts
}

func (o *CupertinoPickerOpts) resolve() {
	if o.DiameterRatio.IsNone() {
		o.DiameterRatio = foundation.Some(pickerDefaultDiameterRatio)
	}
	if o.Magnification.IsNone() {
		o.Magnification = foundation.Some(1.0)
	}
	if o.Squeeze.IsNone() {
		o.Squeeze = foundation.Some(pickerSqueeze)
	}
	if o.ChangeReportingBehavior == widgets.ChangeReportingBehaviorUnspecified {
		o.ChangeReportingBehavior = widgets.ChangeReportingBehaviorOnScrollUpdate
	}
	if o.SelectionOverlay == nil && !o.NoSelectionOverlay {
		o.SelectionOverlay = NewCupertinoPickerDefaultSelectionOverlay()
	}
}

// checkPickerArguments panics in debug builds with a tag for each argument that is not positive.
func checkPickerArguments(itemExtent float64, o CupertinoPickerOpts) {
	if !foundation.DebugMode {
		return
	}
	switch {
	case o.DiameterRatio.Value() <= 0:
		panic(errors.New(tagCupertinoPickerDiameterRatioNotPositive, "diameterRatio must be greater than 0", "diameterRatio", o.DiameterRatio.Value()))
	case o.Magnification.Value() <= 0:
		panic(errors.New(tagCupertinoPickerMagnificationNotPositive, "magnification must be greater than 0", "magnification", o.Magnification.Value()))
	case itemExtent <= 0:
		panic(errors.New(tagCupertinoPickerItemExtentNotPositive, "itemExtent must be greater than 0", "itemExtent", itemExtent))
	case o.Squeeze.Value() <= 0:
		panic(errors.New(tagCupertinoPickerSqueezeNotPositive, "squeeze must be greater than 0", "squeeze", o.Squeeze.Value()))
	}
}

// NewCupertinoPicker creates a picker of the children. In debug builds it panics with the tag
// CupertinoPickerDiameterRatioNotPositive, CupertinoPickerMagnificationNotPositive, CupertinoPickerItemExtentNotPositive
// or CupertinoPickerSqueezeNotPositive when that argument is not greater than 0.
//
// Flutter: cupertino/picker.dart CupertinoPicker.new
func NewCupertinoPicker(itemExtent float64, onSelectedItemChanged func(int), children []widgets.Widget, opts ...CupertinoPickerOpts) *CupertinoPicker {
	o := firstOf(opts)
	o.resolve()
	checkPickerArguments(itemExtent, o)
	var delegate widgets.ListWheelChildDelegate = widgets.NewListWheelChildListDelegate(children)
	if o.Looping {
		delegate = widgets.NewListWheelChildLoopingListDelegate(children)
	}
	return &CupertinoPicker{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), itemExtent: itemExtent,
		onSelectedItemChanged: onSelectedItemChanged, childDelegate: delegate, o: o}
}

// NewCupertinoPickerBuilder creates a picker whose items the builder makes. It panics like NewCupertinoPicker.
//
// Flutter: cupertino/picker.dart CupertinoPicker.builder
func NewCupertinoPickerBuilder(itemExtent float64, onSelectedItemChanged func(int), itemBuilder widgets.NullableIndexedWidgetBuilder, opts ...CupertinoPickerOpts) *CupertinoPicker {
	o := firstOf(opts)
	o.resolve()
	checkPickerArguments(itemExtent, o)
	return &CupertinoPicker{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), itemExtent: itemExtent,
		onSelectedItemChanged: onSelectedItemChanged,
		childDelegate:         widgets.NewListWheelChildBuilderDelegate(itemBuilder, widgets.ListWheelChildBuilderDelegateOpts{ChildCount: o.ChildCount}), o: o}
}

// ItemExtent is the height of each item.
//
// Flutter: cupertino/picker.dart CupertinoPicker.itemExtent
func (w *CupertinoPicker) ItemExtent() float64 { return w.itemExtent }

// BackgroundColor is the background of the picker.
//
// Flutter: cupertino/picker.dart CupertinoPicker.backgroundColor
func (w *CupertinoPicker) BackgroundColor() ui.ColorValue { return w.o.BackgroundColor }

// OnSelectedItemChanged is called with the index of the selected item.
//
// Flutter: cupertino/picker.dart CupertinoPicker.onSelectedItemChanged
func (w *CupertinoPicker) OnSelectedItemChanged() func(int) { return w.onSelectedItemChanged }

// ChildDelegate supplies the items.
//
// Flutter: cupertino/picker.dart CupertinoPicker.childDelegate
func (w *CupertinoPicker) ChildDelegate() widgets.ListWheelChildDelegate { return w.childDelegate }

// CreateState creates the state.
//
// Flutter: cupertino/picker.dart CupertinoPicker.createState
func (w *CupertinoPicker) CreateState() widgets.State { return &cupertinoPickerState{} }

type cupertinoPickerState struct {
	widgets.StateOf[*CupertinoPicker]
	lastHapticIndex      int
	lastMiddlePosition   foundation.Opt[int]
	controller           *widgets.FixedExtentScrollController
	enableHapticFeedback bool
	scrollHandle         foundation.ListenerHandle
	listenedTo           *widgets.FixedExtentScrollController
}

func (s *cupertinoPickerState) effectiveController() *widgets.FixedExtentScrollController {
	if c := s.Widget().o.ScrollController; c != nil {
		return c
	}
	return s.controller
}

func (s *cupertinoPickerState) listen() {
	s.listenedTo = s.effectiveController()
	s.scrollHandle = s.listenedTo.AddListener(s.handleScroll)
}

func (s *cupertinoPickerState) InitState() {
	s.enableHapticFeedback = true
	if s.Widget().o.ScrollController == nil {
		s.controller = widgets.NewFixedExtentScrollController()
	}
	s.lastHapticIndex = s.effectiveController().InitialItem()
	s.listen()
}

func (s *cupertinoPickerState) DidUpdateWidget(old *CupertinoPicker) {
	w := s.Widget()
	switch {
	case w.o.ScrollController != nil && old.o.ScrollController == nil:
		if s.controller != nil {
			s.controller.Dispose()
			s.controller = nil
		}
		s.listenedTo.RemoveListener(s.scrollHandle)
		s.listen()
	case w.o.ScrollController == nil && old.o.ScrollController != nil:
		old.o.ScrollController.RemoveListener(s.scrollHandle)
		s.controller = widgets.NewFixedExtentScrollController()
		s.listen()
	case w.o.ScrollController != nil && w.o.ScrollController != old.o.ScrollController:
		old.o.ScrollController.RemoveListener(s.scrollHandle)
		s.listen()
	}
}

func (s *cupertinoPickerState) Dispose() {
	s.listenedTo.RemoveListener(s.scrollHandle)
	if s.controller != nil {
		s.controller.Dispose()
	}
}

func (s *cupertinoPickerState) handleHapticFeedback(index int) {
	if !s.enableHapticFeedback {
		return
	}
	if foundation.DefaultTargetPlatform() == foundation.TargetPlatformIOS && index != s.lastHapticIndex {
		s.lastHapticIndex = index
		binding := widgets.WidgetsBindingOf(s.Context())
		binding.HapticFeedback().SelectionClick()
		binding.SystemSound().Play(services.SystemSoundTypeTick)
	}
}

func (s *cupertinoPickerState) handleScroll() {
	c := s.effectiveController()
	index := c.SelectedItem()
	fractionalOffset := c.Offset() / s.Widget().itemExtent
	currentPosition := int(math.Floor(fractionalOffset))
	currentItemOffset := fractionalOffset - float64(index)
	if s.lastMiddlePosition.IsNone() || currentPosition != s.lastMiddlePosition.Value() || math.Abs(currentItemOffset) <= 0.1 {
		s.handleHapticFeedback(index)
	}
	s.lastMiddlePosition = foundation.Some(currentPosition)
}

func (s *cupertinoPickerState) handleChildTap(index int) {
	s.enableHapticFeedback = false
	s.effectiveController().AnimateToItem(index, pickerTapToScrollDuration, animation.Curves.EaseInOut).Then(func() {
		s.enableHapticFeedback = true
		// Deviation: Dart continues after the await in a microtask, when a scroll view that was removed during the animation has
		// let go of its controller; TickerFuture.Then runs while the scrollable is disposed, so the controller may have no scroll view.
		if c := s.effectiveController(); c.HasClients() {
			s.lastHapticIndex = c.SelectedItem()
		}
	})
}

func (s *cupertinoPickerState) buildSelectionOverlay(overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	height := w.itemExtent * w.o.Magnification.Value()
	return widgets.NewIgnorePointer(widgets.NewCenter(widgets.NewConstrainedBox(
		rendering.BoxConstraints{MinWidth: math.Inf(1), MaxWidth: math.Inf(1), MinHeight: height, MaxHeight: height}, overlay)))
}

// Build builds the wheel and the selection overlay.
//
// Flutter: cupertino/picker.dart _CupertinoPickerState.build
func (s *cupertinoPickerState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	textStyle := CupertinoThemeOf(bc).TextTheme().PickerTextStyle()
	var background foundation.Opt[ui.Color]
	if w.o.BackgroundColor != nil {
		background = foundation.Some(CupertinoDynamicColorResolve(w.o.BackgroundColor, bc))
	}
	children := []widgets.Widget{widgets.NewPositionedFill(newPickerSemantics(s.effectiveController(),
		widgets.NewListWheelScrollViewUseDelegate(w.itemExtent, &pickerChildDelegateWrapper{wrapped: w.childDelegate, onTappedChild: s.handleChildTap}, widgets.ListWheelScrollViewOpts{
			Controller: s.effectiveController(), Physics: widgets.NewFixedExtentScrollPhysics(nil),
			DiameterRatio: w.o.DiameterRatio, OffAxisFraction: w.o.OffAxisFraction, UseMagnifier: w.o.UseMagnifier, Magnification: w.o.Magnification,
			OverAndUnderCenterOpacity: foundation.Some(pickerOverAndUnderCenterOpacity), Squeeze: w.o.Squeeze, OnSelectedItemChanged: w.onSelectedItemChanged,
			DragStartBehavior: gestures.DragStartBehaviorDown, ChangeReportingBehavior: w.o.ChangeReportingBehavior})))}
	if w.o.SelectionOverlay != nil {
		children = append(children, s.buildSelectionOverlay(w.o.SelectionOverlay))
	}
	return widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: background}),
		widgets.NewDefaultTextStyle(textStyle, widgets.NewStack(children)))
}

// CupertinoPickerDefaultSelectionOverlayOpts are the optional parameters of NewCupertinoPickerDefaultSelectionOverlay.
//
// Flutter: cupertino/picker.dart CupertinoPickerDefaultSelectionOverlay.new
type CupertinoPickerDefaultSelectionOverlayOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Background is the color of the overlay; the default is CupertinoColors.TertiarySystemFill.
	Background ui.ColorValue
	// NotCapStartEdge makes the start edge square.
	NotCapStartEdge bool
	// NotCapEndEdge makes the end edge square.
	NotCapEndEdge bool
}

// CupertinoPickerDefaultSelectionOverlay is the rounded translucent bar drawn over the center item of a picker.
//
// Flutter: cupertino/picker.dart CupertinoPickerDefaultSelectionOverlay
type CupertinoPickerDefaultSelectionOverlay struct {
	widgets.StatelessWidget
	o CupertinoPickerDefaultSelectionOverlayOpts
}

// NewCupertinoPickerDefaultSelectionOverlay creates the overlay.
//
// Flutter: cupertino/picker.dart CupertinoPickerDefaultSelectionOverlay.new
func NewCupertinoPickerDefaultSelectionOverlay(opts ...CupertinoPickerDefaultSelectionOverlayOpts) *CupertinoPickerDefaultSelectionOverlay {
	o := firstOf(opts)
	if o.Background == nil {
		o.Background = CupertinoColors.TertiarySystemFill
	}
	return &CupertinoPickerDefaultSelectionOverlay{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// CapStartEdge is whether the start edge is rounded.
//
// Flutter: cupertino/picker.dart CupertinoPickerDefaultSelectionOverlay.capStartEdge
func (w *CupertinoPickerDefaultSelectionOverlay) CapStartEdge() bool { return !w.o.NotCapStartEdge }

// CapEndEdge is whether the end edge is rounded.
//
// Flutter: cupertino/picker.dart CupertinoPickerDefaultSelectionOverlay.capEndEdge
func (w *CupertinoPickerDefaultSelectionOverlay) CapEndEdge() bool { return !w.o.NotCapEndEdge }

// Background is the color of the overlay.
//
// Flutter: cupertino/picker.dart CupertinoPickerDefaultSelectionOverlay.background
func (w *CupertinoPickerDefaultSelectionOverlay) Background() ui.ColorValue { return w.o.Background }

const (
	pickerOverlayHorizontalMargin = 9.0
	pickerOverlayRadius           = 8.0
)

// Build builds the bar.
//
// Flutter: cupertino/picker.dart CupertinoPickerDefaultSelectionOverlay.build
func (w *CupertinoPickerDefaultSelectionOverlay) Build(bc widgets.BuildContext) widgets.Widget {
	radius := ui.NewRadiusCircular(pickerOverlayRadius)
	start, end := ui.Radius{}, ui.Radius{}
	marginStart, marginEnd := 0.0, 0.0
	if w.CapStartEdge() {
		start, marginStart = radius, pickerOverlayHorizontalMargin
	}
	if w.CapEndEdge() {
		end, marginEnd = radius, pickerOverlayHorizontalMargin
	}
	return widgets.NewContainer(nil, widgets.ContainerOpts{
		Margin: painting.NewEdgeInsetsDirectionalFromSTEB(marginStart, 0, marginEnd, 0),
		Decoration: painting.NewShapeDecoration(painting.NewRoundedSuperellipseBorder(painting.RoundedSuperellipseBorderOpts{
			BorderRadius: painting.NewBorderRadiusDirectionalHorizontal(start, end)}),
			painting.ShapeDecorationOpts{Color: foundation.Some(CupertinoDynamicColorResolve(w.o.Background, bc))})})
}

// pickerSemantics gives a picker the semantics of one adjustable value: the label of the selected item, with the
// labels of its neighbors as the increased and decreased value.
//
// Flutter: cupertino/picker.dart _CupertinoPickerSemantics
type pickerSemantics struct {
	widgets.SingleChildRenderObjectWidget
	scrollController *widgets.FixedExtentScrollController
}

func newPickerSemantics(controller *widgets.FixedExtentScrollController, child widgets.Widget) *pickerSemantics {
	return &pickerSemantics{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child), scrollController: controller}
}

func (w *pickerSemantics) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	widgets.DebugCheckHasDirectionality(bc)
	r := &renderPickerSemantics{textDirection: widgets.DirectionalityOf(bc)}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	r.updateController(nil, w.scrollController)
	return r
}

func (w *pickerSemantics) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	widgets.DebugCheckHasDirectionality(bc)
	r := ro.(*renderPickerSemantics)
	r.setTextDirection(widgets.DirectionalityOf(bc))
	r.updateController(r.controller, w.scrollController)
}

type renderPickerSemantics struct {
	rendering.RenderProxyBox
	controller    *widgets.FixedExtentScrollController
	handle        foundation.ListenerHandle
	textDirection ui.TextDirection
	currentIndex  int
}

func (r *renderPickerSemantics) updateController(old, value *widgets.FixedExtentScrollController) {
	if value == old {
		return
	}
	if old != nil {
		old.RemoveListener(r.handle)
	} else {
		r.currentIndex = value.InitialItem()
	}
	r.handle = value.AddListener(r.handleScrollUpdate)
	r.controller = value
}

func (r *renderPickerSemantics) setTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderPickerSemantics) handleScrollUpdate() {
	if r.controller.SelectedItem() == r.currentIndex {
		return
	}
	r.currentIndex = r.controller.SelectedItem()
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderPickerSemantics) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBox.DescribeSemanticsConfiguration(config)
	config.SetIsSemanticBoundary(true)
	config.SetTextDirection(r.textDirection)
}

func (r *renderPickerSemantics) AssembleSemanticsNode(node *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, children []*semantics.SemanticsNode) {
	if len(children) == 0 {
		r.RenderProxyBox.AssembleSemanticsNode(node, config, children)
		return
	}
	indexed := map[int]*semantics.SemanticsNode{}
	children[0].VisitChildren(func(child *semantics.SemanticsNode) bool {
		if i, ok := child.IndexInParent.Get(); ok {
			indexed[i] = child
		}
		return true
	})
	current := indexed[r.currentIndex]
	if current == nil || current.Label() == "" {
		node.UpdateWith(config, nil)
		return
	}
	config.SetValue(current.Label())
	if next := indexed[r.currentIndex+1]; next != nil && next.Label() != "" {
		config.SetIncreasedValue(next.Label())
		config.SetOnIncrease(func() { r.controller.JumpToItem(r.currentIndex + 1) })
	}
	if previous := indexed[r.currentIndex-1]; previous != nil && previous.Label() != "" {
		config.SetDecreasedValue(previous.Label())
		config.SetOnDecrease(func() { r.controller.JumpToItem(r.currentIndex - 1) })
	}
	node.UpdateWith(config, nil)
}

func (r *renderPickerSemantics) Dispose() {
	r.RenderProxyBox.Dispose()
	r.controller.RemoveListener(r.handle)
}

// pickerChildDelegateWrapper wraps each child of a picker in a gesture detector that scrolls to it when it is tapped.
//
// Flutter: cupertino/picker.dart _CupertinoPickerListWheelChildDelegateWrapper
type pickerChildDelegateWrapper struct {
	wrapped       widgets.ListWheelChildDelegate
	onTappedChild func(index int)
}

func (d *pickerChildDelegateWrapper) Build(bc widgets.BuildContext, index int) widgets.Widget {
	child := d.wrapped.Build(bc, index)
	if child == nil {
		return nil
	}
	return widgets.NewGestureDetector(child, widgets.GestureDetectorOpts{Behavior: rendering.HitTestBehaviorTranslucent, ExcludeFromSemantics: true,
		OnTap: func() { d.onTappedChild(index) }})
}

func (d *pickerChildDelegateWrapper) EstimatedChildCount() foundation.Opt[int] {
	return d.wrapped.EstimatedChildCount()
}
func (d *pickerChildDelegateWrapper) TrueIndexOf(index int) int { return d.wrapped.TrueIndexOf(index) }
func (d *pickerChildDelegateWrapper) ShouldRebuild(old widgets.ListWheelChildDelegate) bool {
	o, ok := old.(*pickerChildDelegateWrapper)
	return !ok || d.wrapped.ShouldRebuild(o.wrapped)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoPickerDiameterRatioNotPositive = "CupertinoPickerDiameterRatioNotPositive"
	tagCupertinoPickerMagnificationNotPositive = "CupertinoPickerMagnificationNotPositive"
	tagCupertinoPickerItemExtentNotPositive    = "CupertinoPickerItemExtentNotPositive"
	tagCupertinoPickerSqueezeNotPositive       = "CupertinoPickerSqueezeNotPositive"
)
