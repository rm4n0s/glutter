package windows

import (
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/ole"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of drag and drop (SPEC §25.10, §8.11): what a drag over a window means
// to the framework (the target), and what a drag that the app starts offers and how it ends (the
// source). The COM objects and the calls to Windows are package ole and drag_windows.go; the formats
// of the clipboard are converted with the functions of clipboard.go.

// dropReadBudget is how long the reading of a drop may take in all, as on the other platforms (SPEC
// §8.11): the formats are read one after the other on the UI thread, and a format that would start
// after the time is over is skipped. A call to a source that does not answer cannot be interrupted;
// only the formats after it are saved.
const dropReadBudget = 5 * time.Second

// dropWindow is the drop target of one window: it implements ole.TargetHandler and turns what OLE
// says into the ui.DropEvents of the framework, and the framework's answer into the effect.
type dropWindow struct {
	view ui.ViewID
	// handler is the handler of the platform at this moment (nil outside Run).
	handler func() platform.Handler
	// toClient turns a screen place into the place in the client area, in physical pixels.
	toClient func(pt win32.Point) (ui.Offset, bool)
	// nameOf is the name of a registered clipboard format ("" for the others).
	nameOf func(format uint32) string
	pid    int
	now    func() time.Time
	budget time.Duration
	log    *slog.Logger

	// The drag that is over the window: entered says the framework heard of it, types and local
	// describe the data on offer.
	entered bool
	types   []string
	local   uint64
}

var _ ole.TargetHandler = (*dropWindow)(nil)

// describe lists the formats of the data as the types the framework sees, once each in the order of the
// source, and the ID of the drag request of this process that the data carries, if any (the format
// "application/x-glutter-drag-<pid>-<id>").
func (d *dropWindow) describe(data ole.DataSource) (types []string, local uint64, formats []uint32) {
	formats, err := data.Formats()
	if err != nil {
		d.log.Warn("the formats of a drag could not be listed", "err", err)
	}
	for _, f := range formats {
		name := d.nameOf(f)
		if id := ole.LocalIDOf(name, d.pid); id != 0 {
			local = id
		}
		if ole.IsMarker(name) {
			continue
		}
		if mime := ole.MIMEOf(f, name); mime != "" && !slices.Contains(types, mime) {
			types = append(types, mime)
		}
	}
	return types, local, formats
}

// event makes the event for the framework with the state of the drag.
func (d *dropWindow) event(phase ui.DropPhase, key uint32, pt win32.Point, effect uint32) ui.DropEvent {
	pos, _ := d.toClient(pt)
	return ui.DropEvent{
		ViewID: d.view, Phase: phase, Position: pos, Types: d.types,
		Actions: ole.OfferedActions(key, ole.ActionsOf(effect)), LocalID: d.local,
	}
}

// deliver gives the event to the handler and returns the effect for the action it accepted.
func (d *dropWindow) deliver(ev ui.DropEvent, allowed uint32) uint32 {
	h := d.handler()
	if h == nil {
		return win32.DropEffectNone
	}
	return ole.EffectFor(h.OnDrop(ev), allowed)
}

// DragEnter is the start of a drag over the window: the formats on offer are listed and the framework
// is told, with the place.
func (d *dropWindow) DragEnter(data ole.DataSource, key uint32, pt win32.Point, effect uint32) uint32 {
	d.types, d.local, _ = d.describe(data)
	d.entered = true
	return d.deliver(d.event(ui.DropPhaseEnter, key, pt, effect), effect)
}

// DragOver is the drag moving, or the keys changing.
func (d *dropWindow) DragOver(key uint32, pt win32.Point, effect uint32) uint32 {
	if !d.entered {
		return win32.DropEffectNone
	}
	return d.deliver(d.event(ui.DropPhaseMove, key, pt, effect), effect)
}

// DragLeave is the drag leaving the window.
func (d *dropWindow) DragLeave() {
	if !d.entered {
		return
	}
	d.entered = false
	if h := d.handler(); h != nil {
		h.OnDrop(ui.DropEvent{ViewID: d.view, Phase: ui.DropPhaseLeave, Types: d.types, LocalID: d.local})
	}
	d.types, d.local = nil, 0
}

// Drop is the drop: the data is read now (the text, the files, the URLs), and the framework hears of the
// drop with it. The effect is the action the framework accepted at that place; none when it refused.
func (d *dropWindow) Drop(data ole.DataSource, key uint32, pt win32.Point, effect uint32) uint32 {
	var formats []uint32
	d.types, d.local, formats = d.describe(data)
	d.entered = false
	text, uris := d.read(data, formats)
	ev := d.event(ui.DropPhaseDrop, key, pt, effect)
	ev.Text, ev.URIs = text, uris
	out := d.deliver(ev, effect)
	d.types, d.local = nil, 0
	return out
}

// The formats of a drop that become text, in the order of preference.
var dropTextFormats = []uint32{win32.CFUnicodeText, win32.CFText, win32.CFOEMText}

// read reads the text and the URIs of a drop. The text is the first of CF_UNICODETEXT, CF_TEXT and
// CF_OEMTEXT that the source has; the URIs are the files of CF_HDROP as file URIs, then the URL of the
// URL formats (UniformResourceLocatorW, else UniformResourceLocator) when it is not among them; a drop
// that has a URL and no text gets the URL as its text. A format that fails to read is left out and
// logged, and so is every format after the time budget.
func (d *dropWindow) read(data ole.DataSource, formats []uint32) (text string, uris []string) {
	start := d.now()
	overBudget := func() bool {
		if d.now().Sub(start) <= d.budget {
			return false
		}
		d.log.Warn("the source of a drop was too slow; the rest of its data is not read", "budget", d.budget.String())
		return true
	}
	read := func(format uint32) ([]byte, bool) {
		if overBudget() {
			return nil, false
		}
		b, err := data.Read(format)
		if err != nil {
			d.log.Warn("a format of a drop could not be read", "format", int(format), "err", err)
			return nil, false
		}
		return b, true
	}
	for _, f := range dropTextFormats {
		if !slices.Contains(formats, f) {
			continue
		}
		if b, ok := read(f); ok {
			if f == win32.CFUnicodeText {
				text = textFromClipboard(b)
			} else {
				text = ansiText(b)
			}
			break
		}
	}
	if slices.Contains(formats, win32.CFHDrop) {
		if b, ok := read(win32.CFHDrop); ok {
			paths, err := pathsFromDropFiles(b)
			if err != nil {
				d.log.Warn("the files of a drop could not be read", "err", err)
			}
			for _, p := range paths {
				uris = append(uris, uriFromPath(p))
			}
		}
	}
	var url string
	for _, wide := range []bool{true, false} {
		name := win32.FormatNameURL
		if wide {
			name = win32.FormatNameURLW
		}
		f := d.formatNamed(formats, name)
		if f == 0 {
			continue
		}
		if b, ok := read(f); ok {
			if url = ole.StringFromData(b, wide); url != "" {
				break
			}
		}
	}
	if url != "" {
		if !slices.Contains(uris, url) {
			uris = append(uris, url)
		}
		if text == "" {
			text = url
		}
	}
	return text, uris
}

// formatNamed is the registered format of the list with the name, 0 for none.
func (d *dropWindow) formatNamed(formats []uint32, name string) uint32 {
	for _, f := range formats {
		if d.nameOf(f) == name {
			return f
		}
	}
	return 0
}

// ansiText reads the bytes of CF_TEXT or CF_OEMTEXT as text: up to the first NUL, one character per byte
// (Latin-1: the code page is not known to the plain-Go code), with the line breaks as \n.
func ansiText(b []byte) string {
	if i := strings.IndexByte(string(b), 0); i >= 0 {
		b = b[:i]
	}
	rs := make([]rune, len(b))
	for i, c := range b {
		rs[i] = rune(c)
	}
	s := string(rs)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// ---- the drags the app starts ----

// buildDragData makes the data a drag offers (SPEC §25.10): the text as CF_UNICODETEXT and CF_TEXT, the
// file URIs as CF_HDROP, the other URIs as the two URL formats (the first one only: they hold one URL),
// and the format that marks the drag as this app's. register gives the number of a registered format.
// A URI that is not a file URI is not a path, and a path is not a URL: each goes where it fits.
func buildDragData(req ui.DragRequest, pid int, register func(name string) (uint32, error)) (*ole.DataSet, error) {
	set := ole.NewDataSet()
	if req.Text != "" {
		set.Add(win32.CFUnicodeText, textBytes(textToClipboard(req.Text)))
		set.Add(win32.CFText, ole.StringData(strings.ReplaceAll(req.Text, "\n", "\r\n"), false))
	}
	var paths []string
	var link string
	for _, uri := range req.URIs {
		if p, ok := pathFromURI(uri); ok {
			paths = append(paths, p)
		} else if link == "" {
			link = uri
		}
	}
	if len(paths) > 0 {
		set.Add(win32.CFHDrop, dropFilesFromPaths(paths))
	}
	if link != "" {
		for _, wide := range []bool{true, false} {
			name := win32.FormatNameURL
			if wide {
				name = win32.FormatNameURLW
			}
			f, err := register(name)
			if err != nil {
				return nil, err
			}
			set.Add(f, ole.StringData(link, wide))
		}
	}
	marker, err := register(ole.LocalMarker(pid, req.ID))
	if err != nil {
		return nil, err
	}
	set.Add(marker, []byte{0})
	return set, nil
}

// dragSession is a drag the app started: it is posted when the framework asks, and runs the modal
// loop of OLE in the turn after.
type dragSession struct {
	view    ui.ViewID
	request ui.DragRequest
	data    *ole.DataSet
	machine *ole.SourceMachine
	// done says the session ended before its loop ran (the window closed).
	done bool
}

// dragRunner starts the drags of the platform and reports how they end. Its calls to Windows are
// functions, so that the rules are tested without it.
type dragRunner struct {
	pid int
	log *slog.Logger
	// buttons are the MK_* bits of the pointer buttons that are down now.
	buttons func() uint32
	// register gives the number of a registered clipboard format.
	register func(name string) (uint32, error)
	// post runs a function on the UI thread, after the handler that asked has returned.
	post func(func())
	// loop runs the modal loop of the drag (DoDragDrop with the objects made from the data and the
	// machine) and reports the effect the target chose and whether the loop ended with a drop.
	loop func(view ui.ViewID, data *ole.DataSet, machine *ole.SourceMachine, allowed uint32) (effect uint32, dropped bool, err error)
	// released lifts the pointer: the button went up inside OLE's loop and the window never heard.
	released func(view ui.ViewID)
	// end tells the handler how the drag ended.
	end func(ui.DragResult)

	session *dragSession
}

// Start begins a drag from the window of the view: it fails with DragInProgress when a drag is going on
// and with DragNoButton when no pointer button is down. The loop runs in a posted function, so the
// handler that called Start has returned (SPEC §25.10).
func (r *dragRunner) Start(view ui.ViewID, req ui.DragRequest) error {
	if r.session != nil {
		return errors.New(tagDragInProgress, "a drag of the app is already going on")
	}
	buttons := r.buttons()
	if buttons&(win32.MKLButton|win32.MKRButton|win32.MKMButton) == 0 {
		return errors.New(tagDragNoButton, "a drag starts with a pointer button that is down")
	}
	data, err := buildDragData(req, r.pid, r.register)
	if err != nil {
		return err
	}
	s := &dragSession{view: view, request: req, data: data, machine: &ole.SourceMachine{Buttons: buttons}}
	r.session = s
	r.post(func() { r.run(s) })
	return nil
}

// run is the posted function: it runs the loop and reports the end.
func (r *dragRunner) run(s *dragSession) {
	var action ui.DropActions
	if !s.done && r.buttons()&s.machine.Buttons != 0 {
		effect, dropped, err := r.loop(s.view, s.data, s.machine, ole.EffectOf(allowedActions(s.request.Actions)))
		switch {
		case err != nil:
			r.log.Error("the drag could not run", "err", err)
		case dropped:
			action = ole.ActionsOf(effect)
		}
	}
	if r.session == s {
		r.session = nil
	}
	r.released(s.view)
	r.end(ui.DragResult{ID: s.request.ID, Action: action})
}

// WindowClosed cancels the drag that started from the window that closes.
func (r *dragRunner) WindowClosed(view ui.ViewID) {
	if s := r.session; s != nil && s.view == view {
		s.done = true
		s.machine.Cancelled = true
	}
}

// Cancel cancels the drag, if there is one (the platform closes).
func (r *dragRunner) Cancel() {
	if s := r.session; s != nil {
		s.done = true
		s.machine.Cancelled = true
	}
}

// Active says whether a drag the app started is going on.
func (r *dragRunner) Active() bool { return r.session != nil }

// allowedActions are the actions a request allows: none means copy (ui.DragRequest).
func allowedActions(a ui.DropActions) ui.DropActions {
	if a == ui.DropActionNone {
		return ui.DropActionCopy
	}
	return a
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDragInProgress = "DragInProgress"
	tagDragNoButton   = "DragNoButton"
)
