package cupertino_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/search_field_test.dart. The cases that read a property
// of the inner CupertinoTextField that has no observable effect (autocorrect, smartQuotesType, smartDashesType,
// enableIMEPersonalizedLearning, textInputAction) are not ported, as the text field has no accessor for them
// (SPEC §12.10, step 34.5), and 'Focused search field hides prefix in higher accessibility text scale modes' is
// covered by TestCupertinoSearchTextFieldTextScale. The scrolling cases use a SliverPersistentHeader that shrinks
// its child as Flutter's SliverResizingHeader does, which the widgets library does not have.

func searchFinder() gluttertest.Finder {
	return gluttertest.ByType[*cupertino.CupertinoSearchTextField]()
}

func searchApp(ts *gluttertest.Tester, field *cupertino.CupertinoSearchTextField) {
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(field)}))
}

func inSearch(f gluttertest.Finder) gluttertest.Finder {
	return gluttertest.Descendant(searchFinder(), f, gluttertest.DescendantOpts{})
}

var (
	searchIcon = gluttertest.ByIcon(cupertino.CupertinoIcons.Search())
	xmarkIcon  = gluttertest.ByIcon(cupertino.CupertinoIcons.XmarkCircleFill())
)

func TestCupertinoSearchTextFieldDecoration(t *testing.T) {
	decoration := func(ts *gluttertest.Tester) *painting.BoxDecoration {
		return ts.Widget(gluttertest.Descendant(searchFinder(), gluttertest.ByType[*widgets.Container]())).(*widgets.Container).Decoration().(*painting.BoxDecoration)
	}
	t.Run("the default field has a border radius of 9 and the tertiary fill", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField())
		d := decoration(ts)
		if d.BorderRadius.(painting.BorderRadius).TopLeft != ui.NewRadiusCircular(9) {
			t.Errorf("radius %v", d.BorderRadius)
		}
		bc := ts.Element(searchFinder())
		if want := cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.TertiarySystemFill, bc); d.Color.OrElse(ui.Color{}) != want {
			t.Errorf("fill %v, want %v", d.Color, want)
		}
	})
	t.Run("a background color and a border radius are shorthands for the decoration", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{
			BackgroundColor: ui.NewColor(0xFF00FF00), BorderRadius: painting.NewBorderRadiusCircular(3)}))
		d := decoration(ts)
		if d.Color.OrElse(ui.Color{}) != ui.NewColor(0xFF00FF00) || d.BorderRadius.(painting.BorderRadius).TopLeft != ui.NewRadiusCircular(3) {
			t.Errorf("decoration %v", d)
		}
	})
	t.Run("the decoration overrides the background color and the border radius", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{
			Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFFFF0000)), BorderRadius: painting.NewBorderRadiusCircular(20)})}))
		d := decoration(ts)
		if d.Color.OrElse(ui.Color{}) != ui.NewColor(0xFFFF0000) || d.BorderRadius.(painting.BorderRadius).TopLeft != ui.NewRadiusCircular(20) {
			t.Errorf("decoration %v", d)
		}
	})
	t.Run("a decoration with a background color or a border radius is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		d := painting.NewBoxDecoration(painting.BoxDecorationOpts{})
		err := recoverError(func() {
			cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Decoration: d, BackgroundColor: ui.NewColor(0xFF000000)})
		})
		checkErrorTag(t, err, "CupertinoSearchTextFieldDecorationAndBackgroundColor")
		err = recoverError(func() {
			cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Decoration: d, BorderRadius: painting.NewBorderRadiusCircular(1)})
		})
		checkErrorTag(t, err, "CupertinoSearchTextFieldDecorationAndBorderRadius")
	})
}

