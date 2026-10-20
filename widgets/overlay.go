// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// OverlayEntryOpts are the optional parameters of NewOverlayEntry.
//
// Flutter: widgets/overlay.dart OverlayEntry.new
type OverlayEntryOpts struct {
	// Opaque is whether the entry hides the entries below it, which are then not
	// built or painted.
	Opaque bool
	// MaintainState is whether the entry keeps its state while an opaque entry is
	// above it.
	MaintainState bool
	// CanSizeOverlay is whether the entry can give a size to an overlay that has
	// unbounded constraints.
	CanSizeOverlay bool
}

// OverlayEntry is a place in an Overlay that can contain a widget. Entries are
// inserted into an Overlay with OverlayState.Insert and removed with Remove.
//
// Flutter: widgets/overlay.dart OverlayEntry
type OverlayEntry struct {
	builder         WidgetBuilder
	opaque          bool
	maintainState   bool
	canSizeOverlay  bool
	notifier        *foundation.ValueNotifier[*overlayEntryWidgetState]
	overlay         *OverlayState
	key             *GlobalKey[*overlayEntryWidgetState]
	disposedByOwner bool
}

// NewOverlayEntry creates an entry that the builder builds.
//
// Flutter: widgets/overlay.dart OverlayEntry.new
func NewOverlayEntry(builder WidgetBuilder, opts ...OverlayEntryOpts) *OverlayEntry {
	o := firstOpts(opts)
	return &OverlayEntry{
		builder: builder, opaque: o.Opaque, maintainState: o.MaintainState, canSizeOverlay: o.CanSizeOverlay,
		notifier: foundation.NewValueNotifier[*overlayEntryWidgetState](nil), key: NewGlobalKey[*overlayEntryWidgetState](),
	}
}

// Builder builds the widget of the entry.
//
// Flutter: widgets/overlay.dart OverlayEntry.builder
func (e *OverlayEntry) Builder() WidgetBuilder { return e.builder }

// Opaque is whether the entry hides the entries below it.
//
// Flutter: widgets/overlay.dart OverlayEntry.opaque
func (e *OverlayEntry) Opaque() bool { return e.opaque }

// SetOpaque changes whether the entry is opaque, and tells the overlay.
//
// Flutter: widgets/overlay.dart OverlayEntry.opaque
func (e *OverlayEntry) SetOpaque(value bool) {
	e.debugCheckNotDisposed()
	if e.opaque == value {
		return
	}
	e.opaque = value
	if e.overlay != nil {
		e.overlay.didChangeEntryOpacity()
	}
}

// MaintainState is whether the entry keeps its state while hidden.
//
// Flutter: widgets/overlay.dart OverlayEntry.maintainState
func (e *OverlayEntry) MaintainState() bool { return e.maintainState }

// SetMaintainState changes whether the entry keeps its state, and tells the
// overlay. In debug builds it panics with the tag OverlayEntryNotInserted when
// the entry is not in an overlay.
//
// Flutter: widgets/overlay.dart OverlayEntry.maintainState
func (e *OverlayEntry) SetMaintainState(value bool) {
	e.debugCheckNotDisposed()
	if e.maintainState == value {
		return
	}
	e.maintainState = value
	if e.overlay == nil {
		if foundation.DebugMode {
			panic(errors.New("OverlayEntryNotInserted", "the maintainState of an OverlayEntry was changed while it is not in an Overlay"))
		}
		return
	}
	e.overlay.didChangeEntryOpacity()
}

// CanSizeOverlay is whether the entry can size an overlay that has unbounded
// constraints.
//
// Flutter: widgets/overlay.dart OverlayEntry.canSizeOverlay
func (e *OverlayEntry) CanSizeOverlay() bool { return e.canSizeOverlay }

// Mounted is whether the entry is in the tree.
//
// Flutter: widgets/overlay.dart OverlayEntry.mounted
func (e *OverlayEntry) Mounted() bool { return e.notifier != nil && e.notifier.Value() != nil }

// AddListener registers a listener that is called when the entry is mounted or
// unmounted.
//
// Flutter: widgets/overlay.dart OverlayEntry.addListener
func (e *OverlayEntry) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	e.debugCheckNotDisposed()
	if e.notifier == nil {
		return foundation.ListenerHandle{}
	}
	return e.notifier.AddListener(listener)
}

