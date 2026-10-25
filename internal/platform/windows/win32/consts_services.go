package win32

// The values of the desktop services of step 28.11 (SPEC §25.9): the notify icon of the tray and
// its balloons, popup menus, ShellExecuteEx and the common item dialogs. They carry no OS tag, so
// that the layout test checks them against the headers.

// The commands of Shell_NotifyIconW.
const (
	// NIMAdd is NIM_ADD.
	//
	// Flutter: no counterpart
	NIMAdd = 0
	// NIMModify is NIM_MODIFY.
	//
	// Flutter: no counterpart
	NIMModify = 1
	// NIMDelete is NIM_DELETE.
	//
	// Flutter: no counterpart
	NIMDelete = 2
	// NIMSetVersion is NIM_SETVERSION.
	//
	// Flutter: no counterpart
	NIMSetVersion = 4
)

// The members of NOTIFYICONDATAW that are set (uFlags).
const (
	// NIFMessage is NIF_MESSAGE: uCallbackMessage is valid.
	//
	// Flutter: no counterpart
	NIFMessage = 0x01
	// NIFIcon is NIF_ICON: hIcon is valid.
	//
	// Flutter: no counterpart
	NIFIcon = 0x02
	// NIFTip is NIF_TIP: szTip is valid.
	//
	// Flutter: no counterpart
	NIFTip = 0x04
	// NIFState is NIF_STATE: dwState and dwStateMask are valid.
	//
	// Flutter: no counterpart
	NIFState = 0x08
	// NIFInfo is NIF_INFO: the balloon members are valid.
	//
	// Flutter: no counterpart
	NIFInfo = 0x10
	// NIFShowTip is NIF_SHOWTIP: the standard tooltip is shown (with version 4 the tip is shown only with it).
	//
	// Flutter: no counterpart
	NIFShowTip = 0x80
)

// The states of a notify icon (dwState).
const (
	// NISHidden is NIS_HIDDEN.
	//
	// Flutter: no counterpart
	NISHidden = 0x01
)

// The flags of a balloon (dwInfoFlags).
const (
	// NIIFNone is NIIF_NONE: no icon.
	//
	// Flutter: no counterpart
	NIIFNone = 0x00
	// NIIFInfo is NIIF_INFO: the information icon.
	//
	// Flutter: no counterpart
	NIIFInfo = 0x01
	// NIIFWarning is NIIF_WARNING: the warning icon.
	//
	// Flutter: no counterpart
	NIIFWarning = 0x02
	// NIIFUser is NIIF_USER: the icon of hBalloonIcon.
	//
	// Flutter: no counterpart
	NIIFUser = 0x04
	// NIIFNoSound is NIIF_NOSOUND: the balloon makes no sound.
	//
	// Flutter: no counterpart
	NIIFNoSound = 0x10
	// NIIFLargeIcon is NIIF_LARGE_ICON: the icon is the large one.
	//
	// Flutter: no counterpart
	NIIFLargeIcon = 0x20
)

// The events a notify icon sends to its window (the low word of lParam with NOTIFYICON_VERSION_4).
const (
	// NotifyIconVersion4 is NOTIFYICON_VERSION_4.
	//
	// Flutter: no counterpart
	NotifyIconVersion4 = 4
	// NINSelect is NIN_SELECT (WM_USER + 0): the icon was selected with the left button or the keyboard.
	//
	// Flutter: no counterpart
	NINSelect = 0x0400
	// NINKeySelect is NIN_KEYSELECT (NINF_KEY | NIN_SELECT): the icon was selected with the keyboard.
	//
	// Flutter: no counterpart
	NINKeySelect = 0x0401
	// NINBalloonShow is NIN_BALLOONSHOW.
	//
	// Flutter: no counterpart
	NINBalloonShow = 0x0402
	// NINBalloonHide is NIN_BALLOONHIDE.
	//
	// Flutter: no counterpart
	NINBalloonHide = 0x0403
	// NINBalloonTimeout is NIN_BALLOONTIMEOUT.
	//
	// Flutter: no counterpart
	NINBalloonTimeout = 0x0404
	// NINBalloonUserClick is NIN_BALLOONUSERCLICK.
	//
	// Flutter: no counterpart
	NINBalloonUserClick = 0x0405
	// WMContextMenu is WM_CONTEXTMENU.
	//
	// Flutter: no counterpart
	WMContextMenu = 0x007B
	// WMCommand is WM_COMMAND.
	//
	// Flutter: no counterpart
	WMCommand = 0x0111
	// IDCancel is IDCANCEL.
	//
	// Flutter: no counterpart
	IDCancel = 2
)

