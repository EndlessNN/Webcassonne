package game

import (
	"cmp"
	"errors"
	"slices"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
)

// Помилки розміщення тайла (US-12). Коди й тексти для клієнта — у room за docs/api.md.
var (
	ErrInvalidRotation = errors.New("обертання поза 0, 90, 180, 270")
	ErrUnknownTile     = errors.New("невідомий тип тайла")
	ErrOccupied        = errors.New("клітинка вже зайнята")
	ErrNoNeighbor      = errors.New("клітинка не межує з жодним тайлом")
	ErrEdgeMismatch    = errors.New("сторони тайлів не збігаються")
)

// Placement — клітинка, куди можна покласти тайл, і допустимі обертання для неї (US-11).
type Placement struct {
	Pos       model.Position
	Rotations []int
}

// CheckPlacement перевіряє, чи можна покласти тайл tileTypeID у pos з обертанням rotation:
// клітинка вільна, має хоча б одного сусіда по стороні, усі суміжні сторони збігаються (технічний аналіз, заборона 5).
func CheckPlacement(tiles map[model.Position]model.PlacedTile, tileTypeID string, pos model.Position, rotation int) error {
	if !ValidRotation(rotation) {
		return ErrInvalidRotation
	}
	t, ok := tileIndex()[tileTypeID]
	if !ok {
		return ErrUnknownTile
	}
	if _, taken := tiles[pos]; taken {
		return ErrOccupied
	}
	mine := edgeTypes(t, rotation)
	neighbors := 0
	for side, off := range sideOffset {
		n, ok := tiles[model.Position{X: pos.X + off.X, Y: pos.Y + off.Y}]
		if !ok {
			continue
		}
		neighbors++
		nt, ok := tileIndex()[n.TileTypeID]
		if !ok {
			return ErrUnknownTile
		}
		if !sidesMatch(mine, side, edgeTypes(nt, n.Rotation)) {
			return ErrEdgeMismatch
		}
	}
	if neighbors == 0 {
		return ErrNoNeighbor
	}
	return nil
}

// LegalPlacements — усі вільні клітинки поруч з полем, куди тайл лягає хоча б в одному обертанні (US-11).
// Порядок детермінований: за y, потім за x; обертання — за зростанням.
func LegalPlacements(tiles map[model.Position]model.PlacedTile, tileTypeID string) []Placement {
	var out []Placement
	for _, pos := range frontier(tiles) {
		var rots []int
		for _, r := range Rotations {
			if CheckPlacement(tiles, tileTypeID, pos, r) == nil {
				rots = append(rots, r)
			}
		}
		if len(rots) > 0 {
			out = append(out, Placement{Pos: pos, Rotations: rots})
		}
	}
	return out
}

// frontier — вільні клітинки, що межують стороною хоча б з одним тайлом.
func frontier(tiles map[model.Position]model.PlacedTile) []model.Position {
	seen := map[model.Position]bool{}
	var out []model.Position
	for p := range tiles {
		for _, off := range sideOffset {
			c := model.Position{X: p.X + off.X, Y: p.Y + off.Y}
			if _, taken := tiles[c]; taken || seen[c] {
				continue
			}
			seen[c] = true
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b model.Position) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})
	return out
}
