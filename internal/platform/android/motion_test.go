package android

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func TestDecodeMotion(t *testing.T) {
	// Two pointers, one historical sample.
	meta := []int64{2, 0, 0x1002, 3, 1000, 2, 1, 0}
	ids := []int32{7, 1, 9, 1}
	vals := make([]float32, motionValuesEach*2*2)
	for i := range vals {
		vals[i] = float32(i)
	}
	ev, generic, ok := decodeMotion(meta, ids, vals, []int64{990}, 0, 2)
	if !ok || generic {
		t.Fatalf("decodeMotion = %v, generic %v", ok, generic)
	}
	if ev.ActionMasked != 2 || ev.Source != 0x1002 || ev.ButtonState != 3 || ev.EventTime != 1000 || ev.PressureMax != 2 {
		t.Errorf("event fields: %+v", ev)
	}
	want := motionPointer{ID: 7, ToolType: 1, X: 0, Y: 1, Pressure: 2, Size: 3, ToolMajor: 4, ToolMinor: 5, Orientation: 6, Tilt: 7, Distance: 8, HScroll: 9, VScroll: 10}
	if !reflect.DeepEqual(ev.Pointers[0], want) {
		t.Errorf("pointer 0 = %+v, want %+v", ev.Pointers[0], want)
	}
	if ev.Pointers[1].ID != 9 || ev.Pointers[1].X != 11 {
		t.Errorf("pointer 1 = %+v", ev.Pointers[1])
	}
	if len(ev.History) != 1 || ev.History[0].Time != 990 || ev.History[0].Pointers[0].X != 22 || ev.History[0].Pointers[1].ID != 9 {
		t.Errorf("history = %+v", ev.History)
	}
	if _, g, ok := decodeMotion([]int64{8, 0, 0x2002, 0, 5, 1, 0, 1}, []int32{0, 3}, make([]float32, motionValuesEach), nil, 0, 1); !ok || !g {
		t.Errorf("a generic event: ok %v generic %v", ok, g)
	}
}

func TestDecodeMotionRejectsArraysThatDoNotFit(t *testing.T) {
	good := []int64{2, 0, 0, 0, 0, 1, 0, 0}
	tests := []struct {
		name  string
		meta  []int64
		ids   []int32
		vals  []float32
		times []int64
	}{
		{"short meta", good[:7], []int32{0, 1}, make([]float32, 11), nil},
		{"ids too short", good, []int32{0}, make([]float32, 11), nil},
		{"values too few", good, []int32{0, 1}, make([]float32, 10), nil},
		{"history without times", []int64{2, 0, 0, 0, 0, 1, 1, 0}, []int32{0, 1}, make([]float32, 22), nil},
		{"negative count", []int64{2, 0, 0, 0, 0, -1, 0, 0}, nil, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, ok := decodeMotion(tt.meta, tt.ids, tt.vals, tt.times, 0, 1); ok {
				t.Error("an inconsistent event was accepted")
			}
		})
	}
}

