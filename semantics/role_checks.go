package semantics

import (
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// checkSemanticsData checks that a node follows the rules of its role; it is
// used in debug builds when the node is added to an update. It returns nil or
// an error with the tag of the rule that is broken.
//
// Flutter: semantics/semantics.dart _DebugSemanticsRoleChecks._checkSemanticsData
func checkSemanticsData(node *SemanticsNode) error {
	var err error
	switch node.role {
	case ui.SemanticsRoleAlertDialog, ui.SemanticsRoleDialog, ui.SemanticsRoleNone, ui.SemanticsRoleTabPanel,
		ui.SemanticsRoleList, ui.SemanticsRoleForm, ui.SemanticsRoleLoadingSpinner:
	case ui.SemanticsRoleTab:
		err = checkTab(node)
	case ui.SemanticsRoleTabBar:
		err = checkTabBar(node)
	case ui.SemanticsRoleTable:
		err = checkTable(node)
	case ui.SemanticsRoleCell:
		if p := node.parent; p == nil || (p.role != ui.SemanticsRoleRow && p.role != ui.SemanticsRoleCell) {
			err = errors.New("SemanticsCellParentInvalid", "A cell must be a child of a row or another cell")
		}
	case ui.SemanticsRoleRow:
		err = checkRow(node)
	case ui.SemanticsRoleColumnHeader:
		if p := node.parent; p == nil || (p.role != ui.SemanticsRoleRow && p.role != ui.SemanticsRoleCell) {
			err = errors.New("SemanticsColumnHeaderParentInvalid", "A columnHeader must be a child or another cell")
		}
	case ui.SemanticsRoleRadioGroup:
		err = checkRadioGroup(node)
	case ui.SemanticsRoleMenu:
		if node.ChildrenCount() < 1 {
			err = errors.New("SemanticsMenuEmpty", "a menu cannot be empty")
		}
	case ui.SemanticsRoleMenuBar:
		if node.ChildrenCount() < 1 {
			err = errors.New("SemanticsMenuBarEmpty", "a menu bar cannot be empty")
		}
	case ui.SemanticsRoleMenuItem:
		if !hasMenuAncestor(node) {
			err = errors.New("SemanticsMenuItemParentInvalid", "A menu item must be a child of a menu or a menu bar")
		}
	case ui.SemanticsRoleMenuItemCheckbox:
		err = checkMenuItemCheckable(node, "SemanticsMenuItemCheckboxNotCheckable", "a menu item checkbox must be checkable",
			"SemanticsMenuItemCheckboxParentInvalid", "A menu item checkbox must be a child of a menu or a menu bar")
	case ui.SemanticsRoleMenuItemRadio:
		err = checkMenuItemCheckable(node, "SemanticsMenuItemRadioNotCheckable", "a menu item radio must be checkable",
			"SemanticsMenuItemRadioParentInvalid", "A menu item radio must be a child of a menu or a menu bar")
	case ui.SemanticsRoleAlert, ui.SemanticsRoleStatus:
		if data := node.GetSemanticsData(); data.FlagsCollection.IsLiveRegion {
			err = errors.New("SemanticsRoleLiveRegionConflict",
				"Node "+itoa(node.id)+" has role "+data.Role.String()+" but is also a live region. A node can not have "+
					data.Role.String()+" and be live region at the same time. Either remove the role or the live region")
		}
	case ui.SemanticsRoleListItem:
		err = checkListItem(node)
	case ui.SemanticsRoleComplementary, ui.SemanticsRoleContentInfo, ui.SemanticsRoleMain:
		err = checkLandmark(node, true)
	case ui.SemanticsRoleNavigation:
		err = checkLandmark(node, false)
	case ui.SemanticsRoleRegion:
		if node.GetSemanticsData().Label() == "" {
			err = errors.New("SemanticsRegionLabelMissing", "A region role should include a label that describes the purpose of the content.")
		}
	case ui.SemanticsRoleProgressBar:
		err = checkProgressBar(node)
	default:
		err = errors.New("SemanticsRoleCheckMissing", "Missing checks for role "+node.GetSemanticsData().Role.String())
	}
	if err != nil {
		return err
	}
	return checkGeneral(node)
}

func checkProgressBar(node *SemanticsNode) error {
	data := node.GetSemanticsData()
	if data.Value() == "" || data.MinValue.OrElse("") == "" || data.MaxValue.OrElse("") == "" {
		return errors.New("SemanticsProgressBarMissingValues", "A progress bar must have a value, a minValue, a maxValue.")
	}
	minVal, errMin := strconv.ParseFloat(data.MinValue.Value(), 64)
	maxVal, errMax := strconv.ParseFloat(data.MaxValue.Value(), 64)
	current, errCur := strconv.ParseFloat(data.Value(), 64)
	hasPercent := false
	var percent float64
	if strings.HasSuffix(data.Value(), "%") {
		p, err := strconv.ParseFloat(strings.TrimSuffix(data.Value(), "%"), 64)
		hasPercent, percent = err == nil, p
	}
	if errMin != nil || errMax != nil || (errCur != nil && !hasPercent) {
		return errors.New("SemanticsProgressBarNotNumeric",
			"Progress bar value, minValue, and maxValue must be valid numbers. value: \""+data.Value()+"\", minValue: \""+
				data.MinValue.Value()+"\", maxValue: \""+data.MaxValue.Value()+"\"")
	}
	if minVal >= maxVal {
		return errors.New("SemanticsProgressBarMinNotLessThanMax",
			"Progress bar minValue ("+data.MinValue.Value()+") must be less than maxValue ("+data.MaxValue.Value()+")")
	}
	if errCur == nil {
		if current < minVal || current > maxVal {
			return errors.New("SemanticsProgressBarValueOutOfRange", "Progress bar value ("+data.Value()+
				") must be between minValue ("+data.MinValue.Value()+") and maxValue ("+data.MaxValue.Value()+")")
		}
		return nil
	}
	if hasPercent && (percent < 0 || percent > 100) {
		return errors.New("SemanticsProgressBarPercentOutOfRange", "Progress bar percentage value ("+data.Value()+") must be between 0% and 100%")
	}
	return nil
}

func checkTab(node *SemanticsNode) error {
	data := node.GetSemanticsData()
	if data.FlagsCollection.IsSelected == ui.TristateNone {
		return errors.New("SemanticsTabNeedsSelectedState", "A tab needs selected states")
	}
	if node.areUserActionsBlocked {
		return nil
	}
	if data.FlagsCollection.IsEnabled != ui.TristateIsFalse && !data.HasAction(ui.SemanticsActionTap) {
		return errors.New("SemanticsTabNeedsTapAction", "A tab must have a tap action")
	}
	return nil
}

func checkTabBar(node *SemanticsNode) error {
	if node.ChildrenCount() < 1 {
		return errors.New("SemanticsTabBarEmpty", "a TabBar cannot be empty")
	}
	var err error
	node.VisitChildren(func(child *SemanticsNode) bool {
		if child.GetSemanticsData().Role != ui.SemanticsRoleTab {
			err = errors.New("SemanticsTabBarChildNotTab", "Children of TabBar must have the tab role")
		}
		return err == nil
	})
	return err
}

func checkTable(node *SemanticsNode) error {
	var err error
	node.VisitChildren(func(child *SemanticsNode) bool {
		if child.GetSemanticsData().Role != ui.SemanticsRoleRow {
			err = errors.New("SemanticsTableChildNotRow", "Children of Table must have the row role")
		}
		return err == nil
	})
	return err
}

func checkRow(node *SemanticsNode) error {
	if p := node.parent; p == nil || p.role != ui.SemanticsRoleTable {
		return errors.New("SemanticsRowParentNotTable", "A row must be a child of a table")
	}
	var err error
	node.VisitChildren(func(child *SemanticsNode) bool {
		if r := child.GetSemanticsData().Role; r != ui.SemanticsRoleCell && r != ui.SemanticsRoleColumnHeader {
			err = errors.New("SemanticsRowChildNotCell", "Children of Row must have the cell or columnHeader role")
		}
		return err == nil
	})
	return err
}

func checkRadioGroup(node *SemanticsNode) error {
	var err error
	hasChecked := false
	var validate SemanticsNodeVisitor
	validate = func(n *SemanticsNode) bool {
		data := n.GetSemanticsData()
		if data.Role == ui.SemanticsRoleRadioGroup {
			return err == nil
		}
		if !data.FlagsCollection.IsInMutuallyExclusiveGroup {
			n.VisitChildren(validate)
			return err == nil
		}
		if data.FlagsCollection.IsChecked == ui.CheckedStateIsTrue {
			if hasChecked {
				err = errors.New("SemanticsRadioGroupMultipleChecked", "Radio groups must not have multiple checked children")
				return false
			}
			hasChecked = true
		}
		return true
	}
	node.VisitChildren(validate)
	return err
}

func hasMenuAncestor(node *SemanticsNode) bool {
	for cur := node; cur != nil && cur.parent != nil; cur = cur.parent {
		if r := cur.parent.role; r == ui.SemanticsRoleMenu || r == ui.SemanticsRoleMenuBar {
			return true
		}
	}
	return false
}

func checkMenuItemCheckable(node *SemanticsNode, tagA, msgA, tagB, msgB string) error {
	if node.GetSemanticsData().FlagsCollection.IsChecked == ui.CheckedStateNone {
		return errors.New(tagA, msgA)
	}
	if !hasMenuAncestor(node) {
		return errors.New(tagB, msgB)
	}
	return nil
}

func checkListItem(node *SemanticsNode) error {
	data := node.GetSemanticsData()
	parent := node.parent
	if parent == nil {
		return errors.New("SemanticsListItemNoParent", "Semantics node "+itoa(node.id)+" has role "+data.Role.String()+" but doesn't have a parent")
	}
	if parent.GetSemanticsData().Role != ui.SemanticsRoleList {
		return errors.New("SemanticsListItemParentNotList", "Semantics node "+itoa(node.id)+" has role "+data.Role.String()+
			", but its parent node "+itoa(parent.id)+" doesn't have the role "+ui.SemanticsRoleList.String()+
			". Please assign the "+ui.SemanticsRoleList.String()+" to node "+itoa(parent.id))
	}
	return nil
}

func isLandmarkRole(d *SemanticsData) bool {
	switch d.Role {
	case ui.SemanticsRoleComplementary, ui.SemanticsRoleContentInfo, ui.SemanticsRoleMain, ui.SemanticsRoleNavigation, ui.SemanticsRoleRegion:
		return true
	}
	return false
}

func isSameRoleExisted(node *SemanticsNode) bool {
	count := 0
	for _, n := range node.owner.nodes {
		if n.GetSemanticsData().Role == node.role {
			count++
			if count > 1 {
				return true
			}
		}
	}
	return false
}

func checkLandmark(node *SemanticsNode, notNested bool) error {
	name := strings.TrimPrefix(node.role.String(), "SemanticsRole.")
	if notNested {
		for cur := node.parent; cur != nil; cur = cur.parent {
			if isLandmarkRole(cur.GetSemanticsData()) {
				return errors.New("SemanticsLandmarkNested", "The "+name+" landmark role should not contained within any other landmark roles.")
			}
		}
	}
	if isSameRoleExisted(node) && node.GetSemanticsData().Label() == "" {
		return errors.New("SemanticsLandmarkLabelMissing", "The "+name+" landmark role should have a unique label as it is used more than once.")
	}
	return nil
}

func checkGeneral(node *SemanticsNode) error {
	data := node.GetSemanticsData()
	if expanded := data.FlagsCollection.IsExpanded.ToBoolOrNull(); expanded != nil {
		hasExpand, hasCollapse := data.HasAction(ui.SemanticsActionExpand), data.HasAction(ui.SemanticsActionCollapse)
		switch {
		case hasExpand && hasCollapse:
			return errors.New("SemanticsExpandableBothActions", "An expandable node cannot have both expand and collapse actions set at the same time.")
		case *expanded && hasExpand:
			return errors.New("SemanticsExpandedHasExpandAction", "An expanded node cannot have an expand action.")
		case !*expanded && hasCollapse:
			return errors.New("SemanticsCollapsedHasCollapseAction", "A collapsed node cannot have a collapse action.")
		}
	}
	if data.FlagsCollection.IsAccessibilityFocusBlocked && data.FlagsCollection.IsFocused != ui.TristateNone {
		return errors.New("SemanticsFocusableAccessibilityBlocked", "A node that is keyboard focusable cannot be set to accessibility unfocusable")
	}
	return nil
}
