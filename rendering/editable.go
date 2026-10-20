// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"math"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

const (
	// caretGap is the gap between the caret and the end of the text, in pixels.
	//
	// Flutter: rendering/editable.dart _kCaretGap
	caretGap = 1.0
	// caretHeightOffset is how much shorter than the line the caret is, at the
	// top and at the bottom, on the platforms that are not Apple's.
	//
	// Flutter: rendering/editable.dart _kCaretHeightOffset
	caretHeightOffset = 2.0
)

// MaxLinesUnlimited is the value of the MaxLines of RenderEditableOpts that
// means Flutter's maxLines: null, no limit to the number of lines. An unset
// MaxLines is Flutter's default, 1.
//
// Deviation: Flutter's maxLines is a nullable int whose default is 1; an
// Opt cannot tell the default from null (SPEC §12.10).
//
// Flutter: rendering/editable.dart RenderEditable.maxLines
const MaxLinesUnlimited = 0

// TextSelectionPoint is a point at which the text selection originates, with
// the direction of the text there.
//
// Flutter: rendering/editable.dart TextSelectionPoint
type TextSelectionPoint struct {
	// Point is the point of the selection, in the coordinates of the editable.
	//
	// Flutter: rendering/editable.dart TextSelectionPoint.point
	Point ui.Offset
	// Direction is the direction of the text at the point; None when the
	// selection is collapsed and the direction is not known.
	//
	// Flutter: rendering/editable.dart TextSelectionPoint.direction
	Direction foundation.Opt[ui.TextDirection]
}

// String describes the point and the direction.
//
// Flutter: rendering/editable.dart TextSelectionPoint.toString
func (p TextSelectionPoint) String() string {
	d, ok := p.Direction.Get()
	switch {
	case !ok:
		return p.Point.String()
	case d == ui.TextDirectionLTR:
		return p.Point.String() + "-ltr"
	default:
		return p.Point.String() + "-rtl"
	}
}

// VerticalCaretMovementRun is a run of vertical caret movements: it moves a
// caret up and down the lines of an editable, remembering the horizontal
// position where the run started, so that the caret comes back to it after
// passing a short line. It is created by RenderEditable.StartVerticalCaretMovement
// and is invalid once the text layout changes.
//
// Flutter: rendering/editable.dart VerticalCaretMovementRun
type VerticalCaretMovementRun struct {
	editable            *RenderEditable
	lineMetrics         []ui.LineMetrics
	currentTextPosition ui.TextPosition
	currentLine         int
	currentOffset       ui.Offset
	invalid             bool
	positionCache       map[int]positionAtOffset
}

type positionAtOffset struct {
	offset   ui.Offset
	position ui.TextPosition
}

// IsValid reports whether the layout of the text is the one the run started
// on. A run that is not valid can not be used.
//
// Flutter: rendering/editable.dart VerticalCaretMovementRun.isValid
func (r *VerticalCaretMovementRun) IsValid() bool {
	if r.invalid {
		return false
	}
	now := r.editable.textPainter.ComputeLineMetrics()
	if len(now) != len(r.lineMetrics) {
		r.invalid = true
		return false
	}
	for i := range now {
		if now[i] != r.lineMetrics[i] {
			r.invalid = true
			return false
		}
	}
	return true
}

func (r *VerticalCaretMovementRun) checkValid() {
	if foundation.DebugMode && !r.IsValid() {
		panic(errors.New("VerticalCaretMovementRunInvalid", "the VerticalCaretMovementRun is no longer valid: the text layout changed"))
	}
}

func (r *VerticalCaretMovementRun) textPositionForLine(lineNumber int) positionAtOffset {
	r.checkValid()
	if cached, ok := r.positionCache[lineNumber]; ok {
		return cached
	}
	newOffset := ui.Offset{Dx: r.currentOffset.Dx, Dy: r.lineMetrics[lineNumber].Baseline}
	entry := positionAtOffset{offset: newOffset, position: r.editable.textPainter.GetPositionForOffset(newOffset)}
	r.positionCache[lineNumber] = entry
	return entry
}

// Current is the position of the caret in the current line.
//
// Flutter: rendering/editable.dart VerticalCaretMovementRun.current
func (r *VerticalCaretMovementRun) Current() ui.TextPosition {
	r.checkValid()
	return r.currentTextPosition
}

// MoveNext moves the caret to the next line, and returns false if it is on the
// last line.
//
// Flutter: rendering/editable.dart VerticalCaretMovementRun.moveNext
func (r *VerticalCaretMovementRun) MoveNext() bool {
	r.checkValid()
	if r.currentLine+1 >= len(r.lineMetrics) {
		return false
	}
	entry := r.textPositionForLine(r.currentLine + 1)
	r.currentLine++
	r.currentOffset, r.currentTextPosition = entry.offset, entry.position
	return true
}

// MovePrevious moves the caret to the previous line, and returns false if it is
// on the first line.
//
// Flutter: rendering/editable.dart VerticalCaretMovementRun.movePrevious
func (r *VerticalCaretMovementRun) MovePrevious() bool {
	r.checkValid()
	if r.currentLine <= 0 {
		return false
	}
	entry := r.textPositionForLine(r.currentLine - 1)
	r.currentLine--
	r.currentOffset, r.currentTextPosition = entry.offset, entry.position
	return true
}

// MoveByOffset moves the caret by the given vertical offset (down for a
// positive one), as far as the text goes. It returns whether the caret moved.
//
// Flutter: rendering/editable.dart VerticalCaretMovementRun.moveByOffset
func (r *VerticalCaretMovementRun) MoveByOffset(offset float64) bool {
	initial := r.currentOffset
	if offset >= 0 {
		for r.currentOffset.Dy < initial.Dy+offset {
			if !r.MoveNext() {
				break
			}
		}
	} else {
		for r.currentOffset.Dy > initial.Dy+offset {
			if !r.MovePrevious() {
				break
			}
		}
	}
	return initial != r.currentOffset
}

// RenderEditable displays some text that can be edited: it paints the text, the
// selection and the caret, scrolls the text in its box along one axis, and turns
// taps into a selection. The widget EditableText wraps it.
//
// Deviation: the floating cursor of iOS is not ported (SPEC §12.10).
//
// Flutter: rendering/editable.dart RenderEditable
type RenderEditable struct {
	RenderBox
	ContainerRenderObject[Box]
	engine      ui.ParagraphFactory
	gestures    *gestures.GestureBinding
	textPainter *painting.TextPainter

	textIntrinsicsCache *painting.TextPainter
	// placeholderDimensions are the sizes of the inline widgets, kept because
	// the intrinsics overwrite them in the painter used for them.
	placeholderDimensions []painting.PlaceholderDimensions

	showCursor        *foundation.ValueNotifier[bool]
	showCursorHandle  foundation.ListenerHandle
	disposeShowCursor bool
	hasFocus          bool
	forceLine         bool
	readOnly          bool
	maxLines          foundation.Opt[int]
	minLines          foundation.Opt[int]
	expands           bool
	selection         foundation.Opt[services.TextSelection]
	offset            ViewportOffset
	offsetHandle      foundation.ListenerHandle
	cursorWidth       float64
	cursorHeight      foundation.Opt[float64]
	paintCursorOnTop  bool
	devicePixelRatio  float64
	obscuringChar     string
	obscureText       bool
	clipBehavior      ui.Clip
	enableInteractive foundation.Opt[bool]
	maxScrollExtent   float64

	ignorePointer bool

	// textSelectionDelegate is the field that owns the value of the text.
	textSelectionDelegate services.TextSelectionDelegate

	startHandleLayerLink *LayerLink
	endHandleLayerLink   *LayerLink
	leaderLayerHandler   LayerHandle[*LeaderLayer]
	endLeaderLayer       LayerHandle[*LeaderLayer]
	clipRectLayer        LayerHandle[*ClipRectLayer]

	selectionStartInViewport *foundation.ValueNotifier[bool]
	selectionEndInViewport   *foundation.ValueNotifier[bool]

	caretPainter               *caretPainter
	selectionPainter           *textHighlightPainter
	autocorrectHighlightPaint  *textHighlightPainter
	painter                    RenderEditablePainter
	foregroundPainter          RenderEditablePainter
	foregroundRenderObject     *renderEditableCustomPaint
	backgroundRenderObject     *renderEditableCustomPaint
	cachedBuiltInPainters      *compositeRenderEditablePainter
	cachedBuiltInForegroundPnt *compositeRenderEditablePainter

	caretPrototype ui.Rect

	// semantics
	cachedAttributedValue        *semantics.AttributedString
	semanticsInfo                []painting.InlineSpanSemanticsInformation
	cachedCombinedSemanticsInfos []painting.InlineSpanSemanticsInformation
	haveCombinedInfos            bool
	cachedChildNodes             editableSemanticsNodes
	cachedLineBreakCount         foundation.Opt[int]

	tap       *gestures.TapGestureRecognizer
	longPress *gestures.LongPressGestureRecognizer

	lastTapDownPosition          foundation.Opt[ui.Offset]
	lastSecondaryTapDownPosition foundation.Opt[ui.Offset]
}

var _ services.TextLayoutMetrics = (*RenderEditable)(nil)

// RenderEditableOpts are the optional parameters of NewRenderEditable.
//
// Flutter: rendering/editable.dart RenderEditable.new
type RenderEditableOpts struct {
	// TextEngine lays out the text. Deviation: Flutter reaches the engine
	// through dart:ui; the widgets pass the engine of the PlatformDispatcher.
	TextEngine ui.ParagraphFactory
	// Gestures is the gesture binding that the tap and the long press of the
	// editable use. Deviation: Flutter's recognizers use GestureBinding.instance
	// (SPEC §12.10). Without it the editable does not recognize gestures.
	Gestures *gestures.GestureBinding
	// Text is the text to display.
	Text painting.InlineSpan
	// TextAlign defaults to start.
	TextAlign ui.TextAlign
	// CursorColor is the color of the caret.
	CursorColor foundation.Opt[ui.Color]
	// BackgroundCursorColor is the color of the caret when the floating cursor
	// is shown. Deviation: the floating cursor is not ported, so it is kept
	// and never used.
	BackgroundCursorColor foundation.Opt[ui.Color]
	// ShowCursor tells whether to paint the caret; the editable makes one that
	// is false when there is none.
	ShowCursor *foundation.ValueNotifier[bool]
	// HasFocus is whether the editable has the focus.
	HasFocus bool
	// MaxLines is the maximum number of lines: unset is 1 (Flutter's default),
	// MaxLinesUnlimited is no limit (Flutter's null), a positive number is that
	// many lines.
	MaxLines foundation.Opt[int]
	// MinLines is the minimum number of lines; it must be positive and not
	// greater than MaxLines.
	MinLines foundation.Opt[int]
	// Expands makes the editable as tall as its constraints allow; MaxLines and
	// MinLines must then be unlimited and unset.
	Expands bool
	// StrutStyle is the strut of the text.
	StrutStyle *painting.StrutStyle
	// SelectionColor is the color of the selected text's background.
	SelectionColor foundation.Opt[ui.Color]
	// TextScaler defaults to painting.TextScalerNoScaling. Deviation: the
	// deprecated textScaleFactor is not ported.
	TextScaler painting.TextScaler
	// Selection is the selection to paint.
	Selection foundation.Opt[services.TextSelection]
	// IgnorePointer makes the editable ignore taps and long presses.
	IgnorePointer bool
	// ReadOnly makes the text not editable.
	ReadOnly bool
	// DisableForceLine lets the editable be narrower than its constraints
	// when the text is short. Flutter's forceLine is true by default.
	DisableForceLine bool
	// TextHeightBehavior says how the line height applies to the first and last
	// lines.
	TextHeightBehavior *ui.TextHeightBehavior
	// TextWidthBasis defaults to painting.TextWidthBasisParent.
	TextWidthBasis painting.TextWidthBasis
	// ObscuringCharacter is the character that replaces the text when
	// obscured; it must be one character. The default is '•'.
	ObscuringCharacter string
	// ObscureText hides the text.
	ObscureText bool
	// Locale selects region-specific glyphs.
	Locale foundation.Opt[ui.Locale]
	// CursorWidth is the width of the caret; the default is 1.
	CursorWidth foundation.Opt[float64]
	// CursorHeight is the height of the caret; the default is the preferred
	// line height.
	CursorHeight foundation.Opt[float64]
	// CursorRadius rounds the corners of the caret.
	CursorRadius foundation.Opt[ui.Radius]
	// PaintCursorAboveText paints the caret over the text.
	PaintCursorAboveText bool
	// CursorOffset is how far the caret is moved from where the text puts it.
	CursorOffset ui.Offset
	// DevicePixelRatio is used to put the caret on a physical pixel; the
	// default is 1.
	DevicePixelRatio foundation.Opt[float64]
	// SelectionHeightStyle defaults to BoxHeightStyleMax.
	SelectionHeightStyle ui.BoxHeightStyle
	// SelectionWidthStyle defaults to BoxWidthStyleMax.
	SelectionWidthStyle ui.BoxWidthStyle
	// EnableInteractiveSelection says whether the user can select; None is
	// "unless the text is obscured".
	EnableInteractiveSelection foundation.Opt[bool]
	// PromptRectRange is the range of text that is highlighted, for the
	// autocorrect prompt.
	PromptRectRange foundation.Opt[ui.TextRange]
	// PromptRectColor is the color of the highlight of PromptRectRange.
	PromptRectColor foundation.Opt[ui.Color]
	// ClipBehavior defaults to ui.ClipHardEdge.
	ClipBehavior ui.Clip
	// Painter paints below the text, after the built-in painters.
	Painter RenderEditablePainter
	// ForegroundPainter paints above the text, after the built-in foreground
	// painters.
	ForegroundPainter RenderEditablePainter
	// Children are the inline children, in the order of the placeholders.
	Children []Box
}

