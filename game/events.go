//go:build js && wasm

package main

import (
	"fmt"
	"math/rand"
)

type EventChoice struct {
	Label  string
	Effect func(g *Game) string // returns result message
}

type EventDef struct {
	Title   string
	Body    string
	Choices []*EventChoice
}

type EventSpawn struct {
	X, Y int
	Def  *EventDef
}

type ActiveEvent struct {
	Def    *EventDef
	Result string // empty = choices showing; non-empty = result showing
}

var allEvents = []*EventDef{
	{
		Title: "Ancient Shrine",
		Body:  "A crumbling shrine flickers with pale light.",
		Choices: []*EventChoice{
			{
				Label: "Pray (-10g)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 10 {
						return "Not enough gold to pray. (need 10g)"
					}
					g.Player.Gold -= 10
					g.Player.BaseMaxHP += 8
					g.Player.RecalcStats()
					heal := 8
					if g.Player.HP+heal > g.Player.MaxHP {
						heal = g.Player.MaxHP - g.Player.HP
					}
					g.Player.HP += heal
					return fmt.Sprintf("The shrine blesses you. +8 max HP, +%d HP.", heal)
				},
			},
			{
				Label: "Defile it (risky)",
				Effect: func(g *Game) string {
					if rand.Intn(2) == 0 {
						g.Player.BaseAtk += 3
						g.Player.RecalcStats()
						return "Dark power flows through you. +3 ATK."
					}
					g.Player.HP -= 10
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					return "The shrine curses you. -10 HP."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave the shrine undisturbed."
				},
			},
		},
	},
	{
		Title: "Murky Pool",
		Body:  "A still pool reflects your tired face.",
		Choices: []*EventChoice{
			{
				Label: "Drink deeply (risky)",
				Effect: func(g *Game) string {
					if rand.Intn(2) == 0 {
						g.Player.HP = g.Player.MaxHP
						return "The water is pure. You feel restored. (full HP)"
					}
					g.Player.Poison = 6
					return "The water tastes foul! Poisoned for 6 turns."
				},
			},
			{
				Label: "Fill a flask (-5g)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 5 {
						return "Not enough gold. (need 5g)"
					}
					g.Player.Gold -= 5
					g.Player.Potions++
					return fmt.Sprintf("You fill a flask. +1 potion. (%d total)", g.Player.Potions)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave the pool alone."
				},
			},
		},
	},
	{
		Title: "Blood Altar",
		Body:  "An altar slicked with old blood.",
		Choices: []*EventChoice{
			{
				Label: "Offer blood (+20g)",
				Effect: func(g *Game) string {
					sacrifice := g.Player.MaxHP / 5
					if sacrifice < 5 {
						sacrifice = 5
					}
					g.Player.HP -= sacrifice
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					g.Player.Gold += 20
					return fmt.Sprintf("You offer blood. -%d HP, +20g.", sacrifice)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You step back from the altar."
				},
			},
		},
	},
	{
		Title: "Crystal Vial",
		Body:  "A humming vial filled with silver light.",
		Choices: []*EventChoice{
			{
				Label: "Drink",
				Effect: func(g *Game) string {
					g.Player.ShieldCharges += 5
					return fmt.Sprintf("A shimmer surrounds you. +5 shield charges. (%d total)", g.Player.ShieldCharges)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You set the vial down carefully."
				},
			},
		},
	},
	{
		Title: "Alchemist's Fire",
		Body:  "A cracked flask leaks acrid smoke.",
		Choices: []*EventChoice{
			{
				Label: "Throw it",
				Effect: func(g *Game) string {
					count := 0
					for _, e := range g.Enemies {
						if e.Alive && g.Tiles[e.Y][e.X].Visible {
							e.Burn = 5
							count++
						}
					}
					if count == 0 {
						return "No enemies in sight. The flask shatters harmlessly."
					}
					return fmt.Sprintf("You hurl the flask! %d enemies catch fire.", count)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You carefully set the flask aside."
				},
			},
		},
	},
	{
		Title: "Dusty Tome",
		Body:  "A leather tome, pages rustling on their own.",
		Choices: []*EventChoice{
			{
				Label: "Study it (risky)",
				Effect: func(g *Game) string {
					if rand.Intn(2) == 0 {
						g.Player.BaseDef += 2
						g.Player.RecalcStats()
						return "Ancient knowledge fortifies you. +2 DEF."
					}
					g.Player.BaseMaxHP -= 6
					if g.Player.BaseMaxHP < 5 {
						g.Player.BaseMaxHP = 5
					}
					g.Player.RecalcStats()
					return "The tome drains your vitality. -6 max HP."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You close the tome and move on."
				},
			},
		},
	},
	{
		Title: "Lucky Bones",
		Body:  "A shady figure grins. 'Care for a wager?'",
		Choices: []*EventChoice{
			{
				Label: "Roll (bet 15g)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 15 {
						return "Not enough gold to bet. (need 15g)"
					}
					g.Player.Gold -= 15
					if rand.Intn(2) == 0 {
						g.Player.Gold += 35
						return "Lucky! You win big. +35g. (net +20g)"
					}
					return "Snake eyes. You lose. (-15g)"
				},
			},
			{
				Label: "Walk away",
				Effect: func(g *Game) string {
					return "You pocket your gold and walk away."
				},
			},
		},
	},
	{
		Title: "Dying Mercenary",
		Body:  "A mercenary slumps against the wall.",
		Choices: []*EventChoice{
			{
				Label: "Give a potion (+2 DEF, +12g)",
				Effect: func(g *Game) string {
					if g.Player.Potions <= 0 {
						return "You have no potions to give."
					}
					g.Player.Potions--
					g.Player.BaseDef += 2
					g.Player.RecalcStats()
					g.Player.Gold += 12
					return fmt.Sprintf("The mercenary thanks you. +2 DEF, +12g. (%d potions left)", g.Player.Potions)
				},
			},
			{
				Label: "Loot their pack",
				Effect: func(g *Game) string {
					if rand.Intn(2) == 0 {
						g.Player.Gold += 15
						return "You find 15g in their pack."
					}
					return "The pack is empty. Nothing found."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave the mercenary to their fate."
				},
			},
		},
	},
}
