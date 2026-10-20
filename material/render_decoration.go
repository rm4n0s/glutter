// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// inputDecoratorFinalLabelScale is the scale of the floating label.
	//
	// Flutter: material/input_decorator.dart _kFinalLabelScale
	inputDecoratorFinalLabelScale = 0.75
	// inputDecoratorInputExtraPadding is the padding that the input content has in
	// addition to the one of the prefix icon: from the M3 spec, the horizontal
	// padding is 12 pixels for the prefix icon and 16 pixels for the input
	// content.
	//
	// Flutter: material/input_decorator.dart _kInputExtraPadding
	inputDecoratorInputExtraPadding = 4.0
	// inputDecoratorSubtextCounterPadding is the padding between the character
	// counter and the helper or error text, to prevent an overlap.
	//
	// Flutter: material/input_decorator.dart _kSubtextCounterPadding
	inputDecoratorSubtextCounterPadding = 16.0
)

// decorationSlot identifies the children of a renderDecoration.
//
// Flutter: material/input_decorator.dart _DecorationSlot
type decorationSlot int

const (
	slotIcon decorationSlot = iota
	slotInput
	slotLabel
	slotHint
	slotPrefix
	slotSuffix
	slotPrefixIcon
	slotSuffixIcon
	slotHelperError
	slotCounter
	slotContainer
)

var decorationSlotNames = [...]string{"icon", "input", "label", "hint", "prefix", "suffix", "prefixIcon", "suffixIcon", "helperError", "counter", "container"}

func (s decorationSlot) String() string { return decorationSlotNames[s] }

// allDecorationSlots are the slots of a decorator, in the order of Flutter's
// enum.
var allDecorationSlots = []any{slotIcon, slotInput, slotLabel, slotHint, slotPrefix, slotSuffix, slotPrefixIcon, slotSuffixIcon, slotHelperError, slotCounter, slotContainer}

// inputBorderGap is the gap in the outline border of an InputDecorator where the
// floating label will appear. It is a notifier: the border container repaints
// when the gap changes.
//
// Flutter: material/input_decorator.dart _InputBorderGap
type inputBorderGap struct {
	foundation.ChangeNotifier
	start  foundation.Opt[float64]
	extent float64
}

func newInputBorderGap() *inputBorderGap { return &inputBorderGap{} }

func (g *inputBorderGap) setStart(v foundation.Opt[float64]) {
	if v != g.start {
		g.start = v
		g.NotifyListeners()
	}
}

func (g *inputBorderGap) setExtent(v float64) {
	if v != g.extent {
		g.extent = v
		g.NotifyListeners()
	}
}

// decoration is an analog of InputDecoration for the decorator widget.
//
// Flutter: material/input_decorator.dart _Decoration
type decoration struct {
	contentPadding         painting.EdgeInsetsDirectional
	isCollapsed            bool
	floatingLabelHeight    float64
	floatingLabelProgress  float64
	floatingLabelAlignment FloatingLabelAlignment
	border                 InputBorder
	borderGap              *inputBorderGap
	alignLabelWithHint     bool
	isDense                bool
	isEmpty                bool
	visualDensity          VisualDensity
	inputGap               float64
	maintainHintSize       bool
	maintainLabelSize      bool
	icon                   widgets.Widget
	input                  widgets.Widget
	label                  widgets.Widget
	hint                   widgets.Widget
	prefix                 widgets.Widget
	suffix                 widgets.Widget
	prefixIcon             widgets.Widget
	suffixIcon             widgets.Widget
	helperError            widgets.Widget
	counter                widgets.Widget
	container              widgets.Widget
}

// equal reports whether the decorations are the same: the widgets by identity.
func (d decoration) equal(o decoration) (equal bool) {
	defer func() {
		if recover() != nil {
			// A border whose dynamic type cannot be compared.
			equal = false
		}
	}()
	return d == o
}

// subtextSize is the size of the subtext (the helper, the error and the
// counter): where its baseline is, how far it goes below the container, and how
// high it is.
//
// Flutter: material/input_decorator.dart _SubtextSize
type subtextSize struct {
	ascent, bottomHeight, subtextHeight float64
}

// decorationLayout holds the layout values computed by renderDecoration.layout,
// which performLayout uses to position all of the children.
//
// Flutter: material/input_decorator.dart _RenderDecorationLayout
type decorationLayout struct {
	inputConstraints rendering.BoxConstraints
	baseline         float64
	containerHeight  float64
	subtext          *subtextSize
	size             ui.Size
}

// baselineGetter gets the baseline of a child laid out with the constraints, the
// real one or the dry one.
//
// Flutter: material/input_decorator.dart _ChildBaselineGetter
type baselineGetter func(child rendering.Box, constraints rendering.BoxConstraints) float64

// renderDecoration lays out and paints the parts of a decorator: the icon, the
// container, the label, the hint, the prefix and suffix, the input, the helper or
// error text and the counter.
//
// Flutter: material/input_decorator.dart _RenderDecoration
type renderDecoration struct {
	rendering.RenderBox
	rendering.SlottedContainer
	decoration        decoration
	textDirection     ui.TextDirection
	textBaseline      ui.TextBaseline
	textAlignVertical foundation.Opt[painting.TextAlignVertical]
	isFocused         bool
	expands           bool
	material3         bool
	// labelTransform records where the label was painted.
	labelTransform *ui.Matrix4
}

func newRenderDecoration(d decoration, textDirection ui.TextDirection, textBaseline ui.TextBaseline, isFocused, expands, material3 bool,
	textAlignVertical foundation.Opt[painting.TextAlignVertical]) *renderDecoration {
	r := &renderDecoration{decoration: d, textDirection: textDirection, textBaseline: textBaseline, isFocused: isFocused,
		expands: expands, material3: material3, textAlignVertical: textAlignVertical}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitSlottedContainer(&r.SlottedContainer, r)
	return r
}

