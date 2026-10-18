package wl

import (
	"testing"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// message reads message n of a wl_message array in C memory.
func readMessage(t *testing.T, array uintptr, count int32, n int) (name, signature string, types []uintptr) {
	t.Helper()
	if n >= int(count) {
		t.Fatalf("message %d requested, but the table has %d", n, count)
	}
	m := (*cMessage)(dl.Pointer(array + uintptr(n)*unsafe.Sizeof(cMessage{})))
	name, signature = dl.GoString(m.Name), dl.GoString(m.Signature)
	slots := 0
	for _, c := range signature {
		if c < '0' || c > '9' {
			if c != '?' {
				slots++
			}
		}
	}
	for k := range slots {
		types = append(types, *(*uintptr)(dl.Pointer(m.Types + uintptr(k)*unsafe.Sizeof(uintptr(0)))))
	}
	return name, signature, types
}

func TestInterfaceTables(t *testing.T) {
	tests := []struct {
		iface      *Interface
		name       string
		version    int32
		requests   int32
		events     int32
		firstReq   string
		firstSig   string
		firstTypes []*Interface
	}{
		{WlDisplayInterface, "wl_display", 1, 2, 2, "sync", "n", []*Interface{WlCallbackInterface}},
		{WlRegistryInterface, "wl_registry", 1, 1, 2, "bind", "usun", []*Interface{nil, nil, nil, nil}},
		{WlSurfaceInterface, "wl_surface", 7, 12, 4, "destroy", "", nil},
		{XdgWmBaseInterface, "xdg_wm_base", 7, 4, 1, "destroy", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := (*cInterface)(dl.Pointer(tt.iface.ptr))
			if got := dl.GoString(c.Name); got != tt.name {
				t.Errorf("name = %q, want %q", got, tt.name)
			}
			if tt.iface.Name != tt.name || tt.iface.Version != uint32(tt.version) {
				t.Errorf("Go side = %q v%d, want %q v%d", tt.iface.Name, tt.iface.Version, tt.name, tt.version)
			}
			if c.Version != tt.version || c.MethodCount != tt.requests || c.EventCount != tt.events {
				t.Errorf("version/requests/events = %d/%d/%d, want %d/%d/%d",
					c.Version, c.MethodCount, c.EventCount, tt.version, tt.requests, tt.events)
			}
			name, sig, types := readMessage(t, c.Methods, c.MethodCount, 0)
			if name != tt.firstReq || sig != tt.firstSig {
				t.Errorf("first request = %q %q, want %q %q", name, sig, tt.firstReq, tt.firstSig)
			}
			for k, want := range tt.firstTypes {
				var wantPtr uintptr
				if want != nil {
					wantPtr = want.ptr
				}
				if types[k] != wantPtr {
					t.Errorf("first request: type %d = %#x, want %#x", k, types[k], wantPtr)
				}
			}
		})
	}

	t.Run("object arguments point at the other interface across protocols", func(t *testing.T) {
		// xdg_wm_base.get_xdg_surface(id: new_id xdg_surface, surface: object wl_surface)
		c := (*cInterface)(dl.Pointer(XdgWmBaseInterface.ptr))
		name, sig, types := readMessage(t, c.Methods, c.MethodCount, 2)
		if name != "get_xdg_surface" || sig != "no" {
			t.Fatalf("request 2 = %q %q, want get_xdg_surface no", name, sig)
		}
		if types[0] != XdgSurfaceInterface.ptr || types[1] != WlSurfaceInterface.ptr {
			t.Errorf("types = %#x, want xdg_surface %#x and wl_surface %#x", types, XdgSurfaceInterface.ptr, WlSurfaceInterface.ptr)
		}
	})
	t.Run("since version and nullable markers", func(t *testing.T) {
		// wl_surface.damage_buffer is since 4; attach has a nullable buffer.
		c := (*cInterface)(dl.Pointer(WlSurfaceInterface.ptr))
		if _, sig, _ := readMessage(t, c.Methods, c.MethodCount, 1); sig != "?oii" {
			t.Errorf("attach signature = %q, want ?oii", sig)
		}
		if _, sig, _ := readMessage(t, c.Methods, c.MethodCount, 9); sig != "4iiii" {
			t.Errorf("damage_buffer signature = %q, want 4iiii", sig)
		}
	})
	t.Run("events create objects of the type table's interface", func(t *testing.T) {
		// wl_data_device.data_offer(id: new_id wl_data_offer)
		c := (*cInterface)(dl.Pointer(WlDataDeviceInterface.ptr))
		name, sig, types := readMessage(t, c.Events, c.EventCount, 0)
		if name != "data_offer" || sig != "n" || types[0] != WlDataOfferInterface.ptr {
			t.Errorf("event 0 = %q %q %#x, want data_offer n %#x", name, sig, types, WlDataOfferInterface.ptr)
		}
	})
}

func TestArgs(t *testing.T) {
	a := newArgs(7)
	defer a.free()
	a.setInt(0, -5)
	a.setUint(1, 0xfedcba98)
	a.setFixed(2, -1.5)
	a.setString(3, "héllo")
	a.setNullableString(4, "")
	a.setObject(5, 0x1234)
	a.setArray(6, []byte{1, 2, 3, 4, 5, 6, 7, 8})

	e := eventArgs(a.ptr) // the same memory read the way events are read
	if got := e.int(0); got != -5 {
		t.Errorf("int = %d, want -5", got)
	}
	if got := e.uint(1); got != 0xfedcba98 {
		t.Errorf("uint = %#x, want 0xfedcba98", got)
	}
	if got := e.fixed(2); got != -1.5 {
		t.Errorf("fixed = %v, want -1.5", got)
	}
	if got := e.str(3); got != "héllo" {
		t.Errorf("string = %q, want héllo", got)
	}
	if got := e.object(4); got != 0 {
		t.Errorf("empty nullable string = %#x, want NULL", got)
	}
	if got := e.object(5); got != 0x1234 {
		t.Errorf("object = %#x, want 0x1234", got)
	}
	if got := e.array(6); string(got) != "\x01\x02\x03\x04\x05\x06\x07\x08" {
		t.Errorf("array = %v", got)
	}
	if got := e.uint32s(6); len(got) != 2 || got[0] != 0x04030201 || got[1] != 0x08070605 {
		t.Errorf("uint32s = %#x, want [0x04030201 0x08070605]", got)
	}
}

func TestFixedRounding(t *testing.T) {
	tests := []struct {
		in   float64
		want float64 // after a round trip through wl_fixed_t
	}{
		{0, 0}, {1, 1}, {-1, -1}, {10.5, 10.5}, {-3.25, -3.25},
		{0.001, 0}, {0.003, 0.00390625}, {1.0 / 3, 0.33203125}, {8388607, 8388607},
	}
	for _, tt := range tests {
		a := newArgs(1)
		a.setFixed(0, tt.in)
		got := eventArgs(a.ptr).fixed(0)
		a.free()
		if got != tt.want {
			t.Errorf("fixed(%v) round trip = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestUint32sToBytes(t *testing.T) {
	got := uint32sToBytes([]uint32{1, 0x01020304})
	want := []byte{1, 0, 0, 0, 4, 3, 2, 1}
	if string(got) != string(want) {
		t.Errorf("uint32sToBytes = %v, want %v", got, want)
	}
	if got := uint32sToBytes(nil); len(got) != 0 {
		t.Errorf("uint32sToBytes(nil) = %v, want empty", got)
	}
}
