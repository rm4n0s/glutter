package gles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
	"github.com/rm4n0s/glutter/ui"
)

// memCache is a ProgramCache in memory that counts what it is asked.
type memCache struct {
	data          map[string][]byte
	loads, stores int
}

func newMemCache() *memCache { return &memCache{data: map[string][]byte{}} }

func (c *memCache) Load(key string) ([]byte, bool) {
	c.loads++
	v, ok := c.data[key]
	return v, ok
}

func (c *memCache) Store(key string, data []byte) {
	c.stores++
	c.data[key] = data
}

// deviceOn makes a device over a fake driver whose answers the test sets.
func deviceOn(t *testing.T, renderer string, binaryFormats int32, refuse bool, cache ProgramCache) (*Device, *fakeGL) {
	t.Helper()
	gl := &fakeGL{versionString: "OpenGL ES 3.2 Mesa 24.1.0", vendor: "Fake", renderer: renderer, binaryFormats: binaryFormats, refuseBinary: refuse}
	d, err := newDevice(gl)
	if err != nil {
		t.Fatal(err)
	}
	d.SetProgramCache(cache)
	return d, gl
}

func TestProgramCache(t *testing.T) {
	type run struct {
		renderer string
		formats  int32
		refuse   bool
		programs []string
		// What the run is expected to do.
		compiled, loaded, refused, stored int
	}
	tests := []struct {
		name string
		runs []run
	}{
		{"the first run compiles and stores, the second loads", []run{
			{"GPU A", 1, false, []string{"solid", "image"}, 2, 0, 0, 2},
			{"GPU A", 1, false, []string{"solid", "image"}, 0, 2, 0, 0},
		}},
		{"only the programs that are used are made", []run{
			{"GPU A", 1, false, []string{"solid"}, 1, 0, 0, 1},
			{"GPU A", 1, false, []string{"solid", "image"}, 1, 1, 0, 1},
		}},
		{"a binary that the driver refuses is compiled again and replaced", []run{
			{"GPU A", 1, false, []string{"solid"}, 1, 0, 0, 1},
			{"GPU A", 1, true, []string{"solid"}, 1, 0, 1, 1},
			{"GPU A", 1, false, []string{"solid"}, 0, 1, 0, 0},
		}},
		{"another driver does not read the binaries of the first", []run{
			{"GPU A", 1, false, []string{"solid"}, 1, 0, 0, 1},
			{"GPU B", 1, false, []string{"solid"}, 1, 0, 0, 1},
			{"GPU A", 1, false, []string{"solid"}, 0, 1, 0, 0},
		}},
		{"a driver with no binary formats compiles and never touches the cache", []run{
			{"GPU A", 0, false, []string{"solid", "image"}, 2, 0, 0, 0},
			{"GPU A", 0, false, []string{"solid", "image"}, 2, 0, 0, 0},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newMemCache()
			for i, r := range tt.runs {
				d, gl := deviceOn(t, r.renderer, r.formats, r.refuse, cache)
				for _, name := range r.programs {
					if _, err := d.Program(name); err != nil {
						t.Fatalf("run %d: Program(%q): %v", i, name, err)
					}
				}
				st := d.ProgramStats()
				if st.Compiled != r.compiled || st.Loaded != r.loaded || st.Refused != r.refused || st.Stored != r.stored {
					t.Errorf("run %d: stats = %+v, want compiled %d, loaded %d, refused %d, stored %d", i, st, r.compiled, r.loaded, r.refused, r.stored)
				}
				if got := gl.calls["compileShader"]; got != 2*r.compiled {
					t.Errorf("run %d: %d shaders compiled, want %d", i, got, 2*r.compiled)
				}
				if st.Compiled+st.Loaded != len(r.programs) || d.Stats().Programs != len(r.programs) {
					t.Errorf("run %d: %d programs, want %d", i, d.Stats().Programs, len(r.programs))
				}
				// A refused binary leaves no GL error behind for the frames.
				if gl.errOnce != 0 {
					t.Errorf("run %d: a GL error is left", i)
				}
				if r.formats == 0 && (cache.loads != 0 || cache.stores != 0) {
					t.Errorf("run %d: the cache was used by a driver with no binary formats", i)
				}
				if gl.liveCount("program") != len(r.programs) {
					t.Errorf("run %d: %d GL programs alive, want %d", i, gl.liveCount("program"), len(r.programs))
				}
			}
		})
	}
}