// RemoveListener removes a listener.
//
// Flutter: widgets/overlay.dart OverlayEntry.removeListener
func (e *OverlayEntry) RemoveListener(handle foundation.ListenerHandle) {
	if e.notifier != nil {
		e.notifier.RemoveListener(handle)
	}
}

func (e *OverlayEntry) debugCheckNotDisposed() {
	if foundation.DebugMode && e.disposedByOwner {
		panic(errors.New("OverlayEntryDisposed", "the OverlayEntry was used after it was disposed"))
	}
}

// Remove removes the entry from its overlay. In debug builds it panics with the
// tag OverlayEntryNotInserted when the entry is not in an overlay.
//
// Flutter: widgets/overlay.dart OverlayEntry.remove
func (e *OverlayEntry) Remove() {
	if foundation.DebugMode && e.overlay == nil {
		panic(errors.New("OverlayEntryNotInserted", "an OverlayEntry should be removed only once"))
	}
	e.debugCheckNotDisposed()
	overlay := e.overlay
	e.overlay = nil
	if overlay == nil || !overlay.Mounted() {
		return
	}
	for i, entry := range overlay.entries {
		if entry == e {
			overlay.entries = append(overlay.entries[:i], overlay.entries[i+1:]...)
			break
		}
	}
	binding := WidgetsBindingOf(overlay.Context())
	if binding.Scheduler().SchedulerPhase() == scheduler.SchedulerPhasePersistentCallbacks {
		// The tree is locked: the overlay is marked dirty after the frame.
		binding.Scheduler().AddPostFrameCallback(func(time.Duration) { overlay.markDirty() })
	} else {
		overlay.markDirty()
	}
}

// MarkNeedsBuild builds the entry again, for the case that the state that its
// builder reads changed.
//
// Flutter: widgets/overlay.dart OverlayEntry.markNeedsBuild
func (e *OverlayEntry) MarkNeedsBuild() {
	e.debugCheckNotDisposed()
	if state, ok := e.key.CurrentState(); ok {
		state.markNeedsBuild()
	}
}

func (e *OverlayEntry) didUnmount() {
	if e.disposedByOwner && e.notifier != nil {
		e.notifier.Dispose()
		e.notifier = nil
	}
}

// Dispose discards the resources of the entry, which must have been removed
// from its overlay. In debug builds it panics with the tag
// OverlayEntryStillInserted when it has not.
//
// Flutter: widgets/overlay.dart OverlayEntry.dispose
func (e *OverlayEntry) Dispose() {
	e.debugCheckNotDisposed()
	if foundation.DebugMode && e.overlay != nil {
		panic(errors.New("OverlayEntryStillInserted", "an OverlayEntry must first be removed from the Overlay before Dispose is called"))
	}
	e.disposedByOwner = true
	if !e.Mounted() && e.notifier != nil {
		// A mounted entry is disposed in didUnmount, to allow notifications until
		// it is unmounted.
		e.notifier.Dispose()
		e.notifier = nil
	}
}

// String describes the entry.
//
// Flutter: widgets/overlay.dart OverlayEntry.toString
func (e *OverlayEntry) String() string {
	s := foundation.DescribeIdentity(e) + "(opaque: " + strconv.FormatBool(e.opaque) + "; maintainState: " + strconv.FormatBool(e.maintainState) + ")"
	if e.disposedByOwner {
		s += "(DISPOSED)"
	}
	return s
}

// overlayEntryWidget is the widget that builds an OverlayEntry.
//
// Flutter: widgets/overlay.dart _OverlayEntryWidget
type overlayEntryWidget struct {
	StatefulWidget
	entry         *OverlayEntry
	overlayState  *OverlayState
	tickerEnabled bool
}

func newOverlayEntryWidget(entry *OverlayEntry, overlayState *OverlayState, tickerEnabled bool) *overlayEntryWidget {
	return &overlayEntryWidget{NewStatefulWidgetBase(WidgetOpts{Key: entry.key}), entry, overlayState, tickerEnabled}
}

func (w *overlayEntryWidget) CreateState() State { return &overlayEntryWidgetState{} }

