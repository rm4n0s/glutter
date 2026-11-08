//go:build linux && !android

package x11

import (
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// The names of the libraries (SPEC §8.1).
const (
	libX11     = "libX11.so.6"
	libX11XCB  = "libX11-xcb.so.1"
	libXi      = "libXi.so.6"
	libXrandr  = "libXrandr.so.2"
	libXcursor = "libXcursor.so.1"
	libXext    = "libXext.so.6"
)

// X11 constants from X.h, Xlib.h and Xutil.h.
const (
	inputOutput = 1 // InputOutput
	inputOnly   = 2 // InputOnly

	cwBackPixel   = 1 << 1  // CWBackPixel
	cwBorderPixel = 1 << 3  // CWBorderPixel
	cwEventMask   = 1 << 11 // CWEventMask
	cwColormap    = 1 << 13 // CWColormap
	cwCursor      = 1 << 14 // CWCursor

	structureNotifyMask     = 1 << 17
	exposureMask            = 1 << 15
	focusChangeMask         = 1 << 21
	propertyChangeMask      = 1 << 22
	substructureNotifyMask  = 1 << 19
	substructureRedirMask   = 1 << 20
	visualIDMask            = 0x1 // VisualIDMask
	visualScreenMask        = 0x2 // VisualScreenMask
	visualDepthMask         = 0x4 // VisualDepthMask
	visualClassMask         = 0x8 // VisualClassMask
	trueColor               = 4   // TrueColor
	zPixmap                 = 2   // ZPixmap
	lsbFirst                = 0   // LSBFirst
	shmCompletion           = 0   // ShmCompletion, the one event of MIT-SHM
	allocNone               = 0   // AllocNone
	propModeReplace         = 0   // PropModeReplace
	xaPrimary               = 1   // XA_PRIMARY
	xaAtom                  = 4   // XA_ATOM
	xaCardinal              = 6   // XA_CARDINAL
	xaString                = 31  // XA_STRING
	xaWindow                = 33  // XA_WINDOW
	xaWMName                = 39  // XA_WM_NAME
	pMinSize                = 1 << 4
	pMaxSize                = 1 << 5
	pPosition               = 1 << 2 // PPosition
	pSize                   = 1 << 3 // PSize
	clientMessage           = 33
	configureNotify         = 22
	destroyNotify           = 17
	exposeEvent             = 12
	genericEvent            = 35
	mapNotify               = 19
	unmapNotify             = 18
	propertyNotify          = 28
	selectionClear          = 29
	selectionRequest        = 30
	selectionNotify         = 31
	propertyNewValue        = 0 // PropertyNewValue
	propertyDelete          = 1 // PropertyDelete
	anyPropertyType         = 0 // AnyPropertyType
	currentTime             = 0 // CurrentTime
	xNone                   = 0 // None
	netWMStateRemove        = 0
	netWMStateAdd           = 1
	xiAllMasterDevices      = 1
	xiAllDevices            = 0
	xiQueryVersionMajor     = 2
	xiQueryVersionMinorWant = 2
)

// xlib holds the bound functions of libX11 and libX11-xcb.
type xlib struct {
	openDisplay          func(name uintptr) uintptr
	closeDisplay         func(display uintptr) int32
	connectionNumber     func(display uintptr) int32
	pending              func(display uintptr) int32
	nextEvent            func(display, event uintptr) int32
	createWindow         func(display uintptr, parent uint64, x, y int32, width, height, border uint32, depth int32, class uint32, visual uintptr, mask uint64, attrs uintptr) uint64
	mapWindow            func(display uintptr, window uint64) int32
	destroyWindow        func(display uintptr, window uint64) int32
	internAtom           func(display, name uintptr, onlyIfExists int32) uint64
	changeProperty       func(display uintptr, window, property, typ uint64, format, mode int32, data uintptr, count int32) int32
	deleteProperty       func(display uintptr, window, property uint64) int32
	setWMProtocols       func(display uintptr, window uint64, protocols uintptr, count int32) int32
	selectInput          func(display uintptr, window uint64, mask int64) int32
	flush                func(display uintptr) int32
	sync                 func(display uintptr, discard int32) int32
	setErrorHandler      func(handler uintptr) uintptr
	defaultRootWindow    func(display uintptr) uint64
	defaultScreen        func(display uintptr) int32
	getEventData         func(display, cookie uintptr) int32
	freeEventData        func(display, cookie uintptr)
	queryExtension       func(display, name, opcode, event, errorBase uintptr) int32
	sendEvent            func(display uintptr, window uint64, propagate int32, mask int64, event uintptr) int32
	resizeWindow         func(display uintptr, window uint64, width, height uint32) int32
	setWMNormalHints     func(display uintptr, window uint64, hints uintptr)
	setClassHint         func(display uintptr, window uint64, hint uintptr) int32
	setTransientForHint  func(display uintptr, window, parent uint64) int32
	iconifyWindow        func(display uintptr, window uint64, screen int32) int32
	withdrawWindow       func(display uintptr, window uint64, screen int32) int32
	getVisualInfo        func(display uintptr, mask int64, template uintptr, count uintptr) uintptr
	free                 func(ptr uintptr) int32
	createColormap       func(display uintptr, window uint64, visual uintptr, alloc int32) uint64
	freeColormap         func(display uintptr, colormap uint64) int32
	defineCursor         func(display uintptr, window, cursor uint64) int32
	freeCursor           func(display uintptr, cursor uint64) int32
	getXCBConnection     func(display uintptr) uintptr
	createBitmap         func(display uintptr, drawable uint64, data uintptr, width, height uint32) uint64
	createPixmapCursor   func(display uintptr, source, mask uint64, foreground, background uintptr, x, y uint32) uint64
	freePixmap           func(display uintptr, pixmap uint64) int32
	setSelectionOwner    func(display uintptr, selection, owner, time uint64) int32
	getSelectionOwner    func(display uintptr, selection uint64) uint64
	convertSelection     func(display uintptr, selection, target, property, requestor, time uint64) int32
	getWindowProperty    func(display uintptr, window, property uint64, offset, length int64, del int32, reqType uint64, actualType, actualFormat, nitems, bytesAfter, prop uintptr) int32
	translateCoordinates func(display uintptr, src, dest uint64, srcX, srcY int32, destX, destY, child uintptr) int32
	getAtomName          func(display uintptr, atom uint64) uintptr
	grabPointer          func(display uintptr, window uint64, ownerEvents int32, eventMask uint32, pointerMode, keyboardMode int32, confineTo, cursor, time uint64) int32
	ungrabPointer        func(display uintptr, time uint64) int32
	warpPointer          func(display uintptr, src, dest uint64, srcX, srcY int32, srcWidth, srcHeight uint32, destX, destY int32) int32

	// What the window needs to show the pixels of the software renderer (SPEC §9.6).
	displayString func(display uintptr) uintptr
	defaultVisual func(display uintptr, screen int32) uintptr
	defaultDepth  func(display uintptr, screen int32) int32
	createGC      func(display uintptr, drawable, mask uint64, values uintptr) uintptr
	freeGC        func(display, gc uintptr) int32
	createImage   func(display, visual uintptr, depth uint32, format, offset int32, data uintptr, width, height uint32, bitmapPad, bytesPerLine int32) uintptr
	putImage      func(display uintptr, drawable uint64, gc, image uintptr, srcX, srcY, dstX, dstY int32, width, height uint32) int32
	destroyImage  func(image uintptr) int32
}

// loadXlib opens libX11 and libX11-xcb and binds the functions. It fails with
// LibraryLoadFailed when a library cannot be loaded and with SymbolMissing when a
// function is absent.
func loadXlib() (*xlib, error) {
	x11, err := dl.Open(libX11, "libX11.so")
	if err != nil {
		return nil, err
	}
	xcb, err := dl.Open(libX11XCB, "libX11-xcb.so")
	if err != nil {
		return nil, err
	}
	l := new(xlib)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.openDisplay, "XOpenDisplay"},
		{&l.closeDisplay, "XCloseDisplay"},
		{&l.connectionNumber, "XConnectionNumber"},
		{&l.pending, "XPending"},
		{&l.nextEvent, "XNextEvent"},
		{&l.createWindow, "XCreateWindow"},
		{&l.mapWindow, "XMapWindow"},
		{&l.translateCoordinates, "XTranslateCoordinates"},
		{&l.destroyWindow, "XDestroyWindow"},
		{&l.internAtom, "XInternAtom"},
		{&l.changeProperty, "XChangeProperty"},
		{&l.deleteProperty, "XDeleteProperty"},
		{&l.setWMProtocols, "XSetWMProtocols"},
		{&l.selectInput, "XSelectInput"},
		{&l.flush, "XFlush"},
		{&l.sync, "XSync"},
		{&l.setErrorHandler, "XSetErrorHandler"},
		{&l.defaultRootWindow, "XDefaultRootWindow"},
		{&l.defaultScreen, "XDefaultScreen"},
		{&l.getEventData, "XGetEventData"},
		{&l.freeEventData, "XFreeEventData"},
		{&l.queryExtension, "XQueryExtension"},
		{&l.sendEvent, "XSendEvent"},
		{&l.resizeWindow, "XResizeWindow"},
		{&l.setWMNormalHints, "XSetWMNormalHints"},
		{&l.setClassHint, "XSetClassHint"},
		{&l.setTransientForHint, "XSetTransientForHint"},
		{&l.iconifyWindow, "XIconifyWindow"},
		{&l.withdrawWindow, "XWithdrawWindow"},
		{&l.getVisualInfo, "XGetVisualInfo"},
		{&l.free, "XFree"},
		{&l.createColormap, "XCreateColormap"},
		{&l.freeColormap, "XFreeColormap"},
		{&l.defineCursor, "XDefineCursor"},
		{&l.freeCursor, "XFreeCursor"},
		{&l.createBitmap, "XCreateBitmapFromData"},
		{&l.createPixmapCursor, "XCreatePixmapCursor"},
		{&l.freePixmap, "XFreePixmap"},
		{&l.setSelectionOwner, "XSetSelectionOwner"},
		{&l.getSelectionOwner, "XGetSelectionOwner"},
		{&l.convertSelection, "XConvertSelection"},
		{&l.getWindowProperty, "XGetWindowProperty"},
		{&l.getAtomName, "XGetAtomName"},
		{&l.grabPointer, "XGrabPointer"},
		{&l.ungrabPointer, "XUngrabPointer"},
		{&l.warpPointer, "XWarpPointer"},
		{&l.displayString, "XDisplayString"},
		{&l.defaultVisual, "XDefaultVisual"},
		{&l.defaultDepth, "XDefaultDepth"},
		{&l.createGC, "XCreateGC"},
		{&l.freeGC, "XFreeGC"},
		{&l.createImage, "XCreateImage"},
		{&l.putImage, "XPutImage"},
		{&l.destroyImage, "XDestroyImage"},
	} {
		if err := x11.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	if err := xcb.Bind(&l.getXCBConnection, "XGetXCBConnection"); err != nil {
		return nil, err
	}
	return l, nil
}

