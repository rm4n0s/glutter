//go:build darwin && !ios && glutter_display

package macos

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// The four-character codes of the Apple events that ask an app to open something (AE/AppleEvents.h,
// AERegistry.h, the Internet suite).
const (
	aeGetURL        = 'G'<<24 | 'U'<<16 | 'R'<<8 | 'L' // kInternetEventClass and kAEGetURL
	aeOpenDocuments = 'o'<<24 | 'd'<<16 | 'o'<<8 | 'c' // kAEOpenDocuments
	aeDirectObject  = '-'<<24 | '-'<<16 | '-'<<8 | '-' // keyDirectObject
)

// sendOpenEvent sends this process an Apple event with a direct object, as Launch Services sends the event of
// a link (GURL) or of documents (odoc) to the app that handles them, and returns the error of the send (0 for
// none). The Apple Event Manager dispatches an event to this process at once, on the main thread.
func sendOpenEvent(class_, id uint32, direct objc.ID) int64 {
	aed := class("NSAppleEventDescriptor")
	ev := aed.Send(sel("appleEventWithEventClass:eventID:targetDescriptor:returnID:transactionID:"),
		class_, id, aed.Send(sel("currentProcessDescriptor")), int16(-1), int32(0))
	ev.Send(sel("setParamDescriptor:forKeyword:"), direct, uint32(aeDirectObject))
	var nserr objc.ID
	const noReply = 1 // NSAppleEventSendNoReply (kAENoReply)
	ev.Send(sel("sendEventWithOptions:timeout:error:"), uint64(noReply), float64(5), &nserr)
	if nserr == 0 {
		return 0
	}
	return send[int64](nserr, "code")
}

// aeTimeout is errAETimeout. AppKit's handler of the documents event suspends the event to reply after the
// delegate opened the files, which a sender in the same process sees as this error though the delegate was
// called; Launch Services, in another process, gets the reply.
const aeTimeout = -1712

// TestOpenEventsReachTheHandler is Theatro's issue (step 56.1): the Apple events of a link of the app's scheme
// (GURL) and of its documents (odoc), which macOS sends where other systems give arguments, reach the handler
// as open requests: the link as it is written, the documents as paths.
func TestOpenEventsReachTheHandler(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	files := []string{filepath.Join(dir, "one file.torrent"), filepath.Join(dir, "δύο.torrent")}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("d4:infod4:name1:xee"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	const link = "magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335aa7c1367a88a&dn=A%20name"
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
		s.onUI(func() {
			objcx.WithPool(func() {
				aed := class("NSAppleEventDescriptor")
				str := objcx.NewString(link)
				defer objcx.Release(str)
				if code := sendOpenEvent(aeGetURL, aeGetURL, aed.Send(sel("descriptorWithString:"), str)); code != 0 {
					t.Errorf("the event of the link was not sent: error %d", code) // not Fatalf: this is the main thread
				}

				list := aed.Send(sel("listDescriptor"))
				for i, f := range files {
					path := objcx.NewString(f)
					url := class("NSURL").Send(sel("fileURLWithPath:"), path)
					objcx.Release(path)
					list.Send(sel("insertDescriptor:atIndex:"), aed.Send(sel("descriptorWithFileURL:"), url), int64(i+1))
				}
				if code := sendOpenEvent(aeCoreEventClass, aeOpenDocuments, list); code != 0 && code != aeTimeout {
					t.Errorf("the event of the documents was not sent: error %d", code)
				}
			})
		})
		if !s.waitFor("the open requests", 5*time.Second, func() bool { return len(h.opensHeard()) >= 2 }) {
			t.Errorf("open requests = %+v, want the link and then the files", h.opensHeard())
			return
		}
		got := h.opensHeard()
		if want := (ui.OpenRequest{URLs: []string{link}}); len(got) != 2 || !reflect.DeepEqual(got[0], want) {
			t.Errorf("open requests = %+v, want two, the first %+v", got, want)
			return
		}
		// The paths are macOS's: it may write a name in another Unicode form than the one it was made with, so
		// the files are compared, not the text.
		if len(got[1].URLs) != 0 || len(got[1].Files) != len(files) {
			t.Fatalf("the request of the documents = %+v, want the %d files and no URL", got[1], len(files))
		}
		for i, f := range files {
			a, errA := os.Stat(f)
			b, errB := os.Stat(got[1].Files[i])
			if errA != nil || errB != nil || !os.SameFile(a, b) {
				t.Errorf("file %d = %q (%v), want the file %q (%v)", i, got[1].Files[i], errB, f, errA)
			}
			if !filepath.IsAbs(got[1].Files[i]) {
				t.Errorf("file %d = %q is not an absolute path", i, got[1].Files[i])
			}
		}
	})
}

// TestOpenRequestBeforeRunWaitsForTheHandler: a request that comes while Run has no handler is not lost.
func TestOpenRequestBeforeRunWaitsForTheHandler(t *testing.T) {
	h := &recorder{}
	want := ui.OpenRequest{URLs: []string{"magnet:?xt=urn:btih:1"}, Files: []string{"/tmp/a.torrent"}}
	var early *Platform
	runAppWith(t, Options{}, h, func(p *Platform) {
		early = p
		p.deliverOpenRequest(want)
	}, func(s *session) {
		if s.p != early {
			t.Error("the body runs on another platform than the one that got the request")
		}
		if !s.waitFor("the request that waited", 5*time.Second, func() bool { return len(h.opensHeard()) == 1 }) {
			t.Errorf("open requests = %+v, want [%+v]", h.opensHeard(), want)
			return
		}
		if got := h.opensHeard()[0]; !reflect.DeepEqual(got, want) {
			t.Errorf("open request = %+v, want %+v", got, want)
		}
	})
}
