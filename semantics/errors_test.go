package semantics_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// roleNode makes a semantic boundary node with the role. The text direction is
// set, because the strings of a node need one.
func roleNode(role ui.SemanticsRole, tweak func(*semantics.SemanticsConfiguration), children ...*semantics.SemanticsNode) *semantics.SemanticsNode {
	config := semantics.NewSemanticsConfiguration()
	config.SetIsSemanticBoundary(true)
	config.SetRole(role)
	config.SetTextDirection(ui.TextDirectionLTR)
	if tweak != nil {
		tweak(config)
	}
	n := nodeWithRect(rect(0, 0, 10, 10))
	n.UpdateWith(config, children)
	return n
}

// sendUnder puts the nodes under a root and sends the update.
func sendUnder(children ...*semantics.SemanticsNode) func() {
	return func() {
		owner := newOwner()
		root := semantics.NewSemanticsNodeRoot(owner)
		root.SetRect(rect(0, 0, 10, 10))
		root.UpdateWith(boundaryConfig(false), children)
		owner.SendSemanticsUpdate()
	}
}

func withValues(value, min, max string) func(*semantics.SemanticsConfiguration) {
	return func(c *semantics.SemanticsConfiguration) {
		c.SetValue(value)
		c.SetMinValue(foundation.Some(min))
		c.SetMaxValue(foundation.Some(max))
	}
}

