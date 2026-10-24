# Manual checks on a phone

Checks that only a real phone shows. Each step's entry in `docs/IMPLEMENTED.md` says "pending" until the
owner has done them and their answers are recorded here. Tool: `adb devices` must list the phone (USB
debugging on). Run the commands from the module root.

## 25.4 GLES on Android

**Status:** pending

Run:

```
go run ./cmd/glutter-android run -env GLUTTER_RENDERER=gles ./examples/gallery
```

Check, and record the answers:

1. Does the gallery appear and follow your finger when you scroll the page and the navigation list?
2. Switch the theme (the moon button at the top right, then Light and Dark): do the colors change at once, with no flicker?
3. Press Home and open the app again: is it drawn again at once, with nothing missing?
4. Record the renderer line of the log (`renderer chosen renderer="gles: ..."`): which GPU does it name?

Answers: _(not yet recorded)_

## 25.5 Vulkan on Android, and choosing the renderer

**Status:** pending

Run:

```
go run ./cmd/glutter-android run ./examples/gallery
```

Check, and record the answers:

1. Read the renderer and the device from the log (`renderer chosen renderer="vulkan: ..."` and the `Vulkan device` line before it, with its `api=`). Which GPU and which Vulkan version does it name? If it says `gles` instead, copy the `renderer not usable` line above it: that is the reason (Android below 10, a driver on the deny list, or a Vulkan error).
2. Turn the phone to landscape and back (turn the automatic rotation on first): is the gallery upright and sharp each time, and does the log say a `Vulkan swapchain` line with `pre_rotation=` 1 or 3 in landscape and 0 in portrait?
3. Scroll the gallery's navigation list and the page: is the scrolling smooth, with no flicker or tearing?
4. Try `go run ./cmd/glutter-android run -env GLUTTER_RENDERER=gles ./examples/gallery`: does it still draw the same?

Answers: _(not yet recorded)_

## 25.6 Touch, mouse, stylus and keys on Android

**Status:** pending

Run:

```
go run ./cmd/glutter-android run ./examples/scrolling
go run ./cmd/glutter-android run ./examples/gallery
go run ./cmd/glutter-android run ./examples/dev/android-input
```

Check, and record the answers:

1. In `examples/scrolling`, fling the list with one finger: does it keep moving after you lift the finger and slow down smoothly, and does a touch on the moving list stop it?
2. In `examples/gallery`, use two fingers on a map or a zoomable widget if the page has one, and drag a slider and a switch: do they follow the finger with no lag?
3. With a stylus, if you have one: does it draw or scroll like a finger, and is its pressure in the log of `examples/dev/android-input`?
4. With a mouse or a trackpad, if you have one (USB or Bluetooth): does the pointer hover, do the buttons and the wheel work, and does the cursor change over a text field?
5. With a Bluetooth keyboard, if you have one: in `examples/dev/android-input`, press K and the arrow keys: does the log say `shortcut ran`? Does Back still leave the app (until step 25.9)?

Answers: _(not yet recorded)_

## 25.7 Insets, system UI, feedback and settings

**Status:** pending

Run (on the phone, ideally one with a display cutout and gesture navigation):

```
go run ./cmd/glutter-android run ./examples/dev/android-system
go run ./cmd/glutter-android run -env GLUTTER_DEMO=immersive ./examples/dev/android-system
go run ./cmd/glutter-android run ./examples/gallery
```

Check, and record the answers:

1. In `examples/gallery` (and in `android-system`): is the app bar below the status bar and the cutout, with nothing hidden under the gesture bar or the navigation buttons? Turn the phone to landscape: is the content clear of the cutout and the bars at the sides?
2. Turn the dark mode of the phone on and off (the quick settings, or Settings → Display): does the gallery's "System" mode follow it at once, without the app restarting, and are the status bar icons light on dark and dark on light?
3. Make the font larger and smaller (Settings → Display → Font size): does the text follow? On Android 14 or later, is very large text scaled less than small text?
4. Long press on a text field or a list item with a long press action (the gallery's "Selection & chips"): do you feel a haptic tick? (Needs "Touch feedback" on in the phone's sound settings.)
5. With `GLUTTER_DEMO=immersive`: are the bars hidden, and does a swipe from the top or bottom edge bring them back for a moment? Does the app keep working?
6. Turn "Remove animations" (Settings → Accessibility) on: does the log show `animationsOff=true` in the `mq` lines (`adb logcat -s dev.glutter.androidsystem`)?
7. Turn the 24-hour clock on and off, and bold text (Settings → Accessibility → Bold text, Android 12 or later): do the `mq` lines show `h24` and `bold` (bold text restarts the activity, not the app)?