// The gaps of the layout. TODO(bleroux): consider defining these values as
// Material tokens, configurable by InputDecorationThemeData.
func (r *renderDecoration) subtextGap() float64 {
	if r.material3 {
		return 4.0
	}
	return 8.0
}

func (r *renderDecoration) prefixToInputGap() float64 {
	if r.material3 {
		return 4.0
	}
	return 0.0
}

func (r *renderDecoration) inputToSuffixGap() float64 {
	if r.material3 {
		return 4.0
	}
	return 0.0
}

// slot is the box in the slot, or nil.
func (r *renderDecoration) slot(s decorationSlot) rendering.Box {
	c := r.SlotChild(s)
	if c == nil {
		return nil
	}
	return c.(rendering.Box)
}

func (r *renderDecoration) icon() rendering.Box        { return r.slot(slotIcon) }
func (r *renderDecoration) input() rendering.Box       { return r.slot(slotInput) }
func (r *renderDecoration) label() rendering.Box       { return r.slot(slotLabel) }
func (r *renderDecoration) hint() rendering.Box        { return r.slot(slotHint) }
func (r *renderDecoration) prefix() rendering.Box      { return r.slot(slotPrefix) }
func (r *renderDecoration) suffix() rendering.Box      { return r.slot(slotSuffix) }
func (r *renderDecoration) prefixIcon() rendering.Box  { return r.slot(slotPrefixIcon) }
func (r *renderDecoration) suffixIcon() rendering.Box  { return r.slot(slotSuffixIcon) }
func (r *renderDecoration) helperError() rendering.Box { return r.slot(slotHelperError) }
func (r *renderDecoration) counter() rendering.Box     { return r.slot(slotCounter) }
func (r *renderDecoration) container() rendering.Box   { return r.slot(slotContainer) }

// hitTestOrder is the children, ordered for hit testing.
func (r *renderDecoration) hitTestOrder() []rendering.Box {
	var out []rendering.Box
	for _, b := range []rendering.Box{r.icon(), r.input(), r.prefixIcon(), r.suffixIcon(), r.prefix(), r.suffix(), r.label(), r.hint(),
		r.helperError(), r.counter(), r.container()} {
		if b != nil {
			out = append(out, b)
		}
	}
	return out
}

func (r *renderDecoration) setDecoration(value decoration) {
	if r.decoration.equal(value) {
		return
	}
	r.decoration = value
	r.MarkNeedsLayout()
}

func (r *renderDecoration) setTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.MarkNeedsLayout()
}

func (r *renderDecoration) setTextBaseline(value ui.TextBaseline) {
	if r.textBaseline == value {
		return
	}
	r.textBaseline = value
	r.MarkNeedsLayout()
}

func (r *renderDecoration) defaultTextAlignVertical() painting.TextAlignVertical {
	if r.isOutlineAligned() {
		return painting.TextAlignVerticalCenter
	}
	return painting.TextAlignVerticalTop
}

func (r *renderDecoration) effectiveTextAlignVertical() painting.TextAlignVertical {
	return r.textAlignVertical.OrElse(r.defaultTextAlignVertical())
}

func (r *renderDecoration) setTextAlignVertical(value foundation.Opt[painting.TextAlignVertical]) {
	if r.textAlignVertical == value {
		return
	}
	// No need to relayout if the effective value is still the same.
	next := value.OrElse(r.defaultTextAlignVertical())
	same := r.effectiveTextAlignVertical().Y == next.Y
	r.textAlignVertical = value
	if !same {
		r.MarkNeedsLayout()
	}
}

func (r *renderDecoration) setIsFocused(value bool) {
	if r.isFocused == value {
		return
	}
	r.isFocused = value
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderDecoration) setExpands(value bool) {
	if r.expands == value {
		return
	}
	r.expands = value
	r.MarkNeedsLayout()
}

// isOutlineAligned says that the decoration should be aligned to accommodate an
// outline border.
func (r *renderDecoration) isOutlineAligned() bool {
	return !r.decoration.isCollapsed && r.decoration.border.IsOutline()
}

func (r *renderDecoration) densityOffset() ui.Offset {
	return r.decoration.visualDensity.BaseSizeAdjustment()
}

// VisitChildrenForSemantics visits the children in the order of the semantics.
//
// Flutter: material/input_decorator.dart _RenderDecoration.visitChildrenForSemantics
func (r *renderDecoration) VisitChildrenForSemantics(visitor rendering.RenderObjectVisitor) {
	visit := func(b rendering.Box) {
		if b != nil {
			visitor(b)
		}
	}
	visit(r.icon())
	visit(r.prefix())
	visit(r.prefixIcon())
	visit(r.label())
	if hint := r.hint(); hint != nil {
		if r.isFocused {
			visitor(hint)
		} else if r.label() == nil {
			visitor(hint)
		}
	}
	visit(r.input())
	visit(r.suffixIcon())
	visit(r.suffix())
	visit(r.container())
	visitor(r.helperError())
	visit(r.counter())
}

func minWidthOf(box rendering.Box, height float64) float64 {
	if box == nil {
		return 0.0
	}
	return box.GetMinIntrinsicWidth(height)
}

func maxWidthOf(box rendering.Box, height float64) float64 {
	if box == nil {
		return 0.0
	}
	return box.GetMaxIntrinsicWidth(height)
}

func minHeightOf(box rendering.Box, width float64) float64 {
	if box == nil {
		return 0.0
	}
	return box.GetMinIntrinsicHeight(width)
}

func boxSizeOf(box rendering.Box) ui.Size {
	if box == nil {
		return ui.Size{}
	}
	return box.Size()
}

// realBaseline is the real baseline of a child laid out with the constraints, or
// its height.
//
// Flutter: material/input_decorator.dart _RenderDecoration._getBaseline
func realBaseline(box rendering.Box, constraints rendering.BoxConstraints) float64 {
	return rendering.ChildLayoutHelper.GetBaseline(box, constraints, ui.TextBaselineAlphabetic).OrElse(box.Size().Height)
}

