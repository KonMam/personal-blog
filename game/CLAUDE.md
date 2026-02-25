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
| `entity.go` | `Entity` struct, gear types + catalog (regular + event-only + class items), `RecalcStats`, `CalcDamage`, `goldDrop` |
| `classes.go` | `ClassDef` struct, all 8 class definitions (4 base + 4 variant), class starting items in `SlotClass` |
| `events.go` | Event structs (`EventDef`, `EventSpawn`, `ActiveEvent`) + `allEvents` slice (36 events) |
| `history.go` | `RunRecord` struct, localStorage load/save, `recordRun`, `ClassWins`, hint-seen tracking |
| `state.go` | `Game` struct, all game phases, spawn logic, input dispatch, turn loop, combat helpers, special room logic |
| `dungeon.go` | Procedural map generation (3×3 sector grid + L corridors) |
| `fov.go` | Raycasting FOV + utility math (`iAbs`, `iSign`) |
| `render.go` | Canvas 2D rendering — tiles, entities, UI strip, overlay panels, title/difficulty/class select screens, death/victory screens |
| `sound.go` | JS bridge for Web Audio: `playSound`, `startAmbient`, `stopAmbient`, `startTitleMusic`, `stopTitleMusic` |
| `main.go` | WASM entry point: canvas setup, keyboard handler, `gameInput` JS global, rAF render loop |
| `../static/game/index.html` | HTML shell: canvas, rescaling, mobile notice, legend |

---

## Game Loop

All input funnels through `HandleInput(key string)` which dispatches on `g.Phase`:

```
PhaseTitle       →  any key = advance to PhaseDifficulty; starts title music
PhaseDifficulty  →  1=Normal, 2=Hard (locked), 3=Nightmare (locked), 4=Daily
PhaseClassSelect →  1-4 = base class; 5-8 = variant class (locked until 3 wins with base)
PhasePlay        →  WASD/arrows = move or bump-attack
                  →  1/2/3 = use potion in that slot
                  →  . or Space = wait (pass turn)
                  →  Tab = toggle message log overlay
                  →  any key dismisses first-run hint overlay (if showing)
                  →  enemyTurn → ComputeFOV after every action
PhaseChest       →  E = equip PendingGear; Esc = dismiss; any other key = nothing
PhaseShop        →  1-6 = buy item, Esc/move = leave
PhaseEvent       →  1/2/3 = choose; only Esc after result = dismiss (free action, no enemyTurn)
PhaseGameOver    →  R = restart (goes to PhaseDifficulty)
PhaseVictory     →  R = restart (goes to PhaseDifficulty)
```

Enemy turn runs immediately after every player action. Events are free actions — no `enemyTurn` is called. `cleanupDeadEnemies()` is called at the start and end of `enemyTurn`. At end of `enemyTurn`: `tickShooters()`, `tickMovingTraps()`, `checkChallengeRooms()`.

---

## Map & Dungeon

- Grid: `MapW=60 × MapH=22` tiles
- **Room layout:** 3×3 sector grid (one room per sector). The 60×22 map splits into 9 sectors of ~20×7 tiles each. One room is placed per sector — this guarantees rooms spread across the full canvas.
- Sector order is **shuffled** each run so player spawn and stairs are in different positions.
- Room sizes: `4-10 wide × 4-6 tall`, connected by L-shaped corridors between consecutive rooms.
- `rooms[0]` = player spawn, `rooms[last]` = stairs (boss room).
- **Spawn order in `newFloor()`:** enemies → merchant → `spawnSpecialRoom` → chests → events → items
  - `spawnSpecialRoom` runs before `spawnChests` so `occupied()` sees the special room's objects
  - Per floor: 1 special room, 2-3 chests, 2 events, 1-2 potions; merchant on floors 2-3

### Biome Themes

| Floor | Theme | Wall color | Floor color |
|-------|-------|-----------|-------------|
| 1 | Blue-gray stone | `#3d4663` | `#1e2236` |
| 2 | Purple/violet crypts | `#4a3363` | `#231e36` |
| 3 | Deep red abyss | `#5c2a2a` | `#2a1515` |

