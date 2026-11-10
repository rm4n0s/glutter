//go:build ios

package ios

import (
	"time"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// This file is the Game Controller framework (SPEC §30.9): GCController's connection notifications, the extended gamepad profile as the
// standard layout (gamecontroller.go), the light of a DualShock or a DualSense, and the haptics of a controller as rumble.
//
// The design follows SDL 3's src/joystick/apple/SDL_mfijoystick.m; the code here is new and is changed.

// The notifications of the Game Controller framework (the strings are the names of the constants).
const (
	controllerConnected    = "GCControllerDidConnectNotification"
	controllerDisconnected = "GCControllerDidDisconnectNotification"
)

// padService is platform.Gamepads over GCController. All its methods but those of the interface belong to the UI thread, where the
// framework posts its notifications and calls its handlers.
//
// Flutter: no counterpart
type padService struct {
	p   *Platform
	set *padSet
	obs objc.ID
	// handlers keeps the blocks of the value-changed handlers of the controllers (copied), by controller, until the controller goes.
	handlers map[uintptr]objc.Block
	// haptics are the engines made for rumble, by controller.
	engines map[uintptr]*rumbleEngine
	loaded  bool
}

var (
	_ platform.Gamepads    = (*padService)(nil)
	_ platform.GamepadHost = (*Platform)(nil)
)

// MappingPlatform implements platform.Gamepads: no table of the pads of this platform has an entry for the standard pads, which report the
// layout themselves.
func (s *padService) MappingPlatform() string { return "iOS" }

// registerPadObserver registers the class of the observer that the notification centre calls for a controller that connected or went.
func registerPadObserver() (*objcx.RegisteredClass, error) {
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "PadObserver", Super: "NSObject",
		Methods: []objcx.Method{
			{Name: "controllerConnected:", Fn: func(self objc.ID, _ objc.SEL, note objc.ID) {
				if p, ok := objcx.LookupAs[*Platform](objcx.HandleOf(self)); ok && !p.closed {
					p.guard(func() { p.pads().connected(send[objc.ID](note, "object")) })
				}
			}},
			{Name: "controllerDisconnected:", Fn: func(self objc.ID, _ objc.SEL, note objc.ID) {
				if p, ok := objcx.LookupAs[*Platform](objcx.HandleOf(self)); ok && !p.closed {
					p.guard(func() { p.pads().disconnected(send[objc.ID](note, "object")) })
				}
			}},
		},
	})
}

// pads is the platform's pad service, made at the first use.
func (p *Platform) pads() *padService {
	if p.padSvc == nil {
		p.padSvc = &padService{p: p, set: newPadSet(), handlers: map[uintptr]objc.Block{}, engines: map[uintptr]*rumbleEngine{}}
	}
	return p.padSvc
}

// Gamepads returns the pad service of the platform.
//
// Flutter: no counterpart
func (p *Platform) Gamepads() platform.Gamepads { return p.pads() }

// Start loads the framework, listens to the controllers that connect and go, and reports the ones that are connected (after Start has
// returned, as the interface says). It fails with the error of the loader (the tag of `dl`) when the Game Controller framework cannot
// be loaded.
func (s *padService) Start(h platform.PadHandler) error {
	if !s.loaded {
		if _, err := dl.OpenFramework("GameController"); err != nil {
			return err
		}
		s.loaded = true
	}
	s.obs = s.p.s.padObserver.New(s.p.handle)
	center := class("NSNotificationCenter").Send(sel("defaultCenter"))
	objcx.WithPool(func() {
		for name, selector := range map[string]string{controllerConnected: "controllerConnected:", controllerDisconnected: "controllerDisconnected:"} {
			n := objcx.NewString(name)
			center.Send(sel("addObserver:selector:name:object:"), s.obs, sel(selector), n, objc.ID(0))
			objcx.Release(n)
		}
		// The controllers that are there already come after Start returns; a wireless controller that is nearby is looked for.
		s.p.Post(func() {
			s.set.start(h)
			s.addAll()
		})
		class("GCController").Send(sel("startWirelessControllerDiscoveryWithCompletionHandler:"), objc.ID(0))
	})
	return nil
}

