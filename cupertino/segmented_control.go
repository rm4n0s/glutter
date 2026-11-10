package cupertino

import (
	"fmt"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The measures of the segmented control (the constants of cupertino/segmented_control.dart).
const (
	segmentedControlMinHeight = 28.0
	segmentedControlFade      = 165 * millisecond
)

var (
	segmentedControlDisableTextColor = ui.NewColor(0x737A7A7A)
)

// CupertinoSegment is a segment of a CupertinoSegmentedControl or of a CupertinoSlidingSegmentedControl: the value
// that it stands for and the widget that it shows.
//
// Deviation: Flutter takes a Map<T, Widget>, which keeps the order of its entries; Go maps do not, so the segments
// are a slice in the order they are shown (SPEC §12.10).
//
// Flutter: no counterpart (the entries of the children map)
type CupertinoSegment[T comparable] struct {
	// Value is the value that the segment stands for.
	Value T
	// Child is the widget that the segment shows.
	Child widgets.Widget
}

// CupertinoSegmentedControlOpts are the optional parameters of NewCupertinoSegmentedControl.
//
// Flutter: cupertino/segmented_control.dart CupertinoSegmentedControl.new
type CupertinoSegmentedControlOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// GroupValue is the value of the selected segment; none selects nothing.
	GroupValue foundation.Opt[T]
	// UnselectedColor is the background of the segments that are not selected, and the color of the selected one's
	// text; the default is the primary contrasting color of the theme.
	UnselectedColor ui.ColorValue
	// SelectedColor is the background of the selected segment, and the color of the others' text; the default is
	// the primary color of the theme.
	SelectedColor ui.ColorValue
	// BorderColor is the color of the borders; the default is the primary color of the theme.
	BorderColor ui.ColorValue
	// PressedColor is the background of the segment that is pressed; the default is the primary color with an
	// opacity of 0.2.
	PressedColor ui.ColorValue
	// DisabledColor is the background of the segments that are disabled.
	DisabledColor ui.ColorValue
	// DisabledTextColor is the color of the text of the segments that are disabled.
	DisabledTextColor ui.ColorValue
	// Padding is the padding around the control; the default is 16 on both sides.
	Padding painting.EdgeInsetsGeometry
	// DisabledChildren are the values of the segments that do not respond to a tap.
	DisabledChildren []T
}

// CupertinoSegmentedControl is an iOS 7 styled segmented control: a horizontal list of mutually exclusive
// segments.
//
// Flutter: cupertino/segmented_control.dart CupertinoSegmentedControl
type CupertinoSegmentedControl[T comparable] struct {
	widgets.StatefulWidget
	children       []CupertinoSegment[T]
	onValueChanged func(value T)
	o              CupertinoSegmentedControlOpts[T]
}

// NewCupertinoSegmentedControl creates a segmented control from at least two segments. In debug builds it panics
// with the tag CupertinoSegmentedControlTooFewChildren, CupertinoSegmentedControlDuplicateValue or
// CupertinoSegmentedControlGroupValueNotAChild when the arguments are not valid.
//
// Flutter: cupertino/segmented_control.dart CupertinoSegmentedControl.new
func NewCupertinoSegmentedControl[T comparable](onValueChanged func(value T), children []CupertinoSegment[T], opts ...CupertinoSegmentedControlOpts[T]) *CupertinoSegmentedControl[T] {
	o := firstOf(opts)
	if foundation.DebugMode {
		seen := map[T]bool{}
		for _, c := range children {
			if seen[c.Value] {
				panic(errors.New(tagCupertinoSegmentedControlDuplicateValue, "two segments of a CupertinoSegmentedControl have the same value", "value", fmt.Sprint(c.Value)))
			}
			seen[c.Value] = true
		}
		switch {
		case len(children) < 2:
			panic(errors.New(tagCupertinoSegmentedControlTooFewChildren, "a CupertinoSegmentedControl needs at least two segments", "count", len(children)))
		case o.GroupValue.IsSome() && !seen[o.GroupValue.Value()]:
			panic(errors.New(tagCupertinoSegmentedControlGroupValueNotAChild, "The groupValue must be either none or one of the values of the segments.", "groupValue", fmt.Sprint(o.GroupValue.Value())))
		}
	}
	return &CupertinoSegmentedControl[T]{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), children, onValueChanged, o}
}

