package svg

import (
	"image/color"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// leftRed is a document of 20 by 10 with a red left half and a blue right half.
	leftRed = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="10"><rect width="10" height="10" fill="#f00"/><rect x="10" width="10" height="10" fill="#00f"/></svg>`
	// noSize is a document that has nothing to give a size.
	noSize = `<svg xmlns="http://www.w3.org/2000/svg"/>`
)

func TestSvgPictureSizes(t *testing.T) {
	tight := func(w, h float64) func(widgets.Widget) widgets.Widget {
		return func(c widgets.Widget) widgets.Widget {
			return widgets.NewAlign(widgets.NewSizedBox(c, widgets.SizedBoxOpts{Width: some(w), Height: some(h)}), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})
		}
	}
	loose := func(c widgets.Widget) widgets.Widget {
		return widgets.NewAlign(c, widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})
	}
	tests := []struct {
		name string
		src  string
		opts SvgPictureOpts
		wrap func(widgets.Widget) widgets.Widget
		want ui.Size
	}{
		{"neither: the size of the document", red20x10, SvgPictureOpts{}, loose, ui.Size{Width: 20, Height: 10}},
		{"width only keeps the aspect ratio", red20x10, SvgPictureOpts{Width: some(40.0)}, loose, ui.Size{Width: 40, Height: 20}},
		{"height only keeps the aspect ratio", red20x10, SvgPictureOpts{Height: some(40.0)}, loose, ui.Size{Width: 80, Height: 40}},
		{"both are used as they are", red20x10, SvgPictureOpts{Width: some(30.0), Height: some(50.0)}, loose, ui.Size{Width: 30, Height: 50}},
		{"larger than the constraints", red20x10, SvgPictureOpts{Width: some(200.0)}, loose, ui.Size{Width: 100, Height: 100}},
		{"tight constraints beat neither", red20x10, SvgPictureOpts{}, tight(60, 60), ui.Size{Width: 60, Height: 60}},
		{"tight constraints beat width only", red20x10, SvgPictureOpts{Width: some(10.0)}, tight(60, 40), ui.Size{Width: 60, Height: 40}},
		{"a viewBox gives the size", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 15"/>`, SvgPictureOpts{}, loose, ui.Size{Width: 30, Height: 15}},
		{"no size draws nothing, and uses width and height", noSize, SvgPictureOpts{Width: some(30.0), Height: some(20.0)}, loose, ui.Size{Width: 30, Height: 20}},
		{"no size and no width is empty", noSize, SvgPictureOpts{}, loose, ui.Size{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, tt.wrap(NewSvgPictureString(tt.src, tt.opts))))
			if got := ts.Size(gluttertest.ByType[*SvgPicture]()); got != tt.want {
				t.Errorf("size = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSvgPictureFitAndClip(t *testing.T) {
	type probe struct {
		x, y int
		red  bool
	}
	tests := []struct {
		name   string
		opts   SvgPictureOpts
		probes []probe
	}{
		{"contain", SvgPictureOpts{}, []probe{{20, 20, true}, {1, 11, true}, {38, 28, true}, {20, 5, false}, {20, 35, false}}},
		{"contain aligned to the top left", SvgPictureOpts{Alignment: painting.AlignmentTopLeft}, []probe{{20, 5, true}, {38, 18, true}, {20, 25, false}}},
		{"contain aligned to the bottom right", SvgPictureOpts{Alignment: painting.AlignmentBottomRight}, []probe{{20, 35, true}, {20, 15, false}}},
		{"fill stretches", SvgPictureOpts{Fit: painting.BoxFitFill}, []probe{{1, 1, true}, {38, 38, true}, {20, 20, true}}},
		{"cover fills the box and is clipped", SvgPictureOpts{Fit: painting.BoxFitCover}, []probe{{1, 1, true}, {38, 38, true}, {45, 20, false}, {20, 45, false}}},
		{"cover without a clip draws outside", SvgPictureOpts{Fit: painting.BoxFitCover, ClipBehavior: ui.ClipNone}, []probe{{38, 38, true}, {45, 20, true}, {55, 20, true}, {65, 20, false}}},
		{"cover with an anti-aliased clip", SvgPictureOpts{Fit: painting.BoxFitCover, ClipBehavior: ui.ClipAntiAlias}, []probe{{38, 38, true}, {45, 20, false}}},
		{"cover with a save layer clip", SvgPictureOpts{Fit: painting.BoxFitCover, ClipBehavior: ui.ClipAntiAliasWithSaveLayer}, []probe{{38, 38, true}, {45, 20, false}}},
		{"fit width", SvgPictureOpts{Fit: painting.BoxFitFitWidth}, []probe{{20, 20, true}, {20, 5, false}, {20, 35, false}}},
		{"fit height is cropped", SvgPictureOpts{Fit: painting.BoxFitFitHeight}, []probe{{1, 1, true}, {38, 38, true}, {45, 20, false}}},
		{"none keeps the size", SvgPictureOpts{Fit: painting.BoxFitNone}, []probe{{20, 20, true}, {11, 16, true}, {5, 20, false}, {20, 12, false}, {35, 20, false}}},
		{"scale down keeps a small picture", SvgPictureOpts{Fit: painting.BoxFitScaleDown}, []probe{{20, 20, true}, {5, 20, false}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			o := tt.opts
			o.Width, o.Height = some(40.0), some(40.0)
			pumpTopLeft(ts, NewSvgPictureString(red20x10, o))
			for _, p := range tt.probes {
				got := pixel(ts, p.x, p.y)
				if isRed := got == redC; isRed != p.red {
					t.Errorf("pixel (%d, %d) = %v, want red = %v", p.x, p.y, got, p.red)
				}
			}
		})
	}
}

func TestSvgPictureMatchTextDirection(t *testing.T) {
	tests := []struct {
		name        string
		dir         ui.TextDirection
		match       bool
		left, right color.RGBA
	}{
		{"left to right", ui.TextDirectionLTR, true, redC, blueC},
		{"right to left, not matched", ui.TextDirectionRTL, false, redC, blueC},
		{"right to left, matched", ui.TextDirectionRTL, true, blueC, redC},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			pumpTopLeft(ts, NewSvgPictureString(leftRed, SvgPictureOpts{MatchTextDirection: tt.match}), tt.dir)
			if got := pixel(ts, 5, 5); got != tt.left {
				t.Errorf("left = %v, want %v", got, tt.left)
			}
			if got := pixel(ts, 15, 5); got != tt.right {
				t.Errorf("right = %v, want %v", got, tt.right)
			}
		})
	}
}

