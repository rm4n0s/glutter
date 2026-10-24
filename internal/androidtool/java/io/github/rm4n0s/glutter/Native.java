package io.github.rm4n0s.glutter;

/**
 * The native methods of the embedding and the calls from Go. Go implements the methods in
 * internal/platform/android (cgo, JNI); Go calls the static methods of GlutterActivity.
 *
 * <p>Ports: the {@code FlutterJNI} of Flutter's embedding
 * (shell/platform/android/io/flutter/embedding/engine/FlutterJNI.java), only the part that starts
 * the engine.
 */
final class Native {
  private Native() {}

  /**
   * Sets an environment variable of the Go program, as {@code os.Setenv} does. Go copies the
   * environment when it starts, so {@code setenv(3)} of libc would not be seen by {@code
   * os.Getenv}.
   */
  static native void setenv(String key, String value);

  /**
   * Runs the program's {@code main.main} on a goroutine locked to its own OS thread, once per
   * process. It returns at once.
   */
  static native void start(GlutterActivity activity);

  /**
   * The view has a new surface. Go acquires the native window and releases it when it has let go
   * of it.
   */
  static native void surfaceCreated(android.view.Surface surface);

  /** The size of the surface in pixels, the density and refresh rate of the display, and its size. */
  static native void surfaceChanged(
      int width, int height, float density, float refreshRate, int displayWidth, int displayHeight);

  /**
   * The surface is going away. It does not return until the renderer has let go of it (at most two
   * seconds), because Android destroys the surface when it returns.
   */
  static native void surfaceDestroyed();

  /**
   * The lifecycle of the activity: 1 resumed, 2 paused, 3 stopped, 4 destroyed for good (finishing,
   * not destroyed to be created again).
   */
  static native void lifecycle(int state);

  /**
   * A new activity of the living process has its view: Go sends again what it keeps on the
   * activity (whether the framework handles back, the text input state).
   */
  static native void activityCreated();

  /**
   * A back: {@code kind} 1 is a pop (a plain system back), 2 the start of a back gesture, 3 its
   * progress, 4 its commit, 5 its cancel; {@code touchX} and {@code touchY} are in pixels and
   * {@code progress} is 0 to 1 (0 for kinds 1, 4 and 5); {@code edge} is 0 unspecified, 1 left, 2
   * right.
   */
  static native void back(int kind, float touchX, float touchY, float progress, int edge);

  /** {@code Activity.onTrimMemory}: the {@code TRIM_MEMORY_*} level. */
  static native void trimMemory(int level);

  /** Whether the activity's window has the focus. */
  static native void focus(boolean focused);

  /**
   * A motion event copied out: {@code meta} is {actionMasked, actionIndex, source, buttonState,
   * eventTime, pointerCount, historySize, generic}; {@code ids} has the pointer id and the tool
   * type of each pointer; {@code vals} has 11 values for each pointer (x, y, pressure, size,
   * toolMajor, toolMinor, orientation, tilt, distance, hscroll, vscroll), first for the event,
   * then for each historical sample, oldest first; {@code histTimes} the time of each sample.
   */
  static native void motionEvent(
      long[] meta,
      int[] ids,
      float[] vals,
      long[] histTimes,
      float pressureMin,
      float pressureMax,
      float hScrollFactor,
      float vScrollFactor);

  /** A key event copied out; {@code id} comes back in {@code GlutterActivity.onKeyHandled}. */
  static native void keyEvent(
      int id,
      int action,
      int keyCode,
      int scanCode,
      int metaState,
      int repeatCount,
      long eventTime,
      int unicodeChar,
      int source,
      int deviceId);

  /**
   * The insets of the window in pixels: left, top, right and bottom of the system bars and the
   * display cutout, of the input method, and of the system gestures.
   */
  static native void insets(int[] values);

  /**
   * The system's settings as {@code key=value} pairs separated by {@code ;} (see settings.go of
   * the Go side): night, font, nonlinear, locales, h24, noanim, bold, contrast.
   */
  static native void settings(String text);

  /** The system showed ({@code true}) or hid its bars because of the user. */
  static native void systemUI(boolean visible);

  /** The density, the refresh rate and the size of the display changed. */
  static native void displayChanged(float density, float refreshRate, int displayWidth, int displayHeight);

  /**
   * The input method edited the text field: the sequence number of this edit (a counter per
   * process, from 1), the text as UTF-8, the selection and the composing region in UTF-16 offsets
   * (the composing region is -1, -1 when there is none).
   */
  static native void textEdit(
      int seq, byte[] text, int base, int extent, int composeBase, int composeExtent);

  /** The answer to a {@code OP_CLIPBOARD_GET}: whether the clipboard has text, and the text as UTF-8. */
  static native void clipboardText(int id, boolean ok, byte[] text);

  /**
   * The answer to a request of the desktop services ({@code DesktopServices}; see desktop.go for
   * the meaning of the code of each operation): the number of the request, the code, and a text as
   * UTF-8 (the reason, or empty).
   */
  static native void desktopResult(int request, int code, byte[] text);

  /**
   * A click on a notification (kind 1), on one of its buttons (2, {@code text} is the button's ID)
   * or its swipe (3, {@code text} is {@code dismissed}); {@code serial} is the number Go gave the
   * notification.
   */
  static native void notificationEvent(int serial, int kind, byte[] text);

  /**
   * The state of the accessibility manager: whether an accessibility service runs, whether touch
   * exploration is on, and Flutter's {@code accessibleNavigation} (touch exploration, or a service
   * that asked for the nodes). Go turns the semantics tree on while a service runs or touch
   * exploration is on.
   */
  static native void a11yState(boolean enabled, boolean touchExploration, boolean accessibleNavigation);

  /**
   * An accessibility service asked for an action: {@code nodeId} is the id of the semantics node,
   * {@code action} the {@code ui.SemanticsAction} bit that {@code performAction} chose, and {@code
   * args} its arguments (see decodeAction in accessibility_actions.go): a byte that says whether the
   * selection is extended for a cursor move, two little-endian ints (base and extent, as UTF-8 byte
   * offsets) for a selection, the UTF-8 text for a text, one int for a custom action.
   */
  static native void a11yAction(int nodeId, int action, byte[] args);
}
