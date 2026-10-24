package io.github.rm4n0s.glutter;

import android.content.Context;
import android.graphics.Insets;
import android.hardware.display.DisplayManager;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.os.SystemClock;
import android.util.DisplayMetrics;
import android.util.Log;
import android.util.SparseArray;
import android.view.Display;
import android.view.DisplayCutout;
import android.view.InputDevice;
import android.view.KeyEvent;
import android.view.MotionEvent;
import android.view.PointerIcon;
import android.view.SurfaceHolder;
import android.view.SurfaceView;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputConnection;
import android.view.ViewConfiguration;
import android.view.WindowInsets;
import android.view.WindowInsetsAnimation;
import android.view.WindowManager;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

/**
 * The view of a Glutter app: a SurfaceView that fills the activity's window. It tells Go about the
 * life of its surface and about the display; Go draws into the surface from its own thread.
 *
 * <p>Ports: {@code FlutterView} and {@code FlutterSurfaceView}
 * (shell/platform/android/io/flutter/embedding/android/FlutterView.java and
 * FlutterSurfaceView.java), the parts that steps 25.3 and 25.6 need: the surface's life, the
 * motion and key events, and the mouse cursor ({@code AndroidTouchProcessor}, {@code
 * KeyboardManager}, {@code MouseCursorPlugin}), and, from step 25.8, the input connection ({@code
 * FlutterView.onCreateInputConnection}) and the insets while the keyboard slides ({@code
 * ImeSyncDeferringInsetsCallback}). The view copies the values out of the events; Go does the
 * conversion.
 */
public class GlutterView extends SurfaceView implements SurfaceHolder.Callback {
  /** Values copied for each pointer: x, y, pressure, size, toolMajor, toolMinor, orientation, tilt, distance, hscroll, vscroll. */
  private static final int VALUES = 11;

  private static final String TAG = "GlutterJava";

  /** Key events that Go has not answered yet, by the id that went with them. Main thread only. */
  private final SparseArray<KeyEvent> pendingKeys = new SparseArray<>();

  /** Key events that are being dispatched again: they are not sent to Go a second time. */
  private final Set<KeyEvent> redispatching = new HashSet<>();

  private int nextKeyId = 1;

  /** The activity's system interface, told of the insets. */
  private final SystemUI systemUI;

  /** Text input of the activity: the input connection comes from it. */
  private TextInput textInput;

  /** The latest insets while the keyboard slides: they are reported when it stops. Main thread. */
  private WindowInsets deferredInsets;

  /** Whether the keyboard is sliding: the insets come from the animation then. */
  private boolean imeAnimating;

  /** Hears of changes of the display: its refresh rate, density and size. */
  private final DisplayManager.DisplayListener displayListener =
      new DisplayManager.DisplayListener() {
        @Override
        public void onDisplayAdded(int displayId) {}

        @Override
        public void onDisplayRemoved(int displayId) {}

        @Override
        public void onDisplayChanged(int displayId) {
          reportDisplay();
        }
      };

  public GlutterView(Context context, SystemUI systemUI) {
    super(context);
    this.systemUI = systemUI;
    getHolder().addCallback(this);
    setFocusable(true);
    setFocusableInTouchMode(true);
  }

  /** Gives the view the text input that makes its input connections. */
  void setTextInput(TextInput textInput) {
    this.textInput = textInput;
    if (Build.VERSION.SDK_INT >= 30) {
      setWindowInsetsAnimationCallback(new ImeInsetsCallback());
    }
  }

  /** A text editor, with an input connection, while a text field of the framework has the focus. */
  @Override
  public boolean onCheckIsTextEditor() {
    return textInput != null && textInput.isActive();
  }

  /**
   * The input connection for the soft keyboard: null unless a text field has the focus.
   *
   * <p>Ports: {@code FlutterView.onCreateInputConnection}
   * (shell/platform/android/io/flutter/embedding/android/FlutterView.java), which asks the text
   * input plugin.
   */
  @Override
  public InputConnection onCreateInputConnection(EditorInfo outAttrs) {
    if (textInput == null) {
      return null;
    }
    return textInput.createInputConnection(outAttrs);
  }

