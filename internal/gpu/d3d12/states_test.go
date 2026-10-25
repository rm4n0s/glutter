package d3d12

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
)

// tex stands for a texture in the tests of the tracker.
type tex struct{ name string }

func tr(res *tex, level int, before, after resState) transition[*tex] {
	return transition[*tex]{Res: res, Level: level, Before: before, After: after}
}

func mustNeed(t *testing.T, s *stateTracker[*tex], res *tex, level int, want resState) {
	t.Helper()
	if err := s.need(res, level, want); err != nil {
		t.Fatalf("need(%s, %d, %v): %v", res.name, level, want, err)
	}
}

// A texture that is drawn into, then sampled, then copied: each operation gets the barriers it needs,
// and the ones of one operation are one batch (SPEC §9.8 "States").
func TestTrackerDrawnSampledCopied(t *testing.T) {
	frame, mask, dst := &tex{"frame"}, &tex{"mask"}, &tex{"dst"}
	s := newStateTracker[*tex]()
	for _, r := range []*tex{frame, mask, dst} {
		s.add(r, 1)
	}

	// The pass draws into the frame and samples the mask: two transitions, one batch.
	s.begin()
	mustNeed(t, s, frame, 0, stateRenderTarget)
	mustNeed(t, s, mask, 0, statePixelShaderResource)
	s.end()
	want := []transition[*tex]{tr(frame, 0, stateCommon, stateRenderTarget), tr(mask, 0, stateCommon, statePixelShaderResource)}
	if got := s.flush(); !reflect.DeepEqual(got, want) {
		t.Errorf("first batch = %v, want %v", got, want)
	}
	if got := s.flush(); len(got) != 0 {
		t.Errorf("a second flush gave %v", got)
	}

	// The next pass draws into the frame again and samples the mask again: nothing changed.
	s.begin()
	mustNeed(t, s, frame, 0, stateRenderTarget)
	mustNeed(t, s, mask, 0, statePixelShaderResource)
	s.end()
	if got := s.flush(); len(got) != 0 {
		t.Errorf("a pass in the same states needed %v", got)
	}

	// A pass that samples the frame: it leaves RENDER_TARGET.
	s.begin()
	mustNeed(t, s, frame, 0, statePixelShaderResource)
	mustNeed(t, s, dst, 0, stateRenderTarget)
	s.end()
	want = []transition[*tex]{tr(frame, 0, stateRenderTarget, statePixelShaderResource), tr(dst, 0, stateCommon, stateRenderTarget)}
	if got := s.flush(); !reflect.DeepEqual(got, want) {
		t.Errorf("second batch = %v, want %v", got, want)
	}

	// The copy reads the frame and writes the destination.
	s.begin()
	mustNeed(t, s, frame, 0, stateCopySource)
	mustNeed(t, s, dst, 0, stateCopyDest)
	s.end()
	want = []transition[*tex]{tr(frame, 0, statePixelShaderResource, stateCopySource), tr(dst, 0, stateRenderTarget, stateCopyDest)}
	if got := s.flush(); !reflect.DeepEqual(got, want) {
		t.Errorf("third batch = %v, want %v", got, want)
	}
	if st, ok := s.state(frame, 0); !ok || st != stateCopySource {
		t.Errorf("state of the frame = %v, %v", st, ok)
	}
}

func TestTrackerLevelsAreTracked(t *testing.T) {
	m := &tex{"mips"}
	s := newStateTracker[*tex]()
	s.add(m, 3)

	// Level 1 is drawn into while level 0 is sampled: the downsampling pass.
	s.begin()
	mustNeed(t, s, m, 0, statePixelShaderResource)
	mustNeed(t, s, m, 1, stateRenderTarget)
	s.end()
	want := []transition[*tex]{tr(m, 0, stateCommon, statePixelShaderResource), tr(m, 1, stateCommon, stateRenderTarget)}
	if got := s.flush(); !reflect.DeepEqual(got, want) {
		t.Errorf("batch = %v, want %v", got, want)
	}
	for level, want := range []resState{statePixelShaderResource, stateRenderTarget, stateCommon} {
		if st, _ := s.state(m, level); st != want {
			t.Errorf("level %d is %v, want %v", level, st, want)
		}
	}

	// Sampling every level: the ones already in the state need nothing, the others change.
	s.begin()
	mustNeed(t, s, m, allLevels, statePixelShaderResource)
	s.end()
	want = []transition[*tex]{tr(m, 1, stateRenderTarget, statePixelShaderResource), tr(m, 2, stateCommon, statePixelShaderResource)}
	if got := s.flush(); !reflect.DeepEqual(got, want) {
		t.Errorf("batch for all levels = %v, want %v", got, want)
	}
}

func TestTrackerConflicts(t *testing.T) {
	tests := []struct {
		name  string
		first int // level, or allLevels
		a     resState
		then  int
		b     resState
	}{
		{"drawn into and sampled in one pass", 0, stateRenderTarget, 0, statePixelShaderResource},
		{"sampled and then drawn into", 0, statePixelShaderResource, 0, stateRenderTarget},
		{"copied onto itself", 0, stateCopySource, 0, stateCopyDest},
		{"every level sampled and one drawn into", allLevels, statePixelShaderResource, 1, stateRenderTarget},
		{"one level drawn into and every level sampled", 1, stateRenderTarget, allLevels, statePixelShaderResource},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &tex{"mips"}
			s := newStateTracker[*tex]()
			s.add(m, 2)
			s.begin()
			mustNeed(t, s, m, tt.first, tt.a)
			before := len(s.batch)
			err := s.need(m, tt.then, tt.b)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tagD3D12StateConflict || !appErr.HasRoute("stateTracker[...].need.D3D12StateConflict") {
				t.Errorf("got %s via %s", appErr.Tag, appErr.Route())
			}
			if appErr.Metadata["state"] == nil || appErr.Metadata["wanted"] == nil {
				t.Errorf("metadata = %v", appErr.Metadata)
			}
			if len(s.batch) != before {
				t.Errorf("the conflict left %d transitions in the batch, it had %d", len(s.batch), before)
			}
		})
	}
}

// Outside an operation nothing is locked, and a new operation forgets the locks of the last one.
func TestTrackerLocksEndWithTheOperation(t *testing.T) {
	m := &tex{"t"}
	s := newStateTracker[*tex]()
	s.add(m, 1)
	s.begin()
	mustNeed(t, s, m, 0, stateRenderTarget)
	s.end()
	s.begin()
	mustNeed(t, s, m, 0, statePixelShaderResource)
	s.end()
	mustNeed(t, s, m, 0, stateCopySource) // no operation open: allowed
	if got := len(s.flush()); got != 3 {
		t.Errorf("%d transitions, want 3", got)
	}
}

func TestTrackerErrors(t *testing.T) {
	m, other := &tex{"m"}, &tex{"other"}
	s := newStateTracker[*tex]()
	s.add(m, 2)
	tests := []struct {
		name  string
		res   *tex
		level int
		tag   string
	}{
		{"a resource that is not tracked", other, 0, tagD3D12ForeignResource},
		{"a level that does not exist", m, 2, tagD3D12LevelInvalid},
		{"a negative level", m, -2, tagD3D12LevelInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.need(tt.res, tt.level, stateCopySource)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute("stateTracker[...].need."+tt.tag) {
				t.Errorf("got %s via %s, want %s", appErr.Tag, appErr.Route(), tt.tag)
			}
		})
	}
	s.forget(m)
	if _, ok := s.state(m, 0); ok {
		t.Error("a forgotten resource is still tracked")
	}
}
