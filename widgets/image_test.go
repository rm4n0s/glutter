package widgets_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// pngBytes encodes a w x h image filled with the color.
func pngBytes(t testing.TB, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func imageRender(ts *gluttertest.Tester) *rendering.RenderImage {
	r, _ := ts.RenderObject(gluttertest.ByType[*RawImage]()).(*rendering.RenderImage)
	return r
}

func TestImageLoadsAMemoryImage(t *testing.T) {
	tests := []struct {
		name     string
		opts     ImageOpts
		wantSize ui.Size
	}{
		{"intrinsic size", ImageOpts{}, ui.Size{Width: 8, Height: 4}},
		{"given size", ImageOpts{Width: foundation.Some(16.0), Height: foundation.Some(16.0)}, ui.Size{Width: 16, Height: 16}},
		{"width keeps the ratio", ImageOpts{Width: foundation.Some(16.0)}, ui.Size{Width: 16, Height: 8}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, NewImageMemory(pngBytes(t, 8, 4, color.RGBA{R: 255, A: 255}), ImageMemoryOpts{ImageOpts: tt.opts}), ui.TextDirectionLTR)
			waitFor(t, ts, func() bool { r := imageRender(ts); return r != nil && r.Image() != nil })
			ts.Pump()
			if got := ts.Size(gluttertest.ByType[*RawImage]()); got != tt.wantSize {
				t.Errorf("size = %v, want %v", got, tt.wantSize)
			}
		})
	}
}

func TestImageFrameAndLoadingBuilders(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var frames []foundation.Opt[int]
	var syncs []bool
	pumpLoose(ts, NewImage(painting.NewMemoryImage(pngBytes(t, 2, 2, color.White)), ImageOpts{
		FrameBuilder: func(bc BuildContext, child Widget, frame foundation.Opt[int], sync bool) Widget {
			frames = append(frames, frame)
			syncs = append(syncs, sync)
			return child
		},
		LoadingBuilder: func(bc BuildContext, child Widget, p foundation.Opt[painting.ImageChunkEvent]) Widget { return child },
	}), ui.TextDirectionLTR)
	if frames[0].IsSome() {
		t.Errorf("the first build has frame %v, want none", frames[0])
	}
	waitFor(t, ts, func() bool { return frames[len(frames)-1].IsSome() })
	if got := frames[len(frames)-1].OrElse(-1); got != 0 {
		t.Errorf("frame = %d, want 0", got)
	}
}

func TestImageErrorBuilder(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var got error
	pumpLoose(ts, NewImageMemory([]byte("not an image"), ImageMemoryOpts{ImageOpts: ImageOpts{
		ErrorBuilder: func(bc BuildContext, err error) Widget { got = err; return newBox(3, 3) },
	}}), ui.TextDirectionLTR)
	waitFor(t, ts, func() bool { return got != nil })
	ts.Pump()
	if ts.Size(gluttertest.ByType[*box]()) != (ui.Size{Width: 3, Height: 3}) {
		t.Error("the error widget is not shown")
	}
	if ts.Exists(gluttertest.ByType[*RawImage]()) {
		t.Error("the image is shown next to the error widget")
	}
}

func TestImageDebugErrorWidget(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewImageMemory([]byte("not an image")), ui.TextDirectionLTR)
	waitFor(t, ts, func() bool { ts.Pump(); return ts.Exists(gluttertest.ByType[*Placeholder]()) })
	ts.TakeErrors()
}

func TestImageGaplessPlayback(t *testing.T) {
	tests := []struct {
		name     string
		gapless  bool
		wantGone bool
	}{
		{"default clears the old image", false, true},
		{"gapless keeps it", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			mk := func(c color.Color, w int) Widget {
				return NewImageMemory(pngBytes(t, w, w, c), ImageMemoryOpts{ImageOpts: ImageOpts{GaplessPlayback: tt.gapless}})
			}
			pumpLoose(ts, mk(color.White, 4), ui.TextDirectionLTR)
			waitFor(t, ts, func() bool { r := imageRender(ts); return r != nil && r.Image() != nil })
			pumpLoose(ts, mk(color.Black, 6), ui.TextDirectionLTR)
			if gone := imageRender(ts).Image() == nil; gone != tt.wantGone {
				t.Errorf("old image gone = %v, want %v", gone, tt.wantGone)
			}
			waitFor(t, ts, func() bool { r := imageRender(ts); return r.Image() != nil && r.Image().Width() == 6 })
		})
	}
}

func TestImageSemantics(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name    string
		opts    ImageOpts
		label   string
		present bool
	}{
		{"label", ImageOpts{SemanticLabel: foundation.Some("a red square")}, "a red square", true},
		{"excluded", ImageOpts{SemanticLabel: foundation.Some("a red square"), ExcludeFromSemantics: true}, "a red square", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.EnsureSemantics()
			pumpLoose(ts, NewImageMemory(pngBytes(t, 2, 2, color.White), ImageMemoryOpts{ImageOpts: tt.opts}), ui.TextDirectionLTR)
			waitFor(t, ts, func() bool { r := imageRender(ts); return r != nil && r.Image() != nil })
			ts.Pump()
			if got := ts.Exists(gluttertest.BySemanticsLabel(tt.label)); got != tt.present {
				t.Errorf("label present = %v, want %v", got, tt.present)
			}
		})
	}
}

