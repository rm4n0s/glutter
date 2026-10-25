// Package windows is the Win32 platform backend (SPEC §25): the windows, the message loop, the
// frame pacing, the displays and the process start of Windows (step 28.2), behind internal/platform;
// the pointer input (step 28.3), the keyboard (step 28.4), the text input and the input methods
// (step 28.5, with package ime); the clipboard, the system settings, the list of fonts and the
// lifecycle (step 28.6), and the tray, the notifications, the file dialogs and the opening of URIs
// (step 28.11). The calls to
// Windows go through package win32 (SPEC §6.3 rule 10).
//
// The files of the backend that call Windows are named *_windows.go. The files with no OS tag hold
// the plain-Go parts, which are tested on any host: the timer heap, the frame requests, the pacer's
// state machine, the styles and sizes of the windows, the console decision, the reader of
// GLUTTER_DEV_EVENTS, the translators of the mouse, the wheel, touch and pen, and the keyboard
// (the key tables of keymap_gen.go, the converter and the message sessions), the conversions and the
// retry of the clipboard (clipboard.go), the settings (settings.go), the lifecycle table
// (lifecycle.go), the list of font files (fonts.go) and the desktop services (step 28.11: services.go
// has Desktop with the checks of URIs, tray.go the tray's model, diff and menu plan, balloon.go the
// queue of the notifications and filedialog.go the options and answers of the file dialogs, all over
// the calls that services_windows.go makes); the model of the input methods is package ime and the
// matching of fonts is internal/text/winfonts.
//
// Flutter: the Windows embedder (engine/src/flutter/shell/platform/windows).
package windows

//go:generate go run ../../../cmd/glutter-gen keys -reference ../../../reference/flutter -out "" -windows-out keymap_gen.go
