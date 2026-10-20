// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/services"
)

// LockState is the requirement of a SingleActivator for a lock key such as Num
// Lock.
//
// Flutter: widgets/shortcuts.dart LockState
type LockState int

// The values of LockState follow Flutter's declaration order.
const (
	// LockStateUnspecified is the zero value; it means LockStateIgnored.
	//
	// Flutter: no counterpart
	LockStateUnspecified LockState = iota
	// LockStateIgnored means that the lock key is not considered.
	//
	// Flutter: widgets/shortcuts.dart LockState.ignored
	LockStateIgnored
	// LockStateLocked requires the lock mode to be on.
	//
	// Flutter: widgets/shortcuts.dart LockState.locked
	LockStateLocked
	// LockStateUnlocked requires the lock mode to be off.
	//
	// Flutter: widgets/shortcuts.dart LockState.unlocked
	LockStateUnlocked
)

// String is the name of the state, like Dart's enum toString.
//
// Flutter: widgets/shortcuts.dart LockState
func (s LockState) String() string {
	switch s {
	case LockStateLocked:
		return "LockState.locked"
	case LockStateUnlocked:
		return "LockState.unlocked"
	}
	return "LockState.ignored"
}

var (
	controlSynonyms = services.ExpandSynonyms(services.NewKeySet(services.LogicalKeyboardKeyControl))
	shiftSynonyms   = services.ExpandSynonyms(services.NewKeySet(services.LogicalKeyboardKeyShift))
	altSynonyms     = services.ExpandSynonyms(services.NewKeySet(services.LogicalKeyboardKeyAlt))
	metaSynonyms    = services.ExpandSynonyms(services.NewKeySet(services.LogicalKeyboardKeyMeta))
)

func intersects(a, b services.KeySet[services.LogicalKeyboardKey]) bool {
	for k := range a {
		if b.Contains(k) {
			return true
		}
	}
	return false
}

// ShortcutActivator is the interface for defining the key combination that
// triggers a shortcut: SingleActivator, CharacterActivator and LogicalKeySet are
// the ones the framework has. An activator is used as the key of a ShortcutMap,
// so it must be comparable (a struct without slices, or a pointer).
//
// Flutter: widgets/shortcuts.dart ShortcutActivator
type ShortcutActivator interface {
	// Triggers are the keys that can trigger the shortcut; the shortcuts manager
	// asks only the activators whose triggers contain the key of an event. A nil
	// result means the activator can be triggered by any key.
	//
	// Flutter: widgets/shortcuts.dart ShortcutActivator.triggers
	Triggers() []services.LogicalKeyboardKey
	// Accepts is whether the key event, with the keyboard state after it, triggers
	// the shortcut.
	//
	// Flutter: widgets/shortcuts.dart ShortcutActivator.accepts
	Accepts(event services.KeyEvent, state *services.HardwareKeyboard) bool
	// DebugDescribeKeys is the keys of the activator as text, such as
	// "Control + Shift + Key C".
	//
	// Flutter: widgets/shortcuts.dart ShortcutActivator.debugDescribeKeys
	DebugDescribeKeys() string
}

// LogicalKeySet is a set of keys that must all be pressed, in any order, for the
// shortcut to trigger: the last one pressed triggers it. A key such as Shift is
// satisfied by either of its sides. It is a comparable value: two sets with the
// same keys are the same map key.
//
// Flutter: widgets/shortcuts.dart LogicalKeySet
type LogicalKeySet struct {
	// encoded is the sorted keys, eight bytes each; it keeps the type comparable.
	encoded string
}

// NewLogicalKeySet creates a set of up to four keys. In debug builds it panics
// with the tag KeySetDuplicateKey when a key appears twice.
//
// Flutter: widgets/shortcuts.dart LogicalKeySet.new
func NewLogicalKeySet(key1 services.LogicalKeyboardKey, more ...services.LogicalKeyboardKey) LogicalKeySet {
	keys := append([]services.LogicalKeyboardKey{key1}, more...)
	set := NewLogicalKeySetFromSet(services.NewKeySet(keys...))
	if foundation.DebugMode && len(set.Keys()) != len(keys) {
		panic(errors.New("KeySetDuplicateKey", "two or more provided keys are identical; each key must appear only once"))
	}
	return set
}

// NewLogicalKeySetFromSet creates a set of the keys. In debug builds it panics
// with the tag KeySetEmpty when there are none.
//
// Flutter: widgets/shortcuts.dart LogicalKeySet.fromSet
func NewLogicalKeySetFromSet(keys services.KeySet[services.LogicalKeyboardKey]) LogicalKeySet {
	if foundation.DebugMode && len(keys) == 0 {
		panic(errors.New("KeySetEmpty", "a key set needs at least one key"))
	}
	sorted := make([]uint64, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, uint64(k))
	}
	slices.Sort(sorted)
	var b strings.Builder
	buf := make([]byte, 8)
	for _, k := range sorted {
		binary.BigEndian.PutUint64(buf, k)
		b.Write(buf)
	}
	return LogicalKeySet{b.String()}
}

// Keys are the keys of the set.
//
// Flutter: widgets/shortcuts.dart KeySet.keys
func (s LogicalKeySet) Keys() services.KeySet[services.LogicalKeyboardKey] {
	out := services.KeySet[services.LogicalKeyboardKey]{}
	for i := 0; i+8 <= len(s.encoded); i += 8 {
		out[services.LogicalKeyboardKey(binary.BigEndian.Uint64([]byte(s.encoded[i:i+8])))] = struct{}{}
	}
	return out
}

