package widgets_test

import (
	"context"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

func TestBuilderBuildsWithItsContext(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var got BuildContext
	pumpLoose(ts, NewBuilder(func(bc BuildContext) Widget { got = bc; return newBox(7, 8) }), ui.TextDirectionLTR)
	if got == nil || got.Widget() == nil {
		t.Fatal("the builder got no context")
	}
	if _, ok := got.Widget().(*Builder); !ok {
		t.Errorf("the context belongs to %T, want *Builder", got.Widget())
	}
	if s := ts.Size(gluttertest.ByType[*box]()); s != (ui.Size{Width: 7, Height: 8}) {
		t.Errorf("size = %v", s)
	}
}

func TestStatefulBuilderRebuildsOnSetState(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	n := 0
	var set StateSetter
	pumpLoose(ts, NewStatefulBuilder(func(bc BuildContext, setState StateSetter) Widget {
		set = setState
		return newBox(float64(10+n), 10)
	}), ui.TextDirectionLTR)
	if w := ts.Size(gluttertest.ByType[*box]()).Width; w != 10 {
		t.Fatalf("width = %v", w)
	}
	set(func() { n = 5 })
	ts.Pump()
	if w := ts.Size(gluttertest.ByType[*box]()).Width; w != 15 {
		t.Errorf("width after SetState = %v, want 15", w)
	}
}

func TestValueListenableBuilder(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	n := foundation.NewValueNotifier(1)
	builds, childBuilds := 0, 0
	child := builderOf(func(BuildContext) Widget { childBuilds++; return newLeaf() })
	w := NewValueListenableBuilder[int](n, func(bc BuildContext, v int, c Widget) Widget {
		builds++
		return NewColumn([]Widget{newBox(float64(v), 5), c})
	}, child)
	pumpLoose(ts, w, ui.TextDirectionLTR)
	if got := ts.Count(gluttertest.ByType[*box]()); got != 2 {
		t.Fatalf("%d boxes, want the built one and the child", got)
	}
	n.SetValue(4)
	ts.Pump()
	if builds != 2 || childBuilds != 1 {
		t.Errorf("builds = %d, child builds = %d, want 2 and 1", builds, childBuilds)
	}
	// A different listenable replaces the old one.
	m := foundation.NewValueNotifier(9)
	pumpLoose(ts, NewValueListenableBuilder[int](m, func(bc BuildContext, v int, c Widget) Widget { builds++; return newBox(float64(v), 5) }, nil), ui.TextDirectionLTR)
	before := builds
	n.SetValue(5)
	ts.Pump()
	if builds != before {
		t.Error("the old listenable still rebuilds")
	}
	m.SetValue(10)
	ts.Pump()
	if builds != before+1 {
		t.Error("the new listenable does not rebuild")
	}
	if w := ts.Size(gluttertest.ByType[*box]()).Width; w != 10 {
		t.Errorf("width = %v, want 10", w)
	}
}

func TestListenableAndAnimatedBuilder(t *testing.T) {
	tests := []struct {
		name string
		mk   func(l foundation.Listenable, b TransitionBuilder, child Widget) Widget
	}{
		{"listenable builder", func(l foundation.Listenable, b TransitionBuilder, c Widget) Widget {
			return NewListenableBuilder(l, b, c)
		}},
		{"animated builder", func(l foundation.Listenable, b TransitionBuilder, c Widget) Widget {
			return NewAnimatedBuilder(l, b, c)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			n := foundation.NewChangeNotifier()
			builds := 0
			w := tt.mk(n, func(bc BuildContext, c Widget) Widget { builds++; return c }, newLeaf())
			pumpLoose(ts, w, ui.TextDirectionLTR)
			n.NotifyListeners()
			n.NotifyListeners()
			ts.Pump()
			if builds != 2 {
				t.Errorf("builds = %d, want 2 (the notifications of one frame build once)", builds)
			}
			pumpLoose(ts, NewSizedBox(nil), ui.TextDirectionLTR)
			if n.HasListeners() {
				t.Error("the builder is still listening after it left the tree")
			}
		})
	}
}

// waitFor pumps the tester until the condition holds, as the goroutines of the
// async builders hand their results over to the UI thread.
func waitFor(t *testing.T, ts *gluttertest.Tester, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		ts.Pump()
		time.Sleep(time.Millisecond)
	}
}

