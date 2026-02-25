# Game Architecture

ASCII roguelike compiled to WebAssembly (Go → WASM). Lives at `/game/` in the repo, deployed to `static/game/`.

## Build

```bash
cd game && GOOS=js GOARCH=wasm go build -o ../static/game/game.wasm .
```

All `.go` files carry `//go:build js && wasm` and are in `package main`. The runtime is served via `wasm_exec.js` (copied from the Go installation at build time).

Preview at `localhost:1313/game` (Hugo dev server must be running).

---

## File Map

| File | Responsibility |
|------|---------------|
| `entity.go` | `Entity` struct, gear types + catalog (regular + event-only), `RecalcStats`, `CalcDamage`, `goldDrop` |
| `events.go` | Event structs (`EventDef`, `EventSpawn`, `ActiveEvent`) + `allEvents` slice (30 events) |
| `history.go` | `RunRecord` struct, localStorage load/save, `recordRun` |
| `state.go` | `Game` struct, all game phases, spawn logic, input dispatch, turn loop, combat helpers |
| `dungeon.go` | Procedural map generation (BSP-style rooms + L corridors) |
| `fov.go` | Raycasting FOV + utility math (`iAbs`, `iSign`) |
| `render.go` | Canvas 2D rendering — tiles, entities, UI strip, overlay panels, death/victory screens |
| `main.go` | WASM entry point: canvas setup, keyboard handler, `gameInput` JS global |
| `../static/game/index.html` | HTML shell: canvas, rescaling, mobile notice |

---

## Game Loop

All input funnels through `HandleInput(key string)` which dispatches on `g.Phase`:

```
PhaseClassSelect →  1/2/3/4 = pick class
PhasePlay        →  WASD/arrows = move or bump-attack
                  →  U = use potion
                  →  enemyTurn → ComputeFOV after every action
PhaseChest       →  E = equip PendingGear, any key = dismiss
PhaseShop        →  1-5 = buy, Esc/move = leave
PhaseEvent       →  1/2/3 = choose; any key after result = dismiss (free action, no enemyTurn)
PhaseGameOver    →  R = restart
PhaseVictory     →  R = restart
```

Enemy turn runs immediately after every player action. Events are free actions — no `enemyTurn` is called. `cleanupDeadEnemies()` is called at the start and end of `enemyTurn`.

---

## Map & Dungeon

- Grid: `MapW=60 × MapH=22` tiles
- Rooms: up to 9, `4-10 wide × 4-7 tall`, connected by L-shaped corridors
- Room ordering is stable: `rooms[0]` = player spawn, `rooms[last]` = stairs
- Spawn order in `newFloor()`: enemies → merchant → chests → events → items
  - Merchant spawns before chests so chest placement can avoid the merchant's room center
  - Per floor: 2-3 chests, 2 events, 1-2 potions; merchant on floors 2 and 3

---

## Entity & Gear

`Entity` is shared by player and enemies. Player-only and enemy-only fields are documented with comments in the struct.

**Stat flow:**
1. `BaseAtk / BaseDef / BaseMaxHP` set at construction — never mutated by gear (events may mutate them)
2. `RecalcStats()` resets derived stats to base, then sums all equipped gear mods
3. Call `RecalcStats()` any time `Equipped` changes or base stats are modified by events
4. `FOVRadius` has no base field — RecalcStats accumulates from current value (known quirk; equipping multiple FOV items drifts the radius)

**Player mechanics (all fields on `Entity`):**

| Field | Effect |
|-------|--------|
| `ShieldCharges` | Each charge absorbs one incoming hit entirely (consumed before HP) |
| `ShieldMod` | Charges refilled by this amount at each `newFloor()` |
| `DoubleStrike` | Attacks twice per bump; Lifesteal applies to both hits |
| `Reach` | Bump attack scans up to Reach tiles ahead (default 1); breaks on wall |
| `Lifesteal` | Heals this many HP per hit landed (capped to MaxHP); max 4 |
| `Dodge` | Percent chance to avoid incoming attack entirely |
| `Poison` | 3 damage/turn at start of `enemyTurn`, decrements; clears on floor descent |
| `BerserkerMod` | Flat ATK bonus when HP < 40%; appends `[Rage!]` to hit message |
| `BurnBonus` | Flat bonus damage to burning enemies; appends `[Pyro!]` to hit message |
| `OnKillShield` | Shield charges gained per kill; appends `[+Xsh]` to kill message |

