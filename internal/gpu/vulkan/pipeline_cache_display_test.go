//go:build linux && glutter_display

package vulkan

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// mapCache is a PipelineCache in memory.
type mapCache struct {
	data          map[string][]byte
	loads, stores int
}

func (c *mapCache) Load(key string) ([]byte, bool) {
	c.loads++
	v, ok := c.data[key]
	return v, ok
}

func (c *mapCache) Store(key string, data []byte) {
	if c.data == nil {
		c.data = map[string][]byte{}
	}
	c.stores++
	c.data[key] = data
}

// TestPipelineCacheSurvivesAClose uses the first Vulkan device of the machine, offscreen (no window is
// opened), because lavapipe keeps no pipeline cache data worth a file: it skips when that device is a CPU or
// there is none.
func TestPipelineCacheSurvivesAClose(t *testing.T) {
	open := func(store PipelineCache) *Device {
		t.Helper()
		d, err := Open(Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), PipelineCache: store})
		if err != nil {
			t.Skipf("there is no Vulkan GPU: %v", err)
		}
		return d
	}
	makePipeline := func(d *Device) {
		t.Helper()
		if _, err := d.NewPipeline(explicit.PipelineDesc{Shader: explicit.ShaderBlit, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: explicit.TextureFormatBGRA8Unorm}); err != nil {
			t.Fatal(err)
		}
	}
	closeChecked := func(d *Device) {
		t.Helper()
		if errs := d.ValidationErrors(); len(errs) > 0 {
			t.Errorf("the validation layer reported %d errors:\n%s", len(errs), strings.Join(errs, "\n"))
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}

	store := &mapCache{}
	first := open(store)
	makePipeline(first)
	if first.info.Type == deviceTypeCPU {
		_ = first.Close()
		t.Skip("the device is a CPU")
	}
	if rep := first.StartupReport(); rep.Built != 1 || rep.CacheBytes != 0 || rep.Shaders <= 0 || rep.Context <= 0 {
		t.Errorf("first report = %+v", rep)
	}
	closeChecked(first) // saves the cache
	if store.stores != 1 || len(store.data) != 1 {
		t.Fatalf("the first run stored %d times, %d entries", store.stores, len(store.data))
	}
	var key string
	var saved []byte
	for k, v := range store.data {
		key, saved = k, v
	}

	second := open(store)
	if rep := second.StartupReport(); rep.CacheBytes != len(saved) {
		t.Errorf("the second run loaded %d bytes, want the %d that were stored", rep.CacheBytes, len(saved))
	}
	makePipeline(second)
	closeChecked(second) // nothing new: the cache is not stored again
	if store.stores != 1 {
		t.Errorf("the second run stored again (%d stores)", store.stores)
	}

	// The cache of another device or driver is not given to this one, and the pipeline is made as before.
	store.data[key] = append([]byte(nil), saved...)
	store.data[key][8] ^= 0xFF // the vendor
	third := open(store)
	if rep := third.StartupReport(); rep.CacheBytes != 0 {
		t.Errorf("the third run loaded %d bytes of a cache that is not its own", rep.CacheBytes)
	}
	makePipeline(third)
	closeChecked(third)
	if store.stores != 2 {
		t.Errorf("the third run did not replace the cache of the other device (%d stores)", store.stores)
	}
}

func TestOpenWithoutAPipelineCache(t *testing.T) {
	d := openLavapipe(t)
	if _, err := d.NewPipeline(explicit.PipelineDesc{Shader: explicit.ShaderBlit, VertexEntry: "vs_main", FragmentEntry: "fs_main", Target: explicit.TextureFormatBGRA8Unorm}); err != nil {
		t.Fatal(err)
	}
	if rep := d.StartupReport(); rep.Built != 1 || rep.CacheBytes != 0 {
		t.Errorf("report = %+v", rep)
	}
	d.SavePipelineCache() // nothing to save to
}
