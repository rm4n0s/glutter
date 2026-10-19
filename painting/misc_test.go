package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func sz(w, h float64) ui.Size { return ui.Size{Width: w, Height: h} }

func TestApplyBoxFit(t *testing.T) {
	tests := []struct {
		name     string
		fit      BoxFit
		in, out  ui.Size
		src, dst ui.Size
	}{
		{"scaleDown small", BoxFitScaleDown, sz(100, 1000), sz(200, 2000), sz(100, 1000), sz(100, 1000)},
		{"scaleDown large", BoxFitScaleDown, sz(300, 3000), sz(200, 2000), sz(300, 3000), sz(200, 2000)},
		{"fitWidth wide", BoxFitFitWidth, sz(2000, 400), sz(1000, 100), sz(2000, 200), sz(1000, 100)},
		{"fitWidth tall", BoxFitFitWidth, sz(2000, 400), sz(1000, 300), sz(2000, 400), sz(1000, 200)},
		{"fitHeight wide", BoxFitFitHeight, sz(400, 2000), sz(100, 1000), sz(200, 2000), sz(100, 1000)},
		{"fitHeight tall", BoxFitFitHeight, sz(400, 2000), sz(300, 1000), sz(400, 2000), sz(200, 1000)},
		{"fill", BoxFitFill, sz(400, 2000), sz(300, 1000), sz(400, 2000), sz(300, 1000)},
		{"contain", BoxFitContain, sz(400, 200), sz(100, 100), sz(400, 200), sz(100, 50)},
		{"cover", BoxFitCover, sz(400, 200), sz(100, 100), sz(200, 200), sz(100, 100)},
		{"none", BoxFitNone, sz(400, 200), sz(100, 300), sz(100, 200), sz(100, 200)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyBoxFit(tt.fit, tt.in, tt.out)
			if got.Source != tt.src || got.Destination != tt.dst {
				t.Errorf("got %v, want %v -> %v", got, tt.src, tt.dst)
			}
		})
	}
	for _, fit := range []BoxFit{BoxFitFill, BoxFitContain, BoxFitCover, BoxFitFitWidth, BoxFitFitHeight, BoxFitNone, BoxFitScaleDown} {
		for _, pair := range [][2]ui.Size{
			{sz(-400, 2000), sz(100, 1000)}, {sz(400, -2000), sz(100, 1000)}, {sz(400, 2000), sz(-100, 1000)}, {sz(400, 2000), sz(100, -1000)},
			{sz(0, 2000), sz(100, 1000)}, {sz(400, 0), sz(100, 1000)}, {sz(400, 2000), sz(0, 1000)}, {sz(400, 2000), sz(100, 0)},
		} {
			if got := ApplyBoxFit(fit, pair[0], pair[1]); got != (FittedSizes{}) {
				t.Errorf("%v %v: got %v, want zero", fit, pair, got)
			}
		}
	}
	if BoxFitFitWidth.String() != "BoxFit.fitWidth" || BoxFitScaleDown.Name() != "scaleDown" {
		t.Error("names are wrong")
	}
}

