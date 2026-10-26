//go:build windows

package windows

import (
	"log/slog"
	"runtime"
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// comWorkers are the threads of Worker that wait for a call, shared by the whole process (like
// ole.Shared): Wine's RPC server keeps an endpoint for each COM apartment that registered an interface,
// which every file dialog does (the drop target of its folder view), and never removes it when the
// apartment ends. The server waits on all its endpoints at once, so it fails for good with
// ERROR_INVALID_PARAMETER at the 64th (WaitForMultipleObjects takes at most 64 handles), and every
// dialog after that waits for ever for a server that is gone. A thread for each call used up the
// endpoints after some thirty dialogs; reused threads need one apartment for each call that runs at the
// same time.
var comWorkers struct {
	sync.Mutex
	idle []*comWorker
}

// comWorker is one thread of Worker. It gets its calls through jobs, which holds one at most: a worker
// is in the idle list before it is given a call.
type comWorker struct {
	jobs chan workerJob
}

// workerJob is a call of Worker and the log of the platform that made it.
type workerJob struct {
	fn  func() func()
	log *slog.Logger
}

// Worker runs fn on a goroutine locked to an OS thread whose COM apartment is a single-threaded one
// (CoInitializeEx), for the calls that wait for the user: the file dialogs and ShellExecuteExW (SPEC
// §6.3 rule 12, §25.9). What fn returns is called when fn is done and the thread is free for another
// call: it hands the answer to the caller, who may make the next call at once and finds the thread
// idle. The thread and its apartment are kept, and end with the process; a call made while the
// others are busy gets a thread of its own (comWorkers).
//
// Flutter: no counterpart
func (s winShell) Worker(fn func() func()) {
	job := workerJob{fn: fn, log: s.p.log}
	comWorkers.Lock()
	if n := len(comWorkers.idle); n > 0 {
		w := comWorkers.idle[n-1]
		comWorkers.idle = comWorkers.idle[:n-1]
		comWorkers.Unlock()
		w.jobs <- job
		return
	}
	comWorkers.Unlock()
	w := &comWorker{jobs: make(chan workerJob, 1)}
	w.jobs <- job
	go w.run()
}

// run is the goroutine of a worker thread: it never unlocks the thread, which ends with the process
// (the apartment is not given up: Wine's file dialog leaves it one reference deeper than it
// gives back, so it would end with the thread, in the middle of another call).
func (w *comWorker) run() {
	runtime.LockOSThread()
	first := true
	for job := range w.jobs {
		if first {
			first = false
			if err := win32.CoInitializeEx(win32.COInitApartmentThreaded); err != nil {
				job.log.Warn("COM could not be initialized on a worker thread; the calls go on without it", "err", err)
			}
		}
		deliver := job.fn()
		comWorkers.Lock()
		comWorkers.idle = append(comWorkers.idle, w)
		comWorkers.Unlock()
		if deliver != nil {
			deliver()
		}
	}
}

// ThreadID is the ID of the calling thread.
//
// Flutter: no counterpart
func (s winShell) ThreadID() uint32 { return win32.GetCurrentThreadID() }

// CancelDialog posts the message of the Cancel button (WM_COMMAND with IDCANCEL) to the top-level
// windows of the thread, which closes the dialog that is open there.
//
// Flutter: no counterpart
func (s winShell) CancelDialog(thread uint32) {
	win32.EnumThreadWindows(thread, func(hwnd uintptr) bool {
		if err := win32.PostMessage(hwnd, win32.WMCommand, win32.IDCancel, 0); err != nil {
			s.p.log.Debug("the Cancel message could not be posted to a dialog", "err", err)
		}
		return true
	})
}
