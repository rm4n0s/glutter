//go:build windows

package windows

import (
	"runtime"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// Worker runs fn on a goroutine of its own, locked to an OS thread whose COM apartment is a
// single-threaded one (CoInitializeEx), for the calls that wait for the user: the file dialogs and
// ShellExecuteExW (SPEC §6.3 rule 12, §25.9). The thread gives the apartment up and is released when fn
// returns. The results are given back through Post by the caller.
//
// Flutter: no counterpart
func (s winShell) Worker(fn func()) {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		if err := win32.CoInitializeEx(win32.COInitApartmentThreaded); err != nil {
			s.p.log.Warn("COM could not be initialized on a worker thread; the call goes on without it", "err", err)
		} else {
			defer win32.CoUninitialize()
		}
		fn()
	}()
}

// ThreadID is the ID of the calling thread.
//
// Flutter: no counterpart
func (s winShell) ThreadID() uint32 { return win32.GetCurrentThreadID() }

// CancelDialog posts the message of the Cancel button (WM_COMMAND with IDCANCEL) to the top-level
// windows of the thread, which closes the dialog that is open there.
//
// Flutter: no counterpart
func (s winShell) CancelDialog(thread uint32) {
	win32.EnumThreadWindows(thread, func(hwnd uintptr) bool {
		if err := win32.PostMessage(hwnd, win32.WMCommand, win32.IDCancel, 0); err != nil {
			s.p.log.Debug("the Cancel message could not be posted to a dialog", "err", err)
		}
		return true
	})
}
