//go:build windows

package win32

import (
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
)

func assertTag(t *testing.T, err error, tag, route string) *errors.Error {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tag, route)
	}
	return appErr
}

func TestRegisterClassExFailureHasARoute(t *testing.T) {
	// A class with no size is refused (ERROR_INVALID_PARAMETER).
	_, err := RegisterClassEx(&WndClassEx{})
	appErr := assertTag(t, err, TagWin32WindowClassFailed, "RegisterClassEx.Win32WindowClassFailed")
	if appErr.Metadata["code"] == nil {
		t.Error("the error has no code")
	}
}

func TestCreateWindowExFailureHasARoute(t *testing.T) {
	_, err := CreateWindowEx(0, "NoSuchClassOfGlutter", "x", 0, 0, 0, 10, 10, 0, 0, 0, 0)
	appErr := assertTag(t, err, TagWin32WindowCreateFailed, "CreateWindowEx.Win32WindowCreateFailed")
	if appErr.Metadata["class"] != "NoSuchClassOfGlutter" || (appErr.Metadata["code"] != 1411 && appErr.Metadata["code"] != 1407) { // ERROR_CLASS_DOES_NOT_EXIST, or ERROR_CANNOT_FIND_WND_CLASS
		t.Errorf("metadata = %v", appErr.Metadata)
	}
}

func TestSetProcessDpiAwarenessContextTwice(t *testing.T) {
	// The first call may fail too, when the manifest or an earlier test set the awareness; the second
	// always does, with "access denied".
	_ = SetProcessDpiAwarenessContext(DPIAwarenessContextPerMonitorAwareV2)
	err := SetProcessDpiAwarenessContext(DPIAwarenessContextPerMonitorAwareV2)
	appErr := assertTag(t, err, TagWindowsDPIAwarenessFailed, "SetProcessDpiAwarenessContext.WindowsDPIAwarenessFailed")
	if appErr.Metadata["code"] != ErrorAccessDenied {
		t.Errorf("code = %v, want ERROR_ACCESS_DENIED", appErr.Metadata["code"])
	}
}

func TestOleInitializeFailsOnAMultiThreadedApartment(t *testing.T) {
	runtime.LockOSThread() // never unlocked: the thread keeps the apartment, so the goroutine's thread is not reused
	ole, err := LoadSystemDLL("ole32.dll")
	if err != nil {
		t.Fatal(err)
	}
	coInit, err := ole.Proc("CoInitializeEx")
	if err != nil {
		t.Fatal(err)
	}
	if r, _ := coInit.Call(0, 0 /* COINIT_MULTITHREADED */); FromResult(r).Failed() {
		t.Fatalf("CoInitializeEx: %v", FromResult(r))
	}
	err = OleInitialize()
	appErr := assertTag(t, err, TagOleInitFailed, "OleInitialize.OleInitFailed")
	if appErr.Metadata["hresult"] != "0x80010106" { // RPC_E_CHANGED_MODE
		t.Errorf("hresult = %v", appErr.Metadata["hresult"])
	}
}

func TestSystemBindings(t *testing.T) {
	if w := GetSystemMetrics(0 /* SM_CXSCREEN */); w <= 0 {
		t.Errorf("SM_CXSCREEN = %d", w)
	}
	if w := GetSystemMetrics(SMCxVirtualScreen); w <= 0 {
		t.Errorf("SM_CXVIRTUALSCREEN = %d", w)
	}
	mon := MonitorFromPoint(Point{}, MonitorDefaultToPrimary)
	if mon == 0 {
		t.Fatal("no primary monitor")
	}
	mi, err := GetMonitorInfo(mon)
	if err != nil {
		t.Fatal(err)
	}
	if mi.Flags&MonitorInfoFPrimary == 0 || mi.Monitor.Right <= mi.Monitor.Left || UTF16ToString(mi.Device[:]) == "" {
		t.Errorf("monitor info = %+v", mi)
	}
	if dpi := GetDpiForMonitor(mon); dpi < 96 {
		t.Errorf("dpi = %d", dpi)
	}
	var found []uintptr
	if err := EnumDisplayMonitors(func(m uintptr, r Rect) bool {
		found = append(found, m)
		return true
	}); err != nil || len(found) == 0 {
		t.Errorf("EnumDisplayMonitors = %v, %v", found, err)
	}
	var dm DevMode
	if !EnumDisplaySettingsEx(UTF16ToString(mi.Device[:]), EnumCurrentSettings, &dm) || dm.PelsWidth == 0 {
		t.Errorf("EnumDisplaySettingsEx = %+v", dm)
	}
	if _, ok := EnumDisplayDevices(UTF16ToString(mi.Device[:]), 0); !ok {
		t.Log("the monitor has no device of its own here") // Wine has none for every adapter
	}
}

func TestRegGetDWORD(t *testing.T) {
	if _, found, err := RegGetDWORD(HKeyCurrentUser, `Software\Glutter\NoSuchKey`, "NoSuchValue"); err != nil || found {
		t.Errorf("a missing value: found %v, %v", found, err)
	}
	// Software\Microsoft\Windows\CurrentVersion\Themes\Personalize exists on Windows and not on Wine: either way no error.
	if _, _, err := RegGetDWORD(HKeyCurrentUser, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "AppsUseLightTheme"); err != nil {
		t.Error(err)
	}
}

func TestWaitableTimerWaits(t *testing.T) {
	timer, err := NewWaitableTimer()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := timer.Close(); err != nil {
			t.Error(err)
		}
	}()
	start := time.Now()
	if err := timer.WaitFor(30 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < 25*time.Millisecond || d > time.Second {
		t.Errorf("WaitFor(30ms) took %v", d)
	}
	start = time.Now()
	if err := timer.WaitFor(0); err != nil || time.Since(start) > 20*time.Millisecond {
		t.Errorf("WaitFor(0) = %v after %v", err, time.Since(start))
	}
	if err := timer.Close(); err != nil {
		t.Errorf("a second Close = %v", err)
	}
}

func TestDwmFlushReturnsOrFails(t *testing.T) {
	// On a machine with composition it waits for the vsync; under Wine it returns at once. A failure
	// is a tagged error.
	if err := DwmFlush(); err != nil {
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != TagCOMCallFailed {
			t.Errorf("DwmFlush failed with %v", err)
		}
	}
}

func TestStretchDIBitsAndBitmaps(t *testing.T) {
	info := BitmapInfo{Header: BitmapInfoHeader{
		Size: uint32(unsafe.Sizeof(BitmapInfoHeader{})), Width: 2, Height: -2, Planes: 1, BitCount: 32, Compression: BIRGB,
	}}
	bmp, bits, err := CreateDIBSection(&info)
	if err != nil || bmp == 0 || bits == 0 {
		t.Fatalf("CreateDIBSection = %v %v %v", bmp, bits, err)
	}
	copy(Bytes(bits, 16), []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4})
	DeleteObject(bmp)
	mask, err := CreateBitmap(8, 8, 1, 1, make([]byte, 16))
	if err != nil || mask == 0 {
		t.Fatalf("CreateBitmap = %v, %v", mask, err)
	}
	DeleteObject(mask)
	if n := StretchDIBits(0, 0, 0, 1, 1, 0, 0, 1, 1, nil, &info, DIBRGBColors, SrcCopy); n != 0 {
		t.Errorf("StretchDIBits with no bits = %d", n)
	}
}
