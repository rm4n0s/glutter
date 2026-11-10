//go:build darwin

package notify

import (
	"context"
	"os"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
)

// The glue of the notifications (SPEC §29.9, §30.6): Center is Backend over UNUserNotificationCenter, whose rules
// are in notify.go. The center needs a bundle with an identifier: without one, the class itself raises an exception,
// so nothing of it is touched (Ready).

// The values of UserNotifications that the glue uses.
const (
	unAuthBadge, unAuthSound, unAuthAlert = 1 << 0, 1 << 1, 1 << 2
	// The presentation options of a notification while the app is in front: a banner, the list and the sound.
	unPresentSound, unPresentList, unPresentBanner = 1 << 1, 1 << 3, 1 << 4
	// unCategoryCustomDismiss makes the center tell the delegate when the user dismisses a notification.
	unCategoryCustomDismiss = 1 << 0
	// The interruption levels: passive makes no sound and does not light the screen; time sensitive breaks through.
	unLevelPassive, unLevelTimeSensitive = 0, 2
	unDefaultActionID, unDismissActionID = "com.apple.UNNotificationDefaultActionIdentifier", "com.apple.UNNotificationDismissActionIdentifier"
)

// Center is the notification center of the app: Backend over UNUserNotificationCenter, on macOS and on iOS.
//
// Flutter: no counterpart
type Center struct {
	// notes receives the responses of the user.
	notes func() *Notifier

	once     sync.Once
	loadErr  error
	delegate objc.ID
}

// NewCenter makes the center. notes gives it the notifier that the responses of the user go to, which may not exist
// yet when the center is made; it returns nil while there is none.
//
// Flutter: no counterpart
func NewCenter(notes func() *Notifier) *Center { return &Center{notes: notes} }

func class(name string) objc.ID { return objc.ID(objcx.MustClass(name)) }

func sel(name string) objc.SEL { return objcx.Sel(name) }

func send[T any](id objc.ID, name string, args ...any) T { return objc.Send[T](id, sel(name), args...) }

// BundleID is the identifier of the main bundle, "" for a program that is not in one.
//
// Flutter: no counterpart
func BundleID() string {
	return objcx.String(class("NSBundle").Send(sel("mainBundle")).Send(sel("bundleIdentifier")))
}

// Ready says whether notifications can be shown.
//
// Flutter: no counterpart
func (c *Center) Ready() (bool, string) {
	if BundleID() == "" {
		return false, "the program is not in an application bundle with an identifier, which UserNotifications needs; build one with glutter-macos or glutter-ios"
	}
	c.once.Do(func() { _, c.loadErr = dl.OpenFramework("UserNotifications") })
	if c.loadErr != nil {
		return false, "UserNotifications could not be loaded: " + c.loadErr.Error()
	}
	return true, ""
}

// handle returns the center, and sets the delegate the first time.
func (c *Center) handle() objc.ID {
	cn := class("UNUserNotificationCenter").Send(sel("currentNotificationCenter"))
	if c.delegate == 0 {
		cls, err := notificationDelegateClass()
		if err == nil {
			c.delegate = cls.New(objcx.NewHandle(c))
			cn.Send(sel("setDelegate:"), c.delegate)
		}
	}
	return cn
}

// notificationDelegateClass is the class of the delegate, registered once for the process.
var notificationDelegateClass = sync.OnceValues(func() (*objcx.RegisteredClass, error) {
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "NotificationDelegate", Super: "NSObject", Protocols: []string{"UNUserNotificationCenterDelegate"},
		Methods: []objcx.Method{
			// A notification that arrives while the app is in front is shown all the same.
			{Name: "userNotificationCenter:willPresentNotification:withCompletionHandler:", Fn: func(_ objc.ID, _ objc.SEL, _, _, handler objc.ID) {
				_, _ = objc.InvokeBlock[struct{}](objc.Block(handler), uint64(unPresentBanner|unPresentList|unPresentSound))
			}},
			{Name: "userNotificationCenter:didReceiveNotificationResponse:withCompletionHandler:", Fn: func(self objc.ID, _ objc.SEL, _, resp, handler objc.ID) {
				if c, ok := objcx.LookupAs[*Center](objcx.HandleOf(self)); ok {
					if n := c.notes(); n != nil {
						action := objcx.String(send[objc.ID](resp, "actionIdentifier"))
						ident := objcx.String(send[objc.ID](send[objc.ID](send[objc.ID](resp, "notification"), "request"), "identifier"))
						switch action {
						case unDefaultActionID:
							action = mobile.DefaultAction
						case unDismissActionID:
							action = ""
						}
						n.Response(ident, action)
					}
				}
				_, _ = objc.InvokeBlock[struct{}](objc.Block(handler))
			}},
		},
	})
})

// await calls start with a block that takes the error of a completion handler and waits for the error to arrive or
// for ctx to end. start gets the block to pass to the system; the block is kept until the handler has run.
func await(ctx context.Context, start func(blk objc.Block)) (errText string, err error) {
	type reply struct{ text string }
	ch := make(chan reply, 1)
	blk := objcx.NewBlock(func(_ objc.Block, e objc.ID) {
		r := reply{}
		if e != 0 {
			r.text = objcx.String(send[objc.ID](e, "localizedDescription"))
			if r.text == "" {
				r.text = "the system gave an error without a description"
			}
		}
		ch <- r
	})
	kept := blk.Copy()
	blk.Release()
	start(kept)
	select {
	case r := <-ch:
		kept.Release()
		return r.text, nil
	case <-ctx.Done():
		return "", ctx.Err() // the block stays alive for the handler that may still come
	}
}