  @Override
  public boolean onTouchEvent(MotionEvent event) {
    sendMotion(event, false);
    return true;
  }

  /** Mouse hover, the scroll wheel and the other pointer movements that are not touches. */
  @Override
  public boolean onGenericMotionEvent(MotionEvent event) {
    boolean isPointer = event.isFromSource(InputDevice.SOURCE_CLASS_POINTER);
    int action = event.getActionMasked();
    if (isPointer && (action == MotionEvent.ACTION_HOVER_MOVE || action == MotionEvent.ACTION_SCROLL)) {
      sendMotion(event, true);
      return true;
    }
    return super.onGenericMotionEvent(event);
  }

  /** Copies the values of a motion event, with its historical samples, out to Go. */
  private void sendMotion(MotionEvent event, boolean generic) {
    int n = event.getPointerCount();
    int history = event.getActionMasked() == MotionEvent.ACTION_MOVE ? event.getHistorySize() : 0;
    long[] meta = {
      event.getActionMasked(), event.getActionIndex(), event.getSource(), event.getButtonState(),
      event.getEventTime(), n, history, generic ? 1 : 0
    };
    int[] ids = new int[2 * n];
    float[] vals = new float[VALUES * n * (1 + history)];
    long[] times = new long[history];
    for (int p = 0; p < n; p++) {
      ids[2 * p] = event.getPointerId(p);
      ids[2 * p + 1] = event.getToolType(p);
    }
    for (int h = 0; h < history; h++) {
      times[h] = event.getHistoricalEventTime(h);
      for (int p = 0; p < n; p++) {
        int o = VALUES * (n * (1 + h) + p);
        vals[o] = event.getHistoricalX(p, h);
        vals[o + 1] = event.getHistoricalY(p, h);
        vals[o + 2] = event.getHistoricalPressure(p, h);
        vals[o + 3] = event.getHistoricalSize(p, h);
        vals[o + 4] = event.getHistoricalToolMajor(p, h);
        vals[o + 5] = event.getHistoricalToolMinor(p, h);
        vals[o + 6] = event.getHistoricalAxisValue(MotionEvent.AXIS_ORIENTATION, p, h);
        vals[o + 7] = event.getHistoricalAxisValue(MotionEvent.AXIS_TILT, p, h);
        vals[o + 8] = event.getHistoricalAxisValue(MotionEvent.AXIS_DISTANCE, p, h);
      }
    }
    for (int p = 0; p < n; p++) {
      int o = VALUES * p;
      vals[o] = event.getX(p);
      vals[o + 1] = event.getY(p);
      vals[o + 2] = event.getPressure(p);
      vals[o + 3] = event.getSize(p);
      vals[o + 4] = event.getToolMajor(p);
      vals[o + 5] = event.getToolMinor(p);
      vals[o + 6] = event.getAxisValue(MotionEvent.AXIS_ORIENTATION, p);
      vals[o + 7] = event.getAxisValue(MotionEvent.AXIS_TILT, p);
      vals[o + 8] = event.getAxisValue(MotionEvent.AXIS_DISTANCE, p);
      vals[o + 9] = event.getAxisValue(MotionEvent.AXIS_HSCROLL, p);
      vals[o + 10] = event.getAxisValue(MotionEvent.AXIS_VSCROLL, p);
    }
    float pressureMin = 0;
    float pressureMax = 1;
    InputDevice device = event.getDevice();
    if (device != null) {
      InputDevice.MotionRange range = device.getMotionRange(MotionEvent.AXIS_PRESSURE);
      if (range != null) {
        pressureMin = range.getMin();
        pressureMax = range.getMax();
      }
    }
    ViewConfiguration config = ViewConfiguration.get(getContext());
    Native.motionEvent(
        meta, ids, vals, times, pressureMin, pressureMax,
        config.getScaledHorizontalScrollFactor(), config.getScaledVerticalScrollFactor());
  }

