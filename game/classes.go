//go:build js && wasm

package main

// ClassDef holds everything needed to set up a new run for a given class.
type ClassDef struct {
	Name      string
	Flavor    string // one-line playstyle description
	Color     string
	BaseHP    int
	BaseAtk   int
	BaseDef   int
	StartItem *Gear // pre-equipped; NOT in any spawn pool
}

// Class starting items — defined here and nowhere else so they never appear
// in chests or the merchant.
var (
	classItemKnight = &Gear{
		Name:      "Ironclad Seal",
		Char:      '◇',
		Color:     "#9F7AEA",
		Slot:      SlotTrinket,
		ShieldMod: 5,
		Desc:      "5 shield charges/floor.",
	}
	classItemRogue = &Gear{
		Name:         "Shadow Wrap",
		Char:         '◈',
		Color:        "#68D391",
		Slot:         SlotArmor,
		DodgeMod:     20,
		LifestealMod: 1,
		Desc:         "20% dodge, lifesteal 1.",
	}
	classItemBerserker = &Gear{
		Name:   "Warlord's Hide",
		Char:   '◈',
		Color:  "#FC8181",
		Slot:   SlotArmor,
		Thorns: 4,
		Desc:   "Thorns 4.",
	}
	classItemAlchemist = &Gear{
		Name:      "Infusion Blade",
		Char:      '†',
		Color:     "#F6AD55",
		Slot:      SlotWeapon,
		AtkMod:    3,
		BurnOnHit: true,
		Desc:      "+3 ATK, attacks burn enemies.",
	}
)

// Variant class starting items
var (
	classItemPaladin = &Gear{
		Name: "Aegis", Char: '◈', Color: "#9F7AEA",
		Slot: SlotArmor, DefMod: 2, ShieldMod: 4,
		Desc: "+2 DEF, 4 shields/floor.",
	}
	classItemShadowblade = &Gear{
		Name: "Phantom Cloak", Char: '◈', Color: "#9F7AEA",
		Slot: SlotArmor, DefMod: 1, DodgeMod: 30,
		Desc: "+1 DEF, 30% dodge.",
	}
	classItemWarlord = &Gear{
		Name: "Battle Axe", Char: '†', Color: "#F6AD55",
		Slot: SlotWeapon, AtkMod: 7, DefMod: -1,
		Desc: "+7 ATK, -1 DEF.",
	}
	classItemPyromancer = &Gear{
		Name: "Pyromancer's Lens", Char: '◇', Color: "#F6AD55",
		Slot: SlotTrinket, BurnBonus: 4,
		Desc: "+4 damage to burning enemies.",
	}
)

var classDefs = []*ClassDef{
	{
		Name:      "Knight",
		Flavor:    "Absorb hits with shield charges. Slow but unstoppable.",
		Color:     "#9F7AEA",
		BaseHP:    38,
		BaseAtk:   4,
		BaseDef:   2,
		StartItem: classItemKnight,
	},
	{
		Name:      "Rogue",
		Flavor:    "Dodge attacks and drain life. Fragile but slippery.",
		Color:     "#68D391",
		BaseHP:    22,
		BaseAtk:   7,
		BaseDef:   0,
		StartItem: classItemRogue,
	},
	{
		Name:      "Berserker",
		Flavor:    "Hit hard and make them bleed. No defense needed.",
		Color:     "#FC8181",
		BaseHP:    28,
		BaseAtk:   9,
		BaseDef:   0,
		StartItem: classItemBerserker,
	},
	{
		Name:      "Alchemist",
		Flavor:    "Set the dungeon on fire. Weak alone, deadly prepared.",
		Color:     "#F6AD55",
		BaseHP:    24,
		BaseAtk:   5,
		BaseDef:   1,
		StartItem: classItemAlchemist,
	},
	// Variant classes (indices 4-7)
	{
		Name:      "Paladin",
		Flavor:    "Faith in shields and sacrifice. Burns with holy purpose.",
		Color:     "#F6E05E",
		BaseHP:    36,
		BaseAtk:   5,
		BaseDef:   3,
		StartItem: classItemPaladin,
	},
	{
		Name:      "Shadowblade",
		Flavor:    "Strike once from the dark. Every hit matters.",
		Color:     "#9F7AEA",
		BaseHP:    20,
		BaseAtk:   9,
		BaseDef:   0,
		StartItem: classItemShadowblade,
	},
	{
		Name:      "Warlord",
		Flavor:    "Lead with fury. The dungeon bends to you.",
		Color:     "#FC8181",
		BaseHP:    32,
		BaseAtk:   10,
		BaseDef:   0,
		StartItem: classItemWarlord,
	},
	{
		Name:      "Pyromancer",
		Flavor:    "Everything burns. So do you, a little.",
		Color:     "#ED8936",
		BaseHP:    22,
		BaseAtk:   6,
		BaseDef:   0,
		StartItem: classItemPyromancer,
	},
}
