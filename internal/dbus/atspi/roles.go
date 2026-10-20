package atspi

import "github.com/rm4n0s/glutter/ui"

// Role is an AT-SPI2 role (AtspiRole of atspi-constants.h). Only the roles that the
// mapping uses are named.
type Role uint32

// The roles that the mapping gives to nodes.
const (
	RoleInvalid           Role = 0
	RoleAcceleratorLabel  Role = 1
	RoleAlert             Role = 2
	RoleAnimation         Role = 3
	RoleCheckBox          Role = 7
	RoleCheckMenuItem     Role = 8
	RoleColumnHeader      Role = 10
	RoleComboBox          Role = 11
	RoleDialog            Role = 16
	RoleFiller            Role = 20
	RoleFrame             Role = 23
	RoleImage             Role = 27
	RoleLabel             Role = 29
	RoleList              Role = 31
	RoleListItem          Role = 32
	RoleMenu              Role = 33
	RoleMenuBar           Role = 34
	RoleMenuItem          Role = 35
	RolePageTab           Role = 37
	RolePageTabList       Role = 38
	RolePanel             Role = 39
	RolePasswordText      Role = 40
	RoleProgressBar       Role = 42
	RolePushButton        Role = 43
	RoleRadioButton       Role = 44
	RoleRadioMenuItem     Role = 45
	RoleScrollBar         Role = 48
	RoleScrollPane        Role = 49
	RoleSeparator         Role = 50
	RoleSlider            Role = 51
	RoleSpinButton        Role = 52
	RoleStatusBar         Role = 54
	RoleTable             Role = 55
	RoleTableCell         Role = 56
	RoleTableColumnHeader Role = 57
	RoleText              Role = 61
	RoleToggleButton      Role = 62
	RoleToolTip           Role = 64
	RoleTree              Role = 65
	RoleWindow            Role = 69
	RoleHeader            Role = 71
	RoleFooter            Role = 72
	RoleParagraph         Role = 73
	RoleApplication       Role = 75
	RoleEntry             Role = 79
	RoleDocumentFrame     Role = 82
	RoleHeading           Role = 83
	RoleSection           Role = 85
	RoleForm              Role = 87
	RoleLink              Role = 88
	RoleTableRow          Role = 90
	RoleListBox           Role = 98
	RoleGrouping          Role = 99
	RoleNotification      Role = 101
	RoleLandmark          Role = 110
	RoleStatic            Role = 116
)

// roleNames are the English names that GetRoleName returns, from atspi-constants.h
// (and at-spi2-core's role-name table); the localized name is the same here.
var roleNames = map[Role]string{
	RoleInvalid: "invalid", RoleAcceleratorLabel: "accelerator label", RoleAlert: "alert", RoleAnimation: "animation",
	RoleCheckBox: "check box", RoleCheckMenuItem: "check menu item", RoleColumnHeader: "column header",
	RoleComboBox: "combo box", RoleDialog: "dialog", RoleFiller: "filler", RoleFrame: "frame", RoleImage: "image",
	RoleLabel: "label", RoleList: "list", RoleListItem: "list item", RoleMenu: "menu", RoleMenuBar: "menu bar",
	RoleMenuItem: "menu item", RolePageTab: "page tab", RolePageTabList: "page tab list", RolePanel: "panel",
	RolePasswordText: "password text", RoleProgressBar: "progress bar", RolePushButton: "push button",
	RoleRadioButton: "radio button", RoleRadioMenuItem: "radio menu item", RoleScrollBar: "scroll bar",
	RoleScrollPane: "scroll pane", RoleSeparator: "separator", RoleSlider: "slider", RoleSpinButton: "spin button",
	RoleStatusBar: "status bar", RoleTable: "table", RoleTableCell: "table cell",
	RoleTableColumnHeader: "table column header", RoleText: "text", RoleToggleButton: "toggle button",
	RoleToolTip: "tool tip", RoleTree: "tree", RoleWindow: "window", RoleHeader: "header", RoleFooter: "footer",
	RoleParagraph: "paragraph", RoleApplication: "application", RoleEntry: "entry",
	RoleDocumentFrame: "document frame", RoleHeading: "heading", RoleSection: "section", RoleForm: "form",
	RoleLink: "link", RoleTableRow: "table row", RoleListBox: "list box", RoleGrouping: "grouping",
	RoleNotification: "notification", RoleLandmark: "landmark", RoleStatic: "static",
}

// Name is the English name of the role, such as "push button".
func (r Role) Name() string {
	if n, ok := roleNames[r]; ok {
		return n
	}
	return "unknown"
}

