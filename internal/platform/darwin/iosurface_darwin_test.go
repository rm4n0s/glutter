//go:build darwin

package darwin

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestIOSurfaceWriteAndRead(t *testing.T) {
	s, err := NewIOSurface(33, 7)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Release()
	if w, h := s.Size(); w != 33 || h != 7 {
		t.Errorf("size = %dx%d", w, h)
	}
	if s.Stride() < 33*4 {
		t.Errorf("stride %d is shorter than a row", s.Stride())
	}
	pix, err := s.Lock()
	if err != nil {
		t.Fatal(err)
	}
	if len(pix) != s.Stride()*7 {
		t.Fatalf("len = %d, want %d", len(pix), s.Stride()*7)
	}
	pix[0], pix[1], pix[2], pix[3] = 1, 2, 3, 255
	last := s.Stride()*6 + 32*4
	pix[last] = 9
	s.Unlock()
	pix, err = s.Lock()
	if err != nil {
		t.Fatal(err)
	}
	if pix[0] != 1 || pix[3] != 255 || pix[last] != 9 {
		t.Errorf("the pixels were not kept: %v %v", pix[:4], pix[last])
	}
	s.Unlock()
	if s.InUse() {
		t.Error("a surface that nobody shows is in use")
	}
}

func TestIOSurfaceCreateFailedRoute(t *testing.T) {
	_, err := NewIOSurface(0, 10)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != TagIOSurfaceCreateFailed {
		t.Fatalf("err = %v", err)
	}
	if !appErr.HasRoute("NewIOSurface.IOSurfaceCreateFailed") {
		t.Errorf("route = %s", appErr.Route())
	}
}

func TestIOSurfacePoolOverRealSurfaces(t *testing.T) {
	p := NewSurfacePool(8, func(w, h int) (Surface, error) {
		s, err := NewIOSurface(w, h)
		if err != nil {
			return nil, err
		}
		return s, nil
	})
	defer p.Close()
	for i := 0; i < 5; i++ {
		b, err := p.Acquire(64, 32)
		if err != nil {
			t.Fatal(err)
		}
		b.Pix[0] = byte(i)
		if _, ok := p.Present(); !ok {
			t.Fatal("not presented")
		}
	}
	if p.Len() > 3 {
		t.Errorf("a pool that nobody reads grew to %d surfaces", p.Len())
	}
}
