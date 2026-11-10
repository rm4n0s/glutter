package a11y

import "github.com/rm4n0s/glutter/internal/platform/darwin/a11ymodel"

// The model of the semantics trees is shared with iOS (internal/platform/darwin/a11ymodel); these are its names in this package.

// RootID is the ID of the root node of the semantics tree of a view.
const RootID = a11ymodel.RootID

type (
	// NodeRef names a node of a view.
	//
	// Flutter: no counterpart
	NodeRef = a11ymodel.NodeRef
	// Node is a node of the mirrored tree.
	//
	// Flutter: no counterpart
	Node = a11ymodel.Node
	// Model is the mirror of the semantics trees of the views of the app.
	//
	// Flutter: no counterpart
	Model = a11ymodel.Model
	// ChangeKind says what a Change is.
	//
	// Flutter: no counterpart
	ChangeKind = a11ymodel.ChangeKind
	// Fields is a set of the properties of a node that Apply watches.
	//
	// Flutter: no counterpart
	Fields = a11ymodel.Fields
	// Change is one change that Apply or the window setters made.
	//
	// Flutter: no counterpart
	Change = a11ymodel.Change
	// Rect is a rectangle in physical pixels.
	//
	// Flutter: no counterpart
	Rect = a11ymodel.Rect
)

// NewModel makes an empty model whose nodes are seen as NSAccessibility sees them (the mapping of props.go).
//
// Flutter: no counterpart
func NewModel() *Model { return a11ymodel.NewModel(describe) }

// describe is the Describer of macOS: the role, label, help, value and state that props.go works out.
func describe(m *Model, n *Node) a11ymodel.Description {
	caret, base := Selection(n)
	return a11ymodel.Description{
		Role: Role(n), Label: LabelOf(m, n), Help: Help(n), Value: ValueText(n),
		Enabled: Enabled(n), Selected: Selected(n), Expanded: ExpandedState(n), Caret: caret, Base: base,
	}
}

// The kinds of change.
const (
	ChangeNodeAdded     = a11ymodel.ChangeNodeAdded
	ChangeNodeRemoved   = a11ymodel.ChangeNodeRemoved
	ChangeChildren      = a11ymodel.ChangeChildren
	ChangeUpdated       = a11ymodel.ChangeUpdated
	ChangeFocus         = a11ymodel.ChangeFocus
	ChangeAnnounce      = a11ymodel.ChangeAnnounce
	ChangeTextSelection = a11ymodel.ChangeTextSelection
	ChangeWindowOpened  = a11ymodel.ChangeWindowOpened
)

// The watched properties.
const (
	FieldRole     = a11ymodel.FieldRole
	FieldLabel    = a11ymodel.FieldLabel
	FieldHelp     = a11ymodel.FieldHelp
	FieldValue    = a11ymodel.FieldValue
	FieldEnabled  = a11ymodel.FieldEnabled
	FieldSelected = a11ymodel.FieldSelected
	FieldExpanded = a11ymodel.FieldExpanded
	FieldFrame    = a11ymodel.FieldFrame
	FieldFlags    = a11ymodel.FieldFlags
)
