package d3d12

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

func TestStagingRingAlignsPlacements(t *testing.T) {
	r := newStagingRing(1 << 20) // 512 KiB for each slot
	half := r.half()
	tests := []struct {
		name      string
		slot, n   int
		alignment int
		want      int
	}{
		{"the first request is at the start of the half", 0, 100, 512, 0},
		{"the next is at the next multiple of 512", 0, 700, 512, 512},
		{"and the next after 700 bytes", 0, 1, 512, 1536},
		{"a pitch of 256 may use the smaller step", 0, 10, 256, 1792},
		{"the other slot has its own half", 1, 64, 512, half},
		{"and its own head", 1, 64, 512, half + 512},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := r.reserve(tt.slot, tt.n, tt.alignment)
			if !ok || got != tt.want {
				t.Errorf("reserve(%d, %d, %d) = %d, %v; want %d", tt.slot, tt.n, tt.alignment, got, ok, tt.want)
			}
			if got%tt.alignment != 0 {
				t.Errorf("offset %d is not a multiple of %d", got, tt.alignment)
			}
		})
	}
}

func TestStagingRingWrapsWithTheFrame(t *testing.T) {
	r := newStagingRing(4096) // 2048 for each slot
	if off, ok := r.reserve(0, 1500, 512); !ok || off != 0 {
		t.Fatalf("first = %d, %v", off, ok)
	}
	// The next placement is at 1536, and 600 bytes from there do not fit in the half.
	if _, ok := r.reserve(0, 600, 512); ok {
		t.Error("a request fitted after the half was taken")
	}
	if r.used(0) != 1500 {
		t.Errorf("a failed request moved the head: used = %d", r.used(0))
	}
	// The other slot is not in the way.
	if off, ok := r.reserve(1, 2048, 512); !ok || off != 2048 {
		t.Errorf("slot 1 = %d, %v; want 2048", off, ok)
	}
	// The frame of slot 0 is done: its half is free and the head is back at the start.
	r.reset(0)
	if off, ok := r.reserve(0, 2048, 512); !ok || off != 0 {
		t.Errorf("after reset = %d, %v; want 0", off, ok)
	}
	if r.used(1) != 2048 {
		t.Errorf("resetting slot 0 changed slot 1: used = %d", r.used(1))
	}
}

func TestStagingRingRequestLargerThanTheRing(t *testing.T) {
	r := newStagingRing(4096)
	for _, n := range []int{2049, 4096, 1 << 20} {
		if _, ok := r.reserve(0, n, 512); ok {
			t.Errorf("a request of %d bytes fitted in a half of %d", n, r.half())
		}
	}
	if r.used(0) != 0 {
		t.Errorf("failed requests took %d bytes", r.used(0))
	}
	if off, ok := r.reserve(0, r.half(), 512); !ok || off != 0 {
		t.Errorf("a request of the whole half = %d, %v", off, ok)
	}
}

func TestStagingRingKeepsTheAlignmentOfTheWholeBuffer(t *testing.T) {
	// The offset counts from the start of the buffer, so a half that does not start at a multiple of
	// the alignment would have to round: the sizes of the device make every half a multiple of 512.
	r := newStagingRing(stagingSize)
	if r.half()%textureDataPlacementAlignment != 0 {
		t.Fatalf("a half of %d is not a multiple of %d", r.half(), textureDataPlacementAlignment)
	}
	off, ok := r.reserve(1, 1, textureDataPlacementAlignment)
	if !ok || off != stagingSize/2 || off%textureDataPlacementAlignment != 0 {
		t.Errorf("slot 1 starts at %d, %v", off, ok)
	}
}

// The layout of a copy has the pitch that explicit.Limits.AlignedRowPitch says, a multiple of 256,
// and the size Direct3D 12 gives a placed footprint: all rows but the last padded.
func TestLayoutOfAgainstAlignedRowPitch(t *testing.T) {
	limits := explicit.DefaultLimits()
	if limits.CopyRowPitchAlignment != textureDataPitchAlignment {
		t.Fatalf("the design limit %d is not the %d of D3D12_TEXTURE_DATA_PITCH_ALIGNMENT", limits.CopyRowPitchAlignment, textureDataPitchAlignment)
	}
	tests := []struct {
		name        string
		w, h, bpp   int
		pitch, size int
		rows        int
	}{
		{"one pixel of rgba", 1, 1, 4, 256, 4, 1},
		{"a row of exactly 256 bytes", 64, 3, 4, 256, 256*2 + 256, 3},
		{"a row a byte over 256", 65, 2, 4, 512, 512 + 260, 2},
		{"r8 rows of 100 pixels", 100, 4, 1, 256, 256*3 + 100, 4},
		{"r8 rows of 257 pixels", 257, 2, 1, 512, 512 + 257, 2},
		{"a wide texture", 8192, 2, 4, 32768, 32768 + 32768, 2},
		{"no rows", 10, 0, 4, 256, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := layoutOf(limits, tt.w, tt.h, tt.bpp)
			if l.RowPitch != tt.pitch || l.Size != tt.size || l.Rows != tt.rows || l.RowBytes != tt.w*tt.bpp {
				t.Errorf("layoutOf(%d, %d, %d) = %+v; want pitch %d size %d rows %d", tt.w, tt.h, tt.bpp, l, tt.pitch, tt.size, tt.rows)
			}
			if l.RowPitch != limits.AlignedRowPitch(tt.w, tt.bpp) || l.RowPitch%256 != 0 {
				t.Errorf("pitch %d differs from AlignedRowPitch %d", l.RowPitch, limits.AlignedRowPitch(tt.w, tt.bpp))
			}
		})
	}
}

func TestAlignUp(t *testing.T) {
	tests := []struct{ n, a, want int }{{0, 256, 0}, {1, 256, 256}, {256, 256, 256}, {257, 256, 512}, {5, 1, 5}, {5, 0, 5}}
	for _, tt := range tests {
		if got := alignUp(tt.n, tt.a); got != tt.want {
			t.Errorf("alignUp(%d, %d) = %d, want %d", tt.n, tt.a, got, tt.want)
		}
	}
}

func TestSlotPool(t *testing.T) {
	p := newSlotPool(4)
	var got []int
	var newHeaps []bool
	for range 6 {
		n, nh := p.alloc()
		got = append(got, n)
		newHeaps = append(newHeaps, nh)
	}
	for i, n := range got {
		if n != i {
			t.Errorf("number %d = %d", i, n)
		}
	}
	// A heap is added for the first number of each heap.
	for i, want := range []bool{true, false, false, false, true, false} {
		if newHeaps[i] != want {
			t.Errorf("newHeap[%d] = %v, want %v", i, newHeaps[i], want)
		}
	}
	if p.heaps() != 2 || p.live != 6 {
		t.Errorf("heaps = %d, live = %d", p.heaps(), p.live)
	}
	if h, s := p.split(5); h != 1 || s != 1 {
		t.Errorf("split(5) = %d, %d", h, s)
	}
	// A freed number is reused first, and needs no heap.
	p.release(2)
	p.release(4)
	if n, nh := p.alloc(); n != 4 || nh {
		t.Errorf("reused = %d, %v; want 4, false (the last freed)", n, nh)
	}
	if n, nh := p.alloc(); n != 2 || nh {
		t.Errorf("reused = %d, %v; want 2, false", n, nh)
	}
	if n, nh := p.alloc(); n != 6 || nh {
		t.Errorf("fresh = %d, %v; want 6, false", n, nh)
	}
	if p.live != 7 {
		t.Errorf("live = %d, want 7", p.live)
	}
}
