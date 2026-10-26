package uia

import (
	"reflect"
	"sync"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/uia/semtree"
	"github.com/rm4n0s/glutter/ui"
)

// fakeSink records the events the core raises.
type fakeSink struct {
	listening bool
	asked     int
	events    []string
	gone      []NodeRef
}

func (s *fakeSink) Listening() bool { s.asked++; return s.listening }
func (s *fakeSink) Structure(ref NodeRef, kind int) {
	s.events = append(s.events, "structure "+refText(ref)+" "+string(rune('0'+kind)))
}
func (s *fakeSink) PropertyChanged(ref NodeRef, id PropertyID, _, _ Value) {
	s.events = append(s.events, "property "+refText(ref)+" "+itoa(int(id)))
}
func (s *fakeSink) Automation(ref NodeRef, id EventID) {
	s.events = append(s.events, "event "+refText(ref)+" "+itoa(int(id)))
}
func (s *fakeSink) Disconnect(ref NodeRef) { s.gone = append(s.gone, ref) }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func refText(r NodeRef) string { return itoa(int(r.View)) + "/" + itoa(r.ID) }

// fakeWindow is the windows of the platform for the tests; Post runs at once.
type fakeWindow struct {
	x, y, w, h float64
	posted     int
}

func (f *fakeWindow) ClientRect(ui.ViewID) (x, y, w, h float64, ok bool) {
	return f.x, f.y, f.w, f.h, true
}
func (f *fakeWindow) HWND(ui.ViewID) uintptr { return 0x1234 }
func (f *fakeWindow) Post(fn func())         { f.posted++; fn() }

// fakeHost is the engine: it records what the bridge tells it.
type fakeHost struct {
	mu      sync.Mutex
	changes []bool
	actions []ui.SemanticsActionEvent
}

func (h *fakeHost) ScreenReaderChanged(on bool) {
	h.mu.Lock()
	h.changes = append(h.changes, on)
	h.mu.Unlock()
}
func (h *fakeHost) PerformAction(ev ui.SemanticsActionEvent) {
	h.mu.Lock()
	h.actions = append(h.actions, ev)
	h.mu.Unlock()
}

func newCore() (*Core, *fakeSink, *fakeWindow, *fakeHost) {
	sink, win, host := &fakeSink{}, &fakeWindow{w: 400, h: 300}, &fakeHost{}
	return NewCore(win, sink), sink, win, host
}

func buttonTree() *ui.SemanticsUpdate {
	btn := semtree.N(1, "Press", 0, 0, 10, 10)
	btn.Flags.IsButton = true
	btn.Actions = ui.SemanticsActionTap
	return semtree.Update(semtree.N(0, "", 0, 0, 100, 100, 1), btn)
}

func TestTheFirstAskTurnsTheBridgeOnOnce(t *testing.T) {
	c, _, _, host := newCore()
	if c.Ask(RootObjectID) {
		t.Error("the bridge answered before it had a host")
	}
	c.Start(host)
	if c.Ask(-4) {
		t.Error("the bridge answered a request for another object")
	}
	if c.On() || len(host.changes) != 0 {
		t.Fatal("the bridge is on before a UI Automation client asked")
	}
	if !c.Ask(RootObjectID) || !c.On() {
		t.Fatal("the first ask did not turn the bridge on")
	}
	if !c.Ask(RootObjectID) {
		t.Error("the second ask was not answered")
	}
	if !reflect.DeepEqual(host.changes, []bool{true}) {
		t.Errorf("the engine was told %v, want [true] once", host.changes)
	}
	c.Close()
	c.Close()
	if !reflect.DeepEqual(host.changes, []bool{true, false}) {
		t.Errorf("after Close the engine was told %v, want [true false]", host.changes)
	}
	if c.Ask(RootObjectID) || c.On() {
		t.Error("the bridge answered after Close")
	}
}

func TestATreeBeforeTheBridgeIsOnIsIgnored(t *testing.T) {
	c, sink, _, host := newCore()
	c.Start(host)
	c.UpdateSemantics(1, buttonTree())
	c.Read(func(m *Model) {
		if m.Root(1) != nil {
			t.Error("the model took a tree while the bridge was off")
		}
	})
	if len(sink.events) != 0 {
		t.Errorf("events = %v, want none", sink.events)
	}
}