func TestRoleCheckRoutes(t *testing.T) {
	skipUnlessDebug(t)
	none := ui.SemanticsRoleNone
	selected := func(c *semantics.SemanticsConfiguration) { c.SetIsSelected(true) }
	checkable := func(c *semantics.SemanticsConfiguration) { c.SetIsChecked(foundation.Some(false)) }
	tests := []struct {
		tag  string
		call func()
	}{
		{"SemanticsTabNeedsSelectedState", sendUnder(roleNode(ui.SemanticsRoleTab, nil))},
		{"SemanticsTabNeedsTapAction", sendUnder(roleNode(ui.SemanticsRoleTab, selected))},
		{"SemanticsTabBarEmpty", sendUnder(roleNode(ui.SemanticsRoleTabBar, nil))},
		{"SemanticsTabBarChildNotTab", sendUnder(roleNode(ui.SemanticsRoleTabBar, nil, roleNode(none, nil)))},
		{"SemanticsTableChildNotRow", sendUnder(roleNode(ui.SemanticsRoleTable, nil, roleNode(none, nil)))},
		{"SemanticsRowParentNotTable", sendUnder(roleNode(ui.SemanticsRoleRow, nil))},
		{"SemanticsRowChildNotCell", sendUnder(roleNode(ui.SemanticsRoleTable, nil, roleNode(ui.SemanticsRoleRow, nil, roleNode(none, nil))))},
		{"SemanticsCellParentInvalid", sendUnder(roleNode(ui.SemanticsRoleCell, nil))},
		{"SemanticsColumnHeaderParentInvalid", sendUnder(roleNode(ui.SemanticsRoleColumnHeader, nil))},
		{"SemanticsRadioGroupMultipleChecked", sendUnder(roleNode(ui.SemanticsRoleRadioGroup, nil,
			roleNode(none, func(c *semantics.SemanticsConfiguration) {
				c.SetIsInMutuallyExclusiveGroup(true)
				c.SetIsChecked(foundation.Some(true))
			}),
			roleNode(none, func(c *semantics.SemanticsConfiguration) {
				c.SetIsInMutuallyExclusiveGroup(true)
				c.SetIsChecked(foundation.Some(true))
			})))},
		{"SemanticsMenuEmpty", sendUnder(roleNode(ui.SemanticsRoleMenu, nil))},
		{"SemanticsMenuBarEmpty", sendUnder(roleNode(ui.SemanticsRoleMenuBar, nil))},
		{"SemanticsMenuItemParentInvalid", sendUnder(roleNode(ui.SemanticsRoleMenuItem, nil))},
		{"SemanticsMenuItemCheckboxNotCheckable", sendUnder(roleNode(ui.SemanticsRoleMenu, nil, roleNode(ui.SemanticsRoleMenuItemCheckbox, nil)))},
		{"SemanticsMenuItemCheckboxParentInvalid", sendUnder(roleNode(ui.SemanticsRoleMenuItemCheckbox, checkable))},
		{"SemanticsMenuItemRadioNotCheckable", sendUnder(roleNode(ui.SemanticsRoleMenu, nil, roleNode(ui.SemanticsRoleMenuItemRadio, nil)))},
		{"SemanticsMenuItemRadioParentInvalid", sendUnder(roleNode(ui.SemanticsRoleMenuItemRadio, checkable))},
		{"SemanticsRoleLiveRegionConflict", sendUnder(roleNode(ui.SemanticsRoleAlert, func(c *semantics.SemanticsConfiguration) { c.SetLiveRegion(true) }))},
		{"SemanticsListItemNoParent", func() {
			owner := newOwner()
			root := semantics.NewSemanticsNodeRoot(owner)
			root.SetRect(rect(0, 0, 10, 10))
			config := semantics.NewSemanticsConfiguration()
			config.SetRole(ui.SemanticsRoleListItem)
			root.UpdateWith(config, nil)
			owner.SendSemanticsUpdate()
		}},
		{"SemanticsListItemParentNotList", sendUnder(roleNode(ui.SemanticsRoleListItem, nil))},
		{"SemanticsLandmarkNested", sendUnder(roleNode(ui.SemanticsRoleRegion, func(c *semantics.SemanticsConfiguration) { c.SetLabel("region") },
			roleNode(ui.SemanticsRoleMain, nil)))},
		{"SemanticsLandmarkLabelMissing", sendUnder(roleNode(ui.SemanticsRoleNavigation, nil), roleNode(ui.SemanticsRoleNavigation, nil))},
		{"SemanticsRegionLabelMissing", sendUnder(roleNode(ui.SemanticsRoleRegion, nil))},
		{"SemanticsProgressBarMissingValues", sendUnder(roleNode(ui.SemanticsRoleProgressBar, nil))},
		{"SemanticsProgressBarNotNumeric", sendUnder(roleNode(ui.SemanticsRoleProgressBar, withValues("x", "0", "10")))},
		{"SemanticsProgressBarMinNotLessThanMax", sendUnder(roleNode(ui.SemanticsRoleProgressBar, withValues("5", "10", "10")))},
		{"SemanticsProgressBarValueOutOfRange", sendUnder(roleNode(ui.SemanticsRoleProgressBar, withValues("11", "0", "10")))},
		{"SemanticsProgressBarPercentOutOfRange", sendUnder(roleNode(ui.SemanticsRoleProgressBar, withValues("101%", "0", "10")))},
		{"SemanticsRoleCheckMissing", sendUnder(roleNode(ui.SemanticsRoleTooltip, nil))},
		{"SemanticsExpandableBothActions", sendUnder(roleNode(none, func(c *semantics.SemanticsConfiguration) {
			c.SetIsExpanded(foundation.Some(true))
			c.SetOnExpand(noop)
			c.SetOnCollapse(noop)
		}))},
		{"SemanticsExpandedHasExpandAction", sendUnder(roleNode(none, func(c *semantics.SemanticsConfiguration) {
			c.SetIsExpanded(foundation.Some(true))
			c.SetOnExpand(noop)
		}))},
		{"SemanticsCollapsedHasCollapseAction", sendUnder(roleNode(none, func(c *semantics.SemanticsConfiguration) {
			c.SetIsExpanded(foundation.Some(false))
			c.SetOnCollapse(noop)
		}))},
		{"SemanticsFocusableAccessibilityBlocked", sendUnder(roleNode(none, func(c *semantics.SemanticsConfiguration) {
			c.SetAccessibilityFocusBlockType(semantics.AccessibilityFocusBlockTypeBlockNode)
			c.SetIsFocused(foundation.Some(false))
		}))},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) { _ = requireTag(t, tc.tag, tc.call) })
	}
}