var unmapSynonyms = map[services.LogicalKeyboardKey][]services.LogicalKeyboardKey{
	services.LogicalKeyboardKeyControl: {services.LogicalKeyboardKeyControlLeft, services.LogicalKeyboardKeyControlRight},
	services.LogicalKeyboardKeyShift:   {services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftRight},
	services.LogicalKeyboardKeyAlt:     {services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyAltRight},
	services.LogicalKeyboardKeyMeta:    {services.LogicalKeyboardKeyMetaLeft, services.LogicalKeyboardKeyMetaRight},
}

var logicalKeySetModifiers = services.NewKeySet(
	services.LogicalKeyboardKeyAlt, services.LogicalKeyboardKeyControl, services.LogicalKeyboardKeyMeta, services.LogicalKeyboardKeyShift)

// Triggers are the keys of the set, with each pseudo-key such as Shift replaced
// by its two sides.
//
// Flutter: widgets/shortcuts.dart LogicalKeySet.triggers
func (s LogicalKeySet) Triggers() []services.LogicalKeyboardKey {
	var out []services.LogicalKeyboardKey
	seen := map[services.LogicalKeyboardKey]bool{}
	for key := range s.Keys() {
		list, ok := unmapSynonyms[key]
		if !ok {
			list = []services.LogicalKeyboardKey{key}
		}
		for _, k := range list {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	slices.Sort(out)
	return out
}

func (s LogicalKeySet) checkKeyRequirements(pressed services.KeySet[services.LogicalKeyboardKey]) bool {
	collapsedRequired := services.CollapseSynonyms(s.Keys())
	collapsedPressed := services.CollapseSynonyms(pressed)
	if len(collapsedRequired) != len(collapsedPressed) {
		return false
	}
	for k := range collapsedRequired {
		if !collapsedPressed.Contains(k) {
			return false
		}
	}
	return true
}

// Accepts is whether the event is a press of one of the triggers while exactly
// the keys of the set are pressed.
//
// Flutter: widgets/shortcuts.dart LogicalKeySet.accepts
func (s LogicalKeySet) Accepts(event services.KeyEvent, state *services.HardwareKeyboard) bool {
	switch event.(type) {
	case *services.KeyDownEvent, *services.KeyRepeatEvent:
	default:
		return false
	}
	return slices.Contains(s.Triggers(), event.LogicalKey()) && s.checkKeyRequirements(state.LogicalKeysPressed())
}

// DebugDescribeKeys is the keys, the modifiers first, joined by " + ".
//
// Flutter: widgets/shortcuts.dart LogicalKeySet.debugDescribeKeys
func (s LogicalKeySet) DebugDescribeKeys() string {
	keys := make([]services.LogicalKeyboardKey, 0, len(s.Keys()))
	for k := range s.Keys() {
		keys = append(keys, k)
	}
	isModifier := func(k services.LogicalKeyboardKey) bool {
		return len(k.Synonyms()) > 0 || logicalKeySetModifiers.Contains(k)
	}
	slices.SortFunc(keys, func(a, b services.LogicalKeyboardKey) int {
		aMod, bMod := isModifier(a), isModifier(b)
		switch {
		case aMod && !bMod:
			return -1
		case bMod && !aMod:
			return 1
		}
		return strings.Compare(a.DebugName(), b.DebugName())
	})
	names := make([]string, len(keys))
	for i, k := range keys {
		names[i] = k.DebugName()
	}
	return strings.Join(names, " + ")
}

// DebugFillProperties adds the keys.
//
// Flutter: widgets/shortcuts.dart LogicalKeySet.debugFillProperties
func (s LogicalKeySet) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("keys", s.Keys(), foundation.DiagnosticsPropertyOpts{Description: s.DebugDescribeKeys()}))
}

// SingleActivator triggers when a key is pressed with the given modifiers: each
// modifier must be pressed if it is set, and must not be pressed if it is not.
// The trigger must not be a modifier.
//
// Flutter: widgets/shortcuts.dart SingleActivator
type SingleActivator struct {
	trigger        services.LogicalKeyboardKey
	control, shift bool
	alt, meta      bool
	numLock        LockState
	excludeRepeats bool
}

// SingleActivatorOpts are the optional parameters of NewSingleActivator.
//
// Flutter: widgets/shortcuts.dart SingleActivator.new
type SingleActivatorOpts struct {
	// Control requires the Control key to be pressed.
	Control bool
	// Shift requires the Shift key to be pressed.
	Shift bool
	// Alt requires the Alt key to be pressed.
	Alt bool
	// Meta requires the Meta key to be pressed.
	Meta bool
	// NumLock requires the Num Lock mode to be on or off; the default ignores it.
	NumLock LockState
	// ExcludeRepeats makes the activator reject the repeats of a key (Flutter's
	// includeRepeats is true by default).
	ExcludeRepeats bool
}

// NewSingleActivator creates an activator for the key. It is a value, so that two
// activators with the same fields are the same map key. In debug builds it panics
// with the tag SingleActivatorTriggerIsModifier when the trigger is a modifier.
//
// Flutter: widgets/shortcuts.dart SingleActivator.new
func NewSingleActivator(trigger services.LogicalKeyboardKey, opts ...SingleActivatorOpts) SingleActivator {
	o := firstOpts(opts)
	if foundation.DebugMode {
		switch trigger {
		case services.LogicalKeyboardKeyControl, services.LogicalKeyboardKeyControlLeft, services.LogicalKeyboardKeyControlRight,
			services.LogicalKeyboardKeyShift, services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftRight,
			services.LogicalKeyboardKeyAlt, services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyAltRight,
			services.LogicalKeyboardKeyMeta, services.LogicalKeyboardKeyMetaLeft, services.LogicalKeyboardKeyMetaRight:
			panic(errors.New("SingleActivatorTriggerIsModifier", "the trigger of a SingleActivator must not be a modifier key; use the control, shift, alt and meta options", "trigger", trigger.String()))
		}
	}
	return SingleActivator{trigger, o.Control, o.Shift, o.Alt, o.Meta, o.NumLock, o.ExcludeRepeats}
}

