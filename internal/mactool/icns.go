package mactool

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/rm4n0s/errors"
	"golang.org/x/image/draw"
)

// icnsEntry is one image of an .icns file: the four-letter type and the size in pixels (the "@2x" types hold twice the pixels of
// the point size in their name).
type icnsEntry struct {
	typ  string
	size int
}

// icnsEntries are the images of the file, in the order that Apple's iconutil writes them: 16 and 32 pixels as PNG ("icp4",
// "icp5"), then the sizes of an app icon.
var icnsEntries = []icnsEntry{
	{"icp4", 16}, {"icp5", 32}, {"ic11", 32}, {"ic12", 64}, {"ic07", 128}, {"ic13", 256}, {"ic08", 256}, {"ic14", 512}, {"ic09", 512}, {"ic10", 1024},
}

// ICNS makes the .icns file of an icon from a PNG that is square and at least 16 pixels wide: each size of the file is the picture
// scaled (Catmull-Rom) and stored as a PNG, which every macOS that Glutter runs on reads. It fails with MacOSIconInvalid for a picture
// that is not that.
//
// Flutter: no counterpart
func ICNS(pngData []byte) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, errors.NewErr(tagMacOSIconInvalid, err).SetMetadata("what", "the icon is not a PNG")
	}
	b := src.Bounds()
	if b.Dx() != b.Dy() || b.Dx() < 16 {
		return nil, errors.New(tagMacOSIconInvalid, "the icon must be a square PNG of at least 16 pixels", "width", b.Dx(), "height", b.Dy())
	}
	var body []byte
	for _, e := range icnsEntries {
		dst := image.NewNRGBA(image.Rect(0, 0, e.size, e.size))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
		var enc bytes.Buffer
		if err := png.Encode(&enc, dst); err != nil {
			return nil, errors.NewErr(tagMacOSIconInvalid, err).SetMetadata("what", "the icon cannot be encoded", "size", e.size)
		}
		body = append(body, e.typ...)
		body = binary.BigEndian.AppendUint32(body, uint32(8+enc.Len()))
		body = append(body, enc.Bytes()...)
	}
	out := append([]byte("icns"), 0, 0, 0, 0)
	binary.BigEndian.PutUint32(out[4:], uint32(8+len(body)))
	return append(out, body...), nil
}

// DefaultIcon returns Glutter's icon as a PNG of 1024 pixels: a white ring with a gap and a bar, the shape of a "G", on the blue of
// the Material seed color, in a rounded square with the margin that macOS app icons have.
//
// Flutter: no counterpart
func DefaultIcon() ([]byte, error) {
	const size = 1024
	unit := float64(size) / 192
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	blue := color.NRGBA{R: 0x1a, G: 0x73, B: 0xe8, A: 0xff}
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	c := float64(size) / 2
	// The rounded square is 824 pixels wide in a canvas of 1024 (the template of macOS app icons), with a radius of 185.
	const half, radius = 412.0, 185.0
	for y := range size {
		for x := range size {
			px, py := float64(x)+0.5-c, float64(y)+0.5-c
			// The signed distance to the rounded square decides the edge, one pixel of it for the coverage.
			qx, qy := math.Abs(px)-(half-radius), math.Abs(py)-(half-radius)
			d := math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - radius
			cover := math.Min(math.Max(0.5-d, 0), 1)
			if cover == 0 {
				continue
			}
			// The "G" is drawn at 80% so that it sits inside the rounded square.
			dx, dy := px/(unit*0.8), py/(unit*0.8)
			r := math.Hypot(dx, dy)
			angle := math.Atan2(dy, dx)
			inRing := r >= 44 && r <= 72
			inGap := angle > -0.9 && angle < -0.05
			inBar := dx >= 4 && dx <= 74 && dy >= -10 && dy <= 10
			col := blue
			if (inRing && !inGap) || inBar {
				col = white
			}
			col.A = uint8(math.Round(255 * cover))
			img.SetNRGBA(x, y, col)
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, errors.NewErr(tagMacOSIconInvalid, err).SetMetadata("what", "the default icon")
	}
	return out.Bytes(), nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMacOSIconInvalid = TagMacOSIconInvalid

	// TagMacOSIconInvalid is the tag of the error for an icon that is not a square PNG of at least 16 pixels, or that cannot be
	// encoded.
	//
	// Flutter: no counterpart
	TagMacOSIconInvalid = "MacOSIconInvalid"
)
