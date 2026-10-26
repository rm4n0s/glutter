//go:build windows

package windows

import (
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// buildMenu makes the popup menu of the plan: strings, separators, disabled and checked items, radio
// marks and submenus, in the order of the plan. DestroyMenu frees it with its submenus.
func buildMenu(entries []menuEntry) (uintptr, error) {
	menu, err := win32.CreatePopupMenu()
	if err != nil {
		return 0, err
	}
	for i, e := range entries {
		info := win32.MenuItemInfo{Mask: win32.MIIMFType | win32.MIIMID, Type: win32.MFTString, ID: e.ID}
		var text *uint16
		if e.Separator {
			info.Type = win32.MFTSeparator
		} else {
			if text, err = win32.UTF16PtrFromString(e.Label); err != nil {
				win32.DestroyMenu(menu)
				return 0, err
			}
			info.Mask |= win32.MIIMString | win32.MIIMState
			info.TypeData = unsafe.Pointer(text)
			if e.Disabled {
				info.State |= win32.MFSGrayed
			}
			if e.Checked {
				info.State |= win32.MFSChecked
			}
			if e.Radio {
				info.Type |= win32.MFTRadioCheck
			}
			if len(e.Children) > 0 {
				sub, err := buildMenu(e.Children)
				if err != nil {
					win32.DestroyMenu(menu)
					return 0, err
				}
				info.Mask |= win32.MIIMSubMenu
				info.SubMenu = sub
			}
		}
		err := win32.InsertMenuItem(menu, uint32(i), &info)
		if err != nil {
			win32.DestroyMenu(menu) // frees the submenus that were inserted too
			if info.SubMenu != 0 {
				win32.DestroyMenu(info.SubMenu) // this one was not inserted
			}
			return 0, err
		}
	}
	return menu, nil
}

// ShowMenu shows the popup menu of the plan at a point of the screen with TrackPopupMenuEx and returns
// the number of the item that the user chose, 0 for none (SPEC §25.9). The service window is the owner:
// it is made the foreground window first and gets a WM_NULL afterwards, which is what makes Windows close
// the menu when the user clicks elsewhere. The menu is a modal loop of Windows: the timer keeps the
// platform's turns going (SPEC §25.3).
//
// Flutter: no counterpart
func (s winShell) ShowMenu(plan *menuPlan, at point) (uint32, error) {
	menu, err := buildMenu(plan.Entries)
	if err != nil {
		return 0, err
	}
	defer win32.DestroyMenu(menu)
	svc := s.p.service.Load()
	if at == (point{}) {
		if pt, ok := win32.GetCursorPos(); ok {
			at = point{X: pt.X, Y: pt.Y}
		}
	}
	win32.SetForegroundWindow(svc)
	if err := win32.SetTimer(svc, modalTimerID, modalTimerInterval); err == nil {
		s.p.modalTimers++
		defer func() {
			win32.KillTimer(svc, modalTimerID)
			s.p.modalTimers--
		}()
	} else {
		s.p.log.Warn("the timer of the menu's modal loop could not be started", "err", err)
	}
	cmd := win32.TrackPopupMenuEx(menu, win32.TPMReturnCmd|win32.TPMNoNotify|win32.TPMRightButton, at.X, at.Y, svc)
	if err := win32.PostMessage(svc, win32.WMNull, 0, 0); err != nil {
		s.p.log.Debug("the WM_NULL that closes the menu could not be posted", "err", err)
	}
	return cmd, nil
}
