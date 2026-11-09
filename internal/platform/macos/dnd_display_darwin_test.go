//go:build darwin && !ios && glutter_display

package macos

import (
	"math"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// fakeInfo is an NSDraggingInfo that the tests make: the pasteboard and the operations that the source allows.
// The place is read through the seam of the platform, since a Go callback cannot return the NSPoint of
// draggingLocation.
type fakeInfo struct {
	pasteboard objc.ID
	mask       uint64
}

var fakeInfoClass = sync.OnceValues(func() (*objcx.RegisteredClass, error) {
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "FakeDraggingInfo", Super: "NSObject",
		Methods: []objcx.Method{
			{Name: "draggingPasteboard", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
				f, _ := objcx.LookupAs[*fakeInfo](objcx.HandleOf(self))
				return f.pasteboard
			}},
			{Name: "draggingSourceOperationMask", Fn: func(self objc.ID, _ objc.SEL) uint64 {
				f, _ := objcx.LookupAs[*fakeInfo](objcx.HandleOf(self))
				return f.mask
			}},
		},
	})
})

// testPasteboard is a private pasteboard (not the clipboard of the user, and not the drag pasteboard) that holds
// the items; each item is its types with their strings.
func testPasteboard(name string, items ...pasteboardEntry) objc.ID {
	ns := objcx.NewString(name)
	defer objcx.Release(ns)
	pb := objc.ID(objcx.MustClass("NSPasteboard")).Send(objcx.Sel("pasteboardWithName:"), ns)
	pb.Send(objcx.Sel("clearContents"))
	objs := make([]objc.ID, len(items))
	for i, e := range items {
		objs[i] = pasteboardItemFor(e)
	}
	list := objcx.NewArray(objs...)
	pb.Send(objcx.Sel("writeObjects:"), list)
	objcx.Release(list)
	for _, o := range objs {
		objcx.Release(o)
	}
	return pb
}

func fileEntry(uri string) pasteboardEntry {
	return pasteboardEntry{Types: []string{pbFileURL}, Strings: map[string]string{pbFileURL: uri}}
}

func textEntry(s string) pasteboardEntry {
	return pasteboardEntry{Types: []string{pbText}, Strings: map[string]string{pbText: s}}
}

// dragInfo makes the fake NSDraggingInfo of a drag over the window at the place (in points from the top left).
func (s *session) dragInfo(w *window, pb objc.ID, x, y float64, mask uint64) objc.ID {
	cls, err := fakeInfoClass()
	if err != nil {
		s.t.Fatal(err)
	}
	info := cls.New(objcx.NewHandle(&fakeInfo{pasteboard: pb, mask: mask}))
	s.p.testDragLocation = func(objc.ID) nsPoint { return w.windowPoint(x, y) }
	return info
}

func (s *session) dropWindow(h *recorder) *window {
	w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
	return w
}

const allOps = dragOpCopy | dragOpMove | dragOpLink

