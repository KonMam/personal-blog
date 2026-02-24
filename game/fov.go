//go:build js && wasm

package main

// ComputeFOV marks tiles as visible/explored using raycasting from (px, py).
func ComputeFOV(tiles [][]Tile, px, py, radius, width, height int) {
	// Clear visibility
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			tiles[y][x].Visible = false
		}
	}

	// Origin is always visible
	tiles[py][px].Visible = true
	tiles[py][px].Explored = true

	// Check every tile within the radius circle
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy > radius*radius {
				continue
			}
			tx, ty := px+dx, py+dy
			if tx < 0 || ty < 0 || tx >= width || ty >= height {
				continue
			}
			if hasLoS(tiles, px, py, tx, ty, width, height) {
				tiles[ty][tx].Visible = true
				tiles[ty][tx].Explored = true
			}
		}
	}
}

// hasLoS returns true if (x0,y0) has line-of-sight to (x1,y1).
// Walls are visible when they ARE the destination but block sight beyond them.
func hasLoS(tiles [][]Tile, x0, y0, x1, y1, width, height int) bool {
	dx := iAbs(x1 - x0)
	dy := -iAbs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	x, y := x0, y0

	for {
		// Reached destination -- visible regardless of tile type
		if x == x1 && y == y1 {
			return true
		}
		// Bounds check
		if x < 0 || y < 0 || x >= width || y >= height {
			return false
		}
		// An intermediate wall (not origin) blocks sight
		if (x != x0 || y != y0) && tiles[y][x].Type == TileWall {
			return false
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
	}
}

func iAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func iSign(x int) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}