**Enemy mechanics:**

| Field | Effect |
|-------|--------|
| `Burn` | 3 damage/turn at start of that enemy's action, decrements |
| `RangeAttack` | Chebyshev range for ranged attack; 0 = melee only |
| `ShieldCharges` | (Guard enemy) absorbs hits before HP |

**Combat — player attacking (`applyHitToEnemy` + `doPlayerAttack` in `state.go`):**
- Enemy shield check first — if `enemy.ShieldCharges > 0`, absorb hit, skip damage
- `CalcDamage()` roll + BerserkerMod (if HP < 40%) + BurnBonus (if enemy burning)
- Reduce enemy HP; if killed: gold drop, kill count, OnKillShield refill
- Lifesteal heal after each hit (first + second if DoubleStrike)
- BurnOnHit: sets `enemy.Burn = 3` on first hit only
- DoubleStrike: runs a full second hit sequence

**Combat — enemy attacking (`doEnemyAttack` in `state.go`):**
- `Player.ShieldCharges > 0` → absorbs hit, decrement, skip
- `rand.Intn(100) < Player.Dodge` → miss, skip
- Apply `max(1, raw - Player.Def)` damage
- Thorns: attacker takes flat damage after hitting player
- Venomancer: on melee hit, `Player.Poison += 2` (capped at 8)

**Archers** (`RangeAttack > 0`): attack if Chebyshev distance ≤ RangeAttack, otherwise BFS-move toward player; never melee.

---

## Gear Catalog (`entity.go`)

Regular pools (available in chests and merchant):
- `GearWeapons` — 10 items, char `†`
- `GearArmors` — 10 items, char `◈`
- `GearTrinkets` — 10 items, char `◇`

Event-only pools (never in chests/merchant, granted by specific events):
- `GearEventWeapons` — 2 items
- `GearEventArmors` — 2 items
- `GearEventTrinkets` — 2 items

**Deduplication:** `Game.UsedGear map[*Gear]bool` — any gear offered (chest spawn, merchant stock, event grant) is marked used and never offered again that run.

**Slots:** `SlotWeapon=0`, `SlotArmor=1`, `SlotTrinket=2` → `Entity.Equipped[3]*Gear`.

**Gear fields that affect mechanics:**
`AtkMod`, `DefMod`, `HPMod`, `FOVMod`, `Thorns`, `DoubleStrike`, `ReachMod`, `LifestealMod`, `DodgeMod`, `ShieldMod`, `BurnOnHit`, `BerserkerMod`, `OnKillShield`, `BurnBonus`

---

## Events System (`events.go`)

**Structs:**
- `EventChoice` — label + `func(g *Game) string` effect (returns result message)
- `EventDef` — title, body, `[]*EventChoice`
- `EventSpawn` — X/Y + `*EventDef` (consumed on trigger, set to nil)
- `ActiveEvent` — `*EventDef` + result string (empty = choices showing; non-empty = result showing)

**Spawn:** 2 events per floor, placed in random non-spawn/non-stairs room centers (skips occupied positions). `Game.UsedEvents map[*EventDef]bool` prevents the same event appearing twice per run.

**30 events total.** Three (Weapon Shrine, Armory of the Fallen, Sacred Reliquary) set `g.PendingGear` from event-only gear slices. `handleEventInput` routes to `PhaseChest` after dismissal when `PendingGear != nil`.

**Adding a new event:** append a `*EventDef` to `allEvents`. No other changes needed.

---

## Run History (`history.go`)

`RunRecord` stores: Class, Outcome ("Victory"/"Died"), Floor, Kills, Gold, Turns.

