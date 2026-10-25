//go:build windows

package windows

import (
	"sync"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// The window classes of the platform (SPEC §25.3, §25.4): GlutterWindow for the windows of the
// app and GlutterService for the hidden window that receives Post's wake-up and the messages that
// Windows sends to top-level windows only. They share one window procedure, and that is one
// callback for the whole process (the runtime makes about 2,000 and never frees one).

const (
	windowClassName  = "GlutterWindow"
	serviceClassName = "GlutterService"
)

// messageTarget handles the messages of one window.
type messageTarget interface {
	handleMessage(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr
}

// targets finds the object of a window by the token that CreateWindowEx was given. The table is
// process-wide because the window procedure is: it is the one place where Windows calls back into
// Go with no argument but the window handle.
var targets = struct {
	mu   sync.Mutex
	next uintptr
	m    map[uintptr]messageTarget
}{m: map[uintptr]messageTarget{}}

// registerTarget gives the target a token, to pass as the last argument of CreateWindowEx.
func registerTarget(t messageTarget) uintptr {
	targets.mu.Lock()
	defer targets.mu.Unlock()
	targets.next++
	targets.m[targets.next] = t
	return targets.next
}

// unregisterTarget forgets the token; later messages of the window get the default handling.
func unregisterTarget(token uintptr) {
	targets.mu.Lock()
	delete(targets.m, token)
	targets.mu.Unlock()
}

func lookupTarget(token uintptr) messageTarget {
	targets.mu.Lock()
	defer targets.mu.Unlock()
	return targets.m[token]
}

// wndProcCallback is the callback of every window of the process, made on first use.
var wndProcCallback = sync.OnceValue(func() uintptr { return win32.NewCallback(wndProc) })

// wndProc is the window procedure. The messages that arrive while CreateWindowEx runs
// (WM_NCCREATE first) carry the token in the CreateStruct; it is kept in the window's user data,
// where the later messages find it, as Flutter's WndProc does.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::WndProc
func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	m := uint32(msg)
	var token uintptr
	if m == win32.WMNCCreate {
		token = win32.View[win32.CreateStruct](lparam).CreateParams
		win32.SetWindowLongPtr(hwnd, win32.GWLPUserData, token)
	} else {
		token = win32.GetWindowLongPtr(hwnd, win32.GWLPUserData)
	}
	if t := lookupTarget(token); t != nil {
		return t.handleMessage(hwnd, m, wparam, lparam)
	}
	return win32.DefWindowProc(hwnd, m, wparam, lparam)
}

// classes are the classes the platform registered.
type classes struct {
	instance uintptr
	names    []string
}

// register registers a class once. The window classes are removed in unregister.
func (c *classes) register(name string, icon, cursor uintptr) error {
	wc := win32.WndClassEx{
		WndProc: wndProcCallback(), Instance: c.instance, Icon: icon, Cursor: cursor, IconSm: icon,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	p, err := win32.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	wc.ClassName = p
	if _, err := win32.RegisterClassEx(&wc); err != nil {
		return err
	}
	c.names = append(c.names, name)
	return nil
}

// unregister removes the classes. A class with windows left cannot be removed; that is not worth
// reporting at the end of the process.
func (c *classes) unregister() {
	for _, name := range c.names {
		_ = win32.UnregisterClass(name, c.instance) // the process ends soon
	}
	c.names = nil
}
