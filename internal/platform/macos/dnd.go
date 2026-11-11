package macos

import (
	"slices"
	"strconv"
	"strings"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of drag and drop (SPEC §29.9, §8.7): what the pasteboard of a drag means to
// the framework (the types, the text and the URIs of a DropEvent), how the operations of AppKit and the actions of
// the framework map to each other, and what a drag that the app starts puts on the pasteboard. The calls to AppKit
// are in dnd_darwin.go.

// The pasteboard types that a drop takes and a drag gives (NSPasteboardType values, which are UTIs).
const (
	pbFileURL   = "public.file-url"
	pbURL       = "public.url"
	pbText      = "public.utf8-plain-text"
	pbPlainText = "public.plain-text"
	pbOldString = "NSStringPboardType"
	pbHTML      = "public.html"
	pbPNG       = "public.png"
	pbJPEG      = "public.jpeg"
)

// mimeOf is the MIME type that the framework sees for a pasteboard type, "" for a type that it does not know.
var mimeOf = map[string]string{
	pbFileURL:   "text/uri-list",
	pbURL:       "text/uri-list",
	pbText:      "text/plain",
	pbPlainText: "text/plain",
	pbOldString: "text/plain",
	pbHTML:      "text/html",
	pbPNG:       "image/png",
	pbJPEG:      "image/jpeg",
}

// dragMarkerPrefix starts the pasteboard type that marks a drag as this app's: the prefix, the process id, "-"
// and the id of the DragRequest; the DropEvents of such a drag carry the id as LocalID.
const dragMarkerPrefix = "dev.glutter.drag-"

// dragMarker is the type that marks the drag of the request of this process.
func dragMarker(pid int, id uint64) string {
	return dragMarkerPrefix + strconv.Itoa(pid) + "-" + strconv.FormatUint(id, 10)
}

// localDragID is the id of the request of this process that a pasteboard type marks, 0 when the type is not the
// marker of this process.
func localDragID(pbType string, pid int) uint64 {
	rest, ok := strings.CutPrefix(pbType, dragMarkerPrefix)
	if !ok {
		return 0
	}
	p, id, ok := strings.Cut(rest, "-")
	if !ok || p != strconv.Itoa(pid) {
		return 0
	}
	n, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// dropItem is one item of a drag pasteboard: its types, and the string it has for each type that it has as a
// string (the files and URLs are strings of their types).
type dropItem struct {
	Types   []string
	Strings map[string]string
}

// dropData is what the framework is told of a drag pasteboard.
type dropData struct {
	// Types are the MIME types on offer, once each in the order of the pasteboard.
	Types []string
	// Text is the text of the drop: the first item that has text, else the first URL that is not a file.
	Text string
	// URIs are the files (as file:// URIs) and the URLs of the items, once each in the order of the items.
	URIs []string
	// Local is the id of the DragRequest of this app that started the drag, 0 for another source.
	Local uint64
}

// describeDrag reads the types and the marker of a drag from its items, without the strings: it is what the
// enter and the move of a drag need (the data is read at the drop).
func describeDrag(items []dropItem, pid int) (types []string, local uint64) {
	for _, it := range items {
		for _, t := range it.Types {
			if id := localDragID(t, pid); id != 0 {
				local = id
			}
			if m := mimeOf[t]; m != "" && !slices.Contains(types, m) {
				types = append(types, m)
			}
		}
	}
	return types, local
}

// readDrop reads the text and the URIs of a drop from the items of its pasteboard. A file is the file URL of its item
// (public.file-url, already a file:// URI); a URL is the public.url of an item that has no file URL, so that the
// item of a file that also has the URL of the file is one URI. The text is the first of public.utf8-plain-text,
// public.plain-text and the old string type that an item has; an item that is a URL and has no text gives the URL
// as the text when no other item has text (as on the other desktops, §8.7).
func readDrop(items []dropItem, pid int) dropData {
	d := dropData{}
	d.Types, d.Local = describeDrag(items, pid)
	var link string
	for _, it := range items {
		if d.Text == "" {
			for _, t := range []string{pbText, pbPlainText, pbOldString} {
				if s := it.Strings[t]; s != "" {
					d.Text = s
					break
				}
			}
		}
		uri := it.Strings[pbFileURL]
		if uri == "" {
			uri = it.Strings[pbURL]
			if uri != "" && link == "" {
				link = uri
			}
		}
		if uri != "" && !slices.Contains(d.URIs, uri) {
			d.URIs = append(d.URIs, uri)
		}
	}
	if d.Text == "" && link != "" {
		d.Text = link
	}
	return d
}

// The NSDragOperation bits.
const (
	dragOpNone    uint64 = 0
	dragOpCopy    uint64 = 1 << 0
	dragOpLink    uint64 = 1 << 1
	dragOpGeneric uint64 = 1 << 2
	dragOpPrivate uint64 = 1 << 3
	dragOpMove    uint64 = 1 << 4
	dragOpDelete  uint64 = 1 << 5
)

// dropActionsOf are the actions that the source of a drag allows (draggingSourceOperationMask): copy, move
// and link, each when the mask has it or has "generic", which is "any".
func dropActionsOf(mask uint64) ui.DropActions {
	var a ui.DropActions
	if mask&(dragOpCopy|dragOpGeneric) != 0 {
		a |= ui.DropActionCopy
	}
	if mask&(dragOpMove|dragOpGeneric) != 0 {
		a |= ui.DropActionMove
	}
	if mask&(dragOpLink|dragOpGeneric) != 0 {
		a |= ui.DropActionLink
	}
	return a
}

// dragOperationFor is the operation that answers a drag: the action that the framework accepted, if the source allows
// it, in the order copy, move, link; none when it accepted nothing or the source allows none of what it accepted.
func dragOperationFor(accepted ui.DropActions, mask uint64) uint64 {
	allowed := dropActionsOf(mask)
	switch {
	case accepted.Has(ui.DropActionCopy) && allowed.Has(ui.DropActionCopy):
		return dragOpCopy
	case accepted.Has(ui.DropActionMove) && allowed.Has(ui.DropActionMove):
		return dragOpMove
	case accepted.Has(ui.DropActionLink) && allowed.Has(ui.DropActionLink):
		return dragOpLink
	}
	return dragOpNone
}

// dragMaskOf is the operation mask of a drag that the app starts, from the actions of the request: none means copy
// (ui.DragRequest).
func dragMaskOf(a ui.DropActions) uint64 {
	if a == ui.DropActionNone {
		return dragOpCopy
	}
	var m uint64
	if a.Has(ui.DropActionCopy) {
		m |= dragOpCopy
	}
	if a.Has(ui.DropActionMove) {
		m |= dragOpMove
	}
	if a.Has(ui.DropActionLink) {
		m |= dragOpLink
	}
	return m
}

// dragResultAction is the action that a finished drag that the app started performed: the operation that the target
// chose (generic counts as copy), none when the drag was cancelled or not accepted.
func dragResultAction(op uint64) ui.DropActions {
	switch {
	case op&dragOpMove != 0:
		return ui.DropActionMove
	case op&dragOpLink != 0:
		return ui.DropActionLink
	case op&(dragOpCopy|dragOpGeneric) != 0:
		return ui.DropActionCopy
	}
	return ui.DropActionNone
}

// pasteboardEntry is one item that a drag puts on the pasteboard: the string for each of its types, in order.
type pasteboardEntry struct {
	Types   []string
	Strings map[string]string
}

// dragEntries make the pasteboard items of a request: the text as an item of public.utf8-plain-text and
// public.plain-text; each file URI as an item of public.file-url, each other URI as an item of public.url; and
// the marker of the drag on the first item, so that a drop in this app knows the drag as its own. A request with no
// text and no URI has no entries. A URI that is empty is left out.
func dragEntries(req ui.DragRequest, pid int) []pasteboardEntry {
	var out []pasteboardEntry
	if req.Text != "" {
		out = append(out, pasteboardEntry{Types: []string{pbText, pbPlainText}, Strings: map[string]string{pbText: req.Text, pbPlainText: req.Text}})
	}
	for _, uri := range req.URIs {
		switch {
		case uri == "":
		case strings.HasPrefix(uri, "file:"):
			out = append(out, pasteboardEntry{Types: []string{pbFileURL}, Strings: map[string]string{pbFileURL: uri}})
		default:
			out = append(out, pasteboardEntry{Types: []string{pbURL}, Strings: map[string]string{pbURL: uri}})
		}
	}
	if len(out) > 0 {
		m := dragMarker(pid, req.ID)
		out[0].Types = append(out[0].Types, m)
		out[0].Strings[m] = "1"
	}
	return out
}
