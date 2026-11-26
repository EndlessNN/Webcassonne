package game

import (
	"errors"
	"maps"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
)

// US-05: нова партія. Очікування — з AC US-05, docs/tiles.md і docs/api.md, не з коду.
func TestUS05_NewGame(t *testing.T) {
	t.Run("успішний старт", func(t *testing.T) {
		g, err := New(rand.New(rand.NewPCG(1, 2)))
		if err != nil {
			t.Fatal(err)
		}

		// У центрі столу стартовий тайл: D у (0, 0) з обертанням 0, без автора, хід 0 (api.md turnNumber).
		wantTiles := map[model.Position]model.PlacedTile{pos(0, 0): {Pos: pos(0, 0), TileTypeID: "D"}}
		if !maps.Equal(g.Tiles, wantTiles) {
			t.Fatalf("tiles = %+v, want %+v", g.Tiles, wantTiles)
		}
		// 71 = 1 у першого гравця + 70 у стопці (AC US-05, api.md deckCount).
		if len(g.Deck) != 70 {
			t.Fatalf("deck = %d, want 70", len(g.Deck))
		}
		if g.CurrentTile == "" {
			t.Fatal("перший гравець не отримав тайл")
		}
		// Перший хід — 1, ходить місце 0, фаза PLACE_TILE (api.md).
		if g.CurrentSeat != 0 || g.Phase != model.PhasePlaceTile || g.TurnNumber != 1 || len(g.Discarded) != 0 {
			t.Fatalf("seat = %d, phase = %s, turn = %d, discarded = %v", g.CurrentSeat, g.Phase, g.TurnNumber, g.Discarded)
		}
		if len(g.Meeples) != 0 || len(g.Scores) != 0 {
			t.Fatalf("meeples = %v, scores = %v", g.Meeples, g.Scores)
		}

		// Склад 71 тайла — docs/tiles.md, «Підсумок»; D: 4 − 1 стартовий.
		want := map[string]int{
			"A": 2, "B": 4, "C": 1, "D": 3, "E": 5, "F": 2, "G": 1, "H": 3, "I": 2, "J": 3, "K": 3, "L": 3,
			"M": 2, "N": 3, "O": 2, "P": 3, "Q": 1, "R": 3, "S": 2, "T": 1, "U": 8, "V": 9, "W": 4, "X": 1,
		}
		got := map[string]int{g.CurrentTile: 1}
		for _, id := range g.Deck {
			got[id]++
		}
		if !maps.Equal(got, want) {
			t.Fatalf("склад стопки = %v, want %v", got, want)
		}
	})
}

// US-07: дія не у свій хід не виконується, а стан гри не змінюється.
func TestUS07_NotYourTurn(t *testing.T) {
	tests := []struct {
		name string
		pos  model.Position
		rot  int
	}{
		// Хід Ірини (місце 1), Олег (місце 0) кладе U 90° праворуч від D — позиція допустима (див. TestUS12_PlaceTile).
		{"покласти тайл не у свій хід", pos(1, 0), 90},
		// api.md: автор перевіряється раніше за правило гри, тож зайнята клітинка дає NOT_YOUR_TURN.
		{"чужий хід на зайняту клітинку", pos(0, 0), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newTestGame("U")
			before := snapshotOf(g)
			if err := PlaceTile(g, 0, "oleg", tt.pos, tt.rot); !errors.Is(err, ErrNotYourTurn) {
				t.Fatalf("err = %v, want %v", err, ErrNotYourTurn)
			}
			assertUnchanged(t, before, g)
		})
	}
}