// Trigger is the key that triggers the shortcut.
//
// Flutter: widgets/shortcuts.dart SingleActivator.trigger
func (a SingleActivator) Trigger() services.LogicalKeyboardKey { return a.trigger }

// Control is whether the Control key must be pressed.
//
// Flutter: widgets/shortcuts.dart SingleActivator.control
func (a SingleActivator) Control() bool { return a.control }

// Shift is whether the Shift key must be pressed.
//
// Flutter: widgets/shortcuts.dart SingleActivator.shift
func (a SingleActivator) Shift() bool { return a.shift }

// Alt is whether the Alt key must be pressed.
//
// Flutter: widgets/shortcuts.dart SingleActivator.alt
func (a SingleActivator) Alt() bool { return a.alt }

// Meta is whether the Meta key must be pressed.
//
// Flutter: widgets/shortcuts.dart SingleActivator.meta
func (a SingleActivator) Meta() bool { return a.meta }

// NumLock is the requirement for Num Lock.
//
// Flutter: widgets/shortcuts.dart SingleActivator.numLock
func (a SingleActivator) NumLock() LockState {
	if a.numLock == LockStateUnspecified {
		return LockStateIgnored
	}
	return a.numLock
}

// IncludeRepeats is whether the repeats of the key trigger the shortcut too.
//
// Flutter: widgets/shortcuts.dart SingleActivator.includeRepeats
func (a SingleActivator) IncludeRepeats() bool { return !a.excludeRepeats }

// Triggers is the one trigger key.
//
// Flutter: widgets/shortcuts.dart SingleActivator.triggers
func (a SingleActivator) Triggers() []services.LogicalKeyboardKey {
	return []services.LogicalKeyboardKey{a.trigger}
}

func (a SingleActivator) shouldAcceptModifiers(pressed services.KeySet[services.LogicalKeyboardKey]) bool {
	return a.control == intersects(pressed, controlSynonyms) &&
		a.shift == intersects(pressed, shiftSynonyms) &&
		a.alt == intersects(pressed, altSynonyms) &&
		a.meta == intersects(pressed, metaSynonyms)
}

func (a SingleActivator) shouldAcceptNumLock(state *services.HardwareKeyboard) bool {
	switch a.NumLock() {
	case LockStateLocked:
		return state.LockModesEnabled().Contains(services.KeyboardLockModeNumLock)
	case LockStateUnlocked:
		return !state.LockModesEnabled().Contains(services.KeyboardLockModeNumLock)
	}
	return true
}

// Accepts is whether the event presses the trigger while the modifiers are as
// required.
//
// Flutter: widgets/shortcuts.dart SingleActivator.accepts
func (a SingleActivator) Accepts(event services.KeyEvent, state *services.HardwareKeyboard) bool {
	switch event.(type) {
	case *services.KeyDownEvent:
	case *services.KeyRepeatEvent:
		if a.excludeRepeats {
			return false
		}
	default:
		return false
	}
	return event.LogicalKey() == a.trigger && a.shouldAcceptModifiers(state.LogicalKeysPressed()) && a.shouldAcceptNumLock(state)
}

// DebugDescribeKeys is the modifiers and the trigger, joined by " + ".
//
// Flutter: widgets/shortcuts.dart SingleActivator.debugDescribeKeys
func (a SingleActivator) DebugDescribeKeys() string {
	var keys []string
	if a.control {
		keys = append(keys, "Control")
	}
	if a.alt {
		keys = append(keys, "Alt")
	}
	if a.meta {
		keys = append(keys, "Meta")
	}
	if a.shift {
		keys = append(keys, "Shift")
	}
	name := a.trigger.DebugName()
	if name == "" {
		name = a.trigger.String()
	}
	return strings.Join(append(keys, name), " + ")
}

// DebugFillProperties adds the keys and whether repeats are excluded.
//
// Flutter: widgets/shortcuts.dart SingleActivator.debugFillProperties
func (a SingleActivator) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewMessageProperty("keys", a.DebugDescribeKeys()))
	p.Add(foundation.NewFlagProperty("includeRepeats", foundation.Some(a.IncludeRepeats()), foundation.FlagPropertyOpts{IfFalse: "excluding repeats"}))
}

// CharacterActivator triggers when a key event produces the character, with the
// given modifiers. Because the character depends on the layout, it is the
// activator for shortcuts like "?" or "+".
//
// Flutter: widgets/shortcuts.dart CharacterActivator
type CharacterActivator struct {
	character      string
	alt, control   bool
	meta           bool
	excludeRepeats bool
}

// CharacterActivatorOpts are the optional parameters of NewCharacterActivator.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.new
type CharacterActivatorOpts struct {
	// Alt requires the Alt key to be pressed.
	Alt bool
	// Control requires the Control key to be pressed.
	Control bool
	// Meta requires the Meta key to be pressed.
	Meta bool
	// ExcludeRepeats makes the activator reject the repeats of a key (Flutter's
	// includeRepeats is true by default).
	ExcludeRepeats bool
}

// NewCharacterActivator creates an activator for the character. It is a value.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.new
func NewCharacterActivator(character string, opts ...CharacterActivatorOpts) CharacterActivator {
	o := firstOpts(opts)
	return CharacterActivator{character, o.Alt, o.Control, o.Meta, o.ExcludeRepeats}
}

// Character is the character that triggers the shortcut.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.character
func (a CharacterActivator) Character() string { return a.character }

// Alt is whether the Alt key must be pressed.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.alt
func (a CharacterActivator) Alt() bool { return a.alt }

// Control is whether the Control key must be pressed.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.control
func (a CharacterActivator) Control() bool { return a.control }

// Meta is whether the Meta key must be pressed.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.meta
func (a CharacterActivator) Meta() bool { return a.meta }

