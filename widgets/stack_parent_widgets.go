// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// PositionedOpts are the optional parameters of NewPositioned.
//
// Flutter: widgets/basic.dart Positioned.new
type PositionedOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Left is the distance that the child's left edge is inset from the left of
	// the stack.
	Left foundation.Opt[float64]
	// Top is the distance that the child's top edge is inset from the top of the
	// stack.
	Top foundation.Opt[float64]
	// Right is the distance that the child's right edge is inset from the right
	// of the stack.
	Right foundation.Opt[float64]
	// Bottom is the distance that the child's bottom edge is inset from the
	// bottom of the stack.
	Bottom foundation.Opt[float64]
	// Width is the child's width; it must not be set together with Left and Right.
	Width foundation.Opt[float64]
	// Height is the child's height; it must not be set together with Top and
	// Bottom.
	Height foundation.Opt[float64]
}

// Positioned controls where a child of a Stack is positioned.
//
// Flutter: widgets/basic.dart Positioned
type Positioned struct {
	ParentDataWidgetOf[*rendering.StackParentData]
	left, top, right, bottom, width, height foundation.Opt[float64]
}

// NewPositioned creates a widget that controls where a child of a Stack is
// positioned. In debug builds it panics with the tag PositionedOverconstrained
// when it has a left, a right and a width, or a top, a bottom and a height.
//
// Flutter: widgets/basic.dart Positioned.new
func NewPositioned(child Widget, opts ...PositionedOpts) *Positioned {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.Left.IsSome() && o.Right.IsSome() && o.Width.IsSome() {
			panic(errors.New("PositionedOverconstrained", "a Positioned cannot have a left, a right and a width"))
		}
		if o.Top.IsSome() && o.Bottom.IsSome() && o.Height.IsSome() {
			panic(errors.New("PositionedOverconstrained", "a Positioned cannot have a top, a bottom and a height"))
		}
	}
	return &Positioned{NewParentDataWidgetBase[*rendering.StackParentData](child, WidgetOpts{Key: o.Key}),
		o.Left, o.Top, o.Right, o.Bottom, o.Width, o.Height}
}

// NewPositionedFromRect creates a Positioned with the left, top, width and height
// of the rect.
//
// Flutter: widgets/basic.dart Positioned.fromRect
func NewPositionedFromRect(rect ui.Rect, child Widget, opts ...WidgetOpts) *Positioned {
	return NewPositioned(child, PositionedOpts{Key: firstOpts(opts).Key, Left: foundation.Some(rect.Left), Top: foundation.Some(rect.Top),
		Width: foundation.Some(rect.Width()), Height: foundation.Some(rect.Height())})
}

// NewPositionedFromRelativeRect creates a Positioned with the left, top, right
// and bottom of the rect.
//
// Flutter: widgets/basic.dart Positioned.fromRelativeRect
func NewPositionedFromRelativeRect(rect rendering.RelativeRect, child Widget, opts ...WidgetOpts) *Positioned {
	return NewPositioned(child, PositionedOpts{Key: firstOpts(opts).Key, Left: foundation.Some(rect.Left), Top: foundation.Some(rect.Top),
		Right: foundation.Some(rect.Right), Bottom: foundation.Some(rect.Bottom)})
}

// PositionedFillOpts are the optional parameters of NewPositionedFill. The
// distances default to zero.
//
// Flutter: widgets/basic.dart Positioned.fill
type PositionedFillOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Left, Top, Right and Bottom are the distances from the edges of the stack.
	Left, Top, Right, Bottom float64
}

// NewPositionedFill creates a Positioned that fills the stack, with the given
// distances to its edges.
//
// Flutter: widgets/basic.dart Positioned.fill
func NewPositionedFill(child Widget, opts ...PositionedFillOpts) *Positioned {
	o := firstOpts(opts)
	return NewPositioned(child, PositionedOpts{Key: o.Key, Left: foundation.Some(o.Left), Top: foundation.Some(o.Top),
		Right: foundation.Some(o.Right), Bottom: foundation.Some(o.Bottom)})
}

// PositionedDirectionalOpts are the optional parameters of
// NewPositionedDirectional and of NewPositionedWithDirection.
//
// Flutter: widgets/basic.dart PositionedDirectional.new
type PositionedDirectionalOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Start is the distance from the start edge of the stack.
	Start foundation.Opt[float64]
	// Top is the distance from the top edge of the stack.
	Top foundation.Opt[float64]
	// End is the distance from the end edge of the stack.
	End foundation.Opt[float64]
	// Bottom is the distance from the bottom edge of the stack.
	Bottom foundation.Opt[float64]
	// Width is the child's width.
	Width foundation.Opt[float64]
	// Height is the child's height.
	Height foundation.Opt[float64]
}

