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

// US-21 «Останній тайл скинуто» (shortcut до підрахунку, api.md «Кінець стопки»): якщо після skip_meeple тайлів,
// які можна покласти, не лишилося, партія завершена; місце, хід і фаза лишаються значеннями на момент завершення.
func TestUS21_EndOfDeck(t *testing.T) {
	tests := []struct {
		name          string
		deck          []string
		wantDiscarded []string
	}{
		{"останній тайл скинуто", []string{"C"}, []string{"C"}},
		{"усі тайли, що лишилися, скинуто", []string{"C", "C"}, []string{"C", "C"}},
		{"стопка порожня", []string{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := afterPlacement(closedCity(), tt.deck)
			if err := SkipMeeple(g, 0, 2); err != nil {
				t.Fatal(err)
			}
			if !Finished(g) || g.CurrentTile != "" || len(g.Deck) != 0 || !slices.Equal(g.Discarded, tt.wantDiscarded) {
				t.Fatalf("finished = %t, currentTile = %q, deck = %v, discarded = %v", Finished(g), g.CurrentTile, g.Deck, g.Discarded)
			}
			// Скинуті тайли на поле не потрапили; хід не перейшов.
			if len(g.Tiles) != 3 || g.CurrentSeat != 0 || g.TurnNumber != 5 || g.Phase != model.PhasePlaceMeeple {
				t.Fatalf("tiles = %d, seat = %d, turn = %d, phase = %s", len(g.Tiles), g.CurrentSeat, g.TurnNumber, g.Phase)
			}

			// api.md: після завершення будь-яка дія — WRONG_PHASE, навіть від того, чий був хід.
			before := snapshotOf(g)
			for seat := range 2 {
				if err := PlaceTile(g, seat, "oleg", pos(1, 0), 90); !errors.Is(err, ErrWrongPhase) {
					t.Fatalf("place_tile місця %d: err = %v, want %v", seat, err, ErrWrongPhase)
				}
				if err := SkipMeeple(g, seat, 2); !errors.Is(err, ErrWrongPhase) {
					t.Fatalf("skip_meeple місця %d: err = %v, want %v", seat, err, ErrWrongPhase)
				}
			}
			assertUnchanged(t, before, g)
		})
	}
}
