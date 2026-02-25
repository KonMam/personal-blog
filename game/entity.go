//go:build js && wasm

package main

import "math/rand"

type EntityType int

const (
	EntityPlayer EntityType = iota
	EntityGoblin
	EntityOrc
	EntityTroll
	EntityArcher
	EntityVenomancer
	EntityGuard
)

type GearSlot int

const (
	SlotWeapon  GearSlot = 0
	SlotArmor   GearSlot = 1
	SlotTrinket GearSlot = 2
)

type Gear struct {
	Name         string
	Char         rune
	Color        string
	Slot         GearSlot
	AtkMod       int
	DefMod       int
	HPMod        int
	FOVMod       int
	Thorns       int
	DoubleStrike bool
	ReachMod     int
	LifestealMod int
	DodgeMod     int
	ShieldMod    int // shield charges granted at each floor start
	BurnOnHit    bool
	BerserkerMod int // bonus flat ATK when HP < 40%
	OnKillShield int // shield charges gained on each kill
	BurnBonus    int // bonus flat damage to burning enemies
	Desc         string
}

// Gear catalogs — these are the regular pools for chests and merchant.

var GearWeapons = []*Gear{
	{Name: "Rusty Sword", Char: '†', Color: "#a0aec0", Slot: SlotWeapon, AtkMod: 2, Desc: "+2 ATK."},
	{Name: "Iron Sword", Char: '†', Color: "#e2e8f0", Slot: SlotWeapon, AtkMod: 5, Desc: "+5 ATK."},
	{Name: "Thief's Dagger", Char: '†', Color: "#68D391", Slot: SlotWeapon, AtkMod: 3, FOVMod: 2, Desc: "+3 ATK, +2 vision."},
	{Name: "Cursed Blade", Char: '†', Color: "#FC8181", Slot: SlotWeapon, AtkMod: 9, HPMod: -10, Desc: "+9 ATK, -10 max HP."},
	{Name: "Battle Axe", Char: '†', Color: "#F6AD55", Slot: SlotWeapon, AtkMod: 7, DefMod: -1, Desc: "+7 ATK, -1 DEF."},
	{Name: "Vampire Fang", Char: '†', Color: "#FC8181", Slot: SlotWeapon, AtkMod: 3, LifestealMod: 1, Desc: "+3 ATK, lifesteal 1."},
	{Name: "Longspear", Char: '†', Color: "#e2e8f0", Slot: SlotWeapon, AtkMod: 5, ReachMod: 1, Desc: "+5 ATK, reach +1."},
	{Name: "Warhammer", Char: '†', Color: "#F6AD55", Slot: SlotWeapon, AtkMod: 6, Thorns: 1, Desc: "+6 ATK, thorns 1."},
	{Name: "Twin Daggers", Char: '†', Color: "#68D391", Slot: SlotWeapon, AtkMod: 4, DoubleStrike: true, Desc: "+4 ATK, double strike."},
	{Name: "Soul Reaper", Char: '†', Color: "#9F7AEA", Slot: SlotWeapon, AtkMod: 5, OnKillShield: 1, Desc: "+5 ATK, 1 shield per kill."},
}

var GearArmors = []*Gear{
	{Name: "Leather Armor", Char: '◈', Color: "#a0aec0", Slot: SlotArmor, DefMod: 2, Desc: "+2 DEF."},
	{Name: "Chain Mail", Char: '◈', Color: "#e2e8f0", Slot: SlotArmor, DefMod: 4, AtkMod: -2, Desc: "+4 DEF, -2 ATK."},
	{Name: "Spiked Armor", Char: '◈', Color: "#FC8181", Slot: SlotArmor, DefMod: 2, Thorns: 2, Desc: "+2 DEF, thorns 2."},
	{Name: "Shadow Cloak", Char: '◈', Color: "#9F7AEA", Slot: SlotArmor, DefMod: 1, FOVMod: 2, Desc: "+1 DEF, +2 vision."},
	{Name: "Cursed Plate", Char: '◈', Color: "#E53E3E", Slot: SlotArmor, DefMod: 7, HPMod: -8, Desc: "+7 DEF, -8 max HP."},
	{Name: "Aegis", Char: '◈', Color: "#9F7AEA", Slot: SlotArmor, DefMod: 2, ShieldMod: 4, Desc: "+2 DEF, 4 shields/floor."},
	{Name: "Evasion Cloak", Char: '◈', Color: "#68D391", Slot: SlotArmor, DefMod: 1, DodgeMod: 20, Desc: "+1 DEF, 20% dodge."},
	{Name: "Scale Armor", Char: '◈', Color: "#F6AD55", Slot: SlotArmor, DefMod: 3, AtkMod: 1, Desc: "+3 DEF, +1 ATK."},
	{Name: "Thornweave", Char: '◈', Color: "#FC8181", Slot: SlotArmor, DefMod: 1, Thorns: 3, LifestealMod: 1, Desc: "+1 DEF, thorns 3, lifesteal 1."},
	{Name: "Battle Harness", Char: '◈', Color: "#F6AD55", Slot: SlotArmor, DefMod: 2, AtkMod: 1, ShieldMod: 2, Desc: "+2 DEF, +1 ATK, 2 shields/floor."},
}