func TestSemanticsErrorRoutes(t *testing.T) {
	skipUnlessDebug(t)
	attributed := semantics.NewAttributedString("text")
	tests := []struct {
		tag  string
		call func()
	}{
		{"SemanticsActionLabelEmpty", func() { semantics.NewCustomSemanticsAction("") }},
		{"SemanticsActionHintEmpty", func() { semantics.NewCustomSemanticsActionOverriding("", ui.SemanticsActionTap) }},
		{"SemanticsConfigurationDuplicated", func() {
			b := semantics.NewChildSemanticsConfigurationsResultBuilder()
			c := semantics.NewSemanticsConfiguration()
			b.MarkAsMergeUp(c)
			b.MarkAsMergeUp(c)
			b.Build()
		}},
		{"AttributedStringInvalid", func() { semantics.NewAttributedString("", ui.NewSpellOutStringAttribute(ui.NewTextRange(0, 1))) }},
		{"SemanticsDataInvalid", func() { semantics.NewSemanticsData(semantics.SemanticsData{AttributedLabel: attributed}) }},
		{"SemanticsPropertiesInvalid", func() {
			semantics.NewSemanticsProperties(semantics.SemanticsProperties{Label: foundation.Some("a"), AttributedLabel: &attributed})
		}},
		{"OrdinalSortKeyOrderInvalid", func() { semantics.NewOrdinalSortKey(math.NaN()) }},
		{"SemanticsBoundaryRequired", func() { semantics.NewSemanticsConfiguration().SetIsMergingSemanticsOfDescendants(true) }},
		{"SemanticsBoundaryRequired", func() {
			c := boundaryConfig(true)
			c.SetIsSemanticBoundary(false)
		}},
		{"SemanticsHandlerNil", func() { semantics.NewSemanticsConfiguration().SetOnTap(nil) }},
		{"SemanticsActionArgumentsUnexpected", func() {
			c := semantics.NewSemanticsConfiguration()
			c.SetOnTap(noop)
			c.GetActionHandler(ui.SemanticsActionTap)(1)
		}},
		{"SemanticsHeadingLevelInvalid", func() { semantics.NewSemanticsConfiguration().SetHeadingLevel(7) }},
		{"SemanticsAbsorbWithExplicitChildren", func() {
			c := semantics.NewSemanticsConfiguration()
			c.ExplicitChildNodes = true
			c.Absorb(semantics.NewSemanticsConfiguration())
		}},
		{"SemanticsRectNotFinite", func() { semantics.NewSemanticsNode().SetRect(rect(0, 0, math.Inf(1), 1)) }},
		{"SemanticsNodeOwnChild", func() {
			n := semantics.NewSemanticsNode()
			n.UpdateWith(nil, []*semantics.SemanticsNode{n})
		}},
		{"SemanticsChildDuplicated", func() {
			c := semantics.NewSemanticsNode()
			semantics.NewSemanticsNode().UpdateWith(nil, []*semantics.SemanticsNode{c, c})
		}},
		{"SemanticsChildAttached", func() {
			c := semantics.NewSemanticsNode()
			c.Attach(newOwner())
			semantics.NewSemanticsNode().UpdateWith(nil, []*semantics.SemanticsNode{c})
		}},
		{"SemanticsCycle", func() {
			parent, child := semantics.NewSemanticsNode(), semantics.NewSemanticsNode()
			parent.UpdateWith(nil, []*semantics.SemanticsNode{child})
			child.UpdateWith(nil, []*semantics.SemanticsNode{parent})
		}},
		{"SemanticsNodeAttached", func() {
			n, owner := semantics.NewSemanticsNode(), newOwner()
			n.Attach(owner)
			n.Attach(owner)
		}},
		{"SemanticsNodeDetached", func() { semantics.NewSemanticsNode().Detach() }},
		{"SemanticsPlatformViewWithChildren", func() {
			c := semantics.NewSemanticsConfiguration()
			c.SetPlatformViewID(foundation.Some(1))
			semantics.NewSemanticsNode().UpdateWith(c, []*semantics.SemanticsNode{semantics.NewSemanticsNode()})
		}},
		{"SemanticsIncreaseWithoutValue", func() {
			c := semantics.NewSemanticsConfiguration()
			c.SetTextDirection(ui.TextDirectionLTR)
			c.SetOnIncrease(noop)
			c.SetValue("5")
			semantics.NewSemanticsNode().UpdateWith(c, nil)
		}},
		{"SemanticsDecreaseWithoutValue", func() {
			c := semantics.NewSemanticsConfiguration()
			c.SetTextDirection(ui.TextDirectionLTR)
			c.SetOnDecrease(noop)
			c.SetValue("5")
			semantics.NewSemanticsNode().UpdateWith(c, nil)
		}},
		{"SemanticsTraversalParentIdentifierDuplicated", func() {
			parent := func() *semantics.SemanticsNode {
				return roleNode(ui.SemanticsRoleNone, func(c *semantics.SemanticsConfiguration) { c.SetTraversalParentIdentifier("same") })
			}
			sendUnder(parent(), parent())()
		}},
		{"SemanticsTraversalCycle", func() {
			inner := roleNode(ui.SemanticsRoleNone, func(c *semantics.SemanticsConfiguration) { c.SetTraversalParentIdentifier("x") })
			outer := roleNode(ui.SemanticsRoleNone, func(c *semantics.SemanticsConfiguration) { c.SetTraversalChildIdentifier("x") }, inner)
			sendUnder(outer)()
		}},
		{"SemanticsHandleDisposedTwice", func() {
			h := renderhost.New().Binding.SemanticsBinding().EnsureSemantics()
			h.Dispose()
			h.Dispose()
		}},
		{"SemanticsUpdateInvalid", func() { ui.NewSemanticsUpdateBuilder().UpdateNode(ui.SemanticsNodeUpdate{ID: -1}) }},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) { _ = requireTag(t, tc.tag, tc.call) })
	}
}
