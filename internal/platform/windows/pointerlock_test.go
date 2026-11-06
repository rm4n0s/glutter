package windows

import "testing"

func TestLockedPointer(t *testing.T) {
	tests := []struct {
		name string
		run  func(tr *pointerTracker)
		want []ev
	}{
		{"motion is the delta of a hover at the held place", func(tr *pointerTracker) {
			tr.move(0, mouseAt(10, 20, 0))
			l := newLockTranslator(tr, 0, 10, 20)
			tr.take()
			l.motion(0, 3, -4)
			l.motion(0, 1, 1)
		}, []ev{
			{change: cHover, kind: kMouse, x: 10, y: 20, dx: 3, dy: -4},
			{change: cHover, kind: kMouse, x: 10, y: 20, dx: 1, dy: 1},
		}},
		{"a lock begun at another place moves the pointer there first", func(tr *pointerTracker) {
			tr.move(0, mouseAt(10, 20, 0))
			tr.take()
			l := newLockTranslator(tr, 0, 50, 60)
			l.motion(0, 2, 2)
		}, []ev{
			{change: cHover, kind: kMouse, x: 50, y: 60},
			{change: cHover, kind: kMouse, x: 50, y: 60, dx: 2, dy: 2},
		}},
		{"a held button makes the motion a move", func(tr *pointerTracker) {
			tr.move(0, mouseAt(10, 20, 0))
			tr.down(0, mouseAt(10, 20, buttonPrimary))
			l := newLockTranslator(tr, 0, 10, 20)
			tr.take()
			l.motion(0, -5, 7)
		}, []ev{
			{change: cMove, kind: kMouse, x: 10, y: 20, dx: -5, dy: 7, buttons: buttonPrimary},
		}},
		{"no motion and absolute reports make nothing", func(tr *pointerTracker) {
			tr.move(0, mouseAt(10, 20, 0))
			l := newLockTranslator(tr, 0, 10, 20)
			tr.take()
			l.motion(0, 0, 0)
			l.motion(mouseMoveAbsolute, 100, 100)
		}, nil},
		{"the first report of an unknown pointer adds it", func(tr *pointerTracker) {
			l := newLockTranslator(tr, 0, 8, 9)
			tr.take()
			tr.states = map[pointerKey]*pointerState{}
			l.motion(0, 1, 2)
		}, []ev{
			{change: cAdd, kind: kMouse, x: 8, y: 9},
			{change: cHover, kind: kMouse, x: 8, y: 9, dx: 1, dy: 2},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newPointerTracker(tickingClock())
			tt.run(tr)
			got := brief(tr.take())
			if len(got) != len(tt.want) {
				t.Fatalf("got %d events %+v, want %d %+v", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("event %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLockPoint(t *testing.T) {
	tests := []struct {
		name         string
		cx, cy, w, h int32
		over         bool
		x, y         int32
	}{
		{"the cursor over the client area", 30, 40, 200, 100, true, 30, 40},
		{"the cursor elsewhere", 30, 40, 200, 100, false, 100, 50},
		{"the cursor on the edge outside", 200, 40, 200, 100, true, 100, 50},
		{"a negative place", -1, 5, 200, 100, true, 100, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := lockPoint(tt.cx, tt.cy, tt.w, tt.h, tt.over)
			if x != tt.x || y != tt.y {
				t.Errorf("lockPoint = %d,%d, want %d,%d", x, y, tt.x, tt.y)
			}
		})
	}
}
