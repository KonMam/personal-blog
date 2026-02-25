//go:build js && wasm

package main

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	MapW      = 60
	MapH      = 22
	MaxFloors = 3
)

type GamePhase int

const (
	PhasePlay GamePhase = iota
	PhaseGameOver
	PhaseVictory
	PhaseChest
	PhaseShop
	PhaseEvent
	PhaseClassSelect
	PhaseDifficulty
	PhaseTitle
)

type Item struct {
	X, Y int
	Type PotionType
}

type Chest struct {
	X, Y      int
	Gold      int
	Gear      *Gear
	Opened    bool // true = fully done (hidden from map)
	GoldTaken bool // true = gold already collected
}

type ShopItem struct {
	Name        string
	Cost        int
	Gear        *Gear      // nil = potion
	PotionGrant PotionType // used when Gear == nil
	Sold        bool
}

type Merchant struct {
	X, Y  int
	Stock []*ShopItem
}

type Trap struct {
	X, Y int
}

type Shooter struct {
	X, Y   int
	DX, DY int // direction of fire
	Timer  int // turns until next shot
	Period int // reset value
}

type MovingTrap struct {
	X, Y   int
	DX, DY int // current movement direction
}

type SacrificeAltar struct {
	X, Y       int
	RewardGear *Gear
	Used       bool
}

type ChallengeRoom struct {
	Bounds   Room
	Triggered bool
	Cleared  bool
	Enemies  []*Entity
	RewardX  int
	RewardY  int
}

type Game struct {
	Tiles       [][]Tile
	Player      *Entity
	Enemies     []*Entity
	Items       []*Item
	Chests      []*Chest
	Merchant    *Merchant
	Events      []*EventSpawn
	ActiveEvent *ActiveEvent
	Traps          []*Trap
	MovingTraps    []*MovingTrap
	Shooters       []*Shooter
	SacrificeAltar *SacrificeAltar
	ChallengeRoom  *ChallengeRoom
	Phase        GamePhase
	Floor        int
	Messages     []string
	PendingGear  *Gear
	PendingChest *Chest // chest that triggered PhaseChest; nil for merchant/event gear
	Turns       int
	Kills       int
	ClassName   string
	RunHistory  []RunRecord
	ClassWins   ClassWins
	UsedGear      map[*Gear]bool     // tracks gear placed in the world this run
	UsedEvents    map[*EventDef]bool // tracks events spawned this run
	LastDamagedAt time.Time         // for damage-flash animation
	ShowLog       bool              // message log overlay visible
	Difficulty    int               // 0=Normal,1=Hard,2=Nightmare,3=Daily
	Seed          int64
	IsDaily       bool
}

func NewGame() *Game {
	return &Game{
		Floor:      1,
		Phase:      PhaseTitle,
		UsedGear:   make(map[*Gear]bool),
		UsedEvents: make(map[*EventDef]bool),
		ClassWins:  loadClassWins(),
	}
}

func (g *Game) newFloor() {
	tiles, rooms := GenerateDungeon(MapW, MapH)
	g.Tiles = tiles
	g.Enemies = nil
	g.Items = nil
	g.Chests = nil
	g.Merchant = nil
	g.Events = nil
	g.ActiveEvent = nil
	g.Traps = nil
	g.MovingTraps = nil
	g.Shooters = nil
	g.SacrificeAltar = nil
	g.ChallengeRoom = nil
	g.PendingChest = nil

	g.addMessage(fmt.Sprintf("You descend to floor %d.", g.Floor))

	// Place player in first room
	if len(rooms) > 0 {
		px, py := rooms[0].Center()
		if g.Player == nil {
			g.Player = NewPlayer(px, py)
		} else {
			g.Player.X, g.Player.Y = px, py
		}
	}

	g.spawnEnemies(rooms)
	if g.Floor >= 2 {
		g.spawnMerchant(rooms)
	}
	g.spawnSpecialRoom(rooms) // must run before spawnChests so occupied() sees its chest/altar
	g.spawnChests(rooms)
	g.spawnEvents(rooms)
	g.spawnItems(rooms)

	// Refill shield charges from gear
	g.Player.ShieldCharges += g.Player.ShieldMod
	// Fortress synergy: +3 bonus shields
	if g.Player.SynergyFortress {
		g.Player.ShieldCharges += 3
	}
	// Poison clears on floor descent (except Nightmare)
	if g.Difficulty < 2 {
		g.Player.Poison = 0
	}
	// PlayerBurn always clears on floor descent
	g.Player.PlayerBurn = 0

	g.recomputeFOV()
}

func (g *Game) applyDifficultyHP(e *Entity) {
	if g.Difficulty == 1 {
		e.HP = e.HP * 5 / 4
		e.MaxHP = e.HP
	} else if g.Difficulty >= 2 {
		e.HP = e.HP * 7 / 5
		e.MaxHP = e.HP
	}
}

func (g *Game) spawnEnemies(rooms []Room) {
	lastRoom := len(rooms) - 1

	for i, room := range rooms {
		if i == 0 {
			continue // player starts here
		}

		// Last room: spawn floor boss
		if i == lastRoom {
			cx, cy := room.Center()
			var boss *Entity
			switch g.Floor {
			case 1:
				boss = NewGoblinKing(cx, cy)
			case 2:
				boss = NewOrcWarchief(cx, cy)
			case 3:
				boss = NewLich(cx, cy)
			}
			if boss != nil {
				g.applyDifficultyHP(boss)
				g.Enemies = append(g.Enemies, boss)
				// Boss drops a guaranteed gear reward on death (tracked via flag)
			}
			// Spawn a couple extra enemies alongside boss
			for n := 0; n < 2; n++ {
				x := room.X + 1 + rand.Intn(room.W-2)
				y := room.Y + 1 + rand.Intn(room.H-2)
				if g.enemyAt(x, y) != nil || (x == cx && y == cy) {
					continue
				}
				e := g.spawnEnemyForFloor(x, y)
				g.applyDifficultyHP(e)
				g.Enemies = append(g.Enemies, e)
			}
			continue
		}

		count := 1 + rand.Intn(2) + (g.Floor - 1)

		for n := 0; n < count; n++ {
			x := room.X + 1 + rand.Intn(room.W-2)
			y := room.Y + 1 + rand.Intn(room.H-2)

			if g.Tiles[y][x].Type == TileStairs {
				continue
			}
			if g.enemyAt(x, y) != nil {
				continue
			}

			var e *Entity
			switch g.Floor {
			case 1:
				if rand.Intn(6) == 0 {
					e = NewArcher(x, y)
				} else {
					e = NewGoblin(x, y)
				}
			case 2:
				// ~12% brute
				if rand.Intn(8) == 0 {
					e = NewBrute(x, y)
				} else {
					roll := rand.Intn(6)
					switch roll {
					case 0:
						e = NewArcher(x, y)
					case 1:
						e = NewGoblin(x, y)
					case 2:
						e = NewVenomancer(x, y)
					default:
						e = NewOrc(x, y)
					}
				}
			case 3:
				// ~12% brute or salamander
				if rand.Intn(8) == 0 {
					if rand.Intn(2) == 0 {
						e = NewBrute(x, y)
					} else {
						e = NewSalamander(x, y)
					}
				} else {
					roll := rand.Intn(8)
					switch roll {
					case 0:
						e = NewArcher(x, y)
					case 1:
						e = NewGoblin(x, y)
					case 2:
						e = NewVenomancer(x, y)
					case 3:
						e = NewGuard(x, y)
					default:
						e = NewOrc(x, y)
					}
				}
			}
			if e != nil {
				g.applyDifficultyHP(e)
				g.Enemies = append(g.Enemies, e)
			}
			_ = n
		}
	}

	// Spawn 1 Mimic per floor on floor 2+
	if g.Floor >= 2 && len(rooms) >= 3 {
		for attempt := 0; attempt < 20; attempt++ {
			idx := 1 + rand.Intn(len(rooms)-2)
			room := rooms[idx]
			cx, cy := room.Center()
			if g.enemyAt(cx, cy) == nil && !g.occupied(cx, cy) {
				m := NewMimic(cx, cy)
				g.applyDifficultyHP(m)
				g.Enemies = append(g.Enemies, m)
				break
			}
		}
	}
}

