package windows

// modalGate counts the modal dialogs that block a window (SPEC §25.4): the window is disabled when the
// first one opens and enabled again when the last one closes, so that two dialogs of the same window
// do not enable it while the other is still open. set does the call to Windows (EnableWindow).
//
// Flutter: shell/platform/windows/host_window.cc HostWindow::UpdateModalStateLayer
//
// Deviation: Flutter keeps only the newest dialog of an owner enabled; here every dialog is (SPEC §12.10).
type modalGate struct{ n int }

// block adds a dialog that blocks the window; the first one disables it.
func (g *modalGate) block(set func(enabled bool)) {
	g.n++
	if g.n == 1 {
		set(false)
	}
}

// unblock removes a dialog; the last one enables the window. A call with no dialog left does nothing.
func (g *modalGate) unblock(set func(enabled bool)) {
	if g.n == 0 {
		return
	}
	g.n--
	if g.n == 0 {
		set(true)
	}
}

// blocked says whether a modal dialog blocks the window.
func (g *modalGate) blocked() bool { return g.n > 0 }
