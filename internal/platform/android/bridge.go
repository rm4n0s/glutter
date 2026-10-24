package android

import (
	"log/slog"
	"sync"
	"time"
)

// destroyWait is how long surfaceDestroyed waits for the UI thread to let go of the surface
// (SPEC §24.3 "The one wait").
const destroyWait = 2 * time.Second

// bridge is where the Java side's calls meet the UI thread: the main thread of Android calls the
// native methods, which put a function in the queue; the UI thread runs the functions between
// frames, in order (SPEC §24.3 "Events"). It also holds the state of the surface, because the
// events that come before the platform exists, and the ones that come while it does not run, must
// not be lost. There is one per process, because JNI has one set of entry points and a living
// process can get a new activity (SPEC §24.2).
//
// Flutter: shell/platform/android/platform_view_android_jni_impl.cc (the calls between the Java
// embedding and the engine)
type bridge struct {
	mu    sync.Mutex
	items []func(*Platform)
	// running is whether a Platform.Run drains the queue now, and wake wakes it.
	running bool
	wake    func()
	log     *slog.Logger
	// wait is how long surfaceDestroyed waits; destroyWait, or less in a test.
	wait time.Duration
	// surf, focused and lifecycle are the state the events change; they are read and written with
	// mu held, on the UI thread or, while no UI thread runs, on the calling thread.
	surf      surfaceState
	focused   bool
	lifecycle lifecycleState
	// settings are the system's settings the Java side reported last.
	settings systemSettings
}

func newBridge() *bridge { return &bridge{wait: destroyWait, log: slog.Default()} }

// post adds fn to the queue and wakes the UI thread. It is safe to call from any goroutine and
// any native thread. Functions posted by one thread run in the order it posted them.
func (b *bridge) post(fn func(*Platform)) {
	b.mu.Lock()
	b.items = append(b.items, fn)
	wake := b.wake
	b.mu.Unlock()
	if wake != nil {
		wake()
	}
}

// pop removes and returns the oldest function.
func (b *bridge) pop() (func(*Platform), bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.items) == 0 {
		return nil, false
	}
	fn := b.items[0]
	b.items[0] = nil
	b.items = b.items[1:]
	return fn, true
}

// start says that a UI thread drains the queue, and how to wake it. It reports false when one
// does already.
func (b *bridge) start(wake func()) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running {
		return false
	}
	b.running, b.wake = true, wake
	return true
}

// stop says that the UI thread no longer drains the queue.
func (b *bridge) stop() {
	b.mu.Lock()
	b.running, b.wake = false, nil
	b.mu.Unlock()
}

// snapshot returns a copy of the surface state.
func (b *bridge) snapshot() surfaceState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.surf
}

// event changes the state with apply and then tells the platform with notify. While a UI thread
// runs, both happen there, in order with the other events; while none does, apply happens now, on
// the calling thread, and nobody needs telling: the platform reads the state when it starts.
func (b *bridge) event(apply func(*bridge), notify func(*Platform)) {
	b.mu.Lock()
	if !b.running {
		apply(b)
		b.mu.Unlock()
		return
	}
	b.items = append(b.items, func(p *Platform) {
		b.mu.Lock()
		apply(b)
		b.mu.Unlock()
		if notify != nil {
			notify(p)
		}
	})
	wake := b.wake
	b.mu.Unlock()
	if wake != nil {
		wake()
	}
}

// eventAndWait is event for a change that the caller waits for: it returns when the UI thread
// has applied the change and told the platform, or when the wait time is over (it then reports
// false and the change happens later). While no UI thread runs it happens at once.
func (b *bridge) eventAndWait(apply func(*bridge), notify func(*Platform)) bool {
	done := make(chan struct{})
	b.mu.Lock()
	if !b.running {
		apply(b)
		b.mu.Unlock()
		return true
	}
	b.items = append(b.items, func(p *Platform) {
		b.mu.Lock()
		apply(b)
		b.mu.Unlock()
		if notify != nil {
			notify(p)
		}
		close(done)
	})
	wake := b.wake
	wait := b.wait
	b.mu.Unlock()
	if wake != nil {
		wake()
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-done:
		return true
	case <-t.C:
		return false
	}
}

// The events of the Java side. Each is called on a thread of Android (the main thread), and
// changes what the UI thread sees (SPEC §24.3, §24.4).

