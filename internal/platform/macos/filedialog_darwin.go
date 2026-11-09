//go:build darwin && !ios

package macos

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// The file dialogs (SPEC §29.9): NSOpenPanel and NSSavePanel as windows of their own, shown with
// beginWithCompletionHandler: so that the UI thread goes on drawing while they are open. The filters are
// planned in filters.go.

// dialogCloseWait is how long a dialog whose context ended is given to close.
const dialogCloseWait = 3 * time.Second

// modalResponseOK is NSModalResponseOK.
const modalResponseOK int64 = 1

// dialogState is the panel of a dialog that is open, for the cancel of a context and for the test that answers it.
// It belongs to the UI thread.
type dialogState struct {
	panel objc.ID
}

// ShowFileDialog shows the dialog and waits for the user (SPEC §29.9). It must not be called on the UI thread. A user
// who closes the dialog is a result with Cancelled set. It fails with FileChooserUnavailable when no panel can be made,
// and with FileChooserFailed when ctx ends while the dialog is open.
//
// Flutter: no counterpart (file_selector_macos)
func (d *desktop) ShowFileDialog(ctx context.Context, spec platform.FileDialogSpec) (platform.FileDialogResult, error) {
	type answer struct {
		res platform.FileDialogResult
		err error
	}
	done := make(chan answer, 1)
	st := &dialogState{}
	d.p.Post(func() {
		d.openPanel(spec, st, func(res platform.FileDialogResult, err error) { done <- answer{res, err} })
	})
	select {
	case a := <-done:
		return a.res, a.err
	case <-ctx.Done():
	}
	// The context ended: the panel is cancelled on the UI thread, and the answer of that is waited for.
	d.p.Post(func() {
		if st.panel != 0 {
			objcx.WithPool(func() { st.panel.Send(sel("cancel:"), objc.ID(0)) })
		}
	})
	select {
	case <-done:
		return platform.FileDialogResult{}, errors.NewErr(tagFileChooserFailed, ctx.Err()).SetMetadata("reason", "the context ended while the dialog was open")
	case <-time.After(dialogCloseWait):
		return platform.FileDialogResult{}, errors.NewErr(tagFileChooserFailed, ctx.Err()).SetMetadata("reason", "the context ended and the dialog did not close")
	}
}

var loadTypes = sync.OnceValue(func() error {
	_, err := dl.OpenFramework("UniformTypeIdentifiers")
	return err
})

// contentTypes makes the NSArray of UTTypes (+1, nil for none) for the extensions: an extension that has no
// type is left out.
func contentTypes(exts []string) objc.ID {
	if len(exts) == 0 || loadTypes() != nil {
		return 0
	}
	var types []objc.ID
	for _, e := range exts {
		s := objcx.NewString(e)
		if t := class("UTType").Send(sel("typeWithFilenameExtension:"), s); t != 0 {
			types = append(types, t)
		}
		objcx.Release(s)
	}
	if len(types) == 0 {
		return 0
	}
	return objcx.NewArray(types...)
}

