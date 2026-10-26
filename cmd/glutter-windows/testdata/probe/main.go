//go:build windows

// Command probe is the Windows program that the Wine tests of glutter-windows run: it reports what
// the application manifest and the resources of its .exe did to the process (GOOS=windows go build
// ./cmd/glutter-windows/testdata/probe, with the manifest through `glutter-windows build`). It prints
// one "key: value" line for each fact:
//
//	dpi-start:     the DPI awareness of the thread when the program starts (PerMonitorV2, PerMonitor,
//	               System or Unaware): the manifest decides it
//	acp:           the active code page (65001 with the manifest's UTF-8 setting)
//	icon-<n>:      the size of the icon that LoadImageW(resource 1) gives for n pixels
//	version:       the ProductVersion of the version information, and its ProductName
//	dpi-platform:  with the argument "platform", the awareness after the Windows platform of Glutter
//	               has started (its New sets it when the manifest did not)
package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"

	windows "github.com/rm4n0s/glutter/internal/platform/windows"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	version  = syscall.NewLazyDLL("version.dll")
)

// awareness names the DPI awareness of the calling thread.
func awareness() string {
	ctx, _, _ := user32.NewProc("GetThreadDpiAwarenessContext").Call()
	for _, c := range []struct {
		name   string
		handle uintptr
	}{{"PerMonitorV2", ^uintptr(3)}, {"PerMonitor", ^uintptr(2)}, {"System", ^uintptr(1)}, {"Unaware", ^uintptr(0)}} {
		if eq, _, _ := user32.NewProc("AreDpiAwarenessContextsEqual").Call(ctx, c.handle); eq != 0 {
			return c.name
		}
	}
	return fmt.Sprintf("unknown(%#x)", ctx)
}

// iconSize loads icon resource 1 at n pixels and returns the size of its colour bitmap.
func iconSize(n int) string {
	inst, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	const imageIcon = 1
	h, _, err := user32.NewProc("LoadImageW").Call(inst, 1, imageIcon, uintptr(n), uintptr(n), 0)
	if h == 0 {
		return fmt.Sprintf("LoadImageW failed: %v", err)
	}
	defer user32.NewProc("DestroyIcon").Call(h)
	var info struct {
		fIcon      int32
		xHot, yHot uint32
		hbmMask    uintptr
		hbmColor   uintptr
	}
	if ok, _, err := user32.NewProc("GetIconInfo").Call(h, uintptr(unsafe.Pointer(&info))); ok == 0 {
		return fmt.Sprintf("GetIconInfo failed: %v", err)
	}
	defer gdi32.NewProc("DeleteObject").Call(info.hbmMask)
	defer gdi32.NewProc("DeleteObject").Call(info.hbmColor)
	var bm struct {
		typ, width, height, widthBytes int32
		planes, bitsPixel              uint16
		bits                           uintptr
	}
	if ok, _, err := gdi32.NewProc("GetObjectW").Call(info.hbmColor, unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm))); ok == 0 {
		return fmt.Sprintf("GetObjectW failed: %v", err)
	}
	return fmt.Sprintf("%dx%d", bm.width, bm.height)
}

// versionString reads a string of the version information of this .exe.
func versionString(name string) string {
	exe, err := os.Executable()
	if err != nil {
		return "no executable: " + err.Error()
	}
	path, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return err.Error()
	}
	size, _, _ := version.NewProc("GetFileVersionInfoSizeW").Call(uintptr(unsafe.Pointer(path)), 0)
	if size == 0 {
		return "no version information"
	}
	data := make([]byte, size)
	if ok, _, err := version.NewProc("GetFileVersionInfoW").Call(uintptr(unsafe.Pointer(path)), 0, size, uintptr(unsafe.Pointer(&data[0]))); ok == 0 {
		return fmt.Sprintf("GetFileVersionInfoW failed: %v", err)
	}
	sub, err := syscall.UTF16PtrFromString(`\StringFileInfo\040904B0\` + name)
	if err != nil {
		return err.Error()
	}
	var value *uint16
	var n uint32
	if ok, _, _ := version.NewProc("VerQueryValueW").Call(uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(sub)), uintptr(unsafe.Pointer(&value)), uintptr(unsafe.Pointer(&n))); ok == 0 {
		return "no " + name
	}
	chars := unsafe.Slice(value, n)
	if len(chars) > 0 && chars[len(chars)-1] == 0 {
		chars = chars[:len(chars)-1]
	}
	return string(utf16.Decode(chars))
}

func main() {
	runtime.LockOSThread()
	fmt.Println("dpi-start:", awareness())
	acp, _, _ := kernel32.NewProc("GetACP").Call()
	fmt.Println("acp:", acp)
	for _, n := range []int{16, 24, 32, 48, 64, 128, 256} {
		fmt.Printf("icon-%d: %s\n", n, iconSize(n))
	}
	fmt.Println("version:", versionString("ProductVersion"), versionString("ProductName"))
	if len(os.Args) > 1 && os.Args[1] == "platform" {
		p, err := windows.New(windows.Options{})
		if err != nil {
			fmt.Println("platform failed:", err)
			os.Exit(1)
		}
		fmt.Println("dpi-platform:", awareness())
		if err := p.Close(); err != nil {
			fmt.Println("close failed:", err)
			os.Exit(1)
		}
	}
}