var GearTrinkets = []*Gear{
	{Name: "Ring of Haste", Char: '◇', Color: "#F6AD55", Slot: SlotTrinket, DoubleStrike: true, Desc: "Attack twice per bump."},
	{Name: "Ring of Life", Char: '◇', Color: "#FC8181", Slot: SlotTrinket, LifestealMod: 2, Desc: "Lifesteal 2 per hit."},
	{Name: "Blazing Ring", Char: '◇', Color: "#F6AD55", Slot: SlotTrinket, BurnOnHit: true, Desc: "Attacks apply Burn 3."},
	{Name: "Ring of Warding", Char: '◇', Color: "#9F7AEA", Slot: SlotTrinket, ShieldMod: 6, Desc: "6 shield charges/floor."},
	{Name: "Twin Fangs", Char: '◇', Color: "#FC8181", Slot: SlotTrinket, AtkMod: 2, DoubleStrike: true, LifestealMod: 1, Desc: "+2 ATK, double strike, lifesteal 1."},
	{Name: "Berserker's Mark", Char: '◇', Color: "#FC8181", Slot: SlotTrinket, BerserkerMod: 5, Desc: "+5 ATK when below 40% HP."},
	{Name: "Executioner's Seal", Char: '◇', Color: "#9F7AEA", Slot: SlotTrinket, OnKillShield: 1, Desc: "+1 shield charge per kill."},
	{Name: "Pyromancer's Lens", Char: '◇', Color: "#F6AD55", Slot: SlotTrinket, BurnBonus: 4, Desc: "+4 damage to burning enemies."},
	{Name: "Ring of Fortitude", Char: '◇', Color: "#48BB78", Slot: SlotTrinket, HPMod: 12, ShieldMod: 1, Desc: "+12 max HP, 1 shield/floor."},
	{Name: "Thorn Ring", Char: '◇', Color: "#FC8181", Slot: SlotTrinket, Thorns: 2, DefMod: 1, Desc: "Thorns 2, +1 DEF."},
}

// Event-only gear — never spawns in chests or merchant stock.

var GearEventWeapons = []*Gear{
	{Name: "Champion's Blade", Char: '†', Color: "#F6E05E", Slot: SlotWeapon, AtkMod: 8, DefMod: 2, Desc: "+8 ATK, +2 DEF."},
	{Name: "Wraithblade", Char: '†', Color: "#9F7AEA", Slot: SlotWeapon, AtkMod: 6, BurnOnHit: true, LifestealMod: 2, Desc: "+6 ATK, burn on hit, lifesteal 2."},
}

var GearEventArmors = []*Gear{
	{Name: "Phantom Cloak", Char: '◈', Color: "#9F7AEA", Slot: SlotArmor, DefMod: 1, DodgeMod: 30, Desc: "+1 DEF, 30% dodge."},
	{Name: "Blessed Plate", Char: '◈', Color: "#F6E05E", Slot: SlotArmor, DefMod: 4, ShieldMod: 4, HPMod: 8, Desc: "+4 DEF, 4 shields/floor, +8 max HP."},
}

var GearEventTrinkets = []*Gear{
	{Name: "Void Ring", Char: '◇', Color: "#9F7AEA", Slot: SlotTrinket, HPMod: -8, DodgeMod: 30, LifestealMod: 2, Desc: "-8 max HP, 30% dodge, lifesteal 2."},
	{Name: "Deathrattle Sigil", Char: '◇', Color: "#FC8181", Slot: SlotTrinket, OnKillShield: 2, BurnBonus: 3, Desc: "+2 shields per kill, +3 vs burning."},
}

type Entity struct {
	X, Y  int
	Char  rune
	Color string
	Name  string
	HP    int
	MaxHP int
	Atk   int
	Def   int
	Type  EntityType
	Alive bool
	// Player-only fields
	BaseAtk       int
	BaseDef       int
	BaseMaxHP     int
	FOVRadius     int
	Thorns        int
	Gold          int
	Potions       int
	Equipped      [3]*Gear
	ShieldCharges int
	ShieldMod     int
	DoubleStrike  bool
	Reach         int
	Lifesteal     int
	Dodge         int
	Poison        int
	BurnOnHit     bool
	BerserkerMod  int
	OnKillShield  int
	BurnBonus     int
	// Enemy-only fields
	Burn        int
	RangeAttack int
}

