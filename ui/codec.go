package ui

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	_ "image/jpeg" // registers the JPEG decoder
	"image/png"
	"math"
	"time"

	"github.com/rm4n0s/errors"
	_ "golang.org/x/image/bmp" // registers the BMP decoder
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder
)

// maxImagePixels is the most pixels an image may have to be decoded: a bound on
// the memory a (possibly malicious) file can make the decoder allocate.
const maxImagePixels = 1 << 26

// defaultFrameDuration is how long a GIF frame lasts when its delay is 10 ms or
// less, as in browsers and Skia.
const defaultFrameDuration = 100 * time.Millisecond

// FrameInfo is one frame of an image: how long to show it, and its pixels.
//
// Flutter: ui/painting.dart FrameInfo
type FrameInfo struct {
	// Duration is how long the frame is shown; 0 for a still image.
	Duration time.Duration
	// Image is the frame. The caller disposes it.
	Image *Image
}

// Codec is a decoded still or animated image, from which the frames are read.
//
// Flutter: ui/painting.dart Codec
type Codec struct {
	frames   []codecFrame
	repeat   int
	next     int
	disposed bool
}

type codecFrame struct {
	duration time.Duration
	image    *Image
}

// FrameCount is the number of frames: 1 for a still image.
//
// Flutter: ui/painting.dart Codec.frameCount
func (c *Codec) FrameCount() int { return len(c.frames) }

// RepetitionCount is the number of times the animation repeats after the first
// play: 0 plays it once, -1 repeats it forever, and a positive number repeats it
// that many times. A still image is 0.
//
// Flutter: ui/painting.dart Codec.repetitionCount
func (c *Codec) RepetitionCount() int { return c.repeat }

// GetNextFrame returns the next frame, going back to the first after the last.
// It fails with the tag CodecDisposed after Dispose.
//
// Deviation: Flutter's returns a Future; the frames are decoded when the codec is
// made, so this returns at once.
//
// Flutter: ui/painting.dart Codec.getNextFrame
func (c *Codec) GetNextFrame(ctx context.Context) (FrameInfo, error) {
	if err := ctx.Err(); err != nil {
		return FrameInfo{}, errors.NewErr(tagImageDecodeCanceled, err)
	}
	if c.disposed {
		return FrameInfo{}, errors.New(tagCodecDisposed, "The codec was disposed.")
	}
	f := c.frames[c.next]
	c.next = (c.next + 1) % len(c.frames)
	return FrameInfo{Duration: f.duration, Image: f.image.Clone()}, nil
}

// Dispose releases the codec. The frames already returned stay valid.
//
// Flutter: ui/painting.dart Codec.dispose
func (c *Codec) Dispose() { c.disposed = true }

// InstantiateImageCodecOpts are the optional arguments of InstantiateImageCodec.
//
// Flutter: ui/painting.dart instantiateImageCodec
type InstantiateImageCodecOpts struct {
	// TargetWidth is the width to scale the image to, in pixels; 0 keeps the
	// original, or with a TargetHeight keeps the aspect ratio.
	TargetWidth int
	// TargetHeight is the height to scale the image to, in pixels; 0 keeps the
	// original, or with a TargetWidth keeps the aspect ratio.
	TargetHeight int
	// AllowUpscaling says whether a target larger than the image is honored; nil
	// means true, Flutter's default. When false a target is held to the size of
	// the image on each axis.
	AllowUpscaling *bool
}

