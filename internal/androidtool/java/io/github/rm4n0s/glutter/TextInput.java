package io.github.rm4n0s.glutter;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.os.Handler;
import android.os.Looper;
import android.os.SystemClock;
import android.text.Html;
import android.text.InputType;
import android.text.TextUtils;
import android.util.Log;
import android.view.KeyEvent;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputConnection;
import android.view.inputmethod.InputMethodManager;
import java.nio.charset.StandardCharsets;

/**
 * Text input, the soft keyboard and the clipboard of the activity. Go sends a command when a text
 * field gets or loses the focus and when the framework changes the editing state (textinput.go);
 * the input method's edits go back to Go as {@code Native.textEdit}. Everything runs on the main
 * thread.
 *
 * <p>The editing state is a copy ({@link ListenableEditingState}) that the input method reads and
 * edits at once. Each edit that is sent to Go carries a sequence number; a state from the
 * framework is applied only if it answers the last edit that was sent, because after a newer edit
 * of the input method it is stale.
 *
 * <p>Ports: {@code TextInputPlugin}
 * (shell/platform/android/io/flutter/plugin/editing/TextInputPlugin.java: the input type and the
 * options of the field, showing and hiding the keyboard, {@code createInputConnection}) and the
 * clipboard methods of {@code PlatformPlugin}
 * (shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java: {@code
 * getClipboardData}, {@code setClipboardData}).
 */
final class TextInput implements ListenableEditingState.EditingStateWatcher {
  // The operations of GlutterActivity.command (textinput.go).
  static final int OP_TEXT_START = 8;
  static final int OP_TEXT_STATE = 9;
  static final int OP_TEXT_STOP = 10;
  static final int OP_CLIPBOARD_SET = 12;
  static final int OP_CLIPBOARD_GET = 13;
  static final int OP_CLIPBOARD_SET_HTML = 25;
  static final int OP_CLIPBOARD_GET_HTML = 26;

  private static final String TAG = "GlutterJava";

  /** The pause between the pieces of a composition of the dev events script. */
  private static final long COMPOSE_STEP_MS = 60;

  private final GlutterView view;
  private final Context context;
  private final InputMethodManager imm;
  private final ListenableEditingState editable;
  private final Handler handler = new Handler(Looper.getMainLooper());

  /** The number of the last edit sent to Go (per process); 0 before any. */
  private static int seq;

  /** Whether a text field has the focus, and its configuration. */
  private boolean active;

  private int inputType;
  private int imeOptions;

  /** The connection that the input method has now, or null. */
  private GlutterInputConnection connection;

  TextInput(GlutterView view) {
    this.view = view;
    this.context = view.getContext();
    this.imm = (InputMethodManager) context.getSystemService(Context.INPUT_METHOD_SERVICE);
    this.editable = new ListenableEditingState(view);
    editable.addEditingStateListener(this);
  }

  /** Whether a text field has the focus: the view is a text editor then. */
  boolean isActive() {
    return active;
  }

  /** Runs one command of Go (main thread). */
  void command(int op, long a, long b, long c, long d, String s) {
    switch (op) {
      case OP_TEXT_START:
        start((int) a, (int) b);
        break;
      case OP_TEXT_STATE:
        applyState((int) a, (int) b, (int) c, s);
        break;
      case OP_TEXT_STOP:
        stop();
        break;
      case OP_CLIPBOARD_SET:
        setClipboard(s);
        break;
      case OP_CLIPBOARD_GET:
        getClipboard((int) a);
        break;
      case OP_CLIPBOARD_SET_HTML:
        setClipboardHtml(s, (int) a, b != 0);
        break;
      case OP_CLIPBOARD_GET_HTML:
        getClipboardHtml((int) a);
        break;
      default:
        Log.w(TAG, "unknown text input command " + op);
    }
  }

  // A text field got the focus, or its configuration changed.
  private void start(int inputType, int imeOptions) {
    this.active = true;
    this.inputType = inputType;
    this.imeOptions = imeOptions;
    // The framework's state follows; until then the field is empty.
    editable.setEditingState("", 0, 0);
    view.requestFocus();
    imm.restartInput(view);
    imm.showSoftInput(view, 0);
  }