func (g *Game) spawnItems(rooms []Room) {
	if len(rooms) < 2 {
		return
	}
	count := 1 + rand.Intn(2) // 1-2 potions
	for range count {
		for attempt := 0; attempt < 20; attempt++ {
			idx := 1 + rand.Intn(len(rooms)-1)
			room := rooms[idx]
			x := room.X + 1 + rand.Intn(room.W-2)
			y := room.Y + 1 + rand.Intn(room.H-2)
			if g.occupied(x, y) {
				continue
			}
			g.Items = append(g.Items, &Item{X: x, Y: y, Type: randomPotionType()})
			break
		}
	}
}

func randomPotionType() PotionType {
	r := rand.Intn(100)
	switch {
	case r < 60:
		return PotionHealing
	case r < 75:
		return PotionAntidote
	case r < 90:
		return PotionMight
	default:
		return PotionGreater
	}
}

func potionTypeName(t PotionType) string {
	switch t {
	case PotionAntidote:
		return "Antidote"
	case PotionMight:
		return "Might Draught"
	case PotionGreater:
		return "Greater Potion"
	default:
		return "Healing Potion"
	}
}

// occupied returns true if any game object occupies (x, y).
func (g *Game) occupied(x, y int) bool {
	for _, chest := range g.Chests {
		if chest.X == x && chest.Y == y {
			return true
		}
	}
	if g.Merchant != nil && g.Merchant.X == x && g.Merchant.Y == y {
		return true
	}
	for _, ev := range g.Events {
		if ev.X == x && ev.Y == y {
			return true
		}
	}
	if g.enemyAt(x, y) != nil {
		return true
	}
	for _, item := range g.Items {
		if item.X == x && item.Y == y {
			return true
		}
	}
	for _, t := range g.Traps {
		if t.X == x && t.Y == y {
			return true
		}
	}
	for _, mt := range g.MovingTraps {
		if mt.X == x && mt.Y == y {
			return true
		}
	}
	if sa := g.SacrificeAltar; sa != nil && !sa.Used && sa.X == x && sa.Y == y {
		return true
	}
	if cr := g.ChallengeRoom; cr != nil && !cr.Cleared {
		cx, cy := cr.Bounds.Center()
		if cx == x && cy == y {
			return true
		}
	}
	return false
}

func (g *Game) spawnChests(rooms []Room) {
	if len(rooms) < 3 {
		return
	}
	count := 2 + rand.Intn(2) // 2-3 chests
	used := map[int]bool{}
	for range count {
		for attempt := 0; attempt < 20; attempt++ {
			// Not first room (player spawn), not last room (stairs)
			idx := 1 + rand.Intn(len(rooms)-2)
			if used[idx] {
				continue
			}
			room := rooms[idx]
			cx, cy := room.Center()
			if g.occupied(cx, cy) {
				continue
			}
			used[idx] = true

			// 50% chance to contain gear (never repeat an item seen this run)
			var gear *Gear
			if rand.Intn(2) == 0 {
				all := append(append(GearWeapons, GearArmors...), GearTrinkets...)
				available := make([]*Gear, 0, len(all))
				for _, item := range all {
					if !g.UsedGear[item] {
						available = append(available, item)
					}
				}
				if len(available) > 0 {
					gear = available[rand.Intn(len(available))]
					g.UsedGear[gear] = true
				}
			}
			g.Chests = append(g.Chests, &Chest{
				X:    cx,
				Y:    cy,
				Gold: 10 + rand.Intn(11), // 10-20g
				Gear: gear,
			})
			break
		}
	}
}

func (g *Game) spawnMerchant(rooms []Room) {
	if len(rooms) < 3 {
		return
	}
	// Hard difficulty: no merchant on floor 2
	if g.Difficulty == 1 && g.Floor == 2 {
		return
	}
	// Pick a random middle room (not player spawn or stairs)
	room := rooms[1] // fallback
	for attempt := 0; attempt < 20; attempt++ {
		idx := 1 + rand.Intn(len(rooms)-2)
		r := rooms[idx]
		rc, ry := r.Center()
		if !g.occupied(rc, ry) {
			room = r
			break
		}
	}
	cx, cy := room.Center()

	pickUnused := func(pool []*Gear) *Gear {
		available := make([]*Gear, 0, len(pool))
		for _, item := range pool {
			if !g.UsedGear[item] {
				available = append(available, item)
			}
		}
		if len(available) == 0 {
			return nil
		}
		chosen := available[rand.Intn(len(available))]
		g.UsedGear[chosen] = true
		return chosen
	}

	w := pickUnused(GearWeapons)
	a := pickUnused(GearArmors)
	t := pickUnused(GearTrinkets)

	costMult := 1.0
	if g.Difficulty >= 2 {
		costMult = 1.3
	}
	scaleCost := func(base int) int {
		return int(float64(base) * costMult)
	}

	stock := []*ShopItem{
		{Name: "Healing Potion (+12 HP)", Cost: scaleCost(15), PotionGrant: PotionHealing},
		{Name: "Antidote (clear effects)", Cost: scaleCost(20), PotionGrant: PotionAntidote},
		{Name: "Might Draught (+5 ATK, 3 turns)", Cost: scaleCost(25), PotionGrant: PotionMight},
	}
	if w != nil {
		stock = append(stock, &ShopItem{Name: w.Name, Cost: scaleCost(35 + rand.Intn(20)), Gear: w})
	}
	if a != nil {
		stock = append(stock, &ShopItem{Name: a.Name, Cost: scaleCost(35 + rand.Intn(20)), Gear: a})
	}
	if t != nil {
		stock = append(stock, &ShopItem{Name: t.Name, Cost: scaleCost(40 + rand.Intn(25)), Gear: t})
	}

	g.Merchant = &Merchant{X: cx, Y: cy, Stock: stock}
}

func (g *Game) spawnEvents(rooms []Room) {
	if len(rooms) < 3 {
		return
	}
	spawned := 0
	usedRooms := map[int]bool{}
	for attempt := 0; attempt < 40 && spawned < 2; attempt++ {
		// Not first room (player spawn), not last room (stairs)
		idx := 1 + rand.Intn(len(rooms)-2)
		if usedRooms[idx] {
			continue
		}
		room := rooms[idx]
		cx, cy := room.Center()

		// Skip if occupied by enemy, chest, or merchant
		if g.occupied(cx, cy) {
			continue
		}

		available := make([]*EventDef, 0, len(allEvents))
		for _, def := range allEvents {
			if !g.UsedEvents[def] {
				available = append(available, def)
			}
		}
		if len(available) == 0 {
			break
		}
		usedRooms[idx] = true
		def := available[rand.Intn(len(available))]
		g.UsedEvents[def] = true
		g.Events = append(g.Events, &EventSpawn{X: cx, Y: cy, Def: def})
		spawned++
	}
}

