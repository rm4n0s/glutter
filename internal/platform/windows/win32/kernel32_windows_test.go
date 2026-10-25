//go:build windows

package win32

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

func TestHeapAllocAndFree(t *testing.T) {
	heap, err := ProcessHeap()
	if err != nil || heap == 0 {
		t.Fatalf("ProcessHeap = %#x, %v", heap, err)
	}
	mem, err := HeapAlloc(heap, HeapZeroMemory, 64)
	if err != nil {
		t.Fatal(err)
	}
	b := Bytes(mem, 64)
	for i, c := range b {
		if c != 0 {
			t.Fatalf("byte %d of a zeroed block is %d", i, c)
		}
	}
	copy(b, "glutter")
	if got := string(Bytes(mem, 7)); got != "glutter" {
		t.Errorf("block holds %q", got)
	}
	if err := HeapFree(heap, 0, mem); err != nil {
		t.Fatal(err)
	}
}

func TestGetModuleHandle(t *testing.T) {
	tests := []struct {
		name, module string
		wantErr      bool
	}{
		{"the program", "", false},
		{"kernel32", "kernel32.dll", false},
		{"a module that is not loaded", "glutter_not_loaded.dll", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := GetModuleHandle(tt.module)
			if !tt.wantErr {
				if err != nil || h == 0 {
					t.Fatalf("GetModuleHandle = %#x, %v", h, err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != "Win32CallFailed" || !appErr.HasRoute("GetModuleHandle->Win32Error.Win32CallFailed") {
				t.Errorf("got %s via %s", appErr.Tag, appErr.Route())
			}
			if appErr.Metadata["function"] != "GetModuleHandleW" || appErr.Metadata["code"] != 126 { // ERROR_MOD_NOT_FOUND
				t.Errorf("metadata = %v", appErr.Metadata)
			}
		})
	}
}

func TestGlobalAlloc(t *testing.T) {
	h, err := GlobalAlloc(GMemMoveable|GMemZeroInit, 32)
	if err != nil || h == 0 {
		t.Fatalf("GlobalAlloc = %#x, %v", h, err)
	}
	if err := GlobalFree(h); err != nil {
		t.Fatal(err)
	}
}

func TestEvents(t *testing.T) {
	manual, err := CreateEvent(true, false)
	if err != nil {
		t.Fatal(err)
	}
	auto, err := CreateEvent(false, true) // signaled at the start
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = CloseHandle(manual)
		_ = CloseHandle(auto)
	})
	wait := func(h uintptr, d time.Duration) WaitResult {
		t.Helper()
		r, err := WaitForSingleObject(h, d)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	steps := []struct {
		name string
		do   func()
		got  func() WaitResult
		want WaitResult
	}{
		{"a manual event starts unsignaled", func() {}, func() WaitResult { return wait(manual, 5*time.Millisecond) }, WaitTimeout},
		{"SetEvent signals it", func() { mustOK(t, SetEvent(manual)) }, func() WaitResult { return wait(manual, 5*time.Millisecond) }, WaitObject0},
		{"it stays signaled", func() {}, func() WaitResult { return wait(manual, 0) }, WaitObject0},
		{"ResetEvent clears it", func() { mustOK(t, ResetEvent(manual)) }, func() WaitResult { return wait(manual, 0) }, WaitTimeout},
		{"an auto event lets one wait through", func() {}, func() WaitResult { return wait(auto, 0) }, WaitObject0},
		{"and resets itself", func() {}, func() WaitResult { return wait(auto, 0) }, WaitTimeout},
	}
	for _, s := range steps {
		s.do()
		if got := s.got(); got != s.want {
			t.Errorf("%s: wait = %#x, want %#x", s.name, got, s.want)
		}
	}

	mustOK(t, SetEvent(manual))
	multi := []struct {
		name    string
		handles []uintptr
		all     bool
		want    WaitResult
	}{
		{"the second of two is signaled", []uintptr{auto, manual}, false, WaitObject0 + 1},
		{"all of two, one not signaled", []uintptr{auto, manual}, true, WaitTimeout},
	}
	for _, m := range multi {
		got, err := WaitForMultipleObjects(m.handles, m.all, 5*time.Millisecond)
		if err != nil || got != m.want {
			t.Errorf("%s: wait = %#x, %v; want %#x", m.name, got, err, m.want)
		}
	}
	for _, bad := range [][]uintptr{nil, make([]uintptr, 65)} {
		_, err := WaitForMultipleObjects(bad, false, 0)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "Win32CallFailed" || !appErr.HasRoute("WaitForMultipleObjects.Win32CallFailed") {
			t.Errorf("%d handles: got %v, want Win32CallFailed", len(bad), err)
		}
	}
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestInvalidHandlesFail(t *testing.T) {
	h, err := CreateEvent(true, false)
	if err != nil {
		t.Fatal(err)
	}
	mustOK(t, CloseHandle(h))
	tests := []struct {
		name string
		call func() error
		fn   string
		path string
	}{
		{"closing a closed handle", func() error { return CloseHandle(h) }, "CloseHandle", "CloseHandle->Win32Error.Win32CallFailed"},
		{"setting a closed event", func() error { return SetEvent(h) }, "SetEvent", "SetEvent->Win32Error.Win32CallFailed"},
		{"resetting a closed event", func() error { return ResetEvent(h) }, "ResetEvent", "ResetEvent->Win32Error.Win32CallFailed"},
		{"waiting for a closed event", func() error { _, err := WaitForSingleObject(h, 0); return err }, "WaitForSingleObject", "WaitForSingleObject->Win32Error.Win32CallFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr, ok := errors.FromError(tt.call())
			if !ok {
				t.Fatal("expected *errors.Error")
			}
			if appErr.Tag != "Win32CallFailed" || !appErr.HasRoute(tt.path) || appErr.Metadata["function"] != tt.fn {
				t.Errorf("got %s via %s with %v, want %s", appErr.Tag, appErr.Route(), appErr.Metadata, tt.path)
			}
		})
	}
}

func TestSleepAndTimeouts(t *testing.T) {
	start := time.Now()
	if err := Sleep(20 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < 15*time.Millisecond {
		t.Errorf("Sleep(20ms) took %v", d)
	}
	tests := []struct {
		d    time.Duration
		want uint32
	}{
		{WaitInfinite, 0xFFFFFFFF},
		{0, 0},
		{1500 * time.Microsecond, 1},
		{2 * time.Second, 2000},
		{time.Duration(1<<62 - 1), 0xFFFFFFFE},
	}
	for _, tt := range tests {
		if got := timeoutMillis(tt.d); got != tt.want {
			t.Errorf("timeoutMillis(%v) = %d, want %d", tt.d, got, tt.want)
		}
	}
}
