# Manual checks on a phone and on a Windows PC

Checks that only a real phone shows (sections 25.4 to 26.5), checks that only a real desktop shows (27.3, 27.7
and 27.10), and, from section 28.2 on, checks that only a Windows 10 or 11 PC shows (Wine, which runs every
automated Windows test here, is not Windows: SPEC §25.13 lists what it cannot show). Each step's entry in
`docs/IMPLEMENTED.md` says "pending" until the owner has done them and their answers are recorded here. Tool for
the phone: `adb devices` must list the phone (USB debugging on). Run the commands from the module root.

**On a Windows PC** the commands need Go 1.27 or later and a checkout of Glutter (nothing else: no C compiler, no
SDK). `go run ./cmd/glutter-windows run ./examples/gallery` builds the `.exe` for the PC and starts it (it shows the
log when the app ends). To try the files on a PC from the Linux machine instead, build there and copy the
`.exe`: `go run ./cmd/glutter-windows build -arch amd64,arm64 -o gallery.exe ./examples/gallery` writes
`gallery_amd64.exe` and `gallery_arm64.exe`. To see the log while the app runs, start the `.exe` from a terminal
(`.\gallery_amd64.exe`); `GLUTTER_FRAME_STATS=1`, `GLUTTER_RENDERER=d3d12|soft` and `GLUTTER_D3D12_ADAPTER=warp`
are environment variables (`set GLUTTER_FRAME_STATS=1` in `cmd`, `$env:GLUTTER_FRAME_STATS=1` in PowerShell).
Say which Windows version and which GPU the answers come from.

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

## 26.4 Spell check

Run on the phone, with its spell checker on (Settings, System, Languages, Spell checker) and the English dictionary installed:

```
go run ./cmd/glutter-android run ./examples/dev/android-spell
```

Check, and record the answers:

1. Type "helo wrld " with the keyboard (a space after the second word): are the misspelled words underlined in red with a wavy line, once you have left them?
2. Tap a misspelled word: does a toolbar open under the field with up to three suggestions and Delete, like the one of Flutter's apps and of the system's text fields? Tap a suggestion: is the word replaced?
3. Tap Delete on another one: is the word removed?
4. Turn the spell checker off in the settings and come back to the app: are the underlines gone after the next edit?

Answers: _(not yet recorded)_

## 26.5 The examples on Android, size and performance

Run on the phone, with a release build (the numbers of a debug build are 5 to 10 times worse):

```
scripts/perf-android.sh -d <serial> -m -o phone.md       # the numbers; paste them in docs/PERFORMANCE.md
go run ./cmd/glutter-android build -release -key key.pk8 -cert cert.pem -o gallery.apk ./examples/gallery
adb install -r gallery.apk
```

(`-key` and `-cert` are the files that the tool made for debug builds, `$XDG_CONFIG_HOME/glutter/android-debug/key.pk8` and `cert.pem`, or your own.)

Check, and record the answers:

1. Open the gallery: is the list of pages in a drawer (the button at the left of the app bar), and does each page fit the width of the screen, with no overflow stripes and no cut text?
2. Scroll the pages (Lists & cards, Data & progress) and open Data & progress, where the progress indicators animate: does it keep up with the display (no stutter)? Switch the theme with the moon button: does the whole app change at once? Change pages through the drawer a few times: is each change quick?
3. Start the counter and the gallery from the launcher: how long until you see the first frame (about)? Is it acceptable?
4. Run the other examples (`layout`, `animations`, `scrolling`, `navigation`, `editor`, `custompaint`, `todo`): does each start and work with touch?
5. The numbers of the first command: do they meet the targets of SPEC §15 (Android)? If not, which ones are not, and may they stay as exceptions or must they be worked on?

Answers: _(not yet recorded)_

## 27.3 The focus of a window (desktop)

**Status:** pending

Run on your desktop session (Wayland or X11), not headless:

```
go run ./examples/multiwindow
```

Check, and record the answers:

