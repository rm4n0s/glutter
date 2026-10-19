package widgets

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func TestScaledInlineWidgetNeedsUnboundedHeight(t *testing.T) {
	skipUnlessDebug(t)
	r := newRenderScaledInlineWidget(ui.PlaceholderAlignmentBottom, ui.TextBaselineAlphabetic, 2)
	bounded := rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(100.0), MaxHeight: foundation.Some(50.0)})
	requireTag(t, "InlineScaleBoundedHeight", func() { r.ComputeDryLayout(bounded) })
}
