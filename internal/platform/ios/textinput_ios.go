//go:build ios

package ios

import (
	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the UIKit side of the text input (SPEC §30.5): a hidden view that implements UITextInput, so that the soft keyboard, the
// input methods (Chinese, Japanese, Korean), the autocorrection, dictation and the hardware keyboard's text compose and type text in the
// field; the model that it reads and writes is in textinput.go. A position of the document is an object of a class of Go whose only datum
// is its UTF-16 offset, and a range is one with two (the offset of its start and its end in the two halves of one word), so that neither
// needs memory of its own.

// The UITextLayoutDirection values that a position is asked for.
const (
	directionRight = 2
	directionLeft  = 3
	directionUp    = 4
	directionDown  = 5
)

// registerText registers the classes of the text input: the view, the positions and the ranges. The view is a UIView and conforms to
// UITextInput; its positions and ranges are subclasses of UITextPosition and UITextRange.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterTextInputPlugin.mm FlutterTextInputView, FlutterTextPosition, FlutterTextRange
func registerText() (view, position, rng *objcx.RegisteredClass, err error) {
	if position, err = objcx.RegisterClass(objcx.ClassDef{Name: "TextPosition", Super: "UITextPosition"}); err != nil {
		return nil, nil, nil, err
	}
	rng, err = objcx.RegisterClass(objcx.ClassDef{
		Name: "TextRange", Super: "UITextRange",
		Methods: []objcx.Method{
			{Name: "start", Fn: func(self objc.ID, _ objc.SEL) objc.ID { s, _ := rangeOfObject(self); return newPosition(position, s) }},
			{Name: "end", Fn: func(self objc.ID, _ objc.SEL) objc.ID { _, e := rangeOfObject(self); return newPosition(position, e) }},
			{Name: "isEmpty", Fn: func(self objc.ID, _ objc.SEL) bool { s, e := rangeOfObject(self); return s == e }},
		},
	})
	if err != nil {
		return nil, nil, nil, err
	}
	view, err = objcx.RegisterClass(objcx.ClassDef{
		Name: "TextInputView", Super: "UIView", Protocols: []string{"UITextInput"},
		Methods: textInputMethods(position, rng),
	})
	return view, position, rng, err
}

// newPosition makes a position object (autoreleased) at an offset.
func newPosition(cls *objcx.RegisteredClass, offset int) objc.ID {
	return objcx.Autorelease(cls.New(objcx.Handle(offset)))
}

// newRange makes a range object (autoreleased) from a start to an end.
func newRange(cls *objcx.RegisteredClass, r textRange) objc.ID {
	return objcx.Autorelease(cls.New(objcx.Handle(uint64(r.Start)<<32 | uint64(r.End))))
}

// offsetOf is the offset of a position object, 0 for nil.
func offsetOf(pos objc.ID) int {
	if pos == 0 {
		return 0
	}
	return int(objcx.HandleOf(pos))
}

// rangeOfObject is the start and the end of a range object, 0 and 0 for nil.
func rangeOfObject(r objc.ID) (start, end int) {
	if r == 0 {
		return 0, 0
	}
	h := uint64(objcx.HandleOf(r))
	return int(h >> 32), int(h & 0xffffffff)
}

// orderedRange is the range between two positions, whichever comes first.
func orderedRange(a, b int) textRange {
	if b < a {
		a, b = b, a
	}
	return textRange{a, b}
}

// textInputMethods are the methods of the text input view: UIKeyInput, UITextInput, the traits that the keyboard reads, and the responder
// methods that keep the system's own menu away (the framework draws the toolbar of the selection).
//
// Flutter: FlutterTextInputPlugin.mm FlutterTextInputView
func textInputMethods(positions, ranges *objcx.RegisteredClass) []objcx.Method {
	// on runs fn for the window of the input view under the guard.
	on := func(self objc.ID, fn func(w *window)) {
		if w := windowOf(self); w != nil && !w.closed {
			w.p.guard(func() { fn(w) })
		}
	}
	// ret runs fn and returns what it gives, zero when there is no window.
	retID := func(self objc.ID, fn func(w *window) objc.ID) (out objc.ID) {
		on(self, func(w *window) { out = fn(w) })
		return out
	}
	retInt := func(self objc.ID, fn func(w *window) int64) (out int64) {
		on(self, func(w *window) { out = fn(w) })
		return out
	}
	trait := func(name string, fn func(c textConfig) int64) objcx.Method {
		return objcx.Method{Name: name, Fn: func(self objc.ID, _ objc.SEL) int64 {
			return retInt(self, func(w *window) int64 { return fn(w.text.config) })
		}}
	}
	pos := func(offset int) objc.ID { return newPosition(positions, offset) }
	clamp := func(w *window, off int) (int, bool) {
		n := w.text.length()
		return min(max(off, 0), n), off >= 0 && off <= n
	}
	return []objcx.Method{
		// UIKeyInput.
		{Name: "hasText", Fn: func(self objc.ID, _ objc.SEL) (has bool) {
			on(self, func(w *window) { has = w.text.hasText() })
			return has
		}},
		{Name: "insertText:", Fn: func(self objc.ID, _ objc.SEL, str objc.ID) {
			on(self, func(w *window) {
				if ev, ok := w.text.insertText(nsString(str)); ok {
					w.sendTextInput(ev)
				}
			})
		}},
		{Name: "deleteBackward", Fn: func(self objc.ID, _ objc.SEL) {
			on(self, func(w *window) {
				if ev, ok := w.text.deleteBackward(); ok {
					w.sendTextInput(ev)
				}
			})
		}},
		// The document, the selection and the marked text.
		{Name: "textInRange:", Fn: func(self objc.ID, _ objc.SEL, r objc.ID) objc.ID {
			return retID(self, func(w *window) objc.ID {
				s, e := rangeOfObject(r)
				return objcx.Autorelease(objcx.NewString(w.text.substring(textRange{s, e})))
			})
		}},
		{Name: "replaceRange:withText:", Fn: func(self objc.ID, _ objc.SEL, r, str objc.ID) {
			on(self, func(w *window) {
				s, e := rangeOfObject(r)
				if ev, ok := w.text.replaceRange(textRange{s, e}, nsString(str)); ok {
					w.sendTextInput(ev)
				}
			})
		}},
		{Name: "selectedTextRange", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
			return retID(self, func(w *window) objc.ID {
				if !w.text.enabled {
					return 0
				}
				return newRange(ranges, w.text.selectedRange())
			})
		}},
		{Name: "setSelectedTextRange:", Fn: func(self objc.ID, _ objc.SEL, r objc.ID) {
			on(self, func(w *window) {
				s, e := rangeOfObject(r)
				if ev, ok := w.text.setSelectedRange(textRange{s, e}); ok {
					w.sendTextInput(ev)
				}
			})
		}},
		{Name: "markedTextRange", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
			return retID(self, func(w *window) objc.ID {
				if r, ok := w.text.markedRange(); ok {
					return newRange(ranges, r)
				}
				return 0
			})
		}},
		{Name: "markedTextStyle", Fn: func(objc.ID, objc.SEL) objc.ID { return 0 }},
		{Name: "setMarkedTextStyle:", Fn: func(objc.ID, objc.SEL, objc.ID) {}},
		{Name: "setMarkedText:selectedRange:", Fn: func(self objc.ID, _ objc.SEL, str objc.ID, loc, length uint64) {
			on(self, func(w *window) {
				if ev, ok := w.text.setMarkedText(nsString(str), textRange{int(loc), int(loc + length)}); ok {
					w.sendTextInput(ev)
				}
			})
		}},
		{Name: "unmarkText", Fn: func(self objc.ID, _ objc.SEL) {
			on(self, func(w *window) {
				if ev, ok := w.text.unmarkText(); ok {
					w.sendTextInput(ev)
				}
			})
		}},
		{Name: "inputDelegate", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
			return retID(self, func(w *window) objc.ID { return w.inputDelegate })
		}},
		{Name: "setInputDelegate:", Fn: func(self objc.ID, _ objc.SEL, d objc.ID) {
			on(self, func(w *window) { w.inputDelegate = d }) // assigned, not retained: UIKit owns the delegate
		}},
		{Name: "tokenizer", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
			return retID(self, func(w *window) objc.ID {
				if w.tokenizer == 0 {
					w.tokenizer = class("UITextInputStringTokenizer").Send(sel("alloc")).Send(sel("initWithTextInput:"), self)
				}
				return w.tokenizer
			})
		}},
		// Positions and ranges.
		{Name: "beginningOfDocument", Fn: func(self objc.ID, _ objc.SEL) objc.ID { return pos(0) }},
		{Name: "endOfDocument", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
			return retID(self, func(w *window) objc.ID { return pos(w.text.length()) })
		}},
		{Name: "textRangeFromPosition:toPosition:", Fn: func(_ objc.ID, _ objc.SEL, a, b objc.ID) objc.ID {
			return newRange(ranges, orderedRange(offsetOf(a), offsetOf(b)))
		}},
		{Name: "positionFromPosition:offset:", Fn: func(self objc.ID, _ objc.SEL, p objc.ID, offset int64) objc.ID {
			return retID(self, func(w *window) objc.ID {
				off, ok := clamp(w, offsetOf(p)+int(offset))
				if !ok {
					return 0
				}
				return pos(off)
			})
		}},
		{Name: "positionFromPosition:inDirection:offset:", Fn: func(self objc.ID, _ objc.SEL, p objc.ID, dir, offset int64) objc.ID {
			return retID(self, func(w *window) objc.ID {
				at := offsetOf(p)
				switch dir {
				case directionRight:
					at += int(offset)
				case directionLeft:
					at -= int(offset)
				}
				off, ok := clamp(w, at)
				if !ok {
					return 0
				}
				return pos(off)
			})
		}},
		{Name: "comparePosition:toPosition:", Fn: func(_ objc.ID, _ objc.SEL, a, b objc.ID) int64 {
			x, y := offsetOf(a), offsetOf(b)
			switch {
			case x < y:
				return -1 // NSOrderedAscending
			case x > y:
				return 1
			}
			return 0
		}},
		{Name: "offsetFromPosition:toPosition:", Fn: func(_ objc.ID, _ objc.SEL, a, b objc.ID) int64 { return int64(offsetOf(b) - offsetOf(a)) }},
		{Name: "positionWithinRange:farthestInDirection:", Fn: func(_ objc.ID, _ objc.SEL, r objc.ID, dir int64) objc.ID {
			s, e := rangeOfObject(r)
			if dir == directionRight || dir == directionDown {
				return pos(e)
			}
			return pos(s)
		}},
		{Name: "characterRangeByExtendingPosition:inDirection:", Fn: func(self objc.ID, _ objc.SEL, p objc.ID, dir int64) objc.ID {
			return retID(self, func(w *window) objc.ID {
				at := offsetOf(p)
				switch dir {
				case directionLeft, directionUp:
					return newRange(ranges, textRange{0, at})
				case directionRight, directionDown:
					return newRange(ranges, textRange{at, w.text.length()})
				}
				return newRange(ranges, textRange{at, at})
			})
		}},
		{Name: "baseWritingDirectionForPosition:inDirection:", Fn: func(objc.ID, objc.SEL, objc.ID, int64) int64 { return -1 }}, // natural
		{Name: "setBaseWritingDirection:forRange:", Fn: func(objc.ID, objc.SEL, int64, objc.ID) {}},
		// Geometry: the rectangle of the caret stands for every rectangle; the framework gives no more.
		{Name: "firstRectForRange:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) (r darwin.CGRect) {
			on(self, func(w *window) { r = w.caretCGRect() })
			return r
		}},
		{Name: "caretRectForPosition:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) (r darwin.CGRect) {
			on(self, func(w *window) { r = w.caretCGRect() })
			return r
		}},
		{Name: "selectionRectsForRange:", Fn: func(objc.ID, objc.SEL, objc.ID) objc.ID { return class("NSArray").Send(sel("array")) }},
		{Name: "closestPositionToPoint:", Fn: func(self objc.ID, _ objc.SEL, _, _ float64) objc.ID {
			return retID(self, func(w *window) objc.ID { return pos(w.text.length()) })
		}},
		{Name: "closestPositionToPoint:withinRange:", Fn: func(_ objc.ID, _ objc.SEL, _, _ float64, r objc.ID) objc.ID {
			_, e := rangeOfObject(r)
			return pos(e)
		}},
		{Name: "characterRangeAtPoint:", Fn: func(objc.ID, objc.SEL, float64, float64) objc.ID { return 0 }},
		{Name: "textStylingAtPosition:inDirection:", Fn: func(objc.ID, objc.SEL, objc.ID, int64) objc.ID { return 0 }},
		// The responder: the view takes the focus, and the system's menu of copy and paste stays away.
		{Name: "canBecomeFirstResponder", Fn: func(objc.ID, objc.SEL) bool { return true }},
		{Name: "canPerformAction:withSender:", Fn: func(objc.ID, objc.SEL, objc.SEL, objc.ID) bool { return false }},
		// The traits that the keyboard reads (UITextInputTraits).
		trait("keyboardType", func(c textConfig) int64 { return int64(c.Keyboard) }),
		trait("returnKeyType", func(c textConfig) int64 { return int64(c.ReturnKey) }),
		trait("autocapitalizationType", func(c textConfig) int64 { return int64(c.Capitalization) }),
		trait("autocorrectionType", func(c textConfig) int64 { return int64(c.Autocorrect) }),
		trait("spellCheckingType", func(c textConfig) int64 { return int64(c.SpellCheck) }),
		trait("smartQuotesType", func(c textConfig) int64 { return int64(c.SmartQuotes) }),
		trait("smartDashesType", func(c textConfig) int64 { return int64(c.SmartDashes) }),
		trait("smartInsertDeleteType", func(c textConfig) int64 { return int64(c.SmartQuotes) }),
		{Name: "isSecureTextEntry", Fn: func(self objc.ID, _ objc.SEL) (secure bool) {
			on(self, func(w *window) { secure = w.text.config.Secure })
			return secure
		}},
		{Name: "enablesReturnKeyAutomatically", Fn: func(objc.ID, objc.SEL) bool { return false }},
		{Name: "textContentType", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
			return retID(self, func(w *window) objc.ID {
				if t := w.text.config.ContentType; t != "" {
					return objcx.Autorelease(objcx.NewString(t))
				}
				return 0
			})
		}},
	}
}

