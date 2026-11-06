// Package winpad is the gamepad service of Windows without the calls into Windows: it finds the pads that XInput serves and the
// HID pads that Raw Input lists, reads them, and reports them to the pad manager through platform.Gamepads (SPEC §28.2). What it
// needs from Windows is behind two interfaces, XInputAPI and HIDSource; the real ones and the window that hears of devices
// arriving are in package windows (gamepads_windows.go), and the tests drive the service with fakes on any host. The code of
// the HID pads is translated to Go and changed from src/joystick/windows/SDL_rawinputjoystick.c of SDL 3 and the XInput code from
// src/joystick/windows/SDL_xinputjoystick.c (zlib license, third_party/sdl/LICENSE).
//
// Flutter: no counterpart
package winpad
