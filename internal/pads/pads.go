// Package pads is the manager of the gamepads of an app (SPEC §28.2): it takes the pads and the raw reports that a platform
// backend gives (internal/platform.Gamepads), finds each pad's mapping in the table (internal/gamepaddb), keeps the state of the
// buttons and axes of the standard layout and tells the listeners, and turns the buttons into key events. The engine owns one
// manager for the app, so a pad moves the focus of every app, whether it imports package gamepad or not; package gamepad is
// the public face of it.
//
// A manager belongs to the UI thread, like the rest of the framework: the platform hands its reports over with Post.
//
// Flutter: no counterpart
package pads

import (
	"log/slog"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

// Host is implemented by the host of the dispatcher (the engine) and found through ui.PlatformDispatcher.HostAs: it gives the
// manager of the app, or nil when the platform has no gamepads.
//
// Flutter: no counterpart
type Host interface {
	PadManager() *Manager
}

// Key is a key event that a pad makes (SPEC §28.2, "Pads as keys").
//
// Flutter: no counterpart
type Key struct {
	Time time.Duration
	Down bool
	// Physical and Logical are the IDs of the key (services.PhysicalKeyboardKey and services.LogicalKeyboardKey).
	Physical, Logical uint64
	// DirectionalPad says that the key is an arrow of the D-pad, and not a game button.
	DirectionalPad bool
}

// Opts are the options of a manager.
//
// Flutter: no counterpart
type Opts struct {
	// Keys receives the key events of the pads; nil turns key events off for good.
	Keys func(Key)
	// Logger receives the notes of the manager; nil logs nothing.
	Logger *slog.Logger
	// Getenv is passed to the table of mappings for the hint: fields; nil means os.Getenv.
	Getenv func(string) string
	// SDKVersion is the version of Android, for the sdk>= fields of the table; 0 elsewhere.
	SDKVersion int
}

// Manager keeps the pads of an app.
//
// Flutter: no counterpart
type Manager struct {
	plat platform.Gamepads
	opts Opts
	log  *slog.Logger

	table   *gamepaddb.Table
	started bool
	// pads are the connected pads that have a mapping, in the order they were connected.
	pads []*Pad
	byID map[platform.PadID]*Pad
	// unmapped are the pads that no mapping is known for, kept so that a mapping the app adds later can take them in.
	unmapped map[platform.PadID]platform.PadInfo
	order    []platform.PadID
	keys     bool

	nextListener int
	listeners    []setListener
}

type setListener struct {
	id int
	fn func()
}

// New makes the manager of the gamepad service plat. It does nothing until Start.
//
// Flutter: no counterpart
func New(plat platform.Gamepads, opts Opts) *Manager {
	m := &Manager{
		plat:     plat,
		opts:     opts,
		log:      opts.Logger,
		byID:     map[platform.PadID]*Pad{},
		unmapped: map[platform.PadID]platform.PadInfo{},
		keys:     opts.Keys != nil,
	}
	if m.log == nil {
		m.log = slog.New(slog.DiscardHandler)
	}
	return m
}

// Start begins to look for pads. Starting a started manager does nothing.
//
// Flutter: no counterpart
func (m *Manager) Start() error {
	if m.started || m.plat == nil {
		return nil
	}
	if err := m.plat.Start(m); err != nil {
		return err
	}
	m.started = true
	return nil
}

// Stop ends the search and lets go of every pad.
//
// Flutter: no counterpart
func (m *Manager) Stop() {
	if !m.started {
		return
	}
	m.started = false
	m.plat.Stop()
	for _, p := range m.pads {
		p.remove(time.Duration(0))
	}
	m.pads, m.order = nil, nil
	m.byID = map[platform.PadID]*Pad{}
	m.unmapped = map[platform.PadID]platform.PadInfo{}
	m.notify()
}

// Available says whether the platform has a gamepad service at all.
//
// Flutter: no counterpart
func (m *Manager) Available() bool { return m != nil && m.plat != nil }

// Pads returns the connected pads that have a mapping, in the order they were connected. The slice is a copy.
//
// Flutter: no counterpart
func (m *Manager) Pads() []*Pad {
	if m == nil {
		return nil
	}
	return append([]*Pad(nil), m.pads...)
}

// Count is the number of connected pads that have a mapping, and At the pad that came i-th: a loop over them allocates nothing.
//
// Flutter: no counterpart
func (m *Manager) Count() int {
	if m == nil {
		return 0
	}
	return len(m.pads)
}

// At returns the i-th connected pad that has a mapping (0 to Count()-1).
//
// Flutter: no counterpart
func (m *Manager) At(i int) *Pad { return m.pads[i] }

// SetKeyEvents turns the key events of the pads on or off. They are on at first, when the manager was given a function for them.
//
// Flutter: no counterpart
func (m *Manager) SetKeyEvents(on bool) { m.keys = on && m.opts.Keys != nil }

// KeyEvents says whether the pads make key events.
//
// Flutter: no counterpart
func (m *Manager) KeyEvents() bool { return m != nil && m.keys }

// AddListener adds a function that is called when a pad is connected or removed, and returns the function that removes it.
//
// Flutter: no counterpart
func (m *Manager) AddListener(fn func()) (remove func()) {
	m.nextListener++
	id := m.nextListener
	m.listeners = append(m.listeners, setListener{id, fn})
	return func() {
		for i, l := range m.listeners {
			if l.id == id {
				m.listeners = append(m.listeners[:i:i], m.listeners[i+1:]...)
				return
			}
		}
	}
}

func (m *Manager) notify() {
	for _, l := range append([]setListener(nil), m.listeners...) {
		l.fn()
	}
}

// Table returns the table of mappings, made on first use.
//
// Flutter: no counterpart
func (m *Manager) Table() *gamepaddb.Table {
	if m.table != nil {
		return m.table
	}
	var sections [][]string
	switch m.platformName() {
	case "Linux":
		sections = [][]string{gamepaddb.Linux}
	case "Windows":
		sections = [][]string{gamepaddb.XInput, gamepaddb.WGI, gamepaddb.DInput}
	case "Android":
		sections = [][]string{gamepaddb.Android}
	case "Web":
		sections = [][]string{gamepaddb.Web}
	}
	t := gamepaddb.NewTable(sections...)
	t.Getenv = m.opts.Getenv
	t.SDKVersion = m.opts.SDKVersion
	m.table = t
	return t
}

func (m *Manager) platformName() string {
	if m.plat == nil {
		return ""
	}
	return m.plat.MappingPlatform()
}

// AddMappings adds mappings in the community's format (one line each: "GUID,name,a:b0,b:b1,..."; lines with a platform: field
// for another platform are left out) and gives the pads that had none the chance to take one. It returns the number of
// mappings that are new. An error tagged GamepadMappingInvalid says the first line that could not be read; the lines before it
// are added.
//
// Flutter: no counterpart
func (m *Manager) AddMappings(text string) (int, error) {
	n, err := m.Table().AddText(text, m.platformName())
	// A mapping may have replaced the one of a pad, or given a pad that had none its mapping.
	m.remap()
	return n, err
}

// AddMapping adds one mapping line, like AddMappings, whatever its platform: field says.
//
// Flutter: no counterpart
func (m *Manager) AddMapping(line string) (int, error) {
	n, err := m.Table().Add(line)
	m.remap()
	return n, err
}

// remap gives the pads whose mapping changed their new bindings, and takes in the pads that had none.
func (m *Manager) remap() {
	changed := false
	for _, id := range append([]platform.PadID(nil), m.order...) {
		info, ok := m.unmapped[id]
		if !ok {
			continue
		}
		if p := m.makePad(info); p != nil {
			delete(m.unmapped, id)
			m.insert(p)
			changed = true
		}
	}
	for _, p := range m.pads {
		if p.info.Standard {
			continue
		}
		if mp := m.mappingFor(p.info); mp != nil && (mp != p.mapping || mp.Text != p.mappingText) {
			p.setMapping(mp)
		}
	}
	if changed {
		m.notify()
	}
}

// insert puts a pad in the list in the order the platform connected them.
func (m *Manager) insert(p *Pad) {
	m.byID[p.info.ID] = p
	m.pads = append(m.pads, p)
	// Keep the order of connection even when a pad took a mapping late.
	rank := map[platform.PadID]int{}
	for i, id := range m.order {
		rank[id] = i
	}
	for i := len(m.pads) - 1; i > 0 && rank[m.pads[i].info.ID] < rank[m.pads[i-1].info.ID]; i-- {
		m.pads[i], m.pads[i-1] = m.pads[i-1], m.pads[i]
	}
}

// mappingFor finds the mapping of a pad: the table's, else the one the backend made from what the pad reports, else the
// default one of the table.
func (m *Manager) mappingFor(info platform.PadInfo) *gamepaddb.Mapping {
	guid := gamepaddb.GUID(info.GUID)
	t := m.Table()
	mp := t.Find(guid)
	if mp == nil && info.Mapping != "" {
		mp = t.SetAuto(guid, info.Name, info.Mapping)
	}
	if mp == nil {
		mp = t.Default()
	}
	return mp
}

// PadAdded implements platform.PadHandler.
func (m *Manager) PadAdded(info platform.PadInfo) {
	if _, dup := m.byID[info.ID]; dup {
		return
	}
	m.order = append(m.order, info.ID)
	p := m.makePad(info)
	if p == nil {
		m.unmapped[info.ID] = info
		m.log.Debug("a pad has no mapping, so it is not a gamepad until the app adds one", "name", info.Name, "guid", gamepaddb.GUID(info.GUID).String())
		return
	}
	m.insert(p)
	m.log.Info("gamepad connected", "name", p.name, "kind", p.kind.String(), "path", info.Path)
	m.notify()
}

// makePad makes the pad of the info, or returns nil when it has no mapping.
func (m *Manager) makePad(info platform.PadInfo) *Pad {
	p := &Pad{m: m, info: info, connected: true}
	p.extras.init(info)
	guid := gamepaddb.GUID(info.GUID)
	if info.Standard {
		p.name = info.Name
		p.kind = gamepaddb.KindOfGUID(guid, info.Name)
		p.face = gamepaddb.FaceStyleOf(p.kind)
		if info.ButtonMask != 0 {
			p.hasButton = info.ButtonMask
		} else {
			for b := gamepaddb.Button(0); b < gamepaddb.ButtonCount; b++ {
				p.hasButton |= 1 << b
			}
		}
		p.hasAxis = 1<<gamepaddb.AxisCount - 1
		return p
	}
	mp := m.mappingFor(info)
	if mp == nil {
		return nil
	}
	if !p.setMapping(mp) {
		return nil
	}
	return p
}

// PadRemoved implements platform.PadHandler.
func (m *Manager) PadRemoved(id platform.PadID) {
	for i, o := range m.order {
		if o == id {
			m.order = append(m.order[:i:i], m.order[i+1:]...)
			break
		}
	}
	if _, ok := m.unmapped[id]; ok {
		delete(m.unmapped, id)
		return
	}
	p, ok := m.byID[id]
	if !ok {
		return
	}
	delete(m.byID, id)
	for i, o := range m.pads {
		if o == p {
			m.pads = append(m.pads[:i:i], m.pads[i+1:]...)
			break
		}
	}
	p.remove(0)
	m.log.Info("gamepad removed", "name", p.name)
	m.notify()
}

var _ platform.PadExtrasHandler = (*Manager)(nil)

// PadTouch implements platform.PadExtrasHandler.
func (m *Manager) PadTouch(t platform.PadTouch) {
	if p := m.byID[t.Pad]; p != nil && p.connected {
		p.touch(t)
	}
}

// PadSensor implements platform.PadExtrasHandler.
func (m *Manager) PadSensor(s platform.PadSensor) {
	if p := m.byID[s.Pad]; p != nil && p.connected {
		p.sensor(s)
	}
}

// PadBattery implements platform.PadExtrasHandler.
func (m *Manager) PadBattery(b platform.PadBattery) {
	if p := m.byID[b.Pad]; p != nil && p.connected {
		p.setPower(b)
	}
}

// PadEvents implements platform.PadHandler.
func (m *Manager) PadEvents(events []platform.PadEvent) {
	for _, e := range events {
		p := m.byID[e.Pad]
		if p == nil || !p.connected {
			continue
		}
		p.report(e)
	}
}