// semanticsRoles maps the SemanticsRole of a node to the AT-SPI2 role. The choices
// follow what browsers do for the ARIA role of the same name (Chromium's
// ax_platform_node_auralinux.cc), since screen readers know those best. A
// SemanticsRole that is not here (none) is decided by the flags, in roleOf.
var semanticsRoles = map[ui.SemanticsRole]Role{
	ui.SemanticsRoleTab:              RolePageTab,
	ui.SemanticsRoleTabBar:           RolePageTabList,
	ui.SemanticsRoleTabPanel:         RolePanel, // AT-SPI2 has no tab panel
	ui.SemanticsRoleDialog:           RoleDialog,
	ui.SemanticsRoleAlertDialog:      RoleAlert,
	ui.SemanticsRoleTable:            RoleTable,
	ui.SemanticsRoleCell:             RoleTableCell,
	ui.SemanticsRoleRow:              RoleTableRow,
	ui.SemanticsRoleColumnHeader:     RoleTableColumnHeader,
	ui.SemanticsRoleDragHandle:       RolePushButton,
	ui.SemanticsRoleSpinButton:       RoleSpinButton,
	ui.SemanticsRoleComboBox:         RoleComboBox,
	ui.SemanticsRoleMenuBar:          RoleMenuBar,
	ui.SemanticsRoleMenu:             RoleMenu,
	ui.SemanticsRoleMenuItem:         RoleMenuItem,
	ui.SemanticsRoleMenuItemCheckbox: RoleCheckMenuItem,
	ui.SemanticsRoleMenuItemRadio:    RoleRadioMenuItem,
	ui.SemanticsRoleAlert:            RoleNotification,
	ui.SemanticsRoleStatus:           RoleStatusBar,
	ui.SemanticsRoleList:             RoleList,
	ui.SemanticsRoleListItem:         RoleListItem,
	ui.SemanticsRoleComplementary:    RoleLandmark,
	ui.SemanticsRoleContentInfo:      RoleLandmark,
	ui.SemanticsRoleMain:             RoleLandmark,
	ui.SemanticsRoleNavigation:       RoleLandmark,
	ui.SemanticsRoleRegion:           RoleLandmark,
	ui.SemanticsRoleForm:             RoleForm,
	ui.SemanticsRoleLoadingSpinner:   RoleProgressBar,
	ui.SemanticsRoleProgressBar:      RoleProgressBar,
	ui.SemanticsRoleRadioGroup:       RolePanel,
	ui.SemanticsRoleTooltip:          RoleToolTip,
	ui.SemanticsRoleHotKey:           RoleAcceleratorLabel,
}

// landmarkNames are the ARIA names of the landmark roles, given as the attribute
// "xml-roles" so that a screen reader can say which landmark it is.
var landmarkNames = map[ui.SemanticsRole]string{
	ui.SemanticsRoleComplementary: "complementary",
	ui.SemanticsRoleContentInfo:   "contentinfo",
	ui.SemanticsRoleMain:          "main",
	ui.SemanticsRoleNavigation:    "navigation",
	ui.SemanticsRoleRegion:        "region",
}

// roleOf is the AT-SPI2 role of a node. The SemanticsRole decides when there is one;
// otherwise the flags do, in this order: text field, slider, link, header, image,
// then the buttons (a check box, a radio button and a toggle button are buttons that
// are checkable, in a group, or toggled), and what has text is a label; the rest is a
// panel. The root of a view (isRoot) is the frame of the window.
func roleOf(n *Node, isRoot bool) Role {
	if isRoot {
		return RoleFrame
	}
	if r, ok := semanticsRoles[n.Role]; ok {
		return r
	}
	f := n.Flags
	switch {
	case f.IsTextField && f.IsObscured:
		return RolePasswordText
	case f.IsTextField:
		return RoleEntry
	case f.IsSlider:
		return RoleSlider
	case f.IsLink:
		return RoleLink
	case f.IsHeader:
		return RoleHeading
	case f.IsImage:
		return RoleImage
	case f.IsButton && f.IsChecked != ui.CheckedStateNone && f.IsInMutuallyExclusiveGroup:
		return RoleRadioButton
	case f.IsButton && f.IsChecked != ui.CheckedStateNone:
		return RoleCheckBox
	case f.IsButton && f.IsToggled != ui.TristateNone:
		return RoleToggleButton
	case f.IsButton:
		return RolePushButton
	case f.IsChecked != ui.CheckedStateNone && f.IsInMutuallyExclusiveGroup:
		return RoleRadioButton
	case f.IsChecked != ui.CheckedStateNone:
		return RoleCheckBox
	case f.IsToggled != ui.TristateNone:
		return RoleToggleButton
	case n.Label != "" || n.Value != "" || n.Hint != "":
		return RoleLabel
	}
	return RolePanel
}

// State is an AT-SPI2 state (AtspiStateType): the number of a bit in the 64 bits of
// the two words that GetState returns.
type State uint32

// The states that the mapping sets.
const (
	StateActive          State = 1
	StateChecked         State = 4
	StateCollapsed       State = 5
	StateEditable        State = 7
	StateEnabled         State = 8
	StateExpandable      State = 9
	StateExpanded        State = 10
	StateFocusable       State = 11
	StateFocused         State = 12
	StateHasTooltip      State = 13
	StateModal           State = 16
	StateMultiLine       State = 17
	StateSelectable      State = 22
	StateSelected        State = 23
	StateSensitive       State = 24
	StateShowing         State = 25
	StateSingleLine      State = 26
	StateVisible         State = 30
	StateIndeterminate   State = 32
	StateRequired        State = 33
	StateInvalidEntry    State = 36
	StateSelectableText  State = 38
	StateCheckable       State = 41
	StateReadOnly        State = 43
	stateLast            State = 43
	stateWordsInProtocol       = 2
)

