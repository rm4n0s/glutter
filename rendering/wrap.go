package rendering

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// WrapAlignment is how the children within a run, or the runs themselves, are
// placed along an axis. The zero value is WrapAlignmentUnspecified: start.
//
// Flutter: rendering/wrap.dart WrapAlignment
type WrapAlignment int

// The values of WrapAlignment follow Flutter's declaration order.
const (
	// WrapAlignmentUnspecified is the zero value: the default (start).
	//
	// Flutter: no counterpart
	WrapAlignmentUnspecified WrapAlignment = iota
	// WrapAlignmentStart places the items at the start of the axis.
	//
	// Flutter: rendering/wrap.dart WrapAlignment.start
	WrapAlignmentStart
	// WrapAlignmentEnd places the items at the end of the axis.
	//
	// Flutter: rendering/wrap.dart WrapAlignment.end
	WrapAlignmentEnd
	// WrapAlignmentCenter places the items in the middle of the axis.
	//
	// Flutter: rendering/wrap.dart WrapAlignment.center
	WrapAlignmentCenter
	// WrapAlignmentSpaceBetween places the free space evenly between the items.
	//
	// Flutter: rendering/wrap.dart WrapAlignment.spaceBetween
	WrapAlignmentSpaceBetween
	// WrapAlignmentSpaceAround places the free space evenly between the items as
	// well as half of that space before and after the first and last item.
	//
	// Flutter: rendering/wrap.dart WrapAlignment.spaceAround
	WrapAlignmentSpaceAround
	// WrapAlignmentSpaceEvenly places the free space evenly between the items as
	// well as before and after the first and last item.
	//
	// Flutter: rendering/wrap.dart WrapAlignment.spaceEvenly
	WrapAlignmentSpaceEvenly
)

// Name returns the Dart enum name, such as "spaceBetween".
//
// Flutter: rendering/wrap.dart WrapAlignment.name
func (a WrapAlignment) Name() string {
	return enumName(int(a), "start", "end", "center", "spaceBetween", "spaceAround", "spaceEvenly")
}

// String returns the name Dart prints, such as "WrapAlignment.start".
//
// Flutter: rendering/wrap.dart WrapAlignment.toString
func (a WrapAlignment) String() string { return "WrapAlignment." + a.Name() }

// distributeSpace returns the space before the first item and between the items.
//
// Flutter: rendering/wrap.dart WrapAlignment._distributeSpace
func (a WrapAlignment) distributeSpace(freeSpace, itemSpacing float64, itemCount int, flipped bool) (leading, between float64) {
	switch a {
	case WrapAlignmentEnd:
		return WrapAlignmentStart.distributeSpace(freeSpace, itemSpacing, itemCount, !flipped)
	case WrapAlignmentSpaceBetween:
		if itemCount < 2 {
			return WrapAlignmentStart.distributeSpace(freeSpace, itemSpacing, itemCount, flipped)
		}
		return 0, freeSpace/float64(itemCount-1) + itemSpacing
	case WrapAlignmentCenter:
		return freeSpace / 2, itemSpacing
	case WrapAlignmentSpaceAround:
		return freeSpace / float64(itemCount) / 2, freeSpace/float64(itemCount) + itemSpacing
	case WrapAlignmentSpaceEvenly:
		return freeSpace / float64(itemCount+1), freeSpace/float64(itemCount+1) + itemSpacing
	}
	if flipped {
		return freeSpace, itemSpacing
	}
	return 0, itemSpacing
}

// WrapCrossAlignment is how the children within a run are placed relative to
// each other in the cross axis. The zero value is WrapCrossAlignmentUnspecified:
// start.
//
// Flutter: rendering/wrap.dart WrapCrossAlignment
type WrapCrossAlignment int