// dryBaseline is the baseline a child would have for the constraints, or the
// height it would have.
//
// Flutter: material/input_decorator.dart _RenderDecoration._getDryBaseline
func dryBaseline(box rendering.Box, constraints rendering.BoxConstraints) float64 {
	if v, ok := rendering.ChildLayoutHelper.GetDryBaseline(box, constraints, ui.TextBaselineAlphabetic).Get(); ok {
		return v
	}
	return rendering.ChildLayoutHelper.DryLayoutChild(box, constraints).Height
}

func boxData(box rendering.Box) *rendering.BoxParentData {
	return box.ParentData().(*rendering.BoxParentData)
}

func (r *renderDecoration) contentPadding() painting.EdgeInsetsDirectional {
	return r.decoration.contentPadding
}

// computeSubtextSizes lays out the helper or error text and the counter; it
// returns nil when none of them takes a space.
//
// Flutter: material/input_decorator.dart _RenderDecoration._computeSubtextSizes
func (r *renderDecoration) computeSubtextSizes(constraints rendering.BoxConstraints, layoutChild rendering.ChildLayouter, getBaseline baselineGetter) *subtextSize {
	counterSize, counterAscent := ui.Size{}, 0.0
	counter := r.counter()
	if counter != nil {
		counterSize, counterAscent = layoutChild(counter, constraints), getBaseline(counter, constraints)
	}
	// Only add padding when the counter is present (maxLength is used).
	counterPadding := 0.0
	if counter != nil {
		counterPadding = inputDecoratorSubtextCounterPadding
	}
	helperErrorConstraints := constraints.Deflate(painting.EdgeInsets{Left: counterSize.Width + counterPadding})
	helperErrorHeight := layoutChild(r.helperError(), helperErrorConstraints).Height
	if helperErrorHeight == 0.0 && counterSize.Height == 0.0 {
		return nil
	}
	// TODO(LongCatIsLooong): the bottomHeight expression doesn't make much sense.
	// Use the real descent and make sure the subtext line box is tall enough for
	// both children (flutter/flutter#13715).
	ascent := math.Max(counterAscent, getBaseline(r.helperError(), helperErrorConstraints)) + r.subtextGap()
	bottomHeight := math.Max(counterAscent, helperErrorHeight) + r.subtextGap()
	subtextHeight := math.Max(counterSize.Height, helperErrorHeight) + r.subtextGap()
	return &subtextSize{ascent: ascent, bottomHeight: bottomHeight, subtextHeight: subtextHeight}
}