// NewRenderEditable creates an editable. textDirection, the layer links of the
// selection handles, the offset of the viewport and the text selection delegate
// are required.
//
// Flutter: rendering/editable.dart RenderEditable.new
func NewRenderEditable(textDirection ui.TextDirection, startHandleLayerLink, endHandleLayerLink *LayerLink, offset ViewportOffset,
	textSelectionDelegate services.TextSelectionDelegate, opts ...RenderEditableOpts) *RenderEditable {
	o := firstOpts(opts)
	maxLines := foundation.Some(1)
	switch {
	case o.MaxLines.IsSome() && o.MaxLines.Value() == MaxLinesUnlimited:
		maxLines = foundation.None[int]()
	case o.MaxLines.IsSome():
		maxLines = o.MaxLines
	}
	cursorWidth := o.CursorWidth.OrElse(1)
	if foundation.DebugMode {
		if m, ok := maxLines.Get(); ok && m <= 0 {
			panic(errors.New("EditableMaxLinesInvalid", "maxLines must be positive", "maxLines", m))
		}
		if m, ok := o.MinLines.Get(); ok && m <= 0 {
			panic(errors.New("EditableMinLinesInvalid", "minLines must be positive", "minLines", m))
		}
		if mx, ok := maxLines.Get(); ok && o.MinLines.IsSome() && mx < o.MinLines.Value() {
			panic(errors.New("EditableMinLinesGreaterThanMaxLines", "minLines can't be greater than maxLines",
				"minLines", o.MinLines.Value(), "maxLines", mx))
		}
		if o.Expands && (maxLines.IsSome() || o.MinLines.IsSome()) {
			panic(errors.New("EditableExpandsWithLines", "minLines and maxLines must be unset when expands is true",
				"minLines", o.MinLines, "maxLines", maxLines))
		}
		if o.ObscuringCharacter != "" && foundation.GraphemeCount(o.ObscuringCharacter) != 1 {
			panic(errors.New("EditableObscuringCharacterInvalid", "the obscuring character must be one character",
				"obscuringCharacter", o.ObscuringCharacter))
		}
		if cursorWidth < 0 {
			panic(errors.New("EditableCursorWidthInvalid", "the cursor width can't be negative", "cursorWidth", cursorWidth))
		}
		if h, ok := o.CursorHeight.Get(); ok && h < 0 {
			panic(errors.New("EditableCursorHeightInvalid", "the cursor height can't be negative", "cursorHeight", h))
		}
	}
	obscuring := o.ObscuringCharacter
	if obscuring == "" {
		obscuring = "•"
	}
	clip := o.ClipBehavior
	if clip == ui.ClipUnspecified {
		clip = ui.ClipHardEdge
	}
	painterMaxLines := foundation.None[int]()
	if maxLines.IsSome() && maxLines.Value() == 1 {
		painterMaxLines = foundation.Some(1)
	}

	r := &RenderEditable{
		engine:   o.TextEngine,
		gestures: o.Gestures,
		textPainter: painting.NewTextPainter(painting.TextPainterOpts{
			TextEngine: o.TextEngine, Text: o.Text, TextAlign: o.TextAlign, TextDirection: textDirection,
			TextScaler: o.TextScaler, Locale: o.Locale, MaxLines: painterMaxLines, StrutStyle: o.StrutStyle,
			TextHeightBehavior: o.TextHeightBehavior, TextWidthBasis: o.TextWidthBasis,
		}),
		showCursor:                o.ShowCursor,
		disposeShowCursor:         o.ShowCursor == nil,
		hasFocus:                  o.HasFocus,
		forceLine:                 !o.DisableForceLine,
		readOnly:                  o.ReadOnly,
		maxLines:                  maxLines,
		minLines:                  o.MinLines,
		expands:                   o.Expands,
		selection:                 o.Selection,
		offset:                    offset,
		cursorWidth:               cursorWidth,
		cursorHeight:              o.CursorHeight,
		paintCursorOnTop:          o.PaintCursorAboveText,
		devicePixelRatio:          o.DevicePixelRatio.OrElse(1),
		obscuringChar:             obscuring,
		obscureText:               o.ObscureText,
		clipBehavior:              clip,
		enableInteractive:         o.EnableInteractiveSelection,
		ignorePointer:             o.IgnorePointer,
		textSelectionDelegate:     textSelectionDelegate,
		startHandleLayerLink:      startHandleLayerLink,
		endHandleLayerLink:        endHandleLayerLink,
		selectionStartInViewport:  foundation.NewValueNotifier(true),
		selectionEndInViewport:    foundation.NewValueNotifier(true),
		caretPainter:              newCaretPainter(),
		selectionPainter:          newTextHighlightPainter(),
		autocorrectHighlightPaint: newTextHighlightPainter(),
	}
	if r.showCursor == nil {
		r.showCursor = foundation.NewValueNotifier(false)
	}
	InitRenderBox(&r.RenderBox, r)
	InitContainerRenderObject(&r.ContainerRenderObject, r)

	if foundation.DebugMode && r.showCursor.Value() && !o.CursorColor.IsSome() {
		panic(errors.New("EditableCursorColorMissing", "the cursor color is needed to show the cursor"))
	}

	heightStyle, widthStyle := o.SelectionHeightStyle, o.SelectionWidthStyle
	if heightStyle == ui.BoxHeightStyleUnspecified {
		heightStyle = ui.BoxHeightStyleMax
	}
	if widthStyle == ui.BoxWidthStyleUnspecified {
		widthStyle = ui.BoxWidthStyleMax
	}
	r.selectionPainter.setHighlightColor(o.SelectionColor)
	if sel, ok := o.Selection.Get(); ok {
		r.selectionPainter.setHighlightedRange(foundation.Some(sel.Range()))
	}
	r.selectionPainter.setSelectionHeightStyle(heightStyle)
	r.selectionPainter.setSelectionWidthStyle(widthStyle)

	r.autocorrectHighlightPaint.setHighlightColor(o.PromptRectColor)
	r.autocorrectHighlightPaint.setHighlightedRange(o.PromptRectRange)

	r.caretPainter.setCaretColor(o.CursorColor)
	r.caretPainter.setCursorRadius(o.CursorRadius)
	r.caretPainter.setCursorOffset(o.CursorOffset)
	r.caretPainter.setBackgroundCursorColor(o.BackgroundCursorColor)

	r.updateForegroundPainter(o.ForegroundPainter)
	r.updatePainter(o.Painter)
	r.AddAll(o.Children)
	return r
}

// SetupParentData gives the child a TextParentData if it has none.
//
// Flutter: rendering/paragraph.dart RenderInlineChildrenContainerDefaults.setupParentData
func (r *RenderEditable) SetupParentData(child RenderObject) {
	if child == RenderObject(r.foregroundRenderObject) || child == RenderObject(r.backgroundRenderObject) {
		r.RenderBox.SetupParentData(child)
		return
	}
	SetupInlineChildParentData(child)
}

// Dispose releases the painters, the text painters and the layers.
//
// Flutter: rendering/editable.dart RenderEditable.dispose
func (r *RenderEditable) Dispose() {
	r.leaderLayerHandler.SetLayer(nil)
	r.endLeaderLayer.SetLayer(nil)
	if r.foregroundRenderObject != nil {
		r.foregroundRenderObject.Dispose()
		r.foregroundRenderObject = nil
	}
	if r.backgroundRenderObject != nil {
		r.backgroundRenderObject.Dispose()
		r.backgroundRenderObject = nil
	}
	r.clipRectLayer.SetLayer(nil)
	r.selectionStartInViewport.Dispose()
	r.selectionEndInViewport.Dispose()
	r.autocorrectHighlightPaint.Dispose()
	r.selectionPainter.Dispose()
	r.caretPainter.Dispose()
	r.textPainter.Dispose()
	if r.textIntrinsicsCache != nil {
		r.textIntrinsicsCache.Dispose()
	}
	if r.disposeShowCursor {
		r.showCursor.Dispose()
		r.disposeShowCursor = false
	}
	r.RenderBox.Dispose()
}

// SystemFontsDidChange lays the editable out again, because the fonts of the
// system changed.
//
// Flutter: rendering/object.dart RelayoutWhenSystemFontsChangeMixin.systemFontsDidChange
func (r *RenderEditable) SystemFontsDidChange() {
	r.MarkNeedsLayout()
	r.textPainter.MarkNeedsLayout()
}

func (r *RenderEditable) updateForegroundPainter(newPainter RenderEditablePainter) {
	effective := RenderEditablePainter(r.builtInForegroundPainters())
	if newPainter != nil {
		effective = newCompositeRenderEditablePainter(r.builtInForegroundPainters(), newPainter)
	}
	if r.foregroundRenderObject == nil {
		r.foregroundRenderObject = newRenderEditableCustomPaint(effective)
		r.AdoptChild(r.foregroundRenderObject)
	} else {
		r.foregroundRenderObject.setPainter(effective)
	}
	r.foregroundPainter = newPainter
}

func (r *RenderEditable) updatePainter(newPainter RenderEditablePainter) {
	effective := RenderEditablePainter(r.builtInPainters())
	if newPainter != nil {
		effective = newCompositeRenderEditablePainter(r.builtInPainters(), newPainter)
	}
	if r.backgroundRenderObject == nil {
		r.backgroundRenderObject = newRenderEditableCustomPaint(effective)
		r.AdoptChild(r.backgroundRenderObject)
	} else {
		r.backgroundRenderObject.setPainter(effective)
	}
	r.painter = newPainter
}