func TestDropOfFilesReachesTheApp(t *testing.T) {
	h := &recorder{dropAccept: ui.DropActionCopy}
	runApp(t, Options{}, h, func(s *session) {
		w := s.dropWindow(h)
		s.onUI(func() {
			pb := testPasteboard("dev.glutter.test.drop-files", fileEntry("file:///tmp/a.txt"), fileEntry("file:///tmp/b%20c.png"))
			info := s.dragInfo(w, pb, 50, 40, allOps)
			if op := objc.Send[uint64](w.view, sel("draggingEntered:"), info); op != dragOpCopy {
				t.Errorf("draggingEntered: = %#x, want copy", op)
			}
			if op := objc.Send[uint64](w.view, sel("draggingUpdated:"), info); op != dragOpCopy {
				t.Errorf("draggingUpdated: = %#x, want copy", op)
			}
			if !objc.Send[bool](w.view, sel("prepareForDragOperation:"), info) {
				t.Error("prepareForDragOperation: said no")
			}
			if !objc.Send[bool](w.view, sel("performDragOperation:"), info) {
				t.Error("performDragOperation: said no to an accepted drop")
			}
		})
		ev := h.takeDrops()
		if len(ev) != 3 {
			t.Fatalf("the app heard %d drop events, want enter, move and drop", len(ev))
		}
		for i, want := range []ui.DropPhase{ui.DropPhaseEnter, ui.DropPhaseMove, ui.DropPhaseDrop} {
			if ev[i].Phase != want || ev[i].ViewID != w.id {
				t.Errorf("event %d = %+v, want phase %v of the view %v", i, ev[i], want, w.id)
			}
			if math.Abs(ev[i].Position.Dx-50*w.scale) > 0.5 || math.Abs(ev[i].Position.Dy-40*w.scale) > 0.5 {
				t.Errorf("event %d is at %v, want %v, %v", i, ev[i].Position, 50*w.scale, 40*w.scale)
			}
			if !slices.Equal(ev[i].Types, []string{"text/uri-list"}) {
				t.Errorf("event %d types = %v", i, ev[i].Types)
			}
			if !ev[i].Actions.Has(ui.DropActionCopy) {
				t.Errorf("event %d actions = %v, want the copy among them", i, ev[i].Actions)
			}
		}
		if got, want := ev[2].URIs, []string{"file:///tmp/a.txt", "file:///tmp/b%20c.png"}; !slices.Equal(got, want) {
			t.Errorf("the dropped files = %v, want %v", got, want)
		}
	})
}

func TestDropOfTextAndOfAURL(t *testing.T) {
	h := &recorder{dropAccept: ui.DropActionCopy}
	runApp(t, Options{}, h, func(s *session) {
		w := s.dropWindow(h)
		drop := func(name string, entries ...pasteboardEntry) ui.DropEvent {
			var out ui.DropEvent
			s.onUI(func() {
				pb := testPasteboard("dev.glutter.test.drop-"+name, entries...)
				info := s.dragInfo(w, pb, 10, 10, allOps)
				objc.Send[uint64](w.view, sel("draggingEntered:"), info)
				objc.Send[bool](w.view, sel("performDragOperation:"), info)
			})
			for _, e := range h.takeDrops() {
				if e.Phase == ui.DropPhaseDrop {
					out = e
				}
			}
			return out
		}
		if e := drop("text", textEntry("hello\nworld")); e.Text != "hello\nworld" || len(e.URIs) != 0 || !slices.Equal(e.Types, []string{"text/plain"}) {
			t.Errorf("a text drop = %+v", e)
		}
		url := pasteboardEntry{Types: []string{pbURL}, Strings: map[string]string{pbURL: "https://example.com/x"}}
		if e := drop("url", url); !slices.Equal(e.URIs, []string{"https://example.com/x"}) || e.Text != "https://example.com/x" {
			t.Errorf("a URL drop = %+v", e)
		}
		other := pasteboardEntry{Types: []string{"com.example.private"}, Strings: map[string]string{"com.example.private": "x"}}
		if e := drop("other", other); e.Text != "" || len(e.URIs) != 0 || len(e.Types) != 0 {
			t.Errorf("a drop of a type that the app does not take = %+v", e)
		}
	})
}

func TestDropThatIsNotAcceptedAndALeave(t *testing.T) {
	h := &recorder{} // accepts nothing
	runApp(t, Options{}, h, func(s *session) {
		w := s.dropWindow(h)
		s.onUI(func() {
			pb := testPasteboard("dev.glutter.test.drop-refused", textEntry("no"))
			info := s.dragInfo(w, pb, 10, 10, allOps)
			if op := objc.Send[uint64](w.view, sel("draggingEntered:"), info); op != dragOpNone {
				t.Errorf("a refused drag got the operation %#x", op)
			}
			if objc.Send[bool](w.view, sel("performDragOperation:"), info) {
				t.Error("a refused drop was performed")
			}
			// A drag that leaves: the app hears the leave once.
			objc.Send[uint64](w.view, sel("draggingEntered:"), info)
			w.view.Send(sel("draggingExited:"), info)
			w.view.Send(sel("draggingExited:"), info) // and not twice
		})
		var leaves int
		for _, e := range h.takeDrops() {
			if e.Phase == ui.DropPhaseLeave {
				leaves++
			}
		}
		if leaves != 1 {
			t.Errorf("the app heard %d leaves, want 1", leaves)
		}
	})
}