func (g *Game) HandleInput(key string) {
	// Message log overlay — Tab toggles; any other key dismisses it
	if key == "Tab" {
		g.ShowLog = !g.ShowLog
		return
	}
	if g.ShowLog {
		g.ShowLog = false
		return
	}

	switch g.Phase {
	case PhaseTitle:
		g.Phase = PhaseDifficulty
		return
	case PhaseDifficulty:
		g.handleDifficultyInput(key)
		return
	case PhaseClassSelect:
		g.handleClassSelectInput(key)
		return
	case PhaseGameOver, PhaseVictory:
		if key == "r" || key == "R" {
			g.restart()
		}
		return
	case PhaseChest:
		g.handleChestInput(key)
		return
	case PhaseShop:
		g.handleShopInput(key)
		return
	case PhaseEvent:
		g.handleEventInput(key)
		return
	}

	// PhasePlay
	if key == "u" || key == "U" {
		g.usePotion()
		return
	}

	// Wait / pass turn
	if key == "." || key == " " {
		g.addMessage("You wait.")
		g.enemyTurn()
		g.recomputeFOV()
		return
	}

	var dx, dy int
	switch key {
	case "ArrowUp", "w", "W":
		dy = -1
	case "ArrowDown", "s", "S":
		dy = 1
	case "ArrowLeft", "a", "A":
		dx = -1
	case "ArrowRight", "d", "D":
		dx = 1
	default:
		return
	}

	g.movePlayer(dx, dy)
}

func (g *Game) handleChestInput(key string) {
	if (key == "e" || key == "E") && g.PendingGear != nil {
		playSound("chest")
		slot := g.PendingGear.Slot
		old := g.Player.Equipped[slot]
		// Record synergies before equipping
		oldW := g.Player.SynergyWildfire
		oldF := g.Player.SynergyFortress
		oldR := g.Player.SynergyRageDrain
		oldRe := g.Player.SynergyReactive
		oldI := g.Player.SynergyInferno
		g.Player.Equipped[slot] = g.PendingGear
		g.Player.RecalcStats()
		if old != nil {
			g.addMessage(fmt.Sprintf("Equipped %s (replaced %s).", g.PendingGear.Name, old.Name))
		} else {
			g.addMessage(fmt.Sprintf("Equipped %s.", g.PendingGear.Name))
		}
		// Synergy notifications
		if !oldW && g.Player.SynergyWildfire {
			g.addMessage("Synergy active: Wildfire!")
		}
		if !oldF && g.Player.SynergyFortress {
			g.addMessage("Synergy active: Fortress!")
		}
		if !oldR && g.Player.SynergyRageDrain {
			g.addMessage("Synergy active: Rage Drain!")
		}
		if !oldRe && g.Player.SynergyReactive {
			g.addMessage("Synergy active: Reactive!")
		}
		if !oldI && g.Player.SynergyInferno {
			g.addMessage("Synergy active: Inferno!")
		}
		// Mark the chest as fully cleared so it disappears from the map
		if g.PendingChest != nil {
			g.PendingChest.Opened = true
		}
	}
	g.PendingGear = nil
	g.PendingChest = nil
	g.Phase = PhasePlay
}

func (g *Game) handleShopInput(key string) {
	switch key {
	case "Escape":
		g.Phase = PhasePlay
	case "ArrowUp", "w", "W", "ArrowDown", "s", "S",
		"ArrowLeft", "a", "A", "ArrowRight", "d", "D":
		g.Phase = PhasePlay
	case "1", "2", "3", "4", "5":
		if g.Merchant == nil {
			return
		}
		idx := int(key[0] - '1')
		if idx >= len(g.Merchant.Stock) {
			return
		}
		item := g.Merchant.Stock[idx]
		if item.Sold {
			g.addMessage("Already sold.")
			return
		}
		if g.Player.Gold < item.Cost {
			g.addMessage(fmt.Sprintf("Need %dg (have %dg).", item.Cost, g.Player.Gold))
			return
		}
		g.Player.Gold -= item.Cost
		item.Sold = true
		if item.Gear != nil {
			g.PendingGear = item.Gear
			g.Phase = PhaseChest
		} else {
			g.Player.Potions++
			g.Player.PotionTypes = append(g.Player.PotionTypes, item.PotionGrant)
			g.addMessage(fmt.Sprintf("Bought %s. Potions: %d. [U] to use.", item.Name, g.Player.Potions))
		}
	}
}

func (g *Game) handleEventInput(key string) {
	if g.ActiveEvent == nil {
		g.Phase = PhasePlay
		return
	}
	if g.ActiveEvent.Result != "" {
		// Any key continues — events are free actions, no enemy turn
		g.ActiveEvent = nil
		if g.PendingGear != nil {
			g.Phase = PhaseChest
		} else {
			g.Phase = PhasePlay
		}
		return
	}
	// Process choice keys
	if key == "1" || key == "2" || key == "3" {
		idx := int(key[0] - '1')
		choices := g.ActiveEvent.Def.Choices
		if idx < len(choices) {
			g.ActiveEvent.Result = choices[idx].Effect(g)
		}
	}
}

func (g *Game) usePotion() {
	if g.Player.Potions <= 0 {
		g.addMessage("No potions. [♥ 0]")
		return
	}
	g.Player.Potions--
	playSound("potion")
	var pt PotionType
	if len(g.Player.PotionTypes) > 0 {
		pt = g.Player.PotionTypes[0]
		g.Player.PotionTypes = g.Player.PotionTypes[1:]
	}
	switch pt {
	case PotionHealing:
		heal := 12
		if g.Player.HP+heal > g.Player.MaxHP {
			heal = g.Player.MaxHP - g.Player.HP
		}
		g.Player.HP += heal
		g.addMessage(fmt.Sprintf("Healing Potion: +%d HP. (%d left)", heal, g.Player.Potions))
	case PotionAntidote:
		g.Player.Poison = 0
		g.Player.PlayerBurn = 0
		heal := 4
		if g.Player.HP+heal > g.Player.MaxHP {
			heal = g.Player.MaxHP - g.Player.HP
		}
		g.Player.HP += heal
		g.addMessage(fmt.Sprintf("Antidote: cleared effects, +%d HP. (%d left)", heal, g.Player.Potions))
	case PotionMight:
		g.Player.TempATKBonus = 5
		g.Player.TempATKTurns = 3
		g.addMessage(fmt.Sprintf("Might Draught: +5 ATK for 3 turns. (%d left)", g.Player.Potions))
	case PotionGreater:
		heal := 25
		if g.Player.HP+heal > g.Player.MaxHP {
			heal = g.Player.MaxHP - g.Player.HP
		}
		g.Player.HP += heal
		g.addMessage(fmt.Sprintf("Greater Potion: +%d HP. (%d left)", heal, g.Player.Potions))
	}
	g.enemyTurn()
	g.recomputeFOV()
}

func (g *Game) restart() {
	g.Floor = 1
	g.Player = nil
	g.Messages = nil
	g.Phase = PhaseDifficulty
	g.PendingGear = nil
	g.Events = nil
	g.ActiveEvent = nil
	g.Turns = 0
	g.Kills = 0
	g.ClassName = ""
	g.UsedGear = make(map[*Gear]bool)
	g.UsedEvents = make(map[*EventDef]bool)
	g.LastDamagedAt = time.Time{}
	g.ShowLog = false
	g.Traps = nil
	g.MovingTraps = nil
	g.Shooters = nil
	g.SacrificeAltar = nil
	g.ChallengeRoom = nil
	g.PendingChest = nil
	g.Difficulty = 0
	g.Seed = 0
	g.IsDaily = false
	g.ClassWins = loadClassWins()
	// newFloor() is called by selectClass() once a class is chosen
}

