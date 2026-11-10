//go:build darwin

package metal

import (
	"log/slog"
	"os"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// DeviceEnvVar is the environment variable that picks the device: "lowpower" is the integrated GPU of a Mac that
// has two; anything else, or nothing, is the system default (SPEC §29.5).
//
// Flutter: no counterpart
const DeviceEnvVar = "GLUTTER_METAL_DEVICE"

// Options are what Open takes.
//
// Flutter: no counterpart
type Options struct {
	// Logger gets the log of the backend; slog.Default() when nil.
	Logger *slog.Logger
	// Getenv reads the environment (GLUTTER_METAL_DEVICE); os.Getenv when nil.
	Getenv func(string) string
	// ShaderCacheDir is the directory of the shader cache (SPEC §9.7), where the binary archive of the
	// pipelines is kept between runs; empty is no archive.
	ShaderCacheDir string
	// Readback makes the textures that a swap chain gives readable (CAMetalLayer.framebufferOnly is
	// false), which the self-capture of the engine needs (SPEC §29.5).
	Readback bool

	// noDevice makes Open find no device, as a Mac without one does: a test uses it.
	noDevice bool
}

// Device is a Metal device and its queue: an explicit.Device. It is used on the UI thread only, and all calls
// of one device are made from one thread.
//
// Flutter: no counterpart
type Device struct {
	opts   Options
	log    *slog.Logger
	dev    objc.ID
	queue  objc.ID
	name   string
	unif   bool // the device has unified memory
	limits explicit.Limits
	feats  explicit.Features
	lowPow bool

	libs    map[*explicit.Shader]*library
	archive *archive

	// frames: the frame being recorded, the serials, the frames the GPU has not finished, and the writes
	// that wait for a command buffer.
	open      *encoder
	serial    uint64
	completed uint64
	inflight  []*frameRec
	pending   []pendingWrite
	// staging are the chunks that hold the data of the writes since the last Submit; free are the chunks of
	// finished frames, ready for reuse.
	staging []*stageChunk
	free    []*stageChunk

	// defSampler is the sampler of a binding that a group leaves out, made at the first need.
	defSampler objc.ID

	lost   error
	closed bool
	stats  Stats
}

// Stats counts what the device did, for the tests: the pipelines that Metal compiled, the ones the binary
// archive had, and the command buffers.
//
// Flutter: no counterpart
type Stats struct {
	PipelinesCompiled  int
	PipelinesFromCache int
	Libraries          int
	Frames             int
}

var _ explicit.Device = (*Device)(nil)

// Open makes the device: the default Metal device, or the low-power one when GLUTTER_METAL_DEVICE says so,
// and its command queue. It fails with MetalDeviceUnavailable when the machine has no Metal device, which the
// engine's chooser answers with the next renderer (SPEC §29.5).
//
// Flutter: no counterpart
func Open(opts Options) (*Device, error) {
	f, err := metalFramework()
	if err != nil {
		return nil, err
	}
	getenv := opts.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	d := &Device{opts: opts, log: log, libs: map[*explicit.Shader]*library{}}
	objcx.WithPool(func() {
		if strings.EqualFold(strings.TrimSpace(getenv(DeviceEnvVar)), "lowpower") && f.copyAllDevices != nil {
			all := f.copyAllDevices()
			for i := 0; i < objcx.ArrayCount(all); i++ {
				if c := objcx.ArrayAt(all, i); sendBool(c, "isLowPower") {
					d.dev, d.lowPow = retain(c), true
					break
				}
			}
			release(all)
		}
		if d.dev == 0 && !opts.noDevice {
			d.dev = f.createSystemDefault()
		}
	})
	if d.dev == 0 {
		return nil, errors.New(TagMetalDeviceUnavailable, "this Mac has no Metal device")
	}
	d.name = objcx.String(call(d.dev, "name"))
	d.unif = sendBool(d.dev, "hasUnifiedMemory") || !managedStorageAvailable
	d.queue = call(d.dev, "newCommandQueue")
	if d.queue == 0 {
		release(d.dev)
		return nil, errors.New(TagMetalDeviceUnavailable, "the Metal device made no command queue", "device", d.name)
	}
	d.limits = explicit.DefaultLimits()
	d.feats = explicit.Features{Float32Filterable: sendBool(d.dev, "supports32BitFloatFiltering")}
	d.archive = openArchive(d)
	log.Debug("metal device", "name", d.name, "unifiedMemory", d.unif, "lowPower", d.lowPow)
	return d, nil
}

// Description implements what explicit.Renderer.Describe asks of a device: "metal: Apple M4 Pro", which the
// engine's log names the renderer by (SPEC §29.5).
//
// Flutter: no counterpart
func (d *Device) Description() string { return "metal: " + d.name }

// Name is the name of the device, such as "Apple M4 Pro" (the engine's startup log says it).
//
// Flutter: no counterpart
func (d *Device) Name() string { return d.name }

// Stats returns what the device did so far.
//
// Flutter: no counterpart
func (d *Device) Stats() Stats { return d.stats }

// Limits implements explicit.Device.
func (d *Device) Limits() explicit.Limits { return d.limits }

// Features implements explicit.Device.
func (d *Device) Features() explicit.Features { return d.feats }

// Err returns the error that lost the device, nil while it works.
//
// Flutter: no counterpart
func (d *Device) Err() error { return d.lost }

// Close implements explicit.Device: it waits for the GPU, saves the binary archive and releases everything.
func (d *Device) Close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	objcx.WithPool(func() {
		if d.open != nil {
			d.open.abandon()
		}
		d.waitAll()
		for _, w := range d.pending {
			w.release()
		}
		d.pending = nil
		d.archive.save()
		d.archive.close()
		for _, l := range d.libs {
			l.release()
		}
		d.libs = nil
		for _, c := range append(d.staging, d.free...) {
			c.release()
		}
		d.staging, d.free = nil, nil
		release(d.defSampler)
		release(d.queue)
		release(d.dev)
	})
	return nil
}

