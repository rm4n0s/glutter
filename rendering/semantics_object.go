package rendering

import (
	"slices"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// ShowOnScreenOpts are the optional parameters of RenderObject.ShowOnScreen.
//
// Flutter: rendering/object.dart RenderObject.showOnScreen
type ShowOnScreenOpts struct {
	// Descendant is the render object that is to be shown; the default is the
	// object itself.
	Descendant RenderObject
	// Rect is the part of the descendant to show, in its coordinates; the
	// default is its paint bounds.
	Rect foundation.Opt[ui.Rect]
	// Duration is how long the scrolling takes; zero jumps.
	Duration time.Duration
	// Curve is the curve of the scrolling animation; the default (nil) is
	// animation.Curves.Linear.
	Curve animation.Curve
}

// ShowOnScreen scrolls the object into view: it asks its parent, which can be a
// viewport. Without a parent that scrolls it does nothing.
//
// Flutter: rendering/object.dart RenderObject.showOnScreen
func (b *RenderObjectBase) ShowOnScreen(opts ShowOnScreenOpts) {
	if b.parent == nil {
		return
	}
	if isNil(opts.Descendant) {
		opts.Descendant = b.self
	}
	b.parent.ShowOnScreen(opts)
}

// DescribeSemanticsConfiguration fills the configuration with the semantics of
// the object. The default adds nothing; an object that has semantics of its own
// overrides it.
//
// Flutter: rendering/object.dart RenderObject.describeSemanticsConfiguration
func (b *RenderObjectBase) DescribeSemanticsConfiguration(*semantics.SemanticsConfiguration) {}

// VisitChildrenForSemantics visits the children that take part in semantics,
// in paint order: the children of VisitChildren, unless an object hides some.
//
// Flutter: rendering/object.dart RenderObject.visitChildrenForSemantics
func (b *RenderObjectBase) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	b.self.VisitChildren(visitor)
}

// AssembleSemanticsNode gives the node its configuration and its children. An
// object that adds nodes of its own overrides it.
//
// Flutter: rendering/object.dart RenderObject.assembleSemanticsNode
func (b *RenderObjectBase) AssembleSemanticsNode(node *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, children []*semantics.SemanticsNode) {
	node.UpdateWith(config, children)
}

// DescribeSemanticsClip is the clip that the object applies to the semantics of
// a child, in its own coordinates, when that differs from the paint clip; None
// if it is the same. A viewport overrides it to include children that are just
// off screen.
//
// Flutter: rendering/object.dart RenderObject.describeSemanticsClip
func (b *RenderObjectBase) DescribeSemanticsClip(RenderObject) foundation.Opt[ui.Rect] {
	return foundation.None[ui.Rect]()
}

// ScheduleInitialSemantics starts the semantics of a root object: it marks the
// object as needing a semantics update. The owner calls it when it gets a
// semantics owner.
//
// Flutter: rendering/object.dart RenderObject.scheduleInitialSemantics
func (b *RenderObjectBase) ScheduleInitialSemantics() {
	if foundation.DebugMode {
		if b.owner == nil || b.parent != nil {
			panic(errors.New("SemanticsInitialNotRoot", "ScheduleInitialSemantics needs an attached root render object",
				"object", foundation.DescribeIdentity(b.self)))
		}
		if b.owner.debugDoingSemantics {
			panic(errors.New("SemanticsDuringSemantics", "ScheduleInitialSemantics was called while the semantics are flushed"))
		}
		if b.owner.semanticsOwner == nil {
			panic(errors.New("SemanticsOwnerMissing", "ScheduleInitialSemantics needs a semantics owner on the pipeline owner"))
		}
	}
	b.owner.nodesNeedingSemanticsUpdate.add(b.self)
	b.owner.nodesNeedingSemanticsGeometryUpdate.add(b.self)
	b.owner.RequestVisualUpdate()
}

// SendSemanticsEvent sends the event about this object: to its semantics node,
// or to the nearest ancestor with one.
//
// Flutter: rendering/object.dart RenderObject.sendSemanticsEvent
func (b *RenderObjectBase) SendSemanticsEvent(event semantics.SemanticsEvent) {
	if b.owner.SemanticsOwner() == nil {
		return
	}
	node := b.semantics().cachedSemanticsNode
	if node != nil && !node.IsMergedIntoParent() {
		node.SendEvent(event)
	} else if b.parent != nil {
		b.parent.SendSemanticsEvent(event)
	}
}

// DebugNeedsSemanticsUpdate reports whether the semantics need an update (debug
// builds only).
//
// Flutter: rendering/object.dart RenderObject.debugNeedsSemanticsUpdate
func (b *RenderObjectBase) DebugNeedsSemanticsUpdate() bool {
	if !foundation.DebugMode {
		return false
	}
	return b.semantics().parentDataDirty()
}

// DebugSemantics is the semantics node of the object, if it formed one (debug
// builds only).
//
// Flutter: rendering/object.dart RenderObject.debugSemantics
func (b *RenderObjectBase) DebugSemantics() *semantics.SemanticsNode {
	if foundation.DebugMode && b.semantics().built {
		return b.semantics().cachedSemanticsNode
	}
	return nil
}

// ClearSemantics throws away the semantics of the object and of its subtree.
//
// Flutter: rendering/object.dart RenderObject.clearSemantics
func (b *RenderObjectBase) ClearSemantics() {
	b.semantics().clear()
	b.self.VisitChildren(func(child RenderObject) { child.ClearSemantics() })
}

// MarkNeedsSemanticsUpdate marks the semantics of the object as changed: call
// it when what DescribeSemanticsConfiguration reports changes. It does nothing
// while the object has no owner or its owner has no semantics.
//
// Flutter: rendering/object.dart RenderObject.markNeedsSemanticsUpdate
func (b *RenderObjectBase) MarkNeedsSemanticsUpdate() {
	if foundation.DebugMode {
		if b.debugDisposed {
			panic(errors.New("RenderObjectDisposedMutated", "MarkNeedsSemanticsUpdate was called on a disposed render object"))
		}
		if b.owner != nil && b.owner.debugDoingSemantics {
			panic(errors.New("SemanticsMarkedWhileFlushing", "MarkNeedsSemanticsUpdate was called while the semantics are flushed",
				"object", foundation.DescribeIdentity(b.self)))
		}
	}
	if b.owner == nil || b.owner.semanticsOwner == nil {
		return
	}
	b.semantics().markNeedsUpdate()
}

