package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// SemanticsOpts are the optional parameters of NewSemantics: the options of the
// widget and every property of semantics.SemanticsProperties, with the same names
// (Flutter's Semantics.new has them as named parameters).
//
// Flutter: widgets/basic.dart Semantics.new
type SemanticsOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Container is whether the node forms a semantic boundary of its own, so that
	// it is not merged into its parent.
	Container bool
	// ExplicitChildNodes is whether the children form nodes of their own instead
	// of merging into this one.
	ExplicitChildNodes bool
	// ExcludeSemantics drops the semantics of the subtree.
	ExcludeSemantics bool
	// BlockUserActions blocks the user actions of the subtree.
	BlockUserActions bool
	// LocaleForSubtree is the locale of the strings of the subtree.
	LocaleForSubtree foundation.Opt[ui.Locale]

	// Enabled is the property of the same name of semantics.SemanticsProperties.
	Enabled foundation.Opt[bool]
	// Checked is the property of the same name of semantics.SemanticsProperties.
	Checked foundation.Opt[bool]
	// Mixed is the property of the same name of semantics.SemanticsProperties.
	Mixed foundation.Opt[bool]
	// Expanded is the property of the same name of semantics.SemanticsProperties.
	Expanded foundation.Opt[bool]
	// Selected is the property of the same name of semantics.SemanticsProperties.
	Selected foundation.Opt[bool]
	// Toggled is the property of the same name of semantics.SemanticsProperties.
	Toggled foundation.Opt[bool]
	// Button is the property of the same name of semantics.SemanticsProperties.
	Button foundation.Opt[bool]
	// Link is the property of the same name of semantics.SemanticsProperties.
	Link foundation.Opt[bool]
	// Header is the property of the same name of semantics.SemanticsProperties.
	Header foundation.Opt[bool]
	// TextField is the property of the same name of semantics.SemanticsProperties.
	TextField foundation.Opt[bool]
	// Slider is the property of the same name of semantics.SemanticsProperties.
	Slider foundation.Opt[bool]
	// KeyboardKey is the property of the same name of semantics.SemanticsProperties.
	KeyboardKey foundation.Opt[bool]
	// ReadOnly is the property of the same name of semantics.SemanticsProperties.
	ReadOnly foundation.Opt[bool]
	// Focusable is the property of the same name of semantics.SemanticsProperties.
	Focusable foundation.Opt[bool]
	// Focused is the property of the same name of semantics.SemanticsProperties.
	Focused foundation.Opt[bool]
	// InMutuallyExclusiveGroup is the property of the same name of semantics.SemanticsProperties.
	InMutuallyExclusiveGroup foundation.Opt[bool]
	// Hidden is the property of the same name of semantics.SemanticsProperties.
	Hidden foundation.Opt[bool]
	// Obscured is the property of the same name of semantics.SemanticsProperties.
	Obscured foundation.Opt[bool]
	// Multiline is the property of the same name of semantics.SemanticsProperties.
	Multiline foundation.Opt[bool]
	// ScopesRoute is the property of the same name of semantics.SemanticsProperties.
	ScopesRoute foundation.Opt[bool]
	// NamesRoute is the property of the same name of semantics.SemanticsProperties.
	NamesRoute foundation.Opt[bool]
	// Image is the property of the same name of semantics.SemanticsProperties.
	Image foundation.Opt[bool]
	// LiveRegion is the property of the same name of semantics.SemanticsProperties.
	LiveRegion foundation.Opt[bool]
	// IsRequired is the property of the same name of semantics.SemanticsProperties.
	IsRequired foundation.Opt[bool]
	// LinkURL is the property of the same name of semantics.SemanticsProperties.
	LinkURL foundation.Opt[string]
	// HeadingLevel is the property of the same name of semantics.SemanticsProperties.
	HeadingLevel foundation.Opt[int]
	// AccessibilityFocusBlockType is the property of the same name of semantics.SemanticsProperties.
	AccessibilityFocusBlockType foundation.Opt[semantics.AccessibilityFocusBlockType]
	// MaxValueLength is the property of the same name of semantics.SemanticsProperties.
	MaxValueLength foundation.Opt[int]
	// CurrentValueLength is the property of the same name of semantics.SemanticsProperties.
	CurrentValueLength foundation.Opt[int]
	// Identifier is the property of the same name of semantics.SemanticsProperties.
	Identifier foundation.Opt[string]
	// TraversalParentIdentifier is the property of the same name of semantics.SemanticsProperties.
	TraversalParentIdentifier any
	// TraversalChildIdentifier is the property of the same name of semantics.SemanticsProperties.
	TraversalChildIdentifier any
	// Label is the property of the same name of semantics.SemanticsProperties.
	Label foundation.Opt[string]
	// Value is the property of the same name of semantics.SemanticsProperties.
	Value foundation.Opt[string]
	// IncreasedValue is the property of the same name of semantics.SemanticsProperties.
	IncreasedValue foundation.Opt[string]
	// DecreasedValue is the property of the same name of semantics.SemanticsProperties.
	DecreasedValue foundation.Opt[string]
	// Hint is the property of the same name of semantics.SemanticsProperties.
	Hint foundation.Opt[string]
	// AttributedLabel is the property of the same name of semantics.SemanticsProperties.
	AttributedLabel *semantics.AttributedString
	// AttributedValue is the property of the same name of semantics.SemanticsProperties.
	AttributedValue *semantics.AttributedString
	// AttributedIncreasedValue is the property of the same name of semantics.SemanticsProperties.
	AttributedIncreasedValue *semantics.AttributedString
	// AttributedDecreasedValue is the property of the same name of semantics.SemanticsProperties.
	AttributedDecreasedValue *semantics.AttributedString
	// AttributedHint is the property of the same name of semantics.SemanticsProperties.
	AttributedHint *semantics.AttributedString
	// Tooltip is the property of the same name of semantics.SemanticsProperties.
	Tooltip foundation.Opt[string]
	// HintOverrides is the property of the same name of semantics.SemanticsProperties.
	HintOverrides *semantics.SemanticsHintOverrides
	// TextDirection is the property of the same name of semantics.SemanticsProperties.
	TextDirection ui.TextDirection
	// SortKey is the property of the same name of semantics.SemanticsProperties.
	SortKey semantics.SemanticsSortKey
	// TagForChildren is the property of the same name of semantics.SemanticsProperties.
	TagForChildren foundation.Opt[semantics.SemanticsTag]
	// Role is the property of the same name of semantics.SemanticsProperties.
	Role foundation.Opt[ui.SemanticsRole]
	// ControlsNodes is the property of the same name of semantics.SemanticsProperties.
	ControlsNodes []string
	// InputType is the property of the same name of semantics.SemanticsProperties.
	InputType foundation.Opt[ui.SemanticsInputType]
	// ValidationResult is the property of the same name of semantics.SemanticsProperties.
	ValidationResult ui.SemanticsValidationResult
	// HitTestBehavior is the property of the same name of semantics.SemanticsProperties.
	HitTestBehavior foundation.Opt[ui.SemanticsHitTestBehavior]
	// OnTap is the property of the same name of semantics.SemanticsProperties.
	OnTap func()
	// OnLongPress is the property of the same name of semantics.SemanticsProperties.
	OnLongPress func()
	// OnScrollLeft is the property of the same name of semantics.SemanticsProperties.
	OnScrollLeft func()
	// OnScrollRight is the property of the same name of semantics.SemanticsProperties.
	OnScrollRight func()
	// OnScrollUp is the property of the same name of semantics.SemanticsProperties.
	OnScrollUp func()
	// OnScrollDown is the property of the same name of semantics.SemanticsProperties.
	OnScrollDown func()
	// OnIncrease is the property of the same name of semantics.SemanticsProperties.
	OnIncrease func()
	// OnDecrease is the property of the same name of semantics.SemanticsProperties.
	OnDecrease func()
	// OnCopy is the property of the same name of semantics.SemanticsProperties.
	OnCopy func()
	// OnCut is the property of the same name of semantics.SemanticsProperties.
	OnCut func()
	// OnPaste is the property of the same name of semantics.SemanticsProperties.
	OnPaste func()
	// OnMoveCursorForwardByCharacter is the property of the same name of semantics.SemanticsProperties.
	OnMoveCursorForwardByCharacter semantics.MoveCursorHandler
	// OnMoveCursorBackwardByCharacter is the property of the same name of semantics.SemanticsProperties.
	OnMoveCursorBackwardByCharacter semantics.MoveCursorHandler
	// OnMoveCursorForwardByWord is the property of the same name of semantics.SemanticsProperties.
	OnMoveCursorForwardByWord semantics.MoveCursorHandler
	// OnMoveCursorBackwardByWord is the property of the same name of semantics.SemanticsProperties.
	OnMoveCursorBackwardByWord semantics.MoveCursorHandler
	// OnSetSelection is the property of the same name of semantics.SemanticsProperties.
	OnSetSelection semantics.SetSelectionHandler
	// OnSetText is the property of the same name of semantics.SemanticsProperties.
	OnSetText semantics.SetTextHandler
	// OnDidGainAccessibilityFocus is the property of the same name of semantics.SemanticsProperties.
	OnDidGainAccessibilityFocus func()
	// OnDidLoseAccessibilityFocus is the property of the same name of semantics.SemanticsProperties.
	OnDidLoseAccessibilityFocus func()
	// OnFocus is the property of the same name of semantics.SemanticsProperties.
	OnFocus func()
	// OnDismiss is the property of the same name of semantics.SemanticsProperties.
	OnDismiss func()
	// OnExpand is the property of the same name of semantics.SemanticsProperties.
	OnExpand func()
	// OnCollapse is the property of the same name of semantics.SemanticsProperties.
	OnCollapse func()
	// CustomSemanticsActions is the property of the same name of semantics.SemanticsProperties.
	CustomSemanticsActions map[semantics.CustomSemanticsAction]func()
	// MinValue is the property of the same name of semantics.SemanticsProperties.
	MinValue foundation.Opt[string]
	// MaxValue is the property of the same name of semantics.SemanticsProperties.
	MaxValue foundation.Opt[string]
}

