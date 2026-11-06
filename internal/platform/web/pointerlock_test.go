package web

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestLockedMouse(t *testing.T) {
	moved := func(dx, dy float64) PointerEvent {
		e := mouseEv(noButtonChange, 0, 500, 500) // the browser's own place is not used
		e.MovementX, e.MovementY = dx, dy
		return e
	}
	tests := []struct {
		name string
		run  func(in *PointerInput) []ui.PointerData
		want []string
	}{
		{"the lock holds the pointer where it was last seen, with no delta", func(in *PointerInput) []ui.PointerData {
			in.Move(mouseEv(noButtonChange, 0, 30, 40), 2)
			return in.Lock(10, 10, 2)
		}, []string{"hover dev-1 id0 @60,80 d0,0 b0"}},
		{"a pointer not seen yet is held at the given place", func(in *PointerInput) []ui.PointerData {
			return in.Lock(10, 20, 1)
		}, []string{"add* dev-1 id0 @10,20 d0,0 b0", "hover dev-1 id0 @10,20 d0,0 b0"}},
		{"motion is the delta, in physical pixels, at the held place", func(in *PointerInput) []ui.PointerData {
			in.Move(mouseEv(noButtonChange, 0, 30, 40), 2)
			in.Lock(0, 0, 2)
			return in.Move(moved(3, -4), 2)
		}, []string{"hover dev-1 id0 @60,80 d6,-8 b0"}},
		{"coalesced events each carry their motion", func(in *PointerInput) []ui.PointerData {
			in.Move(mouseEv(noButtonChange, 0, 30, 40), 1)
			in.Lock(0, 0, 1)
			e := moved(3, 3)
			e.Coalesced = []PointerEvent{moved(1, 1), moved(2, 2)}
			return in.Move(e, 1)
		}, []string{"hover dev-1 id0 @30,40 d1,1 b0", "hover dev-1 id0 @30,40 d2,2 b0"}},
		{"an event with no motion makes nothing", func(in *PointerInput) []ui.PointerData {
			in.Move(mouseEv(noButtonChange, 0, 30, 40), 1)
			in.Lock(0, 0, 1)
			return in.Move(moved(0, 0), 1)
		}, nil},
		{"a button down is at the held place", func(in *PointerInput) []ui.PointerData {
			in.Move(mouseEv(noButtonChange, 0, 30, 40), 1)
			in.Lock(0, 0, 1)
			return in.Down(mouseEv(0, 1, 500, 500), 1)
		}, []string{"down dev-1 id1 @30,40 d0,0 b1"}},
		{"after the unlock the browser's place counts again", func(in *PointerInput) []ui.PointerData {
			in.Move(mouseEv(noButtonChange, 0, 30, 40), 1)
			in.Lock(0, 0, 1)
			in.Unlock()
			return in.Move(mouseEv(noButtonChange, 0, 70, 90), 1)
		}, []string{"hover dev-1 id0 @70,90 d40,50 b0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := NewPointerInput(0, OSLinux)
			var got []string
			for _, d := range tt.run(in) {
				got = append(got, fmtData(d, false))
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("data %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPointerLockState(t *testing.T) {
	type result struct {
		req          lockRequest
		ended, leave bool
		exit         bool
		pending      bool
	}
	tests := []struct {
		name string
		run  func(s *pointerLockState) result
		want result
	}{
		{"no API", func(s *pointerLockState) result { return result{req: s.request(false, true), pending: s.pending()} },
			result{req: lockUnsupported}},
		{"no user activation", func(s *pointerLockState) result { return result{req: s.request(true, false), pending: s.pending()} },
			result{req: lockNoGesture}},
		{"a request is pending until the browser answers", func(s *pointerLockState) result {
			return result{req: s.request(true, true), pending: s.pending()}
		}, result{req: lockAsk, pending: true}},
		{"the answer holds the lock", func(s *pointerLockState) result {
			s.request(true, true)
			ended, leave := s.changed(true)
			return result{ended: ended, leave: leave, pending: s.pending()}
		}, result{pending: true}},
		{"the user leaves the lock (Escape): the app hears of it", func(s *pointerLockState) result {
			s.request(true, true)
			s.changed(true)
			ended, leave := s.changed(false)
			return result{ended: ended, leave: leave, pending: s.pending()}
		}, result{ended: true}},
		{"the app unlocks: the browser is told and the app is not", func(s *pointerLockState) result {
			s.request(true, true)
			s.changed(true)
			exit := s.unlock()
			ended, leave := s.changed(false)
			return result{exit: exit, ended: ended, leave: leave, pending: s.pending()}
		}, result{exit: true}},
		{"a refused request ends the lock", func(s *pointerLockState) result {
			s.request(true, true)
			return result{ended: s.failed(), pending: s.pending()}
		}, result{ended: true}},
		{"an error with no request is nothing", func(s *pointerLockState) result { return result{ended: s.failed()} },
			result{}},
		{"unlock before the answer, then the browser grants it: leave", func(s *pointerLockState) result {
			s.request(true, true)
			exit := s.unlock()
			ended, leave := s.changed(true)
			return result{exit: exit, ended: ended, leave: leave, pending: s.pending()}
		}, result{leave: true}},
		{"unlock with no lock does nothing", func(s *pointerLockState) result { return result{exit: s.unlock()} }, result{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s pointerLockState
			if got := tt.run(&s); got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