// caretCGRect is the rectangle of the caret in the view's points.
func (w *window) caretCGRect() darwin.CGRect {
	r, ok := w.text.caretRect()
	if !ok {
		return darwin.CGRect{}
	}
	return darwin.CGRect{Origin: darwin.CGPoint{X: r.Left, Y: r.Top}, Size: darwin.CGSize{Width: r.Right - r.Left, Height: r.Bottom - r.Top}}
}

// sendTextInput gives the framework an event of the input method.
func (w *window) sendTextInput(ev ui.TextInputEvent) {
	if h := w.p.handler; h != nil {
		h.OnTextInput(ev)
	}
}

// SetTextInput takes the state of the focused text field (platform.Window.SetTextInput): the first state with a field shows the soft keyboard,
// the last hides it, and a change that the app made is told to UIKit around the change so that it drops what it had learned (its
// suggestions and its marked text).
//
// Flutter: FlutterTextInputPlugin.mm showTextInput, hideTextInput, setTextInputEditingState:
func (w *window) SetTextInput(st ui.TextInputState) {
	if w.closed || w.view == 0 {
		return
	}
	objcx.WithPool(func() {
		ch := w.text.setState(st)
		switch {
		case ch.enabled && !ch.wasEnabled:
			w.showKeyboard()
		case !ch.enabled && ch.wasEnabled:
			w.hideKeyboard()
		}
		if !ch.enabled || w.inputView == 0 {
			return
		}
		if ch.configChanged && ch.wasEnabled {
			w.inputView.Send(sel("reloadInputViews"))
		}
		if ch.textChanged || ch.markedLost {
			w.changeTextAround(func() {})
		}
	})
}

