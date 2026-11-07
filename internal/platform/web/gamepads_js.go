//go:build js && wasm

package web

import (
	"sync/atomic"
	"syscall/js"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// The gamepad service of the web (SPEC §28.2, step 31.2): navigator.getGamepads() is read once for each animation frame while a
// pad is connected, and never otherwise (a page with no pad does no work for pads). A pad is found by a read made when the service
// starts and when the browser fires gamepadconnected or gamepaddisconnected; the rest is in gamepads.go.

type padService struct {
	p   *Platform
	ids atomic.Int64

	// Everything below belongs to the UI goroutine.
	h          platform.PadHandler
	poller     *PadPoller
	running    bool
	rafPending bool
	connFn     js.Func
	rafFn      js.Func
	made       bool
}

var _ platform.Gamepads = (*padService)(nil)

// gamepads makes the service on the first call; it is nil where the browser has no Gamepad API.
func (p *Platform) gamepads() platform.Gamepads {
	if p.pads == nil {
		if !isObject(global().Get("navigator")) || global().Get("navigator").Get("getGamepads").Type() != js.TypeFunction {
			return nil
		}
		p.pads = &padService{p: p}
	}
	return p.pads
}

// MappingPlatform implements platform.Gamepads.
func (s *padService) MappingPlatform() string { return "Web" }

// Start implements platform.Gamepads: the listeners are installed and the pads that are connected already are read once.
func (s *padService) Start(h platform.PadHandler) error {
	if s.running {
		return nil
	}
	if !s.made {
		s.made = true
		s.connFn = s.p.keep(js.FuncOf(func(js.Value, []js.Value) any {
			s.p.q.post(s.wake)
			return nil
		}))
		s.rafFn = s.p.keep(js.FuncOf(s.onAnimationFrame))
	}
	s.h, s.running = h, true
	s.poller = NewPadPoller(s.p.os, &s.ids)
	win := global()
	win.Call("addEventListener", "gamepadconnected", s.connFn)
	win.Call("addEventListener", "gamepaddisconnected", s.connFn)
	s.wake()
	return nil
}

// Stop implements platform.Gamepads.
func (s *padService) Stop() {
	if !s.running {
		return
	}
	s.running = false
	win := global()
	win.Call("removeEventListener", "gamepadconnected", s.connFn)
	win.Call("removeEventListener", "gamepaddisconnected", s.connFn)
	s.h, s.poller = nil, nil
}

// wake reads the pads once, now, and starts the frame loop if there is a pad.
func (s *padService) wake() {
	if !s.running {
		return
	}
	s.update(time.Duration(number(global().Get("performance").Call("now"), 0)*float64(time.Millisecond)), readBrowserPads())
}

// update takes a reading and goes on with the loop while there is a pad.
func (s *padService) update(now time.Duration, pads []BrowserPad) {
	if !s.running {
		return
	}
	s.rafPending = false
	s.poller.Update(now, pads, s.h)
	if s.poller.Count() > 0 && !s.rafPending {
		s.rafPending = true
		global().Call("requestAnimationFrame", s.rafFn)
	}
}

// onAnimationFrame is the animation frame callback: it reads the pads and queues the reading, as the callbacks of the platform
// copy what they need into Go values and queue the change (SPEC §23.3).
func (s *padService) onAnimationFrame(_ js.Value, args []js.Value) any {
	ms := number(args[0], 0)
	pads := readBrowserPads()
	s.p.q.post(func() { s.update(FrameTime(ms), pads) })
	return nil
}

// readBrowserPads copies navigator.getGamepads() into Go values; the gaps of the array are left out.
func readBrowserPads() []BrowserPad {
	nav := global().Get("navigator")
	var arr js.Value
	if !try(func() { arr = nav.Call("getGamepads") }) || !isObject(arr) {
		return nil
	}
	n := arr.Length()
	var out []BrowserPad
	for i := 0; i < n; i++ {
		g := arr.Index(i)
		if !isObject(g) {
			continue
		}
		bp := BrowserPad{
			Index:     i,
			ID:        g.Get("id").String(),
			Connected: g.Get("connected").Truthy(),
			Timestamp: number(g.Get("timestamp"), 0),
		}
		if m := g.Get("mapping"); m.Type() == js.TypeString {
			bp.Mapping = m.String()
		}
		buttons, axes := g.Get("buttons"), g.Get("axes")
		if isObject(buttons) {
			bp.Buttons = make([]BrowserButton, buttons.Length())
			for j := range bp.Buttons {
				b := buttons.Index(j)
				bp.Buttons[j] = BrowserButton{Pressed: b.Get("pressed").Truthy(), Value: number(b.Get("value"), 0)}
			}
		}
		if isObject(axes) {
			bp.Axes = make([]float64, axes.Length())
			for j := range bp.Axes {
				bp.Axes[j] = number(axes.Index(j), 0)
			}
		}
		if act := g.Get("vibrationActuator"); isObject(act) {
			effects := act.Get("effects")
			has := func(name string) bool {
				if isObject(effects) {
					return effects.Call("includes", name).Truthy()
				}
				return false
			}
			bp.DualRumble = has("dual-rumble") || (act.Get("type").Type() == js.TypeString && act.Get("type").String() == "dual-rumble")
			bp.TriggerRumble = has("trigger-rumble")
		}
		out = append(out, bp)
	}
	return out
}

// actuator finds the vibration actuator of the pad with an id.
func (s *padService) actuator(id platform.PadID) (js.Value, bool) {
	if s.poller == nil {
		return js.Undefined(), false
	}
	index, ok := s.poller.IndexOf(id)
	if !ok {
		return js.Undefined(), false
	}
	var arr js.Value
	if !try(func() { arr = global().Get("navigator").Call("getGamepads") }) || !isObject(arr) {
		return js.Undefined(), false
	}
	g := arr.Index(index)
	if !isObject(g) {
		return js.Undefined(), false
	}
	act := g.Get("vibrationActuator")
	return act, isObject(act)
}

// The longest an effect can be: Chrome refuses a longer one.
const maxRumble = 5 * time.Second

// play starts an effect and does not wait for its promise (SPEC §23.3); a promise that fails is caught so the console is not
// filled with it.
func (s *padService) play(act js.Value, effect string, params map[string]any) error {
	var res js.Value
	if !try(func() { res = act.Call("playEffect", effect, params) }) {
		return errors.New(tagWebRumbleFailed, "playEffect threw", "effect", effect)
	}
	if isObject(res) && res.Get("catch").Type() == js.TypeFunction {
		res.Call("catch", s.p.noopFn)
	}
	return nil
}

func (s *padService) reset(act js.Value) {
	var res js.Value
	if try(func() { res = act.Call("reset") }) && isObject(res) && res.Get("catch").Type() == js.TypeFunction {
		res.Call("catch", s.p.noopFn)
	}
}

// Rumble implements platform.Gamepads with the "dual-rumble" effect; a duration of 0 stops it.
func (s *padService) Rumble(id platform.PadID, low, high uint16, d time.Duration) error {
	act, ok := s.actuator(id)
	if !ok {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad has no vibration actuator or is not there", "pad", int64(id))
	}
	if d <= 0 || (low == 0 && high == 0) {
		s.reset(act)
		return nil
	}
	return s.play(act, "dual-rumble", map[string]any{
		"startDelay": 0, "duration": float64(min(d, maxRumble) / time.Millisecond),
		"strongMagnitude": float64(low) / 65535, "weakMagnitude": float64(high) / 65535,
	})
}

// RumbleTriggers implements platform.Gamepads with the "trigger-rumble" effect, for the browsers whose actuator has it.
func (s *padService) RumbleTriggers(id platform.PadID, left, right uint16, d time.Duration) error {
	act, ok := s.actuator(id)
	if !ok || !isObject(act.Get("effects")) || !act.Get("effects").Call("includes", "trigger-rumble").Truthy() {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad cannot rumble its triggers", "pad", int64(id))
	}
	if d <= 0 || (left == 0 && right == 0) {
		s.reset(act)
		return nil
	}
	return s.play(act, "trigger-rumble", map[string]any{
		"startDelay": 0, "duration": float64(min(d, maxRumble) / time.Millisecond),
		"leftTrigger": float64(left) / 65535, "rightTrigger": float64(right) / 65535,
	})
}

// SetLED implements platform.Gamepads: the browser has no way to set the light of a pad.
func (s *padService) SetLED(id platform.PadID, r, g, b uint8) error {
	return errors.New(platform.TagGamepadLEDUnavailable, "the browser cannot set the light of a pad", "pad", int64(id))
}

// webServices is the platform.Services of the web: the gamepads, the audio and the motion sensors.
type webServices struct {
	pads    platform.Gamepads
	audio   platform.Audio
	sensors platform.Sensors
}

var _ platform.Services = webServices{}

// Accessibility implements platform.Services: the web has none of the engine's.
func (webServices) Accessibility() platform.Accessibility { return nil }

// Gamepads implements platform.Services.
func (s webServices) Gamepads() platform.Gamepads { return s.pads }

// Audio implements platform.Services: the audio worklet (audio_js.go).
func (s webServices) Audio() platform.Audio { return s.audio }

// Sensors implements platform.Services: DeviceMotionEvent (sensors_js.go).
func (s webServices) Sensors() platform.Sensors { return s.sensors }

// tagWebRumbleFailed is the tag of the error of a rumble that the browser refused (rm4n0s/errors, R5).
const tagWebRumbleFailed = "WebRumbleFailed"
