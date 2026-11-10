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

## 30.7 Pointer lock

**Status:** pending

Run `examples/gpu-triangle` in a debug build (on Linux `go run ./examples/gpu-triangle`; on Windows `go run ./cmd/glutter-windows run ./examples/gpu-triangle`; on the web `go run ./cmd/glutter-web serve ./examples/gpu-triangle`; on a phone `go run ./cmd/glutter-android run ./examples/gpu-triangle`). Press **L** (on the web, click the page first, since the browser wants a click or a key) and move the mouse.

Check, and record the answers for each of: GNOME (Wayland), KDE (Wayland), an X11 session, a Windows PC, Chrome, Firefox, and an Android phone with a mouse:

1. Does the cursor disappear and the cube turn with the mouse, with no limit at the edge of the screen?
2. Does the motion feel the same as without the lock (unaccelerated, no jump when the lock starts)?
3. Does a click while locked still reach the app (the slider and the switch do not move, since the pointer is held where it was)?
4. Does L give the pointer back, at the place it was?
5. Does Alt+Tab (and, on the web, Escape) end the lock, and does the pointer come back at once? Does the lock need L again afterwards?

Answers: _(not yet recorded)_

## 31.6 Audio on Windows, Android and the web

**Status:** pending

`examples/dev/android-audio` plays a 440 Hz tone through the public `audio` package (and, on Android, through a device of the platform too, which it logs); `GLUTTER_AUDIO_CAPTURE=1` also opens a recording. Run it on each of the systems below and record the answers.

**On a Windows PC**: `go run ./cmd/glutter-windows run ./examples/dev/android-audio`.

1. Does the tone play, with no click, gap or crackle for a minute?
2. Plug in headphones (or a USB headset) while it plays and choose them as the default device in the sound settings: does the tone move to them within a second or two, without the app restarting? Back again?
3. Unplug the headphones: does the tone go on from the speakers?
4. With `set GLUTTER_AUDIO_CAPTURE=1` in `cmd` first: does the app record (the log shows `audio capture frames=` growing), and does Windows' microphone privacy setting, when it is off, give `audio capture failed err=AudioCapturePermissionDenied` and not a hang?

**On a phone** (`go run ./cmd/glutter-android run -permission android.permission.RECORD_AUDIO ./examples/dev/android-audio`, with `-env GLUTTER_AUDIO_CAPTURE=1` for the recording):

