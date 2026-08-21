//go:build js && wasm

package winbox

import (
	"math"
	"strings"
	"testing"
)

// style reads back what setStyle wrote, which goes through setProperty.
func style(w *WinBox, name string) string {
	return w.DOM.Get("style").Call("getPropertyValue", name).String()
}

// fresh installs a clean DOM and empties the window stack, which is package
// state that would otherwise carry between tests.
func fresh(t *testing.T) {
	t.Helper()
	closeAll()
	installFakeDOM()
	idCounter = 0
	indexCounter = 10
	t.Cleanup(closeAll)
}

// closeAll shuts every open window, which is what takes its element back out
// of the page. Under a real browser the page belongs to the test harness, so
// the windows have to be removed one by one rather than by clearing the body.
func closeAll() {
	for _, w := range append([]*WinBox(nil), stackWin...) {
		w.Close(true)
	}
	stackWin, stackMin, stackDock = nil, nil, nil
}

// ── Unit ─────────────────────────────────────────────────────────────────────

func TestUnitPxIsTakenLiterally(t *testing.T) {
	if got := Px(240).parse(1000, 100); got != 240 {
		t.Errorf("Px(240) = %v, want 240 whatever the space", got)
	}
}

// A percentage is of the space available, which is what lets a window be half
// the screen without knowing how big the screen is.
func TestUnitPctIsOfTheAvailableSpace(t *testing.T) {
	for _, tc := range []struct {
		pct, base, want float64
	}{
		{50, 1000, 500},
		{100, 1000, 1000},
		{0, 1000, 0},
		{33, 1000, 330},
		{50, 999, 500}, // rounds rather than truncates
	} {
		if got := Pct(tc.pct).parse(tc.base, 0); got != tc.want {
			t.Errorf("Pct(%v) of %v = %v, want %v", tc.pct, tc.base, got, tc.want)
		}
	}
}

// Center is the midpoint of the leftover space, so the window's own size has
// to come into it.
func TestUnitCenterAccountsForTheWindowSize(t *testing.T) {
	if got := Center.parse(1000, 200); got != 400 {
		t.Errorf("centering a 200 wide window in 1000 gave %v, want 400", got)
	}
	if got := Center.parse(1000, 0); got != 500 {
		t.Errorf("centering a zero-width window in 1000 gave %v, want 500", got)
	}
	// A window wider than the space lands at a negative offset rather than
	// being clamped, which is what keeps it centered rather than left-aligned.
	//
	// -99 rather than -100: the rounding is Trunc(x + 0.5), which goes toward
	// zero, so it rounds up on the negative side. That is what the JavaScript
	// this was ported from does, and matching it is the point.
	if got := Center.parse(100, 300); got != -99 {
		t.Errorf("centering a 300 wide window in 100 gave %v, want -99", got)
	}
}

func TestUnitEdgeIsTheFarSide(t *testing.T) {
	if got := Right.parse(1000, 200); got != 800 {
		t.Errorf("right-aligning a 200 wide window in 1000 gave %v, want 800", got)
	}
	if got := Bottom.parse(800, 600); got != 200 {
		t.Errorf("bottom-aligning a 600 tall window in 800 gave %v, want 200", got)
	}
}

// falsy mirrors the JavaScript truthiness the port replaced: an unset unit and
// a zero-pixel one are both "not given", which is why Px(0) cannot be used to
// mean the left edge.
func TestUnitFalsy(t *testing.T) {
	for _, u := range []Unit{{}, Px(0)} {
		if !u.falsy() {
			t.Errorf("%v should be falsy", u)
		}
	}
	for _, u := range []Unit{Px(1), Px(-1), Pct(0), Center, Right} {
		if u.falsy() {
			t.Errorf("%v should not be falsy", u)
		}
	}
}

// ── Geometry ─────────────────────────────────────────────────────────────────

func TestNewAppliesTheGivenSize(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "t", Width: Px(400), Height: Px(300)})
	if w.Width != 400 || w.Height != 300 {
		t.Errorf("size is %vx%v, want 400x300", w.Width, w.Height)
	}
	if got := style(w, "width"); got != "400px" {
		t.Errorf("the element is %q wide, want 400px", got)
	}
}

