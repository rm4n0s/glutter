//go:build darwin && !ios && glutter_display

package macos

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/ui"
)

func (s *session) desktop() *desktop {
	d, ok := s.p.Desktop().(*desktop)
	if !ok {
		s.t.Fatal("Desktop() is not the services of macOS")
	}
	return d
}

func TestDesktopServicesAreProvided(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		d := s.desktop()
		if s.p.Desktop() != mobile.Services(d) {
			t.Error("Desktop() made a second value")
		}
		if d.DesktopName() != "macOS" {
			t.Errorf("DesktopName = %q", d.DesktopName())
		}
		var svc any = d
		if _, ok := svc.(platform.TrayHost); !ok {
			t.Error("no TrayHost")
		}
		if _, ok := svc.(platform.FileDialogHost); !ok {
			t.Error("no FileDialogHost")
		}
	})
}

func TestOpenURIGoesThroughTheWorkspace(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		d := s.desktop()
		var opened []string
		var mu sync.Mutex
		d.openURL = func(uri string) (bool, string) {
			mu.Lock()
			opened = append(opened, uri)
			mu.Unlock()
			return uri != "https://nobody.example/", "no program"
		}
		ctx := context.Background()
		if err := d.OpenURI(ctx, "https://example.com/a"); err != nil {
			t.Errorf("OpenURI: %v", err)
		}
		// The checks come first: nothing blocked reaches the workspace.
		for uri, route := range map[string]string{
			"javascript:alert(1)":        "checkOpenURI." + tagOpenURISchemeBlocked,
			"":                           "checkOpenURI." + tagOpenURIInvalid,
			"file:///Applications/X.app": "checkOpenURI." + tagOpenURISchemeBlocked,
		} {
			err := d.OpenURI(ctx, uri)
			if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute(route) {
				t.Errorf("OpenURI(%q) = %v, want %s", uri, err, route)
			}
		}
		err := d.OpenURI(ctx, "https://nobody.example/")
		if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("desktop.OpenURI."+tagOpenURIUnavailable) {
			t.Errorf("a URI with no program: %v", err)
		}
		mu.Lock()
		defer mu.Unlock()
		if !slices.Equal(opened, []string{"https://example.com/a", "https://nobody.example/"}) {
			t.Errorf("the workspace was asked to open %v", opened)
		}
	})
}

func TestOpenURIOfAnInvalidURLIsUnavailable(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		d := s.desktop()
		// The real workspace is not asked for a URL it can read: the conversion fails first for a text that is not
		// a URL, which cannot be told from a valid one by the checks.
		ok, reason := d.workspaceOpen("http://exa mple.com/ \x7f")
		if ok || reason == "" {
			t.Errorf("workspaceOpen of a text that is not a URL = %v, %q", ok, reason)
		}
	})
}

func TestNotificationsWithoutABundleAreUnavailable(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		d := s.desktop()
		if bundleID() != "" {
			t.Skip("the test binary is in a bundle with an identifier")
		}
		_, err := d.ShowNotification(context.Background(), mobile.NotificationSpec{Title: "x"})
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagNotificationsUnavailable || !appErr.HasRoute("notifier.show."+tagNotificationsUnavailable) {
			t.Errorf("ShowNotification = %v, want NotificationsUnavailable", err)
		}
		if err := d.CloseNotification(context.Background(), 99); err != nil {
			t.Errorf("closing a notification that is gone: %v", err)
		}
	})
}

// menuTitles are the titles of the items of an NSMenu.
func menuTitles(menu objc.ID) []string {
	items := send[objc.ID](menu, "itemArray")
	var out []string
	for i := range objcx.ArrayCount(items) {
		it := objcx.ArrayAt(items, i)
		if send[bool](it, "isSeparatorItem") {
			out = append(out, "-")
			continue
		}
		out = append(out, objcx.String(send[objc.ID](it, "title")))
	}
	return out
}

