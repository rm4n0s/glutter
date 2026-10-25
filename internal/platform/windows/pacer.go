package windows

import "time"

// defaultRefreshRate is the rate of the pacer when the system says 0 or 1 (a monitor that does not
// report its refresh rate), in frames per second.
const defaultRefreshRate = 60

// intervalFor is the time between frames at a refresh rate; the rates 0 and 1 that Windows
// reports for "the hardware default" and anything not plausible are 60 Hz.
func intervalFor(rate float64) time.Duration {
	if rate <= 1 || rate > 1000 {
		rate = defaultRefreshRate
	}
	return time.Duration(float64(time.Second) / rate)
}

// pacerAction is what the vsync thread does next.
type pacerAction int

// The actions of the pacer.
const (
	// pacerIdle: no frame is wanted, so block until one is.
	pacerIdle pacerAction = iota
	// pacerFlush: call DwmFlush, which returns at the compositor's next vsync.
	pacerFlush
	// pacerSleep: wait on the high-resolution timer for the duration, and then tick.
	pacerSleep
)

// pacer is the state machine of the vsync thread (SPEC §25.3). It paces by DwmFlush until DwmFlush
// shows that it does not pace: it returns in less than a quarter of the frame interval twice in a
// row, or fails (Wine's returns at once). From then on it paces by the waitable timer at the
// monitor's refresh rate. The calls to Windows are the thread's; this holds the decisions, so a
// test runs them with numbers.
//
// Flutter: no counterpart (Flutter's embedder waits for the compositor with DwmFlush,
// shell/platform/windows/flutter_windows_view.cc, the DwmFlush after a present)
type pacer struct {
	interval time.Duration
	byTimer  bool
	quick    int // DwmFlush calls in a row that returned too soon
}

// newPacer makes a pacer for a monitor of the refresh rate.
func newPacer(refreshRate float64) *pacer {
	return &pacer{interval: intervalFor(refreshRate)}
}

// setRefreshRate follows a monitor whose rate changed.
func (p *pacer) setRefreshRate(rate float64) { p.interval = intervalFor(rate) }

// usesTimer says whether the pacer gave up DwmFlush.
func (p *pacer) usesTimer() bool { return p.byTimer }

// action says what to do next. wanted is whether a frame is wanted; sinceTick is the time since
// the last tick, which the timer waits out.
func (p *pacer) action(wanted bool, sinceTick time.Duration) (act pacerAction, wait time.Duration) {
	if !wanted {
		return pacerIdle, 0
	}
	if !p.byTimer {
		return pacerFlush, 0
	}
	return pacerSleep, max(p.interval-sinceTick, 0)
}

// flushed records the result of a DwmFlush: how long it took and whether it failed.
func (p *pacer) flushed(took time.Duration, failed bool) {
	if p.byTimer {
		return
	}
	if failed {
		p.byTimer = true
		return
	}
	if took < p.interval/4 {
		p.quick++
		if p.quick >= 2 {
			p.byTimer = true
		}
		return
	}
	p.quick = 0
}