// Children are the segments, in the order they are shown.
//
// Flutter: cupertino/segmented_control.dart CupertinoSegmentedControl.children
func (w *CupertinoSegmentedControl[T]) Children() []CupertinoSegment[T] { return w.children }

// GroupValue is the value of the selected segment.
//
// Flutter: cupertino/segmented_control.dart CupertinoSegmentedControl.groupValue
func (w *CupertinoSegmentedControl[T]) GroupValue() foundation.Opt[T] { return w.o.GroupValue }

// OnValueChanged is called when the user selects a segment.
//
// Flutter: cupertino/segmented_control.dart CupertinoSegmentedControl.onValueChanged
func (w *CupertinoSegmentedControl[T]) OnValueChanged() func(value T) { return w.onValueChanged }

// CreateState creates the state.
//
// Flutter: cupertino/segmented_control.dart CupertinoSegmentedControl.createState
func (w *CupertinoSegmentedControl[T]) CreateState() widgets.State {
	return &segmentedControlState[T]{segmentKeys: map[T]*widgets.GlobalKey[*segmentButtonState[T]]{}}
}

func (w *CupertinoSegmentedControl[T]) disabled(value T) bool {
	for _, d := range w.o.DisabledChildren {
		if d == value {
			return true
		}
	}
	return false
}

// segmentButton is a segment: a radio client with a focus node, around the widget that the segment shows.
//
// Flutter: cupertino/segmented_control.dart _SegmentButton
type segmentButton[T comparable] struct {
	widgets.StatefulWidget
	value   T
	child   widgets.Widget
	enabled bool
}

func (w *segmentButton[T]) CreateState() widgets.State { return &segmentButtonState[T]{} }

type segmentButtonState[T comparable] struct {
	widgets.StateOf[*segmentButton[T]]
	widgets.RadioClientRegistration[T]
	focusNode *widgets.FocusNode
}

func (s *segmentButtonState[T]) InitState() {
	s.focusNode = widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: fmt.Sprintf("CupertinoSegmentedControl<%T>[%v]", s.Widget().value, s.Widget().value)})
}

func (s *segmentButtonState[T]) updateRegistry() {
	if !s.Widget().enabled {
		s.SetRegistry(s, nil)
		return
	}
	r, _ := widgets.RadioGroupMaybeOf[T](s.Context())
	s.SetRegistry(s, r)
}

func (s *segmentButtonState[T]) DidChangeDependencies() { s.updateRegistry() }

func (s *segmentButtonState[T]) DidUpdateWidget(old *segmentButton[T]) {
	if old.enabled != s.Widget().enabled {
		s.updateRegistry()
	}
}

func (s *segmentButtonState[T]) Dispose() {
	s.SetRegistry(s, nil)
	s.focusNode.Dispose()
}

func (s *segmentButtonState[T]) RadioValue() T                 { return s.Widget().value }
func (s *segmentButtonState[T]) FocusNode() *widgets.FocusNode { return s.focusNode }
func (s *segmentButtonState[T]) Tristate() bool                { return false }
func (s *segmentButtonState[T]) Enabled() bool                 { return s.Widget().enabled }

func (s *segmentButtonState[T]) requestFocus() {
	if s.Widget().enabled {
		s.focusNode.RequestFocus()
	}
}

func (s *segmentButtonState[T]) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewFocus(s.Widget().child, widgets.FocusOpts{
		FocusNode:       s.focusNode,
		CanRequestFocus: foundation.Some(s.Widget().enabled),
		OnKeyEvent: func(*widgets.FocusNode, services.KeyEvent) widgets.KeyEventResult {
			return widgets.KeyEventResultIgnored
		},
	})
}