func TestSvgPictureAlignmentDirectional(t *testing.T) {
	// A directional alignment is resolved with the text direction: "start" is the right side in RTL.
	tests := []struct {
		name      string
		dir       ui.TextDirection
		redAtLeft bool
	}{
		{"start in LTR is the left", ui.TextDirectionLTR, true},
		{"start in RTL is the right", ui.TextDirectionRTL, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			pumpTopLeft(ts, NewSvgPictureString(red20x10, SvgPictureOpts{Width: some(40.0), Height: some(10.0), Alignment: painting.AlignmentDirectionalCenterStart}), tt.dir)
			// 20x10 in a 40x10 box, at the start.
			left, right := pixel(ts, 5, 5), pixel(ts, 35, 5)
			if (left == redC) != tt.redAtLeft || (right == redC) == tt.redAtLeft {
				t.Errorf("left = %v, right = %v", left, right)
			}
		})
	}
}

func TestSvgPictureColorFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter *ui.ColorFilter
		want   color.RGBA
	}{
		{"none", nil, redC},
		{"srcIn blue", ui.NewColorFilterMode(ui.NewColorFromARGB(255, 0, 0, 255), ui.BlendModeSrcIn), blueC},
		{"srcIn fully transparent", ui.NewColorFilterMode(ui.NewColorFromARGB(0, 0, 0, 255), ui.BlendModeSrcIn), color.RGBA{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			pumpTopLeft(ts, NewSvgPictureString(red20x10, SvgPictureOpts{ColorFilter: tt.filter}))
			if got := pixel(ts, 10, 5); got != tt.want {
				t.Errorf("pixel = %v, want %v", got, tt.want)
			}
			// What is outside the picture stays clear: the filter is in a layer of the size of the picture.
			if got := pixel(ts, 50, 50); got != (color.RGBA{}) {
				t.Errorf("pixel outside = %v", got)
			}
		})
	}
}

