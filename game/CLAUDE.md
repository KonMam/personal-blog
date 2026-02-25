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
| `state.go` | `Game` struct, all game phases, spawn logic, input dispatch, turn loop, combat helpers, special room logic |
| `dungeon.go` | Procedural map generation (3×3 sector grid + L corridors) |
| `fov.go` | Raycasting FOV + utility math (`iAbs`, `iSign`) |
| `render.go` | Canvas 2D rendering — tiles, entities, UI strip, overlay panels, death/victory screens |
| `main.go` | WASM entry point: canvas setup, keyboard handler, `gameInput` JS global, rAF render loop |
| `../static/game/index.html` | HTML shell: canvas, rescaling, mobile notice, legend |

---

## Game Loop

All input funnels through `HandleInput(key string)` which dispatches on `g.Phase`:

```
PhaseClassSelect →  1/2/3/4 = pick class
PhasePlay        →  WASD/arrows = move or bump-attack
                  →  U = use potion
                  →  . or Space = wait (pass turn)
                  →  Tab = toggle message log overlay
                  →  enemyTurn → ComputeFOV after every action
PhaseChest       →  E = equip PendingGear, any key = dismiss (chest stays if gear not taken)
PhaseShop        →  1-5 = buy, Esc/move = leave
PhaseEvent       →  1/2/3 = choose; any key after result = dismiss (free action, no enemyTurn)
PhaseGameOver    →  R = restart
PhaseVictory     →  R = restart
```

Enemy turn runs immediately after every player action. Events are free actions — no `enemyTurn` is called. `cleanupDeadEnemies()` is called at the start and end of `enemyTurn`. At end of `enemyTurn`: `tickShooters()`, `tickMovingTraps()`, `checkChallengeRooms()`.

---

## Map & Dungeon

- Grid: `MapW=60 × MapH=22` tiles
- **Room layout:** 3×3 sector grid (one room per sector). The 60×22 map splits into 9 sectors of ~20×7 tiles each. One room is placed per sector — this guarantees rooms spread across the full canvas.
- Sector order is **shuffled** each run so player spawn and stairs are in different positions.
- Room sizes: `4-10 wide × 4-6 tall`, connected by L-shaped corridors between consecutive rooms.
- `rooms[0]` = player spawn, `rooms[last]` = stairs.
- **Spawn order in `newFloor()`:** enemies → merchant → `spawnSpecialRoom` → chests → events → items
  - `spawnSpecialRoom` runs before `spawnChests` so `occupied()` sees the special room's objects
  - Per floor: 1 special room, 2-3 chests, 2 events, 1-2 potions; merchant on floors 2-3

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

## Chests (`state.go`)

```go
type Chest struct {
    X, Y      int
    Gold      int
    Gear      *Gear
    Opened    bool // true = fully done (hidden from map)
    GoldTaken bool // true = gold already collected
}
```

- Gold is taken on **first entry** only (`GoldTaken = true`).
- If the chest has gear, `PendingGear` and `PendingChest` are set and phase goes to `PhaseChest`.
- If the player dismisses without equipping (`any key` ≠ E), the chest **remains on the map** — they can return for the gear.
- `Opened = true` (chest disappears) only when the player presses E to equip the gear, or immediately for gold-only chests.
- `Game.PendingChest *Chest` tracks which chest triggered the current `PhaseChest` so it can be marked opened on equip. It is `nil` for merchant/event gear offers.

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

## Special Rooms (`state.go`)

One special room spawns per floor (chosen randomly). `spawnSpecialRoom()` picks a room and calls one of:

### Sacrifice Altar (`spawnSacrificeRoom`)
- Altar `+` (red) placed at room center; 2-3 static spike traps `^` surround it.
- Stepping on altar: pay 8-12 HP, receive 20-35g + gear (`PendingGear` → `PhaseChest`).
- Altar marked `Used = true` after triggered; disappears from map.

### Challenge Room (`spawnChallengeRoom`)
- Room appears empty. On first entry, 2-3 floor-appropriate enemies spawn inside (`cr.Triggered = true`).
- `checkChallengeRooms()` (called end of `enemyTurn`) checks if all `cr.Enemies` are dead → spawns a chest at room center.
- `cr.Cleared = true` once reward chest appears.

### Shooter Room (`spawnShooterRoom`)
- Chest placed at room center as reward.
- 1-2 `Shooter` objects placed on wall tiles at room edges (sides/top/bottom).
- Shooters fire every 3-4 turns; `tickShooters()` decrements `Timer`, fires ray on `Timer == 0`.
- **Fire line overlay** drawn each frame: dim orange tint along the fire path; brightens to yellow the turn before firing (`Timer == 1`). Shooter glyph is `*` orange; turns yellow when about to fire.
- Shooter glyph rendered using `g.Tiles[s.Y][s.X].Explored` (wall tiles are never `Visible`).

