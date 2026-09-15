//go:build js && wasm

package winbox

import (
	"syscall/js"
	"testing"
)

// setActiveElement points the fake document at el, which is what a browser does
// when focus moves into a frame.
func setActiveElement(el js.Value) {
	document.Set("activeElement", el)
}

// TestClickingAFramedPageRaisesItsWindow.
//
// The bug this pins: a window whose content is an <iframe> could not be
// clicked to the front. mousedown does not cross a browsing context, so the
// handler that raises a window on a body click never fired for framed content
// — while a window drawing its own DOM raised normally, which made the framed
// one look uniquely broken.
func TestClickingAFramedPageRaisesItsWindow(t *testing.T) {
	fresh(t)
	framed := New(&Options{Title: "framed", Width: Px(400), Height: Px(300)})
	frame := document.Call("createElement", "iframe")
	framed.Body.Call("appendChild", frame)

	// A second window opened after it is the focused one, as opening focuses.
	other := New(&Options{Title: "other", Width: Px(400), Height: Px(300)})
	if !other.Focused {
		t.Fatalf("precondition: the window opened last should hold focus")
	}
	if framed.Focused {
		t.Fatalf("precondition: the framed window should not hold focus")
	}

	// Clicking into the frame: the page blurs and activeElement becomes the
	// <iframe> element itself.
	setActiveElement(frame)
	focusWindowOfActiveFrame()

	if !framed.Focused {
		t.Errorf("the framed window was not raised by a click inside its frame")
	}
	if other.Focused {
		t.Errorf("the previously focused window kept focus")
	}
}

// TestBlurWithNoFrameFocusedChangesNothing. Leaving the page entirely, or
// focus landing on an ordinary control, must not reorder the stack.
func TestBlurWithNoFrameFocusedChangesNothing(t *testing.T) {
	fresh(t)
	first := New(&Options{Title: "first", Width: Px(400), Height: Px(300)})
	second := New(&Options{Title: "second", Width: Px(400), Height: Px(300)})

	setActiveElement(document.Call("createElement", "button"))
	focusWindowOfActiveFrame()

	if first.Focused || !second.Focused {
		t.Errorf("focus moved: first=%v second=%v, want first=false second=true",
			first.Focused, second.Focused)
	}
}
