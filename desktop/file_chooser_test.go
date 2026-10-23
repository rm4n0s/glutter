package desktop_test

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
	"github.com/rm4n0s/glutter/widgets"
)

const chooser = "org.freedesktop.portal.FileChooser."

// scripted makes the fake bus answer a FileChooser method with the response, and
// keeps what the app asked.
type scripted struct {
	mu   sync.Mutex
	args []any
}

func (s *scripted) answer(t *testing.T, b *fakebus.Bus, method string, code uint32, results map[string]godbus.Variant) {
	t.Helper()
	b.Handle(dbus.PortalDest, dbus.PortalPath, chooser+method, func(c fakebus.Call) ([]any, error) {
		s.mu.Lock()
		s.args = c.Args
		s.mu.Unlock()
		path := dbus.PortalRequestPath(unique, tokenIn(t, c.Args[2].(map[string]godbus.Variant)))
		b.Emit(&godbus.Signal{Path: path, Name: "org.freedesktop.portal.Request.Response", Body: []any{code, results}})
		return []any{path}, nil
	})
}

func (s *scripted) options(t *testing.T) map[string]godbus.Variant {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.args) != 3 {
		t.Fatalf("args = %#v", s.args)
	}
	return s.args[2].(map[string]godbus.Variant)
}

func uris(list ...string) map[string]godbus.Variant {
	return map[string]godbus.Variant{"uris": godbus.MakeVariant(list)}
}