// The values of WrapCrossAlignment follow Flutter's declaration order.
const (
	// WrapCrossAlignmentUnspecified is the zero value: the default (start).
	//
	// Flutter: no counterpart
	WrapCrossAlignmentUnspecified WrapCrossAlignment = iota
	// WrapCrossAlignmentStart places the children at the start of the run in
	// the cross axis.
	//
	// Flutter: rendering/wrap.dart WrapCrossAlignment.start
	WrapCrossAlignmentStart
	// WrapCrossAlignmentEnd places the children at the end of the run in the
	// cross axis.
	//
	// Flutter: rendering/wrap.dart WrapCrossAlignment.end
	WrapCrossAlignmentEnd
	// WrapCrossAlignmentCenter places the children in the middle of the run in
	// the cross axis.
	//
	// Flutter: rendering/wrap.dart WrapCrossAlignment.center
	WrapCrossAlignmentCenter
)

// Name returns the Dart enum name, such as "center".
//
// Flutter: rendering/wrap.dart WrapCrossAlignment.name
func (a WrapCrossAlignment) Name() string { return enumName(int(a), "start", "end", "center") }

// String returns the name Dart prints, such as "WrapCrossAlignment.start".
//
// Flutter: rendering/wrap.dart WrapCrossAlignment.toString
func (a WrapCrossAlignment) String() string { return "WrapCrossAlignment." + a.Name() }

func (a WrapCrossAlignment) flipped() WrapCrossAlignment {
	switch a {
	case WrapCrossAlignmentEnd:
		return WrapCrossAlignmentStart
	case WrapCrossAlignmentCenter:
		return WrapCrossAlignmentCenter
	}
	return WrapCrossAlignmentEnd
}

func (a WrapCrossAlignment) fraction() float64 {
	switch a {
	case WrapCrossAlignmentEnd:
		return 1
	case WrapCrossAlignmentCenter:
		return 0.5
	}
	return 0
}

// wrapRunMetrics is the size and the first child of a run.
//
// Flutter: rendering/wrap.dart _RunMetrics
type wrapRunMetrics struct {
	axisSize     axisSize
	childCount   int
	leadingChild Box
}

// tryAddingNewChild adds the child to the run, or returns a new run that starts
// with it if it does not fit.
func (m *wrapRunMetrics) tryAddingNewChild(child Box, childSize axisSize, flipMainAxis bool, spacing, maxMainExtent float64) *wrapRunMetrics {
	if m.axisSize.main+childSize.main+spacing-maxMainExtent > foundation.PrecisionErrorTolerance {
		return &wrapRunMetrics{axisSize: childSize, childCount: 1, leadingChild: child}
	}
	m.axisSize = m.axisSize.add(childSize.add(axisSize{spacing, 0}))
	m.childCount++
	if flipMainAxis {
		m.leadingChild = child
	}
	return nil
}

// WrapParentData is the parent data of the children of a RenderWrap.
//
// Flutter: rendering/wrap.dart WrapParentData
type WrapParentData struct {
	ContainerBoxParentData[Box]
}

// RenderWrap implements the wrap layout algorithm: the children are placed in
// runs along the main axis, and a new run starts when one does not fit.
//
// Flutter: rendering/wrap.dart RenderWrap
type RenderWrap struct {
	RenderBox
	ContainerRenderObject[Box]
	direction          painting.Axis
	alignment          WrapAlignment
	spacing            float64
	runAlignment       WrapAlignment
	runSpacing         float64
	crossAxisAlignment WrapCrossAlignment
	textDirection      ui.TextDirection
	verticalDirection  painting.VerticalDirection
	clipBehavior       ui.Clip
	hasVisualOverflow  bool
	clipRectLayer      *LayerHandle[*ClipRectLayer]
}

