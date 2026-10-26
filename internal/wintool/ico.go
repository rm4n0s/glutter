package wintool

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/rm4n0s/errors"
	xdraw "golang.org/x/image/draw"
)

// IconSizes are the sizes in pixels of the images in the icon of an .exe (SPEC §25.1): the
// 16 pixels of the title bar and the lists, 24 and 32 for the task bar and the desktop, 48 for the
// large lists, and 64, 128 and 256 for Explorer's large views. The 256 pixels one is stored as a PNG
// and the others as bitmaps, as Windows expects.
//
// Flutter: no counterpart
var IconSizes = []int{16, 24, 32, 48, 64, 128, 256}

// IconImage is one image of an icon: the data of an RT_ICON resource.
//
// Flutter: no counterpart
type IconImage struct {
	// Size is the width and the height in pixels.
	Size int
	// Data is the resource data: a DIB (a BITMAPINFOHEADER with a doubled height, the 32-bit pixels
	// bottom up and the AND mask) for sizes under 256, a PNG for 256.
	Data []byte
}

// Icon is the icon of an .exe made from a PNG.
//
// Flutter: no counterpart
type Icon struct {
	// Images are the RT_ICON resources, one per size of IconSizes, in that order; image i is the
	// resource with ID i+1.
	Images []IconImage
	// Group is the data of the RT_GROUP_ICON resource (a GRPICONDIR that lists Images).
	Group []byte
}

// NewIcon makes the icon of an .exe from a PNG: the image must be square and at least 16 pixels
// wide. It is scaled to each of IconSizes (not scaled if it has that size already). It fails with
// WindowsIconInvalid.
//
// Flutter: no counterpart (windows/runner/resources/app_icon.ico)
func NewIcon(pngData []byte) (*Icon, error) {
	src, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, errors.NewErr(tagWindowsIconInvalid, err).SetMetadata("what", "the icon is not a PNG")
	}
	b := src.Bounds()
	if b.Dx() != b.Dy() {
		return nil, errors.New(tagWindowsIconInvalid, "the icon must be square", "width", b.Dx(), "height", b.Dy())
	}
	if b.Dx() < IconSizes[0] {
		return nil, errors.New(tagWindowsIconInvalid, "the icon must be at least 16 pixels wide", "width", b.Dx())
	}
	icon := &Icon{}
	group := binary.LittleEndian.AppendUint16(nil, 0)  // idReserved
	group = binary.LittleEndian.AppendUint16(group, 1) // idType: icon
	group = binary.LittleEndian.AppendUint16(group, uint16(len(IconSizes)))
	for i, size := range IconSizes {
		img := image.NewNRGBA(image.Rect(0, 0, size, size))
		if b.Dx() == size {
			xdraw.Copy(img, image.Point{}, src, b, xdraw.Src, nil)
		} else {
			xdraw.CatmullRom.Scale(img, img.Bounds(), src, b, xdraw.Src, nil)
		}
		var data []byte
		if size == 256 {
			var out bytes.Buffer
			if err := png.Encode(&out, img); err != nil {
				return nil, errors.NewErr(tagWindowsIconInvalid, err).SetMetadata("what", "the 256 pixel image of the icon")
			}
			data = out.Bytes()
		} else {
			data = iconDIB(img)
		}
		icon.Images = append(icon.Images, IconImage{Size: size, Data: data})
		width := byte(size)
		if size >= 256 {
			width = 0 // 0 means 256
		}
		group = append(group, width, width, 0, 0)           // width, height, colour count, reserved
		group = binary.LittleEndian.AppendUint16(group, 1)  // planes
		group = binary.LittleEndian.AppendUint16(group, 32) // bit count
		group = binary.LittleEndian.AppendUint32(group, uint32(len(data)))
		group = binary.LittleEndian.AppendUint16(group, uint16(i+1)) // the ID of the RT_ICON resource
	}
	icon.Group = group
	return icon, nil
}

// iconDIB encodes the image as the data of an RT_ICON resource that is not a PNG: a
// BITMAPINFOHEADER whose height is twice the image's (it counts the mask), the pixels as 32-bit BGRA
// from the bottom row up, and an AND mask of all zeros (the alpha channel does the masking).
func iconDIB(img *image.NRGBA) []byte {
	size := img.Bounds().Dx()
	maskRow := (size + 31) / 32 * 4
	xor := size * size * 4
	b := make([]byte, 0, 40+xor+maskRow*size)
	b = binary.LittleEndian.AppendUint32(b, 40) // biSize
	b = binary.LittleEndian.AppendUint32(b, uint32(size))
	b = binary.LittleEndian.AppendUint32(b, uint32(size*2))
	b = binary.LittleEndian.AppendUint16(b, 1)  // biPlanes
	b = binary.LittleEndian.AppendUint16(b, 32) // biBitCount
	b = binary.LittleEndian.AppendUint32(b, 0)  // biCompression: BI_RGB
	b = binary.LittleEndian.AppendUint32(b, uint32(xor+maskRow*size))
	b = append(b, make([]byte, 16)...) // pixels per metre, colours used and important
	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			c := img.NRGBAAt(x, y)
			b = append(b, c.B, c.G, c.R, c.A)
		}
	}
	return append(b, make([]byte, maskRow*size)...)
}

// DefaultIcon returns Glutter's icon as a PNG of 256 pixels: a white ring with a gap and a bar, the
// shape of a "G", on the blue of the Material seed color (the shape of the launcher icon of the
// Android tool, drawn here at the size the largest image of an .exe icon has).
//
// Flutter: no counterpart
func DefaultIcon() ([]byte, error) {
	const size = 256
	unit := float64(size) / 192
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	blue := color.NRGBA{R: 0x1a, G: 0x73, B: 0xe8, A: 0xff}
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	c := float64(size) / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := (float64(x)+0.5-c)/unit, (float64(y)+0.5-c)/unit
			d := math.Hypot(dx, dy)
			angle := math.Atan2(dy, dx) // 0 is to the right
			inRing := d >= 44 && d <= 72
			inGap := angle > -0.9 && angle < -0.05
			inBar := dx >= 4 && dx <= 74 && dy >= -10 && dy <= 10
			px := blue
			if (inRing && !inGap) || inBar {
				px = white
			}
			img.SetNRGBA(x, y, px)
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, errors.NewErr(tagWindowsIconInvalid, err).SetMetadata("what", "the default icon")
	}
	return out.Bytes(), nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWindowsIconInvalid = TagWindowsIconInvalid

	// TagWindowsIconInvalid is the tag of the error for an icon that is not a square PNG of at least
	// 16 pixels, or that cannot be encoded.
	TagWindowsIconInvalid = "WindowsIconInvalid"
)