// InstantiateImageCodec decodes an image from its encoded bytes: PNG, JPEG, GIF
// (with every frame and its duration), WebP (not animated) or BMP. The pixels are
// premultiplied RGBA. A scale target resizes every frame with a Catmull-Rom
// filter. It fails with ImageDecodeFailed when the data is not an image of a
// supported format, is damaged, or is too large (more than 67 million pixels),
// with ImageSizeInvalid for a negative target, and with ImageDecodeCanceled when
// ctx is done.
//
// Deviation: Flutter's returns a Future and decodes lazily; this decodes at once.
// The orientation in a JPEG's EXIF data is not applied.
//
// Flutter: ui/painting.dart instantiateImageCodec
func InstantiateImageCodec(ctx context.Context, data []byte, opts ...InstantiateImageCodecOpts) (*Codec, error) {
	o := oneOpts(opts)
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagImageDecodeCanceled, err)
	}
	if o.TargetWidth < 0 || o.TargetHeight < 0 {
		return nil, errors.New(tagImageSizeInvalid, "The target size must not be negative.",
			"targetWidth", o.TargetWidth, "targetHeight", o.TargetHeight)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errors.NewErr(tagImageDecodeFailed, err).SetMetadata("stage", "header")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImageSize || cfg.Height > maxImageSize ||
		cfg.Width*cfg.Height > maxImagePixels {
		return nil, errors.New(tagImageDecodeFailed, "The image is too large to decode.",
			"width", cfg.Width, "height", cfg.Height, "format", format)
	}
	tw, th := targetSize(cfg.Width, cfg.Height, o)

	var raw []rawFrame
	repeat := 0
	if format == "gif" {
		raw, repeat, err = decodeGIF(data)
		if err != nil {
			return nil, err
		}
	} else {
		img, _, decodeErr := image.Decode(bytes.NewReader(data))
		if decodeErr != nil {
			return nil, errors.NewErr(tagImageDecodeFailed, decodeErr).SetMetadata("format", format)
		}
		raw = []rawFrame{{img: toPremultipliedRGBA(img)}}
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagImageDecodeCanceled, err)
	}
	c := &Codec{repeat: repeat}
	for _, f := range raw {
		img := f.img
		if tw != img.Rect.Dx() || th != img.Rect.Dy() {
			img = resizeRGBA(img, tw, th)
		}
		c.frames = append(c.frames, codecFrame{duration: f.duration, image: NewImageFromRGBA(img.Pix, tw, th)})
	}
	return c, nil
}

// targetSize works out the size to scale an image of w x h to.
func targetSize(w, h int, o InstantiateImageCodecOpts) (int, int) {
	tw, th := o.TargetWidth, o.TargetHeight
	switch {
	case tw == 0 && th == 0:
		return w, h
	case tw == 0:
		tw = max(int(math.Round(float64(w)*float64(th)/float64(h))), 1)
	case th == 0:
		th = max(int(math.Round(float64(h)*float64(tw)/float64(w))), 1)
	}
	if o.AllowUpscaling != nil && !*o.AllowUpscaling {
		tw, th = min(tw, w), min(th, h)
	}
	if tw > maxImageSize || th > maxImageSize || tw*th > maxImagePixels {
		tw, th = min(tw, maxImageSize), min(th, maxImageSize)
		for tw*th > maxImagePixels {
			tw, th = max(tw/2, 1), max(th/2, 1)
		}
	}
	return tw, th
}

type rawFrame struct {
	img      *image.RGBA
	duration time.Duration
}

// toPremultipliedRGBA converts any decoded image to premultiplied RGBA with its
// origin at 0, 0.
func toPremultipliedRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok && r.Rect.Min == (image.Point{}) {
		return r
	}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)
	return out
}

// resizeRGBA scales the image to w x h.
func resizeRGBA(img *image.RGBA, w, h int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(out, out.Bounds(), img, img.Bounds(), xdraw.Src, nil)
	return out
}

// decodeGIF decodes every frame of a GIF, each as the whole canvas: a frame is
// drawn over the canvas as the previous frames left it, and the frame's disposal
// method says what the canvas becomes before the next one.
func decodeGIF(data []byte) ([]rawFrame, int, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, 0, errors.NewErr(tagImageDecodeFailed, err).SetMetadata("format", "gif")
	}
	if len(g.Image) == 0 {
		return nil, 0, errors.New(tagImageDecodeFailed, "The GIF has no frames.")
	}
	w, h := g.Config.Width, g.Config.Height
	if w == 0 || h == 0 { // old encoders leave the config empty
		w, h = g.Image[0].Bounds().Dx(), g.Image[0].Bounds().Dy()
	}
	if w*h*len(g.Image) > 2*maxImagePixels {
		return nil, 0, errors.New(tagImageDecodeFailed, "The GIF has too many pixels in all its frames.",
			"width", w, "height", h, "frames", len(g.Image))
	}
	canvas := image.NewRGBA(image.Rect(0, 0, w, h))
	var frames []rawFrame
	for i, f := range g.Image {
		var before *image.RGBA
		if i < len(g.Disposal) && g.Disposal[i] == gif.DisposalPrevious {
			before = cloneRGBA(canvas)
		}
		draw.Draw(canvas, f.Bounds(), f, f.Bounds().Min, draw.Over)
		d := defaultFrameDuration
		if i < len(g.Delay) && g.Delay[i] > 1 {
			d = time.Duration(g.Delay[i]) * 10 * time.Millisecond
		}
		frames = append(frames, rawFrame{img: cloneRGBA(canvas), duration: d})
		if i < len(g.Disposal) {
			switch g.Disposal[i] {
			case gif.DisposalBackground:
				draw.Draw(canvas, f.Bounds(), image.Transparent, image.Point{}, draw.Src)
			case gif.DisposalPrevious:
				canvas = before
			}
		}
	}
	// Go's LoopCount is 0 for forever and -1 for once, Flutter's is the reverse.
	repeat := g.LoopCount
	switch {
	case g.LoopCount == 0:
		repeat = -1
	case g.LoopCount < 0:
		repeat = 0
	}
	if len(frames) == 1 {
		repeat = 0
	}
	return frames, repeat, nil
}

