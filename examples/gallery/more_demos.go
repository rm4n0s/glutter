package main

import (
	"bytes"
	"image"
	stdcolor "image/color"
	"image/png"
	"time"

	m "github.com/rm4n0s/glutter/material"
)

func buildMorePage(bc m.BuildContext) m.Widget {
	top := func(children ...m.Widget) m.Widget {
		return m.NewWrap(children)
	}
	return m.NewListView([]m.Widget{
		top(section(bc, "Adaptive constructors", NewAdaptiveDemos()), section(bc, "Action buttons and their icons", NewActionButtonDemos())),
		top(section(bc, "Ink, InkResponse and RawMaterialButton", NewInkDemos()), section(bc, "Chips and ExpandIcon", NewChipExtrasDemos())),
		top(section(bc, "Dialog and BottomSheet, in place", NewInPlaceDemos()), section(bc, "InputDecorator and date field", NewInputExtrasDemos())),
		top(section(bc, "SliverAppBar and FlexibleSpaceBar", NewSliverAppBarPlain()), section(bc, "SearchAnchor, TooltipVisibility and refresh forms", NewMiscDemos())),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(24.0)}),
	})
}

func gap(h float64) m.Widget { return m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(h)}) }

// AdaptiveDemos has the adaptive forms of the controls. They are the Material
// widgets on every platform.
type AdaptiveDemos struct{ m.StatelessWidget }

// NewAdaptiveDemos creates the demo.
func NewAdaptiveDemos() *AdaptiveDemos { return &AdaptiveDemos{} }

// Build builds the controls.
func (w *AdaptiveDemos) Build(bc m.BuildContext) m.Widget {
	return m.NewColumn([]m.Widget{
		m.NewRow([]m.Widget{
			m.NewCheckboxAdaptive(m.Some(true), func(m.Opt[bool]) {}),
			m.NewRadioAdaptive("a", m.RadioOpts[string]{GroupValue: m.Some("a"), OnChanged: func(m.Opt[string]) {}}),
			m.NewSwitchAdaptive(true, func(bool) {}),
			m.NewCircularProgressIndicatorAdaptive(m.CircularProgressIndicatorOpts{Value: m.Some(0.6)}),
		}, m.RowOpts{MainAxisAlignment: m.MainAxisAlignmentSpaceBetween}),
		m.NewSliderAdaptive(0.4, func(float64) {}),
		m.NewCheckboxListTileAdaptive(m.Some(true), func(m.Opt[bool]) {}, m.CheckboxListTileOpts{Title: m.NewText("CheckboxListTile.adaptive")}),
		m.NewSwitchListTileAdaptive(true, func(bool) {}, m.SwitchListTileOpts{Title: m.NewText("SwitchListTile.adaptive")}),
		m.NewRadioListTileAdaptive("a", m.RadioListTileOpts[string]{GroupValue: m.Some("a"), OnChanged: func(m.Opt[string]) {}, Title: m.NewText("RadioListTile.adaptive")}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch})
}

// ActionButtonDemos has the back, close and drawer buttons, and their icons.
type ActionButtonDemos struct{ m.StatelessWidget }

// NewActionButtonDemos creates the demo.
func NewActionButtonDemos() *ActionButtonDemos { return &ActionButtonDemos{} }

// Build builds the buttons.
func (w *ActionButtonDemos) Build(bc m.BuildContext) m.Widget {
	return m.NewColumn([]m.Widget{
		m.NewRow([]m.Widget{m.NewBackButton(), m.NewCloseButton(), m.NewDrawerButton(), m.NewEndDrawerButton()},
			m.RowOpts{MainAxisAlignment: m.MainAxisAlignmentSpaceBetween}),
		gap(16),
		m.NewRow([]m.Widget{m.NewBackButtonIcon(), m.NewCloseButtonIcon(), m.NewDrawerButtonIcon(), m.NewEndDrawerButtonIcon()},
			m.RowOpts{MainAxisAlignment: m.MainAxisAlignmentSpaceBetween}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch})
}

// gradientImage is a small image drawn in code, for the ink image.
func gradientImage() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, stdcolor.RGBA{R: uint8(x * 4), G: uint8(y * 8), B: 160, A: 255})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		panic(err)
	}
	return out.Bytes()
}

// InkDemos has the ink widgets: a decoration, an image, a response with a splash,
// a raw button and a table row that reacts to taps.
type InkDemos struct{ m.StatefulWidget }

// NewInkDemos creates the demo.
func NewInkDemos() *InkDemos { return &InkDemos{} }

// CreateState creates the state.
func (w *InkDemos) CreateState() m.State { return &inkDemosState{} }

type inkDemosState struct {
	m.StateOf[*InkDemos]
	taps int
}

