//go:build linux && !android

package wayland

import (
	"syscall"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
)

// maxShmBytes bounds one buffer, so that a bad size cannot ask for gigabytes.
const maxShmBytes = 1 << 30

// mfdCloexec is MFD_CLOEXEC for memfd_create.
const mfdCloexec = 1

// shmBuffer is a wl_buffer backed by shared memory that this process also maps,
// so it can draw into it.
type shmBuffer struct {
	buffer        *wl.WlBuffer
	data          []byte
	width, height int32
	// released is true once the compositor sent wl_buffer.release for the
	// buffer, after it was replaced on the surface.
	released bool
	freed    bool
}

// newShmBuffer creates a width x height buffer in XRGB8888 and fills it with
// color (0xAARRGGBB; the alpha byte is ignored by the format). onRelease, if
// not nil, is called when the compositor is done with the buffer. It fails with
// ShmBufferFailed (metadata "op", "size") when the memory cannot be created.
func newShmBuffer(shm *wl.WlShm, width, height int32, color uint32, onRelease func(*shmBuffer)) (b *shmBuffer, err error) {
	return newShmBufferFormat(shm, width, height, wl.WlShmFormatXrgb8888, func(data []byte) { fill(data, color) }, onRelease)
}

// newShmBufferFormat is newShmBuffer with the pixel format and a function that
// writes the pixels (4 bytes each, little endian) into the memory of the buffer.
func newShmBufferFormat(shm *wl.WlShm, width, height int32, format wl.WlShmFormat, paint func(data []byte), onRelease func(*shmBuffer)) (b *shmBuffer, err error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New(tagShmBufferFailed, "a buffer needs a positive size", "op", "size", "width", width, "height", height)
	}
	stride := width * 4
	size := int(stride) * int(height)
	if size > maxShmBytes {
		return nil, errors.New(tagShmBufferFailed, "the buffer is too large", "op", "size", "width", width, "height", height)
	}

	fd, err := memfdCreate("glutter-shm")
	if err != nil {
		return nil, err
	}
	defer func() {
		// The pool has its own copy of the descriptor, so ours is closed on every
		// path. A failure to close is reported unless something failed earlier.
		if cerr := syscall.Close(fd); cerr != nil && err == nil {
			b.free()
			b, err = nil, errors.NewErr(tagShmBufferFailed, cerr).SetMetadata("op", "close", "size", size)
		}
	}()
	if err := syscall.Ftruncate(fd, int64(size)); err != nil {
		return nil, errors.NewErr(tagShmBufferFailed, err).SetMetadata("op", "ftruncate", "size", size)
	}
	data, err := syscall.Mmap(fd, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, errors.NewErr(tagShmBufferFailed, err).SetMetadata("op", "mmap", "size", size)
	}
	paint(data)

	pool := shm.CreatePool(fd, int32(size))
	buffer := pool.CreateBuffer(0, width, height, stride, format)
	pool.Destroy() // the buffer keeps the memory alive
	b = &shmBuffer{buffer: buffer, data: data, width: width, height: height}
	buffer.SetEvents(&wl.WlBufferEvents{OnRelease: func() {
		b.released = true
		if onRelease != nil {
			onRelease(b)
		}
	}})
	return b, nil
}

// fill sets every pixel of data to color, as little-endian 32-bit values.
func fill(data []byte, color uint32) {
	if len(data) < 4 {
		return
	}
	px := unsafe.Slice((*uint32)(unsafe.Pointer(&data[0])), len(data)/4)
	for i := range px {
		px[i] = color
	}
}

// free destroys the wl_buffer and unmaps the memory. It is safe to call twice.
func (b *shmBuffer) free() {
	if b.freed {
		return
	}
	b.freed = true
	b.buffer.Destroy()
	_ = syscall.Munmap(b.data) // nothing to do if it fails: the mapping is gone with the process
	b.data = nil
}

// memfdCreate makes an anonymous in-memory file. The syscall package has the
// number but no wrapper.
func memfdCreate(name string) (int, error) {
	p, err := syscall.BytePtrFromString(name)
	if err != nil {
		return -1, errors.NewErr(tagShmBufferFailed, err).SetMetadata("op", "memfd_create", "name", name)
	}
	if sysMemfdCreate == 0 {
		return -1, errors.NewErr(tagShmBufferFailed, syscall.ENOSYS).SetMetadata("op", "memfd_create", "name", name)
	}
	fd, _, errno := syscall.Syscall(sysMemfdCreate, uintptr(unsafe.Pointer(p)), mfdCloexec, 0)
	if errno != 0 {
		return -1, errors.NewErr(tagShmBufferFailed, errno).SetMetadata("op", "memfd_create", "name", name)
	}
	return int(fd), nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagShmBufferFailed = "ShmBufferFailed"
)
