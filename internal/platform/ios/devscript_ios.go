//go:build ios

package ios

import (
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// devTouchID is the identity of the touch that a script makes: no UITouch has it.
const devTouchID uintptr = 0xd00d

// devFocusLimit is how long a text step of a script waits for a text field to have the focus, and devFocusPoll how often it looks.
const (
	devFocusLimit = 10 * time.Second
	devFocusPoll  = 50 * time.Millisecond
)

// devFrame is the time between the steps of a drag of a script.
const devFrame = 16 * time.Millisecond

// startDevEvents starts the script of GLUTTER_DEV_EVENTS once, after the first window is shown: a short wait first, so that the framework
// has the first frame's layout (an event that came before it would find no widget).
func (p *Platform) startDevEvents() {
	if p.devStarted || len(p.devEvents) == 0 {
		return
	}
	p.devStarted = true
	p.devBegan = time.Now()
	p.addTimer(time.Now().Add(300*time.Millisecond), func() { p.runDevEvents(0) })
}

// ScriptPending says that a script of GLUTTER_DEV_EVENTS was given and has not run its last step yet (platform.ScriptWaiter): the
// engine's self-capture waits for it.
func (p *Platform) ScriptPending() bool { return len(p.devEvents) > 0 && !p.devDone }

// runDevEvents runs the steps from index i in the first window; a wait or a drag arms a timer for the rest.
func (p *Platform) runDevEvents(i int) {
	wins := p.sortedWindows()
	if len(wins) == 0 {
		p.devDone = true
		return
	}
	w := wins[0]
	for ; i < len(p.devEvents); i++ {
		ev := p.devEvents[i]
		switch ev.Kind {
		case "wait":
			next := i + 1
			p.addTimer(time.Now().Add(ev.Wait), func() { p.runDevEvents(next) })
			return
		case "tap":
			w.devTouch(touchBegan, ev.X, ev.Y)
			w.devTouch(touchEnded, ev.X, ev.Y)
		case "drag":
			next := i + 1
			w.devDrag(ev, 0, func() { p.runDevEvents(next) })
			return
		case "key":
			w.devKey(ev)
		case "text", "compose", "composing":
			// The text goes to a field that has the focus: the script waits for one (the framework gives the focus in a frame of its own),
			// and types into none after the limit.
			if !w.text.enabled && time.Since(p.devBegan) < devFocusLimit {
				p.addTimer(time.Now().Add(devFocusPoll), func() { p.runDevEvents(i) })
				return
			}
			if ev.Kind == "text" {
				w.devText(ev.Text)
			} else {
				w.devCompose(ev.Steps, ev.Kind == "compose")
			}
		case "keyboard":
			w.devKeyboard(ev.Height)
		case "keyboardhide":
			w.devKeyboard(0)
		case "chrome":
			w.logChrome()
		}
	}
	p.devDone = true
}

// devTouch is a touch of the script at a place.
func (w *window) devTouch(phase touchPhase, x, y float64) {
	w.dispatchTouches([]touchSample{{ID: devTouchID, Phase: phase, X: x, Y: y, Time: w.now()}})
}

// devDrag moves the finger of a drag a step at a time, and goes on with done when it has lifted.
func (w *window) devDrag(ev DevEvent, step int, done func()) {
	steps := int(ev.Wait / devFrame)
	if steps < 1 {
		steps = 1
	}
	at := func(s int) (float64, float64) {
		f := float64(s) / float64(steps)
		return ev.X + (ev.X2-ev.X)*f, ev.Y + (ev.Y2-ev.Y)*f
	}
	if step == 0 {
		w.devTouch(touchBegan, ev.X, ev.Y)
	}
	step++
	x, y := at(step)
	if step >= steps {
		w.devTouch(touchMoved, x, y)
		w.devTouch(touchEnded, x, y)
		done()
		return
	}
	w.devTouch(touchMoved, x, y)
	w.p.addTimer(time.Now().Add(devFrame), func() { w.devDrag(ev, step, done) })
}

// devKey presses the key of a script with its modifiers and releases them in the opposite order.
func (w *window) devKey(ev DevEvent) {
	var flags uint64
	var held []string
	press := func(phase pressPhase, code uint32, chars string) {
		w.p.keys.handle(pressEvent{phase: phase, keyCode: code, modifierFlags: flags, timestamp: w.now(), characters: chars, ignoring: chars})
	}
	for _, name := range devModifierOrder {
		if containsMod(ev.Mods, name) {
			m := devModifiers[name]
			flags |= m.flag
			press(pressBegan, m.code, "")
			held = append(held, name)
		}
	}
	code := devKeyCodes[ev.Key]
	chars := devKeyCharacters(ev.Key, containsMod(ev.Mods, "shift"))
	press(pressBegan, code, chars)
	press(pressEnded, code, chars)
	for i := len(held) - 1; i >= 0; i-- {
		m := devModifiers[held[i]]
		press(pressEnded, m.code, "")
		flags &^= m.flag
	}
}

// containsMod says whether the "+"-joined modifiers have the name.
func containsMod(mods, name string) bool {
	for _, m := range splitMods(mods) {
		if m == name {
			return true
		}
	}
	return false
}

func splitMods(mods string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(mods); i++ {
		if i == len(mods) || mods[i] == '+' {
			if i > start {
				out = append(out, mods[start:i])
			}
			start = i + 1
		}
	}
	return out
}

// devText types the text as UIKit does for a keyboard that composes nothing: one insertText: on the text input view for each character.
func (w *window) devText(text string) {
	if w.inputView == 0 {
		return
	}
	for _, r := range text {
		if r < 0x20 || r == 0x7f {
			continue // the control characters type nothing
		}
		s := objcx.NewString(string(r))
		w.inputView.Send(sel("insertText:"), s)
		objcx.Release(s)
	}
}

// devCompose is an input method composing: setMarkedText:selectedRange: on the text input view for each step, with the caret at its end,
// and then insertText: with the last one when the composition is committed.
func (w *window) devCompose(steps []string, commit bool) {
	if w.inputView == 0 {
		return
	}
	for _, step := range steps {
		s := objcx.NewString(step)
		w.inputView.Send(sel("setMarkedText:selectedRange:"), s, uint64(u16Len(step)), uint64(0))
		objcx.Release(s)
	}
	if commit && len(steps) > 0 {
		s := objcx.NewString(steps[len(steps)-1])
		w.inputView.Send(sel("insertText:"), s)
		objcx.Release(s)
	}
}

// devKeyboard posts the notification that UIKit posts when the keyboard comes up to the height in points (0 sends it down), with the
// duration of a keyboard's animation, as the system would: the platform's observer takes it from there.
func (w *window) devKeyboard(height float64) {
	screen := class("UIScreen").Send(sel("mainScreen"))
	b := send[darwin.CGRect](screen, "bounds")
	y := b.Size.Height - height
	frame := darwin.CGRect{Origin: darwin.CGPoint{X: 0, Y: y}, Size: darwin.CGSize{Width: b.Size.Width, Height: height}}
	value := class("NSValue").Send(sel("valueWithCGRect:"), frame)
	frameKey, durKey := objcx.NewString("UIKeyboardFrameEndUserInfoKey"), objcx.NewString("UIKeyboardAnimationDurationUserInfoKey")
	duration := objcx.NewNumberFloat(0.25)
	dict := class("NSMutableDictionary").Send(sel("dictionary"))
	dict.Send(sel("setObject:forKey:"), value, frameKey)
	dict.Send(sel("setObject:forKey:"), duration, durKey)
	name := objcx.NewString(keyboardNotification)
	class("NSNotificationCenter").Send(sel("defaultCenter")).Send(sel("postNotificationName:object:userInfo:"), name, objc.ID(0), dict)
	for _, o := range []objc.ID{frameKey, durKey, duration, name} {
		objcx.Release(o)
	}
}