func (g *Game) handleDifficultyInput(key string) {
	completedRuns := len(loadRunHistory())
	switch key {
	case "1":
		g.Difficulty = 0
		g.Seed = time.Now().UnixNano()
		rand.Seed(g.Seed)
		g.Phase = PhaseClassSelect
	case "2":
		if completedRuns < 1 {
			return // locked
		}
		g.Difficulty = 1
		g.Seed = time.Now().UnixNano()
		rand.Seed(g.Seed)
		g.Phase = PhaseClassSelect
	case "3":
		if completedRuns < 10 {
			return // locked
		}
		g.Difficulty = 2
		g.Seed = time.Now().UnixNano()
		rand.Seed(g.Seed)
		g.Phase = PhaseClassSelect
	case "4":
		g.Difficulty = 3
		g.IsDaily = true
		today := time.Now().Truncate(24 * time.Hour).Unix()
		g.Seed = today
		rand.Seed(g.Seed)
		g.Phase = PhaseClassSelect
	}
}

func (g *Game) handleClassSelectInput(key string) {
	switch key {
	case "1", "2", "3", "4":
		g.selectClass(int(key[0] - '1'))
	case "5", "6", "7", "8":
		idx := int(key[0]-'1') // 4-7
		variantIdx := idx
		if variantIdx >= len(classDefs) {
			return
		}
		// Check unlock requirement
		unlock := variantUnlockReq(variantIdx)
		if unlock != "" && g.ClassWins[unlock] < 3 {
			return
		}
		g.selectClass(variantIdx)
	}
}

// variantUnlockReq returns the base class name required to unlock a variant class.
func variantUnlockReq(idx int) string {
	switch idx {
	case 4:
		return "Knight"
	case 5:
		return "Rogue"
	case 6:
		return "Berserker"
	case 7:
		return "Alchemist"
	}
	return ""
}

func (g *Game) selectClass(idx int) {
	def := classDefs[idx]
	g.newFloor() // generates dungeon, creates player with default stats

	// Apply class base stats
	g.Player.BaseAtk = def.BaseAtk
	g.Player.BaseMaxHP = def.BaseHP
	g.Player.BaseDef = def.BaseDef

	// Pre-equip starting item
	g.Player.Equipped[def.StartItem.Slot] = def.StartItem
	g.Player.RecalcStats()

	// Full HP at class-corrected max; correct shield charges from starting gear
	g.Player.HP = g.Player.MaxHP
	g.Player.ShieldCharges = g.Player.ShieldMod
	g.Player.Poison = 0

	// Re-run FOV in case FOVRadius changed
	g.recomputeFOV()

	g.ClassName = def.Name
	g.Phase = PhasePlay
	g.addMessage(fmt.Sprintf("You descend as the %s.", def.Name))
}

func (g *Game) movePlayer(dx, dy int) {
	nx, ny := g.Player.X+dx, g.Player.Y+dy

	if nx < 0 || ny < 0 || nx >= MapW || ny >= MapH {
		return
	}
	if g.Tiles[ny][nx].Type == TileWall {
		return
	}

	// Reach attack: scan ahead in movement direction
	var target *Entity
	for dist := 1; dist <= g.Player.Reach; dist++ {
		tx, ty := g.Player.X+dx*dist, g.Player.Y+dy*dist
		if tx < 0 || ty < 0 || tx >= MapW || ty >= MapH {
			break
		}
		if g.Tiles[ty][tx].Type == TileWall {
			break
		}
		if e := g.enemyAt(tx, ty); e != nil {
			target = e
			break
		}
	}
	if target != nil {
		g.doPlayerAttack(target)
		g.enemyTurn()
		g.recomputeFOV()
		return
	}

	// Move
	g.Player.X, g.Player.Y = nx, ny

	// Static spike trap
	for i, t := range g.Traps {
		if t.X == nx && t.Y == ny {
			g.Traps = append(g.Traps[:i], g.Traps[i+1:]...)
			dmg := 6 + rand.Intn(5) // 6-10
			g.Player.HP -= dmg
			if g.Player.HP < 0 {
				g.Player.HP = 0
			}
			g.addMessage(fmt.Sprintf("Spike trap! -%d HP.", dmg))
			g.LastDamagedAt = time.Now()
			if g.Player.HP <= 0 {
				g.Player.Alive = false
				g.recordRun("Died")
				g.Phase = PhaseGameOver
				g.enemyTurn()
				g.recomputeFOV()
				return
			}
			break
		}
	}

	// Moving spike trap collision (player steps onto trap)
	for _, mt := range g.MovingTraps {
		if mt.X == nx && mt.Y == ny {
			dmg := 6 + rand.Intn(5) // 6-10
			g.Player.HP -= dmg
			if g.Player.HP < 0 {
				g.Player.HP = 0
			}
			g.addMessage(fmt.Sprintf("Moving spike! -%d HP.", dmg))
			g.LastDamagedAt = time.Now()
			if g.Player.HP <= 0 {
				g.Player.Alive = false
				g.recordRun("Died")
				g.Phase = PhaseGameOver
				g.enemyTurn()
				g.recomputeFOV()
				return
			}
			break
		}
	}

	// Sacrifice altar
	if sa := g.SacrificeAltar; sa != nil && !sa.Used && nx == sa.X && ny == sa.Y {
		sa.Used = true
		cost := 8 + rand.Intn(5) // 8-12
		g.Player.HP -= cost
		g.LastDamagedAt = time.Now()
		if g.Player.HP <= 0 {
			g.Player.HP = 0
			g.Player.Alive = false
			g.recordRun("Died")
			g.Phase = PhaseGameOver
			g.enemyTurn()
			g.recomputeFOV()
			return
		}
		gold := 20 + rand.Intn(16) // 20-35g
		g.Player.Gold += gold
		if sa.RewardGear != nil {
			g.addMessage(fmt.Sprintf("You offer blood at the altar. -%d HP. Found %s! +%dg.", cost, sa.RewardGear.Name, gold))
			g.PendingGear = sa.RewardGear
			g.Phase = PhaseChest
		} else {
			g.addMessage(fmt.Sprintf("You offer blood at the altar. -%d HP. +%dg.", cost, gold))
		}
		g.enemyTurn()
		g.recomputeFOV()
		return
	}

	// Challenge room trigger
	if cr := g.ChallengeRoom; cr != nil && !cr.Triggered {
		if inRoom(nx, ny, cr.Bounds) {
			cr.Triggered = true
			g.addMessage("You sense an ambush!")
			count := 2 + rand.Intn(2) // 2-3 enemies
			for i := 0; i < count; i++ {
				for attempt := 0; attempt < 20; attempt++ {
					ex := cr.Bounds.X + 1 + rand.Intn(cr.Bounds.W-2)
					ey := cr.Bounds.Y + 1 + rand.Intn(cr.Bounds.H-2)
					if g.enemyAt(ex, ey) == nil && (ex != nx || ey != ny) {
						e := g.spawnEnemyForFloor(ex, ey)
						cr.Enemies = append(cr.Enemies, e)
						g.Enemies = append(g.Enemies, e)
						break
					}
				}
			}
		}
	}

	// Walk into merchant
	if g.Merchant != nil && g.Merchant.X == nx && g.Merchant.Y == ny {
		g.Phase = PhaseShop
		g.enemyTurn()
		g.recomputeFOV()
		return
	}

	// Walk into chest
	for _, chest := range g.Chests {
		if !chest.Opened && chest.X == nx && chest.Y == ny {
			if chest.Gear != nil {
				// Take gold on first entry only
				if !chest.GoldTaken {
					chest.GoldTaken = true
					g.Player.Gold += chest.Gold
					g.addMessage(fmt.Sprintf("Chest: +%dg and found %s! [E] equip.", chest.Gold, chest.Gear.Name))
				} else {
					g.addMessage(fmt.Sprintf("Found %s! [E] equip, any key to leave.", chest.Gear.Name))
				}
				g.PendingGear = chest.Gear
				g.PendingChest = chest
				g.Phase = PhaseChest
			} else {
				g.Player.Gold += chest.Gold
				g.addMessage(fmt.Sprintf("Chest: +%dg gold!", chest.Gold))
				chest.Opened = true
			}
			g.enemyTurn()
			g.recomputeFOV()
			return
		}
	}

	// Walk into event
	for i, ev := range g.Events {
		if ev.X == nx && ev.Y == ny {
			g.ActiveEvent = &ActiveEvent{Def: ev.Def}
			g.Events = append(g.Events[:i], g.Events[i+1:]...)
			g.Phase = PhaseEvent
			// Events are free actions — no enemyTurn
			return
		}
	}

	// Stairs
	if g.Tiles[ny][nx].Type == TileStairs {
		// Block if any boss is alive
		for _, e := range g.Enemies {
			if e.IsBoss && e.Alive {
				g.addMessage("Defeat the floor boss first!")
				g.enemyTurn()
				g.recomputeFOV()
				return
			}
		}
		if g.Floor < MaxFloors {
			g.Floor++
			playSound("stairs")
			g.newFloor()
			return
		}
		playSound("victory")
		g.Phase = PhaseVictory
		g.recordRun("Victory")
		g.addMessage("You escaped the dungeon. Victory!")
		return
	}

	// Pick up potion (add to inventory)
	for i, item := range g.Items {
		if item.X == nx && item.Y == ny {
			g.Player.Potions++
			g.Player.PotionTypes = append(g.Player.PotionTypes, item.Type)
			g.addMessage(fmt.Sprintf("Picked up %s. (%d total) [U] to use.", potionTypeName(item.Type), g.Player.Potions))
			g.Items = append(g.Items[:i], g.Items[i+1:]...)
			break
		}
	}

	g.enemyTurn()
	g.recomputeFOV()
}