func TestFlutterLogoDecorationLerp(t *testing.T) {
	start := NewFlutterLogoDecoration(FlutterLogoDecorationOpts{TextColor: foundation.Some(ui.NewColor(0xFFD4F144)), Style: FlutterLogoStyleStacked, Margin: NewEdgeInsetsAll(10)})
	end := NewFlutterLogoDecoration(FlutterLogoDecorationOpts{TextColor: foundation.Some(ui.NewColor(0xFF81D4FA)), Style: FlutterLogoStyleStacked, Margin: NewEdgeInsetsAll(10)})
	if FlutterLogoDecorationLerp(nil, nil, 0.5) != nil {
		t.Error("lerp of nils should be nil")
	}
	logo := NewFlutterLogoDecoration()
	if FlutterLogoDecorationLerp(logo, logo, 0.5) != logo {
		t.Error("lerp of identical logos should return the same logo")
	}
	fromStart := FlutterLogoDecorationLerp(start, nil, 0.4)
	if fromStart.TextColor != start.TextColor || fromStart.Style != start.Style || fromStart.Margin != start.Margin.Mul(0.4) {
		t.Errorf("start to nil = %v", fromStart)
	}
	toEnd := FlutterLogoDecorationLerp(nil, end, 0.6)
	if toEnd.TextColor != end.TextColor || toEnd.Style != end.Style || toEnd.Margin != end.Margin.Mul(0.6) {
		t.Errorf("nil to end = %v", toEnd)
	}
	mid := FlutterLogoDecorationLerp(start, end, 0.5)
	if mid.TextColor != start.TextColor.Lerp(end.TextColor, 0.5) || mid.Margin != *EdgeInsetsLerp(&start.Margin, &end.Margin, 0.5) {
		t.Errorf("mid = %v", mid)
	}
	a, b := NewFlutterLogoDecoration(FlutterLogoDecorationOpts{Style: FlutterLogoStyleHorizontal}), NewFlutterLogoDecoration(FlutterLogoDecorationOpts{Style: FlutterLogoStyleStacked})
	if FlutterLogoDecorationLerp(a, b, 0.4).Style != FlutterLogoStyleHorizontal || FlutterLogoDecorationLerp(a, b, 0.5).Style != FlutterLogoStyleStacked {
		t.Error("the style should change at 0.5")
	}
	box := NewBoxDecoration()
	if !decorationsEqual(DecorationLerp(start, box, 0), start) || !decorationsEqual(DecorationLerp(start, box, 1), box) ||
		!decorationsEqual(DecorationLerp(box, end, 0), box) || !decorationsEqual(DecorationLerp(box, end, 1), end) {
		t.Error("Decoration lerp with a BoxDecoration is wrong")
	}
	if got := FlutterLogoDecorationLerp(a, b, 0.25); got.IsComplex() {
		t.Error("a logo in transition is not complex")
	}
	if !start.IsComplex() {
		t.Error("a logo not in transition is complex")
	}
}

func TestFlutterLogoDecorationString(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("release and profile builds describe objects by their short names")
	}
	start := NewFlutterLogoDecoration(FlutterLogoDecorationOpts{TextColor: foundation.Some(ui.NewColor(0xFFD4F144)), Style: FlutterLogoStyleStacked})
	end := NewFlutterLogoDecoration(FlutterLogoDecorationOpts{TextColor: foundation.Some(ui.NewColor(0xFF81D4FA)), Style: FlutterLogoStyleStacked})
	if got, want := start.String(), "FlutterLogoDecoration(textColor: "+ui.NewColor(0xFFD4F144).String()+", style: stacked)"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
	if got, want := FlutterLogoDecorationLerp(nil, end, 0.5).String(),
		"FlutterLogoDecoration(textColor: "+ui.NewColor(0xFF81D4FA).String()+", style: stacked, transition -1.0:0.5)"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}

func TestFlutterLogoPaintsTheMark(t *testing.T) {
	logo := NewFlutterLogoDecoration()
	ops := paintDecoration(t, logo, ui.Offset{}, cfgOf(100, 100, ui.TextDirectionLTR))
	paths := 0
	for _, op := range ops {
		if _, ok := op.(ui.OpDrawPath); ok {
			paths++
		}
	}
	if paths != 4 {
		t.Errorf("the mark should draw 4 paths, drew %d (ops: %d)", paths, len(ops))
	}
	faded := FlutterLogoDecorationLerp(nil, logo, 0.5)
	hasLayer := false
	for _, op := range paintDecoration(t, faded, ui.Offset{}, cfgOf(100, 100, ui.TextDirectionLTR)) {
		if _, ok := op.(ui.OpSaveLayer); ok {
			hasLayer = true
		}
	}
	if !hasLayer {
		t.Error("a faded logo should paint inside a layer")
	}
	empty := NewFlutterLogoDecoration(FlutterLogoDecorationOpts{Margin: NewEdgeInsetsAll(60)})
	if ops := paintDecoration(t, empty, ui.Offset{}, cfgOf(100, 100, ui.TextDirectionLTR)); len(ops) != 0 {
		t.Errorf("a logo with no space painted %v", ops)
	}
}

