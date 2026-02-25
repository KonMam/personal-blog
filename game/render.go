//go:build js && wasm

package main

import (
	"fmt"
	"math/rand"
	"strings"
	"syscall/js"
	"time"
)

// Tile dimensions and canvas layout
const (
	TileW    = 16
	TileH    = 22
	UIHeight = 110
	CanvasW  = MapW * TileW  // 960
	CanvasH  = MapH*TileH + UIHeight
)

// Color palette -- matches the blog's dark aesthetic
const (
	ColorBg             = "#0d0d14"
	ColorWallVisible    = "#3d4663"
	ColorWallExplored   = "#1a1d2e"
	ColorFloorVisible   = "#1e2236"
	ColorFloorExplored  = "#111420"
	ColorDotVisible     = "#2e3450"
	ColorDotExplored    = "#161825"
	ColorStairsVisible  = "#F6E05E"
	ColorStairsExplored = "#6b5c18"
	ColorPlayer         = "#6C8CFF"
	ColorGoblin         = "#FC8181"
	ColorOrc            = "#F56565"
	ColorTroll          = "#E53E3E"
	ColorArcher         = "#B7791F"
	ColorPotion         = "#68D391"
	ColorUI             = "#b8bcc8"
	ColorUIDim          = "#4a5568"
	ColorAccent         = "#6C8CFF"
	ColorHPHigh         = "#48BB78"
	ColorHPMid          = "#F6AD55"
	ColorHPLow          = "#FC8181"
	ColorMsgNew         = "#e2e8f0"
	ColorMsgOld         = "#4a5568"
	ColorSeparator      = "#1a1d2e"
	ColorGold           = "#F6AD55"
	ColorChest          = "#F6E05E"
	ColorMerchant       = "#48BB78"
	ColorShield         = "#9F7AEA"
	ColorPoisonUI       = "#68D391"
	ColorEvent          = "#6C8CFF"
	ColorVenomancer     = "#9AE6B4"
	ColorGuard          = "#90CDF4"
	ColorTrap        = "#ED8936" // static spike trap ^
	ColorMovingTrap  = "#FC8181" // moving spike ◆
	ColorShooter     = "#ED8936" // shooter glyph * (orange, matches fire line)
	ColorShooterWarn = "#F6E05E" // shooter warning (1 turn before fire)
	ColorAltar       = "#E53E3E" // sacrifice altar +
	ColorBossHP      = "#F6E05E" // boss HP bar background tint
	ColorFreeze      = "#90CDF4"
	ColorBleed       = "#FC8181"
	ColorBurn        = "#ED8936"
	ColorSalamander  = "#ED8936"

	GameFont = "bold 15px 'Courier New', 'Lucida Console', monospace"
	UIFont   = "12px Inter, system-ui, sans-serif"
	UIBold   = "bold 12px Inter, system-ui, sans-serif"
)

