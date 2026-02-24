//go:build js && wasm

package main

import "math/rand"

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
}

type Room struct {
	X, Y, W, H int
}

func (r Room) Center() (int, int) {
	return r.X + r.W/2, r.Y + r.H/2
}

func (r Room) Intersects(other Room) bool {
	return r.X-1 <= other.X+other.W &&
		r.X+r.W+1 >= other.X &&
		r.Y-1 <= other.Y+other.H &&
		r.Y+r.H+1 >= other.Y
}

// GenerateDungeon creates a room-and-corridor dungeon.
// Returns the tile grid and the list of rooms in placement order.
func GenerateDungeon(width, height int) ([][]Tile, []Room) {
	tiles := make([][]Tile, height)
	for i := range tiles {
		tiles[i] = make([]Tile, width)
		// All walls by default (zero value of TileType is TileWall)
	}

	rooms := []Room{}
	maxRooms := 9

	for attempt := 0; attempt < 300 && len(rooms) < maxRooms; attempt++ {
		w := rand.Intn(7) + 4  // 4-10 wide
		h := rand.Intn(4) + 4  // 4-7 tall
		x := rand.Intn(width-w-2) + 1
		y := rand.Intn(height-h-2) + 1

		r := Room{x, y, w, h}

		overlaps := false
		for _, existing := range rooms {
			if r.Intersects(existing) {
				overlaps = true
				break
			}
		}
		if overlaps {
			continue
		}

		// Carve room
		for ry := r.Y; ry < r.Y+r.H; ry++ {
			for rx := r.X; rx < r.X+r.W; rx++ {
				tiles[ry][rx].Type = TileFloor
			}
		}

		// Connect to previous room via L-shaped corridor
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

	return tiles, rooms
}

func carveCorridors(tiles [][]Tile, x1, y1, x2, y2 int) {
	if rand.Intn(2) == 0 {
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
