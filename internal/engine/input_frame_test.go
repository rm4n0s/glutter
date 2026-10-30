package engine

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/ui"
)

// frameRequests counts the frame requests the platform got for each view so far.
func frameRequests(e *windowsEngine) map[ui.ViewID]int {
	out := map[ui.ViewID]int{}
	for _, entry := range e.p.Log() {
		if entry.Kind == fake.EntryRequestFrame {
			out[entry.View]++
		}
	}
	return out
}

func keyDown() ui.KeyData { return ui.KeyData{Type: ui.KeyEventTypeDown, Logical: 'q'} }

func pointerDown() []ui.PointerData {
	return []ui.PointerData{{Change: ui.PointerChangeAdd}, {Change: ui.PointerChangeDown}}
}

// TestTheFrameThatHandlesInputAsksForNoSecondFrame checks that a frame request made
// while the frame flushes its input is dropped (SPEC §13), and that the requests that
// are still needed are kept.
func TestTheFrameThatHandlesInputAsksForNoSecondFrame(t *testing.T) {
	// setState is what a key or a click does to the framework: it schedules a frame.
	type setup struct {
		// input arrives before the frame.
		input func(e *windowsEngine)
		// onInput is what the framework does with the input (OnKeyData, OnPointerDataPacket).
		onInput func(e *windowsEngine)
		// onBegin and onDraw are the framework's frame callbacks.
		onBegin, onDraw func(e *windowsEngine)
		second          bool
	}
	schedule := func(e *windowsEngine) { e.d.ScheduleFrame() }
	key := func(e *windowsEngine) { e.h.OnKey(keyDown()) }
	click := func(e *windowsEngine) { e.h.OnPointer(pointerDown()) }
	tests := []struct {
		name  string
		s     setup
		first int // frame requests of view 0 after the frame, beyond those before it
		other int // the same for the second window
	}{
		{"a key that calls SetState", setup{input: key, onInput: schedule}, 0, 0},
		{"a click that calls SetState", setup{input: click, onInput: schedule}, 0, 0},
		{"a key that changes nothing", setup{input: key}, 0, 0},
		{"an animation that the key starts keeps its next frame", setup{input: key, onInput: schedule, onBegin: schedule}, 1, 0},
		{"a SetState in a post-frame callback keeps its frame", setup{input: key, onInput: schedule, onDraw: schedule}, 1, 0},
		{"another window's request is kept", setup{input: key, onInput: schedule, second: true}, 0, 1},
		{"an animation in a second window keeps both frames", setup{input: key, onInput: schedule, onBegin: schedule, second: true}, 1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newWindowsEngine(t)
			var second ui.ViewID
			if tt.s.second {
				v, err := e.d.CreateWindow(ui.WindowSpec{})
				if err != nil {
					t.Fatal(err)
				}
				second = v.ViewID()
				e.size(second, 200, 100)
			}
			e.size(0, 800, 600)
			e.renderAll(nil)
			e.h.OnFrame(0, time.Millisecond) // makes the renderer
			frames := e.log.begin
			begin, draw := e.d.OnBeginFrame, e.d.OnDrawFrame
			e.d.OnBeginFrame = func(d time.Duration) {
				begin(d)
				if tt.s.onBegin != nil {
					tt.s.onBegin(e)
				}
			}
			e.d.OnDrawFrame = func() {
				draw()
				if tt.s.onDraw != nil {
					tt.s.onDraw(e)
				}
			}
			e.d.OnKeyData = func(ui.KeyData) bool {
				if tt.s.onInput != nil {
					tt.s.onInput(e)
				}
				return true
			}
			e.d.OnPointerDataPacket = func(ui.PointerDataPacket) {
				if tt.s.onInput != nil {
					tt.s.onInput(e)
				}
			}
			tt.s.input(e)
			before := frameRequests(e)
			e.h.OnFrame(0, 2*time.Millisecond)
			after := frameRequests(e)
			if got := e.log.begin - frames; got != 1 {
				t.Errorf("frames run = %d, want 1", got)
			}
			if got := after[0] - before[0]; got != tt.first {
				t.Errorf("view 0: %d frame requests during the frame, want %d", got, tt.first)
			}
			if got := after[second] - before[second]; tt.s.second && got != tt.other {
				t.Errorf("second window: %d frame requests during the frame, want %d", got, tt.other)
			}
			if w := e.h.windows[0]; w.wanted != (tt.first > 0) {
				t.Errorf("wanted = %v after the frame, want %v", w.wanted, tt.first > 0)
			}
		})
	}
}

// TestAHeldKeyGivesOneFrameForEachRepeat is the case that was found: a key repeated
// N times, each repeat with a frame, asks for N frames, not 2N.
func TestAHeldKeyGivesOneFrameForEachRepeat(t *testing.T) {
	for _, n := range []int{1, 5, 50} {
		e := newWindowsEngine(t)
		e.size(0, 800, 600)
		e.renderAll(nil)
		e.d.OnKeyData = func(ui.KeyData) bool { e.d.ScheduleFrame(); return true }
		e.h.OnFrame(0, time.Millisecond)
		first := frameRequests(e)[0]
		frames := e.log.begin
		for i := 0; i < n; i++ {
			e.h.OnKey(keyDown())
			e.h.OnFrame(0, time.Duration(i+2)*time.Millisecond)
		}
		if got := e.log.begin - frames; got != n {
			t.Errorf("%d repeats: %d frames run", n, got)
		}
		if got := frameRequests(e)[0] - first; got != n {
			t.Errorf("%d repeats: %d frame requests, want %d (one for each key event)", n, got, n)
		}
	}
}

func TestInputThatArrivesBetweenFramesAsksForAFrame(t *testing.T) {
	e := newWindowsEngine(t)
	e.size(0, 800, 600)
	e.renderAll(nil)
	e.d.OnKeyData = func(ui.KeyData) bool { e.d.ScheduleFrame(); return true }
	e.h.OnFrame(0, time.Millisecond)
	e.h.OnKey(keyDown())
	e.h.OnFrame(0, 2*time.Millisecond)
	before := frameRequests(e)[0]
	e.h.OnPointer(pointerDown())
	if got := frameRequests(e)[0] - before; got != 1 {
		t.Errorf("frame requests for input after the frame = %d, want 1", got)
	}
	if !e.h.windows[0].wanted {
		t.Error("the window does not want the frame it asked for")
	}
}

// TestInputThatTheFlushQueuesGetsItsFrame checks that input queued while the frame
// flushes waits for the next frame, which is asked for.
func TestInputThatTheFlushQueuesGetsItsFrame(t *testing.T) {
	e := newWindowsEngine(t)
	e.size(0, 800, 600)
	e.renderAll(nil)
	pointers := 0
	e.d.OnPointerDataPacket = func(ui.PointerDataPacket) { pointers++ }
	e.d.OnKeyData = func(ui.KeyData) bool { e.h.OnPointer(pointerDown()); return true }
	e.h.OnFrame(0, time.Millisecond)
	e.h.OnKey(keyDown())
	before := frameRequests(e)[0]
	e.h.OnFrame(0, 2*time.Millisecond)
	if got := frameRequests(e)[0] - before; got != 1 {
		t.Errorf("frame requests during the frame = %d, want 1 for the queued input", got)
	}
	if pointers != 0 {
		t.Fatalf("the queued input ran in the frame that queued it: %d", pointers)
	}
	e.h.OnFrame(0, 3*time.Millisecond)
	if pointers != 1 {
		t.Errorf("the queued input ran %d times in the next frame, want 1", pointers)
	}
}