// changeTextAround tells the input delegate that the text and the selection are about to change and then that they did.
//
// Flutter: FlutterTextInputPlugin.mm updateEditingState (textWillChange, textDidChange)
func (w *window) changeTextAround(fn func()) {
	d := w.inputDelegate
	if d == 0 {
		fn()
		return
	}
	d.Send(sel("selectionWillChange:"), w.inputView)
	d.Send(sel("textWillChange:"), w.inputView)
	fn()
	d.Send(sel("textDidChange:"), w.inputView)
	d.Send(sel("selectionDidChange:"), w.inputView)
}

// ensureInputView makes the text input view and puts it in the view, where it takes no room and shows nothing.
func (w *window) ensureInputView() {
	if w.inputView != 0 {
		return
	}
	w.inputView = w.p.s.textView.New(w.handle)
	w.inputView.Send(sel("setFrame:"), darwin.CGRect{Size: darwin.CGSize{Width: 1, Height: 1}})
	w.inputView.Send(sel("setAlpha:"), 0.0)
	w.inputView.Send(sel("setUserInteractionEnabled:"), false)
	w.view.Send(sel("addSubview:"), w.inputView)
}

// showKeyboard gives the focus to the text input view: the soft keyboard comes (with no hardware keyboard), and the traits are read.
func (w *window) showKeyboard() {
	w.ensureInputView()
	w.inputView.Send(sel("reloadInputViews"))
	w.inputView.Send(sel("becomeFirstResponder"))
}

// hideKeyboard takes the focus from the text input view and gives it back to the view, which takes the hardware keyboard's presses.
func (w *window) hideKeyboard() {
	if w.inputView == 0 {
		return
	}
	w.inputView.Send(sel("resignFirstResponder"))
	w.view.Send(sel("becomeFirstResponder"))
}

// closeInput releases the text input view and what it holds.
func (w *window) closeInput() {
	if w.inputView != 0 {
		w.inputView.Send(sel("removeFromSuperview"))
		objcx.Release(w.inputView)
		w.inputView = 0
	}
	if w.tokenizer != 0 {
		objcx.Release(w.tokenizer)
		w.tokenizer = 0
	}
	w.inputDelegate = 0
}
