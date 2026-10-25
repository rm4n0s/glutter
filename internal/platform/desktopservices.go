package platform

import (
	"context"

	"github.com/rm4n0s/glutter/ui"
)

// This file holds the optional interfaces of the desktop services that a platform without D-Bus has
// itself (SPEC §25.9): the value that DesktopProvider returns (Windows implements mobile.Services,
// the interface of Android's Desktop, for opening a URI and notifications) may also implement TrayHost
// and FileDialogHost, which package desktop finds by type assertion. Their types are plain Go: no
// Windows, X11 or D-Bus type crosses this interface.

// TrayHost is implemented by the desktop services of a platform that has a system tray of its own.
//
// Flutter: no counterpart
type TrayHost interface {
	// NewTray puts an icon in the tray. It must be called on the UI thread, and fails with
	// TrayDuplicate when the app has a tray already and with TrayRegisterFailed when the icon cannot
	// be added.
	NewTray(spec TraySpec) (Tray, error)
}

// Tray is the icon of the app in the system tray. Its methods belong to the UI thread.
//
// Flutter: no counterpart
type Tray interface {
	// Update gives the icon its new model; only what differs from the old one is told to the system.
	Update(spec TraySpec)
	// Close takes the icon out of the tray. Closing twice does nothing.
	Close()
}

// TraySpec is what the app asks the tray icon to be.
//
// Flutter: no counterpart
type TraySpec struct {
	// ID and Title name the icon to the system; the platform may not show them.
	ID, Title string
	// Tooltip is the first line of the tip that shows when the pointer rests on the icon and
	// TooltipBody the text under it.
	Tooltip, TooltipBody string
	// Icon is the picture of the icon; the picture nearest to the size of a small icon is used. Without
	// pictures the icon of the program is used.
	Icon ui.WindowIcon
	// Menu is the menu of the icon.
	Menu []TrayMenuItem
	// OnActivate is called, on the UI thread, for a click on the icon (the primary action), and
	// OnSecondaryActivate for a click with the middle button.
	OnActivate, OnSecondaryActivate func()
}

// TrayMenuItem is an entry of the menu of a tray icon: an item, a separator or, with Children, a
// submenu.
//
// Flutter: no counterpart
type TrayMenuItem struct {
	// Label is the text of the item.
	Label string
	// Separator makes the entry a line between two groups; its other fields are ignored.
	Separator bool
	// Disabled greys the item out; Hidden leaves it out of the menu.
	Disabled, Hidden bool
	// Checkable gives the item a check mark and Radio a radio mark; Checked says whether it is set.
	Checkable, Radio, Checked bool
	// Children are the entries of a submenu.
	Children []TrayMenuItem
	// OnPressed is called, on the UI thread, when the user chooses the item.
	OnPressed func()
}

// FileDialogHost is implemented by the desktop services of a platform that has file dialogs of its own.
//
// Flutter: no counterpart
type FileDialogHost interface {
	// ShowFileDialog shows the dialog and waits for the user. It must not be called on the UI thread.
	// A user who closes the dialog is a result with Cancelled set, not an error. It fails with
	// FileChooserUnavailable when the dialog cannot be made and with FileChooserFailed for any other
	// failure; ctx closes the dialog.
	ShowFileDialog(ctx context.Context, spec FileDialogSpec) (FileDialogResult, error)
}

// FileDialogKind is the kind of a file dialog.
//
// Flutter: no counterpart
type FileDialogKind int

// The kinds of file dialog.
const (
	// FileDialogOpen opens one file, several files, or a folder.
	//
	// Flutter: no counterpart
	FileDialogOpen FileDialogKind = iota
	// FileDialogSave saves one file.
	//
	// Flutter: no counterpart
	FileDialogSave
	// FileDialogSaveMany saves several files, with the names given, into one folder.
	//
	// Flutter: no counterpart
	FileDialogSaveMany
)

// FileFilter is a kind of file that a dialog offers: a name and the patterns ("*.png").
//
// Flutter: no counterpart
type FileFilter struct {
	Name     string
	Patterns []string
}

// FileDialogSpec is a dialog to show.
//
// Flutter: no counterpart
type FileDialogSpec struct {
	Kind FileDialogKind
	// Owner is the handle of the window the dialog belongs to, as PortalHandleProvider gives it; empty
	// for none.
	Owner string
	// Title and AcceptLabel are the title of the dialog and the text of its accept button.
	Title, AcceptLabel string
	// Filters are the kinds of file the dialog offers; the first is chosen.
	Filters []FileFilter
	// Multiple and Directory are the options of FileDialogOpen.
	Multiple, Directory bool
	// Folder is the folder the dialog opens in and FileName the name the file starts with.
	Folder, FileName string
	// Names are the names of the files of FileDialogSaveMany.
	Names []string
}

// FileDialogResult is the answer of a file dialog.
//
// Flutter: no counterpart
type FileDialogResult struct {
	// Cancelled says that the user closed the dialog without choosing.
	Cancelled bool
	// Paths are what the user chose, as paths of the file system.
	Paths []string
	// Filter is the name of the filter that was selected, when the dialog says.
	Filter string
}