// Authorize asks for the permission to show alerts, sounds and badges; the system answers at once when the user
// decided before.
//
// Flutter: no counterpart
func (c *Center) Authorize(ctx context.Context) (bool, error) {
	cn := c.handle()
	type reply struct{ granted bool }
	ch := make(chan reply, 1)
	blk := objcx.NewBlock(func(_ objc.Block, granted bool, _ objc.ID) { ch <- reply{granted} })
	kept := blk.Copy()
	blk.Release()
	cn.Send(sel("requestAuthorizationWithOptions:completionHandler:"), uint64(unAuthAlert|unAuthSound|unAuthBadge), kept)
	select {
	case r := <-ch:
		kept.Release()
		return r.granted, nil
	case <-ctx.Done():
		return false, errors.NewErr(tagNotificationsUnavailable, ctx.Err()).SetMetadata("reason", "the context ended while the permission was asked")
	}
}

// SetCategories gives the center the categories, each with its buttons.
//
// Flutter: no counterpart
func (c *Center) SetCategories(cats []Category) error {
	cn := c.handle()
	objcx.WithPool(func() {
		var objs []objc.ID
		for _, cat := range cats {
			var acts []objc.ID
			for _, a := range cat.Actions {
				id, title := objcx.NewString(a.ID), objcx.NewString(a.Label)
				act := class("UNNotificationAction").Send(sel("actionWithIdentifier:title:options:"), id, title, uint64(0))
				objcx.Release(id)
				objcx.Release(title)
				acts = append(acts, act)
			}
			arr := objcx.NewArray(acts...)
			empty := objcx.NewArray()
			cid := objcx.NewString(cat.ID)
			objs = append(objs, class("UNNotificationCategory").Send(sel("categoryWithIdentifier:actions:intentIdentifiers:options:"),
				cid, arr, empty, uint64(unCategoryCustomDismiss)))
			objcx.Release(cid)
			objcx.Release(arr)
			objcx.Release(empty)
		}
		list := objcx.NewArray(objs...)
		set := class("NSSet").Send(sel("setWithArray:"), list)
		objcx.Release(list)
		cn.Send(sel("setNotificationCategories:"), set)
	})
	return nil
}

// Add shows the request: the center replaces the notification that has its identifier.
//
// Flutter: no counterpart
func (c *Center) Add(ctx context.Context, r Request) error {
	cn := c.handle()
	var req objc.ID
	objcx.WithPool(func() {
		content := class("UNMutableNotificationContent").Send(sel("alloc")).Send(sel("init"))
		defer objcx.Release(content)
		set := func(setter, value string) {
			if value == "" {
				return
			}
			s := objcx.NewString(value)
			content.Send(sel(setter), s)
			objcx.Release(s)
		}
		set("setTitle:", r.Title)
		set("setBody:", r.Body)
		set("setCategoryIdentifier:", r.Category)
		if !r.Quiet {
			content.Send(sel("setSound:"), class("UNNotificationSound").Send(sel("defaultSound")))
		}
		switch {
		case r.Critical:
			content.Send(sel("setInterruptionLevel:"), uint64(unLevelTimeSensitive))
		case r.Quiet:
			content.Send(sel("setInterruptionLevel:"), uint64(unLevelPassive))
		}
		if r.IconPath != "" {
			c.attach(content, r.IconPath)
		}
		id := objcx.NewString(r.Identifier)
		req = objcx.Retain(class("UNNotificationRequest").Send(sel("requestWithIdentifier:content:trigger:"), id, content, objc.ID(0)))
		objcx.Release(id)
	})
	defer objcx.Release(req)
	text, err := await(ctx, func(blk objc.Block) {
		cn.Send(sel("addNotificationRequest:withCompletionHandler:"), req, blk)
	})
	if err != nil {
		return err
	}
	if text != "" {
		return errors.New(tagNotificationsUnavailable, "the notification center refused the notification", "reason", text, "identifier", r.Identifier)
	}
	return nil
}

// attach adds the picture of the notification as an attachment, when the file can be read; a file that cannot
// be is left out, as the notification does not depend on it.
func (c *Center) attach(content objc.ID, path string) {
	if _, err := os.Stat(path); err != nil {
		return
	}
	id, url := objcx.NewString("icon"), objcx.NewFileURL(path)
	defer objcx.Release(id)
	att := class("UNNotificationAttachment").Send(sel("attachmentWithIdentifier:URL:options:error:"), id, url, objc.ID(0), objc.ID(0))
	if att == 0 {
		return
	}
	list := objcx.NewArray(att)
	content.Send(sel("setAttachments:"), list)
	objcx.Release(list)
}

// Remove takes the notification off the screen and out of the center.
//
// Flutter: no counterpart
func (c *Center) Remove(identifier string) {
	cn := c.handle()
	objcx.WithPool(func() {
		id := objcx.NewString(identifier)
		list := objcx.NewArray(id)
		cn.Send(sel("removeDeliveredNotificationsWithIdentifiers:"), list)
		cn.Send(sel("removePendingNotificationRequestsWithIdentifiers:"), list)
		objcx.Release(list)
		objcx.Release(id)
	})
}
