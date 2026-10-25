package windows

import (
	"bytes"
	"image"
	"image/draw"
	_ "image/jpeg" // the decoders of the balloon's large icon
	_ "image/png"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
	xdraw "golang.org/x/image/draw"
)

// This file is the plain-Go part of the notifications of Windows (SPEC §25.9): a notification is a
// tray balloon (NIF_INFO). Windows shows one balloon at a time and ignores the time-out the app asks
// for, so the notifications wait in a queue here, a balloon that is shown is replaced in place, and the
// events of the icon (the click, the time-out, the hide) end the balloon that is shown and start the
// next one.

// The limits of the balloon, in UTF-16 units without the NUL: szInfoTitle holds 64 and szInfo 256.
const (
	maxBalloonTitleUnits = 63
	maxBalloonBodyUnits  = 255
)

// balloonDefaultWait is how long a balloon whose notification has no time-out is waited for: the shell
// ends it much sooner (Windows 10 and 11 show it as a toast for a few seconds) and tells the app, but
// a shell that does not (Focus Assist) would stop the queue for ever, so the balloon is ended as
// expired after this time.
const balloonDefaultWait = 15 * time.Second

// balloonItem is a notification that is shown or waits.
type balloonItem struct {
	id   uint32
	spec mobile.NotificationSpec
	// stop cancels the timer of its time-out; icon is its large icon.
	stop      func()
	icon      uintptr
	iconOwned bool
}

// balloonQueue is the notifications of the app. It belongs to the UI thread.
type balloonQueue struct {
	d     *Desktop
	next  uint32
	shown *balloonItem
	queue []*balloonItem
	// owner says the hidden icon that owns the balloons exists (the app has no tray).
	owner      bool
	ownerIcon  uintptr
	ownerOwned bool
	// ignoreHide says the next NIN_BALLOONHIDE is the echo of a hide or a replacement the code made, or
	// of a click, and ends nothing.
	ignoreHide bool
}

// balloonText makes the title and the body of the balloon: the limits of the fields (63 and 255
// units, with an ellipsis). A body that is empty would hide the balloon, so it is a space.
func balloonText(spec mobile.NotificationSpec) (title, body string) {
	title = limitText(spec.Title, maxBalloonTitleUnits)
	body = limitText(spec.Body, maxBalloonBodyUnits)
	if body == "" {
		body = " "
	}
	return title, body
}

// balloonFlags are the dwInfoFlags of the urgency: Critical is the warning icon, Low is silent.
func balloonFlags(urgency mobile.Urgency) uint32 {
	switch urgency {
	case mobile.UrgencyCritical:
		return win32.NIIFWarning
	case mobile.UrgencyLow:
		return win32.NIIFInfo | win32.NIIFNoSound
	}
	return win32.NIIFInfo
}

// balloonWait is how long the balloon is waited for: the time-out the app gave, else balloonDefaultWait.
func balloonWait(spec mobile.NotificationSpec) time.Duration {
	if spec.Timeout > 0 {
		return spec.Timeout
	}
	return balloonDefaultWait
}

// hasDefaultAction says that the notification has the action "default", the click on the balloon.
func hasDefaultAction(spec mobile.NotificationSpec) bool {
	return slices.ContainsFunc(spec.Actions, func(a mobile.Action) bool { return a.ID == mobile.DefaultAction })
}

// ownerID is the notify icon that carries the balloons: the tray's, else the hidden icon of their own.
func (q *balloonQueue) ownerID() uint32 {
	if q.d.tray != nil {
		return trayIconID
	}
	return balloonIconID
}

// ensureOwner adds the hidden icon that owns the balloons when the app has no tray (NIS_HIDDEN, with the
// icon of the program, and version 4), unless it exists.
func (q *balloonQueue) ensureOwner() error {
	if q.d.tray != nil || q.owner {
		return nil
	}
	h, owned, err := q.d.shell.MakeIcon(ui.WindowIcon{}, false)
	if err != nil {
		return errors.New(tagNotificationsUnavailable, "the icon that owns the balloons could not be made", "reason", err.Error())
	}
	data := iconData{ID: balloonIconID, Flags: win32.NIFMessage | win32.NIFIcon | win32.NIFState, Icon: h,
		State: win32.NISHidden, StateMask: win32.NISHidden}
	if err := q.d.shell.Notify(win32.NIMAdd, data); err != nil {
		if owned {
			q.d.shell.FreeIcon(h)
		}
		return errors.New(tagNotificationsUnavailable, "the shell did not take the icon that owns the balloons: this session has no notification area", "reason", err.Error())
	}
	if err := q.d.shell.Notify(win32.NIMSetVersion, iconData{ID: balloonIconID}); err != nil {
		q.d.log.Debug("the shell does not take version 4 of the notify icon; its old messages are read", "err", err)
	}
	q.owner, q.ownerIcon, q.ownerOwned = true, h, owned
	return nil
}

