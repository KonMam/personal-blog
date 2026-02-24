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
	UIHeight = 88
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

	// Items
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
		g.renderOverlay(ctx, "VICTORY", "You escaped the dungeon!  Press R to play again.", ColorAccent)
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

	// Floor label
	setFill(ctx, ColorAccent)
	ctx.Set("font", UIBold)
	ctx.Set("textAlign", "left")
	ctx.Set("textBaseline", "top")
	ctx.Call("fillText", fmt.Sprintf("FLOOR  %d / %d", g.Floor, MaxFloors), 12, top)

	// HP bar
	g.renderHPBar(ctx, top)

	// Messages
	g.renderMessages(ctx, top+22)
}

func (g *Game) renderHPBar(ctx js.Value, y float64) {
	labelX := float64(CanvasW) * 0.32
	barX := labelX + 28.0
	barW := float64(CanvasW) * 0.28
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
	ctx.Set("textAlign", "left")
	ctx.Call("fillText",
		fmt.Sprintf("%d / %d", g.Player.HP, g.Player.MaxHP),
		barX+barW+10, y+1)
}

func (g *Game) renderMessages(ctx js.Value, y float64) {
	msgs := g.Messages
	if len(msgs) > 3 {
		msgs = msgs[len(msgs)-3:]
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

func setFill(ctx js.Value, color string) {
	ctx.Set("fillStyle", color)
}
