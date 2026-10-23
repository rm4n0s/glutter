package desktop

import (
	"context"
	"net/url"
	"path/filepath"
	"reflect"
	"slices"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	openFileMethod  = "org.freedesktop.portal.FileChooser.OpenFile"
	saveFileMethod  = "org.freedesktop.portal.FileChooser.SaveFile"
	saveFilesMethod = "org.freedesktop.portal.FileChooser.SaveFiles"
)

// FileFilter is a kind of file that a file dialog can show: the dialog offers the
// filters in a menu, and shows only the files that match the one that is chosen.
//
// Flutter: no counterpart (file_selector has XTypeGroup)
type FileFilter struct {
	// Name is what the menu shows, such as "Images".
	Name string
	// Patterns are globs such as "*.png". The portal matches them exactly as written,
	// so write both cases ("*.png", "*.PNG") for a name that may have either.
	Patterns []string
	// MIMETypes are MIME types such as "image/png".
	MIMETypes []string
}

// FilesResult is the answer of a file dialog.
//
// Flutter: no counterpart
type FilesResult struct {
	// Cancelled is true when the user closed the dialog without choosing. It is a
	// result, not an error; Paths is empty then.
	Cancelled bool
	// Paths are the files or folders that the user chose, as paths of the file
	// system. Inside a Flatpak sandbox they are the paths the document portal made
	// for them.
	Paths []string
	// Filter is the name of the filter that the user left selected, when the dialog
	// said so; empty otherwise.
	Filter string
}

// OpenFilesOpts are the optional parameters of OpenFiles.
//
// Flutter: no counterpart
type OpenFilesOpts struct {
	// Title is the title of the dialog; the default is the portal's ("Open File").
	Title string
	// AcceptLabel is the text of the button that accepts the dialog.
	AcceptLabel string
	// Filters are the kinds of file the dialog offers; the first one is chosen when
	// the dialog opens.
	Filters []FileFilter
	// Multiple lets the user choose more than one file. Without it the result has one
	// path.
	Multiple bool
	// Directory makes the dialog choose folders instead of files; Filters do not apply.
	Directory bool
	// CurrentFolder is the folder the dialog opens in.
	CurrentFolder string
	// NotModal leaves the window of the app usable while the dialog is open; by
	// default the dialog blocks it.
	NotModal bool
}

// SaveFileOpts are the optional parameters of SaveFile.
//
// Flutter: no counterpart
type SaveFileOpts struct {
	// Title is the title of the dialog; the default is the portal's ("Save File").
	Title string
	// AcceptLabel is the text of the button that accepts the dialog.
	AcceptLabel string
	// Filters are the kinds of file the dialog offers.
	Filters []FileFilter
	// CurrentName is the name that the file starts with, such as "report.pdf".
	CurrentName string
	// CurrentFolder is the folder the dialog opens in.
	CurrentFolder string
	// CurrentFile is the path of the file that the dialog starts with selected, when the
	// user is saving over a file that exists; it sets the folder and the name.
	CurrentFile string
	// NotModal leaves the window of the app usable while the dialog is open.
	NotModal bool
}

// SaveFilesOpts are the optional parameters of SaveFiles.
//
// Flutter: no counterpart
type SaveFilesOpts struct {
	// Title is the title of the dialog; the default is the portal's ("Save Files").
	Title string
	// AcceptLabel is the text of the button that accepts the dialog.
	AcceptLabel string
	// CurrentFolder is the folder the dialog opens in.
	CurrentFolder string
	// NotModal leaves the window of the app usable while the dialog is open.
	NotModal bool
}

// OpenFiles shows the dialog of the desktop for opening files or folders
// (org.freedesktop.portal.FileChooser) as a child of the window of bc, and waits
// for the user. Closing the dialog without a choice is the result with Cancelled
// set, not an error. There is no fallback: without the portal it fails with
// FileChooserUnavailable.
//
// It waits for the user, so it must be called from a goroutine and not on the UI
// thread (it fails with DesktopCallOnUIThread there); ctx closes the dialog.
//
// Flutter: no counterpart (Flutter apps use the file_selector plugin)
func OpenFiles(ctx context.Context, bc widgets.BuildContext, opts ...OpenFilesOpts) (FilesResult, error) {
	o := firstOpts(opts)
	c, err := begin(ctx, bc)
	if err != nil {
		return FilesResult{}, err
	}
	return c.chooseFiles(ctx, openFileMethod, o.Title, func(opt map[string]godbus.Variant) {
		setCommon(opt, o.AcceptLabel, o.NotModal, o.CurrentFolder)
		if o.Multiple {
			opt["multiple"] = godbus.MakeVariant(true)
		}
		if o.Directory {
			opt["directory"] = godbus.MakeVariant(true)
		} else {
			setFilters(opt, o.Filters)
		}
	})
}