func TestOpenFilesReturnsTheChosenPaths(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	var s scripted
	s.answer(t, b, "OpenFile", 0, map[string]godbus.Variant{
		"uris": godbus.MakeVariant([]string{"file:///home/me/a%20b.png", "file:///tmp/c.jpg"}),
		"current_filter": godbus.MakeVariant(struct {
			Name  string
			Rules []struct {
				K uint32
				P string
			}
		}{Name: "Images"}),
	})
	desktop.UseBus(bc, b)

	var res desktop.FilesResult
	err := inGoroutine(t, ts, func() (err error) {
		res, err = desktop.OpenFiles(context.Background(), bc, desktop.OpenFilesOpts{
			Title: "Open image", AcceptLabel: "Pick", Multiple: true, CurrentFolder: "/home/me", NotModal: true,
			Filters: []desktop.FileFilter{
				{Name: "Images", Patterns: []string{"*.png", "*.jpg"}, MIMETypes: []string{"image/webp"}},
				{Name: "All", Patterns: []string{"*"}},
			},
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Cancelled || !reflect.DeepEqual(res.Paths, []string{"/home/me/a b.png", "/tmp/c.jpg"}) || res.Filter != "Images" {
		t.Errorf("result = %+v", res)
	}

	// What the portal was asked.
	calls := b.Calls()
	if len(calls) != 1 || calls[0].Args[0] != "test:0" || calls[0].Args[1] != "Open image" {
		t.Fatalf("calls = %+v", calls)
	}
	opt := s.options(t)
	if opt["accept_label"].Value() != "Pick" || opt["multiple"].Value() != true || opt["modal"].Value() != false {
		t.Errorf("options = %#v", opt)
	}
	if got := opt["current_folder"].Value(); !reflect.DeepEqual(got, []byte("/home/me\x00")) {
		t.Errorf("current_folder = %#v", got)
	}
	if _, ok := opt["directory"]; ok {
		t.Error("directory was set")
	}
	if _, ok := opt["activation_token"]; ok {
		t.Error("the file chooser got an activation token")
	}
	filters := opt["filters"]
	if filters.Signature().String() != "a(sa(us))" {
		t.Errorf("filters signature = %s", filters.Signature())
	}
	if opt["current_filter"].Signature().String() != "(sa(us))" {
		t.Errorf("current_filter signature = %s", opt["current_filter"].Signature())
	}
	// The filters have their names and rules: the globs first, then the MIME types.
	rv := reflect.ValueOf(filters.Value())
	if rv.Len() != 2 || rv.Index(0).Field(0).String() != "Images" {
		t.Fatalf("filters = %#v", filters.Value())
	}
	rules := rv.Index(0).Field(1)
	want := [][2]any{{uint32(0), "*.png"}, {uint32(0), "*.jpg"}, {uint32(1), "image/webp"}}
	if rules.Len() != len(want) {
		t.Fatalf("%d rules, want %d", rules.Len(), len(want))
	}
	for i, w := range want {
		r := rules.Index(i)
		if r.Field(0).Uint() != uint64(w[0].(uint32)) || r.Field(1).String() != w[1] {
			t.Errorf("rule %d = %v %v, want %v", i, r.Field(0), r.Field(1), w)
		}
	}
}

func TestOpenFilesDirectoryModeHasNoFilters(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	var s scripted
	s.answer(t, b, "OpenFile", 0, uris("file:///home/me/Documents"))
	desktop.UseBus(bc, b)
	var res desktop.FilesResult
	err := inGoroutine(t, ts, func() (err error) {
		res, err = desktop.OpenFiles(context.Background(), bc, desktop.OpenFilesOpts{
			Directory: true, Filters: []desktop.FileFilter{{Name: "x", Patterns: []string{"*"}}}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	opt := s.options(t)
	if opt["directory"].Value() != true {
		t.Error("directory was not set")
	}
	if _, ok := opt["filters"]; ok {
		t.Error("a folder dialog got filters")
	}
	if _, ok := opt["multiple"]; ok {
		t.Error("multiple was set by default")
	}
	if !reflect.DeepEqual(res.Paths, []string{"/home/me/Documents"}) {
		t.Errorf("paths = %v", res.Paths)
	}
}

func TestSaveFile(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	var s scripted
	s.answer(t, b, "SaveFile", 0, uris("file:///home/me/report.pdf"))
	desktop.UseBus(bc, b)
	var res desktop.FilesResult
	err := inGoroutine(t, ts, func() (err error) {
		res, err = desktop.SaveFile(context.Background(), bc, desktop.SaveFileOpts{
			Title: "Save report", CurrentName: "report.pdf", CurrentFolder: "/home/me", CurrentFile: "/home/me/old.pdf",
			Filters: []desktop.FileFilter{{Name: "PDF", MIMETypes: []string{"application/pdf"}}}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Paths) != 1 || res.Paths[0] != "/home/me/report.pdf" {
		t.Errorf("paths = %v", res.Paths)
	}
	opt := s.options(t)
	if opt["current_name"].Value() != "report.pdf" || !reflect.DeepEqual(opt["current_file"].Value(), []byte("/home/me/old.pdf\x00")) {
		t.Errorf("options = %#v", opt)
	}
	if calls := b.Calls(); calls[0].Method != chooser+"SaveFile" || calls[0].Args[1] != "Save report" {
		t.Errorf("call = %+v", calls[0])
	}
}

func TestSaveFiles(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	var s scripted
	s.answer(t, b, "SaveFiles", 0, uris("file:///home/me/out/a.txt", "file:///home/me/out/b.txt"))
	desktop.UseBus(bc, b)
	var res desktop.FilesResult
	err := inGoroutine(t, ts, func() (err error) {
		res, err = desktop.SaveFiles(context.Background(), bc, []string{"a.txt", "b.txt"}, desktop.SaveFilesOpts{CurrentFolder: "/home/me/out"})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Paths, []string{"/home/me/out/a.txt", "/home/me/out/b.txt"}) {
		t.Errorf("paths = %v", res.Paths)
	}
	opt := s.options(t)
	if opt["files"].Signature().String() != "aay" || !reflect.DeepEqual(opt["files"].Value(), [][]byte{[]byte("a.txt\x00"), []byte("b.txt\x00")}) {
		t.Errorf("files = %#v (%s)", opt["files"].Value(), opt["files"].Signature())
	}
}

func TestSaveFilesNamesAreChecked(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	desktop.UseBus(bc, b)
	for _, names := range [][]string{nil, {""}, {"a/b.txt"}, {"ok.txt", "../x"}} {
		err := inGoroutine(t, ts, func() error { _, err := desktop.SaveFiles(context.Background(), bc, names); return err })
		if len(names) == 0 {
			assertTag(t, err, "SaveFilesWithoutNames")
		} else {
			assertTag(t, err, "SaveFilesNameInvalid")
		}
	}
	if len(b.Calls()) != 0 {
		t.Error("an invalid name reached the portal")
	}
}

func TestCancellingAFileDialogIsAResult(t *testing.T) {
	for _, method := range []string{"OpenFile", "SaveFile", "SaveFiles"} {
		t.Run(method, func(t *testing.T) {
			ts, bc := app(t)
			b := fakebus.New(unique)
			var s scripted
			s.answer(t, b, method, 1, map[string]godbus.Variant{})
			desktop.UseBus(bc, b)
			var res desktop.FilesResult
			err := inGoroutine(t, ts, func() (err error) {
				switch method {
				case "OpenFile":
					res, err = desktop.OpenFiles(context.Background(), bc)
				case "SaveFile":
					res, err = desktop.SaveFile(context.Background(), bc)
				default:
					res, err = desktop.SaveFiles(context.Background(), bc, []string{"a"})
				}
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			if !res.Cancelled || len(res.Paths) != 0 {
				t.Errorf("result = %+v", res)
			}
		})
	}
}

func TestFileChooserUnavailable(t *testing.T) {
	tests := []struct {
		name  string
		setup func(bc widgets.BuildContext)
	}{
		{"no portal on the bus", func(bc widgets.BuildContext) { desktop.UseBus(bc, fakebus.New(unique)) }},
		{"no session bus", func(bc widgets.BuildContext) {
			desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) { return nil, errors.New(tagDBusConnectFailed, "no bus") })
		}},
		{"the portal has no such method", func(bc widgets.BuildContext) {
			b := fakebus.New(unique)
			b.Handle(dbus.PortalDest, dbus.PortalPath, chooser+"OpenFile", func(fakebus.Call) ([]any, error) {
				return nil, &godbus.Error{Name: "org.freedesktop.DBus.Error.UnknownMethod"}
			})
			desktop.UseBus(bc, b)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := app(t)
			tt.setup(bc)
			err := inGoroutine(t, ts, func() error { _, err := desktop.OpenFiles(context.Background(), bc); return err })
			appErr := tagged(t, err, "FileChooserUnavailable")
			if appErr.Metadata["method"] != chooser+"OpenFile" {
				t.Errorf("metadata = %v", appErr.Metadata)
			}
		})
	}
}

func TestFileChooserFailures(t *testing.T) {
	tests := []struct {
		name    string
		code    uint32
		results map[string]godbus.Variant
		tag     string
	}{
		{"the interaction ended", 2, map[string]godbus.Variant{}, "FileChooserFailed"},
		{"no uris", 0, map[string]godbus.Variant{}, "FileChooserReplyInvalid"},
		{"uris of another type", 0, map[string]godbus.Variant{"uris": godbus.MakeVariant("file:///a")}, "FileChooserReplyInvalid"},
		{"a URI that is not a file", 0, uris("https://example.org/a"), "FileChooserReplyInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := app(t)
			b := fakebus.New(unique)
			var s scripted
			s.answer(t, b, "OpenFile", tt.code, tt.results)
			desktop.UseBus(bc, b)
			err := inGoroutine(t, ts, func() error { _, err := desktop.OpenFiles(context.Background(), bc); return err })
			assertTag(t, err, tt.tag)
		})
	}
}

func TestFileDialogIsClosedWhenTheContextEnds(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	b.Handle(dbus.PortalDest, dbus.PortalPath, chooser+"OpenFile", func(c fakebus.Call) ([]any, error) {
		return []any{dbus.PortalRequestPath(unique, tokenIn(t, c.Args[2].(map[string]godbus.Variant)))}, nil
	})
	var closed sync.WaitGroup
	closed.Add(1)
	b.HandleAnyPath(dbus.PortalDest, "org.freedesktop.portal.Request.Close", func(fakebus.Call) ([]any, error) {
		closed.Done()
		return nil, nil
	})
	desktop.UseBus(bc, b)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := inGoroutine(t, ts, func() error { _, err := desktop.OpenFiles(ctx, bc); return err })
	assertTag(t, err, "PortalRequestAborted")
	closed.Wait()
}

func TestFileDialogOnTheUIThreadFails(t *testing.T) {
	_, bc := app(t)
	_, err := desktop.OpenFiles(context.Background(), bc)
	assertTag(t, err, "DesktopCallOnUIThread")
	_, err = desktop.SaveFile(context.Background(), bc)
	assertTag(t, err, "DesktopCallOnUIThread")
}

func TestTwoOptionsValuesPanic(t *testing.T) {
	_, bc := app(t)
	defer func() {
		if recover() == nil {
			t.Error("two options values did not panic")
		}
	}()
	_, _ = desktop.OpenFiles(context.Background(), bc, desktop.OpenFilesOpts{}, desktop.OpenFilesOpts{})
}

func TestFilterNameOfTheAnswer(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"a structure as godbus decodes it", []any{"Images", []any{[]any{uint32(0), "*.png"}}}, "Images"},
		{"a structure as it was sent", struct {
			Name  string
			Rules []string
		}{"Docs", nil}, "Docs"},
		{"an empty slice", []any{}, ""},
		{"a name that is not a string", []any{3}, ""},
		{"anything else", 7, ""},
		{"nothing", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := desktop.FilterName(tt.in); got != tt.want {
				t.Errorf("FilterName(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
