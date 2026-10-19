package widgets_test

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

type fakeRegistrar struct{}

func (fakeRegistrar) Add(rendering.Selectable)    {}
func (fakeRegistrar) Remove(rendering.Selectable) {}

// placeholderOnly is a PlaceholderSpan that is not a WidgetSpan.
type placeholderOnly struct{ *painting.TextSpan }

func (placeholderOnly) Alignment() ui.PlaceholderAlignment { return ui.PlaceholderAlignmentBottom }
func (placeholderOnly) Baseline() ui.TextBaseline          { return ui.TextBaselineAlphabetic }

func TestTextWidgetErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		fn   func()
		tag  string
	}{
		{"selection registrar without a color", func() {
			NewRichText(painting.NewTextSpan(), RichTextOpts{SelectionRegistrar: fakeRegistrar{}})
		}, "RichTextNoSelectionColor"},
		{"placeholder that is not a widget span", func() {
			ExtractWidgetsFromInlineSpan(placeholderOnly{painting.NewTextSpan()}, painting.TextScalerNoScaling)
		}, "PlaceholderSpanNotWidgetSpan"},
		{"widget span built without dimensions", func() {
			b := ui.NewParagraphBuilder(renderingtest.BundledTextEngine(), nil)
			NewWidgetSpan(newLeaf()).Build(b)
		}, "WidgetSpanNoDimensions"},
		{"widget span built with too few dimensions", func() {
			b := ui.NewParagraphBuilder(renderingtest.BundledTextEngine(), nil)
			NewWidgetSpan(newLeaf()).Build(b, painting.SpanBuildOpts{Dimensions: []painting.PlaceholderDimensions{}})
		}, "WidgetSpanDimensionsMissing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := panicOf(tt.fn)
			if err == nil {
				t.Fatal("no panic")
			}
			_ = requireErrorTag(t, err.(error), tt.tag)
		})
	}
}

func TestInlineWidgetScaleMustBePositive(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	mk := func(fontSize float64) Widget {
		return NewRichText(painting.NewTextSpan(painting.TextSpanOpts{Style: style(fontSize), Children: []painting.InlineSpan{NewWidgetSpan(newBox(10, 10))}}))
	}
	pumpLoose(ts, mk(10), ui.TextDirectionLTR)
	pumpLoose(ts, mk(0), ui.TextDirectionLTR)
	// The failed update leaves the paragraph without its placeholder, which the
	// layout and the paint report too.
	errs := ts.TakeErrors()
	if len(errs) == 0 {
		t.Fatal("no error reported")
	}
	_ = requireErrorTag(t, errs[0], "InlineScaleInvalid")
}

func TestReplaceGestureRecognizersOnlyDuringLayout(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewRawGestureDetector(target()), ui.TextDirectionLTR)
	state := gluttertest.StateOf[*RawGestureDetectorState](ts, gluttertest.ByType[*RawGestureDetector]())
	err := panicOf(func() { state.ReplaceGestureRecognizers(map[reflect.Type]GestureRecognizerFactory{}) })
	if err == nil {
		t.Fatal("no panic")
	}
	_ = requireErrorTag(t, err.(error), "GestureDetectorNotDuringLayout")
}

func TestReplaceSemanticsActions(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	pumpLoose(ts, NewGestureDetector(target(), GestureDetectorOpts{OnTap: func() {}}), ui.TextDirectionLTR)
	state := gluttertest.StateOf[*RawGestureDetectorState](ts, gluttertest.ByType[*RawGestureDetector]())
	state.ReplaceSemanticsActions(map[ui.SemanticsAction]struct{}{ui.SemanticsActionTap: {}})
	h := semanticsHandlerOf(ts)
	if got := h.ValidActions(); len(got) != 1 {
		t.Errorf("valid actions = %v", got)
	}
	// A detector without semantics ignores the call.
	pumpLoose(ts, NewGestureDetector(target(), GestureDetectorOpts{OnTap: func() {}, ExcludeFromSemantics: true}), ui.TextDirectionLTR)
	gluttertest.StateOf[*RawGestureDetectorState](ts, gluttertest.ByType[*RawGestureDetector]()).ReplaceSemanticsActions(nil)
}

func TestPrecacheImageReportsToTheBindingByDefault(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var done <-chan struct{}
	pumpLoose(ts, NewBuilder(func(bc BuildContext) Widget {
		if done == nil {
			done = PrecacheImage(painting.NewMemoryImage([]byte("junk")), bc)
		}
		return newLeaf()
	}), ui.TextDirectionLTR)
	var errs []error
	waitFor(t, ts, func() bool { errs = append(errs, ts.TakeErrors()...); return len(errs) > 0 })
	_ = requireErrorTag(t, errs[0], "ImagePrecacheFailed")
}
