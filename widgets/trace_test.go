package widgets_test

import (
	"bytes"
	rtrace "runtime/trace"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// tracedWidget is a stateless widget that the app (a test package here) declares.
type tracedWidget struct {
	widgets.StatelessWidget
}

func newTracedWidget() *tracedWidget {
	return &tracedWidget{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{})}
}

func (w *tracedWidget) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewColoredBox(ui.NewColor(0xFF336699), nil)
}

// TestFramePhasesAreRegionsOfTheExecutionTrace records a runtime trace around a frame
// and looks for the names of the regions in it (the trace keeps them as strings).
func TestFramePhasesAreRegionsOfTheExecutionTrace(t *testing.T) {
	debug := widgets.NewDebugOptions()
	debug.ProfileBuildsEnabledUserWidgets = true
	var buf bytes.Buffer
	if err := rtrace.Start(&buf); err != nil {
		t.Skipf("a trace is being recorded already: %v", err)
	}
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 100, Height: 100}, Debug: debug})
	ts.PumpWidget(newTracedWidget())
	rtrace.Stop()
	regions := []string{"Animate", "Frame", "BUILD", "LAYOUT", "UPDATING COMPOSITING BITS", "PAINT", "COMPOSITING", "FINALIZE TREE"}
	if foundation.DebugMode {
		// The regions of the widgets that build are for debug builds.
		regions = append(regions, "tracedWidget")
	}
	for _, region := range regions {
		if !bytes.Contains(buf.Bytes(), []byte(region)) {
			t.Errorf("the trace has no region %q", region)
		}
	}
}