  // No text field has the focus.
  private void stop() {
    active = false;
    connection = null;
    editable.setEditingState("", 0, 0);
    imm.hideSoftInputFromWindow(view.getApplicationWindowToken(), 0);
  }

  // The framework's editing state. It is stale if the input method has edited since the edit it
  // answers: then it is dropped, and the framework gets the newer edit.
  private void applyState(int answered, int base, int extent, String text) {
    if (!active || answered != seq) {
      return;
    }
    if (editable.changedInBatchEdit()) {
      // The input method is in the middle of a batch of edits that is not sent yet.
      return;
    }
    editable.setEditingState(text, base, extent);
    // The composing region is gone, the selection is the framework's.
    imm.updateSelection(view, editable.getSelectionStart(), editable.getSelectionEnd(), -1, -1);
  }

  /**
   * The input connection for the input method, or null when no text field has the focus. Fills in
   * what the input method needs to know about the field.
   *
   * <p>Ports: {@code TextInputPlugin.createInputConnection}.
   */
  InputConnection createInputConnection(EditorInfo outAttrs) {
    if (!active) {
      return null;
    }
    outAttrs.inputType = inputType;
    outAttrs.imeOptions =
        imeOptions | EditorInfo.IME_FLAG_NO_FULLSCREEN | EditorInfo.IME_FLAG_NO_EXTRACT_UI;
    outAttrs.initialSelStart = editable.getSelectionStart();
    outAttrs.initialSelEnd = editable.getSelectionEnd();
    int caps =
        InputType.TYPE_TEXT_FLAG_CAP_CHARACTERS
            | InputType.TYPE_TEXT_FLAG_CAP_WORDS
            | InputType.TYPE_TEXT_FLAG_CAP_SENTENCES;
    if ((inputType & caps) != 0 && outAttrs.initialSelStart >= 0) {
      outAttrs.initialCapsMode = TextUtils.getCapsMode(editable, outAttrs.initialSelStart, inputType);
    }
    if (android.os.Build.VERSION.SDK_INT >= 30) {
      outAttrs.setInitialSurroundingText(editable.toString());
    }
    connection = new GlutterInputConnection(view, editable);
    return connection;
  }

  // The input method edited the state (a batch of edits ended): Go hears of it.
  @Override
  public void didChangeEditingState(
      boolean textChanged, boolean selectionChanged, boolean composingRegionChanged) {
    if (!active) {
      return;
    }
    byte[] text = editable.toString().getBytes(StandardCharsets.UTF_8);
    seq++;
    Native.textEdit(
        seq,
        text,
        editable.getSelectionStart(),
        editable.getSelectionEnd(),
        editable.getComposingStart(),
        editable.getComposingEnd());
  }

  // -------- Clipboard --------

  /** Ports: {@code PlatformPlugin.setClipboardData}. */
  private void setClipboard(String text) {
    ClipboardManager clipboard = (ClipboardManager) context.getSystemService(Context.CLIPBOARD_SERVICE);
    try {
      clipboard.setPrimaryClip(ClipData.newPlainText("text", text));
    } catch (RuntimeException e) {
      Log.w(TAG, "the clipboard could not be set", e);
    }
  }

  /** Ports: {@code PlatformPlugin.getClipboardData} for the plain text format. */
  private void getClipboard(int id) {
    String text = null;
    try {
      ClipboardManager clipboard = (ClipboardManager) context.getSystemService(Context.CLIPBOARD_SERVICE);
      if (clipboard.hasPrimaryClip()) {
        ClipData clip = clipboard.getPrimaryClip();
        if (clip != null && clip.getItemCount() > 0) {
          CharSequence item = clip.getItemAt(0).coerceToText(context);
          if (item != null && item.length() > 0) {
            text = item.toString();
          }
        }
      }
    } catch (RuntimeException e) {
      // For instance a SecurityException when the content needs a permission.
      Log.w(TAG, "the clipboard could not be read", e);
    }
    if (text == null) {
      Native.clipboardText(id, false, new byte[0]);
    } else {
      Native.clipboardText(id, true, text.getBytes(StandardCharsets.UTF_8));
    }
  }

