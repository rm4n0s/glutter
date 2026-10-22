package main

import (
	"os"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
)

func buildListsPage(bc m.BuildContext) m.Widget {
	top := func(children ...m.Widget) m.Widget {
		return m.NewRow(children, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
	}
	return m.NewListView([]m.Widget{
		top(section(bc, "Card, filled and outlined", NewCardDemos()), section(bc, "ListTile", NewListTileDemos())),
		top(section(bc, "Badge and Tooltip", NewBadgeTooltipDemos()), section(bc, "Divider and VerticalDivider", NewDividerDemos())),
		top(section(bc, "CheckboxListTile, SwitchListTile and RadioListTile", NewSelectionTileDemos()), section(bc, "ExpansionTile", NewExpansionTileDemos())),
		top(section(bc, "ExpansionPanelList", NewExpansionPanelDemos()), section(bc, "MergeableMaterial", NewMergeableDemo())),
		top(section(bc, "ReorderableListView", NewReorderableDemos())),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(24.0)}),
	})
}

// devHover is what GLUTTER_DEV_HOVER asks to hover, in a debug build:
// tooltip-demo shows the tooltip of the Tooltip demo.
func devHover(getenv func(string) string) string {
	if !foundation.DebugMode {
		return ""
	}
	return getenv("GLUTTER_DEV_HOVER")
}

// CardDemos has the three kinds of card.
type CardDemos struct{ m.StatelessWidget }

// NewCardDemos creates the demo.
func NewCardDemos() *CardDemos { return &CardDemos{} }

