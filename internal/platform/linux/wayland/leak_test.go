//go:build linux && !android

package wayland

import (
	"context"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// clearLogs forgets what the fake compositor and the test handler keep of the events, so that the
// log is not counted as the memory of the platform.
func (e *keyEnv) clearLogs() {
	e.srv.ClearRequests()
	e.h.mu.Lock()
	e.h.keys, e.h.batches, e.h.texts, e.h.drops, e.h.dragEnds = nil, nil, nil, nil, nil
	e.h.mu.Unlock()
	e.seen, e.seenBatches = 0, 0
	e.onLoop(func() bool { e.clock.timers = nil; return true }) // the fake clock keeps every timer it made
}

// cursorStep sets the cursors one after the other, a step each.
func cursorStep(e *keyEnv, cursors ...platform.MouseCursor) func() {
	i := 0
	return func() {
		c := cursors[i%len(cursors)]
		i++
		e.onLoop(func() bool {
			for _, w := range e.p.windows {
				w.SetCursor(c)
			}
			return true
		})
		e.settle()
	}
}

// TestLeaksPlatform drives the Wayland platform with a fake compositor (step 29.21, SPEC §15
// "Memory over time", §8.1): the live heap, the goroutines, and the blocks of C memory that the
// platform made (dl.Malloc, dl.Calloc, dl.CString) and did not free.
func TestLeaksPlatform(t *testing.T) {
	const seatAll = capPointer | capTouch
	tests := []struct {
		name  string
		start func(t *testing.T) *keyEnv
		step  func(t *testing.T, e *keyEnv) func()
		n     int
	}{
		{"key presses", func(t *testing.T) *keyEnv { return startKeyboard(t, "us") }, func(t *testing.T, e *keyEnv) func() {
			return func() {
				e.srv.KeyboardKey(30, true) // a
				e.srv.KeyboardKey(30, false)
				e.settle()
			}
		}, 300},
		{"modifiers and compose keys", func(t *testing.T) *keyEnv { return startKeyboard(t, "us") }, func(t *testing.T, e *keyEnv) func() {
			return func() {
				e.srv.KeyboardModifiers(1, 0, 0, 0)
				e.srv.KeyboardKey(30, true)
				e.srv.KeyboardKey(30, false)
				e.srv.KeyboardModifiers(0, 0, 0, 0)
				e.settle()
			}
		}, 300},
		{"pointer, buttons and wheel", func(t *testing.T) *keyEnv { return startInput(t, capPointer, 0) }, func(t *testing.T, e *keyEnv) func() {
			e.srv.PointerEnter(e.surface, 10, 20)
			e.srv.PointerFrame()
			x := 0.0
			return func() {
				x++
				e.srv.PointerMotion(10+x, 20)
				e.srv.PointerButton(0x110, true)
				e.srv.PointerButton(0x110, false)
				e.srv.PointerAxis(0, 10)
				e.srv.PointerFrame()
				e.settle()
			}
		}, 300},
		{"touches", func(t *testing.T) *keyEnv { return startInput(t, seatAll, 0) }, func(t *testing.T, e *keyEnv) func() {
			id := int32(0)
			return func() {
				id++
				e.srv.TouchDown(e.surface, id, 30, 40)
				e.srv.TouchFrame()
				e.srv.TouchMotion(id, 35, 45)
				e.srv.TouchFrame()
				e.srv.TouchUp(id)
				e.srv.TouchFrame()
				e.settle()
			}
		}, 300},
		{"cursor shapes", func(t *testing.T) *keyEnv { return startInput(t, capPointer, 0, cursorShapeGlobal) }, func(t *testing.T, e *keyEnv) func() {
			e.srv.PointerEnter(e.surface, 10, 20)
			e.srv.PointerFrame()
			e.settle()
			return cursorStep(e, platform.MouseCursorText, platform.MouseCursorClick, platform.MouseCursorWait, platform.MouseCursorBasic)
		}, 300},
		{"cursors of the theme", func(t *testing.T) *keyEnv { return startInput(t, capPointer, 0) }, func(t *testing.T, e *keyEnv) func() {
			e.srv.PointerEnter(e.surface, 10, 20)
			e.srv.PointerFrame()
			e.settle()
			return cursorStep(e, platform.MouseCursorText, platform.MouseCursorClick, platform.MouseCursorWait, platform.MouseCursorBasic)
		}, 300},
		{"clipboard", func(t *testing.T) *keyEnv {
			return clipboardEnv(t, fakecomp.Config{})
		}, func(t *testing.T, e *keyEnv) func() {
			i := 0
			return func() {
				i++
				text := strings.Repeat("clipboard ", 20+i%5)
				if err := e.setText(text); err != nil {
					t.Fatal(err)
				}
				e.srv.OfferSelectionData(map[string][]byte{"text/plain;charset=utf-8": []byte(text)})
				e.settle()
				if r := e.getText(context.Background()); r.err != nil {
					t.Fatal(r.err)
				}
			}
		}, 150},
		{"windows opened and closed", func(t *testing.T) *keyEnv { return startKeyboard(t, "us") }, func(t *testing.T, e *keyEnv) func() {
			return func() {
				var w platform.Window
				var err error
				e.onLoop(func() bool { w, err = e.p.NewWindow(config640()); return true })
				if err != nil {
					t.Fatal(err)
				}
				e.settle()
				e.onLoop(func() bool { w.RequestFrame(); return true })
				e.settle()
				e.onLoop(func() bool { w.Close(); return true })
				e.settle()
			}
		}, 100},
		{"window title and frames", func(t *testing.T) *keyEnv { return startKeyboard(t, "us") }, func(t *testing.T, e *keyEnv) func() {
			i := 0
			return func() {
				i++
				e.onLoop(func() bool {
					for _, w := range e.p.windows {
						w.SetTitle("title " + strings.Repeat("x", i%10))
						w.RequestFrame()
						w.SetTextInput(ui.TextInputState{})
					}
					return true
				})
				e.settle()
			}
		}, 300},
	}
	cheap := func() int { b, _ := dl.CHeapInUse(); return int(b >> 10) } // KB of the C heap, libwayland and libxkbcommon included
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e *keyEnv
			leaktest.Run(t, "wayland/"+tt.name, func() func() {
				e = tt.start(t)
				step := tt.step(t, e)
				return func() { step(); e.clearLogs() }
			}, leaktest.Opts{N: tt.n, CounterSlack: 64, Counters: map[string]func() int{
				"cblocks": func() int { return int(dl.LiveBlocks()) },
				"cheapKB": cheap,
			}})
		})
	}
}