// IncludeRepeats is whether the repeats of the key trigger the shortcut too.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.includeRepeats
func (a CharacterActivator) IncludeRepeats() bool { return !a.excludeRepeats }

// Triggers is nil: any key can produce the character.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.triggers
func (a CharacterActivator) Triggers() []services.LogicalKeyboardKey { return nil }

func (a CharacterActivator) shouldAcceptModifiers(pressed services.KeySet[services.LogicalKeyboardKey]) bool {
	return a.control == intersects(pressed, controlSynonyms) &&
		a.alt == intersects(pressed, altSynonyms) &&
		a.meta == intersects(pressed, metaSynonyms)
}

// Accepts is whether the event produced the character while the modifiers are as
// required.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.accepts
func (a CharacterActivator) Accepts(event services.KeyEvent, state *services.HardwareKeyboard) bool {
	if event.Character() != a.character {
		return false
	}
	switch event.(type) {
	case *services.KeyDownEvent:
	case *services.KeyRepeatEvent:
		if a.excludeRepeats {
			return false
		}
	default:
		return false
	}
	return a.shouldAcceptModifiers(state.LogicalKeysPressed())
}

// DebugDescribeKeys is the modifiers and the quoted character, joined by " + ".
//
// Flutter: widgets/shortcuts.dart CharacterActivator.debugDescribeKeys
func (a CharacterActivator) DebugDescribeKeys() string {
	var keys []string
	if a.alt {
		keys = append(keys, "Alt")
	}
	if a.control {
		keys = append(keys, "Control")
	}
	if a.meta {
		keys = append(keys, "Meta")
	}
	return strings.Join(append(keys, "'"+a.character+"'"), " + ")
}

// DebugFillProperties adds the character and whether repeats are excluded.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.debugFillProperties
func (a CharacterActivator) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewMessageProperty("character", a.DebugDescribeKeys()))
	p.Add(foundation.NewFlagProperty("includeRepeats", foundation.Some(a.IncludeRepeats()), foundation.FlagPropertyOpts{IfFalse: "excluding repeats"}))
}

// ShortcutEntry binds an activator to the intent that it triggers.
//
// Flutter: widgets/shortcuts.dart Shortcuts.shortcuts
type ShortcutEntry struct {
	// Activator is the key combination.
	Activator ShortcutActivator
	// Intent is what the combination triggers.
	Intent Intent
}

// ShortcutMap is a map of activators to intents, like Flutter's
// Map<ShortcutActivator, Intent>. It keeps the order of the entries, which
// decides which of the activators that accept an event wins; an activator that
// appears twice keeps the position of its first entry and the intent of its last.
//
// Deviation: an ordered list of entries, not a Go map, because the order matters
// (SPEC §12.10).
//
// Flutter: widgets/shortcuts.dart Shortcuts.shortcuts
type ShortcutMap []ShortcutEntry

// activatorsEqual compares two activators; an activator that cannot be compared
// equals only itself by identity.
func activatorsEqual(a, b ShortcutActivator) bool {
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	if ta != tb {
		return false
	}
	if ta.Comparable() {
		return a == b
	}
	return false
}

// intentsEqual compares two intents the way Dart's == does for const intents: by
// value if the type can be compared, else they are different.
func intentsEqual(a, b Intent) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	if ta != tb || !ta.Comparable() {
		return false
	}
	defer func() { _ = recover() }()
	return a == b
}

// normalized resolves the duplicate activators, as a map literal does.
func (m ShortcutMap) normalized() ShortcutMap {
	out := make(ShortcutMap, 0, len(m))
	for _, e := range m {
		found := false
		for i := range out {
			if activatorsEqual(out[i].Activator, e.Activator) {
				out[i].Intent = e.Intent
				found = true
				break
			}
		}
		if !found {
			out = append(out, e)
		}
	}
	return out
}

