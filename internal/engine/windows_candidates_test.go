package engine

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// windowsFakes are the three renderers of Windows' list as fakes: d3d12 on hardware, d3d12 on WARP and
// soft. A renderer named in failing fails its first frame with the error in errs (its own tag when
// none is given). made lists the renderers that were made, by their letters, in order: h, w and s.
func windowsFakes(failing map[string]string) (cands []rendererCandidate, made *[]string) {
	made = &[]string{}
	fake := func(letter string) func(*Config, *slog.Logger) renderer {
		return func(*Config, *slog.Logger) renderer {
			*made = append(*made, letter)
			r := &namedRenderer{fakeRenderer: &fakeRenderer{}, name: letter + ": device"}
			if tag, ok := failing[letter]; ok {
				r.err = errors.New(tag, "this renderer does not work")
			}
			return r
		}
	}
	return windowsCandidates(fake("h"), fake("w"), fake("s")), made
}

func TestWindowsCandidatesAreHardwareWarpSoft(t *testing.T) {
	cands, _ := windowsFakes(nil)
	var names, labels []string
	for _, c := range cands {
		names = append(names, c.Name)
		labels = append(labels, c.label())
	}
	if got := strings.Join(names, ","); got != "d3d12,d3d12,soft" {
		t.Errorf("names %q, want d3d12,d3d12,soft (SPEC §9.6)", got)
	}
	if got := strings.Join(labels, ","); got != "d3d12,d3d12 (warp),soft" {
		t.Errorf("labels %q, want d3d12,d3d12 (warp),soft", got)
	}
}

func TestWindowsAdapterEnvIsThePackagesOwn(t *testing.T) {
	// The engine's constant must be the variable that package d3d12 reads (SPEC §9.8); the Windows file
	// compares them with the package itself (backends_windows_test.go).
	if d3d12AdapterEnv != "GLUTTER_D3D12_ADAPTER" {
		t.Errorf("d3d12AdapterEnv = %q", d3d12AdapterEnv)
	}
}

