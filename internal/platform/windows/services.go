package windows

import (
	"context"
	"log/slog"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the desktop services of Windows (SPEC §25.9): Desktop is the value
// that platform.DesktopProvider returns. It implements mobile.Services (open a URI, show and close a
// notification, as Android's Desktop does) and the optional interfaces platform.TrayHost and
// platform.FileDialogHost, which package desktop finds by type assertion. The calls to Windows are
// behind shellAPI, implemented by services_windows.go, so that the tray, the balloon queue, the menus,
// the file dialogs' options and the validation of URIs are tested on any host.

// The IDs of the notify icons of the app: the tray icon and the hidden icon that owns the balloons
// while the app has no tray (SPEC §25.9).
const (
	trayIconID    = 1
	balloonIconID = 2
)

// point is a point of the screen in physical pixels.
type point struct{ X, Y int32 }

// iconData is what the code sends to the shell about a notify icon, in plain Go: services_windows.go
// turns it into a NOTIFYICONDATAW.
type iconData struct {
	ID    uint32
	Flags uint32
	// Icon is the HICON; the shell copies it, so the caller may free it after the call.
	Icon uintptr
	Tip  string
	// State and StateMask are dwState and dwStateMask.
	State, StateMask uint32
	// Info and InfoTitle are the balloon's text and title, InfoFlags its dwInfoFlags and BalloonIcon its
	// large icon.
	Info, InfoTitle string
	InfoFlags       uint32
	BalloonIcon     uintptr
}

// shellAPI is what the services ask of Windows. services_windows.go implements it with the Win32
// calls; the tests use a fake.
type shellAPI interface {
	// Post runs fn on the UI thread; After arms a timer there and returns the function that cancels it.
	Post(fn func())
	After(d time.Duration, fn func()) (stop func())
	// Worker runs fn on a goroutine of its own, on a locked OS thread with COM initialized. ThreadID is
	// the ID of the calling thread and CancelDialog posts the Cancel button's message to the windows of
	// the thread with that ID (it works from any thread).
	Worker(fn func())
	ThreadID() uint32
	CancelDialog(thread uint32)

	// Notify sends the command (win32.NIMAdd, NIMModify, NIMDelete or NIMSetVersion) for the icon. It
	// reports a shell that refuses.
	Notify(command uint32, d iconData) error
	// MakeIcon makes the HICON of the picture of the icon nearest to a small icon (large: the size of a
	// large icon). The zero icon gives the icon of the program, which is not owned and not freed.
	MakeIcon(icon ui.WindowIcon, large bool) (h uintptr, owned bool, err error)
	// FreeIcon frees an icon that MakeIcon made.
	FreeIcon(h uintptr)
	// ShowMenu shows the popup menu at a point of the screen and waits for the user. It returns the ID of
	// the item chosen, 0 for none. It runs a modal loop.
	ShowMenu(plan *menuPlan, at point) (uint32, error)

	// ShellExecute opens the URI with the program that Windows has for it (ShellExecuteExW); it runs on a
	// worker.
	ShellExecute(uri string) error
	// OpenItemDialog makes the common item dialog for opening or saving; it runs on a worker.
	OpenItemDialog(save bool) (itemDialog, error)
}

// Desktop is the desktop services of Windows (SPEC §25.9): opening a URI with ShellExecuteExW, the
// notifications as tray balloons, the system tray and the common item dialogs. ShowNotification,
// CloseNotification, OpenURI and ShowFileDialog may be called from any goroutine but the UI thread's;
// NewTray and the events of the shell belong to the UI thread.
//
// Flutter: no counterpart (Flutter apps use the url_launcher, flutter_local_notifications, tray_manager
// and file_selector plugins)
type Desktop struct {
	shell shellAPI
	log   *slog.Logger

	// The state below belongs to the UI thread.
	tray     *trayIcon
	balloons balloonQueue
	closed   bool

	// taskbarCreated is the number of the registered message of that name (0 when it could not be made).
	taskbarCreated uint32
}

var (
	_ mobile.Services         = (*Desktop)(nil)
	_ platform.TrayHost       = (*Desktop)(nil)
	_ platform.FileDialogHost = (*Desktop)(nil)
)

// newDesktop makes the services over the shell.
func newDesktop(shell shellAPI, log *slog.Logger, taskbarCreated uint32) *Desktop {
	if log == nil {
		log = slog.Default()
	}
	d := &Desktop{shell: shell, log: log, taskbarCreated: taskbarCreated}
	d.balloons.d = d
	return d
}

// limitText cuts s to at most max UTF-16 units, the way the fixed buffers of NOTIFYICONDATAW hold it
// (szTip, szInfoTitle and szInfo without their NUL): a text that is longer ends with an ellipsis, and a
// surrogate pair is never cut in the middle. A NUL is dropped, as the buffer is a C string.
func limitText(s string, max int) string {
	s = strings.ReplaceAll(s, "\x00", "")
	if len(utf16.Encode([]rune(s))) <= max {
		return s
	}
	const ellipsis = "…" // one unit
	units := 0
	var b strings.Builder
	for _, r := range s {
		n := 1
		if r >= 0x10000 {
			n = 2
		}
		if units+n > max-1 {
			break
		}
		units += n
		b.WriteRune(r)
	}
	return b.String() + ellipsis
}

// onUI runs fn on the UI thread and waits for its result. ok is false when ctx ended first; fn runs
// anyway when the loop gets to it, and its result is dropped.
func onUI[T any](ctx context.Context, d *Desktop, fn func() (T, error)) (v T, err error, ok bool) {
	type result struct {
		v   T
		err error
	}
	ch := make(chan result, 1)
	d.shell.Post(func() {
		v, err := fn()
		ch <- result{v, err}
	})
	select {
	case r := <-ch:
		return r.v, r.err, true
	case <-ctx.Done():
		return v, nil, false
	}
}

// OpenURI opens the URI with ShellExecuteExW (verb open) on a worker thread (SPEC §25.9). Windows'
// own validation comes first, because ShellExecute also starts programs: see checkURI. A URI that has
// no handler, or that ShellExecuteEx refuses, is OpenURIUnavailable with the reason.
//
// Flutter: no counterpart (url_launcher_windows: launchUrl)
func (d *Desktop) OpenURI(ctx context.Context, uri string) error {
	if err := checkURI(uri); err != nil {
		return err
	}
	done := make(chan error, 1)
	d.shell.Worker(func() { done <- d.shell.ShellExecute(uri) })
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return errors.NewErr(tagOpenURIUnavailable, ctx.Err()).SetMetadata("uri", uri, "reason", "the context ended before Windows answered")
	}
}

