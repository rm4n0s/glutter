//go:build linux && !android

package pixelpool

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// fakeBuffer is a Buffer that counts how often it was freed.
type fakeBuffer struct {
	data  []byte
	freed int
}

func (b *fakeBuffer) Pixels() []byte { return b.data }
func (b *fakeBuffer) Free()          { b.freed++ }

// fakePool is a Pool over fakeBuffers that remembers them and their release functions.
type fakePool struct {
	*Pool
	made     []*fakeBuffer
	releases []func()
	fail     error
}

func newFakePool() *fakePool {
	f := &fakePool{}
	f.Pool = New(func(width, height int, release func()) (Buffer, error) {
		if f.fail != nil {
			return nil, f.fail
		}
		b := &fakeBuffer{data: make([]byte, 4*width*height)}
		f.made = append(f.made, b)
		f.releases = append(f.releases, release)
		return b, nil
	})
	return f
}

// step is one acquire in a script: a size, whether the frame is presented, and the buffers that
// the compositor releases before it (by order of making).
type step struct {
	width, height int
	release       []int
	skipPresent   bool
	wantBuffer    int // the order of making of the buffer that acquire must give
	wantAge       int
}

func TestPixelPoolBuffersAndAges(t *testing.T) {
	tests := []struct {
		name  string
		steps []step
		made  int
	}{
		{"the first frame has a new buffer of no age", []step{{width: 4, height: 4, wantBuffer: 0, wantAge: 0}}, 1},
		{"the buffer on display is not given again, a second one is made", []step{
			{width: 4, height: 4, wantBuffer: 0, wantAge: 0},
			{width: 4, height: 4, wantBuffer: 1, wantAge: 0},
		}, 2},
		{"two buffers that the compositor releases alternate with age 2", []step{
			{width: 4, height: 4, wantBuffer: 0, wantAge: 0},
			{width: 4, height: 4, wantBuffer: 1, wantAge: 0},
			{width: 4, height: 4, release: []int{0}, wantBuffer: 0, wantAge: 2},
			{width: 4, height: 4, release: []int{1}, wantBuffer: 1, wantAge: 2},
		}, 2},
		{"a compositor that releases at once keeps one buffer, of age 1", []step{
			{width: 4, height: 4, wantBuffer: 0, wantAge: 0},
			{width: 4, height: 4, release: []int{0}, wantBuffer: 0, wantAge: 1},
			{width: 4, height: 4, release: []int{0}, wantBuffer: 0, wantAge: 1},
		}, 1},
		{"of two idle buffers the one written last is taken", []step{
			{width: 4, height: 4, wantBuffer: 0, wantAge: 0},
			{width: 4, height: 4, wantBuffer: 1, wantAge: 0},
			{width: 4, height: 4, wantBuffer: 2, wantAge: 0},
			{width: 4, height: 4, release: []int{0, 1}, wantBuffer: 1, wantAge: 2},
		}, 3},
		{"a frame that is not presented leaves the buffer as it was", []step{
			{width: 4, height: 4, wantBuffer: 0, wantAge: 0},
			{width: 4, height: 4, release: []int{0}, skipPresent: true, wantBuffer: 0, wantAge: 1},
			{width: 4, height: 4, wantBuffer: 0, wantAge: 1},
		}, 1},
		{"a new size makes a new buffer, of no age", []step{
			{width: 4, height: 4, wantBuffer: 0, wantAge: 0},
			{width: 4, height: 4, release: []int{0}, wantBuffer: 0, wantAge: 1},
			{width: 6, height: 4, release: []int{0}, wantBuffer: 1, wantAge: 0},
		}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakePool()
			for i, st := range tt.steps {
				for _, n := range st.release {
					f.releases[n]()
				}
				slot, age, err := f.Acquire(st.width, st.height)
				if err != nil {
					t.Fatalf("step %d: acquire: %v", i, err)
				}
				if got := slot.Buf.(*fakeBuffer); got != f.made[st.wantBuffer] {
					t.Errorf("step %d: got buffer %d, want %d", i, indexOfBuffer(f.made, got), st.wantBuffer)
				}
				if age != st.wantAge {
					t.Errorf("step %d: age = %d, want %d", i, age, st.wantAge)
				}
				if !st.skipPresent {
					f.Present(slot)
				}
			}
			if len(f.made) != tt.made {
				t.Errorf("%d buffers were made, want %d", len(f.made), tt.made)
			}
		})
	}
}

func indexOfBuffer(all []*fakeBuffer, b *fakeBuffer) int {
	for i, o := range all {
		if o == b {
			return i
		}
	}
	return -1
}

func TestPixelPoolFreesWhatIsNotNeeded(t *testing.T) {
	f := newFakePool()
	a, _, _ := f.Acquire(4, 4)
	f.Present(a)
	b, _, _ := f.Acquire(4, 4)
	f.Present(b)
	// A resize: the buffer that is idle is freed when the new size is asked for, and the one
	// that the compositor holds is freed when it lets go.
	f.releases[0]()
	c, _, err := f.Acquire(8, 8)
	if err != nil {
		t.Fatal(err)
	}
	f.Present(c)
	if f.made[0].freed != 1 || f.made[1].freed != 0 {
		t.Errorf("freed = %d and %d after the resize, want 1 and 0", f.made[0].freed, f.made[1].freed)
	}
	f.releases[1]()
	if f.made[1].freed != 1 {
		t.Errorf("the buffer of the old size was freed %d times when released, want 1", f.made[1].freed)
	}
	f.Close()
	f.Close()
	if f.made[2].freed < 1 {
		t.Error("close did not free the buffer in use")
	}
	f.releases[2]() // a release after close does nothing, and does not panic
}

func TestPixelPoolKeepsAtMostThreeIdleBuffers(t *testing.T) {
	f := newFakePool()
	for i := 0; i < 5; i++ {
		s, _, err := f.Acquire(4, 4)
		if err != nil {
			t.Fatal(err)
		}
		f.Present(s)
	}
	for _, r := range f.releases {
		r()
	}
	freed := 0
	for _, b := range f.made {
		freed += b.freed
	}
	if freed != 2 || len(f.slots) != keepBuffers {
		t.Errorf("%d buffers freed and %d kept, want 2 and %d", freed, len(f.slots), keepBuffers)
	}
}

func TestPixelPoolExhaustedRoute(t *testing.T) {
	f := newFakePool()
	for i := 0; i < maxBuffers; i++ {
		s, _, err := f.Acquire(4, 4)
		if err != nil {
			t.Fatalf("acquire %d: %v", i, err)
		}
		f.Present(s)
	}
	_, _, err := f.Acquire(4, 4)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("Pool.Acquire.ShmPoolExhausted") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestPixelPoolBufferFailureIsReturnedAndRetried(t *testing.T) {
	f := newFakePool()
	f.fail = errors.New(tagTestBufferFailed, "no memory", "op", "mmap")
	if _, _, err := f.Acquire(4, 4); err == nil {
		t.Fatal("acquire succeeded with no memory")
	} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagTestBufferFailed {
		t.Fatalf("got %v, want TestBufferFailed", err)
	}
	f.fail = nil
	if _, _, err := f.Acquire(4, 4); err != nil {
		t.Fatalf("acquire after the memory came back: %v", err)
	}
}

// tagTestBufferFailed is the tag of the error that the fake memory gives.
const tagTestBufferFailed = "TestBufferFailed"