func TestSvgPictureAllowDrawingOutsideViewBox(t *testing.T) {
	const overflowing = `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 10 10"><rect x="-5" y="-5" width="20" height="20" fill="#f00"/></svg>`
	tests := []struct {
		name    string
		allow   bool
		wantOut bool
	}{
		{"clipped to the viewBox by default", false, false},
		{"drawn outside when allowed", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			pumpTopLeft(ts, widgets.NewPadding(painting.NewEdgeInsetsAll(20),
				NewSvgPictureString(overflowing, SvgPictureOpts{AllowDrawingOutsideViewBox: tt.allow})))
			if got := pixel(ts, 25, 25); got != redC {
				t.Errorf("inside = %v, want red", got)
			}
			if got := pixel(ts, 17, 25) == redC; got != tt.wantOut {
				t.Errorf("outside is red = %v, want %v", got, tt.wantOut)
			}
		})
	}
}

func TestSvgPictureRepaintsOnlyWhenItChanges(t *testing.T) {
	a := &entry{doc: mustParseDoc(t, red20x10)}
	b := &entry{doc: mustParseDoc(t, red20x10)}
	base := func() *picturePainter {
		return &picturePainter{entry: a, fit: painting.BoxFitContain, alignment: painting.AlignmentCenter, dpr: 1}
	}
	tests := []struct {
		name   string
		change func(p *picturePainter)
		want   bool
	}{
		{"nothing", func(*picturePainter) {}, false},
		{"another document", func(p *picturePainter) { p.entry = b }, true},
		{"fit", func(p *picturePainter) { p.fit = painting.BoxFitCover }, true},
		{"alignment", func(p *picturePainter) { p.alignment = painting.AlignmentTopLeft }, true},
		{"direction", func(p *picturePainter) { p.direction = ui.TextDirectionRTL }, true},
		{"clip", func(p *picturePainter) { p.clip = ui.ClipNone }, true},
		{"flip", func(p *picturePainter) { p.flip = true }, true},
		{"outside", func(p *picturePainter) { p.outside = true }, true},
		{"color filter", func(p *picturePainter) { p.filter = ui.NewColorFilterSaturation(0) }, true},
		{"device pixel ratio", func(p *picturePainter) { p.dpr = 2 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := base()
			tt.change(p)
			if got := p.ShouldRepaint(base()); got != tt.want {
				t.Errorf("ShouldRepaint = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustParseDoc(t *testing.T, src string) *Document {
	t.Helper()
	d, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

const patterned = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="10"><defs><pattern id="p" width="4" height="4" patternUnits="userSpaceOnUse"><rect width="2" height="2" fill="#f00"/></pattern></defs><rect width="20" height="10" fill="url(#p)"/></svg>`

func TestEntryRecordsOnceForAScaleIndependentDocument(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		scales     []float64
		wantPics   int
		wantSameAs [][2]int // pairs of indexes of scales that give the same picture
	}{
		{"a plain document is recorded once", red20x10, []float64{1, 2, 3.7, 0.2}, 1, [][2]int{{0, 1}, {0, 2}, {0, 3}}},
		{"a pattern is recorded per scale", patterned, []float64{1, 2, 3}, 3, nil},
		{"scales that round to the same step share a picture", patterned, []float64{2.01, 2.1, 2.125}, 1, [][2]int{{0, 1}, {0, 2}}},
		{"a scale is rounded up", patterned, []float64{2, 2.01}, 2, nil},
		{"at most four pictures are kept", patterned, []float64{1, 2, 3, 4, 5, 6}, 4, nil},
		{"huge scales are capped", patterned, []float64{100, 1000}, 1, [][2]int{{0, 1}}},
		{"tiny scales are raised", patterned, []float64{0.001, 0.05}, 1, [][2]int{{0, 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &entry{doc: mustParseDoc(t, tt.src)}
			var pics []*ui.Picture
			for _, s := range tt.scales {
				pics = append(pics, e.picture(s, nil))
			}
			if len(e.pics) != tt.wantPics {
				t.Errorf("%d pictures kept, want %d", len(e.pics), tt.wantPics)
			}
			for _, pair := range tt.wantSameAs {
				if pics[pair[0]] != pics[pair[1]] {
					t.Errorf("scales %v and %v gave different pictures", tt.scales[pair[0]], tt.scales[pair[1]])
				}
			}
		})
	}
}

func TestSvgPictureRecordsAtTheScaleItIsDrawnAt(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		dpr   float64
		width float64
		want  []float64 // the scales of the pictures the document recorded, in order
	}{
		{"a plain document records at 1", red20x10, 2, 40, []float64{1}},
		{"a pattern: device pixel ratio times the fit", patterned, 2, 40, []float64{4}},
		{"a pattern at ratio 1.5 and fit 3", patterned, 1.5, 60, []float64{4.5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t, gluttertest.Opts{Size: ui.Size{Width: 100, Height: 100}, DevicePixelRatio: tt.dpr})
			pumpTopLeft(ts, widgets.NewMediaQuery(widgets.NewMediaQueryDataFromView(ts.View, nil), NewSvgPictureString(tt.src, SvgPictureOpts{Width: some(tt.width)})))
			s := gluttertest.StateOf[*svgPictureState](ts, gluttertest.ByType[*SvgPicture]())
			var got []float64
			for _, r := range s.entry.pics {
				got = append(got, r.scale)
			}
			if len(got) != len(tt.want) || (len(got) > 0 && got[0] != tt.want[0]) {
				t.Errorf("recorded at %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSvgPictureDoesNotRecordAgainWhenResized(t *testing.T) {
	ts := newTester(t)
	show := func(w float64) {
		pumpTopLeft(ts, NewSvgPictureString(red20x10, SvgPictureOpts{Width: some(w)}))
	}
	show(20)
	s := gluttertest.StateOf[*svgPictureState](ts, gluttertest.ByType[*SvgPicture]())
	first := s.entry.pics[0].pic
	for _, w := range []float64{40, 80, 10} {
		show(w)
		if len(s.entry.pics) != 1 || s.entry.pics[0].pic != first {
			t.Fatalf("width %v recorded again: %d pictures", w, len(s.entry.pics))
		}
	}
	if got := pixel(ts, 4, 2); got != redC {
		t.Errorf("pixel at the smallest size = %v", got)
	}
}

func TestSvgPictureSemantics(t *testing.T) {
	const titled = `<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><title>A red square</title><rect width="4" height="4" fill="#f00"/></svg>`
	tests := []struct {
		name  string
		src   string
		opts  SvgPictureOpts
		label string
		found bool
	}{
		{"the label of the options", red20x10, SvgPictureOpts{SemanticsLabel: some("a logo")}, "a logo", true},
		{"the title of the document", titled, SvgPictureOpts{}, "A red square", true},
		{"the label beats the title", titled, SvgPictureOpts{SemanticsLabel: some("Square")}, "Square", true},
		{"excluded", red20x10, SvgPictureOpts{SemanticsLabel: some("a logo"), ExcludeFromSemantics: true}, "a logo", false},
		{"excluded with a title", titled, SvgPictureOpts{ExcludeFromSemantics: true}, "A red square", false},
		{"no label and no title", red20x10, SvgPictureOpts{}, "a logo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !foundation.DebugMode {
				t.Skip("the semantics finders read the debug semantics")
			}
			ts := newTester(t)
			ts.EnsureSemantics()
			pumpTopLeft(ts, NewSvgPictureString(tt.src, tt.opts))
			ts.Pump()
			if got := ts.Exists(gluttertest.BySemanticsLabel(tt.label)); got != tt.found {
				t.Errorf("label %q found = %v, want %v", tt.label, got, tt.found)
			}
		})
	}
}
