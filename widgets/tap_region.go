package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// TapRegionCallback is the type of the callbacks of a TapRegion for a pointer
// that goes down.
//
// Flutter: widgets/tap_region.dart TapRegionCallback
type TapRegionCallback func(event *gestures.PointerDownEvent)

// TapRegionUpCallback is the type of the callbacks of a TapRegion for a pointer
// that goes up.
//
// Flutter: widgets/tap_region.dart TapRegionUpCallback
type TapRegionUpCallback func(event *gestures.PointerUpEvent)

// TapRegionRegistry is the interface of the object that tracks the TapRegions:
// the RenderTapRegionSurface.
//
// Flutter: widgets/tap_region.dart TapRegionRegistry
type TapRegionRegistry interface {
	// RegisterTapRegion adds a region to the registry.
	//
	// Flutter: widgets/tap_region.dart TapRegionRegistry.registerTapRegion
	RegisterTapRegion(region *RenderTapRegion)
	// UnregisterTapRegion removes a region from the registry.
	//
	// Flutter: widgets/tap_region.dart TapRegionRegistry.unregisterTapRegion
	UnregisterTapRegion(region *RenderTapRegion)
}

// TapRegionRegistryMaybeOf is the registry of the nearest TapRegionSurface
// above the context, or nil.
//
// Flutter: widgets/tap_region.dart TapRegionRegistry.maybeOf
func TapRegionRegistryMaybeOf(bc BuildContext) TapRegionRegistry {
	if s, ok := FindAncestorRenderObjectOfType[*RenderTapRegionSurface](bc); ok {
		return s
	}
	return nil
}

// TapRegionRegistryOf is the registry of the nearest TapRegionSurface above the
// context. It panics with the tag TapRegionRegistryMissing in debug builds when
// there is none.
//
// Flutter: widgets/tap_region.dart TapRegionRegistry.of
func TapRegionRegistryOf(bc BuildContext) TapRegionRegistry {
	r := TapRegionRegistryMaybeOf(bc)
	if r == nil && foundation.DebugMode {
		panic(errors.New(tagTapRegionRegistryMissing, "TapRegionRegistryOf was called with a context that does not contain a TapRegionSurface widget",
			"context", bc.DescribeElement("context").String()))
	}
	return r
}

// TapRegionSurface is the widget that tracks the pointer events for the
// TapRegions below it, and tells them whether the pointer went down or up inside
// or outside of them. A WidgetsApp has one.
//
// Flutter: widgets/tap_region.dart TapRegionSurface
type TapRegionSurface struct {
	SingleChildRenderObjectWidget
}

// NewTapRegionSurface creates the widget.
//
// Flutter: widgets/tap_region.dart TapRegionSurface.new
func NewTapRegionSurface(child Widget, opts ...WidgetOpts) *TapRegionSurface {
	return &TapRegionSurface{NewSingleChildRenderObjectWidgetBase(child, firstOpts(opts))}
}

// CreateRenderObject creates the surface.
//
// Flutter: widgets/tap_region.dart TapRegionSurface.createRenderObject
func (w *TapRegionSurface) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	var g *gestures.GestureBinding
	if b := MaybeWidgetsBindingOf(bc); b != nil {
		g = b.Gestures()
	}
	return NewRenderTapRegionSurface(g)
}

// RenderTapRegionSurface is the render object of a TapRegionSurface.
//
// Deviation: it doesn't take the tap and long press actions of the semantics
// (SPEC §12.10).
//
// Flutter: widgets/tap_region.dart RenderTapRegionSurface
type RenderTapRegionSurface struct {
	rendering.RenderProxyBoxWithHitTestBehavior
	gestures        *gestures.GestureBinding
	registered      []*RenderTapRegion
	groupIDToRegion map[any][]*RenderTapRegion
}

// NewRenderTapRegionSurface creates the surface; the gesture binding is the one
// whose arena a consumed tap joins.
//
// Flutter: widgets/tap_region.dart RenderTapRegionSurface.new
func NewRenderTapRegionSurface(g *gestures.GestureBinding) *RenderTapRegionSurface {
	r := &RenderTapRegionSurface{gestures: g, groupIDToRegion: map[any][]*RenderTapRegion{}}
	rendering.InitRenderProxyBoxWithHitTestBehavior(&r.RenderProxyBoxWithHitTestBehavior, r, rendering.HitTestBehaviorDeferToChild, nil)
	return r
}

func indexOfRegion(list []*RenderTapRegion, r *RenderTapRegion) int {
	for i, x := range list {
		if x == r {
			return i
		}
	}
	return -1
}