// executableExtensions are the types of file whose file: URI OpenURI refuses: ShellExecute would run
// them (SPEC §25.9).
var executableExtensions = map[string]bool{
	".exe": true, ".com": true, ".bat": true, ".cmd": true, ".msi": true, ".msp": true, ".scr": true,
	".lnk": true, ".pif": true, ".ps1": true, ".psm1": true, ".vbs": true, ".vbe": true, ".js": true,
	".jse": true, ".wsf": true, ".wsh": true, ".hta": true, ".jar": true, ".reg": true, ".cpl": true,
	".msc": true, ".gadget": true, ".application": true, ".appref-ms": true, ".url": true, ".inf": true,
}

// checkURI is the validation of Windows (SPEC §25.9), after Linux's: an empty URI or one with a control
// character (NUL, CR and LF as on Linux, and then any other after the scheme is checked) is
// OpenURIInvalid; a scheme of one letter (a drive path such as C:\x), javascript:, vbscript:
// and a file: URI of an executable type are OpenURISchemeBlocked. Anything else goes to ShellExecuteEx.
func checkURI(uri string) error {
	if uri == "" || strings.ContainsAny(uri, "\x00\r\n") {
		return errors.New(tagOpenURIInvalid, "a URI is not empty and has no control character", "uri", uri)
	}
	scheme := mobile.URIScheme(uri)
	switch {
	case scheme == "":
		return errors.New(tagOpenURIInvalid, "a URI needs a scheme, such as https: or file:", "uri", uri)
	case len(scheme) == 1:
		return errors.New(tagOpenURISchemeBlocked, "a URI with a one-letter scheme is a path of a drive, which ShellExecute would run", "uri", uri, "scheme", scheme)
	case scheme == "javascript" || scheme == "vbscript":
		return errors.New(tagOpenURISchemeBlocked, "a URI that runs script is not opened", "uri", uri, "scheme", scheme)
	case scheme == "file":
		name := fileURIName(uri)
		if executableExtensions[strings.ToLower(path.Ext(name))] {
			return errors.New(tagOpenURISchemeBlocked, "a file: URI of a program or a script is not opened", "uri", uri, "scheme", scheme, "name", name)
		}
	}
	if strings.IndexFunc(uri, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return errors.New(tagOpenURIInvalid, "a URI cannot hold a control character", "uri", uri)
	}
	return nil
}

// fileURIName is the file name of a file: URI, decoded, with the trailing dots and spaces that Windows
// ignores removed (a.exe. is a.exe).
func fileURIName(raw string) string {
	raw = strings.TrimLeft(raw, "\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0b\x0c\x0e\x0f\x10 ")
	rest := raw[strings.IndexByte(raw, ':')+1:]
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		rest = rest[:i]
	}
	if u, err := url.PathUnescape(rest); err == nil {
		rest = u
	}
	rest = strings.ReplaceAll(rest, "\\", "/")
	rest = strings.TrimRight(rest, ". ")
	return path.Base(rest)
}

// ShowNotification shows the notification as a tray balloon, or queues it behind the one that is shown
// (SPEC §25.9). It returns the notification's ID; ReplacesID of a notification that is shown or queued
// replaces it. An app that cannot show a balloon is NotificationsUnavailable and a shell that refuses the
// balloon WindowsBalloonFailed.
//
// Flutter: no counterpart (flutter_local_notifications_windows)
func (d *Desktop) ShowNotification(ctx context.Context, spec mobile.NotificationSpec) (uint32, error) {
	id, err, ok := onUI(ctx, d, func() (uint32, error) { return d.balloons.show(spec) })
	if !ok {
		return 0, errors.NewErr(tagNotificationsUnavailable, ctx.Err()).SetMetadata("reason", "the context ended before the UI thread showed the balloon")
	}
	return id, err
}

