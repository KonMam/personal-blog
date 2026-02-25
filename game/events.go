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
	// 1
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
	// 2
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
	// 3
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
	// 4
	{
		Title: "Crystal Vial",
		Body:  "A humming vial filled with silver light.",
		Choices: []*EventChoice{
			{
				Label: "Drink",
				Effect: func(g *Game) string {
					g.Player.ShieldCharges += 5
					return fmt.Sprintf("A shimmer surrounds you. +5 shields. (%d total)", g.Player.ShieldCharges)
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
	// 5
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
	// 6
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
	// 7
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
	// 8
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
	// 9
	{
		Title: "Weapon Shrine",
		Body:  "Runic weapons line a stone rack.",
		Choices: []*EventChoice{
			{
				Label: "Take the best one",
				Effect: func(g *Game) string {
					g.PendingGear = GearEventWeapons[rand.Intn(len(GearEventWeapons))]
					return fmt.Sprintf("You claim the %s.", g.PendingGear.Name)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave the weapons untouched."
				},
			},
		},
	},
	// 10
	{
		Title: "Armory of the Fallen",
		Body:  "A hero's armor hangs unmolested on the wall.",
		Choices: []*EventChoice{
			{
				Label: "Claim it",
				Effect: func(g *Game) string {
					g.PendingGear = GearEventArmors[rand.Intn(len(GearEventArmors))]
					return fmt.Sprintf("You don the %s.", g.PendingGear.Name)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave the armor where it hangs."
				},
			},
		},
	},
	// 11
	{
		Title: "Sacred Reliquary",
		Body:  "A sealed case holds a pulsing trinket.",
		Choices: []*EventChoice{
			{
				Label: "Take it",
				Effect: func(g *Game) string {
					g.PendingGear = GearEventTrinkets[rand.Intn(len(GearEventTrinkets))]
					return fmt.Sprintf("You pocket the %s.", g.PendingGear.Name)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave the reliquary sealed."
				},
			},
		},
	},
	// 12
	{
		Title: "Blood Price",
		Body:  "A demon offers power for pain.",
		Choices: []*EventChoice{
			{
				Label: "Accept (-20 max HP, +5 ATK)",
				Effect: func(g *Game) string {
					g.Player.BaseMaxHP -= 20
					if g.Player.BaseMaxHP < 5 {
						g.Player.BaseMaxHP = 5
					}
					g.Player.BaseAtk += 5
					g.Player.RecalcStats()
					if g.Player.HP > g.Player.MaxHP {
						g.Player.HP = g.Player.MaxHP
					}
					return "Power surges through you. -20 max HP, +5 ATK."
				},
			},
			{
				Label: "Refuse",
				Effect: func(g *Game) string {
					return "The demon vanishes with a sneer."
				},
			},
		},
	},
	// 13
	{
		Title: "Bonfire",
		Body:  "A warm bonfire crackles in the darkness.",
		Choices: []*EventChoice{
			{
				Label: "Rest (+15 HP)",
				Effect: func(g *Game) string {
					heal := 15
					if g.Player.HP+heal > g.Player.MaxHP {
						heal = g.Player.MaxHP - g.Player.HP
					}
					g.Player.HP += heal
					return fmt.Sprintf("You rest by the fire. +%d HP.", heal)
				},
			},
			{
				Label: "Study the runes (risky)",
				Effect: func(g *Game) string {
					if rand.Intn(2) == 0 {
						g.Player.BaseAtk += 2
						g.Player.RecalcStats()
						return "The runes burn into your mind. +2 ATK."
					}
					g.Player.Poison = 3
					return "The smoke is toxic. Poisoned for 3 turns."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You warm your hands and move on."
				},
			},
		},
	},
	// 14
	{
		Title: "Acid Pit",
		Body:  "A bubbling pit of corrosive acid.",
		Choices: []*EventChoice{
			{
				Label: "Splash visible enemies",
				Effect: func(g *Game) string {
					count := 0
					killed := 0
					for _, e := range g.Enemies {
						if e.Alive && g.Tiles[e.Y][e.X].Visible {
							e.HP -= 6
							count++
							if e.HP <= 0 {
								e.HP = 0
								e.Alive = false
								g.Kills++
								g.Player.Gold += e.goldDrop()
								killed++
							}
						}
					}
					if count == 0 {
						return "No enemies in sight. The acid splashes harmlessly."
					}
					if killed > 0 {
						return fmt.Sprintf("Acid burns %d enemies for 6! %d killed.", count, killed)
					}
					return fmt.Sprintf("Acid burns %d visible enemies for 6 damage each.", count)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You back away from the pit carefully."
				},
			},
		},
	},
	// 15
	{
		Title: "Magic Forge",
		Body:  "A forge still burning with blue flame.",
		Choices: []*EventChoice{
			{
				Label: "Temper weapon (-20g, +2 ATK)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 20 {
						return "Not enough gold. (need 20g)"
					}
					weapon := g.Player.Equipped[SlotWeapon]
					if weapon == nil {
						return "No weapon equipped to temper."
					}
					g.Player.Gold -= 20
					upgraded := *weapon // copy — never mutate catalog pointers
					upgraded.AtkMod += 2
					upgraded.Name = "Tempered " + weapon.Name
					upgraded.Desc = fmt.Sprintf("+%d ATK. [forged]", upgraded.AtkMod)
					g.Player.Equipped[SlotWeapon] = &upgraded
					g.Player.RecalcStats()
					return fmt.Sprintf("The forge tempers your blade. +2 ATK. (%s)", upgraded.Name)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave the forge to cool."
				},
			},
		},
	},
	// 16
	{
		Title: "Necromancer's Tome",
		Body:  "A tome bound in shadow and bone.",
		Choices: []*EventChoice{
			{
				Label: "Read it (risky)",
				Effect: func(g *Game) string {
					if rand.Intn(2) == 0 {
						g.Player.ShieldCharges += 6
						return fmt.Sprintf("Dark knowledge surrounds you. +6 shields. (%d total)", g.Player.ShieldCharges)
					}
					g.Player.Poison = 5
					return "The words rot your mind. Poisoned for 5 turns."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You close the tome without reading."
				},
			},
		},
	},
	// 17
	{
		Title: "Phoenix Feather",
		Body:  "A feather radiates intense heat.",
		Choices: []*EventChoice{
			{
				Label: "Crush it (burn all visible enemies)",
				Effect: func(g *Game) string {
					count := 0
					for _, e := range g.Enemies {
						if e.Alive && g.Tiles[e.Y][e.X].Visible {
							e.Burn = 4
							count++
						}
					}
					if count == 0 {
						return "No enemies in sight. The feather smolders to ash."
					}
					return fmt.Sprintf("The feather ignites! %d enemies catch fire (Burn 4).", count)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave the feather where it glows."
				},
			},
		},
	},
	// 18
	{
		Title: "Warrior's Trophy",
		Body:  "A trophy from a fallen champion.",
		Choices: []*EventChoice{
			{
				Label: "Claim the gold (+12g)",
				Effect: func(g *Game) string {
					g.Player.Gold += 12
					return "You pocket the gold. +12g."
				},
			},
			{
				Label: "Study their technique (+1 DEF)",
				Effect: func(g *Game) string {
					g.Player.BaseDef++
					g.Player.RecalcStats()
					return "You learn from the champion. +1 DEF."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave the trophy in peace."
				},
			},
		},
	},
	// 19
	{
		Title: "Trap Cache",
		Body:  "A suspicious pile of crates. Could be rigged.",
		Choices: []*EventChoice{
			{
				Label: "Loot it (risky)",
				Effect: func(g *Game) string {
					if rand.Intn(2) == 0 {
						g.Player.Gold += 25
						return "No trap! You find 25g inside."
					}
					g.Player.HP -= 8
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					g.Player.Poison = 2
					return "Trapped! -8 HP and poisoned for 2 turns."
				},
			},
			{
				Label: "Leave it alone",
				Effect: func(g *Game) string {
					return "Better safe than sorry."
				},
			},
		},
	},
	// 20
	{
		Title: "Wandering Alchemist",
		Body:  "An alchemist stirs a glowing pot.",
		Choices: []*EventChoice{
			{
				Label: "Buy potions (-15g, +2 potions)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 15 {
						return "Not enough gold. (need 15g)"
					}
					g.Player.Gold -= 15
					g.Player.Potions += 2
					return fmt.Sprintf("You buy two potions. (%d total)", g.Player.Potions)
				},
			},
			{
				Label: "Buy elixir (-30g, +10 max HP)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 30 {
						return "Not enough gold. (need 30g)"
					}
					g.Player.Gold -= 30
					g.Player.BaseMaxHP += 10
					g.Player.RecalcStats()
					heal := 10
					if g.Player.HP+heal > g.Player.MaxHP {
						heal = g.Player.MaxHP - g.Player.HP
					}
					g.Player.HP += heal
					return fmt.Sprintf("You drink the elixir. +10 max HP, +%d HP.", heal)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You have no need for potions right now."
				},
			},
		},
	},
	// 21
	{
		Title: "Oathstone",
		Body:  "A stone inscribed with ancient runes pulses faintly.",
		Choices: []*EventChoice{
			{
				Label: "Touch it (-10 HP, +8 shields)",
				Effect: func(g *Game) string {
					g.Player.HP -= 10
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					g.Player.ShieldCharges += 8
					return fmt.Sprintf("The oath binds you. -10 HP, +8 shields. (%d total)", g.Player.ShieldCharges)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave the stone untouched."
				},
			},
		},
	},
	// 22
	{
		Title: "Dark Bargain",
		Body:  "A cloaked figure waits in silence.",
		Choices: []*EventChoice{
			{
				Label: "Trade HP for power (-10 max HP, +4 ATK, +1 DEF)",
				Effect: func(g *Game) string {
					g.Player.BaseMaxHP -= 10
					if g.Player.BaseMaxHP < 5 {
						g.Player.BaseMaxHP = 5
					}
					g.Player.BaseAtk += 4
					g.Player.BaseDef++
					g.Player.RecalcStats()
					if g.Player.HP > g.Player.MaxHP {
						g.Player.HP = g.Player.MaxHP
					}
					return "The figure nods. -10 max HP, +4 ATK, +1 DEF."
				},
			},
			{
				Label: "Walk away",
				Effect: func(g *Game) string {
					return "The figure watches as you pass."
				},
			},
		},
	},
	// 23
	{
		Title: "Potion Cache",
		Body:  "Dusty shelves hold several forgotten potions.",
		Choices: []*EventChoice{
			{
				Label: "Take them (+2 potions)",
				Effect: func(g *Game) string {
					g.Player.Potions += 2
					return fmt.Sprintf("You pocket two potions. (%d total)", g.Player.Potions)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You already have enough."
				},
			},
		},
	},
	// 24
	{
		Title: "Giant Spider",
		Body:  "A giant spider guards a silk-wrapped bundle.",
		Choices: []*EventChoice{
			{
				Label: "Fight through (-12 HP, +20g)",
				Effect: func(g *Game) string {
					g.Player.HP -= 12
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					g.Player.Gold += 20
					return "You tear through the spider. -12 HP, +20g."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You back away from the massive spider."
				},
			},
		},
	},
	// 25
	{
		Title: "Sage's Study",
		Body:  "Books and scrolls fill a small alcove.",
		Choices: []*EventChoice{
			{
				Label: "Study them (+2 vision)",
				Effect: func(g *Game) string {
					g.Player.FOVRadius += 2
					return "Your perception sharpens permanently. +2 vision."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "No time for reading today."
				},
			},
		},
	},
	// 26
	{
		Title: "Well of Souls",
		Body:  "An ancient well pulses with eerie light.",
		Choices: []*EventChoice{
			{
				Label: "Drink (1-of-3 outcome)",
				Effect: func(g *Game) string {
					switch rand.Intn(3) {
					case 0:
						g.Player.HP = g.Player.MaxHP
						return "The well restores you completely. (full HP)"
					case 1:
						g.Player.Poison = 4
						return "The well poisons you! (4 turns)"
					default:
						g.Player.Gold += 15
						return "Coins glitter at the bottom. +15g."
					}
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You peer into the dark water and walk on."
				},
			},
		},
	},
	// 27
	{
		Title: "Ruined Library",
		Body:  "Decaying books hold traces of old knowledge.",
		Choices: []*EventChoice{
			{
				Label: "Study them (-10g, +1 ATK, +1 DEF)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 10 {
						return "Not enough gold. (need 10g)"
					}
					g.Player.Gold -= 10
					g.Player.BaseAtk++
					g.Player.BaseDef++
					g.Player.RecalcStats()
					return "The old texts teach you. -10g, +1 ATK, +1 DEF."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave the crumbling books alone."
				},
			},
		},
	},
	// 28
	{
		Title: "Execution Block",
		Body:  "A blood-stained block hums with latent violence.",
		Choices: []*EventChoice{
			{
				Label: "Channel it (-8 HP, +3 ATK)",
				Effect: func(g *Game) string {
					g.Player.HP -= 8
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					g.Player.BaseAtk += 3
					g.Player.RecalcStats()
					return "Violence flows through you. -8 HP, +3 ATK."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You step away from the block."
				},
			},
		},
	},
	// 29
	{
		Title: "Echo of the Past",
		Body:  "Visions of a great warrior fill your mind.",
		Choices: []*EventChoice{
			{
				Label: "Embrace them (risky)",
				Effect: func(g *Game) string {
					if rand.Intn(2) == 0 {
						g.Player.BaseAtk += 2
						g.Player.BaseDef += 2
						g.Player.RecalcStats()
						return "The warrior's skill is yours. +2 ATK, +2 DEF."
					}
					g.Player.Poison = 4
					return "The memories overwhelm you. Poisoned for 4 turns."
				},
			},
			{
				Label: "Let them fade",
				Effect: func(g *Game) string {
					return "The visions dissolve like smoke."
				},
			},
		},
	},
	// 30
	{
		Title: "Berserker's Trial",
		Body:  "A test of pain and iron will.",
		Choices: []*EventChoice{
			{
				Label: "Endure it (-12 HP, +3 DEF)",
				Effect: func(g *Game) string {
					g.Player.HP -= 12
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					g.Player.BaseDef += 3
					g.Player.RecalcStats()
					return "You endure the trial. -12 HP, +3 DEF."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You walk past the trial untested."
				},
			},
		},
	},
}