5. Does the tone play with no gap for a minute? What is the latency with a loopback (a recording of the speaker next to the screen's tap)? The target is 30 ms; the log's `buffer` and `burst` are the stream's own numbers.
6. Press Home: does the tone stop at once, and go on when the app is shown again? Does it stop when a call or an alarm takes the audio focus, and go on after?
7. With the recording: does the first start show the system's question? Does "Don't allow" give `audio capture failed err=AudioCapturePermissionDenied` and the tone play on?
8. Plug in headphones (wired or Bluetooth) while it plays: does the tone move to them?

**On the web** (`go run ./cmd/glutter-web serve ./examples/dev/android-audio`, then the address it prints), in Chrome, in Firefox and in Safari Technology Preview:

9. Does the page stay silent until the first click or key, and then play the tone with no gap? (The moment that the sound starts is what a headless test cannot show: it cannot make the gesture that the browser trusts.)
10. Does it play in a tab that the browser puts in the background, as far as that browser allows? Does it come back at once when the tab is shown?
11. With the recording (`?GLUTTER_AUDIO_CAPTURE=1` on the address): does the browser ask for the microphone, and does "Block" give `AudioCapturePermissionDenied`?
12. In the browser's console: is `crossOriginIsolated` true on the served page (and does the sound play the same on a server that sends no such headers)?

Answers: _(not yet recorded)_

## 31.7 Motion sensors and haptics

**Status:** pending

`examples/dev/android-sensors` shows the latest readings of the accelerometer and the gyroscope, logs them twice a second, and has an arrow that points where gravity pulls on the screen. Run it on each of the systems below and record the answers.

**On a phone** (`go run ./cmd/glutter-android run ./examples/dev/android-sensors`):

1. Lay the phone flat on its back: does the accelerometer read about 9.8 on z and about 0 on x and y? Hold it upright: 9.8 on y? On its left side: x about -9.8?
2. Tilt the phone: does the arrow turn with your hand and stay pointing at the floor? Is it smooth (a reading about every frame), with no jumps?
3. Turn the phone about its long axis (a quarter turn in a second): does the gyroscope show about 1.5 rad/s on y while you turn, with the right sign (counterclockwise seen from the top of the phone is positive)?
4. Press Home and come back: do the readings go on, and does the log's `dropped` stay 0 for a few seconds in the app?
5. The log's `readings` and `frames`: at the default 60 Hz is `readings` about the same as `frames` (one reading to each call)?

**On the web** (`go run ./cmd/glutter-web serve ./examples/dev/android-sensors`, the address it prints opened on a phone: it needs https or `localhost`, so use `adb reverse tcp:PORT tcp:PORT` and open `http://localhost:PORT` in the phone's Chrome, or serve over https):

6. Chrome on Android: do the same numbers as in 1 to 3 appear, with the same signs?
7. Safari on an iPhone (over https): does the first tap show the system's question, and do the numbers move after "Allow"? Is the sign of the accelerometer the same as on Android (gravity on +z when the phone lies on its back)? Safari has reported the opposite sign in older versions; if it does, say so, the toolkit does not correct it yet.
8. Firefox on Android: do the numbers appear without a question?
9. The vibration: tap the page of the example (a heavy impact) and press it long (a light one), in the phone's Chrome and in the app on Android: does the phone vibrate for a moment, and less for the long press?

**A gamepad** (DualSense, DualShock 4 or Switch Pro over Bluetooth on Linux with the hidraw access of the guide):

10. With a small program that calls `pad.OpenSensor(sensors.Gyroscope)` and prints the readings: do the numbers follow the pad when you turn it, at about 250 a second for a DualSense? Does the pad stop reporting (the battery) when the sensor is closed?

Answers: _(not yet recorded)_

## 31.11 `examples/game3d`, performance and memory

**Status:** pending

Coin Run (`examples/game3d`) is played with a pad, the keyboard, a locked mouse or the touch. Run it on each of the systems below and record the answers.

**On Linux** (`go run ./examples/game3d`, on the AMD GPU, not the eGPU), **on a Windows PC** (`go run ./cmd/glutter-windows run ./examples/game3d`) **and on a phone** (`go run ./cmd/glutter-android run ./examples/game3d`, with a pad over Bluetooth or USB, and with the touch only):

1. Does the game open in a few seconds, with the character, the level, the coins and the shadow? Is the frame rate 60 or the display's (the target of SPEC §28.6 is 60 at 1920x1080 on the owner's machine)? Does it stay smooth while you run, turn the camera and jump, with no hitch when a coin is picked up or a sound starts?
2. With each pad you have (Xbox, DualSense, Switch Pro, a generic one): does the left stick run, the right stick turn the camera, A jump, Start open the pause menu? In the menu, do the D-pad and A choose and press the buttons, and B (or Start) close it?
3. Rumble: touch the lava (the orange tiles): does the pad rumble for about a third of a second, on every pad that has motors, and the phone vibrate?
4. The keyboard: W A S D (or the arrows for the camera), Space, Esc. The mouse: does a click in the window lock it (the cursor goes), does moving it turn the camera smoothly, and does Esc (or Alt+Tab) give it back and open the menu?
5. The sound: is the music steady, with no gaps or clicks, for a few minutes? Do the coin, the jump and the hit sound at once when they happen (the latency you can hear)? Is the music quieter in the pause menu, and does the switch of the menu turn it off? With a loopback cable or a phone recording the screen and the speaker: is the delay from a button to its sound under 30 ms on Linux and Windows, 20 ms on the phone?
6. The touch (phone, no pad): does a drag on the left half run, one on the right half turn the camera, and does the round button jump?
7. Collect all twelve coins: does "You win!" show, and "Play again" start over? Lose all three hearts: does "Game over" show?

**In Chrome and Firefox** (`go run ./cmd/glutter-web serve ./examples/game3d`): the same 1 to 5. Does the page play sound after the first click (the browsers' rule), is the delay under 120 ms, and does a click lock the mouse (Esc gives it back, and the game pauses)? Does a pad that you press a button of show up (the browsers report a pad only after a press)?

Answers: _(not yet recorded)_

## 32.1 `soft` on Wayland: `wl_shm`

**Status:** pending

`GLUTTER_RENDERER=soft` draws the app on the CPU into `wl_shm` buffers and redraws only what changed. It is chosen by name only until step 32.3, so run each command with the variable. On a **real Wayland session in a virtual machine with no GPU** (GNOME or KDE under GNOME Boxes, virt-manager or VirtualBox: `glxinfo -B` says `llvmpipe`), from a checkout of Glutter:

1. `GLUTTER_RENDERER=soft go run ./examples/gallery`: does the window open with the same picture as without the variable, and does the log say `renderer chosen renderer="soft: shared memory (wl_shm)"`? Is it as smooth as the same app with `GLUTTER_RENDERER=gles` or smoother (move the mouse over the buttons, scroll the list, switch the pages)?
2. `GLUTTER_RENDERER=soft go run ./examples/editor`: click in the text and type. Does the caret blink with no flicker or tear, and does the CPU use (`top`) stay low while it blinks and nothing else happens (compare with `GLUTTER_RENDERER=gles`)?
3. Resize the window by its edge, maximize it and restore it: is every size drawn at once, with no strip of the old picture or of black? Move it to a monitor with another scale, or change the scale in the settings: is the picture sharp at the new scale (a fractional scale included)?
4. `GLUTTER_RENDERER=soft go run ./examples/multiwindow` (open and close windows) and the tray example (hide the window and show it again): does every window draw, and does a window that is shown again show the whole picture, not the one from before it was hidden?
5. Does the app, left running for a few minutes, keep its memory flat (`top`, the `RES` column, or `GLUTTER_FRAME_STATS=1`)?

Answers: _(not yet recorded)_

## 32.2 `soft` on X11: MIT-SHM and `XPutImage`

**Status:** pending

`GLUTTER_RENDERER=soft` on X11 draws the app on the CPU and sends only the damaged rectangles to the X server: with MIT-SHM (shared memory) when the server is on the same machine, with `XPutImage` otherwise. It is chosen by name only until step 32.3, so run each command with the variable and with `GLUTTER_BACKEND=x11`. On a **real X11 session in a virtual machine with no GPU** (an X11 session of GNOME, KDE or Xfce: `glxinfo -B` says `llvmpipe`), from a checkout of Glutter:

1. `GLUTTER_BACKEND=x11 GLUTTER_RENDERER=soft go run ./examples/gallery`: does the window open with the same picture as without `GLUTTER_RENDERER`? Does the log say `X11 pixels go to the server through shared memory path=MIT-SHM` and `renderer chosen renderer="soft: MIT-SHM (X11)"`, and is there no `libEGL warning: DRI3 error` line? Is it as smooth as `GLUTTER_RENDERER=gles` or smoother (move the mouse over the buttons, scroll the list, switch the pages)?
2. `GLUTTER_BACKEND=x11 GLUTTER_RENDERER=soft go run ./examples/editor`: click in the text and type. Does the caret blink with no flicker, and does the CPU use (`top`) stay low while it blinks and nothing else happens (compare with `GLUTTER_RENDERER=gles`)?
3. Resize the window by its edge, maximize it and restore it, then change the scale in the settings: is every size drawn at once, with no strip of the old picture or of black, and is the picture sharp at the new scale? Cover the window with another one and uncover it: is the whole picture back at once?
4. `GLUTTER_X11_NOSHM=1` with the same commands: the log says `X11 pixels go to the server in requests path=XPutImage`, and the app looks the same. Is it noticeably slower than with MIT-SHM (it should be a little, on a large window)?
5. **A remote X display** (`ssh -X other-machine`, then `GLUTTER_BACKEND=x11 GLUTTER_RENDERER=soft go run ./examples/gallery` there, or a program that runs on another machine with `DISPLAY` set to your screen): does the log say `path=XPutImage reason="the display is not on this machine"`, and does the window draw correctly, with a caret that blinks and does not make the connection busy?
6. `GLUTTER_BACKEND=x11 GLUTTER_RENDERER=soft go run ./examples/multiwindow` (open and close windows) and the tray example (hide the window and show it again): does every window draw, and does a window that is shown again show the whole picture? After the app ends, `ipcs -m` shows no segment of it.
7. Does the app, left running for a few minutes, keep its memory flat (`top`, the `RES` column, or `GLUTTER_FRAME_STATS=1`)?

Answers: _(not yet recorded)_

## 32.3 `auto` chooses `soft` when there is no GPU

**Status:** pending

On a machine with no GPU `GLUTTER_RENDERER` unset (or `auto`) now draws with the software renderer, which redraws only what changed. On a **virtual machine with no GPU** (the one of the Emetos report: `glxinfo -B` says `llvmpipe`), Wayland and X11 both, from a checkout of Glutter:

1. `go run ./examples/gallery` with no environment variable: does the log say `renderer not usable, trying the next renderer=gles ... renderer=llvmpipe ... tag=SoftwareRenderer` and then `renderer chosen renderer="soft: ..."`? Does the window look the same as with `GLUTTER_RENDERER=gles`, and is it noticeably quicker (move the mouse over the buttons, scroll the list, resize the window)?
2. `go run ./examples/editor`, click in the text and leave the caret blinking: how much CPU does `top` show for the process, with the default and with `GLUTTER_RENDERER=gles`? (`docs/PERFORMANCE.md` has the numbers of the headless run: the VM's should be several times lower with the default.)
3. **The Emetos app**, before (Glutter 1.11, or `GLUTTER_RENDERER=gles`) and after (1.12, default): is the app fast again? Is there any difference in the picture (text, shadows, clipping)? Is the first window slower to appear than before?
4. On a machine **with a GPU**: the default still chooses Vulkan (`renderer chosen renderer="vulkan: ..."`), and `GLUTTER_RENDERER=gles` still chooses OpenGL ES; nothing mentions `SoftwareRenderer`.
5. `GLUTTER_RENDERER=gles` on the VM: it still runs on llvmpipe (slow, as before), because a name is never second-guessed.

Answers: _(not yet recorded)_

## 33.2 The app, the run loop and windows with `soft` (macOS)

**Status:** pending

On a Mac, `go run ./examples/counter` opens an `NSWindow` drawn by the software renderer into IOSurfaces. The display tests run an accessory app that never takes the focus and ticks its frames with `CVDisplayLink`, so they cannot show what the screen shows: the first show, the traffic lights, `CADisplayLink` on a screen that is awake, full screen and a second display.

1. `go run ./examples/counter` from Terminal: does the window open in front of the other windows, with no white flash before the counter appears?
2. Resize it by a corner, slowly and quickly: does the content follow the pointer with no stretched or stale strip at the edge, and is the text sharp on a Retina display?
3. Zoom it (green button with Option held), then full screen (green button alone) and back; minimize it to the Dock and bring it back: does it draw at once each time, with the right size?
4. If there is a second display with another scale (a Retina laptop screen and an external one), drag the window from one to the other: does it redraw sharp at the new scale as it crosses, and does the counter still react to a click (clicks come with step 33.3; until then only the drawing counts)?
5. Press `Cmd+Q`: does the app end? Is the app's name in the menu bar the name of the executable (`counter`)? Does `Cmd+H` hide it?
6. Leave the counter alone for a minute: does Activity Monitor show close to 0 % CPU and few "Idle wake ups" for it?

Answers: _(not yet recorded)_

## 33.3 Mouse, trackpad, tablets and cursors (macOS)

**Status:** pending

The pointer input of the view is Flutter's: the mouse with its buttons and its enter and exit, a notched wheel as scroll signals, the trackpad as the pan and zoom gesture with the momentum left to the framework, pinch and rotate, and the cursors of `SystemMouseCursors`. The tests send real `NSEvent`s made by the platform; what only hardware shows is below.

1. **A mouse with a wheel and side buttons**: in `go run ./examples/gallery`, does the wheel scroll the lists by about the same as in other apps? With Shift held, does it scroll sideways? Do the back and forward buttons arrive (try a widget that shows pointer buttons)? Does a right click open the context menu of a text field, and a Control-click too?
2. **The trackpad**: two-finger scroll in `go run ./examples/scrolling` (the first tab): does the list follow the fingers, continue after you lift them, and stop when you put a finger on the trackpad? Is the fling about as long as in a native app?
3. **Pinch and rotate** on the trackpad in the gallery's `InteractiveViewer`: does the picture zoom and turn under the fingers, and does a pinch together with a scroll (two fingers moving apart and sideways) act as one gesture?
4. **A force click** on the trackpad: nothing special happens, and the click is an ordinary click.
5. **A tablet**, if there is one (a Wacom or an iPad with Sidecar and an Apple Pencil): does a stroke in a drawing widget get the pressure and the lean, and does the eraser end draw as an eraser (`PointerDeviceKind.invertedStylus`)?
6. **The cursors** in the gallery: the I-beam over text, the arrow over buttons (as in Flutter on a desktop), the pointing hand over links that ask for it, the resize arrows at the edges of a split view; does the cursor go back to the arrow when you leave the window and come in again?

Answers: _(not yet recorded)_

## 33.4 Keyboard (macOS)

**Status:** pending

Keys reach the framework as Flutter's macOS embedder reports them: the physical key from the key code, the logical key from the character of the layout (or the key of the US layout where the layout has no Latin one), the left and right modifiers, CapsLock as a tap, and the Command shortcuts and the macOS text editing bindings. The tests press the keys with real `NSEvent`s made by the scripts (`key:Name,mods` of `GLUTTER_DEV_EVENTS`); what only a real keyboard shows is here. Text itself (typing, accents, input methods) comes with step 33.5, so the answers below are about keys that move the caret, select and delete. In `go run ./examples/editor` (type with the text input of step 33.5 once it is there; until then press the keys on the text that you load with `GLUTTER_DEV_TEXT="Hello world"` in a debug build), with each of these layouts (System Settings, Keyboard, Text Input, Input Sources): **US**, **German**, **French (AZERTY)** and **Russian**:

1. **Shortcuts on every layout**: `Cmd+A` selects everything, `Cmd+Left` and `Cmd+Right` go to the start and the end of the line, `Option+Left` and `Option+Right` by a word, `Cmd+Up` and `Cmd+Down` to the start and the end of the text. On the French layout the key with `Q` on its cap is the one that `Cmd+A` uses on a US keyboard: does `Cmd+A` work on the key labeled `A` (the logical key follows the character) and, on the Russian layout, on the key labeled `Ф` (a key with no Latin character takes the logical key of the US layout)?
2. **Shift**: `Shift+Left` and `Shift+Right` extend the selection by a character, `Option+Shift+Left` by a word, `Cmd+Shift+Left` to the start of the line. Does a second `Shift+Right` after `Shift+Left` shrink it, as in a native text field?
3. **Deleting**: `Backspace` and `Fn+Backspace` (forward delete), `Option+Backspace` (a word), `Cmd+Backspace` (to the start of the line), and the same with Shift.
4. **Emacs keys** (`Ctrl+A`, `Ctrl+E`, `Ctrl+F`, `Ctrl+B`, `Ctrl+N`, `Ctrl+P`, `Ctrl+T`): do they act as in a TextEdit window (start and end of the line, a character, a line, transpose)?
5. **Undo**: `Cmd+Z` undoes and `Cmd+Shift+Z` redoes (once there is text to undo, step 33.5).
6. **The Fn and Globe key**: the Globe key opens the emoji viewer or switches the layout as the system settings say, and the framework sees no stuck key afterwards (press a letter: it types normally).
7. **Modifiers pressed in another app**: hold `Shift` in another app, click into the window of the app and release `Shift` outside it: does the app act as if `Shift` were down? (It must not: the window is told at the first event that the key is up.) Do the same with `Cmd` held while switching with `Cmd+Tab`.
8. **A held key**: hold a letter, then `Cmd+Tab` away and back: the letter does not repeat in the window without you holding it.
9. **Changing the layout while the app runs** (`Ctrl+Space` or the menu bar flag): press `Cmd+A` again on the new layout. Does it still select everything?
10. **Keys that the framework does not take reach the menus**: in an app that has no widget for the key, `Cmd+H` hides the app, `Cmd+Q` quits it (the window asks to close first), and a letter makes the system's beep (as in a native app that does not take the key) rather than nothing, a crash or a stuck key. `Cmd+Tab`, `Cmd+Space` and `Cmd+Q` while a text field has the focus still work. (A test cannot make this: a synthesized key does not go through the key window of an app that is not active.)
11. **CapsLock**: press it twice: the app shows no stuck CapsLock key (a widget that shows `HardwareKeyboard.instance.lockModesEnabled` shows the light's state).

Answers: _(not yet recorded)_

## 33.5 Text input, IMEs and the pasteboard (macOS)

**Status:** pending

A text field takes its text from the input method of macOS through `NSTextInputClient`: the tests call its methods in the order that an input method does (`text`, `compose` and `composing` of `GLUTTER_DEV_EVENTS`), and the pasteboard is tested on a pasteboard of its own. What only the real input methods and the real clipboard show is here. In `go run ./examples/editor`, with each of these input sources (System Settings, Keyboard, Text Input, Input Sources):

1. **Pinyin** (Simplified): type `nihao`: the letters are underlined as you type, the candidate window opens just under the caret (not at a corner of the screen, and it follows the caret when the text wraps or scrolls), `Space` or a digit picks a candidate and the underline goes, `Backspace` shortens the composition, `Escape` cancels it, and `Return` commits the letters as typed.
2. **Japanese** (Kotoeri or Hiragana): type `nihongo`, `Space` converts and moves through the candidates with the arrows, `Return` commits. In the middle of a line, the composition is where the caret is and the text after it moves. Does the candidate window follow the caret when you scroll the editor?
3. **Korean** (2-Set): type `hangul`: the syllable changes as you type and the next syllable starts without losing the first.
4. **The accent menu**: hold `e` (with `ApplePressAndHoldEnabled` on, the default): the menu shows over the caret; picking `é` with its number or the mouse replaces the `e` that was typed. With the setting off (`defaults write -g ApplePressAndHoldEnabled -bool false`, then restart the app) the key repeats instead.
5. **Dead keys**: on the US layout `Option+E` then `e` gives `é`; on a German layout `´` then `a` gives `á`, and `´` then `Space` gives `´`.
6. **Emoji and symbols**: `Ctrl+Cmd+Space` opens the viewer, and picking an emoji types it into the field at the caret.
7. **Dictation** (the shortcut of System Settings, Keyboard, Dictation): speak a few words: they appear in the field, underlined while they are recognized.
8. **Reconversion**: select a word that you typed in Japanese and press the conversion key: the input method reads the text around the caret and offers other candidates.
9. **The window**: click into another field of the app, or into another window, while a composition is going on: the composition ends (committed or dropped, as in TextEdit) and the next key types normally.
10. **The pasteboard**: copy text in the editor (`Cmd+C`) and paste it into TextEdit; copy text in TextEdit and paste it into the editor (`Cmd+V`); copy an image in Preview and paste into an app that reads images; copy files in the Finder and paste them in an app that reads `ClipboardFormatURIList`. Does `Cmd+X` cut? Is a long text (a megabyte) pasted without a pause?

Answers: _(not yet recorded)_

## 33.6 Settings, fonts, lifecycle and quit (macOS)

**Status:** pending

The tests read the real settings and fonts and drive the lifecycle and the quit by calling the platform's own methods (an app that is not active never gets the real events, and a test must not hide the user's apps). What only a real session shows is here. In `go run ./examples/gallery` with `GLUTTER_DEV_THEME=system` (a debug build; the gallery's own Light, System and Dark toggle on its home page does the same), and in `go run ./examples/editor`:

1. **Dark mode**: with the system appearance Auto, switch System Settings, Appearance between Light and Dark (and the Auto time of day if you want to wait for it): the gallery changes its theme within a moment, with no restart, and the app's title bar follows.
2. **Accent color**: change the accent color in System Settings, Appearance: a component that uses the system accent follows (the gallery does not use it; the check is that no restart is needed and nothing breaks).
3. **Accessibility**: turn *Reduce motion* and *Increase contrast* on and off (System Settings, Accessibility, Display): the gallery's animations stop and start (a progress indicator, the page transitions) without a restart.
4. **Language and region**: add a language, move it first, and turn the *24-hour time* on and off (System Settings, General, Language & Region): the time picker of the gallery's Dialogs page shows the 24-hour dial or the AM/PM one the next time it opens.
5. **Fonts**: the text of the gallery is San Francisco (the `a`, `g` and `1` match the Finder's), bold text is bold, and the text field of the Text fields page takes Chinese, Japanese and Korean text with the right glyphs, and an emoji in color (try 😀 👍🏽 🇬🇷 and a family emoji). Does text in Zapfino, Menlo and Times (the editor has no picker: use `GLUTTER_DEV_TEXT` with the family in a small example) look like the system's own?
6. **Hide**: press `Cmd+H` and bring the app back from the Dock: animations pause while it is hidden (watch the CPU in Activity Monitor: an idle hidden app uses none) and resume.
7. **Minimize**: minimize the window to the Dock and bring it back: the same.
8. **Another app in front**: click another app: the caret of the focused field stops blinking and the app goes inactive (the gallery's focus rings go); coming back resumes.
9. **Quit**: `Cmd+Q` quits an app that has one window. In `go run ./examples/multiwindow`, `Cmd+Q` closes the windows one after the other. With an app that cancels the exit request (`AppLifecycleListener.OnExitRequested` returning cancel) the app stays and the windows that were asked before the refusing one are closed. The Dock's right-click Quit and a logout do the same.

Answers: _(not yet recorded)_

## 33.7 Menus (macOS)

**Status:** pending

The tests build the menus and choose their items by calling AppKit's own methods (`performActionForItemAtIndex:`, the delegate's `menuWillOpen:`); they cannot show the menu bar or press keys in it. In `go run ./examples/gallery`, page Menus (the section PlatformMenuBar; the app is the one in front), and in `go run ./examples/editor` (default menu bar):

1. **The menu bar**: the menus are `<app>`, File and View, and the app menu has About, Services, Hide, Hide Others, Show All and Quit with the shortcuts that other apps have (`Cmd+H`, `Opt+Cmd+H`, `Cmd+Q`).
2. **Choosing**: File, New (or `Cmd+N`) writes *New* in the line "Last chosen"; the same for Open, Save and Save As (`Shift+Cmd+S`) and View, Reset zoom (`Cmd+0`). "Disabled item" is grey and does nothing. After you open File, is "opened" counted (the demo does not show it; only that nothing breaks)?
3. **The keyboard**: `Ctrl+F2` focuses the menu bar, the arrows move through the menus and `Return` chooses; the first letters of an item, typed in an open menu, jump to it.
4. **Help search**: Help menu, type "Save": the search finds File, Save and highlights it (the Help menu is AppKit's).
5. **Dividers**: Save and Save As are between two dividers in File; no menu starts or ends with a divider.
6. **Services and the standard items**: Services has the services of other apps; Hide and Hide Others hide windows; About shows the standard panel with the app's name; Quit asks the exit request and quits.
7. **The default menu bar** (the editor has no PlatformMenuBar): Edit, Cut, Copy and Paste work in the editor's text field with the mouse as with `Cmd+X`, `Cmd+C` and `Cmd+V`; Undo and Redo too; Select All selects. Window, Minimize and Zoom act on the window and the Window menu lists the windows (open `go run ./examples/multiwindow`); Enter Full Screen works.
8. **Leaving the bar**: go to a page without the PlatformMenuBar section's page (leave the Menus page): the menu bar is the default one again (Edit, View, Window, Help).

Answers: _(not yet recorded)_


## 33.8 The Metal device (macOS)

**Status:** pending

The tests draw through Metal with the validation layer on and compare every proof scene with the software renderer, but a test has no display to look at, no ProMotion refresh and no second GPU.

1. **Frame times**: run `GLUTTER_FRAME_STATS=1 go run ./examples/gallery` (or `GLUTTER_FRAME_STATS=1 go run ./cmd/glutter-macos shot ./examples/gallery`, not `-o`: it prints the frame numbers) on the Mac's own display, scroll a long page (Lists & cards), and read the frame times the log prints: they are within the budgets of §15 (the 99th percentile under the frame time of the display). Record the Mac, the display and the numbers.
2. **ProMotion**: on a Mac with a ProMotion display (or an external 120 Hz one), the same run shows 120 frames a second while a page scrolls, and the scrolling is smooth to the eye.
3. **Live resize**: drag the window's corner for several seconds: the content follows the window's edge with no stripe of the old size and no flicker (the present is made with `presentsWithTransaction` during the resize).
4. **The log**: the start of the run says `renderer chosen renderer="metal: <your GPU>"`; `GLUTTER_RENDERER=soft` says `soft`.
5. **Second start**: run the gallery twice with `GLUTTER_SHADER_CACHE` unset: the second start's first frame is not slower than the first's (the pipelines come from `~/Library/Caches/glutter/shaders/metal-*.archive`).
6. **A Mac with two GPUs** (an Intel MacBook Pro with an integrated and a discrete GPU), if there is one: `GLUTTER_METAL_DEVICE=lowpower` names the integrated GPU in the log and the gallery draws as with the default one; on a Mac with a discrete GPU the managed-memory path is used, which only that hardware shows.
7. **Moving between displays**: drag the window between a Retina display and a display of scale 1 (or the other way): the content stays sharp and the right size after the move.

Answers: _(not yet recorded)_

## 33.9 The `gpu` package and the 3D view on Metal (macOS)

**Status:** pending

The tests draw the scenes of the 3D view on Metal with the validation layer on and compare them with the goldens of Linux, but a test does not orbit a model with a trackpad or play a game.

1. **CAD viewer**: `go run ./examples/cad`: orbit with the mouse (drag) and with the trackpad (two-finger drag, pinch to zoom, two-finger scroll to pan), the view cube and the standard views, picking a face, a section plane with its cap, the edges and wireframe modes: the model follows the fingers smoothly at the display's rate, with no flicker.
2. **Game**: `go run ./examples/game3d` with the keyboard (WASD, Space, the arrows, Esc to pause): the character moves and jumps and the picture is smooth at the display's rate (120 Hz on a ProMotion display). With a gamepad, after step 33.14.
3. **Resizing**: resize the window of both apps with the mouse: the 3D view resizes with it, sharp, with no stripe of the old size.
4. **Retina and scale 1**: move the window to a display of the other scale: the 3D view stays sharp.

Answers: _(not yet recorded)_

## 33.10 Windows and decorations (macOS)

**Status:** pending

The display tests run in an accessory app, whose windows never take the keys, never go to another Space and are never dragged by a real mouse button, so these need a person at a Mac.

1. **Modal dialog**: `go run ./examples/multiwindow`, press "Open dialog": the dialog opens over the window and the window behind takes no click, no scroll and no key (clicking it brings the dialog forward); Command+W closes the dialog (not the window behind); the window behind has the keys again and is in front. Open two dialogs from a dialog and close them in both orders.
2. **Close button of the blocked window**: with the dialog open, the red button of the window behind does nothing.
3. **App-drawn title bar**: in an app that sets `DecorationsApp`, drag the bar: the window follows the mouse and the buttons of the app on the bar still work; the traffic lights are not covered by the app's content.
4. **Double click on the bar**: set "Double-click a window's title bar to" in System Settings, Desktop & Dock to Zoom, Minimize and Do Nothing, and double-click the bar that the app draws: it zooms, minimizes or does nothing; the Zoom button of the bar always zooms.
5. **Dock**: minimize a window, click the Dock icon: it comes back. With `QuitOnLastWindowClosed` false (an app with a tray icon), close the last window: the app stays in the Dock and the menu bar; Command+Q ends it.
6. **Spaces, Mission Control and Stage Manager**: a window and its dialog on a second Space; Mission Control shows the dialog with its window; full screen on a window with a dialog open (the dialog stays over it); Stage Manager with two windows of the app.

Answers: _(not yet recorded)_

## 33.11 Drag and drop (macOS)

**Status:** pending

The tests call the destination and the source by hand: the real drag follows the mouse of the user, which a test must not move.

1. **Files in**: `go run ./examples/multiwindow`, drag one file, then several, from Finder onto "Drop here": the paths are shown; the cursor shows the copy badge with Option held and the no-drop cursor over a window that has a dialog open.
2. **Text and links in**: select text in TextEdit or Safari and drag it onto "Drop here": it is shown; drag a link from Safari: the URL is shown.
3. **Text out**: drag "Drag me" out of the window into TextEdit: its text lands in the document; into a Finder window: the file or the text clipping lands there; the drag image follows the pointer and goes back when the drop is refused.
4. **Between windows of the app**: drag "Drag me" from one window onto the "Drop here" of another: it is known as the app's own drag.
5. **Escape** during a drag cancels it, and the pointer works normally afterwards (a click on a button in the window right after).

Answers: _(not yet recorded)_

## 33.12 Tray, notifications, file dialogs and open URI (macOS)

**Status:** pending

The panels are views of another process, which a test cannot press (only cancel), the notification center works only from an application bundle, and a status item is a thing the user looks at.

1. **Tray**: `go run ./examples/tray`: the icon is in the menu bar, a template image that follows the light and dark menu bar; the tooltip shows on hover; the menu opens with its items, the check marks, the disabled item and the submenu, and each item does what the example says; the icon goes when the app ends; with no menu a click calls the primary action and a right click the secondary one.
2. **File dialogs**: in an app that opens files (the editor example): open one file, several, a folder; save with a name and a type filter (the panel adds the extension); Cancel; the app goes on drawing while the panel is open.
3. **Open URI**: a link that opens `https://` opens the default browser; `mailto:` opens the mail app; a `file:` URI of a picture opens Preview; a `.app` or `.sh` file URI is refused.
4. **Notifications**, from an app bundle built by step 33.15 (`glutter-macos`): the first notification asks for the permission; allowed, it shows as a banner with its buttons; the buttons and the click call the app; the notification goes after its timeout; replacing one changes it in place; with the permission refused, `Notify` fails with `NotificationsUnavailable` and the app goes on. With `go run` it fails with that tag and says why.

Answers: _(not yet recorded)_

## 33.13 Accessibility (macOS)

**Status:** pending

The tests read the tree through the view's accessibility properties and send the actions by hand. VoiceOver and the Accessibility Inspector are what the elements are for.

1. **VoiceOver** (Cmd+F5) through the gallery (`go run ./examples/gallery`): the window is announced with its title; buttons are read with their labels and as buttons, a switch as a switch with on or off, a slider with its value (and Control+Option+Up and Down change it), a check box with its state, a text field with its label, its text and the caret as you type and move it, a list and its items; VO+Space presses a button and toggles a switch; a dialog is read when it opens; the live region of a snackbar is announced.
2. **Accessibility Inspector** on the same windows (Xcode, Open Developer Tool): the hierarchy has the roles and labels, the frames of the elements are on the controls, the actions listed for a button are `AXPress`, and Press works from the Inspector.
3. **Focus**: tab through the controls with VoiceOver on: the VoiceOver cursor follows the focus ring and the other way round when a control takes the focus by a click.
4. **Two windows**: with a second window open, VoiceOver reads the right tree for each; closing a window leaves no ghost element.

Answers: _(not yet recorded)_

## 33.14 Gamepads and audio (macOS)

**Status:** pending

The tests have no pad and never record: the elements, the values, the HID drivers and the queues are tested over fakes, and a real output plays an inaudible sine. A person with a pad and a microphone checks the rest.

1. **A generic pad** (a USB or Bluetooth one that macOS shows as a HID game pad): `go run ./examples/dev/pad-hid` shows the pad when it is plugged in and removes it when it is unplugged; the buttons, sticks, triggers and the d-pad are in the right places of the standard layout (the SDL table for "Mac OS X" has its GUID or the automatic mapping is used); pressing a button moves the focus of a Material app (the key events of pads).
2. **A DualShock 4 or DualSense and a Switch Pro** over Bluetooth: the pad is seen once (with the HID driver: the light, the rumble, the touchpad and the sensors work in `examples/dev/pad-hid`); pressing the PS button does not open anything of the system that the app does not expect. Unplugging or switching the pad off removes it; connecting it again adds it again.
3. **Coin Run** (`go run ./examples/game3d`) with a pad: it is played with the pad, rumbles on a hit, and the pause menu is driven by the pad's key events.
4. **Rumble of a generic pad**: refused with `GamepadRumbleUnavailable` (SDL plays it through the Force Feedback framework, which is not done); the app goes on.
5. **Audio output**: `examples/game3d` and `go run ./examples/dev/android-audio` (a tone through the public audio package, with its log of the stats) play sound on the speakers; a pause and a resume are heard as such; unplugging the headphones moves the sound to the speakers (the default device is followed) with no crash; plugging in an USB or Bluetooth device and making it the default moves the sound there.
6. **A named device**: opening a device by its `ID` from `audio.System.Devices` and then unplugging it makes the stream `Lost` once.
7. **Capture**: the first recording asks for the microphone (from the app bundle of step 33.15 it uses the bundle's usage string; from `go run` it asks about the terminal); allowed, `GLUTTER_AUDIO_CAPTURE=1 go run ./examples/dev/android-audio` logs the frames that arrive; refused, `Open` fails with `AudioCapturePermissionDenied` and the app goes on. Changing the default input while recording moves the recording to the new input.
8. **Latency**: a loopback or a hand clap and the monitor: the output is within about 30 ms (the target of SPEC §28.6); a 5.1 or 7.1 device plays the channels in the order of its outputs.

Answers: _(not yet recorded)_

## 33.15 The bundle and the tool (macOS)

**Status:** pending

The tests build every example as a signed bundle, run it, and `codesign --verify` it; what they cannot show is what Finder, the Dock and Gatekeeper do with it.

1. **The gallery from Finder**: `go run ./cmd/glutter-macos build -release -o ~/Desktop/Gallery.app ./examples/gallery`, then open it with a double click. macOS warns that the app is from an unidentified developer (it is signed ad hoc: that is expected); Control-click, Open, Open (or System Settings, Privacy & Security, Open Anyway) starts it. The Dock shows its icon (Glutter's "G" in a rounded square, with the margin of macOS icons) and the menu bar says "Gallery"; Cmd+Q quits it; \`Get Info\` shows the version 1.0.0.
2. **A bundle made for the other kind of Mac**: with \`-universal\`, the same \`.app\` opens on an Intel Mac and on an Apple silicon one (\`file Gallery.app/Contents/MacOS/Gallery\` says two architectures); on an Intel Mac the first frame is drawn by Metal (managed memory) and the numbers of \`scripts/perf-macos.sh\` are recorded.
3. **The microphone**: a bundle with \`-usage NSMicrophoneUsageDescription="..."\` that records asks with that text (see 33.14, item 7); built with \`-hardened\` it asks as well.
4. **Notifications** from the bundle (33.12, item 4) are checked here with the bundle of the tray example: \`go run ./cmd/glutter-macos build -o Tray.app ./examples/tray\`, opened from Finder (a notification needs the bundle to be started by Launch Services).
5. **The numbers on your Mac**: \`scripts/perf-macos.sh -m -u\` and the table of "macOS (step 33.15)" in \`docs/PERFORMANCE.md\`; the proposed targets of SPEC §15 ("macOS") are confirmed, or changed, from them (the resident memory of 193 MB against the 60 MB of the desktop target is the open question).

Answers: _(not yet recorded)_

## 34.10 The Cupertino gallery and the release of 1.14

What only a person (or the Linux machine) can do; the results of the Mac are in `docs/IMPLEMENTED.md`.

1. **Rule 8 on the Linux machine**: the Linux, Wine and Android suites of `docs/IMPLEMENTATION.md` rule 8, which a Mac cannot run. A failure is a bug of the step that caused it, fixed before the tag `v1.14.0` (and `v1.13.0`, which waits for the same run) is made and the milestone is published.
2. **`examples/cupertino` on Linux, under Wine and on the Android emulator** (`go run ./examples/cupertino`, `go run ./cmd/glutter-windows run -headless ./examples/cupertino`, `go run ./cmd/glutter-android run ./examples/cupertino`): the five tabs, the dark mode switch on the Settings tab, a push, a sheet, a fullscreen page and a popup, the pull to refresh, and the text (the machine's default font draws where macOS has San Francisco).
3. **The web proof with Chromium**: `go run ./cmd/glutter-web shot -browser chromium -width 390 -height 760 -o docs/implemented-proof-images/34.10-cupertino-gallery-web.png ./examples/cupertino` (the image in the repository was taken from the built-in browser of the Claude app, a Chromium; there is no Chromium on the Mac of this step). Say which renderer it drew with.
4. **On the Mac**: a pull to refresh with the trackpad (the dev events have no drag), the back swipe from the left edge, the large title collapsing, the pickers with a mouse wheel and a trackpad, and the context menu on a long press.

Answers: _(not yet recorded)_

## M33: everything to check on a Mac, in one list

The checks of the steps 33.2 to 33.15 are in their own sections above; this is the order to do them in, from one session, on a Mac with a trackpad, a microphone, headphones, a pad and a second display if there is one. Build the gallery bundle once (\`go run ./cmd/glutter-macos build -release -o Gallery.app ./examples/gallery\`) and use \`go run ./cmd/glutter-macos run ./examples/<name>\` for the rest; \`GLUTTER_FRAME_STATS=1\` and \`GLUTTER_RENDERER=metal|soft\` are environment variables. Say which Mac and which macOS the answers come from, and record them in each section.

| Section | What only a person can see |
|---|---|
| 33.2 | the window, its resizing and the Retina scale; a second display with another scale |
| 33.3 | a trackpad's scroll with momentum, pinch and rotate, a tablet's pressure, the cursors |
| 33.4 | the keyboard layouts (dead keys, a non-Latin layout), Cmd shortcuts, key repeat |
| 33.5 | the IMEs of Chinese, Japanese and Korean, the emoji picker, the pasteboard with other apps |
| 33.6 | dark mode and the accent colour following the system, the fonts of the system, quit and the Dock menu |
| 33.7 | the menu bar, the application menu, shortcuts shown in the items |
| 33.8, 33.9 | Metal on this GPU: the gallery and \`examples/cad\` drawn right, a ProMotion display at its rate |
| 33.10 | several windows, the full screen, window dragging by its background, the double-click on the title bar; **the owner's desktop screenshot of the multi-window example** (not yet taken) |
| 33.11 | drag and drop with Finder and other apps |
| 33.12 | the tray icon in light and dark menu bars, notifications from a bundle, the file dialogs, opening a link |
| 33.13 | VoiceOver and the Accessibility Inspector |
| 33.14 | a pad (generic, DualShock 4, DualSense, Switch Pro), the sound and its devices, the microphone |
| 33.15 | the bundle from Finder, its Dock icon and Gatekeeper's warning, the numbers on another Mac |

A failure found here is a bug of the step that the section names: it is fixed on a branch of its own before 1.13 is announced to anyone.


## 35.4 Touch, the Pencil, the pointer and hardware keys (iOS)

What the Simulator tests cannot do (the headless Simulator takes no real touch; a script gives the view's conversion its touches). Do these on an iPhone and an iPad, with the Simulator's own window for the first two.

1. **Real touches.** Run `go run ./cmd/glutter-ios run ./examples/counter` (a Simulator window) and click the button: the counter counts. Run `./examples/scrolling` and drag the list with the mouse: it scrolls with the iOS physics and bounces at both ends.
2. **Multi-touch.** On a device or with Option held in the Simulator, pinch and rotate in an `InteractiveViewer` (`examples/gallery`, the page of that name).
3. **The Pencil** on an iPad: the pressure of a stroke changes its width in a drawing widget; hover (iPad Pro M2 or later) shows the hover state. Hover is known not to be delivered as the stylus's yet.
4. **A trackpad or mouse** on an iPad: hover highlights buttons, two-finger scroll scrolls with momentum, pinch zooms.
5. **A hardware keyboard** (Magic Keyboard, or the Mac's keyboard in the Simulator): arrows move focus, Tab walks the focus, Cmd+C and the others reach shortcuts, a key that is held does not repeat (UIKit does not repeat it; note whether it should).

Record the answers here when they are known.


## 35.5 Text input and the soft keyboard (iOS)

On an iPhone and an iPad, with a field of `examples/gallery` (the text field page) and `examples/editor`:

1. **The QWERTY keyboard.** Type with autocorrection and suggestions on: the suggestion bar offers words, a tap on one replaces the word, and the correction keeps the caret.
2. **Japanese and Chinese.** Add the keyboards (Kana, Pinyin): the composition underlines, the candidates choose, the commit lands in the field.
3. **Dictation.** The microphone key dictates into the field.
4. **The space-bar cursor** (long press on the space bar): known not to move the framework's caret yet (SPEC §30, "Decided in step 35.5").
5. **A password field** with AutoFill: the keyboard offers the saved password; the text is secure and has no suggestions.
6. **The keyboard's insets.** The app moves above the keyboard as it comes, with the keyboard's own curve; on an iPad with the floating keyboard it does not move.
7. **A hardware keyboard** typing into a field: the letters arrive, and the shortcuts still reach the framework.

Record the answers here when they are known.

## 35.6 System UI and services (iOS)

On an iPhone and an iPad, with `examples/gallery`:

1. **Haptics.** The gallery's haptic buttons (or a `HapticFeedback` call): light, medium and heavy impact, the selection tick and the vibration each feel different on an iPhone; an iPad (no Taptic Engine) does nothing and does not fail.
2. **The paste prompt.** Copy text in another app, then paste into a field of the app: iOS 16 and later asks "Allow Paste"; the app keeps drawing while the prompt is up (the read waits on a goroutine), "Allow" pastes the text, and "Don't Allow" gives an empty paste with the `PasteDenied` error in the log.
3. **The status bar and the home indicator.** `SetSystemUIOverlayStyle` with a dark status bar bar gives white icons; the immersive modes hide the status bar and the home indicator hides itself after a moment; edge to edge brings them back.
4. **Orientations.** `SetPreferredOrientations` with landscape only turns an upright phone to landscape and keeps it there.
5. **Notifications.** The first notification shows the permission prompt; after "Allow" the banner shows with its action buttons, a tap on the banner and on an action reach the callbacks (also from the lock screen and when the app was in the background), and a dismissal reaches `OnClosed`.
6. **Opening a URI.** An `https:` URI opens Safari, a `mailto:` URI the Mail composer, a `tel:` URI the dialer (on an iPhone); a scheme that no app handles fails with `OpenURIUnavailable`.
7. **Spell check.** A text field with spell check on underlines a misspelled word, and a tap on it offers the suggestions of `UITextChecker`.

Record the answers here when they are known.

## 35.7 VoiceOver, Dynamic Type and the settings (iOS)

On an iPhone and an iPad, with `examples/gallery` and `examples/counter`:

1. **VoiceOver.** Turn it on (Settings > Accessibility > VoiceOver): swiping right reads the elements in order (the app bar's buttons and title, the heading and its text, the mode segments with "selected", the seed colors, the cards); a double tap on a segment or a card activates it; the counter's button reads "Increment, button" and a double tap adds one and VoiceOver reads the new number.
2. **Adjustable.** A slider (the gallery's page of sliders) reads "adjustable" and a swipe up or down changes it and reads the new value.
3. **Scrolling.** Three-finger swipes scroll a long page (the gallery's pages) up, down, left and right. The native "page 2 of 5" announcement is not done (SPEC §30.7).
4. **Rotor.** The Headings rotor jumps between the headings (header trait) and the Links rotor between links; containers and the list rotor are not done.
5. **Switch Control and Voice Control.** "Tap Increment" with Voice Control presses the counter's button; Switch Control scans the elements and presses them.
6. **Dynamic Type.** Settings > Display & Text Size > Larger Text: the text of the gallery follows the slider while the app runs, with no restart, and the layout still works at the largest accessibility size (the proof image of the step); Bold Text makes the text bold, Reduce Motion stops the animations, Increase Contrast gives the high contrast scheme.
7. **Dark mode and locales.** The Control Center's dark mode switch changes the gallery at once; adding a language in Settings > General > Language & Region changes the app's locales.
8. **The 24-hour clock.** Settings > General > Date & Time > 24-Hour Time changes the format of a time picker.

Record the answers here when they are known.

## 35.8 Gamepads, audio and Core Motion (iOS)

On an iPhone with a paired controller (a DualSense, a DualShock 4, an Xbox or a MFi pad), with `examples/game3d` and `examples/dev/` sensors page where it exists:

1. **A pad in `game3d`.** Connect the pad (Settings > Bluetooth) before and after the app starts: Coin Run takes it in both cases; the left stick moves, the bottom face button jumps, Menu pauses; disconnecting it in the middle of a run stops the movement without a crash.
2. **Rumble and light.** A coin or a hit rumbles a pad that has haptics (a DualSense, an Xbox pad; the Simulator cannot show it); the light of a DualShock or a DualSense takes the color the app sets; a pad with neither says so with the tags and the game goes on.
3. **Sound with a call.** Play the game's music; an incoming call (or an alarm) pauses the sound; hanging up (or dismissing it) resumes it where it should resume; a call that the user declines does the same.
4. **Routes.** Plug in or connect headphones: the sound follows; pull them out: the sound pauses and the app resumes it when the player asks; the silent switch does not mute a playback stream.
5. **Capture.** A page that opens the microphone asks for the permission once; with it the category is play and record and the speaker still plays (the loudspeaker is not forced to the receiver).
6. **Accelerometer and gyroscope.** In the sensors page: at rest on a table the accelerometer reads about +9.8 on Z; tilting the phone to the right gives a positive X; the gyroscope reads about 0 at rest and turns with the phone; the readings arrive at 60 Hz, and at 100 Hz when asked; closing the page stops the updates.

Record the answers here when they are known.

## 35.9 Every example on iOS, the release of 1.15, and everything to check on an iPhone and an iPad in one list

What only a person (or the Linux machine) can do; the results of the Mac and the Simulator are in `docs/IMPLEMENTED.md`.

1. **Rule 8 on the Linux machine**: the Linux, Wine and Android suites of `docs/IMPLEMENTATION.md` rule 8, which a Mac cannot run. A failure is a bug of the step that caused it, fixed before the tags `v1.13.0`, `v1.14.0` and `v1.15.0` are made and the milestones are published (`git push --follow-tags origin main`).
2. **The release tags**, after item 1: `git tag -a v1.13.0 -m "Glutter 1.13: macOS" <the commit of step-33.16>`, `git tag -a v1.14.0 -m "Glutter 1.14: Cupertino" <the commit of step-34.10>` and `git tag -a v1.15.0 -m "Glutter 1.15: iOS" <the commit of step-35.9>`.
3. **Build for a device and run.** With your team and a development profile that has the phone: `go run ./cmd/glutter-ios build -team <ID> -profile <file> -id <your id> -o Gallery.app ./examples/gallery`, then `xcrun devicectl device install app --device <UDID> Gallery.app` and start it. Say which iPhone, which iPad and which iOS the answers come from.
4. **Every example on a phone**: `animations`, `cad`, `counter`, `cupertino`, `custompaint`, `editor`, `gallery`, `game3d`, `gpu-triangle`, `layout`, `navigation`, `scrolling`, `svg` and `todo` start, draw and respond to touch (`multiwindow` and `tray` are not on iOS).
5. **Performance and ProMotion.** `GLUTTER_FRAME_STATS=1` is not an option on a device without Xcode's environment: use the scheme's environment or the Instruments of Xcode (Time Profiler, Core Animation's frame rate, Allocations, Leaks). The gallery's animation and a fling of a list run at 120 frames a second on a ProMotion phone and 60 on the others; the cold start of an app that was just installed (the first start, then after a restart of the phone) to the first frame; the CPU of an idle app is near zero; the memory of the gallery after ten minutes of use is flat and well under the system's limit (an app that is ended with "jetsam" in the device's log has gone over it). Record the numbers in `docs/PERFORMANCE.md`, "iOS (step 35.9)".
6. **The framework's iOS behaviors**, which the Simulator's tests check only through scripts: the bounce of every list at its start and its end and the fling that runs into it; the back swipe from the left edge on a pushed page (the page follows the finger, a short swipe goes back, a swipe from the middle does not); in a text field, a double tap selects a word, a long press shows the magnifier and the drag moves the caret, the two handles move the ends of the selection, the toolbar offers Cut, Copy, Paste and Select All, and a tap on the caret brings the toolbar back.
7. **The checks of the steps 35.2 to 35.8**, in this order, each in its own section above: 35.2 (rotation, the safe area, the Dynamic Island and the home indicator), 35.3 (Metal in the background and back, a call over the app), 35.4 (touch, the Pencil, the pointer, a hardware keyboard), 35.5 (the soft keyboards, an input method, the emoji keyboard, a dictation), 35.6 (the status bar, haptics, the pasteboard and its prompt, notifications, the spell checker), 35.7 (VoiceOver, Dynamic Type and the switches) and 35.8 (a pad in `game3d`, sound with a call, the accelerometer and gyroscope, rumble and the light).
8. **An iPad**: the gallery's wide layout in the full window, in Split View and in Slide Over (the window's size changes while the app runs), with the keyboard and the trackpad, and in each orientation.

Answers: _(not yet recorded)_

## 36.1 Gamepads start when they are wanted

What only a real pad (and the Linux machine) can show. The unit tests of the Linux service run only on Linux: run them first (`go test ./internal/platform/linux/evdev`), with the rest of rule 8.

1. **`strace`** of `examples/counter` on Linux, before and after (record both): `strace -f -e trace=openat,inotify_add_watch go run ./examples/counter 2>&1 | grep -E "input|hidraw"`. Before the step `/dev/input` is watched and the event nodes of the pads are opened (and `/dev/hidraw*` for a PlayStation, Switch Pro or Bluetooth Xbox pad); after it, by default, the event nodes are opened read only and no hidraw node is opened, and with `GLUTTER_GAMEPADS=off` neither is.
2. **A PlayStation 4 or 5 pad over Bluetooth, with the udev rule for hidraw** (`docs/GUIDE.md`): `examples/counter` leaves the light bar as the pad had it (it changed before); `examples/gallery` is moved with the pad's D-pad and South button (the keys still work in `auto`).
3. **`examples/dev/pad-hid` and `examples/game3d`** still get the light, the rumble and the touchpad; the pad disconnects and connects once when the app starts to use the package (the log shows `gamepad connected` twice), and the app's own listener sees that.
4. **`GLUTTER_GAMEPADS=off go run ./examples/game3d`** has no pad and no keys from one; `GLUTTER_GAMEPADS=ondemand go run ./examples/gallery` has no pad keys, and `game3d` with it plays with the pad and has the light at once, with no reconnection.
5. **Rumble of an Xbox 360 or other evdev pad without a driver**: the first rumble works (the device is opened for writing then), and a pad whose node the user may not write to says `CanRumble() == false`.
6. **On Windows**, with a PlayStation pad: the same as 2 and 3 (the Xbox pads through XInput are unchanged).

Answers: _(not yet recorded)_

## 37.2 Reduce Transparency, Differentiate Without Color and On/Off Labels

What only a person with the system's settings can do.

1. **macOS**: in `examples/cupertino` (a navigation bar and a tab bar over a scrolling list) turn on System Settings > Accessibility > Display > Reduce transparency while the app runs: the bars become solid at once, with nothing blurred behind them, and a dialog or popup is solid; turn it off and the blur returns. Record whether the change needs a frame of input to show.
2. **macOS**: Differentiate without color: the log line of the app (`GLUTTER_...` is not needed) is not changed; an app that reads `MediaQueryDifferentiateWithoutColorOf` sees it change. `examples/dev` has no page for it: write one line in a scratch app, or read `ts` of a test.
3. **iOS**: Settings > Accessibility > Display & Text Size > Reduce Transparency, Differentiate Without Color and On/Off Labels with `examples/cupertino` open in the foreground: the bars turn solid; the `CupertinoSwitch` shows I and O marks with On/Off Labels; each change arrives without leaving the app.
4. **iPad** (Split View): the same, in a window that is not the whole screen.

Answers: _(not yet recorded)_

## 37.3 The rumble of a macOS pad through the Force Feedback framework

Needs a pad that macOS lists as force feedback and that no HID driver of Glutter serves (not a PlayStation 4 or 5, Switch Pro or Bluetooth Xbox pad): a generic USB pad with motors, a Logitech Rumblepad or Dual Action, a wired Xbox 360 clone.

1. Plug it in and run `go run ./examples/dev/pad-hid` (or `examples/game3d`): the pad is listed, and `CanRumble()` is true. Record the pad's name and its vendor and product numbers.
2. Press the button that rumbles (`game3d` rumbles on a hit): the pad vibrates for about the time the app asks (`Rumble(0.5, 0.5, 200 ms)` is a short pulse). Record whether it feels like a rumble or like a one-second wave (SDL's sine has a period of one second); if it is a wave, the period of the effect is the thing to change.
3. Two rumbles at once: the second changes the first (no pile-up), and a rumble of strength 0 stops the motors at once.
4. Unplug the pad while it rumbles: nothing hangs, the pad is removed, and plugging it in again rumbles again.
5. A pad that has no force feedback says `CanRumble() == false` and `Rumble` returns `GamepadRumbleUnavailable`.

Answers: _(not yet recorded)_

## 39.1 The title bar of an app on macOS

An app that draws its own title bar with `WindowDecoration` (Emetos, or `GLUTTER_DECORATIONS=app go run ./examples/counter`), on a Mac.

1. The window has one set of window buttons: the red, yellow and green ones at the left. The bar draws no minimize, maximize or close of its own at the right, and no line around the window.
2. The three buttons are in the middle of the bar's height, and the first widget of the bar (the menu button of Emetos) starts to the right of them, with nothing under them.
3. Resize the window from an edge and from a corner, give it another title, make it fullscreen with the green button and leave fullscreen: the buttons are in the middle of the bar again each time.
4. Drag the empty part of the bar: the window moves. Double-click it: the window zooms (or does what "Double-click a window's title bar to" says).
5. With the system language set to one written from right to left (Arabic, Hebrew): the buttons are at the right, and the bar ends before them.

Answers: _(not yet recorded)_

## 40.2 Frames when the Mac's own screen is asleep

A Mac with macOS 14 or later, used from another computer through Screen Sharing, with its own monitor asleep or off (or with no monitor).

1. Start any example (`go run ./examples/counter`) on the Mac from the other computer: the window draws, the button counts when clicked, and the hover of the pointer shows. Before step 40.2 the window drew its first frame and nothing after it.
2. Wake the Mac's monitor while the app runs and animate something (keep clicking): it goes on smoothly (the view's display link takes the pacing back); let the monitor sleep again: it goes on after a pause of about a tenth of a second.
3. With the monitor on and nobody connected from outside, an app is as before (the view's display link, the display's refresh rate).

Answers: _(not yet recorded)_

