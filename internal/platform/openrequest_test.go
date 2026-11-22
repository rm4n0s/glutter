package platform

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// openHandler takes open requests.
type openHandler struct {
	NopHandler
	requests []ui.OpenRequest
}

func (h *openHandler) OnOpenRequest(r ui.OpenRequest) { h.requests = append(h.requests, r) }

func TestOpenRequestGoesToAHandlerThatTakesIt(t *testing.T) {
	r := ui.OpenRequest{URLs: []string{"magnet:?xt=urn:btih:1"}, Files: []string{"/a.torrent"}}
	if OpenRequest(&plainHandler{}, r) {
		t.Error("OpenRequest = true for a handler without OnOpenRequest")
	}
	h := &openHandler{}
	if !OpenRequest(h, r) {
		t.Error("OpenRequest = false for a handler with OnOpenRequest")
	}
	if want := []ui.OpenRequest{r}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("OnOpenRequest got %+v, want %+v", h.requests, want)
	}
}
