package issues

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// The issue "The software renderer makes every picture's mipmaps again on every frame" of
// Fishball's list of 2026-11-18, fixed in step 49.1 (Glutter 1.17.6): the software renderer made a
// new map of mipmap chains for every frame (and for every damaged rectangle of a frame), so a
// picture scaled down under or beside an animation (a spinner while the next picture is made) was
// made smaller again in every frame, about 5 ms and 1.3 MB of garbage a frame for a picture of
// 1024 x 768. The reproduction is nine frames on one renderer: a spinner over a picture that is
// drawn at half its size with mipmaps, five frames of the spinner turning, another picture of the
// same size, and the first picture again.

// SoftMipmapsName is the name of the issue: what an app of a device test is asked for, and the
// prefix of the lines of its report.
const SoftMipmapsName = "soft-mipmaps"

// SoftMipmapsWidth and SoftMipmapsHeight are the size of the frames of the reproduction, in pixels.
const (
	SoftMipmapsWidth  = 640
	SoftMipmapsHeight = 480
)

// The pictures are 1024 x 768 pixels, drawn at half their size from (0, 0); the spinner turns
// around the center of the picture, inside spinnerBox; the rows of pictureBand are far from it.
var (
	pictureSize = image.Pt(1024, 768)
	spinnerBox  = image.Rect(226, 162, 286, 222)
	pictureBand = image.Rect(0, 0, 512, 150)
)

// softMipmapsColors are the two colors of the blocks of a picture, premultiplied and opaque; seed
// is the green of the first and the blue of the second, so that two pictures differ.
func softMipmapsColors(seed uint8) [2][4]uint8 {
	return [2][4]uint8{{200, seed, 40, 255}, {20, 90, seed, 255}}
}

// SoftMipmapsPicture returns a picture of the reproduction: 1024 x 768 pixels of blocks of 16
// pixels in two colors that the seed changes.
func SoftMipmapsPicture(seed uint8) *ui.Image {
	colors := softMipmapsColors(seed)
	w, h := pictureSize.X, pictureSize.Y
	pix := make([]byte, 4*w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			copy(pix[4*(y*w+x):], colors[(x/16+y/16)%2][:])
		}
	}
	return ui.NewImageFromRGBA(pix, w, h)
}

// SoftMipmapsScene is one frame of the reproduction.
type SoftMipmapsScene struct {
	Name  string
	Scene *ui.Scene
	// Seed is the seed of the picture the frame shows.
	Seed uint8
}

// The seeds of the two pictures.
const (
	softMipmapsFirst  = 60
	softMipmapsSecond = 160
)

// SoftMipmapsScenes returns the frames of the reproduction, in the order a renderer draws them.
// Each picture is recorded once, in a layer of its own, as a repaint boundary keeps it, and the
// spinner is a layer over it that changes in every frame.
func SoftMipmapsScenes() []SoftMipmapsScene {
	pictureLayer := func(img *ui.Image) *ui.Picture {
		rec := ui.NewPictureRecorder()
		c := ui.NewCanvas(rec)
		p := ui.NewPaint()
		p.SetFilterQuality(ui.FilterQualityMedium)
		w, h := float64(img.Width()), float64(img.Height())
		c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, w, h), ui.NewRectFromLTWH(0, 0, w/2, h/2), p)
		return rec.EndRecording()
	}
	frame := func(pic *ui.Picture, turn int) *ui.Scene {
		rec := ui.NewPictureRecorder()
		c := ui.NewCanvas(rec)
		center := spinnerBox.Min.Add(spinnerBox.Max).Div(2)
		c.Translate(float64(center.X), float64(center.Y))
		c.Rotate(float64(turn) * 0.4)
		p := ui.NewPaint()
		p.SetColor(ui.NewColor(0xFF2060FF))
		c.DrawRect(ui.NewRectFromLTWH(-14, -14, 28, 28), p)
		sb := ui.NewSceneBuilder()
		sb.AddPicture(ui.Offset{}, pic)
		sb.AddPicture(ui.Offset{}, rec.EndRecording())
		return sb.Build()
	}
	first, second := pictureLayer(SoftMipmapsPicture(softMipmapsFirst)), pictureLayer(SoftMipmapsPicture(softMipmapsSecond))
	out := []SoftMipmapsScene{{Name: "a spinner over a picture", Scene: frame(first, 0), Seed: softMipmapsFirst}}
	for i := 1; i <= 5; i++ {
		out = append(out, SoftMipmapsScene{Name: fmt.Sprintf("the spinner turns %d", i), Scene: frame(first, i), Seed: softMipmapsFirst})
	}
	return append(out,
		SoftMipmapsScene{Name: "another picture", Scene: frame(second, 6), Seed: softMipmapsSecond},
		SoftMipmapsScene{Name: "the first picture again", Scene: frame(first, 7), Seed: softMipmapsFirst},
	)
}

