package cupertino_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The tests of CupertinoListSection, ported from cupertino/list_section_test.dart (step 34.8).

func listTile() widgets.Widget {
	return cupertino.NewCupertinoListTile(widgets.NewText("CupertinoListTile"))
}

func sectionApp(section widgets.Widget) widgets.Widget {
	return testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(section)})
}

// sectionColumn is the Column that holds the rows and the dividers: the second Column of the app.
func sectionColumn(t *testing.T, ts *gluttertest.Tester) *widgets.Column {
	t.Helper()
	c, ok := ts.Widget(gluttertest.At(gluttertest.ByType[*widgets.Column](), 1)).(*widgets.Column)
	if !ok {
		t.Fatal("the second Column is missing")
	}
	return c
}

func TestCupertinoListSection(t *testing.T) {
	t.Run("shows header and footer", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(sectionApp(cupertino.NewCupertinoListSection(cupertino.CupertinoListSectionOpts{
			Header: widgets.NewText("Header"), Footer: widgets.NewText("Footer"), Children: []widgets.Widget{listTile()}})))
		if !ts.Exists(gluttertest.ByText("Header")) || !ts.Exists(gluttertest.ByText("Footer")) {
			t.Error("the header or the footer is missing")
		}
	})
	dividers := []struct {
		name      string
		inset     bool
		rows      int
		wantChild int
	}{
		{"long dividers in the edge-to-edge section, one row", false, 1, 3},
		{"long dividers in the edge-to-edge section, two rows", false, 2, 5},
		{"no long dividers in the inset grouped section, one row", true, 1, 1},
		{"no long dividers in the inset grouped section, two rows", true, 2, 3},
	}
	for _, c := range dividers {
		t.Run(c.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{})
			var rows []widgets.Widget
			for range c.rows {
				rows = append(rows, listTile())
			}
			o := cupertino.CupertinoListSectionOpts{Children: rows}
			section := cupertino.NewCupertinoListSection(o)
			if c.inset {
				section = cupertino.NewCupertinoListSectionInsetGrouped(o)
			}
			ts.PumpWidget(sectionApp(section))
			if got := len(sectionColumn(t, ts).Children()); got != c.wantChild {
				t.Errorf("%d children, want %d", got, c.wantChild)
			}
		})
	}
	t.Run("sets the background color of the section", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		blue := cupertino.CupertinoColors.SystemBlue
		ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewMediaQuery(widgets.MediaQueryData{DevicePixelRatio: 1},
			cupertino.NewCupertinoListSection(cupertino.CupertinoListSectionOpts{BackgroundColor: blue, Children: []widgets.Widget{listTile()}}))))
		box := ts.Widget(gluttertest.First(gluttertest.ByType[*widgets.DecoratedBox]())).(*widgets.DecoratedBox)
		deco, ok := box.Decoration().(*painting.BoxDecoration)
		if !ok {
			t.Fatalf("the decoration is %T", box.Decoration())
		}
		want := cupertino.CupertinoDynamicColorResolve(blue, ts.Element(gluttertest.ByType[*cupertino.CupertinoListSection]()))
		if got := deco.Color; !got.IsSome() || got.Value() != want {
			t.Errorf("the box is %v, want %v", got, want)
		}
	})
	t.Run("clipBehavior clips the rows", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(sectionApp(cupertino.NewCupertinoListSection(cupertino.CupertinoListSectionOpts{
			ClipBehavior: ui.ClipAntiAlias, Children: []widgets.Widget{listTile()}})))
		if n := ts.Count(gluttertest.ByType[*widgets.ClipRSuperellipse]()); n != 1 {
			t.Errorf("%d clips, want 1", n)
		}
	})
	t.Run("no clipBehavior does not clip the rows", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(sectionApp(cupertino.NewCupertinoListSection(cupertino.CupertinoListSectionOpts{Children: []widgets.Widget{listTile()}})))
		if n := ts.Count(gluttertest.ByType[*widgets.ClipRSuperellipse]()); n != 0 {
			t.Errorf("%d clips, want 0", n)
		}
	})
	separators := []struct {
		name  string
		color ui.ColorValue
	}{
		{"respects separatorColor", ui.NewColor(0xFF8FC133)},
		{"the separator color defaults to CupertinoColors.Separator", nil},
	}
	for _, c := range separators {
		t.Run(c.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{})
			ts.PumpWidget(sectionApp(cupertino.NewCupertinoListSection(cupertino.CupertinoListSectionOpts{
				SeparatorColor: c.color, Children: []widgets.Widget{listTile(), listTile()}})))
			bc := ts.Element(gluttertest.ByType[*cupertino.CupertinoListSection]())
			var src ui.ColorValue = cupertino.CupertinoColors.Separator
			if c.color != nil {
				src = c.color
			}
			want := cupertino.CupertinoDynamicColorResolve(src, bc)
			found := 0
			for _, child := range sectionColumn(t, ts).Children() {
				if box, ok := child.(*widgets.Container); ok {
					found++
					if got := box.Color(); !got.IsSome() || got.Value() != want {
						t.Errorf("a divider is %v, want %v", got, want)
					}
				}
			}
			if found == 0 {
				t.Error("no divider was found")
			}
		})
	}
	t.Run("margin", func(t *testing.T) {
		// The row is under the top margin, the header and its bottom margin of 6; the font of the test decides the height of the header.
		for _, c := range []struct {
			name   string
			margin painting.EdgeInsetsGeometry
			inset  float64
		}{
			{"none by default", nil, 0},
			{"custom", painting.NewEdgeInsetsAll(10), 10},
		} {
			t.Run(c.name, func(t *testing.T) {
				ts := dialogTester(t, ui.Size{})
				row := listTile()
				ts.PumpWidget(sectionApp(cupertino.NewCupertinoListSection(cupertino.CupertinoListSectionOpts{
					Header: widgets.NewText("Header"), Margin: c.margin, Children: []widgets.Widget{row}})))
				header := gluttertest.ByText("Header")
				want := ui.Offset{Dx: c.inset, Dy: ts.TopLeft(header).Dy + ts.Size(header).Height + 6 + c.inset}
				got := ts.TopLeft(gluttertest.ByWidget(row))
				if math.Abs(got.Dx-want.Dx) > 1 || math.Abs(got.Dy-want.Dy) > 1 {
					t.Errorf("the row is at %v, want %v", got, want)
				}
			})
		}
	})
	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewSizedBoxShrink(
			cupertino.NewCupertinoListSection(cupertino.CupertinoListSectionOpts{Header: widgets.NewText("X")})))}))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoListSection]()); got != (ui.Size{}) {
			t.Errorf("the size is %v", got)
		}
	})
	t.Run("a section with nothing in it is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		err := recoverError(func() { cupertino.NewCupertinoListSection() })
		checkErrorTag(t, err, "NewCupertinoListSection->newListSection.CupertinoListSectionEmpty")
	})
}