// NewPositionedWithDirection creates a Positioned whose start and end are
// resolved with the text direction.
//
// Flutter: widgets/basic.dart Positioned.directional
func NewPositionedWithDirection(textDirection ui.TextDirection, child Widget, opts ...PositionedDirectionalOpts) *Positioned {
	o := firstOpts(opts)
	left, right := o.Start, o.End
	if textDirection == ui.TextDirectionRTL {
		left, right = o.End, o.Start
	}
	return NewPositioned(child, PositionedOpts{Key: o.Key, Left: left, Top: o.Top, Right: right, Bottom: o.Bottom, Width: o.Width, Height: o.Height})
}

// Left is the distance from the left edge of the stack.
//
// Flutter: widgets/basic.dart Positioned.left
func (w *Positioned) Left() foundation.Opt[float64] { return w.left }

// Top is the distance from the top edge of the stack.
//
// Flutter: widgets/basic.dart Positioned.top
func (w *Positioned) Top() foundation.Opt[float64] { return w.top }

// Right is the distance from the right edge of the stack.
//
// Flutter: widgets/basic.dart Positioned.right
func (w *Positioned) Right() foundation.Opt[float64] { return w.right }

// Bottom is the distance from the bottom edge of the stack.
//
// Flutter: widgets/basic.dart Positioned.bottom
func (w *Positioned) Bottom() foundation.Opt[float64] { return w.bottom }

// Width is the child's width.
//
// Flutter: widgets/basic.dart Positioned.width
func (w *Positioned) Width() foundation.Opt[float64] { return w.width }

// Height is the child's height.
//
// Flutter: widgets/basic.dart Positioned.height
func (w *Positioned) Height() foundation.Opt[float64] { return w.height }

// ApplyParentData writes the position to the parent data of the render object.
//
// Flutter: widgets/basic.dart Positioned.applyParentData
func (w *Positioned) ApplyParentData(renderObject rendering.RenderObject) {
	pd := renderObject.ParentData().(*rendering.StackParentData)
	needsLayout := false
	for _, f := range []struct {
		dst *foundation.Opt[float64]
		src foundation.Opt[float64]
	}{{&pd.Left, w.left}, {&pd.Top, w.top}, {&pd.Right, w.right}, {&pd.Bottom, w.bottom}, {&pd.Width, w.width}, {&pd.Height, w.height}} {
		if *f.dst != f.src {
			*f.dst = f.src
			needsLayout = true
		}
	}
	if needsLayout {
		if p := renderObject.AsRenderObject().Parent(); p != nil {
			p.MarkNeedsLayout()
		}
	}
}

// DebugTypicalAncestorWidgetClass is the name of the widget this one belongs in.
//
// Flutter: widgets/basic.dart Positioned.debugTypicalAncestorWidgetClass
func (*Positioned) DebugTypicalAncestorWidgetClass() string { return "Stack" }

// PositionedDirectional controls where a child of a Stack is positioned without
// committing to a specific text direction.
//
// Flutter: widgets/basic.dart PositionedDirectional
type PositionedDirectional struct {
	StatelessWidget
	start, top, end, bottom, width, height foundation.Opt[float64]
	child                                  Widget
}

// NewPositionedDirectional creates a widget that positions a child of a Stack
// with the start and end of the ambient text direction.
//
// Flutter: widgets/basic.dart PositionedDirectional.new
func NewPositionedDirectional(child Widget, opts ...PositionedDirectionalOpts) *PositionedDirectional {
	o := firstOpts(opts)
	return &PositionedDirectional{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), o.Start, o.Top, o.End, o.Bottom, o.Width, o.Height, child}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/basic.dart PositionedDirectional.child
func (w *PositionedDirectional) Child() Widget { return w.child }

// Build builds a Positioned for the ambient text direction.
//
// Flutter: widgets/basic.dart PositionedDirectional.build
func (w *PositionedDirectional) Build(bc BuildContext) Widget {
	return NewPositionedWithDirection(DirectionalityOf(bc), w.child, PositionedDirectionalOpts{
		Start: w.start, Top: w.top, End: w.end, Bottom: w.bottom, Width: w.width, Height: w.height})
}

// IndexedStackOpts are the optional parameters of NewIndexedStack.
//
// Flutter: widgets/indexed_stack.dart IndexedStack.new
type IndexedStackOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Alignment is how to align the non-positioned children; the default is
	// AlignmentDirectional.topStart.
	Alignment painting.AlignmentGeometry
	// TextDirection is the direction used to resolve a directional alignment.
	TextDirection ui.TextDirection
	// ClipBehavior is whether the content is clipped when it overflows; the
	// default is ClipHardEdge.
	ClipBehavior ui.Clip
	// Sizing is how to size the non-positioned children; the default is
	// StackFitLoose.
	Sizing rendering.StackFit
	// Index is the index of the child to show; the default is 0.
	Index foundation.Opt[int]
	// NoIndex shows no child at all (Flutter's index: null); it wins over Index.
	NoIndex bool
}

// IndexedStack is a Stack that shows a single child from a list of children.
//
// Deviation: index: null is NoIndex (SPEC §12.10).
//
// Flutter: widgets/indexed_stack.dart IndexedStack
type IndexedStack struct {
	StatelessWidget
	alignment     painting.AlignmentGeometry
	textDirection ui.TextDirection
	clipBehavior  ui.Clip
	sizing        rendering.StackFit
	index         foundation.Opt[int]
	children      []Widget
}