// overlayEntryWidgetState is the state of an overlayEntryWidget.
//
// Flutter: widgets/overlay.dart _OverlayEntryWidgetState
type overlayEntryWidgetState struct {
	StateOf[*overlayEntryWidget]
	theater *renderTheater
	// firstSibling and lastSibling are the ends of the list of the theater children
	// whose paint order is sorted by their z-order index. The children that
	// OverlayPortals add to it are shown above the OverlayEntry of this widget; the
	// ones with larger z-order indexes (those that were shown most recently) are
	// painted last. The list is only read by the theater, as part of its child model:
	// adding or removing a child does not rebuild this widget. It would ideally be in
	// a render object, but there may be none between the theater and the subtree of
	// the entry.
	firstSibling, lastSibling *overlayEntryLocation
}

func (s *overlayEntryWidgetState) InitState() {
	s.Widget().entry.notifier.SetValue(s)
	s.theater, _ = FindAncestorRenderObjectOfType[*renderTheater](s.Context())
}

func (s *overlayEntryWidgetState) DidUpdateWidget(oldWidget *overlayEntryWidget) {
	// OverlayState's Build always returns a theater, so it is safe to assume
	// that state equality implies render object equality.
	if oldWidget.overlayState != s.Widget().overlayState {
		s.theater, _ = FindAncestorRenderObjectOfType[*renderTheater](s.Context())
	}
}

func (s *overlayEntryWidgetState) Dispose() {
	s.firstSibling, s.lastSibling = nil, nil
	entry := s.Widget().entry
	if entry.notifier != nil {
		entry.notifier.SetValue(nil)
	}
	entry.didUnmount()
}

func (s *overlayEntryWidgetState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewTickerMode(w.tickerEnabled, &renderTheaterMarker{
		InheritedWidget: NewInheritedWidgetBase(NewBuilder(w.entry.builder)), theater: s.theater, overlayEntryWidgetState: s,
	})
}

func (s *overlayEntryWidgetState) markNeedsBuild() {
	s.SetState(func() { /* the state that changed is in the builder */ })
}

// renderTheaterMarker tells the widgets below an overlay entry which theater and
// which overlay entry they are in.
//
// Flutter: widgets/overlay.dart _RenderTheaterMarker
type renderTheaterMarker struct {
	InheritedWidget
	theater                 *renderTheater
	overlayEntryWidgetState *overlayEntryWidgetState
}

func (w *renderTheaterMarker) UpdateShouldNotify(old Widget) bool {
	o := old.(*renderTheaterMarker)
	return o.theater != w.theater || o.overlayEntryWidgetState != w.overlayEntryWidgetState
}

// renderTheaterMarkerMaybeOf is the closest marker, or the one of the root
// overlay when targetRootOverlay. It does not look past a LookupBoundary. The
// context depends on the marker when createDependency.
//
// Flutter: widgets/overlay.dart _RenderTheaterMarker.maybeOf
func renderTheaterMarkerMaybeOf(bc BuildContext, targetRootOverlay, createDependency bool) *renderTheaterMarker {
	if targetRootOverlay {
		ancestor := rootRenderTheaterMarkerOf(LookupBoundaryGetElementForInheritedWidgetOfExactType[*renderTheaterMarker](bc))
		if ancestor == nil {
			return nil
		}
		if createDependency {
			return bc.DependOnInheritedElement(ancestor, nil).(*renderTheaterMarker)
		}
		return ancestor.AsElement().widget.(*renderTheaterMarker)
	}
	if createDependency {
		marker, _ := LookupBoundaryDependOnInheritedWidgetOfExactType[*renderTheaterMarker](bc)
		return marker
	}
	marker, _ := LookupBoundaryGetInheritedWidgetOfExactType[*renderTheaterMarker](bc)
	return marker
}