func (g *Game) Render(ctx js.Value) {
	if g.Phase == PhaseTitle {
		g.renderTitleScreen(ctx)
		return
	}
	if g.Phase == PhaseDifficulty {
		g.renderDifficultySelect(ctx)
		return
	}
	if g.Phase == PhaseClassSelect {
		g.renderClassSelect(ctx)
		return
	}

	// Screen shake — slight translate for heavy-hit frames
	shaking := g.ShakeFrames > 0
	if shaking {
		g.ShakeFrames--
		dx := rand.Intn(7) - 3
		dy := rand.Intn(5) - 2
		ctx.Call("save")
		ctx.Call("translate", float64(dx), float64(dy))
	}

	// Background
	setFill(ctx, ColorBg)
	ctx.Call("fillRect", 0, 0, CanvasW, CanvasH)

	// Tiles
	for y := 0; y < MapH; y++ {
		for x := 0; x < MapW; x++ {
			g.renderTile(ctx, x, y)
		}
	}

	// Shooter fire-line overlay (dim = danger zone, bright = about to fire)
	for _, s := range g.Shooters {
		if s.Y < 0 || s.Y >= MapH || s.X < 0 || s.X >= MapW || !g.Tiles[s.Y][s.X].Explored {
			continue
		}
		warn := s.Timer == 1
		x, y := s.X+s.DX, s.Y+s.DY
		for x >= 0 && y >= 0 && x < MapW && y < MapH && g.Tiles[y][x].Type != TileWall {
			if g.Tiles[y][x].Visible {
				if warn {
					// About to fire: bright yellow line
					ctx.Set("fillStyle", "rgba(246, 224, 94, 0.35)")
				} else {
					// Idle: dim orange line
					ctx.Set("fillStyle", "rgba(237, 137, 54, 0.15)")
				}
				ctx.Call("fillRect", float64(x*TileW), float64(y*TileH), float64(TileW), float64(TileH))
			}
			x += s.DX
			y += s.DY
		}
	}

	// Chests
	for _, chest := range g.Chests {
		if !chest.Opened && g.Tiles[chest.Y][chest.X].Visible {
			g.drawChar(ctx, '■', chest.X, chest.Y, ColorChest)
		}
	}

	// Merchant
	if g.Merchant != nil && g.Tiles[g.Merchant.Y][g.Merchant.X].Visible {
		g.drawChar(ctx, '$', g.Merchant.X, g.Merchant.Y, ColorMerchant)
	}

	// Items (potions on floor)
	for _, item := range g.Items {
		if g.Tiles[item.Y][item.X].Visible {
			g.drawChar(ctx, '♥', item.X, item.Y, ColorPotion)
		}
	}

	// Events (? glyph)
	for _, ev := range g.Events {
		if g.Tiles[ev.Y][ev.X].Visible {
			g.drawChar(ctx, '?', ev.X, ev.Y, ColorEvent)
		}
	}

	// Sacrifice altar
	if sa := g.SacrificeAltar; sa != nil && !sa.Used && g.Tiles[sa.Y][sa.X].Visible {
		g.drawChar(ctx, '+', sa.X, sa.Y, ColorAltar)
	}

	// Static spike traps
	for _, t := range g.Traps {
		if g.Tiles[t.Y][t.X].Visible {
			g.drawChar(ctx, '^', t.X, t.Y, ColorTrap)
		}
	}

	// Moving spike traps
	for _, mt := range g.MovingTraps {
		if g.Tiles[mt.Y][mt.X].Visible {
			g.drawChar(ctx, '◆', mt.X, mt.Y, ColorMovingTrap)
		}
	}


	// Last-known enemy positions (ghosts shown when enemy is out of FOV)
	for _, e := range g.Enemies {
		if e.Alive && e.WasSeen && !g.Tiles[e.Y][e.X].Visible {
			lx, ly := e.LastSeenX, e.LastSeenY
			if g.Tiles[ly][lx].Explored && !g.Tiles[ly][lx].Visible {
				g.drawChar(ctx, e.Char, lx, ly, ColorUIDim)
			}
		}
	}

	// Enemies
	for _, e := range g.Enemies {
		if e.Alive && g.Tiles[e.Y][e.X].Visible {
			g.drawChar(ctx, e.Char, e.X, e.Y, e.Color)
			// Skip HP bar for unrevealed Mimics
			if e.Type == EntityMimic && !e.IsRevealed {
				continue
			}
			// Small HP bar at the bottom of the tile
			bx := float64(e.X*TileW) + 2
			by := float64(e.Y*TileH) + float64(TileH) - 3
			bw := float64(TileW - 4)
			ratio := float64(e.HP) / float64(e.MaxHP)
			trackColor := "#1a1d2e"
			if e.IsBoss {
				trackColor = "#2a1a00"
			}
			setFill(ctx, trackColor)
			ctx.Call("fillRect", bx, by, bw, 2)
			var hpColor string
			if e.IsBoss {
				hpColor = ColorBossHP
			} else {
				switch {
				case ratio > 0.6:
					hpColor = ColorHPHigh
				case ratio > 0.3:
					hpColor = ColorHPMid
				default:
					hpColor = ColorHPLow
				}
			}
			setFill(ctx, hpColor)
			ctx.Call("fillRect", bx, by, bw*ratio, 2)
		}
		// Freeze/Bleed status pips — tiny symbols in tile corners
		ctx.Set("textBaseline", "top")
		if e.Frozen > 0 {
			setFill(ctx, ColorFreeze)
			ctx.Set("font", "bold 8px 'Courier New', monospace")
			ctx.Set("textAlign", "right")
			ctx.Call("fillText", "*", float64((e.X+1)*TileW)-1, float64(e.Y*TileH)+1)
			ctx.Set("textAlign", "left")
		}
		if e.Bleed > 0 {
			setFill(ctx, ColorBleed)
			ctx.Set("font", "bold 8px 'Courier New', monospace")
			ctx.Call("fillText", ";", float64(e.X*TileW)+1, float64(e.Y*TileH)+1)
		}
	}

	// Player
	g.drawChar(ctx, g.Player.Char, g.Player.X, g.Player.Y, ColorPlayer)

	// Boss HP bar — full-width strip at the very bottom of the map area
	for _, e := range g.Enemies {
		if e.IsBoss && e.Alive && g.Tiles[e.Y][e.X].Visible {
			bossBarY := float64(MapH*TileH - 7)
			ratio := float64(e.HP) / float64(e.MaxHP)
			ctx.Set("fillStyle", "rgba(20, 10, 0, 0.75)")
			ctx.Call("fillRect", 0, bossBarY, float64(CanvasW), 7)
			hpColor := ColorBossHP
			if ratio < 0.3 {
				hpColor = ColorHPLow
			} else if ratio < 0.6 {
				hpColor = ColorHPMid
			}
			setFill(ctx, hpColor)
			ctx.Call("fillRect", 0, bossBarY, float64(CanvasW)*ratio, 7)
			setFill(ctx, ColorMsgNew)
			ctx.Set("font", "bold 10px 'Courier New', monospace")
			ctx.Set("textAlign", "left")
			ctx.Set("textBaseline", "middle")
			ctx.Call("fillText",
				fmt.Sprintf("  %s  %d/%d", e.Name, e.HP, e.MaxHP),
				0, bossBarY+3.5)
			break
		}
	}

	// Floating damage/heal numbers — drift upward and fade out
	ctx.Set("textBaseline", "middle")
	for i := len(g.FloatingNums) - 1; i >= 0; i-- {
		fn := &g.FloatingNums[i]
		fn.Age += 0.04
		if fn.Age >= 1.0 {
			g.FloatingNums = append(g.FloatingNums[:i], g.FloatingNums[i+1:]...)
			continue
		}
		alpha := 1.0 - fn.Age
		px := float64(fn.X*TileW) + float64(TileW)/2
		py := float64(fn.Y*TileH) - fn.Age*18
		ctx.Set("globalAlpha", alpha)
		setFill(ctx, fn.Color)
		ctx.Set("font", "bold 11px 'Courier New', monospace")
		ctx.Set("textAlign", "center")
		var label string
		if fn.Value < 0 {
			label = fmt.Sprintf("+%d", -fn.Value)
		} else {
			label = fmt.Sprintf("-%d", fn.Value)
		}
		ctx.Call("fillText", label, px, py)
	}
	ctx.Set("globalAlpha", 1.0)
	ctx.Set("textAlign", "left")
	ctx.Set("textBaseline", "top")

	// Boss announcement banner — fades in and out at center of map
	if g.BossAnnounceTimer > 0 {
		g.BossAnnounceTimer -= 0.015
		if g.BossAnnounceTimer < 0 {
			g.BossAnnounceTimer = 0
		}
		alpha := g.BossAnnounceTimer * 2.0
		if alpha > 1.0 {
			alpha = 1.0
		}
		bannerCY := float64(MapH*TileH) / 2
		ctx.Set("globalAlpha", alpha*0.90)
		setFill(ctx, "#08080f")
		ctx.Call("fillRect", 0, bannerCY-24, float64(CanvasW), 48)
		ctx.Set("globalAlpha", alpha)
		setFill(ctx, ColorHPLow)
		ctx.Set("font", "bold 20px 'Courier New', monospace")
		ctx.Set("textAlign", "center")
		ctx.Set("textBaseline", "middle")
		ctx.Call("fillText", "! "+g.BossAnnounce+" rises !", float64(CanvasW)/2, bannerCY)
		ctx.Set("globalAlpha", 1.0)
		ctx.Set("textAlign", "left")
		ctx.Set("textBaseline", "top")
	}

	// End screen-shake frame
	if shaking {
		ctx.Call("restore")
	}

	// UI strip
	g.renderUI(ctx)

	// Damage flash — red tint over the map fades out over 300ms
	if !g.LastDamagedAt.IsZero() {
		if elapsed := time.Since(g.LastDamagedAt).Milliseconds(); elapsed < 300 {
			alpha := float64(300-elapsed) / 300.0 * 0.35
			ctx.Set("fillStyle", fmt.Sprintf("rgba(200, 30, 30, %.3f)", alpha))
			ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))
		}
	}

	// Floor transition — black fade-in when descending stairs
	if g.FloorTransition > 0 {
		ctx.Set("fillStyle", fmt.Sprintf("rgba(0,0,0,%.3f)", g.FloorTransition))
		ctx.Call("fillRect", 0, 0, float64(CanvasW), float64(CanvasH))
		g.FloorTransition -= 0.06
		if g.FloorTransition < 0 {
			g.FloorTransition = 0
		}
	}

	// Overlays
	switch g.Phase {
	case PhaseGameOver:
		g.renderDeathPanel(ctx)
	case PhaseVictory:
		g.renderVictoryPanel(ctx)
	case PhaseChest:
		g.renderChestPanel(ctx)
	case PhaseShop:
		g.renderShopPanel(ctx)
	case PhaseEvent:
		g.renderEventPanel(ctx)
	}

	// Drop potion picker
	if g.ShowDropMode && g.Phase == PhasePlay {
		g.renderDropPanel(ctx)
	}

	// First-run controls hint — shown once until dismissed
	if g.ShowHint && g.Phase == PhasePlay {
		g.renderHintOverlay(ctx)
	}

	// Message log — drawn on top of everything
	if g.ShowLog {
		g.renderLogPanel(ctx)
	}
}

func (g *Game) renderTile(ctx js.Value, x, y int) {
	tile := g.Tiles[y][x]
	if !tile.Explored {
		return
	}

	px := float64(x * TileW)
	py := float64(y * TileH)

	var bg, fg string
	var ch rune

	// Wall char variants — mostly solid, variant 1 uses ▓ for rougher look
	wallChars := [4]rune{'█', '▓', '█', '█'}

	v := int(tile.Variant)

	// Biome color palettes by floor
	var wallV, wallE string
	var floorBgV, floorDotV, floorBgE, floorDotE [4]string
	switch g.Floor {
	case 2: // Purple/violet crypts
		wallV, wallE = "#4a3363", "#1f1430"
		floorBgV = [4]string{"#231e36", "#201b33", "#271f3d", "#231e36"}
		floorDotV = [4]string{"#3a2b52", "#362848", "#40305c", "#342a4c"}
		floorBgE = [4]string{"#130f1f", "#110d1c", "#150f23", "#130f1f"}
		floorDotE = [4]string{"#1a1226", "#171022", "#1e142b", "#181023"}
	case 3: // Deep red abyss
		wallV, wallE = "#5c2a2a", "#2a1010"
		floorBgV = [4]string{"#2a1515", "#271212", "#2d1818", "#2a1515"}
		floorDotV = [4]string{"#452020", "#401c1c", "#4a2424", "#3e1e1e"}
		floorBgE = [4]string{"#180a0a", "#150808", "#1a0c0c", "#180a0a"}
		floorDotE = [4]string{"#1e0c0c", "#1b0b0b", "#210e0e", "#1c0b0b"}
	default: // Floor 1: blue-gray stone
		wallV, wallE = ColorWallVisible, ColorWallExplored
		floorBgV = [4]string{"#1e2236", "#1b1f33", "#212540", "#1e2236"}
		floorDotV = [4]string{"#2e3450", "#2a3048", "#333a5a", "#282e4a"}
		floorBgE = [4]string{"#111420", "#0f1119", "#131724", "#111420"}
		floorDotE = [4]string{"#161825", "#131622", "#191e2c", "#141620"}
	}

	switch tile.Type {
	case TileWall:
		ch = wallChars[v]
		if tile.Visible {
			bg = ColorBg
			fg = wallV
		} else {
			bg = ColorBg
			fg = wallE
		}
	case TileFloor:
		ch = '·'
		if tile.Visible {
			bg = floorBgV[v]
			fg = floorDotV[v]
		} else {
			bg = floorBgE[v]
			fg = floorDotE[v]
		}
	case TileStairs:
		ch = '▼'
		if tile.Visible {
			bg = ColorFloorVisible
			fg = ColorStairsVisible
		} else {
			bg = ColorFloorExplored
			fg = ColorStairsExplored
		}
	}

	setFill(ctx, bg)
	ctx.Call("fillRect", px, py, TileW, TileH)

	setFill(ctx, fg)
	ctx.Set("font", GameFont)
	ctx.Set("textAlign", "center")
	ctx.Set("textBaseline", "middle")
	ctx.Call("fillText", string(ch), px+float64(TileW)/2, py+float64(TileH)/2)
}

