//go:build js && wasm

package winbox

import "testing"

// TestAPressOnATitleBarControlIsNotADrag.
//
// The bug this pins: a tab strip mounted in a window's title bar could not be
// clicked. The bar's mousedown listener runs in the capture phase and stops
// the event, so a press on a tab armed a window drag instead of reaching the
// tab; the release then fell wherever the window had moved to and no click
// formed. A listener on the strip could never run first. The drag handle has
// to stand aside, and it does so for anything under an element carrying
// NoDragClass — but not for the handle's own empty space, which still drags.
func TestAPressOnATitleBarControlIsNotADrag(t *testing.T) {
	fresh(t)
	handle := document.Call("createElement", "div")
	strip := document.Call("createElement", "div")
	tab := document.Call("createElement", "div")
	label := document.Call("createElement", "span")
	handle.Call("appendChild", strip)
	strip.Call("appendChild", tab)
	tab.Call("appendChild", label)

	if pressIsOwned(label, handle) {
		t.Fatalf("nothing is marked yet; the press should be a drag")
	}
	tab.Get("classList").Call("add", NoDragClass)
	if !pressIsOwned(label, handle) {
		t.Errorf("a press on the label inside a marked tab was treated as a drag")
	}
	if !pressIsOwned(tab, handle) {
		t.Errorf("a press on the marked tab itself was treated as a drag")
	}
	if pressIsOwned(strip, handle) {
		t.Errorf("a press on the strip's empty space, above the mark, was not a drag")
	}
	if pressIsOwned(handle, handle) {
		t.Errorf("a press on the handle itself was not a drag")
	}
}

// TestAMarkOutsideTheHandleDoesNotCount. The walk stops at the handle: a
// no-drag mark on the window, or anywhere above the bar, is not a mark on the
// bar's controls.
func TestAMarkOutsideTheHandleDoesNotCount(t *testing.T) {
	fresh(t)
	outer := document.Call("createElement", "div")
	outer.Get("classList").Call("add", NoDragClass)
	handle := document.Call("createElement", "div")
	title := document.Call("createElement", "div")
	outer.Call("appendChild", handle)
	handle.Call("appendChild", title)

	if pressIsOwned(title, handle) {
		t.Errorf("a mark above the handle stopped a drag on the title")
	}
}