func (r *RenderEditable) builtInForegroundPainters() *compositeRenderEditablePainter {
	if r.cachedBuiltInForegroundPnt == nil {
		var painters []RenderEditablePainter
		if r.paintCursorOnTop {
			painters = append(painters, r.caretPainter)
		}
		r.cachedBuiltInForegroundPnt = newCompositeRenderEditablePainter(painters...)
	}
	return r.cachedBuiltInForegroundPnt
}

func (r *RenderEditable) builtInPainters() *compositeRenderEditablePainter {
	if r.cachedBuiltInPainters == nil {
		painters := []RenderEditablePainter{r.autocorrectHighlightPaint, r.selectionPainter}
		if !r.paintCursorOnTop {
			painters = append(painters, r.caretPainter)
		}
		r.cachedBuiltInPainters = newCompositeRenderEditablePainter(painters...)
	}
	return r.cachedBuiltInPainters
}

// ForegroundPainter is the painter that paints above the text.
//
// Flutter: rendering/editable.dart RenderEditable.foregroundPainter
func (r *RenderEditable) ForegroundPainter() RenderEditablePainter { return r.foregroundPainter }

// SetForegroundPainter changes the painter that paints above the text.
//
// Flutter: rendering/editable.dart RenderEditable.foregroundPainter
func (r *RenderEditable) SetForegroundPainter(p RenderEditablePainter) {
	if p == r.foregroundPainter {
		return
	}
	r.updateForegroundPainter(p)
}

// Painter is the painter that paints below the text.
//
// Flutter: rendering/editable.dart RenderEditable.painter
func (r *RenderEditable) Painter() RenderEditablePainter { return r.painter }

// SetPainter changes the painter that paints below the text.
//
// Flutter: rendering/editable.dart RenderEditable.painter
func (r *RenderEditable) SetPainter(p RenderEditablePainter) {
	if p == r.painter {
		return
	}
	r.updatePainter(p)
}

// IgnorePointer is whether the editable ignores taps and long presses.
//
// Flutter: rendering/editable.dart RenderEditable.ignorePointer
func (r *RenderEditable) IgnorePointer() bool { return r.ignorePointer }

// SetIgnorePointer changes whether taps and long presses are ignored.
//
// Flutter: rendering/editable.dart RenderEditable.ignorePointer
func (r *RenderEditable) SetIgnorePointer(value bool) { r.ignorePointer = value }

// TextHeightBehavior says how the line height applies to the first and last
// lines.
//
// Flutter: rendering/editable.dart RenderEditable.textHeightBehavior
func (r *RenderEditable) TextHeightBehavior() *ui.TextHeightBehavior {
	return r.textPainter.TextHeightBehavior()
}

// SetTextHeightBehavior changes the behavior of the line height.
//
// Flutter: rendering/editable.dart RenderEditable.textHeightBehavior
func (r *RenderEditable) SetTextHeightBehavior(value *ui.TextHeightBehavior) {
	if r.textPainter.TextHeightBehavior() == value {
		return
	}
	r.textPainter.SetTextHeightBehavior(value)
	r.MarkNeedsLayout()
}

// TextWidthBasis is how the width of the text is measured.
//
// Flutter: rendering/editable.dart RenderEditable.textWidthBasis
func (r *RenderEditable) TextWidthBasis() painting.TextWidthBasis {
	return r.textPainter.TextWidthBasis()
}

// SetTextWidthBasis changes how the width is measured.
//
// Flutter: rendering/editable.dart RenderEditable.textWidthBasis
func (r *RenderEditable) SetTextWidthBasis(value painting.TextWidthBasis) {
	if r.textPainter.TextWidthBasis() == value {
		return
	}
	r.textPainter.SetTextWidthBasis(value)
	r.MarkNeedsLayout()
}

// DevicePixelRatio is the ratio of device pixels to logical pixels, used to put
// the caret on a physical pixel.
//
// Flutter: rendering/editable.dart RenderEditable.devicePixelRatio
func (r *RenderEditable) DevicePixelRatio() float64 { return r.devicePixelRatio }

// SetDevicePixelRatio changes the pixel ratio.
//
// Flutter: rendering/editable.dart RenderEditable.devicePixelRatio
func (r *RenderEditable) SetDevicePixelRatio(value float64) {
	if r.devicePixelRatio == value {
		return
	}
	r.devicePixelRatio = value
	r.MarkNeedsLayout()
}

// ObscuringCharacter is the character that replaces the text when it is
// obscured.
//
// Flutter: rendering/editable.dart RenderEditable.obscuringCharacter
func (r *RenderEditable) ObscuringCharacter() string { return r.obscuringChar }

// SetObscuringCharacter changes the obscuring character, which must be one
// character.
//
// Flutter: rendering/editable.dart RenderEditable.obscuringCharacter
func (r *RenderEditable) SetObscuringCharacter(value string) {
	if r.obscuringChar == value {
		return
	}
	if foundation.DebugMode && foundation.GraphemeCount(value) != 1 {
		panic(errors.New("EditableObscuringCharacterInvalid", "the obscuring character must be one character", "obscuringCharacter", value))
	}
	r.obscuringChar = value
	r.MarkNeedsLayout()
}

// ObscureText is whether the text is hidden.
//
// Flutter: rendering/editable.dart RenderEditable.obscureText
func (r *RenderEditable) ObscureText() bool { return r.obscureText }

// SetObscureText changes whether the text is hidden.
//
// Flutter: rendering/editable.dart RenderEditable.obscureText
func (r *RenderEditable) SetObscureText(value bool) {
	if r.obscureText == value {
		return
	}
	r.obscureText = value
	r.cachedAttributedValue = nil
	r.MarkNeedsSemanticsUpdate()
}

// SelectionHeightStyle controls how tall the selection highlight boxes are.
//
// Flutter: rendering/editable.dart RenderEditable.selectionHeightStyle
func (r *RenderEditable) SelectionHeightStyle() ui.BoxHeightStyle {
	return r.selectionPainter.selectionHeightStyle
}

// SetSelectionHeightStyle changes the height style of the selection boxes.
//
// Flutter: rendering/editable.dart RenderEditable.selectionHeightStyle
func (r *RenderEditable) SetSelectionHeightStyle(value ui.BoxHeightStyle) {
	r.selectionPainter.setSelectionHeightStyle(value)
}

// SelectionWidthStyle controls how wide the selection highlight boxes are.
//
// Flutter: rendering/editable.dart RenderEditable.selectionWidthStyle
func (r *RenderEditable) SelectionWidthStyle() ui.BoxWidthStyle {
	return r.selectionPainter.selectionWidthStyle
}

// SetSelectionWidthStyle changes the width style of the selection boxes.
//
// Flutter: rendering/editable.dart RenderEditable.selectionWidthStyle
func (r *RenderEditable) SetSelectionWidthStyle(value ui.BoxWidthStyle) {
	r.selectionPainter.setSelectionWidthStyle(value)
}

// TextSelectionDelegate is the field that owns the value of the text.
//
// Flutter: rendering/editable.dart RenderEditable.textSelectionDelegate
func (r *RenderEditable) TextSelectionDelegate() services.TextSelectionDelegate {
	return r.textSelectionDelegate
}

// SetTextSelectionDelegate changes the field that owns the value of the text.
//
// Flutter: rendering/editable.dart RenderEditable.textSelectionDelegate
func (r *RenderEditable) SetTextSelectionDelegate(value services.TextSelectionDelegate) {
	r.textSelectionDelegate = value
}

// SelectionStartInViewport is true while the start of the selection is visible.
//
// Flutter: rendering/editable.dart RenderEditable.selectionStartInViewport
func (r *RenderEditable) SelectionStartInViewport() foundation.ValueListenable[bool] {
	return r.selectionStartInViewport
}

// SelectionEndInViewport is true while the end of the selection is visible.
//
// Flutter: rendering/editable.dart RenderEditable.selectionEndInViewport
func (r *RenderEditable) SelectionEndInViewport() foundation.ValueListenable[bool] {
	return r.selectionEndInViewport
}

func (r *RenderEditable) textPositionVertical(position ui.TextPosition, verticalOffset float64) ui.TextPosition {
	caretOffset := r.textPainter.GetOffsetForCaret(position, r.caretPrototype)
	return r.textPainter.GetPositionForOffset(caretOffset.Translate(0, verticalOffset))
}

// GetLineAtOffset returns the line that the position is on, as a selection. When
// the text is obscured the whole text is one line.
//
// Flutter: rendering/editable.dart RenderEditable.getLineAtOffset
func (r *RenderEditable) GetLineAtOffset(position ui.TextPosition) services.TextSelection {
	line := r.textPainter.GetLineBoundary(position)
	if r.obscureText {
		return services.NewTextSelection(0, len(r.PlainText()))
	}
	return services.NewTextSelection(line.Start, line.End)
}

// GetWordBoundary returns the word that the position is in.
//
// Flutter: rendering/editable.dart RenderEditable.getWordBoundary
func (r *RenderEditable) GetWordBoundary(position ui.TextPosition) ui.TextRange {
	return r.textPainter.GetWordBoundary(position)
}

// GetTextPositionAbove returns the position on the line above, at the same
// horizontal offset.
//
// Flutter: rendering/editable.dart RenderEditable.getTextPositionAbove
func (r *RenderEditable) GetTextPositionAbove(position ui.TextPosition) ui.TextPosition {
	// The caret offset is the upper left corner of the caret: the middle of the
	// line above is half a line above it.
	return r.textPositionVertical(position, -0.5*r.textPainter.PreferredLineHeight())
}

// GetTextPositionBelow returns the position on the line below, at the same
// horizontal offset.
//
// Flutter: rendering/editable.dart RenderEditable.getTextPositionBelow
func (r *RenderEditable) GetTextPositionBelow(position ui.TextPosition) ui.TextPosition {
	return r.textPositionVertical(position, 1.5*r.textPainter.PreferredLineHeight())
}

func (r *RenderEditable) updateSelectionExtentsVisibility(effectiveOffset ui.Offset) {
	sel, ok := r.selection.Get()
	if !ok || !sel.IsValid() {
		r.selectionStartInViewport.SetValue(false)
		r.selectionEndInViewport.SetValue(false)
		return
	}
	visibleRegion := ui.OffsetZero.And(r.Size())
	startOffset := r.textPainter.GetOffsetForCaret(ui.TextPosition{Offset: sel.Start(), Affinity: sel.Affinity}, r.caretPrototype)
	// The check is approximate: the rounding of the layout can put the caret
	// slightly (less than half a pixel) outside.
	const visibleRegionSlop = 0.5
	r.selectionStartInViewport.SetValue(visibleRegion.Inflate(visibleRegionSlop).Contains(startOffset.Add(effectiveOffset)))
	endOffset := r.textPainter.GetOffsetForCaret(ui.TextPosition{Offset: sel.End(), Affinity: sel.Affinity}, r.caretPrototype)
	r.selectionEndInViewport.SetValue(visibleRegion.Inflate(visibleRegionSlop).Contains(endOffset.Add(effectiveOffset)))
}

func (r *RenderEditable) setTextEditingValue(newValue services.TextEditingValue, cause services.SelectionChangedCause) {
	r.textSelectionDelegate.UserUpdateTextEditingValue(newValue, cause)
}

func (r *RenderEditable) setSelection(nextSelection services.TextSelection, cause services.SelectionChangedCause) {
	if nextSelection.IsValid() {
		// The selection was worked out from the text this editable has, which
		// can be a frame behind the value of the field.
		textLength := len(r.textSelectionDelegate.TextEditingValue().Text)
		nextSelection = nextSelection.CopyWith(services.TextSelectionCopy{
			BaseOffset:   foundation.Some(min(nextSelection.BaseOffset, textLength)),
			ExtentOffset: foundation.Some(min(nextSelection.ExtentOffset, textLength)),
		})
	}
	r.setTextEditingValue(r.textSelectionDelegate.TextEditingValue().CopyWith(services.TextEditingValueCopy{
		Selection: foundation.Some(nextSelection),
	}), cause)
}

