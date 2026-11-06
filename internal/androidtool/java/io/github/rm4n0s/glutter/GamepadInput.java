package io.github.rm4n0s.glutter;

import android.content.Context;
import android.hardware.input.InputManager;
import android.os.Build;
import android.os.CombinedVibration;
import android.os.Handler;
import android.os.Looper;
import android.os.VibrationEffect;
import android.os.VibratorManager;
import android.util.Log;
import android.view.InputDevice;
import android.view.KeyEvent;
import android.view.MotionEvent;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

/**
 * The gamepads of the activity: the input devices that are gamepads or joysticks. It tells Go of
 * each one that comes and goes ({@code InputManager.InputDeviceListener}), takes the key events
 * and the joystick motion of those devices instead of the keyboard path, so that each press
 * arrives once, and makes a pad rumble through its {@code VibratorManager} (Android 12 and later).
 *
 * <p>Ports: the Android backend of SDL 3's joystick code (src/joystick/android/SDL_sysjoystick.c
 * and {@code SDLControllerManager.java}), translated and changed: the key codes and axes name the
 * buttons and sticks already, so Go gets them as the standard layout (gamepads.go).
 */
final class GamepadInput implements InputManager.InputDeviceListener {
  /** The operation of {@code GlutterActivity.command} that makes a pad rumble: a is the device id, b and c the low and high strength (0 to 65535), d the milliseconds (0 stops). */
  static final int OP_RUMBLE = 28;

  /** The flag of a pad that can rumble ({@code padFlagRumble} in gamepads.go). */
  private static final int FLAG_RUMBLE = 1;

  private static final String TAG = "GlutterJava";

  /** The axes of a motion event sent to Go, in order (the slots in gamepads.go). */
  private static final int[] AXES = {
    MotionEvent.AXIS_X, MotionEvent.AXIS_Y, MotionEvent.AXIS_Z, MotionEvent.AXIS_RZ,
    MotionEvent.AXIS_RX, MotionEvent.AXIS_RY, MotionEvent.AXIS_LTRIGGER, MotionEvent.AXIS_RTRIGGER,
    MotionEvent.AXIS_BRAKE, MotionEvent.AXIS_GAS, MotionEvent.AXIS_HAT_X, MotionEvent.AXIS_HAT_Y
  };

  private final Context context;
  private final InputManager manager;
  private final Handler handler = new Handler(Looper.getMainLooper());

  /** The devices Go knows as pads. Main thread only. */
  private final Set<Integer> known = new HashSet<>();

  GamepadInput(Context context) {
    this.context = context;
    this.manager = (InputManager) context.getSystemService(Context.INPUT_SERVICE);
  }

  /** Starts to listen for devices and tells Go of those that are there. */
  void start() {
    if (manager == null) {
      return;
    }
    manager.registerInputDeviceListener(this, handler);
    sync();
  }

  /** Stops listening (the activity is going). */
  void stop() {
    if (manager != null) {
      manager.unregisterInputDeviceListener(this);
    }
  }

  /** Tells Go of every pad that is there, and which devices are there at all: a pad that went while there was no activity is dropped. */
  private void sync() {
    List<Integer> present = new ArrayList<>();
    for (int id : manager.getInputDeviceIds()) {
      InputDevice device = InputDevice.getDevice(id);
      if (isPad(device)) {
        add(device);
        present.add(id);
      }
    }
    // The virtual device of the shell's input command is not listed; it stays.
    for (int id : known) {
      if (id < 0 && !present.contains(id)) {
        present.add(id);
      }
    }
    int[] ids = new int[present.size()];
    for (int i = 0; i < ids.length; i++) {
      ids[i] = present.get(i);
    }
    known.retainAll(present);
    Native.padSync(ids);
  }

  /** Whether a device is a gamepad or a joystick: it has those sources and the axes of a stick or a hat, or the A button. */
  static boolean isPad(InputDevice device) {
    if (device == null || device.isVirtual()) {
      return false;
    }
    int sources = device.getSources();
    boolean source =
        (sources & InputDevice.SOURCE_GAMEPAD) == InputDevice.SOURCE_GAMEPAD
            || (sources & InputDevice.SOURCE_JOYSTICK) == InputDevice.SOURCE_JOYSTICK;
    if (!source) {
      return false;
    }
    boolean stick =
        device.getMotionRange(MotionEvent.AXIS_X) != null && device.getMotionRange(MotionEvent.AXIS_Y) != null;
    boolean hat = device.getMotionRange(MotionEvent.AXIS_HAT_X) != null;
    return stick || hat || device.hasKeys(KeyEvent.KEYCODE_BUTTON_A)[0];
  }

