package windows

import (
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// notifyCall is one call of Notify.
type notifyCall struct {
	Command uint32
	Data    iconData
}

type fakeTimer struct {
	at      time.Time
	fn      func()
	stopped bool
}

// fakeShell is shellAPI over memory: it records the calls, runs what is posted at once, and keeps a
// clock that a test moves.
type fakeShell struct {
	mu sync.Mutex

	notify    []notifyCall
	notifyErr func(c notifyCall) error
	menuCmd   uint32
	menuErr   error
	menus     []*menuPlan
	menuAt    []point

	now    time.Time
	timers []*fakeTimer

	nextIcon   uintptr
	madeIcons  []uintptr
	freedIcons []uintptr
	iconErr    error

	executed    []string
	executeErr  error
	dialog      *fakeDialog
	dialogErr   error
	dialogSaved []bool
	cancelled   []uint32
	posted      []func()
	// deferPost keeps the posted functions and never runs them: the UI thread does not answer.
	deferPost bool
}

func newFakeShell() *fakeShell { return &fakeShell{now: time.Unix(1000, 0), nextIcon: 100} }

func (f *fakeShell) Post(fn func()) {
	f.mu.Lock()
	if f.deferPost {
		f.posted = append(f.posted, fn)
		f.mu.Unlock()
		return
	}
	f.mu.Unlock()
	fn()
}

func (f *fakeShell) After(d time.Duration, fn func()) func() {
	f.mu.Lock()
	defer f.mu.Unlock()
	t := &fakeTimer{at: f.now.Add(d), fn: fn}
	f.timers = append(f.timers, t)
	return func() {
		f.mu.Lock()
		t.stopped = true
		f.mu.Unlock()
	}
}

// advance moves the clock and runs the timers that are due, the earliest first.
func (f *fakeShell) advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	f.mu.Unlock()
	for {
		f.mu.Lock()
		var due *fakeTimer
		for _, t := range f.timers {
			if !t.stopped && !t.at.After(f.now) && (due == nil || t.at.Before(due.at)) {
				due = t
			}
		}
		if due != nil {
			due.stopped = true
		}
		f.mu.Unlock()
		if due == nil {
			return
		}
		due.fn()
	}
}

// armed is the number of timers that have not run or been stopped.
func (f *fakeShell) armed() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, t := range f.timers {
		if !t.stopped {
			n++
		}
	}
	return n
}

func (f *fakeShell) Worker(fn func() func()) {
	go func() {
		if deliver := fn(); deliver != nil {
			deliver()
		}
	}()
}

func (f *fakeShell) ThreadID() uint32 { return 77 }

func (f *fakeShell) CancelDialog(thread uint32) {
	f.mu.Lock()
	f.cancelled = append(f.cancelled, thread)
	f.mu.Unlock()
	if f.dialog != nil {
		f.dialog.closeNow()
	}
}

func (f *fakeShell) Notify(command uint32, d iconData) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := notifyCall{command, d}
	f.notify = append(f.notify, c)
	if f.notifyErr != nil {
		return f.notifyErr(c)
	}
	return nil
}

// commands are the commands of the Notify calls so far, as text ("add#1", "modify#2", ...).
func (f *fakeShell) commands() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	names := map[uint32]string{win32.NIMAdd: "add", win32.NIMModify: "modify", win32.NIMDelete: "delete", win32.NIMSetVersion: "version"}
	var out []string
	for _, c := range f.notify {
		out = append(out, fmt.Sprintf("%s#%d", names[c.Command], c.Data.ID))
	}
	return out
}

func (f *fakeShell) lastNotify() notifyCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.notify[len(f.notify)-1]
}

func (f *fakeShell) MakeIcon(icon ui.WindowIcon, large bool) (uintptr, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.iconErr != nil {
		return 0, false, f.iconErr
	}
	if len(icon.Images) == 0 {
		return 1, false, nil // the icon of the program
	}
	f.nextIcon++
	f.madeIcons = append(f.madeIcons, f.nextIcon)
	return f.nextIcon, true, nil
}

