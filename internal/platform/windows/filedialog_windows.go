//go:build windows

package windows

import (
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// comDialog is itemDialog over IFileOpenDialog or IFileSaveDialog (SPEC §25.9). Its methods belong to
// the worker thread that made the dialog: the object lives in that thread's apartment.
type comDialog struct {
	obj  win32.Object
	save bool
}

var _ itemDialog = (*comDialog)(nil)

// OpenItemDialog makes the common item dialog for saving or opening with CoCreateInstance. It fails with
// COMCallFailed, such as for a class that is not registered.
//
// Flutter: no counterpart
func (s winShell) OpenItemDialog(save bool) (itemDialog, error) {
	clsid, iid := &win32.CLSIDFileOpenDialog, &win32.IIDFileOpenDialog
	if save {
		clsid, iid = &win32.CLSIDFileSaveDialog, &win32.IIDFileSaveDialog
	}
	obj, err := win32.CoCreateInstance(clsid, iid)
	if err != nil {
		return nil, err
	}
	return &comDialog{obj: obj, save: save}, nil
}

func (c *comDialog) Options() (uint32, error) {
	var flags uint32
	if err := c.obj.CallHR("IFileDialog::GetOptions", win32.IFileDialogGetOptions, uintptr(unsafe.Pointer(&flags))); err != nil {
		return 0, err
	}
	return flags, nil
}

func (c *comDialog) SetOptions(flags uint32) error {
	return c.obj.CallHR("IFileDialog::SetOptions", win32.IFileDialogSetOptions, uintptr(flags))
}

func (c *comDialog) SetFileTypes(types []fileType) error {
	if len(types) == 0 {
		return nil
	}
	specs := make([]win32.ComdlgFilterSpec, len(types))
	for i, t := range types {
		name, err := win32.UTF16PtrFromString(t.Name)
		if err != nil {
			return err
		}
		spec, err := win32.UTF16PtrFromString(t.Spec)
		if err != nil {
			return err
		}
		specs[i] = win32.ComdlgFilterSpec{Name: name, Spec: spec}
	}
	// The specs hold the texts as pointers, so the call keeps them alive with the array.
	return c.obj.CallHR("IFileDialog::SetFileTypes", win32.IFileDialogSetFileTypes, uintptr(len(specs)), uintptr(unsafe.Pointer(&specs[0])))
}

func (c *comDialog) SetFileTypeIndex(index uint32) error {
	return c.obj.CallHR("IFileDialog::SetFileTypeIndex", win32.IFileDialogSetFileTypeIndex, uintptr(index))
}

// setText calls a method that takes one string (SetTitle, SetOkButtonLabel, SetFileName,
// SetDefaultExtension).
func (c *comDialog) setText(method string, index int, text string) error {
	p, err := win32.UTF16PtrFromString(text)
	if err != nil {
		return err
	}
	return c.obj.CallHR(method, index, uintptr(unsafe.Pointer(p)))
}

func (c *comDialog) SetTitle(title string) error {
	return c.setText("IFileDialog::SetTitle", win32.IFileDialogSetTitle, title)
}

func (c *comDialog) SetOkButtonLabel(label string) error {
	return c.setText("IFileDialog::SetOkButtonLabel", win32.IFileDialogSetOkButtonLabel, label)
}

func (c *comDialog) SetFileName(name string) error {
	return c.setText("IFileDialog::SetFileName", win32.IFileDialogSetFileName, name)
}

func (c *comDialog) SetDefaultExtension(ext string) error {
	return c.setText("IFileDialog::SetDefaultExtension", win32.IFileDialogSetDefaultExtension, ext)
}

// SetFolder makes the folder the dialog opens in: the shell item of the path.
func (c *comDialog) SetFolder(path string) error {
	item, err := win32.SHCreateItemFromParsingName(path, &win32.IIDShellItem)
	if err != nil {
		return err
	}
	defer item.Release()
	return c.obj.CallHR("IFileDialog::SetFolder", win32.IFileDialogSetFolder, item.Ptr())
}

// Show shows the dialog and waits. A user who closes it is Win32Cancelled (HRESULT_FROM_WIN32(ERROR_CANCELLED)).
func (c *comDialog) Show(owner uintptr) error {
	return c.obj.CallHR("IModalWindow::Show", win32.IModalWindowShow, owner)
}

// Paths reads what the user chose: the items of GetResults for the open dialog (one or several), GetResult
// for the save dialog, each as SIGDN_FILESYSPATH.
func (c *comDialog) Paths() ([]string, error) {
	if c.save {
		var item uintptr
		if err := c.obj.CallHR("IFileDialog::GetResult", win32.IFileDialogGetResult, uintptr(unsafe.Pointer(&item))); err != nil {
			return nil, err
		}
		path, err := itemPath(win32.ObjectFromPtr(item))
		if err != nil {
			return nil, err
		}
		return []string{path}, nil
	}
	var array uintptr
	if err := c.obj.CallHR("IFileOpenDialog::GetResults", win32.IFileOpenDialogGetResults, uintptr(unsafe.Pointer(&array))); err != nil {
		return nil, err
	}
	items := win32.ObjectFromPtr(array)
	defer items.Release()
	var count uint32
	if err := items.CallHR("IShellItemArray::GetCount", win32.IShellItemArrayGetCount, uintptr(unsafe.Pointer(&count))); err != nil {
		return nil, err
	}
	paths := make([]string, 0, count)
	for i := range count {
		var item uintptr
		if err := items.CallHR("IShellItemArray::GetItemAt", win32.IShellItemArrayGetItemAt, uintptr(i), uintptr(unsafe.Pointer(&item))); err != nil {
			return nil, err
		}
		path, err := itemPath(win32.ObjectFromPtr(item))
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

// itemPath is the file system path of a shell item; it releases the item.
func itemPath(item win32.Object) (string, error) {
	defer item.Release()
	var name uintptr
	if err := item.CallHR("IShellItem::GetDisplayName", win32.IShellItemGetDisplayName, win32.SIGDNFileSysPath, uintptr(unsafe.Pointer(&name))); err != nil {
		return "", err
	}
	defer win32.CoTaskMemFree(name)
	return win32.UTF16PtrToString(win32.View[uint16](name)), nil
}

func (c *comDialog) FileTypeIndex() (uint32, error) {
	var index uint32
	if err := c.obj.CallHR("IFileDialog::GetFileTypeIndex", win32.IFileDialogGetFileTypeIndex, uintptr(unsafe.Pointer(&index))); err != nil {
		return 0, err
	}
	return index, nil
}

func (c *comDialog) Release() { c.obj.Release() }
