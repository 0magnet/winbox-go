//go:build js && wasm

// Command winbox-js is a wasm module whose only job is to install the
// WinBox.js-compatible constructor (github.com/0magnet/winbox-go/jsapi) on
// globalThis, for pages whose JavaScript already calls `new WinBox({...})`.
//
//	tinygo build -o winbox.wasm -target wasm -no-debug ./cmd/winbox-js
//
// It sets globalThis.__winboxReady to a resolved promise once the constructor
// exists, so page code can wait rather than race the module's instantiation.
package main

import (
	"syscall/js"

	"github.com/0magnet/winbox-go/jsapi"
)

func main() {
	jsapi.InstallGlobal()
	// Signal readiness for the common case where the page loaded this module
	// asynchronously: a resolved promise is awaitable whether the waiter
	// arrives before or after the module started.
	js.Global().Set("__winboxReady", js.Global().Get("Promise").Call("resolve", js.Global().Get("WinBox")))
	if resolve := js.Global().Get("__winboxResolve"); resolve.Type() == js.TypeFunction {
		resolve.Invoke(js.Global().Get("WinBox"))
	}
	select {}
}