// SaveFile shows the dialog of the desktop for saving one file and waits for the
// user. The path in the result is where to write; the file is not made. See
// OpenFiles for the rules of the call, the cancel and the failures.
//
// Flutter: no counterpart (Flutter apps use the file_selector plugin)
func SaveFile(ctx context.Context, bc widgets.BuildContext, opts ...SaveFileOpts) (FilesResult, error) {
	o := firstOpts(opts)
	c, err := begin(ctx, bc)
	if err != nil {
		return FilesResult{}, err
	}
	return c.chooseFiles(ctx, saveFileMethod, o.Title, func(opt map[string]godbus.Variant) {
		setCommon(opt, o.AcceptLabel, o.NotModal, o.CurrentFolder)
		setFilters(opt, o.Filters)
		if o.CurrentName != "" {
			opt["current_name"] = godbus.MakeVariant(o.CurrentName)
		}
		if o.CurrentFile != "" {
			opt["current_file"] = godbus.MakeVariant(byteString(o.CurrentFile))
		}
	})
}

// SaveFiles shows the dialog of the desktop for saving several files into one
// folder, with the names given, and waits for the user. The paths in the result are
// where to write them, in the order of names; none is made. A name is a file name
// without a folder. See OpenFiles for the rules of the call, the cancel and the
// failures.
//
// Flutter: no counterpart (Flutter apps use the file_selector plugin)
func SaveFiles(ctx context.Context, bc widgets.BuildContext, names []string, opts ...SaveFilesOpts) (FilesResult, error) {
	o := firstOpts(opts)
	if len(names) == 0 {
		return FilesResult{}, errors.New(tagSaveFilesWithoutNames, "SaveFiles needs the name of at least one file")
	}
	for _, n := range names {
		if n == "" || filepath.Base(n) != n {
			return FilesResult{}, errors.New(tagSaveFilesNameInvalid, "a name given to SaveFiles must be a file name without a folder", "name", n)
		}
	}
	c, err := begin(ctx, bc)
	if err != nil {
		return FilesResult{}, err
	}
	return c.chooseFiles(ctx, saveFilesMethod, o.Title, func(opt map[string]godbus.Variant) {
		setCommon(opt, o.AcceptLabel, o.NotModal, o.CurrentFolder)
		files := make([][]byte, len(names))
		for i, n := range names {
			files[i] = byteString(n)
		}
		opt["files"] = godbus.MakeVariant(files)
	})
}

func firstOpts[T any](opts []T) T {
	var zero T
	if len(opts) > 1 {
		panic(errors.New(tagMultipleOptions, "a function of package desktop takes at most one options value", "count", len(opts)))
	}
	if len(opts) == 1 {
		return opts[0]
	}
	return zero
}

// byteString is how the portals take a file name or a path: bytes with a closing NUL
// (D-Bus type ay).
func byteString(s string) []byte { return append([]byte(s), 0) }

func setCommon(opt map[string]godbus.Variant, acceptLabel string, notModal bool, folder string) {
	if acceptLabel != "" {
		opt["accept_label"] = godbus.MakeVariant(acceptLabel)
	}
	if notModal {
		opt["modal"] = godbus.MakeVariant(false)
	}
	if folder != "" {
		opt["current_folder"] = godbus.MakeVariant(byteString(folder))
	}
}

// portalRule is one pattern of a filter: kind 0 is a glob, 1 a MIME type.
type portalRule struct {
	Kind    uint32
	Pattern string
}

// portalFilter is a(sa(us)): the name of the filter and its rules.
type portalFilter struct {
	Name  string
	Rules []portalRule
}

func setFilters(opt map[string]godbus.Variant, filters []FileFilter) {
	if len(filters) == 0 {
		return
	}
	list := make([]portalFilter, 0, len(filters))
	for _, f := range filters {
		pf := portalFilter{Name: f.Name, Rules: []portalRule{}}
		for _, p := range f.Patterns {
			pf.Rules = append(pf.Rules, portalRule{Kind: 0, Pattern: p})
		}
		for _, m := range f.MIMETypes {
			pf.Rules = append(pf.Rules, portalRule{Kind: 1, Pattern: m})
		}
		list = append(list, pf)
	}
	opt["filters"] = godbus.MakeVariant(list)
	opt["current_filter"] = godbus.MakeVariant(list[0])
}