// stateNames are the names of the states in the events state-changed ("focused",
// "invalid-entry"), from atspi-constants.h (the nick of the AtspiStateType value).
var stateNames = map[State]string{
	StateActive: "active", StateChecked: "checked", StateCollapsed: "collapsed", StateEditable: "editable",
	StateEnabled: "enabled", StateExpandable: "expandable", StateExpanded: "expanded", StateFocusable: "focusable",
	StateFocused: "focused", StateHasTooltip: "has-tooltip", StateModal: "modal", StateMultiLine: "multi-line",
	StateSelectable: "selectable", StateSelected: "selected", StateSensitive: "sensitive", StateShowing: "showing",
	StateSingleLine: "single-line", StateVisible: "visible", StateIndeterminate: "indeterminate",
	StateRequired: "required", StateInvalidEntry: "invalid-entry", StateSelectableText: "selectable-text",
	StateCheckable: "checkable", StateReadOnly: "read-only",
}

// Name is the name of the state in the events, such as "focused".
func (s State) Name() string { return stateNames[s] }

// StateSet is a set of states.
type StateSet uint64

// Has says whether the set has the state.
func (s StateSet) Has(st State) bool { return s&(1<<st) != 0 }

// With returns the set with the state added.
func (s StateSet) With(st State) StateSet { return s | 1<<st }

// Words are the two 32-bit words of the set, as GetState returns them: the low
// word has the states 0 to 31.
func (s StateSet) Words() []uint32 { return []uint32{uint32(s), uint32(s >> 32)} }

// statesOf maps the flags and actions of a node to the AT-SPI2 states. active is
// whether the node is a frame of the window that has the focus.
//
//	SemanticsFlags               states
//	IsEnabled none or true       ENABLED, SENSITIVE
//	IsHidden                     (none of) SHOWING, VISIBLE
//	IsFocused (any), Focus       FOCUSABLE; FOCUSED when true
//	IsChecked, IsToggled         CHECKABLE; CHECKED when true; INDETERMINATE when mixed
//	IsSelected                   SELECTABLE; SELECTED when true
//	IsExpanded                   EXPANDABLE; EXPANDED when true, COLLAPSED when false
//	IsRequired true              REQUIRED
//	IsTextField                  SINGLE_LINE or MULTI_LINE; EDITABLE unless IsReadOnly; READ_ONLY if it is
//	ScopesRoute and a dialog     MODAL
//	ValidationResult invalid     INVALID_ENTRY
//	Tooltip                      HAS_TOOLTIP
func statesOf(n *Node, isRoot, active bool) StateSet {
	var s StateSet
	f := n.Flags
	if f.IsEnabled != ui.TristateIsFalse {
		s = s.With(StateEnabled).With(StateSensitive)
	}
	if !f.IsHidden {
		s = s.With(StateShowing).With(StateVisible)
	}
	if f.IsFocused != ui.TristateNone || n.Actions&ui.SemanticsActionFocus != 0 {
		s = s.With(StateFocusable)
	}
	if f.IsFocused == ui.TristateIsTrue {
		s = s.With(StateFocused)
	}
	switch f.IsChecked {
	case ui.CheckedStateIsTrue:
		s = s.With(StateCheckable).With(StateChecked)
	case ui.CheckedStateIsFalse:
		s = s.With(StateCheckable)
	case ui.CheckedStateMixed:
		s = s.With(StateCheckable).With(StateIndeterminate)
	}
	switch f.IsToggled {
	case ui.TristateIsTrue:
		s = s.With(StateCheckable).With(StateChecked)
	case ui.TristateIsFalse:
		s = s.With(StateCheckable)
	}
	switch f.IsSelected {
	case ui.TristateIsTrue:
		s = s.With(StateSelectable).With(StateSelected)
	case ui.TristateIsFalse:
		s = s.With(StateSelectable)
	}
	switch f.IsExpanded {
	case ui.TristateIsTrue:
		s = s.With(StateExpandable).With(StateExpanded)
	case ui.TristateIsFalse:
		s = s.With(StateExpandable).With(StateCollapsed)
	}
	if f.IsRequired == ui.TristateIsTrue {
		s = s.With(StateRequired)
	}
	if f.IsTextField {
		if f.IsMultiline {
			s = s.With(StateMultiLine)
		} else {
			s = s.With(StateSingleLine)
		}
		if f.IsReadOnly {
			s = s.With(StateReadOnly)
		} else {
			s = s.With(StateEditable)
		}
		s = s.With(StateSelectableText)
	}
	if f.ScopesRoute && (n.Role == ui.SemanticsRoleDialog || n.Role == ui.SemanticsRoleAlertDialog) {
		s = s.With(StateModal)
	}
	if n.ValidationResult == ui.SemanticsValidationResultInvalid {
		s = s.With(StateInvalidEntry)
	}
	if n.Tooltip != "" {
		s = s.With(StateHasTooltip)
	}
	if isRoot && active {
		s = s.With(StateActive)
	}
	return s
}