// layout returns the values that performLayout uses to position all of the
// children. It lays out all the children except the container, which
// performLayout lays out.
//
// Flutter: material/input_decorator.dart _RenderDecoration._layout
func (r *renderDecoration) layout(constraints rendering.BoxConstraints, layoutChild rendering.ChildLayouter, getBaseline baselineGetter) decorationLayout {
	if foundation.DebugMode && !(constraints.MaxWidth < math.Inf(1)) {
		panic(errors.New("InputDecoratorUnboundedWidth",
			"an InputDecorator, which is typically created by a TextField, cannot have an unbounded width; this happens when the parent widget does not provide a finite width constraint, for example if the InputDecorator is contained by a Row: an Expanded widget or a SizedBox can be used to constrain the width of the InputDecorator or the TextField that contains it"))
	}
	boxConstraints := constraints.Loosen()
	d := r.decoration
	padding := r.contentPadding()

	// Lay out all the widgets used by InputDecorator.
	icon := r.icon()
	iconWidth := 0.0
	if icon != nil {
		iconWidth = layoutChild(icon, boxConstraints).Width
	}
	containerConstraints := boxConstraints.Deflate(painting.EdgeInsets{Left: iconWidth})
	contentConstraints := containerConstraints.Deflate(painting.EdgeInsetsDirectional{
		Start: padding.Start + d.inputGap,
		End:   padding.End + d.inputGap,
	})

	// The helper or error text can occupy the full width less the space occupied
	// by the icon and counter.
	subtext := r.computeSubtextSizes(contentConstraints, layoutChild, getBaseline)

	prefixIcon, suffixIcon := r.prefixIcon(), r.suffixIcon()
	prefixIconSize, suffixIconSize := ui.Size{}, ui.Size{}
	if prefixIcon != nil {
		prefixIconSize = layoutChild(prefixIcon, containerConstraints)
	}
	if suffixIcon != nil {
		suffixIconSize = layoutChild(suffixIcon, containerConstraints)
	}

	prefix, suffix := r.prefix(), r.suffix()
	prefixSize, suffixSize := ui.Size{}, ui.Size{}
	if prefix != nil {
		prefixSize = layoutChild(prefix, contentConstraints)
	}
	if suffix != nil {
		suffixSize = layoutChild(suffix, contentConstraints)
	}

	startInset := iconWidth + prefixSize.Width
	if prefixIcon == nil {
		startInset += padding.Start + d.inputGap
	} else {
		startInset += prefixIconSize.Width + r.prefixToInputGap()
	}
	endInset := suffixSize.Width
	if suffixIcon == nil {
		endInset += padding.End + d.inputGap
	} else {
		endInset += suffixIconSize.Width + r.inputToSuffixGap()
	}
	inputWidth := math.Max(0.0, constraints.MaxWidth-(startInset+endInset))

	label := r.label()
	var topHeight float64
	if label != nil {
		suffixIconSpace := suffixIconSize.Width
		if d.border.IsOutline() {
			suffixIconSpace = ui.LerpDouble(suffixIconSize.Width, padding.End, d.floatingLabelProgress)
		}
		prefixSpace := padding.Start
		if prefixIcon != nil {
			prefixSpace = prefixIconSize.Width
		}
		suffixSpace := padding.End
		if suffixIcon != nil {
			suffixSpace = suffixIconSpace
		}
		labelWidth := math.Max(0.0, constraints.MaxWidth-(d.inputGap*2+iconWidth+prefixSpace+suffixSpace))
		// Increase the available width for the label when it is scaled down.
		invertedLabelScale := ui.LerpDouble(1.00, 1/inputDecoratorFinalLabelScale, d.floatingLabelProgress)
		labelConstraints := boxConstraints.CopyWith(rendering.BoxConstraintsCopy{MaxWidth: foundation.Some(labelWidth * invertedLabelScale)})
		layoutChild(label, labelConstraints)
		labelHeight := d.floatingLabelHeight
		if d.border.IsOutline() {
			topHeight = math.Max(labelHeight-getBaseline(label, labelConstraints), 0.0)
		} else {
			topHeight = labelHeight
		}
	}

	// The height of the input needs to accommodate the label above and the
	// counter and helperError below, when they exist.
	bottomHeight := 0.0
	if subtext != nil {
		bottomHeight = subtext.bottomHeight
	}
	inputConstraints := boxConstraints.
		Deflate(painting.EdgeInsets{Top: padding.Vertical() + topHeight + bottomHeight + r.densityOffset().Dy}).
		Tighten(rendering.TightenOpts{Width: foundation.Some(inputWidth)})

	input, hint := r.input(), r.hint()
	inputSize, hintSize := ui.Size{}, ui.Size{}
	hintConstraints := boxConstraints.Tighten(rendering.TightenOpts{Width: foundation.Some(inputWidth)})
	if input != nil {
		inputSize = layoutChild(input, inputConstraints)
	}
	if hint != nil {
		hintSize = layoutChild(hint, hintConstraints)
	}
	inputBaseline, hintBaseline := 0.0, 0.0
	if input != nil {
		inputBaseline = getBaseline(input, inputConstraints)
	}
	if hint != nil {
		hintBaseline = getBaseline(hint, hintConstraints)
	}

	// The field can be occupied by a hint or by the input itself.
	hintHeight := 0.0
	if d.isEmpty || d.maintainHintSize {
		hintHeight = hintSize.Height
	}
	inputHeight := math.Max(hintHeight, inputSize.Height)
	inputInternalBaseline := math.Max(inputBaseline, hintBaseline)

	prefixBaseline, suffixBaseline := 0.0, 0.0
	if prefix != nil {
		prefixBaseline = getBaseline(prefix, contentConstraints)
	}
	if suffix != nil {
		suffixBaseline = getBaseline(suffix, contentConstraints)
	}

	// Calculate the amount that the prefix and suffix affect the height above and
	// below the input.
	fixHeight := math.Max(prefixBaseline, suffixBaseline)
	fixAboveInput := math.Max(0, fixHeight-inputInternalBaseline)
	fixBelowBaseline := math.Max(prefixSize.Height-prefixBaseline, suffixSize.Height-suffixBaseline)
	// TODO(justinmc): fixBelowInput should have no effect when there is no
	// prefix or suffix below the input (flutter/flutter#66050).
	fixBelowInput := math.Max(0, fixBelowBaseline-(inputHeight-inputInternalBaseline))

	// Calculate the height of the input text container.
	fixIconHeight := math.Max(prefixIconSize.Height, suffixIconSize.Height)
	contentHeight := math.Max(fixIconHeight,
		topHeight+padding.Top+fixAboveInput+inputHeight+fixBelowInput+padding.Bottom+r.densityOffset().Dy)
	minContainerHeight := MinInteractiveDimension
	if d.isDense || d.isCollapsed || r.expands {
		minContainerHeight = inputHeight
	}
	maxContainerHeight := math.Max(0.0, boxConstraints.MaxHeight-bottomHeight)
	var containerHeight float64
	if r.expands {
		containerHeight = maxContainerHeight
	} else {
		containerHeight = math.Min(math.Max(contentHeight, minContainerHeight), maxContainerHeight)
	}

	// Ensure the text is vertically centered in cases where the content is
	// shorter than kMinInteractiveDimension.
	interactiveAdjustment := 0.0
	if minContainerHeight > contentHeight {
		interactiveAdjustment = (minContainerHeight - contentHeight) / 2.0
	}

	// Try to consider the prefix/suffix as part of the text when aligning it. If
	// the prefix/suffix overflows however, allow it to extend outside of the input
	// and align the remaining part of the text and prefix/suffix.
	overflow := math.Max(0, contentHeight-maxContainerHeight)
	// Map textAlignVertical from -1:1 to 0:1 so that it can be used to scale the
	// baseline from its minimum to maximum values.
	alignY := r.effectiveTextAlignVertical().Y
	textAlignVerticalFactor := (alignY + 1.0) / 2.0
	// Adjust to try to fit top overflow inside the input on an inverse scale of
	// textAlignVertical, so that top aligned text adjusts the most and bottom
	// aligned text doesn't adjust at all.
	baselineAdjustment := fixAboveInput - overflow*(1-textAlignVerticalFactor)

	// The baselines that will be used to draw the actual input text content.
	topInputBaseline := padding.Top + topHeight + inputInternalBaseline + baselineAdjustment + interactiveAdjustment + r.densityOffset().Dy/2.0
	maxContentHeight := containerHeight - padding.Vertical() - topHeight - r.densityOffset().Dy
	alignableHeight := fixAboveInput + inputHeight + fixBelowInput
	maxVerticalOffset := maxContentHeight - alignableHeight
	var baseline float64
	if r.isOutlineAligned() {
		// The three main alignments for the baseline when an outline is present are
		//
		//  * top (-1.0): topmost point considering padding.
		//  * center (0.0): the absolute center of the input ignoring padding but
		//      accommodating the border and floating label.
		//  * bottom (1.0): bottommost point considering padding.
		//
		// That means that if the padding is uneven, center is not the exact
		// midpoint of top and bottom. To account for this, the above center and
		// below center alignments are interpolated independently.
		outlineCenterBaseline := inputInternalBaseline + baselineAdjustment/2.0 + (containerHeight-inputHeight)/2.0
		outlineTopBaseline := topInputBaseline
		outlineBottomBaseline := topInputBaseline + maxVerticalOffset
		baseline = interpolateThree(outlineTopBaseline, outlineCenterBaseline, outlineBottomBaseline, alignY)
	} else {
		baseline = topInputBaseline + maxVerticalOffset*textAlignVerticalFactor
	}
	subtextHeight := 0.0
	if subtext != nil {
		subtextHeight = subtext.subtextHeight
	}
	return decorationLayout{
		inputConstraints: inputConstraints,
		containerHeight:  containerHeight,
		baseline:         baseline,
		subtext:          subtext,
		size:             ui.Size{Width: constraints.MaxWidth, Height: containerHeight + subtextHeight},
	}
}

