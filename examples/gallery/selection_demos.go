package main

import (
	"fmt"

	m "github.com/rm4n0s/glutter/material"
)

func buildSelectionPage(bc m.BuildContext) m.Widget {
	top := func(children ...m.Widget) m.Widget {
		return m.NewRow(children, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
	}
	return m.NewListView([]m.Widget{
		top(section(bc, "Checkbox", NewCheckboxDemo()), section(bc, "Radio and RadioGroup", NewRadioDemo())),
		top(section(bc, "Switch", NewSwitchDemo()), section(bc, "Slider and RangeSlider", NewSliderDemo())),
		top(section(bc, "Chip and InputChip", NewInputChipsDemo()), section(bc, "FilterChip and ChoiceChip", NewFilterChipsDemo())),
		top(section(bc, "ActionChip and CircleAvatar", NewActionChipsDemo())),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(24.0)}),
	})
}

// labeled puts a label after a control.
func labeled(control m.Widget, label string) m.Widget {
	return m.NewRow([]m.Widget{control, m.NewText(label)}, m.RowOpts{})
}

// CheckboxDemo has checkboxes in their states: one to toggle, one tristate, one
// disabled and one with an error.
type CheckboxDemo struct{ m.StatefulWidget }

// NewCheckboxDemo creates the demo.
func NewCheckboxDemo() *CheckboxDemo { return &CheckboxDemo{} }

// CreateState creates the state.
func (w *CheckboxDemo) CreateState() m.State { return &checkboxDemoState{} }

type checkboxDemoState struct {
	m.StateOf[*CheckboxDemo]
	plain bool
	tri   m.Opt[bool]
}

func (s *checkboxDemoState) InitState() { s.plain, s.tri = true, m.Some(true) }