// CloseNotification closes the notification with the ID, which calls its OnClose with CloseByApp.
// Closing one that is gone is not an error.
//
// Flutter: no counterpart
func (d *Desktop) CloseNotification(ctx context.Context, id uint32) error {
	_, _, ok := onUI(ctx, d, func() (struct{}, error) { d.balloons.close(id); return struct{}{}, nil })
	if !ok {
		return errors.NewErr(tagNotificationsUnavailable, ctx.Err()).SetMetadata("reason", "the context ended before the UI thread closed the balloon")
	}
	return nil
}

// Close deletes the notify icons and stops the timers; the services are not used afterwards. It runs on
// the UI thread.
//
// Flutter: no counterpart
func (d *Desktop) Close() {
	if d.closed {
		return
	}
	d.closed = true
	d.balloons.shutdown()
	if d.tray != nil {
		d.tray.Close()
	}
}

// onIconMessage handles the callback message of a notify icon: the platform's service window got
// msgTray with these parameters. It belongs to the UI thread.
func (d *Desktop) onIconMessage(wparam, lparam uintptr) {
	if d.closed {
		return
	}
	id, event, at := decodeIconMessage(wparam, lparam)
	d.onIconEvent(id, event, at)
}

// decodeIconMessage reads the parameters of the callback message. With NOTIFYICON_VERSION_4 the low
// word of lParam is the event, the high word the ID of the icon (never 0), and wParam holds the point
// (the anchor) as two signed 16-bit numbers; without it wParam is the ID and the low word of lParam the
// event, with a high word of 0. The two forms are told apart by that high word, so each icon may be on
// either version.
func decodeIconMessage(wparam, lparam uintptr) (id, event uint32, at point) {
	if hi := uint32(lparam>>16) & 0xFFFF; hi != 0 {
		return hi, uint32(lparam) & 0xFFFF,
			point{X: int32(int16(wparam & 0xFFFF)), Y: int32(int16((wparam >> 16) & 0xFFFF))}
	}
	return uint32(wparam), uint32(lparam) & 0xFFFF, point{}
}

// onIconEvent routes the event of a notify icon: the events of balloons go to the queue, whichever icon
// carries them, and the others to the tray.
func (d *Desktop) onIconEvent(id, event uint32, at point) {
	switch event {
	case win32.NINBalloonShow, win32.NINBalloonHide, win32.NINBalloonTimeout, win32.NINBalloonUserClick:
		d.balloons.event(event)
	default:
		if id == trayIconID && d.tray != nil {
			d.tray.event(event, at)
		}
	}
}

// isTaskbarCreated says whether the message is the registered TaskbarCreated.
func (d *Desktop) isTaskbarCreated(msg uint32) bool {
	return d.taskbarCreated != 0 && msg == d.taskbarCreated
}

// onTaskbarCreated adds the notify icons again after Explorer restarted: the new task bar has none of them.
func (d *Desktop) onTaskbarCreated() {
	if d.closed {
		return
	}
	if d.tray != nil {
		d.tray.readd()
	}
	d.balloons.readd()
}

// portalHandlePrefix starts the handle that names a window to the file dialogs (SPEC §25.9).
const portalHandlePrefix = "win32:"

// portalHandleOf is the handle of a window, "win32:" and its HWND in hexadecimal.
func portalHandleOf(hwnd uintptr) string {
	const digits = "0123456789abcdef"
	if hwnd == 0 {
		return portalHandlePrefix + "0"
	}
	var b [16]byte
	i := len(b)
	for hwnd > 0 {
		i--
		b[i] = digits[hwnd&0xF]
		hwnd >>= 4
	}
	return portalHandlePrefix + string(b[i:])
}

// hwndOfHandle reads the handle that portalHandleOf made; ok is false for any other text.
func hwndOfHandle(handle string) (uintptr, bool) {
	hex, found := strings.CutPrefix(handle, portalHandlePrefix)
	if !found || hex == "" || len(hex) > 16 {
		return 0, false
	}
	var v uintptr
	for i := 0; i < len(hex); i++ {
		c := hex[i]
		switch {
		case c >= '0' && c <= '9':
			c -= '0'
		case c >= 'a' && c <= 'f':
			c = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			c = c - 'A' + 10
		default:
			return 0, false
		}
		v = v<<4 | uintptr(c)
	}
	return v, true
}

// The tags of the errors that this file makes (rm4n0s/errors, R5). They are the tags of package desktop,
// which passes them to the app untouched.
const (
	tagNotificationsUnavailable = "NotificationsUnavailable"
	tagOpenURIInvalid           = "OpenURIInvalid"
	tagOpenURISchemeBlocked     = "OpenURISchemeBlocked"
	tagOpenURIUnavailable       = "OpenURIUnavailable"
)