func TestDebugDescribeTransform(t *testing.T) {
	if got := DebugDescribeTransform(nil); len(got) != 1 || got[0] != "null" {
		t.Errorf("nil = %v", got)
	}
	m := ui.NewMatrix4TranslationValues(3, 4, 0)
	got := DebugDescribeTransform(m)
	want := []string{"[0] 1.0,0.0,0.0,3.0", "[1] 0.0,1.0,0.0,4.0", "[2] 0.0,0.0,1.0,0.0", "[3] 0.0,0.0,0.0,1.0"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTransformProperty(t *testing.T) {
	m := ui.NewMatrix4TranslationValues(3, 4, 0)
	p := NewTransformProperty("transform", m)
	multi := p.ValueToString(&foundation.TextTreeConfiguration{LineBreakProperties: true})
	if multi != "[0] 1.0,0.0,0.0,3.0\n[1] 0.0,1.0,0.0,4.0\n[2] 0.0,0.0,1.0,0.0\n[3] 0.0,0.0,0.0,1.0" {
		t.Errorf("multi-line = %q", multi)
	}
	single := p.ValueToString(&foundation.TextTreeConfiguration{LineBreakProperties: false})
	if single != "[1.0,0.0,0.0,3.0; 0.0,1.0,0.0,4.0; 0.0,0.0,1.0,0.0; 0.0,0.0,0.0,1.0]" {
		t.Errorf("single-line = %q", single)
	}
}

type fakeClipContext struct{ canvas *ui.Canvas }

func (f fakeClipContext) Canvas() *ui.Canvas { return f.canvas }

func TestClipContext(t *testing.T) {
	bounds := ui.NewRectFromLTWH(0, 0, 10, 10)
	path := ui.NewPath()
	path.AddRect(bounds)
	rr := ui.NewRRectFromRectXY(bounds, 2, 2)
	rse := ui.NewRSuperellipseFromRectXY(bounds, 2, 2)
	type call func(c ClipContext, clip ui.Clip, painter func())
	calls := map[string]struct {
		call call
		kind string
	}{
		"rect":          {func(c ClipContext, clip ui.Clip, p func()) { ClipRectAndPaint(c, bounds, clip, bounds, p) }, "OpClipRect"},
		"rrect":         {func(c ClipContext, clip ui.Clip, p func()) { ClipRRectAndPaint(c, rr, clip, bounds, p) }, "OpClipRRect"},
		"rsuperellipse": {func(c ClipContext, clip ui.Clip, p func()) { ClipRSuperellipseAndPaint(c, rse, clip, bounds, p) }, "OpClipRSuperellipse"},
		"path":          {func(c ClipContext, clip ui.Clip, p func()) { ClipPathAndPaint(c, path, clip, bounds, p) }, "OpClipPath"},
	}
	behaviors := []struct {
		clip  ui.Clip
		clips bool
		layer bool
		aa    bool
	}{
		{ui.ClipNone, false, false, false}, {ui.ClipHardEdge, true, false, false},
		{ui.ClipAntiAlias, true, false, true}, {ui.ClipAntiAliasWithSaveLayer, true, true, true},
	}
	for name, cc := range calls {
		for _, b := range behaviors {
			t.Run(name+" "+b.clip.String(), func(t *testing.T) {
				painted := false
				ops := recordOps(func(c *ui.Canvas) {
					cc.call(fakeClipContext{c}, b.clip, func() { painted = true; c.DrawRect(bounds, ui.NewPaint()) })
				})
				if !painted {
					t.Error("the painter wasn't called")
				}
				names := make([]string, len(ops))
				clips, layers := 0, 0
				for i, op := range ops {
					names[i] = foundation.RuntimeTypeName(op)
					switch names[i] {
					case "ui.OpClipRect", "ui.OpClipRRect", "ui.OpClipRSuperellipse", "ui.OpClipPath", "OpClipRect", "OpClipRRect", "OpClipRSuperellipse", "OpClipPath":
						clips++
					case "ui.OpSaveLayer", "OpSaveLayer":
						layers++
					}
				}
				if (clips == 1) != b.clips || (layers == 1) != b.layer {
					t.Errorf("ops = %v", names)
				}
				if names[0] != "ui.OpSave" && names[0] != "OpSave" {
					t.Errorf("the first op should be a save: %v", names)
				}
			})
		}
	}
}
