GOROOT := $(shell go env GOROOT)
WASM_EXEC_NEW := $(GOROOT)/lib/wasm/wasm_exec.js
WASM_EXEC_OLD := $(GOROOT)/misc/wasm/wasm_exec.js

.PHONY: build-game

build-game:
	cd game && GOOS=js GOARCH=wasm go build -o ../static/game/game.wasm .
	@if [ -f "$(WASM_EXEC_NEW)" ]; then \
		cp "$(WASM_EXEC_NEW)" static/game/wasm_exec.js; \
	else \
		cp "$(WASM_EXEC_OLD)" static/game/wasm_exec.js; \
	fi
	@echo "✓ Built static/game/game.wasm"
