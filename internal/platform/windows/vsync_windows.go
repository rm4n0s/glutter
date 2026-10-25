//go:build windows

package windows

import (
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// vsyncThread is the pacer of the frames (SPEC §25.3): a goroutine locked to an OS thread that
// waits for the compositor's next vsync with DwmFlush, or, when DwmFlush does not pace (Wine's
// returns at once), for the high-resolution waitable timer at the monitor's refresh rate. It works
// only while a frame is wanted: otherwise it waits on a channel, so an idle app arms nothing. It
// touches no window, timer or posted function; after each wait it posts one message to the
// service window.
//
// Flutter: no counterpart (Flutter's embedder calls DwmFlush after a present,
// shell/platform/windows/flutter_windows_view.cc)
type vsyncThread struct {
	log  *slog.Logger
	post func()

	wanted atomic.Bool
	wake   chan struct{}
	quit   chan struct{}
	done   chan struct{}

	mu sync.Mutex
	pc *pacer
}

func newVsyncThread(log *slog.Logger, post func()) *vsyncThread {
	v := &vsyncThread{
		log: log, post: post, wake: make(chan struct{}, 1), quit: make(chan struct{}), done: make(chan struct{}),
		pc: newPacer(defaultRefreshRate),
	}
	go v.run()
	return v
}

// setRefreshRate follows the primary monitor's rate.
func (v *vsyncThread) setRefreshRate(rate float64) {
	v.mu.Lock()
	v.pc.setRefreshRate(rate)
	v.mu.Unlock()
}

// request asks for one tick. It is safe from any goroutine.
func (v *vsyncThread) request() {
	v.wanted.Store(true)
	select {
	case v.wake <- struct{}{}:
	default: // the thread is awake already
	}
}

// stop ends the thread and waits for it.
func (v *vsyncThread) stop() {
	close(v.quit)
	<-v.done
}

func (v *vsyncThread) run() {
	defer close(v.done)
	runtime.LockOSThread()
	var timer *win32.WaitableTimer
	defer func() {
		if timer != nil {
			_ = timer.Close() // the thread ends; the handle goes with the process otherwise
		}
	}()
	last := time.Now()
	for {
		select {
		case <-v.wake:
		case <-v.quit:
			return
		}
		for v.wanted.Load() {
			v.mu.Lock()
			act, wait := v.pc.action(true, time.Since(last))
			v.mu.Unlock()
			switch act {
			case pacerFlush:
				t0 := time.Now()
				err := win32.DwmFlush()
				v.mu.Lock()
				v.pc.flushed(time.Since(t0), err != nil)
				v.mu.Unlock()
			case pacerSleep:
				if timer == nil {
					var err error
					if timer, err = win32.NewWaitableTimer(); err != nil {
						v.log.Warn("the frame timer could not be made; frames are paced by sleeping", "err", err)
						timer = nil
					}
				}
				if timer != nil {
					if err := timer.WaitFor(wait); err != nil {
						v.log.Warn("the frame timer failed", "err", err)
						time.Sleep(wait)
					}
				} else {
					time.Sleep(wait)
				}
			}
			last = time.Now()
			v.wanted.Store(false)
			v.post()
		}
	}
}