Answers: _(not yet recorded)_

## 25.8 Text input, the soft keyboard and the clipboard

**Status:** pending

Run (with the keyboard you use, Gboard or another):

```
go run ./cmd/glutter-android run ./examples/editor
go run ./cmd/glutter-android run ./examples/dev/android-text
go run ./cmd/glutter-android run -env GLUTTER_DEMO=password ./examples/dev/android-text
go run ./cmd/glutter-android run -env GLUTTER_DEMO=multi ./examples/dev/android-text
```

Check, and record the answers:

1. In `examples/editor` and the first field of `android-text`: type with autocorrect on (a misspelled word, then a space): does the correction replace it, and does one Backspace undo it? Type with swipe typing (glide across the letters): does the word appear, with a space?
2. Does the keyboard push the content up (the field stays visible) and does it move smoothly as it opens and closes?
3. With a Japanese or Chinese input method (Gboard's Japanese 12-key or Pinyin; add it in Settings → Languages): is the text being composed underlined, can you pick a candidate, and does it commit into the field?
4. Insert an emoji from the keyboard: does it appear, and does Backspace remove it whole?
5. The password field (`GLUTTER_DEMO=password`): are the characters hidden, and does the keyboard offer no suggestions or a password keyboard?
6. The multiline field (`GLUTTER_DEMO=multi`): does Enter make a new line? In the single-line field, does the Enter key's label say Done and does it hide the keyboard or submit?
7. Select text in the field and copy it (long press, then Copy), open another app and paste: is it the same text? Copy text in another app and paste it in the field with the toolbar's Paste.
8. Move the caret by swiping on the space bar (Gboard): does the caret follow? (Known limit: it may not, SPEC §24.6.)

Answers: _(not yet recorded)_

## 25.9 Lifecycle, back navigation and exit

**Status:** pending

Run (a phone with Android 14 or later, gesture navigation, and "Predictive back animations" on in Developer options if the phone has that switch):

```
go run ./cmd/glutter-android run ./examples/navigation
go run ./cmd/glutter-android run ./examples/dev/android-back
```

Check, and record the answers:

1. In `examples/navigation`, push a page and swipe back from the left or right edge: does the page follow your finger and then pop, and does a swipe you let go of early cancel and keep the page? Does the system's predictive animation (the page shrinks and the one below shows) play, or does the app show its own? (SPEC §24.8: it depends on the theme's page transitions.)
2. The same with the 3-button navigation (Settings → System → Navigation): does the Back button pop the page?
3. On the first page, does Back (swipe or button) leave the app and show the home screen, and does opening the app again start it where it was (still running) or from the beginning (killed)?
4. Press Home and open the app again from the launcher and from the recent-apps screen: does it come back as it was, with no flicker or black frame? Is the screen drawn again at once after the lock screen?
5. `examples/dev/android-back`: push the third page (it cannot be popped): does Back leave you on it? Look at `adb logcat` (filter on "lifecycle") while you do it: are the `lifecycle` lines in order (Inactive, Hidden, Paused on Home; Resumed on return)?
6. Swipe the app away from the recent-apps screen and start it again: does it start from the beginning?

Answers: _(not yet recorded)_

## 25.10 Fonts on Android

**Status:** pending

Run on the phone, and again with the phone's language set to Japanese, Chinese and Korean (Settings → System → Languages), and on a phone with Android 12 or older if you have one:

```
go run ./cmd/glutter-android run ./examples/dev/android-fonts
go run ./cmd/glutter-android run -env GLUTTER_DEMO=ja-JP ./examples/dev/android-fonts
```

Check, and record the answers:

1. Does every line show real letters (Latin, Greek, Cyrillic, Arabic, Hebrew, Devanagari, Thai, Chinese, Japanese, Korean) and no empty boxes or question marks?
2. Are the emoji in color, and do they look like the phone's own emoji (the 5 emoji of the last line: grinning face, party popper, thumbs up, heart, rocket)?
3. Are the Chinese and Japanese lines drawn with different forms of 直骨 (each in the form of its own language)?
4. Does the Arabic line read right to left, joined, and the Hebrew line right to left?
5. In `examples/gallery`, does the text look right (Roboto, not a fallback font, for Latin text)?