// shortcutMapsEqual is mapEquals: the same activators with equal intents.
func shortcutMapsEqual(a, b ShortcutMap) bool {
	a, b = a.normalized(), b.normalized()
	if len(a) != len(b) {
		return false
	}
	for _, e := range a {
		matched := false
		for _, o := range b {
			if activatorsEqual(e.Activator, o.Activator) {
				matched = intentsEqual(e.Intent, o.Intent)
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func describeIntent(intent Intent) string {
	if d, ok := intent.(foundation.Diagnosticable); ok {
		return foundation.DiagnosticableString(d)
	}
	return foundation.DescribeIdentity(intent)
}

// NewShortcutMapProperty creates the property that describes a shortcut map as
// "{{Key A}: ActivateIntent#00000, ...}".
//
// Flutter: widgets/shortcuts.dart ShortcutMapProperty
func NewShortcutMapProperty(name string, value ShortcutMap, opts ...foundation.DiagnosticsPropertyOpts) *foundation.DiagnosticsProperty {
	o := firstOpts(opts)
	if o.Description == "" {
		parts := make([]string, len(value))
		for i, e := range value {
			parts[i] = fmt.Sprintf("{%s}: %s", e.Activator.DebugDescribeKeys(), describeIntent(e.Intent))
		}
		o.Description = "{" + strings.Join(parts, ", ") + "}"
	}
	return foundation.NewDiagnosticsProperty(name, value, o)
}

type activatorIntentPair struct {
	activator ShortcutActivator
	intent    Intent
}

// ShortcutManager manages the mapping of activators to intents and handles the
// key events that its Shortcuts widget gets. Implement it by embedding
// ShortcutManagerBase to change how key presses are handled.
//
// Flutter: widgets/shortcuts.dart ShortcutManager
type ShortcutManager interface {
	foundation.Listenable
	foundation.Diagnosticable
	// Shortcuts is the map of activators to intents.
	//
	// Flutter: widgets/shortcuts.dart ShortcutManager.shortcuts
	Shortcuts() ShortcutMap
	// SetShortcuts replaces the map, and notifies the listeners if it changed.
	//
	// Flutter: widgets/shortcuts.dart ShortcutManager.shortcuts
	SetShortcuts(shortcuts ShortcutMap)
	// Modal is whether the manager stops the keys it has no shortcut for.
	//
	// Flutter: widgets/shortcuts.dart ShortcutManager.modal
	Modal() bool
	// HandleKeypress handles a key event: it finds the intent and the action for it
	// and invokes it.
	//
	// Flutter: widgets/shortcuts.dart ShortcutManager.handleKeypress
	HandleKeypress(bc BuildContext, event services.KeyEvent) KeyEventResult
	// Dispose releases the manager.
	//
	// Flutter: foundation/change_notifier.dart ChangeNotifier.dispose
	Dispose()

	shortcutManagerBase() *ShortcutManagerBase
}

// ShortcutManagerBase implements ShortcutManager. Embed it and call
// InitShortcutManagerBase with the outer value.
//
// Flutter: widgets/shortcuts.dart ShortcutManager
type ShortcutManagerBase struct {
	foundation.ChangeNotifier
	self      ShortcutManager
	modal     bool
	shortcuts ShortcutMap
	cache     map[services.LogicalKeyboardKey][]activatorIntentPair
	cacheNull []activatorIntentPair
	cached    bool
}

// ShortcutManagerOpts are the optional parameters of NewShortcutManager.
//
// Flutter: widgets/shortcuts.dart ShortcutManager.new
type ShortcutManagerOpts struct {
	// Shortcuts is the map of activators to intents.
	Shortcuts ShortcutMap
	// Modal makes the manager stop the keys that have no shortcut from reaching
	// the handlers above.
	Modal bool
}

// NewShortcutManager creates a manager.
//
// Flutter: widgets/shortcuts.dart ShortcutManager.new
func NewShortcutManager(opts ...ShortcutManagerOpts) *ShortcutManagerBase {
	m := &ShortcutManagerBase{}
	InitShortcutManagerBase(m, m, firstOpts(opts))
	return m
}

// InitShortcutManagerBase sets the outer value of the manager.
//
// Flutter: widgets/shortcuts.dart ShortcutManager.new
func InitShortcutManagerBase(b *ShortcutManagerBase, self ShortcutManager, opts ShortcutManagerOpts) {
	b.self = self
	b.modal = opts.Modal
	b.shortcuts = opts.Shortcuts
}

func (m *ShortcutManagerBase) shortcutManagerBase() *ShortcutManagerBase { return m }

// Modal is whether the manager stops the keys it has no shortcut for.
//
// Flutter: widgets/shortcuts.dart ShortcutManager.modal
func (m *ShortcutManagerBase) Modal() bool { return m.modal }

// Shortcuts is the map of activators to intents.
//
// Flutter: widgets/shortcuts.dart ShortcutManager.shortcuts
func (m *ShortcutManagerBase) Shortcuts() ShortcutMap { return m.shortcuts }

// SetShortcuts replaces the map; the listeners are notified if it differs.
//
// Flutter: widgets/shortcuts.dart ShortcutManager.shortcuts
func (m *ShortcutManagerBase) SetShortcuts(shortcuts ShortcutMap) {
	if !shortcutMapsEqual(m.shortcuts, shortcuts) {
		m.shortcuts = shortcuts
		m.cached = false
		m.NotifyListeners()
	}
}

func (m *ShortcutManagerBase) index() {
	if m.cached {
		return
	}
	m.cache = map[services.LogicalKeyboardKey][]activatorIntentPair{}
	m.cacheNull = nil
	for _, e := range m.shortcuts.normalized() {
		pair := activatorIntentPair{e.Activator, e.Intent}
		triggers := e.Activator.Triggers()
		if triggers == nil {
			m.cacheNull = append(m.cacheNull, pair)
			continue
		}
		for _, trigger := range triggers {
			m.cache[trigger] = append(m.cache[trigger], pair)
		}
	}
	m.cached = true
}

func (m *ShortcutManagerBase) find(event services.KeyEvent, state *services.HardwareKeyboard) Intent {
	m.index()
	candidates := append(slices.Clone(m.cache[event.LogicalKey()]), m.cacheNull...)
	for _, c := range candidates {
		if c.activator.Accepts(event, state) {
			return c.intent
		}
	}
	return nil
}

// HandleKeypress finds the intent for the event and invokes the action bound to it
// at the primary focus; the result depends on the action. When nothing handles
// the event, a modal manager skips the remaining handlers and the others ignore
// the event.
//
// Flutter: widgets/shortcuts.dart ShortcutManager.handleKeypress
func (m *ShortcutManagerBase) HandleKeypress(bc BuildContext, event services.KeyEvent) KeyEventResult {
	intent := m.find(event, bc.Owner().binding.Keyboard())
	var context BuildContext
	if focus := bc.Owner().FocusManager().PrimaryFocus(); focus != nil {
		context = focus.Context()
	}
	if intent != nil && context != nil {
		if action := ActionsMaybeFindIntent(context, intent); action != nil {
			if enabled, invokeResult := ActionsOf(context).InvokeActionIfEnabled(action, intent, context); enabled {
				return action.ToKeyEventResult(intent, invokeResult)
			}
		}
	}
	if m.modal {
		return KeyEventResultSkipRemainingHandlers
	}
	return KeyEventResultIgnored
}

// DebugFillProperties adds the shortcuts and whether the manager is modal.
//
// Flutter: widgets/shortcuts.dart ShortcutManager.debugFillProperties
func (m *ShortcutManagerBase) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("shortcuts", describeShortcutMap(m.shortcuts)))
	p.Add(foundation.NewFlagProperty("modal", foundation.Some(m.modal), foundation.FlagPropertyOpts{IfTrue: "modal", DefaultValue: foundation.Some(false)}))
}

type describedShortcutMap string

func (d describedShortcutMap) String() string { return string(d) }

func describeShortcutMap(m ShortcutMap) describedShortcutMap {
	parts := make([]string, len(m))
	for i, e := range m {
		parts[i] = fmt.Sprintf("%s: %s", foundation.DiagnosticableString(activatorDiagnosticable{e.Activator}), describeIntent(e.Intent))
	}
	return describedShortcutMap("{" + strings.Join(parts, ", ") + "}")
}

// activatorDiagnosticable describes an activator by its type and keys.
type activatorDiagnosticable struct{ activator ShortcutActivator }

func (a activatorDiagnosticable) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	if d, ok := a.activator.(foundation.Diagnosticable); ok {
		d.DebugFillProperties(p)
	}
}

// ToStringShort is the type and identity of the activator.
func (a activatorDiagnosticable) ToStringShort() string {
	return foundation.DescribeIdentity(a.activator)
}

// Shortcuts is a widget that creates key bindings to specific actions for its
// descendants: when a key event reaches it, the manager maps the event to an
// intent and the Actions above the primary focus run it.
//
// Flutter: widgets/shortcuts.dart Shortcuts
type Shortcuts struct {
	StatefulWidget
	manager          ShortcutManager
	shortcuts        ShortcutMap
	child            Widget
	debugLabel       string
	excludeSemantics bool
}

// ShortcutsOpts are the optional parameters of NewShortcuts and
// NewShortcutsManager.
//
// Flutter: widgets/shortcuts.dart Shortcuts.new
type ShortcutsOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// DebugLabel describes the widget in debug output and names its focus node.
	DebugLabel string
	// ExcludeSemantics leaves out the Semantics node of the widget's Focus
	// (Flutter's includeSemantics is true by default).
	ExcludeSemantics bool
}

