//go:build js && wasm

package main

import "syscall/js"

func main() {
	// Size the canvas
	canvas := js.Global().Get("document").Call("getElementById", "game-canvas")
	canvas.Set("width", CanvasW)
	canvas.Set("height", CanvasH)
	ctx := canvas.Call("getContext", "2d")

	g := NewGame()
	g.Render(ctx)

	// Keyboard input
	keyHandler := js.FuncOf(func(_ js.Value, args []js.Value) any {
		event := args[0]
		key := event.Get("key").String()

		// Prevent default scroll behaviour for arrow keys / space
		switch key {
		case "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", " ":
			event.Call("preventDefault")
		}

		g.HandleInput(key)
		g.Render(ctx)
		return nil
	})
	js.Global().Get("document").Call("addEventListener", "keydown", keyHandler)
	defer keyHandler.Release()

	// Block forever -- WASM must stay alive
	<-make(chan struct{})
}