// RegisterTapRegion adds a region.
//
// Flutter: widgets/tap_region.dart RenderTapRegionSurface.registerTapRegion
func (r *RenderTapRegionSurface) RegisterTapRegion(region *RenderTapRegion) {
	if indexOfRegion(r.registered, region) >= 0 {
		return
	}
	r.registered = append(r.registered, region)
	if region.groupID != nil {
		r.groupIDToRegion[region.groupID] = append(r.groupIDToRegion[region.groupID], region)
	}
}

// UnregisterTapRegion removes a region.
//
// Flutter: widgets/tap_region.dart RenderTapRegionSurface.unregisterTapRegion
func (r *RenderTapRegionSurface) UnregisterTapRegion(region *RenderTapRegion) {
	if i := indexOfRegion(r.registered, region); i >= 0 {
		r.registered = append(r.registered[:i:i], r.registered[i+1:]...)
	}
	if region.groupID != nil {
		list := r.groupIDToRegion[region.groupID]
		if i := indexOfRegion(list, region); i >= 0 {
			list = append(list[:i:i], list[i+1:]...)
		}
		if len(list) == 0 {
			delete(r.groupIDToRegion, region.groupID)
		} else {
			r.groupIDToRegion[region.groupID] = list
		}
	}
}

// HitTest hits the children and remembers the result for the entry, so that
// HandleEvent can tell which regions were hit.
//
// Flutter: widgets/tap_region.dart RenderTapRegionSurface.hitTest
func (r *RenderTapRegionSurface) HitTest(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	if !r.Size().Contains(position) {
		return false
	}
	self := rendering.Box(r)
	hitTarget := self.HitTestChildren(result, position) || self.HitTestSelf(position)
	if hitTarget {
		result.Add(&tapRegionHitTestEntry{BoxHitTestEntry: *rendering.NewBoxHitTestEntry(self, position), result: result})
	}
	return hitTarget
}

// tapRegionHitTestEntry is the entry of a tap region surface in a hit test result. It keeps the
// result, so that HandleEvent can tell which regions the hit test reached.
//
// Deviation: Flutter keeps the results in an Expando keyed by the entries, which holds them
// weakly. A map in Go would hold them (and the render objects of their paths) until it is
// cleaned, so the entry carries its result: both go together when the hit test result of the
// event is dropped (SPEC §12.10).
//
// Flutter: widgets/tap_region.dart RenderTapRegionSurface._cachedResults
type tapRegionHitTestEntry struct {
	rendering.BoxHitTestEntry
	result *rendering.BoxHitTestResult
}

// classifyRegions splits the registered regions into the ones the hit test path
// reached (or share a group with one that was) and the others.
func (r *RenderTapRegionSurface) classifyRegions(result *rendering.BoxHitTestResult) (inside, outside []*RenderTapRegion) {
	inSet := map[*RenderTapRegion]bool{}
	for _, entry := range result.Path() {
		region, ok := entry.Target().(*RenderTapRegion)
		if !ok || indexOfRegion(r.registered, region) < 0 {
			continue
		}
		if region.groupID == nil {
			inSet[region] = true
			continue
		}
		for _, g := range r.groupIDToRegion[region.groupID] {
			inSet[g] = true
		}
	}
	for _, region := range r.registered {
		if inSet[region] {
			inside = append(inside, region)
		} else {
			outside = append(outside, region)
		}
	}
	return inside, outside
}

// HandleEvent tells the regions about a pointer that went down or up.
//
// Flutter: widgets/tap_region.dart RenderTapRegionSurface.handleEvent
func (r *RenderTapRegionSurface) HandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) {
	r.DebugHandleEvent(event, entry)
	down, isDown := event.(*gestures.PointerDownEvent)
	up, isUp := event.(*gestures.PointerUpEvent)
	if !isDown && !isUp {
		return
	}
	if len(r.registered) == 0 {
		return
	}
	e, ok := entry.(*tapRegionHitTestEntry)
	if !ok {
		return
	}
	result := e.result
	inside, outside := r.classifyRegions(result)
	consumeOutsideTaps := false
	for _, region := range outside {
		if isDown && region.onTapOutside != nil {
			region.onTapOutside(down)
		} else if isUp && region.onTapUpOutside != nil {
			region.onTapUpOutside(up)
		}
		if region.consumeOutsideTaps {
			consumeOutsideTaps = true
		}
	}
	for _, region := range inside {
		if isDown && region.onTapInside != nil {
			region.onTapInside(down)
		} else if isUp && region.onTapUpInside != nil {
			region.onTapUpInside(up)
		}
	}
	if consumeOutsideTaps && isDown && r.gestures != nil {
		r.gestures.GestureArena.Add(down.Pointer(), dummyTapRecognizer{}).Resolve(gestures.GestureAccepted)
	}
}