// colorTween is a tween between two colors.
type colorTween struct{ begin, end ui.Color }

func (t colorTween) evaluate(c *animation.AnimationController) ui.Color {
	return lerpColor(t.begin, t.end, c.Value())
}

type segmentedControlState[T comparable] struct {
	widgets.StateOf[*CupertinoSegmentedControl[T]]
	widgets.TickerProvider
	pressedKey             foundation.Opt[T]
	selectionControllers   []*animation.AnimationController
	childTweens            []colorTween
	segmentKeys            map[T]*widgets.GlobalKey[*segmentButtonState[T]]
	forwardBackgroundTween colorTween
	reverseBackgroundTween colorTween
	textColorTween         colorTween

	selectedColor, unselectedColor, borderColor, pressedColor ui.Color
	selectedDisabledColor, unselectedDisabledColor            ui.Color
	disabledTextColor                                         ui.Color
	colorsSet                                                 bool
}

func (s *segmentedControlState[T]) createAnimationController() *animation.AnimationController {
	c := animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(segmentedControlFade)})
	c.AddListener(func() { s.SetState(func() {}) })
	return c
}

func (s *segmentedControlState[T]) updateColors() bool {
	w := s.Widget()
	bc := s.Context()
	changed := false
	pick := func(v ui.ColorValue, def ui.Color) ui.Color {
		if v != nil {
			return v.EffectiveColor()
		}
		return def
	}
	theme := CupertinoThemeOf(bc)
	set := func(dst *ui.Color, v ui.Color) {
		if !s.colorsSet || *dst != v {
			changed = true
			*dst = v
		}
	}
	set(&s.disabledTextColor, pick(w.o.DisabledTextColor, segmentedControlDisableTextColor))
	selected := pick(w.o.SelectedColor, theme.PrimaryColor().EffectiveColor())
	set(&s.selectedColor, selected)
	unselected := pick(w.o.UnselectedColor, theme.PrimaryContrastingColor().EffectiveColor())
	set(&s.unselectedColor, unselected)
	selectedDisabled := pick(w.o.DisabledColor, selected.WithOpacity(0.5))
	unselectedDisabled := pick(w.o.DisabledColor, unselected)
	if !s.colorsSet || s.selectedDisabledColor != selectedDisabled || s.unselectedDisabledColor != unselectedDisabled {
		changed = true
		s.selectedDisabledColor, s.unselectedDisabledColor = selectedDisabled, unselectedDisabled
	}
	set(&s.borderColor, pick(w.o.BorderColor, theme.PrimaryColor().EffectiveColor()))
	set(&s.pressedColor, pick(w.o.PressedColor, theme.PrimaryColor().EffectiveColor().WithOpacity(0.2)))
	s.colorsSet = true
	s.forwardBackgroundTween = colorTween{s.pressedColor, s.selectedColor}
	s.reverseBackgroundTween = colorTween{s.unselectedColor, s.selectedColor}
	s.textColorTween = colorTween{s.selectedColor, s.unselectedColor}
	return changed
}

func (s *segmentedControlState[T]) updateAnimationControllers() {
	for _, c := range s.selectionControllers {
		c.Dispose()
	}
	s.selectionControllers = s.selectionControllers[:0]
	s.childTweens = s.childTweens[:0]
	w := s.Widget()
	for _, c := range w.children {
		controller := s.createAnimationController()
		if w.o.GroupValue == foundation.Some(c.Value) {
			s.childTweens = append(s.childTweens, s.reverseBackgroundTween)
			controller.SetValue(1.0)
		} else {
			s.childTweens = append(s.childTweens, s.forwardBackgroundTween)
		}
		s.selectionControllers = append(s.selectionControllers, controller)
	}
}

func (s *segmentedControlState[T]) DidChangeDependencies() {
	if s.updateColors() {
		s.updateAnimationControllers()
	}
}

