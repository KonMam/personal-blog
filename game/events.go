//go:build js && wasm

package main

import "fmt"

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
		Body:  "A crumbling shrine flickers. Something still listens.",
		Choices: []*EventChoice{
			{
				Label: "Kneel and pray (-10g)",
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
					if rng.Intn(2) == 0 {
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
					return "You walk past without kneeling."
				},
			},
		},
	},
	// 2
	{
		Title: "Murky Pool",
		Body:  "A still pool, dark as ink. Something moves beneath the surface.",
		Choices: []*EventChoice{
			{
				Label: "Drink deeply (risky)",
				Effect: func(g *Game) string {
					if rng.Intn(2) == 0 {
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
					g.Player.PotionTypes = append(g.Player.PotionTypes, PotionHealing)
					return fmt.Sprintf("You fill a flask. +1 Healing Potion. (%d total)", g.Player.Potions)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You step back. Probably wise."
				},
			},
		},
	},
	// 3
	{
		Title: "Blood Altar",
		Body:  "The blood here is never fully dry.",
		Choices: []*EventChoice{
			{
				Label: "Bleed for it (+20g)",
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
					return fmt.Sprintf("It drinks eagerly. -%d HP, +20g.", sacrifice)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You keep your blood. For now."
				},
			},
		},
	},
	// 4
	{
		Title: "Unstable Elixir",
		Body:  "A bubbling flask. It smells of power — and regret.",
		Choices: []*EventChoice{
			{
				Label: "Sip carefully (+5 shields, -6 max HP)",
				Effect: func(g *Game) string {
					g.Player.ShieldCharges += 5
					g.Player.BaseMaxHP -= 6
					if g.Player.BaseMaxHP < 5 {
						g.Player.BaseMaxHP = 5
					}
					g.Player.RecalcStats()
					if g.Player.HP > g.Player.MaxHP {
						g.Player.HP = g.Player.MaxHP
					}
					return fmt.Sprintf("A cold shimmer settles. +5 shields, -6 max HP. (%d shields)", g.Player.ShieldCharges)
				},
			},
			{
				Label: "Chug the whole thing (risky)",
				Effect: func(g *Game) string {
					if rng.Intn(2) == 0 {
						heal := 20
						if g.Player.HP+heal > g.Player.MaxHP {
							heal = g.Player.MaxHP - g.Player.HP
						}
						g.Player.HP += heal
						g.Player.ShieldCharges += 8
						return fmt.Sprintf("The elixir surges through you! +%d HP, +8 shields.", heal)
					}
					g.Player.HP -= 12
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					g.Player.Poison = 3
					return "Violent reaction! -12 HP, Poisoned for 3 turns."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You set the flask down before it decides for you."
				},
			},
		},
	},
	// 5
	{
		Title: "Alchemist's Fire",
		Body:  "The flask hisses. It wants to be thrown.",
		Choices: []*EventChoice{
			{
				Label: "Hurl it at them",
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
					return "You leave it. Still hissing."
				},
			},
		},
	},
	// 6
	{
		Title: "Dusty Tome",
		Body:  "A leather tome whose pages turn by themselves.",
		Choices: []*EventChoice{
			{
				Label: "Open it and read (risky)",
				Effect: func(g *Game) string {
					if rng.Intn(2) == 0 {
						g.Player.BaseDef += 2
						g.Player.RecalcStats()
						return "The words rearrange themselves into something useful. +2 DEF."
					}
					g.Player.BaseMaxHP -= 6
					if g.Player.BaseMaxHP < 5 {
						g.Player.BaseMaxHP = 5
					}
					g.Player.RecalcStats()
					return "The words reach back. -6 max HP."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You don't touch it."
				},
			},
		},
	},
	// 7
	{
		Title: "The Armourer's Gift",
		Body:  "A grizzled smith looks you over. 'That could be sharper. Or sturdier. Your call.'",
		Choices: []*EventChoice{
			{
				Label: "Sharpen your weapon (+1 ATK)",
				Effect: func(g *Game) string {
					weapon := g.Player.Equipped[SlotWeapon]
					if weapon == nil {
						return "No weapon equipped to sharpen."
					}
					upgraded := *weapon
					upgraded.AtkMod++
					upgraded.Name = weapon.Name + " +"
					upgraded.Desc = fmt.Sprintf("+%d ATK. [sharpened]", upgraded.AtkMod)
					g.Player.Equipped[SlotWeapon] = &upgraded
					g.Player.RecalcStats()
					return fmt.Sprintf("The smith sharpens your %s. +1 ATK.", weapon.Name)
				},
			},
			{
				Label: "Reinforce your armor (+1 DEF)",
				Effect: func(g *Game) string {
					armor := g.Player.Equipped[SlotArmor]
					if armor == nil {
						return "No armor equipped to reinforce."
					}
					upgraded := *armor
					upgraded.DefMod++
					upgraded.Name = armor.Name + " +"
					upgraded.Desc = fmt.Sprintf("+%d DEF. [reinforced]", upgraded.DefMod)
					g.Player.Equipped[SlotArmor] = &upgraded
					g.Player.RecalcStats()
					return fmt.Sprintf("The smith reinforces your %s. +1 DEF.", armor.Name)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "The smith shrugs and goes back to hammering."
				},
			},
		},
	},
	// 8
	{
		Title: "Dying Mercenary",
		Body:  "A mercenary, bleeding out. Eyes still open.",
		Choices: []*EventChoice{
			{
				Label: "Give them a potion (+2 DEF, +12g)",
				Effect: func(g *Game) string {
					if g.Player.Potions <= 0 {
						return "You have no potions to give."
					}
					g.Player.Potions--
					if len(g.Player.PotionTypes) > 0 {
						g.Player.PotionTypes = g.Player.PotionTypes[1:]
					}
					g.Player.BaseDef += 2
					g.Player.RecalcStats()
					g.Player.Gold += 12
					return fmt.Sprintf("'Worth more than the dying,' they rasp. +2 DEF, +12g. (%d potions left)", g.Player.Potions)
				},
			},
			{
				Label: "Go through their pockets",
				Effect: func(g *Game) string {
					if rng.Intn(2) == 0 {
						g.Player.Gold += 15
						return "You find 15g in their pack."
					}
					return "The pack is empty. Nothing found."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You step over them."
				},
			},
		},
	},
	// 9
	{
		Title: "Weapon Shrine",
		Body:  "Blades etched with old script hang in rows. One hums when you near it.",
		Choices: []*EventChoice{
			{
				Label: "Claim one",
				Effect: func(g *Game) string {
					g.PendingGear = GearEventWeapons[rng.Intn(len(GearEventWeapons))]
					g.UsedGear[g.PendingGear] = true
					return fmt.Sprintf("You claim the %s.", g.PendingGear.Name)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave them to hum alone."
				},
			},
		},
	},
	// 10
	{
		Title: "Armory of the Fallen",
		Body:  "A suit of armor, standing as if the hero just stepped out of it.",
		Choices: []*EventChoice{
			{
				Label: "Take it down",
				Effect: func(g *Game) string {
					g.PendingGear = GearEventArmors[rng.Intn(len(GearEventArmors))]
					g.UsedGear[g.PendingGear] = true
					return fmt.Sprintf("You don the %s.", g.PendingGear.Name)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave it standing."
				},
			},
		},
	},
	// 11
	{
		Title: "Sacred Reliquary",
		Body:  "A sealed reliquary, silver-clasped. Whatever's inside pushes against the lid.",
		Choices: []*EventChoice{
			{
				Label: "Break the seal",
				Effect: func(g *Game) string {
					g.PendingGear = GearEventTrinkets[rng.Intn(len(GearEventTrinkets))]
					g.UsedGear[g.PendingGear] = true
					return fmt.Sprintf("You pocket the %s.", g.PendingGear.Name)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave it shut."
				},
			},
		},
	},
	// 12
	{
		Title: "Blood Price",
		Body:  "A demon leans against the wall, arms folded. It's been waiting.",
		Choices: []*EventChoice{
			{
				Label: "Take the deal (-20 max HP, +5 ATK)",
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
					return "'Pleasure doing business,' it says. Then it's gone. -20 max HP, +5 ATK."
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
		Body:  "A bonfire, no explanation for who lit it. It doesn't matter.",
		Choices: []*EventChoice{
			{
				Label: "Sit and rest (+15 HP)",
				Effect: func(g *Game) string {
					heal := 15
					if g.Player.HP+heal > g.Player.MaxHP {
						heal = g.Player.MaxHP - g.Player.HP
					}
					g.Player.HP += heal
					return fmt.Sprintf("You stop thinking for a while. +%d HP.", heal)
				},
			},
			{
				Label: "Study the runes (risky)",
				Effect: func(g *Game) string {
					if rng.Intn(2) == 0 {
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
		Body:  "A pit that eats stone. You can hear it from the doorway.",
		Choices: []*EventChoice{
			{
				Label: "Kick acid at them",
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
						return "No one to splash. The acid eats the floor instead."
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
					return "You step around it."
				},
			},
		},
	},
	// 15
	{
		Title: "Magic Forge",
		Body:  "A forge burning cold blue. Nobody tends it. It doesn't need them.",
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
					return fmt.Sprintf("The flame runs along the edge and settles in. +2 ATK. (%s)", upgraded.Name)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "The forge doesn't need your permission to keep burning."
				},
			},
		},
	},
	// 16
	{
		Title: "Haunted Cell",
		Body:  "A spirit lingers here. It is angry, but it can be reasoned with.",
		Choices: []*EventChoice{
			{
				Label: "Bargain with it (-8 HP, freeze one enemy)",
				Effect: func(g *Game) string {
					var targets []*Entity
					for _, e := range g.Enemies {
						if e.Alive && g.Tiles[e.Y][e.X].Visible {
							targets = append(targets, e)
						}
					}
					if len(targets) == 0 {
						return "No visible enemies. The spirit fades without claiming a victim."
					}
					target := targets[rng.Intn(len(targets))]
					target.Frozen = 3
					g.Player.HP -= 8
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					return fmt.Sprintf("The spirit obeys. The %s is frozen for 3 turns. -8 HP.", target.Name)
				},
			},
			{
				Label: "Offer it a memory (-1 shield, freeze all visible)",
				Effect: func(g *Game) string {
					if g.Player.ShieldCharges < 1 {
						return "You have no shields to offer. The spirit ignores you."
					}
					g.Player.ShieldCharges--
					count := 0
					for _, e := range g.Enemies {
						if e.Alive && g.Tiles[e.Y][e.X].Visible {
							e.Frozen = 1
							count++
						}
					}
					if count == 0 {
						return "The spirit drinks your memory. No enemies to freeze."
					}
					return fmt.Sprintf("The spirit drinks your memory. %d enemies frozen briefly.", count)
				},
			},
			{
				Label: "Drive it out",
				Effect: func(g *Game) string {
					return "The spirit screams and dissipates."
				},
			},
		},
	},
	// 17
	{
		Title: "Phoenix Feather",
		Body:  "A feather that hasn't cooled in centuries. It knows what it could do.",
		Choices: []*EventChoice{
			{
				Label: "Draw the warmth in (+12 HP, clears ailments)",
				Effect: func(g *Game) string {
					heal := 12
					if g.Player.HP+heal > g.Player.MaxHP {
						heal = g.Player.MaxHP - g.Player.HP
					}
					g.Player.HP += heal
					g.Player.Poison = 0
					g.Player.PlayerBurn = 0
					return fmt.Sprintf("The feather's warmth cleanses you. +%d HP, all ailments cleared.", heal)
				},
			},
			{
				Label: "Crush it in your fist (ignites all visible, burns you too)",
				Effect: func(g *Game) string {
					count := 0
					for _, e := range g.Enemies {
						if e.Alive && g.Tiles[e.Y][e.X].Visible {
							e.Burn = 5
							count++
						}
					}
					g.Player.PlayerBurn = 2
					if count == 0 {
						return "No targets. The heat has nowhere to go. PlayerBurn 2."
					}
					return fmt.Sprintf("Fire erupts! %d enemies ignited (Burn 5). You catch some heat. PlayerBurn 2.", count)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave it glowing."
				},
			},
		},
	},
	// 18
	{
		Title: "Champion's Legacy",
		Body:  "A glass case, untouched. The champion is long dead but their things remain.",
		Choices: []*EventChoice{
			{
				Label: "Take the coins (+18g)",
				Effect: func(g *Game) string {
					g.Player.Gold += 18
					return "Heavy. The champion hoarded well. +18g."
				},
			},
			{
				Label: "Take the medicine pouch (+1 Antidote, +1 Might Draught)",
				Effect: func(g *Game) string {
					g.Player.Potions += 2
					g.Player.PotionTypes = append(g.Player.PotionTypes, PotionAntidote, PotionMight)
					return fmt.Sprintf("You pocket the pouch. +1 Antidote, +1 Might Draught. (%d total)", g.Player.Potions)
				},
			},
			{
				Label: "Read the victory scrolls (+2 ATK)",
				Effect: func(g *Game) string {
					g.Player.BaseAtk += 2
					g.Player.RecalcStats()
					return "The champion's tactics burn into your mind. +2 ATK."
				},
			},
		},
	},
	// 19
	{
		Title: "Trap Cache",
		Body:  "A stack of crates wedged into the corner. They've been here a long time.",
		Choices: []*EventChoice{
			{
				Label: "Tear it open (risky)",
				Effect: func(g *Game) string {
					if rng.Intn(2) == 0 {
						g.Player.Gold += 25
						return "No spring, no needle. Just coin. +25g."
					}
					g.Player.HP -= 8
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					g.Player.Poison = 2
					return "Something stings your hand. Fast. -8 HP, poisoned."
				},
			},
			{
				Label: "Leave it alone",
				Effect: func(g *Game) string {
					return "You've seen this before."
				},
			},
		},
	},
	// 20
	{
		Title: "Wandering Alchemist",
		Body:  "An alchemist, entirely unbothered by the dungeon. The pot smells like progress.",
		Choices: []*EventChoice{
			{
				Label: "Buy potions (-15g, +2 potions)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 15 {
						return "Not enough gold. (need 15g)"
					}
					g.Player.Gold -= 15
					g.Player.Potions += 2
					g.Player.PotionTypes = append(g.Player.PotionTypes, PotionHealing, PotionHealing)
					return fmt.Sprintf("You buy two Healing Potions. (%d total)", g.Player.Potions)
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
					return "You move on."
				},
			},
		},
	},
	// 21
	{
		Title: "Oathstone",
		Body:  "A flat stone with a single oath carved deep into it: 'I will endure.'",
		Choices: []*EventChoice{
			{
				Label: "Press your hand to it (-10 HP, +8 shields)",
				Effect: func(g *Game) string {
					g.Player.HP -= 10
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					g.Player.ShieldCharges += 8
					return fmt.Sprintf("The stone cracks when you lift your hand. The oath is yours now. -10 HP, +8 shields (%d).", g.Player.ShieldCharges)
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
		Title: "Knowledge Exchange",
		Body:  "A scholar's notebook, dense with marginalia. You can only absorb one discipline.",
		Choices: []*EventChoice{
			{
				Label: "Study warfare (+2 ATK, -1 DEF)",
				Effect: func(g *Game) string {
					g.Player.BaseAtk += 2
					g.Player.BaseDef--
					if g.Player.BaseDef < 0 {
						g.Player.BaseDef = 0
					}
					g.Player.RecalcStats()
					return "Combat instincts sharpen. +2 ATK, -1 DEF."
				},
			},
			{
				Label: "Study defence (+2 DEF, -1 ATK)",
				Effect: func(g *Game) string {
					g.Player.BaseDef += 2
					g.Player.BaseAtk--
					if g.Player.BaseAtk < 1 {
						g.Player.BaseAtk = 1
					}
					g.Player.RecalcStats()
					return "Your guard tightens. +2 DEF, -1 ATK."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You close it. Too much ink, not enough time."
				},
			},
		},
	},
	// 23
	{
		Title: "Apothecary's Cache",
		Body:  "Three vials on a dusty shelf, each labeled in a shaking hand. One of them.",
		Choices: []*EventChoice{
			{
				Label: "Take the Healing Potion (+1)",
				Effect: func(g *Game) string {
					g.Player.Potions++
					g.Player.PotionTypes = append(g.Player.PotionTypes, PotionHealing)
					return fmt.Sprintf("+1 Healing Potion. (%d total)", g.Player.Potions)
				},
			},
			{
				Label: "Take the Antidote (+1)",
				Effect: func(g *Game) string {
					g.Player.Potions++
					g.Player.PotionTypes = append(g.Player.PotionTypes, PotionAntidote)
					return fmt.Sprintf("+1 Antidote. (%d total)", g.Player.Potions)
				},
			},
			{
				Label: "Take the Might Draught (+1)",
				Effect: func(g *Game) string {
					g.Player.Potions++
					g.Player.PotionTypes = append(g.Player.PotionTypes, PotionMight)
					return fmt.Sprintf("+1 Might Draught. (%d total)", g.Player.Potions)
				},
			},
		},
	},
	// 24
	{
		Title: "Trapped Chest",
		Body:  "A chest, and you can see the spring. Someone set this up deliberately.",
		Choices: []*EventChoice{
			{
				Label: "Wrench it open (risky)",
				Effect: func(g *Game) string {
					if rng.Intn(2) == 0 {
						gear := g.pickAnyGear()
						if gear != nil {
							g.PendingGear = gear
							return "The chest springs open without triggering. Gear inside!"
						}
						return "The chest opens safely but is empty."
					}
					g.Player.HP -= 15
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					g.Player.Poison = 2
					return "The spring catches your arm. -15 HP, Poisoned."
				},
			},
			{
				Label: "Disarm the mechanism (-12g, safe)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 12 {
						return "Not enough gold to study the mechanism. (need 12g)"
					}
					gear := g.pickAnyGear()
					if gear == nil {
						return "Nothing left to find inside."
					}
					g.Player.Gold -= 12
					g.PendingGear = gear
					return "You carefully disarm the trap and open the chest."
				},
			},
			{
				Label: "Leave it",
				Effect: func(g *Game) string {
					return "Not today."
				},
			},
		},
	},
	// 25
	{
		Title: "The Cartographer",
		Body:  "A mapmaker hunches over parchment. 'I know every tunnel on this floor.'",
		Choices: []*EventChoice{
			{
				Label: "Buy the full map (-15g)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 15 {
						return "Not enough gold. (need 15g)"
					}
					g.Player.Gold -= 15
					for y := range g.Tiles {
						for x := range g.Tiles[y] {
							g.Tiles[y][x].Explored = true
						}
					}
					return "Every room and corridor is now known to you. -15g."
				},
			},
			{
				Label: "Share your notes (-5g, +2 vision)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 5 {
						return "Not enough gold. (need 5g)"
					}
					g.Player.Gold -= 5
					g.Player.FOVRadius += 2
					return "Your awareness sharpens permanently. -5g, +2 vision."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You know enough."
				},
			},
		},
	},
	// 26
	{
		Title: "Well of Souls",
		Body:  "A well that glows from within. You can't see the bottom.",
		Choices: []*EventChoice{
			{
				Label: "Cup your hands and drink",
				Effect: func(g *Game) string {
					switch rng.Intn(3) {
					case 0:
						g.Player.HP = g.Player.MaxHP
						return "Cold and clean. You feel entirely restored."
					case 1:
						g.Player.Poison = 4
						return "It tastes wrong halfway down. Poisoned."
					default:
						g.Player.Gold += 15
						return "The glow was just coins. +15g."
					}
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You walk on."
				},
			},
		},
	},
	// 27
	{
		Title: "Ruined Library",
		Body:  "Shelves of rotting books. Most of it gone, but not all.",
		Choices: []*EventChoice{
			{
				Label: "Study what remains (-10g, +1 ATK +1 DEF)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 10 {
						return "Not enough gold. (need 10g)"
					}
					g.Player.Gold -= 10
					g.Player.BaseAtk++
					g.Player.BaseDef++
					g.Player.RecalcStats()
					return "The knowledge is old but it holds. -10g, +1 ATK, +1 DEF."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You don't have time to read."
				},
			},
		},
	},
	// 28
	{
		Title: "Memory Crystal",
		Body:  "A crystal holding someone else's lifetime. You can only take part of it.",
		Choices: []*EventChoice{
			{
				Label: "Absorb their strength (+2 ATK, +1 DEF, -8 max HP)",
				Effect: func(g *Game) string {
					g.Player.BaseAtk += 2
					g.Player.BaseDef++
					g.Player.BaseMaxHP -= 8
					if g.Player.BaseMaxHP < 5 {
						g.Player.BaseMaxHP = 5
					}
					g.Player.RecalcStats()
					if g.Player.HP > g.Player.MaxHP {
						g.Player.HP = g.Player.MaxHP
					}
					return "The warrior's fury floods you. +2 ATK, +1 DEF, -8 max HP."
				},
			},
			{
				Label: "Absorb their resilience (+10 max HP, -1 ATK)",
				Effect: func(g *Game) string {
					g.Player.BaseMaxHP += 10
					g.Player.HP += 10
					g.Player.BaseAtk--
					if g.Player.BaseAtk < 1 {
						g.Player.BaseAtk = 1
					}
					g.Player.RecalcStats()
					if g.Player.HP > g.Player.MaxHP {
						g.Player.HP = g.Player.MaxHP
					}
					return "The warrior's endurance is yours. +10 max HP, -1 ATK."
				},
			},
			{
				Label: "Shatter it (+15g)",
				Effect: func(g *Game) string {
					g.Player.Gold += 15
					return "The crystal shatters into coin-dust. +15g."
				},
			},
		},
	},
	// 29
	{
		Title: "Echo of the Past",
		Body:  "A ghost of muscle memory. Someone else's battles bleeding into yours.",
		Choices: []*EventChoice{
			{
				Label: "Open yourself to them (risky)",
				Effect: func(g *Game) string {
					if rng.Intn(2) == 0 {
						g.Player.BaseAtk += 2
						g.Player.BaseDef += 2
						g.Player.RecalcStats()
						return "They fought well. Now you do. +2 ATK, +2 DEF."
					}
					g.Player.Poison = 4
					return "Too much at once. Poisoned for 4 turns."
				},
			},
			{
				Label: "Let them fade",
				Effect: func(g *Game) string {
					return "The visions dissolve."
				},
			},
		},
	},
	// 31
	{
		Title: "Altar of Cleansing",
		Body:  "An altar that smells of cold stone and old mercy.",
		Choices: []*EventChoice{
			{
				Label: "Pay for cleansing (-25g)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 25 {
						return "Not enough gold. (need 25g)"
					}
					g.Player.Gold -= 25
					lifted := 0
					for i, gear := range g.Player.Equipped {
						if gear != nil && gear.Cursed {
							g.Player.Equipped[i] = nil
							lifted++
						}
					}
					g.Player.RecalcStats()
					if lifted == 0 {
						return "No cursed gear found. Gold spent anyway."
					}
					return fmt.Sprintf("Curses lifted. (%d item(s) removed)", lifted)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You leave it. Your burdens stay."
				},
			},
		},
	},
	// 30
	{
		Title: "Proving Grounds",
		Body:  "An inscription reads: 'Only those who face the horde may claim the prize.'",
		Choices: []*EventChoice{
			{
				Label: "Enter the trial",
				Effect: func(g *Game) string {
					count := 0
					for _, e := range g.Enemies {
						if e.Alive {
							count++
						}
					}
					dmg := count * 2
					g.Player.BaseAtk += 2
					g.Player.RecalcStats()
					if dmg == 0 {
						return "All enemies are dead. The trial is granted freely. +2 ATK."
					}
					g.Player.HP -= dmg
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					return fmt.Sprintf("The trial sears you. -%d HP (%d enemies), +2 ATK.", dmg, count)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You've nothing to prove."
				},
			},
		},
	},
	// 31
	{
		Title: "The Odds Game",
		Body:  "A shadowy figure gestures to a table of bone dice. \"Care for a wager?\"",
		Choices: []*EventChoice{
			{
				Label: "Small bet (20g)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 20 {
						return "Not enough gold. (need 20g)"
					}
					g.Player.Gold -= 20
					if rng.Intn(2) == 0 {
						g.Player.Gold += 45
						return "The dice like you tonight. +25g net."
					}
					return "'Better luck next time,' it says."
				},
			},
			{
				Label: "High stakes (40g)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 40 {
						return "Not enough gold. (need 40g)"
					}
					g.Player.Gold -= 40
					if rng.Intn(20) < 7 {
						g.Player.Gold += 120
						return "You didn't deserve that. Take it anyway. +80g net."
					}
					return "The figure pockets your gold without a word."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You know better."
				},
			},
		},
	},
	// 32
	{
		Title: "Ancient Tome",
		Body:  "Crumbling pages glow with faint runes. The knowledge is yours — but at a cost.",
		Choices: []*EventChoice{
			{
				Label: "Study offense (+3 ATK, -5 Max HP)",
				Effect: func(g *Game) string {
					g.Player.BaseAtk += 3
					g.Player.BaseMaxHP -= 5
					if g.Player.BaseMaxHP < 5 {
						g.Player.BaseMaxHP = 5
					}
					g.Player.RecalcStats()
					if g.Player.HP > g.Player.MaxHP {
						g.Player.HP = g.Player.MaxHP
					}
					return "Power surges through you. +3 ATK, -5 Max HP."
				},
			},
			{
				Label: "Study defence (+8 Max HP, -2 ATK)",
				Effect: func(g *Game) string {
					g.Player.BaseMaxHP += 8
					g.Player.HP += 8
					g.Player.BaseAtk -= 2
					if g.Player.BaseAtk < 1 {
						g.Player.BaseAtk = 1
					}
					g.Player.RecalcStats()
					return "Resilience blooms. +8 Max HP, -2 ATK."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You close the tome. Some knowledge is best left unread."
				},
			},
		},
	},
	// 33
	{
		Title: "Fountain of Vigour",
		Body:  "The water is impossibly clear. It tastes like something you'd forgotten.",
		Choices: []*EventChoice{
			{
				Label: "Drink deep (+15 HP, clears ailments)",
				Effect: func(g *Game) string {
					heal := 15
					if g.Player.HP+heal > g.Player.MaxHP {
						heal = g.Player.MaxHP - g.Player.HP
					}
					g.Player.HP += heal
					g.Player.Poison = 0
					g.Player.PlayerBurn = 0
					return fmt.Sprintf("Everything that was hurting stops. +%d HP, ailments cleared.", heal)
				},
			},
			{
				Label: "Fill a vial (+1 Healing Potion)",
				Effect: func(g *Game) string {
					g.Player.Potions++
					g.Player.PotionTypes = append(g.Player.PotionTypes, PotionHealing)
					return "You fill a vial from the fountain. +1 Healing Potion."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You don't trust it."
				},
			},
		},
	},
	// 34
	{
		Title: "Wandering Trader",
		Body:  "A cloaked merchant eyes your coin pouch. \"Special price today. One item, one chance.\"",
		Choices: []*EventChoice{
			{
				Label: "Have a look (-15g)",
				Effect: func(g *Game) string {
					if g.Player.Gold < 15 {
						return "Not enough gold. (need 15g)"
					}
					gear := g.pickAnyGear()
					if gear == nil {
						return "Nothing left to sell."
					}
					g.Player.Gold -= 15
					g.PendingGear = gear
					return "The trader reveals their wares."
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You wave the trader off."
				},
			},
		},
	},
	// 35
	{
		Title: "Hidden Cache",
		Body:  "A gap in the wall where stones have fallen. Someone hid something here.",
		Choices: []*EventChoice{
			{
				Label: "Dig through it",
				Effect: func(g *Game) string {
					roll := rng.Intn(10)
					if roll < 5 {
						gear := g.pickAnyGear()
						if gear != nil {
							g.PendingGear = gear
							return "You uncover a piece of equipment in the rubble!"
						}
					}
					if roll < 8 {
						gold := 15 + rng.Intn(21)
						g.Player.Gold += gold
						return fmt.Sprintf("You find %dg buried in the dust.", gold)
					}
					dmg := 5 + rng.Intn(8)
					g.Player.HP -= dmg
					if g.Player.HP < 1 {
						g.Player.HP = 1
					}
					return fmt.Sprintf("Something was hiding in there! -%d HP.", dmg)
				},
			},
			{
				Label: "Leave",
				Effect: func(g *Game) string {
					return "You move on."
				},
			},
		},
	},
}