func (g *Game) drawChar(ctx js.Value, ch rune, x, y int, color string) {
	px := float64(x*TileW) + float64(TileW)/2
	py := float64(y*TileH) + float64(TileH)/2
	setFill(ctx, color)
	ctx.Set("font", GameFont)
	ctx.Set("textAlign", "center")
	ctx.Set("textBaseline", "middle")
	ctx.Call("fillText", string(ch), px, py)
}

func (g *Game) renderUI(ctx js.Value) {
	sepY := float64(MapH * TileH)

	// Separator
	setFill(ctx, ColorSeparator)
	ctx.Call("fillRect", 0, sepY, CanvasW, 1)

	// UI background
	setFill(ctx, "#080810")
	ctx.Call("fillRect", 0, sepY+1, CanvasW, UIHeight-1)

	top := sepY + 8

	ctx.Set("textAlign", "left")
	ctx.Set("textBaseline", "top")

	// --- Line 1: FLOOR | HP bar | Shield | Gold | Potions | Poison ---

	// Floor label
	setFill(ctx, ColorAccent)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", fmt.Sprintf("FLOOR %d/%d", g.Floor, MaxFloors), 12, top)

	// Difficulty badge (shown for non-Normal difficulties)
	if g.Difficulty > 0 {
		var diffLabel, diffColor string
		switch g.Difficulty {
		case 1:
			diffLabel, diffColor = "HARD", ColorHPMid
		case 2:
			diffLabel, diffColor = "NIGHTMARE", ColorHPLow
		case 3:
			diffLabel, diffColor = "DAILY", ColorFreeze
		}
		setFill(ctx, diffColor)
		ctx.Set("font", UIBold)
		ctx.Call("fillText", diffLabel, 100, top)
	}

	// HP bar
	g.renderHPBar(ctx, top)

	// Shield charges (only if player has shield gear or active charges)
	if g.Player.ShieldCharges > 0 || g.Player.ShieldMod > 0 {
		setFill(ctx, ColorShield)
		ctx.Set("font", UIBold)
		ctx.Call("fillText", fmt.Sprintf("◆ %dsh", g.Player.ShieldCharges), 510, top)
	}

	// Gold
	setFill(ctx, ColorGold)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", fmt.Sprintf("¤ %dg", g.Player.Gold), 580, top)

	// Potions — 3 individual labelled slots
	setFill(ctx, ColorPotion)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", "♥", 640, top)

	const slotW = 30.0
	const slotH = 14.0
	const slotGap = 3.0
	potSlotX := 658.0
	potSymbols := [4]string{"H", "A", "M", "G"}
	potColors := [4]string{ColorPotion, ColorFreeze, ColorBurn, "#9AE6B4"}
	for i := 0; i < 3; i++ {
		sx := potSlotX + float64(i)*(slotW+slotGap)
		// Slot background
		setFill(ctx, "#1a1d2e")
		ctx.Call("fillRect", sx, top-1, slotW, slotH)
		if i < len(g.Player.PotionTypes) {
			pt := int(g.Player.PotionTypes[i])
			if pt < 0 || pt > 3 {
				pt = 0
			}
			setFill(ctx, potColors[pt])
			ctx.Set("font", UIBold)
			ctx.Set("textAlign", "center")
			ctx.Call("fillText", potSymbols[pt], sx+slotW/2, top)
			ctx.Set("textAlign", "left")
		} else {
			setFill(ctx, "#2d3748")
			ctx.Set("font", UIBold)
			ctx.Set("textAlign", "center")
			ctx.Call("fillText", "·", sx+slotW/2, top)
			ctx.Set("textAlign", "left")
		}
		// Tiny slot number in corner
		setFill(ctx, "#2d3748")
		ctx.Set("font", "8px Inter, system-ui, sans-serif")
		ctx.Set("textAlign", "right")
		ctx.Call("fillText", fmt.Sprintf("%d", i+1), sx+slotW-1, top+12)
		ctx.Set("textAlign", "left")
	}

	// Status effects + Might indicator — left-to-right cluster before ATK/DEF
	statusX := 770.0
	ctx.Set("textAlign", "left")
	if g.Player.TempATKBonus > 0 {
		setFill(ctx, ColorBurn)
		ctx.Set("font", UIBold)
		ctx.Call("fillText", fmt.Sprintf("⚡+%d(%dt)", g.Player.TempATKBonus, g.Player.TempATKTurns), statusX, top)
		statusX += 70
	}
	if g.Player.Poison > 0 {
		setFill(ctx, ColorPoisonUI)
		ctx.Set("font", UIBold)
		ctx.Call("fillText", fmt.Sprintf("PSN %d", g.Player.Poison), statusX, top)
		statusX += 52
	}
	if g.Player.PlayerBurn > 0 {
		setFill(ctx, ColorBurn)
		ctx.Set("font", UIBold)
		ctx.Call("fillText", fmt.Sprintf("BRN %d", g.Player.PlayerBurn), statusX, top)
	}

	// ATK / DEF — right-aligned
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIBold)
	ctx.Set("textAlign", "right")
	ctx.Call("fillText", fmt.Sprintf("† %d  ◈ %d", g.Player.Atk, g.Player.Def), float64(CanvasW)-12, top)
	ctx.Set("textAlign", "left")

	// --- Line 2: Gear slots + active synergies ---
	gearY := top + 22

	ctx.Set("font", UIFont)
	g.renderGearSlot(ctx, 12, gearY, SlotWeapon)
	g.renderGearSlot(ctx, 330, gearY, SlotArmor)
	g.renderGearSlot(ctx, 648, gearY, SlotTrinket)
	g.renderClassSlotRight(ctx, float64(CanvasW)-12, gearY)

	// Active synergies — small colored labels between trinket and class slots
	type synergyLabel struct{ name, color string }
	var activeSynergies []synergyLabel
	p := g.Player
	if p.SynergyWildfire {
		activeSynergies = append(activeSynergies, synergyLabel{"WILDFIRE", "#F6AD55"})
	}
	if p.SynergyFortress {
		activeSynergies = append(activeSynergies, synergyLabel{"FORTRESS", "#9F7AEA"})
	}
	if p.SynergyRageDrain {
		activeSynergies = append(activeSynergies, synergyLabel{"RAGE", "#FC8181"})
	}
	if p.SynergyReactive {
		activeSynergies = append(activeSynergies, synergyLabel{"REACTIVE", "#68D391"})
	}
	if p.SynergyInferno {
		activeSynergies = append(activeSynergies, synergyLabel{"INFERNO", "#ED8936"})
	}
	if len(activeSynergies) > 0 {
		ctx.Set("font", "bold 9px Inter, system-ui, sans-serif")
		ctx.Set("textAlign", "left")
		sx := 762.0
		for _, syn := range activeSynergies {
			setFill(ctx, syn.color)
			ctx.Call("fillText", "◆"+syn.name, sx, gearY+1)
			w := ctx.Call("measureText", "◆"+syn.name).Get("width").Float()
			sx += w + 6
		}
		ctx.Set("textAlign", "left")
	}

	// --- Lines 3-4: Messages ---
	g.renderMessages(ctx, top+46)
}

func (g *Game) renderGearSlot(ctx js.Value, x, y float64, slot GearSlot) {
	gear := g.Player.Equipped[slot]
	ctx.Set("textAlign", "left")
	ctx.Set("font", UIFont)

	if gear == nil {
		setFill(ctx, ColorUIDim)
		switch slot {
		case SlotWeapon:
			ctx.Call("fillText", "† (no weapon)", x, y)
		case SlotArmor:
			ctx.Call("fillText", "◈ (no armor)", x, y)
		case SlotTrinket:
			ctx.Call("fillText", "◇ (no trinket)", x, y)
		}
		return
	}

	// Draw icon in gear color, then measure its width for offset
	icon := string(gear.Char) + " "
	setFill(ctx, gear.Color)
	ctx.Call("fillText", icon, x, y)
	iconW := ctx.Call("measureText", icon).Get("width").Float()

	// Name only — descriptions are too long for the UI strip
	setFill(ctx, ColorUI)
	ctx.Call("fillText", gear.Name, x+iconW, y)
}

