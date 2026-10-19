package rendering

import (
	"strconv"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// SemanticsBuilderCallback builds the semantics of a custom painter for a
// canvas of the given size.
//
// Flutter: rendering/custom_paint.dart SemanticsBuilderCallback
type SemanticsBuilderCallback func(size ui.Size) []CustomPainterSemantics

// SemanticsPainter is implemented by a CustomPainter that describes its picture
// to the accessibility tools.
//
// Deviation: Flutter's CustomPainter.semanticsBuilder is a getter that returns
// null by default; here it is an optional interface, which the render object
// looks for (SPEC §12.10).
//
// Flutter: rendering/custom_paint.dart CustomPainter.semanticsBuilder
type SemanticsPainter interface {
	// SemanticsBuilder returns the function that builds the semantics, or nil.
	//
	// Flutter: rendering/custom_paint.dart CustomPainter.semanticsBuilder
	SemanticsBuilder() SemanticsBuilderCallback
}

// SemanticsRebuildPainter is implemented by a CustomPainter that tells whether a
// new painter needs new semantics. Without it, ShouldRepaint decides.
//
// Flutter: rendering/custom_paint.dart CustomPainter.shouldRebuildSemantics
type SemanticsRebuildPainter interface {
	// ShouldRebuildSemantics says whether the semantics must be built again
	// because of the new painter, of the same type as the old one.
	//
	// Flutter: rendering/custom_paint.dart CustomPainter.shouldRebuildSemantics
	ShouldRebuildSemantics(oldDelegate CustomPainter) bool
}

func painterSemanticsBuilder(p CustomPainter) SemanticsBuilderCallback {
	if sp, ok := p.(SemanticsPainter); ok {
		return sp.SemanticsBuilder()
	}
	return nil
}

func shouldRebuildPainterSemantics(newPainter, oldPainter CustomPainter) bool {
	if rp, ok := newPainter.(SemanticsRebuildPainter); ok {
		return rp.ShouldRebuildSemantics(oldPainter)
	}
	return newPainter.ShouldRepaint(oldPainter)
}

// CustomPainterSemantics is the semantics information about a rectangle on a
// canvas, which a custom painter gives to the accessibility tools.
//
// Flutter: rendering/custom_paint.dart CustomPainterSemantics
type CustomPainterSemantics struct {
	// Key identifies this object among its siblings: the SemanticsNode with the
	// same key is updated from it. With a nil key the update does not guarantee
	// which node is updated.
	//
	// Flutter: rendering/custom_paint.dart CustomPainterSemantics.key
	Key foundation.Key
	// Rect is the location and size of the box on the canvas where this
	// information applies.
	//
	// Flutter: rendering/custom_paint.dart CustomPainterSemantics.rect
	Rect ui.Rect
	// Transform is the transform from the coordinate system of the canvas to
	// that of its parent; nil for none.
	//
	// Flutter: rendering/custom_paint.dart CustomPainterSemantics.transform
	Transform *ui.Matrix4
	// Properties are assigned to the node that is created or updated.
	//
	// Flutter: rendering/custom_paint.dart CustomPainterSemantics.properties
	Properties *semantics.SemanticsProperties
	// Tags are the tags of the node.
	//
	// Flutter: rendering/custom_paint.dart CustomPainterSemantics.tags
	Tags map[semantics.SemanticsTag]struct{}
}

// DescribeSemanticsConfiguration makes the object a semantic boundary if a
// painter has a semantics builder.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.describeSemanticsConfiguration
func (r *RenderCustomPaint) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBox.DescribeSemanticsConfiguration(config)
	r.backgroundSemanticsBuilder = nil
	r.foregroundSemanticsBuilder = nil
	if r.painter != nil {
		r.backgroundSemanticsBuilder = painterSemanticsBuilder(r.painter)
	}
	if r.foregroundPainter != nil {
		r.foregroundSemanticsBuilder = painterSemanticsBuilder(r.foregroundPainter)
	}
	config.SetIsSemanticBoundary(r.backgroundSemanticsBuilder != nil || r.foregroundSemanticsBuilder != nil)
}