// The members of MENUITEMINFOW that are set (fMask), its types (fType) and states (fState).
const (
	// MIIMState is MIIM_STATE.
	//
	// Flutter: no counterpart
	MIIMState = 0x0001
	// MIIMID is MIIM_ID.
	//
	// Flutter: no counterpart
	MIIMID = 0x0002
	// MIIMSubMenu is MIIM_SUBMENU.
	//
	// Flutter: no counterpart
	MIIMSubMenu = 0x0004
	// MIIMString is MIIM_STRING.
	//
	// Flutter: no counterpart
	MIIMString = 0x0040
	// MIIMFType is MIIM_FTYPE.
	//
	// Flutter: no counterpart
	MIIMFType = 0x0100
	// MFTString is MFT_STRING.
	//
	// Flutter: no counterpart
	MFTString = 0x0000
	// MFTRadioCheck is MFT_RADIOCHECK.
	//
	// Flutter: no counterpart
	MFTRadioCheck = 0x0200
	// MFTSeparator is MFT_SEPARATOR.
	//
	// Flutter: no counterpart
	MFTSeparator = 0x0800
	// MFSGrayed is MFS_GRAYED (MFS_DISABLED has the same value).
	//
	// Flutter: no counterpart
	MFSGrayed = 0x0003
	// MFSChecked is MFS_CHECKED.
	//
	// Flutter: no counterpart
	MFSChecked = 0x0008
	// TPMRightButton is TPM_RIGHTBUTTON.
	//
	// Flutter: no counterpart
	TPMRightButton = 0x0002
	// TPMNoNotify is TPM_NONOTIFY.
	//
	// Flutter: no counterpart
	TPMNoNotify = 0x0080
)

// The values of ShellExecuteExW.
const (
	// SEEMaskNoAsync is SEE_MASK_NOASYNC: wait for the DDE conversation to end before returning.
	//
	// Flutter: no counterpart
	SEEMaskNoAsync = 0x00000100
	// SEEMaskFlagNoUI is SEE_MASK_FLAG_NO_UI: no error dialog.
	//
	// Flutter: no counterpart
	SEEMaskFlagNoUI = 0x00000400
	// SEErrNoAssoc is SE_ERR_NOASSOC: no application is associated with the file or the scheme.
	//
	// Flutter: no counterpart
	SEErrNoAssoc = 31
	// SEErrAssocIncomplete is SE_ERR_ASSOCINCOMPLETE: the association is incomplete.
	//
	// Flutter: no counterpart
	SEErrAssocIncomplete = 27
	// ErrorNoAssociation is ERROR_NO_ASSOCIATION.
	//
	// Flutter: no counterpart
	ErrorNoAssociation = 1155
)