// TestLeaksPlatformConnections makes a platform on a fake compositor and closes it, again and again:
// the interface tables, the xkb context and keymap, the windows and the connection are freed when
// the platform is.
func TestLeaksPlatformConnections(t *testing.T) {
	leaktest.Run(t, "wayland/connect and close", func() func() {
		return func() {
			if _, err := wl.Connect("glutter-probe-for-libwayland"); err != nil {
				if appErr, ok := errors.FromError(err); ok && appErr.Tag == tagLibraryLoadFailed {
					t.Skipf("libwayland-client is not available: %v", err)
				}
			}
			cfg := fakecomp.Config{OutputScale: 1, PreferredBufferScale: 1, Keymap: keymapFor(t, "us")}
			srv, fd := fakecomp.New(t, cfg)
			conn, err := wl.ConnectFD(fd)
			if err != nil {
				t.Fatal(err)
			}
			p, err := newPlatform(conn, Options{ShmFill: true, NoLibdecor: true})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := p.NewWindow(config640()); err != nil {
				t.Fatal(err)
			}
			if err := p.Close(); err != nil {
				t.Fatal(err)
			}
			srv.Wait()
		}
	}, leaktest.Opts{N: 15, Warmup: 3, Counters: map[string]func() int{
		"cblocks": func() int { return int(dl.LiveBlocks()) },
	}})
}
