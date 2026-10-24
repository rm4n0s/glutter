package io.github.rm4n0s.glutter;

import android.app.Activity;
import android.content.pm.ApplicationInfo;
import android.graphics.Color;
import android.os.Build;
import android.util.Log;
import android.view.HapticFeedbackConstants;
import android.view.SoundEffectConstants;
import android.view.View;
import android.view.Window;
import android.view.WindowInsets;
import android.view.WindowInsetsController;
import android.view.WindowManager;

/**
 * The activity's system interface that the framework controls: the bars, the requested orientation,
 * haptic feedback, the sounds of the system and keeping the screen on. Go sends a command for each
 * ({@code systemui.go}); this runs on the main thread.
 *
 * <p>Ports: {@code PlatformPlugin}
 * (shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java), with the platform's
 * {@code WindowInsetsController} where Flutter uses AndroidX's {@code WindowInsetsControllerCompat},
 * and for the modes that hide the bars also on an app that targets API 36 or later, where the
 * system ignores the old {@code SYSTEM_UI_FLAG_*} flags.
 */
final class SystemUI {
  // The operations of GlutterActivity.command (systemui.go).
  static final int OP_HAPTIC = 1;
  static final int OP_SOUND = 2;
  static final int OP_ORIENTATION = 3;
  static final int OP_UI_MODE = 4;
  static final int OP_OVERLAY_STYLE = 5;
  static final int OP_KEEP_SCREEN_ON = 6;
  static final int OP_UI_CHANGE_LISTENER = 7;

  // ui.SystemUIMode.
  private static final int MODE_LEAN_BACK = 1;
  private static final int MODE_IMMERSIVE = 2;
  private static final int MODE_IMMERSIVE_STICKY = 3;
  private static final int MODE_EDGE_TO_EDGE = 4;
  private static final int MODE_MANUAL = 5;

  // The bits of the overlays that stay in the manual mode.
  private static final int OVERLAY_TOP = 1;
  private static final int OVERLAY_BOTTOM = 2;

  /** Flutter's default: the layout does not move when the bars come and go. */
  private static final int DEFAULT_SYSTEM_UI =
      View.SYSTEM_UI_FLAG_LAYOUT_STABLE
          | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN
          | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION;

  private static final String TAG = "GlutterJava";

  private final Activity activity;
  private final boolean debuggable;

  /** The style that was set last, applied again after a mode change. */
  private String lastStyle;

  /** Whether Go listens to the bars appearing and disappearing, and what it heard last. */
  private boolean listening;

  private Boolean lastVisible;

  SystemUI(Activity activity) {
    this.activity = activity;
    this.debuggable = (activity.getApplicationInfo().flags & ApplicationInfo.FLAG_DEBUGGABLE) != 0;
  }

  /** Draws behind the bars and the cutout, as Android 15 requires of apps that target it. */
  void enableEdgeToEdge() {
    Window window = activity.getWindow();
    View decor = window.getDecorView();
    if (Build.VERSION.SDK_INT >= 30) {
      window.setDecorFitsSystemWindows(false);
    } else {
      decor.setSystemUiVisibility(DEFAULT_SYSTEM_UI);
    }
    window.setStatusBarColor(Color.TRANSPARENT);
    window.setNavigationBarColor(Color.TRANSPARENT);
    if (Build.VERSION.SDK_INT >= 28) {
      WindowManager.LayoutParams params = window.getAttributes();
      params.layoutInDisplayCutoutMode =
          Build.VERSION.SDK_INT >= 30
              ? WindowManager.LayoutParams.LAYOUT_IN_DISPLAY_CUTOUT_MODE_ALWAYS
              : WindowManager.LayoutParams.LAYOUT_IN_DISPLAY_CUTOUT_MODE_SHORT_EDGES;
      window.setAttributes(params);
    }
  }

  /** Runs one command of Go (main thread). */
  void command(int op, long a, long b, long c, long d, String s) {
    if (debuggable) {
      Log.i(TAG, "command op=" + op + " a=" + a + " b=" + b + " s=" + s);
    }
    switch (op) {
      case OP_HAPTIC:
        hapticFeedback((int) a);
        break;
      case OP_SOUND:
        if (a == 1) { // click; Flutter plays nothing for the other sounds on Android.
          activity.getWindow().getDecorView().playSoundEffect(SoundEffectConstants.CLICK);
        }
        break;
      case OP_ORIENTATION:
        activity.setRequestedOrientation((int) a);
        break;
      case OP_UI_MODE:
        setUiMode((int) a, (int) b);
        break;
      case OP_OVERLAY_STYLE:
        setOverlayStyle(s);
        break;
      case OP_KEEP_SCREEN_ON:
        if (a != 0) {
          activity.getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        } else {
          activity.getWindow().clearFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        }
        break;
      case OP_UI_CHANGE_LISTENER:
        listening = a != 0;
        lastVisible = null;
        if (Build.VERSION.SDK_INT < 30) {
          listenToVisibility(listening);
        }
        break;
      default:
        break;
    }
  }

