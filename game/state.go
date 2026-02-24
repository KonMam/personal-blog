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
	Phase       GamePhase
	Floor       int
	Messages    []string
	PendingGear *Gear
	Turns       int
	Kills       int
}

func NewGame() *Game {
	g := &Game{Floor: 1}
	g.newFloor()
	return g
}

func (g *Game) newFloor() {
	tiles, rooms := GenerateDungeon(MapW, MapH)
	g.Tiles = tiles
	g.Enemies = nil
	g.Items = nil
	g.Chests = nil
	g.Merchant = nil

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
	g.spawnItems(rooms)
	g.spawnChests(rooms)
	if g.Floor == 2 {
		g.spawnMerchant(rooms)
	}
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
				if rand.Intn(2) == 0 {
					e = NewOrc(x, y)
				} else {
					e = NewGoblin(x, y)
				}
			case 3:
				if i == len(rooms)-1 && n == 0 {
					e = NewTroll(x, y)
				} else if rand.Intn(3) == 0 {
					e = NewGoblin(x, y)
				} else {
					e = NewOrc(x, y)
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
		idx := 1 + rand.Intn(len(rooms)-1)
		room := rooms[idx]
		x := room.X + 1 + rand.Intn(room.W-2)
		y := room.Y + 1 + rand.Intn(room.H-2)
		g.Items = append(g.Items, &Item{X: x, Y: y, HealAmount: 12})
	}
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
				all := append(GearWeapons, GearArmors...)
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
	w := GearWeapons[wi]
	a := GearArmors[ai]

	stock := []*ShopItem{
		{Name: "Healing Potion (+12 HP)", Cost: 15},
		{Name: "Greater Potion (+25 HP)", Cost: 28},
		{Name: w.Name, Cost: 35 + rand.Intn(20), Gear: w},
		{Name: a.Name, Cost: 35 + rand.Intn(20), Gear: a},
	}

	g.Merchant = &Merchant{X: cx, Y: cy, Stock: stock}
}

func (g *Game) HandleInput(key string) {
	switch g.Phase {
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
	case "1", "2", "3", "4":
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
	g.Phase = PhasePlay
	g.PendingGear = nil
	g.Turns = 0
	g.Kills = 0
	g.newFloor()
}

func (g *Game) movePlayer(dx, dy int) {
	nx, ny := g.Player.X+dx, g.Player.Y+dy

	if nx < 0 || ny < 0 || nx >= MapW || ny >= MapH {
		return
	}
	if g.Tiles[ny][nx].Type == TileWall {
		return
	}

	// Bump attack
	if enemy := g.enemyAt(nx, ny); enemy != nil {
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

func (g *Game) enemyTurn() {
	g.Turns++
	g.cleanupDeadEnemies()
	for _, e := range g.Enemies {
		if !e.Alive {
			continue
		}
		if !g.Tiles[e.Y][e.X].Visible {
			continue
		}

		dx := g.Player.X - e.X
		dy := g.Player.Y - e.Y

		// Adjacent: attack
		if iAbs(dx) <= 1 && iAbs(dy) <= 1 && (dx != 0 || dy != 0) {
			rawDmg := e.CalcDamage()
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
				return
			}
			g.addMessage(fmt.Sprintf("The %s hits you for %d damage.", e.Name, finalDmg))

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
