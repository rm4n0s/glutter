//go:build linux && glutter_display

package vulkan

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/ui"
)

// These tests draw scenes through the explicit renderer on Vulkan (lavapipe, validation on)
// and compare them with the software renderer, as the tests of gles do. The renderer needs a
// window to make its swapchain, so they run under the headless compositor:
//
//	scripts/headless-wayland.sh -- go test -tags glutter_display ./internal/gpu/vulkan/

// drawJob is a scene to draw at a size, and what came of it.
type drawJob struct {
	name          string
	scene         *ui.Scene
	width, height int

	got   *image.RGBA
	stats explicit.FrameStats
	err   error
}

// jobHandler draws its jobs at the first frame of the window, in order.
type jobHandler struct {
	platform.NopHandler
	r    *explicit.Renderer
	win  platform.Window
	jobs []*drawJob

	mu   sync.Mutex
	done bool
}

func (h *jobHandler) OnFrame(ui.ViewID, time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return
	}
	h.done = true
	for _, j := range h.jobs {
		j.got, j.err = h.r.Draw(h.win.NativeSurface(), j.width, j.height, j.scene, true)
		j.stats = h.r.LastFrame()
	}
}

func (h *jobHandler) finished() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.done
}

// drawOnWindow draws the jobs through a renderer with SkipUnsupported, in a Wayland window,
// and checks that the validation layer said nothing.
func drawOnWindow(t *testing.T, jobs []*drawJob) {
	t.Helper()
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("there is no Wayland compositor (run under scripts/headless-wayland.sh)")
	}
	p, err := wayland.New("")
	if err != nil {
		t.Fatalf("could not connect to the compositor: %v", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	r, dev := newWindowRenderer(t, func(o *explicit.RendererOpts) {
		o.Background = ui.NewColor(0xFFFFFFFF)
		o.SkipUnsupported = true
	})
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "vulkan shapes test"
	cfg.Size = ui.Size{Width: 128, Height: 128}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := &jobHandler{r: r, win: win, jobs: jobs}
	stop := runPlatform(t, p, h)
	waitUntil(t, h.finished, "the scenes were not drawn")
	stop()
	for _, j := range jobs {
		if j.err != nil {
			t.Fatalf("%s: %v", j.name, j.err)
		}
	}
	checkValidation(t, r, dev)
}

// dumpImage saves an image into the directory named by GLUTTER_TEST_DUMP, when it is set,
// so that a failing comparison can be looked at.
func dumpImage(t *testing.T, name string, img image.Image) {
	t.Helper()
	dir := os.Getenv("GLUTTER_TEST_DUMP")
	if dir == "" {
		return
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Errorf("dump: %v", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Errorf("dump: %v", err)
	}
}

func TestVulkanShapeScenesMatchSoftware(t *testing.T) {
	lavapipe(t)
	scenes := gputest.ShapeScenes()
	jobs := make([]*drawJob, len(scenes))
	for i, sc := range scenes {
		jobs[i] = &drawJob{name: sc.Name, scene: gputest.CanvasScene(sc.Draw), width: gputest.ShapeSize, height: gputest.ShapeSize}
	}
	drawOnWindow(t, jobs)
	for i, sc := range scenes {
		j := jobs[i]
		t.Run(sc.Name, func(t *testing.T) {
			want, err := soft.Render(j.scene, j.width, j.height)
			if err != nil {
				t.Fatal(err)
			}
			slug := strings.ReplaceAll(sc.Name, " ", "-")
			dumpImage(t, "vk-"+slug+"-gpu", j.got)
			dumpImage(t, "vk-"+slug+"-soft", want)
			if j.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
			}
			sc.Tolerance().Check(t, sc.Name, gputest.DiffRect(j.got, want, j.got.Bounds()))
		})
	}
}

// cellNames are the cells of the shapes-clips scene, by grid index.
var cellNames = []string{
	"rect", "rrect", "superellipse", "oval", "circle stroke", "drrect", "arc", "lines and points",
	"vertices", "nine-patch", "image quality", "clipped image",
}

// proofTolerances is how far Vulkan may be from the software renderer in each proof scene it
// draws (the same as gles has for them); shapes-clips is compared by cell.
var proofTolerances = map[string]gputest.Tolerance{
	"shapes-basic": {Over8Permille: 20, Over32Permille: 5},
	"text-tour":    {Over8Permille: 1},
}

func TestVulkanProofScenesMatchSoftware(t *testing.T) {
	lavapipe(t)
	names := []string{"shapes-basic", "shapes-clips", "text-tour"}
	var jobs []*drawJob
	for _, name := range names {
		sc, err := proof.Lookup(name)
		if err != nil {
			t.Fatal(err)
		}
		jobs = append(jobs, &drawJob{name: name, scene: sc.Build(), width: sc.Width, height: sc.Height})
	}
	drawOnWindow(t, jobs)
	for _, j := range jobs {
		t.Run(j.name, func(t *testing.T) {
			want, err := soft.Render(j.scene, j.width, j.height)
			if err != nil {
				t.Fatal(err)
			}
			dumpImage(t, "vk-"+j.name+"-gpu", j.got)
			dumpImage(t, "vk-"+j.name+"-soft", want)
			t.Logf("frame: %+v", j.stats)
			if j.name == "text-tour" {
				// The color emoji are painted with gradients and the shadow is a blur: steps 20.5
				// and 20.6. Everything else is compared, with those places taken from the
				// software image.
				if want := []string{"mask filter", "shader"}; !reflect.DeepEqual(j.stats.Unsupported, want) {
					t.Errorf("unsupported %v, want %v", j.stats.Unsupported, want)
				}
				for _, r := range []image.Rectangle{image.Rect(325, 165, 640, 205), image.Rect(10, 285, 135, 330)} {
					draw.Draw(j.got, r, want, r.Min, draw.Src)
				}
			} else if j.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
			}
			if j.name == "shapes-clips" {
				for i, name := range cellNames {
					cell := image.Rect(i%4*160, i/4*160, i%4*160+160, i/4*160+160)
					gputest.Tolerance{Over8Permille: 30, Over32Permille: 3}.Check(t, name, gputest.DiffRect(j.got, want, cell))
				}
				return
			}
			proofTolerances[j.name].Check(t, j.name, gputest.DiffRect(j.got, want, j.got.Bounds()))
		})
	}
}
