package httpapi

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"slices"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
	"github.com/EndlessNN/Webcassonne/server/internal/room"
)

// Ігрова частина room_state і повідомлення клієнта — docs/api.md §3.

type gameView struct {
	Phase               model.TurnPhase  `json:"phase"`
	TurnNumber          int              `json:"turnNumber"`
	CurrentPlayerID     string           `json:"currentPlayerId"`
	DeckCount           int              `json:"deckCount"`
	CurrentTile         *tileRef         `json:"currentTile"`
	DiscardedTiles      []tileRef        `json:"discardedTiles"`
	LegalPlacements     []placementView  `json:"legalPlacements"`
	LegalMeepleSegments []int            `json:"legalMeepleSegments"`
	Tiles               []placedTileView `json:"tiles"`
	Meeples             []meepleView     `json:"meeples"`
	FinalScores         []finalScoreView `json:"finalScores"`
}

type tileRef struct {
	TileTypeID string `json:"tileTypeId"`
}

type placementView struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Rotations []int `json:"rotations"`
}

type placedTileView struct {
	X          int    `json:"x"`
	Y          int    `json:"y"`
	TileTypeID string `json:"tileTypeId"`
	Rotation   int    `json:"rotation"`
}

type meepleView struct {
	PlayerID string `json:"playerId"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Segment  int    `json:"segment"`
}

type finalScoreView struct {
	PlayerID  string `json:"playerId"`
	Road      int    `json:"road"`
	City      int    `json:"city"`
	Monastery int    `json:"monastery"`
	Field     int    `json:"field"`
	Total     int    `json:"total"`
	Winner    bool   `json:"winner"`
}

func toGameView(snap room.Snapshot) *gameView {
	g := snap.Game
	if g == nil {
		return nil
	}
	v := &gameView{
		Phase:               g.Phase,
		TurnNumber:          g.TurnNumber,
		CurrentPlayerID:     snap.SeatPlayer[g.CurrentSeat],
		DeckCount:           len(g.Deck),
		DiscardedTiles:      make([]tileRef, 0, len(g.Discarded)),
		LegalPlacements:     make([]placementView, 0, len(snap.LegalPlacements)),
		LegalMeepleSegments: []int{}, // shortcut до US-13: міплів ще немає
		Tiles:               make([]placedTileView, 0, len(g.Tiles)),
		Meeples:             make([]meepleView, 0, len(g.Meeples)),
	}
	if g.CurrentTile != "" && snap.Status != model.RoomAbandoned {
		v.CurrentTile = &tileRef{TileTypeID: g.CurrentTile}
	}
	for _, id := range g.Discarded {
		v.DiscardedTiles = append(v.DiscardedTiles, tileRef{TileTypeID: id})
	}
	for _, p := range snap.LegalPlacements {
		v.LegalPlacements = append(v.LegalPlacements, placementView{X: p.Pos.X, Y: p.Pos.Y, Rotations: p.Rotations})
	}
	for _, t := range g.Tiles {
		v.Tiles = append(v.Tiles, placedTileView{X: t.Pos.X, Y: t.Pos.Y, TileTypeID: t.TileTypeID, Rotation: t.Rotation})
	}
	slices.SortFunc(v.Tiles, func(a, b placedTileView) int { // порядок не залежить від обходу map
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})
	for _, m := range g.Meeples {
		v.Meeples = append(v.Meeples, meepleView{PlayerID: m.OwnerID, X: m.Pos.X, Y: m.Pos.Y, Segment: m.Segment})
	}
	if snap.FinalScores != nil {
		v.FinalScores = make([]finalScoreView, 0, len(snap.FinalScores))
		for _, f := range snap.FinalScores {
			v.FinalScores = append(v.FinalScores, finalScoreView{
				PlayerID: f.PlayerID, Road: f.Road, City: f.City, Monastery: f.Monastery, Field: f.Field, Total: f.Total, Winner: f.Winner,
			})
		}
	}
	return v
}

// clientMessage — будь-яке повідомлення клієнт → сервер; поля, яких немає в типі, мають бути відсутні.
type clientMessage struct {
	Type     string `json:"type"`
	X        *int   `json:"x"`
	Y        *int   `json:"y"`
	Rotation *int   `json:"rotation"`
	Segment  *int   `json:"segment"`
}

// handleMessage розбирає повідомлення клієнта і виконує дію від імені гравця. Порядок перевірок —
// docs/api.md: спершу формат (VALIDATION_ERROR), далі перевіряє room.
func handleMessage(rooms *room.Service, sub *room.Sub, data []byte) error {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return room.ErrBadMessage
	}
	var m clientMessage
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	strict := dec.Decode(&m) == nil && errors.Is(dec.Decode(&struct{}{}), io.EOF)

	tile := m.X != nil && m.Y != nil && m.Rotation != nil // поля place_tile
	none := m.X == nil && m.Y == nil && m.Rotation == nil // жодного поля place_tile
	switch head.Type {
	case "start_game", "skip_meeple":
		if !strict || !none || m.Segment != nil {
			return room.ErrBadMessage
		}
		if head.Type == "start_game" {
			return rooms.StartGame(sub.RoomCode, sub.PlayerID)
		}
		return rooms.SkipMeeple(sub.RoomCode, sub.PlayerID)
	case "place_tile":
		if !strict || !tile || m.Segment != nil {
			return room.ErrBadMessage
		}
		return rooms.PlaceTile(sub.RoomCode, sub.PlayerID, model.Position{X: *m.X, Y: *m.Y}, *m.Rotation)
	case "place_meeple":
		if !strict || !none || m.Segment == nil {
			return room.ErrBadMessage
		}
		return rooms.PlaceMeeple(sub.RoomCode, sub.PlayerID, *m.Segment)
	}
	return room.ErrUnknownMessage
}

// wsError — тіло WS-повідомлення error для помилки дії; непередбачені помилки — INTERNAL, деталі лише в лозі.
func wsError(err error) APIError {
	var e *room.Error
	if errors.As(err, &e) {
		return toAPIError(e)
	}
	log.Printf("internal error: %v", err)
	return APIError{Code: "INTERNAL", Message: "Внутрішня помилка сервера"}
}

// Каталог тайлів — GET /api/tiles (docs/api.md §2).

type tilesResponse struct {
	Tiles []tileTypeView `json:"tiles"`
}

type tileTypeView struct {
	ID       string        `json:"id"`
	Count    int           `json:"count"`
	IsStart  bool          `json:"isStart"`
	Segments []segmentView `json:"segments"`
}

type segmentView struct {
	Type           model.FeatureType `json:"type"`
	Points         []string          `json:"points"`
	Pennant        bool              `json:"pennant"`
	AdjacentCities []int             `json:"adjacentCities"`
}

func listTiles(w http.ResponseWriter, _ *http.Request) {
	var res tilesResponse
	for _, t := range room.TileCatalog() {
		tv := tileTypeView{ID: t.ID, Count: t.Count, IsStart: t.IsStart}
		for _, s := range t.Segments {
			tv.Segments = append(tv.Segments, segmentView{
				Type: s.Type, Points: append([]string{}, s.Points...), Pennant: s.Pennant, AdjacentCities: append([]int{}, s.AdjacentCities...),
			})
		}
		res.Tiles = append(res.Tiles, tv)
	}
	writeJSON(w, http.StatusOK, res)
}