// properties makes the SemanticsProperties of the options.
func (o SemanticsOpts) properties() *semantics.SemanticsProperties {
	return semantics.NewSemanticsProperties(semantics.SemanticsProperties{
		Enabled:                         o.Enabled,
		Checked:                         o.Checked,
		Mixed:                           o.Mixed,
		Expanded:                        o.Expanded,
		Selected:                        o.Selected,
		Toggled:                         o.Toggled,
		Button:                          o.Button,
		Link:                            o.Link,
		Header:                          o.Header,
		TextField:                       o.TextField,
		Slider:                          o.Slider,
		KeyboardKey:                     o.KeyboardKey,
		ReadOnly:                        o.ReadOnly,
		Focusable:                       o.Focusable,
		Focused:                         o.Focused,
		InMutuallyExclusiveGroup:        o.InMutuallyExclusiveGroup,
		Hidden:                          o.Hidden,
		Obscured:                        o.Obscured,
		Multiline:                       o.Multiline,
		ScopesRoute:                     o.ScopesRoute,
		NamesRoute:                      o.NamesRoute,
		Image:                           o.Image,
		LiveRegion:                      o.LiveRegion,
		IsRequired:                      o.IsRequired,
		LinkURL:                         o.LinkURL,
		HeadingLevel:                    o.HeadingLevel,
		AccessibilityFocusBlockType:     o.AccessibilityFocusBlockType,
		MaxValueLength:                  o.MaxValueLength,
		CurrentValueLength:              o.CurrentValueLength,
		Identifier:                      o.Identifier,
		TraversalParentIdentifier:       o.TraversalParentIdentifier,
		TraversalChildIdentifier:        o.TraversalChildIdentifier,
		Label:                           o.Label,
		Value:                           o.Value,
		IncreasedValue:                  o.IncreasedValue,
		DecreasedValue:                  o.DecreasedValue,
		Hint:                            o.Hint,
		AttributedLabel:                 o.AttributedLabel,
		AttributedValue:                 o.AttributedValue,
		AttributedIncreasedValue:        o.AttributedIncreasedValue,
		AttributedDecreasedValue:        o.AttributedDecreasedValue,
		AttributedHint:                  o.AttributedHint,
		Tooltip:                         o.Tooltip,
		HintOverrides:                   o.HintOverrides,
		TextDirection:                   o.TextDirection,
		SortKey:                         o.SortKey,
		TagForChildren:                  o.TagForChildren,
		Role:                            o.Role,
		ControlsNodes:                   o.ControlsNodes,
		InputType:                       o.InputType,
		ValidationResult:                o.ValidationResult,
		HitTestBehavior:                 o.HitTestBehavior,
		OnTap:                           o.OnTap,
		OnLongPress:                     o.OnLongPress,
		OnScrollLeft:                    o.OnScrollLeft,
		OnScrollRight:                   o.OnScrollRight,
		OnScrollUp:                      o.OnScrollUp,
		OnScrollDown:                    o.OnScrollDown,
		OnIncrease:                      o.OnIncrease,
		OnDecrease:                      o.OnDecrease,
		OnCopy:                          o.OnCopy,
		OnCut:                           o.OnCut,
		OnPaste:                         o.OnPaste,
		OnMoveCursorForwardByCharacter:  o.OnMoveCursorForwardByCharacter,
		OnMoveCursorBackwardByCharacter: o.OnMoveCursorBackwardByCharacter,
		OnMoveCursorForwardByWord:       o.OnMoveCursorForwardByWord,
		OnMoveCursorBackwardByWord:      o.OnMoveCursorBackwardByWord,
		OnSetSelection:                  o.OnSetSelection,
		OnSetText:                       o.OnSetText,
		OnDidGainAccessibilityFocus:     o.OnDidGainAccessibilityFocus,
		OnDidLoseAccessibilityFocus:     o.OnDidLoseAccessibilityFocus,
		OnFocus:                         o.OnFocus,
		OnDismiss:                       o.OnDismiss,
		OnExpand:                        o.OnExpand,
		OnCollapse:                      o.OnCollapse,
		CustomSemanticsActions:          o.CustomSemanticsActions,
		MinValue:                        o.MinValue,
		MaxValue:                        o.MaxValue,
	})
}

