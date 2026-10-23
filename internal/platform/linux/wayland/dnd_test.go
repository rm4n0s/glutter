//go:build linux

package wayland

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

// Drag and drop on the fake compositor (SPEC §8.11 "Data exchange").

func acceptCopy(ui.DropEvent) ui.DropActions { return ui.DropActionCopy }

func dropPhases(events []ui.DropEvent) []ui.DropPhase {
	out := make([]ui.DropPhase, len(events))
	for i, e := range events {
		out[i] = e.Phase
	}
	return out
}

func samePhases(a, b []ui.DropPhase) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestADragOverTheWindowReachesTheHandler(t *testing.T) {
	e := startInput(t, capPointer, 0)
	e.h.setAnswer(acceptCopy)
	data := map[string][]byte{"text/uri-list": []byte("file:///tmp/a\r\n"), "text/plain;charset=utf-8": []byte("hello")}

	e.srv.DragEnter(fakecomp.DndCopy|fakecomp.DndMove, data, 10.5, 20)
	e.srv.DragMotion(30, 40)
	e.settle()
	events := e.h.dropEvents()
	if !samePhases(dropPhases(events), []ui.DropPhase{ui.DropPhaseEnter, ui.DropPhaseMove}) {
		t.Fatalf("events = %+v", events)
	}
	enter := events[0]
	if enter.ViewID != 0 || enter.Position != (ui.Offset{Dx: 10.5, Dy: 20}) || enter.Actions != ui.DropActionCopy|ui.DropActionMove ||
		strings.Join(enter.Types, ",") != "text/plain;charset=utf-8,text/uri-list" || enter.LocalID != 0 {
		t.Errorf("enter = %+v", enter)
	}
	if events[1].Position != (ui.Offset{Dx: 30, Dy: 40}) {
		t.Errorf("move at %v", events[1].Position)
	}
	// The app accepted the copy: the source is told the type and the action.
	reqs := e.srv.Joined()
	for _, want := range []string{"wl_data_offer.accept serial=101 mime=text/uri-list", "wl_data_offer.set_actions actions=3 preferred=1"} {
		if !strings.Contains(reqs, want) {
			t.Errorf("no %q in\n%s", want, reqs)
		}
	}

	e.srv.DragLeave()
	e.settle()
	if got := dropPhases(e.h.dropEvents()); got[len(got)-1] != ui.DropPhaseLeave {
		t.Errorf("phases = %v, want the leave last", got)
	}
	if n := len(e.srv.RequestsWith("wl_data_offer.destroy")); n < 1 {
		t.Error("the offer of the drag was not destroyed after the leave")
	}
}

func TestADragTheAppRefusesIsNotAccepted(t *testing.T) {
	e := startInput(t, capPointer, 0)
	e.srv.DragEnter(fakecomp.DndCopy, map[string][]byte{"text/plain": []byte("x")}, 1, 1)
	e.settle()
	reqs := e.srv.Joined()
	if !strings.Contains(reqs, "wl_data_offer.accept serial=101 mime=") || strings.Contains(reqs, "wl_data_offer.accept serial=101 mime=text") {
		t.Errorf("the refusal is accept with no type:\n%s", reqs)
	}
	if !strings.Contains(reqs, "wl_data_offer.set_actions actions=0 preferred=0") {
		t.Errorf("a refused drag has no actions:\n%s", reqs)
	}
	e.srv.DragDrop()
	e.settle()
	events := e.h.dropEvents()
	if last := events[len(events)-1]; last.Phase != ui.DropPhaseDrop || last.Text != "" || len(last.URIs) != 0 {
		t.Errorf("the refused drop = %+v: nothing is read", last)
	}
	if strings.Contains(e.srv.Joined(), "wl_data_offer.finish") || strings.Contains(e.srv.Joined(), "wl_data_offer.receive") {
		t.Error("a refused drop was read or finished")
	}
}

