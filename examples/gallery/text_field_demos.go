package main

import (
	"os"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

func buildTextFieldsPage(bc m.BuildContext) m.Widget {
	top := func(children ...m.Widget) m.Widget {
		return m.NewRow(children, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
	}
	return m.NewListView([]m.Widget{
		top(section(bc, "Filled", NewFilledFieldsDemo()), section(bc, "Outlined", NewOutlinedFieldsDemo())),
		top(section(bc, "Form validation", NewFormDemo()), section(bc, "Prefix and suffix", NewAffixFieldsDemo())),
		top(section(bc, "Errors and counters", NewCounterFieldsDemo()), section(bc, "Multi-line and password", NewMultilineFieldsDemo())),
		top(section(bc, "DropdownMenu and DropdownButtonFormField", NewDropdownFieldsDemo()), section(bc, "SelectionArea", NewSelectionAreaDemo())),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(200.0)}),
	})
}

// devTextFields is the list of the things that GLUTTER_DEV_TEXT_FIELDS asks for,
// in a debug build: "focus" focuses the outlined field and "validate" validates
// the form, when the page is shown.
func devTextFields(getenv func(string) string) map[string]bool {
	out := map[string]bool{}
	if !foundation.DebugMode {
		return out
	}
	for _, item := range strings.Split(getenv("GLUTTER_DEV_TEXT_FIELDS"), ",") {
		if item = strings.TrimSpace(item); item != "" {
			out[item] = true
		}
	}
	return out
}

// field is a text field of the demos, 100% of the column wide.
func field(opts m.TextFieldOpts) m.Widget {
	return m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 0, 16), m.NewSizedBox(m.NewTextField(opts), m.SizedBoxOpts{Width: m.Some(columnWidth - 24)}))
}

func text(s string) m.Opt[string] { return m.Some(s) }

// FilledFieldsDemo has filled text fields: an empty one with a label, one with a
// hint and one that is disabled.
type FilledFieldsDemo struct{ m.StatelessWidget }

// NewFilledFieldsDemo creates the demo.
func NewFilledFieldsDemo() *FilledFieldsDemo { return &FilledFieldsDemo{} }