// interpolateThree interpolates between three stops using textAlignVertical. This
// is used to calculate the outline baseline, which ignores padding when the
// alignment is middle. When the alignment is less than zero, it interpolates
// between the centered text box's top and the top of the content padding. When
// the alignment is greater than zero, it interpolates between the centered box's
// top and the position that would align the bottom of the box with the bottom
// padding.
//
// Flutter: material/input_decorator.dart _RenderDecoration._interpolateThree
func interpolateThree(begin, middle, end, y float64) float64 {
	// It's possible for begin, middle, and end to not be in order because of
	// excessive padding. Those cases are handled by using middle.
	var basis float64
	if y <= 0 {
		basis = math.Max(middle-begin, 0)
	} else {
		basis = math.Max(end-middle, 0)
	}
	return middle + basis*y
}

func (r *renderDecoration) horizontalExtras(height float64) (inputWidth float64) {
	d := r.decoration
	if d.isEmpty || d.maintainHintSize {
		return math.Max(minWidthOf(r.input(), height), minWidthOf(r.hint(), height))
	}
	return minWidthOf(r.input(), height)
}

// ComputeMinIntrinsicWidth is the sum of the minimum widths of the parts.
//
// Flutter: material/input_decorator.dart _RenderDecoration.computeMinIntrinsicWidth
func (r *renderDecoration) ComputeMinIntrinsicWidth(height float64) float64 {
	d := r.decoration
	padding := r.contentPadding()
	inputWidth := r.horizontalExtras(height)
	contentWidth := inputWidth
	if d.maintainLabelSize {
		contentWidth = math.Max(inputWidth, minWidthOf(r.label(), height))
	}
	start := padding.Start + d.inputGap
	if r.prefixIcon() != nil {
		start = r.prefixToInputGap()
	}
	end := padding.End + d.inputGap
	if r.suffixIcon() != nil {
		end = r.inputToSuffixGap()
	}
	return minWidthOf(r.icon(), height) + start + minWidthOf(r.prefixIcon(), height) + minWidthOf(r.prefix(), height) +
		contentWidth + minWidthOf(r.suffix(), height) + minWidthOf(r.suffixIcon(), height) + end
}

// ComputeMaxIntrinsicWidth is the sum of the maximum widths of the parts.
//
// Flutter: material/input_decorator.dart _RenderDecoration.computeMaxIntrinsicWidth
func (r *renderDecoration) ComputeMaxIntrinsicWidth(height float64) float64 {
	d := r.decoration
	padding := r.contentPadding()
	var inputWidth float64
	if d.isEmpty || d.maintainHintSize {
		inputWidth = math.Max(maxWidthOf(r.input(), height), maxWidthOf(r.hint(), height))
	} else {
		inputWidth = maxWidthOf(r.input(), height)
	}
	contentWidth := inputWidth
	if d.maintainLabelSize {
		contentWidth = math.Max(inputWidth, maxWidthOf(r.label(), height))
	}
	start := padding.Start + d.inputGap
	if r.prefixIcon() != nil {
		start = r.prefixToInputGap()
	}
	end := padding.End + d.inputGap
	if r.suffixIcon() != nil {
		end = r.inputToSuffixGap()
	}
	return maxWidthOf(r.icon(), height) + start + maxWidthOf(r.prefixIcon(), height) + maxWidthOf(r.prefix(), height) +
		contentWidth + maxWidthOf(r.suffix(), height) + maxWidthOf(r.suffixIcon(), height) + end
}

func lineHeightOf(width float64, boxes ...rendering.Box) float64 {
	height := 0.0
	for _, box := range boxes {
		if box == nil {
			continue
		}
		height = math.Max(minHeightOf(box, width), height)
	}
	// TODO(hansmuller): this should compute the overall line height for the boxes
	// when they've been baseline-aligned (flutter/flutter#13715).
	return height
}