// applyHitToEnemy resolves one hit with all player combat mechanics.
// Synergy activations are appended to the hit message.
func (g *Game) applyHitToEnemy(enemy *Entity, isFirst bool) {
	dmg := g.Player.CalcDamage()
	suffix := ""
	berserkerActive := g.Player.BerserkerMod > 0 && g.Player.HP*10 < g.Player.MaxHP*4
	if berserkerActive {
		dmg += g.Player.BerserkerMod
		suffix += " [Rage!]"
	}
	if g.Player.BurnBonus > 0 && enemy.Burn > 0 {
		dmg += g.Player.BurnBonus
		suffix += " [Pyro!]"
	}
	if g.Player.TempATKBonus > 0 {
		dmg += g.Player.TempATKBonus
		suffix += " [Might!]"
	}
	enemy.HP -= dmg

	// Lich phase-2 trigger (on HP reduction, before kill check)
	if enemy.Type == EntityLich && !enemy.BossPhase2 && enemy.HP*10 < enemy.MaxHP*3 {
		enemy.BossPhase2 = true
		// Teleport to random floor tile in the map
		for attempt := 0; attempt < 30; attempt++ {
			tx := rand.Intn(MapW)
			ty := rand.Intn(MapH)
			if g.Tiles[ty][tx].Type == TileFloor && g.enemyAt(tx, ty) == nil {
				enemy.X, enemy.Y = tx, ty
				break
			}
		}
		g.addMessage("The Lich vanishes in a puff of shadow!")
	}

	if enemy.HP <= 0 {
		enemy.HP = 0
		enemy.Alive = false
		g.Kills++
		gold := enemy.goldDrop()
		g.Player.Gold += gold
		if g.Player.OnKillShield > 0 {
			g.Player.ShieldCharges += g.Player.OnKillShield
			suffix += fmt.Sprintf(" [+%dsh]", g.Player.OnKillShield)
		}
		// Boss gear reward
		if enemy.IsBoss {
			bossGear := g.pickAnyGear()
			if bossGear != nil {
				g.PendingGear = bossGear
			}
		}
		playSound("kill")
		if isFirst {
			g.addMessage(fmt.Sprintf("You slay the %s! +%dg%s", enemy.Name, gold, suffix))
		} else {
			g.addMessage(fmt.Sprintf("Second strike slays the %s! +%dg%s", enemy.Name, gold, suffix))
		}
		if enemy.IsBoss && g.PendingGear != nil {
			g.Phase = PhaseChest
		}
	} else {
		playSound("hit")
		if isFirst {
			g.addMessage(fmt.Sprintf("You hit the %s for %d damage.%s", enemy.Name, dmg, suffix))
		} else {
			g.addMessage(fmt.Sprintf("You strike again for %d damage.%s", dmg, suffix))
		}
	}
	lifestealAmt := g.Player.Lifesteal
	if berserkerActive && g.Player.SynergyRageDrain {
		lifestealAmt++
	}
	if lifestealAmt > 0 {
		g.Player.HP = min(g.Player.MaxHP, g.Player.HP+lifestealAmt)
	}
	if isFirst && g.Player.BurnOnHit && enemy.Alive {
		enemy.Burn = 3
	}
	// Wildfire synergy: second hit also burns
	if !isFirst && g.Player.SynergyWildfire && enemy.Alive {
		enemy.Burn = 3
	}
	// Mimic reveal on first hit
	if enemy.Type == EntityMimic && !enemy.IsRevealed {
		enemy.IsRevealed = true
		enemy.Char = 'M'
		suffix += " [It's a Mimic!]"
		g.addMessage("It's a Mimic!")
	}
	// Freeze
	if g.Player.FreezeChance > 0 && rand.Intn(100) < g.Player.FreezeChance && enemy.Alive {
		enemy.Frozen = 2
		g.addMessage(fmt.Sprintf("The %s is frozen!", enemy.Name))
	}
	// Bleed
	if g.Player.BleedOnHit && enemy.Alive {
		enemy.Bleed = min(6, enemy.Bleed+2)
		g.addMessage(fmt.Sprintf("The %s bleeds!", enemy.Name))
	}
}

// doPlayerAttack handles a player bump attack with all combat mechanics.
func (g *Game) doPlayerAttack(enemy *Entity) {
	// First hit — enemy shield absorbs entirely
	if enemy.ShieldCharges > 0 {
		enemy.ShieldCharges--
		g.addMessage(fmt.Sprintf("The %s's shield absorbs your strike! (%d left)", enemy.Name, enemy.ShieldCharges))
	} else {
		g.applyHitToEnemy(enemy, true)
	}

	// Double strike
	if g.Player.DoubleStrike && enemy.Alive {
		if enemy.ShieldCharges > 0 {
			enemy.ShieldCharges--
			g.addMessage(fmt.Sprintf("Your second strike blocked! (%d shields left)", enemy.ShieldCharges))
		} else {
			g.applyHitToEnemy(enemy, false)
		}
	}
}