  /** Whether a key code is a button of a pad or its D-pad (the table of padButtonOfKey in gamepads.go). */
  static boolean isPadKey(int code) {
    switch (code) {
      case KeyEvent.KEYCODE_BUTTON_A:
      case KeyEvent.KEYCODE_BUTTON_B:
      case KeyEvent.KEYCODE_BUTTON_X:
      case KeyEvent.KEYCODE_BUTTON_Y:
      case KeyEvent.KEYCODE_BUTTON_L1:
      case KeyEvent.KEYCODE_BUTTON_R1:
      case KeyEvent.KEYCODE_BUTTON_L2:
      case KeyEvent.KEYCODE_BUTTON_R2:
      case KeyEvent.KEYCODE_BUTTON_THUMBL:
      case KeyEvent.KEYCODE_BUTTON_THUMBR:
      case KeyEvent.KEYCODE_BUTTON_START:
      case KeyEvent.KEYCODE_BUTTON_SELECT:
      case KeyEvent.KEYCODE_BUTTON_MODE:
      case KeyEvent.KEYCODE_DPAD_UP:
      case KeyEvent.KEYCODE_DPAD_DOWN:
      case KeyEvent.KEYCODE_DPAD_LEFT:
      case KeyEvent.KEYCODE_DPAD_RIGHT:
      case KeyEvent.KEYCODE_DPAD_CENTER:
      case KeyEvent.KEYCODE_MEDIA_RECORD:
        return true;
      default:
        return false;
    }
  }

  private static boolean isPadSource(int source) {
    return (source & InputDevice.SOURCE_GAMEPAD) == InputDevice.SOURCE_GAMEPAD
        || (source & InputDevice.SOURCE_JOYSTICK) == InputDevice.SOURCE_JOYSTICK;
  }

  /** Tells Go of a device that is a pad. */
  private void add(InputDevice device) {
    int id = device.getId();
    if (!known.add(id)) {
      return;
    }
    List<Integer> axes = new ArrayList<>();
    List<Float> ranges = new ArrayList<>();
    for (InputDevice.MotionRange range : device.getMotionRanges()) {
      if ((range.getSource() & InputDevice.SOURCE_JOYSTICK) != InputDevice.SOURCE_JOYSTICK) {
        continue;
      }
      axes.add(range.getAxis());
      ranges.add(range.getMin());
      ranges.add(range.getMax());
    }
    int flags = canRumble(device) ? FLAG_RUMBLE : 0;
    String descriptor = device.getDescriptor();
    Native.padAdded(
        id,
        device.getName().getBytes(StandardCharsets.UTF_8),
        (descriptor == null ? "" : descriptor).getBytes(StandardCharsets.UTF_8),
        device.getVendorId(),
        device.getProductId(),
        flags,
        toInts(axes),
        toFloats(ranges));
  }

  /**
   * Tells Go of the device that a gamepad's key event came from when it is not one the input manager lists: the shell's {@code input
   * gamepad keyevent} has the virtual keyboard's id (-1), and its pad has the axes of a standard one.
   */
  private void addVirtual(int id) {
    if (!known.add(id)) {
      return;
    }
    int[] axes = {
      MotionEvent.AXIS_X, MotionEvent.AXIS_Y, MotionEvent.AXIS_Z, MotionEvent.AXIS_RZ,
      MotionEvent.AXIS_LTRIGGER, MotionEvent.AXIS_RTRIGGER, MotionEvent.AXIS_HAT_X, MotionEvent.AXIS_HAT_Y
    };
    float[] ranges = {-1, 1, -1, 1, -1, 1, -1, 1, 0, 1, 0, 1, -1, 1, -1, 1};
    Native.padAdded(
        id,
        "Virtual Gamepad".getBytes(StandardCharsets.UTF_8),
        ("virtual-" + id).getBytes(StandardCharsets.UTF_8),
        0,
        0,
        0,
        axes,
        ranges);
  }

  private static int[] toInts(List<Integer> list) {
    int[] out = new int[list.size()];
    for (int i = 0; i < out.length; i++) {
      out[i] = list.get(i);
    }
    return out;
  }

