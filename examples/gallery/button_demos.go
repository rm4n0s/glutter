package main

import (
	m "github.com/rm4n0s/glutter/material"
)

func buildButtonsPage(bc m.BuildContext) m.Widget {
	top := func(children ...m.Widget) m.Widget {
		return m.NewRow(children, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
	}
	return m.NewListView([]m.Widget{
		top(section(bc, "ElevatedButton", NewStateButtons(elevatedKind)), section(bc, "FilledButton", NewStateButtons(filledKind))),
		top(section(bc, "FilledButton.tonal", NewStateButtons(tonalKind)), section(bc, "OutlinedButton", NewStateButtons(outlinedKind))),
		top(section(bc, "TextButton", NewStateButtons(textKind)), section(bc, "IconButton", NewIconButtonDemo())),
		top(section(bc, "FloatingActionButton", NewFabSizesDemo()), section(bc, "SegmentedButton", NewSegmentedDemo())),
		top(section(bc, "FloatingActionButtonLocation", NewFabLocationDemo())),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(24.0)}),
	})
}

// buttonKind is the kind of the buttons of a StateButtons.
type buttonKind int

const (
	elevatedKind buttonKind = iota
	filledKind
	tonalKind
	outlinedKind
	textKind
)

func (k buttonKind) button(label string, onPressed func(), o m.ButtonStyleButtonOpts) m.Widget {
	child := m.NewText(label)
	switch k {
	case elevatedKind:
		return m.NewElevatedButton(onPressed, child, o)
	case filledKind:
		return m.NewFilledButton(onPressed, child, o)
	case tonalKind:
		return m.NewFilledButtonTonal(onPressed, child, o)
	case outlinedKind:
		return m.NewOutlinedButton(onPressed, child, o)
	}
	return m.NewTextButton(onPressed, child, o)
}

func (k buttonKind) iconButton(label string, onPressed func(), icon m.Widget, alignment m.IconAlignment) m.Widget {
	io := m.TextButtonIconOpts{Icon: icon, IconAlignment: alignment}
	child := m.NewText(label)
	switch k {
	case elevatedKind:
		return m.NewElevatedButtonIcon(onPressed, child, io)
	case filledKind:
		return m.NewFilledButtonIcon(onPressed, child, io)
	case tonalKind:
		return m.NewFilledButtonTonalIcon(onPressed, child, io)
	case outlinedKind:
		return m.NewOutlinedButtonIcon(onPressed, child, io)
	}
	return m.NewTextButtonIcon(onPressed, child, io)
}

// StateButtons shows a kind of button enabled, disabled, hovered, focused and
// pressed, and with an icon at the start and at the end. The hovered, focused and
// pressed buttons are given those states by their states controllers, so that
// the page shows them at rest; pointing at the buttons and pressing them works
// as for any button.
type StateButtons struct {
	m.StatefulWidget
	kind buttonKind
}

// NewStateButtons creates the buttons of the kind.
func NewStateButtons(kind buttonKind) *StateButtons { return &StateButtons{kind: kind} }

// CreateState creates the state.
func (w *StateButtons) CreateState() m.State { return &stateButtonsState{} }

type stateButtonsState struct {
	m.StateOf[*StateButtons]
	hovered, focused, pressed *m.WidgetStatesController
	presses                   int
}

func (s *stateButtonsState) InitState() {
	s.hovered = m.NewWidgetStatesController(m.WidgetStateHovered)
	s.focused = m.NewWidgetStatesController(m.WidgetStateFocused)
	s.pressed = m.NewWidgetStatesController(m.WidgetStatePressed)
}

func (s *stateButtonsState) Dispose() {
	s.hovered.Dispose()
	s.focused.Dispose()
	s.pressed.Dispose()
}

func (s *stateButtonsState) Build(bc m.BuildContext) m.Widget {
	k := s.Widget().kind
	press := func() { s.SetState(func() { s.presses++ }) }
	count := "Pressed " + itoa(s.presses) + " times"
	return m.NewColumn([]m.Widget{
		m.NewWrap([]m.Widget{
			k.button("Enabled", press, m.ButtonStyleButtonOpts{}),
			k.button("Disabled", nil, m.ButtonStyleButtonOpts{}),
			k.button("Hovered", press, m.ButtonStyleButtonOpts{StatesController: s.hovered}),
			k.button("Focused", press, m.ButtonStyleButtonOpts{StatesController: s.focused}),
			k.button("Pressed", press, m.ButtonStyleButtonOpts{StatesController: s.pressed}),
		}, m.WrapOpts{Spacing: 12, RunSpacing: 12}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(12.0)}),
		m.NewWrap([]m.Widget{
			k.iconButton("Icon", press, m.NewIcon(m.Icons.Add()), m.IconAlignmentStart),
			k.iconButton("Icon at the end", press, m.NewIcon(m.Icons.Send()), m.IconAlignmentEnd),
			k.iconButton("Disabled", nil, m.NewIcon(m.Icons.Add()), m.IconAlignmentStart),
		}, m.WrapOpts{Spacing: 12, RunSpacing: 12}),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 8, 0, 0), m.NewText(count)),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// IconButtonDemo shows the four variants of IconButton: enabled, disabled, and
