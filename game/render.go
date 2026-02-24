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

	GameFont = "bold 15px 'Courier New', 'Lucida Console', monospace"
	UIFont   = "12px Inter, system-ui, sans-serif"
	UIBold   = "bold 12px Inter, system-ui, sans-serif"
)

func (g *Game) Render(ctx js.Value) {
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

	// --- Line 1: FLOOR | HP bar | Gold | Potions ---

	// Floor label
	setFill(ctx, ColorAccent)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", fmt.Sprintf("FLOOR %d/%d", g.Floor, MaxFloors), 12, top)

	// HP bar
	g.renderHPBar(ctx, top)

	// Gold
	goldX := float64(600)
	setFill(ctx, ColorGold)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", fmt.Sprintf("◆ %dg", g.Player.Gold), goldX, top)

	// Potions
	potX := float64(700)
	setFill(ctx, ColorPotion)
	ctx.Set("font", UIBold)
	ctx.Call("fillText", fmt.Sprintf("♥ %d", g.Player.Potions), potX, top)

	// --- Line 2: Gear slots ---
	gearY := top + 22

	// Weapon slot
	ctx.Set("font", UIFont)
	g.renderGearSlot(ctx, 12, gearY, SlotWeapon)

	// Armor slot
	g.renderGearSlot(ctx, float64(CanvasW)/2, gearY, SlotArmor)

	// --- Lines 3-4: Messages ---
	g.renderMessages(ctx, top+46)
}

func (g *Game) renderGearSlot(ctx js.Value, x, y float64, slot GearSlot) {
	gear := g.Player.Equipped[slot]
	ctx.Set("textAlign", "left")
	ctx.Set("font", UIFont)

	if gear == nil {
		setFill(ctx, ColorUIDim)
		if slot == SlotWeapon {
			ctx.Call("fillText", "† (no weapon)", x, y)
		} else {
			ctx.Call("fillText", "◈ (no armor)", x, y)
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

	// Panel box
	boxW := float64(420)
	boxH := float64(180)
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
		iy := by + 38 + float64(i)*32

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

func setFill(ctx js.Value, color string) {
	ctx.Set("fillStyle", color)
}