func TestTrayIsInTheStatusBarWithItsMenu(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		d := s.desktop()
		var pressed []string
		var activated, secondary atomic.Int32
		img := make([]byte, 32*32*4)
		for i := 3; i < len(img); i += 4 {
			img[i] = 255
		}
		spec := platform.TraySpec{
			ID: "t", Title: "Tray", Tooltip: "My app", TooltipBody: "is running",
			Icon: ui.WindowIcon{Images: []ui.WindowIconImage{{Size: 32, Scale: 2, RGBA: img}}},
			Menu: []platform.TrayMenuItem{
				{Label: "Open", OnPressed: func() { pressed = append(pressed, "open") }},
				{Separator: true},
				{Label: "Sound", Checkable: true, Checked: true, OnPressed: func() { pressed = append(pressed, "sound") }},
				{Label: "More", Children: []platform.TrayMenuItem{{Label: "Deep", OnPressed: func() { pressed = append(pressed, "deep") }}}},
			},
			OnActivate: func() { activated.Add(1) }, OnSecondaryActivate: func() { secondary.Add(1) },
		}
		s.onUI(func() {
			tray, err := d.NewTray(spec)
			if err != nil {
				t.Fatalf("NewTray: %v", err)
			}
			ti := tray.(*trayIcon)
			button := send[objc.ID](ti.item, "button")
			if got := objcx.String(send[objc.ID](button, "toolTip")); got != "My app\nis running" {
				t.Errorf("tooltip = %q", got)
			}
			if img := send[objc.ID](button, "image"); img == 0 || !send[bool](img, "isTemplate") {
				t.Error("the icon is not a template image")
			}
			menu := send[objc.ID](ti.item, "menu")
			if got := menuTitles(menu); !slices.Equal(got, []string{"Open", "-", "Sound", "More"}) {
				t.Errorf("menu = %v", got)
			}
			sound := send[objc.ID](menu, "itemArray")
			if st := send[int64](objcx.ArrayAt(sound, 2), "state"); st != menuItemStateOn {
				t.Errorf("the checked item has the state %d", st)
			}
			// A second tray is refused, with its tag and route.
			if _, err := d.NewTray(spec); err == nil {
				t.Error("a second tray was made")
			} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagTrayDuplicate || !appErr.HasRoute("desktop.NewTray."+tagTrayDuplicate) {
				t.Errorf("second tray: %v", err)
			}
			// The items of the menu run their functions, the submenu's too.
			menu.Send(sel("performActionForItemAtIndex:"), int64(0))
			menu.Send(sel("performActionForItemAtIndex:"), int64(2))
			deep := send[objc.ID](objcx.ArrayAt(sound, 3), "submenu")
			deep.Send(sel("performActionForItemAtIndex:"), int64(0))
			if !slices.Equal(pressed, []string{"open", "sound", "deep"}) {
				t.Errorf("pressed = %v", pressed)
			}
			// Update with no menu: the icon takes clicks.
			spec.Menu = nil
			tray.Update(spec)
			if send[objc.ID](ti.item, "menu") != 0 {
				t.Error("the menu stayed after an update without one")
			}
			button.Send(sel("performClick:"), objc.ID(0))
			if activated.Load() != 1 {
				t.Errorf("OnActivate was called %d times", activated.Load())
			}
			tray.Close()
			tray.Close()
			if d.tray != nil {
				t.Error("the tray is still the app's after Close")
			}
			// The next tray can be made.
			t2, err := d.NewTray(platform.TraySpec{Title: "Again"})
			if err != nil {
				t.Fatalf("NewTray after Close: %v", err)
			}
			if title := objcx.String(send[objc.ID](send[objc.ID](t2.(*trayIcon).item, "button"), "title")); title != "Again" {
				t.Errorf("an icon with no picture shows its title: %q", title)
			}
			t2.Close()
		})
		if secondary.Load() != 0 {
			t.Error("OnSecondaryActivate was called for a left click")
		}
	})
}