func TestFutureBuilder(t *testing.T) {
	boom := errors.New("Boom", "boom")
	tests := []struct {
		name      string
		fn        func(context.Context) (string, error)
		initial   foundation.Opt[string]
		wantFirst string
		wantLast  string
	}{
		{"value", func(context.Context) (string, error) { return "ok", nil }, foundation.None[string](),
			"ConnectionState.waiting no-data", "ConnectionState.done ok"},
		{"error", func(context.Context) (string, error) { return "", boom }, foundation.None[string](),
			"ConnectionState.waiting no-data", "ConnectionState.done error boom"},
		{"initial data", func(context.Context) (string, error) { return "new", nil }, foundation.Some("old"),
			"ConnectionState.waiting old", "ConnectionState.done new"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			gate := make(chan struct{})
			var seen []string
			describe := func(s AsyncSnapshot[string]) string {
				switch {
				case s.HasError():
					return s.ConnectionState().String() + " error " + s.Error().Error()
				case s.HasData():
					return s.ConnectionState().String() + " " + s.Data()
				}
				return s.ConnectionState().String() + " no-data"
			}
			pumpLoose(ts, NewFutureBuilder(func(ctx context.Context) (string, error) { <-gate; return tt.fn(ctx) },
				func(bc BuildContext, s AsyncSnapshot[string]) Widget {
					seen = append(seen, describe(s))
					return newLeaf()
				},
				FutureBuilderOpts[string]{InitialData: tt.initial}), ui.TextDirectionLTR)
			if seen[0] != tt.wantFirst {
				t.Errorf("first snapshot = %q, want %q", seen[0], tt.wantFirst)
			}
			close(gate)
			waitFor(t, ts, func() bool { return seen[len(seen)-1] == tt.wantLast })
		})
	}
}

func TestFutureBuilderIdentityRestartsAndNilStops(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	starts := 0
	var last AsyncSnapshot[int]
	mk := func(identity any, fn bool) Widget {
		var f func(context.Context) (int, error)
		if fn {
			f = func(context.Context) (int, error) { starts++; return starts, nil }
		}
		return NewFutureBuilder(f, func(bc BuildContext, s AsyncSnapshot[int]) Widget { last = s; return newLeaf() }, FutureBuilderOpts[int]{Identity: identity})
	}
	pumpLoose(ts, mk("a", true), ui.TextDirectionLTR)
	waitFor(t, ts, func() bool { return last.ConnectionState() == ConnectionStateDone })
	pumpLoose(ts, mk("a", true), ui.TextDirectionLTR)
	ts.Pump()
	if starts != 1 {
		t.Errorf("starts = %d after an update with the same identity, want 1", starts)
	}
	pumpLoose(ts, mk("b", true), ui.TextDirectionLTR)
	waitFor(t, ts, func() bool { return last.HasData() && last.Data() == 2 })
	if starts != 2 {
		t.Errorf("starts = %d after a new identity, want 2", starts)
	}
	pumpLoose(ts, mk("b", false), ui.TextDirectionLTR)
	if last.ConnectionState() != ConnectionStateNone {
		t.Errorf("state without a function = %v, want none", last.ConnectionState())
	}
}

func TestFutureBuilderCancelsItsContext(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	started, cancelled := make(chan struct{}), make(chan struct{})
	pumpLoose(ts, NewFutureBuilder(func(ctx context.Context) (int, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return 0, ctx.Err()
	}, func(BuildContext, AsyncSnapshot[int]) Widget { return newLeaf() }), ui.TextDirectionLTR)
	<-started
	pumpLoose(ts, newLeaf(), ui.TextDirectionLTR)
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("the context was not cancelled when the widget left the tree")
	}
}

func TestStreamBuilder(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ch := make(chan int)
	var states []ConnectionState
	var data []int
	pumpLoose(ts, NewStreamBuilder(ch, func(bc BuildContext, s AsyncSnapshot[int]) Widget {
		states = append(states, s.ConnectionState())
		if s.HasData() {
			data = append(data, s.Data())
		}
		return newLeaf()
	}, StreamBuilderOpts[int]{InitialData: foundation.Some(0)}), ui.TextDirectionLTR)
	if states[0] != ConnectionStateWaiting || data[0] != 0 {
		t.Fatalf("first snapshot = %v %v", states[0], data)
	}
	ch <- 1
	waitFor(t, ts, func() bool { return len(data) > 1 && data[len(data)-1] == 1 })
	ch <- 2
	waitFor(t, ts, func() bool { return data[len(data)-1] == 2 })
	if states[len(states)-1] != ConnectionStateActive {
		t.Errorf("state = %v, want active", states[len(states)-1])
	}
	close(ch)
	waitFor(t, ts, func() bool { return states[len(states)-1] == ConnectionStateDone })
	if data[len(data)-1] != 2 {
		t.Errorf("the last data = %d after the stream closed, want 2", data[len(data)-1])
	}
}

