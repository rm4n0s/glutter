package loop

import (
	"sync"
	"testing"
)

func TestFrameRequestsMergeAndWaitForADrawableWindow(t *testing.T) {
	tests := []struct {
		name    string
		steps   func(f *FrameRequests) []bool
		want    []bool
		wanting bool
	}{
		{"requests while wanted are one frame", func(f *FrameRequests) []bool {
			f.Request()
			f.Request()
			f.Request()
			return []bool{f.Take(true), f.Take(true)}
		}, []bool{true, false}, false},
		{"a request during the frame asks for the next", func(f *FrameRequests) []bool {
			f.Request()
			first := f.Take(true)
			f.Request() // made while the frame is drawn
			return []bool{first, f.Take(true), f.Take(true)}
		}, []bool{true, true, false}, false},
		{"a minimized window keeps its request", func(f *FrameRequests) []bool {
			f.Request()
			return []bool{f.Take(false), f.Take(false), f.Take(true)}
		}, []bool{false, false, true}, false},
		{"clear drops the request", func(f *FrameRequests) []bool {
			f.Request()
			f.Clear()
			return []bool{f.Take(true)}
		}, []bool{false}, false},
		{"no request, no frame", func(f *FrameRequests) []bool { return []bool{f.Take(true)} }, []bool{false}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f FrameRequests
			got := tt.steps(&f)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("step %d = %v, want %v (all: %v)", i, got[i], tt.want[i], got)
				}
			}
		})
	}
}

func TestLinkPacerRunsTheLinkOnlyWhileAFrameIsWanted(t *testing.T) {
	var p LinkPacer
	type step struct {
		wanted      bool
		start, stop bool
	}
	steps := []step{
		{false, false, false}, // idle: nothing runs
		{true, true, false},   // a frame is wanted: start
		{true, false, false},  // still wanted: no second start
		{false, false, true},  // the frame was drawn: stop
		{false, false, false}, // idle again
		{true, true, false},   // wanted again
	}
	for i, s := range steps {
		start, stop := p.Update(s.wanted)
		if start != s.start || stop != s.stop {
			t.Errorf("step %d (wanted %v) = start %v stop %v, want %v %v", i, s.wanted, start, stop, s.start, s.stop)
		}
		if p.Running() != s.wanted {
			t.Errorf("step %d: Running() = %v after wanted = %v", i, p.Running(), s.wanted)
		}
	}
}

func TestTickGateDropsTheTicksOfASlowLoop(t *testing.T) {
	var g TickGate
	if !g.Offer() {
		t.Fatal("the first tick is refused")
	}
	for i := 0; i < 5; i++ {
		if g.Offer() {
			t.Fatal("a tick passed while the last one waits")
		}
	}
	g.Done()
	if !g.Offer() {
		t.Error("the tick after done is refused")
	}
}

func TestTickGateFromManyThreadsPostsOnce(t *testing.T) {
	var g TickGate
	var wg sync.WaitGroup
	var mu sync.Mutex
	passed := 0
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if g.Offer() {
				mu.Lock()
				passed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if passed != 1 {
		t.Errorf("%d ticks passed, want 1", passed)
	}
}
