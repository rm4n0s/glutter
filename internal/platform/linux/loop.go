//go:build linux && !android

package linux

import (
	"context"
	"encoding/binary"
	"sort"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// Events says what is ready on the file descriptor of a Source.
//
// Flutter: no counterpart
type Events uint32

// The bits of Events.
const (
	// Readable means the descriptor has data to read. It is also set when the
	// descriptor hung up or failed, so that reading reveals the error.
	//
	// Flutter: no counterpart
	Readable Events = 1 << iota
	// Writable means the descriptor accepts data. It is reported only to a
	// Source that WantsWrite.
	//
	// Flutter: no counterpart
	Writable
	// Hangup means the other end closed the descriptor or it failed.
	//
	// Flutter: no counterpart
	Hangup
)

// Source is a file descriptor that the Loop waits on, together with what to do
// before and after each wait. The Wayland connection is one; X11 and D-Bus
// connections are others.
//
// Flutter: no counterpart
type Source interface {
	// FD is the descriptor to wait on.
	FD() int
	// WantsWrite reports whether the loop should also wait for the descriptor
	// to become writable, because earlier output could not be sent in full.
	WantsWrite() bool
	// BeforeWait is called before every wait: flush output and prepare to read.
	// An error ends the loop.
	BeforeWait() error
	// AfterWait is called after every wait, with what is ready (possibly
	// nothing, when something else woke the loop): read and dispatch events. An
	// error ends the loop.
	AfterWait(ready Events) error
}

// Loop is the event loop of the UI thread: it waits with epoll on its sources,
// an eventfd for Post and a timerfd for timers (docs/SPEC.md §8.4). An idle loop
// sleeps in epoll_wait with no timer armed.
//
// Run, AddSource, RemoveSource, AfterFunc and Timer.Stop belong to the loop's
// thread, which is the goroutine that calls Run (call them before Run, or from
// functions the loop runs). Post, Stop and Close are safe from any goroutine.
//
// Flutter: no counterpart
type Loop struct {
	epfd, wakeFD, timerFD int

	mu        sync.Mutex // guards the fields below
	posted    []func()
	signalled bool // an eventfd write for the posted functions is pending
	stopped   bool

	// Loop-thread state.
	sources map[int]*sourceEntry
	timers  []*Timer
	nextSeq uint64
}

type sourceEntry struct {
	src   Source
	write bool // EPOLLOUT is in the epoll interest list
}

// NewLoop creates the epoll instance, the eventfd and the timerfd. Close
// releases them.
//
// Flutter: no counterpart
func NewLoop() (*Loop, error) {
	l := &Loop{sources: map[int]*sourceEntry{}, epfd: -1, wakeFD: -1, timerFD: -1}
	var err error
	if l.epfd, err = syscall.EpollCreate1(syscall.EPOLL_CLOEXEC); err != nil {
		return nil, errors.NewErr(tagLoopSyscallFailed, err).SetMetadata("op", "epoll_create1")
	}
	r, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, 0, syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if errno != 0 {
		_ = l.Close()
		return nil, errors.NewErr(tagLoopSyscallFailed, errno).SetMetadata("op", "eventfd2")
	}
	l.wakeFD = int(r)
	r, _, errno = syscall.Syscall(syscall.SYS_TIMERFD_CREATE, clockMonotonic, syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if errno != 0 {
		_ = l.Close()
		return nil, errors.NewErr(tagLoopSyscallFailed, errno).SetMetadata("op", "timerfd_create")
	}
	l.timerFD = int(r)
	for _, fd := range []int{l.wakeFD, l.timerFD} {
		ev := syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fd)}
		if err := syscall.EpollCtl(l.epfd, syscall.EPOLL_CTL_ADD, fd, &ev); err != nil {
			_ = l.Close()
			return nil, errors.NewErr(tagLoopSyscallFailed, err).SetMetadata("op", "epoll_ctl")
		}
	}
	return l, nil
}

// clockMonotonic is CLOCK_MONOTONIC.
const clockMonotonic = 1