// MarkNeedsPaint also asks the painters to paint again, because the layout of
// the text may have changed.
//
// Flutter: rendering/editable.dart RenderEditable.markNeedsPaint
func (r *RenderEditable) MarkNeedsPaint() {
	r.RenderBox.MarkNeedsPaint()
	if r.foregroundRenderObject != nil {
		r.foregroundRenderObject.MarkNeedsPaint()
	}
	if r.backgroundRenderObject != nil {
		r.backgroundRenderObject.MarkNeedsPaint()
	}
}

// PlainText is the text without the styles, as a string.
//
// Flutter: rendering/editable.dart RenderEditable.plainText
func (r *RenderEditable) PlainText() string { return r.textPainter.PlainText() }

// Text is the text to display.
//
// Flutter: rendering/editable.dart RenderEditable.text
func (r *RenderEditable) Text() painting.InlineSpan { return r.textPainter.Text() }

// SetText changes the text.
//
// Flutter: rendering/editable.dart RenderEditable.text
func (r *RenderEditable) SetText(value painting.InlineSpan) {
	if r.textPainter.Text() == value {
		return
	}
	r.cachedLineBreakCount = foundation.None[int]()
	r.textPainter.SetText(value)
	r.cachedAttributedValue = nil
	r.haveCombinedInfos = false
	r.MarkNeedsLayout()
	r.MarkNeedsSemanticsUpdate()
}

func (r *RenderEditable) textIntrinsics() *painting.TextPainter {
	if r.textIntrinsicsCache == nil {
		r.textIntrinsicsCache = painting.NewTextPainter(painting.TextPainterOpts{TextEngine: r.engine})
	}
	p, q := r.textIntrinsicsCache, r.textPainter
	p.SetText(q.Text())
	p.SetTextAlign(q.TextAlign())
	p.SetTextDirection(q.TextDirection())
	p.SetTextScaler(q.TextScaler())
	p.SetMaxLines(q.MaxLines())
	p.SetEllipsis(q.Ellipsis())
	p.SetLocale(q.Locale())
	p.SetStrutStyle(q.StrutStyle())
	p.SetTextWidthBasis(q.TextWidthBasis())
	p.SetTextHeightBehavior(q.TextHeightBehavior())
	return p
}

// TextAlign is how the text is aligned horizontally.
//
// Flutter: rendering/editable.dart RenderEditable.textAlign
func (r *RenderEditable) TextAlign() ui.TextAlign { return r.textPainter.TextAlign() }

// SetTextAlign changes how the text is aligned.
//
// Flutter: rendering/editable.dart RenderEditable.textAlign
func (r *RenderEditable) SetTextAlign(value ui.TextAlign) {
	if r.textPainter.TextAlign() == value {
		return
	}
	r.textPainter.SetTextAlign(value)
	r.MarkNeedsLayout()
}

// TextDirection is the default directionality of the text.
//
// Flutter: rendering/editable.dart RenderEditable.textDirection
func (r *RenderEditable) TextDirection() ui.TextDirection { return r.textPainter.TextDirection() }

// SetTextDirection changes the default directionality.
//
// Flutter: rendering/editable.dart RenderEditable.textDirection
func (r *RenderEditable) SetTextDirection(value ui.TextDirection) {
	if r.textPainter.TextDirection() == value {
		return
	}
	r.textPainter.SetTextDirection(value)
	r.MarkNeedsLayout()
	r.MarkNeedsSemanticsUpdate()
}

// Locale selects region-specific glyphs.
//
// Flutter: rendering/editable.dart RenderEditable.locale
func (r *RenderEditable) Locale() foundation.Opt[ui.Locale] { return r.textPainter.Locale() }

// SetLocale changes the locale.
//
// Flutter: rendering/editable.dart RenderEditable.locale
func (r *RenderEditable) SetLocale(value foundation.Opt[ui.Locale]) {
	if r.textPainter.Locale() == value {
		return
	}
	r.textPainter.SetLocale(value)
	r.MarkNeedsLayout()
}

// StrutStyle is the strut of the text.
//
// Flutter: rendering/editable.dart RenderEditable.strutStyle
func (r *RenderEditable) StrutStyle() *painting.StrutStyle { return r.textPainter.StrutStyle() }

// SetStrutStyle changes the strut.
//
// Flutter: rendering/editable.dart RenderEditable.strutStyle
func (r *RenderEditable) SetStrutStyle(value *painting.StrutStyle) {
	if r.textPainter.StrutStyle() == value {
		return
	}
	r.textPainter.SetStrutStyle(value)
	r.MarkNeedsLayout()
}

// CursorColor is the color of the caret.
//
// Flutter: rendering/editable.dart RenderEditable.cursorColor
func (r *RenderEditable) CursorColor() foundation.Opt[ui.Color] { return r.caretPainter.caretColor }

// SetCursorColor changes the color of the caret.
//
// Flutter: rendering/editable.dart RenderEditable.cursorColor
func (r *RenderEditable) SetCursorColor(value foundation.Opt[ui.Color]) {
	r.caretPainter.setCaretColor(value)
}

// BackgroundCursorColor is the color of the caret while the floating cursor is
// shown (never, in Glutter).
//
// Flutter: rendering/editable.dart RenderEditable.backgroundCursorColor
func (r *RenderEditable) BackgroundCursorColor() foundation.Opt[ui.Color] {
	return r.caretPainter.backgroundCursorColor
}

// SetBackgroundCursorColor changes the color of the background cursor.
//
// Flutter: rendering/editable.dart RenderEditable.backgroundCursorColor
func (r *RenderEditable) SetBackgroundCursorColor(value foundation.Opt[ui.Color]) {
	r.caretPainter.setBackgroundCursorColor(value)
}

// ShowCursor says whether to paint the caret.
//
// Flutter: rendering/editable.dart RenderEditable.showCursor
func (r *RenderEditable) ShowCursor() *foundation.ValueNotifier[bool] { return r.showCursor }

// SetShowCursor changes the notifier that says whether to paint the caret.
//
// Flutter: rendering/editable.dart RenderEditable.showCursor
func (r *RenderEditable) SetShowCursor(value *foundation.ValueNotifier[bool]) {
	if r.showCursor == value {
		return
	}
	if r.Attached() {
		r.showCursor.RemoveListener(r.showCursorHandle)
	}
	if r.disposeShowCursor {
		r.showCursor.Dispose()
		r.disposeShowCursor = false
	}
	r.showCursor = value
	if r.Attached() {
		r.showHideCursor()
		r.showCursorHandle = r.showCursor.AddListener(r.showHideCursor)
	}
}

func (r *RenderEditable) showHideCursor() { r.caretPainter.setShouldPaint(r.showCursor.Value()) }

// HasFocus is whether the editable has the focus.
//
// Flutter: rendering/editable.dart RenderEditable.hasFocus
func (r *RenderEditable) HasFocus() bool { return r.hasFocus }

// SetHasFocus changes whether the editable has the focus.
//
// Flutter: rendering/editable.dart RenderEditable.hasFocus
func (r *RenderEditable) SetHasFocus(value bool) {
	if r.hasFocus == value {
		return
	}
	r.hasFocus = value
	r.MarkNeedsSemanticsUpdate()
}

// ForceLine is whether the editable is as wide as its constraints allow.
//
// Flutter: rendering/editable.dart RenderEditable.forceLine
func (r *RenderEditable) ForceLine() bool { return r.forceLine }

// SetForceLine changes whether the editable is as wide as it can be.
//
// Flutter: rendering/editable.dart RenderEditable.forceLine
func (r *RenderEditable) SetForceLine(value bool) {
	if r.forceLine == value {
		return
	}
	r.forceLine = value
	r.MarkNeedsLayout()
}

// ReadOnly is whether the text can not be edited.
//
// Flutter: rendering/editable.dart RenderEditable.readOnly
func (r *RenderEditable) ReadOnly() bool { return r.readOnly }

// SetReadOnly changes whether the text can be edited.
//
// Flutter: rendering/editable.dart RenderEditable.readOnly
func (r *RenderEditable) SetReadOnly(value bool) {
	if r.readOnly == value {
		return
	}
	r.readOnly = value
	r.MarkNeedsSemanticsUpdate()
}

// MaxLines is the maximum number of lines; None is no limit.
//
// Flutter: rendering/editable.dart RenderEditable.maxLines
func (r *RenderEditable) MaxLines() foundation.Opt[int] { return r.maxLines }

// SetMaxLines changes the maximum number of lines; None is no limit (Flutter's
// null).
//
// Flutter: rendering/editable.dart RenderEditable.maxLines
func (r *RenderEditable) SetMaxLines(value foundation.Opt[int]) {
	if foundation.DebugMode && value.IsSome() && value.Value() <= 0 {
		panic(errors.New("EditableMaxLinesInvalid", "maxLines must be positive", "maxLines", value.Value()))
	}
	if r.maxLines == value {
		return
	}
	r.maxLines = value
	// maxLines == 1 keeps only the first line, so that the height of the first
	// line is known if the text has hard line breaks.
	if value.IsSome() && value.Value() == 1 {
		r.textPainter.SetMaxLines(foundation.Some(1))
	} else {
		r.textPainter.SetMaxLines(foundation.None[int]())
	}
	r.MarkNeedsLayout()
}

// MinLines is the minimum number of lines.
//
// Flutter: rendering/editable.dart RenderEditable.minLines
func (r *RenderEditable) MinLines() foundation.Opt[int] { return r.minLines }

// SetMinLines changes the minimum number of lines.
//
// Flutter: rendering/editable.dart RenderEditable.minLines
func (r *RenderEditable) SetMinLines(value foundation.Opt[int]) {
	if foundation.DebugMode && value.IsSome() && value.Value() <= 0 {
		panic(errors.New("EditableMinLinesInvalid", "minLines must be positive", "minLines", value.Value()))
	}
	if r.minLines == value {
		return
	}
	r.minLines = value
	r.MarkNeedsLayout()
}

// Expands is whether the editable is as tall as its constraints allow.
//
// Flutter: rendering/editable.dart RenderEditable.expands
func (r *RenderEditable) Expands() bool { return r.expands }

// SetExpands changes whether the editable fills its constraints.
//
// Flutter: rendering/editable.dart RenderEditable.expands
func (r *RenderEditable) SetExpands(value bool) {
	if r.expands == value {
		return
	}
	r.expands = value
	r.MarkNeedsLayout()
}

// SelectionColor is the color of the selected text's background.
//
// Flutter: rendering/editable.dart RenderEditable.selectionColor
func (r *RenderEditable) SelectionColor() foundation.Opt[ui.Color] {
	return r.selectionPainter.highlightColor
}

// SetSelectionColor changes the color of the selection.
//
// Flutter: rendering/editable.dart RenderEditable.selectionColor
func (r *RenderEditable) SetSelectionColor(value foundation.Opt[ui.Color]) {
	r.selectionPainter.setHighlightColor(value)
}

// TextScaler scales the font sizes.
//
// Flutter: rendering/editable.dart RenderEditable.textScaler
func (r *RenderEditable) TextScaler() painting.TextScaler { return r.textPainter.TextScaler() }

// SetTextScaler changes how the font sizes are scaled.
//
// Flutter: rendering/editable.dart RenderEditable.textScaler
func (r *RenderEditable) SetTextScaler(value painting.TextScaler) {
	if r.textPainter.TextScaler() == value {
		return
	}
	r.textPainter.SetTextScaler(value)
	r.MarkNeedsLayout()
}

// Selection is the selection to paint; None for no selection.
//
// Flutter: rendering/editable.dart RenderEditable.selection
func (r *RenderEditable) Selection() foundation.Opt[services.TextSelection] { return r.selection }