func TestResizeInPercentIsOfTheRoomAvailable(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "t", Width: Px(100), Height: Px(100)})
	w.Resize(Pct(50), Pct(25))
	// Rounded the way the code rounds — Trunc(x + 0.5) — since a viewport with
	// an odd dimension does not divide evenly and the browser's is whatever it
	// is.
	if want := math.Trunc(rootW/100*50 + 0.5); w.Width != want {
		t.Errorf("50%% of %v gave %v, want %v", rootW, w.Width, want)
	}
	if want := math.Trunc(rootH/100*25 + 0.5); w.Height != want {
		t.Errorf("25%% of %v gave %v, want %v", rootH, w.Height, want)
	}
}

// MaxWidth caps the stored size, so a later percentage resize cannot exceed it.
func TestResizeIsCappedByMax(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "t", Width: Px(100), Height: Px(100), MaxWidth: Px(300), MaxHeight: Px(200)})
	w.Resize(Pct(100), Pct(100))
	if w.Width != 300 || w.Height != 200 {
		t.Errorf("size is %vx%v, want the maximum 300x200", w.Width, w.Height)
	}
}

// MinWidth floors what is applied to the element, but deliberately not what is
// stored: the stored size is what the caller asked for, so growing the minimum
// later does not permanently inflate the window.
func TestResizeIsFlooredByMinOnTheElement(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "t", Width: Px(500), Height: Px(500), MinWidth: Px(200), MinHeight: Px(150)})
	w.Resize(Px(10), Px(10))
	if got := style(w, "width"); got != "200px" {
		t.Errorf("the element is %q wide, want the minimum 200px", got)
	}
	if got := style(w, "height"); got != "150px" {
		t.Errorf("the element is %q tall, want the minimum 150px", got)
	}
	if w.Width != 10 || w.Height != 10 {
		t.Errorf("stored size is %vx%v, want the 10x10 that was asked for", w.Width, w.Height)
	}
}

// Two unset units mean "apply what is stored", which is how a direct change to
// Width or Height is pushed to the element.
func TestResizeWithNoUnitsReappliesTheStoredSize(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "t", Width: Px(400), Height: Px(300)})
	w.Width = 123
	w.Height = 456
	w.applySize()
	if got := style(w, "width"); got != "123px" {
		t.Errorf("the element is %q wide, want 123px", got)
	}
	if w.Width != 123 {
		t.Errorf("re-applying changed the stored width to %v", w.Width)
	}
}

func TestMoveSetsThePosition(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "t", Width: Px(100), Height: Px(100)})
	w.Move(Px(30), Px(40))
	if w.X != 30 || w.Y != 40 {
		t.Errorf("position is %v,%v, want 30,40", w.X, w.Y)
	}
	if got := style(w, "left"); got != "30px" {
		t.Errorf("the element is at %q, want 30px", got)
	}
}

func TestMoveCenterUsesTheWindowSize(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "t", Width: Px(400), Height: Px(200)})
	w.Move(Center, Center)
	if want := math.Trunc((rootW-400)/2 + 0.5); w.X != want {
		t.Errorf("centered x is %v, want %v", w.X, want)
	}
	if want := math.Trunc((rootH-200)/2 + 0.5); w.Y != want {
		t.Errorf("centered y is %v, want %v", w.Y, want)
	}
}

func TestMoveWithNoUnitsReappliesTheStoredPosition(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "t", Width: Px(100), Height: Px(100)})
	w.X, w.Y = 77, 88
	w.applyPos()
	if got := style(w, "left"); got != "77px" {
		t.Errorf("the element is at %q, want 77px", got)
	}
}

// The raw paths move the element without touching the stored position, which
// is what a drag does until it ends.
func TestMoveRawLeavesTheStoredPositionAlone(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "t", Width: Px(100), Height: Px(100)})
	w.Move(Px(10), Px(10))
	w.moveRaw(500, 600)
	if w.X != 10 || w.Y != 10 {
		t.Errorf("a raw move changed the stored position to %v,%v", w.X, w.Y)
	}
	if got := style(w, "left"); got != "500px" {
		t.Errorf("the element is at %q, want the raw 500px", got)
	}
}

func TestResizeRawLeavesTheStoredSizeAlone(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "t", Width: Px(100), Height: Px(100)})
	w.resizeRaw(640, 480)
	if w.Width != 100 || w.Height != 100 {
		t.Errorf("a raw resize changed the stored size to %vx%v", w.Width, w.Height)
	}
	if got := style(w, "width"); got != "640px" {
		t.Errorf("the element is %q wide, want the raw 640px", got)
	}
}

