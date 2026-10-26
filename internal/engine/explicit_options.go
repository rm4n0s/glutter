package engine

import (
	"log/slog"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/ui"
)

// rasterCacheAfter is how many frames in a row a retained layer must be in the scene
// to be drawn from a texture: three, as Flutter's raster cache does.
const rasterCacheAfter = 3

// explicitOptions are the options that the engine gives the explicit renderer over a device (SPEC §9.3):
// the background of the development color, the log, the damage that is told to the compositor (off with
// Config.DisableDamage or GLUTTER_DAMAGE=0), the commands the renderer may leave out
// (Config.SkipUnsupported) and the raster cache (off with Config.DisableRasterCache or
// GLUTTER_RASTER_CACHE=0). The Direct3D 12 renderer of Windows is made with them.
func explicitOptions(cfg *Config, getenv func(string) string, log *slog.Logger) explicit.RendererOpts {
	opts := explicit.RendererOpts{
		Background: ui.NewColor(developmentColor), HasBackground: true, Log: log,
		Damage: !cfg.DisableDamage && getenv(DamageEnv) != "0", SkipUnsupported: cfg.SkipUnsupported,
	}
	if !cfg.DisableRasterCache && getenv(RasterCacheEnv) != "0" {
		opts.RasterCacheAfter = rasterCacheAfter
	}
	return opts
}