// as toggle buttons.
type IconButtonDemo struct{ m.StatefulWidget }

// NewIconButtonDemo creates the demo.
func NewIconButtonDemo() *IconButtonDemo { return &IconButtonDemo{} }

// CreateState creates the state.
func (w *IconButtonDemo) CreateState() m.State { return &iconButtonDemoState{} }

type iconButtonDemoState struct {
	m.StateOf[*IconButtonDemo]
	selected [4]bool
}

func (s *iconButtonDemoState) Build(bc m.BuildContext) m.Widget {
	variants := []struct {
		name string
		make func(onPressed func(), icon m.Widget, o m.IconButtonOpts) m.Widget
	}{
		{"standard", func(p func(), i m.Widget, o m.IconButtonOpts) m.Widget { return m.NewIconButton(p, i, o) }},
		{"filled", func(p func(), i m.Widget, o m.IconButtonOpts) m.Widget { return m.NewIconButtonFilled(p, i, o) }},
		{"filledTonal", func(p func(), i m.Widget, o m.IconButtonOpts) m.Widget { return m.NewIconButtonFilledTonal(p, i, o) }},
		{"outlined", func(p func(), i m.Widget, o m.IconButtonOpts) m.Widget { return m.NewIconButtonOutlined(p, i, o) }},
	}
	rows := make([]m.Widget, len(variants))
	for i, v := range variants {
		i := i
		toggle := func() { s.SetState(func() { s.selected[i] = !s.selected[i] }) }
		rows[i] = m.NewRow([]m.Widget{
			m.NewSizedBox(m.NewText(v.name), m.SizedBoxOpts{Width: m.Some(96.0)}),
			v.make(func() {}, m.NewIcon(m.Icons.Favorite()), m.IconButtonOpts{Tooltip: v.name}),
			v.make(nil, m.NewIcon(m.Icons.Favorite()), m.IconButtonOpts{}),
			v.make(toggle, m.NewIcon(m.Icons.FavoriteBorder()), m.IconButtonOpts{
				IsSelected: m.Some(s.selected[i]), SelectedIcon: m.NewIcon(m.Icons.Favorite()), Tooltip: "Toggle " + v.name}),
			v.make(nil, m.NewIcon(m.Icons.FavoriteBorder()), m.IconButtonOpts{IsSelected: m.Some(true), SelectedIcon: m.NewIcon(m.Icons.Favorite())}),
		}, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentCenter})
	}
	return m.NewColumn(append(rows, m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 4, 0, 0),
		m.NewText("enabled, disabled, a toggle, and a disabled selected toggle"))), m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// FabSizesDemo shows the regular, small, large and extended floating action
// buttons, and a disabled one.
type FabSizesDemo struct{ m.StatelessWidget }

// NewFabSizesDemo creates the demo.
func NewFabSizesDemo() *FabSizesDemo { return &FabSizesDemo{} }

func (w *FabSizesDemo) Build(bc m.BuildContext) m.Widget {
	o := m.FloatingActionButtonOpts{NoHero: true}
	add := func() m.Widget { return m.NewIcon(m.Icons.Add()) }
	return m.NewWrap([]m.Widget{
		m.NewFloatingActionButtonSmall(func() {}, add(), o),
		m.NewFloatingActionButton(func() {}, add(), o),
		m.NewFloatingActionButtonLarge(func() {}, add(), o),
		m.NewFloatingActionButton(nil, add(), o),
		m.NewFloatingActionButtonExtended(func() {}, m.NewText("Create"), m.FloatingActionButtonExtendedOpts{FloatingActionButtonOpts: o, Icon: add()}),
		m.NewFloatingActionButtonExtended(func() {}, m.NewText("Label only"), m.FloatingActionButtonExtendedOpts{FloatingActionButtonOpts: o}),
	}, m.WrapOpts{Spacing: 16, RunSpacing: 16, CrossAxisAlignment: m.WrapCrossAlignmentCenter})
}

// SegmentedDemo shows a single selection, a multiple selection, one with
// icons and a disabled one.
type SegmentedDemo struct{ m.StatefulWidget }

// NewSegmentedDemo creates the demo.
func NewSegmentedDemo() *SegmentedDemo { return &SegmentedDemo{} }

// CreateState creates the state.
func (w *SegmentedDemo) CreateState() m.State { return &segmentedDemoState{} }

type segmentedDemoState struct {
	m.StateOf[*SegmentedDemo]
	single []string
	multi  []int
	icons  []string
}

func (s *segmentedDemoState) InitState() {
	s.single = []string{"Week"}
	s.multi = []int{1, 3}
	s.icons = []string{"list"}
}