// RenderWrapOpts are the optional parameters of NewRenderWrap.
//
// Flutter: rendering/wrap.dart RenderWrap.new
type RenderWrapOpts struct {
	// Direction defaults to painting.AxisHorizontal.
	Direction painting.Axis
	// Alignment defaults to WrapAlignmentStart.
	Alignment WrapAlignment
	// Spacing is the space between the children in a run.
	Spacing float64
	// RunAlignment defaults to WrapAlignmentStart.
	RunAlignment WrapAlignment
	// RunSpacing is the space between the runs.
	RunSpacing float64
	// CrossAxisAlignment defaults to WrapCrossAlignmentStart.
	CrossAxisAlignment WrapCrossAlignment
	// TextDirection is the order of a horizontal wrap; unspecified is null.
	TextDirection ui.TextDirection
	// VerticalDirection defaults to painting.VerticalDirectionDown.
	VerticalDirection painting.VerticalDirection
	// ClipBehavior defaults to ui.ClipNone.
	ClipBehavior ui.Clip
}

// NewRenderWrap creates a wrap with the children (nil for none).
//
// Flutter: rendering/wrap.dart RenderWrap.new
func NewRenderWrap(children []Box, opts ...RenderWrapOpts) *RenderWrap {
	o := firstOpts(opts)
	if o.Direction == painting.AxisUnspecified {
		o.Direction = painting.AxisHorizontal
	}
	if o.Alignment == WrapAlignmentUnspecified {
		o.Alignment = WrapAlignmentStart
	}
	if o.RunAlignment == WrapAlignmentUnspecified {
		o.RunAlignment = WrapAlignmentStart
	}
	if o.CrossAxisAlignment == WrapCrossAlignmentUnspecified {
		o.CrossAxisAlignment = WrapCrossAlignmentStart
	}
	if o.VerticalDirection == painting.VerticalDirectionUnspecified {
		o.VerticalDirection = painting.VerticalDirectionDown
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	r := &RenderWrap{direction: o.Direction, alignment: o.Alignment, spacing: o.Spacing, runAlignment: o.RunAlignment,
		runSpacing: o.RunSpacing, crossAxisAlignment: o.CrossAxisAlignment, textDirection: o.TextDirection,
		verticalDirection: o.VerticalDirection, clipBehavior: o.ClipBehavior,
		clipRectLayer: NewLayerHandle[*ClipRectLayer](nil)}
	InitRenderBox(&r.RenderBox, r)
	InitContainerRenderObject(&r.ContainerRenderObject, r)
	r.AddAll(children)
	return r
}

// Direction is the direction to use as the main axis.
//
// Flutter: rendering/wrap.dart RenderWrap.direction
func (r *RenderWrap) Direction() painting.Axis { return r.direction }

// SetDirection marks the wrap as needing layout when the value changes.
//
// Flutter: rendering/wrap.dart RenderWrap.direction
func (r *RenderWrap) SetDirection(value painting.Axis) {
	if r.direction != value {
		r.direction = value
		r.MarkNeedsLayout()
	}
}

// Alignment is how the children within a run are placed in the main axis.
//
// Flutter: rendering/wrap.dart RenderWrap.alignment
func (r *RenderWrap) Alignment() WrapAlignment { return r.alignment }

// SetAlignment marks the wrap as needing layout when the value changes.
//
// Flutter: rendering/wrap.dart RenderWrap.alignment
func (r *RenderWrap) SetAlignment(value WrapAlignment) {
	if value == WrapAlignmentUnspecified {
		value = WrapAlignmentStart
	}
	if r.alignment != value {
		r.alignment = value
		r.MarkNeedsLayout()
	}
}

// Spacing is how much space to place between children in a run in the main axis.
//
// Flutter: rendering/wrap.dart RenderWrap.spacing
func (r *RenderWrap) Spacing() float64 { return r.spacing }

// SetSpacing marks the wrap as needing layout when the value changes.
//
// Flutter: rendering/wrap.dart RenderWrap.spacing
func (r *RenderWrap) SetSpacing(value float64) {
	if r.spacing != value {
		r.spacing = value
		r.MarkNeedsLayout()
	}
}

// RunAlignment is how the runs themselves are placed in the cross axis.
//
// Flutter: rendering/wrap.dart RenderWrap.runAlignment
func (r *RenderWrap) RunAlignment() WrapAlignment { return r.runAlignment }

// SetRunAlignment marks the wrap as needing layout when the value changes.
//
// Flutter: rendering/wrap.dart RenderWrap.runAlignment
func (r *RenderWrap) SetRunAlignment(value WrapAlignment) {
	if value == WrapAlignmentUnspecified {
		value = WrapAlignmentStart
	}
	if r.runAlignment != value {
		r.runAlignment = value
		r.MarkNeedsLayout()
	}
}

// RunSpacing is how much space to place between the runs in the cross axis.
//
// Flutter: rendering/wrap.dart RenderWrap.runSpacing
func (r *RenderWrap) RunSpacing() float64 { return r.runSpacing }

// SetRunSpacing marks the wrap as needing layout when the value changes.
//
// Flutter: rendering/wrap.dart RenderWrap.runSpacing
func (r *RenderWrap) SetRunSpacing(value float64) {
	if r.runSpacing != value {
		r.runSpacing = value
		r.MarkNeedsLayout()
	}
}

// CrossAxisAlignment is how the children within a run are aligned relative to
// each other in the cross axis.
//
// Flutter: rendering/wrap.dart RenderWrap.crossAxisAlignment
func (r *RenderWrap) CrossAxisAlignment() WrapCrossAlignment { return r.crossAxisAlignment }

// SetCrossAxisAlignment marks the wrap as needing layout when the value changes.
//
// Flutter: rendering/wrap.dart RenderWrap.crossAxisAlignment
func (r *RenderWrap) SetCrossAxisAlignment(value WrapCrossAlignment) {
	if value == WrapCrossAlignmentUnspecified {
		value = WrapCrossAlignmentStart
	}
	if r.crossAxisAlignment != value {
		r.crossAxisAlignment = value
		r.MarkNeedsLayout()
	}
}

// TextDirection determines the order to lay children out horizontally.
// Unspecified is null.
//
// Flutter: rendering/wrap.dart RenderWrap.textDirection
func (r *RenderWrap) TextDirection() ui.TextDirection { return r.textDirection }

// SetTextDirection marks the wrap as needing layout when the value changes.
//
// Flutter: rendering/wrap.dart RenderWrap.textDirection
func (r *RenderWrap) SetTextDirection(value ui.TextDirection) {
	if r.textDirection != value {
		r.textDirection = value
		r.MarkNeedsLayout()
	}
}

// VerticalDirection determines the order to lay children out vertically.
//
// Flutter: rendering/wrap.dart RenderWrap.verticalDirection
func (r *RenderWrap) VerticalDirection() painting.VerticalDirection { return r.verticalDirection }

// SetVerticalDirection marks the wrap as needing layout when the value changes.
//
// Flutter: rendering/wrap.dart RenderWrap.verticalDirection
func (r *RenderWrap) SetVerticalDirection(value painting.VerticalDirection) {
	if value == painting.VerticalDirectionUnspecified {
		value = painting.VerticalDirectionDown
	}
	if r.verticalDirection != value {
		r.verticalDirection = value
		r.MarkNeedsLayout()
	}
}

// ClipBehavior is how the content that overflows is clipped.
//
// Flutter: rendering/wrap.dart RenderWrap.clipBehavior
func (r *RenderWrap) ClipBehavior() ui.Clip { return r.clipBehavior }

// SetClipBehavior marks the wrap as needing paint when the value changes.
//
// Flutter: rendering/wrap.dart RenderWrap.clipBehavior
func (r *RenderWrap) SetClipBehavior(value ui.Clip) {
	if value == ui.ClipUnspecified {
		value = ui.ClipNone
	}
	if value != r.clipBehavior {
		r.clipBehavior = value
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

// debugCheckDirections panics if a direction that the layout needs is missing.
//
// Flutter: rendering/wrap.dart RenderWrap._debugHasNecessaryDirections
func (r *RenderWrap) debugCheckDirections() {
	if r.textDirection != ui.TextDirectionUnspecified {
		return
	}
	name := foundation.RuntimeTypeName(r)
	missing := func(what string) {
		panic(errors.New("WrapDirectionMissing", what, "object", foundation.DescribeIdentity(r)))
	}
	horizontal := r.direction == painting.AxisHorizontal
	first, last := r.FirstChild(), r.LastChild()
	if !isNil(first) && first != last && horizontal {
		missing("Horizontal " + name + " with multiple children has a null textDirection, so the layout order is undefined.")
	}
	if (r.alignment == WrapAlignmentStart || r.alignment == WrapAlignmentEnd) && horizontal {
		missing("Horizontal " + name + " with alignment " + r.alignment.String() + " has a null textDirection, so the alignment cannot be resolved.")
	}
	if (r.runAlignment == WrapAlignmentStart || r.runAlignment == WrapAlignmentEnd) && !horizontal {
		missing("Vertical " + name + " with runAlignment " + r.runAlignment.String() + " has a null textDirection, so the alignment cannot be resolved.")
	}
	if (r.crossAxisAlignment == WrapCrossAlignmentStart || r.crossAxisAlignment == WrapCrossAlignmentEnd) && !horizontal {
		missing("Vertical " + name + " with crossAxisAlignment " + r.crossAxisAlignment.String() + " has a null textDirection, so the alignment cannot be resolved.")
	}
}

// SetupParentData gives the child a WrapParentData if it has none.
//
// Flutter: rendering/wrap.dart RenderWrap.setupParentData
func (r *RenderWrap) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*WrapParentData); !ok {
		child.AsRenderObject().SetParentData(&WrapParentData{})
	}
}

func wrapParentData(child Box) *WrapParentData { return child.ParentData().(*WrapParentData) }

func (r *RenderWrap) intrinsicAcross(extent float64, sum bool, get func(Box, float64) float64) float64 {
	v := 0.0
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		if sum {
			v += get(child, math.Inf(1))
		} else {
			v = math.Max(v, get(child, math.Inf(1)))
		}
	}
	return v
}