// releaseOwner deletes the hidden icon, when the last balloon is gone.
func (q *balloonQueue) releaseOwner() {
	if !q.owner {
		return
	}
	if err := q.d.shell.Notify(win32.NIMDelete, iconData{ID: balloonIconID}); err != nil {
		q.d.log.Debug("the icon that owns the balloons could not be deleted", "err", err)
	}
	if q.ownerOwned && q.ownerIcon != 0 {
		q.d.shell.FreeIcon(q.ownerIcon)
	}
	q.owner, q.ownerIcon, q.ownerOwned = false, 0, false
}

// readd adds the hidden icon again after Explorer restarted.
func (q *balloonQueue) readd() {
	if !q.owner {
		return
	}
	data := iconData{ID: balloonIconID, Flags: win32.NIFMessage | win32.NIFIcon | win32.NIFState, Icon: q.ownerIcon,
		State: win32.NISHidden, StateMask: win32.NISHidden}
	if err := q.d.shell.Notify(win32.NIMAdd, data); err != nil {
		q.d.log.Warn("the icon that owns the balloons could not be added again after Explorer restarted", "err", err)
		return
	}
	if err := q.d.shell.Notify(win32.NIMSetVersion, iconData{ID: balloonIconID}); err != nil {
		q.d.log.Debug("the shell does not take version 4 of the notify icon", "err", err)
	}
}

// display shows the item as the balloon of the owner icon (NIM_MODIFY with NIF_INFO) and arms the timer
// of its time-out. A shell that refuses is WindowsBalloonFailed.
func (q *balloonQueue) display(it *balloonItem) error {
	if err := q.ensureOwner(); err != nil {
		return err
	}
	title, body := balloonText(it.spec)
	data := iconData{ID: q.ownerID(), Flags: win32.NIFInfo, Info: body, InfoTitle: title, InfoFlags: balloonFlags(it.spec.Urgency)}
	if it.icon == 0 && it.spec.Icon != "" {
		if img, ok := readBalloonIcon(it.spec.Icon); ok {
			if h, owned, err := q.d.shell.MakeIcon(ui.WindowIcon{Images: []ui.WindowIconImage{img}}, true); err == nil {
				it.icon, it.iconOwned = h, owned
			} else {
				q.d.log.Debug("the icon of the notification could not be made", "err", err)
			}
		}
	}
	if it.icon != 0 {
		data.InfoFlags = win32.NIIFUser | win32.NIIFLargeIcon | (data.InfoFlags & win32.NIIFNoSound)
		data.BalloonIcon = it.icon
	}
	if err := q.d.shell.Notify(win32.NIMModify, data); err != nil {
		q.freeIcon(it)
		if q.shown == nil && len(q.queue) == 0 {
			q.releaseOwner()
		}
		return errors.New(tagWindowsBalloonFailed, "the shell did not take the balloon", "reason", err.Error(), "title", title)
	}
	if it.stop != nil {
		it.stop()
	}
	id := it.id
	it.stop = q.d.shell.After(balloonWait(it.spec), func() {
		if q.shown != nil && q.shown.id == id {
			q.hide()
			q.end(mobile.CloseExpired)
		}
	})
	return nil
}

// hide takes the balloon off the screen (NIM_MODIFY with an empty text); the echo it may cause is
// ignored.
func (q *balloonQueue) hide() {
	q.ignoreHide = true
	err := q.d.shell.Notify(win32.NIMModify, iconData{ID: q.ownerID(), Flags: win32.NIFInfo})
	if err != nil {
		q.ignoreHide = false
		q.d.log.Debug("the balloon could not be hidden", "err", err)
	}
}

func (q *balloonQueue) freeIcon(it *balloonItem) {
	if it.iconOwned && it.icon != 0 {
		q.d.shell.FreeIcon(it.icon)
	}
	it.icon, it.iconOwned = 0, false
}

// newID returns the ID of a new notification: never 0, and not the ID of one that is alive.
func (q *balloonQueue) newID() uint32 {
	for {
		q.next++
		if q.next == 0 {
			continue
		}
		if q.shown != nil && q.shown.id == q.next {
			continue
		}
		if slices.ContainsFunc(q.queue, func(it *balloonItem) bool { return it.id == q.next }) {
			continue
		}
		return q.next
	}
}