// Stop ends the reports and the discovery.
func (s *padService) Stop() {
	objcx.WithPool(func() {
		if s.obs != 0 {
			class("NSNotificationCenter").Send(sel("defaultCenter")).Send(sel("removeObserver:"), s.obs)
			objcx.Release(s.obs)
			s.obs = 0
		}
		if s.loaded {
			class("GCController").Send(sel("stopWirelessControllerDiscovery"))
		}
	})
	s.set.stop()
}

// addAll reports the controllers that are connected.
func (s *padService) addAll() {
	objcx.WithPool(func() {
		list := send[objc.ID](class("GCController"), "controllers")
		for i := 0; list != 0 && i < objcx.ArrayCount(list); i++ {
			s.connected(objcx.ArrayAt(list, i))
		}
	})
}

// specOf reads what the framework says of the controller: its name, the optional buttons of its extended gamepad, the haptics, the light
// and the battery. A controller with no extended gamepad (a micro gamepad such as the Siri Remote) is not a pad of this backend: ok is
// false.
func specOf(c objc.ID) (spec padSpec, ok bool) {
	gp := send[objc.ID](c, "extendedGamepad")
	if gp == 0 {
		return spec, false
	}
	spec.Name = objcx.String(send[objc.ID](c, "vendorName"))
	if spec.Name == "" {
		spec.Name = "Game Controller"
	}
	has := func(name string) bool { return send[objc.ID](gp, name) != 0 }
	spec.Features = gcFeatures{
		Options: has("buttonOptions"), Home: has("buttonHome"), ThumbstickButtons: has("leftThumbstickButton") && has("rightThumbstickButton"),
	}
	spec.Rumble = send[objc.ID](c, "haptics") != 0
	spec.LED = send[objc.ID](c, "light") != 0
	spec.Battery = -1
	if b := send[objc.ID](c, "battery"); b != 0 {
		if level := send[float32](b, "batteryLevel"); level >= 0 {
			spec.Battery = float64(level)
		}
	}
	return spec, true
}

// connected is a controller that connected (or one that was there when Start ran): it is added, and its extended gamepad's changes are
// followed.
func (s *padService) connected(c objc.ID) {
	spec, ok := specOf(c)
	if !ok {
		return
	}
	key := uintptr(c)
	if _, fresh := s.set.add(key, spec); !fresh {
		return
	}
	gp := send[objc.ID](c, "extendedGamepad")
	blk := objcx.NewBlock(func(_ objc.Block, pad, _ objc.ID) {
		if s.p.closed {
			return
		}
		s.p.guard(func() { s.set.update(key, stateOf(pad), s.p.since()) })
	})
	kept := blk.Copy()
	blk.Release()
	s.handlers[key] = kept
	gp.Send(sel("setValueChangedHandler:"), kept)
}

// disconnected is a controller that went: its handler and its haptics engine are released.
func (s *padService) disconnected(c objc.ID) {
	key := uintptr(c)
	s.set.remove(key)
	if blk, ok := s.handlers[key]; ok {
		if gp := send[objc.ID](c, "extendedGamepad"); gp != 0 {
			gp.Send(sel("setValueChangedHandler:"), objc.ID(0))
		}
		blk.Release()
		delete(s.handlers, key)
	}
	if e, ok := s.engines[key]; ok {
		e.close()
		delete(s.engines, key)
	}
}

// stateOf reads the state of an extended gamepad.
func stateOf(gp objc.ID) gcState {
	button := func(name string) bool {
		b := send[objc.ID](gp, name)
		return b != 0 && send[bool](b, "isPressed")
	}
	value := func(name string) float64 {
		b := send[objc.ID](gp, name)
		if b == 0 {
			return 0
		}
		return float64(send[float32](b, "value"))
	}
	axis := func(stick, name string) float64 {
		s := send[objc.ID](gp, stick)
		if s == 0 {
			return 0
		}
		return float64(send[float32](send[objc.ID](s, name), "value"))
	}
	dpad := send[objc.ID](gp, "dpad")
	pressed := func(name string) bool { return dpad != 0 && send[bool](send[objc.ID](dpad, name), "isPressed") }
	return gcState{
		A: button("buttonA"), B: button("buttonB"), X: button("buttonX"), Y: button("buttonY"),
		LeftShoulder: button("leftShoulder"), RightShoulder: button("rightShoulder"),
		LeftThumbstickButton: button("leftThumbstickButton"), RightThumbstickButton: button("rightThumbstickButton"),
		DPadUp: pressed("up"), DPadDown: pressed("down"), DPadLeft: pressed("left"), DPadRight: pressed("right"),
		Menu: button("buttonMenu"), Options: button("buttonOptions"), Home: button("buttonHome"),
		LeftTrigger: value("leftTrigger"), RightTrigger: value("rightTrigger"),
		LeftX: axis("leftThumbstick", "xAxis"), LeftY: axis("leftThumbstick", "yAxis"),
		RightX: axis("rightThumbstick", "xAxis"), RightY: axis("rightThumbstick", "yAxis"),
	}
}