// renderClassSlotRight draws the class slot right-aligned at rightX,
// showing only the icon and name (no description) to stay compact.
func (g *Game) renderClassSlotRight(ctx js.Value, rightX, y float64) {
	ctx.Set("font", UIFont)
	gear := g.Player.Equipped[SlotClass]

	if gear == nil {
		ctx.Set("textAlign", "right")
		setFill(ctx, ColorUIDim)
		ctx.Call("fillText", "✦ (class item)", rightX, y)
		ctx.Set("textAlign", "left")
		return
	}

	lockStr := "✦ "
	iconStr := string(gear.Char) + " "
	nameStr := gear.Name

	ctx.Set("textAlign", "left")
	lockW := ctx.Call("measureText", lockStr).Get("width").Float()
	iconW := ctx.Call("measureText", iconStr).Get("width").Float()
	nameW := ctx.Call("measureText", nameStr).Get("width").Float()
	startX := rightX - (lockW + iconW + nameW)

	setFill(ctx, ColorUIDim)
	ctx.Call("fillText", lockStr, startX, y)
	setFill(ctx, gear.Color)
	ctx.Call("fillText", iconStr, startX+lockW, y)
	setFill(ctx, ColorUI)
	ctx.Call("fillText", nameStr, startX+lockW+iconW, y)
}

func (g *Game) renderHPBar(ctx js.Value, y float64) {
	labelX := float64(200)
	barX := labelX + 28.0
	barW := float64(220)
	barH := 13.0

	ratio := float64(g.Player.HP) / float64(g.Player.MaxHP)

	// Label
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Set("textAlign", "left")
	ctx.Set("textBaseline", "top")
	ctx.Call("fillText", "HP", labelX, y+1)

	// Bar track
	setFill(ctx, "#1a1d2e")
	ctx.Call("fillRect", barX, y, barW, barH)

	// Bar fill
	var barColor string
	switch {
	case ratio > 0.6:
		barColor = ColorHPHigh
	case ratio > 0.3:
		barColor = ColorHPMid
	default:
		barColor = ColorHPLow
	}
	setFill(ctx, barColor)
	ctx.Call("fillRect", barX, y, barW*ratio, barH)

	// Numbers
	setFill(ctx, ColorUI)
	ctx.Set("font", UIBold)
	ctx.Call("fillText",
		fmt.Sprintf("%d/%d", g.Player.HP, g.Player.MaxHP),
		barX+barW+8, y+1)
}

func (g *Game) renderMessages(ctx js.Value, y float64) {
	msgs := g.Messages
	if len(msgs) > 2 {
		msgs = msgs[len(msgs)-2:]
	}
	ctx.Set("font", UIFont)
	ctx.Set("textAlign", "left")
	ctx.Set("textBaseline", "top")
	for i, msg := range msgs {
		color := ColorMsgOld
		if i == len(msgs)-1 {
			color = ColorMsgNew
		}
		setFill(ctx, color)
		ctx.Call("fillText", msg, 12, y+float64(i)*18)
	}
}

func (g *Game) renderOverlay(ctx js.Value, title, sub, titleColor string) {
	cx := float64(CanvasW) / 2
	cy := float64(MapH*TileH) / 2

	// Dim the map
	ctx.Set("fillStyle", "rgba(10, 10, 20, 0.88)")
	ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))

	ctx.Set("textAlign", "center")
	ctx.Set("textBaseline", "middle")

	setFill(ctx, titleColor)
	ctx.Set("font", "bold 34px Inter, system-ui, sans-serif")
	ctx.Call("fillText", title, cx, cy-22)

	setFill(ctx, ColorUI)
	ctx.Set("font", "15px Inter, system-ui, sans-serif")
	ctx.Call("fillText", sub, cx, cy+18)
}

// gearRarityName returns the rarity label for a gear item based on its color.
func gearRarityName(gear *Gear) string {
	if gear.Cursed {
		return "Cursed"
	}
	switch gear.Color {
	case "#718096":
		return "Common"
	case "#68D391":
		return "Uncommon"
	case "#63B3ED":
		return "Rare"
	case "#9F7AEA":
		return "Epic"
	case "#F6E05E":
		return "Event"
	}
	return ""
}

func (g *Game) renderChestPanel(ctx js.Value) {
	if g.PendingGear == nil {
		return
	}

	slot := g.PendingGear.Slot
	current := g.Player.Equipped[slot]
	isUnknown := g.PendingGear.Unknown

	cx := float64(CanvasW) / 2
	cy := float64(MapH*TileH) / 2

	// Dim the map
	ctx.Set("fillStyle", "rgba(10, 10, 20, 0.85)")
	ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))

	// Compute stat diffs (only when item identity is known)
	type diffEntry struct{ text, color string }
	var diffs []diffEntry
	if !isUnknown {
		ni, oi := g.PendingGear, current
		getI := func(gear *Gear, f func(*Gear) int) int {
			if gear == nil {
				return 0
			}
			return f(gear)
		}
		getB := func(gear *Gear, f func(*Gear) bool) bool {
			if gear == nil {
				return false
			}
			return f(gear)
		}
		addNum := func(label string, delta int) {
			if delta == 0 {
				return
			}
			col := ColorHPHigh
			if delta < 0 {
				col = ColorHPLow
			}
			sign := "+"
			if delta < 0 {
				sign = ""
			}
			diffs = append(diffs, diffEntry{fmt.Sprintf("%s%d %s", sign, delta, label), col})
		}
		addBool := func(label string, gained, lost bool) {
			if gained && !lost {
				diffs = append(diffs, diffEntry{"+" + label, ColorHPHigh})
			} else if lost && !gained {
				diffs = append(diffs, diffEntry{"-" + label, ColorHPLow})
			}
		}
		addNum("ATK", getI(ni, func(g *Gear) int { return g.AtkMod })-getI(oi, func(g *Gear) int { return g.AtkMod }))
		addNum("DEF", getI(ni, func(g *Gear) int { return g.DefMod })-getI(oi, func(g *Gear) int { return g.DefMod }))
		addNum("HP", getI(ni, func(g *Gear) int { return g.HPMod })-getI(oi, func(g *Gear) int { return g.HPMod }))
		addNum("Thorns", getI(ni, func(g *Gear) int { return g.Thorns })-getI(oi, func(g *Gear) int { return g.Thorns }))
		addNum("Drain", getI(ni, func(g *Gear) int { return g.LifestealMod })-getI(oi, func(g *Gear) int { return g.LifestealMod }))
		addNum("Dodge%", getI(ni, func(g *Gear) int { return g.DodgeMod })-getI(oi, func(g *Gear) int { return g.DodgeMod }))
		addNum("Sh/fl", getI(ni, func(g *Gear) int { return g.ShieldMod })-getI(oi, func(g *Gear) int { return g.ShieldMod }))
		addNum("BurnDmg", getI(ni, func(g *Gear) int { return g.BurnBonus })-getI(oi, func(g *Gear) int { return g.BurnBonus }))
		addNum("Rage", getI(ni, func(g *Gear) int { return g.BerserkerMod })-getI(oi, func(g *Gear) int { return g.BerserkerMod }))
		addNum("Freeze%", getI(ni, func(g *Gear) int { return g.FreezeChance })-getI(oi, func(g *Gear) int { return g.FreezeChance }))
		addBool("2x Strike", getB(ni, func(g *Gear) bool { return g.DoubleStrike }), getB(oi, func(g *Gear) bool { return g.DoubleStrike }))
		addBool("Burn Hit", getB(ni, func(g *Gear) bool { return g.BurnOnHit }), getB(oi, func(g *Gear) bool { return g.BurnOnHit }))
		addBool("Bleed Hit", getB(ni, func(g *Gear) bool { return g.BleedOnHit }), getB(oi, func(g *Gear) bool { return g.BleedOnHit }))
		if ni.Cursed && (oi == nil || !oi.Cursed) {
			diffs = append(diffs, diffEntry{fmt.Sprintf("Cursed (-%d)", ni.CursePenalty), ColorHPLow})
		}
	}

	hasDiffs := len(diffs) > 0
	boxW := float64(440)
	boxH := 188.0
	if hasDiffs {
		boxH += 20
	}
	if g.PendingGear.Cursed {
		boxH += 16
	}
	bx := cx - boxW/2
	by := cy - boxH/2

	borderColor := ColorChest
	if g.PendingGear.Cursed {
		borderColor = "#FC8181"
	}
	setFill(ctx, "#10101a")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", borderColor)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textAlign", "center")
	ctx.Set("textBaseline", "top")

	y := by + 14.0

	// Header
	header := "GEAR FOUND"
	if isUnknown {
		header = "MYSTERIOUS ITEM"
	}
	setFill(ctx, borderColor)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", header, cx, y)
	y += 20

	// Cursed warning badge
	if g.PendingGear.Cursed {
		setFill(ctx, "#FC8181")
		ctx.Set("font", "bold 11px Inter, system-ui, sans-serif")
		ctx.Call("fillText", "⚠ CURSED — equipping inflicts a permanent penalty", cx, y)
		y += 16
	}

	// New gear — name + rarity (or mystery)
	if isUnknown {
		setFill(ctx, ColorUIDim)
		ctx.Set("font", "bold 15px Inter, system-ui, sans-serif")
		ctx.Call("fillText", "??? [UNKNOWN]", cx, y)
		y += 20
		setFill(ctx, ColorUIDim)
		ctx.Set("font", UIFont)
		ctx.Call("fillText", "Properties hidden until equipped.", cx, y)
		y += 20
	} else {
		setFill(ctx, g.PendingGear.Color)
		ctx.Set("font", "bold 15px Inter, system-ui, sans-serif")
		rarity := gearRarityName(g.PendingGear)
		gearLabel := string(g.PendingGear.Char) + " " + g.PendingGear.Name
		if rarity != "" {
			gearLabel += "  [" + rarity + "]"
		}
		ctx.Call("fillText", gearLabel, cx, y)
		y += 20
		setFill(ctx, ColorUI)
		ctx.Set("font", UIFont)
		ctx.Call("fillText", g.PendingGear.Desc, cx, y)
		y += 20
	}

	// Stat diffs row
	if hasDiffs {
		ctx.Set("textAlign", "left")
		ctx.Set("font", "11px Inter, system-ui, sans-serif")
		totalW := 0.0
		gaps := make([]float64, len(diffs))
		for i, d := range diffs {
			gaps[i] = ctx.Call("measureText", d.text).Get("width").Float()
			totalW += gaps[i]
		}
		totalW += float64(len(diffs)-1) * 10
		x := cx - totalW/2
		for i, d := range diffs {
			setFill(ctx, d.color)
			ctx.Call("fillText", d.text, x, y)
			x += gaps[i] + 10
		}
		y += 18
		ctx.Set("textAlign", "center")
	}

	// Divider
	setFill(ctx, ColorUIDim)
	ctx.Call("fillRect", bx+20, y, boxW-40, 1)
	y += 8

	// Current slot section
	if current != nil {
		setFill(ctx, ColorUIDim)
		ctx.Set("font", "11px Inter, system-ui, sans-serif")
		ctx.Call("fillText", "replaces", cx, y)
		y += 14
		setFill(ctx, current.Color)
		ctx.Set("font", "bold 13px Inter, system-ui, sans-serif")
		ctx.Call("fillText", string(current.Char)+" "+current.Name, cx, y)
		y += 16
		setFill(ctx, ColorUIDim)
		ctx.Set("font", UIFont)
		ctx.Call("fillText", current.Desc, cx, y)
	} else {
		setFill(ctx, ColorUIDim)
		ctx.Set("font", UIFont)
		ctx.Call("fillText", "(empty slot)", cx, y)
	}

	// Actions
	setFill(ctx, ColorMsgNew)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", "[E] Equip     [Esc] Leave", cx, by+boxH-18)
}