// SetSelection changes the selection to paint.
//
// Flutter: rendering/editable.dart RenderEditable.selection
func (r *RenderEditable) SetSelection(value foundation.Opt[services.TextSelection]) {
	if sameSelection(r.selection, value) {
		return
	}
	r.selection = value
	if sel, ok := value.Get(); ok {
		r.selectionPainter.setHighlightedRange(foundation.Some(sel.Range()))
	} else {
		r.selectionPainter.setHighlightedRange(foundation.None[ui.TextRange]())
	}
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

func sameSelection(a, b foundation.Opt[services.TextSelection]) bool {
	x, okA := a.Get()
	y, okB := b.Get()
	if okA != okB {
		return false
	}
	return !okA || x.Equal(y)
}

// Offset is the offset of the viewport: how far the text is scrolled.
//
// Flutter: rendering/editable.dart RenderEditable.offset
func (r *RenderEditable) Offset() ViewportOffset { return r.offset }

// SetOffset changes the offset of the viewport.
//
// Flutter: rendering/editable.dart RenderEditable.offset
func (r *RenderEditable) SetOffset(value ViewportOffset) {
	if r.offset == value {
		return
	}
	if r.Attached() {
		r.offset.RemoveListener(r.offsetHandle)
	}
	r.offset = value
	if r.Attached() {
		r.offsetHandle = r.offset.AddListener(r.MarkNeedsPaint)
	}
	r.MarkNeedsLayout()
}

// CursorWidth is the width of the caret.
//
// Flutter: rendering/editable.dart RenderEditable.cursorWidth
func (r *RenderEditable) CursorWidth() float64 { return r.cursorWidth }

// SetCursorWidth changes the width of the caret.
//
// Flutter: rendering/editable.dart RenderEditable.cursorWidth
func (r *RenderEditable) SetCursorWidth(value float64) {
	if r.cursorWidth == value {
		return
	}
	r.cursorWidth = value
	r.MarkNeedsLayout()
}

// CursorHeight is the height of the caret: by default the preferred line height
// of the text.
//
// Flutter: rendering/editable.dart RenderEditable.cursorHeight
func (r *RenderEditable) CursorHeight() float64 {
	return r.cursorHeight.OrElse(r.PreferredLineHeight())
}

// SetCursorHeight changes the height of the caret; None is the preferred line
// height.
//
// Flutter: rendering/editable.dart RenderEditable.cursorHeight
func (r *RenderEditable) SetCursorHeight(value foundation.Opt[float64]) {
	if r.cursorHeight == value {
		return
	}
	r.cursorHeight = value
	r.MarkNeedsLayout()
}

// PaintCursorAboveText is whether the caret is painted over the text.
//
// Flutter: rendering/editable.dart RenderEditable.paintCursorAboveText
func (r *RenderEditable) PaintCursorAboveText() bool { return r.paintCursorOnTop }

// SetPaintCursorAboveText changes whether the caret is painted over the text.
//
// Flutter: rendering/editable.dart RenderEditable.paintCursorAboveText
func (r *RenderEditable) SetPaintCursorAboveText(value bool) {
	if r.paintCursorOnTop == value {
		return
	}
	r.paintCursorOnTop = value
	r.cachedBuiltInForegroundPnt = nil
	r.cachedBuiltInPainters = nil
	r.updateForegroundPainter(r.foregroundPainter)
	r.updatePainter(r.painter)
}

// CursorOffset is how far the caret is moved from where the text puts it.
//
// Flutter: rendering/editable.dart RenderEditable.cursorOffset
func (r *RenderEditable) CursorOffset() ui.Offset { return r.caretPainter.cursorOffset }

// SetCursorOffset changes the offset of the caret.
//
// Flutter: rendering/editable.dart RenderEditable.cursorOffset
func (r *RenderEditable) SetCursorOffset(value ui.Offset) { r.caretPainter.setCursorOffset(value) }

// CursorRadius rounds the corners of the caret.
//
// Flutter: rendering/editable.dart RenderEditable.cursorRadius
func (r *RenderEditable) CursorRadius() foundation.Opt[ui.Radius] { return r.caretPainter.cursorRadius }

// SetCursorRadius changes the radius of the caret's corners.
//
// Flutter: rendering/editable.dart RenderEditable.cursorRadius
func (r *RenderEditable) SetCursorRadius(value foundation.Opt[ui.Radius]) {
	r.caretPainter.setCursorRadius(value)
}

// StartHandleLayerLink links the start of the selection to the handle and the
// toolbar that follow it.
//
// Flutter: rendering/editable.dart RenderEditable.startHandleLayerLink
func (r *RenderEditable) StartHandleLayerLink() *LayerLink { return r.startHandleLayerLink }

// SetStartHandleLayerLink changes the layer link of the start handle.
//
// Flutter: rendering/editable.dart RenderEditable.startHandleLayerLink
func (r *RenderEditable) SetStartHandleLayerLink(value *LayerLink) {
	if r.startHandleLayerLink == value {
		return
	}
	r.startHandleLayerLink = value
	r.MarkNeedsPaint()
}

// EndHandleLayerLink links the end of the selection to the handle and the
// toolbar that follow it.
//
// Flutter: rendering/editable.dart RenderEditable.endHandleLayerLink
func (r *RenderEditable) EndHandleLayerLink() *LayerLink { return r.endHandleLayerLink }

// SetEndHandleLayerLink changes the layer link of the end handle.
//
// Flutter: rendering/editable.dart RenderEditable.endHandleLayerLink
func (r *RenderEditable) SetEndHandleLayerLink(value *LayerLink) {
	if r.endHandleLayerLink == value {
		return
	}
	r.endHandleLayerLink = value
	r.MarkNeedsPaint()
}

// EnableInteractiveSelection says whether the user can change the selection;
// None means unless the text is obscured.
//
// Flutter: rendering/editable.dart RenderEditable.enableInteractiveSelection
func (r *RenderEditable) EnableInteractiveSelection() foundation.Opt[bool] {
	return r.enableInteractive
}

// SetEnableInteractiveSelection changes whether the user can select.
//
// Flutter: rendering/editable.dart RenderEditable.enableInteractiveSelection
func (r *RenderEditable) SetEnableInteractiveSelection(value foundation.Opt[bool]) {
	if r.enableInteractive == value {
		return
	}
	r.enableInteractive = value
	r.MarkNeedsLayout()
	r.MarkNeedsSemanticsUpdate()
}

// SelectionEnabled is whether the user can select text.
//
// Flutter: rendering/editable.dart RenderEditable.selectionEnabled
func (r *RenderEditable) SelectionEnabled() bool {
	return r.enableInteractive.OrElse(!r.obscureText)
}

// PromptRectColor is the color of the highlight of the autocorrect prompt.
//
// Flutter: rendering/editable.dart RenderEditable.promptRectColor
func (r *RenderEditable) PromptRectColor() foundation.Opt[ui.Color] {
	return r.autocorrectHighlightPaint.highlightColor
}

// SetPromptRectColor changes the color of the autocorrect highlight.
//
// Flutter: rendering/editable.dart RenderEditable.promptRectColor
func (r *RenderEditable) SetPromptRectColor(value foundation.Opt[ui.Color]) {
	r.autocorrectHighlightPaint.setHighlightColor(value)
}

// SetPromptRectRange changes the range that is highlighted for the autocorrect
// prompt; None for no highlight.
//
// Flutter: rendering/editable.dart RenderEditable.setPromptRectRange
func (r *RenderEditable) SetPromptRectRange(newRange foundation.Opt[ui.TextRange]) {
	r.autocorrectHighlightPaint.setHighlightedRange(newRange)
}

// MaxScrollExtent is how far the text can be scrolled.
//
// Flutter: rendering/editable.dart RenderEditable.maxScrollExtent
func (r *RenderEditable) MaxScrollExtent() float64 { return r.maxScrollExtent }

func (r *RenderEditable) caretMargin() float64 { return caretGap + r.cursorWidth }

// ClipBehavior is how the text is clipped when it overflows.
//
// Flutter: rendering/editable.dart RenderEditable.clipBehavior
func (r *RenderEditable) ClipBehavior() ui.Clip { return r.clipBehavior }

// SetClipBehavior changes how the overflow is clipped.
//
// Flutter: rendering/editable.dart RenderEditable.clipBehavior
func (r *RenderEditable) SetClipBehavior(value ui.Clip) {
	if value == ui.ClipUnspecified {
		value = ui.ClipHardEdge
	}
	if value != r.clipBehavior {
		r.clipBehavior = value
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

// GetBoxesForSelection returns the boxes that the selection covers, in the
// coordinates of the editable.
//
// Flutter: rendering/editable.dart RenderEditable.getBoxesForSelection
func (r *RenderEditable) GetBoxesForSelection(selection services.TextSelection) []ui.TextBox {
	r.computeTextMetricsIfNeeded()
	paintOffset := r.paintOffset()
	boxes := r.textPainter.GetBoxesForSelection(selection.Range(), ui.BoxesForRangeOpts{
		HeightStyle: r.SelectionHeightStyle(), WidthStyle: r.SelectionWidthStyle(),
	})
	out := make([]ui.TextBox, len(boxes))
	for i, b := range boxes {
		out[i] = ui.NewTextBoxFromLTRBD(b.Left+paintOffset.Dx, b.Top+paintOffset.Dy, b.Right+paintOffset.Dx, b.Bottom+paintOffset.Dy, b.Direction)
	}
	return out
}

func (r *RenderEditable) onlyWhitespace(rng ui.TextRange) bool {
	text := r.PlainText()
	start, end := min(rng.Start, len(text)), min(rng.End, len(text))
	for _, c := range text[start:end] {
		if !services.TextLayoutMetricsIsWhitespace(c) {
			return false
		}
	}
	return true
}

func (r *RenderEditable) getNextWord(offset int) (ui.TextRange, bool) {
	for {
		rng := r.textPainter.GetWordBoundary(ui.NewTextPosition(offset))
		if !rng.IsValid() || rng.IsCollapsed() {
			return ui.TextRange{}, false
		}
		if !r.onlyWhitespace(rng) {
			return rng, true
		}
		offset = rng.End
	}
}

func (r *RenderEditable) getPreviousWord(offset int) (ui.TextRange, bool) {
	for offset >= 0 {
		rng := r.textPainter.GetWordBoundary(ui.NewTextPosition(offset))
		if !rng.IsValid() || rng.IsCollapsed() {
			return ui.TextRange{}, false
		}
		if !r.onlyWhitespace(rng) {
			return rng, true
		}
		offset = rng.Start - 1
	}
	return ui.TextRange{}, false
}

// Attach attaches the painters' render objects and creates the recognizers.
//
// Flutter: rendering/editable.dart RenderEditable.attach
func (r *RenderEditable) Attach(owner *PipelineOwner) {
	r.RenderBox.Attach(owner)
	if r.foregroundRenderObject != nil {
		r.foregroundRenderObject.Attach(owner)
	}
	if r.backgroundRenderObject != nil {
		r.backgroundRenderObject.Attach(owner)
	}
	if r.gestures != nil {
		r.tap = gestures.NewTapGestureRecognizer(r.gestures, gestures.TapGestureRecognizerOpts{DebugOwner: r})
		r.tap.OnTapDown = r.handleTapDownDetails
		r.tap.OnTap = r.handleTapGesture
		r.longPress = gestures.NewLongPressGestureRecognizer(r.gestures, gestures.LongPressGestureRecognizerOpts{DebugOwner: r})
		r.longPress.OnLongPress = r.handleLongPressGesture
	}
	r.offsetHandle = r.offset.AddListener(r.MarkNeedsPaint)
	r.showHideCursor()
	r.showCursorHandle = r.showCursor.AddListener(r.showHideCursor)
}

// Detach disposes the recognizers and detaches the painters' render objects.
//
// Flutter: rendering/editable.dart RenderEditable.detach
func (r *RenderEditable) Detach() {
	if r.tap != nil {
		r.tap.Dispose()
		r.tap = nil
	}
	if r.longPress != nil {
		r.longPress.Dispose()
		r.longPress = nil
	}
	r.offset.RemoveListener(r.offsetHandle)
	r.showCursor.RemoveListener(r.showCursorHandle)
	r.RenderBox.Detach()
	if r.foregroundRenderObject != nil {
		r.foregroundRenderObject.Detach()
	}
	if r.backgroundRenderObject != nil {
		r.backgroundRenderObject.Detach()
	}
}

// RedepthChildren updates the depth of the painters' render objects too.
//
// Flutter: rendering/editable.dart RenderEditable.redepthChildren
func (r *RenderEditable) RedepthChildren() {
	if r.foregroundRenderObject != nil {
		r.RedepthChild(r.foregroundRenderObject)
	}
	if r.backgroundRenderObject != nil {
		r.RedepthChild(r.backgroundRenderObject)
	}
	r.RenderBox.RedepthChildren()
}

// VisitChildren visits the render objects of the painters, then the inline
// children.
//
// Flutter: rendering/editable.dart RenderEditable.visitChildren
func (r *RenderEditable) VisitChildren(visitor RenderObjectVisitor) {
	if r.foregroundRenderObject != nil {
		visitor(r.foregroundRenderObject)
	}
	if r.backgroundRenderObject != nil {
		visitor(r.backgroundRenderObject)
	}
	r.RenderBox.VisitChildren(visitor)
}

func (r *RenderEditable) isMultiline() bool {
	return !r.maxLines.IsSome() || r.maxLines.Value() != 1
}

// viewportAxisHorizontal is whether the text scrolls horizontally.
func (r *RenderEditable) viewportAxisHorizontal() bool { return !r.isMultiline() }

func (r *RenderEditable) paintOffset() ui.Offset {
	if r.viewportAxisHorizontal() {
		return ui.Offset{Dx: -r.offset.Pixels()}
	}
	return ui.Offset{Dy: -r.offset.Pixels()}
}

func (r *RenderEditable) viewportExtent() float64 {
	if r.viewportAxisHorizontal() {
		return r.Size().Width
	}
	return r.Size().Height
}

func (r *RenderEditable) getMaxScrollExtent(contentSize ui.Size) float64 {
	if r.viewportAxisHorizontal() {
		return math.Max(0, contentSize.Width-r.Size().Width)
	}
	return math.Max(0, contentSize.Height-r.Size().Height)
}

// We need to check the paint offset here because during animation the start of
// the text may be outside the visible region even when the text fits.
func (r *RenderEditable) hasVisualOverflow() bool {
	return r.maxScrollExtent > 0 || r.paintOffset() != (ui.Offset{})
}

// GetEndpointsForSelection returns the points at the bottom of the start and of
// the end of the selection (one point when it is collapsed), in the coordinates
// of the editable. The handles and the toolbar go there.
//
// Flutter: rendering/editable.dart RenderEditable.getEndpointsForSelection
func (r *RenderEditable) GetEndpointsForSelection(selection services.TextSelection) []TextSelectionPoint {
	r.computeTextMetricsIfNeeded()
	paintOffset := r.paintOffset()
	var boxes []ui.TextBox
	if !selection.IsCollapsed() {
		boxes = r.textPainter.GetBoxesForSelection(selection.Range(), ui.BoxesForRangeOpts{
			HeightStyle: r.SelectionHeightStyle(), WidthStyle: r.SelectionWidthStyle(),
		})
	}
	if len(boxes) == 0 {
		// This doesn't work well at a boundary of left-to-right and
		// right-to-left text.
		caretOffset := r.textPainter.GetOffsetForCaret(selection.Extent(), r.caretPrototype)
		start := ui.Offset{Dy: r.PreferredLineHeight()}.Add(caretOffset).Add(paintOffset)
		return []TextSelectionPoint{{Point: start}}
	}
	first, last := boxes[0], boxes[len(boxes)-1]
	width := r.textPainter.Size().Width
	start := ui.Offset{Dx: ui.ClampDouble(first.Start(), 0, width), Dy: first.Bottom}.Add(paintOffset)
	end := ui.Offset{Dx: ui.ClampDouble(last.End(), 0, width), Dy: last.Bottom}.Add(paintOffset)
	return []TextSelectionPoint{
		{Point: start, Direction: foundation.Some(first.Direction)},
		{Point: end, Direction: foundation.Some(last.Direction)},
	}
}

// GetRectForComposingRange returns the rectangle that contains the composing
// text, or false if the range is not valid or is empty.
//
// Flutter: rendering/editable.dart RenderEditable.getRectForComposingRange
func (r *RenderEditable) GetRectForComposingRange(rng ui.TextRange) (ui.Rect, bool) {
	if !rng.IsValid() || rng.IsCollapsed() {
		return ui.Rect{}, false
	}
	r.computeTextMetricsIfNeeded()
	boxes := r.textPainter.GetBoxesForSelection(rng, ui.BoxesForRangeOpts{
		HeightStyle: r.SelectionHeightStyle(), WidthStyle: r.SelectionWidthStyle(),
	})
	if len(boxes) == 0 {
		return ui.Rect{}, false
	}
	rect := boxes[0].ToRect()
	for _, b := range boxes[1:] {
		rect = rect.ExpandToInclude(b.ToRect())
	}
	return rect.Shift(r.paintOffset()), true
}

// GetPositionForPoint returns the position in the text that is closest to the
// point, given in global coordinates.
//
// Flutter: rendering/editable.dart RenderEditable.getPositionForPoint
func (r *RenderEditable) GetPositionForPoint(globalPosition ui.Offset) ui.TextPosition {
	r.computeTextMetricsIfNeeded()
	return r.textPainter.GetPositionForOffset(r.GlobalToLocal(globalPosition, nil).Sub(r.paintOffset()))
}

// GetLocalRectForCaret returns the rectangle of the caret at the position, in
// the coordinates of the editable, moved to a physical pixel.
//
// Flutter: rendering/editable.dart RenderEditable.getLocalRectForCaret
func (r *RenderEditable) GetLocalRectForCaret(caretPosition ui.TextPosition) ui.Rect {
	r.computeTextMetricsIfNeeded()
	caretPrototype := r.caretPrototype
	caretOffset := r.textPainter.GetOffsetForCaret(caretPosition, caretPrototype)
	caretRect := caretPrototype.Shift(caretOffset.Add(r.CursorOffset()))
	scrollableWidth := math.Max(r.textPainter.Width()+r.caretMargin(), r.Size().Width)
	caretX := ui.ClampDouble(caretRect.Left, 0, math.Max(scrollableWidth-r.caretMargin(), 0))
	caretRect = ui.NewRectFromLTWH(caretX, caretRect.Top, caretRect.Width(), caretRect.Height())

	fullHeight := r.textPainter.GetFullHeightForCaret(caretPosition, caretPrototype)
	// Override the height to take the full height of the glyph at the position.
	// Deviation: only the code of the platforms that are not Apple's is ported.
	caretHeight := r.CursorHeight()
	heightDiff := fullHeight - caretHeight
	// Center the caret vertically along the text.
	caretRect = ui.NewRectFromLTWH(caretRect.Left, caretRect.Top-caretHeightOffset+heightDiff/2, caretRect.Width(), caretHeight)

	caretRect = caretRect.Shift(r.paintOffset())
	return caretRect.Shift(r.snapToPhysicalPixel(caretRect.TopLeft()))
}

// PreferredLineHeight is the height of a line of the text.
//
// Flutter: rendering/editable.dart RenderEditable.preferredLineHeight
func (r *RenderEditable) PreferredLineHeight() float64 { return r.textPainter.PreferredLineHeight() }

func (r *RenderEditable) countHardLineBreaks(text string) int {
	if n, ok := r.cachedLineBreakCount.Get(); ok {
		return n
	}
	count := 0
	for _, c := range text {
		switch c {
		case 0x000A, 0x0085, 0x000B, 0x000C, 0x2028, 0x2029:
			count++
		}
	}
	r.cachedLineBreakCount = foundation.Some(count)
	return count
}

func (r *RenderEditable) preferredHeight(width float64) float64 {
	maxLines := r.maxLines
	minLines := r.minLines
	if !minLines.IsSome() {
		minLines = maxLines
	}
	minHeight := r.PreferredLineHeight() * float64(minLines.OrElse(0))

	if !maxLines.IsSome() {
		var estimated float64
		if math.IsInf(width, 1) {
			estimated = r.PreferredLineHeight() * float64(r.countHardLineBreaks(r.PlainText())+1)
		} else {
			minW, maxW := r.adjustConstraints(0, width)
			p := r.textIntrinsics()
			p.Layout(minW, maxW)
			estimated = p.Height()
		}
		return math.Max(estimated, minHeight)
	}
	// maxLines == 1 forces the scroll direction to be horizontal: report the real
	// height of the (first) line so the text is not clipped.
	if maxLines.Value() == 1 {
		minW, maxW := r.adjustConstraints(0, width)
		p := r.textIntrinsics()
		p.Layout(minW, maxW)
		return p.Height()
	}
	if minLines == maxLines {
		return minHeight
	}
	maxHeight := r.PreferredLineHeight() * float64(maxLines.Value())
	minW, maxW := r.adjustConstraints(0, width)
	p := r.textIntrinsics()
	p.Layout(minW, maxW)
	return ui.ClampDouble(p.Height(), minHeight, maxHeight)
}

// ComputeMinIntrinsicWidth is the minimum width of the text.
//
// Flutter: rendering/editable.dart RenderEditable.computeMinIntrinsicWidth
func (r *RenderEditable) ComputeMinIntrinsicWidth(float64) float64 {
	dims := LayoutInlineChildren(&r.ContainerRenderObject, math.Inf(1), func(child Box, _ BoxConstraints) ui.Size {
		return ui.Size{Width: child.GetMinIntrinsicWidth(math.Inf(1))}
	}, ChildLayoutHelper.GetDryBaseline)
	minW, maxW := r.adjustConstraints(0, math.Inf(1))
	p := r.textIntrinsics()
	p.SetPlaceholderDimensions(dims)
	p.Layout(minW, maxW)
	return p.MinIntrinsicWidth()
}

// ComputeMaxIntrinsicWidth is the width at which the text takes one line, with
// the room for the caret.
//
// Flutter: rendering/editable.dart RenderEditable.computeMaxIntrinsicWidth
func (r *RenderEditable) ComputeMaxIntrinsicWidth(float64) float64 {
	// The height and the baseline do not matter: all the text is laid out in a
	// single line, so 0 stands for the height.
	dims := LayoutInlineChildren(&r.ContainerRenderObject, math.Inf(1), func(child Box, _ BoxConstraints) ui.Size {
		return ui.Size{Width: child.GetMaxIntrinsicWidth(math.Inf(1))}
	}, ChildLayoutHelper.GetDryBaseline)
	minW, maxW := r.adjustConstraints(0, math.Inf(1))
	p := r.textIntrinsics()
	p.SetPlaceholderDimensions(dims)
	p.Layout(minW, maxW)
	return p.MaxIntrinsicWidth() + r.caretMargin()
}

// ComputeMinIntrinsicHeight is the height of the text at the width.
//
// Flutter: rendering/editable.dart RenderEditable.computeMinIntrinsicHeight
func (r *RenderEditable) ComputeMinIntrinsicHeight(width float64) float64 {
	return r.GetMaxIntrinsicHeight(width)
}

// ComputeMaxIntrinsicHeight is the height of the text at the width.
//
// Flutter: rendering/editable.dart RenderEditable.computeMaxIntrinsicHeight
func (r *RenderEditable) ComputeMaxIntrinsicHeight(width float64) float64 {
	r.textIntrinsics().SetPlaceholderDimensions(
		LayoutInlineChildren(&r.ContainerRenderObject, width, ChildLayoutHelper.DryLayoutChild, ChildLayoutHelper.GetDryBaseline))
	return r.preferredHeight(width)
}

// ComputeDistanceToActualBaseline is the distance to the baseline of the first
// line.
//
// Flutter: rendering/editable.dart RenderEditable.computeDistanceToActualBaseline
func (r *RenderEditable) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	r.computeTextMetricsIfNeeded()
	return foundation.Some(r.textPainter.ComputeDistanceToActualBaseline(baseline))
}

// HitTestSelf is true: the editable is hit anywhere.
//
// Flutter: rendering/editable.dart RenderEditable.hitTestSelf
func (r *RenderEditable) HitTestSelf(ui.Offset) bool { return true }

// HitTestChildren adds the span under the position to the result if it is a hit
// test target, else hit tests the inline children.
//
// Flutter: rendering/editable.dart RenderEditable.hitTestChildren
func (r *RenderEditable) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	effective := position.Sub(r.paintOffset())
	// The hit test can't fall through the gaps between the characters of a
	// line: the bounds of a character reach the next one.
	if glyph, ok := r.textPainter.GetClosestGlyphForOffset(effective); ok && glyph.GraphemeClusterLayoutBounds.Contains(effective) {
		span := painting.InlineSpanGetSpanForPosition(r.textPainter.Text(), ui.NewTextPosition(glyph.GraphemeClusterRange.Start))
		if target, ok := span.(gestures.HitTestTarget); ok {
			result.Add(gestures.NewHitTestEntry(target))
			return true
		}
	}
	return HitTestInlineChildren(&r.ContainerRenderObject, result, effective)
}

// HandleEvent hands a pointer-down to the tap and long-press recognizers.
//
// Flutter: rendering/editable.dart RenderEditable.handleEvent
func (r *RenderEditable) HandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) {
	if down, ok := event.(*gestures.PointerDownEvent); ok && !r.ignorePointer && r.tap != nil {
		// Propagates the pointer event to the selection handlers.
		r.tap.AddPointer(down)
		r.longPress.AddPointer(down)
	}
}