// NewShortcuts creates a widget with a map of shortcuts.
//
// Flutter: widgets/shortcuts.dart Shortcuts.new
func NewShortcuts(shortcuts ShortcutMap, child Widget, opts ...ShortcutsOpts) *Shortcuts {
	o := firstOpts(opts)
	return &Shortcuts{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), nil, shortcuts, child, o.DebugLabel, o.ExcludeSemantics}
}

// NewShortcutsManager creates a widget whose shortcuts and handling are the
// manager's.
//
// Flutter: widgets/shortcuts.dart Shortcuts.manager
func NewShortcutsManager(manager ShortcutManager, child Widget, opts ...ShortcutsOpts) *Shortcuts {
	o := firstOpts(opts)
	return &Shortcuts{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), manager, nil, child, o.DebugLabel, o.ExcludeSemantics}
}

// Manager is the manager the widget was given, or nil.
//
// Flutter: widgets/shortcuts.dart Shortcuts.manager
func (w *Shortcuts) Manager() ShortcutManager { return w.manager }

// Shortcuts is the map of shortcuts: the manager's if there is one.
//
// Flutter: widgets/shortcuts.dart Shortcuts.shortcuts
func (w *Shortcuts) Shortcuts() ShortcutMap {
	if w.manager != nil {
		return w.manager.Shortcuts()
	}
	return w.shortcuts
}

// Child is the widget below.
//
// Flutter: widgets/shortcuts.dart Shortcuts.child
func (w *Shortcuts) Child() Widget { return w.child }

// DebugLabel is the label that describes the widget.
//
// Flutter: widgets/shortcuts.dart Shortcuts.debugLabel
func (w *Shortcuts) DebugLabel() string { return w.debugLabel }

// IncludeSemantics is whether the focus of the widget makes a Semantics node.
//
// Flutter: widgets/shortcuts.dart Shortcuts.includeSemantics
func (w *Shortcuts) IncludeSemantics() bool { return !w.excludeSemantics }

// CreateState creates the state.
//
// Flutter: widgets/shortcuts.dart Shortcuts.createState
func (w *Shortcuts) CreateState() State { return &shortcutsState{} }

// DebugFillProperties adds the manager and the shortcuts.
//
// Flutter: widgets/shortcuts.dart Shortcuts.debugFillProperties
func (w *Shortcuts) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	if w.manager != nil {
		p.Add(foundation.NewDiagnosticsProperty("manager", foundation.DiagnosticableString(w.manager)))
	}
	opts := foundation.DiagnosticsPropertyOpts{}
	if w.debugLabel != "" {
		opts.Description = w.debugLabel
	}
	p.Add(NewShortcutMapProperty("shortcuts", w.Shortcuts(), opts))
}

type shortcutsState struct {
	StateOf[*Shortcuts]
	internalManager ShortcutManager
}

func (s *shortcutsState) manager() ShortcutManager {
	if m := s.Widget().manager; m != nil {
		return m
	}
	return s.internalManager
}

func (s *shortcutsState) InitState() {
	if s.Widget().manager == nil {
		s.internalManager = NewShortcutManager()
		s.internalManager.SetShortcuts(s.Widget().shortcuts)
	}
}

func (s *shortcutsState) Dispose() {
	if s.internalManager != nil {
		s.internalManager.Dispose()
	}
}

func (s *shortcutsState) DidUpdateWidget(old *Shortcuts) {
	if s.Widget().manager != old.manager {
		if s.Widget().manager != nil {
			if s.internalManager != nil {
				s.internalManager.Dispose()
			}
			s.internalManager = nil
		} else if s.internalManager == nil {
			s.internalManager = NewShortcutManager()
		}
	}
	if s.internalManager != nil {
		s.internalManager.SetShortcuts(s.Widget().shortcuts)
	}
}

