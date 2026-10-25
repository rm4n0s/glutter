package engine

import (
	"log/slog"
	"os"
	"strings"
)

// The list of the renderers of Windows and the rule for the adapter that names one, in plain Go so that
// the tests run on every host; backends_windows.go gives the constructors (SPEC §9.6, §25.5).

// d3d12AdapterEnv is the environment variable that picks the adapter of Direct3D 12
// (d3d12.AdapterEnvVar; the engine does not import the package here, which only Windows builds).
const d3d12AdapterEnv = "GLUTTER_D3D12_ADAPTER"

// windowsCandidates are the renderers of Windows in the order auto tries them (SPEC §9.6): d3d12 on a
// hardware adapter, d3d12 on WARP, then soft drawn into the window with GDI. The two Direct3D 12 candidates
// are named "d3d12": a renderer asked for by name is the first of them, which never falls back to WARP by
// itself (the user can ask for WARP with GLUTTER_D3D12_ADAPTER=warp). The WARP candidate is left out when
// that variable names an adapter, since the first candidate has tried it already.
func windowsCandidates(hardware, warp, soft func(cfg *Config, log *slog.Logger) renderer) []rendererCandidate {
	return []rendererCandidate{
		{Name: "d3d12", New: hardware},
		{Name: "d3d12", Label: "d3d12 (warp)", New: warp, Skip: d3d12AdapterNamed},
		{Name: "soft", New: soft},
	}
}

// d3d12AdapterNamed says whether GLUTTER_D3D12_ADAPTER names an adapter: it is not empty and not "auto".
func d3d12AdapterNamed(cfg *Config) bool {
	v := strings.TrimSpace(configGetenv(cfg)(d3d12AdapterEnv))
	return v != "" && !strings.EqualFold(v, "auto")
}

// configGetenv is the environment of the engine: Config.Getenv, or os.Getenv when it is nil.
func configGetenv(cfg *Config) func(string) string {
	if cfg.Getenv != nil {
		return cfg.Getenv
	}
	return os.Getenv
}