// Semantics annotates the widget tree with a description of the meaning of the
// widgets: the labels, the flags and the actions that assistive technologies
// use.
//
// Flutter: widgets/basic.dart Semantics
type Semantics struct {
	SingleChildRenderObjectWidget
	properties         *semantics.SemanticsProperties
	container          bool
	explicitChildNodes bool
	excludeSemantics   bool
	blockUserActions   bool
	localeForSubtree   foundation.Opt[ui.Locale]
}

// NewSemantics creates a widget that annotates its child with semantics. A
// container widget (Container: true) forms a node of its own; without it the
// annotations merge into the node of the closest ancestor that has one.
//
// Flutter: widgets/basic.dart Semantics.new
func NewSemantics(child Widget, opts ...SemanticsOpts) *Semantics {
	o := firstOpts(opts)
	return &Semantics{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), o.properties(), o.Container, o.ExplicitChildNodes,
		o.ExcludeSemantics, o.BlockUserActions, o.LocaleForSubtree}
}

// SemanticsFromPropertiesOpts are the optional parameters of
// NewSemanticsFromProperties.
//
// Flutter: widgets/basic.dart Semantics.fromProperties
type SemanticsFromPropertiesOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Container is whether the node forms a semantic boundary of its own.
	Container bool
	// ExplicitChildNodes is whether the children form nodes of their own.
	ExplicitChildNodes bool
	// ExcludeSemantics drops the semantics of the subtree.
	ExcludeSemantics bool
	// BlockUserActions blocks the user actions of the subtree.
	BlockUserActions bool
	// LocaleForSubtree is the locale of the strings of the subtree.
	LocaleForSubtree foundation.Opt[ui.Locale]
}

