package cupertino_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The tests of CupertinoFormRow and CupertinoFormSection, ported from cupertino/form_row_test.dart and
// cupertino/form_section_test.dart (step 34.8).

func formRowApp(row widgets.Widget, opts ...cupertino.CupertinoAppOpts) widgets.Widget {
	o := cupertino.CupertinoAppOpts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	o.Home = widgets.NewCenter(row)
	return testApp(o)
}

func TestCupertinoFormRow(t *testing.T) {
	t.Run("shows the prefix, the child, the helper and the error", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(formRowApp(cupertino.NewCupertinoFormRow(cupertino.NewCupertinoTextField(), cupertino.CupertinoFormRowOpts{
			Prefix: widgets.NewText("Enter Value"), Helper: widgets.NewText("Helper"), Error: widgets.NewText("Error")})))
		for _, text := range []string{"Enter Value", "Helper", "Error"} {
			if ts.Count(gluttertest.ByText(text)) != 1 {
				t.Errorf("%q is not shown once", text)
			}
		}
		if ts.Count(gluttertest.ByType[*cupertino.CupertinoTextField]()) != 1 {
			t.Error("the child is not shown")
		}
	})
	for _, dir := range []struct {
		name        string
		d           ui.TextDirection
		prefixFirst bool
	}{{"LTR puts the child after the prefix", ui.TextDirectionLTR, true}, {"RTL puts the prefix after the child", ui.TextDirectionRTL, false}} {
		t.Run(dir.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{})
			ts.PumpWidget(formRowApp(widgets.NewDirectionality(dir.d, cupertino.NewCupertinoFormRow(cupertino.NewCupertinoTextField(),
				cupertino.CupertinoFormRowOpts{Prefix: widgets.NewText("Enter Value")}))))
			prefixAfterChild := ts.TopLeft(gluttertest.ByType[*widgets.Text]()).Dx > ts.TopLeft(gluttertest.ByType[*cupertino.CupertinoTextField]()).Dx
			if prefixAfterChild == dir.prefixFirst {
				t.Errorf("the prefix is after the child: %v", prefixAfterChild)
			}
		})
	}
	t.Run("shows the helper above the error", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(formRowApp(cupertino.NewCupertinoFormRow(cupertino.NewCupertinoTextField(), cupertino.CupertinoFormRowOpts{
			Helper: widgets.NewText("Helper"), Error: cupertino.NewCupertinoActivityIndicator()})))
		if ts.TopLeft(gluttertest.ByType[*cupertino.CupertinoActivityIndicator]()).Dy <= ts.TopLeft(gluttertest.ByText("Helper")).Dy {
			t.Error("the error is not under the helper")
		}
	})
	t.Run("shows the helper in the label color and the error in red", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(formRowApp(cupertino.NewCupertinoFormRow(cupertino.NewCupertinoTextField(), cupertino.CupertinoFormRowOpts{
			Helper: widgets.NewText("Helper"), Error: widgets.NewText("Error")})))
		bc := ts.Element(gluttertest.ByType[*cupertino.CupertinoFormRow]())
		helper := ts.Widget(gluttertest.First(gluttertest.ByType[*widgets.DefaultTextStyle]())).(*widgets.DefaultTextStyle)
		if got, want := helper.Style().Color.Value(), cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.Label, bc); got != want {
			t.Errorf("the helper is %v, want %v", got, want)
		}
		errStyle := ts.Widget(gluttertest.At(gluttertest.ByType[*widgets.DefaultTextStyle](), ts.Count(gluttertest.ByType[*widgets.DefaultTextStyle]())-1)).(*widgets.DefaultTextStyle)
		if got := errStyle.Style().Color.Value(); got != cupertino.CupertinoColors.DestructiveRed.Color {
			t.Errorf("the error is %v, want %v", got, cupertino.CupertinoColors.DestructiveRed.Color)
		}
	})
	t.Run("adapts to the dark mode of the app", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		for _, b := range []ui.Brightness{ui.BrightnessLight, ui.BrightnessDark} {
			ts.PumpWidget(formRowApp(cupertino.NewCupertinoFormRow(cupertino.NewCupertinoTextField(), cupertino.CupertinoFormRowOpts{
				Prefix: widgets.NewText("Prefix"), Helper: widgets.NewText("Helper")}),
				cupertino.CupertinoAppOpts{Theme: cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: b})}))
			want := cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.Label, ts.Element(gluttertest.ByType[*cupertino.CupertinoFormRow]()))
			for _, text := range []string{"Helper", "Prefix"} {
				if got := paragraphStyle(t, ts, text).Color; !got.IsSome() || got.Value() != want {
					t.Errorf("brightness %v: %s is %v, want %v", b, text, got, want)
				}
			}
		}
	})
	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(formRowApp(widgets.NewSizedBoxShrink(cupertino.NewCupertinoFormRow(widgets.NewText("X")))))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoFormRow]()); got != (ui.Size{}) {
			t.Errorf("the size is %v", got)
		}
	})
}

func formSectionApp(section widgets.Widget) widgets.Widget { return formRowApp(section) }

func formChild() widgets.Widget {
	return cupertino.NewCupertinoFormRow(widgets.NewText("Row"))
}

