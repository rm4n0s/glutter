package semantics

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Assertiveness is how urgently an announcement is spoken.
//
// Flutter: semantics/semantics_event.dart Assertiveness
type Assertiveness int

// The values of Assertiveness. The zero value is AssertivenessPolite.
const (
	// AssertivenessPolite waits for the speech in progress to finish.
	//
	// Flutter: semantics/semantics_event.dart Assertiveness.polite
	AssertivenessPolite Assertiveness = iota
	// AssertivenessAssertive interrupts the speech in progress.
	//
	// Flutter: semantics/semantics_event.dart Assertiveness.assertive
	AssertivenessAssertive
)

// SemanticsEvent is an event that the framework sends to the platform about a
// node, such as an announcement.
//
// Flutter: semantics/semantics_event.dart SemanticsEvent
type SemanticsEvent interface {
	// Type is the type of the event, such as "announce".
	//
	// Flutter: semantics/semantics_event.dart SemanticsEvent.type
	Type() string
	// GetDataMap is the data of the event.
	//
	// Flutter: semantics/semantics_event.dart SemanticsEvent.getDataMap
	GetDataMap() map[string]any
	// ToMap is the message that Flutter sends on its accessibility channel.
	//
	// Flutter: semantics/semantics_event.dart SemanticsEvent.toMap
	ToMap(nodeID foundation.Opt[int]) map[string]any
}

// semanticsEventBase has what every event has.
type semanticsEventBase struct{ typ string }

func (b semanticsEventBase) Type() string { return b.typ }

func eventToMap(e SemanticsEvent, nodeID foundation.Opt[int]) map[string]any {
	m := map[string]any{"type": e.Type(), "data": e.GetDataMap()}
	if id, ok := nodeID.Get(); ok {
		m["nodeId"] = id
	}
	return m
}

// SemanticsEventString describes an event with its sorted data.
//
// Flutter: semantics/semantics_event.dart SemanticsEvent.toString
func SemanticsEventString(e SemanticsEvent, typeName string) string {
	data := e.GetDataMap()
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = k + ": " + fmt.Sprint(data[k])
	}
	return typeName + "(" + strings.Join(pairs, ", ") + ")"
}

// AnnounceSemanticsEvent says that the platform should announce a message.
//
// Flutter: semantics/semantics_event.dart AnnounceSemanticsEvent
type AnnounceSemanticsEvent struct {
	semanticsEventBase
	// Message is the text to announce.
	Message string
	// TextDirection is the direction of the message.
	TextDirection ui.TextDirection
	// ViewID is the view of the announcement.
	ViewID ui.ViewID
	// Assertiveness is how urgent the announcement is.
	Assertiveness Assertiveness
}

// NewAnnounceSemanticsEvent creates an announcement.
//
// Flutter: semantics/semantics_event.dart AnnounceSemanticsEvent.new
func NewAnnounceSemanticsEvent(message string, textDirection ui.TextDirection, viewID ui.ViewID, assertiveness ...Assertiveness) *AnnounceSemanticsEvent {
	e := &AnnounceSemanticsEvent{semanticsEventBase{"announce"}, message, textDirection, viewID, AssertivenessPolite}
	if len(assertiveness) > 0 {
		e.Assertiveness = assertiveness[0]
	}
	return e
}

// GetDataMap is the data of the event.
//
// Flutter: semantics/semantics_event.dart AnnounceSemanticsEvent.getDataMap
func (e *AnnounceSemanticsEvent) GetDataMap() map[string]any {
	m := map[string]any{"viewId": int(e.ViewID), "message": e.Message, "textDirection": int(e.TextDirection) - 1}
	if e.Assertiveness != AssertivenessPolite {
		m["assertiveness"] = int(e.Assertiveness)
	}
	return m
}

// ToMap is the message of the event.
func (e *AnnounceSemanticsEvent) ToMap(nodeID foundation.Opt[int]) map[string]any {
	return eventToMap(e, nodeID)
}

// TooltipSemanticsEvent says that a tooltip was shown.
//
// Flutter: semantics/semantics_event.dart TooltipSemanticsEvent
type TooltipSemanticsEvent struct {
	semanticsEventBase
	// Message is the text of the tooltip.
	Message string
}

// NewTooltipSemanticsEvent creates the event.
//
// Flutter: semantics/semantics_event.dart TooltipSemanticsEvent.new
func NewTooltipSemanticsEvent(message string) *TooltipSemanticsEvent {
	return &TooltipSemanticsEvent{semanticsEventBase{"tooltip"}, message}
}

// GetDataMap is the data of the event.
//
// Flutter: semantics/semantics_event.dart TooltipSemanticsEvent.getDataMap
func (e *TooltipSemanticsEvent) GetDataMap() map[string]any {
	return map[string]any{"message": e.Message}
}

// ToMap is the message of the event.
func (e *TooltipSemanticsEvent) ToMap(nodeID foundation.Opt[int]) map[string]any {
	return eventToMap(e, nodeID)
}

// LongPressSemanticsEvent says that a node was long pressed.
//
// Flutter: semantics/semantics_event.dart LongPressSemanticsEvent
type LongPressSemanticsEvent struct{ semanticsEventBase }

// NewLongPressSemanticsEvent creates the event.
//
// Flutter: semantics/semantics_event.dart LongPressSemanticsEvent.new
func NewLongPressSemanticsEvent() *LongPressSemanticsEvent {
	return &LongPressSemanticsEvent{semanticsEventBase{"longPress"}}
}

// GetDataMap is empty.
//
// Flutter: semantics/semantics_event.dart LongPressSemanticsEvent.getDataMap
func (e *LongPressSemanticsEvent) GetDataMap() map[string]any { return map[string]any{} }

// ToMap is the message of the event.
func (e *LongPressSemanticsEvent) ToMap(nodeID foundation.Opt[int]) map[string]any {
	return eventToMap(e, nodeID)
}

// TapSemanticEvent says that a node was tapped.
//
// Flutter: semantics/semantics_event.dart TapSemanticEvent
type TapSemanticEvent struct{ semanticsEventBase }

// NewTapSemanticEvent creates the event.
//
// Flutter: semantics/semantics_event.dart TapSemanticEvent.new
func NewTapSemanticEvent() *TapSemanticEvent { return &TapSemanticEvent{semanticsEventBase{"tap"}} }

// GetDataMap is empty.
//
// Flutter: semantics/semantics_event.dart TapSemanticEvent.getDataMap
func (e *TapSemanticEvent) GetDataMap() map[string]any { return map[string]any{} }

// ToMap is the message of the event.
func (e *TapSemanticEvent) ToMap(nodeID foundation.Opt[int]) map[string]any {
	return eventToMap(e, nodeID)
}

// FocusSemanticEvent says that a node got the focus.
//
// Flutter: semantics/semantics_event.dart FocusSemanticEvent
type FocusSemanticEvent struct{ semanticsEventBase }

// NewFocusSemanticEvent creates the event.
//
// Flutter: semantics/semantics_event.dart FocusSemanticEvent.new
func NewFocusSemanticEvent() *FocusSemanticEvent {
	return &FocusSemanticEvent{semanticsEventBase{"focus"}}
}

// GetDataMap is empty.
//
// Flutter: semantics/semantics_event.dart FocusSemanticEvent.getDataMap
func (e *FocusSemanticEvent) GetDataMap() map[string]any { return map[string]any{} }

// ToMap is the message of the event.
func (e *FocusSemanticEvent) ToMap(nodeID foundation.Opt[int]) map[string]any {
	return eventToMap(e, nodeID)
}