Tiles have 4 visual variants (`Variant` field) for subtle texture variation.

---

## Entity & Gear

`Entity` is shared by player and enemies. Player-only and enemy-only fields are documented with comments in the struct.

**Stat flow:**
1. `BaseAtk / BaseDef / BaseMaxHP` set at construction — never mutated by gear (events may mutate them)
2. `RecalcStats()` resets derived stats to base, then sums all equipped gear mods (slots 0-3)
3. Call `RecalcStats()` any time `Equipped` changes or base stats are modified by events
4. `FOVRadius` is accumulated from current value in `RecalcStats` (known quirk; equipping multiple FOV items drifts the radius)

**Player mechanics (all fields on `Entity`):**

| Field | Effect |
|-------|--------|
| `ShieldCharges` | Each charge absorbs one incoming hit entirely (consumed before HP) |
| `ShieldMod` | Charges refilled by this amount at each `newFloor()` |
| `DoubleStrike` | Attacks twice per bump; Lifesteal applies to both hits |
| `Reach` | Bump attack scans up to Reach tiles ahead (default 1); breaks on wall |
| `Lifesteal` | Heals this many HP per hit landed (capped to MaxHP); max 4 |
| `Dodge` | Percent chance to avoid incoming attack entirely |
| `Poison` | 3 damage/turn at start of `enemyTurn`, decrements; clears on floor descent (except Nightmare) |
| `PlayerBurn` | 3 damage/turn from Salamander melee hits; always clears on floor descent |
| `BerserkerMod` | Flat ATK bonus when HP < 40%; appends `[Rage!]` to hit message |
| `BurnBonus` | Flat bonus damage to burning enemies; appends `[Pyro!]` to hit message |
| `OnKillShield` | Shield charges gained per kill; appends `[+Xsh]` to kill message |
| `CursePenalty` | Extra incoming damage per hit (from equipped cursed gear) |
| `FreezeChance` | % chance to freeze enemy on hit (skips that enemy's turn) |
| `BleedOnHit` | Apply Bleed 2 on each hit (+2 bleed stacks, max 6) |
| `TempATKBonus / TempATKTurns` | Temporary ATK from Might Draught; counts down per `enemyTurn` |

**Enemy mechanics:**

| Field | Effect |
|-------|--------|
| `Burn` | 3 damage/turn at start of that enemy's action, decrements |
| `RangeAttack` | Chebyshev range for ranged attack; 0 = melee only |
| `ShieldCharges` | (Guard enemy) absorbs hits before HP |
| `Frozen` | Turns remaining frozen (skips that enemy's turn) |
| `Bleed` | 2 damage/turn at start of that enemy's action, decrements |
| `MoveSpeed` | Steps per `enemyTurn`; default 1, Brute = 2 |
| `IsRevealed` | Mimic only: false = displayed as `■` chest glyph, stays still |

**Combat — player attacking (`applyHitToEnemy` + `doPlayerAttack` in `state.go`):**
- Enemy shield check first — if `enemy.ShieldCharges > 0`, absorb hit, skip damage
- `CalcDamage()` roll + BerserkerMod (if HP < 40%) + BurnBonus (if enemy burning) + TempATKBonus (if Might active)
- Reduce enemy HP; if killed: gold drop, kill count, OnKillShield refill, boss gear drop
- Lifesteal heal after each hit (first + second if DoubleStrike); +1 if `SynergyRageDrain` active
- BurnOnHit: sets `enemy.Burn = 3` on first hit only
- Wildfire synergy: second hit also sets `enemy.Burn = 3`
- DoubleStrike: runs a full second hit sequence
- FreezeChance: random check per hit; frozen enemies skip their turn
- BleedOnHit: adds 2 Bleed stacks per hit (capped at 6)
- Mimic reveal: first hit sets `IsRevealed = true`, changes char to `M`

**Combat — enemy attacking (`doEnemyAttack` in `state.go`):**
- `Player.ShieldCharges > 0` → absorbs hit, decrement, skip
- `rand.Intn(100) < Player.Dodge` → miss, skip; Reactive synergy applies thorns on dodge
- Apply `max(1, raw - Player.Def) + CursePenalty` damage
- Thorns: attacker takes flat damage after hitting player
- Venomancer: on melee hit, `Player.Poison += 2` (capped at 8)
- Salamander: on melee hit, `Player.PlayerBurn = min(6, PlayerBurn+2)`
- Heavy hits (≥8 dmg): trigger `ShakeFrames = 4`

**Archers** (`RangeAttack > 0`): attack if Chebyshev distance ≤ RangeAttack, otherwise BFS-move toward player; never melee.

---

## Gear Catalog (`entity.go`)

**Gear slots:** `SlotWeapon=0`, `SlotArmor=1`, `SlotTrinket=2`, `SlotClass=3`
`SlotClass` is the locked class starting item — never replaced by found gear.

Regular pools (available in chests and merchant):
- `GearWeapons` — 13 items, char `†`
- `GearArmors` — 11 items, char `◈`
- `GearTrinkets` — 11 items, char `◇`

Event-only pools (never in chests/merchant, granted by specific events):
- `GearEventWeapons` — 2 items
- `GearEventArmors` — 2 items
- `GearEventTrinkets` — 2 items

**Rarity colors:**
- Common `#718096` · Uncommon `#68D391` · Rare `#63B3ED` · Epic `#9F7AEA` · Event `#F6E05E` · Cursed `#FC8181`

**Deduplication:** `Game.UsedGear map[*Gear]bool` — any gear offered (chest spawn, merchant stock, event grant, boss drop) is marked used and never offered again that run.

**Gear fields that affect mechanics:**
`AtkMod`, `DefMod`, `HPMod`, `FOVMod`, `Thorns`, `DoubleStrike`, `ReachMod`, `LifestealMod`, `DodgeMod`, `ShieldMod`, `BurnOnHit`, `BerserkerMod`, `OnKillShield`, `BurnBonus`, `Cursed`, `CursePenalty`, `FreezeChance`, `BleedOnHit`

**Cursed gear:** `Cursed: true` + `CursePenalty int` — extra damage on every incoming hit. Removable via Altar of Cleansing event (-25g).

---

## Synergy System (`entity.go` — `RecalcStats`)

Synergies are boolean flags computed automatically by `RecalcStats()` from the combined gear stats. They activate when both conditions are met simultaneously.

| Synergy | Condition | Effect |
|---------|-----------|--------|
| `SynergyWildfire` | DoubleStrike + BurnOnHit | Second hit also applies Burn 3 |
| `SynergyFortress` | ShieldMod ≥ 2 + OnKillShield ≥ 1 | +3 bonus shields at every `newFloor()` |
| `SynergyRageDrain` | BerserkerMod ≥ 3 + Lifesteal ≥ 1 | +1 extra lifesteal when in berserker mode (HP < 40%) |
| `SynergyReactive` | Thorns ≥ 2 + Dodge ≥ 15 | On dodge, deal thorns damage to the attacker |
| `SynergyInferno` | BurnBonus ≥ 3 + DoubleStrike | Forces BurnOnHit = true while active |

Synergy activation messages appear in chat when the flag first turns true during an equip.

---

## Classes (`classes.go`)

Defined as `classDefs []*ClassDef`. Each has Name, Flavor, Color, BaseHP, BaseAtk, BaseDef, and a `StartItem *Gear` pre-equipped in `SlotClass`. Class slot items never appear in the spawn pool.

**Base classes (keys 1-4):**

| Key | Class | HP | ATK | DEF | Starting Ability | Flavor |
|-----|-------|----|-----|-----|-----------------|--------|
| 1 | Knight | 38 | 4 | 2 | Ironclad Seal (5 shields/floor) | Tank with shields |
| 2 | Rogue | 22 | 7 | 0 | Shadow Wrap (20% dodge, lifesteal 1) | Fragile, slippery |
| 3 | Berserker | 28 | 9 | 0 | Warlord's Hide (Thorns 4) | High damage, no defense |
| 4 | Alchemist | 24 | 5 | 1 | Infusion Blade (+3 ATK, burn on hit) | Fire-focused |

**Variant classes (keys 5-8, unlock by winning 3 times with corresponding base class):**

| Key | Class | Unlocks from | HP | ATK | DEF | Starting Ability |
|-----|-------|--------------|----|-----|-----|-----------------|
| 5 | Paladin | Knight | 36 | 5 | 3 | Aegis (+2 DEF, 4 shields/floor) |
| 6 | Shadowblade | Rogue | 20 | 9 | 0 | Phantom Cloak (+1 DEF, 30% dodge) |
| 7 | Warlord | Berserker | 32 | 10 | 0 | Battle Axe (+7 ATK, -1 DEF) |
| 8 | Pyromancer | Alchemist | 22 | 6 | 0 | Pyromancer's Lens (+4 dmg vs burning) |

`g.ClassWins map[string]int` (persisted in localStorage key `rogueClassWins`) tracks victories per class name. Variant classes show on the class select screen only when their unlock threshold (3 wins) is met.

---

## Difficulty System (`state.go`)

`g.Difficulty` (int) is set on the difficulty select screen. `g.Seed` is set to `time.Now().UnixNano()` (or today's date for Daily), then `rand.Seed(g.Seed)`.

| Value | Name | Locked until | Enemy HP | Other |
|-------|------|-------------|---------|-------|
| 0 | Normal | always | ×1 | — |
| 1 | Hard | 1 Normal victory | ×1.25 | No merchant on floor 2 |
| 2 | Nightmare | 1 Hard victory | ×1.40 | Poison persists through floor descent; merchant costs ×1.3 |
| 3 | Daily Challenge | always | ×1 | Seed = today's midnight Unix timestamp; shown with date |

`applyDifficultyHP(e *Entity)` is called on every spawned enemy (including boss + minions).

---

## Potions

Player carries a typed potion inventory: `PotionTypes []PotionType` (ordered list, max displayed: 3 slots in UI).

| Type | Key | Effect |
|------|-----|--------|
| `PotionHealing` | H | +12 HP |
| `PotionAntidote` | A | Clear Poison + PlayerBurn, +4 HP |
| `PotionMight` | M | +5 TempATK for 3 turns |
| `PotionGreater` | G | +25 HP |

**Use:** press `1`, `2`, or `3` to use the potion in that slot. Slot display uses letters H/A/M/G colored by type.

**Random floor spawns:** 60% Healing, 15% Antidote, 15% Might Draught, 10% Greater.

**Merchant sells:** Healing Potion (18g), Antidote (24g), Might Draught (30g); prices ×1.3 on Nightmare.

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
- If the player dismisses without equipping (`Esc`), the chest **remains on the map** — they can return for the gear.
- Any key other than `E` or `Esc` does nothing in `PhaseChest` (prevents accidental dismissal).
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

**36 events total.** Three (Weapon Shrine, Armory of the Fallen, Sacred Reliquary) set `g.PendingGear` from event-only gear slices. Two (Trapped Chest, Wandering Trader, Hidden Cache) call `g.pickAnyGear()` for random regular gear. `handleEventInput` routes to `PhaseChest` after `Esc` dismissal when `PendingGear != nil`.

**Result dismissal:** only `Esc` key dismisses the event result screen (not "any key").

**Adding a new event:** append a `*EventDef` to `allEvents`. No other changes needed.

---

## Special Rooms (`state.go`)

One special room spawns per floor (chosen randomly). `spawnSpecialRoom()` picks a room and calls one of:

### Sacrifice Altar (`spawnSacrificeRoom`)
- Altar `+` (red) placed at room center; 2-3 static spike traps `^` surround it.
- Stepping on altar: pay 8-12 HP, receive 12-23g + gear (`PendingGear` → `PhaseChest`).
- Altar marked `Used = true` after triggered; disappears from map.

### Challenge Room (`spawnChallengeRoom`)
- Room appears empty. On first entry, 2-3 floor-appropriate enemies spawn inside (`cr.Triggered = true`).
- `checkChallengeRooms()` (called end of `enemyTurn`) checks if all `cr.Enemies` are dead → spawns a chest at room center.
- `cr.Cleared = true` once reward chest appears.

### Shooter Room (`spawnShooterRoom`)
- Chest placed at room center as reward.
- 1-2 `Shooter` objects placed on wall tiles just outside the room edges (left/right wall at room Y center, top/bottom wall at room X center).
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

## Boss System (`state.go`)

Each floor's **last room** (stairs room) spawns a floor boss + 2 extra regular enemies.

**Stairs are blocked until the floor boss is dead** (`e.IsBoss && e.Alive` check before stair descent).

**Bosses drop a guaranteed gear item** on death via `pickAnyGear()` → `PendingGear` → `PhaseChest`.

| Floor | Boss | Char | HP | ATK | Special |
|-------|------|------|----|-----|---------|
| 1 | Goblin King | `K` | 22 | 5 | Phase 2 at 50% HP: spawns 1 goblin minion adjacent |
| 2 | Orc Warchief | `W` | 30 | 6 | Enrages every 2 turns: +1 ATK, up to 5 stacks |
| 3 | Lich | `L` | 28 | 6 | Ranged 4; Phase 2 at 30% HP: teleports to random floor tile |

**Boss announcement:** first time a boss enters FOV, a `BossAnnounce` banner fades in/out centered on the map area, and a message is logged.

**Boss HP bar:** a full-width strip at the bottom of the map area (y = `MapH*TileH - 7`), visible while the boss is in FOV. Shows name + HP/MaxHP. Color transitions from yellow → orange → red.

---

## Enemy Types

| Char | Name | HP | ATK | Notes |
|------|------|----|-----|-------|
| `@` | Player | — | — | — |
| `g` | Goblin | 8 | 2 | Basic melee, floor 1+ |
| `o` | Orc | 15 | 4 | Tankier melee, floor 1+ |
| `T` | Troll | 30 | 7 | High HP/ATK, floor 3 (regular spawn pool) |
| `a` | Archer | 6 | 3 | Ranged (range 3), floor 1+ |
| `v` | Venomancer | 12 | 3 | Poisons player on melee hit (+2/hit, cap 8), floor 2+ |
| `G` | Guard | 22 | 5 | 3 shield charges, floor 3 |
| `B` | Brute | 22 | 8 | DEF 1, MoveSpeed 2 (moves twice per turn), floor 2+ |
| `■` | Mimic | 15 | 7 | Appears as chest, stays still until first hit; reveals as `M` |
| `s` | Salamander | 14 | 5 | Applies PlayerBurn 2 on melee hit, floor 3 |
| `K` | Goblin King | 22 | 5 | Floor 1 boss; spawns goblin minion at 50% HP |
| `W` | Orc Warchief | 30 | 6 | Floor 2 boss; enrages every 2 turns (+1 ATK, max 5 stacks) |
| `L` | Lich | 28 | 6 | Floor 3 boss; ranged 4; teleports at 30% HP |

---

## Run History (`history.go`)

`RunRecord` stores: Class, Outcome ("Victory"/"Died"), Floor, Kills, Gold, Turns, Difficulty, IsDaily.

- Encoded as pipe-separated string per record, semicolon-separated list in localStorage key `"rogueHistory"`
- Max 10 runs kept, newest first
- `g.recordRun(outcome)` called at death and victory; sets `g.RunHistory`
- Death/victory screens display last 5 runs

**Class wins:** `ClassWins map[string]int` stored in localStorage key `"rogueClassWins"`. Incremented on every Victory. Used to unlock variant classes (threshold: 3 wins per base class).

**Hint seen:** `rogueHintSeen` key — once set, the first-run controls overlay never shows again.

---

## Game Phases

```
PhaseTitle        title screen (any key advances)
PhaseDifficulty   difficulty selection (1-4)
PhaseClassSelect  picking starting class (1-8)
PhasePlay         normal play
PhaseChest        gear equip prompt (PendingGear + optionally PendingChest set)
PhaseShop         merchant panel open
PhaseEvent        event panel open
PhaseGameOver     player dead
PhaseVictory      reached stairs on floor 3
```

`PendingGear` is reused by chests, merchant purchases, boss kills, and gear-giving events. All resolve through `handleChestInput`. `PendingChest` is only set for chest-sourced gear; nil for merchant/event/boss.

`restart()` resets everything and goes to `PhaseDifficulty` (not PhaseClassSelect directly).

---

## Visual Effects (`state.go`, `render.go`)

| Field | Effect |
|-------|--------|
| `FloorTransition float64` | 1.0 = full black; decrements 0.06/frame; triggered on stair descent |
| `ShakeFrames int` | Screen shake for N frames (random ±3px/±2px translate); set to 4 on hits ≥8 dmg |
| `FloatingNums []FloatingNum` | Damage/heal numbers drift upward and fade over ~25 frames |
| `BossAnnounce string` | Boss name shown in announcement banner |
| `BossAnnounceTimer float64` | 1.0 → 0; decrements 0.015/frame; banner alpha = timer×2 (capped 1.0) |
| `ShowHint bool` | First-run controls overlay; dismissed by any key in PhasePlay |

`FloatingNum`: positive value = red damage (`-N`), negative = green heal (`+N`). Color overridable (potions use `ColorPotion`).

---

## Rendering (`render.go`)

Canvas: `CanvasW=960` (60 tiles × 16px), `CanvasH=594` (22 tiles × 22px + 110px UI strip).

**UI strip (line 1):**
```
+8px   FLOOR n/3  [HARD/NIGHTMARE/DAILY badge]  HP bar  HP##/##  [◆ Xsh]  ¤ Xg  ♥ [H][A][M]  [⚡+N(Xt)] [PSN N] [BRN N]   † ATK  ◈ DEF
+30px  † weapon | ◈ armor | ◇ trinket | ✦ class-item (right-aligned)
+54px  older message (dim)
+72px  latest message (bright)
```
- `◆ Xsh` (purple) shown only when `ShieldCharges > 0 || ShieldMod > 0`
- Potion slots: up to 3 displayed with letter code (H=Healing, A=Antidote, M=Might, G=Greater) in type color; empty slots show `·`
- Status cluster: Might (`⚡+N(Xt)` orange), Poison (`PSN N` green), Burn (`BRN N` orange)
- Class item displayed right-aligned with `✦` lock icon

**Render order (each frame in PhasePlay):**
1. Background fill
2. Tiles (with biome color variants per floor)
3. Shooter fire-line overlays (orange tint; yellow when `Timer == 1`)
4. Chests, Merchant, Items, Events
5. Sacrifice altar `+`, static traps `^`, moving traps `◆`, shooters `*`
6. Ghost enemies (last-known positions, dim `ColorUIDim`, when out of FOV)
7. Enemies (with HP bars; freeze `*` + bleed `;` pips in tile corners)
8. Player `@`
9. Boss full-width HP bar strip (when boss is visible)
10. Floating damage/heal numbers (drift up, fade out)
11. Boss announcement banner (fades in/out)
12. Screen shake wrapper (`save/translate/restore`)
13. UI strip
14. Damage flash (red tint, 300ms fade)
15. Floor transition (black fade)
16. Phase overlays (chest panel, shop panel, event panel, death/victory screens)
17. First-run hint overlay (if `ShowHint && PhasePlay`)
18. Message log overlay (Tab)

**Map glyphs:**

| Glyph | Color | Meaning |
|-------|-------|---------|
| `?` | accent blue | event |
| `■` | yellow | chest (also unrevealed Mimic) |
| `$` | green | merchant |
| `♥` | green | potion |
| `▼` | yellow | stairs |
| `^` | orange | static spike trap |
| `◆` | pink/red | moving spike trap |
| `*` | orange (yellow when firing) | shooter |
| `+` | red | sacrifice altar |

**End screens** (`renderDeathPanel` / `renderVictoryPanel`): dim map + centered panel showing class (in class color), floor, all 4 gear slots (weapon/armor/trinket/class), full stats (turns/gold/kills + DMG OUT/IN/POTIONS/STEPS), last 5 run history. Death = red border; victory = accent blue border.

**Additional screens:**
- `renderTitleScreen`: decorative dungeon glyph grid bg + vignette, title, tagline, last run info, pulsing prompt
- `renderDifficultySelect`: panel with 4 options; locked options shown dim with unlock hint
- `renderClassSelect`: dynamic panel showing unlocked classes only; each row has name, stats, starting item + description, flavor text
- `renderHintOverlay`: two-column controls reference; dismissed once via localStorage

**Colors:** all defined as constants at top of `render.go`.

---

## Sound System (`sound.go`)

All functions are no-ops if the host page hasn't loaded the JS sound system.

| Function | JS global | When called |
|----------|-----------|-------------|
| `playSound("hit")` | `playSound` | Player hits enemy |
| `playSound("kill")` | `playSound` | Enemy killed |
| `playSound("hurt")` | `playSound` | Player takes damage |
| `playSound("block")` | `playSound` | Shield absorbs hit |
| `playSound("miss")` | `playSound` | Player dodges |
| `playSound("chest")` | `playSound` | Gear equipped |
| `playSound("potion")` | `playSound` | Potion used |
| `playSound("stairs")` | `playSound` | Floor descent |
| `playSound("victory")` | `playSound` | Victory achieved |
| `playSound("gameover")` | `playSound` | Player dies |
| `startAmbient(floor)` | `startAmbient` | After `newFloor()` |
| `stopAmbient()` | `stopAmbient` | On `recordRun()` |
| `startTitleMusic()` | `startTitleMusic` | Title screen / restart |
| `stopTitleMusic()` | `stopTitleMusic` | On class selection |

---

## HTML Shell (`static/game/index.html`)

- Desktop only — mobile devices (`pointer: coarse`) see a "not supported" notice; game elements are hidden
- `rescale()` applies `transform: scale(x)` on the canvas wrapper if `innerWidth < 960`
- No touch controls, no swipe detection
- Legend rows updated to include all trap/shooter/altar glyphs

---

## Adding Content

**New enemy:**
1. Add `EntityFoo` to `EntityType` iota in `entity.go`
2. Write `NewFoo(x, y int) *Entity`; set `RangeAttack > 0` for ranged, `MoveSpeed > 1` for fast
3. Add gold drop case in `goldDrop()`
4. Spawn in `spawnEnemies()` and/or `spawnEnemyForFloor()` in `state.go`
5. Add color const in `render.go` if needed

**New floor:**
1. Bump `MaxFloors` in `state.go`
2. Add boss case in `spawnEnemies()` (last room) and regular spawns
3. Add biome color case in `renderTile()` in `render.go`
4. Decide if merchant should appear

**New gear:** append to the appropriate slice in `entity.go`. Set any relevant mechanic fields. For event-only gear use the `GearEvent*` slices. For class items define in `classes.go`.

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

**New class:**
1. Define a `classItem*` var in `classes.go` (must use `SlotClass`)
2. Append a `*ClassDef` to `classDefs`
3. If it's a variant, add a case in `variantUnlockReq()` in `state.go`
4. Handle keys in `handleClassSelectInput()` (currently supports keys 1-8)

---

## Known Limitations / Future Work

- No save/load; all state is in-memory for the WASM instance lifetime
- `FOVRadius` has no base field — RecalcStats drifts it upward with multiple FOV items equipped
- Merchant stock is not seeded, changes on restart
- Global leaderboard planned (requires external database + API endpoint)
- `U` key is shown in the hint overlay but is not wired to a handler (slots 1/2/3 are the actual keybindings)
