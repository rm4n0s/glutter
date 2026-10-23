package ui

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// windowHost is a fakeHost that can make windows.
type windowHost struct {
	fakeHost
	next    ViewID
	created []WindowSpec
	gone    []ViewID
	titles  map[ViewID]string
	sizes   map[ViewID]Size
	limits  map[ViewID][2]Size
	hidden  map[ViewID]bool
	focus   []ViewFocusEvent
	failure error
}

func (h *windowHost) CreateWindow(spec WindowSpec) (ViewID, ViewConfiguration, error) {
	if h.failure != nil {
		return 0, ViewConfiguration{}, h.failure
	}
	h.next++
	h.created = append(h.created, spec)
	return h.next, ViewConfiguration{PhysicalSize: spec.Size, DevicePixelRatio: 1}, nil
}
func (h *windowHost) DestroyWindow(v ViewID) { h.gone = append(h.gone, v) }
func (h *windowHost) SetWindowTitle(v ViewID, title string) {
	if h.titles == nil {
		h.titles = map[ViewID]string{}
	}
	h.titles[v] = title
}
func (h *windowHost) SetWindowSize(v ViewID, s Size) {
	if h.sizes == nil {
		h.sizes = map[ViewID]Size{}
	}
	h.sizes[v] = s
}
func (h *windowHost) SetWindowConstraints(v ViewID, min, max Size) {
	if h.limits == nil {
		h.limits = map[ViewID][2]Size{}
	}
	h.limits[v] = [2]Size{min, max}
}
func (h *windowHost) SetWindowHidden(v ViewID, hidden bool) {
	if h.hidden == nil {
		h.hidden = map[ViewID]bool{}
	}
	h.hidden[v] = hidden
}
func (h *windowHost) RequestViewFocus(v ViewID, s ViewFocusState, d ViewFocusDirection) {
	h.focus = append(h.focus, ViewFocusEvent{ViewID: v, State: s, Direction: d})
}

func TestCreateWindowWithoutAHostThatMakesWindows(t *testing.T) {
	tests := []struct {
		name string
		host PlatformHost
	}{
		{"no host", nil},
		{"a host that cannot make windows", &fakeHost{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewPlatformDispatcher(tt.host)
			_, err := d.CreateWindow(WindowSpec{})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute("PlatformDispatcher.CreateWindow.WindowingUnsupported") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			// The calls that need a window do nothing.
			d.SetWindowTitle(1, "x")
			d.SetWindowHidden(1, true)
			d.DestroyWindow(1)
			d.RequestViewFocusChange(1, ViewFocusStateFocused, ViewFocusDirectionForward)
		})
	}
}

func TestCreateAndDestroyWindow(t *testing.T) {
	h := &windowHost{}
	d := NewPlatformDispatcher(h)
	v, err := d.CreateWindow(WindowSpec{Title: "Second", Size: Size{Width: 400, Height: 300}, HasParent: true, Kind: WindowKindDialog})
	if err != nil {
		t.Fatal(err)
	}
	if v.ViewID() != 1 || d.View(1) != v || v.PhysicalSize() != (Size{Width: 400, Height: 300}) {
		t.Fatalf("view = %d, size %v", v.ViewID(), v.PhysicalSize())
	}
	if len(h.created) != 1 || h.created[0].Kind != WindowKindDialog || !h.created[0].HasParent {
		t.Errorf("created = %+v", h.created)
	}

	d.SetWindowTitle(1, "Renamed")
	d.SetWindowSize(1, Size{Width: 500, Height: 350})
	d.SetWindowConstraints(1, Size{Width: 100, Height: 50}, Size{Width: 900, Height: 700})
	if h.titles[1] != "Renamed" || h.sizes[1] != (Size{Width: 500, Height: 350}) || h.limits[1] != [2]Size{{100, 50}, {900, 700}} {
		t.Errorf("title %q size %v limits %v", h.titles[1], h.sizes[1], h.limits[1])
	}
	d.SetWindowHidden(1, true)
	if !h.hidden[1] || d.View(1) != v {
		t.Errorf("hidden %v, view %v: a hidden window keeps its view", h.hidden[1], d.View(1))
	}
	d.SetWindowHidden(1, false)
	if h.hidden[1] {
		t.Error("the window was not shown again")
	}

	d.DestroyWindow(1)
	d.DestroyWindow(1) // a second time does nothing
	if len(h.gone) != 1 || d.View(1) != nil || !v.Destroyed() {
		t.Errorf("gone %v, view %v, destroyed %v", h.gone, d.View(1), v.Destroyed())
	}
	if err := v.Render(&Scene{}); err != nil || len(h.rendered) != 0 {
		t.Errorf("rendering into a destroyed view: %v, %d scenes", err, len(h.rendered))
	}
}