func TestCupertinoSearchTextFieldText(t *testing.T) {
	t.Run("text entries are padded by default", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("initial")
		defer controller.Dispose()
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Controller: controller}))
		got := ts.TopLeft(editableFinder()).Sub(ts.TopLeft(searchFinder()))
		// 5.5 of padding, the prefix icon of 20 and its inset of 6.
		if got.Dx != 31.5 || got.Dy < 8 {
			t.Errorf("offset %v, want 31.5 across and at least 8 down", got)
		}
	})
	t.Run("the keyboard type can change", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{KeyboardType: foundation.Some(services.TextInputTypeNumber)}))
		if got := ts.Widget(editableFinder()).(*widgets.EditableText).KeyboardType(); got != services.TextInputTypeNumber {
			t.Errorf("keyboard type %v", got)
		}
	})
	t.Run("the text can be controlled by the controller", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController()
		defer controller.Dispose()
		var changed []string
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Controller: controller, OnChanged: func(s string) { changed = append(changed, s) }}))
		ts.EnterText(searchFinder(), "text")
		ts.Pump()
		if controller.Text() != "text" || len(changed) != 1 {
			t.Errorf("text %q, changes %v", controller.Text(), changed)
		}
		controller.SetText("more")
		ts.Pump()
		if got := renderEditableOf(ts).PlainText(); got != "more" {
			t.Errorf("the text shown is %q, want the one of the controller", got)
		}
	})
	t.Run("the placeholder is Search and has the secondary label color at 0.6", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField())
		style := ts.Widget(inSearch(gluttertest.ByText("Search"))).(*widgets.Text).Style()
		bc := ts.Element(searchFinder())
		want := cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.SecondaryLabel, bc)
		got := style.Color.OrElse(ui.Color{})
		if got != want.WithAlpha(int(math.Round(255*want.Opacity()))) || math.Abs(want.Opacity()-0.6) > 0.01 {
			t.Errorf("placeholder color %v, want %v", got, want)
		}
	})
	t.Run("the placeholder style does not change the style of the text", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("typed")
		defer controller.Dispose()
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Controller: controller,
			Placeholder: foundation.Some("hint"), PlaceholderStyle: painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(ui.NewColor(0xFFFF0000))})}))
		if got := ts.Widget(gluttertest.ByText("hint")).(*widgets.Text).Style().Color.OrElse(ui.Color{}); got != ui.NewColor(0xFFFF0000) {
			t.Errorf("placeholder color %v", got)
		}
		if got := ts.Widget(editableFinder()).(*widgets.EditableText).Style().Color.OrElse(ui.Color{}); got == ui.NewColor(0xFFFF0000) {
			t.Error("the placeholder style leaked into the text")
		}
	})
	t.Run("the semantics of the default placeholder is localized", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Placeholder: foundation.Some("Find")}))
		if !ts.Exists(gluttertest.ByText("Find")) || ts.Exists(gluttertest.ByText("Search")) {
			t.Error("the placeholder is not the given one")
		}
	})
}