// ComputeMinIntrinsicWidth is the widest child, or the width of the dry layout
// of a vertical wrap.
//
// Flutter: rendering/wrap.dart RenderWrap.computeMinIntrinsicWidth
func (r *RenderWrap) ComputeMinIntrinsicWidth(height float64) float64 {
	if r.direction == painting.AxisHorizontal {
		return r.intrinsicAcross(height, false, func(c Box, e float64) float64 { return c.GetMinIntrinsicWidth(e) })
	}
	return r.GetDryLayout(BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: height}).Width
}

// ComputeMaxIntrinsicWidth is the sum of the children's widths, or the width of
// the dry layout of a vertical wrap.
//
// Flutter: rendering/wrap.dart RenderWrap.computeMaxIntrinsicWidth
func (r *RenderWrap) ComputeMaxIntrinsicWidth(height float64) float64 {
	if r.direction == painting.AxisHorizontal {
		return r.intrinsicAcross(height, true, func(c Box, e float64) float64 { return c.GetMaxIntrinsicWidth(e) })
	}
	return r.GetDryLayout(BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: height}).Width
}

// ComputeMinIntrinsicHeight is the tallest child of a vertical wrap, or the
// height of the dry layout of a horizontal one.
//
// Flutter: rendering/wrap.dart RenderWrap.computeMinIntrinsicHeight
func (r *RenderWrap) ComputeMinIntrinsicHeight(width float64) float64 {
	if r.direction == painting.AxisHorizontal {
		return r.GetDryLayout(BoxConstraints{MaxWidth: width, MaxHeight: math.Inf(1)}).Height
	}
	return r.intrinsicAcross(width, false, func(c Box, e float64) float64 { return c.GetMinIntrinsicHeight(e) })
}