// markLost records the error that lost the device: later calls return it.
func (d *Device) markLost(err error) {
	if d.lost == nil {
		d.lost = err
		d.log.Error("the Metal device was lost", "err", err)
	}
}

// checkLost returns the error of a lost device.
func (d *Device) checkLost() error {
	if d.closed {
		return errors.New(tagGPUDeviceLost, "the Metal device is closed")
	}
	return d.lost
}

// stageChunkSize is the size of a chunk of the staging memory that the writes go through.
const stageChunkSize = 4 << 20

// stageAlign is the alignment of what is staged: the largest pixel size, which a copy to a texture wants.
const stageAlign = 16

// stageChunk is a buffer in memory that the CPU writes and the GPU reads: shared on unified memory, managed
// otherwise, with the ranges that were written (SPEC §29.5).
type stageChunk struct {
	d     *Device
	obj   objc.ID
	size  int
	used  int
	mem   []byte
	dirty dirtyRanges
}

// stage reserves n bytes in the staging memory of the frame and returns the chunk and the offset; the caller
// writes chunk.mem[off:off+n].
func (d *Device) stage(n int) (*stageChunk, int, error) {
	if n < 0 {
		n = 0
	}
	if len(d.staging) > 0 {
		c := d.staging[len(d.staging)-1]
		if off := alignUp(c.used, stageAlign); off+n <= c.size {
			c.used = off + n
			return c, off, nil
		}
	}
	c, err := d.takeChunk(n)
	if err != nil {
		return nil, 0, err
	}
	d.staging = append(d.staging, c)
	c.used = n
	return c, 0, nil
}

// takeChunk finds a free chunk that holds n bytes or makes one.
func (d *Device) takeChunk(n int) (*stageChunk, error) {
	for i, c := range d.free {
		if c.size >= n {
			d.free = append(d.free[:i], d.free[i+1:]...)
			c.used = 0
			return c, nil
		}
	}
	size := max(stageChunkSize, alignUp(n, 1<<16))
	storage := storageShared
	if !d.unif {
		storage = storageManaged
	}
	obj := call(d.dev, "newBufferWithLength:options:", uint64(size), resourceOptions(storage, true))
	if obj == 0 {
		return nil, errors.New(tagGPUOutOfMemory, "Metal made no staging buffer", "size", size)
	}
	addr := send[uintptr](obj, "contents")
	return &stageChunk{d: d, obj: obj, size: size, mem: dl.Bytes(addr, size)}, nil
}

// flush tells Metal about the ranges the CPU wrote of a managed chunk.
func (c *stageChunk) flush() {
	if c.d.unif {
		return
	}
	for _, s := range c.dirty.take() {
		c.obj.Send(sel("didModifyRange:"), nsRange{Location: uint64(s.off), Length: uint64(s.n)})
	}
}

func (c *stageChunk) release() { release(c.obj); c.obj, c.mem = 0, nil }

// recycle puts the chunks of a finished frame back in the free list; a big chunk is released, so that one
// large upload does not keep its memory.
func (d *Device) recycle(chunks []*stageChunk) {
	for _, c := range chunks {
		c.used = 0
		if c.size > 4*stageChunkSize || len(d.free) >= 8 {
			c.release()
			continue
		}
		d.free = append(d.free, c)
	}
}

// label names a Metal object for the debugger and the validation messages.
func label(obj objc.ID, kind string) {
	if obj != 0 {
		call(obj, "setLabel:", objcx.NewString(kind))
	}
}

// ptr is the address that a message takes for an out-parameter.
func ptr[T any](p *T) unsafe.Pointer { return unsafe.Pointer(p) }

// surfaceLayer returns the layer a surface handle names, or an error.
func surfaceLayer(h gpu.SurfaceHandle) (objc.ID, error) {
	if h.Kind != gpu.SurfaceKindMetalLayer || h.Window == 0 {
		return 0, errors.New(TagMetalSwapchainFailed, "the surface is not the layer of a view", "kind", string(h.Kind), "window", h.Window)
	}
	return objc.ID(h.Window), nil
}
