//go:build js && wasm

package winbox

import "testing"

// TestClickingAFramedPageRaisesItsWindow.
//
// The bug this pins: a window whose content is an <iframe> could not be clicked
// to the front. mousedown does not cross a browsing context, so the handler
// that raises a window when its body is clicked never fired for framed
// content; only the title bar and the resize edges worked.
//
// The first attempt at this inferred the click from window blur plus
// document.activeElement, and did not work: focusing a frame does not fire
// blur on the parent, and activeElement is already the frame whenever focus
// last landed there. What is asserted here is the click itself, delivered on
// the frame's own document, which is what the fix listens to.
func TestClickingAFramedPageRaisesItsWindow(t *testing.T) {
	fresh(t)
	framed := New(&Options{Title: "framed", Width: Px(400), Height: Px(300)})
	frame := document.Call("createElement", "iframe")
	framed.Body.Call("appendChild", frame)

	// A second window opened after it is the focused one, as opening focuses.
	other := New(&Options{Title: "other", Width: Px(400), Height: Px(300)})
	if !other.Focused || framed.Focused {
		t.Fatalf("precondition: expected the last-opened window to hold focus")
	}

	// The click lands in the frame's document; what identifies the window is
	// the frame ELEMENT, which is where the walk up to the window starts.
	focusWindowOwning(frame)

	if !framed.Focused {
		t.Errorf("the framed window was not raised by a click inside its frame")
	}
	if other.Focused {
		t.Errorf("the previously focused window kept focus")
	}
}

// TestAClickOutsideAnyWindowChangesNothing. The walk up from the clicked
// element must simply find no window and stop, leaving the stack alone.
func TestAClickOutsideAnyWindowChangesNothing(t *testing.T) {
	fresh(t)
	first := New(&Options{Title: "first", Width: Px(400), Height: Px(300)})
	second := New(&Options{Title: "second", Width: Px(400), Height: Px(300)})

	focusWindowOwning(document.Call("createElement", "div"))

	if first.Focused || !second.Focused {
		t.Errorf("focus moved: first=%v second=%v, want first=false second=true",
			first.Focused, second.Focused)
	}
}

// TestWireFrameFocusListensInsideSameOriginFrames. The wiring must reach the
// frame's own document — that is the only place the click exists — and must
// not wire the same frame twice when the tree is re-rendered.
func TestWireFrameFocusListensInsideSameOriginFrames(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "framed", Width: Px(400), Height: Px(300)})
	frame := document.Call("createElement", "iframe")
	w.Body.Call("appendChild", frame)

	wireFrameFocus()
	if !frame.Get("__wbFocusWired").Truthy() {
		t.Fatalf("the frame was not wired")
	}
	doc := frame.Get("contentDocument")
	if !doc.Truthy() {
		t.Skip("the fake frame exposes no contentDocument")
	}
	got := doc.Get("_listeners").Get("length").Int()
	wireFrameFocus() // a re-render must not stack a second listener
	if again := doc.Get("_listeners").Get("length").Int(); again != got {
		t.Errorf("re-wiring added listeners: %d then %d", got, again)
	}
}