func TestSourceThatAllowsOnlyTheMoveGetsTheMove(t *testing.T) {
	h := &recorder{dropAccept: ui.DropActionCopy | ui.DropActionMove}
	runApp(t, Options{}, h, func(s *session) {
		w := s.dropWindow(h)
		s.onUI(func() {
			pb := testPasteboard("dev.glutter.test.drop-move", fileEntry("file:///tmp/m"))
			info := s.dragInfo(w, pb, 10, 10, dragOpMove)
			if op := objc.Send[uint64](w.view, sel("draggingEntered:"), info); op != dragOpMove {
				t.Errorf("operation = %#x, want move", op)
			}
		})
	})
}

func TestBlockedWindowTakesNoDrop(t *testing.T) {
	h := &recorder{dropAccept: ui.DropActionCopy}
	runApp(t, Options{}, h, func(s *session) {
		parent := s.dropWindow(h)
		s.dialog(parent, true)
		s.waitFor("the dialog's first frame", 5*time.Second, func() bool { return h.frameCount() > 1 })
		h.takeDrops()
		s.onUI(func() {
			pb := testPasteboard("dev.glutter.test.drop-blocked", fileEntry("file:///tmp/x"))
			info := s.dragInfo(parent, pb, 10, 10, allOps)
			if op := objc.Send[uint64](parent.view, sel("draggingEntered:"), info); op != dragOpNone {
				t.Errorf("a window with a modal window over it took a drag: %#x", op)
			}
			if objc.Send[bool](parent.view, sel("performDragOperation:"), info) {
				t.Error("a window with a modal window over it took a drop")
			}
		})
		if ev := h.takeDrops(); len(ev) != 0 {
			t.Errorf("the app heard %d events from a blocked window", len(ev))
		}
	})
}

func TestViewRegistersForTheTypesThatADropTakes(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.dropWindow(h)
		s.onUI(func() {
			arr := send[objc.ID](w.view, "registeredDraggedTypes")
			var got []string
			for i := range objcx.ArrayCount(arr) {
				got = append(got, objcx.String(objcx.ArrayAt(arr, i)))
			}
			for _, want := range droppedTypes {
				if !slices.Contains(got, want) {
					t.Errorf("the view is not registered for %q (it is for %v)", want, got)
				}
			}
		})
	})
}

