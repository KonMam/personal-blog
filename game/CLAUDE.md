# Game Architecture

ASCII roguelike compiled to WebAssembly (Go → WASM). Lives at `/game/game` in the repo, deployed to `static/game/`.

## Build

```bash
make build-game
# expands to:
cd game && GOOS=js GOARCH=wasm go build -o ../static/game/game.wasm .
```

All `.go` files carry `//go:build js && wasm` and are in `package main`. The runtime is served via `wasm_exec.js` (copied from the Go installation at build time).

Preview at `localhost:1313/game` (Hugo dev server must be running).

---

## File Map

| File | Responsibility |
|------|---------------|
| `entity.go` | `Entity` struct, gear types + catalog, `RecalcStats`, `CalcDamage`, `goldDrop` |
| `state.go` | `Game` struct, all game phases, spawn logic, input dispatch, turn loop |
| `dungeon.go` | Procedural map generation (BSP-style rooms + L corridors) |
| `fov.go` | Raycasting FOV + utility math (`iAbs`, `iSign`) |
| `render.go` | Canvas 2D rendering — tiles, entities, UI strip, overlay panels |
| `main.go` | WASM entry point: canvas setup, keyboard handler, `gameInput` JS global |
| `../static/game/index.html` | HTML shell: canvas, mobile D-pad/action buttons, swipe, rescaling |

---

## Game Loop

All input funnels through `HandleInput(key string)` which dispatches on `g.Phase`:

```
PhasePlay  →  movePlayer / usePotion / ...
             → enemyTurn → ComputeFOV
PhaseChest →  E = equip PendingGear, any key = dismiss
PhaseShop  →  1-4 = buy, Esc/move = leave
PhaseGameOver / PhaseVictory  →  R = restart
```

Enemy turn runs immediately after every player action (bump attack, move, potion use, entering chest/shop). `cleanupDeadEnemies()` is called at the start and end of `enemyTurn` to compact the slice.

---

## Map & Dungeon

- Grid: `MapW=60 × MapH=22` tiles
- Rooms: up to 9, `4-10 wide × 4-7 tall`, connected by L-shaped corridors
- Room ordering is stable: `rooms[0]` = player spawn, `rooms[last]` = stairs
- Chests placed at centers of rooms `[1 .. last-2]`
- Merchant placed at `rooms[2].Center()` on floor 2 only

---

## Entity & Gear

`Entity` is shared by player and enemies. Player-only fields are documented with a comment in the struct.

**Stat flow:**
1. `BaseAtk / BaseDef / BaseMaxHP` set at construction — never mutated
2. `RecalcStats()` sums base + all `Gear.{AtkMod,DefMod,HPMod,FOVMod,Thorns}` into the live `Atk / Def / MaxHP / FOVRadius / Thorns`
3. Call `RecalcStats()` any time `Equipped` changes

**Damage formula:**
- Player → enemy: `CalcDamage()` = rand in `[Atk*0.6 .. Atk*1.4]`, min 1. No enemy defence.
- Enemy → player: same roll, then `max(1, raw - Player.Def)`
- Thorns: if `Player.Thorns > 0`, each attacker takes flat thorns damage after hitting

**Gear catalog** (both slices in `entity.go`):
- `GearWeapons` — 5 items, char `†`
- `GearArmors` — 5 items, char `◈`
- Adding a new item: append to the relevant slice. It will automatically appear in chest and merchant pools.

**Slots:** `SlotWeapon = 0`, `SlotArmor = 1` — index directly into `Entity.Equipped[2]`.

---

## Game Phases

```
PhasePlay      normal play
PhaseChest     gear equip prompt (PendingGear set)
PhaseShop      merchant panel open
PhaseGameOver  player dead
PhaseVictory   reached stairs on floor 3
```

`PendingGear` is reused by both chests (found gear) and the merchant (bought gear). Both resolve through `handleChestInput`.

---

## Rendering

Canvas dimensions: `CanvasW = 960` (`60 tiles × 16px`), `CanvasH = 594` (`22 tiles × 22px + 110px UI`).

**UI strip layout (y = tile area bottom):**
```
+8px   FLOOR n/n   [HP bar]   ◆ Xg   ♥ X
+30px  † weapon name  desc  |  ◈ armor name  desc
+54px  older message (dim)
+72px  latest message (bright)
```

**Overlays** (`PhaseChest`, `PhaseShop`, `PhaseGameOver`, `PhaseVictory`) dim the map area and draw a centered panel on top. They are drawn last in `Render()` so they always sit above everything.

**Colors** are all defined as constants at the top of `render.go`. The palette matches the blog's dark theme (`#0d0d14` background, `#6C8CFF` accent).

---

## Mobile Controls

`gameInput(key string)` is exposed as a global JS function from `main.go`. The HTML D-pad and action buttons call it directly via `onclick`. Swipe detection is on the canvas element (`touchstart` / `touchend`).

Touch controls are hidden on `@media (pointer: fine)` (mouse) and shown on `(pointer: coarse)` (touch).

Canvas scaling: after WASM loads, `rescale()` applies `transform: scale(x)` if `innerWidth < 960`. The wrapper div compensates for the visual size change.

---

## Adding Content

**New enemy type:**
1. Add `EntityFoo` const to `EntityType` in `entity.go`
2. Write `NewFoo(x, y int) *Entity`
3. Add a gold drop case in `goldDrop()`
4. Spawn it in `spawnEnemies()` in `state.go`
5. Add color const in `render.go` if needed

**New floor:**
1. Bump `MaxFloors` in `state.go`
2. Add spawn logic for that floor index in `spawnEnemies()`
3. Consider whether the merchant should appear on new floors (currently hardcoded to floor 2)

**New gear:**
- Append to `GearWeapons` or `GearArmors` in `entity.go`. No other changes needed.

**New game phase:**
1. Add const to `GamePhase` iota in `state.go`
2. Add a dispatch branch in `HandleInput`
3. Add a render branch in `Render()` (overlay or UI change)

---

## Known Limitations / Future Work

- Enemies have no pathfinding beyond greedy cardinal/diagonal moves — they get stuck on walls
- No save/load; state is entirely in-memory for the lifetime of the WASM instance
- Potion inventory is a single count (all potions heal 12 HP regardless of source)
- Merchant stock is randomized but not seeded, so it changes on restart
- `spawnChests` places chests at room centers, which can collide with floor potions or the merchant position (no collision check)
- No diagonal player movement; bump-attack only works on cardinal/diagonal adjacency (the `iAbs(dx) <= 1 && iAbs(dy) <= 1` check)