func (b *RenderObjectBase) semantics() *renderObjectSemantics {
	if b.sem == nil {
		b.sem = newRenderObjectSemantics(b.self)
	}
	return b.sem
}

// semanticsParentData is what a render object tells its children about the
// semantics, and what they must know to decide how to take part.
//
// Flutter: rendering/object.dart _SemanticsParentData
type semanticsParentData struct {
	mergeIntoParent             bool
	blocksUserActions           bool
	explicitChildNodes          bool
	tagsForChildren             map[semantics.SemanticsTag]struct{}
	localeForChildren           foundation.Opt[ui.Locale]
	accessibilityFocusBlockType semantics.AccessibilityFocusBlockType
}

func (a *semanticsParentData) equal(b *semanticsParentData) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.mergeIntoParent == b.mergeIntoParent && a.blocksUserActions == b.blocksUserActions &&
		a.explicitChildNodes == b.explicitChildNodes && a.localeForChildren == b.localeForChildren &&
		a.accessibilityFocusBlockType == b.accessibilityFocusBlockType && tagSetsEqual(a.tagsForChildren, b.tagsForChildren)
}

func tagSetsEqual(a, b map[semantics.SemanticsTag]struct{}) bool {
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

// semanticsConfigurationProvider keeps the configuration of a render object:
// the one it described, and the one after the parent's changes.
//
// Flutter: rendering/object.dart _SemanticsConfigurationProvider
type semanticsConfigurationProvider struct {
	renderObject              RenderObject
	isEffectiveConfigWritable bool
	originalConfiguration     *semantics.SemanticsConfiguration
	effectiveConfiguration    *semantics.SemanticsConfiguration
}

func (p *semanticsConfigurationProvider) wasSemanticsBoundary() bool {
	return p.originalConfiguration != nil && p.originalConfiguration.IsSemanticBoundary()
}

func (p *semanticsConfigurationProvider) effective() *semantics.SemanticsConfiguration {
	if p.effectiveConfiguration != nil {
		return p.effectiveConfiguration
	}
	return p.original()
}

func (p *semanticsConfigurationProvider) original() *semantics.SemanticsConfiguration {
	if p.originalConfiguration == nil {
		c := semantics.NewSemanticsConfiguration()
		p.originalConfiguration, p.effectiveConfiguration = c, c
		p.renderObject.DescribeSemanticsConfiguration(c)
		if foundation.DebugMode && c.ExplicitChildNodes && c.ChildConfigurationsDelegate() != nil {
			panic(errors.New("SemanticsExplicitNodesWithDelegate",
				"A SemanticsConfiguration with explicitChildNode set to true cannot have a non-null childConfigsDelegate."))
		}
	}
	return p.originalConfiguration
}

func (p *semanticsConfigurationProvider) updateConfig(callback func(*semantics.SemanticsConfiguration)) {
	if !p.isEffectiveConfigWritable {
		p.effectiveConfiguration = p.original().Copy()
		p.isEffectiveConfigWritable = true
	}
	callback(p.effectiveConfiguration)
}

func (p *semanticsConfigurationProvider) absorbAll(configs []*semantics.SemanticsConfiguration) {
	p.updateConfig(func(c *semantics.SemanticsConfiguration) {
		for _, x := range configs {
			c.Absorb(x)
		}
	})
}

func (p *semanticsConfigurationProvider) reset() {
	p.effectiveConfiguration = p.original()
	p.isEffectiveConfigWritable = false
}

func (p *semanticsConfigurationProvider) clear() {
	p.isEffectiveConfigWritable = false
	p.effectiveConfiguration = nil
	p.originalConfiguration = nil
}

// semanticsFragment is a piece of semantics that can merge into the node of an
// ancestor or a sibling.
//
// Flutter: rendering/object.dart _SemanticsFragment
type semanticsFragment interface {
	configToMergeUp() *semantics.SemanticsConfiguration
	fragmentOwner() *renderObjectSemantics
	markSiblingConfigurationConflict(conflict bool)
}

// incompleteSemanticsFragment stands for a configuration that a delegate made
// up, for children that do not exist yet.
//
// Flutter: rendering/object.dart _IncompleteSemanticsFragment
type incompleteSemanticsFragment struct {
	config *semantics.SemanticsConfiguration
	owner  *renderObjectSemantics
}

func (f *incompleteSemanticsFragment) configToMergeUp() *semantics.SemanticsConfiguration {
	return f.config
}
func (f *incompleteSemanticsFragment) fragmentOwner() *renderObjectSemantics { return f.owner }
func (f *incompleteSemanticsFragment) markSiblingConfigurationConflict(conflict bool) {
	// An incomplete fragment can not conflict.
}

// renderObjectSemantics is the semantics of one render object: how it merges
// with its children, whether it forms a SemanticsNode, and the geometry of that
// node.
//
// Flutter: rendering/object.dart _RenderObjectSemantics
type renderObjectSemantics struct {
	renderObject RenderObject
	configProv   *semanticsConfigurationProvider

	hasSiblingConflict bool
	blocksPrevSibling  foundation.Opt[bool]

	containsIncompleteFragment bool
	built                      bool
	cachedSemanticsNode        *semantics.SemanticsNode
	semanticsNodes             []*semantics.SemanticsNode
	mergeUp                    []semanticsFragment
	children                   []*renderObjectSemantics
	siblingMergeGroups         [][]semanticsFragment

	producedSiblingNodes      []*semantics.SemanticsNode
	producedSiblingNodeGroups map[*semantics.SemanticsNode][]semanticsFragment

	parentData *semanticsParentData
	geometry   *semanticsGeometry

	parentInSemanticsTree *renderObjectSemantics

	currentTreeShapeToken              *struct{}
	firstAncestorNodeWithCleanGeometry *renderObjectSemantics
}

func newRenderObjectSemantics(ro RenderObject) *renderObjectSemantics {
	s := &renderObjectSemantics{renderObject: ro, producedSiblingNodeGroups: map[*semantics.SemanticsNode][]semanticsFragment{}}
	s.configProv = &semanticsConfigurationProvider{renderObject: ro}
	return s
}

func (s *renderObjectSemantics) base() *RenderObjectBase { return s.renderObject.AsRenderObject() }

func (s *renderObjectSemantics) configToMergeUp() *semantics.SemanticsConfiguration {
	if s.shouldFormSemanticsNode() {
		return nil
	}
	return s.configProv.effective()
}

func (s *renderObjectSemantics) fragmentOwner() *renderObjectSemantics { return s }

func (s *renderObjectSemantics) markSiblingConfigurationConflict(conflict bool) {
	s.hasSiblingConflict = conflict
}

func (s *renderObjectSemantics) parent() *renderObjectSemantics {
	if p := s.base().parent; p != nil {
		return p.AsRenderObject().semantics()
	}
	return nil
}

func (s *renderObjectSemantics) isRoot() bool { return s.base().parent == nil }

func (s *renderObjectSemantics) parentDataDirty() bool {
	if s.isRoot() {
		return false
	}
	return s.parentData == nil
}

func (s *renderObjectSemantics) geometryDirty() bool {
	if s.isRoot() {
		return false
	}
	return s.geometry == nil
}

// computeAncestorInfo finds the closest ancestor semantics node with clean
// geometry; the token says whether the tree changed shape since the last time.
func (s *renderObjectSemantics) computeAncestorInfo(token *struct{}) {
	if token == s.currentTreeShapeToken {
		return
	}
	s.currentTreeShapeToken = token
	if s.isRoot() {
		s.firstAncestorNodeWithCleanGeometry = s
		return
	}
	s.firstAncestorNodeWithCleanGeometry = nil
	if s.parentDataDirty() {
		return
	}
	var next *renderObjectSemantics
	if s.shouldFormSemanticsNode() {
		if !s.geometryDirty() {
			s.firstAncestorNodeWithCleanGeometry = s
		}
		next = s.parentInSemanticsTree
	} else {
		next = s
		for !next.parentDataDirty() && !next.shouldFormSemanticsNode() {
			next = next.parent()
		}
	}
	if next == nil {
		return
	}
	if s.firstAncestorNodeWithCleanGeometry == nil {
		next.computeAncestorInfo(token)
		s.firstAncestorNodeWithCleanGeometry = next.firstAncestorNodeWithCleanGeometry
	}
}

func (s *renderObjectSemantics) contributesToSemanticsTree() bool {
	eff := s.configProv.effective()
	return eff.HasBeenAnnotated() || s.containsIncompleteFragment || eff.IsSemanticBoundary() || s.isRoot()
}

func (s *renderObjectSemantics) needsMergingSiblingNodesIntoSelf() bool {
	return s.configProv.effective().IsMergingSemanticsOfDescendants() && len(s.producedSiblingNodes) > 0
}

func (s *renderObjectSemantics) shouldFormSemanticsNode() bool {
	if s.configProv.effective().IsSemanticBoundary() {
		return true
	}
	if s.isRoot() {
		return true
	}
	if !s.contributesToSemanticsTree() {
		return false
	}
	return s.parentData.explicitChildNodes || s.hasSiblingConflict
}

func (s *renderObjectSemantics) isBlockingPreviousSibling() bool {
	if s.blocksPrevSibling.IsSome() {
		return s.blocksPrevSibling.Value()
	}
	blocks := s.configProv.effective().IsBlockingSemanticsOfPreviouslyPaintedNodes
	s.blocksPrevSibling = foundation.Some(blocks)
	if blocks {
		return true
	}
	if s.configProv.effective().IsSemanticBoundary() {
		return false
	}
	s.renderObject.VisitChildrenForSemantics(func(child RenderObject) {
		if child.AsRenderObject().semantics().isBlockingPreviousSibling() {
			s.blocksPrevSibling = foundation.Some(true)
		}
	})
	return s.blocksPrevSibling.Value()
}

func shouldDropSemanticsNode(node *semantics.SemanticsNode) bool { return node.IsInvisible() }

func (s *renderObjectSemantics) markNeedsBuild() {
	s.built = false
	if !s.parentDataDirty() && !s.shouldFormSemanticsNode() {
		return
	}
	for _, group := range s.siblingMergeGroups {
		for _, f := range group {
			sem, ok := f.(*renderObjectSemantics)
			if !ok || sem.parentDataDirty() {
				continue
			}
			if !sem.shouldFormSemanticsNode() {
				sem.markNeedsBuild()
			}
		}
	}
}

func (s *renderObjectSemantics) getNonBlockedChildren() []*renderObjectSemantics {
	var result []*renderObjectSemantics
	s.renderObject.VisitChildrenForSemantics(func(child RenderObject) {
		cs := child.AsRenderObject().semantics()
		if cs.isBlockingPreviousSibling() {
			result = nil
		}
		result = append(result, cs)
	})
	return result
}

func (s *renderObjectSemantics) getTagsForChildren() map[semantics.SemanticsTag]struct{} {
	orig := s.configProv.original().TagsForChildren()
	if s.contributesToSemanticsTree() {
		if orig == nil {
			return nil
		}
		return cloneTags(orig)
	}
	var result map[semantics.SemanticsTag]struct{}
	if orig != nil {
		result = cloneTags(orig)
	}
	if s.parentData != nil && s.parentData.tagsForChildren != nil {
		if result == nil {
			result = s.parentData.tagsForChildren
		} else {
			for t := range s.parentData.tagsForChildren {
				result[t] = struct{}{}
			}
		}
	}
	return result
}

func cloneTags(m map[semantics.SemanticsTag]struct{}) map[semantics.SemanticsTag]struct{} {
	out := make(map[semantics.SemanticsTag]struct{}, len(m))
	for t := range m {
		out[t] = struct{}{}
	}
	return out
}

// updateChildren decides how the children merge, collecting the fragments that
// merge up and the groups that merge into siblings.
func (s *renderObjectSemantics) updateChildren() {
	s.configProv.reset()
	tagsForChildren := s.getTagsForChildren()
	eff := s.configProv.effective()
	explicitChildNodesForChildren := s.isRoot() || eff.ExplicitChildNodes ||
		(!s.contributesToSemanticsTree() && (s.parentData == nil || s.parentData.explicitChildNodes))
	blocksUserAction := (s.parentData != nil && s.parentData.blocksUserActions) || eff.IsBlockingUserActions

	var focusBlock semantics.AccessibilityFocusBlockType
	if s.parentData != nil && s.parentData.accessibilityFocusBlockType == semantics.AccessibilityFocusBlockTypeBlockSubtree {
		focusBlock = semantics.AccessibilityFocusBlockTypeBlockSubtree
	} else {
		focusBlock = eff.AccessibilityFocusBlockType()
	}
	localeForChildren := eff.LocaleForSubtree()
	if localeForChildren.IsNone() && s.parentData != nil {
		localeForChildren = s.parentData.localeForChildren
	}

	s.siblingMergeGroups = nil
	s.mergeUp = nil
	childParentData := &semanticsParentData{
		mergeIntoParent:             (s.parentData != nil && s.parentData.mergeIntoParent) || eff.IsMergingSemanticsOfDescendants(),
		blocksUserActions:           blocksUserAction,
		accessibilityFocusBlockType: focusBlock,
		localeForChildren:           localeForChildren,
		explicitChildNodes:          explicitChildNodesForChildren,
		tagsForChildren:             tagsForChildren,
	}
	mergeUp, groups := s.collectChildMergeUpAndSiblingGroup(childParentData)
	s.mergeUp = append(s.mergeUp, mergeUp...)
	s.siblingMergeGroups = append(s.siblingMergeGroups, groups...)

	oldChildren := s.children
	s.children = nil
	if !s.contributesToSemanticsTree() {
		return
	}
	s.marksConflictsInMergeGroup(s.mergeUp, true)
	for _, g := range s.siblingMergeGroups {
		s.marksConflictsInMergeGroup(g, false)
	}
	var mergeUpConfigs []*semantics.SemanticsConfiguration
	for _, f := range s.mergeUp {
		if c := f.configToMergeUp(); c != nil {
			mergeUpConfigs = append(mergeUpConfigs, c)
		}
	}
	s.configProv.absorbAll(mergeUpConfigs)
	childFragments := mergeUp
	s.mergeUp = []semanticsFragment{s}
	for _, f := range childFragments {
		cs, ok := f.(*renderObjectSemantics)
		if !ok {
			continue
		}
		if cs.shouldFormSemanticsNode() {
			for _, child := range cs.children {
				child.parentInSemanticsTree = cs
			}
			if cs.geometryDirty() {
				s.base().owner.nodesNeedingSemanticsGeometryUpdate.add(cs.renderObject)
			}
			s.children = append(s.children, cs)
		} else {
			s.children = append(s.children, cs.children...)
			s.siblingMergeGroups = append(s.siblingMergeGroups, cs.siblingMergeGroups...)
		}
	}
	if s.isRoot() || s.configProv.effective().IsSemanticBoundary() {
		for _, child := range s.children {
			child.parentInSemanticsTree = s
		}
	}
	for _, removed := range oldChildren {
		if !slices.Contains(s.children, removed) && removed.parentInSemanticsTree == s {
			removed.parentInSemanticsTree = nil
		}
	}

	if s.parentData != nil && s.parentData.tagsForChildren != nil {
		tags := s.parentData.tagsForChildren
		s.configProv.updateConfig(func(c *semantics.SemanticsConfiguration) {
			for t := range tags {
				c.AddTagForChildren(t)
			}
		})
	}
	if focusBlock != s.configProv.effective().AccessibilityFocusBlockType() {
		s.configProv.updateConfig(func(c *semantics.SemanticsConfiguration) { c.SetAccessibilityFocusBlockType(focusBlock) })
	}
	if blocksUserAction != s.configProv.effective().IsBlockingUserActions {
		s.configProv.updateConfig(func(c *semantics.SemanticsConfiguration) { c.IsBlockingUserActions = blocksUserAction })
	}
	if localeForChildren != s.configProv.effective().Locale {
		s.configProv.updateConfig(func(c *semantics.SemanticsConfiguration) { c.Locale = localeForChildren })
	}
	if focusBlock != semantics.AccessibilityFocusBlockTypeNone {
		s.configProv.updateConfig(func(c *semantics.SemanticsConfiguration) { c.SetIsFocused(foundation.None[bool]()) })
	}
}

func (s *renderObjectSemantics) collectChildMergeUpAndSiblingGroup(childParentData *semanticsParentData) (mergeUp []semanticsFragment, groups [][]semanticsFragment) {
	var childConfigurations []*semantics.SemanticsConfiguration
	delegate := s.configProv.effective().ChildConfigurationsDelegate()
	hasDelegate := delegate != nil
	configToFragment := map[*semantics.SemanticsConfiguration]semanticsFragment{}

	needsIncompleteAssumption := hasDelegate && childParentData.explicitChildNodes
	effective := childParentData
	if needsIncompleteAssumption {
		cp := *childParentData
		cp.explicitChildNodes = false
		effective = &cp
	}
	for _, child := range s.getNonBlockedChildren() {
		child.didUpdateParentData(effective)
		for _, fragment := range child.mergeUp {
			if c := fragment.configToMergeUp(); hasDelegate && c != nil {
				childConfigurations = append(childConfigurations, c)
				configToFragment[c] = fragment
			} else {
				mergeUp = append(mergeUp, fragment)
			}
		}
		if !child.contributesToSemanticsTree() {
			groups = append(groups, child.siblingMergeGroups...)
		}
	}
	s.containsIncompleteFragment = false
	if hasDelegate {
		result := delegate(childConfigurations)
		for _, c := range result.MergeUp {
			if f, ok := configToFragment[c]; ok {
				mergeUp = append(mergeUp, f)
			} else {
				s.containsIncompleteFragment = true
				mergeUp = append(mergeUp, &incompleteSemanticsFragment{c, s})
			}
		}
		for _, group := range result.SiblingMergeGroups {
			var fragments []semanticsFragment
			for _, c := range group {
				if f, ok := configToFragment[c]; ok {
					fragments = append(fragments, f)
				} else {
					s.containsIncompleteFragment = true
					fragments = append(fragments, &incompleteSemanticsFragment{c, s})
				}
			}
			groups = append(groups, fragments)
		}
	}
	if !s.containsIncompleteFragment && needsIncompleteAssumption {
		mergeUp, groups = nil, nil
		for _, child := range s.getNonBlockedChildren() {
			child.didUpdateParentData(childParentData)
			mergeUp = append(mergeUp, child.mergeUp...)
			if !child.contributesToSemanticsTree() {
				groups = append(groups, child.siblingMergeGroups...)
			}
		}
	}
	return mergeUp, groups
}

func (s *renderObjectSemantics) didUpdateParentData(newParentData *semanticsParentData) {
	if s.parentData.equal(newParentData) {
		return
	}
	s.markNeedsBuild()
	s.parentData = newParentData
	s.updateChildren()
}

// ensureGeometry computes the geometry of the children whose geometry is
// missing.
func (s *renderObjectSemantics) ensureGeometry() {
	if s.isRoot() {
		if s.geometry == nil || s.geometry.rect != s.renderObject.SemanticBounds() {
			s.markNeedsBuild()
		}
		s.geometry = rootSemanticsGeometry(s.renderObject.SemanticBounds())
	}
	s.updateChildGeometry(true)
}

func (s *renderObjectSemantics) updateChildGeometry(onlyDirtyChildren bool) {
	parentGeometry := s.geometry
	for _, child := range s.children {
		if onlyDirtyChildren && !child.geometryDirty() {
			continue
		}
		childGeometry := computeChildSemanticsGeometry(nil, parentGeometry.paintClipRect, parentGeometry.semanticsClipRect, s, child)
		child.updateGeometry(childGeometry)
	}
	for _, group := range s.siblingMergeGroups {
		for _, f := range group {
			sib, ok := f.(*renderObjectSemantics)
			if !ok {
				continue
			}
			var targets []*renderObjectSemantics
			if sib.shouldFormSemanticsNode() {
				targets = []*renderObjectSemantics{sib}
			} else {
				targets = sib.children
			}
			for _, explicit := range targets {
				if onlyDirtyChildren && !explicit.geometryDirty() {
					continue
				}
				childGeometry := computeChildSemanticsGeometry(parentGeometry.transform, parentGeometry.paintClipRect, parentGeometry.semanticsClipRect, s, explicit)
				explicit.updateGeometry(childGeometry)
			}
		}
	}
}

func (s *renderObjectSemantics) updateGeometry(newGeometry *semanticsGeometry) {
	current := s.geometry
	s.geometry = newGeometry
	if current != nil {
		isHidden := s.configProv.original().IsHidden() || ((s.parentData == nil || !s.parentData.mergeIntoParent) && newGeometry.hidden)
		sizeChanged := current.rect.Size() != newGeometry.rect.Size()
		visibilityChanged := s.configProv.effective().IsHidden() != isHidden
		if !sizeChanged && !visibilityChanged {
			return
		}
	}
	s.markNeedsBuild()
	s.updateChildGeometry(false)
}

// ensureSemanticsNode builds the semantics node of the object, or of its
// subtree if the node exists.
func (s *renderObjectSemantics) ensureSemanticsNode() {
	if !s.built {
		s.buildSemantics(map[int]struct{}{})
	} else {
		s.buildSemanticsSubtree(map[int]struct{}{})
	}
}

func (s *renderObjectSemantics) buildSemantics(usedIDs map[int]struct{}) {
	if s.cachedSemanticsNode != nil {
		for _, node := range s.semanticsNodes {
			if node != s.cachedSemanticsNode {
				node.Tags = nil
			}
		}
	}
	if !s.built {
		s.produceSemanticsNode(usedIDs)
	}
	produced := s.cachedSemanticsNode
	for _, node := range s.semanticsNodes {
		if node != produced {
			if s.parentData != nil && s.parentData.tagsForChildren != nil {
				if node.Tags == nil {
					node.Tags = map[semantics.SemanticsTag]struct{}{}
				}
				for t := range s.parentData.tagsForChildren {
					node.Tags[t] = struct{}{}
				}
			} else if node.Tags != nil && len(node.Tags) == 0 {
				node.Tags = nil
			}
		}
	}
}

func (s *renderObjectSemantics) buildSemanticsSubtree(usedIDs map[int]struct{}) {
	var children []*semantics.SemanticsNode
	for _, child := range s.children {
		if child.parentDataDirty() {
			continue
		}
		if child.cachedSemanticsNode != nil {
			if _, used := usedIDs[child.cachedSemanticsNode.ID()]; used {
				child.markNeedsBuild()
				child.cachedSemanticsNode = nil
			}
		}
		child.buildSemantics(usedIDs)
		children = append(children, child.semanticsNodes...)
	}
	node := s.cachedSemanticsNode
	children = slices.DeleteFunc(children, shouldDropSemanticsNode)
	isHidden := s.configProv.original().IsHidden() || ((s.parentData == nil || !s.parentData.mergeIntoParent) && s.geometry.hidden)
	if s.configProv.effective().IsHidden() != isHidden {
		s.configProv.updateConfig(func(c *semantics.SemanticsConfiguration) { c.SetIsHidden(isHidden) })
	}
	if s.configProv.effective().IsSemanticBoundary() {
		if s.needsMergingSiblingNodesIntoSelf() {
			inner := semantics.NewSemanticsNode(semantics.SemanticsNodeOpts{ShowOnScreen: s.showOnScreen})
			s.renderObject.AssembleSemanticsNode(inner, s.configProv.effective(), children)
			config := semantics.NewSemanticsConfiguration()
			config.SetIsSemanticBoundary(true)
			config.SetIsMergingSemanticsOfDescendants(true)
			node.UpdateWith(config, append([]*semantics.SemanticsNode{inner}, s.producedSiblingNodes...))
		} else {
			s.renderObject.AssembleSemanticsNode(node, s.configProv.effective(), children)
		}
	} else {
		node.UpdateWith(s.configProv.effective(), children)
	}
}

func (s *renderObjectSemantics) showOnScreen() {
	s.renderObject.ShowOnScreen(ShowOnScreenOpts{})
}

func (s *renderObjectSemantics) produceSemanticsNode(usedIDs map[int]struct{}) {
	s.semanticsNodes = nil
	s.producedSiblingNodes = nil
	clear(s.producedSiblingNodeGroups)

	if s.cachedSemanticsNode == nil {
		s.cachedSemanticsNode = s.createSemanticsNode()
	}
	node := s.cachedSemanticsNode
	node.SetIsMergedIntoParent(s.parentData != nil && s.parentData.mergeIntoParent)
	if s.parentData != nil {
		node.Tags = s.parentData.tagsForChildren
	} else {
		node.Tags = nil
	}
	s.updateSemanticsNodeGeometry()

	s.mergeSiblingGroup(usedIDs)
	s.buildSemanticsSubtree(usedIDs)

	s.semanticsNodes = append(s.semanticsNodes, node)
	if !s.needsMergingSiblingNodesIntoSelf() {
		s.semanticsNodes = append(s.semanticsNodes, s.producedSiblingNodes...)
	}
	s.built = true
}

func (s *renderObjectSemantics) createSemanticsNode() *semantics.SemanticsNode {
	if s.isRoot() {
		return semantics.NewSemanticsNodeRoot(s.base().owner.SemanticsOwner(), semantics.SemanticsNodeOpts{ShowOnScreen: s.showOnScreen})
	}
	return semantics.NewSemanticsNode(semantics.SemanticsNodeOpts{ShowOnScreen: s.showOnScreen})
}

func (s *renderObjectSemantics) mergeSiblingGroup(usedIDs map[int]struct{}) {
	for _, group := range s.siblingMergeGroups {
		var configuration *semantics.SemanticsConfiguration
		var node *semantics.SemanticsNode
		var explicitChildren []*renderObjectSemantics
		for _, fragment := range group {
			if sem, ok := fragment.(*renderObjectSemantics); ok {
				if sem.shouldFormSemanticsNode() {
					explicitChildren = append(explicitChildren, sem)
					continue
				}
				explicitChildren = append(explicitChildren, sem.children...)
			}
			if c := fragment.configToMergeUp(); c != nil {
				if node == nil {
					node = fragment.fragmentOwner().cachedSemanticsNode
				}
				if configuration == nil {
					configuration = semantics.NewSemanticsConfiguration()
				}
				configuration.Absorb(c)
			}
		}
		var childrenNodes []*semantics.SemanticsNode
		for _, explicit := range explicitChildren {
			explicit.buildSemantics(usedIDs)
			childrenNodes = append(childrenNodes, explicit.semanticsNodes...)
		}
		if configuration == nil {
			continue
		}
		if node != nil {
			if _, used := usedIDs[node.ID()]; used {
				node = nil
			}
		}
		if node == nil {
			node = semantics.NewSemanticsNode(semantics.SemanticsNodeOpts{ShowOnScreen: s.showOnScreen})
		}
		usedIDs[node.ID()] = struct{}{}
		for _, fragment := range group {
			if fragment.configToMergeUp() != nil {
				fragment.fragmentOwner().built = true
				fragment.fragmentOwner().cachedSemanticsNode = node
			}
		}
		node.UpdateWith(configuration, childrenNodes)
		s.producedSiblingNodes = append(s.producedSiblingNodes, node)
		s.producedSiblingNodeGroups[node] = group

		tags := map[semantics.SemanticsTag]struct{}{}
		for _, fragment := range group {
			if pd := fragment.fragmentOwner().parentData; pd != nil {
				for t := range pd.tagsForChildren {
					tags[t] = struct{}{}
				}
			}
		}
		if len(tags) > 0 {
			if node.Tags == nil {
				node.Tags = tags
			} else {
				for t := range tags {
					node.Tags[t] = struct{}{}
				}
			}
		}
		node.SetIsMergedIntoParent(s.parentData != nil && s.parentData.mergeIntoParent)
	}
	s.updateSiblingNodesGeometries()
}

func (s *renderObjectSemantics) updateSemanticsNodeGeometry() {
	node := s.cachedSemanticsNode
	g := s.geometry
	node.SetRect(g.rect)
	node.SetTransform(g.transform)
	node.ParentSemanticsClipRect = g.semanticsClipRect
	node.ParentPaintClipRect = g.paintClipRect
}

func (s *renderObjectSemantics) updateSiblingNodesGeometries() {
	main := s.geometry
	for _, node := range s.producedSiblingNodes {
		var rect, semanticsClip, paintClip foundation.Opt[ui.Rect]
		for _, fragment := range s.producedSiblingNodeGroups[node] {
			if fragment.fragmentOwner().shouldFormSemanticsNode() {
				continue
			}
			parentGeometry := computeChildSemanticsGeometry(main.transform, main.paintClipRect, main.semanticsClipRect, s, fragment.fragmentOwner())
			bounds := fragment.fragmentOwner().renderObject.SemanticBounds()
			inOwner := bounds
			if sc, ok := parentGeometry.semanticsClipRect.Get(); ok {
				inOwner = sc.Intersect(bounds)
			}
			inParent := parentGeometry.transform.TransformRect(inOwner)
			if r, ok := rect.Get(); ok {
				rect = foundation.Some(r.ExpandToInclude(inParent))
			} else {
				rect = foundation.Some(inParent)
			}
			if sc, ok := parentGeometry.semanticsClipRect.Get(); ok {
				r := parentGeometry.transform.TransformRect(sc)
				if cur, ok := semanticsClip.Get(); ok {
					r = cur.Intersect(r)
				}
				semanticsClip = foundation.Some(r)
			}
			if pc, ok := parentGeometry.paintClipRect.Get(); ok {
				r := parentGeometry.transform.TransformRect(pc)
				if cur, ok := paintClip.Get(); ok {
					r = cur.Intersect(r)
				}
				paintClip = foundation.Some(r)
			}
		}
		node.SetRect(rect.Value())
		node.SetTransform(nil)
		node.ParentSemanticsClipRect = semanticsClip
		node.ParentPaintClipRect = paintClip
	}
}

// markNeedsUpdate throws away what is known about the semantics of the object
// and of its ancestors up to the closest semantic boundary, and schedules an
// update there.
func (s *renderObjectSemantics) markNeedsUpdate() {
	owner := s.base().owner
	owner.nodesNeedingSemanticsGeometryUpdate.add(s.renderObject)
	produced := s.cachedSemanticsNode
	wasBoundary := produced != nil && s.configProv.wasSemanticsBoundary()

	s.configProv.clear()
	s.containsIncompleteFragment = false

	mayProduceSiblingNodes := s.configProv.effective().ChildConfigurationsDelegate() != nil
	isEffectiveBoundary := s.configProv.effective().IsSemanticBoundary() && wasBoundary
	node := s.renderObject
	for node.AsRenderObject().parent != nil && (mayProduceSiblingNodes || !isEffectiveBoundary) {
		ns := node.AsRenderObject().semantics()
		if node != s.renderObject && ns.parentDataDirty() && !mayProduceSiblingNodes {
			break
		}
		ns.parentData = nil
		ns.blocksPrevSibling = foundation.None[bool]()
		if isEffectiveBoundary {
			mayProduceSiblingNodes = false
		}
		mayProduceSiblingNodes = mayProduceSiblingNodes || ns.configProv.effective().ChildConfigurationsDelegate() != nil
		node = node.AsRenderObject().parent
		isEffectiveBoundary = node.AsRenderObject().semantics().configProv.effective().IsSemanticBoundary() && node.AsRenderObject().semantics().built
	}
	nodeSem := node.AsRenderObject().semantics()
	if node != s.renderObject && produced != nil && nodeSem.parentDataDirty() {
		owner.nodesNeedingSemanticsUpdate.remove(s.renderObject)
	}
	if !nodeSem.parentDataDirty() || nodeSem.isRoot() {
		if owner.nodesNeedingSemanticsUpdate.add(node) {
			owner.RequestVisualUpdate()
		}
	}
}

func (s *renderObjectSemantics) marksConflictsInMergeGroup(group []semanticsFragment, isMergeUp bool) {
	conflict := map[semanticsFragment]struct{}{}
	for i, fragment := range group {
		fragment.markSiblingConfigurationConflict(false)
		cfg := fragment.configToMergeUp()
		if cfg == nil {
			continue
		}
		if isMergeUp && !s.configProv.original().IsCompatibleWith(cfg) {
			conflict[fragment] = struct{}{}
		}
		for j := 0; j < i; j++ {
			sibling := group[j]
			if !cfg.IsCompatibleWith(sibling.configToMergeUp()) {
				conflict[fragment] = struct{}{}
				conflict[sibling] = struct{}{}
			}
		}
	}
	for f := range conflict {
		f.markSiblingConfigurationConflict(true)
	}
}

// clear forgets everything about the semantics of the object.
func (s *renderObjectSemantics) clear() {
	s.built = false
	s.cachedSemanticsNode = nil
	s.parentData = nil
	s.geometry = nil
	s.blocksPrevSibling = foundation.None[bool]()
	s.containsIncompleteFragment = false
	s.mergeUp = nil
	s.siblingMergeGroups = nil
	s.children = nil
	s.semanticsNodes = nil
	s.configProv.clear()
}

// DebugFillProperties adds the semantics state of the object.
func (s *renderObjectSemantics) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewStringProperty("owner", foundation.Some(foundation.DescribeIdentity(s.renderObject))))
	p.Add(foundation.NewFlagProperty("noParentData", foundation.Some(s.parentDataDirty()), foundation.FlagPropertyOpts{IfTrue: "NO PARENT DATA"}))
	p.Add(foundation.NewFlagProperty("geometry", foundation.Some(s.geometryDirty()), foundation.FlagPropertyOpts{IfTrue: "NO GEOMETRY"}))
	p.Add(foundation.NewFlagProperty("semanticsBlock",
		foundation.Some(s.configProv.effective().IsBlockingSemanticsOfPreviouslyPaintedNodes), foundation.FlagPropertyOpts{IfTrue: "BLOCK PREVIOUS"}))
	if !s.parentDataDirty() && s.contributesToSemanticsTree() {
		status := "no semantics node"
		if s.built {
			status = "formed " + itoa(s.cachedSemanticsNode.ID())
		} else if s.shouldFormSemanticsNode() {
			status = "needs build"
		}
		p.Add(foundation.NewStringProperty("formedSemanticsNode", foundation.Some(status), foundation.StringPropertyOpts{Unquoted: true}))
	}
	p.Add(foundation.NewFlagProperty("isSemanticBoundary", foundation.Some(s.configProv.effective().IsSemanticBoundary()),
		foundation.FlagPropertyOpts{IfTrue: "semantic boundary"}))
	p.Add(foundation.NewFlagProperty("blocksSemantics", foundation.Some(s.isBlockingPreviousSibling()), foundation.FlagPropertyOpts{IfTrue: "BLOCKS SEMANTICS"}))
}

