//go:build !glutter_release && !glutter_profile

package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// unbalancedPainter is a box painter that saves the canvas without restoring it.
type unbalancedPainter struct{}

func (unbalancedPainter) Paint(canvas *ui.Canvas, _ ui.Offset, _ painting.ImageConfiguration) {
	canvas.Save()
}
func (unbalancedPainter) Dispose() {}

// unbalancedDecoration makes unbalancedPainters.
type unbalancedDecoration struct{ *painting.BoxDecoration }

func (unbalancedDecoration) CreateBoxPainter(func()) painting.BoxPainter { return unbalancedPainter{} }

// misbehavedPainter is a custom painter that saves or restores too often.
type misbehavedPainter struct {
	rendering.CustomPainterBase
	extraSaves, extraRestores int
}

func (p *misbehavedPainter) Paint(canvas *ui.Canvas, _ ui.Size) {
	for i := 0; i < p.extraSaves; i++ {
		canvas.Save()
	}
	for i := 0; i < p.extraRestores; i++ {
		canvas.Restore()
	}
}
func (*misbehavedPainter) ShouldRepaint(rendering.CustomPainter) bool { return false }

func TestRenderObjectErrorRoutes(t *testing.T) {
	loose := rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}
	tests := []struct {
		tag  string
		call func(t *testing.T)
	}{
		{"LimitedBoxNegative", func(t *testing.T) {
			rendering.NewRenderLimitedBox(nil, rendering.RenderLimitedBoxOpts{MaxWidth: some(-1)})
		}},
		{"LimitedBoxNegative", func(t *testing.T) { rendering.NewRenderLimitedBox(nil).SetMaxHeight(-2) }},
		{"AspectRatioInvalid", func(t *testing.T) { rendering.NewRenderAspectRatio(0, nil) }},
		{"AspectRatioInvalid", func(t *testing.T) { rendering.NewRenderAspectRatio(1, nil).SetAspectRatio(inf) }},
		{"AspectRatioUnbounded", func(t *testing.T) {
			rendering.NewRenderAspectRatio(1, nil).GetDryLayout(rendering.BoxConstraints{MaxWidth: inf, MaxHeight: inf})
		}},
		{"IntrinsicStepInvalid", func(t *testing.T) {
			rendering.NewRenderIntrinsicWidth(nil, rendering.RenderIntrinsicWidthOpts{StepWidth: some(0)})
		}},
		{"IntrinsicStepInvalid", func(t *testing.T) { rendering.NewRenderIntrinsicWidth(nil).SetStepHeight(some(-1)) }},
		{"OpacityOutOfRange", func(t *testing.T) {
			rendering.NewRenderOpacity(nil, rendering.RenderOpacityOpts{Opacity: some(1.5)})
		}},
		{"OpacityOutOfRange", func(t *testing.T) { rendering.NewRenderOpacity(nil).SetOpacity(-0.1) }},
		{"ElevationNegative", func(t *testing.T) {
			rendering.NewRenderPhysicalModel(ui.NewColor(0), nil, rendering.RenderPhysicalModelOpts{Elevation: -1})
		}},
		{"ElevationNegative", func(t *testing.T) { rendering.NewRenderPhysicalModel(ui.NewColor(0), nil).SetElevation(-1) }},
		{"PaddingNegative", func(t *testing.T) { rendering.NewRenderPadding(painting.EdgeInsets{Left: -1}, nil) }},
		{"PaddingNegative", func(t *testing.T) {
			rendering.NewRenderPadding(painting.EdgeInsets{}, nil).SetPadding(painting.EdgeInsets{Top: -1})
		}},
		{"SizeFactorNegative", func(t *testing.T) {
			rendering.NewRenderPositionedBox(nil, rendering.RenderPositionedBoxOpts{WidthFactor: some(-1)})
		}},
		{"SizeFactorNegative", func(t *testing.T) {
			rendering.NewRenderFractionallySizedOverflowBox(nil).SetHeightFactor(some(-1))
		}},
		{"RepaintBoundaryNotPainted", func(t *testing.T) { _, _ = rendering.NewRenderRepaintBoundary(nil).ToImage(1) }},
		{"OffstagePerformResize", func(t *testing.T) {
			rendering.NewRenderOffstage(nil, rendering.RenderOffstageOpts{Offstage: foundation.Some(false)}).PerformResize()
		}},
		{"RenderObjectNotChild", func(t *testing.T) { rendering.NewRenderOpacity(nil).PaintsChild(newSizedBox(1, 1)) }},
		{"RenderObjectNotChild", func(t *testing.T) { rendering.NewRenderOffstage(nil).PaintsChild(newSizedBox(1, 1)) }},
		{"ConstraintsTransformNotNormalized", func(t *testing.T) {
			bad := func(rendering.BoxConstraints) rendering.BoxConstraints {
				return rendering.BoxConstraints{MinWidth: 10, MaxWidth: 5, MaxHeight: inf}
			}
			layout(t, newHarness(t), rendering.NewRenderConstraintsTransformBox(painting.AlignmentCenter, ui.TextDirectionUnspecified, bad, newSizedBox(1, 1)),
				withConstraints(loose))
		}},
		{"FollowerLeaderSizeMissing", func(t *testing.T) {
			link := rendering.NewLayerLink()
			rendering.NewLeaderLayer(link).Attach(new(int))
			follower := rendering.NewRenderFollowerLayer(link, newSizedBox(1, 1), rendering.RenderFollowerLayerOpts{
				LeaderAnchor: foundation.Some(painting.AlignmentCenter)})
			layout(t, newHarness(t), follower, atPhase(withConstraints(loose), phasePaint))
		}},
		{"DecorationPainterSaveMismatch", func(t *testing.T) {
			decoration := unbalancedDecoration{painting.NewBoxDecoration()}
			layout(t, newHarness(t), rendering.NewRenderDecoratedBox(decoration, nil), atPhase(withConstraints(loose), phasePaint))
		}},
		{"CustomPainterSaveUnbalanced", func(t *testing.T) {
			p := &misbehavedPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), extraSaves: 2}
			layout(t, newHarness(t), rendering.NewRenderCustomPaint(nil, rendering.RenderCustomPaintOpts{Painter: p}), atPhase(withConstraints(loose), phasePaint))
		}},
		{"CustomPainterRestoreUnbalanced", func(t *testing.T) {
			p := &misbehavedPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), extraRestores: 1}
			layout(t, newHarness(t), rendering.NewRenderCustomPaint(nil, rendering.RenderCustomPaintOpts{ForegroundPainter: p}), atPhase(withConstraints(loose), phasePaint))
		}},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) { requireTag(t, tc.tag, func() { tc.call(t) }) })
	}
}