func (g *Game) renderShopPanel(ctx js.Value) {
	if g.Merchant == nil {
		return
	}

	cx := float64(CanvasW) / 2
	cy := float64(MapH*TileH) / 2

	// Dim the map
	ctx.Set("fillStyle", "rgba(10, 10, 20, 0.85)")
	ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))

	// Panel box — sized dynamically for number of items
	const itemH = 48
	boxW := float64(440)
	boxH := float64(44 + len(g.Merchant.Stock)*itemH + 22)
	bx := cx - boxW/2
	by := cy - boxH/2

	setFill(ctx, "#10101a")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", ColorMerchant)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textBaseline", "top")

	// Header
	setFill(ctx, ColorMerchant)
	ctx.Set("font", UIBold)
	ctx.Set("textAlign", "left")
	ctx.Call("fillText", "MERCHANT", bx+14, by+14)

	// Player gold (top right)
	setFill(ctx, ColorGold)
	ctx.Set("textAlign", "right")
	ctx.Call("fillText", fmt.Sprintf("◆ %dg", g.Player.Gold), bx+boxW-14, by+14)

	// Items
	for i, item := range g.Merchant.Stock {
		iy := by + 38 + float64(i)*float64(itemH)

		canAfford := g.Player.Gold >= item.Cost
		label := fmt.Sprintf("[%d] %s", i+1, item.Name)
		costLabel := fmt.Sprintf("%dg", item.Cost)

		var textColor string
		switch {
		case item.Sold:
			textColor = ColorUIDim
			label = fmt.Sprintf("[%d] %s", i+1, item.Name+" (sold)")
		case !canAfford:
			textColor = "#5a5f6e"
		case item.Gear != nil:
			r := gearRarityName(item.Gear)
			if r != "" {
				label = fmt.Sprintf("[%d] %s  [%s]", i+1, item.Name, r)
			}
			textColor = item.Gear.Color
		default:
			textColor = ColorUI
		}

		ctx.Set("textAlign", "left")
		setFill(ctx, textColor)
		ctx.Set("font", UIFont)
		ctx.Call("fillText", label, bx+14, iy)

		// Gear description + equipped comparison (dimmed lines below name)
		if item.Gear != nil && !item.Sold {
			setFill(ctx, ColorUIDim)
			ctx.Call("fillText", item.Gear.Desc, bx+26, iy+15)
			equipped := g.Player.Equipped[item.Gear.Slot]
			var cmpText string
			if equipped != nil {
				cmpText = "Equipped: " + equipped.Name
			} else {
				cmpText = "Slot empty"
			}
			ctx.Call("fillText", cmpText, bx+26, iy+27)
		}

		if !item.Sold {
			if canAfford {
				setFill(ctx, ColorGold)
			} else {
				setFill(ctx, ColorUIDim)
			}
			ctx.Set("textAlign", "right")
			ctx.Call("fillText", costLabel, bx+boxW-14, iy)
		}
	}

	// Footer
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", "[Esc / move] Leave", cx, by+boxH-18)
}

func (g *Game) renderEventPanel(ctx js.Value) {
	if g.ActiveEvent == nil {
		return
	}

	cx := float64(CanvasW) / 2
	cy := float64(MapH*TileH) / 2

	// Dim the map
	ctx.Set("fillStyle", "rgba(10, 10, 20, 0.85)")
	ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))

	def := g.ActiveEvent.Def

	// Panel size depends on state; wrap result text first so we can size the box
	boxW := float64(460)
	var resultLines []string
	if g.ActiveEvent.Result != "" {
		ctx.Set("font", UIFont)
		resultLines = wrapText(ctx, g.ActiveEvent.Result, boxW-28)
	}
	var boxH float64
	if g.ActiveEvent.Result != "" {
		boxH = float64(20 + len(resultLines)*18 + 14 + 24) // top + lines + gap + footer
		if boxH < 90 {
			boxH = 90
		}
	} else {
		boxH = float64(72 + len(def.Choices)*28)
	}
	bx := cx - boxW/2
	by := cy - boxH/2

	setFill(ctx, "#10101a")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", ColorAccent)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textBaseline", "top")

	if g.ActiveEvent.Result != "" {
		ctx.Set("font", UIFont)
		ctx.Set("textAlign", "center")
		for i, line := range resultLines {
			setFill(ctx, ColorMsgNew)
			ctx.Call("fillText", line, cx, by+16+float64(i)*18)
		}
		setFill(ctx, ColorUIDim)
		ctx.Call("fillText", "[Esc] Continue", cx, by+boxH-18)
		return
	}

	// Pre-choice: title
	setFill(ctx, ColorAccent)
	ctx.Set("font", UIBold)
	ctx.Set("textAlign", "left")
	ctx.Call("fillText", def.Title, bx+14, by+14)

	// Body
	setFill(ctx, ColorUI)
	ctx.Set("font", UIFont)
	ctx.Call("fillText", def.Body, bx+14, by+32)

	// Choices
	for i, choice := range def.Choices {
		setFill(ctx, ColorUI)
		ctx.Set("font", UIFont)
		ctx.Call("fillText", fmt.Sprintf("[%d] %s", i+1, choice.Label), bx+14, by+56+float64(i)*26)
	}
}