  /** ui.HapticFeedbackKind to the constants of the platform, as PlatformPlugin.vibrateHapticFeedback. */
  private void hapticFeedback(int kind) {
    View view = activity.getWindow().getDecorView();
    switch (kind) {
      case 1: // vibrate
        view.performHapticFeedback(HapticFeedbackConstants.LONG_PRESS);
        break;
      case 2: // lightImpact
        view.performHapticFeedback(HapticFeedbackConstants.VIRTUAL_KEY);
        break;
      case 3: // mediumImpact
        view.performHapticFeedback(HapticFeedbackConstants.KEYBOARD_TAP);
        break;
      case 4: // heavyImpact
        view.performHapticFeedback(HapticFeedbackConstants.CONTEXT_CLICK);
        break;
      case 5: // selectionClick
        view.performHapticFeedback(HapticFeedbackConstants.CLOCK_TICK);
        break;
      default:
        break;
    }
  }

  /** SystemChrome.setEnabledSystemUIMode. */
  private void setUiMode(int mode, int overlays) {
    switch (mode) {
      case MODE_EDGE_TO_EDGE:
        enableEdgeToEdge();
        showBars(true, true, false);
        break;
      case MODE_LEAN_BACK:
      case MODE_IMMERSIVE:
      case MODE_IMMERSIVE_STICKY:
        hideBars(mode == MODE_LEAN_BACK ? false : true, mode);
        break;
      case MODE_MANUAL:
        // Everything hidden but the overlays that were asked for; none at all is immersive sticky.
        boolean top = (overlays & OVERLAY_TOP) != 0;
        boolean bottom = (overlays & OVERLAY_BOTTOM) != 0;
        showBars(top, bottom, !top && !bottom);
        break;
      default:
        return;
    }
    if (lastStyle != null) {
      setOverlayStyle(lastStyle);
    }
  }

  /** Hides both bars; they come back by a swipe from the edge ({@code swipe}) or by a tap. */
  @SuppressWarnings("deprecation")
  private void hideBars(boolean swipe, int mode) {
    Window window = activity.getWindow();
    if (Build.VERSION.SDK_INT >= 30) {
      WindowInsetsController c = window.getInsetsController();
      if (c != null) {
        c.setSystemBarsBehavior(
            swipe
                ? WindowInsetsController.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
                : WindowInsetsController.BEHAVIOR_DEFAULT);
        c.hide(WindowInsets.Type.systemBars());
      }
      return;
    }
    int flags =
        View.SYSTEM_UI_FLAG_LAYOUT_STABLE
            | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
            | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN
            | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION
            | View.SYSTEM_UI_FLAG_FULLSCREEN;
    if (mode == MODE_IMMERSIVE) {
      flags |= View.SYSTEM_UI_FLAG_IMMERSIVE;
    } else if (mode == MODE_IMMERSIVE_STICKY) {
      flags |= View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY;
    }
    window.getDecorView().setSystemUiVisibility(flags);
  }

  /** Shows the status bar and the navigation bar as asked and hides the others. */
  @SuppressWarnings("deprecation")
  private void showBars(boolean status, boolean navigation, boolean stickyWhenNone) {
    Window window = activity.getWindow();
    if (Build.VERSION.SDK_INT >= 30) {
      WindowInsetsController c = window.getInsetsController();
      if (c == null) {
        return;
      }
      int show = 0;
      int hide = 0;
      if (status) {
        show |= WindowInsets.Type.statusBars();
      } else {
        hide |= WindowInsets.Type.statusBars();
      }
      if (navigation) {
        show |= WindowInsets.Type.navigationBars();
      } else {
        hide |= WindowInsets.Type.navigationBars();
      }
      c.setSystemBarsBehavior(
          stickyWhenNone || hide != 0
              ? WindowInsetsController.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
              : WindowInsetsController.BEHAVIOR_DEFAULT);
      if (show != 0) {
        c.show(show);
      }
      if (hide != 0) {
        c.hide(hide);
      }
      return;
    }
    // SystemChrome.setEnabledSystemUIOverlays: hide all, then bring back the ones asked for.
    int flags =
        DEFAULT_SYSTEM_UI | View.SYSTEM_UI_FLAG_FULLSCREEN | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION;
    if (stickyWhenNone) {
      flags |= View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY;
    }
    if (status) {
      flags &= ~View.SYSTEM_UI_FLAG_FULLSCREEN;
    }
    if (navigation) {
      flags &= ~View.SYSTEM_UI_FLAG_HIDE_NAVIGATION;
    }
    window.getDecorView().setSystemUiVisibility(flags);
  }