func (s *segmentedDemoState) Build(bc m.BuildContext) m.Widget {
	segments := func() []m.ButtonSegment[string] {
		return []m.ButtonSegment[string]{
			{Value: "Day", Label: m.NewText("Day")},
			{Value: "Week", Label: m.NewText("Week")},
			{Value: "Month", Label: m.NewText("Month")},
		}
	}
	multiSegments := []m.ButtonSegment[int]{
		{Value: 1, Label: m.NewText("S")}, {Value: 2, Label: m.NewText("M")}, {Value: 3, Label: m.NewText("L")}, {Value: 4, Label: m.NewText("XL"), Disabled: true},
	}
	iconSegments := []m.ButtonSegment[string]{
		{Value: "list", Icon: m.NewIcon(m.Icons.Menu()), Tooltip: "List"},
		{Value: "grid", Icon: m.NewIcon(m.Icons.Search()), Tooltip: "Grid"},
		{Value: "map", Icon: m.NewIcon(m.Icons.Explore()), Tooltip: "Map"},
	}
	gap := m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)})
	return m.NewColumn([]m.Widget{
		m.NewSegmentedButton(segments(), s.single, func(v []string) { s.SetState(func() { s.single = v }) }),
		gap,
		m.NewSegmentedButton(multiSegments, s.multi, func(v []int) { s.SetState(func() { s.multi = v }) }, m.SegmentedButtonOpts{MultiSelectionEnabled: true}),
		gap,
		m.NewSegmentedButton(iconSegments, s.icons, func(v []string) { s.SetState(func() { s.icons = v }) }),
		gap,
		m.NewSegmentedButton(segments(), []string{"Day"}, nil),
		gap,
		m.NewSegmentedButton([]m.ButtonSegment[string]{
			{Value: "a", Label: m.NewText("Text"), Icon: m.NewIcon(m.Icons.Add())}, {Value: "b", Label: m.NewText("Icon"), Icon: m.NewIcon(m.Icons.Send())}},
			s.single[:0], func(v []string) {}, m.SegmentedButtonOpts{EmptySelectionAllowed: true, NoShowSelectedIcon: true}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// FabLocationDemo is a small Scaffold whose floating action button moves to the
// location that a SegmentedButton chooses, with the scaling animation.
type FabLocationDemo struct{ m.StatefulWidget }

// NewFabLocationDemo creates the demo.
func NewFabLocationDemo() *FabLocationDemo { return &FabLocationDemo{} }

// CreateState creates the state.
func (w *FabLocationDemo) CreateState() m.State { return &fabLocationDemoState{} }

type fabLocationDemoState struct {
	m.StateOf[*FabLocationDemo]
	location string
}

var fabLocations = map[string]m.FloatingActionButtonLocation{
	"endFloat":       m.FloatingActionButtonLocationEndFloat,
	"centerFloat":    m.FloatingActionButtonLocationCenterFloat,
	"startTop":       m.FloatingActionButtonLocationStartTop,
	"centerDocked":   m.FloatingActionButtonLocationCenterDocked,
	"endContained":   m.FloatingActionButtonLocationEndContained,
	"miniStartFloat": m.FloatingActionButtonLocationMiniStartFloat,
}

func (s *fabLocationDemoState) InitState() { s.location = "endFloat" }

func (s *fabLocationDemoState) Build(bc m.BuildContext) m.Widget {
	segments := make([]m.ButtonSegment[string], 0, 3)
	for _, n := range []string{"endFloat", "centerDocked", "startTop"} {
		segments = append(segments, m.ButtonSegment[string]{Value: n, Label: m.NewText(n)})
	}
	chooser := m.NewSegmentedButton(segments, []string{s.location}, func(v []string) { s.SetState(func() { s.location = v[0] }) })
	// The mini location is reached with a text button.
	mini := m.NewTextButton(func() { s.SetState(func() { s.location = "miniStartFloat" }) }, m.NewText("miniStartFloat"))
	var fab m.Widget
	if s.location == "miniStartFloat" {
		fab = m.NewFloatingActionButtonSmall(func() {}, m.NewIcon(m.Icons.Add()), m.FloatingActionButtonOpts{NoHero: true})
	} else {
		fab = m.NewFloatingActionButton(func() {}, m.NewIcon(m.Icons.Add()), m.FloatingActionButtonOpts{NoHero: true})
	}
	return m.NewColumn([]m.Widget{
		m.NewWrap([]m.Widget{chooser, mini}, m.WrapOpts{Spacing: 8, RunSpacing: 8, CrossAxisAlignment: m.WrapCrossAlignmentCenter}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(12.0)}),
		m.NewSizedBox(framed(bc, m.NewScaffold(m.ScaffoldOpts{
			Body: m.NewCenter(m.NewText("Location: " + s.location)),
			BottomNavigationBar: m.NewBottomAppBar(m.BottomAppBarOpts{
				Height: m.Some(64.0),
				Child:  m.NewRow([]m.Widget{NewIconAction(m.Icons.Menu(), "Menu", func() {})}),
			}),
			FloatingActionButton:         fab,
			FloatingActionButtonLocation: fabLocations[s.location],
		})), m.SizedBoxOpts{Width: m.Some(columnWidth), Height: m.Some(300.0)}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}