// renderTheaterMarkerOf is renderTheaterMarkerMaybeOf that panics with the tag
// OverlayMissing in debug builds when there is none.
//
// Flutter: widgets/overlay.dart _RenderTheaterMarker.of
func renderTheaterMarkerOf(bc BuildContext, targetRootOverlay bool) *renderTheaterMarker {
	marker := renderTheaterMarkerMaybeOf(bc, targetRootOverlay, true)
	if marker == nil {
		panic(errors.New("OverlayMissing", "No Overlay widget found. "+foundation.RuntimeTypeName(bc.Widget())+
			" widgets require an Overlay widget ancestor; an overlay lets widgets float on top of other widget children; to introduce an Overlay widget, you can either directly include one, or use a widget that contains an Overlay itself, such as a Navigator, WidgetApp, MaterialApp, or CupertinoApp",
			"context", bc.(Element).String()))
	}
	return marker
}

func rootRenderTheaterMarkerOf(marker InheritedElementSelf) InheritedElementSelf {
	if marker == nil {
		return nil
	}
	var ancestor InheritedElementSelf
	marker.VisitAncestorElements(func(element Element) bool {
		ancestor = LookupBoundaryGetElementForInheritedWidgetOfExactType[*renderTheaterMarker](element)
		return false
	})
	if ancestor == nil {
		return marker
	}
	return rootRenderTheaterMarkerOf(ancestor)
}

// OverlayOpts are the optional parameters of NewOverlay.
//
// Flutter: widgets/overlay.dart Overlay.new
type OverlayOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// InitialEntries are the entries the overlay starts with, bottom first.
	InitialEntries []*OverlayEntry
	// ClipBehavior is how the overlay clips its entries; the default is
	// ClipHardEdge.
	ClipBehavior ui.Clip
	// AlwaysSizeToContent is whether the overlay always sizes itself to the
	// entry that can size it, instead of to the biggest size it is given.
	AlwaysSizeToContent bool
}

// Overlay is a stack of entries that can be managed independently. It lets
// widgets float visual elements on top of other widgets.
//
// Flutter: widgets/overlay.dart Overlay
type Overlay struct {
	StatefulWidget
	initialEntries      []*OverlayEntry
	clipBehavior        ui.Clip
	alwaysSizeToContent bool
}

// NewOverlay creates an overlay.
//
// Flutter: widgets/overlay.dart Overlay.new
func NewOverlay(opts ...OverlayOpts) *Overlay {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &Overlay{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), o.InitialEntries, o.ClipBehavior, o.AlwaysSizeToContent}
}

// InitialEntries are the entries the overlay starts with.
//
// Flutter: widgets/overlay.dart Overlay.initialEntries
func (w *Overlay) InitialEntries() []*OverlayEntry { return w.initialEntries }

// ClipBehavior is how the overlay clips its entries.
//
// Flutter: widgets/overlay.dart Overlay.clipBehavior
func (w *Overlay) ClipBehavior() ui.Clip { return w.clipBehavior }

// AlwaysSizeToContent is whether the overlay always sizes itself to its content.
//
// Flutter: widgets/overlay.dart Overlay.alwaysSizeToContent
func (w *Overlay) AlwaysSizeToContent() bool { return w.alwaysSizeToContent }

// CreateState creates the state.
//
// Flutter: widgets/overlay.dart Overlay.createState
func (w *Overlay) CreateState() State { return &OverlayState{} }

// OverlayWrapOpts are the optional parameters of NewOverlayWrap.
//
// Flutter: widgets/overlay.dart Overlay.wrap
type OverlayWrapOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ClipBehavior is how the overlay clips; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// AlwaysSizeToContent is whether the overlay sizes itself to the child.
	AlwaysSizeToContent bool
}

// NewOverlayWrap creates an overlay with one opaque entry that contains the
// child and can size the overlay, so that the overlay is as big as the child.
//
// Flutter: widgets/overlay.dart Overlay.wrap
func NewOverlayWrap(child Widget, opts ...OverlayWrapOpts) Widget {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &wrappingOverlay{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), o.ClipBehavior, o.AlwaysSizeToContent, child}
}

// OverlayOfOpts are the optional parameters of OverlayOf and OverlayMaybeOf.
//
// Flutter: widgets/overlay.dart Overlay.of
type OverlayOfOpts struct {
	// RootOverlay is whether to look for the furthest overlay.
	RootOverlay bool
	// DebugRequiredFor is the widget that requires the overlay, which is named in
	// the error.
	DebugRequiredFor Widget
}