func NewPlayer(x, y int) *Entity {
	return &Entity{
		X:         x,
		Y:         y,
		Char:      '@',
		Color:     ColorPlayer,
		Name:      "You",
		HP:        30,
		MaxHP:     30,
		Atk:       5,
		Def:       0,
		BaseAtk:   5,
		BaseDef:   0,
		BaseMaxHP: 30,
		FOVRadius: 8,
		Reach:     1,
		Type:      EntityPlayer,
		Alive:     true,
	}
}

func NewGoblin(x, y int) *Entity {
	return &Entity{
		X:     x,
		Y:     y,
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
		X:     x,
		Y:     y,
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
		X:     x,
		Y:     y,
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

func NewArcher(x, y int) *Entity {
	return &Entity{
		X:           x,
		Y:           y,
		Char:        'a',
		Color:       ColorArcher,
		Name:        "Archer",
		HP:          6,
		MaxHP:       6,
		Atk:         3,
		Type:        EntityArcher,
		Alive:       true,
		RangeAttack: 3,
	}
}

func NewVenomancer(x, y int) *Entity {
	return &Entity{
		X:     x,
		Y:     y,
		Char:  'v',
		Color: ColorVenomancer,
		Name:  "Venomancer",
		HP:    12,
		MaxHP: 12,
		Atk:   3,
		Type:  EntityVenomancer,
		Alive: true,
	}
}

func NewGuard(x, y int) *Entity {
	return &Entity{
		X:             x,
		Y:             y,
		Char:          'G',
		Color:         ColorGuard,
		Name:          "Guard",
		HP:            22,
		MaxHP:         22,
		Atk:           5,
		Type:          EntityGuard,
		Alive:         true,
		ShieldCharges: 3,
	}
}

// RecalcStats recomputes derived stats from base values + equipped gear.
func (p *Entity) RecalcStats() {
	atk := p.BaseAtk
	def := p.BaseDef
	maxHP := p.BaseMaxHP
	fov := p.FOVRadius
	thorns := 0
	shieldMod := 0
	lifesteal := 0
	dodge := 0
	reach := 1
	doubleStrike := false
	burnOnHit := false
	berserkerMod := 0
	onKillShield := 0
	burnBonus := 0

	for _, g := range p.Equipped {
		if g == nil {
			continue
		}
		atk += g.AtkMod
		def += g.DefMod
		maxHP += g.HPMod
		fov += g.FOVMod
		thorns += g.Thorns
		shieldMod += g.ShieldMod
		lifesteal += g.LifestealMod
		dodge += g.DodgeMod
		reach += g.ReachMod
		berserkerMod += g.BerserkerMod
		onKillShield += g.OnKillShield
		burnBonus += g.BurnBonus
		if g.DoubleStrike {
			doubleStrike = true
		}
		if g.BurnOnHit {
			burnOnHit = true
		}
	}

	if atk < 1 {
		atk = 1
	}
	if def < 0 {
		def = 0
	}
	if maxHP < 1 {
		maxHP = 1
	}
	if fov < 3 {
		fov = 3
	}
	if reach < 1 {
		reach = 1
	}
	if dodge > 100 {
		dodge = 100
	}
	if lifesteal > 4 {
		lifesteal = 4
	}

	p.Atk = atk
	p.Def = def
	p.Thorns = thorns
	if p.HP > maxHP {
		p.HP = maxHP
	}
	p.MaxHP = maxHP
	p.FOVRadius = fov
	p.ShieldMod = shieldMod
	p.Lifesteal = lifesteal
	p.Dodge = dodge
	p.Reach = reach
	p.DoubleStrike = doubleStrike
	p.BurnOnHit = burnOnHit
	p.BerserkerMod = berserkerMod
	p.OnKillShield = onKillShield
	p.BurnBonus = burnBonus
}

// CalcDamage returns a randomized damage roll for this entity.
func (e *Entity) CalcDamage() int {
	lo := e.Atk * 3 / 5
	hi := e.Atk * 7 / 5
	if hi <= lo {
		hi = lo + 1
	}
	dmg := lo + rand.Intn(hi-lo+1)
	if dmg < 1 {
		dmg = 1
	}
	return dmg
}

// goldDrop returns gold dropped when this entity is killed.
func (e *Entity) goldDrop() int {
	switch e.Type {
	case EntityGoblin:
		return 2 + rand.Intn(4) // 2-5
	case EntityOrc:
		return 6 + rand.Intn(7) // 6-12
	case EntityTroll:
		return 20 + rand.Intn(11) // 20-30
	case EntityArcher:
		return 4 + rand.Intn(5) // 4-8
	case EntityVenomancer:
		return 5 + rand.Intn(4) // 5-8
	case EntityGuard:
		return 15 + rand.Intn(8) // 15-22
	}
	return 0
}
