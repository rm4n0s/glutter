package windows

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// This file is the plain-Go part of the file dialogs of Windows (SPEC §25.9): which options a dialog
// gets, its file types, and how its answer becomes a result. The dialog itself, IFileOpenDialog or
// IFileSaveDialog, is behind itemDialog (filedialog_windows.go), so that all of it is tested on any host.

// fileType is one entry of the file types of a dialog (COMDLG_FILTERSPEC): a name and the patterns
// joined by ";".
type fileType struct{ Name, Spec string }

// itemDialog is the common item dialog as the code of this file uses it. The methods fail with the
// tagged errors of package win32; Show fails with Win32Cancelled when the user closes the dialog.
type itemDialog interface {
	// Options reads the options the dialog starts with and SetOptions sets them.
	Options() (uint32, error)
	SetOptions(flags uint32) error
	SetFileTypes(types []fileType) error
	// SetFileTypeIndex selects a file type (counted from 1).
	SetFileTypeIndex(index uint32) error
	SetTitle(title string) error
	SetOkButtonLabel(label string) error
	// SetFolder sets the folder the dialog opens in and SetFileName the name that it starts with.
	SetFolder(path string) error
	SetFileName(name string) error
	SetDefaultExtension(ext string) error
	// Show shows the dialog, modal to the window (0 for none), and waits.
	Show(owner uintptr) error
	// Paths are the file system paths of what the user chose (IShellItem::GetDisplayName with
	// SIGDN_FILESYSPATH).
	Paths() ([]string, error)
	// FileTypeIndex is the file type that was selected, counted from 1.
	FileTypeIndex() (uint32, error)
	// Release frees the dialog.
	Release()
}

// dialogCloseWait is how long ShowFileDialog waits for a dialog to close after its context ended.
const dialogCloseWait = 3 * time.Second

// fileTypes makes the file types of the filters: the patterns are joined by ";", a filter with no
// pattern shows every file, and a filter with no name is named by its patterns.
func fileTypes(filters []platform.FileFilter) []fileType {
	out := make([]fileType, 0, len(filters))
	for _, f := range filters {
		var pats []string
		for _, p := range f.Patterns {
			if p = strings.TrimSpace(strings.ReplaceAll(p, ";", "")); p != "" {
				pats = append(pats, p)
			}
		}
		spec := "*.*"
		if len(pats) > 0 {
			spec = strings.Join(pats, ";")
		}
		name := f.Name
		if name == "" {
			name = spec
		}
		out = append(out, fileType{Name: name, Spec: spec})
	}
	return out
}

// defaultExtension is the extension a saved file gets when the user types none: the one of the first
// pattern of the first filter when that is of the form *.ext with letters and digits only.
func defaultExtension(filters []platform.FileFilter) string {
	if len(filters) == 0 || len(filters[0].Patterns) == 0 {
		return ""
	}
	ext, ok := strings.CutPrefix(filters[0].Patterns[0], "*.")
	if !ok || ext == "" {
		return ""
	}
	for _, c := range ext {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return ""
		}
	}
	return ext
}

// dialogOptions are the options of the dialog for the spec (SPEC §25.9), added to those it starts with.
func dialogOptions(spec platform.FileDialogSpec) uint32 {
	flags := uint32(win32.FOSForceFileSystem | win32.FOSPathMustExist | win32.FOSNoChangeDir)
	switch spec.Kind {
	case platform.FileDialogSave:
		flags |= win32.FOSOverwritePrompt
	case platform.FileDialogSaveMany:
		flags |= win32.FOSPickFolders
	default:
		flags |= win32.FOSFileMustExist
		if spec.Directory {
			flags |= win32.FOSPickFolders
		}
		if spec.Multiple {
			flags |= win32.FOSAllowMultiSelect
		}
	}
	return flags
}