func (s *checkboxDemoState) Build(bc m.BuildContext) m.Widget {
	return m.NewColumn([]m.Widget{
		labeled(m.NewCheckbox(m.Some(s.plain), func(v m.Opt[bool]) { s.SetState(func() { s.plain = v.Value() }) }), "Checked, tap to toggle"),
		labeled(m.NewCheckbox(s.tri, func(v m.Opt[bool]) { s.SetState(func() { s.tri = v }) }, m.CheckboxOpts{Tristate: true}), "Tristate: true, none and false"),
		labeled(m.NewCheckbox(m.Some(true), nil), "Disabled"),
		labeled(m.NewCheckbox(m.Some(false), nil), "Disabled, unchecked"),
		labeled(m.NewCheckbox(m.Some(false), func(m.Opt[bool]) {}, m.CheckboxOpts{IsError: true}), "Error"),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// RadioDemo has a group of radio buttons, and a disabled one.
type RadioDemo struct{ m.StatefulWidget }

// NewRadioDemo creates the demo.
func NewRadioDemo() *RadioDemo { return &RadioDemo{} }

// CreateState creates the state.
func (w *RadioDemo) CreateState() m.State { return &radioDemoState{} }

type radioDemoState struct {
	m.StateOf[*RadioDemo]
	value m.Opt[string]
}

func (s *radioDemoState) InitState() { s.value = m.Some("tea") }

func (s *radioDemoState) Build(bc m.BuildContext) m.Widget {
	return m.NewColumn([]m.Widget{
		m.NewRadioGroup(func(v m.Opt[string]) { s.SetState(func() { s.value = v }) },
			m.NewColumn([]m.Widget{
				labeled(m.NewRadio("tea", m.RadioOpts[string]{}), "Tea"),
				labeled(m.NewRadio("coffee", m.RadioOpts[string]{}), "Coffee"),
				labeled(m.NewRadio("water", m.RadioOpts[string]{}), "Water"),
			}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart}),
			m.RadioGroupOpts[string]{GroupValue: s.value}),
		labeled(m.NewRadio("tea", m.RadioOpts[string]{GroupValue: m.Some("tea")}), "Disabled, selected"),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(12, 4, 0, 0), m.NewText(fmt.Sprintf("Selected: %v", s.value.OrElse("none")))),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// SwitchDemo has switches: plain, with icons on the thumb, and disabled.
type SwitchDemo struct{ m.StatefulWidget }

// NewSwitchDemo creates the demo.
func NewSwitchDemo() *SwitchDemo { return &SwitchDemo{} }

// CreateState creates the state.
func (w *SwitchDemo) CreateState() m.State { return &switchDemoState{} }

type switchDemoState struct {
	m.StateOf[*SwitchDemo]
	plain, icons bool
}

func (s *switchDemoState) InitState() { s.plain, s.icons = true, true }

func (s *switchDemoState) Build(bc m.BuildContext) m.Widget {
	thumbIcon := m.WidgetStatePropertyResolveWith(func(states m.WidgetStates) *m.Icon {
		if states.Has(m.WidgetStateSelected) {
			return m.NewIcon(m.Icons.Check())
		}
		return m.NewIcon(m.Icons.Close())
	})
	return m.NewColumn([]m.Widget{
		labeled(m.NewSwitch(s.plain, func(v bool) { s.SetState(func() { s.plain = v }) }), "On, tap to toggle"),
		labeled(m.NewSwitch(s.icons, func(v bool) { s.SetState(func() { s.icons = v }) }, m.SwitchOpts{ThumbIcon: thumbIcon}), "With icons"),
		labeled(m.NewSwitch(false, func(bool) {}), "Off"),
		labeled(m.NewSwitch(true, nil), "Disabled, on"),
		labeled(m.NewSwitch(false, nil), "Disabled, off"),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// SliderDemo has a continuous slider, a discrete one with a label that stays
// visible, a disabled one and a range slider.
type SliderDemo struct{ m.StatefulWidget }

// NewSliderDemo creates the demo.
func NewSliderDemo() *SliderDemo { return &SliderDemo{} }

// CreateState creates the state.
func (w *SliderDemo) CreateState() m.State { return &sliderDemoState{} }

type sliderDemoState struct {
	m.StateOf[*SliderDemo]
	continuous, discrete float64
	rangeValues          m.RangeValues
}

func (s *sliderDemoState) InitState() {
	s.continuous, s.discrete = 0.4, 60
	s.rangeValues = m.RangeValues{Start: 20, End: 70}
}

func (s *sliderDemoState) Build(bc m.BuildContext) m.Widget {
	width := columnWidth - 24
	box := func(child m.Widget) m.Widget {
		return m.NewSizedBox(child, m.SizedBoxOpts{Width: m.Some(width), Height: m.Some(48.0)})
	}
	labels := m.RangeLabels{Start: fmt.Sprintf("%.0f", s.rangeValues.Start), End: fmt.Sprintf("%.0f", s.rangeValues.End)}
	alwaysLabel := func(child m.Widget) m.Widget {
		return m.NewSliderTheme(m.NewSliderThemeData(m.SliderThemeDataOpts{ShowValueIndicator: m.ShowValueIndicatorAlwaysVisible}), child)
	}
	return m.NewColumn([]m.Widget{
		box(m.NewSlider(s.continuous, func(v float64) { s.SetState(func() { s.continuous = v }) })),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 24, 0, 0), box(alwaysLabel(m.NewSlider(s.discrete, func(v float64) { s.SetState(func() { s.discrete = v }) },
			m.SliderOpts{Min: 0, Max: m.Some(100.0), Divisions: m.Some(10), Label: m.Some(fmt.Sprintf("%.0f", s.discrete))})))),
		box(m.NewSlider(0.7, nil)),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 24, 0, 0), box(alwaysLabel(m.NewRangeSlider(s.rangeValues, func(v m.RangeValues) { s.SetState(func() { s.rangeValues = v }) },
			m.RangeSliderOpts{Min: 0, Max: m.Some(100.0), Labels: m.Some(labels)})))),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// chips lays chips out in rows.
func chips(children ...m.Widget) m.Widget {
	return m.NewWrap(children, m.WrapOpts{Spacing: 8, RunSpacing: 8})
}

// InputChipsDemo has a chip, and input chips with an avatar that can be deleted.
type InputChipsDemo struct{ m.StatefulWidget }

// NewInputChipsDemo creates the demo.
func NewInputChipsDemo() *InputChipsDemo { return &InputChipsDemo{} }

// CreateState creates the state.
func (w *InputChipsDemo) CreateState() m.State { return &inputChipsDemoState{} }

type inputChipsDemoState struct {
	m.StateOf[*InputChipsDemo]
	people []string
}

func (s *inputChipsDemoState) InitState() { s.people = []string{"Ada", "Grace", "Linus", "Margaret"} }

func (s *inputChipsDemoState) Build(bc m.BuildContext) m.Widget {
	var people []m.Widget
	for _, name := range s.people {
		name := name
		people = append(people, m.NewInputChip(m.NewText(name), m.InputChipOpts{
			Avatar:    m.NewCircleAvatar(m.CircleAvatarOpts{Child: m.NewText(name[:1])}),
			OnDeleted: func() { s.SetState(func() { s.remove(name) }) },
		}))
	}
	if len(people) == 0 {
		people = append(people, m.NewActionChip(m.NewText("Restore"), m.ActionChipOpts{OnPressed: func() {
			s.SetState(func() { s.InitState() })
		}}))
	}
	return m.NewColumn([]m.Widget{
		chips(m.NewChip(m.NewText("Chip")), m.NewChip(m.NewText("With a delete icon"), m.ChipOpts{OnDeleted: func() {}})),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(12.0)}),
		chips(people...),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(12.0)}),
		chips(m.NewInputChip(m.NewText("Disabled input chip")), m.NewInputChip(m.NewText("Selected"), m.InputChipOpts{Selected: true, OnSelected: func(bool) {}})),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

func (s *inputChipsDemoState) remove(name string) {
	kept := s.people[:0:0]
	for _, p := range s.people {
		if p != name {
			kept = append(kept, p)
		}
	}
	s.people = kept
}

// FilterChipsDemo has filter chips that select any number of options and choice
// chips that select one.
type FilterChipsDemo struct{ m.StatefulWidget }

// NewFilterChipsDemo creates the demo.
func NewFilterChipsDemo() *FilterChipsDemo { return &FilterChipsDemo{} }

// CreateState creates the state.
func (w *FilterChipsDemo) CreateState() m.State { return &filterChipsDemoState{} }

type filterChipsDemoState struct {
	m.StateOf[*FilterChipsDemo]
	filters map[string]bool
	size    int
}

func (s *filterChipsDemoState) InitState() {
	s.filters = map[string]bool{"Vegan": true, "Gluten free": false, "Organic": true}
	s.size = 1
}

func (s *filterChipsDemoState) Build(bc m.BuildContext) m.Widget {
	var filters, sizes []m.Widget
	for _, name := range []string{"Vegan", "Gluten free", "Organic"} {
		name := name
		filters = append(filters, m.NewFilterChip(m.NewText(name), m.FilterChipOpts{
			Selected: s.filters[name], OnSelected: func(v bool) { s.SetState(func() { s.filters[name] = v }) }}))
	}
	for i, name := range []string{"Small", "Medium", "Large"} {
		i := i
		sizes = append(sizes, m.NewChoiceChip(m.NewText(name), s.size == i, m.ChoiceChipOpts{
			OnSelected: func(v bool) { s.SetState(func() { s.size = i }) }}))
	}
	return m.NewColumn([]m.Widget{
		chips(filters...),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(12.0)}),
		chips(sizes...),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(12.0)}),
		chips(m.NewFilterChipElevated(m.NewText("Elevated"), m.FilterChipOpts{OnSelected: func(bool) {}}),
			m.NewChoiceChip(m.NewText("Disabled"), false)),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// ActionChipsDemo has action chips, and avatars with initials, an icon and an
// image color.
type ActionChipsDemo struct{ m.StatefulWidget }

// NewActionChipsDemo creates the demo.
func NewActionChipsDemo() *ActionChipsDemo { return &ActionChipsDemo{} }

// CreateState creates the state.
func (w *ActionChipsDemo) CreateState() m.State { return &actionChipsDemoState{} }

type actionChipsDemoState struct {
	m.StateOf[*ActionChipsDemo]
	count int
}

func (s *actionChipsDemoState) Build(bc m.BuildContext) m.Widget {
	colors := m.ThemeOf(bc).ColorScheme
	press := func() { s.SetState(func() { s.count++ }) }
	return m.NewColumn([]m.Widget{
		chips(
			m.NewActionChip(m.NewText("Add"), m.ActionChipOpts{Avatar: m.NewIcon(m.Icons.Add()), OnPressed: press}),
			m.NewActionChipElevated(m.NewText("Share"), m.ActionChipOpts{Avatar: m.NewIcon(m.Icons.Share()), OnPressed: press}),
			m.NewActionChip(m.NewText("Disabled"), m.ActionChipOpts{Avatar: m.NewIcon(m.Icons.Block())}),
		),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 8, 0, 0), m.NewText(fmt.Sprintf("Pressed %d times", s.count))),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
		m.NewRow([]m.Widget{
			m.NewCircleAvatar(m.CircleAvatarOpts{Child: m.NewText("AB")}),
			m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(12.0)}),
			m.NewCircleAvatar(m.CircleAvatarOpts{Radius: m.Some(28.0), BackgroundColor: m.Some(colors.Tertiary()), ForegroundColor: m.Some(colors.OnTertiary()),
				Child: m.NewIcon(m.Icons.Person())}),
			m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(12.0)}),
			m.NewCircleAvatar(m.CircleAvatarOpts{Radius: m.Some(16.0), BackgroundColor: m.Some(colors.Error()), Child: m.NewText("!")}),
		}, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentCenter}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}