func TestProgramCacheKeyHasTheSources(t *testing.T) {
	cache := newMemCache()
	d, _ := deviceOn(t, "GPU A", 1, false, cache)
	if _, err := d.NewProgram("p", "#version 300 es\nvoid main(){}", "#version 300 es\nvoid main(){}"); err != nil {
		t.Fatal(err)
	}
	d2, _ := deviceOn(t, "GPU A", 1, false, cache)
	// The same name with a changed fragment shader is another program.
	if _, err := d2.NewProgram("p", "#version 300 es\nvoid main(){}", "#version 300 es\nvoid main(){ }"); err != nil {
		t.Fatal(err)
	}
	if st := d2.ProgramStats(); st.Compiled != 1 || st.Loaded != 0 {
		t.Errorf("a changed source read the old binary: %+v", st)
	}
	if len(cache.data) != 2 {
		t.Errorf("the cache has %d entries, want 2", len(cache.data))
	}
}

func TestProgramCacheEntryWithoutBinaryIsIgnored(t *testing.T) {
	cache := newMemCache()
	d, _ := deviceOn(t, "GPU A", 1, false, cache)
	if _, err := d.Program("solid"); err != nil {
		t.Fatal(err)
	}
	for k := range cache.data {
		cache.data[k] = []byte{1, 2} // a file cut short
	}
	d2, _ := deviceOn(t, "GPU A", 1, false, cache)
	if _, err := d2.Program("solid"); err != nil {
		t.Fatal(err)
	}
	if st := d2.ProgramStats(); st.Compiled != 1 || st.Loaded != 0 {
		t.Errorf("stats = %+v", st)
	}
}

func TestProgramCacheOnDisk(t *testing.T) {
	t.Run("the files are the cache", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "shaders")
		first, _ := deviceOn(t, "GPU A", 1, false, shadercache.New(dir, nil))
		if _, err := first.Program("solid"); err != nil {
			t.Fatal(err)
		}
		second, _ := deviceOn(t, "GPU A", 1, false, shadercache.New(dir, nil))
		if _, err := second.Program("solid"); err != nil {
			t.Fatal(err)
		}
		if st := second.ProgramStats(); st.Loaded != 1 || st.Compiled != 0 {
			t.Errorf("the second device: %+v", st)
		}
	})
	t.Run("a cache that cannot be written is reported once and ignored", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		var reports []error
		d, _ := deviceOn(t, "GPU A", 1, false, shadercache.New(filepath.Join(blocker, "shaders"), func(err error) { reports = append(reports, err) }))
		for _, name := range []string{"solid", "image"} {
			if _, err := d.Program(name); err != nil {
				t.Fatalf("Program(%q): %v", name, err)
			}
		}
		if st := d.ProgramStats(); st.Compiled != 2 {
			t.Errorf("stats = %+v", st)
		}
		if len(reports) != 1 {
			t.Fatalf("reports = %d, want 1", len(reports))
		}
		appErr, ok := errors.FromError(reports[0])
		if !ok || appErr.Tag != "ShaderCacheWriteFailed" {
			t.Errorf("report = %v", reports[0])
		}
	})
}

func TestStartupReport(t *testing.T) {
	cache := newMemCache()
	d, _ := deviceOn(t, "GPU A", 1, false, cache)
	if _, err := d.Program("solid"); err != nil {
		t.Fatal(err)
	}
	r := &Renderer{ctx: &Context{dev: d}, opened: 5}
	rep := r.StartupReport()
	if rep.Context != 5 || rep.Built != 1 || rep.Cached != 0 || rep.Refused != 0 || rep.Shaders <= 0 {
		t.Errorf("report = %+v", rep)
	}
}

// The programs of a frame are made when the frame needs them: an empty frame and a frame of one rectangle make a few
// of the programs, not all of them (SPEC §9.7).
func TestOnlyTheProgramsOfTheFrameAreMade(t *testing.T) {
	all, err := ShaderNames()
	if err != nil {
		t.Fatal(err)
	}
	cache := newMemCache()
	r, _, gl := newTestRenderer(RendererOpts{ProgramCache: cache})
	gl.binaryFormats = 1
	s, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Render(s, nil, ui.Size{Width: 40, Height: 30}, 1); err != nil {
		t.Fatal(err)
	}
	empty := r.StartupReport()
	if empty.Built >= len(all)/2 {
		t.Errorf("an empty frame built %d of the %d programs", empty.Built, len(all))
	}
	if r.StartupReport().Context <= 0 {
		t.Error("the report has no time for the context")
	}
	if len(cache.data) != empty.Built {
		t.Errorf("the cache has %d entries for %d programs built", len(cache.data), empty.Built)
	}
	if st := r.ctx.dev.ProgramStats(); st.Stored != empty.Built {
		t.Errorf("stored %d, built %d", st.Stored, empty.Built)
	}
}
