// Package windows is the Win32 platform backend (SPEC §25): the windows, the message loop, the
// frame pacing, the displays and the process start of Windows (step 28.2), behind internal/platform;
// the input, the text input, the clipboard and the settings come with steps 28.3 to 28.6. The
// calls to Windows go through package win32 (SPEC §6.3 rule 10).
//
// The files of the backend that call Windows are named *_windows.go. The files with no OS tag hold
// the plain-Go parts, which are tested on any host: the timer heap, the frame requests, the pacer's
// state machine, the styles and sizes of the windows, the console decision and the reader of
// GLUTTER_DEV_EVENTS.
//
// Flutter: the Windows embedder (engine/src/flutter/shell/platform/windows).
package windows