// doEnemyAttack applies one enemy attack to the player.
// Returns true if the player dies.
func (g *Game) doEnemyAttack(e *Entity, isRanged bool) bool {
	rawDmg := e.CalcDamage()

	// Shield absorbs the hit
	if g.Player.ShieldCharges > 0 {
		g.Player.ShieldCharges--
		playSound("block")
		g.addMessage(fmt.Sprintf("Shield absorbs the %s's attack! (%d left)", e.Name, g.Player.ShieldCharges))
		return false
	}

	// Dodge check
	if rand.Intn(100) < g.Player.Dodge {
		playSound("miss")
		g.addMessage(fmt.Sprintf("You dodge the %s's attack!", e.Name))
		// Reactive synergy: on dodge, deal thorns damage to attacker
		if g.Player.SynergyReactive && g.Player.Thorns > 0 {
			e.HP -= g.Player.Thorns
			if e.HP <= 0 {
				e.HP = 0
				e.Alive = false
				g.Kills++
				gold := e.goldDrop()
				g.Player.Gold += gold
				g.addMessage(fmt.Sprintf("Reactive thorns slay the %s! +%dg", e.Name, gold))
			} else {
				g.addMessage(fmt.Sprintf("Reactive thorns deal %d to the %s.", g.Player.Thorns, e.Name))
			}
		}
		return false
	}

	finalDmg := rawDmg - g.Player.Def
	if finalDmg < 1 {
		finalDmg = 1
	}
	finalDmg += g.Player.CursePenalty
	playSound("hurt")
	g.Player.HP -= finalDmg
	g.LastDamagedAt = time.Now()
	if g.Player.HP <= 0 {
		g.Player.HP = 0
		g.Player.Alive = false
		playSound("gameover")
		g.Phase = PhaseGameOver
		g.recordRun("Died")
		g.addMessage("You died. Press R to restart.")
		return true
	}

	if isRanged {
		g.addMessage(fmt.Sprintf("An arrow from the %s hits you for %d damage.", e.Name, finalDmg))
	} else {
		g.addMessage(fmt.Sprintf("The %s hits you for %d damage.", e.Name, finalDmg))
	}

	// Thorns
	if g.Player.Thorns > 0 {
		e.HP -= g.Player.Thorns
		if e.HP <= 0 {
			e.HP = 0
			e.Alive = false
			g.Kills++
			gold := e.goldDrop()
			g.Player.Gold += gold
			g.addMessage(fmt.Sprintf("Thorns slay the %s! +%dg", e.Name, gold))
		} else {
			g.addMessage(fmt.Sprintf("Thorns deal %d to the %s.", g.Player.Thorns, e.Name))
		}
	}

	// Venomancer poisons on melee hit
	if e.Type == EntityVenomancer && !isRanged {
		g.Player.Poison += 2
		if g.Player.Poison > 8 {
			g.Player.Poison = 8
		}
		g.addMessage(fmt.Sprintf("Venom seeps in! Poison: %d turns.", g.Player.Poison))
	}
	// Salamander applies PlayerBurn on melee hit
	if e.Type == EntitySalamander && !isRanged {
		g.Player.PlayerBurn = min(6, g.Player.PlayerBurn+2)
		g.addMessage("The salamander's flames scorch you! Burning!")
	}
	return false
}

func (g *Game) enemyTurn() {
	g.Turns++

	// Player poison tick
	if g.Player.Poison > 0 {
		g.Player.HP -= 3
		g.Player.Poison--
		if g.Player.HP <= 0 {
			g.Player.HP = 0
			g.Player.Alive = false
			g.Phase = PhaseGameOver
			g.recordRun("Died")
			playSound("gameover")
		g.addMessage("Poison kills you. Press R to restart.")
			return
		}
		g.addMessage(fmt.Sprintf("Poison deals 3 damage. (%d turns left)", g.Player.Poison))
	}

	// Player burn tick
	if g.Player.PlayerBurn > 0 {
		g.Player.HP -= 3
		g.Player.PlayerBurn--
		g.LastDamagedAt = time.Now()
		if g.Player.HP <= 0 {
			g.Player.HP = 0
			g.Player.Alive = false
			g.Phase = PhaseGameOver
			g.recordRun("Died")
			playSound("gameover")
		g.addMessage("You burn to death. Press R to restart.")
			return
		}
		g.addMessage(fmt.Sprintf("You are burning! -3 HP. (%d turns)", g.Player.PlayerBurn))
	}

	// TempATK countdown
	if g.Player.TempATKTurns > 0 {
		g.Player.TempATKTurns--
		if g.Player.TempATKTurns == 0 {
			g.Player.TempATKBonus = 0
			g.addMessage("Might fades.")
		}
	}

	g.cleanupDeadEnemies()
	for _, e := range g.Enemies {
		if !e.Alive {
			continue
		}
		if !g.Tiles[e.Y][e.X].Visible {
			continue
		}

		// Bleed tick
		if e.Bleed > 0 {
			e.HP -= 2
			e.Bleed--
			if e.HP <= 0 {
				e.HP = 0
				e.Alive = false
				g.Kills++
				gold := e.goldDrop()
				g.Player.Gold += gold
				g.addMessage(fmt.Sprintf("The %s bleeds to death! +%dg", e.Name, gold))
				continue
			}
			g.addMessage(fmt.Sprintf("The %s bleeds for 2 damage.", e.Name))
		}

		// Mimic stays still until revealed
		if e.Type == EntityMimic && !e.IsRevealed {
			continue
		}

		// Burn tick (before acting)
		if e.Burn > 0 {
			e.HP -= 3
			e.Burn--
			if e.HP <= 0 {
				e.HP = 0
				e.Alive = false
				g.Kills++
				gold := e.goldDrop()
				g.Player.Gold += gold
				g.addMessage(fmt.Sprintf("The %s burns to death! +%dg", e.Name, gold))
				continue
			}
			g.addMessage(fmt.Sprintf("The %s burns for 3 damage.", e.Name))
		}

		// Boss phase-2 triggers (before action)
		if e.Type == EntityGoblinKing && !e.BossPhase2 && e.HP*2 < e.MaxHP {
			e.BossPhase2 = true
			g.addMessage("The Goblin King calls for aid!")
			// Spawn 1 goblin adjacent
			for _, off := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				ax, ay := e.X+off[0], e.Y+off[1]
				if ax >= 0 && ay >= 0 && ax < MapW && ay < MapH &&
					g.Tiles[ay][ax].Type == TileFloor && g.enemyAt(ax, ay) == nil {
					minion := NewGoblin(ax, ay)
					g.applyDifficultyHP(minion)
					g.Enemies = append(g.Enemies, minion)
					break
				}
			}
		}
		if e.Type == EntityOrcWarchief {
			e.EnrageTurns--
			if e.EnrageTurns <= 0 {
				e.EnrageTurns = 2
				if e.EnrageStacks < 5 {
					e.EnrageStacks++
					e.Atk++
					g.addMessage("The Warchief rages! [ATK↑]")
				}
			}
		}

		// Freeze check
		if e.Frozen > 0 {
			e.Frozen--
			g.addMessage(fmt.Sprintf("The %s is frozen! (skips turn)", e.Name))
			continue
		}

		dx := g.Player.X - e.X
		dy := g.Player.Y - e.Y

		// Ranged attack
		if e.RangeAttack > 0 {
			cheby := max(iAbs(dx), iAbs(dy))
			if cheby <= e.RangeAttack {
				if g.doEnemyAttack(e, true) {
					return
				}
				continue
			}
			g.moveEnemy(e)
			continue
		}

		// Adjacent: melee attack
		if iAbs(dx)+iAbs(dy) == 1 {
			if g.doEnemyAttack(e, false) {
				return
			}
			continue
		}

		// Move (Brute moves twice)
		steps := e.MoveSpeed
		if steps < 1 {
			steps = 1
		}
		for i := 0; i < steps; i++ {
			ddx := g.Player.X - e.X
			ddy := g.Player.Y - e.Y
			if iAbs(ddx)+iAbs(ddy) == 1 {
				break // already adjacent
			}
			g.moveEnemy(e)
		}
	}
	g.cleanupDeadEnemies()
	g.tickShooters()
	g.tickMovingTraps()
	g.checkChallengeRooms()
}

