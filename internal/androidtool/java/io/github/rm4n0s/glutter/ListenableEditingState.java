package io.github.rm4n0s.glutter;

import android.text.Editable;
import android.text.Selection;
import android.text.SpannableStringBuilder;
import android.util.Log;
import android.view.View;
import android.view.inputmethod.BaseInputConnection;
import java.util.ArrayList;

/**
 * The editing state of the text field that has the focus (text, selection and composing region),
 * kept as an {@code Editable} that the input method edits. It tells its watchers when the state
 * changes; while a batch edit is open the notification waits for the outermost {@code
 * endBatchEdit}.
 *
 * <p>Ports: {@code ListenableEditingState}
 * (shell/platform/android/io/flutter/plugin/editing/ListenableEditingState.java), without the
 * text editing deltas, which the framework of Glutter does not take (SPEC §24.6). A state that
 * comes from the framework is applied with {@link #setEditingState}, which notifies nobody.
 */
final class ListenableEditingState extends SpannableStringBuilder {
  /** Hears of the changes of the editing state (main thread). */
  interface EditingStateWatcher {
    void didChangeEditingState(
        boolean textChanged, boolean selectionChanged, boolean composingRegionChanged);
  }

  private static final String TAG = "GlutterJava";

  private int batchEditNestDepth = 0;
  private final ArrayList<EditingStateWatcher> listeners = new ArrayList<>();

  /** True while the framework's state is being applied: nobody is told. */
  private boolean muted;

  private String textWhenBeginBatchEdit = "";
  private int selectionStartWhenBeginBatchEdit = -1;
  private int selectionEndWhenBeginBatchEdit = -1;
  private int composingStartWhenBeginBatchEdit = -1;
  private int composingEndWhenBeginBatchEdit = -1;

  /** Only for {@code setComposingRegion}: a connection over this editable. */
  private final BaseInputConnection dummyConnection;

  ListenableEditingState(View view) {
    super();
    final Editable self = this;
    dummyConnection =
        new BaseInputConnection(view, true) {
          @Override
          public Editable getEditable() {
            return self;
          }
        };
  }

  /**
   * Starts a batch edit, during which change notifications wait until all batch edits end. Batch
   * edits nest.
   */
  void beginBatchEdit() {
    batchEditNestDepth++;
    if (batchEditNestDepth == 1) {
      saveStateOfBatchStart();
    }
  }

  /** Ends the current batch edit, and tells the watchers if it was the outermost one and the state changed. */
  void endBatchEdit() {
    if (batchEditNestDepth == 0) {
      Log.e(TAG, "endBatchEdit called without a matching beginBatchEdit");
      return;
    }
    if (batchEditNestDepth == 1) {
      boolean textChanged = !toString().equals(textWhenBeginBatchEdit);
      boolean selectionChanged =
          selectionStartWhenBeginBatchEdit != getSelectionStart()
              || selectionEndWhenBeginBatchEdit != getSelectionEnd();
      boolean composingRegionChanged =
          composingStartWhenBeginBatchEdit != getComposingStart()
              || composingEndWhenBeginBatchEdit != getComposingEnd();
      // The batch is over before the watchers run, so that they see a state without a batch.
      batchEditNestDepth = 0;
      notifyListenersIfNeeded(textChanged, selectionChanged, composingRegionChanged);
      return;
    }
    batchEditNestDepth--;
  }

  /** Whether a batch edit is open. */
  boolean inBatchEdit() {
    return batchEditNestDepth > 0;
  }

  /** Whether the state is not what it was when the outermost batch edit began. */
  boolean changedInBatchEdit() {
    return batchEditNestDepth > 0
        && (!toString().equals(textWhenBeginBatchEdit)
            || selectionStartWhenBeginBatchEdit != getSelectionStart()
            || selectionEndWhenBeginBatchEdit != getSelectionEnd()
            || composingStartWhenBeginBatchEdit != getComposingStart()
            || composingEndWhenBeginBatchEdit != getComposingEnd());
  }

  private void saveStateOfBatchStart() {
    textWhenBeginBatchEdit = toString();
    selectionStartWhenBeginBatchEdit = getSelectionStart();
    selectionEndWhenBeginBatchEdit = getSelectionEnd();
    composingStartWhenBeginBatchEdit = getComposingStart();
    composingEndWhenBeginBatchEdit = getComposingEnd();
  }

  /**
   * Updates the composing region. If the range is invalid or empty, the composing region is
   * removed.
   */
  void setComposingRange(int composingStart, int composingEnd) {
    if (composingStart < 0 || composingStart >= composingEnd) {
      BaseInputConnection.removeComposingSpans(this);
    } else {
      dummyConnection.setComposingRegion(composingStart, composingEnd);
    }
  }

  /**
   * Replaces the whole state, with the framework's (text, selection from {@code base} to {@code
   * extent}, no composing region), without telling the watchers. UTF-16 offsets.
   */
  void setEditingState(String text, int base, int extent) {
    muted = true;
    try {
      replace(0, length(), text);
      BaseInputConnection.removeComposingSpans(this);
      if (base < 0 || extent < 0) {
        Selection.removeSelection(this);
      } else {
        Selection.setSelection(this, Math.min(base, length()), Math.min(extent, length()));
      }
      if (batchEditNestDepth > 0) {
        // What the batch compares with is this state now: the change is not the input method's.
        saveStateOfBatchStart();
      }
    } finally {
      muted = false;
    }
  }

  void addEditingStateListener(EditingStateWatcher listener) {
    listeners.add(listener);
  }

  void removeEditingStateListener(EditingStateWatcher listener) {
    listeners.remove(listener);
  }

  @Override
  public SpannableStringBuilder replace(int start, int end, CharSequence tb, int tbstart, int tbend) {
    boolean textChanged = end - start != tbend - tbstart;
    for (int i = 0; i < end - start && !textChanged; i++) {
      textChanged |= charAt(start + i) != tb.charAt(tbstart + i);
    }

    final int selectionStart = getSelectionStart();
    final int selectionEnd = getSelectionEnd();
    final int composingStart = getComposingStart();
    final int composingEnd = getComposingEnd();

    final SpannableStringBuilder editable = super.replace(start, end, tb, tbstart, tbend);

    if (batchEditNestDepth > 0) {
      return editable;
    }

    final boolean selectionChanged =
        getSelectionStart() != selectionStart || getSelectionEnd() != selectionEnd;
    final boolean composingRegionChanged =
        getComposingStart() != composingStart || getComposingEnd() != composingEnd;
    notifyListenersIfNeeded(textChanged, selectionChanged, composingRegionChanged);
    return editable;
  }

  private void notifyListenersIfNeeded(
      boolean textChanged, boolean selectionChanged, boolean composingChanged) {
    if (muted || !(textChanged || selectionChanged || composingChanged)) {
      return;
    }
    // A copy: a watcher may add or remove watchers.
    for (EditingStateWatcher listener : new ArrayList<>(listeners)) {
      listener.didChangeEditingState(textChanged, selectionChanged, composingChanged);
    }
  }

  final int getSelectionStart() {
    return Selection.getSelectionStart(this);
  }

  final int getSelectionEnd() {
    return Selection.getSelectionEnd(this);
  }

  final int getComposingStart() {
    return BaseInputConnection.getComposingSpanStart(this);
  }

  final int getComposingEnd() {
    return BaseInputConnection.getComposingSpanEnd(this);
  }
}