func (w *FilledFieldsDemo) Build(bc m.BuildContext) m.Widget {
	fill := m.InputDecoration{Filled: m.Some(true), Border: m.NewUnderlineInputBorder()}
	with := func(f func(d *m.InputDecoration)) *m.InputDecoration { d := fill; f(&d); return &d }
	return m.NewColumn([]m.Widget{
		field(m.TextFieldOpts{Decoration: with(func(d *m.InputDecoration) { d.LabelText = text("Name") })}),
		field(m.TextFieldOpts{Decoration: with(func(d *m.InputDecoration) {
			d.LabelText = text("Nickname")
			d.HintText = text("What friends call you")
			d.HelperText = text("Optional")
		})}),
		field(m.TextFieldOpts{Enabled: m.Some(false), Decoration: with(func(d *m.InputDecoration) { d.LabelText = text("Disabled") })}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// OutlinedFieldsDemo has outlined text fields; the first one has the focus when
// GLUTTER_DEV_TEXT_FIELDS asks for it.
type OutlinedFieldsDemo struct{ m.StatelessWidget }

// NewOutlinedFieldsDemo creates the demo.
func NewOutlinedFieldsDemo() *OutlinedFieldsDemo { return &OutlinedFieldsDemo{} }

func (w *OutlinedFieldsDemo) Build(bc m.BuildContext) m.Widget {
	outline := m.NewOutlineInputBorder()
	focus := devTextFields(os.Getenv)["focus"]
	return m.NewColumn([]m.Widget{
		field(m.TextFieldOpts{Autofocus: focus, Decoration: &m.InputDecoration{LabelText: text("Search"), Border: outline, HelperText: text("Type to search")}}),
		field(m.TextFieldOpts{Decoration: &m.InputDecoration{LabelText: text("City"), Border: outline, HintText: text("Athens")}}),
		field(m.TextFieldOpts{Enabled: m.Some(false), Decoration: &m.InputDecoration{LabelText: text("Disabled"), Border: outline}}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// AffixFieldsDemo has text fields with icons, prefixes and suffixes.
type AffixFieldsDemo struct{ m.StatelessWidget }

// NewAffixFieldsDemo creates the demo.
func NewAffixFieldsDemo() *AffixFieldsDemo { return &AffixFieldsDemo{} }

func (w *AffixFieldsDemo) Build(bc m.BuildContext) m.Widget {
	outline := m.NewOutlineInputBorder()
	return m.NewColumn([]m.Widget{
		field(m.TextFieldOpts{Decoration: &m.InputDecoration{LabelText: text("Email"), Border: outline,
			PrefixIcon: m.NewIcon(m.Icons.Email()), SuffixIcon: m.NewIcon(m.Icons.Check())}}),
		field(m.TextFieldOpts{KeyboardType: m.Some(m.TextInputTypeNumber), Decoration: &m.InputDecoration{LabelText: text("Amount"), Border: outline,
			PrefixText: text("$ "), SuffixText: text(" USD")}}),
		field(m.TextFieldOpts{Decoration: &m.InputDecoration{LabelText: text("Website"), Border: outline, PrefixText: text("https://"), Icon: m.NewIcon(m.Icons.Language())}}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// CounterFieldsDemo has a field with a character counter, one over its limit and
// one with an error.
type CounterFieldsDemo struct{ m.StatefulWidget }

// NewCounterFieldsDemo creates the demo.
func NewCounterFieldsDemo() *CounterFieldsDemo { return &CounterFieldsDemo{} }

// CreateState creates the state.
func (w *CounterFieldsDemo) CreateState() m.State { return &counterFieldsDemoState{} }

type counterFieldsDemoState struct {
	m.StateOf[*CounterFieldsDemo]
	short, over *m.TextEditingController
}

func (s *counterFieldsDemoState) InitState() {
	s.short = m.NewTextEditingController("Hello")
	s.over = m.NewTextEditingController("This text is too long")
}

func (s *counterFieldsDemoState) Dispose() {
	s.short.Dispose()
	s.over.Dispose()
}

func (s *counterFieldsDemoState) Build(bc m.BuildContext) m.Widget {
	outline := m.NewOutlineInputBorder()
	return m.NewColumn([]m.Widget{
		field(m.TextFieldOpts{Controller: s.short, MaxLength: m.Some(20), Decoration: &m.InputDecoration{LabelText: text("Title"), Border: outline}}),
		field(m.TextFieldOpts{Controller: s.over, MaxLength: m.Some(10), MaxLengthEnforcement: services.MaxLengthEnforcementNone,
			Decoration: &m.InputDecoration{LabelText: text("Short code"), Border: outline}}),
		field(m.TextFieldOpts{Decoration: &m.InputDecoration{LabelText: text("Username"), Border: outline, ErrorText: text("That name is taken")}}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// MultilineFieldsDemo has a multi-line field and a password field with a button
// that shows the password.
type MultilineFieldsDemo struct{ m.StatefulWidget }

// NewMultilineFieldsDemo creates the demo.
func NewMultilineFieldsDemo() *MultilineFieldsDemo { return &MultilineFieldsDemo{} }

// CreateState creates the state.
func (w *MultilineFieldsDemo) CreateState() m.State { return &multilineFieldsDemoState{} }

type multilineFieldsDemoState struct {
	m.StateOf[*MultilineFieldsDemo]
	showPassword bool
}

func (s *multilineFieldsDemoState) Build(bc m.BuildContext) m.Widget {
	outline := m.NewOutlineInputBorder()
	icon, tooltip := m.Icons.Visibility(), "Show the password"
	if s.showPassword {
		icon, tooltip = m.Icons.VisibilityOff(), "Hide the password"
	}
	return m.NewColumn([]m.Widget{
		field(m.TextFieldOpts{MinLines: m.Some(3), MaxLines: m.Some(5), Decoration: &m.InputDecoration{LabelText: text("Notes"), Border: outline,
			AlignLabelWithHint: m.Some(true), HintText: text("Up to five lines")}}),
		field(m.TextFieldOpts{ObscureText: !s.showPassword, Decoration: &m.InputDecoration{LabelText: text("Password"), Border: outline,
			SuffixIcon: m.NewIconButton(func() { s.SetState(func() { s.showPassword = !s.showPassword }) }, m.NewIcon(icon), m.IconButtonOpts{Tooltip: tooltip})}}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// FormDemo is a form with an e-mail field and a password field that are
// validated by a button, and by the typing once the form was validated.
type FormDemo struct{ m.StatefulWidget }

// NewFormDemo creates the demo.
func NewFormDemo() *FormDemo { return &FormDemo{} }

// CreateState creates the state.
func (w *FormDemo) CreateState() m.State { return &formDemoState{} }

type formDemoState struct {
	m.StateOf[*FormDemo]
	form   *m.GlobalKey[*m.FormState]
	status string
}

// InitState validates the form after the first frame when GLUTTER_DEV_TEXT_FIELDS
// asks for it.
func (s *formDemoState) InitState() {
	s.form = m.NewGlobalKey[*m.FormState]()
	s.status = "Not submitted"
	if devTextFields(os.Getenv)["validate"] {
		m.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
			if f, ok := s.form.CurrentState(); ok && s.Mounted() {
				f.Validate()
			}
		})
	}
}

func (s *formDemoState) Build(bc m.BuildContext) m.Widget {
	outline := m.NewOutlineInputBorder()
	return m.NewForm(m.NewColumn([]m.Widget{
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 0, 16), m.NewSizedBox(m.NewTextFormField(m.TextFormFieldOpts{
			TextFieldOpts: m.TextFieldOpts{Decoration: &m.InputDecoration{LabelText: text("Email"), Border: outline}, KeyboardType: m.Some(m.TextInputTypeEmailAddress)},
			Validator: func(v m.Opt[string]) m.Opt[string] {
				if !strings.Contains(v.OrElse(""), "@") {
					return text("Enter a valid email")
				}
				return m.None[string]()
			},
			AutovalidateMode: m.AutovalidateModeOnUserInteraction,
		}), m.SizedBoxOpts{Width: m.Some(columnWidth - 24)})),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 0, 16), m.NewSizedBox(m.NewTextFormField(m.TextFormFieldOpts{
			TextFieldOpts: m.TextFieldOpts{ObscureText: true, Decoration: &m.InputDecoration{LabelText: text("Password"), Border: outline}},
			Validator: func(v m.Opt[string]) m.Opt[string] {
				if len(v.OrElse("")) < 8 {
					return text("Use at least 8 characters")
				}
				return m.None[string]()
			},
		}), m.SizedBoxOpts{Width: m.Some(columnWidth - 24)})),
		m.NewRow([]m.Widget{
			m.NewFilledButton(func() {
				f, _ := s.form.CurrentState()
				ok := f.Validate()
				s.SetState(func() {
					s.status = "The form has errors"
					if ok {
						s.status = "Submitted"
					}
				})
			}, m.NewText("Validate")),
			m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(12.0)}),
			m.NewTextButton(func() {
				f, _ := s.form.CurrentState()
				f.Reset()
				s.SetState(func() { s.status = "Not submitted" })
			}, m.NewText("Reset")),
			m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(12.0)}),
			m.NewText(s.status),
		}, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentCenter}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart}), m.FormOpts{Key: s.form})
}

// DropdownFieldsDemo has a DropdownMenu and a DropdownButtonFormField.
type DropdownFieldsDemo struct{ m.StatefulWidget }

// NewDropdownFieldsDemo creates the demo.
func NewDropdownFieldsDemo() *DropdownFieldsDemo { return &DropdownFieldsDemo{} }

// CreateState creates the state.
func (w *DropdownFieldsDemo) CreateState() m.State { return &dropdownFieldsDemoState{} }

type dropdownFieldsDemoState struct {
	m.StateOf[*DropdownFieldsDemo]
	color string
}

func (s *dropdownFieldsDemoState) Build(bc m.BuildContext) m.Widget {
	entries := []m.DropdownMenuEntry[string]{
		m.NewDropdownMenuEntry("red", "Red"), m.NewDropdownMenuEntry("green", "Green"),
		m.NewDropdownMenuEntry("blue", "Blue"), m.NewDropdownMenuEntry("black", "Black", m.DropdownMenuEntryOpts{Disabled: true}),
	}
	var items []*m.DropdownMenuItem[string]
	for _, size := range []string{"Small", "Medium", "Large"} {
		items = append(items, m.NewDropdownMenuItem(m.NewText(size), m.DropdownMenuItemOpts[string]{Value: m.Some(size)}))
	}
	return m.NewColumn([]m.Widget{
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 0, 16), m.NewDropdownMenu(entries, m.DropdownMenuOpts[string]{
			Width: m.Some(220.0), Label: m.NewText("Color"), LeadingIcon: m.NewIcon(m.Icons.Palette()),
			OnSelected: func(v m.Opt[string]) { s.SetState(func() { s.color = v.OrElse("") }) },
		})),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 0, 16), m.NewText("Selected color: "+s.color)),
		m.NewSizedBox(m.NewDropdownButtonFormField(items, func(m.Opt[string]) {}, m.DropdownButtonFormFieldOpts[string]{
			Decoration: &m.InputDecoration{LabelText: text("Size"), Border: m.NewOutlineInputBorder()}, IsExpanded: true,
			Validator: func(v m.Opt[string]) m.Opt[string] {
				if v.IsNone() {
					return text("Pick a size")
				}
				return m.None[string]()
			},
			AutovalidateMode: m.AutovalidateModeOnUserInteraction,
		}), m.SizedBoxOpts{Width: m.Some(220.0)}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// SelectionAreaDemo has a text that the user selects with the pointer, with a
// context menu on the right button.
type SelectionAreaDemo struct{ m.StatelessWidget }

// NewSelectionAreaDemo creates the demo.
func NewSelectionAreaDemo() *SelectionAreaDemo { return &SelectionAreaDemo{} }

func (w *SelectionAreaDemo) Build(bc m.BuildContext) m.Widget {
	theme := m.ThemeOf(bc)
	return m.NewSizedBox(m.NewSelectionArea(m.NewColumn([]m.Widget{
		m.NewText("Select this text", m.TextOpts{Style: theme.TextTheme.TitleMedium}),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 8, 0, 0), m.NewText("Drag the pointer over it, double-click a word, or press the right button for the menu.")),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})), m.SizedBoxOpts{Width: m.Some(columnWidth - 24)})
}
