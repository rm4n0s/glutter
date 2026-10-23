//go:build linux

package vulkan

import (
	"sort"

	"github.com/rm4n0s/errors"
)

// memoryBlockSize is the size of the blocks of device memory that the allocator
// asks for, per memory type (SPEC §9.4); a larger request gets a block of its own.
const memoryBlockSize = 64 << 20

// span is a free range of a block.
type span struct{ offset, size uint64 }

// freeList is the free ranges of one block, sorted by offset and never adjacent: a
// range that is freed next to another joins it.
type freeList struct {
	spans []span
}

func newFreeList(size uint64) *freeList { return &freeList{spans: []span{{0, size}}} }

func alignUp64(n, alignment uint64) uint64 {
	if alignment <= 1 {
		return n
	}
	return (n + alignment - 1) / alignment * alignment
}

// alloc takes size bytes at an offset that is a multiple of alignment, from the first
// range that has room.
func (f *freeList) alloc(size, alignment uint64) (offset uint64, ok bool) {
	for i, s := range f.spans {
		start := alignUp64(s.offset, alignment)
		if start+size > s.offset+s.size {
			continue
		}
		var rest []span
		if start > s.offset {
			rest = append(rest, span{s.offset, start - s.offset})
		}
		if end := start + size; end < s.offset+s.size {
			rest = append(rest, span{end, s.offset + s.size - end})
		}
		f.spans = append(f.spans[:i], append(rest, f.spans[i+1:]...)...)
		return start, true
	}
	return 0, false
}

// free gives the range back and joins it to the ranges next to it.
func (f *freeList) free(offset, size uint64) {
	i := sort.Search(len(f.spans), func(i int) bool { return f.spans[i].offset > offset })
	f.spans = append(f.spans, span{})
	copy(f.spans[i+1:], f.spans[i:])
	f.spans[i] = span{offset, size}
	if i+1 < len(f.spans) && f.spans[i].offset+f.spans[i].size == f.spans[i+1].offset {
		f.spans[i].size += f.spans[i+1].size
		f.spans = append(f.spans[:i+1], f.spans[i+2:]...)
	}
	if i > 0 && f.spans[i-1].offset+f.spans[i-1].size == f.spans[i].offset {
		f.spans[i-1].size += f.spans[i].size
		f.spans = append(f.spans[:i], f.spans[i+1:]...)
	}
}

// whole says whether the range [0, size) is entirely free.
func (f *freeList) whole(size uint64) bool {
	return len(f.spans) == 1 && f.spans[0].offset == 0 && f.spans[0].size == size
}

// memoryProps is what the allocator needs of VkPhysicalDeviceMemoryProperties: the
// property flags of each memory type.
type memoryProps struct {
	types []uint32
}

func newMemoryProps(p physicalDeviceMemoryProperties) memoryProps {
	m := memoryProps{}
	for i := uint32(0); i < p.memoryTypeCount && i < maxMemoryTypes; i++ {
		m.types = append(m.types, p.memoryTypes[i].propertyFlags)
	}
	return m
}

// findType returns the first memory type that the resource accepts (a bit of
// typeBits) and that has all the required properties and, if there is one, all the
// preferred ones too; failing that, the first with the required ones.
func (m memoryProps) findType(typeBits, required, preferred uint32) (uint32, bool) {
	for _, want := range []uint32{required | preferred, required} {
		for i, flags := range m.types {
			if typeBits&(1<<uint(i)) != 0 && flags&want == want {
				return uint32(i), true
			}
		}
	}
	return 0, false
}

// memoryBackend is the Vulkan calls the allocator makes, so that a test can answer
// them.
type memoryBackend interface {
	// allocate makes a block of device memory of the type.
	allocate(size uint64, typeIndex uint32) (deviceMemoryH, error)
	// mapAll maps the whole block for good; the pointer stays valid until free.
	mapAll(mem deviceMemoryH, size uint64) (uintptr, error)
	// free releases the block.
	free(mem deviceMemoryH)
}

// allocKey separates the blocks of a memory type for buffers (linear) and for images
// (optimal), so that the buffer-image granularity never matters.
type allocKey struct {
	typeIndex uint32
	linear    bool
}

// memBlock is one block of device memory and its free ranges.
type memBlock struct {
	mem  deviceMemoryH
	size uint64
	ptr  uintptr // the mapping, 0 when the type is not host visible
	free *freeList
	key  allocKey
}

// allocation is a range of a block.
type allocation struct {
	block        *memBlock
	offset, size uint64
}

// mapped returns the address of the allocation in the mapped block, or 0.
func (a allocation) mapped() uintptr {
	if a.block == nil || a.block.ptr == 0 {
		return 0
	}
	return a.block.ptr + uintptr(a.offset)
}

// allocator hands out ranges of 64 MiB blocks of device memory, one list of blocks
// for each memory type and kind of resource (SPEC §9.4).
type allocator struct {
	backend memoryBackend
	props   memoryProps
	blocks  map[allocKey][]*memBlock
}

func newAllocator(b memoryBackend, p memoryProps) *allocator {
	return &allocator{backend: b, props: p, blocks: map[allocKey][]*memBlock{}}
}

// alloc finds room for a resource with the requirements. host asks for memory that
// the CPU can map and that needs no flushes (host visible and coherent); otherwise
// the memory is device local when there is such a type. linear is true for buffers.
// It fails with GPUOutOfMemory when no memory type fits or the driver says it is out
// of memory.
func (a *allocator) alloc(req memoryRequirements, host, linear bool) (allocation, error) {
	required, preferred := uint32(0), uint32(memoryDeviceLocal)
	if host {
		required, preferred = memoryHostVisible|memoryHostCoherent, 0
	}
	typeIndex, ok := a.props.findType(req.memoryTypeBits, required, preferred)
	if !ok {
		return allocation{}, errors.New(tagGPUOutOfMemory, "no memory type fits the resource",
			"type_bits", int(req.memoryTypeBits), "host", host)
	}
	key := allocKey{typeIndex, linear}
	for _, b := range a.blocks[key] {
		if off, ok := b.free.alloc(req.size, req.alignment); ok {
			return allocation{b, off, req.size}, nil
		}
	}
	size := max(uint64(memoryBlockSize), req.size)
	mem, err := a.backend.allocate(size, typeIndex)
	if err != nil {
		return allocation{}, err
	}
	b := &memBlock{mem: mem, size: size, free: newFreeList(size), key: key}
	if a.props.types[typeIndex]&memoryHostVisible != 0 {
		if b.ptr, err = a.backend.mapAll(mem, size); err != nil {
			a.backend.free(mem)
			return allocation{}, err
		}
	}
	a.blocks[key] = append(a.blocks[key], b)
	off, _ := b.free.alloc(req.size, req.alignment)
	return allocation{b, off, req.size}, nil
}

// release gives an allocation back. A block that is empty again and is larger than the
// usual size (it was made for one big resource) is freed.
func (a *allocator) release(al allocation) {
	if al.block == nil {
		return
	}
	b := al.block
	b.free.free(al.offset, al.size)
	if b.size > memoryBlockSize && b.free.whole(b.size) {
		blocks := a.blocks[b.key]
		for i, other := range blocks {
			if other == b {
				a.blocks[b.key] = append(blocks[:i], blocks[i+1:]...)
				break
			}
		}
		a.backend.free(b.mem)
	}
}

// close frees every block.
func (a *allocator) close() {
	for key, blocks := range a.blocks {
		for _, b := range blocks {
			a.backend.free(b.mem)
		}
		delete(a.blocks, key)
	}
}
