package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func at(h, m, s int) time.Time { return time.Date(2026, 10, 19, h, m, s, 0, time.UTC) }

// TestClockGolden draws the clock at a fixed time: the clock function is given
// to the widget's constructor.
func TestClockGolden(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 420, Height: 480}})
	ts.PumpWidget(newClockApp(func() time.Time { return at(10, 8, 35) }))
	ts.MatchesGolden("clock")
}

func TestClockShouldRepaint(t *testing.T) {
	tests := []struct {
		name     string
		old, new rendering.CustomPainter
		want     bool
	}{
		{"same second", newClockPainter(at(1, 2, 3), nil), newClockPainter(at(1, 2, 3), nil), false},
		{"another second", newClockPainter(at(1, 2, 3), nil), newClockPainter(at(1, 2, 4), nil), true},
		{"another minute, same second", newClockPainter(at(1, 2, 3), nil), newClockPainter(at(1, 3, 3), nil), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.new.ShouldRepaint(tt.old); got != tt.want {
				t.Errorf("ShouldRepaint = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestClockRepaintsOnlyWhenTheSecondChanges updates the state the way the
// goroutine does and counts the paints.
func TestClockRepaintsOnlyWhenTheSecondChanges(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 420, Height: 480}})
	now := at(10, 0, 0)
	paints := 0
	app := newClockApp(func() time.Time { return now })
	app.onPaint = func() { paints++ }
	ts.PumpWidget(app)
	if paints != 1 {
		t.Fatalf("paints = %d after the first frame, want 1", paints)
	}
	state := gluttertest.StateOf[*clockState](ts, gluttertest.ByType[*clockApp]())
	update := func(tm time.Time) {
		state.SetState(func() { state.time = tm })
		ts.Pump()
	}
	update(at(10, 0, 0)) // rebuilt, but the same second
	if paints != 1 {
		t.Errorf("paints = %d after a rebuild in the same second, want 1", paints)
	}
	update(at(10, 0, 1))
	if paints != 2 {
		t.Errorf("paints = %d after a new second, want 2", paints)
	}
	if ts.Count(gluttertest.ByText("10:00:01")) != 1 {
		t.Error("the caption does not show the new time")
	}
}

// TestClockTicks runs the real goroutine for a short while: the state keeps
// asking for the time once a second.
func TestClockTicks(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 420, Height: 480}})
	calls := make(chan struct{}, 8)
	ts.PumpWidget(newClockApp(func() time.Time {
		select {
		case calls <- struct{}{}:
		default:
		}
		return time.Now()
	}))
	<-calls // the first call, in InitState
	select {
	case <-calls:
	case <-time.After(3 * time.Second):
		t.Fatal("the clock goroutine did not run")
	}
	ts.PumpWidget(widgets.NewSizedBox(nil)) // dispose the state: the goroutine stops
}
