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

