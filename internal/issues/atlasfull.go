package issues

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// The issue "A frame with many shadows stops the app (AtlasFull)" of Fishball's list of
// 2026-11-15, fixed in step 46.1 (Glutter 1.17.4): a frame whose shadows need more tiles of the
// coverage atlas at once than the pages the atlas keeps failed with AtlasFull, and an error of a
// frame ends RunApp. The reproduction is four frames of 1920 x 1080 pixels on one renderer: ten
// shadows of panels, the same again, one shadow, and the ten again.

// AtlasFullWidth and AtlasFullHeight are the size of the frames of the reproduction, in pixels.
const (
	AtlasFullWidth  = 1920
	AtlasFullHeight = 1080
)

// AtlasFullName is the name of the issue: what an app of a device test is asked for, and the
// prefix of the lines of its report.
const AtlasFullName = "atlas-full"

// ShadowsScene draws n shadows of panels that fill a surface of AtlasFullWidth x
// AtlasFullHeight pixels, each of a size of its own, so that no two share their tiles of the
// coverage atlas.
func ShadowsScene(n int) *ui.Scene {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	for i := 0; i < n; i++ {
		path := ui.NewPath()
		path.AddRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20+float64(i), 20+float64(i), 1800-float64(7*i), 1000-float64(5*i)), 12, 12))
		c.DrawShadow(path, ui.NewColor(0xFF000000), 6, false)
	}
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	return sb.Build()
}

// AtlasFullScene is one frame of the reproduction.
type AtlasFullScene struct {
	Name  string
	Scene *ui.Scene
}

// AtlasFullScenes returns the frames of the reproduction, in the order a renderer draws them:
// the frame of the issue, the same again (its tiles are kept), a small frame (the pages over
// what the atlas keeps are freed), and the first again (on pages made again).
func AtlasFullScenes() []AtlasFullScene {
	big, small := ShadowsScene(10), ShadowsScene(1)
	return []AtlasFullScene{
		{Name: "many shadows", Scene: big},
		{Name: "many shadows again", Scene: big},
		{Name: "one shadow", Scene: small},
		{Name: "many shadows on new pages", Scene: big},
	}
}

// AtlasFullStats is what a renderer says of a frame it drew (the fields of the same names of
// explicit.FrameStats and gles.FrameStats).
type AtlasFullStats struct {
	TilesRasterized int
	AtlasFull       int
	Skipped         int
	Unsupported     []string
}

// AtlasFullFrame is what came of one frame.
type AtlasFullFrame struct {
	Name string
	AtlasFullStats
	// Pixels is the SHA-256 of the frame's pixels, in hexadecimal.
	Pixels string
	// Shadow is how much a pixel just under the largest panel differs from a pixel of the
	// background in a corner of the frame, summed over the four channels: nothing where no
	// shadow was drawn, whatever the background is (white in a window, nothing off screen).
	Shadow int
}

// AtlasFullFrameOf makes the record of a frame from its pixels and the renderer's numbers.
func AtlasFullFrameOf(name string, img *image.RGBA, st AtlasFullStats) AtlasFullFrame {
	sum := sha256.Sum256(img.Pix)
	under, corner := img.RGBAAt(1000, 1024), img.RGBAAt(AtlasFullWidth-4, AtlasFullHeight-4)
	shadow := 0
	for _, d := range []int{int(under.R) - int(corner.R), int(under.G) - int(corner.G), int(under.B) - int(corner.B), int(under.A) - int(corner.A)} {
		if d < 0 {
			d = -d
		}
		shadow += d
	}
	return AtlasFullFrame{Name: name, AtlasFullStats: st, Pixels: hex.EncodeToString(sum[:]), Shadow: shadow}
}

// AtlasFullReport is what came of the frames of the reproduction on one renderer.
type AtlasFullReport struct {
	Frames []AtlasFullFrame
}

// AtlasFullDraw draws a scene of the given size on the renderer under test and returns the
// pixels of the frame and the renderer's numbers of it.
type AtlasFullDraw func(scene *ui.Scene, width, height int) (*image.RGBA, AtlasFullStats, error)

// RunAtlasFull draws the frames of the reproduction and returns the report and the pixels of
// the first frame. An error of a frame, which is the issue itself on the code before the fix
// (AtlasFull), ends the run: the report then has the frames before it.
func RunAtlasFull(draw AtlasFullDraw) (AtlasFullReport, *image.RGBA, error) {
	var (
		rep   AtlasFullReport
		first *image.RGBA
	)
	for i, sc := range AtlasFullScenes() {
		img, st, err := draw(sc.Scene, AtlasFullWidth, AtlasFullHeight)
		if err != nil {
			return rep, first, err
		}
		if img == nil || img.Bounds().Dx() != AtlasFullWidth || img.Bounds().Dy() != AtlasFullHeight {
			return rep, first, errors.New(tagAtlasFullNoFrame, "the renderer gave no frame of the size asked for", "frame", sc.Name)
		}
		if i == 0 {
			first = img
		}
		rep.Frames = append(rep.Frames, AtlasFullFrameOf(sc.Name, img, st))
	}
	return rep, first, nil
}

