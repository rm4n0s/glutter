package android

import (
	"encoding/hex"
	"math"
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/ui"
)

// identity is the identity transform, column major.
func identity() [16]float32 {
	return [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// goldenUpdate is the fixture of the golden payload: a root with one button, "OK" with the hint
// "Press", that can be tapped. GlutterAccessibility.java repeats the bytes in its header comment.
func goldenUpdate() *ui.SemanticsUpdate {
	b := ui.NewSemanticsUpdateBuilder()
	nan := math.NaN()
	b.UpdateNode(ui.SemanticsNodeUpdate{
		ID: 0, Rect: ui.Rect{Right: 200, Bottom: 100}, Transform: identity(), HitTestTransform: identity(),
		TextSelectionBase: -1, TextSelectionExtent: -1, PlatformViewID: -1, MaxValueLength: -1, CurrentValueLength: -1,
		ScrollPosition: nan, ScrollExtentMax: nan, ScrollExtentMin: nan,
		ChildrenInTraversalOrder: []int{1}, ChildrenInHitTestOrder: []int{1},
	})
	b.UpdateNode(ui.SemanticsNodeUpdate{
		ID: 1, Flags: ui.SemanticsFlags{IsButton: true, IsEnabled: ui.TristateIsTrue}, Actions: ui.SemanticsActionTap,
		Rect: ui.Rect{Left: 10, Top: 10, Right: 110, Bottom: 50}, Transform: identity(), HitTestTransform: identity(),
		Label: "OK", Hint: "Press", TextSelectionBase: -1, TextSelectionExtent: -1, PlatformViewID: -1,
		MaxValueLength: -1, CurrentValueLength: -1, ScrollPosition: nan, ScrollExtentMax: nan, ScrollExtentMin: nan,
	})
	return b.Build()
}

// goldenHex is the payload of goldenUpdate (649 bytes). GlutterAccessibility.java repeats it.
const goldenHex = "01020000000000000000000000000000000400000000000000ffffffff0000000000000000ffffff" +
	"ffffffffff0000c07f0000c07f0000c07f0000000000000000000000000000000000000000000000" +
	"0000000000000048430000c8420000803f000000000000000000000000000000000000803f000000" +
	"000000000000000000000000000000803f000000000000000000000000000000000000803f000080" +
	"3f000000000000000000000000000000000000803f00000000000000000000000000000000000080" +
	"3f000000000000000000000000000000000000803f0000000011000000616e64726f69642e766965" +
	"772e5669657700000000000000000000000000000000000100000001000000010000000100000000" +
	"00000001000000c8000000010000000f00000000000000ffffffff0000000000000000ffffffffff" +
	"ffffff0000c07f0000c07f0000c07f00000000000000000000000000000000000000000000204100" +
	"0020410000dc42000048420000803f000000000000000000000000000000000000803f0000000000" +
	"00000000000000000000000000803f000000000000000000000000000000000000803f0000803f00" +
	"0000000000000000000000000000000000803f000000000000000000000000000000000000803f00" +
	"0000000000000000000000000000000000803f0000000015000000616e64726f69642e7769646765" +
	"742e427574746f6e00000000020000004f4b00000000000201020000004f4b000000000000000000" +
	"000000020500000050726573730000000000000000000000000201020000004f4b00000000000000" +
	"00000000000205000000507265737300000000000000000000000000000000000000000000010000" +
	"001000000000000000"

// TestSemanticsPayloadGolden checks the encoder against goldenHex, which the header comment of
// GlutterAccessibility.java repeats: the two halves of the format agree on these bytes.
func TestSemanticsPayloadGolden(t *testing.T) {
	update := goldenUpdate()
	nodes := make([]androidNode, len(update.Nodes))
	for i, n := range update.Nodes {
		nodes[i] = newAndroidNode(n, nil)
	}
	got := hex.EncodeToString(encodeSemantics(nodes))
	if got != goldenHex {
		t.Errorf("the payload changed (update GlutterAccessibility.java as well):\n got %s\nwant %s", got, goldenHex)
	}
	raw, err := hex.DecodeString(goldenHex)
	if err != nil {
		t.Fatal(err)
	}
	back, err := decodeSemantics(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != 2 || back[1].Label != "OK" || back[1].ClassName != classButton || back[1].FlutterActions != uint32(ui.SemanticsActionTap) {
		t.Errorf("the golden payload decodes to %+v", back)
	}
}

// A node with every field set comes back as it went.
func TestSemanticsPayloadRoundTrip(t *testing.T) {
	full := androidNode{
		ID: 7, Flags: 0x8000_0001, FlutterActions: 0x3ffffff, Props: 0x7fff, Checked: 2, Expanded: 1, Granularities: 3, RangeType: rangeFloat,
		MaxTextLength: 12, ScrollChildren: 5, ScrollIndex: 2, SelectionBase: 3, SelectionExtent: 4,
		ScrollPosition: 10, ScrollExtentMax: float32(math.Inf(1)), ScrollExtentMin: float32(math.Inf(-1)),
		RangeMin: -1, RangeMax: 1, RangeCurrent: 0.5, CollectionRows: 5, CollectionColumns: 1,
		Left: 1, Top: 2, Right: 3, Bottom: 4, Transform: identity(), HitTestTransform: identity(),
		Identifier: "id", ClassName: classEditText, Value: "v\u00e9\U0001F600", Label: "l", Tooltip: "t",
		Text:    []styledText{{Kind: kindValue, Text: "v"}},
		Content: []styledText{{Kind: kindLabel, Text: "l", URL: "https://example.com", Locale: "en-US", Attrs: []styledAttr{{Start: 0, End: 1, Type: attrSpellOut}, {Start: 0, End: 1, Type: attrLocale, Locale: "el"}}}},
		Hint:    []styledText{{Kind: kindHint, Text: "h"}}, Increased: []styledText{{Kind: kindValue, Text: "up"}}, Decreased: []styledText{{Kind: kindValue, Text: "down"}},
		TraversalChildren: []int{1, 2, 3}, HitTestChildren: []int{3, 2, 1},
		AndroidActions: []androidAction{{ID: androidActionClick, Label: "Open"}, {ID: 267386882}},
	}
	want := []androidNode{{ID: 0}, full}
	got, err := decodeSemantics(encodeSemantics(want))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip:\n got %+v\nwant %+v", got, want)
	}
}

// The smallest node is the size the decoder uses to bound the count of nodes.
func TestSemanticsPayloadMinimumNodeSize(t *testing.T) {
	if got := len(encodeSemantics([]androidNode{{}})); got != 5+minNodeBytes {
		t.Errorf("an empty node takes %d bytes, minNodeBytes says %d", got-5, minNodeBytes)
	}
}

func TestDecodeSemanticsRejectsInvalidPayloads(t *testing.T) {
	good := encodeSemantics([]androidNode{{ID: 1, Label: "x", TraversalChildren: []int{2}}})
	tests := []struct {
		name    string
		payload []byte
		route   string
	}{
		{"empty", nil, "decodeSemantics->wireReader.u8->wireReader.take.AndroidSemanticsPayloadInvalid"},
		{"another version", append([]byte{9}, good[1:]...), "decodeSemantics.AndroidSemanticsPayloadInvalid"},
		{"cut in a node", good[:len(good)/2], "decodeSemantics->wireReader.count.AndroidSemanticsPayloadInvalid"},
		{"cut in the last count", good[:len(good)-2], "wireReader.take.AndroidSemanticsPayloadInvalid"},
		{"more nodes than bytes", []byte{1, 0xff, 0xff, 0xff, 0x7f}, "decodeSemantics->wireReader.count.AndroidSemanticsPayloadInvalid"},
		{"a negative count", []byte{1, 0xff, 0xff, 0xff, 0xff}, "wireReader.count.AndroidSemanticsPayloadInvalid"},
		{"bytes after the last node", append(append([]byte(nil), good...), 0), "decodeSemantics.AndroidSemanticsPayloadInvalid"},
		{"too large", make([]byte, maxSemanticsPayload+1), "decodeSemantics.AndroidSemanticsPayloadInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeSemantics(tt.payload)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}