  /**
   * Puts the HTML next to the text on the clipboard, as one clip ({@code ClipData.newHtmlText}).
   * {@code both} is the HTML followed by the text, {@code htmlLength} the length of the HTML in
   * UTF-16 units (so that either can hold any character), {@code hasText} whether there is a
   * text. A clip needs a plain text for the programs that paste no HTML, so without one it is the
   * text of the HTML ({@code Html.fromHtml}). Ports: {@code PlatformPlugin.setClipboardData}.
   */
  private void setClipboardHtml(String both, int htmlLength, boolean hasText) {
    String html = both.substring(0, htmlLength);
    String text =
        hasText
            ? both.substring(htmlLength)
            : Html.fromHtml(html, Html.FROM_HTML_MODE_COMPACT).toString();
    ClipboardManager clipboard = (ClipboardManager) context.getSystemService(Context.CLIPBOARD_SERVICE);
    try {
      clipboard.setPrimaryClip(ClipData.newHtmlText("text", text, html));
    } catch (RuntimeException e) {
      Log.w(TAG, "the clipboard could not be set", e);
    }
  }

  /**
   * Reads the HTML of the clipboard: the answer is {@code Native.clipboardText} with the HTML as
   * the text. Ports: {@code PlatformPlugin.getClipboardData}.
   */
  private void getClipboardHtml(int id) {
    String html = null;
    try {
      ClipboardManager clipboard = (ClipboardManager) context.getSystemService(Context.CLIPBOARD_SERVICE);
      ClipData clip = clipboard.hasPrimaryClip() ? clipboard.getPrimaryClip() : null;
      if (clip != null && clip.getDescription().hasMimeType("text/html")) {
        for (int i = 0; i < clip.getItemCount() && html == null; i++) {
          String item = clip.getItemAt(i).getHtmlText();
          if (item != null && item.length() > 0) {
            html = item;
          }
        }
      }
    } catch (RuntimeException e) {
      // For instance a SecurityException when the content needs a permission.
      Log.w(TAG, "the clipboard could not be read", e);
    }
    if (html == null) {
      Native.clipboardText(id, false, new byte[0]);
    } else {
      Native.clipboardText(id, true, html.getBytes(StandardCharsets.UTF_8));
    }
  }

  // -------- The dev events script (debug builds) --------

  /** Commits {@code text} through the input connection as an input method does. */
  void injectText(String text) {
    GlutterInputConnection ic = connection;
    if (!active || ic == null) {
      Log.w(TAG, "text injected but no text field has an input connection");
      return;
    }
    ic.beginBatchEdit();
    ic.commitText(text, 1);
    ic.endBatchEdit();
  }

  /**
   * Does what the keyboard's Enter key does: the field's action, or, when the field has none (a
   * multiline field), an Enter key through the connection, as Gboard does.
   */
  void injectAction() {
    GlutterInputConnection ic = connection;
    if (!active || ic == null) {
      Log.w(TAG, "action injected but no text field has an input connection");
      return;
    }
    int action = imeOptions & EditorInfo.IME_MASK_ACTION;
    if (action == EditorInfo.IME_ACTION_NONE || action == EditorInfo.IME_ACTION_UNSPECIFIED) {
      long now = SystemClock.uptimeMillis();
      ic.sendKeyEvent(new KeyEvent(now, now, KeyEvent.ACTION_DOWN, KeyEvent.KEYCODE_ENTER, 0));
      ic.sendKeyEvent(new KeyEvent(now, now, KeyEvent.ACTION_UP, KeyEvent.KEYCODE_ENTER, 0));
    } else {
      ic.performEditorAction(action);
    }
  }

  /**
   * Composes the comma separated pieces one after the other, 60 ms apart, and commits the last.
   */
  void injectCompose(String pieces) {
    final GlutterInputConnection ic = connection;
    if (!active || ic == null) {
      Log.w(TAG, "composition injected but no text field has an input connection");
      return;
    }
    final String[] parts = pieces.split(",", -1);
    for (int i = 0; i < parts.length; i++) {
      final String piece = parts[i];
      final boolean last = i == parts.length - 1;
      handler.postDelayed(
          new Runnable() {
            @Override
            public void run() {
              if (connection != ic) {
                return;
              }
              ic.setComposingText(piece, 1);
              if (last) {
                ic.finishComposingText();
              }
            }
          },
          COMPOSE_STEP_MS * i);
    }
  }
}