// The callbacks are how a host follows a window; they have to report the
// applied geometry rather than the requested one.
func TestOnMoveAndOnResizeReportWhatWasApplied(t *testing.T) {
	fresh(t)
	var moved, resized [2]float64
	w := New(&Options{
		Title: "t", Width: Px(100), Height: Px(100), MinWidth: Px(200),
		OnMove:   func(_ *WinBox, x, y float64) { moved = [2]float64{x, y} },
		OnResize: func(_ *WinBox, wd, h float64) { resized = [2]float64{wd, h} },
	})
	w.Move(Px(11), Px(22))
	if moved != [2]float64{11, 22} {
		t.Errorf("OnMove got %v, want 11,22", moved)
	}
	w.Resize(Px(50), Px(60))
	if resized[0] != 200 {
		t.Errorf("OnResize got width %v, want the 200 the minimum forced", resized[0])
	}
}

// ── Classes and title ────────────────────────────────────────────────────────

func TestClassHelpers(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "t"})

	if w.HasClass("nope") {
		t.Error("a class that was never added is reported present")
	}
	w.AddClass("mine")
	if !w.HasClass("mine") {
		t.Error("an added class is not reported present")
	}
	w.AddClass("mine") // twice must not double up
	w.RemoveClass("mine")
	if w.HasClass("mine") {
		t.Error("a removed class is still present")
	}
	w.ToggleClass("mine")
	if !w.HasClass("mine") {
		t.Error("toggle did not add the class")
	}
	w.ToggleClass("mine")
	if w.HasClass("mine") {
		t.Error("toggle did not remove the class")
	}
}

func TestSetTitle(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "first"})
	if w.Title != "first" {
		t.Errorf("Title = %q, want first", w.Title)
	}
	w.SetTitle("second")
	if w.Title != "second" {
		t.Errorf("Title = %q after SetTitle, want second", w.Title)
	}
	if got := getByClass(w.DOM, "wb-title").Get("textContent").String(); got != "second" {
		t.Errorf("the titlebar says %q, want second", got)
	}
}

// ── Stack ────────────────────────────────────────────────────────────────────

func TestStackHoldsEveryOpenWindow(t *testing.T) {
	fresh(t)
	a := New(&Options{Title: "a"})
	b := New(&Options{Title: "b"})
	if got := len(Stack()); got != 2 {
		t.Fatalf("the stack holds %d windows, want 2", got)
	}
	_ = a
	_ = b
}

func TestCloseRemovesTheWindowFromTheStack(t *testing.T) {
	fresh(t)
	a := New(&Options{Title: "a"})
	New(&Options{Title: "b"})

	if a.Close(true) {
		t.Error("Close(true) reported it was canceled")
	}
	if got := len(Stack()); got != 1 {
		t.Errorf("the stack holds %d windows after a close, want 1", got)
	}
	for _, w := range Stack() {
		if w == a {
			t.Error("the closed window is still in the stack")
		}
	}
}

// A host can refuse a close, which is what an unsaved-changes prompt needs.
func TestOnCloseCanCancelTheClose(t *testing.T) {
	fresh(t)
	w := New(&Options{
		Title:   "a",
		OnClose: func(_ *WinBox, force bool) bool { return !force },
	})
	if !w.Close(false) {
		t.Error("Close(false) was not canceled by OnClose")
	}
	if len(Stack()) != 1 {
		t.Error("a canceled close still removed the window")
	}
	if w.Close(true) {
		t.Error("Close(true) was canceled; force should override")
	}
	if len(Stack()) != 0 {
		t.Error("a forced close left the window in the stack")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "a"})
	w.Close(true)
	w.Close(true) // must not panic or remove someone else
	if len(Stack()) != 0 {
		t.Errorf("the stack holds %d windows", len(Stack()))
	}
}

func TestHideAndShow(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "a"})
	w.Hide()
	if !w.Hidden {
		t.Error("the window is not marked hidden")
	}
	w.Show()
	if w.Hidden {
		t.Error("the window is still marked hidden after Show")
	}
}

