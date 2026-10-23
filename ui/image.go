package ui

import (
	"context"

	"github.com/rm4n0s/errors"
)

// PixelFormat is the layout of the pixels given to DecodeImageFromPixels.
//
// Flutter: ui/painting.dart PixelFormat
type PixelFormat int

// The values of PixelFormat follow Flutter's declaration order.
const (
	// PixelFormatUnspecified is the zero value: PixelFormatRGBA8888.
	//
	// Flutter: no counterpart
	PixelFormatUnspecified PixelFormat = iota
	// PixelFormatRGBA8888 is Flutter's PixelFormat.rgba8888: each pixel is four
	// bytes, red, green, blue and alpha in this order, with premultiplied alpha.
	//
	// Flutter: ui/painting.dart PixelFormat.rgba8888
	PixelFormatRGBA8888
	// PixelFormatBGRA8888 is Flutter's PixelFormat.bgra8888: each pixel is four
	// bytes, blue, green, red and alpha in this order, with premultiplied alpha.
	//
	// Flutter: ui/painting.dart PixelFormat.bgra8888
	PixelFormatBGRA8888
)

// Deviation: Flutter's PixelFormat also has rgbaFloat32 and r32Float, which the
// software renderer cannot show; they are not provided.

// imageData is the pixels that clones of an image share.
type imageData struct {
	width, height int
	pix           []byte
}

// Image is an image that has been decoded, or rendered, into pixels. It is a
// handle: Clone makes another handle on the same pixels, and each handle is
// disposed by itself.
//
// Flutter: ui/painting.dart Image
type Image struct {
	data     *imageData
	disposed bool
}

// DecodeImageFromPixelsOpts are the optional arguments of DecodeImageFromPixels.
//
// Flutter: ui/painting.dart decodeImageFromPixels
type DecodeImageFromPixelsOpts struct {
	// RowBytes is the number of bytes between the start of one row and the next;
	// 0 means the rows are tightly packed, 4 * width.
	RowBytes int
}

// DecodeImageFromPixels makes an image from raw pixels. The pixels are copied.
// The error ImageSizeInvalid says the size is not valid, ImageFormatUnsupported
// that the format is not known, and ImagePixelsInvalid that the pixels are too
// few for the size and the row bytes.
//
// TODO(step 2.10): the target size options.
//
// Deviation: Flutter delivers the image through a callback; this returns it.
//
// Flutter: ui/painting.dart decodeImageFromPixels
func DecodeImageFromPixels(ctx context.Context, pixels []byte, width, height int, format PixelFormat, opts ...DecodeImageFromPixelsOpts) (*Image, error) {
	o := oneOpts(opts)
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagImageDecodeCanceled, err)
	}
	if width <= 0 || height <= 0 || width > maxImageSize || height > maxImageSize {
		return nil, errors.New(tagImageSizeInvalid, "Invalid image dimensions.", "width", width, "height", height)
	}
	if format == PixelFormatUnspecified {
		format = PixelFormatRGBA8888
	}
	if format != PixelFormatRGBA8888 && format != PixelFormatBGRA8888 {
		return nil, errors.New(tagImageFormatUnsupported, "The pixel format is not supported.", "format", int(format))
	}
	rowBytes := o.RowBytes
	if rowBytes == 0 {
		rowBytes = 4 * width
	}
	if rowBytes < 4*width || len(pixels) < rowBytes*(height-1)+4*width {
		return nil, errors.New(tagImagePixelsInvalid, "There are not enough pixels for the size.",
			"width", width, "height", height, "rowBytes", rowBytes, "length", len(pixels))
	}
	pix := make([]byte, 4*width*height)
	for y := 0; y < height; y++ {
		src := pixels[y*rowBytes : y*rowBytes+4*width]
		dst := pix[y*4*width : (y+1)*4*width]
		copy(dst, src)
		if format == PixelFormatBGRA8888 {
			for x := 0; x < len(dst); x += 4 {
				dst[x], dst[x+2] = dst[x+2], dst[x]
			}
		}
	}
	return &Image{data: &imageData{width: width, height: height, pix: pix}}, nil
}

// maxImageSize is the largest width or height of an image, in pixels.
const maxImageSize = 1 << 15

// NewImageFromRGBA wraps premultiplied RGBA pixels that the caller will not
// change afterwards, without copying them. It is for the renderers, which make
// images by drawing; apps use DecodeImageFromPixels. It panics with the tag
// ImagePixelsInvalid when the pixels are not exactly 4 * width * height bytes.
//
// Flutter: no counterpart
func NewImageFromRGBA(pix []byte, width, height int) *Image {
	if width <= 0 || height <= 0 || len(pix) != 4*width*height {
		panic(errors.New(tagImagePixelsInvalid, "The pixels do not fit the size.", "width", width, "height", height, "length", len(pix)))
	}
	return &Image{data: &imageData{width: width, height: height, pix: pix}}
}

// Width is the width of the image in pixels.
//
// Flutter: ui/painting.dart Image.width
func (i *Image) Width() int { return i.data.width }

// Height is the height of the image in pixels.
//
// Flutter: ui/painting.dart Image.height
func (i *Image) Height() int { return i.data.height }

// Pixels returns the pixels as premultiplied RGBA, 4 bytes each, row by row from
// the top, without padding. They are shared: callers must not change them.
//
// Flutter: no counterpart (Image.toByteData copies)
func (i *Image) Pixels() []byte { return i.data.pix }

// Clone makes another handle on the same pixels.
//
// Flutter: ui/painting.dart Image.clone
func (i *Image) Clone() *Image { return &Image{data: i.data} }

// IsCloneOf says whether the two handles share their pixels.
//
// Flutter: ui/painting.dart Image.isCloneOf
func (i *Image) IsCloneOf(other *Image) bool { return i.data == other.data }

// Dispose releases the handle. The pixels stay while another handle needs them.
// Drawing a disposed image is a programmer error.
//
// Flutter: ui/painting.dart Image.dispose
func (i *Image) Dispose() { i.disposed = true }

// IsDisposed says whether the handle has been disposed.
//
// Flutter: ui/painting.dart Image.debugDisposed
func (i *Image) IsDisposed() bool { return i.disposed }

// String describes the image, as Dart's toString does: "[100×50]".
//
// Flutter: ui/painting.dart Image.toString
func (i *Image) String() string {
	return "[" + itoa(i.data.width) + "×" + itoa(i.data.height) + "]"
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagImagePixelsInvalid = "ImagePixelsInvalid"
)