  private static float[] toFloats(List<Float> list) {
    float[] out = new float[list.size()];
    for (int i = 0; i < out.length; i++) {
      out[i] = list.get(i);
    }
    return out;
  }

  private static boolean canRumble(InputDevice device) {
    if (Build.VERSION.SDK_INT < 31) {
      return false;
    }
    VibratorManager manager = device.getVibratorManager();
    return manager != null && manager.getVibratorIds().length > 0;
  }

  @Override
  public void onInputDeviceAdded(int deviceId) {
    InputDevice device = InputDevice.getDevice(deviceId);
    if (isPad(device)) {
      add(device);
    }
  }

  @Override
  public void onInputDeviceRemoved(int deviceId) {
    if (known.remove(deviceId)) {
      Native.padRemoved(deviceId);
    }
  }

  @Override
  public void onInputDeviceChanged(int deviceId) {
    // A device whose sources changed may have become a pad, or stopped being one.
    InputDevice device = InputDevice.getDevice(deviceId);
    if (isPad(device)) {
      add(device);
    } else if (known.contains(deviceId) && deviceId >= 0) {
      onInputDeviceRemoved(deviceId);
    }
  }

  /**
   * A key event: a button of a pad (of a device that is a gamepad or joystick) goes to Go as a pad's and is taken; any other key
   * is left to the keyboard path. Main thread.
   */
  boolean onKey(KeyEvent event) {
    if (!isPadSource(event.getSource()) || !isPadKey(event.getKeyCode()) || event.getAction() == KeyEvent.ACTION_MULTIPLE) {
      return false;
    }
    int id = event.getDeviceId();
    if (!known.contains(id)) {
      InputDevice device = event.getDevice();
      if (isPad(device)) {
        add(device);
      } else {
        addVirtual(id);
      }
    }
    Native.padKey(id, event.getAction() == KeyEvent.ACTION_DOWN, event.getKeyCode(), event.getEventTime());
    return true;
  }

  /** The motion of a joystick: the values of the axes go to Go in the order of {@link #AXES}. Main thread. */
  boolean onMotion(MotionEvent event) {
    if ((event.getSource() & InputDevice.SOURCE_JOYSTICK) != InputDevice.SOURCE_JOYSTICK
        || event.getActionMasked() != MotionEvent.ACTION_MOVE) {
      return false;
    }
    int id = event.getDeviceId();
    if (!known.contains(id)) {
      InputDevice device = event.getDevice();
      if (!isPad(device)) {
        return false;
      }
      add(device);
    }
    float[] values = new float[AXES.length];
    for (int i = 0; i < AXES.length; i++) {
      values[i] = event.getAxisValue(AXES[i]);
    }
    Native.padMotion(id, values, event.getEventTime());
    return true;
  }

  /** Makes the pad's vibrators run, or stops them (main thread). */
  void command(int op, long a, long b, long c, long d) {
    if (op != OP_RUMBLE || Build.VERSION.SDK_INT < 31) {
      return;
    }
    try {
      InputDevice device = InputDevice.getDevice((int) a);
      if (device == null) {
        return;
      }
      VibratorManager manager = device.getVibratorManager();
      int[] ids = manager.getVibratorIds();
      if (ids.length == 0) {
        return;
      }
      if (d <= 0) {
        manager.cancel();
        return;
      }
      int low = amplitude(b);
      int high = amplitude(c);
      CombinedVibration.ParallelCombination combination = CombinedVibration.startParallel();
      if (ids.length >= 2) {
        // The first vibrator is the strong (low frequency) motor and the second the weak one.
        if (low > 0) {
          combination.addVibrator(ids[0], VibrationEffect.createOneShot(d, low));
        }
        if (high > 0) {
          combination.addVibrator(ids[1], VibrationEffect.createOneShot(d, high));
        }
        if (low == 0 && high == 0) {
          manager.cancel();
          return;
        }
      } else {
        combination.addVibrator(ids[0], VibrationEffect.createOneShot(d, Math.max(low, high)));
      }
      manager.vibrate(combination.combine());
    } catch (RuntimeException e) {
      Log.w(TAG, "a gamepad could not rumble: " + e);
    }
  }

  /** The amplitude of {@code VibrationEffect} (1 to 255, 0 for none) of a strength from 0 to 65535. */
  private static int amplitude(long strength) {
    if (strength <= 0) {
      return 0;
    }
    return (int) Math.max(1, Math.min(255, strength >> 8));
  }
}
