//go:build linux

package dl

import (
	"strings"
	"testing"
)

func TestLiveBlocks(t *testing.T) {
	tests := []struct {
		name  string
		alloc func() []uintptr
		want  int64 // the blocks alive after the allocations
	}{
		{"nothing", func() []uintptr { return nil }, 0},
		{"malloc, calloc and a string", func() []uintptr { return []uintptr{Malloc(8), Calloc(2, 4), CString("hello")} }, 3},
		{"many strings", func() []uintptr {
			var out []uintptr
			for range 50 {
				out = append(out, CString("x"))
			}
			return out
		}, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := LiveBlocks()
			ptrs := tt.alloc()
			if got := LiveBlocks() - before; got != tt.want {
				t.Fatalf("%d blocks alive after the allocations, want %d", got, tt.want)
			}
			for _, p := range ptrs {
				Free(p)
			}
			Free(0) // does nothing, and is not counted
			if got := LiveBlocks() - before; got != 0 {
				t.Errorf("%d blocks alive after Free, want 0", got)
			}
		})
	}
}

func TestTrackBlocksNamesTheSiteOfABlockThatIsNotFreed(t *testing.T) {
	stop := TrackBlocks()
	freed := Malloc(16)
	kept := leakyAllocation()
	Free(freed)
	left := stop()
	Free(kept)
	if len(left) != 1 {
		t.Fatalf("%d blocks left, want 1: %+v", len(left), left)
	}
	if left[0].Size != 64 {
		t.Errorf("size = %d, want 64", left[0].Size)
	}
	if !strings.Contains(strings.Join(left[0].Stack, "\n"), "leakyAllocation") {
		t.Errorf("the stack does not name the function that allocated: %v", left[0].Stack)
	}
}

// leakyAllocation is the allocation site that the test looks for.
func leakyAllocation() uintptr { return Calloc(1, 64) }

func TestCHeapInUse(t *testing.T) {
	before, ok := CHeapInUse()
	if !ok {
		t.Skip("libc has no mallinfo2")
	}
	const size = 4 << 20
	p := Malloc(size)
	during, _ := CHeapInUse()
	Free(p)
	after, _ := CHeapInUse()
	if during-before < size {
		t.Errorf("the C heap grew by %d bytes for a block of %d", during-before, size)
	}
	if after-before > size/2 {
		t.Errorf("the C heap is %d bytes larger after the block is freed", after-before)
	}
}
