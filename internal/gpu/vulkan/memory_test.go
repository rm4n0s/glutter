//go:build linux

package vulkan

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestFreeList(t *testing.T) {
	f := newFreeList(100)
	a, ok := f.alloc(10, 1)
	b, _ := f.alloc(20, 1)
	c, _ := f.alloc(30, 1)
	if !ok || a != 0 || b != 10 || c != 30 {
		t.Fatalf("offsets %d %d %d, want 0 10 30", a, b, c)
	}
	f.free(b, 20) // a hole between two live ranges
	if want := []span{{10, 20}, {60, 40}}; !reflect.DeepEqual(f.spans, want) {
		t.Errorf("spans %v, want %v", f.spans, want)
	}
	f.free(a, 10) // joins the hole on its right
	if want := []span{{0, 30}, {60, 40}}; !reflect.DeepEqual(f.spans, want) {
		t.Errorf("spans %v, want %v", f.spans, want)
	}
	f.free(c, 30) // joins both sides
	if !f.whole(100) {
		t.Errorf("after freeing everything the list is %v, want one range of 100", f.spans)
	}
}

func TestFreeListAlignment(t *testing.T) {
	f := newFreeList(1000)
	f.alloc(1, 1)
	off, ok := f.alloc(16, 256)
	if !ok || off != 256 {
		t.Fatalf("aligned alloc at %d (%v), want 256", off, ok)
	}
	// The gap before the aligned range is still free and usable.
	if gap, ok := f.alloc(100, 1); !ok || gap != 1 {
		t.Errorf("the gap was not reused: %d, %v", gap, ok)
	}
	if _, ok := f.alloc(2000, 1); ok {
		t.Error("an allocation larger than the block succeeded")
	}
	if _, ok := f.alloc(800, 256); ok {
		t.Error("an aligned allocation that does not fit after alignment succeeded")
	}
}

func TestFreeListFirstFit(t *testing.T) {
	f := newFreeList(100)
	f.alloc(10, 1)
	b, _ := f.alloc(10, 1)
	f.alloc(10, 1)
	f.free(b, 10)
	if off, _ := f.alloc(10, 1); off != 10 {
		t.Errorf("first fit gave %d, want the hole at 10", off)
	}
	f.free(10, 10)
	if off, _ := f.alloc(11, 1); off != 30 {
		t.Errorf("an 11 byte request gave %d, want 30 (the hole is 10)", off)
	}
}

