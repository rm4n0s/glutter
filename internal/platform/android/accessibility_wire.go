package android

import (
	"math"

	"github.com/rm4n0s/errors"
)

// The wire format of the semantics updates that the UI thread sends to the main thread (command
// opSemantics, SPEC §24.10). Flutter's AccessibilityBridge.updateSemantics takes a ByteBuffer, a
// String[] and a ByteBuffer[] that the engine fills (PlatformViewAndroid::UpdateSemantics); here one
// byte array carries all three, and every property that Android's AccessibilityNodeInfo needs is
// already worked out by Go (accessibility_info.go), so the Java side only applies it.
// GlutterAccessibility.java decodes this layout; the two MUST agree, and the golden bytes of
// TestSemanticsPayloadGolden are repeated in the header comment of that class.
//
// All numbers are little-endian. "str" is a u32 byte count followed by that many bytes of UTF-8;
// a count of 0 is the empty string, which Flutter's engine sends as "no string" (-1). All text
// offsets are UTF-16 code units (Go converts from the byte offsets of the framework).
//
//	payload   := u8 version (=1), u32 nodeCount, node * nodeCount
//	node      := i32 id
//	             u32 flags          Flutter's SemanticsFlag bits (AccessibilityBridge.Flag): bit 0 HAS_CHECKED_STATE,
//	                                1 IS_CHECKED, 2 IS_SELECTED, 3 IS_BUTTON, 4 IS_TEXT_FIELD, 5 IS_FOCUSED,
//	                                6 HAS_ENABLED_STATE, 7 IS_ENABLED, 8 IS_IN_MUTUALLY_EXCLUSIVE_GROUP, 9 IS_HEADER,
//	                                10 IS_OBSCURED, 11 SCOPES_ROUTE, 12 NAMES_ROUTE, 13 IS_HIDDEN, 14 IS_IMAGE,
//	                                15 IS_LIVE_REGION, 16 HAS_TOGGLED_STATE, 17 IS_TOGGLED, 18 HAS_IMPLICIT_SCROLLING,
//	                                19 IS_MULTILINE, 20 IS_READ_ONLY, 21 IS_FOCUSABLE, 22 IS_LINK, 23 IS_SLIDER,
//	                                24 IS_KEYBOARD_KEY, 25 IS_CHECK_STATE_MIXED, 26 HAS_EXPANDED_STATE, 27 IS_EXPANDED,
//	                                28 HAS_SELECTED_STATE, 29 HAS_REQUIRED_STATE, 30 IS_REQUIRED,
//	                                31 IS_ACCESSIBILITY_FOCUS_BLOCKED
//	             u32 actions        ui.SemanticsAction bits (AccessibilityBridge.Action): bit 0 TAP, 1 LONG_PRESS,
//	                                2 SCROLL_LEFT, 3 SCROLL_RIGHT, 4 SCROLL_UP, 5 SCROLL_DOWN, 6 INCREASE, 7 DECREASE,
//	                                8 SHOW_ON_SCREEN, 9 MOVE_CURSOR_FORWARD_BY_CHARACTER, 10 ..._BACKWARD_BY_CHARACTER,
//	                                11 SET_SELECTION, 12 COPY, 13 CUT, 14 PASTE, 15 DID_GAIN_ACCESSIBILITY_FOCUS,
//	                                16 DID_LOSE_ACCESSIBILITY_FOCUS, 17 CUSTOM_ACTION, 18 DISMISS,
//	                                19 MOVE_CURSOR_FORWARD_BY_WORD, 20 ..._BACKWARD_BY_WORD, 21 SET_TEXT, 22 FOCUS,
//	                                23 SCROLL_TO_OFFSET, 24 EXPAND, 25 COLLAPSE
//	             u32 props          what AccessibilityNodeInfo is told: bit 0 important, 1 focusable, 2 enabled,
//	                                3 clickable, 4 longClickable, 5 scrollable, 6 checkable, 7 selected, 8 password,
//	                                9 editable, 10 dismissable, 11 heading, 12 live region (polite), 13 canOpenPopup,
//	                                14 text field (bit 4 of flags, as the class name is not enough)
//	             u8  checked        0 not checked, 1 checked, 2 mixed (meaningful when props bit 6 is set)
//	             u8  expanded       0 no expanded state, 1 collapsed, 2 expanded
//	             u8  granularities  MOVEMENT_GRANULARITY_* bits the node supports: 1 character, 2 word
//	             u8  rangeType      0 no RangeInfo, 1 float, 2 indeterminate
//	             i32 maxTextLength  AccessibilityNodeInfo.setMaxTextLength, or -1
//	             i32 scrollChildren
//	             i32 scrollIndex
//	             i32 selectionBase  UTF-16 offset in the value, or -1
//	             i32 selectionExtent
//	             f32 scrollPosition, f32 scrollExtentMax, f32 scrollExtentMin    (NaN when unset, +-Inf when unbounded)
//	             f32 rangeMin, f32 rangeMax, f32 rangeCurrent                    (0 when rangeType is 0)
//	             i32 collectionRows, i32 collectionColumns                       (0, 0 when the node is no collection)
//	             f32 left, f32 top, f32 right, f32 bottom                        (the rect in the node's own space)
//	             f32 * 16 transform                                              (to the parent in traversal order, column major)
//	             f32 * 16 hitTestTransform                                       (to the parent for hit testing)
//	             str identifier     the view id resource name
//	             str className     the AccessibilityNodeInfo class name
//	             str value          plain value, label and tooltip (the events and the cursor prediction use them)
//	             str label
//	             str tooltip
//	             styledList text    the text of a text field (its value)
//	             styledList content value, label and hint, joined with ", " by the Java side (the content description)
//	             styledList hint    label and hint, joined with ", " (the hint text of a text field)
//	             styledList increased   the value after an increase (0 or 1 item)
//	             styledList decreased   the value after a decrease (0 or 1 item)
//	             u32 n, i32 * n traversalChildren    ids, in traversal order
//	             u32 n, i32 * n hitTestChildren      ids, in hit test order
//	             u32 n, (i32 actionId, str label) * n  the AccessibilityAction ids to add (and the labels of the ones with one);
//	                                the custom actions are id + 267386881 (AccessibilityBridge.FIRST_RESOURCE_ID)
//	styledList := u8 n, styled * n
//	styled    := u8 kind (0 value, 1 label, 2 hint), str text, str url, str locale,
//	             u32 n, (u32 start, u32 end, u8 type (0 spell out, 1 locale), str locale) * n
//
// url, when it is not empty, is a URLSpan over the whole text, locale a LocaleSpan over the whole
// text, and each attribute a TtsSpan (spell out) or a LocaleSpan over its range, as
// AccessibilityStringBuilder does.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityBridge.java AccessibilityBridge.updateSemantics
const semanticsPayloadVersion = 1