  /**
   * SystemChrome.setSystemUIOverlayStyle: seven fields separated by commas, empty for what the app
   * does not say (encodeOverlayStyle in systemui.go).
   */
  @SuppressWarnings("deprecation")
  private void setOverlayStyle(String style) {
    lastStyle = style;
    String[] f = style.split(",", -1);
    if (f.length != 7) {
      return;
    }
    Window window = activity.getWindow();
    // The icons: 1 light icons (a dark bar), 2 dark icons (a light bar).
    if (!f[1].isEmpty()) {
      setLightAppearance(true, f[1].equals("2"));
    }
    if (!f[0].isEmpty() && Build.VERSION.SDK_INT < 35) {
      window.setStatusBarColor((int) Long.parseLong(f[0]));
    }
    if (!f[2].isEmpty() && Build.VERSION.SDK_INT >= 29) {
      window.setStatusBarContrastEnforced(f[2].equals("1"));
    }
    if (Build.VERSION.SDK_INT >= 26) {
      if (!f[5].isEmpty()) {
        setLightAppearance(false, f[5].equals("2"));
      }
      if (!f[3].isEmpty() && Build.VERSION.SDK_INT < 35) {
        window.setNavigationBarColor((int) Long.parseLong(f[3]));
      }
    }
    if (!f[4].isEmpty() && Build.VERSION.SDK_INT >= 28 && Build.VERSION.SDK_INT < 35) {
      window.setNavigationBarDividerColor((int) Long.parseLong(f[4]));
    }
    if (!f[6].isEmpty() && Build.VERSION.SDK_INT >= 29) {
      window.setNavigationBarContrastEnforced(f[6].equals("1"));
    }
  }

  /** The icons of the status bar ({@code status}) or the navigation bar: dark ones on a light bar. */
  @SuppressWarnings("deprecation")
  private void setLightAppearance(boolean status, boolean light) {
    Window window = activity.getWindow();
    if (Build.VERSION.SDK_INT >= 30) {
      WindowInsetsController c = window.getInsetsController();
      if (c != null) {
        int mask =
            status
                ? WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS
                : WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS;
        c.setSystemBarsAppearance(light ? mask : 0, mask);
      }
      return;
    }
    View decor = window.getDecorView();
    int flag =
        status ? View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR : View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR;
    int visibility = decor.getSystemUiVisibility();
    decor.setSystemUiVisibility(light ? visibility | flag : visibility & ~flag);
  }

  /** Before API 30 the visibility flags of the decor view say whether the bars are shown. */
  @SuppressWarnings("deprecation")
  private void listenToVisibility(boolean on) {
    View decor = activity.getWindow().getDecorView();
    if (!on) {
      decor.setOnSystemUiVisibilityChangeListener(null);
      return;
    }
    decor.setOnSystemUiVisibilityChangeListener(
        new View.OnSystemUiVisibilityChangeListener() {
          @Override
          public void onSystemUiVisibilityChange(int visibility) {
            Native.systemUI((visibility & View.SYSTEM_UI_FLAG_FULLSCREEN) == 0);
          }
        });
  }

  /** Called with the insets of the window: tells Go when the bars appear or disappear. */
  void onInsets(WindowInsets insets) {
    if (!listening || Build.VERSION.SDK_INT < 30) {
      return;
    }
    boolean visible =
        insets.isVisible(WindowInsets.Type.statusBars())
            || insets.isVisible(WindowInsets.Type.navigationBars());
    if (lastVisible == null || lastVisible != visible) {
      boolean first = lastVisible == null;
      lastVisible = visible;
      // The first value is the state at the time of listening, not a change.
      if (!first) {
        Native.systemUI(visible);
      }
    }
  }
}