func TestCupertinoFormSection(t *testing.T) {
	t.Run("shows the header and the footer", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(formSectionApp(cupertino.NewCupertinoFormSection([]widgets.Widget{formChild()},
			cupertino.CupertinoFormSectionOpts{Header: widgets.NewText("Header"), Footer: widgets.NewText("Footer")})))
		if !ts.Exists(gluttertest.ByText("Header")) || !ts.Exists(gluttertest.ByText("Footer")) {
			t.Error("the header or the footer is missing")
		}
	})
	for _, c := range []struct {
		name  string
		inset bool
		rows  int
		want  int
	}{
		{"long dividers in the edge-to-edge section, one row", false, 1, 3}, {"long dividers in the edge-to-edge section, two rows", false, 2, 5},
		{"no long dividers in the inset grouped section, one row", true, 1, 1}, {"no long dividers in the inset grouped section, two rows", true, 2, 3},
	} {
		t.Run(c.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{})
			var rows []widgets.Widget
			for range c.rows {
				rows = append(rows, formChild())
			}
			section := cupertino.NewCupertinoFormSection(rows)
			if c.inset {
				section = cupertino.NewCupertinoFormSectionInsetGrouped(rows)
			}
			ts.PumpWidget(formSectionApp(section))
			if got := len(sectionColumn(t, ts).Children()); got != c.want {
				t.Errorf("%d children, want %d", got, c.want)
			}
		})
	}
	t.Run("clips the rows only when it is asked to", func(t *testing.T) {
		for _, clip := range []ui.Clip{ui.ClipAntiAlias, ui.ClipNone} {
			ts := dialogTester(t, ui.Size{})
			ts.PumpWidget(formSectionApp(cupertino.NewCupertinoFormSection([]widgets.Widget{formChild()}, cupertino.CupertinoFormSectionOpts{ClipBehavior: clip})))
			want := 0
			if clip == ui.ClipAntiAlias {
				want = 1
			}
			if n := ts.Count(gluttertest.ByType[*widgets.ClipRSuperellipse]()); n != want {
				t.Errorf("clip %v: %d clips, want %d", clip, n, want)
			}
		}
	})
	t.Run("sets the background color", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		blue := cupertino.CupertinoColors.SystemBlue
		ts.PumpWidget(formSectionApp(cupertino.NewCupertinoFormSection([]widgets.Widget{formChild()}, cupertino.CupertinoFormSectionOpts{BackgroundColor: blue})))
		box := ts.Widget(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoFormSection](), gluttertest.ByType[*widgets.DecoratedBox]()))).(*widgets.DecoratedBox)
		deco := box.Decoration().(*painting.BoxDecoration)
		if want := cupertino.CupertinoDynamicColorResolve(blue, ts.Element(gluttertest.ByType[*cupertino.CupertinoFormSection]())); deco.Color != foundation.Some(want) {
			t.Errorf("the background is %v, want %v", deco.Color, want)
		}
	})
	t.Run("does not double up the padding of the header", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		header := widgets.NewText("Header")
		ts.PumpWidget(formSectionApp(cupertino.NewCupertinoFormSection([]widgets.Widget{cupertino.NewCupertinoTextFormFieldRow()},
			cupertino.CupertinoFormSectionOpts{Header: header})))
		if got := ts.TopLeft(gluttertest.ByWidget(header)); got != (ui.Offset{Dx: 20, Dy: 22}) {
			t.Errorf("the header is at %v, want (20, 22)", got)
		}
	})
	t.Run("does not double up the padding of the footer", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		footer := widgets.NewText("Footer")
		row := cupertino.NewCupertinoTextFormFieldRow()
		ts.PumpWidget(formSectionApp(cupertino.NewCupertinoFormSection([]widgets.Widget{row}, cupertino.CupertinoFormSectionOpts{Footer: footer})))
		got := ts.TopLeft(gluttertest.ByWidget(footer))
		// The footer is under the row, with the padding of 20 at its start and none above it.
		rowBottom := ts.TopLeft(gluttertest.ByWidget(row)).Dy + ts.Size(gluttertest.ByWidget(row)).Height
		if got.Dx != 20 || math.Abs(got.Dy-rowBottom) > 12 || got.Dy < rowBottom {
			t.Errorf("the footer is at %v, the row ends at %v", got, rowBottom)
		}
	})
	t.Run("sets a custom margin", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		child := cupertino.NewCupertinoTextFormFieldRow()
		ts.PumpWidget(formSectionApp(cupertino.NewCupertinoFormSection([]widgets.Widget{child}, cupertino.CupertinoFormSectionOpts{Margin: painting.NewEdgeInsetsAll(35)})))
		got := ts.TopLeft(gluttertest.ByWidget(child))
		if math.Abs(got.Dx-35) > 1 || math.Abs(got.Dy-(22+35)) > 1 {
			t.Errorf("the child is at %v, want (35, 57)", got)
		}
	})
	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(formSectionApp(widgets.NewSizedBoxShrink(cupertino.NewCupertinoFormSection([]widgets.Widget{widgets.NewText("X"), widgets.NewText("Y")}))))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoFormSection]()); got != (ui.Size{}) {
			t.Errorf("the size is %v", got)
		}
	})
	t.Run("a section with no child is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		err := recoverError(func() { cupertino.NewCupertinoFormSection(nil) })
		requireErrorTag(t, err, "NewCupertinoFormSection->newFormSection.CupertinoFormSectionEmpty")
		err = recoverError(func() { cupertino.NewCupertinoFormSectionInsetGrouped(nil) })
		requireErrorTag(t, err, "NewCupertinoFormSectionInsetGrouped->newFormSection.CupertinoFormSectionEmpty")
	})
}
