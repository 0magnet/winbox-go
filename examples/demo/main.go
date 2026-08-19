//go:build js && wasm

// Demo for winbox-go, mirroring the winbox.js demo page.
package main

import (
	"fmt"
	"syscall/js"

	winbox "github.com/0magnet/winbox-go"
)

var document = js.Global().Get("document")

// onClick wires a page button to a Go handler. The js.Func intentionally
// lives for the lifetime of the page.
func onClick(id string, fn func()) {
	btn := document.Call("getElementById", id)
	if !btn.Truthy() {
		return
	}
	btn.Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		fn()
		return nil
	}))
}

const lorem = `<p style="padding: 0 14px">Lorem ipsum dolor sit amet, consetetur sadipscing elitr,
sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua.
At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata
sanctus est Lorem ipsum dolor sit amet.</p>`

func main() {
	winbox.New(&winbox.Options{
		Title: "WinBox in Go",
		HTML:  "<h2 style='padding:0 14px'>Hello from Go!</h2>" + lorem,
		X:     winbox.Center,
		Y:     winbox.Center,
		Width: winbox.Px(520),
	})

	count := 0

	onClick("btn-basic", func() {
		count++
		winbox.New(&winbox.Options{
			Title: fmt.Sprintf("Window #%d", count),
			HTML:  lorem,
			X:     winbox.Px(float64(30 + count*30)),
			Y:     winbox.Px(float64(30 + count*30)),
		})
	})

	onClick("btn-color", func() {
		winbox.New(&winbox.Options{
			Title:      "Custom Color",
			Background: "#ff005d",
			Border:     4,
			X:          winbox.Center,
			Y:          winbox.Center,
			Width:      winbox.Px(450),
			Height:     winbox.Px(250),
			HTML:       lorem,
		})
	})

	onClick("btn-modal", func() {
		winbox.New(&winbox.Options{
			Title: "Modal Window",
			Modal: true,
			HTML:  lorem,
		})
	})

	onClick("btn-limits", func() {
		winbox.New(&winbox.Options{
			Title:    "Limited Viewport",
			X:        winbox.Right,
			Width:    winbox.Pct(40),
			Height:   winbox.Pct(60),
			MinWidth: winbox.Px(200),
			Top:      winbox.Px(60),
			Bottom:   winbox.Px(60),
			HTML:     "<p style='padding:0 14px'>40% x 60%, right-aligned, viewport padded 60px top/bottom. Try maximizing.</p>",
		})
	})

	onClick("btn-mount", func() {
		winbox.New(&winbox.Options{
			Title: "Mounted DOM",
			Mount: document.Call("getElementById", "content").Call("cloneNode", true),
			X:     winbox.Px(120),
			Y:     winbox.Px(120),
		})
	})

	onClick("btn-iframe", func() {
		winbox.New(&winbox.Options{
			Title: "iframe: Wikipedia",
			URL:   "https://wikipedia.org",
			X:     winbox.Px(180),
			Y:     winbox.Px(180),
			OnLoad: func(w *winbox.WinBox) {
				fmt.Println("iframe loaded")
			},
		})
	})

	onClick("btn-popup", func() {
		winbox.New(&winbox.Options{
			Title:  "Popup (no controls)",
			Class:  []string{"no-min", "no-max", "no-full", "no-resize", "no-move"},
			X:      winbox.Center,
			Y:      winbox.Center,
			Width:  winbox.Px(340),
			Height: winbox.Px(160),
			HTML:   "<p style='padding:0 14px'>A fixed popup. Only closable.</p>",
		})
	})

	onClick("btn-confirm", func() {
		winbox.New(&winbox.Options{
			Title: "Close Confirmation",
			HTML:  "<p style='padding:0 14px'>Try to close this window.</p>",
			X:     winbox.Center,
			Y:     winbox.Center,
			OnClose: func(w *winbox.WinBox, force bool) bool {
				return !force && !js.Global().Call("confirm", "Close window?").Bool()
			},
		})
	})

	// Docking — the one feature here that WinBox.js does not have. Each of
	// these is an ordinary window that happens to be pinned to an edge; try
	// maximizing another window afterwards to see the reserved area, and drag
	// a dock's inner edge to resize it or its titlebar to pull it off.
	var docks []*winbox.WinBox

	onClick("btn-dock-left", func() {
		docks = append(docks, winbox.New(&winbox.Options{
			Title:    "Docked left",
			HTML:     "<p style='padding:0 14px'>Reserving 260px.</p>" + lorem,
			Dock:     winbox.EdgeLeft,
			DockSize: winbox.Px(260),
			MinWidth: winbox.Px(120),
		}))
	})

	onClick("btn-dock-bottom", func() {
		docks = append(docks, winbox.New(&winbox.Options{
			Title:     "Docked bottom",
			HTML:      "<p style='padding:0 14px'>Minimize a window — it stacks above this, not under it.</p>",
			Dock:      winbox.EdgeBottom,
			DockSize:  winbox.Px(180),
			MinHeight: winbox.Px(80),
		}))
	})

	onClick("btn-dock-overlay", func() {
		docks = append(docks, winbox.New(&winbox.Options{
			Title:    "Docked right (overlay)",
			HTML:     "<p style='padding:0 14px'>Reserves nothing — maximize covers it.</p>" + lorem,
			Dock:     winbox.EdgeRight,
			DockSize: winbox.Pct(25),
			DockMode: winbox.DockOverlay,
		}))
	})

	onClick("btn-undock", func() {
		for _, w := range docks {
			w.Undock()
		}
	})

	// keep the Go runtime alive so callbacks keep working
	select {}
}