// dummyTapRecognizer is a member of the gesture arena that does nothing, to win
// the arena of a pointer whose tap outside is consumed.
//
// Flutter: widgets/tap_region.dart _DummyTapRecognizer
type dummyTapRecognizer struct{}

func (dummyTapRecognizer) AcceptGesture(int) {}
func (dummyTapRecognizer) RejectGesture(int) {}

// TapRegionOpts are the optional parameters of NewTapRegion.
//
// Flutter: widgets/tap_region.dart TapRegion.new
type TapRegionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Disabled turns the region off: it doesn't register with the surface
	// (Flutter's enabled is true by default).
	Disabled bool
	// Behavior is how the region behaves during hit testing; the default is
	// HitTestBehaviorDeferToChild.
	Behavior rendering.HitTestBehavior
	// OnTapOutside is called when a pointer goes down outside the region.
	OnTapOutside TapRegionCallback
	// OnTapInside is called when a pointer goes down inside the region.
	OnTapInside TapRegionCallback
	// OnTapUpOutside is called when a pointer goes up outside the region.
	OnTapUpOutside TapRegionUpCallback
	// OnTapUpInside is called when a pointer goes up inside the region.
	OnTapUpInside TapRegionUpCallback
	// GroupID makes the regions with the same ID one: a tap inside any of them is
	// inside all of them.
	GroupID any
	// ConsumeOutsideTaps stops the pointer outside the region from doing
	// anything else, such as a tap on a button.
	ConsumeOutsideTaps bool
	// DebugLabel names the region in debug output.
	DebugLabel string
}

// TapRegion is a widget that tells its callbacks when a pointer goes down or up
// inside or outside of it, and of the regions of its group.
//
// Flutter: widgets/tap_region.dart TapRegion
type TapRegion struct {
	SingleChildRenderObjectWidget
	o TapRegionOpts
}

// NewTapRegion creates the widget.
//
// Flutter: widgets/tap_region.dart TapRegion.new
func NewTapRegion(child Widget, opts ...TapRegionOpts) *TapRegion {
	o := firstOpts(opts)
	if foundation.ReleaseMode {
		o.DebugLabel = ""
	}
	return &TapRegion{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), o: o}
}

// Enabled is whether the region registers with the surface.
//
// Flutter: widgets/tap_region.dart TapRegion.enabled
func (w *TapRegion) Enabled() bool { return !w.o.Disabled }

// GroupID is the group of the region.
//
// Flutter: widgets/tap_region.dart TapRegion.groupId
func (w *TapRegion) GroupID() any { return w.o.GroupID }

// CreateRenderObject creates the region.
//
// Flutter: widgets/tap_region.dart TapRegion.createRenderObject
func (w *TapRegion) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	isCurrent := ModalRouteIsCurrentOf(bc).OrElse(true)
	r := &RenderTapRegion{registry: TapRegionRegistryMaybeOf(bc), enabled: !w.o.Disabled, consumeOutsideTaps: isCurrent && w.o.ConsumeOutsideTaps,
		onTapInside: w.o.OnTapInside, onTapUpInside: w.o.OnTapUpInside, groupID: w.o.GroupID, DebugLabel: w.o.DebugLabel}
	if isCurrent {
		r.onTapOutside, r.onTapUpOutside = w.o.OnTapOutside, w.o.OnTapUpOutside
	}
	rendering.InitRenderProxyBoxWithHitTestBehavior(&r.RenderProxyBoxWithHitTestBehavior, r, w.o.Behavior, nil)
	return r
}

// UpdateRenderObject copies the configuration to the region.
//
// Flutter: widgets/tap_region.dart TapRegion.updateRenderObject
func (w *TapRegion) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*RenderTapRegion)
	isCurrent := ModalRouteIsCurrentOf(bc).OrElse(true)
	r.SetRegistry(TapRegionRegistryMaybeOf(bc))
	r.SetEnabled(!w.o.Disabled)
	r.SetConsumeOutsideTaps(isCurrent && w.o.ConsumeOutsideTaps)
	r.Behavior = w.o.Behavior
	r.SetGroupID(w.o.GroupID)
	r.onTapOutside, r.onTapUpOutside = nil, nil
	if isCurrent {
		r.onTapOutside, r.onTapUpOutside = w.o.OnTapOutside, w.o.OnTapUpOutside
	}
	r.onTapInside, r.onTapUpInside = w.o.OnTapInside, w.o.OnTapUpInside
	r.DebugLabel = w.o.DebugLabel
}

// DebugFillProperties adds the properties of the region.
//
// Flutter: widgets/tap_region.dart TapRegion.debugFillProperties
func (w *TapRegion) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewFlagProperty("enabled", foundation.Some(w.Enabled()), foundation.FlagPropertyOpts{IfFalse: "DISABLED", DefaultValue: foundation.Some(true)}))
	p.Add(foundation.NewStringProperty("debugLabel", optString(w.o.DebugLabel)))
}

