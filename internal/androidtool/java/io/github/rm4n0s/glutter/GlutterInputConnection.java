package io.github.rm4n0s.glutter;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.text.Editable;
import android.text.Selection;
import android.view.KeyEvent;
import android.view.View;
import android.view.inputmethod.BaseInputConnection;
import android.view.inputmethod.CursorAnchorInfo;
import android.view.inputmethod.ExtractedText;
import android.view.inputmethod.ExtractedTextRequest;
import android.view.inputmethod.InputMethodManager;

/**
 * The connection between the input method and the editing state of the focused text field. The
 * input method's edits are applied to the {@link ListenableEditingState}, which tells {@link
 * TextInput} (and this connection, for the input method) when the outermost batch edit ends.
 *
 * <p>Ports: {@code InputConnectionAdaptor}
 * (shell/platform/android/io/flutter/plugin/editing/InputConnectionAdaptor.java). Differences: the
 * editor action and the keys are not sent to the framework through a channel but as key events to
 * the view, whose key path handles Backspace, Enter and the arrows in the framework, so the
 * adaptor's own handling of those keys ({@code handleKeyEvent}) is not ported; content committed
 * by the keyboard and private commands are not in 1.6.
 */
final class GlutterInputConnection extends BaseInputConnection
    implements ListenableEditingState.EditingStateWatcher {
  private final View view;
  private final ListenableEditingState editable;
  private final InputMethodManager imm;

  private ExtractedTextRequest extractRequest;
  private boolean monitorCursorUpdate;
  private CursorAnchorInfo.Builder cursorAnchorInfoBuilder;
  private final ExtractedText extractedText = new ExtractedText();
  private int batchEditNestDepth;

  GlutterInputConnection(View view, ListenableEditingState editable) {
    super(view, true);
    this.view = view;
    this.editable = editable;
    this.imm = (InputMethodManager) view.getContext().getSystemService(Context.INPUT_METHOD_SERVICE);
    editable.addEditingStateListener(this);
  }

  private ExtractedText getExtractedText(ExtractedTextRequest request) {
    extractedText.startOffset = 0;
    extractedText.partialStartOffset = -1;
    extractedText.partialEndOffset = -1;
    extractedText.selectionStart = editable.getSelectionStart();
    extractedText.selectionEnd = editable.getSelectionEnd();
    extractedText.text =
        request == null || (request.flags & GET_TEXT_WITH_STYLES) == 0 ? editable.toString() : editable;
    return extractedText;
  }

  private CursorAnchorInfo getCursorAnchorInfo() {
    if (cursorAnchorInfoBuilder == null) {
      cursorAnchorInfoBuilder = new CursorAnchorInfo.Builder();
    } else {
      cursorAnchorInfoBuilder.reset();
    }
    cursorAnchorInfoBuilder.setSelectionRange(editable.getSelectionStart(), editable.getSelectionEnd());
    final int composingStart = editable.getComposingStart();
    final int composingEnd = editable.getComposingEnd();
    if (composingStart >= 0 && composingEnd > composingStart) {
      cursorAnchorInfoBuilder.setComposingText(
          composingStart, editable.toString().subSequence(composingStart, composingEnd));
    } else {
      cursorAnchorInfoBuilder.setComposingText(-1, "");
    }
    return cursorAnchorInfoBuilder.build();
  }

  @Override
  public Editable getEditable() {
    return editable;
  }

  @Override
  public boolean beginBatchEdit() {
    editable.beginBatchEdit();
    batchEditNestDepth += 1;
    return super.beginBatchEdit();
  }

  @Override
  public boolean endBatchEdit() {
    boolean result = super.endBatchEdit();
    batchEditNestDepth -= 1;
    editable.endBatchEdit();
    return result;
  }

  // The edits below are batch edits, so that the framework hears of one change for each call even
  // when the base class changes the text and the spans in several steps.

  @Override
  public boolean commitText(CharSequence text, int newCursorPosition) {
    beginBatchEdit();
    boolean result = super.commitText(text, newCursorPosition);
    endBatchEdit();
    return result;
  }

  @Override
  public boolean deleteSurroundingText(int beforeLength, int afterLength) {
    if (editable.getSelectionStart() == -1) {
      return true;
    }
    beginBatchEdit();
    boolean result = super.deleteSurroundingText(beforeLength, afterLength);
    endBatchEdit();
    return result;
  }

  @Override
  public boolean deleteSurroundingTextInCodePoints(int beforeLength, int afterLength) {
    beginBatchEdit();
    boolean result = super.deleteSurroundingTextInCodePoints(beforeLength, afterLength);
    endBatchEdit();
    return result;
  }

  @Override
  public boolean setComposingRegion(int start, int end) {
    beginBatchEdit();
    boolean result = super.setComposingRegion(start, end);
    endBatchEdit();
    return result;
  }

  @Override
  public boolean setComposingText(CharSequence text, int newCursorPosition) {
    boolean result;
    beginBatchEdit();
    if (text.length() == 0) {
      result = super.commitText(text, newCursorPosition);
    } else {
      result = super.setComposingText(text, newCursorPosition);
    }
    endBatchEdit();
    return result;
  }

  @Override
  public boolean finishComposingText() {
    beginBatchEdit();
    boolean result = super.finishComposingText();
    endBatchEdit();
    return result;
  }

  @Override
  public boolean setSelection(int start, int end) {
    beginBatchEdit();
    boolean result = super.setSelection(start, end);
    endBatchEdit();
    return result;
  }

  // When there's not enough vertical screen space, the IME may enter fullscreen mode and this
  // method will be used to get (a portion of) the currently edited text.
  @Override
  public ExtractedText getExtractedText(ExtractedTextRequest request, int flags) {
    final boolean textMonitor = (flags & GET_EXTRACTED_TEXT_MONITOR) != 0;
    // Enables text monitoring if the relevant flag is set (see didChangeEditingState).
    extractRequest = textMonitor ? request : null;
    return getExtractedText(request);
  }

  @Override
  public boolean requestCursorUpdates(int cursorUpdateMode) {
    if ((cursorUpdateMode & CURSOR_UPDATE_IMMEDIATE) != 0) {
      imm.updateCursorAnchorInfo(view, getCursorAnchorInfo());
    }
    // Enables cursor monitoring (see didChangeEditingState).
    monitorCursorUpdate = (cursorUpdateMode & CURSOR_UPDATE_MONITOR) != 0;
    return true;
  }

  @Override
  public void closeConnection() {
    super.closeConnection();
    editable.removeEditingStateListener(this);
    for (; batchEditNestDepth > 0; batchEditNestDepth--) {
      editable.endBatchEdit();
    }
  }

  /**
   * The keys of the soft keyboard (Backspace, Enter, the arrows) are dispatched to the view like
   * the keys of a hardware keyboard; the framework handles them.
   */
  @Override
  public boolean sendKeyEvent(KeyEvent event) {
    if (event.getKeyCode() == KeyEvent.KEYCODE_ENTER && event.getAction() == KeyEvent.ACTION_DOWN) {
      // The word that the keyboard composes is committed before the key: the framework puts a new line
      // or does the action at the caret, and the composing text is gone by then.
      finishComposingText();
    }
    return super.sendKeyEvent(event);
  }

  /**
   * The action key of the keyboard: the framework turns the Enter key of a field into the field's
   * action, so the connection sends it as Enter.
   */
  @Override
  public boolean performEditorAction(int actionCode) {
    finishComposingText();
    long now = android.os.SystemClock.uptimeMillis();
    view.dispatchKeyEvent(new KeyEvent(now, now, KeyEvent.ACTION_DOWN, KeyEvent.KEYCODE_ENTER, 0));
    view.dispatchKeyEvent(new KeyEvent(now, now, KeyEvent.ACTION_UP, KeyEvent.KEYCODE_ENTER, 0));
    return true;
  }

  @Override
  public boolean performContextMenuAction(int id) {
    beginBatchEdit();
    final boolean result = doPerformContextMenuAction(id);
    endBatchEdit();
    return result;
  }

  private boolean doPerformContextMenuAction(int id) {
    if (id == android.R.id.selectAll) {
      setSelection(0, editable.length());
      return true;
    } else if (id == android.R.id.cut) {
      int selStart = Selection.getSelectionStart(editable);
      int selEnd = Selection.getSelectionEnd(editable);
      if (selStart != selEnd) {
        int selMin = Math.min(selStart, selEnd);
        int selMax = Math.max(selStart, selEnd);
        CharSequence textToCut = editable.subSequence(selMin, selMax);
        clipboard().setPrimaryClip(ClipData.newPlainText("text", textToCut));
        editable.delete(selMin, selMax);
        setSelection(selMin, selMin);
      }
      return true;
    } else if (id == android.R.id.copy) {
      int selStart = Selection.getSelectionStart(editable);
      int selEnd = Selection.getSelectionEnd(editable);
      if (selStart != selEnd) {
        CharSequence textToCopy =
            editable.subSequence(Math.min(selStart, selEnd), Math.max(selStart, selEnd));
        clipboard().setPrimaryClip(ClipData.newPlainText("text", textToCopy));
      }
      return true;
    } else if (id == android.R.id.paste) {
      ClipData clip = clipboard().getPrimaryClip();
      if (clip != null && clip.getItemCount() > 0) {
        CharSequence textToPaste = clip.getItemAt(0).coerceToText(view.getContext());
        int selStart = Math.max(0, Selection.getSelectionStart(editable));
        int selEnd = Math.max(0, Selection.getSelectionEnd(editable));
        int selMin = Math.min(selStart, selEnd);
        int selMax = Math.max(selStart, selEnd);
        if (selMin != selMax) {
          editable.delete(selMin, selMax);
        }
        editable.insert(selMin, textToPaste);
        int newSelStart = selMin + textToPaste.length();
        setSelection(newSelStart, newSelStart);
      }
      return true;
    }
    return false;
  }

  private ClipboardManager clipboard() {
    return (ClipboardManager) view.getContext().getSystemService(Context.CLIPBOARD_SERVICE);
  }

  // -------- ListenableEditingState watcher --------
  @Override
  public void didChangeEditingState(
      boolean textChanged, boolean selectionChanged, boolean composingRegionChanged) {
    // Tells the input method that the state changed. updateSelection is mandatory (it skips the
    // message when nothing changed since the last call); the extracted text and the cursor anchor
    // are on demand, if the input method asked for them.
    imm.updateSelection(
        view,
        editable.getSelectionStart(),
        editable.getSelectionEnd(),
        editable.getComposingStart(),
        editable.getComposingEnd());
    if (extractRequest != null) {
      imm.updateExtractedText(view, extractRequest.token, getExtractedText(extractRequest));
    }
    if (monitorCursorUpdate) {
      imm.updateCursorAnchorInfo(view, getCursorAnchorInfo());
    }
  }
}