// A touch goes through the processor and the converter (the add before the down, the remove after
// the up) to the handler, in one batch for each event; a generic event that is not a pointer
// movement is not handled.
func TestOnMotionDeliversPointerData(t *testing.T) {
	h := newFakeHost()
	p := newPlatform(newBridge(), h, nil)
	rec := &recorder{}
	p.handler = rec
	win, err := p.NewWindow(platform.WindowConfig{})
	if err != nil || win == nil {
		t.Fatal(err)
	}
	down := motionEvent{ActionMasked: 0, Pointers: []motionPointer{{ID: 0, ToolType: 1, X: 10, Y: 20, Pressure: 1}}, PressureMax: 1, EventTime: 5}
	p.onMotion(down, false, 48, 48)
	up := down
	up.ActionMasked, up.EventTime = 1, 25
	p.onMotion(up, false, 48, 48)
	p.onMotion(motionEvent{ActionMasked: 5, Pointers: down.Pointers, Source: 0x2}, true, 48, 48) // not a hover or a scroll
	got := rec.pointerChanges()
	want := [][]string{{"PointerChangeAdd", "PointerChangeDown"}, {"PointerChangeUp", "PointerChangeRemove"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pointer batches %v, want %v", got, want)
	}
}

// syncRecorder is a handler that answers key data at once, as the engine's does.
type syncRecorder struct {
	recorder
	answer bool
	keys   []ui.KeyData
}

func (s *syncRecorder) OnKeySync(d ui.KeyData) bool {
	s.keys = append(s.keys, d)
	return s.answer
}
func (s *syncRecorder) OnPointerSync([]ui.PointerData)    {}
func (s *syncRecorder) OnTextInputSync(ui.TextInputEvent) {}

// The framework's answer to a key goes back to the Java side with the id that came with the event:
// a key that it handled is taken, one that it did not is dispatched again (SPEC §24.5).
func TestOnKeyAnswersTheJavaSide(t *testing.T) {
	k := keyEvent{Action: 0, KeyCode: 39, ScanCode: 37, EventTime: 10} // KEYCODE_K down
	tests := []struct {
		name    string
		handler func() platform.Handler
		want    string
	}{
		{"handled", func() platform.Handler { return &syncRecorder{answer: true} }, "7:true"},
		{"not handled", func() platform.Handler { return &syncRecorder{answer: false} }, "7:false"},
		{"no handler yet", func() platform.Handler { return nil }, "7:false"},
		{"a handler that cannot answer at once takes the key", func() platform.Handler { return &recorder{} }, "7:true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFakeHost()
			p := newPlatform(newBridge(), h, nil)
			p.handler = tt.handler()
			p.onKey(7, k)
			h.mu.Lock()
			defer h.mu.Unlock()
			if len(h.keyAnswers) != 1 || h.keyAnswers[0] != tt.want {
				t.Errorf("answers %v, want [%s]", h.keyAnswers, tt.want)
			}
		})
	}
	// The key data that the framework got is Flutter's: KeyK, the character k.
	h := newFakeHost()
	p := newPlatform(newBridge(), h, nil)
	s := &syncRecorder{answer: true}
	p.handler = s
	k.UnicodeChar = 'k'
	p.onKey(1, k)
	if len(s.keys) == 0 || s.keys[len(s.keys)-1].Physical != 0x7000e || s.keys[len(s.keys)-1].Logical != 0x6b {
		t.Errorf("key data %+v, want KeyK down", s.keys)
	}
}

// The native entry points queue their events for the UI thread, in order, and the platform runs
// them there.
func TestBridgeQueuesMotionAndKeyEvents(t *testing.T) {
	h := newFakeHost()
	b := newBridge()
	p := newPlatform(b, h, nil)
	s := &syncRecorder{answer: true}
	p.handler = s
	if _, err := p.NewWindow(platform.WindowConfig{}); err != nil {
		t.Fatal(err)
	}
	b.motion(motionEvent{ActionMasked: 0, Pointers: []motionPointer{{ToolType: 1, X: 1, Y: 2}}, PressureMax: 1}, false, 48, 48)
	b.key(3, keyEvent{Action: 0, KeyCode: 39, ScanCode: 37})
	for i := 0; i < 2; i++ {
		fn, ok := b.pop()
		if !ok {
			t.Fatalf("event %d was not queued", i)
		}
		fn(p)
	}
	if _, ok := b.pop(); ok {
		t.Error("more events than were sent")
	}
	if got := s.pointerChanges(); len(got) != 1 || got[0][1] != "PointerChangeDown" {
		t.Errorf("pointer batches %v", got)
	}
	if len(s.keys) == 0 || h.keyAnswers[0] != "3:true" {
		t.Errorf("keys %v, answers %v", s.keys, h.keyAnswers)
	}
}