// ComputeMaxIntrinsicHeight is the sum of the children's heights of a vertical
// wrap, or the height of the dry layout of a horizontal one.
//
// Flutter: rendering/wrap.dart RenderWrap.computeMaxIntrinsicHeight
func (r *RenderWrap) ComputeMaxIntrinsicHeight(width float64) float64 {
	if r.direction == painting.AxisHorizontal {
		return r.GetDryLayout(BoxConstraints{MaxWidth: width, MaxHeight: math.Inf(1)}).Height
	}
	return r.intrinsicAcross(width, true, func(c Box, e float64) float64 { return c.GetMaxIntrinsicHeight(e) })
}

// ComputeDistanceToActualBaseline is the highest baseline among the children.
//
// Flutter: rendering/wrap.dart RenderWrap.computeDistanceToActualBaseline
func (r *RenderWrap) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	return DefaultComputeDistanceToHighestActualBaseline(&r.ContainerRenderObject, baseline)
}

func (r *RenderWrap) mainAxisExtent(s ui.Size) float64 {
	if r.direction == painting.AxisHorizontal {
		return s.Width
	}
	return s.Height
}

func (r *RenderWrap) crossAxisExtent(s ui.Size) float64 {
	if r.direction == painting.AxisHorizontal {
		return s.Height
	}
	return s.Width
}