func (g *Game) cleanupDeadEnemies() {
	live := g.Enemies[:0]
	for _, e := range g.Enemies {
		if e.Alive {
			live = append(live, e)
		}
	}
	g.Enemies = live
}

func (g *Game) bfsNextStep(e *Entity) (dx, dy int, ok bool) {
	startIdx := e.Y*MapW + e.X
	goalIdx := g.Player.Y*MapW + g.Player.X

	parent := make([]int, MapW*MapH)
	for i := range parent {
		parent[i] = -1
	}
	parent[startIdx] = startIdx
	queue := []int{startIdx}

	dirs := [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	found := false
	for len(queue) > 0 && !found {
		cur := queue[0]
		queue = queue[1:]
		cx, cy := cur%MapW, cur/MapW
		for _, d := range dirs {
			nx, ny := cx+d[0], cy+d[1]
			if nx < 0 || ny < 0 || nx >= MapW || ny >= MapH {
				continue
			}
			if g.Tiles[ny][nx].Type == TileWall {
				continue
			}
			nIdx := ny*MapW + nx
			if nIdx == goalIdx {
				parent[nIdx] = cur
				found = true
				break
			}
			if g.enemyAt(nx, ny) != nil {
				continue
			}
			if parent[nIdx] != -1 {
				continue
			}
			parent[nIdx] = cur
			queue = append(queue, nIdx)
		}
	}

	if !found {
		return 0, 0, false
	}

	// Backtrack to find the first step from start
	cur := goalIdx
	for parent[cur] != startIdx {
		cur = parent[cur]
	}
	fx, fy := cur%MapW, cur/MapW
	return fx - e.X, fy - e.Y, true
}

func (g *Game) moveEnemy(e *Entity) {
	dx, dy, ok := g.bfsNextStep(e)
	if !ok {
		return
	}
	nx, ny := e.X+dx, e.Y+dy
	if nx < 0 || ny < 0 || nx >= MapW || ny >= MapH {
		return
	}
	if g.Tiles[ny][nx].Type == TileWall {
		return
	}
	if g.enemyAt(nx, ny) != nil {
		return
	}
	if nx == g.Player.X && ny == g.Player.Y {
		return
	}
	e.X, e.Y = nx, ny
}

func (g *Game) enemyAt(x, y int) *Entity {
	for _, e := range g.Enemies {
		if e.Alive && e.X == x && e.Y == y {
			return e
		}
	}
	return nil
}

// recomputeFOV recalculates visibility then updates last-seen tracking.
func (g *Game) recomputeFOV() {
	ComputeFOV(g.Tiles, g.Player.X, g.Player.Y, g.Player.FOVRadius, MapW, MapH)
	g.updateLastSeen()
	g.checkFirstSightings()
}

// updateLastSeen records each currently-visible enemy's position for ghost rendering.
func (g *Game) updateLastSeen() {
	for _, e := range g.Enemies {
		if e.Alive && g.Tiles[e.Y][e.X].Visible {
			e.WasSeen = true
			e.LastSeenX = e.X
			e.LastSeenY = e.Y
		}
	}
}

// checkFirstSightings fires one-time messages when notable enemies first enter FOV.
func (g *Game) checkFirstSightings() {
	if g.Phase != PhasePlay {
		return
	}
	for _, e := range g.Enemies {
		if e.Alive && !e.Announced && g.Tiles[e.Y][e.X].Visible {
			switch e.Type {
			case EntityTroll:
				g.addMessage("A massive troll blocks the path!")
				e.Announced = true
			case EntityGoblinKing:
				g.addMessage("The Goblin King roars!")
				e.Announced = true
			case EntityOrcWarchief:
				g.addMessage("The Orc Warchief raises his axe!")
				e.Announced = true
			case EntityLich:
				g.addMessage("The Lich awakens!")
				e.Announced = true
			}
		}
	}
}

// inRoom returns true if (x,y) is within a room's floor interior.
func inRoom(x, y int, r Room) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// pickAnyGear picks a random unused gear item from all regular pools.
func (g *Game) pickAnyGear() *Gear {
	all := append(append(append([]*Gear{}, GearWeapons...), GearArmors...), GearTrinkets...)
	var available []*Gear
	for _, item := range all {
		if !g.UsedGear[item] {
			available = append(available, item)
		}
	}
	if len(available) == 0 {
		return nil
	}
	chosen := available[rand.Intn(len(available))]
	g.UsedGear[chosen] = true
	return chosen
}

// spawnEnemyForFloor spawns a floor-appropriate enemy at (x,y).
func (g *Game) spawnEnemyForFloor(x, y int) *Entity {
	switch g.Floor {
	case 1:
		if rand.Intn(4) == 0 {
			return NewArcher(x, y)
		}
		return NewGoblin(x, y)
	case 2:
		switch rand.Intn(5) {
		case 0:
			return NewArcher(x, y)
		case 1:
			return NewVenomancer(x, y)
		case 2:
			return NewBrute(x, y)
		default:
			return NewOrc(x, y)
		}
	default: // floor 3
		switch rand.Intn(5) {
		case 0:
			return NewGuard(x, y)
		case 1:
			return NewArcher(x, y)
		case 2:
			return NewSalamander(x, y)
		case 3:
			return NewBrute(x, y)
		default:
			return NewOrc(x, y)
		}
	}
}

// spawnSpecialRoom picks one of the four special room types and sets it up.
func (g *Game) spawnSpecialRoom(rooms []Room) {
	if len(rooms) < 3 {
		return
	}
	roomType := rand.Intn(4)
	for attempt := 0; attempt < 30; attempt++ {
		idx := 1 + rand.Intn(len(rooms)-2)
		room := rooms[idx]
		cx, cy := room.Center()
		if g.occupied(cx, cy) {
			continue
		}
		switch roomType {
		case 0:
			g.spawnSacrificeRoom(room)
		case 1:
			g.spawnChallengeRoom(room)
		case 2:
			g.spawnShooterRoom(room)
		case 3:
			g.spawnMovingTrapRoom(room)
		}
		return
	}
}

// spawnSacrificeRoom: altar at center surrounded by static spike traps.
// Player steps on altar, pays HP, and receives a gear reward.
func (g *Game) spawnSacrificeRoom(room Room) {
	cx, cy := room.Center()
	gear := g.pickAnyGear()
	g.SacrificeAltar = &SacrificeAltar{X: cx, Y: cy, RewardGear: gear}

	// Surround altar with 2-3 spike traps
	offsets := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-1, -1}, {1, 1}, {-1, 1}, {1, -1}}
	rand.Shuffle(len(offsets), func(i, j int) { offsets[i], offsets[j] = offsets[j], offsets[i] })
	count := 2 + rand.Intn(2)
	placed := 0
	for _, off := range offsets {
		if placed >= count {
			break
		}
		tx, ty := cx+off[0], cy+off[1]
		if ty >= 0 && ty < MapH && tx >= 0 && tx < MapW &&
			g.Tiles[ty][tx].Type == TileFloor && !g.occupied(tx, ty) {
			g.Traps = append(g.Traps, &Trap{X: tx, Y: ty})
			placed++
		}
	}
}