// DebugDescribeChildren describes the children that form nodes.
func (s *renderObjectSemantics) DebugDescribeChildren() []foundation.DiagnosticsNode {
	out := make([]foundation.DiagnosticsNode, len(s.children))
	for i, c := range s.children {
		out[i] = foundation.ToDiagnosticsNode(c)
	}
	return out
}

// ToStringShort is the name of the object the semantics belong to.
func (s *renderObjectSemantics) ToStringShort() string { return "_RenderObjectSemantics" }

// DebugDumpRenderObjectSemanticsTree is the dump of the semantics compilation of
// a render tree, for debugging the framework.
//
// Flutter: rendering/object.dart debugDumpRenderObjectSemanticsTree
func DebugDumpRenderObjectSemanticsTree(root RenderObject) string {
	return foundation.ToStringDeep(root.AsRenderObject().semantics())
}

// semanticsGeometry is where a semantics node is, in the coordinates of its
// parent node, and what clips it.
//
// Flutter: rendering/object.dart _SemanticsGeometry
type semanticsGeometry struct {
	transform         *ui.Matrix4
	semanticsClipRect foundation.Opt[ui.Rect]
	paintClipRect     foundation.Opt[ui.Rect]
	rect              ui.Rect
	hidden            bool
}

func rootSemanticsGeometry(rect ui.Rect) *semanticsGeometry {
	return &semanticsGeometry{transform: ui.NewMatrix4Identity(), rect: rect}
}