func (r *RenderWrap) offsetOf(main, cross float64) ui.Offset {
	if r.direction == painting.AxisHorizontal {
		return ui.Offset{Dx: main, Dy: cross}
	}
	return ui.Offset{Dx: cross, Dy: main}
}

// areAxesFlipped reports whether the main and the cross axes run backwards.
//
// Flutter: rendering/wrap.dart RenderWrap._areAxesFlipped
func (r *RenderWrap) areAxesFlipped() (flipMain, flipCross bool) {
	flipHorizontal := r.textDirection == ui.TextDirectionRTL
	flipVertical := r.verticalDirection == painting.VerticalDirectionUp
	if r.direction == painting.AxisHorizontal {
		return flipHorizontal, flipVertical
	}
	return flipVertical, flipHorizontal
}

func (r *RenderWrap) childConstraintsAndLimit(constraints BoxConstraints) (BoxConstraints, float64) {
	if r.direction == painting.AxisHorizontal {
		return BoxConstraints{MaxWidth: constraints.MaxWidth, MaxHeight: math.Inf(1)}, constraints.MaxWidth
	}
	return BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: constraints.MaxHeight}, constraints.MaxHeight
}

// ComputeDryBaseline is the highest baseline among the children for the
// constraints.
//
// Flutter: rendering/wrap.dart RenderWrap.computeDryBaseline
func (r *RenderWrap) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if isNil(r.FirstChild()) {
		return foundation.None[float64]()
	}
	childConstraints, _ := r.childConstraintsAndLimit(constraints)
	childrenAxisSize, runMetrics := r.computeRuns(constraints, ChildLayoutHelper.DryLayoutChild)
	containerAxisSize := childrenAxisSize.applyConstraints(constraints, r.direction)
	baselineOffset := NoBaseline
	r.positionChildren(runMetrics, childrenAxisSize, containerAxisSize,
		func(offset ui.Offset, child Box) {
			baselineOffset = baselineOffset.MinOf(NewBaselineOffset(child.GetDryBaseline(childConstraints, baseline)).Add(offset.Dy))
		},
		func(child Box) ui.Size { return child.GetDryLayout(childConstraints) })
	return baselineOffset.Offset
}

