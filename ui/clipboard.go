package ui

import "context"

// ClipboardKind is the selection a clipboard call is about.
//
// Flutter: no counterpart (Flutter has the system clipboard only)
type ClipboardKind int

// The values of ClipboardKind.
const (
	// ClipboardKindUnspecified is the zero value: the system clipboard.
	//
	// Flutter: no counterpart
	ClipboardKindUnspecified ClipboardKind = iota
	// ClipboardKindSystem is the clipboard of copy and paste.
	//
	// Flutter: no counterpart
	ClipboardKindSystem
	// ClipboardKindPrimary is the primary selection of Linux desktops: the text
	// that is selected, which a middle click pastes.
	//
	// Flutter: no counterpart
	ClipboardKindPrimary
)

// String names the kind.
//
// Flutter: no counterpart
func (k ClipboardKind) String() string {
	switch k {
	case ClipboardKindSystem, ClipboardKindUnspecified:
		return "ClipboardKind.system"
	case ClipboardKindPrimary:
		return "ClipboardKind.primary"
	}
	return "ClipboardKind.invalid"
}

// The formats of the clipboard content, as MIME types.
const (
	// ClipboardFormatText is plain text (UTF-8).
	//
	// Flutter: services/clipboard.dart Clipboard.kTextPlain
	ClipboardFormatText = "text/plain"
	// ClipboardFormatPNG is an image in the PNG format.
	//
	// Flutter: no counterpart
	ClipboardFormatPNG = "image/png"
	// ClipboardFormatURIList is a list of URIs, such as the files that a file
	// manager copied ("file:///home/me/a.txt").
	//
	// Flutter: no counterpart
	ClipboardFormatURIList = "text/uri-list"
)

// ClipboardContent is what a clipboard holds, in the formats it has. A content
// with several formats is one copy: the same thing in different shapes (a file
// manager offers the URIs and their names as text).
//
// Flutter: no counterpart
type ClipboardContent struct {
	// Text is the plain text; HasText says whether there is any (the empty
	// string is a text).
	Text    string
	HasText bool
	// PNG is the image, in the PNG format; nil for none.
	PNG []byte
	// URIs are the URIs of the files or other resources; nil for none.
	URIs []string
}

// IsEmpty reports whether the content has no format.
//
// Flutter: no counterpart
func (c ClipboardContent) IsEmpty() bool { return !c.HasText && len(c.PNG) == 0 && len(c.URIs) == 0 }

// Has reports whether the content has the format.
//
// Flutter: no counterpart
func (c ClipboardContent) Has(format string) bool {
	switch format {
	case ClipboardFormatText:
		return c.HasText
	case ClipboardFormatPNG:
		return len(c.PNG) > 0
	case ClipboardFormatURIList:
		return len(c.URIs) > 0
	}
	return false
}

// RichClipboardHost is what the engine implements to give access to the
// clipboards beyond the text of the system one: the primary selection, images
// and files. The dispatcher uses it when the host has it; a host without it
// fails the calls with ClipboardUnavailable, except the text of the system
// clipboard, which goes through ClipboardHost. The functions that the methods
// take are called later on the UI thread, never before the method has returned.
//
// Flutter: no counterpart
type RichClipboardHost interface {
	// SetClipboard makes the content the content of the selection of the kind.
	SetClipboard(ctx context.Context, kind ClipboardKind, content ClipboardContent, done func(err error))
	// GetClipboard reads the content of the selection of the kind in the format
	// (one of the ClipboardFormat constants). The content is empty when the
	// selection has no such format.
	GetClipboard(ctx context.Context, kind ClipboardKind, format string, done func(content ClipboardContent, err error))
}