// ComputeMinIntrinsicHeight is the height of the container and the subtext.
//
// Flutter: material/input_decorator.dart _RenderDecoration.computeMinIntrinsicHeight
func (r *renderDecoration) ComputeMinIntrinsicHeight(width float64) float64 {
	d := r.decoration
	padding := r.contentPadding()
	iconHeight := minHeightOf(r.icon(), width)
	iconWidth := minWidthOf(r.icon(), iconHeight)
	width = math.Max(width-iconWidth, 0.0)
	prefixIconHeight := minHeightOf(r.prefixIcon(), width)
	prefixIconWidth := minWidthOf(r.prefixIcon(), prefixIconHeight)
	suffixIconHeight := minHeightOf(r.suffixIcon(), width)
	suffixIconWidth := minWidthOf(r.suffixIcon(), suffixIconHeight)
	width = math.Max(width-padding.Horizontal()-d.inputGap*2, 0.0)

	// TODO(LongCatIsLooong): use computeSubtextSizes for the subtext intrinsic
	// sizes (flutter/flutter#13715).
	counter := r.counter()
	counterHeight := minHeightOf(counter, width)
	counterWidth := minWidthOf(counter, counterHeight)
	// Only add padding when the counter is present (maxLength is used).
	counterPadding := 0.0
	if counter != nil {
		counterPadding = inputDecoratorSubtextCounterPadding
	}
	helperErrorAvailableWidth := math.Max(width-counterWidth-counterPadding, 0.0)
	helperErrorHeight := minHeightOf(r.helperError(), helperErrorAvailableWidth)
	subtextHeight := math.Max(counterHeight, helperErrorHeight)
	if subtextHeight > 0.0 {
		subtextHeight += r.subtextGap()
	}

	prefixHeight := minHeightOf(r.prefix(), width)
	prefixWidth := minWidthOf(r.prefix(), prefixHeight)
	suffixHeight := minHeightOf(r.suffix(), width)
	suffixWidth := minWidthOf(r.suffix(), suffixHeight)
	availableInputWidth := math.Max(width-prefixWidth-suffixWidth-prefixIconWidth-suffixIconWidth, 0.0)
	var inputHeight float64
	if d.isEmpty {
		inputHeight = lineHeightOf(availableInputWidth, r.input(), r.hint())
	} else {
		inputHeight = lineHeightOf(availableInputWidth, r.input())
	}
	inputMaxHeight := math.Max(inputHeight, math.Max(prefixHeight, suffixHeight))
	labelHeight := 0.0
	if r.label() != nil {
		labelHeight = d.floatingLabelHeight
	}
	contentHeight := padding.Top + labelHeight + inputMaxHeight + padding.Bottom + r.densityOffset().Dy
	containerHeight := math.Max(math.Max(iconHeight, contentHeight), math.Max(prefixIconHeight, suffixIconHeight))
	minContainerHeight := MinInteractiveDimension
	if d.isDense || r.expands {
		minContainerHeight = 0.0
	}
	return math.Max(containerHeight, minContainerHeight) + subtextHeight
}

// ComputeMaxIntrinsicHeight is the minimum intrinsic height.
//
// Flutter: material/input_decorator.dart _RenderDecoration.computeMaxIntrinsicHeight
func (r *renderDecoration) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.GetMinIntrinsicHeight(width)
}

// ComputeDistanceToActualBaseline is the baseline of the input.
//
// Flutter: material/input_decorator.dart _RenderDecoration.computeDistanceToActualBaseline
func (r *renderDecoration) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	input := r.input()
	if input == nil {
		return foundation.Some(0.0)
	}
	return foundation.Some(boxData(input).Offset.Dy + input.GetDistanceToActualBaseline(baseline).OrElse(input.Size().Height))
}

// ComputeDryBaseline is the baseline of the input for the constraints.
//
// Flutter: material/input_decorator.dart _RenderDecoration.computeDryBaseline
func (r *renderDecoration) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	input := r.input()
	if input == nil {
		return foundation.Some(0.0)
	}
	layout := r.layout(constraints, rendering.ChildLayoutHelper.DryLayoutChild, dryBaseline)
	base := 0.0
	if baseline == ui.TextBaselineIdeographic {
		heightOf := func(b ui.TextBaseline) float64 {
			if v, ok := input.GetDryBaseline(layout.inputConstraints, b).Get(); ok {
				return v
			}
			return input.GetDryLayout(layout.inputConstraints).Height
		}
		base = heightOf(ui.TextBaselineIdeographic) - heightOf(ui.TextBaselineAlphabetic)
	}
	return foundation.Some(base + layout.baseline)
}

// ComputeDryLayout is the size for the constraints.
//
// Flutter: material/input_decorator.dart _RenderDecoration.computeDryLayout
func (r *renderDecoration) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	layout := r.layout(constraints, rendering.ChildLayoutHelper.DryLayoutChild, dryBaseline)
	return constraints.Constrain(layout.size)
}