func TestFindMemoryType(t *testing.T) {
	const (
		dl  = memoryDeviceLocal
		hv  = memoryHostVisible
		hc  = memoryHostCoherent
		all = hv | hc
	)
	m := memoryProps{types: []uint32{dl, all, dl | all, hv}}
	tests := []struct {
		name                      string
		bits, required, preferred uint32
		want                      uint32
		ok                        bool
	}{
		{"device local is preferred", 0xF, 0, dl, 0, true},
		{"host visible and coherent is required", 0xF, all, 0, 1, true},
		{"the first type that has the preferred too", 0xF, all, dl, 2, true},
		{"a type that the resource does not accept is skipped", 0b1110, 0, dl, 2, true},
		{"no type has the preferred flag: the required ones will do", 0b0010, all, dl, 1, true},
		{"no type has the required flags", 0b1001, all, 0, 0, false},
		{"no type is accepted", 0, 0, 0, 0, false},
		{"preferred is dropped when nothing has it", 0b1000, 0, dl, 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := m.findType(tt.bits, tt.required, tt.preferred)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Errorf("findType = %d, %v, want %d, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

// fakeMemory is a memoryBackend that records what the allocator asks for.
type fakeMemory struct {
	next      deviceMemoryH
	allocated map[deviceMemoryH]uint64
	freed     []deviceMemoryH
	mapped    int
	failWith  error
}

func newFakeMemory() *fakeMemory { return &fakeMemory{next: 1, allocated: map[deviceMemoryH]uint64{}} }

func (f *fakeMemory) allocate(size uint64, typeIndex uint32) (deviceMemoryH, error) {
	if f.failWith != nil {
		return 0, f.failWith
	}
	m := f.next
	f.next++
	f.allocated[m] = size
	return m, nil
}

func (f *fakeMemory) mapAll(mem deviceMemoryH, size uint64) (uintptr, error) {
	f.mapped++
	return 0x10000 * uintptr(mem), nil
}

func (f *fakeMemory) free(mem deviceMemoryH) {
	f.freed = append(f.freed, mem)
	delete(f.allocated, mem)
}

func TestAllocator(t *testing.T) {
	props := memoryProps{types: []uint32{memoryDeviceLocal, memoryHostVisible | memoryHostCoherent}}
	req := func(size, align uint64) memoryRequirements {
		return memoryRequirements{size: size, alignment: align, memoryTypeBits: 0b11}
	}
	fm := newFakeMemory()
	a := newAllocator(fm, props)

	host1, err := a.alloc(req(100, 16), true, true)
	if err != nil {
		t.Fatal(err)
	}
	host2, _ := a.alloc(req(100, 16), true, true)
	if host1.block != host2.block || host2.offset != 112 {
		t.Errorf("two small host allocations: offsets %d and %d, blocks shared %v; want a shared block and 112",
			host1.offset, host2.offset, host1.block == host2.block)
	}
	if host1.mapped() == 0 || host2.mapped() != host1.mapped()+112 {
		t.Errorf("mapped addresses %#x and %#x", host1.mapped(), host2.mapped())
	}
	if fm.mapped != 1 || len(fm.allocated) != 1 || fm.allocated[host1.block.mem] != memoryBlockSize {
		t.Errorf("%d blocks (%v), %d mappings; want one block of %d, mapped once", len(fm.allocated), fm.allocated, fm.mapped, memoryBlockSize)
	}

	// A device-local texture goes in a block of the other type, which is not mapped.
	tex, err := a.alloc(req(4096, 256), false, false)
	if err != nil {
		t.Fatal(err)
	}
	if tex.block == host1.block || tex.mapped() != 0 || tex.block.key.typeIndex != 0 {
		t.Errorf("texture block: shared %v, mapped %#x, type %d", tex.block == host1.block, tex.mapped(), tex.block.key.typeIndex)
	}
	// A buffer and an image of the same type never share a block.
	devBuf, _ := a.alloc(req(64, 16), false, true)
	if devBuf.block == tex.block {
		t.Error("a buffer and an optimal image share a block")
	}

	// A request larger than a block gets a block of its own, freed when it is empty.
	big, err := a.alloc(req(memoryBlockSize+1, 256), true, true)
	if err != nil {
		t.Fatal(err)
	}
	if big.block.size != memoryBlockSize+1 {
		t.Errorf("the big block is %d bytes", big.block.size)
	}
	a.release(big)
	if !reflect.DeepEqual(fm.freed, []deviceMemoryH{big.block.mem}) {
		t.Errorf("freed %v after releasing the big allocation, want its block", fm.freed)
	}
	// An ordinary block is kept when it is empty, and the range is reused.
	a.release(host1)
	a.release(host2)
	again, _ := a.alloc(req(100, 16), true, true)
	if again.block != host1.block || again.offset != 0 || len(fm.freed) != 1 {
		t.Errorf("reuse: offset %d, same block %v, %d blocks freed", again.offset, again.block == host1.block, len(fm.freed))
	}

	a.release(allocation{}) // an empty allocation is ignored
	a.close()
	if len(fm.allocated) != 0 {
		t.Errorf("close left %d blocks", len(fm.allocated))
	}
}

func TestAllocatorErrors(t *testing.T) {
	props := memoryProps{types: []uint32{memoryDeviceLocal}}
	req := memoryRequirements{size: 64, alignment: 8, memoryTypeBits: 0b1}

	t.Run("no memory type fits", func(t *testing.T) {
		a := newAllocator(newFakeMemory(), props)
		_, err := a.alloc(req, true, true) // the only type is not host visible
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "GPUOutOfMemory" || !appErr.HasRoute("allocator.alloc.GPUOutOfMemory") {
			t.Errorf("got %v, want GPUOutOfMemory via allocator.alloc", err)
		}
	})
	t.Run("the driver is out of memory", func(t *testing.T) {
		fm := newFakeMemory()
		fm.failWith = vkError("vkAllocateMemory", resultErrorOutOfDevice)
		a := newAllocator(fm, props)
		_, err := a.alloc(req, false, true)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "GPUOutOfMemory" || !appErr.HasRoute("vkError.GPUOutOfMemory") {
			t.Errorf("got %v, want the GPUOutOfMemory of vkError, unchanged", err)
		}
		if appErr != nil && appErr.Metadata["call"] != "vkAllocateMemory" {
			t.Errorf("metadata = %v", appErr.Metadata)
		}
	})
}
