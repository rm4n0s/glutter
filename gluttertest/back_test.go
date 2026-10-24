package gluttertest

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// backObserver records what the binding gives to an observer of the back gesture.
type backObserver struct {
	widgets.WidgetsBindingObserverBase
	take bool
	log  []string
}

func (o *backObserver) DidPopRoute() bool {
	o.log = append(o.log, "pop")
	return true
}

func (o *backObserver) HandleStartBackGesture(e services.PredictiveBackEvent) bool {
	o.log = append(o.log, "start "+e.String())
	return o.take
}

func (o *backObserver) HandleUpdateBackGestureProgress(e services.PredictiveBackEvent) {
	o.log = append(o.log, "update "+e.String())
}

func (o *backObserver) HandleCommitBackGesture() { o.log = append(o.log, "commit") }

func (o *backObserver) HandleCancelBackGesture() { o.log = append(o.log, "cancel") }

func TestBackGestureDriverReachesTheObservers(t *testing.T) {
	left := func(progress, x, y float64) string {
		return services.NewPredictiveBackEvent(foundation.Some(ui.Offset{Dx: x, Dy: y}), progress, services.SwipeEdgeLeft).String()
	}
	tests := []struct {
		name     string
		take     bool
		play     func(ts *Tester) (took bool)
		wantTook bool
		wantLog  []string
	}{
		{"a gesture that is taken goes on to the commit", true, func(ts *Tester) bool {
			took := ts.StartBackGesture(0, BackGestureOpts{Touch: foundation.Some(ui.Offset{Dx: 5, Dy: 300})})
			ts.UpdateBackGestureProgress(0.35, BackGestureOpts{Touch: foundation.Some(ui.Offset{Dx: 100, Dy: 300})})
			ts.CommitBackGesture()
			return took
		}, true, []string{"start " + left(0, 5, 300), "update " + left(0.35, 100, 300), "commit"}},
		{"a gesture that is canceled", true, func(ts *Tester) bool {
			took := ts.StartBackGesture(0.1, BackGestureOpts{Touch: foundation.Some(ui.Offset{Dx: 5, Dy: 300})})
			ts.CancelBackGesture()
			return took
		}, true, []string{"start " + left(0.1, 5, 300), "cancel"}},
		{"a gesture that no observer takes", false, func(ts *Tester) bool {
			return ts.StartBackGesture(0, BackGestureOpts{Touch: foundation.Some(ui.Offset{Dx: 5, Dy: 300})})
		}, false, []string{"start " + left(0, 5, 300)}},
		{"the right edge", true, func(ts *Tester) bool {
			return ts.StartBackGesture(0.2, BackGestureOpts{Touch: foundation.Some(ui.Offset{Dx: 790, Dy: 10}), SwipeEdge: ui.BackSwipeEdgeRight})
		}, true, []string{"start " + services.NewPredictiveBackEvent(foundation.Some(ui.Offset{Dx: 790, Dy: 10}), 0.2, services.SwipeEdgeRight).String()}},
		{"no touch offset is a button event", true, func(ts *Tester) bool {
			return ts.StartBackGesture(0)
		}, true, []string{"start " + services.NewPredictiveBackEvent(foundation.None[ui.Offset](), 0, services.SwipeEdgeLeft).String()}},
		{"a pop without a gesture", true, func(ts *Tester) bool {
			ts.PopRoute()
			return false
		}, false, []string{"pop"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := New(t)
			ts.PumpWidget(newRow())
			o := &backObserver{take: tt.take}
			ts.Binding.AddObserver(o)
			t.Cleanup(func() { ts.Binding.RemoveObserver(o) })
			if took := tt.play(ts); took != tt.wantTook {
				t.Errorf("took = %v, want %v", took, tt.wantTook)
			}
			if !reflect.DeepEqual(o.log, tt.wantLog) {
				t.Errorf("the observer saw\n%s\nwant\n%s", fmt.Sprint(o.log), fmt.Sprint(tt.wantLog))
			}
		})
	}
}
