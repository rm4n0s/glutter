//go:build ios

package ios

import "time"

// devTouchID is the identity of the touch that a script makes: no UITouch has it.
const devTouchID uintptr = 0xd00d

// devFrame is the time between the steps of a drag of a script.
const devFrame = 16 * time.Millisecond

// startDevEvents starts the script of GLUTTER_DEV_EVENTS once, after the first window is shown: a short wait first, so that the framework
// has the first frame's layout (an event that came before it would find no widget).
func (p *Platform) startDevEvents() {
	if p.devStarted || len(p.devEvents) == 0 {
		return
	}
	p.devStarted = true
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
