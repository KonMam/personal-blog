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

// startAmbient begins procedural ambient music for the given floor (1-3).
func startAmbient(floor int) {
	fn := js.Global().Get("startAmbient")
	if fn.IsUndefined() || fn.IsNull() {
		return
	}
	fn.Invoke(floor)
}

// stopAmbient fades out and stops the ambient music.
func stopAmbient() {
	fn := js.Global().Get("stopAmbient")
	if fn.IsUndefined() || fn.IsNull() {
		return
	}
	fn.Invoke()
}