func (s *segmentedControlState[T]) DidUpdateWidget(old *CupertinoSegmentedControl[T]) {
	w := s.Widget()
	if s.updateColors() || len(old.children) != len(w.children) {
		s.updateAnimationControllers()
	}
	if old.o.GroupValue != w.o.GroupValue {
		for i, c := range w.children {
			if w.o.GroupValue == foundation.Some(c.Value) {
				s.childTweens[i] = s.forwardBackgroundTween
				s.selectionControllers[i].Forward()
			} else {
				s.childTweens[i] = s.reverseBackgroundTween
				s.selectionControllers[i].Reverse()
			}
		}
	}
}

func (s *segmentedControlState[T]) Dispose() {
	for _, c := range s.selectionControllers {
		c.Dispose()
	}
}

func (s *segmentedControlState[T]) onTapDown(key T) {
	if s.pressedKey.IsNone() && s.Widget().o.GroupValue != foundation.Some(key) {
		s.SetState(func() { s.pressedKey = foundation.Some(key) })
	}
}

func (s *segmentedControlState[T]) onTapCancel() {
	s.SetState(func() { s.pressedKey = foundation.None[T]() })
}

func (s *segmentedControlState[T]) onTap(key T) {
	if s.pressedKey != foundation.Some(key) {
		return
	}
	w := s.Widget()
	if !w.disabled(key) {
		if k := s.segmentKeys[key]; k != nil {
			if st, ok := k.CurrentState(); ok {
				st.requestFocus()
			}
		}
		if w.o.GroupValue != foundation.Some(key) {
			w.onValueChanged(key)
		}
	}
	s.SetState(func() { s.pressedKey = foundation.None[T]() })
}

func (s *segmentedControlState[T]) textColor(index int, key T) ui.Color {
	w := s.Widget()
	switch {
	case w.disabled(key):
		return s.disabledTextColor
	case s.selectionControllers[index].IsAnimating():
		return s.textColorTween.evaluate(s.selectionControllers[index])
	case w.o.GroupValue == foundation.Some(key):
		return s.unselectedColor
	}
	return s.selectedColor
}

func (s *segmentedControlState[T]) backgroundColor(index int, key T) ui.Color {
	w := s.Widget()
	switch {
	case w.disabled(key):
		if w.o.GroupValue == foundation.Some(key) {
			return s.selectedDisabledColor
		}
		return s.unselectedDisabledColor
	case s.selectionControllers[index].IsAnimating():
		return s.childTweens[index].evaluate(s.selectionControllers[index])
	case w.o.GroupValue == foundation.Some(key):
		return s.selectedColor
	case s.pressedKey == foundation.Some(key):
		return s.pressedColor
	}
	return s.unselectedColor
}