// LastSecondaryTapDownPosition is the global position of the last secondary tap
// down (a right click); None before there was one.
//
// Flutter: rendering/editable.dart RenderEditable.lastSecondaryTapDownPosition
func (r *RenderEditable) LastSecondaryTapDownPosition() foundation.Opt[ui.Offset] {
	return r.lastSecondaryTapDownPosition
}

// HandleSecondaryTapDown records the position of a secondary tap down, for the
// selection and the context menu.
//
// Flutter: rendering/editable.dart RenderEditable.handleSecondaryTapDown
func (r *RenderEditable) HandleSecondaryTapDown(details gestures.TapDownDetails) {
	r.lastTapDownPosition = foundation.Some(details.GlobalPosition)
	r.lastSecondaryTapDownPosition = foundation.Some(details.GlobalPosition)
}

// HandleTapDown records the position of a tap down, for the selection that
// follows.
//
// Flutter: rendering/editable.dart RenderEditable.handleTapDown
func (r *RenderEditable) HandleTapDown(details gestures.TapDownDetails) {
	r.lastTapDownPosition = foundation.Some(details.GlobalPosition)
}

func (r *RenderEditable) handleTapDownDetails(details gestures.TapDownDetails) {
	r.HandleTapDown(details)
}

// HandleTap selects the position of the last tap down.
//
// Flutter: rendering/editable.dart RenderEditable.handleTap
func (r *RenderEditable) HandleTap() { r.SelectPosition(services.SelectionChangedCauseTap) }

