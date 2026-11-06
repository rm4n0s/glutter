//go:build js && wasm

package main

import (
	"encoding/json"
	"log"
	"syscall/js"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/gamepad"
	"github.com/rm4n0s/glutter/internal/platform/web"
	m "github.com/rm4n0s/glutter/material"
)

// observations is what the program posts as the result "pads".
type observations struct {
	// ReadsBeforeConnect is how many times the page read navigator.getGamepads before a pad was plugged in (the read made when the
	// service starts, and nothing after it).
	ReadsBeforeConnect int `json:"readsBeforeConnect"`
	// Pads are the names of the pads the framework knew after the plug, and their kinds.
	Pads  []string `json:"pads"`
	Kinds []string `json:"kinds"`
	// ReadsWhileConnected is how many reads were made in half a second with the pad plugged in (about one for each frame).
	ReadsWhileConnected int `json:"readsWhileConnected"`
	// FocusStart is the button that had the focus at the start, CountAfterA the counts after the A button was pressed, FocusAfterDown
	// the button that had the focus after the D-pad was pressed down, and CountAfterDownA the counts after A again.
	FocusStart      int    `json:"focusStart"`
	CountAfterA     [2]int `json:"countAfterA"`
	FocusAfterDown  int    `json:"focusAfterDown"`
	CountAfterDownA [2]int `json:"countAfterDownA"`
	// Axis is the left stick's value that the pad reported, and Trigger the right trigger's.
	AxisLeftX    float64 `json:"axisLeftX"`
	TriggerRight float64 `json:"triggerRight"`
	// CanRumble says the pad can rumble; Effects are the effects that the vibration actuator played, as "type duration strong weak",
	// and Resets how many times it was reset.
	CanRumble bool     `json:"canRumble"`
	Effects   []string `json:"effects"`
	Resets    int      `json:"resets"`
	// ThrowTag is the tag of the error of a rumble whose actuator throws, and NoActuatorTag of one on a pad that has no actuator.
	ThrowTag      string `json:"throwTag"`
	NoActuatorTag string `json:"noActuatorTag"`
	// PadsAfterUnplug is how many pads the framework knew after the pad was unplugged, and ReadsAfterUnplug how many reads the page
	// made in half a second after that (none: the page reads the pads only while one is connected).
	PadsAfterUnplug  int `json:"padsAfterUnplug"`
	ReadsAfterUnplug int `json:"readsAfterUnplug"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// The script of the browser's pads is installed before the app starts, so that the first read of the platform sees it.
	installFakePads()
	p := newProbe()
	go func() {
		postResult("pads", observe(p))
	}()
	return m.RunApp(newPadApp(p), m.RunAppOpts{Window: m.WindowConfig{Title: "Pad test"}})
}

// installFakePads replaces navigator.getGamepads with a function that answers from globalThis.fakePads and counts its calls in
// globalThis.padReads. The pad it makes is what Chrome gives for an Xbox 360 controller with the standard mapping.
func installFakePads() {
	js.Global().Get("Function").New(`
		globalThis.fakePads = [];
		globalThis.padReads = 0;
		globalThis.padEffects = [];
		globalThis.padResets = 0;
		navigator.getGamepads = function () { globalThis.padReads++; return globalThis.fakePads.slice(); };
		globalThis.makePad = function () {
			var buttons = [];
			for (var i = 0; i < 17; i++) buttons.push({pressed: false, value: 0});
			return {
				id: "Xbox 360 Controller (XInput STANDARD GAMEPAD)", index: 0, connected: true, mapping: "standard",
				timestamp: performance.now(), axes: [0, 0, 0, 0], buttons: buttons,
				vibrationActuator: {
					type: "dual-rumble", effects: ["dual-rumble"],
					playEffect: function (type, p) {
						globalThis.padEffects.push(type + " " + p.duration + " " + p.strongMagnitude.toFixed(2) + " " + p.weakMagnitude.toFixed(2));
						return Promise.resolve("complete");
					},
					reset: function () { globalThis.padResets++; return Promise.resolve("complete"); }
				}
			};
		};
		globalThis.setPad = function (fn) {
			var pad = globalThis.fakePads[0];
			fn(pad);
			pad.timestamp = performance.now();
		};
	`).Invoke()
}

// observe waits for the app and plays the pad. It runs on a goroutine of its own, as the driver of the input test does.
func observe(p *probe) observations {
	var o observations
	<-p.ready
	g := js.Global()
	settle := func(d time.Duration) { time.Sleep(d) }
	reads := func() int { return g.Get("padReads").Int() }
	settle(time.Second) // the first frames
	o.ReadsBeforeConnect = reads()

	// Plug the pad in: the browser fires gamepadconnected.
	g.Get("fakePads").Call("push", g.Call("makePad"))
	g.Call("dispatchEvent", g.Get("Event").New("gamepadconnected"))
	settle(300 * time.Millisecond)
	p.onUI(func() {
		for _, pad := range gamepad.ForBinding(p.binding).Pads() {
			o.Pads = append(o.Pads, pad.Name())
			o.Kinds = append(o.Kinds, pad.Kind().String())
			o.CanRumble = pad.CanRumble()
		}
		o.FocusStart = p.focused()
	})
	before := reads()
	settle(500 * time.Millisecond)
	o.ReadsWhileConnected = reads() - before

	press := func(index int, down bool) {
		g.Call("setPad", js.FuncOf(func(_ js.Value, args []js.Value) any {
			b := args[0].Get("buttons").Index(index)
			b.Set("pressed", down)
			b.Set("value", map[bool]float64{true: 1, false: 0}[down])
			return nil
		}))
		settle(150 * time.Millisecond)
	}
	// A activates the button that has the focus.
	press(0, true)
	press(0, false)
	p.onUI(func() { o.CountAfterA = p.counts })
	// The D-pad down moves the focus, and A activates the next button.
	press(13, true)
	press(13, false)
	p.onUI(func() { o.FocusAfterDown = p.focused() })
	press(0, true)
	press(0, false)
	p.onUI(func() { o.CountAfterDownA = p.counts })

	// The left stick and the right trigger are the browser's axis 0 and the analog value of button 7.
	g.Call("setPad", js.FuncOf(func(_ js.Value, args []js.Value) any {
		args[0].Get("axes").SetIndex(0, 0.5)
		args[0].Get("buttons").Index(7).Set("value", 1)
		args[0].Get("buttons").Index(7).Set("pressed", true)
		return nil
	}))
	settle(150 * time.Millisecond)
	p.onUI(func() {
		for _, pad := range gamepad.ForBinding(p.binding).Pads() {
			o.AxisLeftX = pad.State().Axis(gamepad.LeftX)
			o.TriggerRight = pad.State().Axis(gamepad.RightTrigger)
		}
	})

	// Rumble through the vibration actuator, then stop it.
	p.onUI(func() {
		for _, pad := range gamepad.ForBinding(p.binding).Pads() {
			if err := pad.Rumble(0.5, 0.25, time.Second); err != nil {
				log.Println("rumble:", err)
			}
		}
	})
	settle(100 * time.Millisecond)
	p.onUI(func() {
		for _, pad := range gamepad.ForBinding(p.binding).Pads() {
			if err := pad.Rumble(0, 0, 0); err != nil {
				log.Println("rumble stop:", err)
			}
		}
	})
	settle(100 * time.Millisecond)
	effects := g.Get("padEffects")
	for i := 0; i < effects.Length(); i++ {
		o.Effects = append(o.Effects, effects.Index(i).String())
	}
	o.Resets = g.Get("padResets").Int()

	// An actuator that throws, and a pad with none: the errors have their tags.
	tagOf := func(err error) string {
		if appErr, ok := errors.FromError(err); ok {
			return appErr.Tag
		}
		return ""
	}
	g.Call("setPad", js.FuncOf(func(_ js.Value, args []js.Value) any {
		args[0].Get("vibrationActuator").Set("playEffect", g.Get("Function").New("throw new Error('boom')"))
		return nil
	}))
	settle(100 * time.Millisecond)
	p.onUI(func() {
		for _, pad := range gamepad.ForBinding(p.binding).Pads() {
			o.ThrowTag = tagOf(pad.Rumble(0.5, 0.5, time.Second))
		}
	})
	g.Call("setPad", js.FuncOf(func(_ js.Value, args []js.Value) any {
		args[0].Set("vibrationActuator", js.Undefined())
		return nil
	}))
	settle(100 * time.Millisecond)
	p.onUI(func() {
		for _, pad := range gamepad.ForBinding(p.binding).Pads() {
			o.NoActuatorTag = tagOf(pad.Rumble(0.5, 0.5, time.Second))
		}
	})

	// Unplug it: the page stops reading.
	g.Set("fakePads", g.Get("Array").New())
	g.Call("dispatchEvent", g.Get("Event").New("gamepaddisconnected"))
	settle(300 * time.Millisecond)
	p.onUI(func() { o.PadsAfterUnplug = len(gamepad.ForBinding(p.binding).Pads()) })
	before = reads()
	settle(500 * time.Millisecond)
	o.ReadsAfterUnplug = reads() - before
	return o
}

// postResult posts a result to the test server; the answer is not waited for.
func postResult(kind string, data any) {
	body, err := json.Marshal(map[string]any{"kind": kind, "data": data})
	if err != nil {
		panic(err)
	}
	init := map[string]any{"method": "POST", "headers": map[string]any{"Content-Type": "application/json"}, "body": string(body)}
	js.Global().Call("fetch", web.ResultsPath, init)
}