// SoftMipmapsStats is what a renderer's cache of mipmap chains says after a frame (the fields of
// the same names of soft.CacheStats): the chains it holds, and how many it has made since it was
// made.
type SoftMipmapsStats struct {
	Chains int
	Built  int
}

// SoftMipmapsFrame is what came of one frame.
type SoftMipmapsFrame struct {
	Name string
	// Built is how many mipmap chains the frame made, and Chains how many the renderer holds
	// after it.
	Built, Chains int
	// Picture and Spinner are the SHA-256, in hexadecimal, of the pixels of a band of the picture
	// far from the spinner and of the box the spinner turns in.
	Picture, Spinner string
	// Block says whether the middle of the first block of the picture, at half its size, has the
	// color of the picture the frame shows, within 3 in each channel.
	Block bool
}

// hashRect returns the SHA-256 of the pixels of a rectangle of the image, in hexadecimal.
func hashRect(img *image.RGBA, r image.Rectangle) string {
	h := sha256.New()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		o := img.PixOffset(r.Min.X, y)
		h.Write(img.Pix[o : o+4*r.Dx()])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// SoftMipmapsFrameOf makes the record of a frame from its pixels and the chains it made and the
// renderer holds.
func SoftMipmapsFrameOf(sc SoftMipmapsScene, img *image.RGBA, built, chains int) SoftMipmapsFrame {
	want := softMipmapsColors(sc.Seed)[0]
	got := img.RGBAAt(4, 4)
	block := true
	for i, v := range []uint8{got.R, got.G, got.B, got.A} {
		if d := int(v) - int(want[i]); d < -3 || d > 3 {
			block = false
		}
	}
	return SoftMipmapsFrame{Name: sc.Name, Built: built, Chains: chains, Picture: hashRect(img, pictureBand), Spinner: hashRect(img, spinnerBox), Block: block}
}

// SoftMipmapsReport is what came of the frames of the reproduction on one renderer.
type SoftMipmapsReport struct {
	Frames []SoftMipmapsFrame
}

// SoftMipmapsDraw draws a scene of the given size on the renderer under test, as one frame, and
// returns the pixels of the frame and what the renderer's cache of mipmap chains says after it.
type SoftMipmapsDraw func(scene *ui.Scene, width, height int) (*image.RGBA, SoftMipmapsStats, error)

// RunSoftMipmaps draws the frames of the reproduction and returns the report. An error of a frame
// ends the run: the report then has the frames before it.
func RunSoftMipmaps(draw SoftMipmapsDraw) (SoftMipmapsReport, error) {
	var (
		rep  SoftMipmapsReport
		prev SoftMipmapsStats
	)
	for i, sc := range SoftMipmapsScenes() {
		img, st, err := draw(sc.Scene, SoftMipmapsWidth, SoftMipmapsHeight)
		if err != nil {
			return rep, err
		}
		if img == nil || img.Bounds() != image.Rect(0, 0, SoftMipmapsWidth, SoftMipmapsHeight) {
			return rep, errors.New(tagSoftMipmapsNoFrame, "the renderer gave no frame of the size asked for", "frame", sc.Name)
		}
		built := st.Built
		if i > 0 {
			built -= prev.Built
		}
		prev = st
		rep.Frames = append(rep.Frames, SoftMipmapsFrameOf(sc, img, built, st.Chains))
	}
	return rep, nil
}

// Problems checks the report and returns what is wrong, one line for each finding; none when the
// renderer made the chain of each picture once, kept both, and drew every frame with the picture
// it shows and the spinner where it turned to.
func (r SoftMipmapsReport) Problems() []string {
	var out []string
	scenes := SoftMipmapsScenes()
	if len(r.Frames) != len(scenes) {
		return []string{fmt.Sprintf("the report has %d frames, want %d", len(r.Frames), len(scenes))}
	}
	first, last := r.Frames[0], len(r.Frames)-1
	for i, f := range r.Frames {
		if f.Name != scenes[i].Name {
			out = append(out, fmt.Sprintf("frame %d is %q, want %q", i, f.Name, scenes[i].Name))
		}
		// A picture's chain is made when the picture is first drawn, and never again.
		want, chains := 0, 1
		switch i {
		case 0, last - 1:
			want = 1
		}
		if i >= last-1 {
			chains = 2
		}
		if f.Built != want {
			out = append(out, fmt.Sprintf("%s: the frame made %d mipmap chains, want %d", f.Name, f.Built, want))
		}
		if f.Chains != chains {
			out = append(out, fmt.Sprintf("%s: the renderer holds %d mipmap chains, want %d", f.Name, f.Chains, chains))
		}
		if !f.Block {
			out = append(out, fmt.Sprintf("%s: the picture at half its size does not have the color of its first block", f.Name))
		}
		if scenes[i].Seed == scenes[0].Seed && f.Picture != first.Picture {
			out = append(out, fmt.Sprintf("%s: the picture's pixels are not those of the first frame", f.Name))
		}
		if i > 0 && f.Spinner == r.Frames[i-1].Spinner {
			out = append(out, fmt.Sprintf("%s: the spinner did not turn", f.Name))
		}
	}
	if r.Frames[last-1].Picture == first.Picture {
		out = append(out, "another picture: the pixels are those of the first picture")
	}
	return out
}

// softMipmapsLine is the format of a line of the report: its prefix says what the line is, so
// that the lines are found in the output of an app among others.
const softMipmapsLine = SoftMipmapsName + ": frame %q built %d chains %d picture %s spinner %s block %t"

// Lines returns the report as text, one line for each frame, for an app on a device to write into
// a file or its output.
func (r SoftMipmapsReport) Lines() []string {
	out := make([]string, 0, len(r.Frames))
	for _, f := range r.Frames {
		out = append(out, fmt.Sprintf(softMipmapsLine, f.Name, f.Built, f.Chains, f.Picture, f.Spinner, f.Block))
	}
	return out
}

// ParseSoftMipmaps reads a report from the lines of an app's file or output; lines that are not of
// the report are passed over. It fails with IssueReportInvalid for a line of the report that does
// not read.
func ParseSoftMipmaps(lines []string) (SoftMipmapsReport, error) {
	var rep SoftMipmapsReport
	for _, line := range lines {
		at := strings.Index(line, SoftMipmapsName+": frame ")
		if at < 0 {
			continue
		}
		var f SoftMipmapsFrame
		if _, err := fmt.Sscanf(line[at:], softMipmapsLine, &f.Name, &f.Built, &f.Chains, &f.Picture, &f.Spinner, &f.Block); err != nil {
			return SoftMipmapsReport{}, errors.NewErr(TagIssueReportInvalid, err).SetMetadata("line", line)
		}
		rep.Frames = append(rep.Frames, f)
	}
	return rep, nil
}

// The tags of the errors that this file makes.
const tagSoftMipmapsNoFrame = "SoftMipmapsNoFrame"