// The limits of a payload: the Java side and the decoder reject what is above them or what has
// fewer bytes than its counts claim.
const (
	// maxSemanticsPayload is the largest payload the decoder accepts.
	maxSemanticsPayload = 64 << 20
	// minNodeBytes is the size of a node with empty strings, lists and children.
	minNodeBytes = 253
)

// The bits of androidNode.Props.
const (
	propImportant = 1 << iota
	propFocusable
	propEnabled
	propClickable
	propLongClickable
	propScrollable
	propCheckable
	propSelected
	propPassword
	propEditable
	propDismissable
	propHeading
	propLivePolite
	propCanOpenPopup
	propTextField
)

// The values of androidNode.RangeType.
const (
	rangeNone = iota
	rangeFloat
	rangeIndeterminate
)

// The values of the kind of a styledText: the part of the node's text it comes from.
const (
	kindValue = iota
	kindLabel
	kindHint
)

// The types of a styledAttr.
const (
	attrSpellOut = iota
	attrLocale
)

// styledAttr is a span over a range of a text, in UTF-16 units.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityStringBuilder.java StringAttribute
type styledAttr struct {
	Start, End int
	Type       uint8
	Locale     string
}

// styledText is a string with the spans that AccessibilityStringBuilder gives it.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityStringBuilder.java AccessibilityStringBuilder
type styledText struct {
	Kind              uint8
	Text, URL, Locale string
	Attrs             []styledAttr
}

// androidAction is an AccessibilityAction to add to the node: its id and its label.
//
// Flutter: shell/platform/android/io/flutter/view/BaseRoleConfigurator.java BaseRoleConfigurator
type androidAction struct {
	ID    int
	Label string
}

// androidNode is a semantics node as the Java side takes it: the raw state that it needs for its
// events and its tree, and the properties of the AccessibilityNodeInfo that Go has already worked
// out (accessibility_info.go).
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityBridge.java AccessibilityBridge.SemanticsNode
type androidNode struct {
	ID                                               int
	Flags, FlutterActions, Props                     uint32
	Checked, Expanded, Granularities, RangeType      uint8
	MaxTextLength, ScrollChildren, ScrollIndex       int
	SelectionBase, SelectionExtent                   int
	ScrollPosition, ScrollExtentMax, ScrollExtentMin float32
	RangeMin, RangeMax, RangeCurrent                 float32
	CollectionRows, CollectionColumns                int
	Left, Top, Right, Bottom                         float32
	Transform, HitTestTransform                      [16]float32
	Identifier, ClassName, Value, Label, Tooltip     string
	Text, Content, Hint, Increased, Decreased        []styledText
	TraversalChildren, HitTestChildren               []int
	AndroidActions                                   []androidAction
}