// RenderTapRegion is the render object of a TapRegion.
//
// Flutter: widgets/tap_region.dart RenderTapRegion
type RenderTapRegion struct {
	rendering.RenderProxyBoxWithHitTestBehavior
	registry           TapRegionRegistry
	registered         bool
	enabled            bool
	consumeOutsideTaps bool
	groupID            any
	onTapOutside       TapRegionCallback
	onTapInside        TapRegionCallback
	onTapUpOutside     TapRegionUpCallback
	onTapUpInside      TapRegionUpCallback
	// DebugLabel names the region in debug output.
	//
	// Flutter: widgets/tap_region.dart RenderTapRegion.debugLabel
	DebugLabel string
}

// SetEnabled turns the region on or off.
//
// Flutter: widgets/tap_region.dart RenderTapRegion.enabled
func (r *RenderTapRegion) SetEnabled(value bool) {
	if r.enabled != value {
		r.enabled = value
		r.MarkNeedsLayout()
	}
}

// SetConsumeOutsideTaps sets whether the taps outside the region are consumed.
//
// Flutter: widgets/tap_region.dart RenderTapRegion.consumeOutsideTaps
func (r *RenderTapRegion) SetConsumeOutsideTaps(value bool) {
	if r.consumeOutsideTaps != value {
		r.consumeOutsideTaps = value
		r.MarkNeedsLayout()
	}
}

// SetGroupID sets the group of the region.
//
// Flutter: widgets/tap_region.dart RenderTapRegion.groupId
func (r *RenderTapRegion) SetGroupID(value any) {
	if r.groupID != value {
		if r.registered {
			r.registry.UnregisterTapRegion(r)
			r.registered = false
		}
		r.groupID = value
		r.MarkNeedsLayout()
	}
}

// SetRegistry sets the registry the region registers with.
//
// Flutter: widgets/tap_region.dart RenderTapRegion.registry
func (r *RenderTapRegion) SetRegistry(value TapRegionRegistry) {
	if r.registry != value {
		if r.registered {
			r.registry.UnregisterTapRegion(r)
			r.registered = false
		}
		r.registry = value
		r.MarkNeedsLayout()
	}
}

// Layout lays out as a proxy box and registers the region with the surface when
// it is enabled.
//
// Flutter: widgets/tap_region.dart RenderTapRegion.layout
func (r *RenderTapRegion) Layout(constraints rendering.Constraints, opts ...rendering.LayoutOpts) {
	r.RenderProxyBoxWithHitTestBehavior.Layout(constraints, opts...)
	if r.registry == nil {
		return
	}
	if r.registered {
		r.registry.UnregisterTapRegion(r)
	}
	shouldBeRegistered := r.enabled && r.registry != nil
	if shouldBeRegistered {
		r.registry.RegisterTapRegion(r)
	}
	r.registered = shouldBeRegistered
}

// Dispose unregisters the region.
//
// Flutter: widgets/tap_region.dart RenderTapRegion.dispose
func (r *RenderTapRegion) Dispose() {
	if r.registered {
		r.registry.UnregisterTapRegion(r)
	}
	r.RenderProxyBoxWithHitTestBehavior.Dispose()
}

// TextFieldTapRegionGroupID is the group ID of the tap regions of text fields
// (Flutter's EditableText type): a tap in any of them doesn't unfocus the others'
// field, such as the toolbar's buttons.
//
// Flutter: widgets/tap_region.dart TextFieldTapRegion.groupId
type TextFieldTapRegionGroupID struct{}

// TextFieldTapRegionOpts are the optional parameters of NewTextFieldTapRegion.
//
// Flutter: widgets/tap_region.dart TextFieldTapRegion.new
type TextFieldTapRegionOpts struct {
	TapRegionOpts
	// NoGroupID leaves the region out of the group of the text fields; the
	// default group is TextFieldTapRegionGroupID.
	NoGroupID bool
}

// NewTextFieldTapRegion creates a TapRegion that belongs to the group of the
// text fields, so that a tap on a text field's toolbar is not a tap outside of
// the field.
//
// Flutter: widgets/tap_region.dart TextFieldTapRegion.new
func NewTextFieldTapRegion(child Widget, opts ...TextFieldTapRegionOpts) *TapRegion {
	o := firstOpts(opts)
	if o.GroupID == nil && !o.NoGroupID {
		o.GroupID = TextFieldTapRegionGroupID{}
	}
	return NewTapRegion(child, o.TapRegionOpts)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTapRegionRegistryMissing = "TapRegionRegistryMissing"
)