// NewIndexedStack creates a stack that shows one of the children.
//
// Flutter: widgets/indexed_stack.dart IndexedStack.new
func NewIndexedStack(children []Widget, opts ...IndexedStackOpts) *IndexedStack {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentDirectionalTopStart
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.Sizing == rendering.StackFitUnspecified {
		o.Sizing = rendering.StackFitLoose
	}
	index := foundation.Some(o.Index.OrElse(0))
	if o.NoIndex {
		index = foundation.None[int]()
	}
	return &IndexedStack{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), o.Alignment, o.TextDirection, o.ClipBehavior, o.Sizing, index, children}
}

// Alignment is how to align the non-positioned children.
//
// Flutter: widgets/indexed_stack.dart IndexedStack.alignment
func (w *IndexedStack) Alignment() painting.AlignmentGeometry { return w.alignment }

// TextDirection is the direction used to resolve a directional alignment.
//
// Flutter: widgets/indexed_stack.dart IndexedStack.textDirection
func (w *IndexedStack) TextDirection() ui.TextDirection { return w.textDirection }

// ClipBehavior is whether the content is clipped when it overflows.
//
// Flutter: widgets/indexed_stack.dart IndexedStack.clipBehavior
func (w *IndexedStack) ClipBehavior() ui.Clip { return w.clipBehavior }

// Sizing is how to size the non-positioned children.
//
// Flutter: widgets/indexed_stack.dart IndexedStack.sizing
func (w *IndexedStack) Sizing() rendering.StackFit { return w.sizing }

// Index is the index of the child to show, if any.
//
// Flutter: widgets/indexed_stack.dart IndexedStack.index
func (w *IndexedStack) Index() foundation.Opt[int] { return w.index }

// Children are the child widgets of the stack.
//
// Flutter: widgets/indexed_stack.dart IndexedStack.children
func (w *IndexedStack) Children() []Widget { return w.children }

// Build builds the stack, with each child in a scope that tells Visibility.of
// whether the child is the shown one.
//
// Flutter: widgets/indexed_stack.dart IndexedStack.build
func (w *IndexedStack) Build(BuildContext) Widget {
	wrapped := make([]Widget, len(w.children))
	for i, c := range w.children {
		selected := index(w.index, i)
		wrapped[i] = newVisibilityScope(selected, NewExcludeFocus(c, ExcludeFocusOpts{NotExcluding: selected}))
	}
	return newRawIndexedStack(w, wrapped)
}

func index(shown foundation.Opt[int], i int) bool {
	n, ok := shown.Get()
	return ok && n == i
}

// rawIndexedStack is the Stack of an IndexedStack: its render object is a
// RenderIndexedStack.
//
// Flutter: widgets/indexed_stack.dart _RawIndexedStack
type rawIndexedStack struct {
	Stack
	index foundation.Opt[int]
}

func newRawIndexedStack(w *IndexedStack, children []Widget) *rawIndexedStack {
	n, ok := w.index.Get()
	if foundation.DebugMode && ok && (n != 0 || len(children) != 0) && (n < 0 || n >= len(children)) {
		panic(errors.New("IndexedStackIndexInvalid", "the index must be null or within the range of children",
			"index", n, "children", len(children)))
	}
	return &rawIndexedStack{*NewStack(children, StackOpts{Alignment: w.alignment, TextDirection: w.textDirection,
		ClipBehavior: w.clipBehavior, Fit: w.sizing}), w.index}
}

func (w *rawIndexedStack) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	w.debugCheckDirectionality(bc)
	return rendering.NewRenderIndexedStack(nil, rendering.RenderIndexedStackOpts{
		RenderStackOpts: rendering.RenderStackOpts{Alignment: w.alignment, TextDirection: firstDirection(w.textDirection, bc),
			Fit: w.fit, ClipBehavior: w.clipBehavior},
		Index: w.index})
}

func (w *rawIndexedStack) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	w.debugCheckDirectionality(bc)
	r := renderObject.(*rendering.RenderIndexedStack)
	r.SetIndex(w.index)
	r.SetFit(w.fit)
	r.SetClipBehavior(w.clipBehavior)
	r.SetAlignment(w.alignment)
	r.SetTextDirection(firstDirection(w.textDirection, bc))
}

func (w *rawIndexedStack) CreateElement() Element {
	e := &indexedStackElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// indexedStackElement is the element of a rawIndexedStack: only the shown child
// is onstage.
//
// Flutter: widgets/indexed_stack.dart _IndexedStackElement
type indexedStackElement struct {
	MultiChildRenderObjectElement
}

// DebugVisitOnstageChildren visits the shown child, if any.
func (e *indexedStackElement) DebugVisitOnstageChildren(visitor ElementVisitor) {
	n, ok := e.widget.(*rawIndexedStack).index.Get()
	if !ok {
		return
	}
	i := 0
	e.VisitChildren(func(c Element) {
		if i == n {
			visitor(c)
		}
		i++
	})
}
