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

func TestLinkWatchGivesWayToTheFallbackWhileThePreferredLinkIsSilent(t *testing.T) {
	type step struct {
		name string
		do   func(w *LinkWatch) bool
		// want is what the call says; fallback and watching are the state after it.
		want, fallback, watching bool
	}
	start := func(w *LinkWatch) bool { return w.Start() }
	stop := func(w *LinkWatch) bool { w.Stop(); return false }
	tick := func(w *LinkWatch) bool { return w.Tick() }
	check := func(w *LinkWatch) bool { return w.Check() }
	tests := []struct {
		name  string
		steps []step
	}{
		{"a link that ticks keeps the pacing", []step{
			{"start", start, false, false, true},
			{"tick", tick, false, false, true},
			{"an interval with a tick", check, false, false, true},
			{"tick", tick, false, false, true},
			{"another one", check, false, false, true},
			{"stop", stop, false, false, false},
			{"a check after the stop says nothing", check, false, false, false},
		}},
		{"a silent link gives way after one interval", []step{
			{"start", start, false, false, true},
			{"an interval with no tick", check, true, true, false},
			{"the fallback paces: no more checks start it", check, false, true, false},
			{"stop", stop, false, true, false},
			{"the next start goes to the fallback at once", start, true, true, false},
			{"stop", stop, false, true, false},
		}},
		{"a tick of the preferred link takes the pacing back", []step{
			{"start", start, false, false, true},
			{"silent", check, true, true, false},
			{"the screen woke: a tick", tick, true, false, true},
			{"it keeps ticking", tick, false, false, true},
			{"an interval with ticks", check, false, false, true},
			{"stop", stop, false, false, false},
			{"the next start is the preferred link alone", start, false, false, true},
		}},
		{"a link that falls silent later gives way then", []step{
			{"start", start, false, false, true},
			{"tick", tick, false, false, true},
			{"an interval with a tick", check, false, false, true},
			{"an interval with none: the screen went to sleep", check, true, true, false},
		}},
		{"a tick that comes after the stop takes the pacing back without stopping anything", []step{
			{"start", start, false, false, true},
			{"silent", check, true, true, false},
			{"stop", stop, false, true, false},
			{"a late tick", tick, false, false, false},
			{"start", start, false, false, true},
		}},
		{"the ticks of one start are not counted for the next", []step{
			{"start", start, false, false, true},
			{"tick", tick, false, false, true},
			{"stop", stop, false, false, false},
			{"start", start, false, false, true},
			{"an interval with no tick since the start", check, true, true, false},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w LinkWatch
			for _, s := range tt.steps {
				if got := s.do(&w); got != s.want {
					t.Fatalf("%s: said %v, want %v", s.name, got, s.want)
				}
				if w.Fallback() != s.fallback || w.Watching() != s.watching {
					t.Fatalf("%s: fallback %v watching %v, want %v %v", s.name, w.Fallback(), w.Watching(), s.fallback, s.watching)
				}
			}
		})
	}
}