// The values of the common item dialogs.
const (
	// FOSOverwritePrompt is FOS_OVERWRITEPROMPT.
	//
	// Flutter: no counterpart
	FOSOverwritePrompt = 0x00000002
	// FOSNoChangeDir is FOS_NOCHANGEDIR.
	//
	// Flutter: no counterpart
	FOSNoChangeDir = 0x00000008
	// FOSPickFolders is FOS_PICKFOLDERS.
	//
	// Flutter: no counterpart
	FOSPickFolders = 0x00000020
	// FOSForceFileSystem is FOS_FORCEFILESYSTEM.
	//
	// Flutter: no counterpart
	FOSForceFileSystem = 0x00000040
	// FOSAllowMultiSelect is FOS_ALLOWMULTISELECT.
	//
	// Flutter: no counterpart
	FOSAllowMultiSelect = 0x00000200
	// FOSPathMustExist is FOS_PATHMUSTEXIST.
	//
	// Flutter: no counterpart
	FOSPathMustExist = 0x00000800
	// FOSFileMustExist is FOS_FILEMUSTEXIST.
	//
	// Flutter: no counterpart
	FOSFileMustExist = 0x00001000
	// SIGDNFileSysPath is SIGDN_FILESYSPATH.
	//
	// Flutter: no counterpart
	SIGDNFileSysPath = 0x80058000
	// COInitApartmentThreaded is COINIT_APARTMENTTHREADED.
	//
	// Flutter: no counterpart
	COInitApartmentThreaded = 0x2
	// CLSCTXInprocServer is CLSCTX_INPROC_SERVER.
	//
	// Flutter: no counterpart
	CLSCTXInprocServer = 0x1
)

// The positions of the methods in the vtables of the common item dialogs and of the shell items.
// IFileOpenDialog and IFileSaveDialog start with all of IFileDialog's methods (which start with
// IModalWindow's). The layout test checks each against the headers.
const (
	// IModalWindowShow is the index of IModalWindow::Show.
	//
	// Flutter: no counterpart
	IModalWindowShow = 3
	// IFileDialogSetFileTypes is the index of IFileDialog::SetFileTypes.
	//
	// Flutter: no counterpart
	IFileDialogSetFileTypes = 4
	// IFileDialogSetFileTypeIndex is the index of IFileDialog::SetFileTypeIndex.
	//
	// Flutter: no counterpart
	IFileDialogSetFileTypeIndex = 5
	// IFileDialogGetFileTypeIndex is the index of IFileDialog::GetFileTypeIndex.
	//
	// Flutter: no counterpart
	IFileDialogGetFileTypeIndex = 6
	// IFileDialogSetOptions is the index of IFileDialog::SetOptions.
	//
	// Flutter: no counterpart
	IFileDialogSetOptions = 9
	// IFileDialogGetOptions is the index of IFileDialog::GetOptions.
	//
	// Flutter: no counterpart
	IFileDialogGetOptions = 10
	// IFileDialogSetFolder is the index of IFileDialog::SetFolder.
	//
	// Flutter: no counterpart
	IFileDialogSetFolder = 12
	// IFileDialogSetFileName is the index of IFileDialog::SetFileName.
	//
	// Flutter: no counterpart
	IFileDialogSetFileName = 15
	// IFileDialogSetTitle is the index of IFileDialog::SetTitle.
	//
	// Flutter: no counterpart
	IFileDialogSetTitle = 17
	// IFileDialogSetOkButtonLabel is the index of IFileDialog::SetOkButtonLabel.
	//
	// Flutter: no counterpart
	IFileDialogSetOkButtonLabel = 18
	// IFileDialogGetResult is the index of IFileDialog::GetResult.
	//
	// Flutter: no counterpart
	IFileDialogGetResult = 20
	// IFileDialogSetDefaultExtension is the index of IFileDialog::SetDefaultExtension.
	//
	// Flutter: no counterpart
	IFileDialogSetDefaultExtension = 22
	// IFileDialogClose is the index of IFileDialog::Close.
	//
	// Flutter: no counterpart
	IFileDialogClose = 23
	// IFileOpenDialogGetResults is the index of IFileOpenDialog::GetResults.
	//
	// Flutter: no counterpart
	IFileOpenDialogGetResults = 27
	// IShellItemGetDisplayName is the index of IShellItem::GetDisplayName.
	//
	// Flutter: no counterpart
	IShellItemGetDisplayName = 5
	// IShellItemArrayGetCount is the index of IShellItemArray::GetCount.
	//
	// Flutter: no counterpart
	IShellItemArrayGetCount = 7
	// IShellItemArrayGetItemAt is the index of IShellItemArray::GetItemAt.
	//
	// Flutter: no counterpart
	IShellItemArrayGetItemAt = 8
)
