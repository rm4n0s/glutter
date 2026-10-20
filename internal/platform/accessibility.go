package platform

import (
	"context"

	"github.com/rm4n0s/glutter/ui"
)

// Accessibility is the bridge to the assistive technologies of the desktop (SPEC §8.10).
// The engine starts it, gives it the semantics trees of the views, and it asks for the
// semantics tree only while a screen reader is on. All methods but Close are for the
// UI thread; the host is called on the UI thread too.
//
// Flutter: no counterpart (the embedder's accessibility bridge)
type Accessibility interface {
	// Start follows the screen reader in the background until ctx ends or Close.
	// A desktop without an accessibility service is not an error: the bridge stays off.
	Start(ctx context.Context, host AccessibilityHost)
	// UpdateSemantics gives the bridge the changes to the semantics tree of a view.
	UpdateSemantics(view ui.ViewID, update *ui.SemanticsUpdate)
	// SetWindow tells the title of the window of the view, which names the frame of its
	// tree, and whether it has the keyboard focus.
	SetWindow(view ui.ViewID, title string, active bool)
	// RemoveView forgets the tree of a view that was closed.
	RemoveView(view ui.ViewID)
	// Close stops the bridge. It may be called from any goroutine, and twice.
	Close()
}

// AccessibilityHost is what the engine gives the bridge. Both methods are called on
// the UI thread.
//
// Flutter: no counterpart
type AccessibilityHost interface {
	// ScreenReaderChanged tells that a screen reader started or stopped: the framework
	// then makes the semantics tree, or stops making it
	// (PlatformDispatcher.SetSemanticsEnabled).
	ScreenReaderChanged(enabled bool)
	// PerformAction carries out the request of an assistive technology to perform an
	// action on a node (PlatformDispatcher.DispatchSemanticsActionEvent).
	PerformAction(event ui.SemanticsActionEvent)
}