func TestStreamBuilderSwitchesStreams(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	a, b := make(chan string), make(chan string)
	var last AsyncSnapshot[string]
	mk := func(c chan string) Widget {
		return NewStreamBuilder(c, func(bc BuildContext, s AsyncSnapshot[string]) Widget { last = s; return newLeaf() })
	}
	pumpLoose(ts, mk(a), ui.TextDirectionLTR)
	pumpLoose(ts, mk(b), ui.TextDirectionLTR)
	b <- "from b"
	waitFor(t, ts, func() bool { return last.HasData() && last.Data() == "from b" })
	// The old stream is not listened to any more.
	select {
	case a <- "from a":
		t.Error("the old stream still has a reader")
	case <-time.After(20 * time.Millisecond):
	}
	pumpLoose(ts, NewStreamBuilder[string](nil, func(bc BuildContext, s AsyncSnapshot[string]) Widget { last = s; return newLeaf() }), ui.TextDirectionLTR)
	if last.ConnectionState() != ConnectionStateNone {
		t.Errorf("state without a stream = %v, want none", last.ConnectionState())
	}
}

func TestAsyncSnapshot(t *testing.T) {
	boom := errors.New("Boom", "boom")
	data := NewAsyncSnapshotWithData(ConnectionStateDone, 3)
	failed := NewAsyncSnapshotWithError[int](ConnectionStateDone, boom)
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"data", data.Data(), 3},
		{"has data", data.HasData(), true},
		{"no error", data.HasError(), false},
		{"error", failed.Error(), boom},
		{"has error", failed.HasError(), true},
		{"nothing state", NewAsyncSnapshotNothing[int]().ConnectionState(), ConnectionStateNone},
		{"waiting state", NewAsyncSnapshotWaiting[int]().ConnectionState(), ConnectionStateWaiting},
		{"in state", data.InState(ConnectionStateActive).ConnectionState(), ConnectionStateActive},
		{"require data", data.RequireData(), 3},
		{"string", data.String(), "AsyncSnapshot(ConnectionState.done, 3, <nil>)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
	if p := panicOf(func() { failed.RequireData() }); p != boom {
		t.Errorf("RequireData panicked with %v, want the error", p)
	}
	err := panicOf(func() { NewAsyncSnapshotNothing[int]().RequireData() })
	if err == nil {
		t.Fatal("no panic for the empty snapshot")
	}
	_ = requireErrorTag(t, err.(error), "AsyncSnapshotEmpty")
}

func TestLayoutBuilderGetsTheConstraints(t *testing.T) {
	tests := []struct {
		name string
		w    Widget
		want rendering.BoxConstraints
	}{
		{"tight", NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(120.0), Height: foundation.Some(80.0)}), rendering.NewBoxConstraintsTight(ui.Size{Width: 120, Height: 80})},
		{"loose", NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(50.0), MaxHeight: foundation.Some(60.0)}), nil),
			rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(50.0), MaxHeight: foundation.Some(60.0)})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var got rendering.BoxConstraints
			lb := NewLayoutBuilder(func(bc BuildContext, c rendering.BoxConstraints) Widget { got = c; return newBox(10, 10) })
			var wrapped Widget
			switch outer := tt.w.(type) {
			case *SizedBox:
				wrapped = NewSizedBox(lb, SizedBoxOpts{Width: outer.Width(), Height: outer.Height()})
			default:
				wrapped = NewConstrainedBox(tt.want, lb)
			}
			pumpLoose(ts, wrapped, ui.TextDirectionLTR)
			if got != tt.want {
				t.Errorf("constraints = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLayoutBuilderRebuildsOnlyWhenNeeded(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	builds := 0
	mk := func(width float64, builder LayoutWidgetBuilder) Widget {
		return NewSizedBox(NewLayoutBuilder(builder), SizedBoxOpts{Width: foundation.Some(width), Height: foundation.Some(10.0)})
	}
	same := func(bc BuildContext, c rendering.BoxConstraints) Widget {
		builds++
		return NewAlign(newBox(c.MaxWidth/2, 5), AlignOpts{Alignment: painting.AlignmentTopLeft})
	}
	pumpLoose(ts, mk(100, same), ui.TextDirectionLTR)
	if builds != 1 {
		t.Fatalf("builds = %d", builds)
	}
	if w := ts.Size(gluttertest.ByType[*box]()).Width; w != 50 {
		t.Errorf("child width = %v, want 50", w)
	}
	pumpLoose(ts, mk(200, same), ui.TextDirectionLTR)
	if builds != 2 {
		t.Errorf("builds = %d after new constraints, want 2", builds)
	}
	if w := ts.Size(gluttertest.ByType[*box]()).Width; w != 100 {
		t.Errorf("child width = %v, want 100", w)
	}
}

func TestLayoutBuilderRebuildsWhenItsStateChanges(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	n := foundation.NewValueNotifier(10.0)
	pumpLoose(ts, NewSizedBox(NewLayoutBuilder(func(bc BuildContext, c rendering.BoxConstraints) Widget {
		return NewValueListenableBuilder[float64](n, func(bc BuildContext, v float64, _ Widget) Widget {
			return NewAlign(newBox(v, 5), AlignOpts{Alignment: painting.AlignmentTopLeft})
		}, nil)
	}), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(10.0)}), ui.TextDirectionLTR)
	n.SetValue(33)
	ts.Pump()
	if w := ts.Size(gluttertest.ByType[*box]()).Width; w != 33 {
		t.Errorf("child width = %v, want 33", w)
	}
}

