package iostool

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/rm4n0s/errors"
	"golang.org/x/image/draw"
)

// IconBase is the start of the names of the icon files and the name of the icon in the Info.plist.
//
// Flutter: no counterpart
const IconBase = "AppIcon"

// IconFile is a PNG of the icon of an app: its name in the bundle and its size in pixels.
//
// Flutter: no counterpart
type IconFile struct {
	// Name is the file name, with the size and the scale that iOS looks for ("AppIcon60x60@2x.png").
	Name string
	// Pixels is the width and the height of the picture.
	Pixels int
}

// IconFiles are the icons an app of iOS 15 or later needs without an asset catalog: the iPhone's 60 points at 2x and 3x, and the
// iPad's 76 points at 2x and its 83.5 points at 2x (the Pro). The 1024 pixels of the App Store are the owner's, with the
// provisioning profile.
//
// Flutter: no counterpart
func IconFiles() []IconFile {
	return []IconFile{
		{IconBase + "60x60@2x.png", 120},
		{IconBase + "60x60@3x.png", 180},
		{IconBase + "76x76@2x~ipad.png", 152},
		{IconBase + "83.5x83.5@2x~ipad.png", 167},
	}
}

// Icons makes the PNG files of an icon from a square PNG of at least 167 pixels. It fails with IOSIconInvalid when the picture is
// not a PNG, not square, or smaller than the largest file.
//
// Flutter: no counterpart
func Icons(pngData []byte) (map[string][]byte, error) {
	src, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, errors.NewErr(tagIOSIconInvalid, err).SetMetadata("what", "the icon is not a PNG")
	}
	b := src.Bounds()
	if b.Dx() != b.Dy() {
		return nil, errors.New(tagIOSIconInvalid, "the icon is not square", "width", b.Dx(), "height", b.Dy())
	}
	files := IconFiles()
	largest := 0
	for _, f := range files {
		largest = max(largest, f.Pixels)
	}
	if b.Dx() < largest {
		return nil, errors.New(tagIOSIconInvalid, "the icon is smaller than the largest file", "size", b.Dx(), "needed", largest)
	}
	out := map[string][]byte{}
	for _, f := range files {
		dst := image.NewNRGBA(image.Rect(0, 0, f.Pixels, f.Pixels))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
		// iOS icons have no alpha: the picture is drawn over the blue of its background.
		flat := image.NewNRGBA(dst.Bounds())
		draw.Draw(flat, flat.Bounds(), &image.Uniform{C: color.NRGBA{R: 0x1a, G: 0x73, B: 0xe8, A: 0xff}}, image.Point{}, draw.Src)
		draw.Draw(flat, flat.Bounds(), dst, image.Point{}, draw.Over)
		var buf bytes.Buffer
		if err := png.Encode(&buf, flat); err != nil {
			return nil, errors.NewErr(tagIOSIconInvalid, err).SetMetadata("what", "the icon cannot be encoded", "file", f.Name)
		}
		out[f.Name] = buf.Bytes()
	}
	return out, nil
}

// DefaultIcon returns Glutter's icon as a PNG of 1024 pixels: a white ring with a gap and a bar, the shape of a "G", on the blue of
// the Material seed color, over the whole square (iOS rounds the corners itself).
//
// Flutter: no counterpart
func DefaultIcon() ([]byte, error) {
	const size = 1024
	unit := float64(size) / 192
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	blue := color.NRGBA{R: 0x1a, G: 0x73, B: 0xe8, A: 0xff}
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	c := float64(size) / 2
	for y := range size {
		for x := range size {
			dx, dy := (float64(x)+0.5-c)/unit, (float64(y)+0.5-c)/unit
			r := math.Hypot(dx, dy)
			angle := math.Atan2(dy, dx)
			inRing := r >= 44 && r <= 72
			inGap := angle > -0.9 && angle < -0.05
			inBar := dx >= 4 && dx <= 74 && dy >= -10 && dy <= 10
			col := blue
			if (inRing && !inGap) || inBar {
				col = white
			}
			img.SetNRGBA(x, y, col)
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, errors.NewErr(tagIOSIconInvalid, err).SetMetadata("what", "the default icon")
	}
	return out.Bytes(), nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagIOSIconInvalid = TagIOSIconInvalid

	// TagIOSIconInvalid is the tag of the error for an icon that is not a square PNG of at least 167 pixels, or that cannot be
	// encoded.
	//
	// Flutter: no counterpart
	TagIOSIconInvalid = "IOSIconInvalid"
)
