package widgets_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestDebugDumpsPrintTheTreesThroughTheDebugPrint(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the dumps print in debug builds only")
	}
	var lines []string
	debug := widgets.NewDebugOptions()
	debug.Print = func(s string) { lines = append(lines, s) }
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 100}, Debug: debug})
	ts.EnsureSemantics()
	label := foundation.NewKey("label")
	ts.PumpWidget(widgets.NewSemantics(widgets.NewSizedBox(widgets.NewColoredBox(ui.NewColor(0xFF336699), nil), widgets.SizedBoxOpts{
		Width: foundation.Some(50.0), Height: foundation.Some(20.0), Key: label,
	}), widgets.SemanticsOpts{Label: foundation.Some("dump me"), TextDirection: ui.TextDirectionLTR}))
	bc := ts.Element(gluttertest.ByKey(label))

	tests := []struct {
		name string
		dump func(widgets.BuildContext)
		want []string
	}{
		{"app", widgets.DebugDumpApp, []string{"SizedBox", "ColoredBox", "Semantics"}},
		{"render tree", widgets.DebugDumpRenderTree, []string{"RenderView", "RenderSemanticsAnnotations", "RenderConstrainedBox"}},
		{"layer tree", widgets.DebugDumpLayerTree, []string{"TransformLayer", "PictureLayer"}},
		{"semantics tree", func(bc widgets.BuildContext) { widgets.DebugDumpSemanticsTree(bc) }, []string{"SemanticsNode", "dump me"}},
		{"pipeline owners", widgets.DebugDumpPipelineOwnerTree, []string{"PipelineOwner"}},
		{"focus tree", widgets.DebugDumpFocusTree, []string{"FocusManager"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines = nil
			tt.dump(bc)
			if len(lines) != 1 {
				t.Fatalf("%d lines printed, want one dump", len(lines))
			}
			for _, w := range tt.want {
				if !strings.Contains(lines[0], w) {
					t.Errorf("the dump does not mention %q:\n%s", w, lines[0])
				}
			}
		})
	}
}