  /**
   * Sends every key event to Go at once and takes it; when the framework did not handle it, {@link
   * #keyHandled} dispatches it again (as {@code KeyboardManager} does), so that Back, the volume
   * keys and the soft keyboard's keys still work.
   */
  @Override
  public boolean dispatchKeyEvent(KeyEvent event) {
    if (redispatching.remove(event)) {
      return super.dispatchKeyEvent(event);
    }
    if (event.getAction() == KeyEvent.ACTION_MULTIPLE) {
      return super.dispatchKeyEvent(event);
    }
    int id = nextKeyId++;
    pendingKeys.put(id, event);
    Native.keyEvent(
        id, event.getAction(), event.getKeyCode(), event.getScanCode(), event.getMetaState(),
        event.getRepeatCount(), event.getEventTime(), event.getUnicodeChar(), event.getSource(),
        event.getDeviceId());
    return true;
  }

  /** The answer of the framework to a key event (main thread). */
  void keyHandled(int id, boolean handled) {
    KeyEvent event = pendingKeys.get(id);
    pendingKeys.remove(id);
    if (event == null || handled) {
      return;
    }
    redispatching.add(event);
    getRootView().dispatchKeyEvent(event);
    redispatching.remove(event);
  }

  /** The dev events script (main thread): a synthetic tap, swipe or key, dispatched like real input. */
  void inject(int kind, float x1, float y1, float x2, float y2, int ms, String key) {
    final Handler handler = getHandler() != null ? getHandler() : new Handler(Looper.getMainLooper());
    if (kind >= 4 && kind <= 6) {
      if (textInput == null) {
        Log.w(TAG, "text injected but there is no text input");
      } else if (kind == 4) {
        textInput.injectText(key);
      } else if (kind == 6) {
        textInput.injectAction();
      } else {
        textInput.injectCompose(key);
      }
      return;
    }
    if (kind == 3) {
      int code = KeyEvent.keyCodeFromString(key);
      long now = SystemClock.uptimeMillis();
      dispatchKeyEvent(new KeyEvent(now, now, KeyEvent.ACTION_DOWN, code, 0));
      dispatchKeyEvent(new KeyEvent(now, now + 20, KeyEvent.ACTION_UP, code, 0));
      return;
    }
    final long downTime = SystemClock.uptimeMillis();
    dispatchTouchEvent(touch(downTime, downTime, MotionEvent.ACTION_DOWN, x1, y1));
    if (kind == 1) {
      final float x = x1;
      final float y = y1;
      handler.postDelayed(
          new Runnable() {
            @Override
            public void run() {
              long now = SystemClock.uptimeMillis();
              dispatchTouchEvent(touch(downTime, now, MotionEvent.ACTION_UP, x, y));
            }
          },
          50);
      return;
    }
    // A swipe: a move every 16 ms along the line, then the up.
    final int steps = Math.max(1, ms / 16);
    for (int i = 1; i <= steps; i++) {
      final float f = (float) i / steps;
      final float x = x1 + (x2 - x1) * f;
      final float y = y1 + (y2 - y1) * f;
      final int action = i == steps ? MotionEvent.ACTION_UP : MotionEvent.ACTION_MOVE;
      handler.postDelayed(
          new Runnable() {
            @Override
            public void run() {
              dispatchTouchEvent(touch(downTime, SystemClock.uptimeMillis(), action, x, y));
            }
          },
          (long) ms * i / steps);
    }
  }

  /** A one-finger touch event of the touch screen. */
  private static MotionEvent touch(long downTime, long eventTime, int action, float x, float y) {
    MotionEvent.PointerProperties props = new MotionEvent.PointerProperties();
    props.id = 0;
    props.toolType = MotionEvent.TOOL_TYPE_FINGER;
    MotionEvent.PointerCoords coords = new MotionEvent.PointerCoords();
    coords.x = x;
    coords.y = y;
    coords.pressure = action == MotionEvent.ACTION_UP ? 0f : 1f;
    coords.size = 1f;
    return MotionEvent.obtain(
        downTime, eventTime, action, 1, new MotionEvent.PointerProperties[] {props},
        new MotionEvent.PointerCoords[] {coords}, 0, 0, 1f, 1f, 0, 0, InputDevice.SOURCE_TOUCHSCREEN, 0);
  }