// ComputeDryLayout is the size the constraints would give.
//
// Flutter: rendering/wrap.dart RenderWrap.computeDryLayout
func (r *RenderWrap) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	childConstraints, mainAxisLimit := r.childConstraintsAndLimit(constraints)
	var mainExtent, crossExtent, runMain, runCross float64
	childCount := 0
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		childSize := ChildLayoutHelper.DryLayoutChild(child, childConstraints)
		childMain, childCross := r.mainAxisExtent(childSize), r.crossAxisExtent(childSize)
		if childCount > 0 && runMain+childMain+r.spacing > mainAxisLimit {
			mainExtent = math.Max(mainExtent, runMain)
			crossExtent += runCross + r.runSpacing
			runMain, runCross, childCount = 0, 0, 0
		}
		runMain += childMain
		runCross = math.Max(runCross, childCross)
		if childCount > 0 {
			runMain += r.spacing
		}
		childCount++
	}
	crossExtent += runCross
	mainExtent = math.Max(mainExtent, runMain)
	if r.direction == painting.AxisHorizontal {
		return constraints.Constrain(ui.Size{Width: mainExtent, Height: crossExtent})
	}
	return constraints.Constrain(ui.Size{Width: crossExtent, Height: mainExtent})
}

// PerformLayout divides the children into runs, sizes the wrap around them and
// positions them.
//
// Flutter: rendering/wrap.dart RenderWrap.performLayout
func (r *RenderWrap) PerformLayout() {
	constraints := r.BoxConstraints()
	if foundation.DebugMode {
		r.debugCheckDirections()
	}
	if isNil(r.FirstChild()) {
		r.SetSize(constraints.Smallest())
		r.hasVisualOverflow = false
		return
	}
	childrenAxisSize, runMetrics := r.computeRuns(constraints, ChildLayoutHelper.LayoutChild)
	containerAxisSize := childrenAxisSize.applyConstraints(constraints, r.direction)
	r.SetSize(containerAxisSize.toSize(r.direction))
	freeAxisSize := containerAxisSize.sub(childrenAxisSize)
	r.hasVisualOverflow = freeAxisSize.main < 0 || freeAxisSize.cross < 0
	r.positionChildren(runMetrics, freeAxisSize, containerAxisSize,
		func(offset ui.Offset, child Box) { wrapParentData(child).Offset = offset },
		func(child Box) ui.Size { return child.Size() })
}

// computeRuns lays the children out and divides them into runs.
//
// Flutter: rendering/wrap.dart RenderWrap._computeRuns
func (r *RenderWrap) computeRuns(constraints BoxConstraints, layoutChild ChildLayouter) (axisSize, []*wrapRunMetrics) {
	childConstraints, mainAxisLimit := r.childConstraintsAndLimit(constraints)
	flipMainAxis, _ := r.areAxesFlipped()
	var runMetrics []*wrapRunMetrics
	var currentRun *wrapRunMetrics
	var childrenAxisSize axisSize
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		childSize := axisSizeFromSize(layoutChild(child, childConstraints), r.direction)
		var newRun *wrapRunMetrics
		if currentRun == nil {
			newRun = &wrapRunMetrics{axisSize: childSize, childCount: 1, leadingChild: child}
		} else {
			newRun = currentRun.tryAddingNewChild(child, childSize, flipMainAxis, r.spacing, mainAxisLimit)
		}
		if newRun != nil {
			runMetrics = append(runMetrics, newRun)
			if currentRun != nil {
				childrenAxisSize = childrenAxisSize.add(currentRun.axisSize.flipped())
			}
			currentRun = newRun
		}
	}
	totalRunSpacing := r.runSpacing * float64(len(runMetrics)-1)
	childrenAxisSize = childrenAxisSize.add(axisSize{totalRunSpacing, 0}.add(currentRun.axisSize.flipped()))
	return childrenAxisSize.flipped(), runMetrics
}