// NewSemanticsFromProperties creates a widget with semantics given as a
// SemanticsProperties.
//
// Flutter: widgets/basic.dart Semantics.fromProperties
func NewSemanticsFromProperties(properties *semantics.SemanticsProperties, child Widget, opts ...SemanticsFromPropertiesOpts) *Semantics {
	o := firstOpts(opts)
	return &Semantics{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), properties, o.Container, o.ExplicitChildNodes,
		o.ExcludeSemantics, o.BlockUserActions, o.LocaleForSubtree}
}

// Properties are the semantics properties of the widget.
//
// Flutter: widgets/basic.dart Semantics.properties
func (w *Semantics) Properties() *semantics.SemanticsProperties { return w.properties }

// Container is whether the widget forms a semantic boundary of its own.
//
// Flutter: widgets/basic.dart Semantics.container
func (w *Semantics) Container() bool { return w.container }

// ExplicitChildNodes is whether the children form nodes of their own.
//
// Flutter: widgets/basic.dart Semantics.explicitChildNodes
func (w *Semantics) ExplicitChildNodes() bool { return w.explicitChildNodes }

// ExcludeSemantics is whether the semantics of the subtree are dropped.
//
// Flutter: widgets/basic.dart Semantics.excludeSemantics
func (w *Semantics) ExcludeSemantics() bool { return w.excludeSemantics }

