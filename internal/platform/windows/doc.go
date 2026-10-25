// Package windows is the Win32 platform backend (SPEC §25): the windows, the message loop, the
// input, the text input, the clipboard and the settings of Windows, behind internal/platform. It
// has no code yet: step 28.1 adds the groundwork in package win32 (the DLL loading, the COM calls,
// the structures and the checks of their layout), and steps 28.2 to 28.6 add the platform itself.
//
// Every file of the backend carries the windows build tag (or is named *_windows.go) and calls
// Windows with the standard library's syscall through package win32 (SPEC §6.3 rule 10). This
// untagged file keeps the package listed on every OS.
//
// Flutter: the Windows embedder (engine/src/flutter/shell/platform/windows).
package windows
