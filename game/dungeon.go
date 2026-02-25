//go:build js && wasm

package main


type TileType int

const (
	TileWall TileType = iota
	TileFloor
	TileStairs
)

type Tile struct {
	Type     TileType
	Visible  bool
	Explored bool
	Variant  byte // 0-3, for floor/wall visual variety
}

type Room struct {
	X, Y, W, H int
}

func (r Room) Center() (int, int) {
	return r.X + r.W/2, r.Y + r.H/2
}

// GenerateDungeon creates a room-and-corridor dungeon.
// Returns the tile grid and the list of rooms in placement order.
//
// The map is divided into a 3×3 sector grid (one room per sector) so rooms
// are spread evenly across the whole canvas. Sector order is shuffled each
// run, so player spawn and stairs land in different corners every time.
func GenerateDungeon(width, height int) ([][]Tile, []Room) {
	tiles := make([][]Tile, height)
	for i := range tiles {
		tiles[i] = make([]Tile, width)
	}

	const (
		gridCols = 3
		gridRows = 3
		padding  = 1 // wall gap around each sector's inner area
	)

	type sector struct{ col, row int }
	sectors := make([]sector, 0, gridCols*gridRows)
	for r := 0; r < gridRows; r++ {
		for c := 0; c < gridCols; c++ {
			sectors = append(sectors, sector{c, r})
		}
	}
	rng.Shuffle(len(sectors), func(i, j int) { sectors[i], sectors[j] = sectors[j], sectors[i] })
	// Ensure spawn (sectors[0]) and stairs (sectors[last]) are well-separated.
	// Re-shuffle until their Manhattan distance in the sector grid is >= 3.
	for {
		first, last := sectors[0], sectors[len(sectors)-1]
		if iAbs(first.col-last.col)+iAbs(first.row-last.row) >= 3 {
			break
		}
		rng.Shuffle(len(sectors), func(i, j int) { sectors[i], sectors[j] = sectors[j], sectors[i] })
	}

	sectorW := width / gridCols  // 20
	sectorH := height / gridRows // 7

	rooms := []Room{}

	for _, sec := range sectors {
		sx := sec.col * sectorW
		sy := sec.row * sectorH
		sw := sectorW
		sh := sectorH
		// Last column/row absorbs any remainder
		if sec.col == gridCols-1 {
			sw = width - sx
		}
		if sec.row == gridRows-1 {
			sh = height - sy
		}

		innerW := sw - 2*padding
		innerH := sh - 2*padding
		if innerW < 4 || innerH < 4 {
			continue
		}

		// Random room size within the sector interior
		rw := 4 + rng.Intn(min(7, innerW-3))
		rh := 4 + rng.Intn(min(4, innerH-3))

		// Random position within the remaining slack
		xSlack := innerW - rw
		ySlack := innerH - rh
		rx := sx + padding + rng.Intn(max(1, xSlack+1))
		ry := sy + padding + rng.Intn(max(1, ySlack+1))

		// Clamp to safe map bounds
		if rx < 1 {
			rx = 1
		}
		if ry < 1 {
			ry = 1
		}
		if rx+rw > width-1 {
			rw = width - 1 - rx
		}
		if ry+rh > height-1 {
			rh = height - 1 - ry
		}
		if rw < 3 || rh < 3 {
			continue
		}

		r := Room{rx, ry, rw, rh}

		// Carve room floor
		for ry2 := r.Y; ry2 < r.Y+r.H; ry2++ {
			for rx2 := r.X; rx2 < r.X+r.W; rx2++ {
				tiles[ry2][rx2].Type = TileFloor
			}
		}

		// Connect to previous room via L-corridor
		if len(rooms) > 0 {
			cx1, cy1 := r.Center()
			cx2, cy2 := rooms[len(rooms)-1].Center()
			carveCorridors(tiles, cx1, cy1, cx2, cy2)
		}

		rooms = append(rooms, r)
	}

	// Place stairs in the last room's center
	if len(rooms) > 0 {
		last := rooms[len(rooms)-1]
		cx, cy := last.Center()
		tiles[cy][cx].Type = TileStairs
	}

	// Assign visual variants to all floor and wall tiles
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			tiles[y][x].Variant = byte(rng.Intn(4))
		}
	}

	return tiles, rooms
}

func carveCorridors(tiles [][]Tile, x1, y1, x2, y2 int) {
	if rng.Intn(2) == 0 {
		carveH(tiles, x1, x2, y1)
		carveV(tiles, y1, y2, x2)
	} else {
		carveV(tiles, y1, y2, x1)
		carveH(tiles, x1, x2, y2)
	}
}

func carveH(tiles [][]Tile, x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		tiles[y][x].Type = TileFloor
	}
}

func carveV(tiles [][]Tile, y1, y2, x int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		tiles[y][x].Type = TileFloor
	}
}