// wireWriter appends the fields of a payload.
type wireWriter struct{ b []byte }

func (w *wireWriter) u8(v uint8) { w.b = append(w.b, v) }

func (w *wireWriter) u32(v uint32) {
	w.b = append(w.b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func (w *wireWriter) i32(v int) { w.u32(uint32(int32(v))) }

func (w *wireWriter) f32(v float32) { w.u32(math.Float32bits(v)) }

func (w *wireWriter) str(s string) {
	w.u32(uint32(len(s)))
	w.b = append(w.b, s...)
}

func (w *wireWriter) styledList(l []styledText) {
	w.u8(uint8(len(l)))
	for _, s := range l {
		w.u8(s.Kind)
		w.str(s.Text)
		w.str(s.URL)
		w.str(s.Locale)
		w.u32(uint32(len(s.Attrs)))
		for _, a := range s.Attrs {
			w.u32(uint32(a.Start))
			w.u32(uint32(a.End))
			w.u8(a.Type)
			w.str(a.Locale)
		}
	}
}

func (w *wireWriter) ids(l []int) {
	w.u32(uint32(len(l)))
	for _, id := range l {
		w.i32(id)
	}
}

// encodeSemantics makes the payload of the nodes (the layout at the top of this file).
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityBridge.java AccessibilityBridge.updateSemantics
func encodeSemantics(nodes []androidNode) []byte {
	w := &wireWriter{b: make([]byte, 0, 64+len(nodes)*minNodeBytes)}
	w.u8(semanticsPayloadVersion)
	w.u32(uint32(len(nodes)))
	for i := range nodes {
		n := &nodes[i]
		w.i32(n.ID)
		w.u32(n.Flags)
		w.u32(n.FlutterActions)
		w.u32(n.Props)
		w.u8(n.Checked)
		w.u8(n.Expanded)
		w.u8(n.Granularities)
		w.u8(n.RangeType)
		w.i32(n.MaxTextLength)
		w.i32(n.ScrollChildren)
		w.i32(n.ScrollIndex)
		w.i32(n.SelectionBase)
		w.i32(n.SelectionExtent)
		w.f32(n.ScrollPosition)
		w.f32(n.ScrollExtentMax)
		w.f32(n.ScrollExtentMin)
		w.f32(n.RangeMin)
		w.f32(n.RangeMax)
		w.f32(n.RangeCurrent)
		w.i32(n.CollectionRows)
		w.i32(n.CollectionColumns)
		w.f32(n.Left)
		w.f32(n.Top)
		w.f32(n.Right)
		w.f32(n.Bottom)
		for _, f := range n.Transform {
			w.f32(f)
		}
		for _, f := range n.HitTestTransform {
			w.f32(f)
		}
		w.str(n.Identifier)
		w.str(n.ClassName)
		w.str(n.Value)
		w.str(n.Label)
		w.str(n.Tooltip)
		w.styledList(n.Text)
		w.styledList(n.Content)
		w.styledList(n.Hint)
		w.styledList(n.Increased)
		w.styledList(n.Decreased)
		w.ids(n.TraversalChildren)
		w.ids(n.HitTestChildren)
		w.u32(uint32(len(n.AndroidActions)))
		for _, a := range n.AndroidActions {
			w.i32(a.ID)
			w.str(a.Label)
		}
	}
	return w.b
}

// wireReader reads the fields of a payload. The first field that does not fit sets err, and every
// read after it returns zero values, so that the decoder checks once.
type wireReader struct {
	b   []byte
	pos int
	err error
}

// take returns the next n bytes. It fails with AndroidSemanticsPayloadInvalid when fewer are left.
func (r *wireReader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || n > len(r.b)-r.pos {
		r.err = errors.New(tagAndroidSemanticsPayloadInvalid, "the payload ends before a field does", "offset", r.pos, "need", n, "size", len(r.b))
		return nil
	}
	s := r.b[r.pos : r.pos+n]
	r.pos += n
	return s
}

func (r *wireReader) u8() uint8 {
	if s := r.take(1); s != nil {
		return s[0]
	}
	return 0
}

func (r *wireReader) u32() uint32 {
	if s := r.take(4); s != nil {
		return uint32(s[0]) | uint32(s[1])<<8 | uint32(s[2])<<16 | uint32(s[3])<<24
	}
	return 0
}

func (r *wireReader) i32() int { return int(int32(r.u32())) }

func (r *wireReader) f32() float32 { return math.Float32frombits(r.u32()) }

func (r *wireReader) str() string { return string(r.take(int(r.u32()))) }

// count reads a u32 count of items that take at least size bytes each, and fails like take when
// the rest of the payload cannot hold them.
func (r *wireReader) count(size int) int {
	n := int(r.u32())
	if r.err == nil && (n < 0 || n > (len(r.b)-r.pos)/size) {
		r.err = errors.New(tagAndroidSemanticsPayloadInvalid, "a count is larger than the rest of the payload", "offset", r.pos, "count", n, "size", len(r.b))
		return 0
	}
	return n
}

func (r *wireReader) styledList() []styledText {
	n := int(r.u8())
	var out []styledText
	for i := 0; i < n && r.err == nil; i++ {
		s := styledText{Kind: r.u8(), Text: r.str(), URL: r.str(), Locale: r.str()}
		for j, na := 0, r.count(13); j < na && r.err == nil; j++ {
			s.Attrs = append(s.Attrs, styledAttr{Start: int(r.u32()), End: int(r.u32()), Type: r.u8(), Locale: r.str()})
		}
		out = append(out, s)
	}
	return out
}

func (r *wireReader) ids() []int {
	var out []int
	for i, n := 0, r.count(4); i < n && r.err == nil; i++ {
		out = append(out, r.i32())
	}
	return out
}

// decodeSemantics reads a payload back into its nodes. The device never needs it (the Java side has
// its own decoder); it is the other half of the golden test, and it makes sure that the encoder
// writes what the counts say. It fails with AndroidSemanticsPayloadInvalid on a payload that is
// too large, has another version, is cut short, has counts larger than its bytes, or has bytes left.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityBridge.java AccessibilityBridge.updateSemantics
func decodeSemantics(payload []byte) ([]androidNode, error) {
	if len(payload) > maxSemanticsPayload {
		return nil, errors.New(tagAndroidSemanticsPayloadInvalid, "the payload is larger than the limit", "size", len(payload), "limit", maxSemanticsPayload)
	}
	r := &wireReader{b: payload}
	if v := r.u8(); r.err == nil && v != semanticsPayloadVersion {
		return nil, errors.New(tagAndroidSemanticsPayloadInvalid, "the payload has another version", "version", int(v))
	}
	var nodes []androidNode
	for i, n := 0, r.count(minNodeBytes); i < n && r.err == nil; i++ {
		var nd androidNode
		nd.ID = r.i32()
		nd.Flags, nd.FlutterActions, nd.Props = r.u32(), r.u32(), r.u32()
		nd.Checked, nd.Expanded, nd.Granularities, nd.RangeType = r.u8(), r.u8(), r.u8(), r.u8()
		nd.MaxTextLength, nd.ScrollChildren, nd.ScrollIndex = r.i32(), r.i32(), r.i32()
		nd.SelectionBase, nd.SelectionExtent = r.i32(), r.i32()
		nd.ScrollPosition, nd.ScrollExtentMax, nd.ScrollExtentMin = r.f32(), r.f32(), r.f32()
		nd.RangeMin, nd.RangeMax, nd.RangeCurrent = r.f32(), r.f32(), r.f32()
		nd.CollectionRows, nd.CollectionColumns = r.i32(), r.i32()
		nd.Left, nd.Top, nd.Right, nd.Bottom = r.f32(), r.f32(), r.f32(), r.f32()
		for j := range nd.Transform {
			nd.Transform[j] = r.f32()
		}
		for j := range nd.HitTestTransform {
			nd.HitTestTransform[j] = r.f32()
		}
		nd.Identifier, nd.ClassName, nd.Value, nd.Label, nd.Tooltip = r.str(), r.str(), r.str(), r.str(), r.str()
		nd.Text, nd.Content, nd.Hint, nd.Increased, nd.Decreased = r.styledList(), r.styledList(), r.styledList(), r.styledList(), r.styledList()
		nd.TraversalChildren, nd.HitTestChildren = r.ids(), r.ids()
		for j, na := 0, r.count(8); j < na && r.err == nil; j++ {
			nd.AndroidActions = append(nd.AndroidActions, androidAction{ID: r.i32(), Label: r.str()})
		}
		nodes = append(nodes, nd)
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.pos != len(payload) {
		return nil, errors.New(tagAndroidSemanticsPayloadInvalid, "the payload has bytes after its last node", "offset", r.pos, "size", len(payload))
	}
	return nodes, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagAndroidSemanticsPayloadInvalid = "AndroidSemanticsPayloadInvalid"
