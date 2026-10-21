//go:build linux

package vulkan

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

func TestArenaAllocAlignsAndZeroes(t *testing.T) {
	a := &arena{}
	defer a.free()
	for _, align := range []uintptr{1, 2, 4, 8, 16} {
		p := a.alloc(13, align)
		if p%align != 0 {
			t.Errorf("alloc(13, %d) = %#x, not aligned", align, p)
		}
		for i, b := range dl.Bytes(p, 13) {
			if b != 0 {
				t.Fatalf("alloc(13, %d): byte %d is %d, want 0", align, i, b)
			}
		}
		for i := range dl.Bytes(p, 13) {
			dl.Bytes(p, 13)[i] = 0xFF // dirty it: the next alloc and the reuse after reset must be zero
		}
	}
}

func TestArenaGrowsAndReuses(t *testing.T) {
	a := &arena{}
	defer a.free()
	first := a.alloc(100, 8)
	big := a.alloc(arenaChunk*2, 8) // larger than a chunk: a chunk of its own
	if len(a.chunks) != 2 {
		t.Fatalf("%d chunks after a big request, want 2", len(a.chunks))
	}
	dl.Bytes(big, arenaChunk*2)[arenaChunk*2-1] = 7
	small := a.alloc(100, 8)
	if small == first || small == big {
		t.Error("two live allocations share memory")
	}
	a.reset()
	again := a.alloc(100, 8)
	if again != first {
		t.Errorf("after reset the first allocation is at %#x, want %#x again", again, first)
	}
	dl.Bytes(again, 100)[0] = 1
	a.reset()
	if b := dl.Bytes(a.alloc(100, 8), 100); b[0] != 0 {
		t.Error("reused memory is not zeroed")
	}
	if len(a.chunks) != 3 { // the first chunk, the big one, and the one that "small" needed
		t.Errorf("reset changed the chunks: %d", len(a.chunks))
	}
}

func TestArenaMarkAndRewind(t *testing.T) {
	a := &arena{}
	defer a.free()
	a.alloc(10, 8)
	m := a.mark()
	p := a.alloc(50, 8)
	a.alloc(arenaChunk*2, 8) // another chunk after the mark
	a.rewind(m)
	if q := a.alloc(50, 8); q != p {
		t.Errorf("after rewind the allocation is at %#x, want %#x", q, p)
	}
	empty := &arena{}
	defer empty.free()
	m = empty.mark() // a mark of an arena that has no chunk yet
	empty.alloc(5, 8)
	empty.rewind(m)
	if empty.cur != 0 {
		t.Errorf("rewind to the empty mark left cur = %d", empty.cur)
	}
}

func TestArenaPutAndStrings(t *testing.T) {
	a := &arena{}
	defer a.free()
	info := applicationInfo{sType: 7, apiVersion: 0x403000}
	p := put(a, info)
	if got := *(*applicationInfo)(dl.Pointer(p)); got != info {
		t.Errorf("put round trip = %+v, want %+v", got, info)
	}
	if p%8 != 0 {
		t.Errorf("put address %#x is not 8-aligned", p)
	}
	vs := putSlice(a, []uint32{1, 2, 3})
	if b := dl.Bytes(vs, 12); b[0] != 1 || b[4] != 2 || b[8] != 3 {
		t.Errorf("putSlice wrote %v", b)
	}
	if putSlice(a, []uint32(nil)) != 0 {
		t.Error("putSlice of nothing is not 0")
	}
	names := a.cstrings([]string{"VK_KHR_surface", "VK_EXT_debug_utils"})
	if names == 0 || dl.GoString(ptr(names)) != "VK_KHR_surface" || dl.GoString(ptr(names+8)) != "VK_EXT_debug_utils" {
		t.Error("cstrings did not make an array of C strings")
	}
	if a.cstrings(nil) != 0 {
		t.Error("cstrings of nothing is not 0")
	}
	o := a.out()
	*(*uint64)(dl.Pointer(o)) = 0x1122334455667788
	if u64(o) != 0x1122334455667788 || u32(o) != 0x55667788 {
		t.Errorf("out readers: %#x %#x", u64(o), u32(o))
	}
}