func TestLayoutBuilderNilIsAnError(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewSizedBox(NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget { return nil }),
		SizedBoxOpts{Width: foundation.Some(10.0), Height: foundation.Some(10.0)}), ui.TextDirectionLTR)
	_ = requireErrorTag(t, ts.TakeError(), "BuildReturnedNil")
}

func TestOrientationBuilder(t *testing.T) {
	tests := []struct {
		name          string
		width, height float64
		want          Orientation
	}{
		{"wide", 200, 100, OrientationLandscape},
		{"tall", 100, 200, OrientationPortrait},
		{"square", 100, 100, OrientationPortrait},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var got Orientation
			pumpLoose(ts, NewSizedBox(NewOrientationBuilder(func(bc BuildContext, o Orientation) Widget { got = o; return newLeaf() }),
				SizedBoxOpts{Width: foundation.Some(tt.width), Height: foundation.Some(tt.height)}), ui.TextDirectionLTR)
			if got != tt.want {
				t.Errorf("orientation = %v, want %v", got, tt.want)
			}
		})
	}
}

type scrolled struct{ by float64 }

func TestNotificationListener(t *testing.T) {
	tests := []struct {
		name    string
		handle  bool
		outer   bool
		wantIn  int
		wantOut int
	}{
		{"handled stops the bubbling", true, true, 1, 0},
		{"unhandled bubbles on", false, true, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var in, out int
			ts := gluttertest.New(t, gluttertest.Opts{})
			var target BuildContext
			pumpLoose(ts, NewNotificationListener(func(n scrolled) bool { out++; return true },
				NewNotificationListener(
					func(n scrolled) bool { in++; return tt.handle },
					NewBuilder(func(bc BuildContext) Widget { target = bc; return newLeaf() }))), ui.TextDirectionLTR)
			DispatchNotification(target, scrolled{3})
			if in != tt.wantIn || out != tt.wantOut {
				t.Errorf("inner = %d, outer = %d, want %d and %d", in, out, tt.wantIn, tt.wantOut)
			}
		})
	}
}

func TestNotificationListenerTypeFilter(t *testing.T) {
	var layouts, scrolls int
	ts := gluttertest.New(t, gluttertest.Opts{})
	var target BuildContext
	pumpLoose(ts, NewNotificationListener(func(scrolled) bool { scrolls++; return false }, NewNotificationListener(
		func(LayoutChangedNotification) bool { layouts++; return false },
		NewBuilder(func(bc BuildContext) Widget { target = bc; return newLeaf() }))), ui.TextDirectionLTR)
	DispatchNotification(target, LayoutChangedNotification{})
	DispatchNotification(target, scrolled{1})
	DispatchNotification(nil, scrolled{1})
	// A listener without a callback lets everything through.
	pumpLoose(ts, NewNotificationListener[scrolled](nil, NewBuilder(func(bc BuildContext) Widget { target = bc; return newLeaf() })), ui.TextDirectionLTR)
	DispatchNotification(target, scrolled{1})
	if layouts != 1 || scrolls != 1 {
		t.Errorf("layouts = %d, scrolls = %d, want 1 and 1", layouts, scrolls)
	}
}

func TestNotificationListenerUpdatesItsCallback(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var target BuildContext
	got := ""
	mk := func(label string) Widget {
		return NewNotificationListener(func(scrolled) bool { got = label; return true }, NewBuilder(func(bc BuildContext) Widget { target = bc; return newLeaf() }))
	}
	pumpLoose(ts, mk("a"), ui.TextDirectionLTR)
	pumpLoose(ts, mk("b"), ui.TextDirectionLTR)
	DispatchNotification(target, scrolled{})
	if got != "b" {
		t.Errorf("got %q, want b", got)
	}
}
