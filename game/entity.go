//go:build js && wasm

package main

import "math/rand"

type EntityType int

const (
	EntityPlayer EntityType = iota
	EntityGoblin
	EntityOrc
	EntityTroll
)

type Entity struct {
	X, Y  int
	Char  rune
	Color string
	Name  string
	HP    int
	MaxHP int
	Atk   int
	Type  EntityType
	Alive bool
}

func NewPlayer(x, y int) *Entity {
	return &Entity{
		X: x, Y: y,
		Char:  '@',
		Color: ColorPlayer,
		Name:  "You",
		HP:    30,
		MaxHP: 30,
		Atk:   5,
		Type:  EntityPlayer,
		Alive: true,
	}
}

func NewGoblin(x, y int) *Entity {
	return &Entity{
		X: x, Y: y,
		Char:  'g',
		Color: ColorGoblin,
		Name:  "Goblin",
		HP:    8,
		MaxHP: 8,
		Atk:   2,
		Type:  EntityGoblin,
		Alive: true,
	}
}

func NewOrc(x, y int) *Entity {
	return &Entity{
		X: x, Y: y,
		Char:  'o',
		Color: ColorOrc,
		Name:  "Orc",
		HP:    15,
		MaxHP: 15,
		Atk:   4,
		Type:  EntityOrc,
		Alive: true,
	}
}

func NewTroll(x, y int) *Entity {
	return &Entity{
		X: x, Y: y,
		Char:  'T',
		Color: ColorTroll,
		Name:  "Troll",
		HP:    30,
		MaxHP: 30,
		Atk:   7,
		Type:  EntityTroll,
		Alive: true,
	}
}

// AttackTarget deals damage and returns the amount dealt.
func (attacker *Entity) AttackTarget(target *Entity) int {
	lo := attacker.Atk * 3 / 5
	hi := attacker.Atk * 7 / 5
	if hi <= lo {
		hi = lo + 1
	}
	dmg := lo + rand.Intn(hi-lo+1)
	if dmg < 1 {
		dmg = 1
	}
	target.HP -= dmg
	if target.HP <= 0 {
		target.HP = 0
		target.Alive = false
	}
	return dmg
}