// US-09: тайл зі стопки на початку ходу і автоматичне скидання тайла, який неможливо покласти.
func TestUS09_DrawTile(t *testing.T) {
	// afterPlacement — хід Олега (місце 0) з 2 гравців у фазі PLACE_MEEPLE: тайл уже покладено, далі черга Ірини.
	afterPlacement := func(tiles map[model.Position]model.PlacedTile, deck []string) *model.Game {
		last := pos(0, 1)
		return &model.Game{
			Deck: deck, CurrentTile: "B", CurrentSeat: 0, Phase: model.PhasePlaceMeeple,
			TurnNumber: 5, LastPlaced: &last, Tiles: tiles,
		}
	}
	// Поле з AC «Тайл, який неможливо покласти»: D у (0, 0) і E 180° у (0, −1) (місто E на півдні ↔ місто D).
	// Відкриті сторони: E 180° — N, E, W поле (FFF); D — E і W поле-дорога-поле, S поле.
	// C (місто на всіх сторонах) не лягає в жодну клітинку, в жодному обертанні.
	closedCity := func() map[model.Position]model.PlacedTile {
		return board(
			model.PlacedTile{Pos: pos(0, 1), TileTypeID: "B", PlacedBy: "oleg", TurnNumber: 5},
			model.PlacedTile{Pos: pos(0, -1), TileTypeID: "E", Rotation: 180},
		)
	}

	t.Run("отримання тайла на початку ходу", func(t *testing.T) {
		// Настав хід Ірини, у стопці 50 тайлів: верх U, далі 49 B. Обидва лягають поруч з D.
		deck := append([]string{"U"}, slices.Repeat([]string{"B"}, 49)...)
		g := afterPlacement(board(model.PlacedTile{Pos: pos(0, 1), TileTypeID: "B", PlacedBy: "oleg", TurnNumber: 5}), deck)
		if len(g.Deck) != 50 {
			t.Fatalf("підготовка: deck = %d", len(g.Deck))
		}

		if err := SkipMeeple(g, 0, 2); err != nil {
			t.Fatal(err)
		}
		// Ірина бачить свій новий тайл, лічильник стопки показує 49.
		if g.CurrentTile != "U" || len(g.Deck) != 49 {
			t.Fatalf("currentTile = %q, deck = %d, want U, 49", g.CurrentTile, len(g.Deck))
		}
		if g.CurrentSeat != 1 || g.Phase != model.PhasePlaceTile || g.TurnNumber != 6 || len(g.Discarded) != 0 {
			t.Fatalf("seat = %d, phase = %s, turn = %d, discarded = %v", g.CurrentSeat, g.Phase, g.TurnNumber, g.Discarded)
		}
	})

	t.Run("тайл, який неможливо покласти", func(t *testing.T) {
		g := afterPlacement(closedCity(), []string{"C", "U", "B"})

		if err := SkipMeeple(g, 0, 2); err != nil {
			t.Fatal(err)
		}
		// C скинуто, Ірина отримує наступний тайл (U); зі стопки пішли обидва.
		if !slices.Equal(g.Discarded, []string{"C"}) || g.CurrentTile != "U" || !slices.Equal(g.Deck, []string{"B"}) {
			t.Fatalf("discarded = %v, currentTile = %q, deck = %v", g.Discarded, g.CurrentTile, g.Deck)
		}
		if g.CurrentSeat != 1 || g.Phase != model.PhasePlaceTile {
			t.Fatalf("seat = %d, phase = %s", g.CurrentSeat, g.Phase)
		}
		if len(g.Tiles) != 3 {
			t.Fatalf("скинутий тайл потрапив на поле: %+v", g.Tiles)
		}
	})

	t.Run("скинуті тайли очищуються на наступному витягуванні", func(t *testing.T) {
		// api.md discardedTiles: «Очищується на наступному витягуванні».
		g := afterPlacement(closedCity(), []string{"U", "B"})
		g.Discarded = []string{"C"}

		if err := SkipMeeple(g, 0, 2); err != nil {
			t.Fatal(err)
		}
		if len(g.Discarded) != 0 || g.CurrentTile != "U" {
			t.Fatalf("discarded = %v, currentTile = %q", g.Discarded, g.CurrentTile)
		}
	})

	t.Run("NOT_YOUR_TURN: skip_meeple від неактивного гравця", func(t *testing.T) {
		g := afterPlacement(closedCity(), []string{"U", "B"})
		before := snapshotOf(g)
		if err := SkipMeeple(g, 1, 2); !errors.Is(err, ErrNotYourTurn) {
			t.Fatalf("err = %v, want %v", err, ErrNotYourTurn)
		}
		assertUnchanged(t, before, g)
	})

	t.Run("WRONG_PHASE: skip_meeple у фазі PLACE_TILE", func(t *testing.T) {
		g := newTestGame("U")
		before := snapshotOf(g)
		if err := SkipMeeple(g, 1, 2); !errors.Is(err, ErrWrongPhase) {
			t.Fatalf("err = %v, want %v", err, ErrWrongPhase)
		}
		assertUnchanged(t, before, g)
	})
}