func (g *semanticsGeometry) isVisible() bool { return !g.rect.IsEmpty() && !g.transform.IsZero() }

func transformSemanticsRect(rect foundation.Opt[ui.Rect], transform *ui.Matrix4, inverse bool) foundation.Opt[ui.Rect] {
	r, ok := rect.Get()
	if !ok {
		return rect
	}
	if r.IsEmpty() || transform.IsZero() {
		return foundation.Some(ui.RectZero)
	}
	if inverse {
		return foundation.Some(transform.InverseTransformRect(r))
	}
	return foundation.Some(transform.TransformRect(r))
}

func intersectOptRects(a, b foundation.Opt[ui.Rect]) foundation.Opt[ui.Rect] {
	bv, ok := b.Get()
	if !ok {
		return a
	}
	if av, ok := a.Get(); ok {
		return foundation.Some(av.Intersect(bv))
	}
	return b
}

// computeChildSemanticsGeometry computes the geometry of the child semantics in
// the coordinates of the parent semantics, following the render objects between
// them.
//
// Flutter: rendering/object.dart _SemanticsGeometry.computeChildGeometry
func computeChildSemanticsGeometry(parentTransform *ui.Matrix4, parentPaintClipRect, parentSemanticsClipRect foundation.Opt[ui.Rect],
	parent, child *renderObjectSemantics) *semanticsGeometry {
	childRO := child.renderObject
	parentRO := parent.renderObject
	childToCommon := []RenderObject{childRO}
	for childRO.Depth() > parentRO.Depth() {
		childRO = childRO.Parent()
		childToCommon = append(childToCommon, childRO)
	}
	var paintClip, semanticsClip foundation.Opt[ui.Rect]
	transform := ui.NewMatrix4Identity()
	for i := len(childToCommon) - 1; i > 0; i-- {
		nodeParent, node := childToCommon[i], childToCommon[i-1]
		localPaint := transformSemanticsRect(nodeParent.DescribeApproximatePaintClip(node), transform, false)
		localSemantics := transformSemanticsRect(nodeParent.DescribeSemanticsClip(node), transform, false)
		paintClip = intersectOptRects(paintClip, localPaint)
		if localSemantics.IsSome() {
			semanticsClip = localSemantics
		} else if sc, ok := semanticsClip.Get(); ok {
			if lp, ok := localPaint.Get(); ok {
				semanticsClip = foundation.Some(sc.Intersect(lp))
			} else {
				semanticsClip = foundation.Some(sc.Intersect(sc))
			}
		}
		nodeParent.ApplyPaintTransform(node, transform)
	}
	if semanticsClip.IsNone() {
		semanticsClip = intersectOptRects(paintClip, parentSemanticsClipRect)
	}
	paintClip = intersectOptRects(paintClip, parentPaintClipRect)

	if paintClip.IsSome() || semanticsClip.IsSome() {
		inverted := transform.Clone()
		if inverted.Invert() != 0 {
			semanticsClip = transformSemanticsRect(semanticsClip, inverted, false)
			paintClip = transformSemanticsRect(paintClip, inverted, false)
		} else {
			semanticsClip, paintClip = foundation.None[ui.Rect](), foundation.None[ui.Rect]()
		}
	}
	if parentTransform != nil {
		ui.MatrixMultiplyInPlace(parentTransform, transform)
	}
	bounds := child.renderObject.SemanticBounds()
	rect := bounds
	if sc, ok := semanticsClip.Get(); ok {
		rect = sc.Intersect(bounds)
	}
	hidden := false
	if pc, ok := paintClip.Get(); ok {
		paintRect := pc.Intersect(rect)
		hidden = paintRect.IsEmpty() && !rect.IsEmpty()
		if !hidden {
			rect = paintRect
		}
	}
	return &semanticsGeometry{transform: transform, paintClipRect: paintClip, semanticsClipRect: semanticsClip, rect: rect, hidden: hidden}
}

// orderedSet is a set that keeps the order of insertion, like Dart's default
// Set: the order decides the IDs of the new semantics nodes.
type orderedSet[T comparable] struct {
	index map[T]struct{}
	list  []T
}

func (s *orderedSet[T]) add(v T) bool {
	if _, ok := s.index[v]; ok {
		return false
	}
	if s.index == nil {
		s.index = map[T]struct{}{}
	}
	s.index[v] = struct{}{}
	s.list = append(s.list, v)
	return true
}

func (s *orderedSet[T]) remove(v T) {
	if _, ok := s.index[v]; !ok {
		return
	}
	delete(s.index, v)
	for i, x := range s.list {
		if x == v {
			s.list = append(s.list[:i], s.list[i+1:]...)
			break
		}
	}
}

// take returns the items and empties the set.
func (s *orderedSet[T]) take() []T {
	out := s.list
	s.list, s.index = nil, nil
	return out
}