// BlockUserActions is whether the user actions of the subtree are blocked.
//
// Flutter: widgets/basic.dart Semantics.blockUserActions
func (w *Semantics) BlockUserActions() bool { return w.blockUserActions }

// LocaleForSubtree is the locale of the strings of the subtree.
//
// Flutter: widgets/basic.dart Semantics.localeForSubtree
func (w *Semantics) LocaleForSubtree() foundation.Opt[ui.Locale] { return w.localeForSubtree }

// textDirection is the direction of the labels: the one of the properties, or,
// when the properties have text, the one of the closest Directionality.
func (w *Semantics) textDirection(bc BuildContext) ui.TextDirection {
	p := w.properties
	if p.TextDirection != ui.TextDirectionUnspecified {
		return p.TextDirection
	}
	containsText := p.AttributedLabel != nil || p.Label.IsSome() || p.Value.IsSome() || p.Hint.IsSome() || p.Tooltip.IsSome()
	if !containsText {
		return ui.TextDirectionUnspecified
	}
	return maybeTextDirection(bc)
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Semantics.createRenderObject
func (w *Semantics) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderSemanticsAnnotations(nil, w.properties, rendering.SemanticsAnnotationsOpts{
		Container: w.container, ExplicitChildNodes: w.explicitChildNodes, ExcludeSemantics: w.excludeSemantics,
		BlockUserActions: w.blockUserActions, LocaleForSubtree: w.localeForSubtree, TextDirection: w.textDirection(bc)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart Semantics.updateRenderObject
func (w *Semantics) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderSemanticsAnnotations)
	r.SetContainer(w.container)
	r.SetExplicitChildNodes(w.explicitChildNodes)
	r.SetExcludeSemantics(w.excludeSemantics)
	r.SetBlockUserActions(w.blockUserActions)
	r.SetLocaleForSubtree(w.localeForSubtree)
	r.SetProperties(w.properties)
	r.SetTextDirection(w.textDirection(bc))
}

// MergeSemantics merges the semantics of its descendants into one node.
//
// Flutter: widgets/basic.dart MergeSemantics
type MergeSemantics struct{ SingleChildRenderObjectWidget }

// NewMergeSemantics creates a widget that merges the semantics of its subtree.
//
// Flutter: widgets/basic.dart MergeSemantics.new
func NewMergeSemantics(child Widget, opts ...WidgetOpts) *MergeSemantics {
	return &MergeSemantics{NewSingleChildRenderObjectWidgetBase(child, firstOpts(opts))}
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart MergeSemantics.createRenderObject
func (w *MergeSemantics) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderMergeSemantics(nil)
}

// ExcludeSemanticsOpts are the optional parameters of NewExcludeSemantics.
//
// Flutter: widgets/basic.dart ExcludeSemantics.new
type ExcludeSemanticsOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Excluding is whether this widget excludes its subtree from the semantics
	// tree; the default is true.
	Excluding foundation.Opt[bool]
}

