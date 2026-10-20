package engine

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

func TestContentLostInvalidatesTheWindowsDamage(t *testing.T) {
	e := newWindowsEngine(t)
	e.size(0, 400, 300)
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}})
	e.h.OnFrame(0, time.Millisecond) // the first frame makes the renderer
	e.h.OnContentLost(0)
	e.h.OnContentLost(99) // a window that does not exist is ignored
	e.r.mu.Lock()
	got := append([]gpu.SurfaceHandle(nil), e.r.invalidated...)
	e.r.mu.Unlock()
	if len(got) != 1 || got[0] != handleOf(0) {
		t.Errorf("invalidated = %v, want [%v]", got, handleOf(0))
	}
}

func TestGPUOptionsFollowTheConfigAndTheEnvironment(t *testing.T) {
	tests := []struct {
		name          string
		cfg           Config
		env           map[string]string
		cache, damage bool
	}{
		{"both are on by default", Config{}, nil, true, true},
		{"the config turns the cache off", Config{DisableRasterCache: true}, nil, false, true},
		{"the config turns the damage off", Config{DisableDamage: true}, nil, true, false},
		{"the environment turns the cache off", Config{}, map[string]string{RasterCacheEnv: "0"}, false, true},
		{"the environment turns the damage off", Config{}, map[string]string{DamageEnv: "0"}, true, false},
		{"any other value keeps them on", Config{}, map[string]string{RasterCacheEnv: "1", DamageEnv: "yes"}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			cfg.Getenv = func(k string) string { return tt.env[k] }
			r := newGLESRenderer(&cfg, quiet()).(*glesRenderer)
			if r.cache != tt.cache || r.damage != tt.damage {
				t.Errorf("cache = %v, damage = %v, want %v and %v", r.cache, r.damage, tt.cache, tt.damage)
			}
		})
	}
}
