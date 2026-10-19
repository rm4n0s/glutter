package widgets_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

func target() Widget { return NewColoredBox(ui.NewColor(0xFF336699), newBox(100, 100)) }

func TestGestureDetectorCallbacks(t *testing.T) {
	var log []string
	add := func(s string) func() { return func() { log = append(log, s) } }
	tests := []struct {
		name string
		opts GestureDetectorOpts
		act  func(ts *gluttertest.Tester, at ui.Offset)
		want []string
	}{
		{"tap", GestureDetectorOpts{OnTap: add("tap"), OnTapDown: func(gestures.TapDownDetails) { log = append(log, "down") },
			OnTapUp: func(gestures.TapUpDetails) { log = append(log, "up") }}, func(ts *gluttertest.Tester, at ui.Offset) { ts.TapAt(at) }, []string{"down", "up", "tap"}},
		{"double tap", GestureDetectorOpts{OnDoubleTap: add("double")}, func(ts *gluttertest.Tester, at ui.Offset) {
			ts.TapAt(at)
			ts.Pump(50 * time.Millisecond)
			ts.TapAt(at)
		}, []string{"double"}},
		{"long press", GestureDetectorOpts{OnLongPress: add("long")}, func(ts *gluttertest.Tester, at ui.Offset) { ts.LongPressAt(at) }, []string{"long"}},
		{"secondary tap", GestureDetectorOpts{OnSecondaryTap: add("secondary")}, func(ts *gluttertest.Tester, at ui.Offset) {
			ts.TapAt(at, gluttertest.GestureOpts{Buttons: foundation.Some(gestures.SecondaryButton)})
		}, []string{"secondary"}},
		{"tap with only a secondary callback ignores the primary button", GestureDetectorOpts{OnSecondaryTap: add("secondary")},
			func(ts *gluttertest.Tester, at ui.Offset) { ts.TapAt(at) }, nil},
		{"horizontal drag", GestureDetectorOpts{OnHorizontalDragStart: func(gestures.DragStartDetails) { log = append(log, "start") },
			OnHorizontalDragEnd: func(gestures.DragEndDetails) { log = append(log, "end") }},
			func(ts *gluttertest.Tester, at ui.Offset) { ts.DragFrom(at, ui.Offset{Dx: 50}) }, []string{"start", "end"}},
		{"vertical drag ignores horizontal moves", GestureDetectorOpts{OnVerticalDragStart: func(gestures.DragStartDetails) { log = append(log, "start") }},
			func(ts *gluttertest.Tester, at ui.Offset) { ts.DragFrom(at, ui.Offset{Dx: 50}) }, nil},
		{"pan", GestureDetectorOpts{OnPanStart: func(gestures.DragStartDetails) { log = append(log, "pan") }},
			func(ts *gluttertest.Tester, at ui.Offset) { ts.DragFrom(at, ui.Offset{Dx: 60, Dy: 60}) }, []string{"pan"}},
		{"scale", GestureDetectorOpts{OnScaleStart: func(gestures.ScaleStartDetails) { log = append(log, "scale") }},
			func(ts *gluttertest.Tester, at ui.Offset) { ts.DragFrom(at, ui.Offset{Dx: 80}) }, []string{"scale"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log = nil
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, NewGestureDetector(target(), tt.opts), ui.TextDirectionLTR)
			tt.act(ts, ts.Center(gluttertest.ByType[*GestureDetector]()))
			ts.PumpAndSettle()
			if !reflect.DeepEqual(dedupe(log), tt.want) {
				t.Errorf("log = %v, want %v", log, tt.want)
			}
		})
	}
}

func dedupe(in []string) []string {
	var out []string
	for _, s := range in {
		if len(out) == 0 || out[len(out)-1] != s {
			out = append(out, s)
		}
	}
	return out
}

func TestGestureDetectorOnlyTheTopmostWins(t *testing.T) {
	var log []string
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewGestureDetector(NewGestureDetector(target(), GestureDetectorOpts{OnTap: func() { log = append(log, "inner") }}),
		GestureDetectorOpts{OnTap: func() { log = append(log, "outer") }}), ui.TextDirectionLTR)
	ts.TapAt(ts.Center(gluttertest.ByType[*box]()))
	if !reflect.DeepEqual(log, []string{"inner"}) {
		t.Errorf("log = %v, want [inner]", log)
	}
}

func TestGestureDetectorBehavior(t *testing.T) {
	tests := []struct {
		name     string
		child    Widget
		behavior rendering.HitTestBehavior
		at       ui.Offset
		want     int
	}{
		{"child hit", target(), rendering.HitTestBehaviorUnspecified, ui.Offset{Dx: 50, Dy: 50}, 1},
		{"outside the child is not hit by default", target(), rendering.HitTestBehaviorUnspecified, ui.Offset{Dx: 300, Dy: 300}, 0},
		{"no child fills the space and is translucent", nil, rendering.HitTestBehaviorUnspecified, ui.Offset{Dx: 5, Dy: 5}, 1},
		{"opaque", NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}), rendering.HitTestBehaviorOpaque, ui.Offset{Dx: 50, Dy: 50}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taps := 0
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, NewGestureDetector(tt.child, GestureDetectorOpts{OnTap: func() { taps++ }, Behavior: tt.behavior}), ui.TextDirectionLTR)
			ts.TapAt(tt.at, gluttertest.GestureOpts{})
			if taps != tt.want {
				t.Errorf("taps = %d, want %d", taps, tt.want)
			}
		})
	}
}