// OverlayMaybeOf is the state of the closest Overlay above the context, and
// false if there is none.
//
// Flutter: widgets/overlay.dart Overlay.maybeOf
func OverlayMaybeOf(bc BuildContext, opts ...OverlayOfOpts) (*OverlayState, bool) {
	marker := renderTheaterMarkerMaybeOf(bc, firstOpts(opts).RootOverlay, false)
	if marker == nil {
		return nil, false
	}
	return marker.overlayEntryWidgetState.Widget().overlayState, true
}

// OverlayOf is the state of the closest Overlay above the context. In debug
// builds it panics with the tag OverlayMissing when there is none.
//
// Flutter: widgets/overlay.dart Overlay.of
func OverlayOf(bc BuildContext, opts ...OverlayOfOpts) *OverlayState {
	o := firstOpts(opts)
	result, ok := OverlayMaybeOf(bc, o)
	if !ok && foundation.DebugMode {
		required := "Some"
		if o.DebugRequiredFor != nil {
			required = foundation.RuntimeTypeName(o.DebugRequiredFor)
		}
		summary := "No Overlay widget found."
		if LookupBoundaryDebugIsHidingAncestorStateOfType[*OverlayState](bc) {
			summary = "No Overlay widget found within the closest LookupBoundary. There is an ancestor Overlay widget, but it is hidden by a LookupBoundary."
		}
		panic(errors.New("OverlayMissing", summary+" "+required+" widgets require an Overlay widget ancestor for correct operation. "+
			"The most common way to add an Overlay to an application is to include a MaterialApp, CupertinoApp or Navigator widget in the runApp() call.",
			"context", bc.(Element).String()))
	}
	return result
}

// OverlayState is the state of an Overlay: it manages the entries.
//
// Flutter: widgets/overlay.dart OverlayState
type OverlayState struct {
	StateOf[*Overlay]
	TickerProvider
	entries []*OverlayEntry
}

// InitState inserts the initial entries.
func (s *OverlayState) InitState() { s.InsertAll(s.Widget().initialEntries) }

func (s *OverlayState) indexOf(entry *OverlayEntry) int {
	for i, e := range s.entries {
		if e == entry {
			return i
		}
	}
	return -1
}

func (s *OverlayState) insertionIndex(below, above *OverlayEntry) int {
	if below != nil {
		return s.indexOf(below)
	}
	if above != nil {
		return s.indexOf(above) + 1
	}
	return len(s.entries)
}

func (s *OverlayState) debugCanInsertEntry(entry *OverlayEntry) {
	if !s.Mounted() {
		panic(errors.New("OverlayInsertDisposed", "attempted to insert an OverlayEntry to an already disposed Overlay",
			"entry", entry.String(), "overlay", foundation.DescribeIdentity(s)))
	}
	if s.indexOf(entry) != -1 {
		panic(errors.New("OverlayEntryAlreadyInserted", "the specified entry is already present in the target Overlay; consider calling Remove on the OverlayEntry before inserting it to a different Overlay, or switching to the OverlayPortal API to avoid manual OverlayEntry management",
			"entry", entry.String(), "overlay", foundation.DescribeIdentity(s)))
	}
	if entry.overlay != nil {
		panic(errors.New("OverlayEntryInOtherOverlay", "the specified entry is already present in a different Overlay; consider calling Remove on the OverlayEntry before inserting it to a different Overlay, or switching to the OverlayPortal API to avoid manual OverlayEntry management",
			"entry", entry.String(), "overlay", foundation.DescribeIdentity(s)))
	}
}

func (s *OverlayState) debugVerifyInsertPosition(above, below *OverlayEntry, newEntries []*OverlayEntry) {
	contains := func(list []*OverlayEntry, e *OverlayEntry) bool {
		for _, x := range list {
			if x == e {
				return true
			}
		}
		return false
	}
	if above != nil && below != nil {
		panic(errors.New("OverlayInsertPositionInvalid", "only one of above and below may be specified"))
	}
	if above != nil && (above.overlay != s || s.indexOf(above) == -1 || (newEntries != nil && !contains(newEntries, above))) {
		panic(errors.New("OverlayInsertPositionInvalid", "the provided entry used for above must be present in the Overlay (and in the new entries)"))
	}
	if below != nil && (below.overlay != s || s.indexOf(below) == -1 || (newEntries != nil && !contains(newEntries, below))) {
		panic(errors.New("OverlayInsertPositionInvalid", "the provided entry used for below must be present in the Overlay (and in the new entries)"))
	}
}

