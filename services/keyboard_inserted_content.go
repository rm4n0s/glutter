package services

import (
	"bytes"
	"fmt"
)

// KeyboardInsertedContent is a content, such as an image, that the input method
// inserted into a text field.
//
// Flutter: services/keyboard_inserted_content.dart KeyboardInsertedContent
type KeyboardInsertedContent struct {
	// MimeType is the MIME type of the content, such as "image/png".
	//
	// Flutter: services/keyboard_inserted_content.dart KeyboardInsertedContent.mimeType
	MimeType string
	// URI is the URI of the content.
	//
	// Flutter: services/keyboard_inserted_content.dart KeyboardInsertedContent.uri
	URI string
	// Data is the bytes of the content, if the input method gave them.
	//
	// Flutter: services/keyboard_inserted_content.dart KeyboardInsertedContent.data
	Data []byte
}

// NewKeyboardInsertedContent creates the description of an inserted content.
//
// Flutter: services/keyboard_inserted_content.dart KeyboardInsertedContent.new
func NewKeyboardInsertedContent(mimeType, uri string, data []byte) KeyboardInsertedContent {
	return KeyboardInsertedContent{MimeType: mimeType, URI: uri, Data: data}
}

// HasData is whether the content has bytes.
//
// Flutter: services/keyboard_inserted_content.dart KeyboardInsertedContent.hasData
func (c KeyboardInsertedContent) HasData() bool { return len(c.Data) > 0 }

// Equal reports whether two descriptions have the same type, URI and data.
//
// Flutter: services/keyboard_inserted_content.dart KeyboardInsertedContent.==
func (c KeyboardInsertedContent) Equal(other KeyboardInsertedContent) bool {
	return c.MimeType == other.MimeType && c.URI == other.URI && bytes.Equal(c.Data, other.Data)
}

// String returns what Dart prints for the content.
func (c KeyboardInsertedContent) String() string {
	return fmt.Sprintf("KeyboardInsertedContent(%s, %s, %v)", c.MimeType, c.URI, c.Data)
}
