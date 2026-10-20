package widgets_test

import (
	"slices"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// The drags of the platform: DragTarget receives the drags of other applications,
// and a Draggable with an Export starts one (SPEC §8.11).

func externalTarget(key foundation.Key, opts DragTargetOpts[*ExternalDragData], candidates *[]*ExternalDragData) Widget {
	return NewDragTarget(func(_ BuildContext, c []foundation.Opt[*ExternalDragData], _ []any) Widget {
		*candidates = nil
		for _, d := range c {
			*candidates = append(*candidates, d.Value())
		}
		return dragBox(key, "Drop here", 100)
	}, opts)
}

func TestAnExternalDragReachesADragTarget(t *testing.T) {
	ts := gluttertest.New(t)
	var accepted []*ExternalDragData
	var candidates []*ExternalDragData
	left := 0
	ts.PumpWidget(dragApp(externalTarget(tgtKey, DragTargetOpts[*ExternalDragData]{
		OnAcceptWithDetails: func(d DragTargetDetails[*ExternalDragData]) { accepted = append(accepted, d.Data) },
		OnLeave:             func(foundation.Opt[*ExternalDragData]) { left++ },
	}, &candidates)))
	over := ts.Center(gluttertest.ByKey(tgtKey))
	types := []string{"text/uri-list", "text/plain"}

	if got := ts.Drop(0, ui.DropPhaseEnter, ui.Offset{Dx: 700, Dy: 500}, ui.DropEvent{Types: types, Actions: ui.DropActionCopy | ui.DropActionMove}); got != ui.DropActionNone {
		t.Errorf("a drag outside the target is accepted: %v", got)
	}
	got := ts.Drop(0, ui.DropPhaseMove, over, ui.DropEvent{Types: types, Actions: ui.DropActionCopy | ui.DropActionMove})
	ts.Pump()
	if got != ui.DropActionCopy {
		t.Errorf("over the target the action is %v, want copy", got)
	}
	if len(candidates) != 1 || !candidates[0].HasFiles() || candidates[0].Dropped {
		t.Fatalf("candidates = %+v", candidates)
	}
	// Leaving without a drop.
	ts.Drop(0, ui.DropPhaseMove, ui.Offset{Dx: 700, Dy: 500}, ui.DropEvent{Types: types})
	ts.Pump()
	if len(candidates) != 0 || left != 1 {
		t.Errorf("after leaving: candidates %v, left %d", candidates, left)
	}
	ts.Drop(0, ui.DropPhaseMove, over, ui.DropEvent{Types: types})
	ts.Drop(0, ui.DropPhaseLeave, over, ui.DropEvent{})
	ts.Pump()
	if len(candidates) != 0 || left != 2 || len(accepted) != 0 {
		t.Errorf("after the leave: candidates %v left %d accepted %v", candidates, left, accepted)
	}

	// The drop carries the data.
	ts.Drop(0, ui.DropPhaseEnter, over, ui.DropEvent{Types: types, Actions: ui.DropActionCopy})
	action := ts.Drop(0, ui.DropPhaseDrop, over, ui.DropEvent{Types: types, Actions: ui.DropActionCopy,
		URIs: []string{"file:///tmp/a%20b.txt", "http://example.com/x", "file://localhost/home/me/c.png"}, Text: "hello"})
	ts.Pump()
	if action != ui.DropActionCopy || len(accepted) != 1 {
		t.Fatalf("drop: action %v, accepted %d", action, len(accepted))
	}
	d := accepted[0]
	if !d.Dropped || d.Text != "hello" || !slices.Equal(d.Files(), []string{"/tmp/a b.txt", "/home/me/c.png"}) {
		t.Errorf("dropped data = %+v, files %v", d, d.Files())
	}
}

func TestAnExternalDragIsOnlyForTargetsThatTakeIt(t *testing.T) {
	ts := gluttertest.New(t)
	ints, anys := 0, 0
	ts.PumpWidget(dragApp(
		intTarget(tgtKey, DragTargetOpts[int]{OnMove: func(DragTargetDetails[int]) { ints++ }}),
		NewDragTarget(func(BuildContext, []foundation.Opt[any], []any) Widget { return dragBox(tgt2Key, "Any", 100) },
			DragTargetOpts[any]{OnMove: func(DragTargetDetails[any]) { anys++ }}),
	))
	for _, key := range []foundation.Key{tgtKey, tgt2Key} {
		ts.Drop(0, ui.DropPhaseMove, ts.Center(gluttertest.ByKey(key)), ui.DropEvent{Types: []string{"text/plain"}})
	}
	ts.Drop(0, ui.DropPhaseLeave, ui.Offset{}, ui.DropEvent{})
	if ints != 0 || anys != 1 {
		t.Errorf("a DragTarget[int] saw %d moves and a DragTarget[any] %d, want 0 and 1", ints, anys)
	}
}

func TestAnExternalDragCanBeRefusedByType(t *testing.T) {
	ts := gluttertest.New(t)
	var candidates []*ExternalDragData
	ts.PumpWidget(dragApp(externalTarget(tgtKey, DragTargetOpts[*ExternalDragData]{
		OnWillAcceptWithDetails: func(d DragTargetDetails[*ExternalDragData]) bool { return d.Data.HasFiles() },
	}, &candidates)))
	over := ts.Center(gluttertest.ByKey(tgtKey))
	if got := ts.Drop(0, ui.DropPhaseEnter, over, ui.DropEvent{Types: []string{"text/plain"}, Actions: ui.DropActionCopy}); got != ui.DropActionNone {
		t.Errorf("text over a target that wants files: %v", got)
	}
	ts.Drop(0, ui.DropPhaseLeave, over, ui.DropEvent{})
	if got := ts.Drop(0, ui.DropPhaseEnter, over, ui.DropEvent{Types: []string{"text/uri-list"}, Actions: ui.DropActionMove}); got != ui.DropActionMove {
		t.Errorf("files over a target that wants files, offered move only: %v", got)
	}
	ts.Drop(0, ui.DropPhaseLeave, over, ui.DropEvent{})
}

func TestADropWhereNothingAcceptsIsNotAccepted(t *testing.T) {
	ts := gluttertest.New(t)
	var candidates []*ExternalDragData
	ts.PumpWidget(dragApp(externalTarget(tgtKey, DragTargetOpts[*ExternalDragData]{}, &candidates)))
	got := ts.Drop(0, ui.DropPhaseDrop, ui.Offset{Dx: 700, Dy: 500}, ui.DropEvent{Types: []string{"text/plain"}, Text: "x"})
	if got != ui.DropActionNone {
		t.Errorf("a drop outside every target is accepted: %v", got)
	}
}

func TestADraggableWithAnExportStartsAPlatformDrag(t *testing.T) {
	ts := gluttertest.New(t)
	var accepted []int
	var gotData []any
	started, ended, completed, canceled := 0, 0, 0, 0
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
			Data: foundation.Some(42),
			Export: func(n int) *DragExport {
				return &DragExport{Text: "forty-two", URIs: []string{"file:///tmp/x"}, Actions: ui.DropActionCopy}
			},
			OnDragStarted:       func() { started++ },
			OnDragEnd:           func(DraggableDetails) { ended++ },
			OnDragCompleted:     func() { completed++ },
			OnDraggableCanceled: func(_ gestures.Velocity, _ ui.Offset) { canceled++ },
		}),
		NewDragTarget(func(BuildContext, []foundation.Opt[int], []any) Widget { return dragBox(tgtKey, "Target", 100) },
			DragTargetOpts[int]{OnAcceptWithDetails: func(d DragTargetDetails[int]) { accepted = append(accepted, d.Data) }}),
		NewDragTarget(func(BuildContext, []foundation.Opt[any], []any) Widget { return dragBox(tgt2Key, "Any", 100) },
			DragTargetOpts[any]{OnAcceptWithDetails: func(d DragTargetDetails[any]) { gotData = append(gotData, d.Data) }}),
	))
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	drags := ts.Drags()
	if len(drags) != 1 || drags[0].Text != "forty-two" || !slices.Equal(drags[0].URIs, []string{"file:///tmp/x"}) || drags[0].ID == 0 {
		t.Fatalf("drags = %+v", drags)
	}
	if started != 1 || textShown(ts, "Dragging") {
		t.Errorf("started %d, in-window feedback %v: the platform draws it", started, textShown(ts, "Dragging"))
	}
	// The pointer is the platform's now: its cancel changes nothing here.
	g.Cancel()
	ts.Pump()
	if ended != 0 {
		t.Fatalf("OnDragEnd was called %d times before the platform ended the drag", ended)
	}

	// The platform plays the drag over the app's own window, with the ID.
	id := drags[0].ID
	over := ts.Center(gluttertest.ByKey(tgtKey))
	ev := ui.DropEvent{Types: []string{"text/plain"}, Actions: ui.DropActionCopy, LocalID: id}
	ts.Drop(0, ui.DropPhaseEnter, over, ev)
	ts.Drop(0, ui.DropPhaseMove, over, ev)
	ts.Drop(0, ui.DropPhaseDrop, over, ev)
	ts.Pump()
	if !slices.Equal(accepted, []int{42}) {
		t.Errorf("the target got %v, want the Draggable's own data (42)", accepted)
	}
	// A target of any type gets it too, as the same value, not as an ExternalDragData.
	other := ts.Center(gluttertest.ByKey(tgt2Key))
	ts.Drop(0, ui.DropPhaseDrop, other, ev)
	if len(gotData) != 1 || gotData[0] != 42 {
		t.Errorf("the any target got %v", gotData)
	}

	ts.Dispatcher().DispatchDragEnd(ui.DragResult{ID: id, Action: ui.DropActionCopy})
	ts.Pump()
	if ended != 1 || completed != 1 || canceled != 0 {
		t.Errorf("ended %d completed %d canceled %d, want 1 1 0", ended, completed, canceled)
	}
	ts.Dispatcher().DispatchDragEnd(ui.DragResult{ID: id, Action: ui.DropActionCopy})
	if ended != 1 {
		t.Error("a drag that ended is reported again")
	}
}

