package semantics

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// SemanticsHintOverrides are the hints that replace the default ones of the tap
// and long press actions.
//
// Flutter: semantics/semantics.dart SemanticsHintOverrides
type SemanticsHintOverrides struct {
	// OnTapHint is the hint of the tap action; "" for none.
	OnTapHint string
	// OnLongPressHint is the hint of the long press action; "" for none.
	OnLongPressHint string
}

// IsNotEmpty is whether any hint is set.
//
// Flutter: semantics/semantics.dart SemanticsHintOverrides.isNotEmpty
func (h *SemanticsHintOverrides) IsNotEmpty() bool {
	return h.OnTapHint != "" || h.OnLongPressHint != ""
}

// Equal is Dart's ==.
//
// Flutter: semantics/semantics.dart SemanticsHintOverrides.==
func (h *SemanticsHintOverrides) Equal(o *SemanticsHintOverrides) bool {
	if h == nil || o == nil {
		return h == o
	}
	return *h == *o
}

// DebugFillProperties adds the hints.
//
// Flutter: semantics/semantics.dart SemanticsHintOverrides.debugFillProperties
func (h *SemanticsHintOverrides) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	opt := func(s string) foundation.Opt[string] {
		if s == "" {
			return foundation.None[string]()
		}
		return foundation.Some(s)
	}
	p.Add(foundation.NewStringProperty("onTapHint", opt(h.OnTapHint), foundation.StringPropertyOpts{DefaultValue: foundation.DefaultNull}))
	p.Add(foundation.NewStringProperty("onLongPressHint", opt(h.OnLongPressHint), foundation.StringPropertyOpts{DefaultValue: foundation.DefaultNull}))
}

// DebugDescribeChildren is empty.
func (h *SemanticsHintOverrides) DebugDescribeChildren() []foundation.DiagnosticsNode { return nil }

// SemanticsProperties is the semantics information that a Semantics widget
// gives to its render object. A field that is not set leaves the property
// alone.
//
// Flutter: semantics/semantics.dart SemanticsProperties
type SemanticsProperties struct {
	// Enabled, Checked, Mixed, Expanded, Toggled, Selected, Button, Link,
	// Header, TextField, Slider, KeyboardKey, ReadOnly, Focused, Focusable,
	// InMutuallyExclusiveGroup, Hidden, Obscured, Multiline, ScopesRoute,
	// NamesRoute, Image, LiveRegion and IsRequired set the flags of the node.
	Enabled, Checked, Mixed, Expanded, Selected, Toggled, Button, Link, Header, TextField, Slider, KeyboardKey, ReadOnly foundation.Opt[bool]
	// Focusable is deprecated in Flutter: use Focused.
	Focusable, Focused, InMutuallyExclusiveGroup, Hidden, Obscured, Multiline, ScopesRoute, NamesRoute, Image, LiveRegion foundation.Opt[bool]
	// IsRequired is whether a form field is required.
	IsRequired foundation.Opt[bool]
	// LinkURL is the URL of a link; Link must be true.
	LinkURL foundation.Opt[string]
	// HeadingLevel is 1 to 6.
	HeadingLevel foundation.Opt[int]
	// AccessibilityFocusBlockType is what the accessibility focus is blocked for.
	AccessibilityFocusBlockType foundation.Opt[AccessibilityFocusBlockType]
	// MaxValueLength and CurrentValueLength are the limits of a text field.
	MaxValueLength, CurrentValueLength foundation.Opt[int]
	// Identifier is the identifier for tests.
	Identifier foundation.Opt[string]
	// TraversalParentIdentifier and TraversalChildIdentifier tie nodes of
	// different branches into one traversal order.
	TraversalParentIdentifier, TraversalChildIdentifier any
	// Label, Value, IncreasedValue, DecreasedValue and Hint are the strings;
	// the Attributed ones have attributes and replace them. Setting both a
	// string and its attributed version is an error.
	Label, Value, IncreasedValue, DecreasedValue, Hint                                                   foundation.Opt[string]
	AttributedLabel, AttributedValue, AttributedIncreasedValue, AttributedDecreasedValue, AttributedHint *AttributedString
	// Tooltip is the tooltip.
	Tooltip foundation.Opt[string]
	// HintOverrides replace the default hints of the tap and long press.
	HintOverrides *SemanticsHintOverrides
	// TextDirection is the direction of the strings.
	TextDirection ui.TextDirection
	// SortKey orders the node among its siblings for traversal.
	SortKey SemanticsSortKey
	// TagForChildren is the tag that the node gives to its children.
	TagForChildren foundation.Opt[SemanticsTag]
	// Role is the role.
	Role foundation.Opt[ui.SemanticsRole]
	// ControlsNodes are the identifiers of the nodes that this one controls.
	ControlsNodes []string
	// InputType is the input type of a text field.
	InputType foundation.Opt[ui.SemanticsInputType]
	// ValidationResult is the validation result.
	ValidationResult ui.SemanticsValidationResult
	// HitTestBehavior is the hit test behavior.
	HitTestBehavior foundation.Opt[ui.SemanticsHitTestBehavior]
	// The handlers of the actions; nil leaves the action out.
	OnTap, OnLongPress, OnScrollLeft, OnScrollRight, OnScrollUp, OnScrollDown, OnIncrease, OnDecrease, OnCopy, OnCut, OnPaste VoidCallback
	// OnMoveCursorForwardByCharacter and the three others move the cursor.
	OnMoveCursorForwardByCharacter, OnMoveCursorBackwardByCharacter, OnMoveCursorForwardByWord, OnMoveCursorBackwardByWord MoveCursorHandler
	// OnSetSelection sets the selection of a text field.
	OnSetSelection SetSelectionHandler
	// OnSetText replaces the text of a text field.
	OnSetText SetTextHandler
	// OnDidGainAccessibilityFocus, OnDidLoseAccessibilityFocus, OnFocus,
	// OnDismiss, OnExpand and OnCollapse are the handlers of the other actions.
	OnDidGainAccessibilityFocus, OnDidLoseAccessibilityFocus, OnFocus, OnDismiss, OnExpand, OnCollapse VoidCallback
	// CustomSemanticsActions are the custom actions with their handlers.
	CustomSemanticsActions map[CustomSemanticsAction]VoidCallback
	// MinValue and MaxValue are the limits of the value, as text.
	MinValue, MaxValue foundation.Opt[string]
}