func (s *segmentedControlState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var gestureChildren []widgets.Widget
	var backgroundColors []ui.Color
	selectedIndex, pressedIndex := foundation.None[int](), foundation.None[int]()
	for index, c := range w.children {
		currentKey := c.Value
		if w.o.GroupValue == foundation.Some(currentKey) {
			selectedIndex = foundation.Some(index)
		}
		if s.pressedKey == foundation.Some(currentKey) {
			pressedIndex = foundation.Some(index)
		}
		textColor := s.textColor(index, currentKey)
		textStyle := widgets.DefaultTextStyleOf(bc).Style().CopyWith(painting.TextStyleCopy{Color: foundation.Some(textColor)})
		iconTheme := widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(textColor)})
		isEnabled := !w.disabled(currentKey)
		segmentKey := s.segmentKeys[currentKey]
		if segmentKey == nil {
			segmentKey = widgets.NewGlobalKey[*segmentButtonState[T]]()
			s.segmentKeys[currentKey] = segmentKey
		}
		var onTapDown func(details gestures.TapDownDetails)
		var onTapCancel func()
		if isEnabled {
			onTapDown = func(gestures.TapDownDetails) { s.onTapDown(currentKey) }
			onTapCancel = s.onTapCancel
		}
		var cursor services.MouseCursor = services.MouseCursorDefer
		if foundation.IsWeb {
			cursor = services.SystemMouseCursors.Click
		}
		child := widgets.Widget(widgets.NewCenter(c.Child))
		child = &segmentButton[T]{
			widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: segmentKey}), currentKey,
			widgets.NewMouseRegion(widgets.NewGestureDetector(
				widgets.NewIconTheme(iconTheme, widgets.NewDefaultTextStyle(textStyle, widgets.NewSemantics(child, widgets.SemanticsOpts{
					Button: foundation.Some(true), InMutuallyExclusiveGroup: foundation.Some(true),
					Selected: foundation.Some(w.o.GroupValue == foundation.Some(currentKey)),
				}))),
				widgets.GestureDetectorOpts{
					Behavior:    rendering.HitTestBehaviorOpaque,
					OnTapDown:   onTapDown,
					OnTapCancel: onTapCancel,
					OnTap: func() {
						if isEnabled {
							if st, ok := segmentKey.CurrentState(); ok {
								st.requestFocus()
							}
						}
						s.onTap(currentKey)
					},
				}), widgets.MouseRegionOpts{Cursor: cursor}),
			isEnabled,
		}
		backgroundColors = append(backgroundColors, s.backgroundColor(index, currentKey))
		gestureChildren = append(gestureChildren, child)
	}
	box := &segmentedControlRenderWidget{
		MultiChildRenderObjectWidget: widgets.NewMultiChildRenderObjectWidgetBase(gestureChildren),
		selectedIndex:                selectedIndex, pressedIndex: pressedIndex, backgroundColors: backgroundColors, borderColor: s.borderColor,
	}
	var padding painting.EdgeInsetsGeometry = painting.NewEdgeInsetsSymmetric(0, 16.0)
	if w.o.Padding != nil {
		padding = w.o.Padding
	}
	return widgets.NewActions(widgets.ActionMap{widgets.IntentTypeOf[widgets.VoidCallbackIntent](): widgets.NewVoidCallbackAction()},
		widgets.NewRadioGroup(func(value foundation.Opt[T]) {
			if v, ok := value.Get(); ok && !w.disabled(v) {
				w.onValueChanged(v)
			}
		}, widgets.NewPadding(padding, widgets.NewUnconstrainedBox(box, widgets.UnconstrainedBoxOpts{ConstrainedAxis: painting.AxisHorizontal})),
			widgets.RadioGroupOpts[T]{GroupValue: w.o.GroupValue}))
}

// segmentedControlRenderWidget makes the render object that lays the segments out and paints their backgrounds.
//
// Flutter: cupertino/segmented_control.dart _SegmentedControlRenderWidget
type segmentedControlRenderWidget struct {
	widgets.MultiChildRenderObjectWidget
	selectedIndex, pressedIndex foundation.Opt[int]
	backgroundColors            []ui.Color
	borderColor                 ui.Color
}

func (w *segmentedControlRenderWidget) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	r := &renderSegmentedControl{textDirection: widgets.DirectionalityOf(bc), selectedIndex: w.selectedIndex, pressedIndex: w.pressedIndex,
		backgroundColors: w.backgroundColors, borderColor: w.borderColor}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitContainerRenderObject(&r.ContainerRenderObject, r)
	return r
}

func (w *segmentedControlRenderWidget) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderSegmentedControl)
	r.setTextDirection(widgets.DirectionalityOf(bc))
	r.setSelectedIndex(w.selectedIndex)
	r.setPressedIndex(w.pressedIndex)
	r.setBackgroundColors(w.backgroundColors)
	r.setBorderColor(w.borderColor)
}

// segmentedControlParentData is the parent data of a segment: the rounded shape that surrounds it.
//
// Flutter: cupertino/segmented_control.dart _SegmentedControlContainerBoxParentData
type segmentedControlParentData struct {
	rendering.ContainerBoxParentData[rendering.Box]
	surroundingRect ui.RSuperellipse
}

func segmentedControlParent(child rendering.Box) *segmentedControlParentData {
	return child.ParentData().(*segmentedControlParentData)
}