func TestAnExportedDragThatIsNotAcceptedIsCanceled(t *testing.T) {
	ts := gluttertest.New(t)
	canceled, completed := 0, 0
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
			Data: foundation.Some(1), Export: func(int) *DragExport { return &DragExport{Text: "x"} },
			OnDragCompleted:     func() { completed++ },
			OnDraggableCanceled: func(_ gestures.Velocity, _ ui.Offset) { canceled++ },
		}),
	))
	ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	ts.Dispatcher().DispatchDragEnd(ui.DragResult{ID: ts.Drags()[0].ID, Action: ui.DropActionNone})
	ts.Pump()
	if canceled != 1 || completed != 0 {
		t.Errorf("canceled %d completed %d, want 1 0", canceled, completed)
	}
}

func TestADraggableWithAnExportDragsInTheWindowWhenThePlatformCannot(t *testing.T) {
	ts := gluttertest.New(t)
	ts.FailDrags(errors.New("DragUnsupported", "no"))
	var accepted []int
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
			Data: foundation.Some(5), Export: func(int) *DragExport { return &DragExport{Text: "x"} }}),
		intTarget(tgtKey, DragTargetOpts[int]{OnAcceptWithDetails: func(d DragTargetDetails[int]) { accepted = append(accepted, d.Data) }}),
	))
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	if !textShown(ts, "Dragging") || len(ts.Drags()) != 0 {
		t.Fatal("the feedback is not shown in the window")
	}
	g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
	ts.Pump()
	g.Up()
	ts.Pump()
	if !slices.Equal(accepted, []int{5}) {
		t.Errorf("accepted = %v", accepted)
	}
}

func TestDragExportOfADraggableWithoutDataStaysInTheWindow(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
			Export: func(int) *DragExport { return &DragExport{Text: "x"} }}),
	))
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	if len(ts.Drags()) != 0 || !textShown(ts, "Dragging") {
		t.Error("a draggable without data started a platform drag")
	}
	g.Up()
	ts.Pump()
}

func TestStartDragWithoutAHostThatCanFails(t *testing.T) {
	d := ui.NewPlatformDispatcher(nil)
	err := d.StartDrag(0, ui.DragRequest{ID: 1})
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("PlatformDispatcher.StartDrag.DragUnsupported") && appErr.Tag != "DragUnsupported" {
		t.Errorf("err = %v", err)
	}
}
