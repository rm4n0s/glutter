package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from the group 'cupertino handles' of packages/flutter/test/cupertino/text_selection_test.dart
// (the goldens of the custom and the transparent handle) and check the geometry of the handles, which the iOS
// selection tests read from the field (the cases about the handles that scale with rich text need a controller with
// styled spans, and the geometry that they check is the one that is tested here). The group 'canSelectAll' is about
// the deprecated canSelectAll of the controls, which is not ported (SPEC §12.10, step 34.5).

func TestCupertinoTextSelectionHandleGeometry(t *testing.T) {
	controls := cupertino.CupertinoTextSelectionControls{}
	for _, lineHeight := range []float64{10, 18, 40} {
		size := controls.GetHandleSize(lineHeight)
		if size.Width != 12 || size.Height != lineHeight+12-1.5 {
			t.Errorf("handle size at %v: %v", lineHeight, size)
		}
		left := controls.GetHandleAnchor(rendering.TextSelectionHandleTypeLeft, lineHeight)
		right := controls.GetHandleAnchor(rendering.TextSelectionHandleTypeRight, lineHeight)
		collapsed := controls.GetHandleAnchor(rendering.TextSelectionHandleTypeCollapsed, lineHeight)
		if left != (ui.Offset{Dx: 6, Dy: size.Height}) {
			t.Errorf("left anchor at %v: %v", lineHeight, left)
		}
		if right != (ui.Offset{Dx: 6, Dy: size.Height - 12 + 1.5}) {
			t.Errorf("right anchor at %v: %v", lineHeight, right)
		}
		if collapsed.Dx != 6 || collapsed.Dy <= lineHeight || collapsed.Dy >= size.Height {
			t.Errorf("collapsed anchor at %v: %v", lineHeight, collapsed)
		}
	}
}

func TestCupertinoTextSelectionHandleGoldens(t *testing.T) {
	for _, tt := range []struct {
		name  string
		color ui.Color
	}{{"cupertino.handle.custom", ui.NewColor(0xFF9C27B0)}, {"cupertino.handle.transparent", ui.NewColor(0x550000AA)}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Theme: testTheme(cupertino.CupertinoThemeDataOpts{SelectionHandleColor: tt.color}),
				Home: widgets.NewColoredBox(cupertino.CupertinoColors.White, widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 70),
					widgets.NewFittedBox(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
						return cupertino.CupertinoTextSelectionControls{}.BuildHandle(bc, rendering.TextSelectionHandleTypeRight, 10, nil)
					}))))}))
			ts.MatchesGolden(tt.name)
		})
	}
}

func TestCupertinoTextSelectionHandleAtTheCollapsedTypeIsEmpty(t *testing.T) {
	ts := gluttertest.New(t)
	var size ui.Size
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		size = cupertino.CupertinoTextSelectionControls{}.GetHandleSize(18)
		return cupertino.CupertinoTextSelectionControls{}.BuildHandle(bc, rendering.TextSelectionHandleTypeCollapsed, 18, nil)
	}))}))
	if got := ts.Size(gluttertest.ByType[*widgets.SizedBox]()); got != size {
		t.Errorf("the collapsed handle is %v, want the handle size %v", got, size)
	}
}