// OverlayInsertOpts say where an entry is inserted: just below or just above an
// entry that is in the overlay; the default is above all.
//
// Flutter: widgets/overlay.dart OverlayState.insert
type OverlayInsertOpts struct {
	// Below is the entry the new one goes below.
	Below *OverlayEntry
	// Above is the entry the new one goes above.
	Above *OverlayEntry
}

// Insert adds the entry to the overlay, above all the entries by default.
//
// Flutter: widgets/overlay.dart OverlayState.insert
func (s *OverlayState) Insert(entry *OverlayEntry, opts ...OverlayInsertOpts) {
	o := firstOpts(opts)
	if foundation.DebugMode {
		s.debugVerifyInsertPosition(o.Above, o.Below, nil)
		s.debugCanInsertEntry(entry)
	}
	entry.overlay = s
	s.SetState(func() {
		index := s.insertionIndex(o.Below, o.Above)
		s.entries = append(s.entries[:index], append([]*OverlayEntry{entry}, s.entries[index:]...)...)
	})
}

// InsertAll adds the entries to the overlay, above all the entries by default,
// in the order that they are given.
//
// Flutter: widgets/overlay.dart OverlayState.insertAll
func (s *OverlayState) InsertAll(entries []*OverlayEntry, opts ...OverlayInsertOpts) {
	o := firstOpts(opts)
	if foundation.DebugMode {
		s.debugVerifyInsertPosition(o.Above, o.Below, nil)
		for _, entry := range entries {
			s.debugCanInsertEntry(entry)
		}
	}
	if len(entries) == 0 {
		return
	}
	for _, entry := range entries {
		entry.overlay = s
	}
	s.SetState(func() {
		index := s.insertionIndex(o.Below, o.Above)
		s.entries = append(s.entries[:index], append(append([]*OverlayEntry{}, entries...), s.entries[index:]...)...)
	})
}

// Rearrange removes the given entries from the overlay and adds them back in the
// given order; the entries that are not given go below or above them.
//
// Flutter: widgets/overlay.dart OverlayState.rearrange
func (s *OverlayState) Rearrange(newEntries []*OverlayEntry, opts ...OverlayInsertOpts) {
	o := firstOpts(opts)
	if foundation.DebugMode {
		s.debugVerifyInsertPosition(o.Above, o.Below, newEntries)
		seen := map[*OverlayEntry]bool{}
		for _, entry := range newEntries {
			if entry.overlay != nil && entry.overlay != s {
				panic(errors.New("OverlayEntryInOtherOverlay", "one or more of the specified entries are already present in another Overlay"))
			}
			if seen[entry] {
				panic(errors.New("OverlayEntryDuplicate", "one or more of the specified entries are specified multiple times"))
			}
			seen[entry] = true
		}
	}
	if len(newEntries) == 0 {
		return
	}
	if len(s.entries) == len(newEntries) {
		same := true
		for i := range newEntries {
			if s.entries[i] != newEntries[i] {
				same = false
				break
			}
		}
		if same {
			return
		}
	}
	given := map[*OverlayEntry]bool{}
	for _, entry := range newEntries {
		given[entry] = true
		if entry.overlay == nil {
			entry.overlay = s
		}
	}
	var old []*OverlayEntry
	for _, entry := range s.entries {
		if !given[entry] {
			old = append(old, entry)
		}
	}
	s.SetState(func() {
		s.entries = append([]*OverlayEntry{}, newEntries...)
		index := s.insertionIndex(o.Below, o.Above)
		s.entries = append(s.entries[:index], append(old, s.entries[index:]...)...)
	})
}

func (s *OverlayState) markDirty() {
	if s.Mounted() {
		s.SetState(func() {})
	}
}