func TestADropGivesTheAppTheData(t *testing.T) {
	e := startInput(t, capPointer, 0)
	e.h.setAnswer(func(ui.DropEvent) ui.DropActions { return ui.DropActionMove })
	data := map[string][]byte{
		"text/uri-list":            []byte("# comment\r\nfile:///tmp/a%20b.txt\r\nfile:///tmp/c\r\n"),
		"text/plain;charset=utf-8": []byte("héllo"),
		"image/png":                []byte("not read"),
	}
	e.srv.DragEnter(fakecomp.DndMove, data, 5, 6)
	e.srv.DragDrop()
	waitUntil(t, func() bool { return len(e.h.dropEvents()) == 2 }, "the drop never reached the handler")
	drop := e.h.dropEvents()[1]
	if drop.Phase != ui.DropPhaseDrop || drop.Text != "héllo" || len(drop.URIs) != 2 || drop.URIs[0] != "file:///tmp/a%20b.txt" || drop.URIs[1] != "file:///tmp/c" {
		t.Errorf("drop = %+v", drop)
	}
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_data_offer.finish")) == 1 }, "the drop was not finished")
	if got := e.srv.RequestsWith("wl_data_offer.receive"); len(got) != 2 {
		t.Errorf("receive requests = %v, want the text and the list of URIs only", got)
	}
	// The leave that follows a drop changes nothing.
	e.srv.DragLeave()
	e.settle()
	if n := len(e.h.dropEvents()); n != 2 {
		t.Errorf("%d events after the leave that follows the drop", n)
	}
}

func TestADropAnsweredWithNoneIsNotFinished(t *testing.T) {
	e := startInput(t, capPointer, 0)
	e.h.setAnswer(func(ev ui.DropEvent) ui.DropActions {
		if ev.Phase == ui.DropPhaseDrop {
			return ui.DropActionNone // the data arrived, and the app does not want it after all
		}
		return ui.DropActionCopy
	})
	e.srv.DragEnter(fakecomp.DndCopy, map[string][]byte{"text/plain;charset=utf-8": []byte("x")}, 1, 1)
	e.srv.DragDrop()
	waitUntil(t, func() bool { return len(e.h.dropEvents()) == 2 }, "no drop")
	e.settle()
	if len(e.srv.RequestsWith("wl_data_offer.finish")) != 0 {
		t.Error("a drop the app answered with none was finished")
	}
}

func TestADragOfTheAppItselfCarriesItsRequestID(t *testing.T) {
	e := startInput(t, capPointer, 0)
	e.h.setAnswer(acceptCopy)
	marker := localMarker(77)
	other := localMarkerPrefix + "1-77" // another process's drag with the same id
	if os.Getpid() == 1 {
		other = localMarkerPrefix + "2-77"
	}
	e.srv.DragEnter(fakecomp.DndCopy, map[string][]byte{"text/plain": []byte("x"), marker: nil}, 1, 1)
	e.settle()
	enter := e.h.dropEvents()[0]
	if enter.LocalID != 77 || strings.Contains(strings.Join(enter.Types, ","), localMarkerPrefix) {
		t.Errorf("enter = %+v: want the local id and no marker among the types", enter)
	}
	e.srv.DragLeave()
	e.srv.DragEnter(fakecomp.DndCopy, map[string][]byte{"text/plain": []byte("x"), other: nil}, 1, 1)
	e.settle()
	if got := e.h.dropEvents(); got[len(got)-1].LocalID != 0 {
		t.Errorf("the drag of another process is local: %+v", got[len(got)-1])
	}
	e.srv.DragLeave()
	e.settle()
}

func TestLocalMarkers(t *testing.T) {
	pid := strconv.Itoa(os.Getpid())
	tests := []struct {
		mime string
		want uint64
	}{
		{localMarker(5), 5},
		{localMarkerPrefix + pid + "-12", 12},
		{localMarkerPrefix + "0-12", 0},
		{localMarkerPrefix + pid + "-x", 0},
		{localMarkerPrefix + pid, 0},
		{"text/plain", 0},
		{"", 0},
	}
	for _, tt := range tests {
		if got := localIDOf(tt.mime); got != tt.want {
			t.Errorf("localIDOf(%q) = %d, want %d", tt.mime, got, tt.want)
		}
	}
}

// pressButton plays the press that starts a drag: the pointer enters the window and
// the button goes down.
func pressButton(e *keyEnv) {
	e.srv.PointerEnter(e.surface, 5, 5)
	e.srv.PointerFrame()
	e.srv.PointerButton(0x110, true)
	e.srv.PointerFrame()
	e.settle()
}