// renderEndHistory draws the run history section starting at (bx, startY).
// Returns the Y coordinate after the last drawn element.
func (g *Game) renderEndHistory(ctx js.Value, bx, startY, boxW float64) float64 {
	runs := g.RunHistory
	if len(runs) > 5 {
		runs = runs[:5]
	}
	if len(runs) == 0 {
		return startY
	}

	// Separator
	setFill(ctx, ColorSeparator)
	ctx.Call("fillRect", bx+14, startY, boxW-28, 1)

	y := startY + 14
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIBold)
	ctx.Set("textAlign", "left")
	ctx.Call("fillText", "RECENT RUNS", bx+20, y)
	y += 16

	for i, r := range runs {
		var color string
		if i == 0 {
			if r.Outcome == "Victory" {
				color = ColorHPHigh
			} else {
				color = ColorHPLow
			}
		} else {
			color = ColorMsgOld
		}
		setFill(ctx, color)
		ctx.Set("font", UIFont)
		scoreStr := ""
		if r.Score > 0 {
			scoreStr = fmt.Sprintf("  %d pts", r.Score)
		}
		ctx.Call("fillText",
			fmt.Sprintf("%s — %s  F%d  %dK  %dg%s",
				r.Outcome, r.Class, r.Floor, r.Kills, r.Gold, scoreStr),
			bx+20, y)
		y += 14
	}
	return y + 4
}

func (g *Game) renderDeathPanel(ctx js.Value) {
	cx := float64(CanvasW) / 2

	// Dim the map
	ctx.Set("fillStyle", "rgba(10, 10, 20, 0.88)")
	ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))

	runs := g.RunHistory
	if len(runs) > 5 {
		runs = runs[:5]
	}
	historyH := 0.0
	if len(runs) > 0 {
		historyH = 18 + float64(len(runs))*14 + 8
	}

	boxW := float64(480)
	boxH := 274 + historyH + 36
	bx := cx - boxW/2
	by := float64(MapH*TileH)/2 - boxH/2

	setFill(ctx, "#10101a")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", ColorHPLow)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textBaseline", "top")
	ctx.Set("textAlign", "center")

	// Title
	setFill(ctx, ColorHPLow)
	ctx.Set("font", "bold 26px Inter, system-ui, sans-serif")
	ctx.Call("fillText", "YOU DIED", cx, by+14)

	// Class + floor
	className := g.ClassName
	if className == "" {
		className = "Unknown"
	}
	classColor := ColorUIDim
	for _, def := range classDefs {
		if def.Name == className {
			classColor = def.Color
			break
		}
	}
	setFill(ctx, classColor)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", fmt.Sprintf("%s · Floor %d", className, g.Floor), cx, by+50)

	// Score
	score := 0
	if len(g.RunHistory) > 0 {
		score = g.RunHistory[0].Score
	}
	setFill(ctx, ColorGold)
	ctx.Set("font", "bold 20px Inter, system-ui, sans-serif")
	ctx.Call("fillText", fmt.Sprintf("SCORE  %d", score), cx, by+68)

	// Separator
	setFill(ctx, ColorSeparator)
	ctx.Call("fillRect", bx+14, by+96, boxW-28, 1)

	// Gear (4 slots)
	ctx.Set("textAlign", "left")
	ctx.Set("font", UIFont)
	gearLabels := []string{"†", "◈", "◇", "✦"}
	for i, slot := range []GearSlot{SlotWeapon, SlotArmor, SlotTrinket, SlotClass} {
		gy := by + 108 + float64(i)*16
		gear := g.Player.Equipped[slot]
		if gear != nil {
			setFill(ctx, gear.Color)
			ctx.Call("fillText", fmt.Sprintf("%s %s", string(gear.Char), gear.Name), bx+20, gy)
		} else {
			setFill(ctx, ColorUIDim)
			ctx.Call("fillText", gearLabels[i]+" (empty)", bx+20, gy)
		}
	}

	// Separator
	setFill(ctx, ColorSeparator)
	ctx.Call("fillRect", bx+14, by+176, boxW-28, 1)

	// Stats row 1
	ctx.Set("textAlign", "center")
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Call("fillText", "TURNS", cx-110, by+190)
	ctx.Call("fillText", "GOLD", cx, by+190)
	ctx.Call("fillText", "KILLS", cx+110, by+190)

	ctx.Set("font", "bold 16px Inter, system-ui, sans-serif")
	setFill(ctx, ColorMsgNew)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Turns), cx-110, by+206)
	setFill(ctx, ColorGold)
	ctx.Call("fillText", fmt.Sprintf("%dg", g.Player.Gold), cx, by+206)
	setFill(ctx, ColorHPLow)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Kills), cx+110, by+206)

	// Stats row 2
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Call("fillText", "DMG OUT", cx-172, by+228)
	ctx.Call("fillText", "DMG IN", cx-57, by+228)
	ctx.Call("fillText", "POTIONS", cx+57, by+228)
	ctx.Call("fillText", "STEPS", cx+172, by+228)

	ctx.Set("font", "bold 14px Inter, system-ui, sans-serif")
	setFill(ctx, ColorHPMid)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Player.DamageDealt), cx-172, by+244)
	setFill(ctx, ColorHPLow)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Player.DamageTaken), cx-57, by+244)
	setFill(ctx, ColorPotion)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Player.PotionsUsed), cx+57, by+244)
	setFill(ctx, ColorUIDim)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Player.Steps), cx+172, by+244)

	// History
	g.renderEndHistory(ctx, bx, by+264, boxW)

	// Footer
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", "[R] Play again", cx, by+boxH-18)
}

func (g *Game) renderVictoryPanel(ctx js.Value) {
	cx := float64(CanvasW) / 2

	// Dim the map
	ctx.Set("fillStyle", "rgba(10, 10, 20, 0.88)")
	ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))

	runs := g.RunHistory
	if len(runs) > 5 {
		runs = runs[:5]
	}
	historyH := 0.0
	if len(runs) > 0 {
		historyH = 18 + float64(len(runs))*14 + 8
	}

	boxW := float64(480)
	boxH := 294 + historyH + 36
	bx := cx - boxW/2
	by := float64(MapH*TileH)/2 - boxH/2

	setFill(ctx, "#10101a")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", ColorAccent)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textBaseline", "top")
	ctx.Set("textAlign", "center")

	// Title
	setFill(ctx, ColorAccent)
	ctx.Set("font", "bold 26px Inter, system-ui, sans-serif")
	ctx.Call("fillText", "VICTORY", cx, by+14)

	// Class
	className := g.ClassName
	if className == "" {
		className = "Unknown"
	}
	classColor := ColorUIDim
	for _, def := range classDefs {
		if def.Name == className {
			classColor = def.Color
			break
		}
	}
	setFill(ctx, classColor)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", fmt.Sprintf("%s · You escaped the dungeon!", className), cx, by+50)

	// Score
	score := 0
	if len(g.RunHistory) > 0 {
		score = g.RunHistory[0].Score
	}
	setFill(ctx, ColorGold)
	ctx.Set("font", "bold 20px Inter, system-ui, sans-serif")
	ctx.Call("fillText", fmt.Sprintf("SCORE  %d", score), cx, by+68)

	// Separator
	setFill(ctx, ColorSeparator)
	ctx.Call("fillRect", bx+14, by+96, boxW-28, 1)

	// Gear (4 slots)
	ctx.Set("textAlign", "left")
	ctx.Set("font", UIFont)
	gearLabels := []string{"†", "◈", "◇", "✦"}
	for i, slot := range []GearSlot{SlotWeapon, SlotArmor, SlotTrinket, SlotClass} {
		gy := by + 108 + float64(i)*16
		gear := g.Player.Equipped[slot]
		if gear != nil {
			setFill(ctx, gear.Color)
			ctx.Call("fillText", fmt.Sprintf("%s %s", string(gear.Char), gear.Name), bx+20, gy)
		} else {
			setFill(ctx, ColorUIDim)
			ctx.Call("fillText", gearLabels[i]+" (empty)", bx+20, gy)
		}
	}

	// Separator
	setFill(ctx, ColorSeparator)
	ctx.Call("fillRect", bx+14, by+176, boxW-28, 1)

	// Stats
	ctx.Set("textAlign", "center")
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Call("fillText", "TURNS", cx-110, by+190)
	ctx.Call("fillText", "GOLD", cx, by+190)
	ctx.Call("fillText", "KILLS", cx+110, by+190)

	ctx.Set("font", "bold 18px Inter, system-ui, sans-serif")
	setFill(ctx, ColorMsgNew)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Turns), cx-110, by+208)
	setFill(ctx, ColorGold)
	ctx.Call("fillText", fmt.Sprintf("%dg", g.Player.Gold), cx, by+208)
	setFill(ctx, ColorHPHigh)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Kills), cx+110, by+208)

	// Stats row 2
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Call("fillText", "DMG OUT", cx-172, by+230)
	ctx.Call("fillText", "DMG IN", cx-57, by+230)
	ctx.Call("fillText", "POTIONS", cx+57, by+230)
	ctx.Call("fillText", "STEPS", cx+172, by+230)

	ctx.Set("font", "bold 14px Inter, system-ui, sans-serif")
	setFill(ctx, ColorHPMid)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Player.DamageDealt), cx-172, by+246)
	setFill(ctx, ColorHPLow)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Player.DamageTaken), cx-57, by+246)
	setFill(ctx, ColorPotion)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Player.PotionsUsed), cx+57, by+246)
	setFill(ctx, ColorUIDim)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Player.Steps), cx+172, by+246)

	// History
	g.renderEndHistory(ctx, bx, by+268, boxW)

	// Footer
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", "[R] Play again", cx, by+boxH-18)
}