// findPanel finds the open or save panel that is on the screen.
func findPanel() objc.ID {
	list := send[objc.ID](objc.ID(objcx.MustClass("NSApplication")).Send(objcx.Sel("sharedApplication")), "windows")
	panelClass := objc.ID(objcx.MustClass("NSSavePanel"))
	for i := range objcx.ArrayCount(list) {
		w := objcx.ArrayAt(list, i)
		if objc.Send[bool](w, objcx.Sel("isKindOfClass:"), panelClass) && send[bool](w, "isVisible") {
			return w
		}
	}
	return 0
}

// answerPanel waits for a panel to show and cancels it: the panels are views of another process, which only
// cancel: reaches (ok: is not implemented for a remote panel).
func (s *session) answerPanel() {
	s.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var done bool
		s.onUI(func() {
			if p := findPanel(); p != 0 {
				p.Send(sel("cancel:"), objc.ID(0))
				done = true
			}
		})
		if done {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	s.t.Error("no panel was shown in 10 s")
}

// fakePanel answers URL and URLs as a panel does after the user chose: the answer of a panel that was shown cannot be
// made by a test, so panelResult reads this one.
type fakePanel struct{ paths []string }

var fakePanelClass = sync.OnceValues(func() (*objcx.RegisteredClass, error) {
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "FakePanel", Super: "NSObject",
		Methods: []objcx.Method{
			{Name: "URL", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
				f, _ := objcx.LookupAs[*fakePanel](objcx.HandleOf(self))
				if len(f.paths) == 0 {
					return 0
				}
				return objcx.Autorelease(objcx.NewFileURL(f.paths[0]))
			}},
			{Name: "URLs", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
				f, _ := objcx.LookupAs[*fakePanel](objcx.HandleOf(self))
				var urls []objc.ID
				for _, p := range f.paths {
					urls = append(urls, objcx.NewFileURL(p))
				}
				arr := objcx.NewArray(urls...)
				for _, u := range urls {
					objcx.Release(u)
				}
				return objcx.Autorelease(arr)
			}},
		},
	})
})

func TestPanelResultReadsWhatTheUserChose(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		d := s.desktop()
		cls, err := fakePanelClass()
		if err != nil {
			t.Fatal(err)
		}
		tests := []struct {
			name  string
			spec  platform.FileDialogSpec
			paths []string
			resp  int64
			want  platform.FileDialogResult
		}{
			{"save, with the filter of the file", platform.FileDialogSpec{Kind: platform.FileDialogSave, Filters: []platform.FileFilter{
				{Name: "Images", Patterns: []string{"*.png"}}, {Name: "Text", Patterns: []string{"*.txt"}}}},
				[]string{"/tmp/out.TXT"}, modalResponseOK, platform.FileDialogResult{Paths: []string{"/tmp/out.TXT"}, Filter: "Text"}},
			{"open several", platform.FileDialogSpec{Kind: platform.FileDialogOpen, Multiple: true},
				[]string{"/tmp/a", "/tmp/b"}, modalResponseOK, platform.FileDialogResult{Paths: []string{"/tmp/a", "/tmp/b"}}},
			{"a folder has no filter", platform.FileDialogSpec{Kind: platform.FileDialogOpen, Directory: true, Filters: []platform.FileFilter{{Name: "x", Patterns: []string{"*.x"}}}},
				[]string{"/tmp/dir.x"}, modalResponseOK, platform.FileDialogResult{Paths: []string{"/tmp/dir.x"}}},
			{"save several: the folder with the names", platform.FileDialogSpec{Kind: platform.FileDialogSaveMany, Names: []string{"a.png", "b.png"}},
				[]string{"/tmp/out"}, modalResponseOK, platform.FileDialogResult{Paths: []string{"/tmp/out/a.png", "/tmp/out/b.png"}}},
			{"cancel", platform.FileDialogSpec{Kind: platform.FileDialogOpen}, []string{"/tmp/a"}, 0, platform.FileDialogResult{Cancelled: true}},
		}
		for _, tc := range tests {
			s.onUI(func() {
				panel := cls.New(objcx.NewHandle(&fakePanel{paths: tc.paths}))
				got := d.panelResult(panel, tc.resp, tc.spec, planOf(tc.spec))
				objcx.Release(panel)
				if got.Cancelled != tc.want.Cancelled || !slices.Equal(got.Paths, tc.want.Paths) || got.Filter != tc.want.Filter {
					t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
				}
			})
		}
	})
}

