//go:build linux && !android

package wayland

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// fakeBuffer is a poolBuffer that counts how often it was freed.
type fakeBuffer struct {
	data  []byte
	freed int
}

func (b *fakeBuffer) pixels() []byte { return b.data }
func (b *fakeBuffer) free()          { b.freed++ }

// fakePool is a pixelPool over fakeBuffers that remembers them and their release functions.
type fakePool struct {
	*pixelPool
	made     []*fakeBuffer
	releases []func()
	fail     error
}

func newFakePool() *fakePool {
	f := &fakePool{}
	f.pixelPool = newPixelPool(func(width, height int, release func()) (poolBuffer, error) {
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
				slot, age, err := f.acquire(st.width, st.height)
				if err != nil {
					t.Fatalf("step %d: acquire: %v", i, err)
				}
				if got := slot.buf.(*fakeBuffer); got != f.made[st.wantBuffer] {
					t.Errorf("step %d: got buffer %d, want %d", i, indexOfBuffer(f.made, got), st.wantBuffer)
				}
				if age != st.wantAge {
					t.Errorf("step %d: age = %d, want %d", i, age, st.wantAge)
				}
				if !st.skipPresent {
					f.present(slot)
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
	a, _, _ := f.acquire(4, 4)
	f.present(a)
	b, _, _ := f.acquire(4, 4)
	f.present(b)
	// A resize: the buffer that is idle is freed when the new size is asked for, and the one
	// that the compositor holds is freed when it lets go.
	f.releases[0]()
	c, _, err := f.acquire(8, 8)
	if err != nil {
		t.Fatal(err)
	}
	f.present(c)
	if f.made[0].freed != 1 || f.made[1].freed != 0 {
		t.Errorf("freed = %d and %d after the resize, want 1 and 0", f.made[0].freed, f.made[1].freed)
	}
	f.releases[1]()
	if f.made[1].freed != 1 {
		t.Errorf("the buffer of the old size was freed %d times when released, want 1", f.made[1].freed)
	}
	f.close()
	f.close()
	if f.made[2].freed < 1 {
		t.Error("close did not free the buffer in use")
	}
	f.releases[2]() // a release after close does nothing, and does not panic
}

func TestPixelPoolKeepsAtMostThreeIdleBuffers(t *testing.T) {
	f := newFakePool()
	for i := 0; i < 5; i++ {
		s, _, err := f.acquire(4, 4)
		if err != nil {
			t.Fatal(err)
		}
		f.present(s)
	}
	for _, r := range f.releases {
		r()
	}
	freed := 0
	for _, b := range f.made {
		freed += b.freed
	}
	if freed != 2 || len(f.slots) != keepPoolBuffers {
		t.Errorf("%d buffers freed and %d kept, want 2 and %d", freed, len(f.slots), keepPoolBuffers)
	}
}

func TestPixelPoolExhaustedRoute(t *testing.T) {
	f := newFakePool()
	for i := 0; i < maxPoolBuffers; i++ {
		s, _, err := f.acquire(4, 4)
		if err != nil {
			t.Fatalf("acquire %d: %v", i, err)
		}
		f.present(s)
	}
	_, _, err := f.acquire(4, 4)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("pixelPool.acquire.ShmPoolExhausted") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestPixelPoolBufferFailureIsReturnedAndRetried(t *testing.T) {
	f := newFakePool()
	f.fail = errors.New(tagShmBufferFailed, "no memory", "op", "mmap")
	if _, _, err := f.acquire(4, 4); err == nil {
		t.Fatal("acquire succeeded with no memory")
	} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagShmBufferFailed {
		t.Fatalf("got %v, want ShmBufferFailed", err)
	}
	f.fail = nil
	if _, _, err := f.acquire(4, 4); err != nil {
		t.Fatalf("acquire after the memory came back: %v", err)
	}
}
