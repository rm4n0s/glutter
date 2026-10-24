package io.github.rm4n0s.glutter;

import android.graphics.Rect;
import android.opengl.Matrix;
import android.os.Build;
import android.os.Bundle;
import android.text.SpannableString;
import android.text.TextUtils;
import android.text.style.LocaleSpan;
import android.text.style.TtsSpan;
import android.text.style.URLSpan;
import android.util.Log;
import android.view.MotionEvent;
import android.view.View;
import android.view.ViewParent;
import android.view.accessibility.AccessibilityEvent;
import android.view.accessibility.AccessibilityManager;
import android.view.accessibility.AccessibilityNodeInfo;
import android.view.accessibility.AccessibilityNodeProvider;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Iterator;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Set;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * The accessibility bridge of the activity's view: it makes the semantics tree of the framework
 * visible to TalkBack and the other accessibility services (SPEC §24.10). Go (accessibility.go)
 * turns each semantics update into a payload, with every property of an {@code
 * AccessibilityNodeInfo} already worked out; this class keeps its own copy of the tree on the main
 * thread, answers {@code createAccessibilityNodeInfo}, {@code findFocus} and {@code performAction}
 * from it without waiting for Go, sends the events that Flutter sends, and sends the actions back to
 * Go ({@code Native.a11yAction}). Everything here runs on the main thread.
 *
 * <p>Ports: {@code AccessibilityBridge} and {@code AccessibilityStringBuilder}
 * (shell/platform/android/io/flutter/view/AccessibilityBridge.java and
 * AccessibilityStringBuilder.java): {@code updateSemantics}, {@code SemanticsNode.updateWith},
 * {@code updateRecursively}, {@code hitTest}, {@code createAccessibilityNodeInfo} (the decisions that
 * need the tree or the focus; the rest, the role configurators of BaseRoleConfigurator and its
 * subclasses, are Go's), {@code performAction}, {@code findFocus}, {@code onAccessibilityHoverEvent}
 * and the events of {@code updateSemantics}. Not ported: platform views (their nodes are never
 * made), the announcement and tooltip messages of the accessibility channel (the framework does not
 * send them yet), the iOS parts.
 *
 * <h2>The payload (version 1, little-endian)</h2>
 *
 * <pre>
 * payload   := u8 version(=1), u32 nodeCount, node * nodeCount
 * node      := i32 id, u32 flags, u32 actions, u32 props, u8 checked, u8 expanded, u8 granularities,
 *              u8 rangeType, i32 maxTextLength, i32 scrollChildren, i32 scrollIndex,
 *              i32 selectionBase, i32 selectionExtent,
 *              f32 scrollPosition, f32 scrollExtentMax, f32 scrollExtentMin,
 *              f32 rangeMin, f32 rangeMax, f32 rangeCurrent, i32 collectionRows, i32 collectionColumns,
 *              f32 left, f32 top, f32 right, f32 bottom, f32 * 16 transform, f32 * 16 hitTestTransform,
 *              str identifier, str className, str value, str label, str tooltip,
 *              styledList text, styledList content, styledList hint, styledList increased,
 *              styledList decreased, u32 n, i32 * n traversalChildren, u32 n, i32 * n hitTestChildren,
 *              u32 n, (i32 actionId, str label) * n
 * styledList := u8 n, styled * n
 * styled    := u8 kind (0 value, 1 label, 2 hint), str text, str url, str locale,
 *              u32 n, (u32 start, u32 end, u8 type (0 spell out, 1 locale), str locale) * n
 * str       := u32 byteCount, UTF-8 bytes (0 bytes is "no string")
 * </pre>
 *
 * The fields, the bits of {@code flags}, {@code actions} and {@code props} are documented in
 * accessibility_wire.go; the two MUST agree. A payload that is cut short, too large or
 * inconsistent is logged and dropped as a whole. The golden payload of {@code
 * TestSemanticsPayloadGolden} (a root and a button "OK" with the hint "Press" that can be tapped,
 * 649 bytes) is the constant {@code goldenHex} of accessibility_wire_test.go, which that test
 * compares with the encoder's output. Here it is, hex, with the line breaks for reading only:
 *
 * <pre>
 * 01020000000000000000000000000000000400000000000000ffffffff0000000000000000ffffffffffffffff0000c0
 * 7f0000c07f0000c07f00000000000000000000000000000000000000000000000000000000000048430000c842000080
 * 3f000000000000000000000000000000000000803f000000000000000000000000000000000000803f00000000000000
 * 0000000000000000000000803f0000803f000000000000000000000000000000000000803f0000000000000000000000
 * 00000000000000803f000000000000000000000000000000000000803f0000000011000000616e64726f69642e766965
 * 772e566965770000000000000000000000000000000000010000000100000001000000010000000000000001000000c8
 * 000000010000000f00000000000000ffffffff0000000000000000ffffffffffffffff0000c07f0000c07f0000c07f00
 * 0000000000000000000000000000000000000000002041000020410000dc42000048420000803f000000000000000000
 * 000000000000000000803f000000000000000000000000000000000000803f0000000000000000000000000000000000
 * 00803f0000803f000000000000000000000000000000000000803f000000000000000000000000000000000000803f00
 * 0000000000000000000000000000000000803f0000000015000000616e64726f69642e7769646765742e427574746f6e
 * 00000000020000004f4b00000000000201020000004f4b00000000000000000000000002050000005072657373000000
 * 0000000000000000000201020000004f4b00000000000000000000000002050000005072657373000000000000000000
 * 00000000000000000000000000010000001000000000000000
 * </pre>
 *
 * It reads: version 1, two nodes; node 0 has the id 0, flags 0, actions 0, props 4 (enabled), the
 * class "android.view.View" and the child 1 (twice, for the traversal and the hit test order); node 1
 * has the id 1, the flags 0xc8 (button, has an enabled state, enabled), the action 1 (tap), the props
 * 0xf (important, focusable, enabled, clickable), the class "android.widget.Button", the label "OK"
 * and the hint "Press" (the content list and the hint list each hold "OK" and "Press", the text list
 * is empty), and the one Android action 0x10 (click).
 *
 * <h2>The logs</h2>
 *
 * Every event that is sent is logged at Info under {@code GlutterJava} as {@code a11y event
 * type=NAME node=ID text=TEXT}, followed by {@code live=1} for a change of a live region and {@code
 * pane=1} for the title of the pane (API 28 and later, where the view sends the event itself); the
 * bridge logs {@code a11y bridge on} and {@code a11y bridge off} when it changes and {@code a11y
 * tree nodes=N} after each update that was applied.
 */
final class GlutterAccessibility extends AccessibilityNodeProvider {
  // The operations of GlutterActivity.command (accessibility.go).
  static final int OP_SEMANTICS = 16;
  static final int OP_FORCE = 17;
  static final int OP_RESET = 18;

  private static final String TAG = "GlutterJava";

  private static final int VERSION = 1;
  private static final int MAX_PAYLOAD = 64 << 20;
  private static final int MIN_NODE_BYTES = 253;

  private static final int ROOT_NODE_ID = 0;
  private static final int FIRST_RESOURCE_ID = 267386881;
  private static final int ACTION_SHOW_ON_SCREEN = 16908342;
  private static final float SCROLL_EXTENT_FOR_INFINITY = 100000.0f;
  private static final float SCROLL_POSITION_CAP_FOR_INFINITY = 70000.0f;

  // AccessibilityBridge.Flag
  private static final int F_IS_CHECKED = 1 << 1;
  private static final int F_IS_SELECTED = 1 << 2;
  private static final int F_IS_TEXT_FIELD = 1 << 4;
  private static final int F_IS_FOCUSED = 1 << 5;
  private static final int F_IS_HEADER = 1 << 9;
  private static final int F_SCOPES_ROUTE = 1 << 11;
  private static final int F_NAMES_ROUTE = 1 << 12;
  private static final int F_IS_HIDDEN = 1 << 13;
  private static final int F_IS_LIVE_REGION = 1 << 15;
  private static final int F_IS_TOGGLED = 1 << 17;
  private static final int F_HAS_IMPLICIT_SCROLLING = 1 << 18;
  private static final int F_IS_CHECK_STATE_MIXED = 1 << 25;
  private static final int F_HAS_EXPANDED_STATE = 1 << 26;
  private static final int F_IS_EXPANDED = 1 << 27;

  // AccessibilityBridge.Action
  private static final int A_TAP = 1;
  private static final int A_LONG_PRESS = 1 << 1;
  private static final int A_SCROLL_LEFT = 1 << 2;
  private static final int A_SCROLL_RIGHT = 1 << 3;
  private static final int A_SCROLL_UP = 1 << 4;
  private static final int A_SCROLL_DOWN = 1 << 5;
  private static final int A_INCREASE = 1 << 6;
  private static final int A_DECREASE = 1 << 7;
  private static final int A_SHOW_ON_SCREEN = 1 << 8;
  private static final int A_MOVE_CURSOR_FORWARD_BY_CHARACTER = 1 << 9;
  private static final int A_MOVE_CURSOR_BACKWARD_BY_CHARACTER = 1 << 10;
  private static final int A_SET_SELECTION = 1 << 11;
  private static final int A_COPY = 1 << 12;
  private static final int A_CUT = 1 << 13;
  private static final int A_PASTE = 1 << 14;
  private static final int A_DID_GAIN_ACCESSIBILITY_FOCUS = 1 << 15;
  private static final int A_DID_LOSE_ACCESSIBILITY_FOCUS = 1 << 16;
  private static final int A_CUSTOM_ACTION = 1 << 17;
  private static final int A_DISMISS = 1 << 18;
  private static final int A_MOVE_CURSOR_FORWARD_BY_WORD = 1 << 19;
  private static final int A_MOVE_CURSOR_BACKWARD_BY_WORD = 1 << 20;
  private static final int A_SET_TEXT = 1 << 21;
  private static final int A_FOCUS = 1 << 22;
  private static final int A_EXPAND = 1 << 24;
  private static final int A_COLLAPSE = 1 << 25;

  // The bits of the props of a node (accessibility_wire.go)
  private static final int P_IMPORTANT = 1;
  private static final int P_FOCUSABLE = 1 << 1;
  private static final int P_ENABLED = 1 << 2;
  private static final int P_CLICKABLE = 1 << 3;
  private static final int P_LONG_CLICKABLE = 1 << 4;
  private static final int P_SCROLLABLE = 1 << 5;
  private static final int P_CHECKABLE = 1 << 6;
  private static final int P_SELECTED = 1 << 7;
  private static final int P_PASSWORD = 1 << 8;
  private static final int P_EDITABLE = 1 << 9;
  private static final int P_DISMISSABLE = 1 << 10;
  private static final int P_HEADING = 1 << 11;
  private static final int P_LIVE_POLITE = 1 << 12;
  private static final int P_CAN_OPEN_POPUP = 1 << 13;
  private static final int P_TEXT_FIELD = 1 << 14;

  /** A piece of text with the spans of AccessibilityStringBuilder; the kind says where it is from. */
  private static final class Styled {
    int kind;
    String text;
    String url;
    String locale;
    int[] starts;
    int[] ends;
    int[] types;
    String[] locales;

    /** The text with its spans ({@code AccessibilityStringBuilder.build}). */
    CharSequence build() {
      SpannableString s = new SpannableString(text);
      for (int i = 0; i < starts.length; i++) {
        int start = Math.max(0, Math.min(starts[i], text.length()));
        int end = Math.max(start, Math.min(ends[i], text.length()));
        if (types[i] == 0) {
          s.setSpan(new TtsSpan.Builder<>(TtsSpan.TYPE_VERBATIM).build(), start, end, 0);
        } else {
          s.setSpan(new LocaleSpan(Locale.forLanguageTag(locales[i])), start, end, 0);
        }
      }
      if (url != null && !url.isEmpty()) {
        s.setSpan(new URLSpan(url), 0, text.length(), 0);
      }
      if (locale != null && !locale.isEmpty()) {
        s.setSpan(new LocaleSpan(Locale.forLanguageTag(locale)), 0, text.length(), 0);
      }
      return s;
    }

    static Styled plain(int kind, String text) {
      Styled s = new Styled();
      s.kind = kind;
      s.text = text;
      s.starts = new int[0];
      s.ends = new int[0];
      s.types = new int[0];
      s.locales = new String[0];
      return s;
    }
  }

  /** One decoded node record: the state of a semantics node (accessibility_wire.go). */
  private static final class Data {
    int id;
    int flags;
    int actions;
    int props;
    int checked;
    int expanded;
    int granularities;
    int rangeType;
    int maxTextLength;
    int scrollChildren;
    int scrollIndex;
    int selectionBase;
    int selectionExtent;
    float scrollPosition;
    float scrollExtentMax;
    float scrollExtentMin;
    float rangeMin;
    float rangeMax;
    float rangeCurrent;
    int collectionRows;
    int collectionColumns;
    float left;
    float top;
    float right;
    float bottom;
    float[] transform = new float[16];
    float[] hitTestTransform = new float[16];
    String identifier;
    String className;
    String value;
    String label;
    String tooltip;
    Styled[] text;
    Styled[] content;
    Styled[] hint;
    Styled[] increased;
    Styled[] decreased;
    int[] traversalChildren;
    int[] hitTestChildren;
    int[] actionIds;
    String[] actionLabels;

    /**
     * The record of a node that a parent names before the node itself arrives: it is empty, with
     * identity transforms, as a SemanticsNode of Flutter's bridge that was not updated yet.
     */
    static Data placeholder(int id) {
      Data d = new Data();
      d.id = id;
      d.maxTextLength = -1;
      d.selectionBase = 0;
      d.selectionExtent = 0;
      Matrix.setIdentityM(d.transform, 0);
      Matrix.setIdentityM(d.hitTestTransform, 0);
      d.className = "android.view.View";
      d.text = new Styled[0];
      d.content = new Styled[0];
      d.hint = new Styled[0];
      d.increased = new Styled[0];
      d.decreased = new Styled[0];
      d.traversalChildren = new int[0];
      d.hitTestChildren = new int[0];
      d.actionIds = new int[0];
      d.actionLabels = new String[0];
      return d;
    }
  }

  /** A node of the tree: its latest record, the previous values the events compare with, the links. */
  private static final class Node {
    final int id;
    Data d;
    Node parent;
    final ArrayList<Node> childrenInTraversalOrder = new ArrayList<>();
    final ArrayList<Node> childrenInHitTestOrder = new ArrayList<>();

    boolean hadPreviousConfig;
    int previousFlags;
    int previousActions;
    int previousSelectionBase;
    int previousSelectionExtent;
    float previousScrollPosition;
    String previousValue;
    String previousLabel;
    int previousNodeId = -1;

    boolean inverseTransformDirty = true;
    float[] inverseTransform;
    boolean globalGeometryDirty = true;
    float[] globalTransform;
    Rect globalRect;

    Node(int id) {
      this.id = id;
    }

    boolean hasFlag(int flag) {
      return d != null && (d.flags & flag) != 0;
    }

    boolean hasAction(int action) {
      return d != null && (d.actions & action) != 0;
    }

    boolean hadFlag(int flag) {
      return (previousFlags & flag) != 0;
    }

    boolean hadAction(int action) {
      return (previousActions & action) != 0;
    }

    boolean isFocusable() {
      return d != null && (d.props & P_FOCUSABLE) != 0;
    }

    boolean didScroll() {
      return !Float.isNaN(d.scrollPosition)
          && !Float.isNaN(previousScrollPosition)
          && previousScrollPosition != d.scrollPosition;
    }

    boolean didChangeLabel() {
      if (d.label == null && previousLabel == null) {
        return false;
      }
      return d.label == null || !d.label.equals(previousLabel);
    }

    boolean didChangeCheckedState() {
      if (!hadPreviousConfig) {
        return false;
      }
      return hasFlag(F_IS_CHECKED) != hadFlag(F_IS_CHECKED)
          || hasFlag(F_IS_CHECK_STATE_MIXED) != hadFlag(F_IS_CHECK_STATE_MIXED)
          || hasFlag(F_IS_TOGGLED) != hadFlag(F_IS_TOGGLED);
    }

    boolean didChangeExpandState() {
      if (!hadPreviousConfig) {
        return false;
      }
      return hasFlag(F_IS_EXPANDED) != hadFlag(F_IS_EXPANDED)
          || hasFlag(F_HAS_EXPANDED_STATE) != hadFlag(F_HAS_EXPANDED_STATE);
    }

    Node getAncestor(java.util.function.Predicate<Node> tester) {
      Node next = parent;
      while (next != null) {
        if (tester.test(next)) {
          return next;
        }
        next = next.parent;
      }
      return null;
    }

    /** The first non-empty label of a node that names the route, below this one. */
    String getRouteName() {
      if (hasFlag(F_NAMES_ROUTE) && d.label != null && !d.label.isEmpty()) {
        return d.label;
      }
      for (Node child : childrenInTraversalOrder) {
        String name = child.getRouteName();
        if (name != null && !name.isEmpty()) {
          return name;
        }
      }
      return null;
    }

    void collectRoutes(List<Node> edges) {
      if (hasFlag(F_SCOPES_ROUTE)) {
        edges.add(this);
      }
      for (Node child : childrenInTraversalOrder) {
        child.collectRoutes(edges);
      }
    }

    void ensureInverseTransform() {
      if (!inverseTransformDirty) {
        return;
      }
      inverseTransformDirty = false;
      if (inverseTransform == null) {
        inverseTransform = new float[16];
      }
      if (!Matrix.invertM(inverseTransform, 0, d.hitTestTransform, 0)) {
        java.util.Arrays.fill(inverseTransform, 0);
      }
    }

    /** The deepest focusable node at the point, which is in this node's space. */
    Node hitTest(float[] point) {
      final float w = point[3];
      final float x = point[0] / w;
      final float y = point[1] / w;
      if (x < d.left || x >= d.right || y < d.top || y >= d.bottom) {
        return null;
      }
      final float[] transformed = new float[4];
      for (Node child : childrenInHitTestOrder) {
        if (child.d == null || child.hasFlag(F_IS_HIDDEN)) {
          continue;
        }
        child.ensureInverseTransform();
        Matrix.multiplyMV(transformed, 0, child.inverseTransform, 0, point, 0);
        Node result = child.hitTest(transformed);
        if (result != null) {
          return result;
        }
      }
      return isFocusable() ? this : null;
    }

    /** Works out the global transform and rectangle of this node and its subtree. */
    void updateRecursively(
        float[] ancestorTransform, Set<Node> visited, boolean forceUpdate, boolean useHitTestTransform) {
      visited.add(this);
      if (globalGeometryDirty) {
        forceUpdate = true;
      }
      if (forceUpdate) {
        if (globalTransform == null) {
          globalTransform = new float[16];
        }
        float[] nodeTransform = useHitTestTransform ? d.hitTestTransform : d.transform;
        Matrix.multiplyMM(globalTransform, 0, ancestorTransform, 0, nodeTransform, 0);
        final float[] sample = new float[4];
        sample[2] = 0;
        sample[3] = 1;
        final float[] p1 = new float[4];
        final float[] p2 = new float[4];
        final float[] p3 = new float[4];
        final float[] p4 = new float[4];
        sample[0] = d.left;
        sample[1] = d.top;
        transformPoint(p1, globalTransform, sample);
        sample[0] = d.right;
        sample[1] = d.top;
        transformPoint(p2, globalTransform, sample);
        sample[0] = d.right;
        sample[1] = d.bottom;
        transformPoint(p3, globalTransform, sample);
        sample[0] = d.left;
        sample[1] = d.bottom;
        transformPoint(p4, globalTransform, sample);
        if (globalRect == null) {
          globalRect = new Rect();
        }
        globalRect.set(
            Math.round(Math.min(Math.min(p1[0], p2[0]), Math.min(p3[0], p4[0]))),
            Math.round(Math.min(Math.min(p1[1], p2[1]), Math.min(p3[1], p4[1]))),
            Math.round(Math.max(Math.max(p1[0], p2[0]), Math.max(p3[0], p4[0]))),
            Math.round(Math.max(Math.max(p1[1], p2[1]), Math.max(p3[1], p4[1]))));
        globalGeometryDirty = false;
      }
      int previous = -1;
      for (Node child : childrenInTraversalOrder) {
        child.previousNodeId = previous;
        previous = child.id;
        child.updateRecursively(globalTransform, visited, forceUpdate, false);
      }
      for (Node child : childrenInHitTestOrder) {
        if (!visited.contains(child)) {
          child.updateRecursively(globalTransform, visited, forceUpdate, true);
        }
      }
    }

    private static void transformPoint(float[] result, float[] transform, float[] point) {
      Matrix.multiplyMV(result, 0, transform, 0, point, 0);
      final float w = result[3];
      result[0] /= w;
      result[1] /= w;
      result[2] /= w;
      result[3] = 0;
    }
  }

  private final GlutterView view;
  private final AccessibilityManager manager;

  /** The tree, by id (AccessibilityBridge.flutterSemanticsTree). */
  private final Map<Integer, Node> tree = new HashMap<>();

  private final List<Integer> flutterNavigationStack = new ArrayList<>();
  private int previousRouteId = ROOT_NODE_ID;
  private Node accessibilityFocused;
  private Node inputFocused;
  private Node lastInputFocused;
  private Node hovered;

  /** Whether the bridge is forced on (debug builds, GLUTTER_A11Y). */
  private boolean forced;

  /** The state last reported to Go, and whether it was reported at all. */
  private boolean reported;

  private boolean reportedEnabled;
  private boolean reportedTouch;
  private boolean reportedNavigation;

  /** Flutter's accessibleNavigation. */
  private boolean navigation;

  /** Whether the bridge was on at the last look, for the logs. */
  private boolean wasOn;

  private boolean started;

  private final AccessibilityManager.AccessibilityStateChangeListener stateListener =
      new AccessibilityManager.AccessibilityStateChangeListener() {
        @Override
        public void onAccessibilityStateChanged(boolean enabled) {
          if (!enabled) {
            navigation = false;
          }
          stateChanged();
        }
      };

  private final AccessibilityManager.TouchExplorationStateChangeListener touchListener =
      new AccessibilityManager.TouchExplorationStateChangeListener() {
        @Override
        public void onTouchExplorationStateChanged(boolean touch) {
          if (touch) {
            navigation = true;
          } else {
            navigation = false;
            onTouchExplorationExit();
          }
          stateChanged();
        }
      };

  GlutterAccessibility(GlutterView view) {
    this.view = view;
    this.manager =
        (AccessibilityManager) view.getContext().getSystemService(android.content.Context.ACCESSIBILITY_SERVICE);
  }

  /** Starts to follow the accessibility manager and tells Go its state. */
  void start() {
    if (started || manager == null) {
      return;
    }
    started = true;
    manager.addAccessibilityStateChangeListener(stateListener);
    manager.addTouchExplorationStateChangeListener(touchListener);
    navigation = manager.isTouchExplorationEnabled();
    stateChanged();
  }

  /** Stops following the accessibility manager (the activity goes). */
  void release() {
    if (!started || manager == null) {
      return;
    }
    started = false;
    manager.removeAccessibilityStateChangeListener(stateListener);
    manager.removeTouchExplorationStateChangeListener(touchListener);
  }

  /** Whether the bridge works now: a service runs, or it is forced. */
  boolean isActive() {
    return forced || (manager != null && manager.isEnabled());
  }

  private boolean isOn() {
    return forced
        || (manager != null && (manager.isEnabled() || manager.isTouchExplorationEnabled()));
  }

  /** The accessibility manager's state changed (or the bridge is forced): tells Go and the log. */
  private void stateChanged() {
    boolean enabled = manager != null && manager.isEnabled();
    boolean touch = manager != null && manager.isTouchExplorationEnabled();
    boolean on = isOn();
    if (on != wasOn) {
      wasOn = on;
      Log.i(TAG, on ? "a11y bridge on" : "a11y bridge off");
      if (!on) {
        reset();
      }
    }
    if (!reported
        || enabled != reportedEnabled
        || touch != reportedTouch
        || navigation != reportedNavigation) {
      reported = true;
      reportedEnabled = enabled;
      reportedTouch = touch;
      reportedNavigation = navigation;
      Native.a11yState(enabled, touch, navigation);
    }
  }

  /** Flutter's {@code setAccessibleNavigation}. */
  private void setAccessibleNavigation(boolean value) {
    if (navigation == value) {
      return;
    }
    navigation = value;
    stateChanged();
  }

  /** One operation (16 to 18) of the command call. {@code bytes} is the payload of operation 16. */
  void command(int op, long a, byte[] bytes) {
    if (op == OP_FORCE) {
      forced = a != 0;
      stateChanged();
    } else if (op == OP_RESET) {
      reset();
    } else if (op == OP_SEMANTICS) {
      if (!isOn()) {
        // The bridge is off: an update on its way is stale.
        return;
      }
      List<Data> nodes;
      try {
        nodes = decode(bytes);
      } catch (RuntimeException e) {
        Log.w(TAG, "a11y payload dropped: " + e.getMessage());
        return;
      }
      updateSemantics(nodes);
    }
  }

  // ---- The payload ----

  private static final class Reader {
    final ByteBuffer b;

    Reader(byte[] data) {
      b = ByteBuffer.wrap(data).order(ByteOrder.LITTLE_ENDIAN);
    }

    void need(int n) {
      if (n < 0 || n > b.remaining()) {
        throw new IllegalArgumentException("the payload ends before a field does (offset " + b.position() + ")");
      }
    }

    int u8() {
      need(1);
      return b.get() & 0xff;
    }

    int i32() {
      need(4);
      return b.getInt();
    }

    float f32() {
      need(4);
      return b.getFloat();
    }

    String str() {
      int n = i32();
      need(n);
      if (n == 0) {
        return null;
      }
      byte[] s = new byte[n];
      b.get(s);
      return new String(s, StandardCharsets.UTF_8);
    }

    /** A count of items of at least {@code size} bytes that the rest of the payload must hold. */
    int count(int size) {
      int n = i32();
      if (n < 0 || n > b.remaining() / size) {
        throw new IllegalArgumentException("a count is larger than the rest of the payload (offset " + b.position() + ")");
      }
      return n;
    }

    int[] ids() {
      int n = count(4);
      int[] out = new int[n];
      for (int i = 0; i < n; i++) {
        out[i] = i32();
      }
      return out;
    }

    Styled[] styledList() {
      int n = u8();
      Styled[] out = new Styled[n];
      for (int i = 0; i < n; i++) {
        Styled s = new Styled();
        s.kind = u8();
        s.text = str();
        s.url = str();
        s.locale = str();
        if (s.text == null) {
          s.text = "";
        }
        int na = count(13);
        s.starts = new int[na];
        s.ends = new int[na];
        s.types = new int[na];
        s.locales = new String[na];
        for (int j = 0; j < na; j++) {
          s.starts[j] = i32();
          s.ends[j] = i32();
          s.types[j] = u8();
          s.locales[j] = str();
        }
        out[i] = s;
      }
      return out;
    }
  }

  /** Decodes a payload; throws {@code IllegalArgumentException} for one that is not valid. */
  private static List<Data> decode(byte[] payload) {
    if (payload == null || payload.length > MAX_PAYLOAD) {
      throw new IllegalArgumentException("the payload is missing or larger than the limit");
    }
    Reader r = new Reader(payload);
    int version = r.u8();
    if (version != VERSION) {
      throw new IllegalArgumentException("the payload has version " + version);
    }
    int n = r.count(MIN_NODE_BYTES);
    List<Data> out = new ArrayList<>(n);
    for (int i = 0; i < n; i++) {
      Data d = new Data();
      d.id = r.i32();
      d.flags = r.i32();
      d.actions = r.i32();
      d.props = r.i32();
      d.checked = r.u8();
      d.expanded = r.u8();
      d.granularities = r.u8();
      d.rangeType = r.u8();
      d.maxTextLength = r.i32();
      d.scrollChildren = r.i32();
      d.scrollIndex = r.i32();
      d.selectionBase = r.i32();
      d.selectionExtent = r.i32();
      d.scrollPosition = r.f32();
      d.scrollExtentMax = r.f32();
      d.scrollExtentMin = r.f32();
      d.rangeMin = r.f32();
      d.rangeMax = r.f32();
      d.rangeCurrent = r.f32();
      d.collectionRows = r.i32();
      d.collectionColumns = r.i32();
      d.left = r.f32();
      d.top = r.f32();
      d.right = r.f32();
      d.bottom = r.f32();
      for (int j = 0; j < 16; j++) {
        d.transform[j] = r.f32();
      }
      for (int j = 0; j < 16; j++) {
        d.hitTestTransform[j] = r.f32();
      }
      d.identifier = r.str();
      d.className = r.str();
      d.value = r.str();
      d.label = r.str();
      d.tooltip = r.str();
      d.text = r.styledList();
      d.content = r.styledList();
      d.hint = r.styledList();
      d.increased = r.styledList();
      d.decreased = r.styledList();
      d.traversalChildren = r.ids();
      d.hitTestChildren = r.ids();
      int na = r.count(8);
      d.actionIds = new int[na];
      d.actionLabels = new String[na];
      for (int j = 0; j < na; j++) {
        d.actionIds[j] = r.i32();
        d.actionLabels[j] = r.str();
      }
      if (d.className == null) {
        d.className = "android.view.View";
      }
      out.add(d);
    }
    if (r.b.hasRemaining()) {
      throw new IllegalArgumentException("the payload has bytes after its last node");
    }
    return out;
  }

  // ---- The tree ----

  private Node getOrCreate(int id) {
    Node node = tree.get(id);
    if (node == null) {
      node = new Node(id);
      node.d = Data.placeholder(id);
      tree.put(id, node);
    }
    return node;
  }

  /** {@code SemanticsNode.updateWith}: the new record, the previous values, the children. */
  private void updateWith(Node node, Data d) {
    node.hadPreviousConfig = true;
    Data old = node.d;
    if (old != null) {
      node.previousValue = old.value;
      node.previousLabel = old.label;
      node.previousFlags = old.flags;
      node.previousActions = old.actions;
      node.previousSelectionBase = old.selectionBase;
      node.previousSelectionExtent = old.selectionExtent;
      node.previousScrollPosition = old.scrollPosition;
    } else {
      node.previousValue = null;
      node.previousLabel = null;
      node.previousFlags = 0;
      node.previousActions = 0;
      node.previousSelectionBase = 0;
      node.previousSelectionExtent = 0;
      node.previousScrollPosition = 0;
    }
    node.d = d;
    node.inverseTransformDirty = true;
    node.globalGeometryDirty = true;
    node.childrenInTraversalOrder.clear();
    for (int id : d.traversalChildren) {
      Node child = getOrCreate(id);
      child.parent = node;
      node.childrenInTraversalOrder.add(child);
    }
    node.childrenInHitTestOrder.clear();
    for (int id : d.hitTestChildren) {
      Node child = getOrCreate(id);
      child.parent = node;
      node.childrenInHitTestOrder.add(child);
    }
  }

  /** {@code AccessibilityBridge.updateSemantics}: the tree, the events. */
  private void updateSemantics(List<Data> nodes) {
    ArrayList<Node> updated = new ArrayList<>();
    for (Data d : nodes) {
      Node node = getOrCreate(d.id);
      updateWith(node, d);
      if (node.hasFlag(F_IS_HIDDEN)) {
        continue;
      }
      if (node.hasFlag(F_IS_FOCUSED)) {
        inputFocused = node;
      }
      if (node.hadPreviousConfig) {
        updated.add(node);
      }
    }

    Set<Node> visited = new HashSet<>();
    Node root = tree.get(ROOT_NODE_ID);
    List<Node> newRoutes = new ArrayList<>();
    if (root != null && root.d != null) {
      final float[] identity = new float[16];
      Matrix.setIdentityM(identity, 0);
      root.updateRecursively(identity, visited, false, false);
      root.collectRoutes(newRoutes);
    }

    // A TYPE_WINDOW_STATE_CHANGED when the most recent route changed.
    Node lastAdded = null;
    for (Node node : newRoutes) {
      if (!flutterNavigationStack.contains(node.id)) {
        lastAdded = node;
      }
    }
    if (lastAdded == null && !newRoutes.isEmpty()) {
      lastAdded = newRoutes.get(newRoutes.size() - 1);
    }
    if (lastAdded != null
        && (lastAdded.id != previousRouteId || newRoutes.size() != flutterNavigationStack.size())) {
      previousRouteId = lastAdded.id;
      onWindowNameChange(lastAdded);
    }
    flutterNavigationStack.clear();
    for (Node node : newRoutes) {
      flutterNavigationStack.add(node.id);
    }

    Iterator<Map.Entry<Integer, Node>> it = tree.entrySet().iterator();
    while (it.hasNext()) {
      Node node = it.next().getValue();
      if (!visited.contains(node)) {
        willRemoveSemanticsNode(node);
        it.remove();
      }
    }
    Log.i(TAG, "a11y tree nodes=" + tree.size());

    sendWindowContentChangeEvent(0, AccessibilityEvent.CONTENT_CHANGE_TYPE_SUBTREE, false);

    for (Node object : updated) {
      if (object.d == null || !tree.containsKey(object.id)) {
        continue;
      }
      if (object.didScroll()) {
        sendScrolled(object);
      }
      if (object.hasFlag(F_IS_LIVE_REGION) && object.didChangeLabel()) {
        sendWindowContentChangeEvent(object.id, AccessibilityEvent.CONTENT_CHANGE_TYPE_SUBTREE, true);
      }
      if (Build.VERSION.SDK_INT >= 36) {
        if (object.didChangeCheckedState()) {
          sendWindowContentChangeEvent(object.id, AccessibilityEvent.CONTENT_CHANGE_TYPE_CHECKED, false);
        }
        if (object.didChangeExpandState()) {
          sendWindowContentChangeEvent(object.id, AccessibilityEvent.CONTENT_CHANGE_TYPE_EXPANDED, false);
        }
      }
      if (accessibilityFocused != null
          && accessibilityFocused.id == object.id
          && !object.hadFlag(F_IS_SELECTED)
          && object.hasFlag(F_IS_SELECTED)) {
        AccessibilityEvent event = obtainEvent(object.id, AccessibilityEvent.TYPE_VIEW_SELECTED);
        event.getText().add(object.d.label);
        sendEvent(event, object.id, false);
      }

      // The input focus, only when it changed since the last update.
      if (inputFocused != null
          && inputFocused.id == object.id
          && (lastInputFocused == null || lastInputFocused.id != inputFocused.id)) {
        lastInputFocused = inputFocused;
        sendEvent(obtainEvent(object.id, AccessibilityEvent.TYPE_VIEW_FOCUSED), object.id, false);
      } else if (inputFocused == null) {
        lastInputFocused = null;
      }

      if (inputFocused != null
          && inputFocused.id == object.id
          && object.hadFlag(F_IS_TEXT_FIELD)
          && object.hasFlag(F_IS_TEXT_FIELD)
          && (accessibilityFocused == null || accessibilityFocused.id == inputFocused.id)) {
        String oldValue = object.previousValue != null ? object.previousValue : "";
        String newValue = object.d.value != null ? object.d.value : "";
        AccessibilityEvent event = createTextChangedEvent(object.id, oldValue, newValue);
        if (event != null) {
          sendEvent(event, object.id, false);
        }
        if (object.previousSelectionBase != object.d.selectionBase
            || object.previousSelectionExtent != object.d.selectionExtent) {
          AccessibilityEvent selection =
              obtainEvent(object.id, AccessibilityEvent.TYPE_VIEW_TEXT_SELECTION_CHANGED);
          selection.getText().add(newValue);
          selection.setFromIndex(object.d.selectionBase);
          selection.setToIndex(object.d.selectionExtent);
          selection.setItemCount(newValue.length());
          sendEvent(selection, object.id, false);
        }
      }
    }
  }

  private void sendScrolled(Node object) {
    AccessibilityEvent event = obtainEvent(object.id, AccessibilityEvent.TYPE_VIEW_SCROLLED);
    // Android does not support unbounded scrolling: a large bound that can never be reached stands in.
    float position = object.d.scrollPosition;
    float max = object.d.scrollExtentMax;
    if (Float.isInfinite(object.d.scrollExtentMax)) {
      max = SCROLL_EXTENT_FOR_INFINITY;
      if (position > SCROLL_POSITION_CAP_FOR_INFINITY) {
        position = SCROLL_POSITION_CAP_FOR_INFINITY;
      }
    }
    if (Float.isInfinite(object.d.scrollExtentMin)) {
      max += SCROLL_EXTENT_FOR_INFINITY;
      if (position < -SCROLL_POSITION_CAP_FOR_INFINITY) {
        position = -SCROLL_POSITION_CAP_FOR_INFINITY;
      }
      position += SCROLL_EXTENT_FOR_INFINITY;
    } else {
      max -= object.d.scrollExtentMin;
      position -= object.d.scrollExtentMin;
    }
    if (object.hadAction(A_SCROLL_UP) || object.hadAction(A_SCROLL_DOWN)) {
      event.setScrollY((int) position);
      event.setMaxScrollY((int) max);
    } else if (object.hadAction(A_SCROLL_LEFT) || object.hadAction(A_SCROLL_RIGHT)) {
      event.setScrollX((int) position);
      event.setMaxScrollX((int) max);
    }
    if (object.d.scrollChildren > 0) {
      // TalkBack adds 1 to the index by itself.
      event.setItemCount(object.d.scrollChildren);
      event.setFromIndex(object.d.scrollIndex);
      int visibleChildren = 0;
      for (Node child : object.childrenInHitTestOrder) {
        if (!child.hasFlag(F_IS_HIDDEN)) {
          visibleChildren += 1;
        }
      }
      event.setToIndex(object.d.scrollIndex + visibleChildren - 1);
    }
    Log.d(TAG, "a11y scroll node=" + object.id + " position=" + position + " max=" + max);
    sendEvent(event, object.id, false);
  }

  private AccessibilityEvent createTextChangedEvent(int id, String oldValue, String newValue) {
    AccessibilityEvent e = obtainEvent(id, AccessibilityEvent.TYPE_VIEW_TEXT_CHANGED);
    e.setBeforeText(oldValue);
    e.getText().add(newValue);
    int i;
    for (i = 0; i < oldValue.length() && i < newValue.length(); ++i) {
      if (oldValue.charAt(i) != newValue.charAt(i)) {
        break;
      }
    }
    if (i >= oldValue.length() && i >= newValue.length()) {
      return null; // The text did not change.
    }
    int firstDifference = i;
    e.setFromIndex(firstDifference);
    int oldIndex = oldValue.length() - 1;
    int newIndex = newValue.length() - 1;
    while (oldIndex >= firstDifference && newIndex >= firstDifference) {
      if (oldValue.charAt(oldIndex) != newValue.charAt(newIndex)) {
        break;
      }
      --oldIndex;
      --newIndex;
    }
    e.setRemovedCount(oldIndex - firstDifference + 1);
    e.setAddedCount(newIndex - firstDifference + 1);
    return e;
  }

  /** {@code willRemoveSemanticsNode}: the focus and the hover leave a node that goes. */
  private void willRemoveSemanticsNode(Node removed) {
    removed.parent = null;
    if (accessibilityFocused == removed) {
      sendEvent(
          obtainEvent(accessibilityFocused.id, AccessibilityEvent.TYPE_VIEW_ACCESSIBILITY_FOCUS_CLEARED),
          accessibilityFocused.id,
          false);
      accessibilityFocused = null;
    }
    if (inputFocused == removed) {
      inputFocused = null;
    }
    if (hovered == removed) {
      hovered = null;
    }
  }

  /** {@code AccessibilityBridge.reset}: the tree goes, and so does the focus. */
  private void reset() {
    boolean had = !tree.isEmpty();
    tree.clear();
    flutterNavigationStack.clear();
    previousRouteId = ROOT_NODE_ID;
    if (accessibilityFocused != null) {
      sendEvent(
          obtainEvent(accessibilityFocused.id, AccessibilityEvent.TYPE_VIEW_ACCESSIBILITY_FOCUS_CLEARED),
          accessibilityFocused.id,
          false);
    }
    accessibilityFocused = null;
    inputFocused = null;
    lastInputFocused = null;
    hovered = null;
    if (had) {
      Log.i(TAG, "a11y tree nodes=0");
      sendWindowContentChangeEvent(0, AccessibilityEvent.CONTENT_CHANGE_TYPE_SUBTREE, false);
    }
  }

  // ---- Events ----

  @SuppressWarnings("deprecation")
  private AccessibilityEvent obtainEvent(int virtualViewId, int eventType) {
    AccessibilityEvent event = AccessibilityEvent.obtain(eventType);
    event.setPackageName(view.getContext().getPackageName());
    event.setSource(view, virtualViewId);
    return event;
  }

  private void sendWindowContentChangeEvent(int virtualViewId, int changeType, boolean live) {
    AccessibilityEvent event = obtainEvent(virtualViewId, AccessibilityEvent.TYPE_WINDOW_CONTENT_CHANGED);
    event.setContentChangeTypes(changeType);
    sendEvent(event, virtualViewId, live);
  }

  private void sendAccessibilityEvent(int viewId, int eventType) {
    if (!isActive()) {
      return;
    }
    sendEvent(obtainEvent(viewId, eventType), viewId, false);
  }

  /**
   * Sends an event to the accessibility manager, and logs it (the device tests read the log). When
   * the bridge is forced and no service runs, the event is only logged: the manager refuses events
   * while no service listens.
   */
  private void sendEvent(AccessibilityEvent event, int id, boolean live) {
    if (!isActive()) {
      return;
    }
    Log.i(
        TAG,
        "a11y event type="
            + typeName(event.getEventType())
            + " node="
            + id
            + " text="
            + eventText(event, id)
            + (live ? " live=1" : ""));
    if (manager == null || !manager.isEnabled()) {
      return;
    }
    ViewParent parent = view.getParent();
    if (parent != null) {
      parent.requestSendAccessibilityEvent(view, event);
    }
  }

  private static String typeName(int type) {
    String name = AccessibilityEvent.eventTypeToString(type);
    return name.startsWith("TYPE_") ? name.substring(5) : name;
  }

  /** The text of an event for the log: its own, else what the node says. */
  private String eventText(AccessibilityEvent event, int id) {
    List<CharSequence> texts = event.getText();
    if (texts != null && !texts.isEmpty()) {
      return TextUtils.join(", ", texts);
    }
    Node node = tree.get(id);
    CharSequence described = node == null || node.d == null ? null : describe(node);
    return described == null ? "" : described.toString();
  }

  /** What the node says: its text, or its content description. */
  private static CharSequence describe(Node node) {
    CharSequence text = (node.d.props & P_TEXT_FIELD) != 0 ? compose(node.d.text) : null;
    if (text == null) {
      text = compose(node.d.content);
    }
    if (text == null && node.d.tooltip != null) {
      text = node.d.tooltip;
    }
    return text;
  }

  /** Window name: the pane title of the view (API 28 and later), else a window state event. */
  private void onWindowNameChange(Node route) {
    String routeName = route.getRouteName();
    if (routeName == null) {
      // An empty name does not move the focus; a space does, without being read.
      routeName = " ";
    }
    if (Build.VERSION.SDK_INT >= 28) {
      view.setAccessibilityPaneTitle(routeName);
      if (isActive()) {
        Log.i(TAG, "a11y event type=WINDOW_STATE_CHANGED node=" + route.id + " text=" + routeName + " pane=1");
      }
    } else {
      AccessibilityEvent event = obtainEvent(route.id, AccessibilityEvent.TYPE_WINDOW_STATE_CHANGED);
      event.getText().add(routeName);
      sendEvent(event, route.id, false);
    }
  }

  // ---- Text ----

  /** The pieces joined with ", " ({@code SemanticsNode.getValueLabelHint}); null for none. */
  private static CharSequence compose(Styled[] pieces) {
    CharSequence result = null;
    for (Styled piece : pieces) {
      CharSequence word = piece.build();
      if (word.length() > 0) {
        result = result == null || result.length() == 0 ? word : TextUtils.concat(result, ", ", word);
      }
    }
    return result;
  }

  /** The pieces with the one of the value replaced (null removes it): the prediction of an action. */
  private static Styled[] withValue(Styled[] pieces, Styled value) {
    ArrayList<Styled> out = new ArrayList<>();
    if (value != null) {
      out.add(value);
    }
    for (Styled piece : pieces) {
      if (piece.kind != 0) {
        out.add(piece);
      }
    }
    return out.toArray(new Styled[0]);
  }

  // ---- AccessibilityNodeProvider ----

  @Override
  public AccessibilityNodeInfo createAccessibilityNodeInfo(int virtualViewId) {
    setAccessibleNavigation(true);
    if (virtualViewId == View.NO_ID) {
      @SuppressWarnings("deprecation")
      AccessibilityNodeInfo result = AccessibilityNodeInfo.obtain(view);
      view.onInitializeAccessibilityNodeInfo(result);
      if (tree.containsKey(ROOT_NODE_ID)) {
        result.addChild(view, ROOT_NODE_ID);
      }
      result.setImportantForAccessibility(false);
      return result;
    }
    Node node = tree.get(virtualViewId);
    if (node == null || node.d == null) {
      return null;
    }
    return nodeInfo(node);
  }

  @SuppressWarnings("deprecation")
  private AccessibilityNodeInfo nodeInfo(Node node) {
    final Data d = node.d;
    final int id = node.id;
    AccessibilityNodeInfo result = AccessibilityNodeInfo.obtain(view, id);
    result.setImportantForAccessibility((d.props & P_IMPORTANT) != 0);
    // A work-around for flutter/flutter#21030.
    result.setViewIdResourceName(d.identifier == null ? "" : d.identifier);
    result.setPackageName(view.getContext().getPackageName());
    result.setClassName(d.className);
    result.setSource(view, id);

    // configureFocusable
    result.setFocusable((d.props & P_FOCUSABLE) != 0);
    if (inputFocused != null) {
      result.setFocused(inputFocused.id == id);
    }
    if (accessibilityFocused != null) {
      result.setAccessibilityFocused(accessibilityFocused.id == id);
    }
    if (accessibilityFocused != null && accessibilityFocused.id == id) {
      result.addAction(AccessibilityNodeInfo.ACTION_CLEAR_ACCESSIBILITY_FOCUS);
    } else {
      result.addAction(AccessibilityNodeInfo.ACTION_ACCESSIBILITY_FOCUS);
    }
    // The actions that Go chose, with the labels of the ones that have one.
    for (int i = 0; i < d.actionIds.length; i++) {
      int action = d.actionIds[i];
      if ((action == AccessibilityNodeInfo.ACTION_EXPAND || action == AccessibilityNodeInfo.ACTION_COLLAPSE)
          && Build.VERSION.SDK_INT < 36) {
        continue;
      }
      result.addAction(new AccessibilityNodeInfo.AccessibilityAction(action, d.actionLabels[i]));
    }
    result.setClickable((d.props & P_CLICKABLE) != 0);
    result.setLongClickable((d.props & P_LONG_CLICKABLE) != 0);
    result.setScrollable((d.props & P_SCROLLABLE) != 0);
    result.setDismissable((d.props & P_DISMISSABLE) != 0);
    result.setSelected((d.props & P_SELECTED) != 0);
    result.setCanOpenPopup((d.props & P_CAN_OPEN_POPUP) != 0);
    if (Build.VERSION.SDK_INT >= 28) {
      result.setHeading((d.props & P_HEADING) != 0);
    }
    boolean textField = (d.props & P_TEXT_FIELD) != 0;
    boolean focusedField = textField && accessibilityFocused != null && accessibilityFocused.id == id;
    if ((d.props & P_LIVE_POLITE) != 0 || focusedField) {
      result.setLiveRegion(View.ACCESSIBILITY_LIVE_REGION_POLITE);
    }
    // configureTextField
    if (textField) {
      result.setPassword((d.props & P_PASSWORD) != 0);
      result.setEditable((d.props & P_EDITABLE) != 0);
      if (d.selectionBase != -1 && d.selectionExtent != -1) {
        result.setTextSelection(d.selectionBase, d.selectionExtent);
      }
      result.setMovementGranularities(d.granularities);
      if (d.maxTextLength >= 0) {
        result.setMaxTextLength(d.maxTextLength);
      }
    }
    // configureCollection and configureCollectionItem
    if (d.collectionRows > 0 && shouldSetCollectionInfo(node)) {
      if (Build.VERSION.SDK_INT < 33) {
        result.setCollectionInfo(
            AccessibilityNodeInfo.CollectionInfo.obtain(d.collectionRows, d.collectionColumns, false));
      } else {
        result.setCollectionInfo(
            new AccessibilityNodeInfo.CollectionInfo(d.collectionRows, d.collectionColumns, false));
      }
    }
    if (shouldSetCollectionItemInfo(node)) {
      Node parent = node.parent;
      boolean vertical = !(parent.hasAction(A_SCROLL_LEFT) || parent.hasAction(A_SCROLL_RIGHT));
      int index = parent.childrenInTraversalOrder.indexOf(node);
      boolean header = node.hasFlag(F_IS_HEADER);
      int row = vertical ? index : 0;
      int column = vertical ? 0 : index;
      if (Build.VERSION.SDK_INT < 33) {
        result.setCollectionItemInfo(
            AccessibilityNodeInfo.CollectionItemInfo.obtain(row, 1, column, 1, header));
      } else {
        result.setCollectionItemInfo(new AccessibilityNodeInfo.CollectionItemInfo(row, 1, column, 1, header));
      }
    }
    // configureLabelAndValue
    CharSequence content = compose(d.content);
    if (textField) {
      result.setText(compose(d.text));
      if (Build.VERSION.SDK_INT >= 28) {
        result.setHintText(compose(d.hint));
      }
    } else if ((d.flags & F_SCOPES_ROUTE) == 0) {
      if (Build.VERSION.SDK_INT < 28 && d.tooltip != null) {
        content = content != null ? content : "";
        content = content + "\n" + d.tooltip;
      }
      if (content != null) {
        result.setContentDescription(content);
      }
    }
    if (Build.VERSION.SDK_INT >= 28 && d.tooltip != null) {
      result.setTooltipText(d.tooltip);
      if (compose(d.content) == null) {
        result.setContentDescription(d.tooltip);
      }
    }
    // configureCheckable
    result.setCheckable((d.props & P_CHECKABLE) != 0);
    if ((d.props & P_CHECKABLE) != 0) {
      if (Build.VERSION.SDK_INT >= 36) {
        result.setChecked(
            d.checked == 2
                ? AccessibilityNodeInfo.CHECKED_STATE_PARTIAL
                : d.checked == 1
                    ? AccessibilityNodeInfo.CHECKED_STATE_TRUE
                    : AccessibilityNodeInfo.CHECKED_STATE_FALSE);
      } else {
        result.setChecked(d.checked == 1);
      }
    }
    // configureExpanded
    if (Build.VERSION.SDK_INT >= 36 && d.expanded != 0) {
      result.setExpandedState(
          d.expanded == 2
              ? AccessibilityNodeInfo.EXPANDED_STATE_FULL
              : AccessibilityNodeInfo.EXPANDED_STATE_COLLAPSED);
    }
    // The role: the range info of a progress bar.
    if (d.rangeType == 1) {
      result.setRangeInfo(
          AccessibilityNodeInfo.RangeInfo.obtain(
              AccessibilityNodeInfo.RangeInfo.RANGE_TYPE_FLOAT, d.rangeMin, d.rangeMax, d.rangeCurrent));
    } else if (d.rangeType == 2) {
      if (Build.VERSION.SDK_INT >= 36) {
        result.setRangeInfo(
            AccessibilityNodeInfo.RangeInfo.obtain(
                AccessibilityNodeInfo.RangeInfo.RANGE_TYPE_INDETERMINATE, 0.0f, 0.0f, 0.0f));
      } else {
        result.setRangeInfo(
            AccessibilityNodeInfo.RangeInfo.obtain(AccessibilityNodeInfo.RangeInfo.RANGE_TYPE_FLOAT, 0.0f, 0.0f, 0.0f));
      }
    }

    if (node.parent != null) {
      result.setParent(view, node.parent.id);
    } else {
      result.setParent(view);
    }
    if (node.previousNodeId != -1) {
      result.setTraversalAfter(view, node.previousNodeId);
    }
    Rect bounds = node.globalRect == null ? new Rect() : node.globalRect;
    if (node.parent != null && node.parent.globalRect != null) {
      Rect parentBounds = node.parent.globalRect;
      Rect inParent = new Rect(bounds);
      inParent.offset(-parentBounds.left, -parentBounds.top);
      result.setBoundsInParent(inParent);
    } else {
      result.setBoundsInParent(bounds);
    }
    result.setBoundsInScreen(boundsInScreen(bounds));
    result.setVisibleToUser(true);
    result.setEnabled((d.props & P_ENABLED) != 0);
    for (Node child : node.childrenInTraversalOrder) {
      if (child.d == null || child.hasFlag(F_IS_HIDDEN)) {
        continue;
      }
      result.addChild(view, child.id);
    }
    return result;
  }

  /** The bounds with the view's offset on the screen. */
  private Rect boundsInScreen(Rect bounds) {
    Rect inScreen = new Rect(bounds);
    int[] location = new int[2];
    view.getLocationOnScreen(location);
    inScreen.offset(location[0], location[1]);
    return inScreen;
  }

  /**
   * A list is a collection for TalkBack only while it holds the accessibility focus, and when the
   * focus leaves it (so that "out of list" is said); a list of one child is none.
   */
  private boolean shouldSetCollectionInfo(Node node) {
    if (node.d.scrollChildren <= 1) {
      return false;
    }
    final Node target = node;
    boolean focusInside = accessibilityFocused != null && accessibilityFocused.getAncestor(o -> o == target) != null;
    boolean focusInScroller =
        accessibilityFocused != null && accessibilityFocused.getAncestor(o -> o.hasFlag(F_HAS_IMPLICIT_SCROLLING)) != null;
    return focusInside || !focusInScroller;
  }

  private boolean shouldSetCollectionItemInfo(Node node) {
    return node.parent != null
        && node.parent.d != null
        && shouldSetCollectionInfo(node.parent)
        && node.parent.hasFlag(F_HAS_IMPLICIT_SCROLLING);
  }

  @Override
  public AccessibilityNodeInfo findFocus(int focus) {
    // The input focus falls back to the accessibility focus.
    if (focus == AccessibilityNodeInfo.FOCUS_INPUT && inputFocused != null) {
      return createAccessibilityNodeInfo(inputFocused.id);
    }
    if ((focus == AccessibilityNodeInfo.FOCUS_INPUT || focus == AccessibilityNodeInfo.FOCUS_ACCESSIBILITY)
        && accessibilityFocused != null) {
      return createAccessibilityNodeInfo(accessibilityFocused.id);
    }
    return null;
  }

  // ---- Actions ----

  /** Sends an action to Go (Native.a11yAction). */
  private static void dispatch(int id, int action, byte[] args) {
    Native.a11yAction(id, action, args);
  }

  private static void dispatch(int id, int action) {
    dispatch(id, action, new byte[0]);
  }

  private static byte[] ints(int... values) {
    ByteBuffer b = ByteBuffer.allocate(4 * values.length).order(ByteOrder.LITTLE_ENDIAN);
    for (int v : values) {
      b.putInt(v);
    }
    return b.array();
  }

  /** The byte offset in the UTF-8 of {@code s} of the UTF-16 offset: the framework counts bytes. */
  static int utf8Offset(String s, int utf16) {
    if (s == null) {
      return Math.max(0, utf16);
    }
    int end = Math.max(0, Math.min(utf16, s.length()));
    if (end > 0 && end < s.length() && Character.isHighSurrogate(s.charAt(end - 1))) {
      end++;
    }
    return s.substring(0, end).getBytes(StandardCharsets.UTF_8).length;
  }

  @Override
  public boolean performAction(int virtualViewId, int accessibilityAction, Bundle arguments) {
    final Node node = tree.get(virtualViewId);
    if (node == null || node.d == null) {
      return false;
    }
    final Data d = node.d;
    switch (accessibilityAction) {
      case AccessibilityNodeInfo.ACTION_CLICK:
        dispatch(virtualViewId, A_TAP);
        return true;
      case AccessibilityNodeInfo.ACTION_LONG_CLICK:
        dispatch(virtualViewId, A_LONG_PRESS);
        return true;
      case AccessibilityNodeInfo.ACTION_SCROLL_FORWARD:
        if (node.hasAction(A_SCROLL_UP)) {
          dispatch(virtualViewId, A_SCROLL_UP);
        } else if (node.hasAction(A_SCROLL_LEFT)) {
          dispatch(virtualViewId, A_SCROLL_LEFT);
        } else if (node.hasAction(A_INCREASE)) {
          // The value that the action will give is shown at once.
          d.value = d.increased.length > 0 ? d.increased[0].text : null;
          predictValue(d, d.increased.length > 0 ? d.increased[0] : null);
          sendAccessibilityEvent(virtualViewId, AccessibilityEvent.TYPE_VIEW_SELECTED);
          dispatch(virtualViewId, A_INCREASE);
        } else {
          return false;
        }
        return true;
      case AccessibilityNodeInfo.ACTION_SCROLL_BACKWARD:
        if (node.hasAction(A_SCROLL_DOWN)) {
          dispatch(virtualViewId, A_SCROLL_DOWN);
        } else if (node.hasAction(A_SCROLL_RIGHT)) {
          dispatch(virtualViewId, A_SCROLL_RIGHT);
        } else if (node.hasAction(A_DECREASE)) {
          d.value = d.decreased.length > 0 ? d.decreased[0].text : null;
          predictValue(d, d.decreased.length > 0 ? d.decreased[0] : null);
          sendAccessibilityEvent(virtualViewId, AccessibilityEvent.TYPE_VIEW_SELECTED);
          dispatch(virtualViewId, A_DECREASE);
        } else {
          return false;
        }
        return true;
      case AccessibilityNodeInfo.ACTION_PREVIOUS_AT_MOVEMENT_GRANULARITY:
        return performCursorMoveAction(node, arguments, false);
      case AccessibilityNodeInfo.ACTION_NEXT_AT_MOVEMENT_GRANULARITY:
        return performCursorMoveAction(node, arguments, true);
      case AccessibilityNodeInfo.ACTION_CLEAR_ACCESSIBILITY_FOCUS:
        // The node is reset before the event, or TalkBack may think it is still focused.
        if (accessibilityFocused != null && accessibilityFocused.id == virtualViewId) {
          accessibilityFocused = null;
        }
        dispatch(virtualViewId, A_DID_LOSE_ACCESSIBILITY_FOCUS);
        sendAccessibilityEvent(virtualViewId, AccessibilityEvent.TYPE_VIEW_ACCESSIBILITY_FOCUS_CLEARED);
        return true;
      case AccessibilityNodeInfo.ACTION_ACCESSIBILITY_FOCUS:
        if (accessibilityFocused == null) {
          // Android does not invalidate the view when it focuses a node.
          view.invalidate();
        }
        // The node is set before the event, or TalkBack may think it is not focused yet.
        accessibilityFocused = node;
        dispatch(virtualViewId, A_DID_GAIN_ACCESSIBILITY_FOCUS);
        sendAccessibilityEvent(virtualViewId, AccessibilityEvent.TYPE_VIEW_ACCESSIBILITY_FOCUSED);
        if (node.hasAction(A_INCREASE) || node.hasAction(A_DECREASE)) {
          // A SeekBar announces itself after this event.
          sendAccessibilityEvent(virtualViewId, AccessibilityEvent.TYPE_VIEW_SELECTED);
        }
        return true;
      case ACTION_SHOW_ON_SCREEN:
        dispatch(virtualViewId, A_SHOW_ON_SCREEN);
        return true;
      case AccessibilityNodeInfo.ACTION_SET_SELECTION:
        {
          int base;
          int extent;
          if (arguments != null
              && arguments.containsKey(AccessibilityNodeInfo.ACTION_ARGUMENT_SELECTION_START_INT)
              && arguments.containsKey(AccessibilityNodeInfo.ACTION_ARGUMENT_SELECTION_END_INT)) {
            base = arguments.getInt(AccessibilityNodeInfo.ACTION_ARGUMENT_SELECTION_START_INT);
            extent = arguments.getInt(AccessibilityNodeInfo.ACTION_ARGUMENT_SELECTION_END_INT);
          } else {
            // Clear the selection.
            base = d.selectionExtent;
            extent = d.selectionExtent;
          }
          dispatch(virtualViewId, A_SET_SELECTION, ints(utf8Offset(d.value, base), utf8Offset(d.value, extent)));
          // Voice Access expects the node to change at once; a wrong guess is corrected by the next update.
          d.selectionBase = base;
          d.selectionExtent = extent;
          return true;
        }
      case AccessibilityNodeInfo.ACTION_COPY:
        dispatch(virtualViewId, A_COPY);
        return true;
      case AccessibilityNodeInfo.ACTION_CUT:
        dispatch(virtualViewId, A_CUT);
        return true;
      case AccessibilityNodeInfo.ACTION_PASTE:
        dispatch(virtualViewId, A_PASTE);
        return true;
      case AccessibilityNodeInfo.ACTION_DISMISS:
        dispatch(virtualViewId, A_DISMISS);
        return true;
      case AccessibilityNodeInfo.ACTION_SET_TEXT:
        {
          String newText = "";
          if (arguments != null && arguments.containsKey(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE)) {
            CharSequence given = arguments.getCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE);
            newText = given == null ? "" : given.toString();
          }
          dispatch(virtualViewId, A_SET_TEXT, newText.getBytes(StandardCharsets.UTF_8));
          // The new text is shown at once, without the old spans.
          d.value = newText.isEmpty() ? null : newText;
          Styled plain = newText.isEmpty() ? null : Styled.plain(0, newText);
          d.text = withValue(d.text, plain);
          d.content = withValue(d.content, plain);
          return true;
        }
      case AccessibilityNodeInfo.ACTION_EXPAND:
        dispatch(virtualViewId, A_EXPAND);
        return true;
      case AccessibilityNodeInfo.ACTION_COLLAPSE:
        dispatch(virtualViewId, A_COLLAPSE);
        return true;
      case AccessibilityNodeInfo.ACTION_FOCUS:
        // Deviation: Flutter's bridge does not handle the input focus action; the framework has one.
        if (node.hasAction(A_FOCUS)) {
          dispatch(virtualViewId, A_FOCUS);
          return true;
        }
        return false;
      default:
        {
          // A custom action: the id is the framework's plus FIRST_RESOURCE_ID.
          final int customId = accessibilityAction - FIRST_RESOURCE_ID;
          for (int id : d.actionIds) {
            if (id == accessibilityAction) {
              dispatch(virtualViewId, A_CUSTOM_ACTION, ints(customId));
              return true;
            }
          }
          return false;
        }
    }
  }

  /** The value that an increase or a decrease will give, in the text of the node. */
  private static void predictValue(Data d, Styled next) {
    d.content = withValue(d.content, next);
    d.text = withValue(d.text, next);
  }

  private boolean performCursorMoveAction(Node node, Bundle arguments, boolean forward) {
    if (arguments == null) {
      return false;
    }
    final Data d = node.d;
    final int granularity = arguments.getInt(AccessibilityNodeInfo.ACTION_ARGUMENT_MOVEMENT_GRANULARITY_INT);
    final boolean extendSelection = arguments.getBoolean(AccessibilityNodeInfo.ACTION_ARGUMENT_EXTEND_SELECTION_BOOLEAN);
    final byte[] extend = new byte[] {(byte) (extendSelection ? 1 : 0)};
    // Voice Access expects the node to change at once; a wrong guess is corrected by the next update.
    final int previousBase = d.selectionBase;
    final int previousExtent = d.selectionExtent;
    predictCursorMovement(d, granularity, forward, extendSelection);
    if (previousBase != d.selectionBase || previousExtent != d.selectionExtent) {
      final String value = d.value != null ? d.value : "";
      AccessibilityEvent selection = obtainEvent(node.id, AccessibilityEvent.TYPE_VIEW_TEXT_SELECTION_CHANGED);
      selection.getText().add(value);
      selection.setFromIndex(d.selectionBase);
      selection.setToIndex(d.selectionExtent);
      selection.setItemCount(value.length());
      sendEvent(selection, node.id, false);
    }
    switch (granularity) {
      case AccessibilityNodeInfo.MOVEMENT_GRANULARITY_CHARACTER:
        if (forward && node.hasAction(A_MOVE_CURSOR_FORWARD_BY_CHARACTER)) {
          dispatch(node.id, A_MOVE_CURSOR_FORWARD_BY_CHARACTER, extend);
          return true;
        }
        if (!forward && node.hasAction(A_MOVE_CURSOR_BACKWARD_BY_CHARACTER)) {
          dispatch(node.id, A_MOVE_CURSOR_BACKWARD_BY_CHARACTER, extend);
          return true;
        }
        break;
      case AccessibilityNodeInfo.MOVEMENT_GRANULARITY_WORD:
        if (forward && node.hasAction(A_MOVE_CURSOR_FORWARD_BY_WORD)) {
          dispatch(node.id, A_MOVE_CURSOR_FORWARD_BY_WORD, extend);
          return true;
        }
        if (!forward && node.hasAction(A_MOVE_CURSOR_BACKWARD_BY_WORD)) {
          dispatch(node.id, A_MOVE_CURSOR_BACKWARD_BY_WORD, extend);
          return true;
        }
        break;
      case AccessibilityNodeInfo.MOVEMENT_GRANULARITY_LINE:
      case AccessibilityNodeInfo.MOVEMENT_GRANULARITY_PARAGRAPH:
      case AccessibilityNodeInfo.MOVEMENT_GRANULARITY_PAGE:
        // Not supported by the framework: the request is taken and nothing is sent.
        return true;
      default:
        break;
    }
    return false;
  }

  /** {@code predictCursorMovement}: where the cursor will be, in UTF-16 units. */
  private static void predictCursorMovement(Data node, int granularity, boolean forward, boolean extendSelection) {
    if (node.selectionExtent < 0 || node.selectionBase < 0) {
      return;
    }
    final String value = node.value != null ? node.value : "";
    switch (granularity) {
      case AccessibilityNodeInfo.MOVEMENT_GRANULARITY_CHARACTER:
        if (forward && node.selectionExtent < value.length()) {
          node.selectionExtent += 1;
        } else if (!forward && node.selectionExtent > 0) {
          node.selectionExtent -= 1;
        }
        break;
      case AccessibilityNodeInfo.MOVEMENT_GRANULARITY_WORD:
        if (forward && node.selectionExtent < value.length()) {
          Pattern pattern = Pattern.compile("\\p{L}(\\b)");
          Matcher result = pattern.matcher(value.substring(node.selectionExtent));
          // The first result is discarded: it is the word the cursor is in; the next word is wanted.
          result.find();
          if (result.find()) {
            node.selectionExtent += result.start(1);
          } else {
            node.selectionExtent = value.length();
          }
        } else if (!forward && node.selectionExtent > 0) {
          // The last beginning of a word.
          Pattern pattern = Pattern.compile("(?s:.*)(\\b)\\p{L}");
          Matcher result = pattern.matcher(value.substring(0, node.selectionExtent));
          if (result.find()) {
            node.selectionExtent = result.start(1);
          }
        }
        break;
      case AccessibilityNodeInfo.MOVEMENT_GRANULARITY_LINE:
        if (forward && node.selectionExtent < value.length()) {
          Pattern pattern = Pattern.compile("(?!^)(\\n)");
          Matcher result = pattern.matcher(value.substring(node.selectionExtent));
          if (result.find()) {
            node.selectionExtent += result.start(1);
          } else {
            node.selectionExtent = value.length();
          }
        } else if (!forward && node.selectionExtent > 0) {
          Pattern pattern = Pattern.compile("(?s:.*)(\\n)");
          Matcher result = pattern.matcher(value.substring(0, node.selectionExtent));
          if (result.find()) {
            node.selectionExtent = result.start(1);
          } else {
            node.selectionExtent = 0;
          }
        }
        break;
      case AccessibilityNodeInfo.MOVEMENT_GRANULARITY_PARAGRAPH:
      case AccessibilityNodeInfo.MOVEMENT_GRANULARITY_PAGE:
        node.selectionExtent = forward ? value.length() : 0;
        break;
      default:
        break;
    }
    if (!extendSelection) {
      node.selectionBase = node.selectionExtent;
    }
  }

  // ---- Touch exploration ----

  /**
   * A hover event of the view: touch exploration moves the finger over the nodes. Returns true when
   * it was handled.
   */
  boolean onHover(MotionEvent event) {
    if (manager == null || !manager.isTouchExplorationEnabled() || tree.isEmpty()) {
      return false;
    }
    Node root = tree.get(ROOT_NODE_ID);
    if (root == null || root.d == null) {
      return false;
    }
    int action = event.getAction();
    if (action == MotionEvent.ACTION_HOVER_ENTER || action == MotionEvent.ACTION_HOVER_MOVE) {
      handleTouchExploration(event.getX(), event.getY());
    } else if (action == MotionEvent.ACTION_HOVER_EXIT) {
      onTouchExplorationExit();
    } else {
      Log.d(TAG, "unexpected accessibility hover event: " + event);
      return false;
    }
    return true;
  }

  private void onTouchExplorationExit() {
    if (hovered != null) {
      sendAccessibilityEvent(hovered.id, AccessibilityEvent.TYPE_VIEW_HOVER_EXIT);
      hovered = null;
    }
  }

  private void handleTouchExploration(float x, float y) {
    Node root = tree.get(ROOT_NODE_ID);
    if (root == null || root.d == null) {
      return;
    }
    Node under = root.hitTest(new float[] {x, y, 0, 1});
    if (under != hovered) {
      // ENTER before EXIT is how Android wants it.
      if (under != null) {
        sendAccessibilityEvent(under.id, AccessibilityEvent.TYPE_VIEW_HOVER_ENTER);
      }
      if (hovered != null) {
        sendAccessibilityEvent(hovered.id, AccessibilityEvent.TYPE_VIEW_HOVER_EXIT);
      }
      hovered = under;
    }
  }

  // ---- The dev events script ----

  /** The AccessibilityNodeInfo action of a name of the dev events script, or 0. */
  private static int devAction(String name) {
    switch (name) {
      case "click":
        return AccessibilityNodeInfo.ACTION_CLICK;
      case "longclick":
        return AccessibilityNodeInfo.ACTION_LONG_CLICK;
      case "scrollforward":
        return AccessibilityNodeInfo.ACTION_SCROLL_FORWARD;
      case "scrollbackward":
        return AccessibilityNodeInfo.ACTION_SCROLL_BACKWARD;
      case "focus":
        return AccessibilityNodeInfo.ACTION_FOCUS;
      case "a11yfocus":
        return AccessibilityNodeInfo.ACTION_ACCESSIBILITY_FOCUS;
      case "a11yclear":
        return AccessibilityNodeInfo.ACTION_CLEAR_ACCESSIBILITY_FOCUS;
      case "dismiss":
        return AccessibilityNodeInfo.ACTION_DISMISS;
      case "copy":
        return AccessibilityNodeInfo.ACTION_COPY;
      case "cut":
        return AccessibilityNodeInfo.ACTION_CUT;
      case "paste":
        return AccessibilityNodeInfo.ACTION_PASTE;
      case "expand":
        return AccessibilityNodeInfo.ACTION_EXPAND;
      case "collapse":
        return AccessibilityNodeInfo.ACTION_COLLAPSE;
      default:
        return 0;
    }
  }

  /** Whether the node says the label in its text, content description or hint. */
  private static boolean says(Node node, String label) {
    Styled[][] lists = {node.d.text, node.d.content, node.d.hint};
    for (Styled[] list : lists) {
      CharSequence text = compose(list);
      if (text != null && text.toString().contains(label)) {
        return true;
      }
    }
    return false;
  }

  private static boolean offers(Node node, int action) {
    if (action == AccessibilityNodeInfo.ACTION_ACCESSIBILITY_FOCUS
        || action == AccessibilityNodeInfo.ACTION_CLEAR_ACCESSIBILITY_FOCUS) {
      return true;
    }
    if (action == AccessibilityNodeInfo.ACTION_FOCUS) {
      return node.hasAction(A_FOCUS);
    }
    for (int id : node.d.actionIds) {
      if (id == action) {
        return true;
      }
    }
    return false;
  }

  /** The first node of the subtree (this one first, then the children in traversal order) that offers the action. */
  private static Node findOffering(Node node, int action) {
    if (node.d != null && !node.hasFlag(F_IS_HIDDEN) && offers(node, action)) {
      return node;
    }
    for (Node child : node.childrenInTraversalOrder) {
      Node found = findOffering(child, action);
      if (found != null) {
        return found;
      }
    }
    return null;
  }

  /**
   * The dev events script's {@code a11y:action,label}: the action is performed through {@code
   * performAction}, as TalkBack would, on the first node (by id) whose text, content description or
   * hint contains the label and that offers the action; when none does, on the first node of the
   * subtree of the first match that does (a label on a container whose list scrolls); when none does,
   * on the first match.
   */
  void injectAction(String key) {
    int comma = key.indexOf(',');
    if (comma < 0) {
      Log.w(TAG, "a11y dev action not understood: " + key);
      return;
    }
    String name = key.substring(0, comma);
    String label = key.substring(comma + 1);
    int action = devAction(name);
    if (action == 0) {
      Log.w(TAG, "a11y dev action unknown: " + name);
      return;
    }
    List<Integer> ids = new ArrayList<>(tree.keySet());
    Collections.sort(ids);
    Node match = null;
    for (int id : ids) {
      Node node = tree.get(id);
      if (node == null || node.d == null || node.hasFlag(F_IS_HIDDEN) || !says(node, label)) {
        continue;
      }
      if (offers(node, action)) {
        match = node;
        break;
      }
      if (match == null) {
        match = node;
      }
    }
    if (match == null) {
      Log.w(TAG, "a11y dev action " + name + ": no node says \"" + label + "\" (" + tree.size() + " nodes)");
      return;
    }
    if (!offers(match, action)) {
      Node inside = findOffering(match, action);
      if (inside != null) {
        match = inside;
      }
    }
    boolean done = performAction(match.id, action, null);
    Log.i(TAG, "a11y dev action " + name + " node=" + match.id + " label=" + label + " result=" + done);
  }
}