// xext holds the functions of libXext's MIT-SHM, with which the software renderer shows pixels
// that the server reads from shared memory.
type xext struct {
	queryExtension func(display uintptr) int32
	getEventBase   func(display uintptr) int32
	attach         func(display, info uintptr) int32
	detach         func(display, info uintptr) int32
	createImage    func(display, visual uintptr, depth uint32, format int32, data, info uintptr, width, height uint32) uintptr
	putImage       func(display uintptr, drawable uint64, gc, image uintptr, srcX, srcY, dstX, dstY int32, width, height uint32, sendEvent int32) int32
}

func loadXext() (*xext, error) {
	lib, err := dl.Open(libXext, "libXext.so")
	if err != nil {
		return nil, err
	}
	l := new(xext)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.queryExtension, "XShmQueryExtension"},
		{&l.getEventBase, "XShmGetEventBase"},
		{&l.attach, "XShmAttach"},
		{&l.detach, "XShmDetach"},
		{&l.createImage, "XShmCreateImage"},
		{&l.putImage, "XShmPutImage"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// sysv holds the functions of libc that make a segment of System V shared memory, which MIT-SHM
// shares with the X server.
type sysv struct {
	shmget func(key int32, size uintptr, flags int32) int32
	shmat  func(id int32, addr uintptr, flags int32) uintptr
	shmdt  func(addr uintptr) int32
	shmctl func(id, cmd int32, buf uintptr) int32
}

func loadSysv() (*sysv, error) {
	lib, err := dl.Open("libc.so.6", "libc.so")
	if err != nil {
		return nil, err
	}
	l := new(sysv)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.shmget, "shmget"},
		{&l.shmat, "shmat"},
		{&l.shmdt, "shmdt"},
		{&l.shmctl, "shmctl"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// xi holds the bound functions of libXi, which the input needs.
type xi struct {
	queryVersion func(display, major, minor uintptr) int32
	selectEvents func(display uintptr, window uint64, masks uintptr, count int32) int32
	queryDevice  func(display uintptr, device int32, count uintptr) uintptr
	freeInfo     func(info uintptr)
	// ungrabDevice (XIUngrabDevice) ends the grab that a pressed button gives the
	// window, so that the window manager can take the pointer for a move or a resize.
	ungrabDevice func(display uintptr, device int32, time uint64) int32
}

func loadXi() (*xi, error) {
	lib, err := dl.Open(libXi, "libXi.so")
	if err != nil {
		return nil, err
	}
	l := new(xi)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.queryVersion, "XIQueryVersion"},
		{&l.selectEvents, "XISelectEvents"},
		{&l.queryDevice, "XIQueryDevice"},
		{&l.freeInfo, "XIFreeDeviceInfo"},
		{&l.ungrabDevice, "XIUngrabDevice"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// xrandr holds the functions of libXrandr: the refresh rate for the frame pacer,
// and the monitors.
type xrandr struct {
	getScreenInfo  func(display uintptr, root uint64) uintptr
	currentRate    func(config uintptr) int16
	freeScreenInfo func(config uintptr)

	queryExtension func(display, eventBase, errorBase uintptr) int32
	selectInput    func(display uintptr, window uint64, mask int32)
	getMonitors    func(display uintptr, window uint64, activeOnly int32, count uintptr) uintptr
	freeMonitors   func(monitors uintptr)
	updateConfig   func(event uintptr) int32
}

func loadXrandr() (*xrandr, error) {
	lib, err := dl.Open(libXrandr, "libXrandr.so")
	if err != nil {
		return nil, err
	}
	l := new(xrandr)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.getScreenInfo, "XRRGetScreenInfo"},
		{&l.currentRate, "XRRConfigCurrentRate"},
		{&l.freeScreenInfo, "XRRFreeScreenConfigInfo"},
		{&l.queryExtension, "XRRQueryExtension"},
		{&l.selectInput, "XRRSelectInput"},
		{&l.getMonitors, "XRRGetMonitors"},
		{&l.freeMonitors, "XRRFreeMonitors"},
		{&l.updateConfig, "XRRUpdateConfiguration"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// xcursor holds the functions of libXcursor that load cursors.
type xcursor struct {
	libraryLoadCursor func(display, name uintptr) uint64
	setTheme          func(display, name uintptr) int32
	setDefaultSize    func(display uintptr, size int32) int32
}

func loadXcursor() (*xcursor, error) {
	lib, err := dl.Open(libXcursor, "libXcursor.so")
	if err != nil {
		return nil, err
	}
	l := new(xcursor)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.libraryLoadCursor, "XcursorLibraryLoadCursor"},
		{&l.setTheme, "XcursorSetTheme"},
		{&l.setDefaultSize, "XcursorSetDefaultSize"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// The libraries load once; a failure is returned on every call. Xlib is
// required, the others are optional (SPEC §8.1).
var (
	loadXlibOnce    = sync.OnceValues(loadXlib)
	loadXiOnce      = sync.OnceValues(loadXi)
	loadXrandrOnce  = sync.OnceValues(loadXrandr)
	loadXcursorOnce = sync.OnceValues(loadXcursor)
	loadXextOnce    = sync.OnceValues(loadXext)
	loadSysvOnce    = sync.OnceValues(loadSysv)
)
