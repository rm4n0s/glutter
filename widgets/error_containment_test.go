package widgets

import (
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

func errorWidgets(rt *renderTree) []Element {
	return rt.find(func(w Widget) bool { _, ok := w.(*ErrorWidget); return ok })
}

// TestBuildPanicBecomesErrorWidget checks the recovery point of Build (SPEC
// §14.2): the panic is reported with the tag BuildPanicked and the element
// path, an ErrorWidget takes the place of the widget, and its siblings and the
// rest of the tree keep working.
func TestBuildPanicBecomesErrorWidget(t *testing.T) {
	tests := []struct {
		name    string
		panicV  any
		wantTag string
		wantMsg string
	}{
		{"a string", "boom", foundation.BuildPanicked, "boom"},
		{"an error", errors.New(tagPlain, "plain failure"), "Plain", "plain failure"},
		{"a tagged error keeps its tag", errors.New(tagMyBuildFailure, "tagged failure", "k", 1), "MyBuildFailure", "tagged failure"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := newRenderTree(t)
			tr.collectPanics = true
			siblingBuilds := 0
			fail := true
			var setState func(func())
			bad := newBuilder(func(_ BuildContext, ss func(func())) Widget {
				setState = ss
				if fail {
					panic(tc.panicV)
				}
				return newBox(10, 10)
			})
			sibling := newBuilder(func(BuildContext, func(func())) Widget { siblingBuilds++; return newBox(10, 10) })
			tr.pump(newColumn(bad, sibling))

			if len(tr.errs) != 1 {
				t.Fatalf("%d errors reported, want 1: %v", len(tr.errs), tr.errs)
			}
			appErr, ok := errors.FromError(tr.errs[0])
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", tr.errs[0])
			}
			if appErr.Tag != tc.wantTag || !appErr.HasRoute(tc.wantTag) {
				t.Errorf("unexpected failure path: %s (want %s)", appErr.Route(), tc.wantTag)
			}
			if !strings.Contains(appErr.Error(), tc.wantMsg) {
				t.Errorf("error %q does not say %q", appErr.Error(), tc.wantMsg)
			}
			if got := appErr.Metadata["context"]; got == nil || !strings.HasPrefix(got.(string), "while building ") {
				t.Errorf("context = %v, want 'while building ...'", got)
			}
			chain, _ := appErr.Metadata["creatorChain"].(string)
			if !strings.Contains(chain, "builder") || appErr.Metadata["stack"] == nil {
				t.Errorf("metadata = %v, want the creator chain and the stack", appErr.Metadata)
			}
			ews := errorWidgets(tr)
			if len(ews) != 1 {
				t.Fatalf("%d ErrorWidgets, want 1", len(ews))
			}
			if foundation.DebugMode {
				if msg := ews[0].AsElement().Widget().(*ErrorWidget).Message(); !strings.Contains(msg, tc.wantMsg) {
					t.Errorf("ErrorWidget message = %q, want it to say %q", msg, tc.wantMsg)
				}
			}
			if siblingBuilds != 1 {
				t.Errorf("sibling built %d times, want 1", siblingBuilds)
			}

			// The widget that failed builds again when it is fixed.
			fail = false
			setState(func() {})
			tr.pumpFrame()
			if n := len(errorWidgets(tr)); n != 0 {
				t.Errorf("%d ErrorWidgets after the fix, want 0", n)
			}
			if len(tr.errs) != 1 {
				t.Errorf("%d errors after the fix, want the one from before", len(tr.errs))
			}
			if siblingBuilds != 1 {
				t.Errorf("sibling built %d times after the fix, want 1", siblingBuilds)
			}
		})
	}
}

