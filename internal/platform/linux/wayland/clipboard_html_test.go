//go:build linux && !android

package wayland

import (
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

func TestClipboardReadsHTMLOfAnotherApp(t *testing.T) {
	const fragment = "<b>Grüße</b> 世界 <i>café</i>"
	tests := []struct {
		name      string
		offered   map[string][]byte
		wantHTML  string
		wantAsked string
	}{
		{"charset form", map[string][]byte{"text/html;charset=utf-8": []byte(fragment), "UTF8_STRING": []byte("text")}, fragment, "text/html;charset=utf-8"},
		{"bare form", map[string][]byte{"text/html": []byte(fragment)}, fragment, "text/html"},
		{"both forms, the charset one is read", map[string][]byte{"text/html": []byte("<p>bare</p>"), "text/html;charset=utf-8": []byte(fragment)}, fragment, "text/html;charset=utf-8"},
		{"invalid UTF-8 becomes U+FFFD", map[string][]byte{"text/html": []byte("<b>a\xffb</b>")}, "<b>a�b</b>", "text/html"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := clipboardEnv(t, fakecomp.Config{})
			e.srv.OfferSelectionData(tt.offered)
			e.settle()
			r := e.getContent(ui.ClipboardKindSystem, ui.ClipboardFormatHTML)
			if r.err != nil || r.content.HTML != tt.wantHTML || r.content.HasText {
				t.Errorf("HTML = %+v, want %q", r, tt.wantHTML)
			}
			if !strings.Contains(e.srv.Joined(), "wl_data_offer.receive mime="+tt.wantAsked) {
				t.Errorf("the app did not ask for %q", tt.wantAsked)
			}
		})
	}
}

func TestClipboardHTMLThatTheOwnerLacksIsEmpty(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{})
	e.srv.OfferSelectionData(map[string][]byte{"UTF8_STRING": []byte("only text")})
	e.settle()
	if r := e.getContent(ui.ClipboardKindSystem, ui.ClipboardFormatHTML); r.err != nil || !r.content.IsEmpty() {
		t.Errorf("HTML = %+v, want empty", r)
	}
	if strings.Contains(e.srv.Joined(), "wl_data_offer.receive mime=text/html") {
		t.Error("the app asked for a format that the owner does not offer")
	}
}

func TestClipboardSetContentOffersHTMLNextToText(t *testing.T) {
	const fragment = "<p>naïve <b>café</b></p>"
	tests := []struct {
		name      string
		content   ui.ClipboardContent
		wantMimes []string
		lackMimes []string
	}{
		{
			"text and HTML",
			ui.ClipboardContent{Text: "naïve café", HasText: true, HTML: fragment},
			[]string{"text/plain;charset=utf-8", "text/plain", "UTF8_STRING", "text/html;charset=utf-8", "text/html"},
			nil,
		},
		{
			"HTML only",
			ui.ClipboardContent{HTML: fragment},
			[]string{"text/html;charset=utf-8", "text/html"},
			[]string{"text/plain", "UTF8_STRING"},
		},
		{
			"text only",
			ui.ClipboardContent{Text: "x", HasText: true},
			[]string{"text/plain"},
			[]string{"text/html", "text/html;charset=utf-8"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := clipboardEnv(t, fakecomp.Config{})
			if err := e.setContent(ui.ClipboardKindSystem, tt.content); err != nil {
				t.Fatal(err)
			}
			mimes := e.srv.ClientSelectionMimes(false)
			for _, m := range tt.wantMimes {
				if !slices.Contains(mimes, m) {
					t.Errorf("the source lacks %q: %v", m, mimes)
				}
			}
			for _, m := range tt.lackMimes {
				if slices.Contains(mimes, m) {
					t.Errorf("the source has %q: %v", m, mimes)
				}
			}
			for _, m := range []string{"text/html;charset=utf-8", "text/html"} {
				if tt.content.HTML == "" {
					break
				}
				if data, err := e.srv.ReadClientSelection(false, m); err != nil || string(data) != fragment {
					t.Errorf("%s = %q, %v", m, data, err)
				}
			}
			if tt.content.HasText {
				if data, err := e.srv.ReadClientSelection(false, "UTF8_STRING"); err != nil || string(data) != tt.content.Text {
					t.Errorf("UTF8_STRING = %q, %v", data, err)
				}
			}
			// What the app owns comes back without a round trip, in the same shape.
			r := e.getContent(ui.ClipboardKindSystem, ui.ClipboardFormatHTML)
			if r.err != nil || r.content.HTML != tt.content.HTML {
				t.Errorf("own HTML = %+v, want %q", r, tt.content.HTML)
			}
			if strings.Contains(e.srv.Joined(), "wl_data_offer.receive") {
				t.Error("the app asked the compositor for its own content")
			}
		})
	}
}
