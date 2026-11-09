package macos

import (
	"sync"
	"testing"
)

func TestFrameRequestsMergeAndWaitForADrawableWindow(t *testing.T) {
	tests := []struct {
		name    string
		steps   func(f *frameRequests) []bool
		want    []bool
		wanting bool
	}{
		{"requests while wanted are one frame", func(f *frameRequests) []bool {
			f.request()
			f.request()
			f.request()
			return []bool{f.take(true), f.take(true)}
		}, []bool{true, false}, false},
		{"a request during the frame asks for the next", func(f *frameRequests) []bool {
			f.request()
			first := f.take(true)
			f.request() // made while the frame is drawn
			return []bool{first, f.take(true), f.take(true)}
		}, []bool{true, true, false}, false},
		{"a minimized window keeps its request", func(f *frameRequests) []bool {
			f.request()
			return []bool{f.take(false), f.take(false), f.take(true)}
		}, []bool{false, false, true}, false},
		{"clear drops the request", func(f *frameRequests) []bool {
			f.request()
			f.clear()
			return []bool{f.take(true)}
		}, []bool{false}, false},
		{"no request, no frame", func(f *frameRequests) []bool { return []bool{f.take(true)} }, []bool{false}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f frameRequests
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
	var p linkPacer
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
		start, stop := p.update(s.wanted)
		if start != s.start || stop != s.stop {
			t.Errorf("step %d (wanted %v) = start %v stop %v, want %v %v", i, s.wanted, start, stop, s.start, s.stop)
		}
	}
}

func TestTickGateDropsTheTicksOfASlowLoop(t *testing.T) {
	var g tickGate
	if !g.offer() {
		t.Fatal("the first tick is refused")
	}
	for i := 0; i < 5; i++ {
		if g.offer() {
			t.Fatal("a tick passed while the last one waits")
		}
	}
	g.done()
	if !g.offer() {
		t.Error("the tick after done is refused")
	}
}

func TestTickGateFromManyThreadsPostsOnce(t *testing.T) {
	var g tickGate
	var wg sync.WaitGroup
	var mu sync.Mutex
	passed := 0
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if g.offer() {
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