  /** Shows the mouse cursor of a {@code PointerIcon} type (main thread). */
  void setCursorType(int type) {
    setPointerIcon(PointerIcon.getSystemIcon(getContext(), type));
  }

  @Override
  protected void onAttachedToWindow() {
    super.onAttachedToWindow();
    DisplayManager dm = (DisplayManager) getContext().getSystemService(Context.DISPLAY_SERVICE);
    if (dm != null) {
      dm.registerDisplayListener(displayListener, new Handler(Looper.getMainLooper()));
    }
  }

  @Override
  protected void onDetachedFromWindow() {
    imeAnimating = false;
    deferredInsets = null;
    DisplayManager dm = (DisplayManager) getContext().getSystemService(Context.DISPLAY_SERVICE);
    if (dm != null) {
      dm.unregisterDisplayListener(displayListener);
    }
    super.onDetachedFromWindow();
  }

  /**
   * The insets of the window as Flutter works them out: the system bars and the display cutout are
   * the padding, the input method the view insets, the system gestures the gesture insets (all in
   * pixels).
   *
   * <p>Ports: {@code FlutterView.onApplyWindowInsets}
   * (shell/platform/android/io/flutter/embedding/android/FlutterView.java).
   */
  @Override
  @SuppressWarnings("deprecation")
  public WindowInsets onApplyWindowInsets(WindowInsets insets) {
    WindowInsets result = super.onApplyWindowInsets(insets);
    if (imeAnimating) {
      // The keyboard slides: the insets of the animation are reported, and these (the end state)
      // when it stops.
      deferredInsets = insets;
    } else {
      reportInsets(insets);
    }
    systemUI.onInsets(insets);
    return result;
  }

  /** Works the insets out and tells Go (pixels). */
  @SuppressWarnings("deprecation")
  private void reportInsets(WindowInsets insets) {
    int[] v = new int[12];
    if (Build.VERSION.SDK_INT >= 30) {
      Insets bars = insets.getInsets(WindowInsets.Type.systemBars());
      Insets ime = insets.getInsets(WindowInsets.Type.ime());
      Insets gestures = insets.getInsets(WindowInsets.Type.systemGestures());
      v[0] = bars.left;
      v[1] = bars.top;
      v[2] = bars.right;
      v[3] = bars.bottom;
      v[4] = ime.left;
      v[5] = ime.top;
      v[6] = ime.right;
      v[7] = ime.bottom;
      v[8] = gestures.left;
      v[9] = gestures.top;
      v[10] = gestures.right;
      v[11] = gestures.bottom;
      // The display cutout and the waterfall edges count as padding too, the larger one winning.
      DisplayCutout cutout = insets.getDisplayCutout();
      if (cutout != null) {
        Insets waterfall = cutout.getWaterfallInsets();
        v[0] = Math.max(Math.max(v[0], waterfall.left), cutout.getSafeInsetLeft());
        v[1] = Math.max(Math.max(v[1], waterfall.top), cutout.getSafeInsetTop());
        v[2] = Math.max(Math.max(v[2], waterfall.right), cutout.getSafeInsetRight());
        v[3] = Math.max(Math.max(v[3], waterfall.bottom), cutout.getSafeInsetBottom());
      }
    } else {
      int visibility = getWindowSystemUiVisibility();
      boolean statusBarVisible = (SYSTEM_UI_FLAG_FULLSCREEN & visibility) == 0;
      boolean navigationBarVisible = (SYSTEM_UI_FLAG_HIDE_NAVIGATION & visibility) == 0;
      int keyboard = guessBottomKeyboardInset(insets);
      v[0] = insets.getSystemWindowInsetLeft();
      v[1] = statusBarVisible ? insets.getSystemWindowInsetTop() : 0;
      v[2] = insets.getSystemWindowInsetRight();
      v[3] = navigationBarVisible && keyboard == 0 ? insets.getSystemWindowInsetBottom() : 0;
      v[7] = keyboard;
      if (Build.VERSION.SDK_INT == 29) {
        Insets gestures = insets.getSystemGestureInsets();
        v[8] = gestures.left;
        v[9] = gestures.top;
        v[10] = gestures.right;
        v[11] = gestures.bottom;
      }
    }
    for (int i = 0; i < v.length; i++) {
      v[i] = Math.max(0, v[i]);
    }
    Native.insets(v);
  }

