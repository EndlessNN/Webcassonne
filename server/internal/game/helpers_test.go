package game

import (
	"maps"
	"slices"
	"testing"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
)

// pos — коротший запис позиції в таблицях.
func pos(x, y int) model.Position { return model.Position{X: x, Y: y} }

// board — стартовий D у (0, 0) з обертанням 0 плюс додаткові тайли.
func board(extra ...model.PlacedTile) map[model.Position]model.PlacedTile {
	tiles := map[model.Position]model.PlacedTile{
		pos(0, 0): {Pos: pos(0, 0), TileTypeID: "D"},
	}
	for _, t := range extra {
		tiles[t.Pos] = t
	}
	return tiles
}

// gameSnapshot — копія стану, яку можна порівняти до і після відхиленої дії.
type gameSnapshot struct {
	deck        []string
	currentTile string
	discarded   []string
	seat        int
	phase       model.TurnPhase
	turn        int
	lastPlaced  *model.Position // копія значення, а не вказівник гри
	tiles       map[model.Position]model.PlacedTile
	meeples     []model.Meeple
	scores      []model.ScoreEntry
}

func snapshotOf(g *model.Game) gameSnapshot {
	var last *model.Position
	if g.LastPlaced != nil {
		p := *g.LastPlaced
		last = &p
	}
	return gameSnapshot{
		deck:        slices.Clone(g.Deck),
		currentTile: g.CurrentTile,
		discarded:   slices.Clone(g.Discarded),
		seat:        g.CurrentSeat,
		phase:       g.Phase,
		turn:        g.TurnNumber,
		lastPlaced:  last,
		tiles:       maps.Clone(g.Tiles),
		meeples:     slices.Clone(g.Meeples),
		scores:      slices.Clone(g.Scores),
	}
}

// assertUnchanged — стан гри після помилки збігається зі станом до дії.
func assertUnchanged(t *testing.T, before gameSnapshot, g *model.Game) {
	t.Helper()
	after := snapshotOf(g)
	if !slices.Equal(before.deck, after.deck) || before.currentTile != after.currentTile ||
		!slices.Equal(before.discarded, after.discarded) || before.seat != after.seat ||
		before.phase != after.phase || before.turn != after.turn || !samePos(before.lastPlaced, after.lastPlaced) ||
		!maps.Equal(before.tiles, after.tiles) || !slices.Equal(before.meeples, after.meeples) || !slices.Equal(before.scores, after.scores) {
		t.Fatalf("стан змінився попри помилку:\nдо    %+v\nпісля %+v", before, after)
	}
}

// samePos — обидві позиції відсутні або однакові за значенням.
func samePos(a, b *model.Position) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}