func TestImageConstructorsMakeTheProviders(t *testing.T) {
	tests := []struct {
		name string
		w    *Image
		want func(p painting.ImageProvider) bool
	}{
		{"network", NewImageNetwork("http://example.invalid/a.png"), func(p painting.ImageProvider) bool { _, ok := p.(*painting.NetworkImage); return ok }},
		{"network resized", NewImageNetwork("http://example.invalid/a.png", ImageNetworkOpts{CacheWidth: foundation.Some(10)}),
			func(p painting.ImageProvider) bool { _, ok := p.(*painting.ResizeImage); return ok }},
		{"asset", NewImageAsset("a.png"), func(p painting.ImageProvider) bool { _, ok := p.(*painting.AssetImage); return ok }},
		{"asset with a scale", NewImageAsset("a.png", ImageAssetOpts{Scale: foundation.Some(2.0)}),
			func(p painting.ImageProvider) bool { _, ok := p.(*painting.ExactAssetImage); return ok }},
		{"file", NewImageFile("/tmp/a.png"), func(p painting.ImageProvider) bool { _, ok := p.(painting.FileImage); return ok }},
		{"memory", NewImageMemory([]byte{1}), func(p painting.ImageProvider) bool { _, ok := p.(*painting.MemoryImage); return ok }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.want(tt.w.Image()) {
				t.Errorf("provider = %T", tt.w.Image())
			}
		})
	}
}

func TestImageCacheSizeMustBePositive(t *testing.T) {
	skipUnlessDebug(t)
	err := panicOf(func() { NewImageMemory([]byte{1}, ImageMemoryOpts{CacheWidth: foundation.Some(0)}) })
	if err == nil {
		t.Fatal("no panic")
	}
	_ = requireErrorTag(t, err.(error), "ImageInvalid")
}

func TestRawImageConfiguresTheRenderObject(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ui1 := ui.NewImageFromRGBA(make([]byte, 4*4*4), 4, 4)
	pumpLoose(ts, NewRawImage(ui1, RawImageOpts{Width: foundation.Some(20.0), Fit: painting.BoxFitFill, Scale: 2}), ui.TextDirectionLTR)
	r := imageRender(ts)
	if r == nil || r.Width() != foundation.Some(20.0) || r.Scale() != 2 || r.Fit() != painting.BoxFitFill {
		t.Fatalf("render image = %+v", r)
	}
	if r.Image() == ui1 || !r.Image().IsCloneOf(ui1) {
		t.Error("the render object does not hold a clone of the image")
	}
	pumpLoose(ts, NewRawImage(ui1, RawImageOpts{Width: foundation.Some(30.0)}), ui.TextDirectionLTR)
	if r.Width() != foundation.Some(30.0) || r.Scale() != 1 {
		t.Errorf("width = %v, scale = %v after the update", r.Width(), r.Scale())
	}
	pumpLoose(ts, newLeaf(), ui.TextDirectionLTR)
	if r.Image() != nil {
		t.Error("the render object still holds the image after it was unmounted")
	}
}

func TestRawImageDirection(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(NewRawImage(nil, RawImageOpts{MatchTextDirection: true}))
	_ = requireErrorTag(t, ts.TakeError(), "DirectionalityMissing")
}

func TestImageIcon(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	img := painting.NewMemoryImage(pngBytes(t, 4, 4, color.White))
	pumpLoose(ts, NewImageIcon(img, ImageIconOpts{Size: foundation.Some(30.0), Color: foundation.Some(ui.NewColor(0xFF00FF00))}), ui.TextDirectionLTR)
	waitFor(t, ts, func() bool { r := imageRender(ts); return r != nil && r.Image() != nil })
	r := imageRender(ts)
	if r.Width() != foundation.Some(30.0) || r.Color() != foundation.Some(ui.NewColor(0xFF00FF00)) || r.Fit() != painting.BoxFitScaleDown {
		t.Errorf("width = %v, color = %v, fit = %v", r.Width(), r.Color(), r.Fit())
	}
	pumpLoose(ts, NewImageIcon(img, ImageIconOpts{UseOriginalColors: true}), ui.TextDirectionLTR)
	if r := imageRender(ts); r.Color().IsSome() || r.Width() != foundation.Some(24.0) {
		t.Errorf("color = %v, width = %v with the original colors", r.Color(), r.Width())
	}
	pumpLoose(ts, NewImageIcon(nil, ImageIconOpts{Size: foundation.Some(12.0)}), ui.TextDirectionLTR)
	if got := ts.Size(gluttertest.ByType[*ImageIcon]()); got != (ui.Size{Width: 12, Height: 12}) {
		t.Errorf("empty icon size = %v, want 12x12", got)
	}
	skipUnlessDebug(t)
	err := panicOf(func() {
		NewImageIcon(img, ImageIconOpts{Color: foundation.Some(ui.NewColor(1)), UseOriginalColors: true})
	})
	if err == nil {
		t.Fatal("no panic")
	}
	_ = requireErrorTag(t, err.(error), "ImageIconColorAndOriginal")
}

func TestPrecacheImage(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var done <-chan struct{}
	pumpLoose(ts, NewBuilder(func(bc BuildContext) Widget {
		if done == nil {
			done = PrecacheImage(painting.NewMemoryImage(pngBytes(t, 2, 2, color.White)), bc)
		}
		return newLeaf()
	}), ui.TextDirectionLTR)
	waitFor(t, ts, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})
}

func TestPrecacheImageReportsFailures(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var done <-chan struct{}
	var got error
	pumpLoose(ts, NewBuilder(func(bc BuildContext) Widget {
		if done == nil {
			done = PrecacheImage(painting.NewMemoryImage([]byte("junk")), bc, PrecacheImageOpts{OnError: func(err error) { got = err }})
		}
		return newLeaf()
	}), ui.TextDirectionLTR)
	waitFor(t, ts, func() bool { return got != nil })
	select {
	case <-done:
	default:
		t.Error("the channel is not closed after a failure")
	}
}