func TestEventsAreRaisedOnlyWhileAClientListens(t *testing.T) {
	c, sink, _, host := newCore()
	c.Start(host)
	c.Ask(RootObjectID)
	c.SetWindow(1, "Win", true)
	c.UpdateSemantics(1, buttonTree())

	// Nobody listens: a change raises nothing, and the sink is asked once for the whole update.
	sink.asked = 0
	edited := semtree.N(1, "Go", 0, 0, 10, 10)
	edited.Flags.IsButton = true
	edited.Actions = ui.SemanticsActionTap
	c.UpdateSemantics(1, semtree.Update(edited))
	if len(sink.events) != 0 || sink.asked != 1 {
		t.Errorf("nobody listens: events %v, asked %d times; want none, once", sink.events, sink.asked)
	}

	sink.listening = true
	edited.Label = "Done"
	c.UpdateSemantics(1, semtree.Update(edited))
	want := []string{"property 1/1 30005"}
	if !reflect.DeepEqual(sink.events, want) {
		t.Errorf("events = %v, want %v", sink.events, want)
	}

	// A node that leaves is disconnected whether or not anyone listens.
	sink.listening = false
	sink.events = nil
	c.UpdateSemantics(1, semtree.Update(semtree.N(0, "", 0, 0, 100, 100)))
	if len(sink.gone) != 1 || sink.gone[0] != (NodeRef{1, 1}) {
		t.Errorf("disconnected = %v, want node 1", sink.gone)
	}
}

func TestFocusAndStructureAndTextEvents(t *testing.T) {
	c, sink, _, host := newCore()
	sink.listening = true
	c.Start(host)
	c.Ask(RootObjectID)
	c.SetWindow(1, "Win", true)
	field := semtree.N(1, "Name", 0, 0, 10, 10)
	field.Flags.IsTextField = true
	field.Flags.IsFocused = ui.TristateIsFalse
	field.Value = "ab"
	c.UpdateSemantics(1, semtree.Update(semtree.N(0, "", 0, 0, 100, 100, 1), field))
	if len(sink.events) != 1 || sink.events[0] != "structure 1/-1 2" {
		t.Fatalf("the first tree: events = %v, want the fragment root's children to be invalidated", sink.events)
	}
	sink.events = nil
	field.Flags.IsFocused = ui.TristateIsTrue
	field.Value = "abc"
	field.TextSelectionBase, field.TextSelectionExtent = 3, 3
	c.UpdateSemantics(1, semtree.Update(field))
	have := map[string]bool{}
	for _, e := range sink.events {
		have[e] = true
	}
	for _, want := range []string{"event 1/1 20005", "event 1/1 20015", "event 1/1 20014", "property 1/1 30045"} {
		if !have[want] {
			t.Errorf("missing %q in %v", want, sink.events)
		}
	}
}

func TestRequestsPostTheActionToTheUIThread(t *testing.T) {
	c, _, win, host := newCore()
	c.Start(host)
	c.Ask(RootObjectID)
	c.UpdateSemantics(1, buttonTree())
	st := c.Request(func(m *Model) ([]ui.SemanticsActionEvent, Status) { return m.Invoke(NodeRef{1, 1}) })
	if st != StatusOK || win.posted != 1 {
		t.Fatalf("status %v, posted %d; want ok and one post", st, win.posted)
	}
	want := []ui.SemanticsActionEvent{{ViewID: 1, NodeID: 1, Type: ui.SemanticsActionTap}}
	if !reflect.DeepEqual(host.actions, want) {
		t.Errorf("the framework got %+v, want %+v", host.actions, want)
	}
	if st := c.Request(func(m *Model) ([]ui.SemanticsActionEvent, Status) { return m.Invoke(NodeRef{1, 77}) }); st != StatusGone || win.posted != 1 {
		t.Errorf("a request for a node that is gone: status %v, posts %d; want gone and no post", st, win.posted)
	}
}

func TestCloseDisconnectsEveryNode(t *testing.T) {
	c, sink, _, host := newCore()
	c.Start(host)
	c.Ask(RootObjectID)
	c.UpdateSemantics(1, buttonTree())
	c.UpdateSemantics(2, semtree.Update(semtree.N(0, "", 0, 0, 10, 10)))
	c.Close()
	if len(sink.gone) != 3 {
		t.Errorf("disconnected %v, want the three nodes of the two views", sink.gone)
	}
	c.UpdateSemantics(1, buttonTree()) // after Close: nothing
	c.Read(func(m *Model) {
		if len(m.Views()) != 0 {
			t.Error("the model took a tree after Close")
		}
	})
}
