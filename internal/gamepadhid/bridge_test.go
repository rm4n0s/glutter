package gamepadhid

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// recPadHandler is a platform.PadHandler that takes the extras too.
type recPadHandler struct {
	added     []platform.PadInfo
	removed   []platform.PadID
	events    []platform.PadEvent
	touches   []platform.PadTouch
	sensors   []platform.PadSensor
	batteries []platform.PadBattery
}

func (h *recPadHandler) PadAdded(i platform.PadInfo)      { h.added = append(h.added, i) }
func (h *recPadHandler) PadRemoved(id platform.PadID)     { h.removed = append(h.removed, id) }
func (h *recPadHandler) PadEvents(e []platform.PadEvent)  { h.events = append(h.events, e...) }
func (h *recPadHandler) PadTouch(t platform.PadTouch)     { h.touches = append(h.touches, t) }
func (h *recPadHandler) PadSensor(s platform.PadSensor)   { h.sensors = append(h.sensors, s) }
func (h *recPadHandler) PadBattery(b platform.PadBattery) { h.batteries = append(h.batteries, b) }

func newBridge(h platform.PadHandler) (*Bridge, *[]string) {
	var notes []string
	b := &Bridge{
		ID: 7, Info: sonyPS4(BusUSB),
		Deliver:  func(fn func(platform.PadHandler)) { fn(h) },
		OnReady:  func(Caps) { notes = append(notes, "ready") },
		OnFailed: func(error) { notes = append(notes, "failed") },
		OnLost:   func() { notes = append(notes, "lost") },
	}
	return b, &notes
}

func TestBridgeAnnouncesAPadThatIsReady(t *testing.T) {
	var h recPadHandler
	b, notes := newBridge(&h)
	b.Info.Path = "/dev/hidraw3"
	caps := Caps{Name: "PS4 Controller", Kind: gamepaddb.KindPS4, Rumble: true, LED: true, Sensors: true, Rate: 250, Touchpads: 1, Buttons: ps4Buttons}
	b.Ready(caps)
	if len(h.added) != 1 {
		t.Fatalf("added = %+v", h.added)
	}
	i := h.added[0]
	if i.ID != 7 || i.Name != "PS4 Controller" || i.Path != "/dev/hidraw3" || !i.Standard || !i.Rumble || !i.LED || i.RumbleTriggers ||
		!i.Sensors || i.SensorRate != 250 || i.Touchpads != 1 || i.ButtonMask != ps4Buttons || i.Battery != -1 {
		t.Errorf("info = %+v", i)
	}
	if gamepaddb.GUID(i.GUID) != GUID(b.Info, caps) {
		t.Error("the GUID is not the one of the driver")
	}
	if got, ok := b.Caps(); !ok || got != caps {
		t.Errorf("Caps() = %+v, %v", got, ok)
	}
	if len(*notes) != 1 || (*notes)[0] != "ready" {
		t.Errorf("notes = %v", *notes)
	}
}

func TestBridgeNameComesFromTheDeviceWhenTheDriverGivesNone(t *testing.T) {
	var h recPadHandler
	b, _ := newBridge(&h)
	b.Info = DeviceInfo{Vendor: 0x0f0d, Product: 0x0001, Name: "Fighting Pad"}
	b.Ready(Caps{})
	if h.added[0].Name != "Fighting Pad" {
		t.Errorf("name = %q", h.added[0].Name)
	}
}

func TestBridgeTurnsSnapshotsIntoEvents(t *testing.T) {
	var h recPadHandler
	b, _ := newBridge(&h)
	b.Ready(Caps{})
	var s padstd.Snapshot
	s.Buttons[gamepaddb.South] = true
	s.Axes[gamepaddb.LeftX] = 100
	b.State(time.Second, &s)
	if len(h.events) != 2 || h.events[0].Pad != 7 || h.events[0].Kind != platform.PadButton || h.events[0].Index != int(gamepaddb.South) ||
		h.events[1].Kind != platform.PadAxis || h.events[1].Value != 100 || h.events[0].Time != time.Second {
		t.Errorf("events = %+v", h.events)
	}
	// The same state again changes nothing.
	b.State(2*time.Second, &s)
	if len(h.events) != 2 {
		t.Errorf("an unchanged state made events: %+v", h.events[2:])
	}
}

func TestBridgePassesTheExtrasToAHandlerThatTakesThem(t *testing.T) {
	var h recPadHandler
	b, _ := newBridge(&h)
	b.Ready(Caps{})
	b.Touch(platform.PadTouch{Finger: 1, Down: true})
	b.Sensor(platform.PadSensor{Kind: platform.PadAccelerometer})
	b.Battery(platform.PadBattery{Percent: 40})
	if len(h.touches) != 1 || h.touches[0].Pad != 7 || len(h.sensors) != 1 || h.sensors[0].Pad != 7 || len(h.batteries) != 1 || h.batteries[0].Pad != 7 {
		t.Errorf("extras: %+v %+v %+v", h.touches, h.sensors, h.batteries)
	}
}

func TestBridgeDropsTheExtrasForAHandlerThatTakesNone(t *testing.T) {
	var h onlyPadHandler
	b, _ := newBridge(&h)
	b.Ready(Caps{})
	// A handler that is not a platform.PadExtrasHandler gets nothing, and nothing breaks.
	b.Touch(platform.PadTouch{})
	b.Sensor(platform.PadSensor{})
	b.Battery(platform.PadBattery{})
	if len(h.added) != 1 {
		t.Error("the pad was not announced")
	}
}

// onlyPadHandler is a PadHandler with nothing else.
type onlyPadHandler struct{ added []platform.PadInfo }

func (h *onlyPadHandler) PadAdded(i platform.PadInfo)   { h.added = append(h.added, i) }
func (h *onlyPadHandler) PadRemoved(platform.PadID)     {}
func (h *onlyPadHandler) PadEvents([]platform.PadEvent) {}

func TestBridgeLostRemovesThePad(t *testing.T) {
	var h recPadHandler
	b, notes := newBridge(&h)
	b.Ready(Caps{})
	b.Lost()
	if len(h.removed) != 1 || h.removed[0] != 7 {
		t.Errorf("removed = %v", h.removed)
	}
	if got := *notes; len(got) != 2 || got[1] != "lost" {
		t.Errorf("notes = %v", got)
	}
}

func TestBridgeFailedTellsTheBackendAndAnnouncesNothing(t *testing.T) {
	var h recPadHandler
	b, notes := newBridge(&h)
	b.Failed(nil)
	if len(h.added) != 0 || len(*notes) != 1 || (*notes)[0] != "failed" {
		t.Errorf("added %v, notes %v", h.added, *notes)
	}
}

func TestBridgeRemovesThePadOnce(t *testing.T) {
	var h recPadHandler
	b, _ := newBridge(&h)
	b.Ready(Caps{})
	b.Remove()
	b.Lost()
	b.Remove()
	if len(h.removed) != 1 {
		t.Errorf("removed = %v, want once", h.removed)
	}
}

func TestBridgeDoesNotRemoveAPadThatWasNeverAnnounced(t *testing.T) {
	var h recPadHandler
	b, _ := newBridge(&h)
	b.Remove()
	if len(h.removed) != 0 {
		t.Errorf("removed = %v", h.removed)
	}
}