func TestCreateWindowFailure(t *testing.T) {
	h := &windowHost{failure: errors.New(tagWaylandProtocolMissing, "no")}
	d := NewPlatformDispatcher(h)
	_, err := d.CreateWindow(WindowSpec{})
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("WaylandProtocolMissing") {
		t.Errorf("error = %v, want the host's own", err)
	}
	if len(d.Views()) != 0 {
		t.Error("a view was made for a window that failed")
	}
}

func TestWindowCloseRequestIsHandedToTheOwner(t *testing.T) {
	tests := []struct {
		name  string
		owner func(ViewID) bool
		exit  AppExitResponse
		want  bool
	}{
		{"no owner and no objection", nil, AppExitResponseExit, true},
		{"the owner takes it", func(ViewID) bool { return true }, AppExitResponseExit, false},
		{"the owner does not know the view", func(ViewID) bool { return false }, AppExitResponseExit, true},
		{"the app cancels the exit", nil, AppExitResponseCancel, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewPlatformDispatcher(&fakeHost{})
			d.OnWindowCloseRequested = tt.owner
			d.OnExitRequested = func(ViewID) AppExitResponse { return tt.exit }
			if got := d.RequestClose(0); got != tt.want {
				t.Errorf("RequestClose = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestViewFocus(t *testing.T) {
	h := &windowHost{}
	d := NewPlatformDispatcher(h)
	var got []ViewFocusEvent
	d.DispatchViewFocus(ViewFocusEvent{ViewID: 1}) // nobody listens yet
	d.OnViewFocusChange = func(e ViewFocusEvent) { got = append(got, e) }
	want := ViewFocusEvent{ViewID: 2, State: ViewFocusStateFocused, Direction: ViewFocusDirectionBackward}
	d.DispatchViewFocus(want)
	if len(got) != 1 || got[0] != want {
		t.Errorf("events = %+v, want %+v", got, want)
	}
	d.RequestViewFocusChange(2, ViewFocusStateFocused, ViewFocusDirectionForward)
	if len(h.focus) != 1 || h.focus[0] != (ViewFocusEvent{ViewID: 2, State: ViewFocusStateFocused, Direction: ViewFocusDirectionForward}) {
		t.Errorf("requests = %+v", h.focus)
	}
}

func TestARemovedViewRendersNothing(t *testing.T) {
	h := &fakeHost{}
	d := NewPlatformDispatcher(h)
	v := d.AddView(3, ViewConfiguration{})
	d.RemoveView(3)
	d.RemoveView(3) // twice is fine
	if !v.Destroyed() || d.View(3) != nil {
		t.Errorf("destroyed = %v, view = %v", v.Destroyed(), d.View(3))
	}
	if err := v.Render(&Scene{}); err != nil || len(h.rendered) != 0 {
		t.Errorf("render after remove: %v, %d scenes", err, len(h.rendered))
	}
}

func TestStartDragWithoutAHostThatCanFails(t *testing.T) {
	d := NewPlatformDispatcher(nil)
	appErr, ok := errors.FromError(d.StartDrag(0, DragRequest{ID: 1}))
	if !ok || !appErr.HasRoute("PlatformDispatcher.StartDrag.DragUnsupported") {
		t.Errorf("err = %v", appErr)
	}
}

func TestDropEventsWithoutHandlersAreHarmless(t *testing.T) {
	d := NewPlatformDispatcher(nil)
	if got := d.DispatchDrop(DropEvent{Phase: DropPhaseDrop}); got != DropActionNone {
		t.Errorf("DispatchDrop without a handler = %v", got)
	}
	d.DispatchDragEnd(DragResult{ID: 1}) // nothing to call, nothing to panic
	d.OnDrop = func(e DropEvent) DropActions { return DropActionLink }
	if got := d.DispatchDrop(DropEvent{}); got != DropActionLink {
		t.Errorf("DispatchDrop = %v", got)
	}
}

func TestDropActionsHas(t *testing.T) {
	tests := []struct {
		set, a DropActions
		want   bool
	}{
		{DropActionCopy | DropActionMove, DropActionCopy, true},
		{DropActionCopy, DropActionMove, false},
		{DropActionNone, DropActionCopy, false},
		{DropActionCopy, DropActionNone, false},
		{DropActionCopy | DropActionLink, DropActionCopy | DropActionLink, true},
	}
	for _, tt := range tests {
		if got := tt.set.Has(tt.a); got != tt.want {
			t.Errorf("%v.Has(%v) = %v", tt.set, tt.a, got)
		}
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWaylandProtocolMissing = "WaylandProtocolMissing"
)