func TestPanelsAreMadeFromTheSpec(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		d := s.desktop()
		dir, _ := filepath.EvalSymlinks(t.TempDir())
		s.onUI(func() {
			spec := platform.FileDialogSpec{Kind: platform.FileDialogSave, Title: "Save it", AcceptLabel: "Keep", Folder: dir, FileName: "out.txt",
				Filters: []platform.FileFilter{{Name: "Text", Patterns: []string{"*.txt", "*.md"}}}}
			p := d.makePanel(spec, planOf(spec))
			defer objcx.Release(p)
			if got := objcx.String(send[objc.ID](p, "message")); got != "Save it" {
				t.Errorf("message = %q", got)
			}
			if got := objcx.String(send[objc.ID](p, "prompt")); got != "Keep" {
				t.Errorf("prompt = %q", got)
			}
			if got := objcx.String(send[objc.ID](p, "nameFieldStringValue")); got != "out.txt" {
				t.Errorf("name = %q", got)
			}
			if got := pathOfURL(send[objc.ID](p, "directoryURL")); got != dir {
				t.Errorf("folder = %q, want %q", got, dir)
			}
			if n := objcx.ArrayCount(send[objc.ID](p, "allowedContentTypes")); n != 2 {
				t.Errorf("%d content types, want 2 (txt and md)", n)
			}
			open := platform.FileDialogSpec{Kind: platform.FileDialogOpen, Multiple: true, Directory: true}
			o := d.makePanel(open, planOf(open))
			defer objcx.Release(o)
			if !send[bool](o, "allowsMultipleSelection") || !send[bool](o, "canChooseDirectories") || send[bool](o, "canChooseFiles") {
				t.Error("an open panel for folders, several of them, has the wrong options")
			}
			many := platform.FileDialogSpec{Kind: platform.FileDialogSaveMany, Names: []string{"a"}}
			m := d.makePanel(many, planOf(many))
			defer objcx.Release(m)
			if send[bool](m, "canChooseFiles") || !send[bool](m, "canChooseDirectories") {
				t.Error("the panel that saves several files does not choose a folder")
			}
		})
	})
}

func TestFileDialogCancelReturnsCancelled(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		d := s.desktop()
		ch := make(chan platform.FileDialogResult, 1)
		go func() {
			res, err := d.ShowFileDialog(context.Background(), platform.FileDialogSpec{Kind: platform.FileDialogOpen})
			if err != nil {
				t.Errorf("ShowFileDialog: %v", err)
			}
			ch <- res
		}()
		s.answerPanel()
		select {
		case res := <-ch:
			if !res.Cancelled || len(res.Paths) != 0 {
				t.Errorf("a cancelled dialog = %+v", res)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("the dialog did not return")
		}
	})
}

func TestFileDialogEndsWithItsContext(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		d := s.desktop()
		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() {
			_, err := d.ShowFileDialog(ctx, platform.FileDialogSpec{Kind: platform.FileDialogOpen})
			errCh <- err
		}()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			var shown bool
			s.onUI(func() { shown = findPanel() != 0 })
			if shown {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		cancel()
		select {
		case err := <-errCh:
			if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagFileChooserFailed || !appErr.HasRoute("desktop.ShowFileDialog."+tagFileChooserFailed) {
				t.Errorf("the dialog of a context that ended: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("the dialog did not end with its context")
		}
		var left bool
		s.onUI(func() { left = findPanel() != 0 })
		if left {
			t.Error("the panel is still on the screen")
		}
	})
}
