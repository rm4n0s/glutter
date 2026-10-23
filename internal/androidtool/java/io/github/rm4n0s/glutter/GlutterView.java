package io.github.rm4n0s.glutter;

import android.content.Context;
import android.os.Build;
import android.util.DisplayMetrics;
import android.view.Display;
import android.view.Surface;
import android.view.SurfaceHolder;
import android.view.SurfaceView;
import android.view.WindowManager;

/**
 * The view of a Glutter app: a SurfaceView that fills the activity's window. It tells Go about the
 * life of its surface and about the display; Go draws into the surface from its own thread.
 *
 * <p>Ports: {@code FlutterView} and {@code FlutterSurfaceView}
 * (shell/platform/android/io/flutter/embedding/android/FlutterView.java and
 * FlutterSurfaceView.java), the parts that step 25.3 needs: the surface's life.
 */
public class GlutterView extends SurfaceView implements SurfaceHolder.Callback {
  public GlutterView(Context context) {
    super(context);
    getHolder().addCallback(this);
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