// Problems checks the report and returns what is wrong, one line for each finding; none when
// the renderer drew the frame of the issue whole, kept its tiles for the same frame, freed the
// pages over what the atlas keeps after the small frame, and drew the first frame again with
// the pixels it had.
func (r AtlasFullReport) Problems() []string {
	var out []string
	scenes := AtlasFullScenes()
	if len(r.Frames) != len(scenes) {
		return []string{fmt.Sprintf("the report has %d frames, want %d", len(r.Frames), len(scenes))}
	}
	for i, f := range r.Frames {
		if f.Name != scenes[i].Name {
			out = append(out, fmt.Sprintf("frame %d is %q, want %q", i, f.Name, scenes[i].Name))
		}
		if f.AtlasFull != 0 || f.Skipped != 0 || len(f.Unsupported) != 0 {
			out = append(out, fmt.Sprintf("%s: the frame left something out: %d shapes for a full atlas, %d commands skipped, %v", f.Name, f.AtlasFull, f.Skipped, f.Unsupported))
		}
	}
	// More tiles than the pages the atlas keeps hold when they are all of the largest slot.
	slots := gpu.AtlasPageSize / gpu.MaxAtlasTile
	if most := gpu.AtlasMaxPages * slots * slots; r.Frames[0].TilesRasterized <= most {
		out = append(out, fmt.Sprintf("the first frame rasterized %d tiles, which fit the %d of the pages the atlas keeps: it is not the frame of the issue", r.Frames[0].TilesRasterized, most))
	}
	if n := r.Frames[1].TilesRasterized; n != 0 {
		out = append(out, fmt.Sprintf("the same frame again rasterized %d tiles, want none", n))
	}
	// The small frame freed pages, so some tiles of the large one are made again.
	if n := r.Frames[3].TilesRasterized; n == 0 || n >= r.Frames[0].TilesRasterized {
		out = append(out, fmt.Sprintf("the frame after the small one rasterized %d tiles, the first %d: the small frame freed no page, or all of them", n, r.Frames[0].TilesRasterized))
	}
	for _, i := range []int{1, 3} {
		if r.Frames[i].Pixels != r.Frames[0].Pixels {
			out = append(out, fmt.Sprintf("%s: the pixels are not those of the first frame", r.Frames[i].Name))
		}
	}
	if r.Frames[2].Pixels == r.Frames[0].Pixels {
		out = append(out, "the frame of one shadow has the pixels of the frame of ten")
	}
	if s := r.Frames[0].Shadow; s < 16 {
		out = append(out, fmt.Sprintf("no shadow under the panels: the pixel there differs from the background by %d", s))
	}
	return out
}

// atlasFullLine is the format of a line of the report: its prefix says what the line is, so
// that the lines are found in the output of an app among others.
const atlasFullLine = AtlasFullName + ": frame %q tiles %d full %d skipped %d unsupported %q pixels %s shadow %d"

// Lines returns the report as text, one line for each frame, for an app on a device to write
// into a file or its output.
func (r AtlasFullReport) Lines() []string {
	out := make([]string, 0, len(r.Frames))
	for _, f := range r.Frames {
		out = append(out, fmt.Sprintf(atlasFullLine, f.Name, f.TilesRasterized, f.AtlasFull, f.Skipped, strings.Join(f.Unsupported, "|"), f.Pixels, f.Shadow))
	}
	return out
}

// ParseAtlasFull reads a report from the lines of an app's file or output; lines that are not
// of the report are passed over. It fails with IssueReportInvalid for a line of the report
// that does not read.
func ParseAtlasFull(lines []string) (AtlasFullReport, error) {
	var rep AtlasFullReport
	for _, line := range lines {
		at := strings.Index(line, AtlasFullName+": frame ")
		if at < 0 {
			continue
		}
		var (
			f           AtlasFullFrame
			unsupported string
		)
		if _, err := fmt.Sscanf(line[at:], atlasFullLine, &f.Name, &f.TilesRasterized, &f.AtlasFull, &f.Skipped, &unsupported, &f.Pixels, &f.Shadow); err != nil {
			return AtlasFullReport{}, errors.NewErr(TagIssueReportInvalid, err).SetMetadata("line", line)
		}
		if unsupported != "" {
			f.Unsupported = strings.Split(unsupported, "|")
		}
		rep.Frames = append(rep.Frames, f)
	}
	return rep, nil
}

// TagIssueReportInvalid is the tag of the error of a report that does not read.
const TagIssueReportInvalid = "IssueReportInvalid"

// The tags of the errors that this file makes.
const tagAtlasFullNoFrame = "AtlasFullNoFrame"