// spawnChallengeRoom: enemies spawn on entry; clear them for a chest.
func (g *Game) spawnChallengeRoom(room Room) {
	cx, cy := room.Center()
	g.ChallengeRoom = &ChallengeRoom{
		Bounds:  room,
		RewardX: cx,
		RewardY: cy,
	}
}

// spawnShooterRoom: wall-mounted launchers fire periodically; chest in center.
func (g *Game) spawnShooterRoom(room Room) {
	cx, cy := room.Center()
	gear := g.pickAnyGear()
	g.Chests = append(g.Chests, &Chest{X: cx, Y: cy, Gold: 15 + rand.Intn(11), Gear: gear})

	type candidate struct{ x, y, dx, dy int }
	candidates := []candidate{
		{room.X - 1, cy, 1, 0},         // left wall → shoots right
		{room.X + room.W, cy, -1, 0},   // right wall → shoots left
		{cx, room.Y - 1, 0, 1},         // top wall → shoots down
		{cx, room.Y + room.H, 0, -1},   // bottom wall → shoots up
	}
	rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })

	count := 1 + rand.Intn(2) // 1-2 shooters
	placed := 0
	for _, c := range candidates {
		if placed >= count {
			break
		}
		if c.x >= 0 && c.y >= 0 && c.x < MapW && c.y < MapH && g.Tiles[c.y][c.x].Type == TileWall {
			period := 3 + rand.Intn(2) // fire every 3-4 turns
			g.Shooters = append(g.Shooters, &Shooter{
				X: c.x, Y: c.y, DX: c.dx, DY: c.dy,
				Timer: period, Period: period,
			})
			placed++
		}
	}
}

// spawnMovingTrapRoom: bouncing spike traps; chest in center.
func (g *Game) spawnMovingTrapRoom(room Room) {
	cx, cy := room.Center()
	gear := g.pickAnyGear()
	g.Chests = append(g.Chests, &Chest{X: cx, Y: cy, Gold: 15 + rand.Intn(11), Gear: gear})

	useHoriz := room.W >= room.H
	count := 2 + rand.Intn(2) // 2-3 moving traps

	// Divide the primary axis into count segments so traps spread across the room.
	// Alternate starting direction so they don't all move the same way.
	var inner int
	if useHoriz {
		inner = room.W - 2
	} else {
		inner = room.H - 2
	}

	for i := 0; i < count; i++ {
		seg := inner / count
		if seg < 1 {
			seg = 1
		}
		offset := i * seg
		if seg > 1 {
			offset += rand.Intn(seg)
		}
		if offset >= inner {
			offset = inner - 1
		}

		dir := 1
		if i%2 == 1 {
			dir = -1
		}

		var x, y, dx, dy int
		if useHoriz {
			dx, dy = dir, 0
			x = room.X + 1 + offset
			y = room.Y + 1 + rand.Intn(room.H-2)
		} else {
			dx, dy = 0, dir
			x = room.X + 1 + rand.Intn(room.W-2)
			y = room.Y + 1 + offset
		}

		// Avoid spawning on the chest
		if x == cx && y == cy {
			if x+1 < room.X+room.W-1 {
				x++
			} else if x-1 > room.X {
				x--
			} else {
				continue
			}
		}
		g.MovingTraps = append(g.MovingTraps, &MovingTrap{X: x, Y: y, DX: dx, DY: dy})
	}
}

// tickShooters advances all shooter timers and fires when they reach zero.
func (g *Game) tickShooters() {
	if g.Phase == PhaseGameOver || g.Phase == PhaseVictory {
		return
	}
	for _, s := range g.Shooters {
		s.Timer--
		if s.Timer > 0 {
			continue
		}
		s.Timer = s.Period
		// Trace the ray; hit player if in line
		x, y := s.X+s.DX, s.Y+s.DY
		for x >= 0 && y >= 0 && x < MapW && y < MapH && g.Tiles[y][x].Type != TileWall {
			if g.Player.X == x && g.Player.Y == y {
				dmg := 8 + rand.Intn(5) // 8-12
				g.Player.HP -= dmg
				if g.Player.HP < 0 {
					g.Player.HP = 0
				}
				g.addMessage(fmt.Sprintf("Fireball! -%d HP.", dmg))
				g.LastDamagedAt = time.Now()
				if g.Player.HP <= 0 {
					g.Player.Alive = false
					g.recordRun("Died")
					g.Phase = PhaseGameOver
				}
				break
			}
			x += s.DX
			y += s.DY
		}
	}
}

// tickMovingTraps moves all bouncing spike traps and damages the player on collision.
func (g *Game) tickMovingTraps() {
	if g.Phase == PhaseGameOver || g.Phase == PhaseVictory {
		return
	}
	for _, mt := range g.MovingTraps {
		nx, ny := mt.X+mt.DX, mt.Y+mt.DY
		// Reverse on wall hit
		if nx < 0 || ny < 0 || nx >= MapW || ny >= MapH || g.Tiles[ny][nx].Type == TileWall {
			mt.DX, mt.DY = -mt.DX, -mt.DY
			nx, ny = mt.X+mt.DX, mt.Y+mt.DY
			if nx < 0 || ny < 0 || nx >= MapW || ny >= MapH || g.Tiles[ny][nx].Type == TileWall {
				continue // pinned, stay
			}
		}
		mt.X, mt.Y = nx, ny
		if g.Player.X == nx && g.Player.Y == ny {
			dmg := 6 + rand.Intn(5) // 6-10
			g.Player.HP -= dmg
			if g.Player.HP < 0 {
				g.Player.HP = 0
			}
			g.addMessage(fmt.Sprintf("Moving spike! -%d HP.", dmg))
			g.LastDamagedAt = time.Now()
			if g.Player.HP <= 0 {
				g.Player.Alive = false
				g.recordRun("Died")
				g.Phase = PhaseGameOver
			}
		}
	}
}

// checkChallengeRooms spawns the reward chest once all challenge enemies are defeated.
func (g *Game) checkChallengeRooms() {
	cr := g.ChallengeRoom
	if cr == nil || !cr.Triggered || cr.Cleared || len(cr.Enemies) == 0 {
		return
	}
	for _, e := range cr.Enemies {
		if e.Alive {
			return
		}
	}
	cr.Cleared = true
	gear := g.pickAnyGear()
	g.Chests = append(g.Chests, &Chest{
		X: cr.RewardX, Y: cr.RewardY,
		Gold: 15 + rand.Intn(11), // 15-25g
		Gear: gear,
	})
	g.addMessage("Challenge cleared! A chest appears.")
}

func (g *Game) addMessage(msg string) {
	g.Messages = append(g.Messages, msg)
	if len(g.Messages) > 50 {
		g.Messages = g.Messages[1:]
	}
}