func (g *Game) renderTitleScreen(ctx js.Value) {
	setFill(ctx, ColorBg)
	ctx.Call("fillRect", 0, 0, CanvasW, CanvasH)

	cx := float64(CanvasW) / 2
	cy := float64(CanvasH) / 2

	// Draw a decorative grid of dim dungeon characters as a background
	ctx.Set("font", GameFont)
	ctx.Set("textAlign", "center")
	ctx.Set("textBaseline", "middle")
	glyphs := []rune{'█', '·', '▓', '█', '·', '█', '·', '▓'}
	for row := 0; row < MapH; row++ {
		for col := 0; col < MapW; col++ {
			g := glyphs[(row*7+col*3)%len(glyphs)]
			if g == '·' {
				ctx.Set("fillStyle", "rgba(30,34,54,0.5)")
			} else {
				ctx.Set("fillStyle", "rgba(22,25,40,0.5)")
			}
			ctx.Call("fillText", string(g),
				float64(col*TileW)+float64(TileW)/2,
				float64(row*TileH)+float64(TileH)/2)
		}
	}

	// Vignette — darken edges so title pops
	grad := ctx.Call("createRadialGradient", cx, cy, 100, cx, cy, float64(CanvasW)*0.72)
	grad.Call("addColorStop", 0, "rgba(13,13,20,0)")
	grad.Call("addColorStop", 1, "rgba(13,13,20,0.92)")
	ctx.Set("fillStyle", grad)
	ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))

	ctx.Set("textAlign", "center")
	ctx.Set("textBaseline", "middle")

	// Glyph icon
	setFill(ctx, ColorAccent)
	ctx.Set("font", "bold 42px 'Courier New', monospace")
	ctx.Call("fillText", "▼", cx, cy-72)

	// Title
	setFill(ctx, ColorMsgNew)
	ctx.Set("font", "bold 52px Inter, system-ui, sans-serif")
	ctx.Call("fillText", "DUNGEON", cx, cy-20)

	// Tagline
	setFill(ctx, ColorUIDim)
	ctx.Set("font", "16px Inter, system-ui, sans-serif")
	ctx.Call("fillText", "Three floors. One chance.", cx, cy+26)

	// Last run info
	history := loadRunHistory()
	if len(history) > 0 {
		r := history[0]
		var runColor string
		if r.Outcome == "Victory" {
			runColor = ColorHPHigh
		} else {
			runColor = ColorUIDim
		}
		setFill(ctx, runColor)
		ctx.Set("font", UIFont)
		ctx.Call("fillText",
			fmt.Sprintf("Last run: %s — %s  Floor %d  %dg  %d kills",
				r.Class, r.Outcome, r.Floor, r.Gold, r.Kills),
			cx, cy+56)
	}

	// Prompt — pulse using sine of current time
	pulse := (time.Now().UnixMilli()/600)%2 == 0
	if pulse {
		setFill(ctx, ColorAccent)
	} else {
		setFill(ctx, ColorUIDim)
	}
	ctx.Set("font", UIBold)
	ctx.Call("fillText", "Press any key to begin", cx, cy+90)

	// UI strip background
	setFill(ctx, "#080810")
	ctx.Call("fillRect", 0, float64(MapH*TileH), CanvasW, UIHeight)
}

func (g *Game) renderDifficultySelect(ctx js.Value) {
	setFill(ctx, ColorBg)
	ctx.Call("fillRect", 0, 0, CanvasW, CanvasH)

	cx := float64(CanvasW) / 2
	cy := float64(CanvasH) / 2

	boxW := float64(500)
	boxH := float64(220)
	bx := cx - boxW/2
	by := cy - boxH/2

	setFill(ctx, "#10101a")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", ColorAccent)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textBaseline", "top")
	setFill(ctx, ColorAccent)
	ctx.Set("font", "bold 20px Inter, system-ui, sans-serif")
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", "SELECT DIFFICULTY", cx, by+14)

	history := loadRunHistory()
	victories := countVictories(history)
	hardVictories := countHardVictories(history)

	type diffOpt struct {
		key     string
		label   string
		locked  bool
		lockMsg string
		color   string
	}
	opts := []diffOpt{
		{"1", "Normal", false, "", ColorHPHigh},
		{"2", "Hard  (enemy HP ×1.25, no floor-2 merchant)", victories < 1, "(win on Normal to unlock)", ColorHPMid},
		{"3", "Nightmare  (HP ×1.40, poison persists)", hardVictories < 1, "(win on Hard to unlock)", ColorHPLow},
		{"4", fmt.Sprintf("Daily Challenge  (%s)", time.Now().Format("Jan 02")), false, "", ColorAccent},
	}

	for i, opt := range opts {
		oy := by + 48 + float64(i)*38
		ctx.Set("textAlign", "left")
		if opt.locked {
			setFill(ctx, ColorUIDim)
			ctx.Set("font", UIFont)
			ctx.Call("fillText", fmt.Sprintf("[%s] %s  %s", opt.key, opt.label, opt.lockMsg), bx+20, oy)
		} else {
			setFill(ctx, opt.color)
			ctx.Set("font", UIBold)
			ctx.Call("fillText", fmt.Sprintf("[%s] %s", opt.key, opt.label), bx+20, oy)
		}
	}
}

func (g *Game) renderClassSelect(ctx js.Value) {
	// Full dark background
	setFill(ctx, ColorBg)
	ctx.Call("fillRect", 0, 0, CanvasW, CanvasH)

	cx := float64(CanvasW) / 2
	cy := float64(CanvasH) / 2

	// Base classes (4) + variant classes (4, if any unlocked)
	baseDefs := classDefs[:4]
	variantDefs := classDefs[4:]

	const rowH = 82
	rowCount := len(baseDefs)
	for i := range variantDefs {
		req := variantUnlockReq(4 + i)
		if req == "" || g.ClassWins[req] >= 3 {
			rowCount++
		}
	}
	boxW := float64(700)
	boxH := float64(56 + rowCount*rowH + 24)
	bx := cx - boxW/2
	by := cy - boxH/2

	setFill(ctx, "#10101a")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", ColorAccent)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textBaseline", "top")

	// Title
	setFill(ctx, ColorAccent)
	ctx.Set("font", "bold 20px Inter, system-ui, sans-serif")
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", "CHOOSE YOUR CLASS", cx, by+14)

	row := 0
	for i, def := range baseDefs {
		ry := by + 52 + float64(row)*rowH
		renderClassRow(ctx, def, i+1, bx, ry, boxW, true, "")
		if i < len(baseDefs)-1 || rowCount > len(baseDefs) {
			setFill(ctx, ColorSeparator)
			ctx.Call("fillRect", bx+1, ry+float64(rowH)-2, boxW-2, 1)
		}
		row++
	}

	for i, def := range variantDefs {
		req := variantUnlockReq(4 + i)
		wins := g.ClassWins[req]
		unlocked := req == "" || wins >= 3
		if !unlocked {
			continue
		}
		ry := by + 52 + float64(row)*rowH
		renderClassRow(ctx, def, 5+i, bx, ry, boxW, true, "")
		if row < rowCount-1 {
			setFill(ctx, ColorSeparator)
			ctx.Call("fillRect", bx+1, ry+float64(rowH)-2, boxW-2, 1)
		}
		row++
	}

	// Footer
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", "[1–8] Choose your class", cx, by+boxH-18)
}