- Encoded as pipe-separated string per record, semicolon-separated list in localStorage key `"rogueHistory"`
- Max 10 runs kept, newest first
- `g.recordRun(outcome)` called at death and victory; sets `g.RunHistory`
- Death/victory screens display last 5 runs

---

## Game Phases

```
PhaseClassSelect  picking starting class
PhasePlay         normal play
PhaseChest        gear equip prompt (PendingGear set)
PhaseShop         merchant panel open
PhaseEvent        event panel open
PhaseGameOver     player dead
PhaseVictory      reached stairs on floor 3
```

`PendingGear` is reused by chests, merchant purchases, and gear-giving events. All resolve through `handleChestInput`.

---

## Rendering (`render.go`)

Canvas: `CanvasW=960` (60 tiles × 16px), `CanvasH=594` (22 tiles × 22px + 110px UI strip).

**UI strip:**
```
+8px   FLOOR n/3   [HP bar]  HP##/##   [◆ Xsh]   ◆ Xg   ♥ X   [☠ X turns]
+30px  † weapon desc  |  ◈ armor desc  |  ◇ trinket desc
+54px  older message (dim)
+72px  latest message (bright)
```
- `◆ Xsh` (purple) shown only when `ShieldCharges > 0 || ShieldMod > 0`
- `☠ X` (green) shown only when `Player.Poison > 0`

**Map glyphs:** `?` accent blue (event), `■` yellow (chest), `$` green (merchant), `♥` green (potion), `▼` yellow (stairs).

**End screens** (`renderDeathPanel` / `renderVictoryPanel`): dim map + centered panel showing class, floor, all 3 gear slots, stats (turns/gold/kills), last 5 run history. Death = red border; victory = accent blue border.

**Colors:** all defined as constants at top of `render.go`.

---

## HTML Shell (`static/game/index.html`)

- Desktop only — mobile devices (`pointer: coarse`) see a "not supported" notice; game elements are hidden
- `rescale()` applies `transform: scale(x)` on the canvas wrapper if `innerWidth < 960`
- No touch controls, no swipe detection

---

## Classes

Defined in `state.go` as `classDefs []ClassDef`. Each has Name, description, and stat modifiers applied to the base player at `selectClass()`. `g.ClassName` stores the chosen name for display in end screens and history.

---

## Enemy Types

| Char | Name | Notes |
|------|------|-------|
| `@` | Player | — |
| `g` | Goblin | Basic melee, floor 1+ |
| `o` | Orc | Tankier melee, floor 1+ |
| `T` | Troll | High HP/ATK, floor 3 |
| `a` | Archer | Ranged (range 3), floor 2+ |
| `v` | Venomancer | Poisons player on hit (+2/hit, cap 8), floor 2+ |
| `G` | Guard | 3 shield charges, floor 3 |

---

## Adding Content

**New enemy:**
1. Add `EntityFoo` to `EntityType` iota in `entity.go`
2. Write `NewFoo(x, y int) *Entity`; set `RangeAttack > 0` for ranged
3. Add gold drop case in `goldDrop()`
4. Spawn in `spawnEnemies()` in `state.go`
5. Add color const in `render.go` if needed

**New floor:**
1. Bump `MaxFloors` in `state.go`
2. Add spawn case in `spawnEnemies()`
3. Decide if merchant should appear (currently floors 2–3)

**New gear:** append to the appropriate slice in `entity.go`. Set any relevant mechanic fields. For event-only gear use the `GearEvent*` slices.

**New event:** append `*EventDef` to `allEvents` in `events.go`.

**New game phase:**
1. Add const to `GamePhase` iota in `state.go`
2. Add dispatch branch in `HandleInput`
3. Add render branch in `Render()`

---

## Known Limitations / Future Work

- No save/load; all state is in-memory for the WASM instance lifetime
- `FOVRadius` has no base field — RecalcStats drifts it upward with multiple FOV items equipped
- Merchant stock is not seeded, changes on restart
- Potions always heal 12 HP regardless of source
- Global leaderboard planned (requires external database + API endpoint)