Answers: _(not yet recorded)_

## 25.11 TalkBack

**Status:** pending

Turn TalkBack on (Settings → Accessibility → TalkBack), then run:

```
go run ./cmd/glutter-android run ./examples/gallery
go run ./cmd/glutter-android run ./examples/dev/android-a11y
```

Check, and record the answers:

1. Explore by touch (drag one finger over the screen): does TalkBack read the element under your finger (buttons say "button", text fields "edit box", switches and checkboxes their state)?
2. Swipe right and left with one finger: does the focus go to the next and the previous element, in a sensible order, and scroll into view when the next element is off screen?
3. In `android-a11y`, double-tap the "Increment" button: does the count change, and does TalkBack read the new "Count: N" (a live region)?
4. Scroll the list with two fingers, and with the TalkBack gesture (swipe up then down with one finger, or the local context menu's "Scroll forward"): does the list scroll and does TalkBack say what is shown?
5. Double-tap the "Name" field and type with the keyboard: does TalkBack read the characters you type, and can you move the cursor with the volume keys or the text editing controls?
6. In `examples/gallery`, open a dialog or a menu: does TalkBack announce it, and is the page below not reachable while it is open?
7. Turn TalkBack off: does the app go back to normal at once, and does it stay fast (no slower than before)?

Answers: _(not yet recorded)_

## 25.12 Services on Android

**Status:** pending

Run on the phone (with Android 13 or later, a fresh install so that the permission has not been decided):

```
go run ./cmd/glutter-android run ./examples/dev/android-services
```

Check, and record the answers:

1. Tap "Open link": does the browser open example.com? Tap "Open mail": does the mail app open with a new message (or does Android ask which app)?
2. Tap "Open bad": does the app stay as it is, with no crash (the log says `OpenURIUnavailable`)?
3. Tap "Notify": does the system ask for the permission to send notifications, once? After you allow it, does the notification appear in the shade with the title "Glutter", the text and a "Reply" button?
4. Pull the shade down and tap "Reply": does the app come to the front, and does the log say `notification action id=reply`? Tap the notification's body (when it has the default action) and swipe a notification away: does the log say `notification closed`?
5. Tap "Notify" again after you denied the permission (or turn the app's notifications off in Settings): does nothing appear and the log say `NotificationsUnavailable`, with no second permission prompt in the same run?
6. Tap "Close" after "Notify": does the notification disappear?

Answers: _(not yet recorded)_

## 26.2 The Material text selection toolbar and the magnifier

**Status:** pending (not required by the plan: the magnifier on the GPU renderers is only tested on desktop GPUs and, with the software renderer, on the emulator)

Run on the phone (the default renderer is Vulkan or OpenGL ES):

```
go run ./cmd/glutter-android run -env GLUTTER_DEMO=single ./examples/dev/android-text
```

Check, and record the answers:

1. Type a few words in the first field and long-press a word: does it get selected, with two handles and the toolbar above it (Cut, Copy, Paste, Select all)? Does a long toolbar show the overflow button, and does it open the second page with the back arrow?
2. Drag a handle: does a magnifier appear above your finger, showing the text under it enlarged, following the finger, and disappear when you let go?
3. Drag the cursor handle of an empty selection (tap once, then drag the drop-shaped handle): does the magnifier show too?
4. Does the toolbar look like the phone's own (white rounded bar, dark text), and the dark theme's one dark?

Answers: _(not yet recorded)_

## 26.3 Predictive back transitions

Run on the phone (Android 14 or later, with "Predictive back animations" on in the developer options for the system's own animation; the app's transition does not need it):

```
go run ./cmd/glutter-android run ./examples/dev/android-back
```

Check, and record the answers:

1. Tap "Push the second page", then swipe in from the left edge slowly and hold halfway: does the second page shrink, get rounded corners and follow your finger, with the home page showing around it?
2. Let go before the middle of the screen: does the page go back to full size (cancel)? Swipe again and let go past the middle: does the page leave and the home page stay?
3. Press the back button (or the three-button navigation's back) instead of the gesture: does the page leave with a fade and a small slide, not the shrinking?
4. Compare with a system app or a Flutter app on the same phone: is the motion of the same kind?

Answers: _(not yet recorded)_