func (s *shortcutsState) handleOnKeyEvent(node *FocusNode, event services.KeyEvent) KeyEventResult {
	if node.Context() == nil {
		return KeyEventResultIgnored
	}
	return s.manager().HandleKeypress(node.Context(), event)
}

func (s *shortcutsState) Build(BuildContext) Widget {
	w := s.Widget()
	label := "Shortcuts"
	if w.debugLabel != "" {
		label = "Shortcuts: " + w.debugLabel
	}
	return NewFocus(w.child, FocusOpts{
		DebugLabel: label, CanRequestFocus: foundation.Some(false), OnKeyEvent: s.handleOnKeyEvent, ExcludeSemantics: w.excludeSemantics,
	})
}

// CallbackShortcutEntry binds an activator to a function.
//
// Flutter: widgets/shortcuts.dart CallbackShortcuts.bindings
type CallbackShortcutEntry struct {
	// Activator is the key combination.
	Activator ShortcutActivator
	// Callback is called when the combination is pressed.
	Callback func()
}

// CallbackShortcuts is a widget that binds key combinations to functions, with no
// intents or actions: a quick way to handle a key in a part of the tree. Every
// binding that accepts an event runs, and the event is then handled.
//
// Flutter: widgets/shortcuts.dart CallbackShortcuts
type CallbackShortcuts struct {
	StatelessWidget
	bindings []CallbackShortcutEntry
	child    Widget
}

// NewCallbackShortcuts creates the widget.
//
// Flutter: widgets/shortcuts.dart CallbackShortcuts.new
func NewCallbackShortcuts(bindings []CallbackShortcutEntry, child Widget, opts ...WidgetOpts) *CallbackShortcuts {
	return &CallbackShortcuts{NewStatelessWidgetBase(opts...), bindings, child}
}

// Bindings are the key bindings.
//
// Flutter: widgets/shortcuts.dart CallbackShortcuts.bindings
func (w *CallbackShortcuts) Bindings() []CallbackShortcutEntry { return w.bindings }

// Child is the widget below.
//
// Flutter: widgets/shortcuts.dart CallbackShortcuts.child
func (w *CallbackShortcuts) Child() Widget { return w.child }

// Build builds a Focus that applies the bindings to the key events.
//
// Flutter: widgets/shortcuts.dart CallbackShortcuts.build
func (w *CallbackShortcuts) Build(bc BuildContext) Widget {
	keyboard := bc.Owner().binding.Keyboard()
	return NewFocus(w.child, FocusOpts{
		CanRequestFocus: foundation.Some(false), SkipTraversal: foundation.Some(true),
		OnKeyEvent: func(_ *FocusNode, event services.KeyEvent) KeyEventResult {
			result := KeyEventResultIgnored
			for _, entry := range w.bindings {
				if entry.Activator.Accepts(event, keyboard) {
					entry.Callback()
					result = KeyEventResultHandled
				}
			}
			return result
		},
	})
}

// ShortcutRegistryEntry is a token that the code that registered shortcuts keeps
// to replace or remove them.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistryEntry
type ShortcutRegistryEntry struct {
	registry *ShortcutRegistry
}

// Registry is the registry the entry belongs to.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistryEntry.registry
func (e *ShortcutRegistryEntry) Registry() *ShortcutRegistry { return e.registry }

// ReplaceAll replaces the shortcuts of the entry.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistryEntry.replaceAll
func (e *ShortcutRegistryEntry) ReplaceAll(value ShortcutMap) { e.registry.replaceAll(e, value) }

// Dispose removes the shortcuts of the entry; the entry can not be used again.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistryEntry.dispose
func (e *ShortcutRegistryEntry) Dispose() { e.registry.disposeEntry(e) }

// ShortcutRegistry is a class used by the ShortcutRegistrar to keep the shortcuts
// that code anywhere in the app registered: the registrar's manager handles them
// all.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistry
type ShortcutRegistry struct {
	foundation.ChangeNotifier
	scheduler          interface{ AddPostFrameCallback(scheduler.FrameCallback) }
	registered         []*registeredShortcuts
	notificationQueued bool
	disposed           bool
}

type registeredShortcuts struct {
	entry     *ShortcutRegistryEntry
	shortcuts ShortcutMap
}

// NewShortcutRegistry creates a registry. The scheduler runs the notification of
// the listeners after the frame (WidgetsBinding.Scheduler()).
//
// Flutter: widgets/shortcuts.dart ShortcutRegistry.new
func NewShortcutRegistry(frames interface{ AddPostFrameCallback(scheduler.FrameCallback) }) *ShortcutRegistry {
	return &ShortcutRegistry{scheduler: frames}
}

// Dispose releases the registry.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistry.dispose
func (r *ShortcutRegistry) Dispose() {
	r.ChangeNotifier.Dispose()
	r.disposed = true
}

// Shortcuts are the shortcuts of all the entries.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistry.shortcuts
func (r *ShortcutRegistry) Shortcuts() ShortcutMap {
	r.DebugAssertNotDisposed()
	var out ShortcutMap
	for _, e := range r.registered {
		out = append(out, e.shortcuts...)
	}
	return out
}

func (r *ShortcutRegistry) find(entry *ShortcutRegistryEntry) *registeredShortcuts {
	for _, e := range r.registered {
		if e.entry == entry {
			return e
		}
	}
	return nil
}

// AddAll registers the shortcuts and returns the entry to change or remove them.
// In debug builds it panics with the tag ShortcutRegistryEmpty when there are none
// and with ShortcutRegistryDuplicate when an activator is already registered.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistry.addAll
func (r *ShortcutRegistry) AddAll(value ShortcutMap) *ShortcutRegistryEntry {
	r.DebugAssertNotDisposed()
	if foundation.DebugMode && len(value) == 0 {
		panic(errors.New("ShortcutRegistryEmpty", "cannot register an empty map of shortcuts"))
	}
	entry := &ShortcutRegistryEntry{r}
	r.registered = append(r.registered, &registeredShortcuts{entry, value})
	r.debugCheckForDuplicates()
	r.notifyListenersNextFrame()
	return entry
}