func renderClassRow(ctx js.Value, def *ClassDef, keyNum int, bx, ry, boxW float64, unlocked bool, lockMsg string) {
	if !unlocked {
		setFill(ctx, ColorUIDim)
		ctx.Set("font", "bold 14px Inter, system-ui, sans-serif")
		ctx.Set("textAlign", "left")
		ctx.Call("fillText", fmt.Sprintf("[%d] %s  %s", keyNum, def.Name, lockMsg), bx+20, ry+28)
		return
	}
	setFill(ctx, def.Color)
	ctx.Set("font", "bold 14px Inter, system-ui, sans-serif")
	ctx.Set("textAlign", "left")
	ctx.Call("fillText", fmt.Sprintf("[%d] %s", keyNum, def.Name), bx+20, ry)

	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Set("textAlign", "right")
	ctx.Call("fillText",
		fmt.Sprintf("HP %d   ATK %d   DEF %d", def.BaseHP, def.BaseAtk, def.BaseDef),
		bx+boxW-20, ry+2)

	item := def.StartItem
	ctx.Set("textAlign", "left")
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Call("fillText", "✦ ", bx+30, ry+22)
	prefW := ctx.Call("measureText", "✦ ").Get("width").Float()
	icon := string(item.Char) + " "
	setFill(ctx, item.Color)
	ctx.Call("fillText", icon, bx+30+prefW, ry+22)
	iconW := ctx.Call("measureText", icon).Get("width").Float()
	setFill(ctx, ColorUI)
	ctx.Call("fillText", item.Name+"  "+item.Desc, bx+30+prefW+iconW, ry+22)

	setFill(ctx, ColorUIDim)
	ctx.Call("fillText", def.Flavor, bx+30, ry+42)

	if def.BuildHint != "" {
		setFill(ctx, ColorAccent)
		ctx.Set("font", "10px Inter, system-ui, sans-serif")
		ctx.Call("fillText", def.BuildHint, bx+30, ry+58)
	}
}

func (g *Game) renderHintOverlay(ctx js.Value) {
	cx := float64(CanvasW) / 2
	boxW := 580.0
	boxH := 260.0
	bx := cx - boxW/2
	by := float64(MapH*TileH)/2 - boxH/2

	// Background
	ctx.Set("fillStyle", "rgba(8, 8, 16, 0.94)")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", ColorAccent)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textBaseline", "top")
	ctx.Set("textAlign", "center")
	setFill(ctx, ColorAccent)
	ctx.Set("font", "bold 14px Inter, system-ui, sans-serif")
	ctx.Call("fillText", "CONTROLS", cx, by+12)

	setFill(ctx, ColorSeparator)
	ctx.Call("fillRect", bx+14, by+32, boxW-28, 1)

	type hint struct{ key, desc string }
	hints := []hint{
		{"WASD / Arrows", "Move"},
		{"Bump enemy", "Attack"},
		{"1 / 2 / 3", "Use potion in slot"},
		{"U", "Use potion (slot 1)"},
		{"E", "Equip gear from chest"},
		{".", "Wait one turn"},
		{"1 / 2 / 3", "Event choice"},
		{"R", "Restart after death"},
		{"Tab", "Toggle message log"},
	}

	lx := bx + 26
	rx := bx + 310
	ctx.Set("textAlign", "left")
	ctx.Set("font", UIFont)
	for i, h := range hints {
		col := lx
		if i >= 5 {
			col = rx
		}
		row := i
		if i >= 5 {
			row = i - 5
		}
		y := by + 46 + float64(row)*20
		setFill(ctx, ColorAccent)
		ctx.Call("fillText", h.key, col, y)
		setFill(ctx, ColorUI)
		ctx.Call("fillText", "  "+h.desc, col+90, y)
	}

	setFill(ctx, ColorUIDim)
	ctx.Set("textAlign", "center")
	ctx.Set("font", "11px Inter, system-ui, sans-serif")
	ctx.Call("fillText", "[ Press any key to dismiss ]", cx, by+boxH-16)
}

func (g *Game) renderLogPanel(ctx js.Value) {
	const (
		boxW     = 700.0
		maxLines = 22
		lineH    = 18.0
	)
	boxH := 32.0 + maxLines*lineH + 28.0 // header + lines + footer
	cx := float64(CanvasW) / 2
	cy := float64(MapH*TileH) / 2
	bx := cx - boxW/2
	by := cy - boxH/2

	// Dim background
	ctx.Set("fillStyle", "rgba(10, 10, 20, 0.92)")
	ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))

	// Panel
	setFill(ctx, "#10101a")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", ColorAccent)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textBaseline", "top")

	// Header
	setFill(ctx, ColorAccent)
	ctx.Set("font", UIBold)
	ctx.Set("textAlign", "left")
	ctx.Call("fillText", "MESSAGE LOG", bx+14, by+10)

	// Separator
	setFill(ctx, ColorSeparator)
	ctx.Call("fillRect", bx+1, by+28, boxW-2, 1)

	// Messages — show the most recent maxLines, oldest at top
	msgs := g.Messages
	if len(msgs) > maxLines {
		msgs = msgs[len(msgs)-maxLines:]
	}
	ctx.Set("font", UIFont)
	ctx.Set("textAlign", "left")
	for i, msg := range msgs {
		age := len(msgs) - 1 - i // 0 = newest
		var color string
		switch {
		case age == 0:
			color = ColorMsgNew
		case age <= 4:
			color = ColorUI
		default:
			color = ColorMsgOld
		}
		setFill(ctx, color)
		ctx.Call("fillText", msg, bx+14, by+34+float64(i)*lineH)
	}

	// Footer
	setFill(ctx, ColorUIDim)
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", "[Tab] Close", cx, by+boxH-18)
}

func (g *Game) renderDropPanel(ctx js.Value) {
	n := len(g.Player.PotionTypes)
	if n == 0 {
		g.ShowDropMode = false
		return
	}

	cx := float64(CanvasW) / 2
	cy := float64(MapH*TileH) / 2
	boxW := 340.0
	boxH := float64(42 + n*26 + 26)
	bx := cx - boxW/2
	by := cy - boxH/2

	ctx.Set("fillStyle", "rgba(10, 10, 20, 0.82)")
	ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))

	setFill(ctx, "#10101a")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", ColorHPLow)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textBaseline", "top")
	setFill(ctx, ColorHPLow)
	ctx.Set("font", UIBold)
	ctx.Set("textAlign", "left")
	ctx.Call("fillText", "DROP POTION", bx+14, by+14)

	type potInfo struct{ letter, color, desc string }
	info := [4]potInfo{
		{"H", ColorPotion, "Healing Potion (+12 HP)"},
		{"A", ColorFreeze, "Antidote (clear effects, +4 HP)"},
		{"M", ColorBurn, "Might Draught (+5 ATK, 3 turns)"},
		{"G", "#9AE6B4", "Greater Potion (+25 HP)"},
	}

	for i, pt := range g.Player.PotionTypes {
		idx := int(pt)
		if idx < 0 || idx > 3 {
			idx = 0
		}
		p := info[idx]
		iy := by + 38 + float64(i)*26
		setFill(ctx, ColorUIDim)
		ctx.Set("font", UIBold)
		ctx.Call("fillText", fmt.Sprintf("[%d]", i+1), bx+14, iy)
		setFill(ctx, p.color)
		ctx.Call("fillText", p.letter, bx+42, iy)
		setFill(ctx, ColorUI)
		ctx.Set("font", UIFont)
		ctx.Call("fillText", p.desc, bx+58, iy)
	}

	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", "any other key cancels", cx, by+boxH-16)
}

func setFill(ctx js.Value, color string) {
	ctx.Set("fillStyle", color)
}

// wrapText splits text into lines that fit within maxW pixels.
// ctx font must be set before calling so measureText uses the right metrics.
func wrapText(ctx js.Value, text string, maxW float64) []string {
	words := strings.Fields(text)
	var lines []string
	cur := ""
	for _, word := range words {
		test := cur
		if test != "" {
			test += " "
		}
		test += word
		if ctx.Call("measureText", test).Get("width").Float() > maxW && cur != "" {
			lines = append(lines, cur)
			cur = word
		} else {
			cur = test
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