// joinFolder joins a folder and a name the way Windows does.
func joinFolder(folder, name string) string {
	return strings.TrimRight(folder, `\/`) + `\` + name
}

// ShowFileDialog shows the common item dialog on a worker thread and waits for the user (SPEC §25.9).
// A user who closes the dialog is a result with Cancelled set. A dialog that cannot be made is
// FileChooserUnavailable and any other failure FileChooserFailed. ctx closes the dialog.
//
// Flutter: no counterpart (file_selector_windows)
func (d *Desktop) ShowFileDialog(ctx context.Context, spec platform.FileDialogSpec) (platform.FileDialogResult, error) {
	type answer struct {
		res platform.FileDialogResult
		err error
	}
	var thread atomic.Uint32
	done := make(chan answer, 1)
	d.shell.Worker(func() {
		thread.Store(d.shell.ThreadID())
		res, err := runFileDialog(d.shell, d.log, spec)
		done <- answer{res, err}
	})
	select {
	case a := <-done:
		return a.res, a.err
	case <-ctx.Done():
	}
	// The context ended: the dialog is closed with the Cancel button's message, again until the dialog
	// has appeared and answered (the thread may not have made its window yet).
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	deadline := time.After(dialogCloseWait)
	for {
		if id := thread.Load(); id != 0 {
			d.shell.CancelDialog(id)
		}
		select {
		case <-done:
			return platform.FileDialogResult{}, errors.NewErr(tagFileChooserFailed, ctx.Err()).SetMetadata("reason", "the context ended while the dialog was open")
		case <-deadline:
			return platform.FileDialogResult{}, errors.NewErr(tagFileChooserFailed, ctx.Err()).SetMetadata("reason", "the context ended and the dialog did not close")
		case <-tick.C:
		}
	}
}

// runFileDialog makes the dialog with the options of the spec, shows it and reads the answer. It runs
// on the worker thread. The settings of the dialog that Windows may refuse (a folder that does not exist,
// a label) are logged and skipped: the dialog still works.
func runFileDialog(shell shellAPI, log *slog.Logger, spec platform.FileDialogSpec) (platform.FileDialogResult, error) {
	save := spec.Kind == platform.FileDialogSave
	dlg, err := shell.OpenItemDialog(save)
	if err != nil {
		return platform.FileDialogResult{}, errors.New(tagFileChooserUnavailable, "the item dialog of Windows could not be made", "reason", err.Error())
	}
	defer dlg.Release()
	soft := func(what string, err error) {
		if err != nil {
			log.Debug("a setting of the file dialog was not taken", "setting", what, "err", err)
		}
	}
	start, err := dlg.Options()
	if err != nil {
		return platform.FileDialogResult{}, errors.New(tagFileChooserFailed, "the options of the dialog could not be read", "reason", err.Error())
	}
	if err := dlg.SetOptions(start | dialogOptions(spec)); err != nil {
		return platform.FileDialogResult{}, errors.New(tagFileChooserFailed, "the options of the dialog could not be set", "reason", err.Error())
	}
	filters := spec.Filters
	if spec.Directory || spec.Kind == platform.FileDialogSaveMany {
		filters = nil
	}
	if len(filters) > 0 {
		if err := dlg.SetFileTypes(fileTypes(filters)); err != nil {
			return platform.FileDialogResult{}, errors.New(tagFileChooserFailed, "the file types of the dialog could not be set", "reason", err.Error())
		}
		soft("file type index", dlg.SetFileTypeIndex(1))
		if save {
			if ext := defaultExtension(filters); ext != "" {
				soft("default extension", dlg.SetDefaultExtension(ext))
			}
		}
	}
	if spec.Title != "" {
		soft("title", dlg.SetTitle(spec.Title))
	}
	if spec.AcceptLabel != "" {
		soft("accept label", dlg.SetOkButtonLabel(spec.AcceptLabel))
	}
	if spec.Folder != "" {
		soft("folder", dlg.SetFolder(spec.Folder))
	}
	if spec.FileName != "" && save {
		soft("file name", dlg.SetFileName(spec.FileName))
	}
	var owner uintptr
	if h, ok := hwndOfHandle(spec.Owner); ok {
		owner = h
	}
	if err := dlg.Show(owner); err != nil {
		if appErr, ok := errors.FromError(err); ok && appErr.Tag == win32.TagWin32Cancelled {
			return platform.FileDialogResult{Cancelled: true}, nil
		}
		return platform.FileDialogResult{}, errors.New(tagFileChooserFailed, "the dialog could not be shown", "reason", err.Error())
	}
	paths, err := dlg.Paths()
	if err != nil {
		return platform.FileDialogResult{}, errors.New(tagFileChooserFailed, "the path of the choice could not be read", "reason", err.Error())
	}
	if len(paths) == 0 {
		return platform.FileDialogResult{}, errors.New(tagFileChooserFailed, "the dialog was accepted without a choice")
	}
	res := platform.FileDialogResult{Paths: paths}
	if spec.Kind == platform.FileDialogSaveMany {
		res.Paths = nil
		for _, name := range spec.Names {
			res.Paths = append(res.Paths, joinFolder(paths[0], name))
		}
	}
	if len(filters) > 0 {
		if i, err := dlg.FileTypeIndex(); err == nil && i >= 1 && int(i) <= len(filters) {
			res.Filter = filters[i-1].Name
		}
	}
	return res, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5). They are the tags of package desktop.
const (
	tagFileChooserFailed      = "FileChooserFailed"
	tagFileChooserUnavailable = "FileChooserUnavailable"
)