// ExcludeSemantics drops all the semantics of its descendants.
//
// Flutter: widgets/basic.dart ExcludeSemantics
type ExcludeSemantics struct {
	SingleChildRenderObjectWidget
	excluding bool
}

// NewExcludeSemantics creates a widget that drops the semantics of its subtree.
//
// Flutter: widgets/basic.dart ExcludeSemantics.new
func NewExcludeSemantics(child Widget, opts ...ExcludeSemanticsOpts) *ExcludeSemantics {
	o := firstOpts(opts)
	return &ExcludeSemantics{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), o.Excluding.OrElse(true)}
}

// Excluding is whether the subtree is excluded.
//
// Flutter: widgets/basic.dart ExcludeSemantics.excluding
func (w *ExcludeSemantics) Excluding() bool { return w.excluding }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart ExcludeSemantics.createRenderObject
func (w *ExcludeSemantics) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderExcludeSemantics(nil, rendering.RenderExcludeSemanticsOpts{Excluding: foundation.Some(w.excluding)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart ExcludeSemantics.updateRenderObject
func (w *ExcludeSemantics) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderExcludeSemantics).SetExcluding(w.excluding)
}

// BlockSemanticsOpts are the optional parameters of NewBlockSemantics.
//
// Flutter: widgets/basic.dart BlockSemantics.new
type BlockSemanticsOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Blocking is whether this widget blocks the semantics of widgets that were
	// painted before it in the same semantic container; the default is true.
	Blocking foundation.Opt[bool]
}

// BlockSemantics drops all the semantics of the widgets that were painted before
// it in the same semantic container.
//
// Flutter: widgets/basic.dart BlockSemantics
type BlockSemantics struct {
	SingleChildRenderObjectWidget
	blocking bool
}

// NewBlockSemantics creates a widget that blocks the semantics of the widgets
// painted before it.
//
// Flutter: widgets/basic.dart BlockSemantics.new
func NewBlockSemantics(child Widget, opts ...BlockSemanticsOpts) *BlockSemantics {
	o := firstOpts(opts)
	return &BlockSemantics{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), o.Blocking.OrElse(true)}
}

// Blocking is whether the widget blocks the earlier semantics.
//
// Flutter: widgets/basic.dart BlockSemantics.blocking
func (w *BlockSemantics) Blocking() bool { return w.blocking }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart BlockSemantics.createRenderObject
func (w *BlockSemantics) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderBlockSemantics(nil, rendering.RenderBlockSemanticsOpts{Blocking: foundation.Some(w.blocking)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart BlockSemantics.updateRenderObject
func (w *BlockSemantics) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderBlockSemantics).SetBlocking(w.blocking)
}

// IndexedSemantics annotates a child of a scrollable with its index, so that
// assistive technologies know its place in the list.
//
// Flutter: widgets/basic.dart IndexedSemantics
type IndexedSemantics struct {
	SingleChildRenderObjectWidget
	index int
}

// NewIndexedSemantics creates a widget that gives the child its index.
//
// Flutter: widgets/basic.dart IndexedSemantics.new
func NewIndexedSemantics(index int, child Widget, opts ...WidgetOpts) *IndexedSemantics {
	return &IndexedSemantics{NewSingleChildRenderObjectWidgetBase(child, firstOpts(opts)), index}
}

// Index is the index of the child.
//
// Flutter: widgets/basic.dart IndexedSemantics.index
func (w *IndexedSemantics) Index() int { return w.index }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart IndexedSemantics.createRenderObject
func (w *IndexedSemantics) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderIndexedSemantics(nil, w.index)
}

// UpdateRenderObject copies the index to the render object.
//
// Flutter: widgets/basic.dart IndexedSemantics.updateRenderObject
func (w *IndexedSemantics) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderIndexedSemantics).SetIndex(w.index)
}
