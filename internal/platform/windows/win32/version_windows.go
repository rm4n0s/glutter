//go:build windows

package win32

import "unsafe"

var versionDLL = &dllTable{name: "version.dll"}

// SystemDirectory is the path of the System32 directory (GetSystemDirectoryW), such as
// C:\Windows\system32. It fails with Win32CallFailed.
//
// Flutter: no counterpart
func SystemDirectory() (string, error) {
	var buf [260]uint16
	r, errno, err := kern32.call("GetSystemDirectoryW", uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if err != nil {
		return "", err
	}
	if r == 0 || int(r) >= len(buf) {
		return "", Win32Error("GetSystemDirectoryW", uint32(errno))
	}
	return UTF16ToString(buf[:r]), nil
}

// FileVersionInfo reads the file version of the file at path from its version resource
// (GetFileVersionInfoSizeW, GetFileVersionInfoW and VerQueryValueW). It fails with Win32CallFailed
// when the file has no version resource or cannot be read (the metadata "function" and "code").
//
// Flutter: no counterpart
func FileVersionInfo(path string) (FileVersion, error) {
	p, err := UTF16PtrFromString(path)
	if err != nil {
		return FileVersion{}, err
	}
	size, errno, err := versionDLL.call("GetFileVersionInfoSizeW", uintptr(unsafe.Pointer(p)), 0)
	if err != nil {
		return FileVersion{}, err
	}
	if size == 0 {
		return FileVersion{}, Win32Error("GetFileVersionInfoSizeW", uint32(errno))
	}
	data := make([]byte, size)
	r, errno, err := versionDLL.call("GetFileVersionInfoW", uintptr(unsafe.Pointer(p)), 0, size, uintptr(unsafe.Pointer(&data[0])))
	if err != nil {
		return FileVersion{}, err
	}
	if r == 0 {
		return FileVersion{}, Win32Error("GetFileVersionInfoW", uint32(errno))
	}
	root, err := UTF16PtrFromString(`\`)
	if err != nil {
		return FileVersion{}, err
	}
	var info *VSFixedFileInfo
	var length uint32
	r, errno, err = versionDLL.call("VerQueryValueW", uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(root)),
		uintptr(unsafe.Pointer(&info)), uintptr(unsafe.Pointer(&length)))
	if err != nil {
		return FileVersion{}, err
	}
	if r == 0 || info == nil || length < uint32(unsafe.Sizeof(VSFixedFileInfo{})) {
		return FileVersion{}, Win32Error("VerQueryValueW", uint32(errno))
	}
	// info points into data, which is alive here.
	return FileVersionOf(*info), nil
}