// Close releases the descriptors of the loop. Do not use the loop afterwards.
//
// Flutter: no counterpart
func (l *Loop) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var first error
	for _, fd := range []*int{&l.timerFD, &l.wakeFD, &l.epfd} {
		if *fd < 0 {
			continue
		}
		if err := syscall.Close(*fd); err != nil && first == nil {
			first = errors.NewErr(tagLoopSyscallFailed, err).SetMetadata("op", "close")
		}
		*fd = -1
	}
	return first
}

// wake makes epoll_wait return. It holds the lock while it writes, so a
// concurrent Close cannot free the descriptor, and let the number be reused,
// between reading it and writing to it.
func (l *Loop) wake() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.wakeFD < 0 {
		return // closed
	}
	var one [8]byte
	binary.LittleEndian.PutUint64(one[:], 1)
	// A full eventfd counter (EAGAIN) means a wakeup is already pending.
	_, _ = syscall.Write(l.wakeFD, one[:])
}

// Post queues fn to run on the loop's thread. It is safe to call from any
// goroutine and wakes the loop. The functions of one goroutine run in the order
// that goroutine posted them.
//
// Flutter: no counterpart
func (l *Loop) Post(fn func()) {
	l.mu.Lock()
	l.posted = append(l.posted, fn)
	needWake := !l.signalled
	l.signalled = true
	l.mu.Unlock()
	if needWake {
		l.wake()
	}
}

// Stop makes Run return nil as soon as it can. It is safe from any goroutine.
//
// Flutter: no counterpart
func (l *Loop) Stop() {
	l.mu.Lock()
	l.stopped = true
	l.mu.Unlock()
	l.wake()
}

func (l *Loop) isStopped() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stopped
}

// AddSource starts waiting on the source's descriptor.
//
// Flutter: no counterpart
func (l *Loop) AddSource(s Source) error {
	fd := s.FD()
	ev := syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fd)}
	if err := syscall.EpollCtl(l.epfd, syscall.EPOLL_CTL_ADD, fd, &ev); err != nil {
		return errors.NewErr(tagLoopSyscallFailed, err).SetMetadata("op", "epoll_ctl_add", "fd", fd)
	}
	l.sources[fd] = &sourceEntry{src: s}
	return nil
}

// RemoveSource stops waiting on the source's descriptor. Call it before closing
// the descriptor.
//
// Flutter: no counterpart
func (l *Loop) RemoveSource(s Source) error {
	fd := s.FD()
	delete(l.sources, fd)
	if err := syscall.EpollCtl(l.epfd, syscall.EPOLL_CTL_DEL, fd, nil); err != nil {
		return errors.NewErr(tagLoopSyscallFailed, err).SetMetadata("op", "epoll_ctl_del", "fd", fd)
	}
	return nil
}

// Timer is a function scheduled with Loop.AfterFunc.
//
// Flutter: no counterpart
type Timer struct {
	loop *Loop
	when time.Time
	seq  uint64
	fn   func()
	live bool
}

// AfterFunc runs fn on the loop's thread after at least d. A timer that is due
// runs on the next pass of the loop.
//
// Flutter: no counterpart
func (l *Loop) AfterFunc(d time.Duration, fn func()) (*Timer, error) {
	l.nextSeq++
	t := &Timer{loop: l, when: time.Now().Add(d), seq: l.nextSeq, fn: fn, live: true}
	l.timers = append(l.timers, t)
	sort.SliceStable(l.timers, func(i, j int) bool {
		if !l.timers[i].when.Equal(l.timers[j].when) {
			return l.timers[i].when.Before(l.timers[j].when)
		}
		return l.timers[i].seq < l.timers[j].seq
	})
	return t, l.armTimer()
}

// Stop cancels the timer. It reports whether the timer was still waiting.
//
// Flutter: no counterpart
func (t *Timer) Stop() bool {
	if !t.live {
		return false
	}
	t.live = false
	l := t.loop
	for i, other := range l.timers {
		if other == t {
			l.timers = append(l.timers[:i], l.timers[i+1:]...)
			break
		}
	}
	_ = l.armTimer() // the earliest deadline may have changed
	return true
}