func (s *inkDemosState) Build(bc m.BuildContext) m.Widget {
	scheme := m.ThemeOf(bc).ColorScheme
	tap := func() { s.SetState(func() { s.taps++ }) }
	return m.NewColumn([]m.Widget{
		m.NewText("Taps: " + itoa(s.taps)),
		gap(8),
		m.NewInk(m.NewInkWell(m.NewCenter(m.NewText("Ink with a decoration")), m.InkWellOpts{OnTap: tap}),
			m.InkOpts{Decoration: m.NewBoxDecoration(m.BoxDecorationOpts{Color: m.Some(scheme.SecondaryContainer()), BorderRadius: m.NewBorderRadiusCircular(12)}),
				Height: m.Some(56.0)}),
		gap(8),
		m.NewInkImage(m.NewMemoryImage(gradientImage()), m.NewInkResponse(m.NewCenter(m.NewText("InkImage and InkResponse", m.TextOpts{Style: m.NewTextStyle(m.TextStyleOpts{Color: m.Some(m.Colors.White)})})),
			m.InkResponseOpts{OnTap: tap}), m.InkImageOpts{Fit: m.BoxFitCover, Height: m.Some(56.0)}),
		gap(8),
		m.NewRawMaterialButton(tap, m.NewText("RawMaterialButton"), m.RawMaterialButtonOpts{FillColor: m.Some(scheme.TertiaryContainer())}),
		gap(8),
		m.NewTable([]m.TableRow{m.NewTableRow([]m.Widget{
			m.NewTableRowInkWell(m.NewPadding(m.NewEdgeInsetsAll(12), m.NewText("TableRowInkWell")), m.TableRowInkWellOpts{OnTap: tap}),
			m.NewTableRowInkWell(m.NewPadding(m.NewEdgeInsetsAll(12), m.NewText("in a table row")), m.TableRowInkWellOpts{OnTap: tap}),
		})}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch})
}

// ChipExtrasDemos has a raw chip, an elevated choice chip and the expand icon.
type ChipExtrasDemos struct{ m.StatefulWidget }

// NewChipExtrasDemos creates the demo.
func NewChipExtrasDemos() *ChipExtrasDemos { return &ChipExtrasDemos{} }

// CreateState creates the state.
func (w *ChipExtrasDemos) CreateState() m.State { return &chipExtrasDemosState{} }

type chipExtrasDemosState struct {
	m.StateOf[*ChipExtrasDemos]
	chosen, expanded bool
}

func (s *chipExtrasDemosState) Build(bc m.BuildContext) m.Widget {
	return m.NewColumn([]m.Widget{
		m.NewWrap([]m.Widget{
			m.NewRawChip(m.NewText("RawChip"), m.RawChipOpts{Avatar: m.NewIcon(m.Icons.Star()), OnDeleted: func() {}}),
			m.NewChoiceChipElevated(m.NewText("Elevated choice"), s.chosen, m.ChoiceChipOpts{OnSelected: func(v bool) { s.SetState(func() { s.chosen = v }) }}),
		}, m.WrapOpts{Spacing: 8, RunSpacing: 8}),
		gap(16),
		m.NewRow([]m.Widget{
			m.NewExpandIcon(func(expanded bool) { s.SetState(func() { s.expanded = !expanded }) }, m.ExpandIconOpts{IsExpanded: s.expanded}),
			m.NewText("ExpandIcon"),
		}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// InPlaceDemos has a Dialog, an AlertDialog and a BottomSheet in the page rather
// than in a route.
type InPlaceDemos struct{ m.StatelessWidget }

// NewInPlaceDemos creates the demo.
func NewInPlaceDemos() *InPlaceDemos { return &InPlaceDemos{} }

// Build builds the surfaces.
func (w *InPlaceDemos) Build(bc m.BuildContext) m.Widget {
	return m.NewColumn([]m.Widget{
		m.NewSizedBox(m.NewDialog(m.NewPadding(m.NewEdgeInsetsAll(24), m.NewText("Dialog in place"))), m.SizedBoxOpts{Width: m.Some(300.0), Height: m.Some(100.0)}),
		m.NewSizedBox(m.NewAlertDialogAdaptive(m.AlertDialogOpts{Title: m.NewText("AlertDialog"), Content: m.NewText("In place."),
			Actions: []m.Widget{m.NewTextButton(func() {}, m.NewText("OK"))}}), m.SizedBoxOpts{Width: m.Some(300.0), Height: m.Some(190.0)}),
		m.NewSizedBox(m.NewBottomSheet(func() {}, func(m.BuildContext) m.Widget {
			return m.NewPadding(m.NewEdgeInsetsAll(24), m.NewText("BottomSheet in place"))
		}, m.BottomSheetOpts{NoEnableDrag: true, ShowDragHandle: m.Some(true)}), m.SizedBoxOpts{Width: m.Some(300.0), Height: m.Some(100.0)}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// InputExtrasDemos has an InputDecorator around a plain widget and the date
// field of the date picker's input mode.
type InputExtrasDemos struct{ m.StatelessWidget }

// NewInputExtrasDemos creates the demo.
func NewInputExtrasDemos() *InputExtrasDemos { return &InputExtrasDemos{} }

// Build builds the fields.
func (w *InputExtrasDemos) Build(bc m.BuildContext) m.Widget {
	now := time.Now()
	return m.NewColumn([]m.Widget{
		m.NewSizedBox(m.NewInputDecorator(m.InputDecoration{LabelText: text("InputDecorator"), Border: m.NewOutlineInputBorder(), HelperText: text("Around any widget")},
			m.InputDecoratorOpts{IsEmpty: false, Child: m.NewText("A plain Text")}), m.SizedBoxOpts{Width: m.Some(300.0)}),
		gap(16),
		m.NewSizedBox(m.NewInputDatePickerFormField(now.AddDate(-1, 0, 0), now.AddDate(1, 0, 0), m.InputDatePickerFormFieldOpts{InitialDate: now}), m.SizedBoxOpts{Width: m.Some(300.0)}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// SliverAppBarPlain has the plain SliverAppBar with a FlexibleSpaceBar, in a
// scroll view.
type SliverAppBarPlain struct{ m.StatelessWidget }

// NewSliverAppBarPlain creates the demo.
func NewSliverAppBarPlain() *SliverAppBarPlain { return &SliverAppBarPlain{} }

// Build builds the scroll view.
func (w *SliverAppBarPlain) Build(bc m.BuildContext) m.Widget {
	items := make([]m.Widget, 8)
	for i := range items {
		items[i] = m.NewListTile(m.ListTileOpts{Title: m.NewText("Item " + itoa(i+1))})
	}
	return m.NewSizedBox(framed(bc, m.NewCustomScrollView([]m.Widget{
		m.NewSliverAppBar(m.SliverAppBarOpts{Pinned: m.Some(true), ExpandedHeight: m.Some(120.0), NoAutomaticallyImplyLeading: true,
			FlexibleSpace: m.NewFlexibleSpaceBar(m.FlexibleSpaceBarOpts{Title: m.NewText("FlexibleSpaceBar")})}),
		m.NewSliverList(m.NewSliverChildListDelegate(items)),
	})), m.SizedBoxOpts{Height: m.Some(300.0)})
}

// MiscDemos has a search anchor in its plain form, a tooltip that is switched off
// and the adaptive forms of the refresh indicator.
type MiscDemos struct{ m.StatelessWidget }

// NewMiscDemos creates the demo.
func NewMiscDemos() *MiscDemos { return &MiscDemos{} }

// Build builds the demos.
func (w *MiscDemos) Build(bc m.BuildContext) m.Widget {
	rows := func(prefix string) m.Widget {
		return m.NewListView([]m.Widget{m.NewListTile(m.ListTileOpts{Title: m.NewText(prefix + " 1")}), m.NewListTile(m.ListTileOpts{Title: m.NewText(prefix + " 2")}),
			m.NewListTile(m.ListTileOpts{Title: m.NewText(prefix + " 3")})}, m.ListViewOpts{Physics: m.NewAlwaysScrollableScrollPhysics(nil)})
	}
	refresh := func() <-chan struct{} {
		done := make(chan struct{})
		time.AfterFunc(time.Second, func() { close(done) })
		return done
	}
	return m.NewColumn([]m.Widget{
		m.NewSearchAnchor(func(bc m.BuildContext, c *m.SearchController) m.Widget {
			return m.NewFilledButtonTonal(func() { c.OpenView() }, m.NewText("SearchAnchor"))
		}, func(_ m.BuildContext, _ *m.SearchController, deliver func([]m.Widget)) {
			deliver([]m.Widget{m.NewListTile(m.ListTileOpts{Title: m.NewText("A suggestion")})})
		}),
		gap(12),
		m.NewTooltipVisibility(false, m.NewTooltip(m.NewFilledButtonTonal(func() {}, m.NewText("No tooltip")), m.TooltipOpts{Message: "Never shown"})),
		gap(12),
		m.NewSizedBox(framed(bc, m.NewRefreshIndicatorAdaptive(refresh, rows("Adaptive"))), m.SizedBoxOpts{Height: m.Some(110.0)}),
		gap(8),
		m.NewSizedBox(framed(bc, m.NewRefreshIndicatorNoSpinner(refresh, rows("No spinner"))), m.SizedBoxOpts{Height: m.Some(110.0)}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch})
}
