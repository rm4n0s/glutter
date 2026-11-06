package fake

import (
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// Gamepads is the scripted gamepad service of the fake platform: a test connects and removes pads and sends their reports, and
// the handler gets them on the platform's UI thread, in order, as a real backend's would (SPEC §28.8).
//
// Flutter: no counterpart
type Gamepads struct {
	post func(func())

	mu      sync.Mutex
	handler platform.PadHandler
	pads    map[platform.PadID]platform.PadInfo
	nextID  platform.PadID
	rumbles []Rumble
	started int
}

// Rumble is a rumble or light request that the fake service recorded.
//
// Flutter: no counterpart
type Rumble struct {
	Pad platform.PadID
	// Triggers says that the request was for the motors of the triggers.
	Triggers bool
	// LED says that the request was a colour, which Color holds (red, green, blue).
	LED       bool
	Color     [3]uint8
	Low, High uint16
	Duration  time.Duration
}

// NewGamepads makes a scripted gamepad service whose handler is called through post, which runs a function on the UI thread.
//
// Flutter: no counterpart
func NewGamepads(post func(func())) *Gamepads {
	return &Gamepads{post: post, pads: map[platform.PadID]platform.PadInfo{}, nextID: 1}
}

// Gamepads returns the scripted gamepad service, which Services gives the engine.
//
// Flutter: no counterpart
func (p *Platform) Gamepads() *Gamepads {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pads == nil {
		p.pads = NewGamepads(p.Post)
	}
	return p.pads
}

// MappingPlatform implements platform.Gamepads: the fake platform is Linux, whose evdev numbering the tests use.
func (g *Gamepads) MappingPlatform() string { return "Linux" }

// Start implements platform.Gamepads: the pads that were connected before are reported to h.
func (g *Gamepads) Start(h platform.PadHandler) error {
	g.mu.Lock()
	g.handler = h
	g.started++
	infos := g.sortedInfos()
	g.mu.Unlock()
	g.post(func() {
		for _, info := range infos {
			h.PadAdded(info)
		}
	})
	return nil
}

// Stop implements platform.Gamepads.
func (g *Gamepads) Stop() {
	g.mu.Lock()
	g.handler = nil
	g.mu.Unlock()
}

// Starts returns how many times the service was started.
//
// Flutter: no counterpart
func (g *Gamepads) Starts() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.started
}

func (g *Gamepads) sortedInfos() []platform.PadInfo {
	var infos []platform.PadInfo
	for id := platform.PadID(1); id < g.nextID; id++ {
		if info, ok := g.pads[id]; ok {
			infos = append(infos, info)
		}
	}
	return infos
}

// Connect plugs in a pad: it gets the next ID, which is returned and set in the info that the handler receives.
//
// Flutter: no counterpart
func (g *Gamepads) Connect(info platform.PadInfo) platform.PadID {
	g.mu.Lock()
	info.ID = g.nextID
	g.nextID++
	g.pads[info.ID] = info
	h := g.handler
	g.mu.Unlock()
	if h != nil {
		g.post(func() { h.PadAdded(info) })
	}
	return info.ID
}

// Disconnect unplugs the pad.
//
// Flutter: no counterpart
func (g *Gamepads) Disconnect(id platform.PadID) {
	g.mu.Lock()
	_, ok := g.pads[id]
	delete(g.pads, id)
	h := g.handler
	g.mu.Unlock()
	if ok && h != nil {
		g.post(func() { h.PadRemoved(id) })
	}
}

// Send makes the reports arrive, in one call of the handler.
//
// Flutter: no counterpart
func (g *Gamepads) Send(events ...platform.PadEvent) {
	g.mu.Lock()
	h := g.handler
	g.mu.Unlock()
	if h == nil {
		return
	}
	events = append([]platform.PadEvent(nil), events...)
	g.post(func() { h.PadEvents(events) })
}

// Rumbles returns the requests of the app to rumble the pads and set their lights.
//
// Flutter: no counterpart
func (g *Gamepads) Rumbles() []Rumble {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]Rumble(nil), g.rumbles...)
}

// Rumble implements platform.Gamepads.
func (g *Gamepads) Rumble(id platform.PadID, low, high uint16, d time.Duration) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	info, ok := g.pads[id]
	if !ok || !info.Rumble {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the fake pad cannot rumble", "pad", int64(id))
	}
	g.rumbles = append(g.rumbles, Rumble{Pad: id, Low: low, High: high, Duration: d})
	return nil
}

// RumbleTriggers implements platform.Gamepads.
func (g *Gamepads) RumbleTriggers(id platform.PadID, left, right uint16, d time.Duration) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	info, ok := g.pads[id]
	if !ok || !info.RumbleTriggers {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the fake pad cannot rumble its triggers", "pad", int64(id))
	}
	g.rumbles = append(g.rumbles, Rumble{Pad: id, Triggers: true, Low: left, High: right, Duration: d})
	return nil
}

// SetLED implements platform.Gamepads.
func (g *Gamepads) SetLED(id platform.PadID, r, gr, b uint8) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	info, ok := g.pads[id]
	if !ok || !info.LED {
		return errors.New(platform.TagGamepadLEDUnavailable, "the fake pad has no light", "pad", int64(id))
	}
	g.rumbles = append(g.rumbles, Rumble{Pad: id, LED: true, Color: [3]uint8{r, gr, b}})
	return nil
}

var _ platform.Gamepads = (*Gamepads)(nil)