  /**
   * Follows the keyboard's animation frame by frame (API 30 and later), so that the layout moves
   * with it. The insets of the final state, which the system dispatches when the animation
   * starts, are held back until it ends.
   *
   * <p>Ports: {@code ImeSyncDeferringInsetsCallback}
   * (shell/platform/android/io/flutter/plugin/editing/ImeSyncDeferringInsetsCallback.java), as far
   * as reporting each step of the animation; Flutter also defers the final inset to the animation's
   * start, which the change of the layout of this view does not need.
   */
  private final class ImeInsetsCallback extends WindowInsetsAnimation.Callback {
    ImeInsetsCallback() {
      super(WindowInsetsAnimation.Callback.DISPATCH_MODE_CONTINUE_ON_SUBTREE);
    }

    @Override
    public void onPrepare(WindowInsetsAnimation animation) {
      if ((animation.getTypeMask() & WindowInsets.Type.ime()) != 0) {
        imeAnimating = true;
      }
    }

    @Override
    public WindowInsets onProgress(WindowInsets insets, List<WindowInsetsAnimation> animations) {
      if (imeAnimating) {
        reportInsets(insets);
      }
      return insets;
    }

    @Override
    public void onEnd(WindowInsetsAnimation animation) {
      if (!imeAnimating || (animation.getTypeMask() & WindowInsets.Type.ime()) == 0) {
        return;
      }
      imeAnimating = false;
      WindowInsets last = deferredInsets;
      deferredInsets = null;
      if (last == null) {
        last = getRootWindowInsets();
      }
      if (last != null) {
        reportInsets(last);
      }
    }
  }

  /** Whether the bottom inset is the keyboard's: it is when it is higher than 18% of the screen. */
  @SuppressWarnings("deprecation")
  private int guessBottomKeyboardInset(WindowInsets insets) {
    int screenHeight = getRootView().getHeight();
    if (screenHeight == 0) {
      // Before the first layout the root view has no height: the display's is the one to compare.
      screenHeight = getResources().getDisplayMetrics().heightPixels;
    }
    if (insets.getSystemWindowInsetBottom() < screenHeight * 0.18) {
      return 0;
    }
    return insets.getSystemWindowInsetBottom();
  }

  /** Tells Go the density, the refresh rate and the size of the display. */
  void reportDisplay() {
    DisplayMetrics real = new DisplayMetrics();
    float refresh = 60;
    Display display = findDisplay();
    if (display != null) {
      display.getRealMetrics(real);
      refresh = display.getRefreshRate();
    }
    float density = getResources().getDisplayMetrics().density;
    Native.displayChanged(density, refresh, real.widthPixels, real.heightPixels);
  }

  @Override
  public void surfaceCreated(SurfaceHolder holder) {
    Native.surfaceCreated(holder.getSurface());
  }

  @Override
  public void surfaceChanged(SurfaceHolder holder, int format, int width, int height) {
    DisplayMetrics real = new DisplayMetrics();
    float refresh = 60;
    Display display = findDisplay();
    if (display != null) {
      display.getRealMetrics(real);
      refresh = display.getRefreshRate();
    }
    float density = getResources().getDisplayMetrics().density;
    Native.surfaceChanged(width, height, density, refresh, real.widthPixels, real.heightPixels);
  }

  @Override
  public void surfaceDestroyed(SurfaceHolder holder) {
    // Blocks until Go has let go of the surface (SPEC §24.3 "The one wait").
    Native.surfaceDestroyed();
  }

  /** The display the view is on, or the default one before it is attached. */
  private Display findDisplay() {
    if (Build.VERSION.SDK_INT >= 30) {
      Display d = getDisplay();
      if (d != null) {
        return d;
      }
    }
    WindowManager wm = (WindowManager) getContext().getSystemService(Context.WINDOW_SERVICE);
    return wm == null ? null : wm.getDefaultDisplay();
  }
}
