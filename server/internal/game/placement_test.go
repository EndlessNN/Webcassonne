package game

import (
	"errors"
	"testing"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
)

// newTestGame — партія після старту: стартовий D у (0, 0) з обертанням 0, хід Ірини (місце 1), фаза PLACE_TILE.
// За docs/tiles.md, D: місто на півночі (N0 N1 N2), дорога E1–W1, поле на всьому півдні (S0 S1 S2).
func newTestGame(currentTile string) *model.Game {
	return &model.Game{
		Deck:        []string{"B", "B"},
		CurrentTile: currentTile,
		CurrentSeat: 1,
		Phase:       model.PhasePlaceTile,
		TurnNumber:  2,
		Tiles: map[model.Position]model.PlacedTile{
			{X: 0, Y: 0}: {Pos: model.Position{X: 0, Y: 0}, TileTypeID: "D"},
		},
	}
}

// US-12: розміщення тайла. Очікування — з AC US-12 і docs/tiles.md, не з коду.
func TestUS12_PlaceTile(t *testing.T) {
	tests := []struct {
		name     string
		tile     string
		pos      model.Position
		rotation int
		want     error // nil — тайл має бути покладено
	}{
		// U (дорога N1–S1), обернений на 90 → дорога E1–W1; захід U (поле, дорога, поле) ↔ схід D (поле, дорога, поле).
		{"розміщення на допустиму позицію праворуч від стартового", "U", model.Position{X: 1, Y: 0}, 90, nil},
		// E (місто на півночі) під D: місто E проти поля на півдні D.
		{"недопустима позиція: місто проти поля", "E", model.Position{X: 0, Y: 1}, 0, ErrEdgeMismatch},
		// Та сама клітинка, E обернений на 180 → місто на півдні, поле на півночі проти поля D.
		{"та сама клітинка після обертання стає допустимою", "E", model.Position{X: 0, Y: 1}, 180, nil},
		// E обернений на 180 над D: місто E (південь) ↔ місто D (північ).
		{"місто до міста над стартовим", "E", model.Position{X: 0, Y: -1}, 180, nil},
		// U без обертання праворуч від D: поле U (захід) проти дороги D (схід).
		{"дорога проти поля", "U", model.Position{X: 1, Y: 0}, 0, ErrEdgeMismatch},
		{"клітинка без сусідів", "B", model.Position{X: 5, Y: 5}, 0, ErrNoNeighbor},
		{"лише діагональний сусід — це не сусід", "B", model.Position{X: 1, Y: 1}, 0, ErrNoNeighbor},
		{"зайнята клітинка", "B", model.Position{X: 0, Y: 0}, 0, ErrOccupied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newTestGame(tt.tile)
			err := PlaceTile(g, 1, "iryna", tt.pos, tt.rotation)

			if tt.want != nil {
				if !errors.Is(err, tt.want) {
					t.Fatalf("err = %v, want %v", err, tt.want)
				}
				// Тайл не розміщується, стан гри не змінюється.
				if len(g.Tiles) != 1 || g.Phase != model.PhasePlaceTile || g.LastPlaced != nil ||
					g.CurrentTile != tt.tile || g.CurrentSeat != 1 || g.TurnNumber != 2 || len(g.Deck) != 2 {
					t.Fatalf("стан змінився попри помилку: %+v", g)
				}
				return
			}

			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			want := model.PlacedTile{Pos: tt.pos, TileTypeID: tt.tile, Rotation: tt.rotation, PlacedBy: "iryna", TurnNumber: 2}
			if got, ok := g.Tiles[tt.pos]; !ok || got != want {
				t.Fatalf("tile at %v = %+v, want %+v", tt.pos, got, want)
			}
			if len(g.Tiles) != 2 {
				t.Fatalf("tiles = %d, want 2", len(g.Tiles))
			}
			// Ірині пропонується поставити міпла або пропустити.
			if g.Phase != model.PhasePlaceMeeple || g.LastPlaced == nil || *g.LastPlaced != tt.pos {
				t.Fatalf("phase = %s, lastPlaced = %v", g.Phase, g.LastPlaced)
			}
			if g.CurrentSeat != 1 || len(g.Deck) != 2 {
				t.Fatalf("хід або стопка змінилися: seat = %d, deck = %d", g.CurrentSeat, len(g.Deck))
			}
		})
	}
}