1. Open the second window. Click each window in turn: does the text field of the window you click take the keyboard (type a letter), and does the one you left stop showing a caret?
2. Press `Alt+Tab` (or your desktop's window switch) to move between the two windows and back: does the caret follow the active window each time, with no click?
3. Minimize one window and bring it back from the task bar: is its caret there again, without a click?

Answers: _(not yet recorded)_

## 27.7 The cursor of a pressed pointer (desktop)

**Status:** pending

Run on your desktop session:

```
go run ./examples/dev/sash
```

Check, and record the answers:

1. Move over the thin blue bar between the two panels: do you get the resize arrows? Over the right panel, the hand?
2. Press on the bar and drag quickly to the right, over the right panel, and out of the window: do the resize arrows stay the whole time, and does the left panel follow the mouse?
3. Let go over the right panel: does the hand show at once? Let go outside the window and come back: is the cursor the one of what is under it?
4. Press on the right panel (not on the bar) and drag over the bar: does the cursor stay the hand until you let go (the bar does not take it)?

Answers: _(not yet recorded)_

## 27.10 The cold start with a GPU (desktop)

**Status:** pending

Run on your desktop session, on the machine with the GPU (a headless compositor cannot present GPU buffers, so the numbers of `docs/PERFORMANCE.md` for 27.10 are llvmpipe's):

```
scripts/perf.sh -n 5 -s 5
```

and, for the parts of one start:

```
GLUTTER_FRAME_STATS=1 go run -tags glutter_release ./examples/counter 2>&1 | grep -E "stage=|renderer chosen"
```

Check, and record the answers:

1. The table's two cold start rows (empty caches, both warm) and the table of the parts: how long is the first draw, and which part is the largest (the Vulkan try, the GPU context, the shaders, the frame)? Does `from_cache` show every program on the second start?
2. With `auto`, does the Vulkan try succeed on your desktop (the log says `renderer chosen renderer="vulkan: ..."`), so the 600 ms that it costs on a headless compositor is not paid?
3. `ls ~/.cache/glutter/shaders`: are there files after the first start, and does `GLUTTER_SHADER_CACHE=0` leave the directory alone?

Answers: _(not yet recorded)_

## 28.2 The window and the message loop (Windows)

**Status:** pending

Run on the PC:

```
go run ./cmd/glutter-windows run ./examples/counter
```

Check, and record the answers:

1. Does the window open without a white flash (the first thing you see is the app's own background)?
2. Resize it by a corner and by each edge: does the content follow the pointer, with no stretched or black stripes?
3. Maximize and restore with the title-bar buttons and by double-clicking the title bar: does the content follow, with no frame of the old size? Full screen has no button in any example: if you have an app that calls `SetFullscreen` (`WindowConfig.State`), the task bar must be covered in full screen and come back after it; otherwise skip this.
4. With two monitors of different scales (100% and 150%), drag the window from one to the other: is it sharp at once on each, with no blurry frame?
5. Change the scale in Settings (System, Display) while the app runs: does the app become sharp at the new scale without a restart?
6. Switch Windows between light and dark mode (Settings, Personalization, Colors): does the title bar follow?
7. Press Alt+Tab to another program and back, minimize to the task bar and restore, and close the window: does each work, and does the process end (Task Manager shows no `counter` left)?

Answers: _(not yet recorded)_

## 28.3 Mouse, touch, pen and cursors (Windows)

**Status:** pending

Run on the PC:

```
go run ./cmd/glutter-windows run ./examples/gallery
go run ./cmd/glutter-windows run ./examples/dev/input
```

Check, and record the answers:

1. Press every button of the mouse, the side ones included (`examples/dev/input` shows each event): do the left, middle, right, back and forward buttons each give their own event?
2. Press on a button of the gallery and drag out of the window, then let go outside: does the drag go on outside the window, and does the app see the release?
3. Turn a wheel with notches, and a free-spinning wheel if there is one: does the gallery's list scroll by the amount you expect, and does the free-spinning wheel scroll smoothly?
4. On a precision touchpad: scroll with two fingers in the gallery (does it follow your fingers?) and pinch in the `InteractiveViewer` of "More components" (does it zoom; SPEC §25.6 says a pinch is Control plus wheel)?
5. On a touch screen, if there is one: tap, drag a list, put two fingers on `InteractiveViewer`, press and hold a text field for the selection toolbar. Which work?
6. With a pen, if there is one: draw in `examples/custompaint` or the pen page of `examples/dev/input`: is the pressure used, does the eraser end give `invertedStylus`, does the barrel button give the secondary button?
7. The cursors: over a text field (I-beam), a button (hand), the edge of a resizable panel and the borders of the window (resize arrows): are they the Windows cursors, each at the right place?

Answers: _(not yet recorded)_

## 28.4 Keyboard (Windows)

**Status:** pending

Run on a PC with at least a German or French layout and a Russian or Greek one added (Settings, Time & language, Language & region):

```
go run ./cmd/glutter-windows run ./examples/editor
```

Check, and record the answers:

1. Type letters in the editor on each layout. Does Control+C, Control+V, Control+Z and Control+A work on every layout, the Russian or Greek one included?
2. AltGr characters (on a German layout `AltGr+Q` is `@`, `AltGr+E` is `€`): are they typed, with no stray Control action?
3. Dead keys: `^`, `´` and `` ` `` then a letter (`e`), then the same dead key twice: is the composed letter typed, and the accent itself for a doubled dead key?
4. The numpad with Num Lock on and off: digits, arrows, Home and End?
5. Windows-key combinations (Win+D, Win+E, Win+Space): do they stay the system's, and does the app get no stray key?
6. Alt+F4 closes the window; Alt+Space opens the window menu; F10 does not stick the menu to the app: do they?
7. Hold a letter: does the key repeat at the rate and after the delay that Control Panel (Keyboard) sets?
8. Switch layouts with Win+Space while the app runs: does the next key follow the new layout at once?

Answers: _(not yet recorded)_

## 28.5 Text input and IMEs (Windows)

**Status:** pending

Run on a PC with the Microsoft IMEs added (Settings, Time & language, Language & region, Add a language: Japanese, Chinese (Simplified) Pinyin and Korean):

```
go run ./cmd/glutter-windows run ./examples/editor
```

Check, and record the answers:

1. Japanese: type `nihongo`, convert with Space, choose a candidate, commit with Enter. Is the preedit drawn inline in the editor, underlined, and is the committed text right?
2. Chinese Pinyin: type `zhongwen` and choose a candidate with the number keys. Korean: type `han` `geul`. Do they work the same way?
3. Is the candidate window next to the caret at 100% and at 150% scale, and on the second monitor? Does it stay clear of the line you type in (it should not cover the preedit)?
4. Press Win+. (the emoji panel): does it insert an emoji into the editor?
5. On a touch screen, if there is one: does the touch keyboard type into the editor, and does its handwriting panel? (SPEC §25.7 does not do the handwriting panel; record what happens.)
6. Click out of the field while a preedit is shown: is the composition cancelled, with no stray text?

Answers: _(not yet recorded)_

## 28.6 Clipboard, settings, fonts and lifecycle (Windows)

**Status:** pending

Run on the PC:

```
go run ./cmd/glutter-windows run ./examples/editor
go run ./cmd/glutter-windows run ./examples/gallery
go run ./cmd/glutter-windows run ./examples/dev/text-tour
```

Check, and record the answers:

1. In the editor, copy text and paste it into Notepad, then copy from Notepad and paste into the editor. Copy a file in Explorer: does the editor's field take the path (or nothing, as the SPEC says)? Copy an image in Paint: does a Glutter widget that reads the clipboard see it (use `examples/dev/input` if the editor does not)?
2. Switch Windows between dark and light mode while the gallery runs, and change the accent color: does the gallery change at once?
3. Change the text size (Settings, Accessibility, Text size), turn on a high contrast theme, and turn off animation effects (Accessibility, Visual effects): does the app follow each at once?
4. `examples/dev/text-tour`: are the CJK text and the color emoji shown with real glyphs (Wine shows boxes)? Is Segoe UI the font of the Material text, and are the letters sharp?
5. Does the caret of the editor blink at the rate of Control Panel (Keyboard, Cursor blink rate), and stop when you set it to "none"?
6. Log off, or restart, with the editor open and unsaved text: does the session end without the app hanging it?

Answers: _(not yet recorded)_

## 28.7 Shaders for Direct3D 12 (Windows)

**Status:** pending

Run on the PC (with Go; the test runs on Windows as well as under Wine):

```
go test -tags glutter_wine -v -run TestWineEveryEntryPointCompiles ./internal/gpu/d3d12
```

Check, and record the answers:

1. Do all 16 entry points compile (`--- PASS` for each)?
2. Search the output for `messages:` lines: are there warnings of Microsoft's compiler, in particular `X4014` or `X4121` (a derivative in non-uniform control flow)? Paste any that appear. (Wine's compiler does not enforce these rules, so this is the first time the HLSL meets Microsoft's.)
3. The `d3dcompiler_47.dll` version that the test logs: which one is it?

Answers: _(not yet recorded)_

## 28.8 The Direct3D 12 device: adapter, memory and resources (Windows)

**Status:** pending

Run on the PC:

```
go run ./cmd/glutter-windows run -env GLUTTER_FRAME_STATS=1 ./examples/counter
go test -tags glutter_wine -v -run 'TestWine' ./internal/gpu/d3d12
```

and install the optional Windows feature "Graphics Tools" (Settings, System, Optional features) for the debug layer.

Check, and record the answers:

1. The line `Direct3D 12 device`: which adapter, feature level and `max_feature_level` does it name? Is it the GPU you expect (Wine says a made-up GeForce GTX 470)?
2. With Graphics Tools: `GLUTTER_D3D12_DEBUG=1` on the gallery while you scroll it: does the log report any debug-layer errors? (The step reads the info queue after each submit.)
3. `GLUTTER_D3D12_ADAPTER=warp` (and `GLUTTER_D3D12_ADAPTER=0`, `1` on a PC with two GPUs): does the app run on that adapter, and does the log say so?
4. Device removal: disable the GPU in Device Manager while the app runs (or `dxcap`/the "TDR" test of your choice): does the app recover (a second device) or end with `GPUDeviceLost`, never hang?
5. The Wine tests of the package that ran on the PC: do any fail (the buffer of 0 bytes must be refused here)? Paste the failures.

Answers: _(not yet recorded)_

## 28.9 The Direct3D 12 device: pipelines, passes and parity (Windows)

**Status:** pending

Run on the PC:

```
go test -tags glutter_wine -v -run 'TestWineEveryProofSceneMatchesSoft' ./internal/gpu/d3d12
go run ./cmd/glutter-windows run -env GLUTTER_FRAME_STATS=1 ./examples/gallery
```

Check, and record the answers:

1. Does the parity test (Direct3D 12 against the software renderer over every proof scene) pass on the real driver? Which scenes, if any, are outside the tolerance, and by how much?
2. Run the gallery twice with `GLUTTER_FRAME_STATS=1`: does the second start log `pipeline library present=true loaded_bytes=...` (a real pipeline library, saved after the first frame and loaded at the next start; Wine's vkd3d has none)?
3. The first frame: how long do the `startup` lines say it takes on the first and on the second start (`GLUTTER_SHADER_CACHE=0` for the cold case)? Under vkd3d on lavapipe it is 2 to 12 seconds.
4. Does the gallery look the same as the screenshots in `docs/implemented-proof-images/` (no missing text, wrong colors or banding)?

Answers: _(not yet recorded)_

## 28.10 Presentation and the renderer choice (Windows)

**Status:** pending

Run on the PC:

```
go run ./cmd/glutter-windows run ./examples/gallery
go run ./cmd/glutter-windows run -env GLUTTER_D3D12_ADAPTER=warp ./examples/gallery
go run ./cmd/glutter-windows run -env GLUTTER_RENDERER=soft ./examples/gallery
```

Check, and record the answers:

1. Read the `renderer chosen` line of the first command: which adapter? On a laptop with two GPUs, is it the high-performance one?
2. Resize the window by its edges, and drag it between monitors, with the gallery open: is there tearing, a stretched frame, or a black flash?
3. Do `GLUTTER_D3D12_ADAPTER=warp` and `GLUTTER_RENDERER=soft` both draw the gallery correctly (the second is GDI into the window)? After a `soft` frame, does anything of the Direct3D surface show through (SPEC §22: DWM may keep showing a flip-model surface after GDI draws; say what you see)?
4. With the Graphics Tools feature installed, does `GLUTTER_D3D12_DEBUG=1` report no errors while you scroll the gallery?
5. Switch the user (fast user switching), lock the screen, sleep and wake the PC, and disable then enable the GPU in Device Manager: does the app recover or end with its tag, never hang?
6. Connect with Remote Desktop and start the gallery in the session: does it draw (WARP or `soft`)?

Answers: _(not yet recorded)_

## 28.11 Desktop services: tray, notifications, file dialogs, open URI (Windows)

**Status:** pending

Run on the PC (Windows 10 and, if possible, 11):

```
go run ./cmd/glutter-windows run ./examples/tray
```

Check, and record the answers:

1. Does the tray icon appear (in the overflow area at first: drag it to the bar)? Does a left click show the window, a right click show the menu with its separator, and the tooltip show the app's name?
2. Does a click outside the menu close it? Does a click on "Quit" end the app?
3. Restart Explorer (`taskkill /f /im explorer.exe`, then `explorer`): does the icon come back?
4. "New notification": does it appear as a toast (Windows 10 and 11), and does a click on it show the window? Click it twice quickly: does the second wait for the first? With Focus Assist (Do not disturb) on, does nothing appear and nothing break?
5. If the icon is hidden in the overflow area: is the balloon shown anyway? If not, the icon must be shown while a notification is up (not done yet): say which.
6. "Open file…": open one file, several files, a folder, a save with the overwrite prompt, the filters, and cancel: does each give the right result?
7. Open `https://` and `mailto:` links from the app (`desktop.OpenURI`): do they open in the default programs, and is a `file:` link to an `.exe` refused?

Answers: _(not yet recorded)_

## 28.12 Drag and drop, and several windows (Windows)

**Status:** pending

Run on the PC:

```
go run ./cmd/glutter-windows run ./examples/multiwindow
go run ./cmd/glutter-windows run ./examples/editor
go run ./cmd/glutter-windows run ./examples/tray
```

Check, and record the answers:

1. Drag files from Explorer onto "Drop here" of `multiwindow`, the editor and the tray example; drag text from Notepad and from a web page: are the paths and the text shown?
2. Drag "Drag me" out of `multiwindow` into Notepad (text) and into Explorer: does it drop, and do Shift (move), Control (copy) and Alt (link) change the cursor?
3. Drag between two windows of the app: does it work, and does Escape during a drag cancel it?
4. Open the dialog of `multiwindow`: is the owner disabled, does the dialog open centered over it, and does a click on the owner's task-bar button flash the dialog?
5. Alt+Tab and the task bar with several windows: one button per window, and none for the dialog?
6. Close the first window of `multiwindow` while the second is open: does the app go on, and end with the last window?

Answers: _(not yet recorded)_

## 28.13 Accessibility: UI Automation (Windows)

**Status:** pending

Run on the PC, with Narrator on (Win+Ctrl+Enter) and, for the second pass, NVDA:

```
go run ./cmd/glutter-windows run ./examples/gallery
go run ./cmd/glutter-windows run ./examples/counter
```

Check, and record the answers:

1. In the gallery, move through the elements with Narrator: does each read its name and role? Does Enter or Space activate a button, does a checkbox say its state, does a slider say its value and change with the arrow keys, does a text field read its text and the caret while you type, does a list scroll as you move through it?
2. In the counter, press the button: does Narrator say the new count (a live region)?
3. Repeat with NVDA: any difference?
4. Inspect the tree with Accessibility Insights for Windows (or `inspect.exe`): are the names, control types, patterns and bounding rectangles right? Press the counter's button through `Invoke` from the tool: does the count change?
5. Do the property, structure, text and live-region events reach the tool (the tool's event view)?
6. Turn on a high contrast theme: does the app follow it, and does Narrator keep working?
7. On an arm64 PC, if you have one: does the app run and does Narrator read it (the arm64 thunks are only built and vetted here)?

Answers: _(not yet recorded)_

## 28.14 The `.exe` and `glutter-windows` (Windows)

**Status:** pending

Run on the PC (or build on the Linux machine and copy the file):

```
go run ./cmd/glutter-windows build -o gallery.exe ./examples/gallery
```

Check, and record the answers:

1. Double-click `gallery.exe`: does no console window open?
2. Does the icon show in Explorer and in the task bar? Right-click, Properties, Details: does it show the version (1.0.0.0) and the name?
3. Is the window sharp at 150% scale from the first frame?
4. Does SmartScreen ask once on the first start (the file is unsigned)? Is the second start silent?
5. Start it from a terminal (`.\gallery.exe`): do the log lines appear there? Started from Explorer, do they go to DebugView (Sysinternals)?
6. `go run ./cmd/glutter-windows shot -o gallery.png ./examples/gallery` on the PC: does it save the frame, and does it look right?

Answers: _(not yet recorded)_

## 28.15 The sweep on a PC: every example, performance, and the memory

**Status:** pending

Build the release `.exe` files, as the numbers of `docs/PERFORMANCE.md` do, and run each example:

```
go run ./cmd/glutter-windows build -release -o counter.exe ./examples/counter
go run ./cmd/glutter-windows build -release -o gallery.exe ./examples/gallery
go run ./cmd/glutter-windows run ./examples/layout        # and animations, scrolling, navigation, editor, custompaint, todo, multiwindow, tray
```

Say the Windows version, the CPU and the GPU. The numbers of `scripts/perf-windows.sh` are Wine's (vkd3d on lavapipe) and say nothing about this PC: read this PC's from Task Manager and the log as below.

Check, and record the answers:

1. Every example: does it start, draw, and respond to the mouse and the keyboard? Which, if any, shows something wrong (text, colors, a missing control)?
2. The gallery: scroll "Lists & cards" and "Data & progress" (where the progress indicators animate): does it keep up with the display (no stutter)? Switch the theme with the button at the top right: does the whole app change at once? Change pages through the list a few times: is each change quick?
3. The idle CPU: start `counter.exe`, wait 10 seconds, and read its CPU in Task Manager (Details tab, the CPU column, and the Performance tab if you want the history). Is it 0% (SPEC §15: no frames scheduled, no timers armed)? Do the same for the gallery on its home page.
4. The cold start by eye: double-click `counter.exe` and `gallery.exe` from Explorer. How long until you see the first frame (about)? Is it acceptable? Then run `set GLUTTER_FRAME_STATS=1` and start `counter.exe` from a terminal: what does `first frame presented since_run=` say, with an empty shader cache (delete `%LOCALAPPDATA%\glutter\shaders` first) and with a warm one?
5. The memory of the counter: in Task Manager (Details tab, add the column "Memory (private working set)"), how much does `counter.exe` take after 10 seconds? And the gallery? SPEC §15 proposes less than 60 MB for the counter.
6. The frame statistics of an animation: `set GLUTTER_FRAME_STATS=1`, `set GLUTTER_GALLERY_PAGE=data`, `set GLUTTER_GALLERY_ANIMATE=1`, start `gallery.exe` from a terminal and let it run 20 seconds: paste a few of the `frames` lines (fps, build_avg, raster_avg, over_16ms).
7. The sizes: how big are `counter.exe` and `gallery.exe` (the numbers of the Wine table are in `docs/PERFORMANCE.md`)? Do you accept them?
8. The proposed targets of SPEC §15 ("Windows", marked "proposed, to be confirmed by the owner"): do this PC's numbers meet them? If not, which ones are not, and may they stay as exceptions or must they be worked on?

Answers: _(not yet recorded)_