func TestTheAppStartsADrag(t *testing.T) {
	e := startInput(t, capPointer, 0)
	pressButton(e)
	req := ui.DragRequest{ID: 9, Text: "some text", URIs: []string{"file:///tmp/a"}, Actions: ui.DropActionCopy | ui.DropActionMove}
	if err := e.onLoopErr(func() error { return e.p.windows[0].StartDrag(req) }); err != nil {
		t.Fatal(err)
	}
	e.settle()
	reqs := e.srv.Joined()
	for _, want := range []string{"wl_data_source.offer mime=text/uri-list", "wl_data_source.offer mime=" + localMarker(9),
		"wl_data_source.offer mime=text/plain;charset=utf-8", "wl_data_source.set_actions actions=3", "wl_data_device.start_drag"} {
		if !strings.Contains(reqs, want) {
			t.Errorf("no %q in\n%s", want, reqs)
		}
	}
	if !strings.Contains(reqs, "icon=true") {
		t.Errorf("the drag has no icon:\n%s", reqs)
	}
	if got, err := e.srv.ReadDragSource("text/plain;charset=utf-8"); err != nil || string(got) != "some text" {
		t.Errorf("text = %q, %v", got, err)
	}
	if got, err := e.srv.ReadDragSource("text/uri-list"); err != nil || string(got) != "file:///tmp/a\r\n" {
		t.Errorf("uris = %q, %v", got, err)
	}

	// A second drag has to wait.
	err := e.onLoopErr(func() error { return e.p.windows[0].StartDrag(ui.DragRequest{ID: 10, Text: "x"}) })
	assertRoute(t, err, "DragInProgress")

	e.srv.DragSourceAction(fakecomp.DndMove)
	e.srv.DragSourceDropPerformed()
	e.srv.DragSourceFinished()
	waitUntil(t, func() bool { return len(e.h.dragResults()) == 1 }, "the end of the drag never reached the handler")
	if got := e.h.dragResults()[0]; got.ID != 9 || got.Action != ui.DropActionMove {
		t.Errorf("result = %+v", got)
	}
	// The next drag can start.
	if err := e.onLoopErr(func() error { return e.p.windows[0].StartDrag(ui.DragRequest{ID: 11, Text: "y"}) }); err != nil {
		t.Errorf("the drag after the first: %v", err)
	}
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_data_device.start_drag")) == 2 }, "the second drag never started")
	e.srv.DragSourceCancelled()
	waitUntil(t, func() bool { return len(e.h.dragResults()) == 2 }, "the cancel never reached the handler")
	if got := e.h.dragResults()[1]; got.ID != 11 || got.Action != ui.DropActionNone {
		t.Errorf("result = %+v", got)
	}
}

func TestADragNeedsAButtonAndACompositorThatCan(t *testing.T) {
	e := startInput(t, capPointer, 0)
	err := e.onLoopErr(func() error { return e.p.windows[0].StartDrag(ui.DragRequest{ID: 1, Text: "x"}) })
	assertRoute(t, err, "DragNoButton")

	globals := fakecomp.DefaultGlobals()
	kept := globals[:0]
	for _, g := range globals {
		if g.Interface != "wl_data_device_manager" {
			kept = append(kept, g)
		}
	}
	e = startKeyboardNoFocus(t, fakecomp.Config{Capabilities: capPointer, Globals: kept})
	err = e.onLoopErr(func() error { return e.p.windows[0].StartDrag(ui.DragRequest{ID: 1, Text: "x"}) })
	assertRoute(t, err, "DragUnsupported")
}

func TestAPipeThatCannotBeMadeFailsTheRead(t *testing.T) {
	e := startInput(t, capPointer, 0)
	err := e.onLoopErr(func() error {
		e.p.clip.dnd.pipe = func() (*os.File, *os.File, error) { return nil, nil, errors.New(tagPipeBroken, "no descriptors left") }
		_, err := e.p.clip.dnd.receivePipe(&offer{receive: func(string, int) {}}, "text/plain")
		return err
	})
	assertRoute(t, err, "dnd.receivePipe.DropReadFailed")
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagPipeBroken = "PipeBroken"
)
