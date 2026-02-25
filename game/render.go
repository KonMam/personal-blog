//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"
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

	GameFont = "bold 15px 'Courier New', 'Lucida Console', monospace"
	UIFont   = "12px Inter, system-ui, sans-serif"
	UIBold   = "bold 12px Inter, system-ui, sans-serif"
)

func (g *Game) Render(ctx js.Value) {
	if g.Phase == PhaseClassSelect {
		g.renderClassSelect(ctx)
		return
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

	// Enemies
	for _, e := range g.Enemies {
		if e.Alive && g.Tiles[e.Y][e.X].Visible {
			g.drawChar(ctx, e.Char, e.X, e.Y, e.Color)
		}
	}

	// Player
	g.drawChar(ctx, g.Player.Char, g.Player.X, g.Player.Y, ColorPlayer)

	// UI strip
	g.renderUI(ctx)

	// Overlays
	switch g.Phase {
	case PhaseGameOver:
		g.renderOverlay(ctx, "YOU DIED", "Press R to restart", ColorHPLow)
	case PhaseVictory:
		g.renderVictoryPanel(ctx)
	case PhaseChest:
		g.renderChestPanel(ctx)
	case PhaseShop:
		g.renderShopPanel(ctx)
	case PhaseEvent:
		g.renderEventPanel(ctx)
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

	switch tile.Type {
	case TileWall:
		ch = '█'
		if tile.Visible {
			bg = ColorBg
			fg = ColorWallVisible
		} else {
			bg = ColorBg
			fg = ColorWallExplored
		}
	case TileFloor:
		ch = '·'
		if tile.Visible {
			bg = ColorFloorVisible
			fg = ColorDotVisible
		} else {
			bg = ColorFloorExplored
			fg = ColorDotExplored
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
	ctx.Call("fillText", fmt.Sprintf("◆ %dg", g.Player.Gold), 580, top)

	// Potions
	setFill(ctx, ColorPotion)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", fmt.Sprintf("♥ %d", g.Player.Potions), 700, top)

	// Poison indicator
	if g.Player.Poison > 0 {
		setFill(ctx, ColorPoisonUI)
		ctx.Set("font", UIBold)
		ctx.Call("fillText", fmt.Sprintf("☠ %d", g.Player.Poison), 780, top)
	}

	// --- Line 2: Gear slots ---
	gearY := top + 22

	ctx.Set("font", UIFont)
	g.renderGearSlot(ctx, 12, gearY, SlotWeapon)
	g.renderGearSlot(ctx, 330, gearY, SlotArmor)
	g.renderGearSlot(ctx, 650, gearY, SlotTrinket)

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

	// Name + desc in UI color, positioned after icon
	setFill(ctx, ColorUI)
	ctx.Call("fillText", fmt.Sprintf("%s  %s", gear.Name, gear.Desc), x+iconW, y)
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

func (g *Game) renderChestPanel(ctx js.Value) {
	if g.PendingGear == nil {
		return
	}

	cx := float64(CanvasW) / 2
	cy := float64(MapH*TileH) / 2

	// Dim the map
	ctx.Set("fillStyle", "rgba(10, 10, 20, 0.85)")
	ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))

	// Panel box
	boxW := float64(380)
	boxH := float64(130)
	bx := cx - boxW/2
	by := cy - boxH/2

	setFill(ctx, "#10101a")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", ColorChest)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textAlign", "center")
	ctx.Set("textBaseline", "top")

	// Header
	setFill(ctx, ColorChest)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", "GEAR FOUND", cx, by+14)

	// Gear name
	setFill(ctx, g.PendingGear.Color)
	ctx.Set("font", "bold 16px Inter, system-ui, sans-serif")
	ctx.Call("fillText", string(g.PendingGear.Char)+" "+g.PendingGear.Name, cx, by+36)

	// Gear description
	setFill(ctx, ColorUI)
	ctx.Set("font", UIFont)
	ctx.Call("fillText", g.PendingGear.Desc, cx, by+58)

	// Current equipped
	slot := g.PendingGear.Slot
	current := g.Player.Equipped[slot]
	currentText := "(empty slot)"
	if current != nil {
		currentText = "Replaces: " + current.Name
	}
	setFill(ctx, ColorUIDim)
	ctx.Call("fillText", currentText, cx, by+76)

	// Actions
	setFill(ctx, ColorMsgNew)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", "[E] Equip     [any key] Leave", cx, by+100)
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
	const itemH = 28
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
		default:
			textColor = ColorUI
		}

		ctx.Set("textAlign", "left")
		setFill(ctx, textColor)
		ctx.Set("font", UIFont)
		ctx.Call("fillText", label, bx+14, iy)

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

	// Panel size depends on state
	boxW := float64(460)
	var boxH float64
	if g.ActiveEvent.Result != "" {
		boxH = 110
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
		// Post-choice: result
		setFill(ctx, ColorMsgNew)
		ctx.Set("font", UIFont)
		ctx.Set("textAlign", "center")
		ctx.Call("fillText", g.ActiveEvent.Result, cx, by+30)
		setFill(ctx, ColorUIDim)
		ctx.Call("fillText", "[any key] Continue", cx, by+56)
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

func (g *Game) renderVictoryPanel(ctx js.Value) {
	cx := float64(CanvasW) / 2
	cy := float64(MapH*TileH) / 2

	// Dim the map
	ctx.Set("fillStyle", "rgba(10, 10, 20, 0.88)")
	ctx.Call("fillRect", 0, 0, CanvasW, float64(MapH*TileH))

	// Panel box
	boxW := float64(380)
	boxH := float64(170)
	bx := cx - boxW/2
	by := cy - boxH/2

	setFill(ctx, "#10101a")
	ctx.Call("fillRect", bx, by, boxW, boxH)
	ctx.Set("strokeStyle", ColorAccent)
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", bx+0.5, by+0.5, boxW-1, boxH-1)

	ctx.Set("textAlign", "center")
	ctx.Set("textBaseline", "top")

	// Title
	setFill(ctx, ColorAccent)
	ctx.Set("font", "bold 26px Inter, system-ui, sans-serif")
	ctx.Call("fillText", "VICTORY", cx, by+14)

	// Subtitle
	setFill(ctx, ColorUI)
	ctx.Set("font", UIFont)
	ctx.Call("fillText", "You escaped the dungeon!", cx, by+50)

	// Stat labels
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Call("fillText", "TURNS", cx-110, by+76)
	ctx.Call("fillText", "GOLD", cx, by+76)
	ctx.Call("fillText", "KILLS", cx+110, by+76)

	// Stat values
	ctx.Set("font", "bold 18px Inter, system-ui, sans-serif")
	setFill(ctx, ColorMsgNew)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Turns), cx-110, by+94)
	setFill(ctx, ColorGold)
	ctx.Call("fillText", fmt.Sprintf("%dg", g.Player.Gold), cx, by+94)
	setFill(ctx, ColorHPLow)
	ctx.Call("fillText", fmt.Sprintf("%d", g.Kills), cx+110, by+94)

	// Footer
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Call("fillText", "[R] Play again", cx, by+boxH-22)
}

func (g *Game) renderClassSelect(ctx js.Value) {
	// Full dark background
	setFill(ctx, ColorBg)
	ctx.Call("fillRect", 0, 0, CanvasW, CanvasH)

	cx := float64(CanvasW) / 2
	cy := float64(CanvasH) / 2

	// Panel — 4 rows × 76px + header + footer
	const rowH = 76
	boxW := float64(700)
	boxH := float64(56 + len(classDefs)*rowH + 24)
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

	// Class rows
	for i, def := range classDefs {
		ry := by + 52 + float64(i)*rowH

		// [N] ClassName
		setFill(ctx, def.Color)
		ctx.Set("font", "bold 14px Inter, system-ui, sans-serif")
		ctx.Set("textAlign", "left")
		ctx.Call("fillText", fmt.Sprintf("[%d] %s", i+1, def.Name), bx+20, ry)

		// Stats right-aligned on the same line
		setFill(ctx, ColorUIDim)
		ctx.Set("font", UIFont)
		ctx.Set("textAlign", "right")
		ctx.Call("fillText",
			fmt.Sprintf("HP %d   ATK %d   DEF %d", def.BaseHP, def.BaseAtk, def.BaseDef),
			bx+boxW-20, ry+2)

		// Starting item
		item := def.StartItem
		ctx.Set("textAlign", "left")
		icon := string(item.Char) + " "
		setFill(ctx, item.Color)
		ctx.Set("font", UIFont)
		ctx.Call("fillText", icon, bx+30, ry+22)
		iconW := ctx.Call("measureText", icon).Get("width").Float()
		setFill(ctx, ColorUI)
		ctx.Call("fillText", item.Name+"  "+item.Desc, bx+30+iconW, ry+22)

		// Flavor text
		setFill(ctx, ColorUIDim)
		ctx.Call("fillText", def.Flavor, bx+30, ry+42)

		// Row separator (not after last)
		if i < len(classDefs)-1 {
			setFill(ctx, ColorSeparator)
			ctx.Call("fillRect", bx+1, ry+float64(rowH)-2, boxW-2, 1)
		}
	}

	// Footer
	setFill(ctx, ColorUIDim)
	ctx.Set("font", UIFont)
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", "[1–4] Choose your class", cx, by+boxH-18)
}

func setFill(ctx js.Value, color string) {
	ctx.Set("fillStyle", color)
}
