package windows

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func TestFileTypes(t *testing.T) {
	tests := []struct {
		name string
		in   []platform.FileFilter
		want []fileType
	}{
		{"patterns are joined by a semicolon", []platform.FileFilter{{Name: "Images", Patterns: []string{"*.png", "*.jpg"}}}, []fileType{{"Images", "*.png;*.jpg"}}},
		{"one pattern", []platform.FileFilter{{Name: "Text", Patterns: []string{"*.txt"}}}, []fileType{{"Text", "*.txt"}}},
		{"no patterns shows every file", []platform.FileFilter{{Name: "All files"}}, []fileType{{"All files", "*.*"}}},
		{"empty patterns are left out", []platform.FileFilter{{Name: "X", Patterns: []string{"", " ", "*.go"}}}, []fileType{{"X", "*.go"}}},
		{"only empty patterns show every file", []platform.FileFilter{{Name: "X", Patterns: []string{""}}}, []fileType{{"X", "*.*"}}},
		{"a semicolon inside a pattern is dropped", []platform.FileFilter{{Name: "X", Patterns: []string{"*.a;b"}}}, []fileType{{"X", "*.ab"}}},
		{"no name: the patterns name it", []platform.FileFilter{{Patterns: []string{"*.md", "*.txt"}}}, []fileType{{"*.md;*.txt", "*.md;*.txt"}}},
		{"many filters keep their order", []platform.FileFilter{
			{Name: "A", Patterns: []string{"*.a"}}, {Name: "B", Patterns: []string{"*.b"}}, {Name: "C"}, {Name: "D", Patterns: []string{"*.d"}},
		}, []fileType{{"A", "*.a"}, {"B", "*.b"}, {"C", "*.*"}, {"D", "*.d"}}},
		{"none", nil, []fileType{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fileTypes(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fileTypes = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultExtension(t *testing.T) {
	tests := []struct {
		name string
		in   []platform.FileFilter
		want string
	}{
		{"the first pattern of the first filter", []platform.FileFilter{{Patterns: []string{"*.png", "*.jpg"}}}, "png"},
		{"capitals stay", []platform.FileFilter{{Patterns: []string{"*.PDF"}}}, "PDF"},
		{"digits", []platform.FileFilter{{Patterns: []string{"*.mp4"}}}, "mp4"},
		{"a wildcard in the extension", []platform.FileFilter{{Patterns: []string{"*.t*"}}}, ""},
		{"every file", []platform.FileFilter{{Patterns: []string{"*.*"}}}, ""},
		{"a name pattern", []platform.FileFilter{{Patterns: []string{"README*"}}}, ""},
		{"no patterns", []platform.FileFilter{{Name: "all"}}, ""},
		{"no filters", nil, ""},
		{"a second filter does not count", []platform.FileFilter{{Name: "all"}, {Patterns: []string{"*.txt"}}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultExtension(tt.in); got != tt.want {
				t.Errorf("defaultExtension = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunFileDialogOptions(t *testing.T) {
	const base = win32.FOSForceFileSystem | win32.FOSPathMustExist | win32.FOSNoChangeDir
	tests := []struct {
		name string
		spec platform.FileDialogSpec
		save bool
		want uint32
	}{
		{"open one file", platform.FileDialogSpec{Kind: platform.FileDialogOpen}, false, base | win32.FOSFileMustExist},
		{"open several files", platform.FileDialogSpec{Kind: platform.FileDialogOpen, Multiple: true}, false, base | win32.FOSFileMustExist | win32.FOSAllowMultiSelect},
		{"open a folder", platform.FileDialogSpec{Kind: platform.FileDialogOpen, Directory: true}, false, base | win32.FOSFileMustExist | win32.FOSPickFolders},
		{"open several folders", platform.FileDialogSpec{Kind: platform.FileDialogOpen, Directory: true, Multiple: true}, false,
			base | win32.FOSFileMustExist | win32.FOSPickFolders | win32.FOSAllowMultiSelect},
		{"save asks before overwriting", platform.FileDialogSpec{Kind: platform.FileDialogSave}, true, base | win32.FOSOverwritePrompt},
		{"save several is the folder picker", platform.FileDialogSpec{Kind: platform.FileDialogSaveMany, Names: []string{"a"}}, false, base | win32.FOSPickFolders},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sh := newFakeShell()
			dlg := newFakeDialog()
			dlg.paths = []string{`C:\x`}
			sh.dialog = dlg
			if _, err := runFileDialog(sh, quietLog(), tt.spec); err != nil {
				t.Fatal(err)
			}
			start := uint32(win32.FOSNoChangeDir | 0x400)
			if dlg.setFlags != start|tt.want {
				t.Errorf("options = %#x, want %#x (the dialog's own %#x kept)", dlg.setFlags, start|tt.want, start)
			}
			if got := sh.dialogSaved[0]; got != tt.save {
				t.Errorf("OpenItemDialog(save) = %v, want %v", got, tt.save)
			}
			if !dlg.released {
				t.Error("the dialog was not released")
			}
		})
	}
}

func TestRunFileDialogSettingsAndResults(t *testing.T) {
	filters := []platform.FileFilter{{Name: "Images", Patterns: []string{"*.png", "*.jpg"}}, {Name: "Text", Patterns: []string{"*.txt"}}}
	t.Run("open with everything", func(t *testing.T) {
		sh := newFakeShell()
		dlg := newFakeDialog()
		dlg.paths = []string{`C:\a.png`, `C:\b.png`}
		dlg.chosen = 2
		sh.dialog = dlg
		spec := platform.FileDialogSpec{Kind: platform.FileDialogOpen, Owner: "win32:1a2b", Title: "Open image", AcceptLabel: "Pick",
			Filters: filters, Multiple: true, Folder: `C:\pics`, FileName: "ignored"}
		res, err := runFileDialog(sh, quietLog(), spec)
		if err != nil {
			t.Fatal(err)
		}
		if res.Cancelled || !reflect.DeepEqual(res.Paths, dlg.paths) || res.Filter != "Text" {
			t.Errorf("result = %+v", res)
		}
		if dlg.owner != 0x1a2b || dlg.title != "Open image" || dlg.label != "Pick" || dlg.folder != `C:\pics` {
			t.Errorf("owner %#x title %q label %q folder %q", dlg.owner, dlg.title, dlg.label, dlg.folder)
		}
		if dlg.fileName != "" || dlg.ext != "" {
			t.Errorf("an open dialog got a file name %q or an extension %q", dlg.fileName, dlg.ext)
		}
		wantTypes := []fileType{{"Images", "*.png;*.jpg"}, {"Text", "*.txt"}}
		if !reflect.DeepEqual(dlg.types, wantTypes) || dlg.index != 1 {
			t.Errorf("types %v, index %d", dlg.types, dlg.index)
		}
	})
	t.Run("a folder takes no filters", func(t *testing.T) {
		sh := newFakeShell()
		dlg := newFakeDialog()
		dlg.paths = []string{`C:\dir`}
		sh.dialog = dlg
		res, err := runFileDialog(sh, quietLog(), platform.FileDialogSpec{Kind: platform.FileDialogOpen, Directory: true, Filters: filters})
		if err != nil || dlg.types != nil || res.Filter != "" {
			t.Errorf("types %v, result %+v, err %v", dlg.types, res, err)
		}
	})
	t.Run("save has a name and the default extension", func(t *testing.T) {
		sh := newFakeShell()
		dlg := newFakeDialog()
		dlg.paths = []string{`C:\out\report.png`}
		dlg.chosen = 1
		sh.dialog = dlg
		res, err := runFileDialog(sh, quietLog(), platform.FileDialogSpec{Kind: platform.FileDialogSave, FileName: "report.png", Filters: filters})
		if err != nil {
			t.Fatal(err)
		}
		if dlg.fileName != "report.png" || dlg.ext != "png" || res.Filter != "Images" || !reflect.DeepEqual(res.Paths, dlg.paths) {
			t.Errorf("file name %q ext %q result %+v", dlg.fileName, dlg.ext, res)
		}
	})
	t.Run("save several joins the folder with each name", func(t *testing.T) {
		sh := newFakeShell()
		dlg := newFakeDialog()
		dlg.paths = []string{`C:\out\`}
		dlg.chosen = 1
		sh.dialog = dlg
		res, err := runFileDialog(sh, quietLog(), platform.FileDialogSpec{Kind: platform.FileDialogSaveMany, Names: []string{"a.txt", "b.txt"}, Filters: filters})
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{`C:\out\a.txt`, `C:\out\b.txt`}; !reflect.DeepEqual(res.Paths, want) || dlg.types != nil || res.Filter != "" {
			t.Errorf("paths %v, types %v, filter %q", res.Paths, dlg.types, res.Filter)
		}
	})
	t.Run("a drive root", func(t *testing.T) {
		if got := joinFolder(`C:\`, "x.txt"); got != `C:\x.txt` {
			t.Errorf("joinFolder = %q", got)
		}
		if got := joinFolder(`\\server\share`, "x.txt"); got != `\\server\share\x.txt` {
			t.Errorf("joinFolder = %q", got)
		}
	})
	t.Run("a setting that Windows refuses does not stop the dialog", func(t *testing.T) {
		sh := newFakeShell()
		dlg := newFakeDialog()
		dlg.paths = []string{`C:\a`}
		dlg.failOn = "SetFolder"
		sh.dialog = dlg
		res, err := runFileDialog(sh, quietLog(), platform.FileDialogSpec{Folder: `C:\nope`, Title: "T"})
		if err != nil || len(res.Paths) != 1 || dlg.title != "T" {
			t.Errorf("result %+v, err %v, title %q", res, err, dlg.title)
		}
	})
	t.Run("a filter index outside the filters gives no name", func(t *testing.T) {
		sh := newFakeShell()
		dlg := newFakeDialog()
		dlg.paths = []string{`C:\a`}
		dlg.chosen = 9
		sh.dialog = dlg
		res, err := runFileDialog(sh, quietLog(), platform.FileDialogSpec{Filters: filters})
		if err != nil || res.Filter != "" {
			t.Errorf("result %+v, err %v", res, err)
		}
	})
	t.Run("a handle that is not ours means no owner", func(t *testing.T) {
		sh := newFakeShell()
		dlg := newFakeDialog()
		dlg.paths = []string{`C:\a`}
		sh.dialog = dlg
		if _, err := runFileDialog(sh, quietLog(), platform.FileDialogSpec{Owner: "x11:1a"}); err != nil || dlg.owner != 0 {
			t.Errorf("owner %#x, err %v", dlg.owner, err)
		}
	})
}

func TestRunFileDialogCancelIsAResultNotAnError(t *testing.T) {
	sh := newFakeShell()
	dlg := newFakeDialog()
	// What win32.HResultError makes of HRESULT_FROM_WIN32(ERROR_CANCELLED).
	dlg.showErr = win32.HResultError("IModalWindow::Show", win32.ErrorCancelled)
	sh.dialog = dlg
	res, err := runFileDialog(sh, quietLog(), platform.FileDialogSpec{})
	if err != nil || !res.Cancelled || len(res.Paths) != 0 {
		t.Errorf("result %+v, err %v", res, err)
	}
	if !dlg.released {
		t.Error("the dialog was not released")
	}
}

func TestRunFileDialogFailures(t *testing.T) {
	tests := []struct {
		name  string
		setup func(sh *fakeShell, dlg *fakeDialog)
		tag   string
		route string
	}{
		{"the dialog cannot be made", func(sh *fakeShell, dlg *fakeDialog) {
			sh.dialogErr = errors.New(win32.TagCOMCallFailed, "class not registered")
		},
			tagFileChooserUnavailable, "runFileDialog." + tagFileChooserUnavailable},
		{"the options cannot be read", func(sh *fakeShell, dlg *fakeDialog) { dlg.failOn = "Options" },
			tagFileChooserFailed, "runFileDialog." + tagFileChooserFailed},
		{"the options cannot be set", func(sh *fakeShell, dlg *fakeDialog) { dlg.failOn = "SetOptions" },
			tagFileChooserFailed, "runFileDialog." + tagFileChooserFailed},
		{"the file types cannot be set", func(sh *fakeShell, dlg *fakeDialog) { dlg.failOn = "SetFileTypes" },
			tagFileChooserFailed, "runFileDialog." + tagFileChooserFailed},
		{"the dialog fails", func(sh *fakeShell, dlg *fakeDialog) { dlg.showErr = errors.New(win32.TagCOMCallFailed, "boom") },
			tagFileChooserFailed, "runFileDialog." + tagFileChooserFailed},
		{"the path cannot be read", func(sh *fakeShell, dlg *fakeDialog) { dlg.pathsErr = errors.New(win32.TagCOMCallFailed, "boom") },
			tagFileChooserFailed, "runFileDialog." + tagFileChooserFailed},
		{"an accepted dialog with no choice", func(sh *fakeShell, dlg *fakeDialog) { dlg.paths = nil },
			tagFileChooserFailed, "runFileDialog." + tagFileChooserFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sh := newFakeShell()
			dlg := newFakeDialog()
			dlg.paths = []string{`C:\a`}
			sh.dialog = dlg
			tt.setup(sh, dlg)
			_, err := runFileDialog(sh, quietLog(), platform.FileDialogSpec{Filters: []platform.FileFilter{{Name: "x", Patterns: []string{"*.x"}}}})
			assertTagRoute(t, err, tt.tag, tt.route)
			appErr, _ := errors.FromError(err)
			if tt.name != "an accepted dialog with no choice" && appErr.Metadata["reason"] == nil {
				t.Errorf("metadata = %v: the reason is missing", appErr.Metadata)
			}
		})
	}
}

func TestShowFileDialogRunsOnAWorker(t *testing.T) {
	d, sh := newTestDesktop()
	dlg := newFakeDialog()
	dlg.paths = []string{`C:\a.txt`}
	sh.dialog = dlg
	res, err := d.ShowFileDialog(t.Context(), platform.FileDialogSpec{Title: "T"})
	if err != nil || !reflect.DeepEqual(res.Paths, dlg.paths) {
		t.Errorf("result %+v, err %v", res, err)
	}
	// A failure comes back with the route of the call.
	sh.dialogErr = errors.New(win32.TagCOMCallFailed, "class not registered")
	_, err = d.ShowFileDialog(t.Context(), platform.FileDialogSpec{})
	assertTagRoute(t, err, tagFileChooserUnavailable, "Desktop.ShowFileDialog.func1->runFileDialog."+tagFileChooserUnavailable)
}

func TestShowFileDialogClosesWithItsContext(t *testing.T) {
	d, sh := newTestDesktop()
	dlg := newFakeDialog()
	dlg.block = true
	sh.dialog = dlg
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		_, err := d.ShowFileDialog(ctx, platform.FileDialogSpec{})
		errc <- err
	}()
	select {
	case <-dlg.started:
	case <-time.After(5 * time.Second):
		t.Fatal("the dialog was not shown")
	}
	cancel()
	select {
	case err := <-errc:
		assertTagRoute(t, err, tagFileChooserFailed, "Desktop.ShowFileDialog."+tagFileChooserFailed)
	case <-time.After(5 * time.Second):
		t.Fatal("the dialog did not close with its context")
	}
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if len(sh.cancelled) == 0 || sh.cancelled[0] != 77 {
		t.Errorf("the Cancel message was posted to threads %v, want the worker's (77)", sh.cancelled)
	}
}