// NewSemanticsProperties checks the properties and returns them. In debug
// builds it panics with the tag SemanticsPropertiesInvalid if both a string and
// its attributed version are set, the heading level is not 1 to 6, or a link
// URL is set without Link.
//
// Flutter: semantics/semantics.dart SemanticsProperties.new
func NewSemanticsProperties(p SemanticsProperties) *SemanticsProperties {
	if foundation.DebugMode {
		fail := func(msg string) { panic(errors.New(tagSemanticsPropertiesInvalid, msg)) }
		both := func(name string, s foundation.Opt[string], a *AttributedString) {
			if s.IsSome() && a != nil {
				fail("Only one of " + name + " or attributed" + name + " should be provided")
			}
		}
		both("label", p.Label, p.AttributedLabel)
		both("value", p.Value, p.AttributedValue)
		both("increasedValue", p.IncreasedValue, p.AttributedIncreasedValue)
		both("decreasedValue", p.DecreasedValue, p.AttributedDecreasedValue)
		both("hint", p.Hint, p.AttributedHint)
		if l, ok := p.HeadingLevel.Get(); ok && (l <= 0 || l > 6) {
			fail("Heading level must be between 1 and 6")
		}
		if p.LinkURL.IsSome() && !p.Link.OrElse(false) {
			fail("If linkUrl is set then link must be true")
		}
	}
	return &p
}

// ToStringShort is "SemanticsProperties".
func (p *SemanticsProperties) ToStringShort() string { return "SemanticsProperties" }

// DebugFillProperties adds the properties that are set.
//
// Flutter: semantics/semantics.dart SemanticsProperties.debugFillProperties
func (p *SemanticsProperties) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	null := foundation.DefaultNull
	boolProp := func(name string, v foundation.Opt[bool]) {
		b.Add(foundation.NewDiagnosticsPropertyOpt(name, v, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	}
	str := func(name string, v foundation.Opt[string]) {
		b.Add(foundation.NewStringProperty(name, v, foundation.StringPropertyOpts{DefaultValue: null}))
	}
	attr := func(name string, v *AttributedString) {
		b.Add(NewAttributedStringProperty(name, v, false, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	}
	boolProp("checked", p.Checked)
	boolProp("mixed", p.Mixed)
	boolProp("expanded", p.Expanded)
	boolProp("selected", p.Selected)
	boolProp("isRequired", p.IsRequired)
	str("identifier", p.Identifier)
	b.Add(foundation.NewDiagnosticsProperty("traversalParentIdentifier", p.TraversalParentIdentifier, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewDiagnosticsProperty("traversalChildIdentifier", p.TraversalChildIdentifier, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	str("label", p.Label)
	attr("attributedLabel", p.AttributedLabel)
	str("value", p.Value)
	attr("attributedValue", p.AttributedValue)
	// Flutter reports the value under increasedValue and decreasedValue.
	str("increasedValue", p.Value)
	attr("attributedIncreasedValue", p.AttributedIncreasedValue)
	str("decreasedValue", p.Value)
	attr("attributedDecreasedValue", p.AttributedDecreasedValue)
	str("hint", p.Hint)
	attr("attributedHint", p.AttributedHint)
	str("tooltip", p.Tooltip)
	b.Add(foundation.NewEnumProperty("textDirection", optDirection(p.TextDirection), foundation.EnumPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewEnumProperty("role", p.Role, foundation.EnumPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewEnumProperty("validationResult", foundation.Some(p.ValidationResult),
		foundation.EnumPropertyOpts{DefaultValue: ui.SemanticsValidationResultNone}))
	b.Add(foundation.NewDiagnosticsProperty("sortKey", p.SortKey, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewDiagnosticsProperty("hintOverrides", p.HintOverrides, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
}

// DebugDescribeChildren is empty.
func (p *SemanticsProperties) DebugDescribeChildren() []foundation.DiagnosticsNode { return nil }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSemanticsPropertiesInvalid = "SemanticsPropertiesInvalid"
)
