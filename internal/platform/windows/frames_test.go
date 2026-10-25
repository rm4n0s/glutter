package windows

import "testing"

func TestFrameRequestsMerge(t *testing.T) {
	tests := []struct {
		name  string
		steps []string // "request", "take", "take-blocked" (cannot draw)
		want  []bool   // the result of each take
	}{
		{"a frame asked for is drawn once", []string{"request", "take", "take"}, []bool{true, false}},
		{"requests merge into one frame", []string{"request", "request", "request", "take", "take"}, []bool{true, false}},
		{"a request made during a frame asks for the next", []string{"request", "take", "request", "take"}, []bool{true, true}},
		{"no frame asked for", []string{"take"}, []bool{false}},
		{"a window that cannot draw keeps the request", []string{"request", "take-blocked", "take-blocked", "take"}, []bool{false, false, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f frameRequests
			var got []bool
			for _, s := range tt.steps {
				switch s {
				case "request":
					f.request()
				case "take":
					got = append(got, f.take(true))
				case "take-blocked":
					got = append(got, f.take(false))
				}
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("take %d = %v, want %v (all: %v)", i, got[i], tt.want[i], got)
				}
			}
		})
	}
}

func TestFrameRequestsClear(t *testing.T) {
	var f frameRequests
	f.request()
	f.clear() // a resize drew the frame
	if f.take(true) {
		t.Error("a frame drawn by a resize is drawn again")
	}
}

func TestVsyncGate(t *testing.T) {
	var g vsyncGate
	if g.shouldRequest(false) {
		t.Error("asked for a tick while no window wants a frame: an idle app must arm nothing")
	}
	if !g.shouldRequest(true) {
		t.Error("did not ask for a tick although a frame is wanted")
	}
	if g.shouldRequest(true) {
		t.Error("asked for a second tick while one is on its way")
	}
	g.delivered()
	if !g.shouldRequest(true) {
		t.Error("did not ask again after the tick came")
	}
}