// renderSegmentedControl lays the segments out in a line with the same size, and paints their backgrounds and
// borders.
//
// Flutter: cupertino/segmented_control.dart _RenderSegmentedControl
type renderSegmentedControl struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
	selectedIndex, pressedIndex foundation.Opt[int]
	textDirection               ui.TextDirection
	backgroundColors            []ui.Color
	borderColor                 ui.Color
}

func (r *renderSegmentedControl) setSelectedIndex(v foundation.Opt[int]) {
	if r.selectedIndex != v {
		r.selectedIndex = v
		r.MarkNeedsPaint()
	}
}

func (r *renderSegmentedControl) setPressedIndex(v foundation.Opt[int]) {
	if r.pressedIndex != v {
		r.pressedIndex = v
		r.MarkNeedsPaint()
	}
}

func (r *renderSegmentedControl) setTextDirection(v ui.TextDirection) {
	if r.textDirection != v {
		r.textDirection = v
		r.MarkNeedsLayout()
	}
}

func (r *renderSegmentedControl) setBackgroundColors(v []ui.Color) {
	r.backgroundColors = v
	r.MarkNeedsPaint()
}

func (r *renderSegmentedControl) setBorderColor(v ui.Color) {
	if r.borderColor != v {
		r.borderColor = v
		r.MarkNeedsPaint()
	}
}

func (r *renderSegmentedControl) eachChild(f func(child rendering.Box)) {
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		f(child)
	}
}

func (r *renderSegmentedControl) ComputeMinIntrinsicWidth(height float64) float64 {
	minWidth := 0.0
	r.eachChild(func(c rendering.Box) { minWidth = math.Max(minWidth, c.GetMinIntrinsicWidth(height)) })
	return minWidth * float64(r.ChildCount())
}

func (r *renderSegmentedControl) ComputeMaxIntrinsicWidth(height float64) float64 {
	maxWidth := 0.0
	r.eachChild(func(c rendering.Box) { maxWidth = math.Max(maxWidth, c.GetMaxIntrinsicWidth(height)) })
	return maxWidth * float64(r.ChildCount())
}

func (r *renderSegmentedControl) ComputeMinIntrinsicHeight(width float64) float64 {
	minHeight := 0.0
	r.eachChild(func(c rendering.Box) { minHeight = math.Max(minHeight, c.GetMinIntrinsicHeight(width)) })
	return minHeight
}

func (r *renderSegmentedControl) ComputeMaxIntrinsicHeight(width float64) float64 {
	maxHeight := 0.0
	r.eachChild(func(c rendering.Box) { maxHeight = math.Max(maxHeight, c.GetMaxIntrinsicHeight(width)) })
	return maxHeight
}

func (r *renderSegmentedControl) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	return rendering.DefaultComputeDistanceToHighestActualBaseline(&r.ContainerRenderObject, baseline)
}

func (r *renderSegmentedControl) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*segmentedControlParentData); !ok {
		child.AsRenderObject().SetParentData(&segmentedControlParentData{})
	}
}

func (r *renderSegmentedControl) layoutRects(next func(rendering.Box) rendering.Box, leftChild, rightChild rendering.Box) {
	start := 0.0
	radius := ui.NewRadiusCircular(3.0)
	for child := leftChild; child != nil; child = next(child) {
		pd := segmentedControlParent(child)
		pd.Offset = ui.Offset{Dx: start, Dy: 0.0}
		size := child.Size()
		childRect := ui.NewRectFromLTWH(start, 0.0, size.Width, size.Height)
		switch child {
		case leftChild:
			pd.surroundingRect = ui.NewRSuperellipseFromRectAndCorners(childRect, ui.Corners{TopLeft: radius, BottomLeft: radius})
		case rightChild:
			pd.surroundingRect = ui.NewRSuperellipseFromRectAndCorners(childRect, ui.Corners{TopRight: radius, BottomRight: radius})
		default:
			pd.surroundingRect = ui.NewRSuperellipseFromRectAndCorners(childRect)
		}
		start += size.Width
	}
}