// TestUpdateChildPanicBecomesErrorWidget checks the second recovery point of
// ComponentElement.PerformRebuild: a child that cannot be created (here InitState
// panics) is replaced by an ErrorWidget and the parent stays.
func TestUpdateChildPanicBecomesErrorWidget(t *testing.T) {
	tr := newRenderTree(t)
	tr.collectPanics = true
	bad := newSpy(newBox(10, 10))
	bad.onInitState = func(BuildContext) { panic("init boom") }
	tr.pump(newColumn(newBuilder(func(BuildContext, func(func())) Widget { return bad }), newBox(10, 10)))
	if len(tr.errs) != 1 {
		t.Fatalf("%d errors reported, want 1: %v", len(tr.errs), tr.errs)
	}
	appErr := requireErrorTag(t, tr.errs[0], foundation.BuildPanicked)
	if !strings.Contains(appErr.Error(), "init boom") {
		t.Errorf("error %q does not say init boom", appErr.Error())
	}
	if n := len(errorWidgets(tr)); n != 1 {
		t.Errorf("%d ErrorWidgets, want 1", n)
	}
}

func TestCustomErrorWidgetBuilder(t *testing.T) {
	tr := newRenderTree(t)
	tr.collectPanics = true
	var got error
	tr.owner.ErrorWidgetBuilder = func(err error) Widget { got = err; return newBox(1, 1) }
	tr.pump(newBuilder(func(BuildContext, func(func())) Widget { panic("custom") }))
	if got == nil || !strings.Contains(got.Error(), "custom") {
		t.Errorf("builder got %v, want the error about custom", got)
	}
	if n := len(errorWidgets(tr)); n != 0 {
		t.Errorf("%d ErrorWidgets, want the custom widget instead", n)
	}
}

// TestBuildPanicWithoutReporterPropagates checks that a panic is not swallowed
// when there is no reporter to receive it.
func TestBuildPanicWithoutReporterPropagates(t *testing.T) {
	tr := newTree(t)
	tr.owner.OnError = nil
	requireTag(t, "NobodyListens", func() {
		tr.pump(newBuilder(func(BuildContext, func(func())) Widget { panic(errors.New(tagNobodyListens, "x")) }))
	})
}

type bomb struct {
	WidgetBase
	onRebuild func()
}

type bombElement struct{ ElementBase }

func (w *bomb) CreateElement() Element {
	e := &bombElement{}
	InitElement(&e.ElementBase, e, w)
	return e
}

func (e *bombElement) Mount(parent Element, slot any) {
	e.ElementBase.Mount(parent, slot)
	e.dirty = false
}

func (e *bombElement) PerformRebuild() {
	e.ElementBase.PerformRebuild()
	e.widget.(*bomb).onRebuild()
}

// TestRebuildPanicIsReportedAndTheNextElementBuilds checks BuildScope's
// recovery point: an element whose rebuild panics outside Build is reported and
// the other dirty elements are still built.
func TestRebuildPanicIsReportedAndTheNextElementBuilds(t *testing.T) {
	tr := newTree(t)
	tr.collectPanics = true
	var setState func(func())
	builds := 0
	b := newBuilder(func(_ BuildContext, ss func(func())) Widget { setState = ss; builds++; return newLeaf() })
	tr.pump(newMulti(&bomb{onRebuild: func() { panic("rebuild boom") }}, b))
	bombEl := findByType[*bomb](tr)[0]
	bombEl.MarkNeedsBuild()
	setState(func() {})
	builds = 0
	tr.pumpFrame()
	if builds != 1 {
		t.Errorf("builder built %d times, want 1: the panic stopped the other rebuilds", builds)
	}
	if len(tr.errs) != 1 {
		t.Fatalf("%d errors reported, want 1: %v", len(tr.errs), tr.errs)
	}
	appErr := requireErrorTag(t, tr.errs[0], foundation.BuildPanicked)
	if appErr.Metadata["context"] != "while rebuilding dirty elements" {
		t.Errorf("context = %v", appErr.Metadata["context"])
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMyBuildFailure = "MyBuildFailure"
	tagNobodyListens  = "NobodyListens"
	tagPlain          = "Plain"
)
