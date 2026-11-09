//go:build darwin

package darwin

import (
	"sync"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// pixelFormatBGRA is the four-character code 'BGRA' that IOSurface names the byte order blue, green,
// red, alpha by.
const pixelFormatBGRA = 0x42475241

// ioSurfaceFuncs are the functions of the IOSurface framework that the pixel target uses.
type ioSurfaceFuncs struct {
	create         func(props cf.Ref) uintptr
	lock           func(s uintptr, options uint32, seed *uint32) int32
	unlock         func(s uintptr, options uint32, seed *uint32) int32
	baseAddress    func(s uintptr) uintptr
	bytesPerRow    func(s uintptr) uint64
	isInUse        func(s uintptr) bool
	setValue       func(s uintptr, key, value cf.Ref)
	keyWidth       cf.Ref
	keyHeight      cf.Ref
	keyBytesPerEl  cf.Ref
	keyPixelFormat cf.Ref
	keyColorSpace  cf.Ref
	// srgb is the property list of the sRGB color space, 0 when Core Graphics could not make it.
	srgb cf.Ref
}

var ioSurface = sync.OnceValues(func() (*ioSurfaceFuncs, error) {
	lib, err := dl.OpenFramework("IOSurface")
	if err != nil {
		return nil, err
	}
	f := new(ioSurfaceFuncs)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&f.create, "IOSurfaceCreate"}, {&f.lock, "IOSurfaceLock"}, {&f.unlock, "IOSurfaceUnlock"},
		{&f.baseAddress, "IOSurfaceGetBaseAddress"}, {&f.bytesPerRow, "IOSurfaceGetBytesPerRow"},
		{&f.isInUse, "IOSurfaceIsInUse"}, {&f.setValue, "IOSurfaceSetValue"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	for _, c := range []struct {
		dst    *cf.Ref
		symbol string
	}{
		{&f.keyWidth, "kIOSurfaceWidth"}, {&f.keyHeight, "kIOSurfaceHeight"},
		{&f.keyBytesPerEl, "kIOSurfaceBytesPerElement"}, {&f.keyPixelFormat, "kIOSurfacePixelFormat"},
		{&f.keyColorSpace, "kIOSurfaceColorSpace"},
	} {
		addr, err := lib.Symbol(c.symbol)
		if err != nil {
			return nil, err
		}
		*c.dst = *(*cf.Ref)(dl.Pointer(addr))
	}
	f.srgb = srgbPropertyList()
	return f, nil
})

// srgbPropertyList is the property list of the sRGB color space, which IOSurface keeps as the color space
// of a surface, or 0 when it cannot be made.
func srgbPropertyList() cf.Ref {
	cg, err := dl.OpenFramework("CoreGraphics")
	if err != nil {
		return 0
	}
	var create func(name cf.Ref) uintptr
	var copyList func(space uintptr) cf.Ref
	if cg.Bind(&create, "CGColorSpaceCreateWithName") != nil || cg.Bind(&copyList, "CGColorSpaceCopyPropertyList") != nil {
		return 0
	}
	addr, err := cg.Symbol("kCGColorSpaceSRGB")
	if err != nil {
		return 0
	}
	space := create(*(*cf.Ref)(dl.Pointer(addr)))
	if space == 0 {
		return 0
	}
	defer cf.Release(cf.Ref(space))
	return copyList(space)
}

// IOSurface is a surface that the CPU draws into and the window server shows, of BGRA with alpha 255
// (gpu.PixelFormatBGRX8888), in the sRGB color space (SPEC §29.5). It implements Surface.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterSurface.mm FlutterSurface (an IOSurface
// that a Metal texture wraps; here the CPU writes it)
type IOSurface struct {
	f             *ioSurfaceFuncs
	ref           uintptr
	width, height int
	stride        int
}

// NewIOSurface makes a surface of the size in pixels. It fails with IOSurfaceCreateFailed, with the size
// as metadata, when the framework is missing or the system cannot make the surface, and with the
// error of the loader when a symbol is missing.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterSurface.mm FlutterSurface initWithSize
func NewIOSurface(width, height int) (*IOSurface, error) {
	f, err := ioSurface()
	if err != nil {
		return nil, err
	}
	if width <= 0 || height <= 0 {
		return nil, errors.New(TagIOSurfaceCreateFailed, "the size of a surface must be positive", "width", width, "height", height)
	}
	keys := []cf.Ref{f.keyWidth, f.keyHeight, f.keyBytesPerEl, f.keyPixelFormat}
	nums := []cf.Ref{cf.NewNumber(int64(width)), cf.NewNumber(int64(height)), cf.NewNumber(4), cf.NewNumber(pixelFormatBGRA)}
	defer func() {
		for _, n := range nums {
			cf.Release(n)
		}
	}()
	props := cf.NewDictionary(keys, nums)
	defer cf.Release(props)
	ref := f.create(props)
	if ref == 0 {
		return nil, errors.New(TagIOSurfaceCreateFailed, "IOSurfaceCreate returned no surface", "width", width, "height", height)
	}
	if f.srgb != 0 {
		f.setValue(ref, f.keyColorSpace, f.srgb)
	}
	s := &IOSurface{f: f, ref: ref, width: width, height: height, stride: int(f.bytesPerRow(ref))}
	return s, nil
}

// Ref is the IOSurfaceRef, for a layer's contents.
func (s *IOSurface) Ref() uintptr { return s.ref }

// Size is the size in pixels.
func (s *IOSurface) Size() (int, int) { return s.width, s.height }

// Stride is the length of a row in bytes (IOSurface rounds it up).
func (s *IOSurface) Stride() int { return s.stride }

// Lock makes the memory writable and returns it. It fails with IOSurfaceLockFailed.
func (s *IOSurface) Lock() ([]byte, error) {
	if rc := s.f.lock(s.ref, 0, nil); rc != 0 {
		return nil, errors.New(TagIOSurfaceLockFailed, "IOSurfaceLock failed", "kern_return", int(rc))
	}
	base := s.f.baseAddress(s.ref)
	return unsafe.Slice((*byte)(dl.Pointer(base)), s.stride*s.height), nil
}

// Unlock ends the CPU access.
func (s *IOSurface) Unlock() { s.f.unlock(s.ref, 0, nil) }

// InUse says whether the window server (or any other client) still uses the surface.
func (s *IOSurface) InUse() bool { return s.f.isInUse(s.ref) }

// Release gives the surface up; the window server keeps it as long as it shows it.
func (s *IOSurface) Release() {
	if s.ref != 0 {
		cf.Release(cf.Ref(s.ref))
		s.ref = 0
	}
}

var _ Surface = (*IOSurface)(nil)

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagIOSurfaceCreateFailed is the tag of the error of a surface that the system could not make.
	TagIOSurfaceCreateFailed = "IOSurfaceCreateFailed"
	// TagIOSurfaceLockFailed is the tag of the error of a surface that could not be locked for the CPU.
	TagIOSurfaceLockFailed = "IOSurfaceLockFailed"
)
