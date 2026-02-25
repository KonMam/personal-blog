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
	EntityGoblinKing
	EntityOrcWarchief
	EntityLich
	EntityBrute
	EntityMimic
	EntitySalamander
)

type GearSlot int

const (
	SlotWeapon  GearSlot = 0
	SlotArmor   GearSlot = 1
	SlotTrinket GearSlot = 2
)

type PotionType int

const (
	PotionHealing  PotionType = iota
	PotionAntidote            // clear burn+poison, +4 HP
	PotionMight               // TempATK+5 for 3 turns
	PotionGreater             // +25 HP
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
	Cursed       bool
	CursePenalty int  // extra incoming damage per hit while equipped
	FreezeChance int  // % chance to freeze enemy on hit
	BleedOnHit   bool // apply Bleed 2 on each hit
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
	{Name: "Plague Dagger", Char: '†', Color: "#FC8181", Slot: SlotWeapon, AtkMod: 8, DoubleStrike: true, Cursed: true, CursePenalty: 4, Desc: "+8 ATK, double strike (cursed: +4 dmg/hit)"},
	{Name: "Glacial Edge", Char: '†', Color: "#90CDF4", Slot: SlotWeapon, AtkMod: 5, FreezeChance: 30, Desc: "+5 ATK, 30% freeze on hit"},
	{Name: "Serrated Blade", Char: '†', Color: "#FC8181", Slot: SlotWeapon, AtkMod: 5, BleedOnHit: true, Desc: "+5 ATK, bleed on hit (+2 bleed)"},
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
	{Name: "Hexplate", Char: '◈', Color: "#FC8181", Slot: SlotArmor, DefMod: 8, Cursed: true, CursePenalty: 3, Desc: "+8 DEF (cursed: +3 dmg/hit)"},
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
	{Name: "Soulbane Ring", Char: '◇', Color: "#9F7AEA", Slot: SlotTrinket, AtkMod: 4, DodgeMod: 15, Cursed: true, CursePenalty: 3, Desc: "+4 ATK, 15% dodge (cursed: +3 dmg/hit)"},
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
	PotionTypes   []PotionType // typed potion inventory
	Equipped      [3]*Gear
	ShieldCharges int
	ShieldMod     int
	DoubleStrike  bool
	Reach         int
	Lifesteal     int
	Dodge         int
	Poison        int
	PlayerBurn    int // turns of 3-dmg fire from Salamander
	BurnOnHit     bool
	BerserkerMod  int
	OnKillShield  int
	BurnBonus     int
	CursePenalty  int // accumulated from equipped cursed gear
	FreezeChance  int // accumulated from equipped gear
	BleedOnHit    bool
	TempATKBonus  int
	TempATKTurns  int
	// Synergy flags
	SynergyWildfire  bool
	SynergyFortress  bool
	SynergyRageDrain bool
	SynergyReactive  bool
	SynergyInferno   bool
	// Enemy-only fields
	Burn        int
	RangeAttack int
	Announced   bool // first-sight announcement already fired
	WasSeen     bool // has ever entered player FOV
	LastSeenX   int  // position when last visible
	LastSeenY   int
	// Boss fields
	IsBoss       bool
	BossPhase2   bool
	EnrageStacks int
	EnrageTurns  int
	// New enemy fields
	MoveSpeed  int  // default 1, Brute = 2
	IsRevealed bool // Mimic: false = appear as chest
	// Status effect fields (enemy-only)
	Frozen int // turns remaining frozen
	Bleed  int // turns remaining bleeding
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

func NewGoblinKing(x, y int) *Entity {
	return &Entity{
		X: x, Y: y, Char: 'K', Color: "#FC8181",
		Name: "Goblin King", HP: 22, MaxHP: 22, Atk: 5,
		Type: EntityGoblinKing, Alive: true, IsBoss: true, MoveSpeed: 1,
	}
}

func NewOrcWarchief(x, y int) *Entity {
	return &Entity{
		X: x, Y: y, Char: 'W', Color: "#F6AD55",
		Name: "Orc Warchief", HP: 30, MaxHP: 30, Atk: 6,
		Type: EntityOrcWarchief, Alive: true, IsBoss: true, EnrageTurns: 2, MoveSpeed: 1,
	}
}

func NewLich(x, y int) *Entity {
	return &Entity{
		X: x, Y: y, Char: 'L', Color: "#9F7AEA",
		Name: "Lich", HP: 28, MaxHP: 28, Atk: 6, RangeAttack: 4,
		Type: EntityLich, Alive: true, IsBoss: true, MoveSpeed: 1,
	}
}

func NewBrute(x, y int) *Entity {
	return &Entity{
		X: x, Y: y, Char: 'B', Color: "#E53E3E",
		Name: "Brute", HP: 22, MaxHP: 22, Atk: 8, Def: 1,
		Type: EntityBrute, Alive: true, MoveSpeed: 2,
	}
}

func NewMimic(x, y int) *Entity {
	return &Entity{
		X: x, Y: y, Char: '■', Color: "#F6E05E",
		Name: "Mimic", HP: 15, MaxHP: 15, Atk: 7,
		Type: EntityMimic, Alive: true, IsRevealed: false, MoveSpeed: 1,
	}
}

func NewSalamander(x, y int) *Entity {
	return &Entity{
		X: x, Y: y, Char: 's', Color: "#ED8936",
		Name: "Salamander", HP: 14, MaxHP: 14, Atk: 5,
		Type: EntitySalamander, Alive: true, MoveSpeed: 1,
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

	// New gear fields
	cursePenalty := 0
	freezeChance := 0
	bleedOnHit := false
	for _, g := range p.Equipped {
		if g == nil {
			continue
		}
		cursePenalty += g.CursePenalty
		freezeChance += g.FreezeChance
		if g.BleedOnHit {
			bleedOnHit = true
		}
	}
	if freezeChance > 100 {
		freezeChance = 100
	}

	// Synergies
	p.SynergyWildfire = doubleStrike && burnOnHit
	p.SynergyFortress = shieldMod >= 2 && onKillShield >= 1
	p.SynergyRageDrain = berserkerMod >= 3 && lifesteal >= 1
	p.SynergyReactive = thorns >= 2 && dodge >= 15
	p.SynergyInferno = burnBonus >= 3 && doubleStrike
	if p.SynergyInferno {
		burnOnHit = true
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
	p.CursePenalty = cursePenalty
	p.FreezeChance = freezeChance
	p.BleedOnHit = bleedOnHit
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
	case EntityGoblinKing:
		return 30 + rand.Intn(11) // 30-40
	case EntityOrcWarchief:
		return 35 + rand.Intn(16) // 35-50
	case EntityLich:
		return 40 + rand.Intn(16) // 40-55
	case EntityBrute:
		return 12 + rand.Intn(8) // 12-19
	case EntityMimic:
		return 15 + rand.Intn(11) // 15-25
	case EntitySalamander:
		return 8 + rand.Intn(6) // 8-13
	}
	return 0
}