// itimerspec is struct itimerspec: the interval and the first expiry.
type itimerspec struct {
	intervalSec, intervalNsec int64
	valueSec, valueNsec       int64
}

// armTimer points the timerfd at the earliest timer, or disarms it.
func (l *Loop) armTimer() error {
	var spec itimerspec
	if len(l.timers) > 0 {
		d := max(time.Until(l.timers[0].when), time.Nanosecond) // zero would disarm
		spec.valueSec, spec.valueNsec = int64(d/time.Second), int64(d%time.Second)
	}
	_, _, errno := syscall.Syscall6(syscall.SYS_TIMERFD_SETTIME, uintptr(l.timerFD), 0, uintptr(unsafe.Pointer(&spec)), 0, 0, 0)
	if errno != 0 {
		return errors.NewErr(tagLoopSyscallFailed, errno).SetMetadata("op", "timerfd_settime")
	}
	return nil
}

// runTimers runs the timers that are due, in deadline order.
func (l *Loop) runTimers() error {
	for len(l.timers) > 0 && !l.timers[0].when.After(time.Now()) {
		t := l.timers[0]
		l.timers = l.timers[1:]
		t.live = false
		t.fn()
	}
	return l.armTimer()
}

// runPosted runs the functions posted so far.
func (l *Loop) runPosted() {
	l.mu.Lock()
	batch := l.posted
	l.posted = nil
	l.signalled = false
	l.mu.Unlock()
	for _, fn := range batch {
		fn()
	}
}

// drain empties an eventfd or timerfd counter.
func drain(fd int) {
	var buf [8]byte
	_, _ = syscall.Read(fd, buf[:])
}

// Run runs the loop on the calling goroutine until ctx is cancelled or Stop is
// called, and then returns nil, or until a source fails, and then returns its
// error unchanged. Each pass: BeforeWait of every source, epoll_wait, AfterWait
// of every source (display events), the due timers, and the posted functions.
//
// Flutter: no counterpart
func (l *Loop) Run(ctx context.Context) error {
	stopWatch := context.AfterFunc(ctx, l.wake)
	defer stopWatch()
	events := make([]syscall.EpollEvent, 32)
	for {
		if ctx.Err() != nil || l.isStopped() {
			return nil
		}
		for fd, e := range l.sources {
			if err := e.src.BeforeWait(); err != nil {
				return err
			}
			if want := e.src.WantsWrite(); want != e.write {
				mask := uint32(syscall.EPOLLIN)
				if want {
					mask |= syscall.EPOLLOUT
				}
				ev := syscall.EpollEvent{Events: mask, Fd: int32(fd)}
				if err := syscall.EpollCtl(l.epfd, syscall.EPOLL_CTL_MOD, fd, &ev); err != nil {
					return errors.NewErr(tagLoopSyscallFailed, err).SetMetadata("op", "epoll_ctl_mod", "fd", fd)
				}
				e.write = want
			}
		}

		n, err := syscall.EpollWait(l.epfd, events, -1)
		if err != nil && err != syscall.EINTR {
			return errors.NewErr(tagLoopSyscallFailed, err).SetMetadata("op", "epoll_wait")
		}
		if n < 0 {
			n = 0
		}
		ready := map[int]uint32{}
		for _, ev := range events[:n] {
			ready[int(ev.Fd)] |= ev.Events
		}

		for fd, e := range l.sources {
			if err := e.src.AfterWait(toEvents(ready[fd])); err != nil {
				return err
			}
		}
		if ready[l.timerFD] != 0 {
			drain(l.timerFD)
		}
		if err := l.runTimers(); err != nil {
			return err
		}
		if ready[l.wakeFD] != 0 {
			drain(l.wakeFD)
			l.runPosted()
		}
	}
}

// toEvents converts epoll bits to Events.
func toEvents(e uint32) Events {
	var out Events
	if e&syscall.EPOLLIN != 0 {
		out |= Readable
	}
	if e&syscall.EPOLLOUT != 0 {
		out |= Writable
	}
	if e&(syscall.EPOLLHUP|syscall.EPOLLERR) != 0 {
		out |= Hangup | Readable
	}
	return out
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagLoopSyscallFailed = "LoopSyscallFailed"
)
