//go:build js && wasm

package main

import (
	"fmt"
	"math/rand"
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
)

type Item struct {
	X, Y       int
	HealAmount int
}

type Chest struct {
	X, Y   int
	Gold   int
	Gear   *Gear
	Opened bool
}

type ShopItem struct {
	Name string
	Cost int
	Gear *Gear // nil = potion
	Sold bool
}

type Merchant struct {
	X, Y  int
	Stock []*ShopItem
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
	Phase       GamePhase
	Floor       int
	Messages    []string
	PendingGear *Gear
	Turns       int
	Kills       int
}

func NewGame() *Game {
	return &Game{Floor: 1, Phase: PhaseClassSelect}
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
	if g.Floor == 2 {
		g.spawnMerchant(rooms)
	}
	g.spawnChests(rooms)
	g.spawnEvents(rooms)
	g.spawnItems(rooms)

	// Refill shield charges from gear
	g.Player.ShieldCharges += g.Player.ShieldMod
	// Poison clears on floor descent
	g.Player.Poison = 0

	ComputeFOV(g.Tiles, g.Player.X, g.Player.Y, g.Player.FOVRadius, MapW, MapH)
}

func (g *Game) spawnEnemies(rooms []Room) {
	for i, room := range rooms {
		if i == 0 {
			continue // player starts here
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
				e = NewGoblin(x, y)
			case 2:
				roll := rand.Intn(4)
				switch roll {
				case 0:
					e = NewArcher(x, y)
				case 1:
					e = NewGoblin(x, y)
				default:
					e = NewOrc(x, y)
				}
			case 3:
				if i == len(rooms)-1 && n == 0 {
					e = NewTroll(x, y)
				} else {
					roll := rand.Intn(5)
					switch roll {
					case 0:
						e = NewArcher(x, y)
					case 1:
						e = NewGoblin(x, y)
					default:
						e = NewOrc(x, y)
					}
				}
			}
			if e != nil {
				g.Enemies = append(g.Enemies, e)
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
			g.Items = append(g.Items, &Item{X: x, Y: y, HealAmount: 12})
			break
		}
	}
}

// occupied returns true if a chest, merchant, event, enemy, or existing item is at (x, y).
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
	return false
}