func TestCupertinoSearchTextFieldIcons(t *testing.T) {
	t.Run("the prefix is in front of the text and the suffix after it", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("text")
		defer controller.Dispose()
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Controller: controller, SuffixMode: cupertino.OverlayVisibilityModeAlways}))
		text := ts.TopLeft(editableFinder()).Dx
		if ts.TopLeft(searchIcon).Dx >= text || ts.TopLeft(xmarkIcon).Dx <= text {
			t.Errorf("prefix at %v, text at %v, suffix at %v", ts.TopLeft(searchIcon).Dx, text, ts.TopLeft(xmarkIcon).Dx)
		}
	})
	t.Run("a custom prefix and suffix icon are used", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{
			PrefixIcon: widgets.NewIcon(cupertino.CupertinoIcons.Home()), SuffixIcon: widgets.NewIcon(cupertino.CupertinoIcons.Settings()),
			SuffixMode: cupertino.OverlayVisibilityModeAlways}))
		if !ts.Exists(gluttertest.ByIcon(cupertino.CupertinoIcons.Home())) || !ts.Exists(gluttertest.ByIcon(cupertino.CupertinoIcons.Settings())) || ts.Exists(searchIcon) {
			t.Error("the icons are not the custom ones")
		}
	})
	t.Run("the default prefix and suffix insets are aligned", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField())
		if !ts.Exists(searchIcon) || ts.Exists(xmarkIcon) {
			t.Fatal("the icons at first are wrong")
		}
		ts.EnterText(searchFinder(), "text input")
		ts.Pump()
		if !ts.Exists(searchIcon) || !ts.Exists(xmarkIcon) {
			t.Fatal("the icons with text are wrong")
		}
		width := ts.Size(gluttertest.ByType[*cupertino.CupertinoApp]()).Width
		if got := ts.TopLeft(searchIcon).Dx; got != 6 {
			t.Errorf("search icon at %v, want 6", got)
		}
		if got := ts.BottomRight(searchIcon).Dx; got != 26 {
			t.Errorf("search icon ends at %v, want 26", got)
		}
		if got := ts.TopLeft(xmarkIcon).Dx; got != width-25 {
			t.Errorf("xmark icon at %v, want %v", got, width-25)
		}
		if got := ts.BottomRight(xmarkIcon).Dx; got != width-5 {
			t.Errorf("xmark icon ends at %v, want %v", got, width-5)
		}
		if got := ts.Size(searchIcon); got != (ui.Size{Width: 20, Height: 20}) {
			t.Errorf("icon size %v", got)
		}
	})
	t.Run("the attachments go to the right places in RTL", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewDirectionality(ui.TextDirectionRTL, widgets.NewCenter(
			cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{SuffixMode: cupertino.OverlayVisibilityModeAlways})))}))
		width := ts.Size(gluttertest.ByType[*cupertino.CupertinoApp]()).Width
		if got := ts.TopLeft(searchIcon).Dx; got != width-26 {
			t.Errorf("search icon at %v, want %v", got, width-26)
		}
		if got := ts.BottomRight(xmarkIcon).Dx; got != 25 {
			t.Errorf("xmark icon ends at %v, want 25", got)
		}
	})
	t.Run("the insets can be changed", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{SuffixMode: cupertino.OverlayVisibilityModeAlways,
			PrefixInsets: painting.EdgeInsets{}, SuffixInsets: painting.EdgeInsets{}}))
		width := ts.Size(gluttertest.ByType[*cupertino.CupertinoApp]()).Width
		if got := ts.TopLeft(searchIcon).Dx; got != 0 {
			t.Errorf("search icon at %v, want 0", got)
		}
		if got := ts.BottomRight(xmarkIcon).Dx; got != width {
			t.Errorf("xmark icon ends at %v, want %v", got, width)
		}
	})
	t.Run("the suffix respects its visibility mode", func(t *testing.T) {
		for _, tc := range []struct {
			mode         cupertino.OverlayVisibilityMode
			empty, typed bool
		}{
			{cupertino.OverlayVisibilityModeNever, false, false},
			{cupertino.OverlayVisibilityModeEditing, false, true},
			{cupertino.OverlayVisibilityModeNotEditing, true, false},
			{cupertino.OverlayVisibilityModeAlways, true, true},
		} {
			ts := gluttertest.New(t)
			searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{SuffixMode: tc.mode}))
			if got := ts.Exists(xmarkIcon); got != tc.empty {
				t.Errorf("mode %v, empty: shown %v, want %v", tc.mode, got, tc.empty)
			}
			ts.EnterText(searchFinder(), "x")
			ts.Pump()
			if got := ts.Exists(xmarkIcon); got != tc.typed {
				t.Errorf("mode %v, typed: shown %v, want %v", tc.mode, got, tc.typed)
			}
		}
	})
	t.Run("the clear button removes the text and calls OnChanged when there was some", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("text")
		defer controller.Dispose()
		var changed []string
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Controller: controller, OnChanged: func(s string) { changed = append(changed, s) }}))
		ts.Tap(xmarkIcon)
		ts.Pump()
		if controller.Text() != "" || len(changed) != 1 || changed[0] != "" {
			t.Errorf("text %q, changes %v", controller.Text(), changed)
		}
	})
	t.Run("OnChanged is not called when the text was empty", func(t *testing.T) {
		ts := gluttertest.New(t)
		calls := 0
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{
			SuffixMode: cupertino.OverlayVisibilityModeAlways, OnChanged: func(string) { calls++ }}))
		ts.Tap(xmarkIcon)
		ts.Pump()
		if calls != 0 {
			t.Errorf("OnChanged was called %d times", calls)
		}
	})
	t.Run("a custom suffix tap replaces the clearing", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := widgets.NewTextEditingController("text")
		defer controller.Dispose()
		taps := 0
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Controller: controller, OnSuffixTap: func() { taps++ }}))
		ts.Tap(xmarkIcon)
		ts.Pump()
		if taps != 1 || controller.Text() != "text" {
			t.Errorf("taps %d, text %q", taps, controller.Text())
		}
	})
}

