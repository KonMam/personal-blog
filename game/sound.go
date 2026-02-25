//go:build js && wasm

package main

import "syscall/js"

// playSound fires a named sound effect via the JS Web Audio API.
// It is a no-op if the host page hasn't loaded the sound system.
func playSound(name string) {
	fn := js.Global().Get("playSound")
	if fn.IsUndefined() || fn.IsNull() {
		return
	}
	fn.Invoke(name)
}
