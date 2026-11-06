package model

import (
	"bytes"
	"path"
	"strings"
)

// DetectImageType returns the media type of an image from the extension of its name, or from its first bytes when the name does
// not say: "image/png", "image/jpeg", "image/gif", "image/bmp", "image/webp", or "" when it is none of them.
//
// Flutter: no counterpart
func DetectImageType(name string, data []byte) string {
	switch strings.ToLower(path.Ext(strings.ReplaceAll(name, "\\", "/"))) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg", ".jpe":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".webp":
		return "image/webp"
	}
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return "image/png"
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg"
	case bytes.HasPrefix(data, []byte("GIF8")):
		return "image/gif"
	case bytes.HasPrefix(data, []byte("BM")):
		return "image/bmp"
	case len(data) >= 12 && bytes.HasPrefix(data, []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return "image/webp"
	}
	return ""
}
