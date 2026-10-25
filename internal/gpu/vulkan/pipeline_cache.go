//go:build linux

package vulkan

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// PipelineCache keeps the data of the Vulkan pipeline cache between runs: shadercache.Cache is one. A Store that
// fails is the cache's business (it reports it once); the device goes on (SPEC §9.7).
//
// Flutter: no counterpart
type PipelineCache interface {
	Load(key string) ([]byte, bool)
	Store(key string, data []byte)
}

// pipelineCacheHeaderSize is the size of VkPipelineCacheHeaderVersionOne: the size, the version, the vendor, the
// device and the UUID of the driver build.
const pipelineCacheHeaderSize = 16 + uuidSize

// pipelineCacheHeaderVersionOne is VK_PIPELINE_CACHE_HEADER_VERSION_ONE.
const pipelineCacheHeaderVersionOne = 1

// pipelineCacheKey is the key of the cache of a device in the PipelineCache: the vendor, the device and the UUID of
// the driver build, so that two GPUs of a machine, or a new driver, have a file of their own.
func pipelineCacheKey(info physicalInfo) string {
	return fmt.Sprintf("vulkan-pipeline-cache-%08x-%08x-%s", info.VendorID, info.DeviceID, hex.EncodeToString(info.PipelineCacheUUID[:]))
}

// validPipelineCacheHeader says whether the data is a pipeline cache that the device may be given: its header (the
// size, the version 1, the vendor, the device and the UUID) is the device's. The driver checks the data again, but
// not every driver does it well, and a cache of another GPU is of no use.
func validPipelineCacheHeader(data []byte, info physicalInfo) bool {
	if len(data) < pipelineCacheHeaderSize {
		return false
	}
	le := binary.LittleEndian
	return le.Uint32(data[0:]) >= pipelineCacheHeaderSize &&
		le.Uint32(data[4:]) == pipelineCacheHeaderVersionOne &&
		le.Uint32(data[8:]) == info.VendorID &&
		le.Uint32(data[12:]) == info.DeviceID &&
		[uuidSize]byte(data[16:16+uuidSize]) == info.PipelineCacheUUID
}

// pipelineCaching is what the device keeps of its pipeline cache.
type pipelineCaching struct {
	store  PipelineCache
	key    string
	handle uint64
	// loaded is the size of the data the cache was made from, 0 for an empty cache (nothing was stored, or it did
	// not match the device).
	loaded int
	// pipelines and took count the pipelines made, and the time vkCreateGraphicsPipelines took.
	pipelines int
	took      time.Duration
	// opened is how long Open took.
	opened time.Duration
	// saved is the size of the data last given to the store.
	saved int
}

// initPipelineCache makes the VkPipelineCache of the device, from the data of the store if it matches the device.
// A cache that cannot be made leaves the device without one: pipelines are then compiled each run.
func (d *Device) initPipelineCache(store PipelineCache) {
	pc := &d.pcache
	pc.store = store
	var initial []byte
	if store != nil {
		pc.key = pipelineCacheKey(d.info)
		if data, ok := store.Load(pc.key); ok && validPipelineCacheHeader(data, d.info) {
			initial = data
		}
	}
	if store == nil {
		return
	}
	a := d.arena
	defer a.rewind(a.mark())
	info := pipelineCacheCreateInfo{sType: sTypePipelineCacheCreateInfo}
	if len(initial) > 0 {
		data := a.alloc(uintptr(len(initial)), 8)
		copy(dl.Bytes(data, len(initial)), initial)
		info.initialDataSize, info.pInitialData = uintptr(len(initial)), data
	}
	out := a.out()
	if r := d.f.createPipelineCache(d.dev, put(a, info), 0, out); r != resultSuccess {
		if len(initial) > 0 {
			// The driver did not take the data: an empty cache.
			info.initialDataSize, info.pInitialData = 0, 0
			out = a.out()
			if r = d.f.createPipelineCache(d.dev, put(a, info), 0, out); r != resultSuccess {
				return
			}
			initial = nil
		} else {
			return
		}
	}
	pc.handle = u64(out)
	pc.loaded = len(initial)
	pc.saved = len(initial)
}

// SavePipelineCache gives the data of the pipeline cache to the store, when it has grown since it was loaded or last
// saved. The renderer calls it after the first frame and the device at Close, so that a second start finds the
// pipelines that the first one needed.
//
// Flutter: no counterpart
func (d *Device) SavePipelineCache() {
	pc := &d.pcache
	if pc.handle == 0 || pc.store == nil || d.dev == 0 {
		return
	}
	a := d.arena
	defer a.rewind(a.mark())
	size := a.out()
	if r := d.f.getPipelineCacheData(d.dev, pc.handle, size, 0); r != resultSuccess || ptr(size) == 0 {
		return
	}
	n := int(ptr(size))
	if n == pc.saved || n < pipelineCacheHeaderSize {
		return // nothing new, or nothing worth keeping
	}
	data := a.alloc(uintptr(n), 8)
	if r := d.f.getPipelineCacheData(d.dev, pc.handle, size, data); (r != resultSuccess && r != resultIncomplete) || int(ptr(size)) > n {
		return
	}
	got := append([]byte(nil), dl.Bytes(data, int(ptr(size)))...)
	if !validPipelineCacheHeader(got, d.info) {
		// A driver that keeps no data (lavapipe gives a header that is cut short): nothing worth a file.
		return
	}
	pc.store.Store(pc.key, got)
	pc.saved = len(got)
}

// StartupReport says how the start went: how long Open took, and the pipelines made so far.
//
// Flutter: no counterpart
func (d *Device) StartupReport() gpu.StartupReport {
	pc := &d.pcache
	return gpu.StartupReport{Context: pc.opened, Shaders: pc.took, Built: pc.pipelines, CacheBytes: pc.loaded}
}