// surfaceCreated is SurfaceHolder.Callback.surfaceCreated: the view has a new native window,
// which the caller acquired. A window left over from a missed destroy is released.
func (b *bridge) surfaceCreated(window uintptr) {
	var old uintptr
	b.event(func(b *bridge) { old = b.surf.created(window) }, func(p *Platform) {
		if old != 0 {
			p.surfaceLost(old)
		}
		p.surfaceChangedState()
	})
}

// surfaceChanged is surfaceChanged: the size of the surface and the display.
func (b *bridge) surfaceChanged(width, height int, density, refresh float64, displayWidth, displayHeight int) {
	var differs bool
	b.event(func(b *bridge) {
		differs = b.surf.changed(width, height, density, refresh, displayWidth, displayHeight)
	}, func(p *Platform) {
		if differs {
			p.metricsChanged()
		}
	})
}

// surfaceDestroyed is surfaceDestroyed. It must not return while the renderer still draws into
// the surface, so it waits for the UI thread to let go of it, for at most the wait time (SPEC
// §24.3 "The one wait"); the UI thread is between frames when it does.
func (b *bridge) surfaceDestroyed() {
	var old uintptr
	ok := b.eventAndWait(func(b *bridge) { old = b.surf.destroyed() }, func(p *Platform) {
		if old != 0 {
			p.surfaceLost(old)
		}
	})
	if !ok {
		b.log.Warn("the UI thread did not let go of the surface in time; going on", "wait", b.wait.String())
	}
}

// lifecycleChanged is the activity's lifecycle (SPEC §24.8).
func (b *bridge) lifecycleChanged(state lifecycleState) {
	b.event(func(b *bridge) { b.lifecycle = state }, func(p *Platform) { p.lifecycleChanged() })
}

// focusChanged is Activity.onWindowFocusChanged.
func (b *bridge) focusChanged(focused bool) {
	b.event(func(b *bridge) { b.focused = focused }, func(p *Platform) { p.focusChanged() })
}

// snapshotAll returns the surface state, the lifecycle and the focus together.
func (b *bridge) snapshotAll() (surfaceState, lifecycleState, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.surf, b.lifecycle, b.focused
}

// insetsChanged is the insets of the window (SPEC §24.7): the metrics of the view change.
func (b *bridge) insetsChanged(in insets) {
	var differs bool
	b.event(func(b *bridge) { differs = b.surf.insetsChanged(in) }, func(p *Platform) {
		if differs {
			p.metricsChanged()
		}
	})
}

// displayChanged is a change of the display alone: its density, refresh rate or size.
func (b *bridge) displayChanged(density, refresh float64, displayWidth, displayHeight int) {
	var differs bool
	b.event(func(b *bridge) { differs = b.surf.displayChanged(density, refresh, displayWidth, displayHeight) }, func(p *Platform) {
		if differs {
			p.metricsChanged()
		}
	})
}

// settingsChanged is the text of the system's settings (settings.go). A text that cannot be read
// is logged and dropped.
func (b *bridge) settingsChanged(text string) {
	s, err := parseSettings(text)
	if err != nil {
		b.log.Error("the system's settings were not understood", "err", err)
		return
	}
	var differs bool
	b.event(func(b *bridge) {
		differs = !sameSettings(b.settings, s)
		b.settings = s
	}, func(p *Platform) {
		if differs {
			p.settingsChanged()
		}
	})
}

// systemUIChanged is the system showing (true) or hiding (false) its bars, because of the user.
func (b *bridge) systemUIChanged(visible bool) {
	b.event(func(*bridge) {}, func(p *Platform) {
		if p.win != nil && p.win.uiChange != nil {
			p.win.uiChange(visible)
		}
	})
}

// settingsSnapshot returns the settings the Java side reported last.
func (b *bridge) settingsSnapshot() systemSettings {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.settings
}

// textEdit is an edit of the input method, with the number that the Java side gave it (SPEC §24.6).
func (b *bridge) textEdit(seq int, text string, base, extent, composeBase, composeExtent int) {
	b.post(func(p *Platform) { p.textEdit(seq, text, base, extent, composeBase, composeExtent) })
}

// clipboardText is the activity's answer to a read of the clipboard.
func (b *bridge) clipboardText(id int, ok bool, text string) {
	b.post(func(p *Platform) { p.clipboardText(id, ok, text) })
}
