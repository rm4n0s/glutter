//go:build windows

package windows

import (
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// SetIcon sets the icon of the window (SPEC §25.4): the pictures nearest to the sizes of the big
// and the small icon at the window's DPI become icons with CreateIconIndirect and WM_SETICON. The
// zero icon gives the window the icon of its class again, which is the icon resource of the .exe
// (id 1), else the system's. It fails with WindowIconInvalid when a picture is wrong.
//
// Flutter: shell/platform/windows/host_window.cc (the class icon, idi_app_icon)
func (w *window) SetIcon(icon ui.WindowIcon) error {
	if err := icon.Validate(); err != nil {
		return err
	}
	if w.closed {
		return nil
	}
	w.releaseIcons()
	if len(icon.Images) == 0 {
		win32.SendIconMessage(w.hwnd, win32.IconBig, 0)
		win32.SendIconMessage(w.hwnd, win32.IconSmall, 0)
		return nil
	}
	big := nearestImage(icon.Images, int(win32.GetSystemMetricsForDpi(win32.SMCxIcon, w.dpi)))
	small := nearestImage(icon.Images, int(win32.GetSystemMetricsForDpi(win32.SMCxSmIcon, w.dpi)))
	for i, img := range []ui.WindowIconImage{big, small} {
		h, err := makeIcon(img)
		if err != nil {
			return err
		}
		w.icons[i] = h
		which := uintptr(win32.IconBig)
		if i == 1 {
			which = win32.IconSmall
		}
		win32.SendIconMessage(w.hwnd, which, h)
	}
	return nil
}

// releaseIcons frees the icons that SetIcon made.
func (w *window) releaseIcons() {
	for i, h := range w.icons {
		if h != 0 {
			win32.DestroyIcon(h)
			w.icons[i] = 0
		}
	}
}

// nearestImage is the picture whose size is closest to the wanted one, preferring a larger one on
// a tie so that the icon is scaled down.
func nearestImage(images []ui.WindowIconImage, want int) ui.WindowIconImage {
	best := images[0]
	for _, img := range images[1:] {
		d, bd := abs(img.Size-want), abs(best.Size-want)
		if d < bd || (d == bd && img.Size > best.Size) {
			best = img
		}
	}
	return best
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// makeIcon makes an icon from a picture of straight (not premultiplied) RGBA: a 32-bit
// top-down DIB for the colors and an empty mask, as the alpha channel decides.
func makeIcon(img ui.WindowIconImage) (uintptr, error) {
	info := win32.BitmapInfo{Header: win32.BitmapInfoHeader{
		Size: uint32(unsafe.Sizeof(win32.BitmapInfoHeader{})), Width: int32(img.Size), Height: -int32(img.Size),
		Planes: 1, BitCount: 32, Compression: win32.BIRGB,
	}}
	color, bits, err := win32.CreateDIBSection(&info)
	if err != nil {
		return 0, err
	}
	defer win32.DeleteObject(color)
	dst := win32.Bytes(bits, img.Size*img.Size*4)
	for i := 0; i+3 < len(img.RGBA); i += 4 {
		dst[i], dst[i+1], dst[i+2], dst[i+3] = img.RGBA[i+2], img.RGBA[i+1], img.RGBA[i], img.RGBA[i+3]
	}
	mask, err := win32.CreateBitmap(int32(img.Size), int32(img.Size), 1, 1, make([]byte, (img.Size+15)/16*2*img.Size))
	if err != nil {
		return 0, err
	}
	defer win32.DeleteObject(mask)
	ii := win32.IconInfo{Icon: 1, Mask: mask, Color: color}
	return win32.CreateIconIndirect(&ii)
}