func (r *ShortcutRegistry) notifyListenersNextFrame() {
	if r.notificationQueued {
		return
	}
	r.scheduler.AddPostFrameCallback(func(time.Duration) {
		r.notificationQueued = false
		if !r.disposed {
			r.NotifyListeners()
		}
	})
	r.notificationQueued = true
}

func (r *ShortcutRegistry) replaceAll(entry *ShortcutRegistryEntry, value ShortcutMap) {
	r.DebugAssertNotDisposed()
	r.debugCheckEntryIsValid(entry)
	r.find(entry).shortcuts = value
	r.debugCheckForDuplicates()
	r.notifyListenersNextFrame()
}

func (r *ShortcutRegistry) disposeEntry(entry *ShortcutRegistryEntry) {
	r.debugCheckEntryIsValid(entry)
	before := len(r.registered)
	r.registered = slices.DeleteFunc(r.registered, func(e *registeredShortcuts) bool { return e.entry == entry })
	if len(r.registered) != before {
		r.notifyListenersNextFrame()
	}
}

func (r *ShortcutRegistry) debugCheckEntryIsValid(entry *ShortcutRegistryEntry) {
	if !foundation.DebugMode || r.find(entry) != nil {
		return
	}
	if entry.registry == r {
		panic(errors.New("ShortcutRegistryEntryDisposed",
			"entry "+foundation.DescribeIdentity(entry)+" is invalid: the entry has already been disposed of; tokens are not valid after dispose is called on them, and should no longer be used"))
	}
	panic(errors.New("ShortcutRegistryEntryForeign",
		"foreign entry "+foundation.DescribeIdentity(entry)+" used: this entry was not created by this registry, it was created by "+
			foundation.DescribeIdentity(entry.registry)+", and should be used with that registry instead"))
}

func (r *ShortcutRegistry) debugCheckForDuplicates() {
	if !foundation.DebugMode {
		return
	}
	for i, e := range r.registered {
		for _, s := range e.shortcuts {
			for _, other := range r.registered[:i] {
				for _, o := range other.shortcuts {
					if activatorsEqual(s.Activator, o.Activator) {
						panic(errors.New("ShortcutRegistryDuplicate",
							"ShortcutRegistry: received a duplicate registration for the shortcut "+s.Activator.DebugDescribeKeys()+
								" in "+foundation.DescribeIdentity(e.entry)+" and "+foundation.DescribeIdentity(other.entry),
							"activator", s.Activator.DebugDescribeKeys()))
					}
				}
			}
		}
	}
}

// ShortcutRegistryOf is the registry of the closest ShortcutRegistrar above the
// context, and the context depends on it. In debug builds it panics with the tag
// ShortcutRegistrarNotFound when there is none.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistry.of
func ShortcutRegistryOf(bc BuildContext) *ShortcutRegistry {
	registry := ShortcutRegistryMaybeOf(bc)
	if foundation.DebugMode && registry == nil {
		panic(errors.New("ShortcutRegistrarNotFound",
			"unable to find a ShortcutRegistrar widget in the context: ShortcutRegistry.of() was called with a context that does not contain a ShortcutRegistrar widget",
			"context", describeContext(bc)))
	}
	return registry
}

// ShortcutRegistryMaybeOf is ShortcutRegistryOf that returns nil instead of
// panicking.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistry.maybeOf
func ShortcutRegistryMaybeOf(bc BuildContext) *ShortcutRegistry {
	if scope, ok := DependOn[*shortcutRegistrarScope](bc); ok {
		return scope.registry
	}
	return nil
}

// ShortcutRegistrar is a widget that holds a ShortcutRegistry and handles the
// shortcuts registered with it. Put one above the app.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistrar
type ShortcutRegistrar struct {
	StatefulWidget
	child Widget
}

// NewShortcutRegistrar creates the widget.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistrar.new
func NewShortcutRegistrar(child Widget, opts ...WidgetOpts) *ShortcutRegistrar {
	return &ShortcutRegistrar{NewStatefulWidgetBase(opts...), child}
}

// Child is the widget below.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistrar.child
func (w *ShortcutRegistrar) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/shortcuts.dart ShortcutRegistrar.createState
func (w *ShortcutRegistrar) CreateState() State { return &shortcutRegistrarState{} }

type shortcutRegistrarState struct {
	StateOf[*ShortcutRegistrar]
	registry *ShortcutRegistry
	manager  *ShortcutManagerBase
	handle   foundation.ListenerHandle
}

func (s *shortcutRegistrarState) InitState() {
	s.registry = NewShortcutRegistry(WidgetsBindingOf(s.Context()).Scheduler())
	s.manager = NewShortcutManager()
	s.handle = s.registry.AddListener(func() { s.manager.SetShortcuts(s.registry.Shortcuts()) })
}

func (s *shortcutRegistrarState) Dispose() {
	s.registry.RemoveListener(s.handle)
	s.registry.Dispose()
	s.manager.Dispose()
}

func (s *shortcutRegistrarState) Build(BuildContext) Widget {
	return &shortcutRegistrarScope{NewInheritedWidgetBase(
		NewShortcutsManager(s.manager, s.Widget().child, ShortcutsOpts{DebugLabel: "<Shortcut Registrar>"})), s.registry}
}

type shortcutRegistrarScope struct {
	InheritedWidget
	registry *ShortcutRegistry
}

func (w *shortcutRegistrarScope) UpdateShouldNotify(old Widget) bool {
	return w.registry != old.(*shortcutRegistrarScope).registry
}
