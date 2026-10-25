//go:build windows

package windows

import (
	"runtime"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// ShellExecute opens the URI with ShellExecuteExW and the verb "open" (SPEC §25.9). It runs on a worker
// thread with COM initialized: the shell may use COM and DDE to start the program, and the call waits
// for that (SEE_MASK_NOASYNC) so that the thread does not end under it. No error dialog is shown. A URI
// that has no handler (SE_ERR_NOASSOC, SE_ERR_ASSOCINCOMPLETE) or that the shell refuses is
// OpenURIUnavailable with the reason.
//
// Flutter: no counterpart (url_launcher_windows)
func (s winShell) ShellExecute(uri string) error {
	verb, err := win32.UTF16PtrFromString("open")
	if err != nil {
		return errors.New(tagOpenURIUnavailable, "the verb could not be made", "uri", uri, "reason", err.Error())
	}
	file, err := win32.UTF16PtrFromString(uri)
	if err != nil {
		return errors.New(tagOpenURIInvalid, "a URI has no NUL in it", "uri", uri)
	}
	info := win32.ShellExecuteInfo{
		Mask: win32.SEEMaskNoAsync | win32.SEEMaskFlagNoUI, Show: win32.SWShowNormal,
		Verb: uintptr(unsafe.Pointer(verb)), File: uintptr(unsafe.Pointer(file)),
	}
	ok, errno, instApp, err := win32.ShellExecuteEx(&info)
	runtime.KeepAlive(verb)
	runtime.KeepAlive(file)
	if err != nil {
		return errors.New(tagOpenURIUnavailable, "ShellExecuteExW could not be called", "uri", uri, "reason", err.Error())
	}
	if ok {
		return nil
	}
	reason := "ShellExecuteExW failed"
	if instApp == win32.SEErrNoAssoc || instApp == win32.SEErrAssocIncomplete || errno == win32.ErrorNoAssociation {
		reason = "no program is associated with the URI"
	}
	return errors.New(tagOpenURIUnavailable, "Windows cannot open the URI", "uri", uri, "reason", reason, "code", int(errno), "hInstApp", int(instApp))
}