// chooseFiles calls a method of the FileChooser portal and turns its answer into a
// result.
func (c *call) chooseFiles(ctx context.Context, method, title string, fill func(map[string]godbus.Variant)) (FilesResult, error) {
	bus, err := c.s.connect(ctx, "FileChooserUnavailable")
	if err != nil {
		// connect tags the failure with the tag given; the portal method is added to it here.
		if appErr, ok := errors.FromError(err); ok {
			return FilesResult{}, appErr.SetMetadata("method", method)
		}
		return FilesResult{}, err
	}
	resp, err := dbus.PortalRequest(ctx, bus, method, func(token string) []any {
		opt := c.portalOptions(token, nil)
		delete(opt, "activation_token") // the file chooser has none
		fill(opt)
		return []any{c.handle, title, opt}
	})
	if err != nil {
		if portalMissing(err) {
			return FilesResult{}, errors.NewErr(tagFileChooserUnavailable, err).SetMetadata("method", method)
		}
		return FilesResult{}, err
	}
	switch resp.Code {
	case dbus.PortalResponseCancelled:
		return FilesResult{Cancelled: true}, nil
	case dbus.PortalResponseSuccess:
		return parseFilesResult(resp.Results, method)
	}
	return FilesResult{}, errors.New(tagFileChooserFailed, "the file chooser portal ended the interaction without a result",
		"method", method, "code", uint32(resp.Code))
}

// parseFilesResult reads the uris and the filter of a successful answer.
func parseFilesResult(results map[string]godbus.Variant, method string) (FilesResult, error) {
	uris, ok := results["uris"].Value().([]string)
	if !ok {
		return FilesResult{}, errors.New(tagFileChooserReplyInvalid, "the answer of the file chooser has no list of URIs",
			"method", method, "keys", mapKeys(results))
	}
	out := FilesResult{Paths: make([]string, 0, len(uris))}
	for _, raw := range uris {
		path, err := pathOfURI(raw)
		if err != nil {
			return FilesResult{}, err
		}
		out.Paths = append(out.Paths, path)
	}
	if v, ok := results["current_filter"]; ok {
		out.Filter = filterName(v.Value())
	}
	return out, nil
}

// filterName is the name in a filter of the answer, (sa(us)): the first field of the
// structure, which godbus gives as a slice of values (a bus) or as the struct that
// was sent (a test).
func filterName(v any) string {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice:
		if rv.Len() > 0 {
			name, _ := rv.Index(0).Interface().(string)
			return name
		}
	case reflect.Struct:
		if rv.NumField() > 0 && rv.Field(0).Kind() == reflect.String {
			return rv.Field(0).String()
		}
	}
	return ""
}

// pathOfURI turns a file: URI of the answer into a path.
func pathOfURI(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "file" {
		return "", errors.New(tagFileChooserReplyInvalid, "the file chooser answered with a URI that is not a file: URI", "uri", raw)
	}
	return u.Path, nil
}

func mapKeys(m map[string]godbus.Variant) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// portalMissing says whether err is the bus telling that nobody serves the portal:
// no connection to the bus, no such service, no such method or object, or no reply.
func portalMissing(err error) bool {
	if appErr, ok := errors.FromError(err); ok && appErr.Tag == dbus.TagConnectFailed {
		return true
	}
	// A real bus gives the error as a godbus.Error value; a test double may give a pointer.
	var name string
	var dbusErr godbus.Error
	var dbusErrPtr *godbus.Error
	switch {
	case errors.As(err, &dbusErr):
		name = dbusErr.Name
	case errors.As(err, &dbusErrPtr):
		name = dbusErrPtr.Name
	default:
		return false
	}
	switch name {
	case "org.freedesktop.DBus.Error.ServiceUnknown", "org.freedesktop.DBus.Error.NameHasNoOwner",
		"org.freedesktop.DBus.Error.UnknownMethod", "org.freedesktop.DBus.Error.UnknownObject",
		"org.freedesktop.DBus.Error.UnknownInterface", "org.freedesktop.DBus.Error.NoReply",
		"org.freedesktop.DBus.Error.Disconnected":
		return true
	}
	return false
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFileChooserFailed       = "FileChooserFailed"
	tagFileChooserReplyInvalid = "FileChooserReplyInvalid"
	tagFileChooserUnavailable  = "FileChooserUnavailable"
	tagMultipleOptions         = "MultipleOptions"
	tagSaveFilesNameInvalid    = "SaveFilesNameInvalid"
	tagSaveFilesWithoutNames   = "SaveFilesWithoutNames"
)