func (r *renderSegmentedControl) calculateChildSize(constraints rendering.BoxConstraints) ui.Size {
	maxHeight := segmentedControlMinHeight
	childWidth := constraints.MinWidth / float64(r.ChildCount())
	r.eachChild(func(c rendering.Box) { childWidth = math.Max(childWidth, c.GetMaxIntrinsicWidth(math.Inf(1))) })
	childWidth = math.Min(childWidth, constraints.MaxWidth/float64(r.ChildCount()))
	r.eachChild(func(c rendering.Box) { maxHeight = math.Max(maxHeight, c.GetMaxIntrinsicHeight(childWidth)) })
	return ui.Size{Width: childWidth, Height: maxHeight}
}

func (r *renderSegmentedControl) computeOverallSizeFromChildSize(constraints rendering.BoxConstraints, childSize ui.Size) ui.Size {
	return constraints.Constrain(ui.Size{Width: childSize.Width * float64(r.ChildCount()), Height: childSize.Height})
}

func (r *renderSegmentedControl) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	childConstraints := rendering.NewBoxConstraintsTight(r.calculateChildSize(constraints))
	result := foundation.None[float64]()
	r.eachChild(func(c rendering.Box) {
		if v, ok := c.GetDryBaseline(childConstraints, baseline).Get(); ok {
			if cur, has := result.Get(); !has || v < cur {
				result = foundation.Some(v)
			}
		}
	})
	return result
}

func (r *renderSegmentedControl) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	return r.computeOverallSizeFromChildSize(constraints, r.calculateChildSize(constraints))
}

func (r *renderSegmentedControl) PerformLayout() {
	constraints := r.BoxConstraints()
	childSize := r.calculateChildSize(constraints)
	childConstraints := rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(childSize.Width), Height: foundation.Some(childSize.Height)})
	r.eachChild(func(c rendering.Box) { c.Layout(childConstraints, rendering.LayoutOpts{ParentUsesSize: true}) })
	if r.textDirection == ui.TextDirectionRTL {
		r.layoutRects(r.ChildBefore, r.LastChild(), r.FirstChild())
	} else {
		r.layoutRects(r.ChildAfter, r.FirstChild(), r.LastChild())
	}
	r.SetSize(r.computeOverallSizeFromChildSize(constraints, childSize))
}

func (r *renderSegmentedControl) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	index := 0
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		r.paintChild(context, offset, child, index)
		index++
	}
}

func (r *renderSegmentedControl) paintChild(context *rendering.PaintingContext, offset ui.Offset, child rendering.Box, childIndex int) {
	pd := segmentedControlParent(child)
	canvas := context.Canvas()
	fill := ui.NewPaint()
	fill.SetColor(r.backgroundColors[childIndex])
	fill.SetStyle(ui.PaintingStyleFill)
	canvas.DrawRSuperellipse(pd.surroundingRect.Shift(offset), fill)
	border := ui.NewPaint()
	border.SetColor(r.borderColor)
	border.SetStrokeWidth(1.0)
	border.SetStyle(ui.PaintingStyleStroke)
	canvas.DrawRSuperellipse(pd.surroundingRect.Shift(offset), border)
	context.PaintChild(child, pd.Offset.Add(offset))
}

func (r *renderSegmentedControl) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	for child := r.LastChild(); child != nil; child = segmentedControlParent(child).PreviousSibling {
		pd := segmentedControlParent(child)
		if pd.surroundingRect.OuterRect().Contains(position) {
			offset := pd.Offset
			return result.AddWithPaintOffset(&offset, position, func(result *rendering.BoxHitTestResult, local ui.Offset) bool {
				return child.HitTest(result, local)
			})
		}
	}
	return false
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoSegmentedControlTooFewChildren      = "CupertinoSegmentedControlTooFewChildren"
	tagCupertinoSegmentedControlDuplicateValue      = "CupertinoSegmentedControlDuplicateValue"
	tagCupertinoSegmentedControlGroupValueNotAChild = "CupertinoSegmentedControlGroupValueNotAChild"
)