### Moving Trap Room (`spawnMovingTrapRoom`)
- Chest placed at room center as reward.
- 2-3 `MovingTrap` objects bounce along the room's primary axis (horizontal if W≥H, vertical otherwise).
- Traps are spread across the axis via segment-based placement; alternate starting direction (1, -1, 1 …) so they move toward each other.
- `tickMovingTraps()`: each trap moves one tile per turn; reverses direction on wall collision.
- Damage (6-10 HP) applied when **trap moves onto player** OR **player moves onto trap** (checked in `movePlayer()`).

**Structs:**
```go
type Trap struct { X, Y int }                          // static spike
type Shooter struct { X, Y, DX, DY, Timer, Period int } // wall launcher
type MovingTrap struct { X, Y, DX, DY int }            // bouncing spike
type SacrificeAltar struct { X, Y int; RewardGear *Gear; Used bool }
type ChallengeRoom struct {
    Bounds Room; Triggered, Cleared bool
    Enemies []*Entity; RewardX, RewardY int
}
```

All five are fields on `Game` and reset to nil in `newFloor()` and `restart()`.

**`occupied()` covers:** Chests, Merchant, Events, Enemies, Items, Traps, MovingTraps, SacrificeAltar (if !Used), ChallengeRoom center (if !Cleared).

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
PhaseChest        gear equip prompt (PendingGear + optionally PendingChest set)
PhaseShop         merchant panel open
PhaseEvent        event panel open
PhaseGameOver     player dead
PhaseVictory      reached stairs on floor 3
```

`PendingGear` is reused by chests, merchant purchases, and gear-giving events. All resolve through `handleChestInput`. `PendingChest` is only set for chest-sourced gear; nil for merchant/event.

---

## Rendering (`render.go`)

Canvas: `CanvasW=960` (60 tiles × 16px), `CanvasH=594` (22 tiles × 22px + 110px UI strip).

**UI strip:**
```
+8px   FLOOR n/3   [HP bar]  HP##/##   [◆ Xsh]   ◆ Xg   ♥ X   [☠ X turns]   † ATK  ◈ DEF
+30px  † weapon desc  |  ◈ armor desc  |  ◇ trinket desc
+54px  older message (dim)
+72px  latest message (bright)
```
- `◆ Xsh` (purple) shown only when `ShieldCharges > 0 || ShieldMod > 0`
- `☠ X` (green) shown only when `Player.Poison > 0`

**Render order (each frame):**
1. Background fill
2. Tiles
3. Shooter fire-line overlays (orange tint along fire path; yellow when `Timer == 1`)
4. Chests, Merchant, Items, Events
5. Sacrifice altar `+`, static traps `^`, moving traps `◆`, shooters `*`
6. Ghost enemies (last-known positions, dim, when out of FOV)
7. Enemies (with HP bars)
8. Player
9. UI strip
10. Damage flash (red tint, 300ms fade via rAF loop)
11. Phase overlays (chest panel, shop panel, event panel, death/victory screens)
12. Message log overlay (Tab)

**Map glyphs:**

| Glyph | Color | Meaning |
|-------|-------|---------|
| `?` | accent blue | event |
| `■` | yellow | chest |
| `$` | green | merchant |
| `♥` | green | potion |
| `▼` | yellow | stairs |
| `^` | orange | static spike trap |
| `◆` | pink/red | moving spike trap |
| `*` | orange (yellow when firing) | shooter |
| `+` | red | sacrifice altar |

**End screens** (`renderDeathPanel` / `renderVictoryPanel`): dim map + centered panel showing class, floor, all 3 gear slots, stats (turns/gold/kills), last 5 run history. Death = red border; victory = accent blue border.

**Colors:** all defined as constants at top of `render.go`.

---

## HTML Shell (`static/game/index.html`)

- Desktop only — mobile devices (`pointer: coarse`) see a "not supported" notice; game elements are hidden
- `rescale()` applies `transform: scale(x)` on the canvas wrapper if `innerWidth < 960`
- No touch controls, no swipe detection
- Legend rows updated to include all trap/shooter/altar glyphs

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
4. Spawn in `spawnEnemies()` and/or `spawnEnemyForFloor()` in `state.go`
5. Add color const in `render.go` if needed

**New floor:**
1. Bump `MaxFloors` in `state.go`
2. Add spawn case in `spawnEnemies()` and `spawnEnemyForFloor()`
3. Decide if merchant should appear (currently floors 2–3)

**New gear:** append to the appropriate slice in `entity.go`. Set any relevant mechanic fields. For event-only gear use the `GearEvent*` slices.

**New event:** append `*EventDef` to `allEvents` in `events.go`.

**New special room type:**
1. Write `spawnFooRoom(room Room)` in `state.go`; add any new structs/fields to `Game`
2. Add a case in `spawnSpecialRoom()` (bump `rand.Intn` range)
3. Reset new fields in `newFloor()` and `restart()`
4. Add tick/check function if needed; call it at end of `enemyTurn()`
5. Add render code in `Render()` (fire-line overlays go after tiles, glyphs go before enemies)
6. Update `occupied()` if the room places objects that block other spawns
7. Update legend in `index.html`

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
