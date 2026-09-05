//go:build js && wasm

package jsapi

import (
	"syscall/js"
	"testing"

	winbox "github.com/0magnet/winbox-go"
)

// obj builds a JS options object from key/value pairs.
func obj(kv ...any) js.Value {
	o := js.Global().Get("Object").New()
	for i := 0; i+1 < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

// The option parsers are the only logic this package owns — past them it is
// the core library, which has its own tests. What matters here is that the
// loose values JavaScript callers really pass arrive as the right Unit.

func TestUnitReadsEveryFormWinBoxJSAccepts(t *testing.T) {
	for _, tc := range []struct {
		name string
		val  any
		want winbox.Unit
	}{
		{"a number is pixels", 250, winbox.Px(250)},
		{"a plain string is pixels", "250", winbox.Px(250)},
		{"a px string ignores the suffix", "250px", winbox.Px(250)},
		{"a percentage is of the available space", "40%", winbox.Pct(40)},
		{"center", "center", winbox.Center},
		{"center is case-insensitive", "CENTER", winbox.Center},
		{"right is the far side", "right", winbox.Right},
		{"bottom is the far side", "bottom", winbox.Bottom},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := unit(obj("v", tc.val), "v"); got != tc.want {
				t.Fatalf("unit(%v) = %#v, want %#v", tc.val, got, tc.want)
			}
		})
	}
}

func TestUnitOfAMissingKeyIsUnset(t *testing.T) {
	// An unset Unit is what tells the core to pick its own default, so a key
	// the caller simply did not pass must not become Px(0).
	if got := unit(obj(), "width"); got != (winbox.Unit{}) {
		t.Fatalf("a missing key produced %#v, want the zero Unit", got)
	}
	if got := unit(js.Undefined(), "width"); got != (winbox.Unit{}) {
		t.Fatalf("a missing options object produced %#v, want the zero Unit", got)
	}
}

func TestNumAcceptsNumbersAndNumericStrings(t *testing.T) {
	// browse.js passes border:"1"; other callers pass border:1. Both are
	// current in the wild, so both have to mean the same thing.
	if got := num(obj("border", "1"), "border"); got != 1 {
		t.Fatalf(`num("1") = %v, want 1`, got)
	}
	if got := num(obj("border", 1), "border"); got != 1 {
		t.Fatalf("num(1) = %v, want 1", got)
	}
	if got := num(obj("border", "nonsense"), "border"); got != 0 {
		t.Fatalf("an unparseable string gave %v, want 0", got)
	}
	if got := num(obj(), "border"); got != 0 {
		t.Fatalf("a missing key gave %v, want 0", got)
	}
}

func TestClassListAcceptsStringsAndArrays(t *testing.T) {
	arr := js.Global().Get("Array").New()
	arr.Call("push", "skywire-wb")
	arr.Call("push", "no-full")

	for _, tc := range []struct {
		name string
		val  any
		want []string
	}{
		{"an array of names", arr, []string{"skywire-wb", "no-full"}},
		{"a space-separated string", "skywire-wb no-full", []string{"skywire-wb", "no-full"}},
		{"a single name", "modal", []string{"modal"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := classList(obj("class", tc.val))
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}

	if got := classList(obj()); got != nil {
		t.Fatalf("a missing class key gave %v, want nil", got)
	}
}

func TestFalseArgOnlyMatchesLiteralFalse(t *testing.T) {
	// minimize(false) restores while minimize() minimizes, so anything other
	// than a literal false — a missing argument, 0, "" — must not be read as
	// the toggle-off form.
	if !falseArg([]js.Value{js.ValueOf(false)}) {
		t.Fatal("literal false was not recognized")
	}
	for _, v := range []any{true, 0, "", nil, 1} {
		if falseArg([]js.Value{js.ValueOf(v)}) {
			t.Fatalf("%#v was read as the toggle-off form", v)
		}
	}
	if falseArg(nil) {
		t.Fatal("a missing argument was read as the toggle-off form")
	}
}

func TestUnitArgReadsPositionalArguments(t *testing.T) {
	args := []js.Value{js.ValueOf(120), js.ValueOf("center")}
	if got := unitArg(args, 0); got != winbox.Px(120) {
		t.Fatalf("first argument = %#v, want Px(120)", got)
	}
	if got := unitArg(args, 1); got != winbox.Center {
		t.Fatalf("second argument = %#v, want Center", got)
	}
	// move() and resize() with no arguments re-apply the stored geometry,
	// which the core expresses as the zero Unit.
	if got := unitArg(args, 2); got != (winbox.Unit{}) {
		t.Fatalf("a missing argument gave %#v, want the zero Unit", got)
	}
}
