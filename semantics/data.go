package semantics

import (
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// SemanticsData holds the semantics information of a node, with what the nodes
// merged into it add.
//
// Flutter: semantics/semantics.dart SemanticsData
type SemanticsData struct {
	// FlagsCollection are the flags of the node.
	FlagsCollection ui.SemanticsFlags
	// Actions are the supported actions as a bit field.
	Actions ui.SemanticsAction
	// Identifier is the identifier for tests.
	Identifier string
	// TraversalParentIdentifier and TraversalChildIdentifier tie nodes of
	// different branches into one traversal order; nil if none.
	TraversalParentIdentifier, TraversalChildIdentifier any
	// AttributedLabel, AttributedValue, AttributedIncreasedValue,
	// AttributedDecreasedValue and AttributedHint are the strings of the node.
	AttributedLabel, AttributedValue, AttributedIncreasedValue, AttributedDecreasedValue, AttributedHint AttributedString
	// Tooltip is the tooltip.
	Tooltip string
	// HeadingLevel is 0 to 6; 0 is not a heading.
	HeadingLevel int
	// TextDirection is the direction of the strings; unspecified if unknown.
	TextDirection ui.TextDirection
	// TextSelection is the selection of a text field.
	TextSelection foundation.Opt[services.TextSelection]
	// ScrollChildCount and ScrollIndex describe a scrollable.
	ScrollChildCount, ScrollIndex foundation.Opt[int]
	// ScrollPosition, ScrollExtentMax and ScrollExtentMin describe a scrollable.
	ScrollPosition, ScrollExtentMax, ScrollExtentMin foundation.Opt[float64]
	// PlatformViewID is the platform view that the node stands for.
	PlatformViewID foundation.Opt[int]
	// MaxValueLength and CurrentValueLength are the limits of a text field.
	MaxValueLength, CurrentValueLength foundation.Opt[int]
	// LinkURL is the URL of a link.
	LinkURL foundation.Opt[string]
	// Rect is the rectangle of the node in its own coordinates.
	Rect ui.Rect
	// Tags are the tags of the node; nil if none.
	Tags map[SemanticsTag]struct{}
	// Transform maps the node to its parent; nil is the identity.
	Transform *ui.Matrix4
	// CustomSemanticsActionIDs are the IDs of the custom actions, in order; nil
	// if none.
	CustomSemanticsActionIDs []int
	// Role is the role.
	Role ui.SemanticsRole
	// ControlsNodes are the identifiers of the nodes that this one controls;
	// nil if none.
	ControlsNodes []string
	// ValidationResult is the validation result.
	ValidationResult ui.SemanticsValidationResult
	// HitTestBehavior is the hit test behavior.
	HitTestBehavior ui.SemanticsHitTestBehavior
	// InputType is the input type.
	InputType ui.SemanticsInputType
	// Locale is the locale of the strings.
	Locale foundation.Opt[ui.Locale]
	// MinValue and MaxValue are the limits of the value.
	MinValue, MaxValue foundation.Opt[string]
}

// NewSemanticsData checks the data and returns it. In debug builds it panics
// with the tag SemanticsDataInvalid if a string has no text direction, the
// heading level is not 0 to 6, or a link URL is set on a node that is not a
// link.
//
// Flutter: semantics/semantics.dart SemanticsData.new
func NewSemanticsData(d SemanticsData) *SemanticsData {
	if foundation.DebugMode {
		fail := func(msg string, kv ...any) { panic(errors.New("SemanticsDataInvalid", msg, kv...)) }
		for _, s := range []struct {
			name, text string
		}{{"tooltip", d.Tooltip}, {"label", d.AttributedLabel.String}, {"value", d.AttributedValue.String},
			{"decreasedValue", d.AttributedDecreasedValue.String}, {"increasedValue", d.AttributedIncreasedValue.String},
			{"hint", d.AttributedHint.String}} {
			if s.text != "" && d.TextDirection == ui.TextDirectionUnspecified {
				fail("A SemanticsData object with a "+s.name+" had no textDirection.", "text", s.text)
			}
		}
		if d.HeadingLevel < 0 || d.HeadingLevel > 6 {
			fail("Heading level must be between 0 and 6", "level", d.HeadingLevel)
		}
		if d.LinkURL.IsSome() && !d.FlagsCollection.IsLink {
			fail("A SemanticsData object with a linkUrl must have the isLink flag set to true")
		}
	}
	return &d
}

// HasAction is whether the action is supported.
//
// Flutter: semantics/semantics.dart SemanticsData.hasAction
func (d *SemanticsData) HasAction(action ui.SemanticsAction) bool { return d.Actions&action != 0 }

// Label is the text of the label.
//
// Flutter: semantics/semantics.dart SemanticsData.label
func (d *SemanticsData) Label() string { return d.AttributedLabel.String }

// Value is the text of the value.
//
// Flutter: semantics/semantics.dart SemanticsData.value
func (d *SemanticsData) Value() string { return d.AttributedValue.String }

// IncreasedValue is the text of the increased value.
//
// Flutter: semantics/semantics.dart SemanticsData.increasedValue
func (d *SemanticsData) IncreasedValue() string { return d.AttributedIncreasedValue.String }

// DecreasedValue is the text of the decreased value.
//
// Flutter: semantics/semantics.dart SemanticsData.decreasedValue
func (d *SemanticsData) DecreasedValue() string { return d.AttributedDecreasedValue.String }

// Hint is the text of the hint.
//
// Flutter: semantics/semantics.dart SemanticsData.hint
func (d *SemanticsData) Hint() string { return d.AttributedHint.String }

// ToStringShort is "SemanticsData".
//
// Flutter: semantics/semantics.dart SemanticsData.toStringShort
func (d *SemanticsData) ToStringShort() string { return "SemanticsData" }

// String describes the data on one line.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d *SemanticsData) String() string { return foundation.DiagnosticableString(d) }

func setTags(m map[SemanticsTag]struct{}) []string {
	out := make([]string, 0, len(m))
	for t := range m {
		out = append(out, t.Name())
	}
	slices.Sort(out)
	return out
}

// DebugFillProperties adds the rectangle, the transform, the actions, the
// flags, the strings and the rest.
//
// Flutter: semantics/semantics.dart SemanticsData.debugFillProperties
func (d *SemanticsData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	null := foundation.DefaultNull
	p.Add(foundation.NewDiagnosticsProperty("rect", d.Rect, foundation.DiagnosticsPropertyOpts{ShowName: foundation.Some(false)}))
	p.Add(painting.NewTransformProperty("transform", d.Transform, foundation.DiagnosticsPropertyOpts{ShowName: foundation.Some(false), DefaultValue: null}))
	actionSummary := []string{}
	for _, a := range ui.SemanticsActionValues() {
		if d.Actions&a != 0 {
			actionSummary = append(actionSummary, a.Name())
		}
	}
	custom := []string{}
	for _, id := range d.CustomSemanticsActionIDs {
		if a, ok := CustomSemanticsActionGetAction(id); ok {
			custom = append(custom, a.Label())
		}
	}
	p.Add(foundation.NewIterableProperty("actions", actionSummary, foundation.IterablePropertyOpts{NoIfEmpty: true}))
	p.Add(foundation.NewIterableProperty("customActions", custom, foundation.IterablePropertyOpts{NoIfEmpty: true}))
	p.Add(foundation.NewIterableProperty("flags", append([]string{}, d.FlagsCollection.ToStrings()...), foundation.IterablePropertyOpts{NoIfEmpty: true}))
	p.Add(foundation.NewStringProperty("identifier", foundation.Some(d.Identifier), foundation.StringPropertyOpts{DefaultValue: ""}))
	p.Add(foundation.NewDiagnosticsProperty("traversalParentIdentifier", d.TraversalParentIdentifier, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	p.Add(foundation.NewDiagnosticsProperty("traversalChildIdentifier", d.TraversalChildIdentifier, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	p.Add(NewAttributedStringProperty("label", &d.AttributedLabel, false))
	p.Add(NewAttributedStringProperty("value", &d.AttributedValue, false))
	p.Add(NewAttributedStringProperty("increasedValue", &d.AttributedIncreasedValue, false))
	p.Add(NewAttributedStringProperty("decreasedValue", &d.AttributedDecreasedValue, false))
	p.Add(NewAttributedStringProperty("hint", &d.AttributedHint, false))
	p.Add(foundation.NewStringProperty("tooltip", foundation.Some(d.Tooltip), foundation.StringPropertyOpts{DefaultValue: ""}))
	p.Add(foundation.NewEnumProperty("textDirection", optDirection(d.TextDirection), foundation.EnumPropertyOpts{DefaultValue: null}))
	if sel, ok := d.TextSelection.Get(); ok && sel.IsValid() {
		p.Add(foundation.NewMessageProperty("textSelection", "["+itoa(sel.Start())+", "+itoa(sel.End())+"]"))
	}
	nullNum := foundation.NumberPropertyOpts{DefaultValue: null}
	p.Add(foundation.NewIntProperty("platformViewId", d.PlatformViewID, nullNum))
	p.Add(foundation.NewIntProperty("maxValueLength", d.MaxValueLength, nullNum))
	p.Add(foundation.NewIntProperty("currentValueLength", d.CurrentValueLength, nullNum))
	p.Add(foundation.NewIntProperty("scrollChildren", d.ScrollChildCount, nullNum))
	p.Add(foundation.NewIntProperty("scrollIndex", d.ScrollIndex, nullNum))
	p.Add(foundation.NewDoubleProperty("scrollExtentMin", d.ScrollExtentMin, nullNum))
	p.Add(foundation.NewDoubleProperty("scrollPosition", d.ScrollPosition, nullNum))
	p.Add(foundation.NewDoubleProperty("scrollExtentMax", d.ScrollExtentMax, nullNum))
	p.Add(foundation.NewIntProperty("headingLevel", foundation.Some(d.HeadingLevel), foundation.NumberPropertyOpts{DefaultValue: 0}))
	p.Add(foundation.NewDiagnosticsPropertyOpt("linkUrl", d.LinkURL, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	if d.ControlsNodes != nil {
		p.Add(foundation.NewIterableProperty("controls", d.ControlsNodes, foundation.IterablePropertyOpts{NoIfEmpty: true}))
	}
	if d.Role != ui.SemanticsRoleNone {
		p.Add(foundation.NewEnumProperty("role", foundation.Some(d.Role), foundation.EnumPropertyOpts{DefaultValue: ui.SemanticsRoleNone}))
	}
	if d.ValidationResult != ui.SemanticsValidationResultNone {
		p.Add(foundation.NewEnumProperty("validationResult", foundation.Some(d.ValidationResult),
			foundation.EnumPropertyOpts{DefaultValue: ui.SemanticsValidationResultNone}))
	}
	p.Add(foundation.NewStringProperty("minValue", d.MinValue, foundation.StringPropertyOpts{DefaultValue: null}))
	p.Add(foundation.NewStringProperty("maxValue", d.MaxValue, foundation.StringPropertyOpts{DefaultValue: null}))
}

// DebugDescribeChildren is empty.
func (d *SemanticsData) DebugDescribeChildren() []foundation.DiagnosticsNode { return nil }

func optDirection(d ui.TextDirection) foundation.Opt[ui.TextDirection] {
	if d == ui.TextDirectionUnspecified {
		return foundation.None[ui.TextDirection]()
	}
	return foundation.Some(d)
}

func setEqualStrings(a, b []string) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	as, bs := slices.Clone(a), slices.Clone(b)
	slices.Sort(as)
	slices.Sort(bs)
	return slices.Equal(as, bs)
}

func setEqualTags(a, b map[SemanticsTag]struct{}) bool {
	if (a == nil) != (b == nil) || len(a) != len(b) {
		return false
	}
	for t := range a {
		if _, ok := b[t]; !ok {
			return false
		}
	}
	return true
}

func matrixEqual(a, b *ui.Matrix4) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equals(b)
}

// Equal is Dart's ==: all the fields are equal.
//
// Flutter: semantics/semantics.dart SemanticsData.==
func (d *SemanticsData) Equal(o *SemanticsData) bool {
	return d.FlagsCollection == o.FlagsCollection && d.Actions == o.Actions && d.Identifier == o.Identifier &&
		d.TraversalParentIdentifier == o.TraversalParentIdentifier && d.TraversalChildIdentifier == o.TraversalChildIdentifier &&
		d.AttributedLabel.Equal(o.AttributedLabel) && d.AttributedValue.Equal(o.AttributedValue) &&
		d.AttributedIncreasedValue.Equal(o.AttributedIncreasedValue) && d.AttributedDecreasedValue.Equal(o.AttributedDecreasedValue) &&
		d.AttributedHint.Equal(o.AttributedHint) && d.Tooltip == o.Tooltip && d.TextDirection == o.TextDirection &&
		d.Rect == o.Rect && setEqualTags(d.Tags, o.Tags) && optEqual(d.ScrollChildCount, o.ScrollChildCount) &&
		optEqual(d.ScrollIndex, o.ScrollIndex) && optEqual(d.TextSelection, o.TextSelection) &&
		optEqual(d.ScrollPosition, o.ScrollPosition) && optEqual(d.ScrollExtentMax, o.ScrollExtentMax) &&
		optEqual(d.ScrollExtentMin, o.ScrollExtentMin) && optEqual(d.PlatformViewID, o.PlatformViewID) &&
		optEqual(d.MaxValueLength, o.MaxValueLength) && optEqual(d.CurrentValueLength, o.CurrentValueLength) &&
		matrixEqual(d.Transform, o.Transform) && d.HeadingLevel == o.HeadingLevel && optEqual(d.LinkURL, o.LinkURL) &&
		d.Role == o.Role && d.ValidationResult == o.ValidationResult && d.InputType == o.InputType &&
		d.HitTestBehavior == o.HitTestBehavior && slices.Equal(d.CustomSemanticsActionIDs, o.CustomSemanticsActionIDs) &&
		setEqualStrings(d.ControlsNodes, o.ControlsNodes) && optEqual(d.MinValue, o.MinValue) && optEqual(d.MaxValue, o.MaxValue)
}
