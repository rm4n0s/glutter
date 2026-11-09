package darwin

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// fakeSurface is a surface whose use by the window server a test sets.
type fakeSurface struct {
	w, h     int
	inUse    bool
	locked   bool
	released bool
	pix      []byte
}

func (f *fakeSurface) Size() (int, int) { return f.w, f.h }
func (f *fakeSurface) Stride() int      { return f.w * 4 }
func (f *fakeSurface) Lock() ([]byte, error) {
	f.locked = true
	return f.pix, nil
}
func (f *fakeSurface) Unlock()     { f.locked = false }
func (f *fakeSurface) InUse() bool { return f.inUse }
func (f *fakeSurface) Release()    { f.released = true }

func newFakePool(max int) (*SurfacePool, *[]*fakeSurface) {
	var made []*fakeSurface
	p := NewSurfacePool(max, func(w, h int) (Surface, error) {
		s := &fakeSurface{w: w, h: h, pix: make([]byte, w*h*4)}
		made = append(made, s)
		return s, nil
	})
	return p, &made
}

func TestSurfacePoolAges(t *testing.T) {
	p, made := newFakePool(8)
	// The first buffer is new: age 0.
	b1, err := p.Acquire(10, 10)
	if err != nil || b1.Age != 0 {
		t.Fatalf("first buffer: age %d, err %v", b1.Age, err)
	}
	if _, ok := p.Present(); !ok {
		t.Fatal("Present of an acquired buffer says nothing was acquired")
	}
	// The second is a new surface (the first is shown): age 0.
	b2, _ := p.Acquire(10, 10)
	if b2.Age != 0 || len(*made) != 2 {
		t.Fatalf("second buffer: age %d, %d surfaces", b2.Age, len(*made))
	}
	p.Present()
	// The third reuses the first, which was shown two presentations ago: age 2... the first was
	// presented first (seq 1) and the pool is at seq 2, so the age is 2.
	b3, _ := p.Acquire(10, 10)
	if b3.Age != 2 || len(*made) != 2 {
		t.Fatalf("third buffer: age %d, %d surfaces (want 2, 2)", b3.Age, len(*made))
	}
	p.Present()
	b4, _ := p.Acquire(10, 10)
	if b4.Age != 2 {
		t.Errorf("fourth buffer: age %d, want 2 (the second surface, shown two presentations ago)", b4.Age)
	}
}

func TestSurfacePoolSkipsSurfacesInUse(t *testing.T) {
	p, made := newFakePool(8)
	p.Acquire(4, 4)
	p.Present()
	p.Acquire(4, 4)
	p.Present()
	// The surface of the first presentation is still read by the window server.
	(*made)[0].inUse = true
	b, _ := p.Acquire(4, 4)
	if b.Age != 0 || len(*made) != 3 {
		t.Errorf("a surface in use was handed out or no new one was made: age %d, %d surfaces", b.Age, len(*made))
	}
	if b.Surface == Surface((*made)[0]) || b.Surface == Surface((*made)[1]) {
		t.Error("the buffer is a surface that is shown or in use")
	}
}

func TestSurfacePoolExhausted(t *testing.T) {
	p, made := newFakePool(2)
	p.Acquire(4, 4)
	p.Present()
	p.Acquire(4, 4)
	p.Present()
	for _, s := range *made {
		s.inUse = true
	}
	_, err := p.Acquire(4, 4)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != TagIOSurfacePoolExhausted {
		t.Fatalf("err = %v, want %s", err, TagIOSurfacePoolExhausted)
	}
	if !appErr.HasRoute("SurfacePool.Acquire.IOSurfacePoolExhausted") {
		t.Errorf("route = %s", appErr.Route())
	}
}

func TestSurfacePoolGivesUpWhatWasNotPresented(t *testing.T) {
	p, made := newFakePool(8)
	b1, _ := p.Acquire(4, 4)
	p.Present()
	_ = b1
	b2, _ := p.Acquire(4, 4) // a new surface
	first := b2.Surface
	// The frame is abandoned: the next Acquire gives the same surface back, with age 0 (its content is
	// not a frame) and unlocked in between.
	b3, _ := p.Acquire(4, 4)
	if b3.Surface != first {
		t.Error("an abandoned surface was not reused")
	}
	if b3.Age != 0 {
		t.Errorf("age of an abandoned surface = %d, want 0", b3.Age)
	}
	if len(*made) != 2 {
		t.Errorf("%d surfaces, want 2", len(*made))
	}
	if _, ok := p.Present(); !ok {
		t.Error("nothing to present")
	}
	if _, ok := p.Present(); ok {
		t.Error("a second Present with nothing acquired says it presented")
	}
}

func TestSurfacePoolNewSizeReleasesTheOldSurfaces(t *testing.T) {
	p, made := newFakePool(8)
	p.Acquire(4, 4)
	p.Present()
	p.Acquire(4, 4)
	p.Present()
	b, _ := p.Acquire(8, 6)
	if b.Age != 0 || b.Width != 8 || b.Height != 6 {
		t.Errorf("buffer = age %d, %dx%d", b.Age, b.Width, b.Height)
	}
	for i, s := range (*made)[:2] {
		if !s.released {
			t.Errorf("surface %d of the old size was not released", i)
		}
	}
	if p.Len() != 1 {
		t.Errorf("pool holds %d surfaces, want 1", p.Len())
	}
	p.Close()
	if !(*made)[2].released || p.Len() != 0 {
		t.Error("Close did not release the surfaces")
	}
}