func (r *RenderEditable) handleTapGesture() { r.HandleTap() }

// HandleDoubleTap selects the word at the last tap down.
//
// Flutter: rendering/editable.dart RenderEditable.handleDoubleTap
func (r *RenderEditable) HandleDoubleTap() { r.SelectWord(services.SelectionChangedCauseDoubleTap) }

// HandleLongPress selects the word at the last tap down.
//
// Flutter: rendering/editable.dart RenderEditable.handleLongPress
func (r *RenderEditable) HandleLongPress() { r.SelectWord(services.SelectionChangedCauseLongPress) }

func (r *RenderEditable) handleLongPressGesture() { r.HandleLongPress() }

func (r *RenderEditable) lastTapDown() ui.Offset {
	p, ok := r.lastTapDownPosition.Get()
	if !ok && foundation.DebugMode {
		panic(errors.New("EditableNoTapDown", "the editable was asked to select at the position of a tap down, but there was none"))
	}
	return p
}

// SelectPosition moves the caret to the position of the last tap down.
//
// Flutter: rendering/editable.dart RenderEditable.selectPosition
func (r *RenderEditable) SelectPosition(cause services.SelectionChangedCause) {
	r.SelectPositionAt(r.lastTapDown(), foundation.None[ui.Offset](), cause)
}

// SelectPositionAt selects from the global position from to the global position
// to (a caret at from when to is None).
//
// Flutter: rendering/editable.dart RenderEditable.selectPositionAt
func (r *RenderEditable) SelectPositionAt(from ui.Offset, to foundation.Opt[ui.Offset], cause services.SelectionChangedCause) {
	r.computeTextMetricsIfNeeded()
	paintOffset := r.paintOffset()
	fromPosition := r.textPainter.GetPositionForOffset(r.GlobalToLocal(from, nil).Sub(paintOffset))
	baseOffset, extentOffset := fromPosition.Offset, fromPosition.Offset
	if t, ok := to.Get(); ok {
		extentOffset = r.textPainter.GetPositionForOffset(r.GlobalToLocal(t, nil).Sub(paintOffset)).Offset
	}
	r.setSelection(services.NewTextSelection(baseOffset, extentOffset, services.TextSelectionOpts{Affinity: fromPosition.Affinity}), cause)
}

// WordBoundaries gives the boundaries of the words of the text.
//
// Flutter: rendering/editable.dart RenderEditable.wordBoundaries
func (r *RenderEditable) WordBoundaries() *painting.WordBoundary {
	return r.textPainter.WordBoundaries()
}

// SelectWord selects the word at the position of the last tap down.
//
// Flutter: rendering/editable.dart RenderEditable.selectWord
func (r *RenderEditable) SelectWord(cause services.SelectionChangedCause) {
	r.SelectWordsInRange(r.lastTapDown(), foundation.None[ui.Offset](), cause)
}

// SelectWordsInRange selects from the word at the global position from to the
// word at the global position to (just the first word when to is None).
//
// Flutter: rendering/editable.dart RenderEditable.selectWordsInRange
func (r *RenderEditable) SelectWordsInRange(from ui.Offset, to foundation.Opt[ui.Offset], cause services.SelectionChangedCause) {
	r.computeTextMetricsIfNeeded()
	paintOffset := r.paintOffset()
	fromPosition := r.textPainter.GetPositionForOffset(r.GlobalToLocal(from, nil).Sub(paintOffset))
	fromWord := r.GetWordAtOffset(fromPosition)
	toPosition := fromPosition
	if t, ok := to.Get(); ok {
		toPosition = r.textPainter.GetPositionForOffset(r.GlobalToLocal(t, nil).Sub(paintOffset))
	}
	toWord := fromWord
	if toPosition != fromPosition {
		toWord = r.GetWordAtOffset(toPosition)
	}
	fromBeforeTo := fromWord.Start() < toWord.End()
	base, extent := fromWord.Extent().Offset, toWord.Base().Offset
	if fromBeforeTo {
		base, extent = fromWord.Base().Offset, toWord.Extent().Offset
	}
	r.setSelection(services.NewTextSelection(base, extent, services.TextSelectionOpts{Affinity: fromWord.Affinity}), cause)
}

// SelectWordEdge moves the caret to the edge of the word at the last tap down
// that is nearest to it.
//
// Flutter: rendering/editable.dart RenderEditable.selectWordEdge
func (r *RenderEditable) SelectWordEdge(cause services.SelectionChangedCause) {
	r.computeTextMetricsIfNeeded()
	position := r.textPainter.GetPositionForOffset(r.GlobalToLocal(r.lastTapDown(), nil).Sub(r.paintOffset()))
	word := r.textPainter.GetWordBoundary(position)
	var sel services.TextSelection
	if position.Offset <= word.Start {
		sel = services.NewTextSelectionCollapsed(word.Start)
	} else {
		sel = services.NewTextSelectionCollapsed(word.End, services.TextSelectionOpts{Affinity: ui.TextAffinityUpstream})
	}
	r.setSelection(sel, cause)
}

// GetWordAtOffset returns the selection of the word at the position. Past the
// end of the text it is a caret at the end; for obscured text the whole text is
// the word.
//
// Flutter: rendering/editable.dart RenderEditable.getWordAtOffset
func (r *RenderEditable) GetWordAtOffset(position ui.TextPosition) services.TextSelection {
	plain := r.PlainText()
	// When long-pressing past the end of the text, make a collapsed cursor.
	if position.Offset >= len(plain) {
		return services.NewTextSelectionFromPosition(ui.TextPosition{Offset: len(plain), Affinity: ui.TextAffinityUpstream})
	}
	// If the text is obscured, the entire sentence is one word.
	if r.obscureText {
		return services.NewTextSelection(0, len(plain))
	}
	word := r.textPainter.GetWordBoundary(position)
	effectiveOffset := position.Offset
	if position.Affinity == ui.TextAffinityUpstream {
		// upstream affinity is effectively -1 in text position.
		effectiveOffset = position.Offset - 1
	}
	if foundation.DebugMode && effectiveOffset < 0 {
		panic(errors.New("EditableWordOffsetNegative", "the effective offset of a word position is negative", "position", position.String()))
	}
	// Deviation: Flutter has a rule for iOS and Android here (selecting the
	// previous word when the position is on whitespace); the other platforms
	// select the word at the position, and only that is ported.
	return services.NewTextSelection(word.Start, word.End)
}

func (r *RenderEditable) adjustConstraints(minWidth, maxWidth float64) (float64, float64) {
	availableMaxWidth := math.Max(0, maxWidth-r.caretMargin())
	availableMinWidth := math.Min(minWidth, availableMaxWidth)
	outMin := availableMinWidth
	if r.forceLine {
		outMin = availableMaxWidth
	}
	outMax := math.Inf(1)
	if r.isMultiline() {
		outMax = availableMaxWidth
	}
	return outMin, outMax
}

// computeTextMetricsIfNeeded lays the text out if the layout was marked as
// dirty or the constraints changed. The public methods that expose the metrics
// of the text call it, and so does paint: an invisible render object kept in
// the tree by a KeepAlive may paint without having been laid out.
//
// Flutter: rendering/editable.dart RenderEditable._computeTextMetricsIfNeeded
func (r *RenderEditable) computeTextMetricsIfNeeded() {
	c := r.BoxConstraints()
	minW, maxW := r.adjustConstraints(c.MinWidth, c.MaxWidth)
	r.textPainter.Layout(minW, maxW)
}

// computeCaretPrototype works out the rectangle of the caret at the origin.
//
// Flutter: rendering/editable.dart RenderEditable._computeCaretPrototype
func (r *RenderEditable) computeCaretPrototype() {
	// Deviation: only the code of the platforms that are not Apple's is ported.
	r.caretPrototype = ui.NewRectFromLTWH(0, caretHeightOffset, r.cursorWidth, r.CursorHeight()-2*caretHeightOffset)
}

