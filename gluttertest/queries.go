package gluttertest

import (
	"fmt"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Elements are the elements the finder finds in the tree under test.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.elementList
func (t *Tester) Elements(f Finder) []widgets.Element { return f.Find(t) }

// Widgets are the widgets the finder finds.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.widgetList
func (t *Tester) Widgets(f Finder) []widgets.Widget {
	var ws []widgets.Widget
	for _, e := range f.Find(t) {
		ws = append(ws, e.Widget())
	}
	return ws
}

// Count is how many elements the finder finds.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.any
func (t *Tester) Count(f Finder) int { return len(f.Find(t)) }

// Exists is whether the finder finds anything.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.any
func (t *Tester) Exists(f Finder) bool { return t.Count(f) > 0 }

// Element is the one element the finder finds. The test fails with the tag
// FinderNoMatch when there is none, and FinderManyMatches when there are more
// than one.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.element
func (t *Tester) Element(f Finder) widgets.Element {
	t.tb.Helper()
	found := f.Find(t)
	switch len(found) {
	case 1:
		return found[0]
	case 0:
		t.fail(errors.New("FinderNoMatch", "the finder found nothing: "+f.Describe(), "finder", f.Describe()))
	default:
		t.fail(errors.New("FinderManyMatches", fmt.Sprintf("the finder found %d elements, expected one: %s", len(found), f.Describe()),
			"finder", f.Describe(), "count", len(found)))
	}
	return nil
}

// Widget is the one widget the finder finds (see Element).
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.widget
func (t *Tester) Widget(f Finder) widgets.Widget {
	t.tb.Helper()
	e := t.Element(f)
	if e == nil {
		return nil
	}
	return e.Widget()
}

// WidgetAs is the one widget the finder finds, as the type T. The test fails
// with the tag FinderWrongType if it has another type.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.widget
func WidgetAs[T widgets.Widget](t *Tester, f Finder) T {
	t.tb.Helper()
	var zero T
	w := t.Widget(f)
	typed, ok := w.(T)
	if !ok {
		t.fail(errors.New("FinderWrongType", fmt.Sprintf("the finder found a %T, expected another type: %s", w, f.Describe()),
			"finder", f.Describe()))
		return zero
	}
	return typed
}

// StateOf is the state of the one StatefulWidget the finder finds, as the type
// T. The test fails with the tag FinderNotStateful when the element has no
// state, and FinderWrongType when the state has another type.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.state
func StateOf[T widgets.State](t *Tester, f Finder) T {
	t.tb.Helper()
	var zero T
	e := t.Element(f)
	if e == nil {
		return zero
	}
	stateful, ok := e.(*widgets.StatefulElement)
	if !ok {
		t.fail(errors.New("FinderNotStateful", "the element has no state: "+f.Describe(), "finder", f.Describe()))
		return zero
	}
	state, ok := stateful.State().(T)
	if !ok {
		t.fail(errors.New("FinderWrongType", fmt.Sprintf("the state is a %T, expected another type: %s", stateful.State(), f.Describe()),
			"finder", f.Describe()))
		return zero
	}
	return state
}

// RenderObject is the render object of the one element the finder finds.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.renderObject
func (t *Tester) RenderObject(f Finder) rendering.RenderObject {
	t.tb.Helper()
	e := t.Element(f)
	if e == nil {
		return nil
	}
	return e.RenderObject()
}

// rect is the box of the one element the finder finds, in logical pixels of the
// view.
func (t *Tester) box(f Finder) (rendering.Box, ui.Offset) {
	t.tb.Helper()
	e := t.Element(f)
	if e == nil {
		return nil, ui.Offset{}
	}
	ro := e.RenderObject()
	b, ok := ro.(rendering.Box)
	if !ok || ro == nil {
		t.fail(errors.New("FinderNotBox", fmt.Sprintf("the element has no render box: %s", f.Describe()), "finder", f.Describe()))
		return nil, ui.Offset{}
	}
	return b, b.LocalToGlobal(ui.OffsetZero, nil)
}

// TopLeft is the position of the top left corner of the one widget the finder
// finds, in logical pixels of the view.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.getTopLeft
func (t *Tester) TopLeft(f Finder) ui.Offset {
	t.tb.Helper()
	_, pos := t.box(f)
	return pos
}

// Size is the size of the render box of the one widget the finder finds.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.getSize
func (t *Tester) Size(f Finder) ui.Size {
	t.tb.Helper()
	b, _ := t.box(f)
	if b == nil {
		return ui.Size{}
	}
	return b.Size()
}

// Center is the position of the center of the one widget the finder finds, in
// logical pixels of the view.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.getCenter
func (t *Tester) Center(f Finder) ui.Offset {
	t.tb.Helper()
	b, pos := t.box(f)
	if b == nil {
		return ui.Offset{}
	}
	s := b.Size()
	return ui.Offset{Dx: pos.Dx + s.Width/2, Dy: pos.Dy + s.Height/2}
}

// BottomRight is the position of the bottom right corner of the one widget the
// finder finds.
//
// Flutter: flutter_test/lib/src/widget_tester.dart WidgetController.getBottomRight
func (t *Tester) BottomRight(f Finder) ui.Offset {
	t.tb.Helper()
	b, pos := t.box(f)
	if b == nil {
		return ui.Offset{}
	}
	s := b.Size()
	return ui.Offset{Dx: pos.Dx + s.Width, Dy: pos.Dy + s.Height}
}
