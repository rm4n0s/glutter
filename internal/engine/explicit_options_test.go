package engine

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestExplicitOptions(t *testing.T) {
	tests := []struct {
		name            string
		cfg             Config
		env             map[string]string
		damage, skip    bool
		rasterCacheOnOf bool
	}{
		{"the defaults", Config{}, nil, true, false, true},
		{"GLUTTER_DAMAGE=0", Config{}, map[string]string{"GLUTTER_DAMAGE": "0"}, false, false, true},
		{"GLUTTER_DAMAGE=1 is on", Config{}, map[string]string{"GLUTTER_DAMAGE": "1"}, true, false, true},
		{"Config.DisableDamage", Config{DisableDamage: true}, nil, false, false, true},
		{"the environment does not turn damage on over the config", Config{DisableDamage: true}, map[string]string{"GLUTTER_DAMAGE": "1"}, false, false, true},
		{"GLUTTER_RASTER_CACHE=0", Config{}, map[string]string{"GLUTTER_RASTER_CACHE": "0"}, true, false, false},
		{"Config.DisableRasterCache", Config{DisableRasterCache: true}, nil, true, false, false},
		{"Config.SkipUnsupported", Config{SkipUnsupported: true}, nil, true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, _ := recordingLog()
			opts := explicitOptions(&tt.cfg, func(k string) string { return tt.env[k] }, log)
			if opts.Damage != tt.damage || opts.SkipUnsupported != tt.skip || (opts.RasterCacheAfter > 0) != tt.rasterCacheOnOf {
				t.Errorf("damage %v, skip %v, raster cache after %d; want %v, %v, on %v", opts.Damage, opts.SkipUnsupported, opts.RasterCacheAfter, tt.damage, tt.skip, tt.rasterCacheOnOf)
			}
			if !opts.HasBackground || opts.Background != ui.NewColor(developmentColor) || opts.Log != log {
				t.Errorf("background %v (%v), log %v", opts.Background, opts.HasBackground, opts.Log)
			}
		})
	}
}