// since is the time since the platform started, the clock of the pad events.
func (p *Platform) since() time.Duration { return time.Since(p.start) }

// controllerOf is the controller of the pad, 0 when there is none.
func (s *padService) controllerOf(id platform.PadID) objc.ID {
	key, ok := s.set.keyOf(id)
	if !ok {
		return 0
	}
	return objc.ID(key)
}

// SetLED sets the color of the light of the controller (a DualShock or a DualSense has one). It fails with GamepadLEDUnavailable when the
// controller has none or is gone.
func (s *padService) SetLED(id platform.PadID, r, g, b uint8) error {
	c := s.controllerOf(id)
	if c == 0 {
		return errors.New(platform.TagGamepadLEDUnavailable, "the pad is not connected", "pad", int64(id))
	}
	var light objc.ID
	objcx.WithPool(func() { light = send[objc.ID](c, "light") })
	if light == 0 {
		return errors.New(platform.TagGamepadLEDUnavailable, "the controller has no light", "pad", int64(id))
	}
	objcx.WithPool(func() {
		color := class("GCColor").Send(sel("alloc")).Send(sel("initWithRed:green:blue:"), float32(r)/255, float32(g)/255, float32(b)/255)
		light.Send(sel("setColor:"), color)
		objcx.Release(color)
	})
	return nil
}

// RumbleTriggers fails with GamepadRumbleUnavailable: the Game Controller framework has localities for the triggers of a controller, but
// the motors of the triggers are not driven by this backend (the handles' engine is what Rumble uses).
func (s *padService) RumbleTriggers(id platform.PadID, _, _ uint16, _ time.Duration) error {
	return errors.New(platform.TagGamepadRumbleUnavailable, "the triggers' own motors are not driven on iOS", "pad", int64(id))
}

// Rumble makes the controller rumble for the time with CoreHaptics: the engine of the controller's default locality plays a continuous
// event whose intensity is the strength of the two motors and whose sharpness says how much of it is the high-frequency one. A strength of
// zero (or a zero time) stops it. It fails with GamepadRumbleUnavailable when the controller has no haptics or is gone, and with
// GamepadRumbleFailed when CoreHaptics refuses.
func (s *padService) Rumble(id platform.PadID, low, high uint16, d time.Duration) error {
	c := s.controllerOf(id)
	if c == 0 {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad is not connected", "pad", int64(id))
	}
	key := uintptr(c)
	e := s.engines[key]
	if e == nil {
		var haptics objc.ID
		objcx.WithPool(func() { haptics = send[objc.ID](c, "haptics") })
		if haptics == 0 {
			return errors.New(platform.TagGamepadRumbleUnavailable, "the controller has no haptics", "pad", int64(id))
		}
		var err error
		if e, err = newRumbleEngine(haptics); err != nil {
			return err
		}
		s.engines[key] = e
	}
	intensity, sharpness := rumbleParams(low, high)
	if intensity == 0 || d <= 0 {
		e.stop()
		return nil
	}
	return e.play(intensity, sharpness, d)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGamepadRumbleFailed = "GamepadRumbleFailed"
)

// rumbleEngine is the CHHapticEngine of a controller and the player of the rumble that plays now.
type rumbleEngine struct {
	engine objc.ID
	player objc.ID
	// intensityID, sharpnessID and continuous are the CoreHaptics constants that name the parameters and the event type (strings that
	// the framework owns).
	intensityID, sharpnessID, continuous objc.ID
	started                              bool
}