// snapToPhysicalPixel returns the offset to add to the offset to put it on a
// physical pixel.
func (r *RenderEditable) snapToPhysicalPixel(sourceOffset ui.Offset) ui.Offset {
	globalOffset := r.LocalToGlobal(sourceOffset, nil)
	pixelMultiple := 1.0 / r.devicePixelRatio
	snap := func(v float64) float64 {
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return 0
		}
		return math.Round(v/pixelMultiple)*pixelMultiple - v
	}
	return ui.Offset{Dx: snap(globalOffset.Dx), Dy: snap(globalOffset.Dy)}
}

// ComputeDryLayout is the size the editable takes at the constraints.
//
// Flutter: rendering/editable.dart RenderEditable.computeDryLayout
func (r *RenderEditable) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	minW, maxW := r.adjustConstraints(constraints.MinWidth, constraints.MaxWidth)
	p := r.textIntrinsics()
	p.SetPlaceholderDimensions(LayoutInlineChildren(&r.ContainerRenderObject, constraints.MaxWidth, ChildLayoutHelper.DryLayoutChild, ChildLayoutHelper.GetDryBaseline))
	p.Layout(minW, maxW)
	width := constraints.MaxWidth
	if !r.forceLine {
		width = constraints.ConstrainWidth(p.Size().Width + r.caretMargin())
	}
	return ui.Size{Width: width, Height: constraints.ConstrainHeight(r.preferredHeight(constraints.MaxWidth))}
}

// ComputeDryBaseline is the baseline of the first line at the constraints.
//
// Flutter: rendering/editable.dart RenderEditable.computeDryBaseline
func (r *RenderEditable) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	minW, maxW := r.adjustConstraints(constraints.MinWidth, constraints.MaxWidth)
	p := r.textIntrinsics()
	p.SetPlaceholderDimensions(LayoutInlineChildren(&r.ContainerRenderObject, constraints.MaxWidth, ChildLayoutHelper.DryLayoutChild, ChildLayoutHelper.GetDryBaseline))
	p.Layout(minW, maxW)
	return foundation.Some(p.ComputeDistanceToActualBaseline(baseline))
}

// PerformLayout lays out the inline children and the text, sizes the editable
// and tells the offset how far the text can scroll.
//
// Flutter: rendering/editable.dart RenderEditable.performLayout
func (r *RenderEditable) PerformLayout() {
	constraints := r.BoxConstraints()
	r.placeholderDimensions = LayoutInlineChildren(&r.ContainerRenderObject, constraints.MaxWidth, ChildLayoutHelper.LayoutChild, ChildLayoutHelper.GetBaseline)
	minW, maxW := r.adjustConstraints(constraints.MinWidth, constraints.MaxWidth)
	r.textPainter.SetPlaceholderDimensions(r.placeholderDimensions)
	r.textPainter.Layout(minW, maxW)
	PositionInlineChildren(&r.ContainerRenderObject, r.textPainter.InlinePlaceholderBoxes())
	r.computeCaretPrototype()

	width := constraints.MaxWidth
	if !r.forceLine {
		width = constraints.ConstrainWidth(r.textPainter.Width() + r.caretMargin())
	}
	var preferredHeight float64
	switch {
	case !r.maxLines.IsSome():
		preferredHeight = math.Max(r.textPainter.Height(), r.PreferredLineHeight()*float64(r.minLines.OrElse(0)))
	case r.maxLines.Value() == 1:
		preferredHeight = r.textPainter.Height()
	default:
		maxLines := r.maxLines.Value()
		preferredHeight = ui.ClampDouble(r.textPainter.Height(),
			r.PreferredLineHeight()*float64(r.minLines.OrElse(maxLines)), r.PreferredLineHeight()*float64(maxLines))
	}
	r.SetSize(ui.Size{Width: width, Height: constraints.ConstrainHeight(preferredHeight)})
	contentSize := ui.Size{Width: r.textPainter.Width() + r.caretMargin(), Height: r.textPainter.Height()}

	painterConstraints := NewBoxConstraintsTight(contentSize)
	if r.foregroundRenderObject != nil {
		r.foregroundRenderObject.Layout(painterConstraints)
	}
	if r.backgroundRenderObject != nil {
		r.backgroundRenderObject.Layout(painterConstraints)
	}

	r.maxScrollExtent = r.getMaxScrollExtent(contentSize)
	r.offset.ApplyViewportDimension(r.viewportExtent())
	r.offset.ApplyContentDimensions(0, r.maxScrollExtent)
}

// StartVerticalCaretMovement starts a run of vertical caret movements at the
// position.
//
// Flutter: rendering/editable.dart RenderEditable.startVerticalCaretMovement
func (r *RenderEditable) StartVerticalCaretMovement(startPosition ui.TextPosition) *VerticalCaretMovementRun {
	metrics := r.textPainter.ComputeLineMetrics()
	line, offset := r.lineNumberFor(startPosition, metrics)
	return &VerticalCaretMovementRun{
		editable: r, lineMetrics: metrics, currentTextPosition: startPosition,
		currentLine: line, currentOffset: offset, positionCache: map[int]positionAtOffset{},
	}
}

func (r *RenderEditable) lineNumberFor(startPosition ui.TextPosition, metrics []ui.LineMetrics) (int, ui.Offset) {
	offset := r.textPainter.GetOffsetForCaret(startPosition, ui.Rect{})
	for _, m := range metrics {
		if m.Baseline > offset.Dy {
			return m.LineNumber, ui.Offset{Dx: offset.Dx, Dy: m.Baseline}
		}
	}
	// Flutter asserts that the position is the start of the text here. No position
	// gets here: the caret of a position that is not laid out is the one at the end
	// of the text, which is above the baseline of the last line.
	dy := 0.0
	if len(metrics) > 0 {
		last := metrics[len(metrics)-1]
		dy = last.Baseline + last.Descent
	}
	return max(0, len(metrics)-1), ui.Offset{Dx: offset.Dx, Dy: dy}
}

func (r *RenderEditable) paintContents(context *PaintingContext, offset ui.Offset) {
	effectiveOffset := offset.Add(r.paintOffset())
	if r.selection.IsSome() {
		r.updateSelectionExtentsVisibility(effectiveOffset)
	}
	// The painters paint in the coordinate space of the viewport, since the
	// coordinate space of the text painter is not known to high level widgets.
	if r.backgroundRenderObject != nil {
		context.PaintChild(r.backgroundRenderObject, offset)
	}
	r.textPainter.Paint(context.Canvas(), effectiveOffset)
	PaintInlineChildren(&r.ContainerRenderObject, context, effectiveOffset)
	if r.foregroundRenderObject != nil {
		context.PaintChild(r.foregroundRenderObject, offset)
	}
}

func (r *RenderEditable) paintHandleLayers(context *PaintingContext, endpoints []TextSelectionPoint, offset ui.Offset) {
	size := r.Size()
	startPoint := endpoints[0].Point
	startPoint = ui.Offset{Dx: ui.ClampDouble(startPoint.Dx, 0, size.Width), Dy: ui.ClampDouble(startPoint.Dy, 0, size.Height)}
	r.leaderLayerHandler.SetLayer(NewLeaderLayer(r.startHandleLayerLink, LeaderLayerOpts{Offset: startPoint.Add(offset)}))
	noPaint := func(*PaintingContext, ui.Offset) {}
	context.PushLayer(r.leaderLayerHandler.Layer(), noPaint, ui.OffsetZero, foundation.None[ui.Rect]())
	selection := r.selection.Value()
	if len(endpoints) == 2 {
		endPoint := endpoints[1].Point
		endPoint = ui.Offset{Dx: ui.ClampDouble(endPoint.Dx, 0, size.Width), Dy: ui.ClampDouble(endPoint.Dy, 0, size.Height)}
		r.endLeaderLayer.SetLayer(NewLeaderLayer(r.endHandleLayerLink, LeaderLayerOpts{Offset: endPoint.Add(offset)}))
		context.PushLayer(r.endLeaderLayer.Layer(), noPaint, ui.OffsetZero, foundation.None[ui.Rect]())
	} else if selection.IsCollapsed() {
		r.endLeaderLayer.SetLayer(NewLeaderLayer(r.endHandleLayerLink, LeaderLayerOpts{Offset: startPoint.Add(offset)}))
		context.PushLayer(r.endLeaderLayer.Layer(), noPaint, ui.OffsetZero, foundation.None[ui.Rect]())
	}
}

// ApplyPaintTransform leaves the transform alone for the painters' render
// objects, and translates it by the offset of an inline child.
//
// Flutter: rendering/editable.dart RenderEditable.applyPaintTransform
func (r *RenderEditable) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	if child == RenderObject(r.foregroundRenderObject) || child == RenderObject(r.backgroundRenderObject) {
		return
	}
	DefaultApplyPaintTransformInline(child.(Box), transform)
}

// Paint paints the text, the selection and the caret, clipped when the text
// overflows, and puts the layers that the selection handles follow.
//
// Flutter: rendering/editable.dart RenderEditable.paint
func (r *RenderEditable) Paint(context *PaintingContext, offset ui.Offset) {
	r.computeTextMetricsIfNeeded()
	if r.hasVisualOverflow() && r.clipBehavior != ui.ClipNone {
		r.clipRectLayer.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()), r.paintContents,
			PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: r.clipRectLayer.Layer()}))
	} else {
		r.clipRectLayer.SetLayer(nil)
		r.paintContents(context, offset)
	}
	if sel, ok := r.selection.Get(); ok && sel.IsValid() {
		r.paintHandleLayers(context, r.GetEndpointsForSelection(sel), offset)
	}
}

// DescribeApproximatePaintClip is the box of the editable when it clips.
//
// Flutter: rendering/editable.dart RenderEditable.describeApproximatePaintClip
func (r *RenderEditable) DescribeApproximatePaintClip(RenderObject) foundation.Opt[ui.Rect] {
	if r.clipBehavior == ui.ClipNone || !r.hasVisualOverflow() {
		return foundation.None[ui.Rect]()
	}
	return foundation.Some(ui.OffsetZero.And(r.Size()))
}

// DebugFillProperties adds the properties of the editable.
//
// Flutter: rendering/editable.dart RenderEditable.debugFillProperties
func (r *RenderEditable) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderBox.DebugFillProperties(b)
	b.Add(painting.NewColorProperty("cursorColor", r.CursorColor()))
	b.Add(foundation.NewDiagnosticsProperty("showCursor", r.showCursor))
	b.Add(foundation.NewIntProperty("maxLines", r.maxLines))
	b.Add(foundation.NewIntProperty("minLines", r.minLines))
	b.Add(foundation.NewDiagnosticsProperty("expands", r.expands, foundation.DiagnosticsPropertyOpts{DefaultValue: false}))
	b.Add(painting.NewColorProperty("selectionColor", r.SelectionColor()))
	b.Add(foundation.NewDiagnosticsProperty("textScaler", r.TextScaler(), foundation.DiagnosticsPropertyOpts{DefaultValue: painting.TextScalerNoScaling}))
	b.Add(foundation.NewDiagnosticsPropertyOpt("locale", r.Locale(), foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	b.Add(foundation.NewDiagnosticsPropertyOpt("selection", r.selection))
	b.Add(foundation.NewDiagnosticsProperty("offset", r.offset))
}

// DebugDescribeChildren describes the text.
//
// Flutter: rendering/editable.dart RenderEditable.debugDescribeChildren
func (r *RenderEditable) DebugDescribeChildren() []foundation.DiagnosticsNode {
	if r.Text() == nil {
		return nil
	}
	return []foundation.DiagnosticsNode{foundation.ToDiagnosticsNode(r.Text(), foundation.DiagnosticsNodeOpts{
		Name: foundation.Some("text"), Style: foundation.DiagnosticsTreeStyleTransition})}
}

// obscured returns the text with every character replaced by the obscuring
// character. It is only used for the semantics.
func (r *RenderEditable) obscured() string {
	return strings.Repeat(r.obscuringChar, foundation.GraphemeCount(r.PlainText()))
}