// PerformLayout lays out the children and positions them.
//
// Flutter: material/input_decorator.dart _RenderDecoration.performLayout
func (r *renderDecoration) PerformLayout() {
	constraints := r.BoxConstraints()
	r.labelTransform = nil
	layout := r.layout(constraints, rendering.ChildLayoutHelper.LayoutChild, realBaseline)
	r.SetSize(constraints.Constrain(layout.size))
	overallWidth := layout.size.Width
	d := r.decoration
	padding := r.contentPadding()
	icon, container := r.icon(), r.container()

	if container != nil {
		containerConstraints := rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{
			Height: foundation.Some(layout.containerHeight),
			Width:  foundation.Some(overallWidth - boxSizeOf(icon).Width),
		})
		container.Layout(containerConstraints, rendering.LayoutOpts{ParentUsesSize: true})
		x := boxSizeOf(icon).Width
		if r.textDirection == ui.TextDirectionRTL {
			x = 0.0
		}
		boxData(container).Offset = ui.Offset{Dx: x, Dy: 0.0}
	}

	height := layout.containerHeight
	centerLayout := func(box rendering.Box, x float64) float64 {
		boxData(box).Offset = ui.Offset{Dx: x, Dy: (height - box.Size().Height) / 2.0}
		return box.Size().Width
	}

	if icon != nil {
		x := 0.0
		if r.textDirection == ui.TextDirectionRTL {
			x = overallWidth - icon.Size().Width
		}
		centerLayout(icon, x)
	}

	subtextBaseline := layout.containerHeight
	if layout.subtext != nil {
		subtextBaseline += layout.subtext.ascent
	}
	counter, helperError := r.counter(), r.helperError()
	helperErrorBaseline := helperError.GetDistanceToBaseline(ui.TextBaselineAlphabetic).Value()
	counterBaseline := 0.0
	if counter != nil {
		counterBaseline = counter.GetDistanceToBaseline(ui.TextBaselineAlphabetic).Value()
	}

	var start, end float64
	if r.textDirection == ui.TextDirectionLTR {
		start = padding.Start + boxSizeOf(icon).Width
		end = overallWidth - padding.End
		boxData(helperError).Offset = ui.Offset{Dx: start + d.inputGap, Dy: subtextBaseline - helperErrorBaseline}
		if counter != nil {
			boxData(counter).Offset = ui.Offset{Dx: end - counter.Size().Width - d.inputGap, Dy: subtextBaseline - counterBaseline}
		}
	} else {
		start = overallWidth - padding.Start - boxSizeOf(icon).Width
		end = padding.End
		boxData(helperError).Offset = ui.Offset{Dx: start - helperError.Size().Width - d.inputGap, Dy: subtextBaseline - helperErrorBaseline}
		if counter != nil {
			boxData(counter).Offset = ui.Offset{Dx: end + d.inputGap, Dy: subtextBaseline - counterBaseline}
		}
	}

	baseline := layout.baseline
	baselineLayout := func(box rendering.Box, x float64) float64 {
		boxData(box).Offset = ui.Offset{Dx: x, Dy: baseline - box.GetDistanceToBaseline(ui.TextBaselineAlphabetic).Value()}
		return box.Size().Width
	}

	label, prefix, prefixIcon, suffix, suffixIcon := r.label(), r.prefix(), r.prefixIcon(), r.suffix(), r.suffixIcon()
	input, hint := r.input(), r.hint()
	if r.textDirection == ui.TextDirectionRTL {
		if prefixIcon != nil {
			start += padding.Start
			start -= centerLayout(prefixIcon, start-prefixIcon.Size().Width)
			start -= r.prefixToInputGap()
		} else {
			start -= d.inputGap
		}
		if label != nil {
			if d.alignLabelWithHint {
				baselineLayout(label, start-label.Size().Width)
			} else {
				centerLayout(label, start-label.Size().Width)
			}
		}
		if prefix != nil {
			start -= baselineLayout(prefix, start-prefix.Size().Width)
		}
		if input != nil {
			baselineLayout(input, start-input.Size().Width)
		}
		if hint != nil {
			baselineLayout(hint, start-hint.Size().Width)
		}
		if suffixIcon != nil {
			end -= padding.End
			end += centerLayout(suffixIcon, end)
			end += r.inputToSuffixGap()
		} else {
			end += d.inputGap
		}
		if suffix != nil {
			baselineLayout(suffix, end)
		}
	} else {
		if prefixIcon != nil {
			start -= padding.Start
			start += centerLayout(prefixIcon, start)
			start += r.prefixToInputGap()
		} else {
			start += d.inputGap
		}
		if label != nil {
			if d.alignLabelWithHint {
				baselineLayout(label, start)
			} else {
				centerLayout(label, start)
			}
		}
		if prefix != nil {
			start += baselineLayout(prefix, start)
		}
		if input != nil {
			baselineLayout(input, start)
		}
		if hint != nil {
			baselineLayout(hint, start)
		}
		if suffixIcon != nil {
			end += padding.End
			end -= centerLayout(suffixIcon, end-suffixIcon.Size().Width)
			end -= r.inputToSuffixGap()
		} else {
			end -= d.inputGap
		}
		if suffix != nil {
			baselineLayout(suffix, end-suffix.Size().Width)
		}
	}

	if label != nil {
		labelX := boxData(label).Offset.Dx
		// +1 shifts the range of x from (-1.0, 1.0) to (0.0, 2.0).
		floatAlign := d.floatingLabelAlignment.x + 1
		floatWidth := boxSizeOf(label).Width * inputDecoratorFinalLabelScale
		// When the floating label is centered, its x is relative to the
		// container's x and is independent of the label's x.
		if r.textDirection == ui.TextDirectionRTL {
			offsetToPrefixIcon := 0.0
			if prefixIcon != nil && !d.alignLabelWithHint && r.material3 {
				offsetToPrefixIcon = boxSizeOf(prefixIcon).Width - padding.End
			}
			d.borderGap.setStart(foundation.Some(ui.LerpDouble(
				labelX+boxSizeOf(label).Width+offsetToPrefixIcon,
				boxSizeOf(container).Width/2.0+floatWidth/2.0,
				floatAlign)))
		} else {
			// The value of the gap start is relative to the origin of the border
			// container, which is inset by the icon's width. Although, when the
			// floating label is centered, it's already relative to the container.
			offsetToPrefixIcon := 0.0
			if prefixIcon != nil && !d.alignLabelWithHint && r.material3 {
				offsetToPrefixIcon = -boxSizeOf(prefixIcon).Width + padding.Start
			}
			d.borderGap.setStart(foundation.Some(ui.LerpDouble(
				labelX-boxSizeOf(icon).Width+offsetToPrefixIcon,
				boxSizeOf(container).Width/2.0-floatWidth/2.0,
				floatAlign)))
		}
		d.borderGap.setExtent(label.Size().Width * inputDecoratorFinalLabelScale)
	} else {
		d.borderGap.setStart(foundation.None[float64]())
		d.borderGap.setExtent(0.0)
	}
}

func (r *renderDecoration) paintLabel(context *rendering.PaintingContext, offset ui.Offset) {
	context.PaintChild(r.label(), offset)
}

