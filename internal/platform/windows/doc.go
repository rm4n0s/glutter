// Package windows is the Win32 platform backend (SPEC §25): the windows, the message loop, the
// frame pacing, the displays and the process start of Windows (step 28.2), behind internal/platform;
// the pointer input (step 28.3), the keyboard (step 28.4), the text input and the input methods
// (step 28.5, with package ime); the clipboard and the settings come with step 28.6. The calls to
// Windows go through package win32 (SPEC §6.3 rule 10).
//
// The files of the backend that call Windows are named *_windows.go. The files with no OS tag hold
// the plain-Go parts, which are tested on any host: the timer heap, the frame requests, the pacer's
// state machine, the styles and sizes of the windows, the console decision, the reader of
// GLUTTER_DEV_EVENTS, the translators of the mouse, the wheel, touch and pen, and the keyboard
// (the key tables of keymap_gen.go, the converter and the message sessions); the model of the input
// methods is package ime.
//
// Flutter: the Windows embedder (engine/src/flutter/shell/platform/windows).
package windows

//go:generate go run ../../../cmd/glutter-gen keys -reference ../../../reference/flutter -out "" -windows-out keymap_gen.go
