package game

import (
	"errors"
	"slices"
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

// US-11: допустимі позиції в усіх обертаннях. Очікування — ручний розрахунок за docs/tiles.md:
// стартовий D (обертання 0) має N — місто, E і W — поле-дорога-поле, S — поле.
// Обертання за годинниковою (П-6): при 90° західна сторона тайла стає північною.
func TestUS11_LegalPlacements(t *testing.T) {
	all := []int{0, 90, 180, 270}
	tests := []struct {
		name  string
		tiles map[model.Position]model.PlacedTile
		tile  string
		want  []Placement // порядок позицій і обертань api.md не задає — порівнюємо як множини
	}{
		{
			// E — місто на півночі, решта поле. Над D: місто E має бути на півдні → лише 180.
			// Під D: північ E має бути полем → 90, 180, 270. Праворуч і ліворуч у E немає дороги.
			"тайл E: місто лише до міста, поле до поля", board(), "E",
			[]Placement{{pos(0, -1), []int{180}}, {pos(0, 1), []int{90, 180, 270}}},
		},
		{
			// U — пряма дорога N–S. Над D немає міста. Праворуч, під і ліворуч — лише дорога схід–захід (90, 270).
			"тайл U: дорога в трьох позиціях", board(), "U",
			[]Placement{{pos(-1, 0), []int{90, 270}}, {pos(1, 0), []int{90, 270}}, {pos(0, 1), []int{90, 270}}},
		},
		{
			// C — місто на всіх сторонах: лише над D, у будь-якому обертанні.
			"тайл C: лише до міста стартового", board(), "C",
			[]Placement{{pos(0, -1), all}},
		},
		{
			// Місто D закрите E 180°; відкриті сторони лише поле і поле-дорога-поле (як в AC US-09).
			"тайл, який неможливо покласти", board(model.PlacedTile{Pos: pos(0, -1), TileTypeID: "E", Rotation: 180}), "C",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LegalPlacements(tt.tiles, tt.tile)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d позицій %+v, want %d %+v", len(got), got, len(tt.want), tt.want)
			}
			byPos := map[model.Position][]int{}
			for _, p := range got {
				byPos[p.Pos] = slices.Sorted(slices.Values(p.Rotations))
			}
			for _, w := range tt.want {
				if r, ok := byPos[w.Pos]; !ok || !slices.Equal(r, w.Rotations) {
					t.Fatalf("позиція %v: rotations = %v (є: %t), want %v; усе: %+v", w.Pos, r, ok, w.Rotations, got)
				}
			}
		})
	}
}

// US-12: клітинка з двома сусідами — мають збігатися всі суміжні сторони (api.md: «хоча б одна суміжна сторона
// не збігається» → ILLEGAL_PLACEMENT).
func TestUS12_PlaceTileTwoNeighbors(t *testing.T) {
	// Поле: D (0, 0); E 180° у (0, −1) — схід E 180° поле; U 90° у (1, 0) — північ U 90° поле.
	// Клітинка (1, −1) межує із заходу з E 180°, з півдня з U 90°.
	tiles := func() map[model.Position]model.PlacedTile {
		return board(
			model.PlacedTile{Pos: pos(0, -1), TileTypeID: "E", Rotation: 180},
			model.PlacedTile{Pos: pos(1, 0), TileTypeID: "U", Rotation: 90},
		)
	}
	tests := []struct {
		name     string
		rotation int
		want     error
	}{
		// E 0°: захід поле ↔ поле, південь поле ↔ поле.
		{"обидві сторони збігаються", 0, nil},
		// E 180°: захід поле ↔ поле збігається, але південь місто ↔ поле U — ні.
		{"збігається лише одна з двох сторін", 180, ErrEdgeMismatch},
		// E 90°: місто на сході — там сусіда немає; захід поле, південь поле.
		{"місто у бік без сусіда", 90, nil},
		// E 270°: місто на заході ↔ поле E 180°.
		{"місто проти поля з другого сусіда", 270, ErrEdgeMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newTestGame("E")
			g.Tiles = tiles()
			before := snapshotOf(g)
			err := PlaceTile(g, 1, "iryna", pos(1, -1), tt.rotation)
			if tt.want != nil {
				if !errors.Is(err, tt.want) {
					t.Fatalf("err = %v, want %v", err, tt.want)
				}
				assertUnchanged(t, before, g)
				return
			}
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			want := model.PlacedTile{Pos: pos(1, -1), TileTypeID: "E", Rotation: tt.rotation, PlacedBy: "iryna", TurnNumber: 2}
			if g.Tiles[pos(1, -1)] != want || len(g.Tiles) != 4 || g.Phase != model.PhasePlaceMeeple {
				t.Fatalf("tile = %+v, tiles = %d, phase = %s", g.Tiles[pos(1, -1)], len(g.Tiles), g.Phase)
			}
		})
	}
}

// Негативні випадки з api.md без сценарію в AC: обертання поза {0, 90, 180, 270} і place_tile не в тій фазі.
func TestUS12_PlaceTileRejected(t *testing.T) {
	t.Run("VALIDATION_ERROR: обертання 45", func(t *testing.T) {
		g := newTestGame("U")
		before := snapshotOf(g)
		if err := PlaceTile(g, 1, "iryna", pos(1, 0), 45); !errors.Is(err, ErrInvalidRotation) {
			t.Fatalf("err = %v, want %v", err, ErrInvalidRotation)
		}
		assertUnchanged(t, before, g)
	})
	t.Run("WRONG_PHASE: другий тайл у фазі PLACE_MEEPLE", func(t *testing.T) {
		g := newTestGame("U")
		if err := PlaceTile(g, 1, "iryna", pos(1, 0), 90); err != nil {
			t.Fatalf("підготовка: %v", err)
		}
		before := snapshotOf(g)
		if err := PlaceTile(g, 1, "iryna", pos(-1, 0), 90); !errors.Is(err, ErrWrongPhase) {
			t.Fatalf("err = %v, want %v", err, ErrWrongPhase)
		}
		assertUnchanged(t, before, g)
	})
}
