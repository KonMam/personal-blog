//go:build js && wasm

package main

import (
	"fmt"
	"math/rand"
)

const (
	MapW      = 60
	MapH      = 22
	FOVRadius = 8
	MaxFloors = 3
)

type GamePhase int

const (
	PhasePlay GamePhase = iota
	PhaseGameOver
	PhaseVictory
)

type Item struct {
	X, Y       int
	HealAmount int
}

type Game struct {
	Tiles    [][]Tile
	Player   *Entity
	Enemies  []*Entity
	Items    []*Item
	Phase    GamePhase
	Floor    int
	Messages []string
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
	ComputeFOV(g.Tiles, g.Player.X, g.Player.Y, FOVRadius, MapW, MapH)
}

func (g *Game) spawnEnemies(rooms []Room) {
	for i, room := range rooms {
		if i == 0 {
			continue // Player starts here
		}

		// 1-3 enemies per room, more on deeper floors
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
				// Boss in the last room
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
	count := 1 + rand.Intn(2) // 1-2 potions per floor
	for range count {
		idx := 1 + rand.Intn(len(rooms)-1)
		room := rooms[idx]
		x := room.X + 1 + rand.Intn(room.W-2)
		y := room.Y + 1 + rand.Intn(room.H-2)
		g.Items = append(g.Items, &Item{X: x, Y: y, HealAmount: 12})
	}
}

func (g *Game) HandleInput(key string) {
	if g.Phase != PhasePlay {
		if key == "r" || key == "R" {
			g.restart()
		}
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

func (g *Game) restart() {
	g.Floor = 1
	g.Player = nil
	g.Messages = nil
	g.Phase = PhasePlay
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
		dmg := g.Player.AttackTarget(enemy)
		if !enemy.Alive {
			g.addMessage(fmt.Sprintf("You slay the %s!", enemy.Name))
			g.removeEnemy(enemy)
		} else {
			g.addMessage(fmt.Sprintf("You hit the %s for %d damage.", enemy.Name, dmg))
		}
		g.enemyTurn()
		ComputeFOV(g.Tiles, g.Player.X, g.Player.Y, FOVRadius, MapW, MapH)
		return
	}

	// Move
	g.Player.X, g.Player.Y = nx, ny

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

	// Pick up potion
	for i, item := range g.Items {
		if item.X == nx && item.Y == ny {
			heal := item.HealAmount
			if g.Player.HP+heal > g.Player.MaxHP {
				heal = g.Player.MaxHP - g.Player.HP
			}
			g.Player.HP += heal
			g.addMessage(fmt.Sprintf("You drink a potion and restore %d HP.", heal))
			g.Items = append(g.Items[:i], g.Items[i+1:]...)
			break
		}
	}

	g.enemyTurn()
	ComputeFOV(g.Tiles, g.Player.X, g.Player.Y, FOVRadius, MapW, MapH)
}

func (g *Game) enemyTurn() {
	for _, e := range g.Enemies {
		if !e.Alive {
			continue
		}
		// Only enemies the player can currently see take their turn
		if !g.Tiles[e.Y][e.X].Visible {
			continue
		}

		dx := g.Player.X - e.X
		dy := g.Player.Y - e.Y

		// Adjacent: attack
		if iAbs(dx) <= 1 && iAbs(dy) <= 1 && (dx != 0 || dy != 0) {
			dmg := e.AttackTarget(g.Player)
			if !g.Player.Alive {
				g.Phase = PhaseGameOver
				g.addMessage("You died. Press R to restart.")
				return
			}
			g.addMessage(fmt.Sprintf("The %s hits you for %d damage.", e.Name, dmg))
			continue
		}

		// Move toward player
		g.moveEnemy(e)
	}
}

func (g *Game) moveEnemy(e *Entity) {
	dx := iSign(g.Player.X - e.X)
	dy := iSign(g.Player.Y - e.Y)

	// Try diagonal, then cardinal axes
	moves := [][2]int{{dx, dy}, {dx, 0}, {0, dy}}
	for _, m := range moves {
		nx, ny := e.X+m[0], e.Y+m[1]
		if nx < 0 || ny < 0 || nx >= MapW || ny >= MapH {
			continue
		}
		if g.Tiles[ny][nx].Type == TileWall {
			continue
		}
		if g.enemyAt(nx, ny) != nil {
			continue
		}
		if nx == g.Player.X && ny == g.Player.Y {
			continue
		}
		e.X, e.Y = nx, ny
		return
	}
}

func (g *Game) enemyAt(x, y int) *Entity {
	for _, e := range g.Enemies {
		if e.Alive && e.X == x && e.Y == y {
			return e
		}
	}
	return nil
}

func (g *Game) removeEnemy(e *Entity) {
	for i, en := range g.Enemies {
		if en == e {
			g.Enemies = append(g.Enemies[:i], g.Enemies[i+1:]...)
			return
		}
	}
}

func (g *Game) addMessage(msg string) {
	g.Messages = append(g.Messages, msg)
	if len(g.Messages) > 50 {
		g.Messages = g.Messages[1:]
	}
}