// newRumbleEngine makes the engine of the haptics of a controller, with the constants of CoreHaptics read from its symbols.
func newRumbleEngine(haptics objc.ID) (*rumbleEngine, error) {
	ch, err := dl.OpenFramework("CoreHaptics")
	if err != nil {
		return nil, err
	}
	e := &rumbleEngine{}
	for ptr, name := range map[*objc.ID]string{
		&e.intensityID: "CHHapticEventParameterIDHapticIntensity", &e.sharpnessID: "CHHapticEventParameterIDHapticSharpness",
		&e.continuous: "CHHapticEventTypeHapticContinuous",
	} {
		addr, err := ch.Symbol(name)
		if err != nil {
			return nil, err
		}
		*ptr = *(*objc.ID)(dl.Pointer(addr))
	}
	loc := objcx.NewString("GCHapticsLocalityDefault")
	defer objcx.Release(loc)
	if addr, err := dl.OpenFramework("GameController"); err == nil {
		if sym, err := addr.Symbol("GCHapticsLocalityDefault"); err == nil {
			e.engine = haptics.Send(sel("createEngineWithLocality:"), *(*objc.ID)(dl.Pointer(sym)))
		}
	}
	if e.engine == 0 {
		return nil, errors.New(platform.TagGamepadRumbleUnavailable, "the controller's haptics made no engine")
	}
	objcx.Retain(e.engine)
	return e, nil
}

// play starts a continuous haptic event for the duration, replacing the one that plays.
func (e *rumbleEngine) play(intensity, sharpness float32, d time.Duration) error {
	var failure string
	objcx.WithPool(func() {
		var nserr objc.ID
		if !e.started {
			if !send[bool](e.engine, "startAndReturnError:", unsafePtr(&nserr)) {
				failure = reasonOf(nserr)
				return
			}
			e.started = true
		}
		param := func(id objc.ID, v float32) objc.ID {
			return class("CHHapticEventParameter").Send(sel("alloc")).Send(sel("initWithParameterID:value:"), id, v)
		}
		pi, ps := param(e.intensityID, intensity), param(e.sharpnessID, sharpness)
		params := objcx.NewArray(pi, ps)
		event := class("CHHapticEvent").Send(sel("alloc")).Send(sel("initWithEventType:parameters:relativeTime:duration:"),
			e.continuous, params, float64(0), d.Seconds())
		events := objcx.NewArray(event)
		pattern := class("CHHapticPattern").Send(sel("alloc")).Send(sel("initWithEvents:parameters:error:"), events, objc.ID(0), unsafePtr(&nserr))
		for _, o := range []objc.ID{pi, ps, params, event, events} {
			objcx.Release(o)
		}
		if pattern == 0 {
			failure = reasonOf(nserr)
			return
		}
		defer objcx.Release(pattern)
		player := send[objc.ID](e.engine, "createPlayerWithPattern:error:", pattern, unsafePtr(&nserr))
		if player == 0 {
			failure = reasonOf(nserr)
			return
		}
		objcx.Retain(player)
		e.stopPlayer()
		e.player = player
		if !send[bool](player, "startAtTime:error:", float64(0), unsafePtr(&nserr)) {
			failure = reasonOf(nserr)
		}
	})
	if failure != "" {
		return errors.New(tagGamepadRumbleFailed, "CoreHaptics refused the rumble", "reason", failure)
	}
	return nil
}

// stopPlayer stops and releases the player that plays.
func (e *rumbleEngine) stopPlayer() {
	if e.player == 0 {
		return
	}
	var nserr objc.ID
	send[bool](e.player, "stopAtTime:error:", float64(0), unsafePtr(&nserr))
	objcx.Release(e.player)
	e.player = 0
}

// stop ends the rumble that plays.
func (e *rumbleEngine) stop() { objcx.WithPool(e.stopPlayer) }

// close ends the rumble and releases the engine.
func (e *rumbleEngine) close() {
	objcx.WithPool(func() {
		e.stopPlayer()
		if e.started {
			e.engine.Send(sel("stopWithCompletionHandler:"), objc.ID(0))
			e.started = false
		}
		objcx.Release(e.engine)
		e.engine = 0
	})
}

// unsafePtr is the address of an object pointer that a call fills (an NSError**).
func unsafePtr(p *objc.ID) unsafe.Pointer { return unsafe.Pointer(p) }