// AssembleSemanticsNode gives the node the children of the painters around
// those of the child. In debug builds it panics with the tag
// CustomPaintSemanticsNoChild if the object has no child but gets children.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.assembleSemanticsNode
func (r *RenderCustomPaint) AssembleSemanticsNode(node *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, children []*semantics.SemanticsNode) {
	if foundation.DebugMode && !r.hasChild() && len(children) > 0 {
		panic(errors.New("CustomPaintSemanticsNoChild", foundation.RuntimeTypeName(r.self)+
			" does not have a child widget but received a non-empty list of child SemanticsNode",
			"children", strconv.Itoa(len(children))))
	}
	var background, foreground []CustomPainterSemantics
	if r.backgroundSemanticsBuilder != nil {
		background = r.backgroundSemanticsBuilder(r.Size())
	}
	r.backgroundSemanticsNodes = updateSemanticsChildren(r.backgroundSemanticsNodes, background)
	if r.foregroundSemanticsBuilder != nil {
		foreground = r.foregroundSemanticsBuilder(r.Size())
	}
	r.foregroundSemanticsNodes = updateSemanticsChildren(r.foregroundSemanticsNodes, foreground)

	final := make([]*semantics.SemanticsNode, 0, len(children)+len(r.backgroundSemanticsNodes)+len(r.foregroundSemanticsNodes))
	final = append(final, r.backgroundSemanticsNodes...)
	final = append(final, children...)
	final = append(final, r.foregroundSemanticsNodes...)
	r.RenderProxyBox.AssembleSemanticsNode(node, config, final)
}

// ClearSemantics forgets the nodes of the painters too.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.clearSemantics
func (r *RenderCustomPaint) ClearSemantics() {
	r.RenderProxyBox.ClearSemantics()
	r.backgroundSemanticsNodes = nil
	r.foregroundSemanticsNodes = nil
}

// updateSemanticsChildren updates the old nodes from the new semantics. Nodes
// that match by key are kept; the others are updated in the order of the list
// (the algorithm of RenderObjectElement.updateChildren). In debug builds it
// panics with the tag CustomPainterSemanticsKeyDuplicated for a duplicate key.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint._updateSemanticsChildren
func updateSemanticsChildren(oldSemantics []*semantics.SemanticsNode, newChildSemantics []CustomPainterSemantics) []*semantics.SemanticsNode {
	if foundation.DebugMode {
		keys := map[foundation.Key]int{}
		for i, child := range newChildSemantics {
			if child.Key == nil {
				continue
			}
			if _, dup := keys[child.Key]; dup {
				panic(errors.New("CustomPainterSemanticsKeyDuplicated", "Failed to update the list of CustomPainterSemantics: duplicate key found",
					"key", child.Key, "position", i))
			}
			keys[child.Key] = i
		}
	}
	newChildrenTop, oldChildrenTop := 0, 0
	newChildrenBottom, oldChildrenBottom := len(newChildSemantics)-1, len(oldSemantics)-1
	newChildren := make([]*semantics.SemanticsNode, len(newChildSemantics))

	for oldChildrenTop <= oldChildrenBottom && newChildrenTop <= newChildrenBottom {
		oldChild, newSemantics := oldSemantics[oldChildrenTop], newChildSemantics[newChildrenTop]
		if !canUpdateSemanticsChild(oldChild, newSemantics) {
			break
		}
		newChildren[newChildrenTop] = updateSemanticsChild(oldChild, newSemantics)
		newChildrenTop++
		oldChildrenTop++
	}
	for oldChildrenTop <= oldChildrenBottom && newChildrenTop <= newChildrenBottom {
		oldChild, newChild := oldSemantics[oldChildrenBottom], newChildSemantics[newChildrenBottom]
		if !canUpdateSemanticsChild(oldChild, newChild) {
			break
		}
		oldChildrenBottom--
		newChildrenBottom--
	}
	haveOldChildren := oldChildrenTop <= oldChildrenBottom
	oldKeyedChildren := map[foundation.Key]*semantics.SemanticsNode{}
	if haveOldChildren {
		for oldChildrenTop <= oldChildrenBottom {
			if oldChild := oldSemantics[oldChildrenTop]; oldChild.Key != nil {
				oldKeyedChildren[oldChild.Key] = oldChild
			}
			oldChildrenTop++
		}
	}
	for newChildrenTop <= newChildrenBottom {
		var oldChild *semantics.SemanticsNode
		newSemantics := newChildSemantics[newChildrenTop]
		if haveOldChildren && newSemantics.Key != nil {
			if oc := oldKeyedChildren[newSemantics.Key]; oc != nil {
				if canUpdateSemanticsChild(oc, newSemantics) {
					// A match: remove it so it is not unsynced later.
					delete(oldKeyedChildren, newSemantics.Key)
					oldChild = oc
				}
			}
		}
		newChildren[newChildrenTop] = updateSemanticsChild(oldChild, newSemantics)
		newChildrenTop++
	}
	newChildrenBottom = len(newChildSemantics) - 1
	oldChildrenBottom = len(oldSemantics) - 1
	for oldChildrenTop <= oldChildrenBottom && newChildrenTop <= newChildrenBottom {
		oldChild, newSemantics := oldSemantics[oldChildrenTop], newChildSemantics[newChildrenTop]
		newChildren[newChildrenTop] = updateSemanticsChild(oldChild, newSemantics)
		newChildrenTop++
		oldChildrenTop++
	}
	return newChildren
}