// Paint paints the container, the label (scaled and moved to its floating
// position), the icons, the affixes, the hint, the input and the subtext.
//
// Flutter: material/input_decorator.dart _RenderDecoration.paint
func (r *renderDecoration) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	doPaint := func(child rendering.Box) {
		if child != nil {
			context.PaintChild(child, boxData(child).Offset.Add(offset))
		}
	}
	d := r.decoration
	padding := r.contentPadding()
	container, label, prefixIcon := r.container(), r.label(), r.prefixIcon()
	doPaint(container)
	if label != nil {
		labelOffset := boxData(label).Offset
		labelHeight, labelWidth := boxSizeOf(label).Height, boxSizeOf(label).Width
		// +1 shifts the range of x from (-1.0, 1.0) to (0.0, 2.0).
		floatAlign := d.floatingLabelAlignment.x + 1
		floatWidth := labelWidth * inputDecoratorFinalLabelScale
		borderSide := d.border.BorderSide()
		t := d.floatingLabelProgress
		// The center of the outline border label ends up a little below the center
		// of the top border line.
		isOutlineBorder := d.border.IsOutline()
		// Temporary opt-in fix for flutter/flutter#54028: center the scaled label
		// relative to the border.
		outlinedFloatingY := (-labelHeight*inputDecoratorFinalLabelScale)/2.0 - borderSide.StrokeOffset()/2.0
		floatingY := padding.Top + r.densityOffset().Dy/2
		if isOutlineBorder {
			floatingY = outlinedFloatingY
		}
		scale := ui.LerpDouble(1.0, inputDecoratorFinalLabelScale, t)
		centeredFloatX := boxData(container).Offset.Dx + boxSizeOf(container).Width/2.0 - floatWidth/2.0
		var startX, floatStartX float64
		if r.textDirection == ui.TextDirectionRTL { // the origin is on the right
			startX = labelOffset.Dx + labelWidth*(1.0-scale)
			floatStartX = startX
			if prefixIcon != nil && !d.alignLabelWithHint && isOutlineBorder && r.material3 {
				floatStartX += boxSizeOf(prefixIcon).Width - padding.End
			}
		} else { // the origin is on the left
			startX = labelOffset.Dx
			floatStartX = startX
			if prefixIcon != nil && !d.alignLabelWithHint && isOutlineBorder && r.material3 {
				floatStartX += -boxSizeOf(prefixIcon).Width + padding.Start
			}
		}
		floatEndX := ui.LerpDouble(floatStartX, centeredFloatX, floatAlign)
		dx := ui.LerpDouble(startX, floatEndX, t)
		dy := ui.LerpDouble(0.0, floatingY-labelOffset.Dy, t)
		transform := ui.NewMatrix4Identity()
		transform.Translate(float32(dx), float32(labelOffset.Dy+dy), 0)
		transform.Scale(float32(scale), float32(scale), float32(scale))
		r.labelTransform = transform
		old, _ := r.Layer().(*rendering.TransformLayer)
		if layer := context.PushTransform(r.NeedsCompositing(), offset, transform, r.paintLabel, old); layer != nil {
			r.SetLayer(layer)
		} else {
			r.SetLayer(nil)
		}
	} else {
		r.SetLayer(nil)
	}
	doPaint(r.icon())
	doPaint(r.prefix())
	doPaint(r.suffix())
	doPaint(prefixIcon)
	doPaint(r.suffixIcon())
	if d.isEmpty {
		doPaint(r.hint())
	}
	doPaint(r.input())
	doPaint(r.helperError())
	doPaint(r.counter())
}

// ApplyPaintTransform applies the transform of the label, where it was painted.
//
// Flutter: material/input_decorator.dart _RenderDecoration.applyPaintTransform
func (r *renderDecoration) ApplyPaintTransform(child rendering.RenderObject, transform *ui.Matrix4) {
	if label := r.label(); label != nil && child == rendering.RenderObject(label) && r.labelTransform != nil {
		labelOffset := boxData(label).Offset
		transform.MultiplyBy(r.labelTransform)
		transform.Translate(float32(-labelOffset.Dx), float32(-labelOffset.Dy), 0)
	}
	r.RenderBox.ApplyPaintTransform(child, transform)
}

// HitTestSelf is true: the decorator takes the hits that no part takes.
//
// Flutter: material/input_decorator.dart _RenderDecoration.hitTestSelf
func (r *renderDecoration) HitTestSelf(ui.Offset) bool { return true }

// HitTestChildren hits the children, in order.
//
// Flutter: material/input_decorator.dart _RenderDecoration.hitTestChildren
func (r *renderDecoration) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	for _, child := range r.hitTestOrder() {
		// The label must be handled specially since we've transformed it.
		offset := boxData(child).Offset
		child := child
		if result.AddWithPaintOffset(&offset, position, func(result *rendering.BoxHitTestResult, transformed ui.Offset) bool {
			return child.HitTest(result, transformed)
		}) {
			return true
		}
	}
	return false
}

// childSemanticsConfigurationDelegate merges the prefix, suffix and their icons
// in groups of their own, and the other children up.
//
// Flutter: material/input_decorator.dart _RenderDecoration._childSemanticsConfigurationDelegate
func (r *renderDecoration) childSemanticsConfigurationDelegate(childConfigs []*semantics.SemanticsConfiguration) *semantics.ChildSemanticsConfigurationsResult {
	builder := semantics.NewChildSemanticsConfigurationsResultBuilder()
	var order []semantics.SemanticsTag
	mergeGroups := map[semantics.SemanticsTag][]*semantics.SemanticsConfiguration{}
	tags := []semantics.SemanticsTag{decoratorPrefixSemanticsTag, decoratorPrefixIconSemanticsTag, decoratorSuffixSemanticsTag, decoratorSuffixIconSemanticsTag}
	for _, childConfig := range childConfigs {
		var found *semantics.SemanticsTag
		for i := range tags {
			if childConfig.TagsChildrenWith(tags[i]) {
				found = &tags[i]
				break
			}
		}
		if found != nil {
			if _, ok := mergeGroups[*found]; !ok {
				order = append(order, *found)
			}
			mergeGroups[*found] = append(mergeGroups[*found], childConfig)
		} else {
			builder.MarkAsMergeUp(childConfig)
		}
	}
	for _, tag := range order {
		builder.MarkAsSiblingMergeGroup(mergeGroups[tag])
	}
	return builder.Build()
}

// DescribeSemanticsConfiguration sets the delegate that merges the children.
//
// Flutter: material/input_decorator.dart _RenderDecoration.describeSemanticsConfiguration
func (r *renderDecoration) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderBox.DescribeSemanticsConfiguration(config)
	config.SetChildConfigurationsDelegate(r.childSemanticsConfigurationDelegate)
}

// Dispose releases the transform layer.
func (r *renderDecoration) Dispose() {
	r.SetLayer(nil)
	r.RenderBox.Dispose()
}
