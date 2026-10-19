package platform

import (
	"context"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// Platform is the contract between the engine and an operating system backend.
// Only the engine talks to it, and only on the UI thread, except for Post.
//
// Flutter: no counterpart (the embedder API)
type Platform interface {
	// Run owns the UI thread's event loop. It delivers events to h and returns
	// nil when ctx is cancelled or the platform has nothing left to show. A
	// second call while Run is active fails with PlatformAlreadyRunning.
	Run(ctx context.Context, h Handler) error
	// Post runs fn on the UI thread. It is safe to call from any goroutine and
	// wakes the loop. Functions posted by one goroutine run in the order that
	// goroutine posted them.
	Post(fn func())
	// NewWindow creates a top-level window; the engine turns it into a view.
	NewWindow(cfg WindowConfig) (Window, error)
	// Displays lists the connected displays.
	Displays() []Display
	// Clipboard returns the system clipboard.
	Clipboard() Clipboard
	// Settings returns the current system settings. Changes arrive through
	// Handler.OnSettings.
	Settings() Settings
	// Services returns the optional desktop capabilities of this platform, or
	// nil when it has none.
	Services() Services
}

// Window is one top-level native window. All methods are for the UI thread.
//
// Flutter: no counterpart (the embedder's view API)
type Window interface {
	// ID returns the ID of the view this window shows.
	ID() ui.ViewID
	// NativeSurface returns the handle that a GPU backend renders into.
	NativeSurface() gpu.SurfaceHandle
	// SetTitle sets the window title.
	SetTitle(title string)
	// SetSize asks for a new window size, in logical pixels.
	SetSize(size ui.Size)
	// SetState asks for the window to be maximized, minimized, fullscreen or
	// back to normal.
	SetState(state WindowState)
	// SetCursor sets the mouse cursor shown over the window.
	SetCursor(cursor MouseCursor)
	// SetTextInput tells the IME about the focused text field. A zero
	// ui.TextInputState with Enabled false disables text input.
	SetTextInput(state ui.TextInputState)
	// RequestFrame asks for Handler.OnFrame to be called for this window, at the
	// next moment the compositor wants a frame. At most one frame is in flight
	// per window: requests made while one is in flight are merged into one more
	// frame, after the compositor is done with the first. The handler must
	// present exactly one frame for each OnFrame, or no further frame follows.
	RequestFrame()
	// StartDrag starts a drag-and-drop operation from this window.
	StartDrag(data DragData) error
	// Close destroys the window. Destroy the GPU surface of the window first.
	Close()
	// TODO(step 1.5): more setters (min/max size, icon, decorations).
}

// Handler receives everything the platform reports. The engine implements it.
// Its methods are called on the UI thread, inside Platform.Run.
//
// Flutter: no counterpart (the embedder's callbacks into the engine)
type Handler interface {
	// OnPointer delivers a batch of pointer reports, in order.
	OnPointer(data []ui.PointerData)
	// OnKey delivers one key event.
	OnKey(data ui.KeyData)
	// OnTextInput delivers a text-input event from the IME.
	OnTextInput(event ui.TextInputEvent)
	// OnFrame asks for a frame of the view. The duration is the frame time
	// since an arbitrary fixed point in the past.
	OnFrame(view ui.ViewID, frameTime time.Duration)
	// OnMetrics reports a new size or scale of the view.
	OnMetrics(view ui.ViewID, metrics ViewMetrics)
	// OnLifecycle reports a change of the application lifecycle state.
	OnLifecycle(state ui.AppLifecycleState)
	// OnDrop delivers a drag-and-drop event over the view.
	OnDrop(view ui.ViewID, event DropEvent)
	// OnSettings reports changed system settings.
	OnSettings(settings Settings)
	// OnCloseRequest reports that the user asked to close the view. Returning
	// true closes the window; false keeps it open.
	OnCloseRequest(view ui.ViewID) bool
}