func TestCupertinoSearchTextFieldForwarding(t *testing.T) {
	t.Run("OnTap is forwarded", func(t *testing.T) {
		ts := gluttertest.New(t)
		taps := 0
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{OnTap: func() { taps++ }}))
		ts.Tap(searchFinder())
		if taps != 1 {
			t.Errorf("taps %d", taps)
		}
	})
	t.Run("Enabled is forwarded", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Enabled: foundation.Some(false)}))
		if ts.Widget(gluttertest.ByType[*cupertino.CupertinoTextField]()).(*cupertino.CupertinoTextField).Enabled() {
			t.Error("the inner field is enabled")
		}
	})
	t.Run("Autofocus gives the focus", func(t *testing.T) {
		ts := gluttertest.New(t)
		node := widgets.NewFocusNode(widgets.FocusNodeOpts{})
		defer node.Dispose()
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Autofocus: true, FocusNode: node}))
		ts.Pump()
		if !node.HasFocus() {
			t.Error("no focus")
		}
	})
	t.Run("the cursor options are forwarded", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Autofocus: true,
			CursorWidth: foundation.Some(3.0), CursorHeight: foundation.Some(30.0), CursorRadius: foundation.Some(ui.NewRadiusCircular(4)),
			CursorColor: ui.NewColor(0xFF123456)}))
		ts.Pump()
		r := renderEditableOf(ts)
		if r.CursorWidth() != 3 || r.CursorHeight() != 30 || r.CursorRadius().OrElse(ui.Radius{}) != ui.NewRadiusCircular(4) || r.CursorColor().OrElse(ui.Color{}) != ui.NewColor(0xFF123456) {
			t.Errorf("width %v, height %v, radius %v, color %v", r.CursorWidth(), r.CursorHeight(), r.CursorRadius(), r.CursorColor())
		}
	})
	t.Run("the default cursor is 2 wide with a radius of 2", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Autofocus: true}))
		ts.Pump()
		if r := renderEditableOf(ts); r.CursorWidth() != 2 || r.CursorRadius().OrElse(ui.Radius{}) != ui.NewRadiusCircular(2) {
			t.Errorf("width %v, radius %v", r.CursorWidth(), r.CursorRadius())
		}
	})
	t.Run("a search field of zero area does not crash", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewSizedBoxShrink(cupertino.NewCupertinoSearchTextField()))}))
		if got := ts.Size(searchFinder()); got != (ui.Size{}) {
			t.Errorf("size %v", got)
		}
	})
	t.Run("a controller and a focus node that change", func(t *testing.T) {
		ts := gluttertest.New(t)
		searchApp(ts, cupertino.NewCupertinoSearchTextField())
		ts.EnterText(searchFinder(), "kept")
		ts.Pump()
		controller := widgets.NewTextEditingController("given")
		defer controller.Dispose()
		node := widgets.NewFocusNode(widgets.FocusNodeOpts{})
		defer node.Dispose()
		searchApp(ts, cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Controller: controller, FocusNode: node}))
		ts.Pump()
		searchApp(ts, cupertino.NewCupertinoSearchTextField())
		ts.Pump()
		if !ts.Exists(searchFinder()) {
			t.Fatal("the field is gone")
		}
		if got := renderEditableOf(ts).PlainText(); got != "given" {
			t.Errorf("text %q after the controller was taken away, want the controller's text", got)
		}
	})
}

// shrinkingHeader is a header that is as high as the field and shrinks while the list scrolls, as Flutter's
// SliverResizingHeader does.
type shrinkingHeader struct {
	widgets.SliverPersistentHeaderDelegateBase
	extent float64
	child  widgets.Widget
}

func (d *shrinkingHeader) Build(widgets.BuildContext, float64, bool) widgets.Widget  { return d.child }
func (d *shrinkingHeader) MinExtent() float64                                        { return 0 }
func (d *shrinkingHeader) MaxExtent() float64                                        { return d.extent }
func (d *shrinkingHeader) ShouldRebuild(widgets.SliverPersistentHeaderDelegate) bool { return true }