func TestStartDragOffersItsItemsAndReportsTheEnd(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.dropWindow(h)
		var items, source objc.ID
		var began int
		s.onUI(func() {
			ev := mouseEvent(w, eventTypeLeftMouseDown, 1)
			s.p.testEvent = func() objc.ID { return ev }
			s.p.beginDrag = func(_ *window, its, _, src objc.ID) {
				began++
				items, source = objcx.Retain(its), src
			}
			req := ui.DragRequest{ID: 7, Text: "dragged text", URIs: []string{"file:///tmp/a.txt", "https://example.com"}, Actions: ui.DropActionCopy | ui.DropActionMove}
			if err := w.StartDrag(req); err != nil {
				t.Fatalf("StartDrag: %v", err)
			}
			if began != 1 {
				t.Fatalf("the session began %d times", began)
			}
			// The items are the text, the file and the URL; the first has the marker of the drag.
			if n := objcx.ArrayCount(items); n != 3 {
				t.Fatalf("the drag has %d items, want 3", n)
			}
			item := func(i int) objc.ID { return send[objc.ID](objcx.ArrayAt(items, i), "item") }
			stringOf := func(it objc.ID, typ string) string {
				ns := objcx.NewString(typ)
				defer objcx.Release(ns)
				return objcx.String(it.Send(sel("stringForType:"), ns))
			}
			if got := stringOf(item(0), pbText); got != "dragged text" {
				t.Errorf("the text item has %q", got)
			}
			if got := stringOf(item(0), dragMarker(os.Getpid(), 7)); got != "1" {
				t.Errorf("the first item has no marker of the drag: %q", got)
			}
			if got := stringOf(item(1), pbFileURL); got != "file:///tmp/a.txt" {
				t.Errorf("the file item has %q", got)
			}
			if got := stringOf(item(2), pbURL); got != "https://example.com" {
				t.Errorf("the URL item has %q", got)
			}
			// The source allows what the request allows, and the drag in progress refuses a second one.
			if op := objc.Send[uint64](source, sel("draggingSession:sourceOperationMaskForDraggingContext:"), objc.ID(0), int64(0)); op != dragOpCopy|dragOpMove {
				t.Errorf("the source allows %#x, want copy and move", op)
			}
			err := w.StartDrag(req)
			if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagDragInProgress || !appErr.HasRoute("window.StartDrag."+tagDragInProgress) {
				t.Errorf("a second drag: %v", err)
			}
			// The end: the target moved the data.
			objc.Send[struct{}](source, sel("draggingSession:endedAtPoint:operation:"), objc.ID(0), nsPoint{X: 1, Y: 2}, dragOpMove)
			objcx.Release(items)
		})
		h.mu.Lock()
		defer h.mu.Unlock()
		if len(h.dragEnds) != 1 || h.dragEnds[0] != (ui.DragResult{ID: 7, Action: ui.DropActionMove}) {
			t.Errorf("the end of the drag = %+v, want move for id 7", h.dragEnds)
		}
	})
}

func TestStartDragFailures(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.dropWindow(h)
		s.onUI(func() {
			s.p.beginDrag = func(*window, objc.ID, objc.ID, objc.ID) { t.Error("a drag began that should have failed") }
			cases := []struct {
				name  string
				event func() objc.ID
				req   ui.DragRequest
				tag   string
			}{
				{"no event", func() objc.ID { return 0 }, ui.DragRequest{ID: 1, Text: "x"}, tagDragNoButton},
				{"a release is not a button that is down", func() objc.ID { return mouseEvent(w, eventTypeLeftMouseUp, 1) }, ui.DragRequest{ID: 1, Text: "x"}, tagDragNoButton},
				{"nothing to offer", func() objc.ID { return mouseEvent(w, eventTypeLeftMouseDown, 1) }, ui.DragRequest{ID: 2}, tagDragEmpty},
			}
			for _, c := range cases {
				s.p.testEvent = c.event
				err := w.StartDrag(c.req)
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != c.tag || !appErr.HasRoute("window.StartDrag."+c.tag) {
					t.Errorf("%s: %v", c.name, err)
				}
			}
			if s.p.drag != nil {
				t.Error("a drag that failed left a session")
			}
		})
	})
}

func TestADragOfTheAppIsKnownAsItsOwnWhenDropped(t *testing.T) {
	h := &recorder{dropAccept: ui.DropActionCopy}
	runApp(t, Options{}, h, func(s *session) {
		w := s.dropWindow(h)
		s.onUI(func() {
			entries := dragEntries(ui.DragRequest{ID: 11, Text: "mine"}, os.Getpid())
			pb := testPasteboard("dev.glutter.test.drop-local", entries...)
			info := s.dragInfo(w, pb, 10, 10, allOps)
			objc.Send[uint64](w.view, sel("draggingEntered:"), info)
			objc.Send[bool](w.view, sel("performDragOperation:"), info)
		})
		ev := h.takeDrops()
		if len(ev) != 2 || ev[0].LocalID != 11 || ev[1].LocalID != 11 || ev[1].Text != "mine" {
			t.Errorf("events = %+v, want the local id 11 on the enter and the drop, with the text", ev)
		}
	})
}