// show shows the notification now when no balloon is on the screen, and queues it otherwise. A
// notification that replaces the one shown takes its place in place (same ID, the old one's OnClose is
// not called, as with the other platforms), and one that replaces a queued one replaces it in the queue.
func (q *balloonQueue) show(spec mobile.NotificationSpec) (uint32, error) {
	if id := spec.ReplacesID; id != 0 {
		if q.shown != nil && q.shown.id == id {
			it := &balloonItem{id: id, spec: spec}
			old := q.shown
			q.freeIcon(old)
			if old.stop != nil {
				old.stop()
			}
			if err := q.display(it); err != nil {
				// The old balloon is still on the screen but nobody waits for it any more.
				q.shown = nil
				q.startNext()
				return 0, err
			}
			q.shown = it
			return id, nil
		}
		for i, it := range q.queue {
			if it.id == id {
				q.queue[i] = &balloonItem{id: id, spec: spec}
				return id, nil
			}
		}
	}
	it := &balloonItem{id: q.newID(), spec: spec}
	if q.shown == nil {
		if err := q.display(it); err != nil {
			return 0, err
		}
		q.shown = it
		return it.id, nil
	}
	q.queue = append(q.queue, it)
	return it.id, nil
}

// close closes the notification with the ID: the balloon that is shown is hidden and the next starts; a
// queued one leaves the queue. Its OnClose gets CloseByApp.
func (q *balloonQueue) close(id uint32) {
	if q.shown != nil && q.shown.id == id {
		q.hide()
		q.end(mobile.CloseByApp)
		return
	}
	for i, it := range q.queue {
		if it.id == id {
			q.queue = slices.Delete(q.queue, i, i+1)
			if it.spec.OnClose != nil {
				it.spec.OnClose(mobile.CloseByApp)
			}
			return
		}
	}
}

// end finishes the balloon that is shown with the reason, calls its OnClose, and starts the next one.
func (q *balloonQueue) end(reason mobile.CloseReason) {
	it := q.shown
	if it == nil {
		return
	}
	q.shown = nil
	if it.stop != nil {
		it.stop()
		it.stop = nil
	}
	q.freeIcon(it)
	if it.spec.OnClose != nil {
		it.spec.OnClose(reason)
	}
	q.startNext()
}

// startNext shows the first queued notification that the shell takes; one that it refuses is told its
// OnClose with CloseUnknown and logged. With nothing left, the hidden owner icon is deleted.
func (q *balloonQueue) startNext() {
	for len(q.queue) > 0 {
		it := q.queue[0]
		q.queue = q.queue[1:]
		if err := q.display(it); err != nil {
			q.d.log.Warn("a queued notification could not be shown", "err", err)
			if it.spec.OnClose != nil {
				it.spec.OnClose(mobile.CloseUnknown)
			}
			continue
		}
		q.shown = it
		return
	}
	q.releaseOwner()
}

// event handles an event of the shell about the balloon that is shown. NIN_BALLOONUSERCLICK is the
// click: the action "default" is called and the balloon is dismissed. NIN_BALLOONTIMEOUT is the shell
// ending it (CloseExpired), and NIN_BALLOONHIDE is the balloon disappearing otherwise (the user
// dismissed it). NIN_BALLOONSHOW says nothing.
func (q *balloonQueue) event(event uint32) {
	switch event {
	case win32.NINBalloonUserClick:
		it := q.shown
		if it == nil {
			return
		}
		if it.spec.OnAction != nil && hasDefaultAction(it.spec) {
			it.spec.OnAction(mobile.DefaultAction)
		}
		// A click may be followed by a NIN_BALLOONHIDE of the same balloon, which must not end the next.
		q.ignoreHide = true
		q.end(mobile.CloseDismissed)
	case win32.NINBalloonTimeout:
		q.end(mobile.CloseExpired)
	case win32.NINBalloonHide:
		if q.ignoreHide {
			q.ignoreHide = false
			return
		}
		q.end(mobile.CloseDismissed)
	}
}

// shutdown ends the queue: the timers stop, the icons are freed and the owner icon is deleted.
func (q *balloonQueue) shutdown() {
	for _, it := range append([]*balloonItem{q.shown}, q.queue...) {
		if it == nil {
			continue
		}
		if it.stop != nil {
			it.stop()
		}
		q.freeIcon(it)
	}
	q.shown, q.queue = nil, nil
	q.releaseOwner()
}

// balloonIconSize is the side, in pixels, that a file for the large icon of a balloon is scaled to.
const balloonIconSize = 32

// readBalloonIcon reads the picture of a notification's Icon when it names an image file (a PNG or a
// JPEG, by its content) and scales it to a square of the size of a large icon. A theme icon name, a
// missing file or any other content is not an error: the balloon has no large icon then.
func readBalloonIcon(name string) (ui.WindowIconImage, bool) {
	if !strings.ContainsAny(name, `/\`) {
		return ui.WindowIconImage{}, false // a name of an icon theme
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return ui.WindowIconImage{}, false
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ui.WindowIconImage{}, false
	}
	dst := image.NewNRGBA(image.Rect(0, 0, balloonIconSize, balloonIconSize))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return ui.WindowIconImage{Size: balloonIconSize, Scale: 1, RGBA: dst.Pix}, true
}

// The tag of the error that this file makes (rm4n0s/errors, R5).
const tagWindowsBalloonFailed = "WindowsBalloonFailed"
