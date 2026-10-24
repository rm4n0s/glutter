package io.github.rm4n0s.glutter;

import android.app.UiModeManager;
import android.content.Context;
import android.content.res.Configuration;
import android.database.ContentObserver;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.os.LocaleList;
import android.provider.Settings;
import android.text.format.DateFormat;
import android.util.DisplayMetrics;
import android.util.TypedValue;
import java.util.Locale;

/**
 * Reads the system's settings that the app follows and tells Go when they change: the night mode,
 * the font scale, the languages, the 24-hour clock, animations, bold text and contrast. Go gets a
 * text of {@code key=value} pairs ({@code settings.go}).
 *
 * <p>Ports: {@code FlutterView.sendUserSettingsToFlutter}
 * (shell/platform/android/io/flutter/embedding/android/FlutterView.java), {@code
 * SettingsChannel}, {@code LocalizationPlugin.sendLocalesToFlutter} and the accessibility features
 * of {@code AccessibilityBridge} (animations, bold text, contrast).
 */
final class SystemSettings {
  /** Text scaling that is not linear came with Android 14. */
  private static final int NONLINEAR_SINCE = 34;

  /** FontWeight.bold - FontWeight.normal (AccessibilityBridge.BOLD_TEXT_WEIGHT_ADJUSTMENT). */
  private static final int BOLD_TEXT_WEIGHT_ADJUSTMENT = 300;

  private SystemSettings() {}

  /** The settings as the text that Go reads. */
  static String describe(Context context) {
    Configuration c = context.getResources().getConfiguration();
    StringBuilder b = new StringBuilder();
    boolean night =
        (c.uiMode & Configuration.UI_MODE_NIGHT_MASK) == Configuration.UI_MODE_NIGHT_YES;
    b.append("night=").append(night ? 1 : 0);
    b.append(";font=").append(c.fontScale);
    b.append(";nonlinear=").append(Build.VERSION.SDK_INT >= NONLINEAR_SINCE ? 1 : 0);
    b.append(";locales=").append(locales(c));
    b.append(";h24=").append(DateFormat.is24HourFormat(context) ? 1 : 0);
    b.append(";noanim=").append(animationsOff(context) ? 1 : 0);
    b.append(";bold=").append(bold(c) ? 1 : 0);
    b.append(";contrast=").append(highContrast(context) ? 1 : 0);
    return b.toString();
  }

  /** The user's languages, most preferred first, as BCP 47 tags separated by commas. */
  private static String locales(Configuration c) {
    StringBuilder b = new StringBuilder();
    if (Build.VERSION.SDK_INT >= 24) {
      LocaleList list = c.getLocales();
      for (int i = 0; i < list.size(); i++) {
        if (i > 0) {
          b.append(',');
        }
        b.append(list.get(i).toLanguageTag());
      }
    } else {
      b.append(c.locale.toLanguageTag());
    }
    return b.toString();
  }

  /**
   * Animations are off when the animator duration scale or the transition animation scale is 0
   * (what "remove animations" and the developer option set). Flutter looks at the transition scale
   * only.
   */
  private static boolean animationsOff(Context context) {
    android.content.ContentResolver r = context.getContentResolver();
    return Settings.Global.getFloat(r, Settings.Global.ANIMATOR_DURATION_SCALE, 1f) == 0f
        || Settings.Global.getFloat(r, Settings.Global.TRANSITION_ANIMATION_SCALE, 1f) == 0f;
  }

  /** Bold text of the system's accessibility settings (API 31 and later). */
  private static boolean bold(Configuration c) {
    return Build.VERSION.SDK_INT >= 31
        && c.fontWeightAdjustment != Configuration.FONT_WEIGHT_ADJUSTMENT_UNDEFINED
        && c.fontWeightAdjustment >= BOLD_TEXT_WEIGHT_ADJUSTMENT;
  }

  /** Any contrast above the standard one of the system's contrast setting (API 34 and later). */
  private static boolean highContrast(Context context) {
    if (Build.VERSION.SDK_INT < 34) {
      return false;
    }
    UiModeManager m = (UiModeManager) context.getSystemService(Context.UI_MODE_SERVICE);
    return m != null && m.getContrast() > 0f;
  }

  /**
   * What the system makes of a font size of {@code size} logical pixels: the {@code sp} unit with the
   * scale of the system, as logical pixels. It is not linear on Android 14 and later.
   *
   * <p>Ports: {@code FlutterJNI.getScaledFontSize}
   * (shell/platform/android/io/flutter/embedding/engine/FlutterJNI.java).
   */
  static float scaledFontSize(Context context, float size) {
    DisplayMetrics metrics = context.getResources().getDisplayMetrics();
    return TypedValue.applyDimension(TypedValue.COMPLEX_UNIT_SP, size, metrics) / metrics.density;
  }

  /** Sends the settings to Go now and whenever a setting that is not in the configuration changes. */
  static void watch(final Context context) {
    final Handler main = new Handler(Looper.getMainLooper());
    ContentObserver observer =
        new ContentObserver(main) {
          @Override
          public void onChange(boolean selfChange) {
            Native.settings(describe(context));
          }
        };
    android.content.ContentResolver r = context.getContentResolver();
    r.registerContentObserver(Settings.Global.getUriFor(Settings.Global.ANIMATOR_DURATION_SCALE), false, observer);
    r.registerContentObserver(Settings.Global.getUriFor(Settings.Global.TRANSITION_ANIMATION_SCALE), false, observer);
    r.registerContentObserver(Settings.System.getUriFor(Settings.System.TIME_12_24), false, observer);
    if (Build.VERSION.SDK_INT >= 34) {
      UiModeManager m = (UiModeManager) context.getSystemService(Context.UI_MODE_SERVICE);
      if (m != null) {
        m.addContrastChangeListener(
            context.getMainExecutor(),
            new UiModeManager.ContrastChangeListener() {
              @Override
              public void onContrastChanged(float contrast) {
                Native.settings(describe(context));
              }
            });
      }
    }
  }
}