func TestMinimizeAndRestore(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "a", Width: Px(400), Height: Px(300)})
	w.Move(Px(50), Px(60))

	w.Minimize()
	if !w.Min {
		t.Error("the window is not marked minimized")
	}
	w.Restore()
	if w.Min {
		t.Error("the window is still marked minimized after Restore")
	}
	// Restoring has to give back the geometry it had, or a minimize round trip
	// loses where the window was.
	if w.X != 50 || w.Y != 60 {
		t.Errorf("restored to %v,%v, want 50,60", w.X, w.Y)
	}
	if w.Width != 400 || w.Height != 300 {
		t.Errorf("restored to %vx%v, want 400x300", w.Width, w.Height)
	}
}

func TestMaximizeAndRestoreKeepTheOriginalGeometry(t *testing.T) {
	fresh(t)
	w := New(&Options{Title: "a", Width: Px(400), Height: Px(300)})
	w.Move(Px(50), Px(60))

	w.Maximize()
	if !w.Max {
		t.Error("the window is not marked maximized")
	}
	w.Restore()
	if w.Max {
		t.Error("the window is still marked maximized after Restore")
	}
	if w.X != 50 || w.Y != 60 || w.Width != 400 || w.Height != 300 {
		t.Errorf("restored to %v,%v %vx%v, want 50,60 400x300", w.X, w.Y, w.Width, w.Height)
	}
}

// A window manager parks a minimized window in a slot along the bottom, and a
// maximized one over the whole viewport, by the same move-and-resize path a
// drag takes — so OnMove and OnResize fire for both. An app persisting geometry
// from those callbacks, which is what they are for, needs to be able to tell
// being PARKED from being ARRANGED, and Min and Max are the only signal there
// is. They are therefore set BEFORE the window is parked, not after.
//
// The failure this pins down was not theoretical: a control panel saved the
// minimize slot as its remembered size, came back from the minimize as a bare
// title bar, and came back that way on every later load, because 251x35 had
// been written to localStorage.
func TestParkingDoesNotLookLikeAnArrangementToACallback(t *testing.T) {
	fresh(t)
	// What is saved is what the callback was TOLD, not what the window has
	// stored: the raw path leaves the stored geometry alone, so an app reading
	// w.Width would never have seen this — and would also never know the window
	// had been dragged, which is the whole reason to have the arguments.
	saved := [4]float64{}
	w := New(&Options{
		Title: "a", Width: Px(400), Height: Px(300),
		OnMove: func(w *WinBox, x, y float64) {
			if w.Min || w.Max {
				return // parked by the manager, not placed by the person
			}
			saved[0], saved[1] = x, y
		},
		OnResize: func(w *WinBox, wd, h float64) {
			if w.Min || w.Max {
				return
			}
			saved[2], saved[3] = wd, h
		},
	})
	w.Move(Px(50), Px(60))
	want := [4]float64{50, 60, 400, 300}
	if saved != want {
		t.Fatalf("after arranging, saved %v, want %v", saved, want)
	}

	w.Minimize()
	if saved != want {
		t.Errorf("minimizing overwrote the saved geometry with %v, want %v left alone", saved, want)
	}
	w.Restore()
	if saved != want {
		t.Errorf("after a minimize round trip, saved %v, want %v", saved, want)
	}

	w.Maximize()
	if saved != want {
		t.Errorf("maximizing overwrote the saved geometry with %v, want %v left alone", saved, want)
	}
	w.Restore()
	if saved != want {
		t.Errorf("after a maximize round trip, saved %v, want %v", saved, want)
	}
}

// ── CSS ──────────────────────────────────────────────────────────────────────

// The stylesheet goes in once however many windows are built. Counted by its
// own id rather than by the size of the head, since a real page has other
// things in there.
func TestInjectCSSOnlyAddsTheStylesheetOnce(t *testing.T) {
	fresh(t)
	for i := 0; i < 5; i++ {
		InjectCSS()
	}
	head := document.Get("head")
	n := 0
	for i := 0; i < head.Get("children").Length(); i++ {
		if head.Get("children").Index(i).Get("id").String() == styleID {
			n++
		}
	}
	if n != 1 {
		t.Errorf("the head holds %d copies of the stylesheet after five InjectCSS calls, want 1", n)
	}
}

func TestCSSIsNotEmpty(t *testing.T) {
	if !strings.Contains(CSS(), "winbox") && len(CSS()) == 0 {
		t.Error("the embedded stylesheet is empty")
	}
}