// makePanel makes the panel of the spec (+1), or 0 when AppKit makes none.
func (d *desktop) makePanel(spec platform.FileDialogSpec, plan filterPlan) objc.ID {
	save := spec.Kind == platform.FileDialogSave
	var panel objc.ID
	objcx.WithPool(func() {
		if save {
			panel = objcx.Retain(class("NSSavePanel").Send(sel("savePanel"))) // the pool ends before the panel does
		} else {
			panel = objcx.Retain(class("NSOpenPanel").Send(sel("openPanel")))
		}
	})
	if panel == 0 {
		return 0
	}
	objcx.WithPool(func() {
		str := func(setter, v string) {
			if v == "" {
				return
			}
			s := objcx.NewString(v)
			panel.Send(sel(setter), s)
			objcx.Release(s)
		}
		str("setMessage:", spec.Title)
		str("setPrompt:", spec.AcceptLabel)
		if spec.Folder != "" {
			url := objcx.NewFileURL(spec.Folder)
			panel.Send(sel("setDirectoryURL:"), url)
			objcx.Release(url)
		}
		panel.Send(sel("setCanCreateDirectories:"), true)
		switch {
		case save:
			str("setNameFieldStringValue:", spec.FileName)
			if types := contentTypes(plan.saveExtensions()); types != 0 {
				panel.Send(sel("setAllowedContentTypes:"), types)
				objcx.Release(types)
			}
		case spec.Kind == platform.FileDialogSaveMany:
			panel.Send(sel("setCanChooseFiles:"), false)
			panel.Send(sel("setCanChooseDirectories:"), true)
		default:
			panel.Send(sel("setAllowsMultipleSelection:"), spec.Multiple)
			panel.Send(sel("setCanChooseDirectories:"), spec.Directory)
			panel.Send(sel("setCanChooseFiles:"), !spec.Directory)
			if types := contentTypes(plan.extensions()); types != 0 {
				panel.Send(sel("setAllowedContentTypes:"), types)
				objcx.Release(types)
			}
		}
	})
	return panel
}

// planOf is the plan of the filters of the dialog: a dialog of folders has none.
func planOf(spec platform.FileDialogSpec) filterPlan {
	if spec.Directory || spec.Kind == platform.FileDialogSaveMany {
		return filterPlan{}
	}
	return planFilters(spec.Filters)
}

// openPanel makes the panel of the spec and shows it; done is called, on the UI thread, with the answer.
func (d *desktop) openPanel(spec platform.FileDialogSpec, st *dialogState, done func(platform.FileDialogResult, error)) {
	plan := planOf(spec)
	panel := d.makePanel(spec, plan)
	if panel == 0 {
		done(platform.FileDialogResult{}, errors.New(tagFileChooserUnavailable, "AppKit made no panel for the dialog"))
		return
	}
	st.panel = panel
	blk := objcx.NewBlock(func(_ objc.Block, resp int64) {
		d.p.guard(func() {
			res := d.panelResult(panel, resp, spec, plan)
			objcx.Release(st.panel)
			st.panel = 0
			done(res, nil)
		})
	})
	kept := blk.Copy()
	blk.Release()
	objcx.WithPool(func() {
		panel.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
		panel.Send(sel("beginWithCompletionHandler:"), kept)
	})
}

// panelResult reads the answer of the panel: the paths that the user chose and the filter of the first.
func (d *desktop) panelResult(panel objc.ID, resp int64, spec platform.FileDialogSpec, plan filterPlan) platform.FileDialogResult {
	if resp != modalResponseOK {
		return platform.FileDialogResult{Cancelled: true}
	}
	var paths []string
	objcx.WithPool(func() {
		if spec.Kind == platform.FileDialogSave {
			paths = append(paths, pathOfURL(send[objc.ID](panel, "URL")))
			return
		}
		urls := send[objc.ID](panel, "URLs")
		for i := range objcx.ArrayCount(urls) {
			paths = append(paths, pathOfURL(objcx.ArrayAt(urls, i)))
		}
	})
	res := platform.FileDialogResult{Paths: paths}
	if spec.Kind == platform.FileDialogSaveMany && len(paths) > 0 {
		folder := paths[0]
		res.Paths = nil
		for _, name := range spec.Names {
			res.Paths = append(res.Paths, filepath.Join(folder, name))
		}
	}
	if len(paths) > 0 && !spec.Directory && spec.Kind != platform.FileDialogSaveMany {
		res.Filter = plan.filterNameFor(paths[0])
	}
	return res
}

// pathOfURL is the path of the file URL.
func pathOfURL(url objc.ID) string {
	if url == 0 {
		return ""
	}
	return objcx.String(send[objc.ID](url, "path"))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5); package desktop matches them by value.
const (
	tagFileChooserUnavailable = "FileChooserUnavailable"
	tagFileChooserFailed      = "FileChooserFailed"
)