// Build builds the cards.
func (w *CardDemos) Build(bc m.BuildContext) m.Widget {
	content := func(title, subtitle string) m.Widget {
		return m.NewSizedBox(m.NewListTile(m.ListTileOpts{
			Leading:  m.NewIcon(m.Icons.Album()),
			Title:    m.NewText(title),
			Subtitle: m.NewText(subtitle),
		}), m.SizedBoxOpts{Width: m.Some(360.0)})
	}
	return m.NewColumn([]m.Widget{
		m.NewCard(content("Elevated card", "The default card")),
		m.NewCardFilled(content("Filled card", "Highest surface container")),
		m.NewCardOutlined(content("Outlined card", "A border and no shadow")),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// ListTileDemos has list tiles of one, two and three lines, a selected one and a
// disabled one, divided by lines.
type ListTileDemos struct{ m.StatefulWidget }

// NewListTileDemos creates the demo.
func NewListTileDemos() *ListTileDemos { return &ListTileDemos{} }

// CreateState creates the state.
func (w *ListTileDemos) CreateState() m.State { return &listTileDemosState{} }

type listTileDemosState struct {
	m.StateOf[*ListTileDemos]
	taps int
}

func (s *listTileDemosState) Build(bc m.BuildContext) m.Widget {
	tiles := []m.Widget{
		m.NewListTile(m.ListTileOpts{Title: m.NewText("One line"), OnTap: func() { s.SetState(func() { s.taps++ }) }}),
		m.NewListTile(m.ListTileOpts{Leading: m.NewCircleAvatar(m.CircleAvatarOpts{Child: m.NewText("A")}), Title: m.NewText("Two lines"),
			Subtitle: m.NewText("With a leading avatar"), Trailing: m.NewIcon(m.Icons.MoreVert())}),
		m.NewListTile(m.ListTileOpts{Leading: m.NewIcon(m.Icons.Mail()), Title: m.NewText("Three lines"),
			Subtitle: m.NewText("The subtitle is long enough to need a second line of text, so the tile grows."), IsThreeLine: m.Some(true)}),
		m.NewListTile(m.ListTileOpts{Title: m.NewText("Selected"), Selected: true, Trailing: m.NewIcon(m.Icons.Check())}),
		m.NewListTile(m.ListTileOpts{Title: m.NewText("Disabled"), Subtitle: m.NewText("It does not react"), Disabled: true}),
	}
	return m.NewColumn([]m.Widget{
		framed(bc, m.NewSizedBox(m.NewColumn(m.ListTileDivideTiles(bc, tiles, m.None[m.Color]()), m.ColumnOpts{}), m.SizedBoxOpts{Width: m.Some(360.0)})),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 8, 0, 0), m.NewText("Taps on the first tile: "+itoa(s.taps))),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// ReorderableDemos has a ReorderableListView with a header, whose items the user
// drags by their handles.
type ReorderableDemos struct{ m.StatefulWidget }

// NewReorderableDemos creates the demo.
func NewReorderableDemos() *ReorderableDemos { return &ReorderableDemos{} }

// CreateState creates the state.
func (w *ReorderableDemos) CreateState() m.State {
	return &reorderableDemosState{order: []int{0, 1, 2, 3, 4, 5}, cards: []int{0, 1, 2, 3}}
}

type reorderableDemosState struct {
	m.StateOf[*ReorderableDemos]
	order, cards []int
}

func (s *reorderableDemosState) Build(bc m.BuildContext) m.Widget {
	list := m.NewReorderableListViewBuilder(func(_ m.BuildContext, i int) m.Widget {
		n := s.order[i]
		return m.NewListTile(m.ListTileOpts{Key: m.NewValueKey(n), Leading: m.NewCircleAvatar(m.CircleAvatarOpts{Child: m.NewText(itoa(n + 1))}),
			Title: m.NewText("Item " + itoa(n+1)), Subtitle: m.NewText("Drag the handle at the end")})
	}, len(s.order), m.ReorderableListViewOpts{
		Primary:       m.Some(false),
		OnReorderItem: func(oldIndex, newIndex int) { s.SetState(func() { moveItem(s.order, oldIndex, newIndex) }) },
		Header:        m.NewPadding(m.NewEdgeInsetsSymmetric(16, 8), m.NewText("Reorder the items")),
	})
	// The cards are made of children and scroll sideways: the handle is at the bottom.
	cards := make([]m.Widget, len(s.cards))
	for i, n := range s.cards {
		cards[i] = m.NewSizedBox(m.NewCard(m.NewCenter(m.NewText("Card "+itoa(n+1)))), m.SizedBoxOpts{Key: m.NewValueKey(n), Width: m.Some(96.0)})
	}
	row := m.NewReorderableListView(cards, m.ReorderableListViewOpts{
		ScrollDirection: m.AxisHorizontal, Primary: m.Some(false),
		OnReorderItem: func(oldIndex, newIndex int) { s.SetState(func() { moveItem(s.cards, oldIndex, newIndex) }) },
	})
	return m.NewColumn([]m.Widget{
		framed(bc, m.NewSizedBox(list, m.SizedBoxOpts{Width: m.Some(360.0), Height: m.Some(320.0)})),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(8.0)}),
		framed(bc, m.NewSizedBox(row, m.SizedBoxOpts{Width: m.Some(360.0), Height: m.Some(96.0)})),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// moveItem moves the item at oldIndex so that it is at newIndex, as the
// ReorderableListView asks.
func moveItem(items []int, oldIndex, newIndex int) {
	item := items[oldIndex]
	copy(items[oldIndex:], items[oldIndex+1:])
	copy(items[newIndex+1:], items[newIndex:len(items)-1])
	items[newIndex] = item
}

// SelectionTileDemos has a checkbox, a switch and two radio list tiles.
type SelectionTileDemos struct{ m.StatefulWidget }

// NewSelectionTileDemos creates the demo.
func NewSelectionTileDemos() *SelectionTileDemos { return &SelectionTileDemos{} }

// CreateState creates the state.
func (w *SelectionTileDemos) CreateState() m.State { return &selectionTileDemosState{} }

type selectionTileDemosState struct {
	m.StateOf[*SelectionTileDemos]
	checked, on bool
	size        m.Opt[string]
}

func (s *selectionTileDemosState) InitState() { s.checked, s.on, s.size = true, false, m.Some("small") }

func (s *selectionTileDemosState) Build(bc m.BuildContext) m.Widget {
	return framed(bc, m.NewSizedBox(m.NewColumn([]m.Widget{
		m.NewCheckboxListTile(m.Some(s.checked), func(v m.Opt[bool]) { s.SetState(func() { s.checked = v.Value() }) },
			m.CheckboxListTileOpts{Title: m.NewText("CheckboxListTile"), Subtitle: m.NewText("The control is at the end")}),
		m.NewSwitchListTile(s.on, func(v bool) { s.SetState(func() { s.on = v }) },
			m.SwitchListTileOpts{Title: m.NewText("SwitchListTile"), Secondary: m.NewIcon(m.Icons.Lightbulb())}),
		m.NewRadioGroup(func(v m.Opt[string]) { s.SetState(func() { s.size = v }) }, m.NewColumn([]m.Widget{
			m.NewRadioListTile("small", m.RadioListTileOpts[string]{Title: m.NewText("Small"), ControlAffinity: m.ListTileControlAffinityLeading}),
			m.NewRadioListTile("large", m.RadioListTileOpts[string]{Title: m.NewText("Large"), ControlAffinity: m.ListTileControlAffinityLeading}),
		}, m.ColumnOpts{}), m.RadioGroupOpts[string]{GroupValue: s.size}),
		m.NewPadding(m.NewEdgeInsetsAll(16), m.NewText("Selected: "+s.size.OrElse("none"))),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart}), m.SizedBoxOpts{Width: m.Some(360.0)}))
}

// ExpansionTileDemos has expansion tiles that start collapsed and expanded.
type ExpansionTileDemos struct{ m.StatelessWidget }

// NewExpansionTileDemos creates the demo.
func NewExpansionTileDemos() *ExpansionTileDemos { return &ExpansionTileDemos{} }

// Build builds the tiles.
func (w *ExpansionTileDemos) Build(bc m.BuildContext) m.Widget {
	return framed(bc, m.NewSizedBox(m.NewColumn([]m.Widget{
		m.NewExpansionTile(m.NewText("Collapsed"), m.ExpansionTileOpts{Subtitle: m.NewText("Tap to open"),
			Children: []m.Widget{m.NewListTile(m.ListTileOpts{Title: m.NewText("Inside one")}), m.NewListTile(m.ListTileOpts{Title: m.NewText("Inside two")})}}),
		m.NewExpansionTile(m.NewText("Expanded"), m.ExpansionTileOpts{InitiallyExpanded: true, Leading: m.NewIcon(m.Icons.Folder()),
			Children: []m.Widget{m.NewListTile(m.ListTileOpts{Title: m.NewText("Inside one")}), m.NewListTile(m.ListTileOpts{Title: m.NewText("Inside two")})}}),
		m.NewExpansionTile(m.NewText("Control at the start"), m.ExpansionTileOpts{ControlAffinity: m.ListTileControlAffinityLeading,
			Children: []m.Widget{m.NewListTile(m.ListTileOpts{Title: m.NewText("Inside")})}}),
	}, m.ColumnOpts{}), m.SizedBoxOpts{Width: m.Some(360.0)}))
}

// ExpansionPanelDemos has an independent ExpansionPanelList and a radio one.
type ExpansionPanelDemos struct{ m.StatefulWidget }

// NewExpansionPanelDemos creates the demo.
func NewExpansionPanelDemos() *ExpansionPanelDemos { return &ExpansionPanelDemos{} }

// CreateState creates the state.
func (w *ExpansionPanelDemos) CreateState() m.State { return &expansionPanelDemosState{} }

type expansionPanelDemosState struct {
	m.StateOf[*ExpansionPanelDemos]
	open [3]bool
}

func (s *expansionPanelDemosState) Build(bc m.BuildContext) m.Widget {
	header := func(label string) m.ExpansionPanelHeaderBuilder {
		return func(bc m.BuildContext, isExpanded bool) m.Widget {
			return m.NewListTile(m.ListTileOpts{Title: m.NewText(label)})
		}
	}
	body := func(text string) m.Widget {
		return m.NewAlign(m.NewPadding(m.NewEdgeInsetsAll(16), m.NewText(text)), m.AlignOpts{Alignment: m.AlignmentCenterLeft})
	}
	return m.NewSizedBox(m.NewColumn([]m.Widget{
		m.NewExpansionPanelList([]*m.ExpansionPanel{
			m.NewExpansionPanel(header("Panel one"), body("The body of the first panel."), m.ExpansionPanelOpts{IsExpanded: s.open[0]}),
			m.NewExpansionPanel(header("Panel two (tap the header)"), body("The body of the second panel."), m.ExpansionPanelOpts{IsExpanded: s.open[1], CanTapOnHeader: true}),
		}, m.ExpansionPanelListOpts{ExpansionCallback: func(i int, expanded bool) { s.SetState(func() { s.open[i] = expanded }) }}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
		m.NewExpansionPanelListRadio([]*m.ExpansionPanelRadio{
			m.NewExpansionPanelRadio("a", header("Radio A"), body("Only one radio panel is open."), m.ExpansionPanelOpts{CanTapOnHeader: true}),
			m.NewExpansionPanelRadio("b", header("Radio B"), body("Opening this closes the other."), m.ExpansionPanelOpts{CanTapOnHeader: true}),
		}, m.ExpansionPanelListOpts{InitialOpenPanelValue: "a"}),
	}, m.ColumnOpts{}), m.SizedBoxOpts{Width: m.Some(360.0)})
}

// MergeableDemo has slices of material with a gap that opens and closes.
type MergeableDemo struct{ m.StatefulWidget }

// NewMergeableDemo creates the demo.
func NewMergeableDemo() *MergeableDemo { return &MergeableDemo{} }

// CreateState creates the state.
func (w *MergeableDemo) CreateState() m.State { return &mergeableDemoState{} }

type mergeableDemoState struct {
	m.StateOf[*MergeableDemo]
	gap bool
}

func (s *mergeableDemoState) Build(bc m.BuildContext) m.Widget {
	slice := func(i int, label string) m.MergeableMaterialItem {
		return m.NewMaterialSlice(m.NewValueKey(i), m.NewListTile(m.ListTileOpts{Title: m.NewText(label)}))
	}
	items := []m.MergeableMaterialItem{slice(1, "First slice"), slice(2, "Second slice")}
	if s.gap {
		items = []m.MergeableMaterialItem{slice(1, "First slice"), m.NewMaterialGap(m.NewValueKey(100), 24), slice(2, "Second slice")}
	}
	return m.NewSizedBox(m.NewColumn([]m.Widget{
		m.NewMergeableMaterial(items, m.MergeableMaterialOpts{HasDividers: true, Elevation: m.Some(2.0)}),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 16, 0, 0), m.NewTextButton(func() { s.SetState(func() { s.gap = !s.gap }) }, m.NewText("Toggle the gap"))),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart}), m.SizedBoxOpts{Width: m.Some(360.0)})
}

// DividerDemos has dividers with indents, and a vertical divider.
type DividerDemos struct{ m.StatelessWidget }

// NewDividerDemos creates the demo.
func NewDividerDemos() *DividerDemos { return &DividerDemos{} }

// Build builds the dividers.
func (w *DividerDemos) Build(bc m.BuildContext) m.Widget {
	label := func(s string) m.Widget { return m.NewText(s, m.TextOpts{Style: m.ThemeOf(bc).TextTheme.BodySmall}) }
	return m.NewSizedBox(m.NewColumn([]m.Widget{
		label("Divider"), m.NewDivider(),
		label("Indented, 3 thick"), m.NewDivider(m.DividerOpts{Indent: m.Some(40.0), EndIndent: m.Some(40.0), Thickness: m.Some(3.0)}),
		label("Colored"), m.NewDivider(m.DividerOpts{Color: m.Some(m.ThemeOf(bc).ColorScheme.Primary())}),
		m.NewSizedBox(m.NewRow([]m.Widget{
			m.NewText("Left"), m.NewVerticalDivider(), m.NewText("Middle"), m.NewVerticalDivider(m.VerticalDividerOpts{Thickness: m.Some(3.0)}), m.NewText("Right"),
		}, m.RowOpts{MainAxisAlignment: m.MainAxisAlignmentSpaceEvenly}), m.SizedBoxOpts{Height: m.Some(40.0)}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart}), m.SizedBoxOpts{Width: m.Some(360.0)})
}

// BadgeTooltipDemos has badges on icons and a button with a tooltip.
type BadgeTooltipDemos struct{ m.StatefulWidget }

// NewBadgeTooltipDemos creates the demo.
func NewBadgeTooltipDemos() *BadgeTooltipDemos { return &BadgeTooltipDemos{} }

// CreateState creates the state.
func (w *BadgeTooltipDemos) CreateState() m.State { return &badgeTooltipDemosState{} }

type badgeTooltipDemosState struct {
	m.StateOf[*BadgeTooltipDemos]
	key *m.GlobalKey[*m.TooltipState]
}

// InitState shows the tooltip that GLUTTER_DEV_HOVER asks for.
func (s *badgeTooltipDemosState) InitState() {
	s.key = m.NewGlobalKey[*m.TooltipState]()
	if devHover(os.Getenv) != "tooltip-demo" {
		return
	}
	m.WidgetsBindingOf(s.Context()).Gestures().NewTimer(devOpenDelay, func() {
		if !s.Mounted() {
			return
		}
		if st, ok := s.key.CurrentState(); ok {
			st.EnsureTooltipVisible()
		}
	})
}

func (s *badgeTooltipDemosState) Build(bc m.BuildContext) m.Widget {
	return m.NewColumn([]m.Widget{
		m.NewRow([]m.Widget{
			m.NewBadge(m.NewIcon(m.Icons.Mail(), m.IconOpts{Size: m.Some(32.0)})),
			m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(32.0)}),
			m.NewBadge(m.NewIcon(m.Icons.Notifications(), m.IconOpts{Size: m.Some(32.0)}), m.BadgeOpts{Label: m.NewText("3")}),
			m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(32.0)}),
			m.NewBadgeCount(1234, m.NewIcon(m.Icons.ShoppingCart(), m.IconOpts{Size: m.Some(32.0)}), m.BadgeCountOpts{MaxCount: m.Some(99)}),
		}, m.RowOpts{}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(24.0)}),
		m.NewTooltip(m.NewFilledButtonTonal(func() {}, m.NewText("Hover me")),
			m.TooltipOpts{Key: s.key, Message: "Tooltips show after the Material delay", WaitDuration: m.Some(500 * time.Millisecond)}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(48.0)}),
		m.NewIconButton(func() {}, m.NewIcon(m.Icons.Info()), m.IconButtonOpts{Tooltip: "Info"}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}