func (g *Game) spawnChests(rooms []Room) {
	if len(rooms) < 3 {
		return
	}
	count := 1 + rand.Intn(2) // 1-2 chests
	used := map[int]bool{}
	for range count {
		for attempt := 0; attempt < 20; attempt++ {
			// Not first room (player spawn), not last room (stairs)
			idx := 1 + rand.Intn(len(rooms)-2)
			if used[idx] {
				continue
			}
			used[idx] = true
			room := rooms[idx]
			cx, cy := room.Center()

			// 50% chance to contain gear
			var gear *Gear
			if rand.Intn(2) == 0 {
				all := append(append(GearWeapons, GearArmors...), GearTrinkets...)
				gear = all[rand.Intn(len(all))]
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
	room := rooms[2]
	cx, cy := room.Center()

	wi := rand.Intn(len(GearWeapons))
	ai := rand.Intn(len(GearArmors))
	ti := rand.Intn(len(GearTrinkets))
	w := GearWeapons[wi]
	a := GearArmors[ai]
	t := GearTrinkets[ti]

	stock := []*ShopItem{
		{Name: "Healing Potion (+12 HP)", Cost: 15},
		{Name: "Greater Potion (+25 HP)", Cost: 28},
		{Name: w.Name, Cost: 35 + rand.Intn(20), Gear: w},
		{Name: a.Name, Cost: 35 + rand.Intn(20), Gear: a},
		{Name: t.Name, Cost: 40 + rand.Intn(25), Gear: t},
	}

	g.Merchant = &Merchant{X: cx, Y: cy, Stock: stock}
}

func (g *Game) spawnEvents(rooms []Room) {
	if len(rooms) < 3 {
		return
	}
	for attempt := 0; attempt < 20; attempt++ {
		// Not first room (player spawn), not last room (stairs)
		idx := 1 + rand.Intn(len(rooms)-2)
		room := rooms[idx]
		cx, cy := room.Center()

		// Skip if occupied by enemy, chest, or merchant
		if g.enemyAt(cx, cy) != nil {
			continue
		}
		occupied := false
		for _, chest := range g.Chests {
			if chest.X == cx && chest.Y == cy {
				occupied = true
				break
			}
		}
		if occupied {
			continue
		}
		if g.Merchant != nil && g.Merchant.X == cx && g.Merchant.Y == cy {
			continue
		}

		def := allEvents[rand.Intn(len(allEvents))]
		g.Events = append(g.Events, &EventSpawn{X: cx, Y: cy, Def: def})
		return
	}
}

func (g *Game) HandleInput(key string) {
	switch g.Phase {
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
		slot := g.PendingGear.Slot
		old := g.Player.Equipped[slot]
		g.Player.Equipped[slot] = g.PendingGear
		g.Player.RecalcStats()
		if old != nil {
			g.addMessage(fmt.Sprintf("Equipped %s (replaced %s).", g.PendingGear.Name, old.Name))
		} else {
			g.addMessage(fmt.Sprintf("Equipped %s.", g.PendingGear.Name))
		}
	}
	g.PendingGear = nil
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
		g.Phase = PhasePlay
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
	heal := 12
	if g.Player.HP+heal > g.Player.MaxHP {
		heal = g.Player.MaxHP - g.Player.HP
	}
	g.Player.HP += heal
	g.addMessage(fmt.Sprintf("Used potion, +%d HP. (%d left)", heal, g.Player.Potions))
	g.enemyTurn()
	ComputeFOV(g.Tiles, g.Player.X, g.Player.Y, g.Player.FOVRadius, MapW, MapH)
}

func (g *Game) restart() {
	g.Floor = 1
	g.Player = nil
	g.Messages = nil
	g.Phase = PhaseClassSelect
	g.PendingGear = nil
	g.Events = nil
	g.ActiveEvent = nil
	g.Turns = 0
	g.Kills = 0
	// newFloor() is called by selectClass() once a class is chosen
}

func (g *Game) handleClassSelectInput(key string) {
	switch key {
	case "1", "2", "3", "4":
		g.selectClass(int(key[0] - '1'))
	}
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
	ComputeFOV(g.Tiles, g.Player.X, g.Player.Y, g.Player.FOVRadius, MapW, MapH)

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
		ComputeFOV(g.Tiles, g.Player.X, g.Player.Y, g.Player.FOVRadius, MapW, MapH)
		return
	}

	// Move
	g.Player.X, g.Player.Y = nx, ny

	// Walk into merchant
	if g.Merchant != nil && g.Merchant.X == nx && g.Merchant.Y == ny {
		g.Phase = PhaseShop
		g.enemyTurn()
		ComputeFOV(g.Tiles, g.Player.X, g.Player.Y, g.Player.FOVRadius, MapW, MapH)
		return
	}

	// Walk into chest
	for _, chest := range g.Chests {
		if !chest.Opened && chest.X == nx && chest.Y == ny {
			chest.Opened = true
			g.Player.Gold += chest.Gold
			if chest.Gear != nil {
				g.addMessage(fmt.Sprintf("Chest: +%dg and found %s!", chest.Gold, chest.Gear.Name))
				g.PendingGear = chest.Gear
				g.Phase = PhaseChest
			} else {
				g.addMessage(fmt.Sprintf("Chest: +%dg gold!", chest.Gold))
			}
			g.enemyTurn()
			ComputeFOV(g.Tiles, g.Player.X, g.Player.Y, g.Player.FOVRadius, MapW, MapH)
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
		if g.Floor < MaxFloors {
			g.Floor++
			g.newFloor()
			return
		}
		g.Phase = PhaseVictory
		g.addMessage("You escaped the dungeon. Victory!")
		return
	}

	// Pick up potion (add to inventory)
	for i, item := range g.Items {
		if item.X == nx && item.Y == ny {
			g.Player.Potions++
			g.addMessage(fmt.Sprintf("Picked up potion. (%d total) [U] to use.", g.Player.Potions))
			g.Items = append(g.Items[:i], g.Items[i+1:]...)
			break
		}
	}

	g.enemyTurn()
	ComputeFOV(g.Tiles, g.Player.X, g.Player.Y, g.Player.FOVRadius, MapW, MapH)
}

// doPlayerAttack handles a player bump attack with all combat mechanics.
func (g *Game) doPlayerAttack(enemy *Entity) {
	dmg := g.Player.CalcDamage()
	enemy.HP -= dmg
	if enemy.HP <= 0 {
		enemy.HP = 0
		enemy.Alive = false
		g.Kills++
		gold := enemy.goldDrop()
		g.Player.Gold += gold
		g.addMessage(fmt.Sprintf("You slay the %s! +%dg", enemy.Name, gold))
	} else {
		g.addMessage(fmt.Sprintf("You hit the %s for %d damage.", enemy.Name, dmg))
	}
	if g.Player.Lifesteal > 0 {
		g.Player.HP = min(g.Player.MaxHP, g.Player.HP+g.Player.Lifesteal)
	}
	if g.Player.BurnOnHit && enemy.Alive {
		enemy.Burn = 3
	}
	if g.Player.DoubleStrike && enemy.Alive {
		dmg2 := g.Player.CalcDamage()
		enemy.HP -= dmg2
		if enemy.HP <= 0 {
			enemy.HP = 0
			enemy.Alive = false
			g.Kills++
			gold := enemy.goldDrop()
			g.Player.Gold += gold
			g.addMessage(fmt.Sprintf("You slay the %s! +%dg", enemy.Name, gold))
		} else {
			g.addMessage(fmt.Sprintf("You strike again for %d damage.", dmg2))
		}
		if g.Player.Lifesteal > 0 {
			g.Player.HP = min(g.Player.MaxHP, g.Player.HP+g.Player.Lifesteal)
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
		g.addMessage(fmt.Sprintf("Shield absorbs the %s's attack! (%d left)", e.Name, g.Player.ShieldCharges))
		return false
	}

	// Dodge check
	if rand.Intn(100) < g.Player.Dodge {
		g.addMessage(fmt.Sprintf("You dodge the %s's attack!", e.Name))
		return false
	}

	finalDmg := rawDmg - g.Player.Def
	if finalDmg < 1 {
		finalDmg = 1
	}
	g.Player.HP -= finalDmg
	if g.Player.HP <= 0 {
		g.Player.HP = 0
		g.Player.Alive = false
		g.Phase = PhaseGameOver
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
			g.addMessage("Poison kills you. Press R to restart.")
			return
		}
		g.addMessage(fmt.Sprintf("Poison deals 3 damage. (%d turns left)", g.Player.Poison))
	}

	g.cleanupDeadEnemies()
	for _, e := range g.Enemies {
		if !e.Alive {
			continue
		}
		if !g.Tiles[e.Y][e.X].Visible {
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

		dx := g.Player.X - e.X
		dy := g.Player.Y - e.Y

		// Ranged attack (Archer)
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

		g.moveEnemy(e)
	}
	g.cleanupDeadEnemies()
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

func (g *Game) addMessage(msg string) {
	g.Messages = append(g.Messages, msg)
	if len(g.Messages) > 50 {
		g.Messages = g.Messages[1:]
	}
}