func (f *fakeShell) FreeIcon(h uintptr) {
	f.mu.Lock()
	f.freedIcons = append(f.freedIcons, h)
	f.mu.Unlock()
}

func (f *fakeShell) ShowMenu(plan *menuPlan, at point) (uint32, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.menus = append(f.menus, plan)
	f.menuAt = append(f.menuAt, at)
	return f.menuCmd, f.menuErr
}

func (f *fakeShell) ShellExecute(uri string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.executed = append(f.executed, uri)
	return f.executeErr
}

func (f *fakeShell) OpenItemDialog(save bool) (itemDialog, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dialogSaved = append(f.dialogSaved, save)
	if f.dialogErr != nil {
		return nil, f.dialogErr
	}
	return f.dialog, nil
}

// fakeDialog is itemDialog over memory.
type fakeDialog struct {
	mu       sync.Mutex
	options  uint32
	setFlags uint32
	types    []fileType
	index    uint32
	title    string
	label    string
	folder   string
	fileName string
	ext      string
	owner    uintptr
	showErr  error
	paths    []string
	pathsErr error
	chosen   uint32
	released bool
	// failOn names a method that fails.
	failOn string
	// block makes Show wait until closeNow.
	block   bool
	closed  chan struct{}
	started chan struct{}
}

func newFakeDialog() *fakeDialog {
	return &fakeDialog{options: 0x00000008 | 0x00000400 /* FOS_NOCHANGEDIR... a start value */, closed: make(chan struct{}), started: make(chan struct{}, 1)}
}

func (d *fakeDialog) fail(name string) error {
	if d.failOn == name {
		return errors.New(win32.TagCOMCallFailed, "a COM method failed", "method", name)
	}
	return nil
}

func (d *fakeDialog) Options() (uint32, error) { return d.options, d.fail("Options") }
func (d *fakeDialog) SetOptions(flags uint32) error {
	d.setFlags = flags
	return d.fail("SetOptions")
}
func (d *fakeDialog) SetFileTypes(t []fileType) error { d.types = t; return d.fail("SetFileTypes") }
func (d *fakeDialog) SetFileTypeIndex(i uint32) error { d.index = i; return d.fail("SetFileTypeIndex") }
func (d *fakeDialog) SetTitle(s string) error         { d.title = s; return d.fail("SetTitle") }
func (d *fakeDialog) SetOkButtonLabel(s string) error { d.label = s; return d.fail("SetOkButtonLabel") }
func (d *fakeDialog) SetFolder(s string) error        { d.folder = s; return d.fail("SetFolder") }
func (d *fakeDialog) SetFileName(s string) error      { d.fileName = s; return d.fail("SetFileName") }
func (d *fakeDialog) SetDefaultExtension(s string) error {
	d.ext = s
	return d.fail("SetDefaultExtension")
}

func (d *fakeDialog) Show(owner uintptr) error {
	d.owner = owner
	if d.block {
		d.started <- struct{}{}
		<-d.closed
		return errors.New(win32.TagWin32Cancelled, "the user cancelled")
	}
	return d.showErr
}

func (d *fakeDialog) closeNow() {
	d.mu.Lock()
	defer d.mu.Unlock()
	select {
	case <-d.closed:
	default:
		close(d.closed)
	}
}

func (d *fakeDialog) Paths() ([]string, error) { return d.paths, d.pathsErr }
func (d *fakeDialog) FileTypeIndex() (uint32, error) {
	return d.chosen, d.fail("FileTypeIndex")
}
func (d *fakeDialog) Release() { d.released = true }

// quietLog is a logger that writes nothing.
func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// newTestDesktop makes the services over a fake shell.
func newTestDesktop() (*Desktop, *fakeShell) {
	sh := newFakeShell()
	return newDesktop(sh, quietLog(), 0xC123), sh
}

// mobileSpec is a notification with the title and a body.
func mobileSpec(title string) mobile.NotificationSpec {
	return mobile.NotificationSpec{Title: title, Body: "body of " + title}
}