func cloneRGBA(img *image.RGBA) *image.RGBA {
	out := image.NewRGBA(img.Rect)
	copy(out.Pix, img.Pix)
	return out
}

// ImageByteFormat is the encoding Image.ToByteData produces.
//
// Flutter: ui/painting.dart ImageByteFormat
type ImageByteFormat int

// The values of ImageByteFormat follow Flutter's declaration order.
//
// Deviation: Flutter's rawExtendedRgba128 is not provided: images here are 8 bits
// per channel.
const (
	// ImageByteFormatUnspecified is the zero value: ImageByteFormatRawRGBA.
	//
	// Flutter: no counterpart
	ImageByteFormatUnspecified ImageByteFormat = iota
	// ImageByteFormatRawRGBA is Flutter's ImageByteFormat.rawRgba: RGBA in row
	// order with premultiplied alpha, 8 bits per channel.
	//
	// Flutter: ui/painting.dart ImageByteFormat.rawRgba
	ImageByteFormatRawRGBA
	// ImageByteFormatRawStraightRGBA is Flutter's ImageByteFormat.rawStraightRgba:
	// RGBA in row order with straight alpha, 8 bits per channel.
	//
	// Flutter: ui/painting.dart ImageByteFormat.rawStraightRgba
	ImageByteFormatRawStraightRGBA
	// ImageByteFormatRawUnmodified is Flutter's ImageByteFormat.rawUnmodified:
	// the image's own format, which here is premultiplied RGBA.
	//
	// Flutter: ui/painting.dart ImageByteFormat.rawUnmodified
	ImageByteFormatRawUnmodified
	// ImageByteFormatPNG is Flutter's ImageByteFormat.png: a PNG file.
	//
	// Flutter: ui/painting.dart ImageByteFormat.png
	ImageByteFormatPNG
)

// ToByteData returns the pixels of the image in the format. The result is a new
// slice. It fails with the tag ImageDisposed on a disposed image, and with
// ImageFormatUnsupported for an unknown format.
//
// Deviation: Flutter's returns a Future.
//
// Flutter: ui/painting.dart Image.toByteData
func (i *Image) ToByteData(format ImageByteFormat) ([]byte, error) {
	if i.disposed {
		return nil, errors.New(tagImageDisposed, "The image was disposed.")
	}
	w, h := i.data.width, i.data.height
	switch format {
	case ImageByteFormatUnspecified, ImageByteFormatRawRGBA, ImageByteFormatRawUnmodified:
		return append([]byte(nil), i.data.pix...), nil
	case ImageByteFormatRawStraightRGBA, ImageByteFormatPNG:
		out := make([]byte, len(i.data.pix))
		for p := 0; p < len(out); p += 4 {
			c := color.NRGBAModel.Convert(color.RGBA{R: i.data.pix[p], G: i.data.pix[p+1], B: i.data.pix[p+2], A: i.data.pix[p+3]}).(color.NRGBA)
			out[p], out[p+1], out[p+2], out[p+3] = c.R, c.G, c.B, c.A
		}
		if format == ImageByteFormatRawStraightRGBA {
			return out, nil
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, &image.NRGBA{Pix: out, Stride: 4 * w, Rect: image.Rect(0, 0, w, h)}); err != nil {
			return nil, errors.NewErr(tagImageEncodeFailed, err)
		}
		return buf.Bytes(), nil
	}
	return nil, errors.New(tagImageFormatUnsupported, "The byte format is not supported.", "format", int(format))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCodecDisposed          = "CodecDisposed"
	tagImageDecodeCanceled    = "ImageDecodeCanceled"
	tagImageDecodeFailed      = "ImageDecodeFailed"
	tagImageDisposed          = "ImageDisposed"
	tagImageEncodeFailed      = "ImageEncodeFailed"
	tagImageFormatUnsupported = "ImageFormatUnsupported"
	tagImageSizeInvalid       = "ImageSizeInvalid"
)
