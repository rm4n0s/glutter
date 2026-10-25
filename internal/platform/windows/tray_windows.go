//go:build windows

package windows

import (
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// Notify sends the command for the notify icon with Shell_NotifyIconW. The service window is the owner
// of every icon and msgTray is the callback message (SPEC §25.9); NIM_SETVERSION asks for
// NOTIFYICON_VERSION_4. The texts are cut to their buffers (the code has cut them with an ellipsis
// before). It fails with Win32CallFailed when the shell refuses.
//
// Flutter: no counterpart
func (s winShell) Notify(command uint32, d iconData) error {
	n := win32.NotifyIconData{
		Wnd: s.p.service.Load(), ID: d.ID, Flags: d.Flags, CallbackMessage: msgTray, Icon: d.Icon,
		State: d.State, StateMask: d.StateMask, InfoFlags: d.InfoFlags, BalloonIcon: d.BalloonIcon,
	}
	n.Size = uint32(unsafe.Sizeof(n))
	if command == win32.NIMSetVersion {
		n.Version = win32.NotifyIconVersion4
	}
	for _, f := range []struct {
		dst  []uint16
		text string
	}{{n.Tip[:], d.Tip}, {n.Info[:], d.Info}, {n.InfoTitle[:], d.InfoTitle}} {
		if _, err := win32.CopyUTF16(f.dst, f.text); err != nil {
			return err
		}
	}
	return win32.ShellNotifyIcon(command, &n)
}

// MakeIcon makes the HICON for a notify icon from the picture nearest to the size of a small icon (a
// large one for the balloon) at the DPI of the primary monitor. An icon with no pictures is the icon of
// the program, which is not owned.
//
// Flutter: no counterpart
func (s winShell) MakeIcon(icon ui.WindowIcon, large bool) (uintptr, bool, error) {
	if err := icon.Validate(); err != nil {
		return 0, false, err
	}
	if len(icon.Images) == 0 {
		return s.p.icon, false, nil
	}
	metric := win32.SMCxSmIcon
	if large {
		metric = win32.SMCxIcon
	}
	dpi := win32.GetDpiForMonitor(win32.MonitorFromPoint(win32.Point{}, win32.MonitorDefaultToPrimary))
	h, err := makeIcon(nearestImage(icon.Images, int(win32.GetSystemMetricsForDpi(metric, dpi))))
	if err != nil {
		return 0, false, err
	}
	return h, true, nil
}

// FreeIcon frees an icon that MakeIcon made.
//
// Flutter: no counterpart
func (s winShell) FreeIcon(h uintptr) { win32.DestroyIcon(h) }