// positionChildren positions the children of every run.
//
// Flutter: rendering/wrap.dart RenderWrap._positionChildren
func (r *RenderWrap) positionChildren(runMetrics []*wrapRunMetrics, freeAxisSize, containerAxisSize axisSize,
	positionChild func(offset ui.Offset, child Box), getChildSize func(child Box) ui.Size) {
	crossAxisFreeSpace := math.Max(0, freeAxisSize.cross)
	flipMainAxis, flipCrossAxis := r.areAxesFlipped()
	effectiveCrossAlignment := r.crossAxisAlignment
	if flipCrossAxis {
		effectiveCrossAlignment = effectiveCrossAlignment.flipped()
	}
	runLeadingSpace, runBetweenSpace := r.runAlignment.distributeSpace(crossAxisFreeSpace, r.runSpacing, len(runMetrics), flipCrossAxis)
	next := r.ChildAfter
	if flipMainAxis {
		next = r.ChildBefore
	}
	runCrossAxisOffset := runLeadingSpace
	for i := range runMetrics {
		run := runMetrics[i]
		if flipCrossAxis {
			run = runMetrics[len(runMetrics)-1-i]
		}
		runCrossAxisExtent := run.axisSize.cross
		mainAxisFreeSpace := math.Max(0, containerAxisSize.main-run.axisSize.main)
		childLeadingSpace, childBetweenSpace := r.alignment.distributeSpace(mainAxisFreeSpace, r.spacing, run.childCount, flipMainAxis)
		childMainAxisOffset := childLeadingSpace
		remaining := run.childCount
		for child := run.leadingChild; !isNil(child) && remaining > 0; child, remaining = next(child), remaining-1 {
			childAxisSize := axisSizeFromSize(getChildSize(child), r.direction)
			childCrossAxisOffset := effectiveCrossAlignment.fraction() * (runCrossAxisExtent - childAxisSize.cross)
			positionChild(r.offsetOf(childMainAxisOffset, runCrossAxisOffset+childCrossAxisOffset), child)
			childMainAxisOffset += childAxisSize.main + childBetweenSpace
		}
		runCrossAxisOffset += runCrossAxisExtent + runBetweenSpace
	}
}

// HitTestChildren hit tests the children from the front.
//
// Flutter: rendering/wrap.dart RenderWrap.hitTestChildren
func (r *RenderWrap) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	return DefaultHitTestChildren(&r.ContainerRenderObject, result, position)
}

func (r *RenderWrap) defaultPaint(context *PaintingContext, offset ui.Offset) {
	DefaultPaint(&r.ContainerRenderObject, context, offset)
}

// Paint paints the children, clipped to the wrap if they overflow it.
//
// Flutter: rendering/wrap.dart RenderWrap.paint
func (r *RenderWrap) Paint(context *PaintingContext, offset ui.Offset) {
	if r.hasVisualOverflow && r.clipBehavior != ui.ClipNone {
		r.clipRectLayer.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()),
			r.defaultPaint, PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: r.clipRectLayer.Layer()}))
		return
	}
	r.clipRectLayer.SetLayer(nil)
	r.defaultPaint(context, offset)
}

// Dispose lets the clip layer go.
//
// Flutter: rendering/wrap.dart RenderWrap.dispose
func (r *RenderWrap) Dispose() {
	r.clipRectLayer.SetLayer(nil)
	r.RenderBox.Dispose()
}

// DebugFillProperties adds the direction, the alignments and the spacings.
//
// Flutter: rendering/wrap.dart RenderWrap.debugFillProperties
func (r *RenderWrap) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderBox.DebugFillProperties(b)
	b.Add(foundation.NewEnumProperty("direction", foundation.Some(r.direction)))
	b.Add(foundation.NewEnumProperty("alignment", foundation.Some(r.alignment)))
	b.Add(foundation.NewDoubleProperty("spacing", foundation.Some(r.spacing)))
	b.Add(foundation.NewEnumProperty("runAlignment", foundation.Some(r.runAlignment)))
	b.Add(foundation.NewDoubleProperty("runSpacing", foundation.Some(r.runSpacing)))
	// Flutter reports the run spacing under this name, too.
	b.Add(foundation.NewDoubleProperty("crossAxisAlignment", foundation.Some(r.runSpacing)))
	b.Add(foundation.NewEnumProperty("textDirection", optEnum(r.textDirection, ui.TextDirectionUnspecified), foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultNull}))
	b.Add(foundation.NewEnumProperty("verticalDirection", foundation.Some(r.verticalDirection), foundation.EnumPropertyOpts{DefaultValue: painting.VerticalDirectionDown}))
}