func canUpdateSemanticsChild(oldChild *semantics.SemanticsNode, newSemantics CustomPainterSemantics) bool {
	return oldChild.Key == newSemantics.Key
}

func updateSemanticsChild(oldChild *semantics.SemanticsNode, newSemantics CustomPainterSemantics) *semantics.SemanticsNode {
	newChild := oldChild
	if newChild == nil {
		newChild = semantics.NewSemanticsNode(semantics.SemanticsNodeOpts{Key: newSemantics.Key})
	}
	p := newSemantics.Properties
	config := semantics.NewSemanticsConfiguration()
	setBool := func(v foundation.Opt[bool], set func(bool)) {
		if b, ok := v.Get(); ok {
			set(b)
		}
	}
	setOpt := func(v foundation.Opt[bool], set func(foundation.Opt[bool])) {
		if v.IsSome() {
			set(v)
		}
	}
	if r, ok := p.Role.Get(); ok {
		config.SetRole(r)
	}
	if p.SortKey != nil {
		config.SetSortKey(p.SortKey)
	}
	setOpt(p.Checked, config.SetIsChecked)
	setOpt(p.Mixed, config.SetIsCheckStateMixed)
	setBool(p.Selected, config.SetIsSelected)
	setBool(p.Button, config.SetIsButton)
	setOpt(p.Expanded, config.SetIsExpanded)
	setBool(p.Link, config.SetIsLink)
	if p.LinkURL.IsSome() {
		config.SetLinkURL(p.LinkURL)
	}
	setBool(p.TextField, config.SetIsTextField)
	setBool(p.Slider, config.SetIsSlider)
	setBool(p.KeyboardKey, config.SetIsKeyboardKey)
	setBool(p.ReadOnly, config.SetIsReadOnly)
	setBool(p.Focusable, config.SetIsFocusable)
	setOpt(p.Focused, config.SetIsFocused)
	if t, ok := p.AccessibilityFocusBlockType.Get(); ok {
		config.SetAccessibilityFocusBlockType(t)
	}
	setOpt(p.Enabled, config.SetIsEnabled)
	setBool(p.InMutuallyExclusiveGroup, config.SetIsInMutuallyExclusiveGroup)
	setBool(p.Obscured, config.SetIsObscured)
	setBool(p.Multiline, config.SetIsMultiline)
	setBool(p.Hidden, config.SetIsHidden)
	setBool(p.Header, config.SetIsHeader)
	if l, ok := p.HeadingLevel.Get(); ok {
		config.SetHeadingLevel(l)
	}
	setBool(p.ScopesRoute, config.SetScopesRoute)
	setBool(p.NamesRoute, config.SetNamesRoute)
	setBool(p.LiveRegion, config.SetLiveRegion)
	setOpt(p.IsRequired, config.SetIsRequired)
	if p.MaxValueLength.IsSome() {
		config.SetMaxValueLength(p.MaxValueLength)
	}
	if p.CurrentValueLength.IsSome() {
		config.SetCurrentValueLength(p.CurrentValueLength)
	}
	setOpt(p.Toggled, config.SetIsToggled)
	setBool(p.Image, config.SetIsImage)
	if s, ok := p.Label.Get(); ok {
		config.SetLabel(s)
	}
	if s, ok := p.Value.Get(); ok {
		config.SetValue(s)
	}
	if s, ok := p.IncreasedValue.Get(); ok {
		config.SetIncreasedValue(s)
	}
	if s, ok := p.DecreasedValue.Get(); ok {
		config.SetDecreasedValue(s)
	}
	if s, ok := p.Hint.Get(); ok {
		config.SetHint(s)
	}
	if s, ok := p.Identifier.Get(); ok {
		config.SetIdentifier(s)
	}
	if p.TraversalParentIdentifier != nil {
		config.SetTraversalParentIdentifier(p.TraversalParentIdentifier)
	}
	if p.TraversalChildIdentifier != nil {
		config.SetTraversalChildIdentifier(p.TraversalChildIdentifier)
	}
	if s, ok := p.Tooltip.Get(); ok {
		config.SetTooltip(s)
	}
	if p.HintOverrides != nil {
		config.SetHintOverrides(p.HintOverrides)
	}
	if tag, ok := p.TagForChildren.Get(); ok {
		config.AddTagForChildren(tag)
	}
	if p.ControlsNodes != nil {
		config.SetControlsNodes(p.ControlsNodes)
	}
	if p.TextDirection != ui.TextDirectionUnspecified {
		config.SetTextDirection(p.TextDirection)
	}
	if config.ValidationResult() != p.ValidationResult {
		config.SetValidationResult(p.ValidationResult)
	}
	if b, ok := p.HitTestBehavior.Get(); ok {
		config.SetHitTestBehavior(b)
	}
	if t, ok := p.InputType.Get(); ok {
		config.SetInputType(t)
	}
	if p.MinValue.IsSome() {
		config.SetMinValue(p.MinValue)
	}
	if p.MaxValue.IsSome() {
		config.SetMaxValue(p.MaxValue)
	}
	if p.OnTap != nil {
		config.SetOnTap(p.OnTap)
	}
	if p.OnLongPress != nil {
		config.SetOnLongPress(p.OnLongPress)
	}
	if p.OnScrollLeft != nil {
		config.SetOnScrollLeft(p.OnScrollLeft)
	}
	if p.OnScrollRight != nil {
		config.SetOnScrollRight(p.OnScrollRight)
	}
	if p.OnScrollUp != nil {
		config.SetOnScrollUp(p.OnScrollUp)
	}
	if p.OnScrollDown != nil {
		config.SetOnScrollDown(p.OnScrollDown)
	}
	if p.OnIncrease != nil {
		config.SetOnIncrease(p.OnIncrease)
	}
	if p.OnDecrease != nil {
		config.SetOnDecrease(p.OnDecrease)
	}
	if p.OnCopy != nil {
		config.SetOnCopy(p.OnCopy)
	}
	if p.OnCut != nil {
		config.SetOnCut(p.OnCut)
	}
	if p.OnPaste != nil {
		config.SetOnPaste(p.OnPaste)
	}
	if p.OnMoveCursorForwardByCharacter != nil {
		config.SetOnMoveCursorForwardByCharacter(p.OnMoveCursorForwardByCharacter)
	}
	if p.OnMoveCursorBackwardByCharacter != nil {
		config.SetOnMoveCursorBackwardByCharacter(p.OnMoveCursorBackwardByCharacter)
	}
	if p.OnMoveCursorForwardByWord != nil {
		config.SetOnMoveCursorForwardByWord(p.OnMoveCursorForwardByWord)
	}
	if p.OnMoveCursorBackwardByWord != nil {
		config.SetOnMoveCursorBackwardByWord(p.OnMoveCursorBackwardByWord)
	}
	if p.OnSetSelection != nil {
		config.SetOnSetSelection(p.OnSetSelection)
	}
	if p.OnSetText != nil {
		config.SetOnSetText(p.OnSetText)
	}
	if p.OnDidGainAccessibilityFocus != nil {
		config.SetOnDidGainAccessibilityFocus(p.OnDidGainAccessibilityFocus)
	}
	if p.OnDidLoseAccessibilityFocus != nil {
		config.SetOnDidLoseAccessibilityFocus(p.OnDidLoseAccessibilityFocus)
	}
	if p.OnFocus != nil {
		config.SetOnFocus(p.OnFocus)
	}
	if p.OnDismiss != nil {
		config.SetOnDismiss(p.OnDismiss)
	}
	if p.OnExpand != nil {
		config.SetOnExpand(p.OnExpand)
	}
	if p.OnCollapse != nil {
		config.SetOnCollapse(p.OnCollapse)
	}

	// As of now a CustomPainter does not support multiple tree levels.
	newChild.UpdateWith(config, nil)
	newChild.SetRect(newSemantics.Rect)
	newChild.SetTransform(newSemantics.Transform)
	newChild.Tags = newSemantics.Tags
	return newChild
}

// DebugFillProperties adds the painters, the preferred size and the hints.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.debugFillProperties
func (r *RenderCustomPaint) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	describe := func(p CustomPainter) string {
		if p == nil {
			return "null"
		}
		return foundation.DescribeIdentity(p)
	}
	level := foundation.DiagnosticLevelFine
	if r.foregroundPainter != nil {
		level = foundation.DiagnosticLevelInfo
	}
	b.Add(foundation.NewMessageProperty("painter", describe(r.painter)))
	b.Add(foundation.NewMessageProperty("foregroundPainter", describe(r.foregroundPainter), foundation.MessagePropertyOpts{Level: level}))
	b.Add(foundation.NewDiagnosticsProperty("preferredSize", r.preferredSize, foundation.DiagnosticsPropertyOpts{DefaultValue: ui.SizeZero}))
	b.Add(foundation.NewDiagnosticsProperty("isComplex", r.IsComplex, foundation.DiagnosticsPropertyOpts{DefaultValue: false}))
	b.Add(foundation.NewDiagnosticsProperty("willChange", r.WillChange, foundation.DiagnosticsPropertyOpts{DefaultValue: false}))
}