func TestGestureDetectorConflicts(t *testing.T) {
	skipUnlessDebug(t)
	drag := func(gestures.DragStartDetails) {}
	update := func(gestures.ScaleStartDetails) {}
	tests := []struct {
		name string
		opts GestureDetectorOpts
	}{
		{"pan and scale", GestureDetectorOpts{OnPanStart: drag, OnScaleStart: update}},
		{"vertical, horizontal and pan", GestureDetectorOpts{OnVerticalDragStart: drag, OnHorizontalDragStart: drag, OnPanStart: drag}},
		{"vertical, horizontal and scale", GestureDetectorOpts{OnVerticalDragStart: drag, OnHorizontalDragStart: drag, OnScaleStart: update}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := panicOf(func() { NewGestureDetector(nil, tt.opts) })
			if err == nil {
				t.Fatal("no panic")
			}
			_ = requireErrorTag(t, err.(error), "GestureDetectorConflict")
		})
	}
}

func TestRawGestureDetectorKeepsRecognizersBetweenBuilds(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var created, initialized int
	factory := func() GestureRecognizerFactory {
		return NewGestureRecognizerFactoryWithHandlers(
			func() *gestures.TapGestureRecognizer {
				created++
				return gestures.NewTapGestureRecognizer(ts.Binding.Gestures())
			},
			func(r *gestures.TapGestureRecognizer) { initialized++; r.OnTap = func() {} })
	}
	tapType := reflect.TypeFor[*gestures.TapGestureRecognizer]()
	detector := func(g map[reflect.Type]GestureRecognizerFactory) Widget {
		return NewRawGestureDetector(target(), RawGestureDetectorOpts{Gestures: g})
	}
	pumpLoose(ts, detector(map[reflect.Type]GestureRecognizerFactory{tapType: factory()}), ui.TextDirectionLTR)
	pumpLoose(ts, detector(map[reflect.Type]GestureRecognizerFactory{tapType: factory()}), ui.TextDirectionLTR)
	if created != 1 || initialized != 2 {
		t.Errorf("created = %d, initialized = %d, want 1 and 2", created, initialized)
	}
	state := gluttertest.StateOf[*RawGestureDetectorState](ts, gluttertest.ByType[*RawGestureDetector]())
	if got := len(state.Recognizers()); got != 1 {
		t.Errorf("%d recognizers, want 1", got)
	}
	pumpLoose(ts, detector(nil), ui.TextDirectionLTR)
	if got := len(state.Recognizers()); got != 0 {
		t.Errorf("%d recognizers after removing the gesture, want 0", got)
	}
}

func TestRawGestureDetectorFactoryTypeMustMatch(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	f := NewGestureRecognizerFactoryWithHandlers(
		func() *gestures.TapGestureRecognizer { return gestures.NewTapGestureRecognizer(ts.Binding.Gestures()) },
		func(*gestures.TapGestureRecognizer) {})
	pumpLoose(ts, NewRawGestureDetector(target(), RawGestureDetectorOpts{
		Gestures: map[reflect.Type]GestureRecognizerFactory{reflect.TypeFor[*gestures.LongPressGestureRecognizer](): f}}), ui.TextDirectionLTR)
	_ = requireErrorTag(t, ts.TakeError(), "GestureRecognizerFactoryTypeMismatch")
}

func semanticsHandlerOf(ts *gluttertest.Tester) *rendering.RenderSemanticsGestureHandler {
	for _, e := range ts.Elements(gluttertest.ByPredicate("any widget", func(Widget) bool { return true })) {
		if h, ok := e.RenderObject().(*rendering.RenderSemanticsGestureHandler); ok {
			return h
		}
	}
	return nil
}

func TestGestureDetectorSemanticsActions(t *testing.T) {
	var log []string
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewGestureDetector(target(), GestureDetectorOpts{
		OnTap:                func() { log = append(log, "tap") },
		OnTapDown:            func(gestures.TapDownDetails) { log = append(log, "down") },
		OnLongPress:          func() { log = append(log, "long") },
		OnHorizontalDragEnd:  func(gestures.DragEndDetails) { log = append(log, "hend") },
		OnVerticalDragUpdate: func(gestures.DragUpdateDetails) { log = append(log, "vupdate") },
	}), ui.TextDirectionLTR)
	h := semanticsHandlerOf(ts)
	if h == nil {
		t.Fatal("no semantics gesture handler")
	}
	h.OnTap()()
	h.OnLongPress()()
	h.OnHorizontalDragUpdate()(gestures.NewDragUpdateDetails(ui.Offset{}, gestures.DragUpdateOpts{Delta: ui.Offset{Dx: 5}}))
	h.OnVerticalDragUpdate()(gestures.NewDragUpdateDetails(ui.Offset{}, gestures.DragUpdateOpts{Delta: ui.Offset{Dy: 5}}))
	want := []string{"down", "tap", "long", "hend", "vupdate"}
	if !reflect.DeepEqual(log, want) {
		t.Errorf("log = %v, want %v", log, want)
	}
}

func TestGestureDetectorExcludeFromSemantics(t *testing.T) {
	tests := []struct {
		name    string
		exclude bool
		want    bool
	}{
		{"included", false, true},
		{"excluded", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, NewGestureDetector(target(), GestureDetectorOpts{OnTap: func() {}, ExcludeFromSemantics: tt.exclude}), ui.TextDirectionLTR)
			if got := semanticsHandlerOf(ts) != nil; got != tt.want {
				t.Errorf("has a handler = %v, want %v", got, tt.want)
			}
		})
	}
}
