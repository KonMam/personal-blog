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
| `events.go` | Event structs (`EventDef`, `EventSpawn`, `ActiveEvent`) + `allEvents` slice (8 events) |
| `state.go` | `Game` struct, all game phases, spawn logic, input dispatch, turn loop, combat helpers |
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
PhaseShop  →  1-5 = buy, Esc/move = leave
PhaseEvent →  1/2/3 = choose; any key after result = dismiss (free action, no enemyTurn)
PhaseGameOver / PhaseVictory  →  R = restart
```

Enemy turn runs immediately after every player action (bump attack, move, potion use, entering chest/shop). Events are free actions — no `enemyTurn` is called. `cleanupDeadEnemies()` is called at the start and end of `enemyTurn`.

---

## Map & Dungeon

- Grid: `MapW=60 × MapH=22` tiles
- Rooms: up to 9, `4-10 wide × 4-7 tall`, connected by L-shaped corridors
- Room ordering is stable: `rooms[0]` = player spawn, `rooms[last]` = stairs
- Spawn order in `newFloor()`: enemies → merchant (floors 2+) → chests → events → items
  - Merchant spawns before chests so chest placement can avoid the merchant's room center
  - Per floor: 2-3 chests, 2 events, 1-2 potions; merchant on floors 2 and 3

---

## Entity & Gear

`Entity` is shared by player and enemies. Player-only and enemy-only fields are documented with comments in the struct.

**Stat flow:**
1. `BaseAtk / BaseDef / BaseMaxHP` set at construction — never mutated by events/gear
2. `RecalcStats()` resets new-mechanics stats to their hardcoded base (Reach=1, Dodge=0, Lifesteal=0, ShieldMod=0, DoubleStrike=false, BurnOnHit=false), then sums all equipped gear mods
3. Call `RecalcStats()` any time `Equipped` changes or base stats are modified (events)
4. `FOVRadius` has no base field — RecalcStats accumulates from current value (known quirk)

**Combat — player attacking:**
- `doPlayerAttack(enemy)` in `state.go` handles the full sequence:
  - `CalcDamage()` roll, apply to enemy HP, kill check
  - Lifesteal heal (per hit, capped to MaxHP)
  - BurnOnHit: sets `enemy.Burn = 3` on first hit
  - DoubleStrike: second `CalcDamage()` + kill check + Lifesteal
- Reach: `movePlayer` scans 1..`Player.Reach` tiles ahead before moving; attacks first enemy found, breaks on wall

**Combat — enemy attacking:**
- `doEnemyAttack(e, isRanged bool)` in `state.go`:
  - ShieldCharges: absorbs hit entirely, decrement, continue
  - Dodge: `rand.Intn(100) < Player.Dodge` → skip damage
  - Apply `max(1, raw - Player.Def)` damage
  - Thorns: attacker takes flat damage after hitting player
- Archers (`RangeAttack > 0`): attack if Chebyshev distance ≤ RangeAttack, otherwise move; never melee

**Status effects:**
- `Player.Poison`: 3 damage/turn at start of `enemyTurn`, decrements; clears on floor descent
- `Enemy.Burn`: 3 damage/turn at start of that enemy's action, decrements; sources: BurnOnHit gear, Alchemist's Fire event
- `Player.ShieldCharges`: refilled by `ShieldMod` at each `newFloor()`; absorbs one hit per charge

**Gear catalog** (all slices in `entity.go`):
- `GearWeapons` — 10 items, char `†`
- `GearArmors` — 10 items, char `◈`
- `GearTrinkets` — 10 items, char `◇`
- `GearEventWeapons` / `GearEventArmors` / `GearEventTrinkets` — 2 each, event-only (never in chests/merchant)
- Adding a new item: append to the relevant slice. Chests draw from the three regular pools; merchant stocks one of each.

**Slots:** `SlotWeapon = 0`, `SlotArmor = 1`, `SlotTrinket = 2` — index into `Entity.Equipped[3]`.

---

## Events System

Events are spawned one per floor in a random non-spawn/non-stairs room center (avoiding enemy/chest/merchant positions). Walking onto a `?` tile opens `PhaseEvent`.

**`events.go`** defines:
- `EventChoice` — label + `func(g *Game) string` effect
- `EventDef` — title, body, slice of choices
- `EventSpawn` — X/Y + pointer to def (consumed on trigger)
- `ActiveEvent` — current def + result string (empty = choices showing)

**`allEvents`** contains 30 events. Three of them (Weapon Shrine, Armory of the Fallen, Sacred Reliquary) set `g.PendingGear` from the event-only gear slices; `handleEventInput` routes to `PhaseChest` after dismissal when `PendingGear != nil`.

Adding a new event: append a `*EventDef` to `allEvents` in `events.go`. No other changes needed.

---

## Game Phases

```
PhasePlay      normal play
PhaseChest     gear equip prompt (PendingGear set)
PhaseShop      merchant panel open (1-5 to buy)
PhaseEvent     event panel open
PhaseGameOver  player dead
PhaseVictory   reached stairs on floor 3
```

`PendingGear` is reused by both chests and merchant purchases. Both resolve through `handleChestInput`.

---

## Rendering

Canvas dimensions: `CanvasW = 960` (`60 tiles × 16px`), `CanvasH = 594` (`22 tiles × 22px + 110px UI`).

**UI strip layout (y = tile area bottom):**
```
+8px   FLOOR n/n   [HP bar]  HP##/##   [◆ Xsh]   ◆ Xg   ♥ X   [☠ X]
+30px  † weapon  desc  |  ◈ armor  desc  |  ◇ trinket  desc
+54px  older message (dim)
+72px  latest message (bright)
```
- `◆ Xsh` (purple) shown only when `ShieldCharges > 0 || ShieldMod > 0`
- `☠ X` (green) shown only when `Player.Poison > 0`

**Map glyphs:**
- `?` (accent blue) — unvisited event tile
- `■` (yellow) — unopened chest
- `$` (green) — merchant
- `♥` (green) — floor potion

**Overlays** (`PhaseChest`, `PhaseShop`, `PhaseEvent`, `PhaseGameOver`, `PhaseVictory`) dim the map and draw a centered panel on top, drawn last in `Render()`.

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
2. Write `NewFoo(x, y int) *Entity` (set `RangeAttack > 0` for a ranged enemy)
3. Add a gold drop case in `goldDrop()`
4. Spawn it in `spawnEnemies()` in `state.go`
5. Add color const in `render.go` if needed

**New floor:**
1. Bump `MaxFloors` in `state.go`
2. Add spawn logic for that floor index in `spawnEnemies()`
3. Consider whether merchant should appear on new floors (currently hardcoded to floor 2)

**New gear:**
- Append to `GearWeapons`, `GearArmors`, or `GearTrinkets` in `entity.go`
- Set relevant new fields: `DoubleStrike`, `ReachMod`, `LifestealMod`, `DodgeMod`, `ShieldMod`, `BurnOnHit`, `BerserkerMod`, `OnKillShield`, `BurnBonus`
- No other changes needed — chests and merchant pick from all three pools automatically
- For event-only gear, append to `GearEventWeapons`, `GearEventArmors`, or `GearEventTrinkets` instead

**New event:**
- Append a `*EventDef` to `allEvents` in `events.go`. No other changes needed.

**New game phase:**
1. Add const to `GamePhase` iota in `state.go`
2. Add a dispatch branch in `HandleInput`
3. Add a render branch in `Render()` (overlay or UI change)

---

## Known Limitations / Future Work

- No save/load; state is entirely in-memory for the lifetime of the WASM instance
- Potion inventory is a single count (all potions heal 12 HP regardless of source)
- Merchant stock is randomized but not seeded, so it changes on restart
- `FOVRadius` has no base field; RecalcStats accumulates from current value — equipping multiple FOV items drifts the radius upward
- No diagonal player movement
