package widgets

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func TestErrorWidget(t *testing.T) {
	err := errors.New("SomethingFailed", "something failed")
	tests := []struct {
		name    string
		w       *ErrorWidget
		message string
		err     error
	}{
		{"from an error", NewErrorWidget(err), err.Error(), err},
		{"from a nil error", NewErrorWidget(nil), "Error", nil},
		{"with details", NewErrorWidgetWithDetails("details", nil), "details", nil},
		{"with details and an error", NewErrorWidgetWithDetails("details", err), "details", err},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.w.Message() != tc.message || tc.w.Err() != tc.err {
				t.Errorf("message = %q, err = %v", tc.w.Message(), tc.w.Err())
			}
			if tc.w.Key() == nil {
				t.Error("an error widget has no key")
			}
			rt := newRenderTree(t)
			rt.pump(newCenter(tc.w))
			ro, ok := renderElementOfType[*ErrorWidget](t, rt).RenderObject().(*rendering.RenderErrorBox)
			if !ok || ro.Message() != tc.message {
				t.Errorf("render object = %v", ro)
			}
			if size := ro.Size(); size != (ui.Size{Width: 800, Height: 600}) {
				t.Errorf("size = %v, want the whole view", size)
			}
		})
	}
	// Two error widgets never update each other: the keys are unique.
	if CanUpdate(NewErrorWidget(nil), NewErrorWidget(nil)) {
		t.Error("two error widgets can update each other")
	}
}

func TestErrorWidgetProperties(t *testing.T) {
	skipUnlessDebug(t)
	b := foundation.NewDiagnosticPropertiesBuilder()
	NewErrorWidget(errors.New("Tag", "message")).DebugFillProperties(b)
	if len(b.Properties) != 1 || b.Properties[0].Name().OrElse("") != "message" {
		t.Errorf("properties = %v", b.Properties)
	}
}

// plainProxy is a ProxyWidget with its own element (like NotificationListener).
type plainProxy struct {
	ProxyWidget
	notified *[]Widget
}

func (w *plainProxy) CreateElement() Element {
	e := &plainProxyElement{}
	InitProxyElement(&e.ProxyElement, e, w)
	return e
}

type plainProxyElement struct{ ProxyElement }

func (e *plainProxyElement) NotifyClients(old Widget) {
	*e.widget.(*plainProxy).notified = append(*e.widget.(*plainProxy).notified, old)
}

func TestProxyElement(t *testing.T) {
	var notified []Widget
	mk := func(child Widget) *plainProxy {
		return &plainProxy{NewProxyWidgetBase(child), &notified}
	}
	tr := newTree(t)
	first := mk(newLeaf())
	tr.pump(first)
	if len(notified) != 0 {
		t.Fatalf("notified on mount: %v", notified)
	}
	if first.Child() == nil || len(findByType[*leaf](tr)) != 1 {
		t.Error("the proxy did not build its child")
	}
	second := mk(newLeaf())
	tr.pump(second)
	if len(notified) != 1 || notified[0] != Widget(first) {
		t.Errorf("notified = %v, want the old widget", notified)
	}
}

func TestReassemble(t *testing.T) {
	// Reassemble marks the element and its descendants dirty, and tells the
	// state; the next build clears the flags.
	var hooked int
	tr := newTree(t)
	tr.pump(newWrapper(&reassembleWidget{NewStatefulWidgetBase(), &reassembleSpy{hook: func() { hooked++ }}}))
	el := elementOfType[*reassembleWidget](t, tr)
	tr.owner.BuildScope(tr.root, func() { tr.root.Reassemble() })
	if hooked != 1 {
		t.Errorf("Reassemble hook called %d times, want 1", hooked)
	}
	for _, e := range []Element{el, elementOfType[*wrapper](t, tr)} {
		if e.Dirty() {
			t.Errorf("%v was not rebuilt after the reassemble", e)
		}
	}
}

type reassembleSpy struct{ hook func() }

type reassembleWidget struct {
	StatefulWidget
	spy *reassembleSpy
}

func (w *reassembleWidget) CreateState() State { return &reassembleState{} }

type reassembleState struct{ StateOf[*reassembleWidget] }

func (s *reassembleState) Reassemble()               { s.Widget().spy.hook() }
func (s *reassembleState) Build(BuildContext) Widget { return newLeaf() }

// TestErrorWidgetInTheTreeOfABinding checks that the error box of a binding's
// tree is made (with the text engine of the dispatcher) and laid out in the view.
func TestErrorWidgetInTheTreeOfABinding(t *testing.T) {
	skipUnlessDebug(t)
	bt := newBindingTree(t, bindingTreeOpts{collectPanics: true})
	bt.pump(builderOf(func(BuildContext) Widget { panic("shown in the error box") }))
	if len(bt.errs) != 1 {
		t.Fatalf("%d errors, want 1: %v", len(bt.errs), bt.errs)
	}
	ews := bt.find(func(w Widget) bool { _, ok := w.(*ErrorWidget); return ok })
	if len(ews) != 1 {
		t.Fatalf("%d ErrorWidgets, want 1", len(ews))
	}
	box := ews[0].AsElement().RenderObject().(*rendering.RenderErrorBox)
	if size := box.Size(); size.Width != 800 || size.Height != 600 {
		t.Errorf("the error box has size %v, want the 800x600 of the view", size)
	}
}