// DebugIsVisible is whether the entry is visible: no opaque entry is above it.
// In debug builds only; release builds report false.
//
// Flutter: widgets/overlay.dart OverlayState.debugIsVisible
func (s *OverlayState) DebugIsVisible(entry *OverlayEntry) bool {
	result := false
	if foundation.DebugMode {
		if s.indexOf(entry) == -1 {
			panic(errors.New("OverlayEntryNotInserted", "the entry is not in the overlay"))
		}
		for i := len(s.entries) - 1; i > 0; i-- {
			candidate := s.entries[i]
			if candidate == entry {
				result = true
				break
			}
			if candidate.opaque {
				break
			}
		}
	}
	return result
}

func (s *OverlayState) didChangeEntryOpacity() {
	// The opacity of the entry is used in Build, so the state changed.
	s.SetState(func() {})
}

// Build builds the theater of the entries that are on stage, and the ones that
// keep their state off stage.
//
// Flutter: widgets/overlay.dart OverlayState.build
func (s *OverlayState) Build(BuildContext) Widget {
	// This list is filled backwards and then reversed before it is added to the
	// tree.
	var children []Widget
	onstage := true
	onstageCount := 0
	for i := len(s.entries) - 1; i >= 0; i-- {
		entry := s.entries[i]
		if onstage {
			onstageCount++
			children = append(children, newOverlayEntryWidget(entry, s, true))
			if entry.opaque {
				onstage = false
			}
		} else if entry.maintainState {
			children = append(children, newOverlayEntryWidget(entry, s, false))
		}
	}
	for i, j := 0, len(children)-1; i < j; i, j = i+1, j-1 {
		children[i], children[j] = children[j], children[i]
	}
	w := s.Widget()
	return newTheater(len(children)-onstageCount, w.clipBehavior, w.alwaysSizeToContent, children)
}

// wrappingOverlay is the widget of NewOverlayWrap.
//
// Flutter: widgets/overlay.dart _WrappingOverlay
type wrappingOverlay struct {
	StatefulWidget
	clipBehavior        ui.Clip
	alwaysSizeToContent bool
	child               Widget
}

func (w *wrappingOverlay) CreateState() State { return &wrappingOverlayState{} }

type wrappingOverlayState struct {
	StateOf[*wrappingOverlay]
	entry *OverlayEntry
}

func (s *wrappingOverlayState) InitState() {
	s.entry = NewOverlayEntry(func(BuildContext) Widget { return s.Widget().child }, OverlayEntryOpts{CanSizeOverlay: true, Opaque: true})
}

func (s *wrappingOverlayState) DidUpdateWidget(*wrappingOverlay) { s.entry.MarkNeedsBuild() }

func (s *wrappingOverlayState) Dispose() {
	s.entry.Remove()
	s.entry.Dispose()
}

func (s *wrappingOverlayState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewOverlay(OverlayOpts{ClipBehavior: w.clipBehavior, AlwaysSizeToContent: w.alwaysSizeToContent, InitialEntries: []*OverlayEntry{s.entry}})
}

// DebugCheckHasOverlay panics with the tag OverlayMissing in debug builds if
// there is no Overlay widget above the context (within the closest
// LookupBoundary), and returns true.
//
// Flutter: widgets/debug.dart debugCheckHasOverlay
func DebugCheckHasOverlay(bc BuildContext) bool {
	if foundation.DebugMode {
		if _, ok := LookupBoundaryFindAncestorWidgetOfExactType[*Overlay](bc); !ok {
			hiddenByBoundary := LookupBoundaryDebugIsHidingAncestorWidgetOfExactType[*Overlay](bc)
			summary, within := "No Overlay widget found.", ""
			if hiddenByBoundary {
				summary = "No Overlay widget found within the closest LookupBoundary. There is an ancestor Overlay widget, but it is hidden by a LookupBoundary."
				within = " within the closest LookupBoundary"
			}
			panic(errors.New("OverlayMissing", summary+" "+foundation.RuntimeTypeName(bc.Widget())+
				" widgets require an Overlay widget ancestor"+within+"; an overlay lets widgets float on top of other widget children; to introduce an Overlay widget, you can either directly include one, or use a widget that contains an Overlay itself, such as a Navigator, WidgetApp, MaterialApp, or CupertinoApp",
				"context", bc.(Element).String()))
		}
	}
	return true
}