func TestWindowsChooser(t *testing.T) {
	handle := gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Window: 1}
	tests := []struct {
		name string
		want string
		// failing maps a letter to the tag its first frame fails with.
		failing map[string]string
		adapter string // GLUTTER_D3D12_ADAPTER
		// wantUsed is the letter of the renderer that drew the frames, wantMade the letters made, wantTag the
		// tag of the error of Draw, wantLogged the renderers named in the "not usable" lines, with their tags.
		wantUsed   string
		wantMade   string
		wantTag    string
		wantLogged []string
	}{
		{name: "the hardware adapter works", want: "auto", wantUsed: "h", wantMade: "h"},
		{name: "an empty name is auto", want: "", wantUsed: "h", wantMade: "h"},
		{name: "the hardware adapter has no device: WARP", want: "auto", failing: map[string]string{"h": "D3D12NoAdapter"},
			wantUsed: "w", wantMade: "hw", wantLogged: []string{"renderer=d3d12 err.Tag=D3D12NoAdapter"}},
		{name: "the device cannot make level 11_0: WARP", want: "auto", failing: map[string]string{"h": "D3D12Unsupported"},
			wantUsed: "w", wantMade: "hw", wantLogged: []string{"tag=D3D12Unsupported"}},
		{name: "a library that does not load: WARP", want: "auto", failing: map[string]string{"h": "D3D12LoadFailed"},
			wantUsed: "w", wantMade: "hw", wantLogged: []string{"tag=D3D12LoadFailed"}},
		{name: "a swap chain that cannot be made: WARP", want: "auto", failing: map[string]string{"h": "D3D12SwapchainFailed"},
			wantUsed: "w", wantMade: "hw", wantLogged: []string{"tag=D3D12SwapchainFailed"}},
		{name: "a shader that does not compile: WARP", want: "auto", failing: map[string]string{"h": "D3D12ShaderCompileFailed"},
			wantUsed: "w", wantMade: "hw", wantLogged: []string{"tag=D3D12ShaderCompileFailed"}},
		{name: "a device lost at the first frame: WARP", want: "auto", failing: map[string]string{"h": "GPUDeviceLost"},
			wantUsed: "w", wantMade: "hw", wantLogged: []string{"tag=GPUDeviceLost"}},
		{name: "no memory at the first frame: WARP", want: "auto", failing: map[string]string{"h": "GPUOutOfMemory"},
			wantUsed: "w", wantMade: "hw", wantLogged: []string{"tag=GPUOutOfMemory"}},
		{name: "WARP missing too: soft", want: "auto", failing: map[string]string{"h": "D3D12NoAdapter", "w": "D3D12NoAdapter"},
			wantUsed: "s", wantMade: "hws", wantLogged: []string{"renderer=d3d12 ", "renderer=\"d3d12 (warp)\""}},
		{name: "every renderer fails but soft", want: "auto", failing: map[string]string{"h": "D3D12Unsupported", "w": "D3D12CallFailed"},
			wantUsed: "s", wantMade: "hws", wantLogged: []string{"tag=D3D12Unsupported", "tag=D3D12CallFailed"}},
		{name: "every renderer fails", want: "auto", failing: map[string]string{"h": "D3D12NoAdapter", "w": "D3D12NoAdapter", "s": "GDIPresentFailed"},
			wantMade: "hws", wantTag: "GDIPresentFailed", wantLogged: []string{"renderer=d3d12 ", "renderer=\"d3d12 (warp)\"", "tag=GDIPresentFailed"}},
		{name: "an adapter that is named leaves WARP out", want: "auto", adapter: "GeForce", failing: map[string]string{"h": "D3D12NoAdapter"},
			wantUsed: "s", wantMade: "hs", wantLogged: []string{"tag=D3D12NoAdapter"}},
		{name: "a number leaves WARP out too", want: "auto", adapter: "1", failing: map[string]string{"h": "D3D12NoAdapter"},
			wantUsed: "s", wantMade: "hs", wantLogged: []string{"tag=D3D12NoAdapter"}},
		{name: "auto in the variable does not", want: "auto", adapter: "auto", failing: map[string]string{"h": "D3D12NoAdapter"},
			wantUsed: "w", wantMade: "hw", wantLogged: []string{"tag=D3D12NoAdapter"}},
		{name: "a blank variable does not", want: "auto", adapter: "  ", failing: map[string]string{"h": "D3D12NoAdapter"},
			wantUsed: "w", wantMade: "hw", wantLogged: []string{"tag=D3D12NoAdapter"}},
		{name: "d3d12 by name is the hardware renderer", want: "d3d12", wantUsed: "h", wantMade: "h"},
		{name: "d3d12 by name never falls back to WARP", want: "d3d12", failing: map[string]string{"h": "D3D12NoAdapter"},
			wantMade: "h", wantTag: "D3D12NoAdapter"},
		{name: "d3d12 by name with an adapter that is named never falls back", want: "d3d12", adapter: "nothing", failing: map[string]string{"h": "D3D12NoAdapter"},
			wantMade: "h", wantTag: "D3D12NoAdapter"},
		{name: "soft by name", want: "soft", failing: map[string]string{"h": "D3D12NoAdapter"}, wantUsed: "s", wantMade: "s"},
		{name: "soft by name never falls back", want: "soft", failing: map[string]string{"s": "GDIPresentFailed"}, wantMade: "s", wantTag: "GDIPresentFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cands, made := windowsFakes(tt.failing)
			log, buf := recordingLog()
			cfg := &Config{Getenv: func(k string) string {
				if k == "GLUTTER_D3D12_ADAPTER" {
					return tt.adapter
				}
				return ""
			}}
			c, err := newChooser(cfg, log, tt.want, cands)
			if err != nil {
				t.Fatalf("newChooser: %v", err)
			}
			for frame := 0; frame < 3; frame++ { // the choice is made once
				_, err = c.Draw(handle, 10, 10, nil, false)
				if tt.wantTag != "" {
					if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tt.wantTag {
						t.Fatalf("frame %d: Draw = %v, want the tag %s", frame, err, tt.wantTag)
					}
					break
				}
				if err != nil {
					t.Fatalf("frame %d: %v", frame, err)
				}
			}
			if got := strings.Join(*made, ""); got != tt.wantMade {
				t.Errorf("made %q, want %q", got, tt.wantMade)
			}
			if tt.wantUsed != "" {
				if got := c.Describe(); !strings.HasPrefix(got, tt.wantUsed+":") {
					t.Errorf("drawn by %q, want the renderer %s", got, tt.wantUsed)
				}
			}
			lines := nonUsableLines(buf.String())
			if len(lines) != len(tt.wantLogged) {
				t.Fatalf("the log has %d failed tries, want %d:\n%s", len(lines), len(tt.wantLogged), buf)
			}
			for i, want := range tt.wantLogged {
				if !strings.Contains(lines[i], want) {
					t.Errorf("failed try %d is %q, want it to contain %q", i, lines[i], want)
				}
			}
		})
	}
}

// nonUsableLines are the lines of the log that say a renderer was given up at its first frame.
func nonUsableLines(log string) []string {
	var out []string
	for _, l := range strings.Split(log, "\n") {
		if strings.Contains(l, "renderer not usable") {
			out = append(out, l)
		}
	}
	return out
}

func TestWindowsRendererNamesOtherThanTheirOwnAreUnavailable(t *testing.T) {
	tests := []struct {
		name string
		want string
		tag  string
	}{
		{"gles is not on Windows", "gles", "RendererUnavailable"},
		{"vulkan is not on Windows", "vulkan", "RendererUnavailable"},
		{"webgpu is not on Windows", "webgpu", "RendererUnavailable"},
		{"webgl is not on Windows", "webgl", "RendererUnavailable"},
		{"a name nobody has", "metal", "RendererUnknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cands, made := windowsFakes(nil)
			_, err := newChooser(&Config{}, nil, tt.want, cands)
			assertRoute(t, err, tt.tag, "newChooser."+tt.tag)
			if len(*made) != 0 {
				t.Errorf("%d renderers were made before the names were checked", len(*made))
			}
		})
	}
}

// A candidate that is left out is not tried and not logged; when every candidate is left out the chooser says so.
func TestChooserSkipsCandidatesAndSaysWhenNoneIsLeft(t *testing.T) {
	handle := gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Window: 1}
	skip := func(*Config) bool { return true }
	cand := func(*Config, *slog.Logger) renderer { t.Error("a skipped candidate was made"); return &fakeRenderer{} }
	c, err := newChooser(&Config{}, nil, "auto", []rendererCandidate{{Name: "d3d12", New: cand, Skip: skip}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Draw(handle, 10, 10, nil, false)
	assertRoute(t, err, "RendererUnavailable", "chooser.Draw->chooser.try.RendererUnavailable")
}
