package macos

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

const dndPID = 4242

func fileItem(uri string) dropItem {
	return dropItem{Types: []string{pbFileURL, "dyn.age8u"}, Strings: map[string]string{pbFileURL: uri}}
}

func TestReadDrop(t *testing.T) {
	tests := []struct {
		name  string
		items []dropItem
		want  dropData
	}{
		{"one file", []dropItem{fileItem("file:///Users/a/one.txt")},
			dropData{Types: []string{"text/uri-list"}, URIs: []string{"file:///Users/a/one.txt"}}},
		{"several files, in order", []dropItem{fileItem("file:///a"), fileItem("file:///b"), fileItem("file:///c")},
			dropData{Types: []string{"text/uri-list"}, URIs: []string{"file:///a", "file:///b", "file:///c"}}},
		{"the same file twice is one", []dropItem{fileItem("file:///a"), fileItem("file:///a")},
			dropData{Types: []string{"text/uri-list"}, URIs: []string{"file:///a"}}},
		{"a URL and its title: the URL is the text when there is none", []dropItem{
			{Types: []string{pbURL}, Strings: map[string]string{pbURL: "https://example.com/x"}}},
			dropData{Types: []string{"text/uri-list"}, URIs: []string{"https://example.com/x"}, Text: "https://example.com/x"}},
		{"a link from a browser has the URL and the text, which stays", []dropItem{
			{Types: []string{pbURL, pbText}, Strings: map[string]string{pbURL: "https://example.com", pbText: "Example"}}},
			dropData{Types: []string{"text/uri-list", "text/plain"}, URIs: []string{"https://example.com"}, Text: "Example"}},
		{"text", []dropItem{{Types: []string{pbText, pbPlainText}, Strings: map[string]string{pbText: "hello\nworld", pbPlainText: "hello\nworld"}}},
			dropData{Types: []string{"text/plain"}, Text: "hello\nworld"}},
		{"the old string type is text too", []dropItem{{Types: []string{pbOldString}, Strings: map[string]string{pbOldString: "old"}}},
			dropData{Types: []string{"text/plain"}, Text: "old"}},
		{"a file whose item also has its URL is one URI", []dropItem{
			{Types: []string{pbFileURL, pbURL}, Strings: map[string]string{pbFileURL: "file:///a", pbURL: "file:///a"}}},
			dropData{Types: []string{"text/uri-list"}, URIs: []string{"file:///a"}}},
		{"a type that the app does not take is not offered", []dropItem{
			{Types: []string{"com.example.private", "dyn.age8u"}}},
			dropData{}},
		{"the images are offered as their types", []dropItem{{Types: []string{pbPNG, pbHTML}}},
			dropData{Types: []string{"image/png", "text/html"}}},
		{"a drag of this app carries its id", []dropItem{
			{Types: []string{pbText, dragMarker(dndPID, 7)}, Strings: map[string]string{pbText: "mine"}}},
			dropData{Types: []string{"text/plain"}, Text: "mine", Local: 7}},
		{"a drag of another process does not", []dropItem{
			{Types: []string{pbText, dragMarker(dndPID+1, 7)}, Strings: map[string]string{pbText: "theirs"}}},
			dropData{Types: []string{"text/plain"}, Text: "theirs"}},
		{"nothing", nil, dropData{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := readDrop(tc.items, dndPID)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("readDrop = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestLocalDragID(t *testing.T) {
	tests := []struct {
		in   string
		want uint64
	}{
		{dragMarker(dndPID, 1), 1},
		{dragMarker(dndPID, 18446744073709551615), 18446744073709551615},
		{dragMarker(dndPID+1, 1), 0},
		{dragMarkerPrefix + "4242", 0},
		{dragMarkerPrefix + "4242-x", 0},
		{dragMarkerPrefix + "42420-1", 0},
		{"public.utf8-plain-text", 0},
		{"", 0},
	}
	for _, tc := range tests {
		if got := localDragID(tc.in, dndPID); got != tc.want {
			t.Errorf("localDragID(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestDropActionsAndOperations(t *testing.T) {
	const copyAction, moveAction, linkAction = ui.DropActionCopy, ui.DropActionMove, ui.DropActionLink
	actions := []struct {
		mask uint64
		want ui.DropActions
	}{
		{dragOpNone, 0},
		{dragOpCopy, copyAction},
		{dragOpMove, moveAction},
		{dragOpLink, linkAction},
		{dragOpCopy | dragOpMove, copyAction | moveAction},
		{dragOpGeneric, copyAction | moveAction | linkAction}, // generic is any
		{dragOpPrivate | dragOpDelete, 0},
	}
	for _, tc := range actions {
		if got := dropActionsOf(tc.mask); got != tc.want {
			t.Errorf("dropActionsOf(%#x) = %v, want %v", tc.mask, got, tc.want)
		}
	}
	ops := []struct {
		accepted ui.DropActions
		mask     uint64
		want     uint64
	}{
		{copyAction, dragOpCopy | dragOpMove | dragOpLink, dragOpCopy},
		{moveAction, dragOpCopy | dragOpMove | dragOpLink, dragOpMove},
		{linkAction, dragOpCopy | dragOpMove | dragOpLink, dragOpLink},
		{copyAction | moveAction, dragOpMove, dragOpMove},        // the source allows only the move
		{copyAction, dragOpMove, dragOpNone},                     // the source does not allow what was accepted
		{ui.DropActionNone, dragOpCopy | dragOpMove, dragOpNone}, // nothing accepted
		{copyAction, dragOpGeneric, dragOpCopy},
		{linkAction, dragOpGeneric, dragOpLink},
		{copyAction | moveAction | linkAction, dragOpGeneric, dragOpCopy}, // copy first
	}
	for _, tc := range ops {
		if got := dragOperationFor(tc.accepted, tc.mask); got != tc.want {
			t.Errorf("dragOperationFor(%v, %#x) = %#x, want %#x", tc.accepted, tc.mask, got, tc.want)
		}
	}
}

func TestDragMaskAndResult(t *testing.T) {
	masks := []struct {
		in   ui.DropActions
		want uint64
	}{
		{ui.DropActionNone, dragOpCopy}, // none means copy
		{ui.DropActionCopy, dragOpCopy},
		{ui.DropActionMove, dragOpMove},
		{ui.DropActionCopy | ui.DropActionLink, dragOpCopy | dragOpLink},
		{ui.DropActionCopy | ui.DropActionMove | ui.DropActionLink, dragOpCopy | dragOpMove | dragOpLink},
	}
	for _, tc := range masks {
		if got := dragMaskOf(tc.in); got != tc.want {
			t.Errorf("dragMaskOf(%v) = %#x, want %#x", tc.in, got, tc.want)
		}
	}
	results := []struct {
		op   uint64
		want ui.DropActions
	}{
		{dragOpNone, ui.DropActionNone},
		{dragOpCopy, ui.DropActionCopy},
		{dragOpGeneric, ui.DropActionCopy},
		{dragOpMove, ui.DropActionMove},
		{dragOpLink, ui.DropActionLink},
		{dragOpDelete, ui.DropActionNone},
	}
	for _, tc := range results {
		if got := dragResultAction(tc.op); got != tc.want {
			t.Errorf("dragResultAction(%#x) = %v, want %v", tc.op, got, tc.want)
		}
	}
}

func TestDragEntries(t *testing.T) {
	marker := dragMarker(dndPID, 5)
	tests := []struct {
		name string
		req  ui.DragRequest
		want []pasteboardEntry
	}{
		{"text", ui.DragRequest{ID: 5, Text: "hi"}, []pasteboardEntry{
			{Types: []string{pbText, pbPlainText, marker}, Strings: map[string]string{pbText: "hi", pbPlainText: "hi", marker: "1"}}}},
		{"a file and a URL", ui.DragRequest{ID: 5, URIs: []string{"file:///a.txt", "https://example.com"}}, []pasteboardEntry{
			{Types: []string{pbFileURL, marker}, Strings: map[string]string{pbFileURL: "file:///a.txt", marker: "1"}},
			{Types: []string{pbURL}, Strings: map[string]string{pbURL: "https://example.com"}}}},
		{"text and files: the marker is on the first item", ui.DragRequest{ID: 5, Text: "t", URIs: []string{"file:///a"}}, []pasteboardEntry{
			{Types: []string{pbText, pbPlainText, marker}, Strings: map[string]string{pbText: "t", pbPlainText: "t", marker: "1"}},
			{Types: []string{pbFileURL}, Strings: map[string]string{pbFileURL: "file:///a"}}}},
		{"an empty URI is left out", ui.DragRequest{ID: 5, URIs: []string{"", "file:///a"}}, []pasteboardEntry{
			{Types: []string{pbFileURL, marker}, Strings: map[string]string{pbFileURL: "file:///a", marker: "1"}}}},
		{"nothing to offer", ui.DragRequest{ID: 5}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dragEntries(tc.req, dndPID)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("dragEntries = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// A drag that this app offers is read back as the same drop: what goes on the pasteboard is what a drop reads.
func TestDragEntriesRoundTrip(t *testing.T) {
	req := ui.DragRequest{ID: 9, Text: "some text", URIs: []string{"file:///Users/a/b.png", "https://example.com/p"}}
	var items []dropItem
	for _, e := range dragEntries(req, dndPID) {
		items = append(items, dropItem(e))
	}
	got := readDrop(items, dndPID)
	want := dropData{
		Types: []string{"text/plain", "text/uri-list"}, Text: "some text",
		URIs: []string{"file:///Users/a/b.png", "https://example.com/p"}, Local: 9,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip = %+v, want %+v", got, want)
	}
}
