package explicit_test

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
)

// cachingDevice is a fake device that, like Vulkan's, keeps a pipeline cache.
type cachingDevice struct {
	*fakedevice.Device
	saves  int
	report gpu.StartupReport
}

func (d *cachingDevice) SavePipelineCache()               { d.saves++ }
func (d *cachingDevice) StartupReport() gpu.StartupReport { return d.report }

func TestRendererSavesThePipelineCacheAfterTheFirstFrame(t *testing.T) {
	tests := []struct {
		name      string
		frames    int
		wantSaves int
	}{
		{"one frame", 1, 1},
		{"three frames still save once", 3, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev := &cachingDevice{Device: fakedevice.New(), report: gpu.StartupReport{Built: 4, CacheBytes: 100}}
			dev.CompleteReadsOnSubmit = true
			r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { return dev, nil })
			if rep := r.StartupReport(); rep != (gpu.StartupReport{}) {
				t.Errorf("a report before the device is open: %+v", rep)
			}
			for range tt.frames {
				if _, err := r.Draw(window, 40, 30, nil, false); err != nil {
					t.Fatal(err)
				}
			}
			if dev.saves != tt.wantSaves {
				t.Errorf("saves = %d, want %d", dev.saves, tt.wantSaves)
			}
			if rep := r.StartupReport(); rep != dev.report {
				t.Errorf("report = %+v, want the device's %+v", rep, dev.report)
			}
		})
	}
}

func TestRendererWithADeviceThatHasNoPipelineCache(t *testing.T) {
	dev := fakedevice.New()
	dev.CompleteReadsOnSubmit = true
	r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { return dev, nil })
	if _, err := r.Draw(window, 40, 30, nil, false); err != nil {
		t.Fatal(err)
	}
	if rep := r.StartupReport(); rep != (gpu.StartupReport{}) {
		t.Errorf("report = %+v, want none", rep)
	}
}
