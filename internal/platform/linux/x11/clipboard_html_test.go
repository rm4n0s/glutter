//go:build linux && !android && glutter_display

package x11

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestClipboardPassesHTMLNextToText(t *testing.T) {
	big := "<p>" + strings.Repeat("café 世界 ", 100000) + "</p>" // over incrThreshold
	if len(big) <= incrThreshold {
		t.Fatalf("the fragment is only %d bytes", len(big))
	}
	tests := []struct {
		name    string
		content ui.ClipboardContent
	}{
		{"text and HTML", ui.ClipboardContent{Text: "naïve café", HasText: true, HTML: "<b>naïve</b> café"}},
		{"HTML only", ui.ClipboardContent{HTML: "<i>世界</i>"}},
		{"a fragment sent in chunks (INCR)", ui.ClipboardContent{Text: "t", HasText: true, HTML: big}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, reader := startClipboard(t), startClipboard(t)
			if err := setContent(t, owner, ui.ClipboardKindSystem, tt.content); err != nil {
				t.Fatal(err)
			}
			if r := getContent(t, reader, ui.ClipboardKindSystem, ui.ClipboardFormatHTML); r.err != nil || r.content.HTML != tt.content.HTML {
				t.Errorf("HTML = %d bytes, err %v, want %d bytes", len(r.content.HTML), r.err, len(tt.content.HTML))
			}
			r := getContent(t, reader, ui.ClipboardKindSystem, ui.ClipboardFormatText)
			if r.err != nil || r.content.HasText != tt.content.HasText || r.content.Text != tt.content.Text {
				t.Errorf("text = %+v, want %q (%v)", r, tt.content.Text, tt.content.HasText)
			}
			// The owner reads its own content back.
			if r := getContent(t, owner, ui.ClipboardKindSystem, ui.ClipboardFormatHTML); r.content.HTML != tt.content.HTML {
				t.Errorf("own HTML = %d bytes, want %d", len(r.content.HTML), len(tt.content.HTML))
			}
		})
	}
}

func TestClipboardHTMLThatTheOwnerLacksIsEmpty(t *testing.T) {
	owner, reader := startClipboard(t), startClipboard(t)
	if err := setContent(t, owner, ui.ClipboardKindSystem, ui.ClipboardContent{Text: "text only", HasText: true}); err != nil {
		t.Fatal(err)
	}
	if r := getContent(t, reader, ui.ClipboardKindSystem, ui.ClipboardFormatHTML); r.err != nil || !r.content.IsEmpty() {
		t.Errorf("HTML = %+v, want empty", r)
	}
}