func TestCupertinoSearchTextFieldFadesOnScroll(t *testing.T) {
	scrollApp := func(ts *gluttertest.Tester, extent float64, suffixMode cupertino.OverlayVisibilityMode) {
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCustomScrollView([]widgets.Widget{
			widgets.NewSliverPersistentHeader(&shrinkingHeader{extent: extent, child: cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{SuffixMode: suffixMode})},
				widgets.SliverPersistentHeaderOpts{Pinned: true}),
			widgets.NewSliverToBoxAdapter(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(2000.0)})),
		}))}))
	}
	opacityOf := func(ts *gluttertest.Tester, icon gluttertest.Finder) float64 {
		return ts.Widget(gluttertest.Ancestor(icon, gluttertest.ByType[*widgets.Opacity](), gluttertest.DescendantOpts{})).(*widgets.Opacity).Opacity()
	}
	t.Run("the icons and the placeholder fade while the header shrinks", func(t *testing.T) {
		ts := gluttertest.New(t)
		// The natural height of the field: 8 + 20 + 8 around the icons, and the text with its padding.
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoSearchTextField())}))
		height := ts.Size(searchFinder()).Height
		scrollApp(ts, height, cupertino.OverlayVisibilityModeAlways)
		placeholder := inSearch(gluttertest.ByText("Search"))
		if opacityOf(ts, searchIcon) != 1 || opacityOf(ts, xmarkIcon) != 1 {
			t.Fatal("the icons are not opaque at first")
		}
		alphaOf := func() int { return ts.Widget(placeholder).(*widgets.Text).Style().Color.OrElse(ui.Color{}).Alpha() }
		full := alphaOf()
		scroll := func(dy float64) {
			// The drag of the tester first moves by the touch slop, which scrolls nothing.
			ts.DragFrom(ts.Center(gluttertest.ByType[*widgets.CustomScrollView]()), ui.Offset{Dy: dy - gluttertest.DragSlopDefault})
			ts.PumpAndSettle()
		}
		scroll(-height / 10)
		if o := opacityOf(ts, searchIcon); !(o > 0 && o < 1) {
			t.Errorf("the prefix has the opacity %v after a tenth of the height", o)
		}
		if o := opacityOf(ts, xmarkIcon); !(o > 0 && o < 1) {
			t.Errorf("the suffix has the opacity %v after a tenth of the height", o)
		}
		if a := alphaOf(); a <= 0 || a >= full {
			t.Errorf("the placeholder has the alpha %v of %v", a, full)
		}
		scroll(-8 * height / 10)
		if opacityOf(ts, searchIcon) != 0 || opacityOf(ts, xmarkIcon) != 0 || alphaOf() != 0 {
			t.Errorf("not faded away: %v %v %v", opacityOf(ts, searchIcon), opacityOf(ts, xmarkIcon), alphaOf())
		}
	})
	t.Run("the top padding shrinks while the header does", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoSearchTextField())}))
		height := ts.Size(searchFinder()).Height
		scrollApp(ts, height, cupertino.OverlayVisibilityModeEditing)
		before := ts.TopLeft(searchIcon).Dy - ts.TopLeft(searchFinder()).Dy
		ts.DragFrom(ts.Center(gluttertest.ByType[*widgets.CustomScrollView]()), ui.Offset{Dy: -height/2 - gluttertest.DragSlopDefault})
		ts.PumpAndSettle()
		after := ts.TopLeft(searchIcon).Dy - ts.TopLeft(searchFinder()).Dy
		if after >= before {
			t.Errorf("the icon is %v from the top, it was %v", after, before)
		}
	})
}

func TestCupertinoSearchTextFieldTextScale(t *testing.T) {
	ts := gluttertest.New(t)
	scaleFactor, iconSize := 3.0, 10.0
	node := widgets.NewFocusNode(widgets.FocusNodeOpts{})
	defer node.Dispose()
	var setState widgets.StateSetter
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewStatefulBuilder(func(bc widgets.BuildContext, set widgets.StateSetter) widgets.Widget {
		setState = set
		data := widgets.MediaQueryOf(bc).CopyWith(widgets.MediaQueryDataCopyOpts{TextScaler: foundation.Some[painting.TextScaler](painting.NewTextScalerLinear(scaleFactor))})
		return widgets.NewMediaQuery(data, cupertino.NewCupertinoPageScaffold(widgets.NewCenter(cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{
			Placeholder: foundation.Some("Search"), FocusNode: node, PrefixIcon: widgets.NewIcon(cupertino.CupertinoIcons.Add()),
			SuffixIcon: widgets.NewIcon(cupertino.CupertinoIcons.Xmark()), SuffixMode: cupertino.OverlayVisibilityModeAlways, ItemSize: foundation.Some(iconSize)}))))
	})}))
	add, xmark := gluttertest.ByIcon(cupertino.CupertinoIcons.Add()), gluttertest.ByIcon(cupertino.CupertinoIcons.Xmark())
	square := func(v float64) ui.Size { return ui.Size{Width: v, Height: v} }
	for _, icon := range []gluttertest.Finder{add, xmark} {
		if got := ts.Size(icon); got != square(scaleFactor*iconSize) {
			t.Fatalf("an icon is %v, want %v", got, square(scaleFactor*iconSize))
		}
	}
	node.RequestFocus()
	ts.PumpAndSettle()
	// The prefix icon shrinks at the higher accessibility text scale modes.
	if got := ts.Size(add); got != (ui.Size{}) {
		t.Errorf("the prefix is %v, want zero", got)
	}
	if got := ts.Size(xmark); got != square(scaleFactor*iconSize) {
		t.Errorf("the suffix is %v", got)
	}
	setState(func() { scaleFactor = 2.9 })
	ts.PumpAndSettle()
	for _, icon := range []gluttertest.Finder{add, xmark} {
		if got := ts.Size(icon); math.Abs(got.Width-scaleFactor*iconSize) > 1e-9 {
			t.Errorf("below the threshold an icon is %v, want %v", got, square(scaleFactor*iconSize))
		}
	}
}
