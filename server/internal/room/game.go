package room

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/EndlessNN/Webcassonne/server/internal/game"
	"github.com/EndlessNN/Webcassonne/server/internal/model"
)

// Помилки ігрових дій — коди й тексти з docs/api.md §1 («Порядок перевірок і тексти WS-помилок»).
var (
	ErrUnknownMessage   = &Error{Code: "VALIDATION_ERROR", Message: "Невідомий тип повідомлення"}
	ErrBadMessage       = &Error{Code: "VALIDATION_ERROR", Message: "Неправильне повідомлення"}
	ErrNotHost          = &Error{Code: "NOT_HOST", Message: "Почати гру може лише організатор"}
	ErrNotEnoughPlayers = &Error{Code: "NOT_ENOUGH_PLAYERS", Message: "Потрібно щонайменше 2 гравці"}
	ErrWrongPhase       = &Error{Code: "WRONG_PHASE", Message: "Ця дія зараз недоступна"}
	ErrNotYourTurn      = &Error{Code: "NOT_YOUR_TURN", Message: "Зараз хід іншого гравця"}
	ErrCellOccupied     = &Error{Code: "ILLEGAL_PLACEMENT", Message: "Клітинка вже зайнята"}
	ErrNoNeighbor       = &Error{Code: "ILLEGAL_PLACEMENT", Message: "Тайл має межувати з іншим тайлом"}
	ErrEdgeMismatch     = &Error{Code: "ILLEGAL_PLACEMENT", Message: "Сторони тайлів мають збігатися"}
	ErrIllegalMeeple    = &Error{Code: "ILLEGAL_MEEPLE", Message: "Міпла не можна поставити на цей сегмент"}
)

// colorsBySeat — кольори за порядком ходу (П-9).
var colorsBySeat = [model.MaxPlayers]model.Color{model.ColorRed, model.ColorBlue, model.ColorGreen, model.ColorYellow, model.ColorBlack}

// Snapshot — знімок кімнати для читання поза s.mu разом з обчисленими полями room_state.
type Snapshot struct {
	model.Room                        // глибока копія
	LegalPlacements []game.Placement  // лише у PLAYING, фаза PLACE_TILE; інакше nil
	FinalScores     []game.FinalScore // лише у FINISHED; інакше nil
	MeeplesInSupply map[string]int    // playerID → міпли в запасі (US-16)
	SeatPlayer      map[int]string    // місце → playerID
}

// TileCatalog — статичний каталог типів тайлів для GET /api/tiles.
func TileCatalog() []game.TileType { return game.Catalog() }

// StartGame запускає партію (US-05): випадковий порядок місць, кольори за місцями, стартовий тайл, стопка.
func (s *Service) StartGame(code, playerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.store.Get(code)
	switch {
	case !ok:
		return ErrRoomNotFound
	case r.Status != model.RoomLobby:
		return ErrGameStarted
	case r.HostPlayerID != playerID:
		return ErrNotHost
	case len(r.Players) < model.MinPlayers:
		return ErrNotEnoughPlayers
	}
	g, err := game.New(s.rng)
	if err != nil {
		return fmt.Errorf("start game %s: %w", code, err)
	}
	for i, seat := range s.rng.Perm(len(r.Players)) {
		r.Players[i].Seat = seat
		r.Players[i].Color = colorsBySeat[seat]
	}
	r.Game = g
	r.Status = model.RoomPlaying
	s.broadcast(r)
	return nil
}

// PlaceTile кладе поточний тайл (US-12). Порядок перевірок — docs/api.md: формат → статус → автор → фаза → правило.
func (s *Service) PlaceTile(code, playerID string, pos model.Position, rotation int) error {
	if !game.ValidRotation(rotation) {
		return ErrBadMessage
	}
	return s.act(code, playerID, func(r *model.Room, seat int) error {
		return game.PlaceTile(r.Game, seat, playerID, pos, rotation)
	})
}

// SkipMeeple завершує хід без міпла (US-14). Якщо стопка скінчилася — кімната FINISHED.
func (s *Service) SkipMeeple(code, playerID string) error {
	return s.act(code, playerID, func(r *model.Room, seat int) error {
		if err := game.SkipMeeple(r.Game, seat, len(r.Players)); err != nil {
			return err
		}
		if game.Finished(r.Game) {
			r.Status = model.RoomFinished
			r.FinishedAt = s.now()
		}
		return nil
	})
}

// PlaceMeeple — shortcut до US-13: legalMeepleSegments завжди порожній, тож у своїй фазі дія відхиляється з ILLEGAL_MEEPLE.
func (s *Service) PlaceMeeple(code, playerID string, segment int) error {
	return s.act(code, playerID, func(r *model.Room, seat int) error {
		switch {
		case seat != r.Game.CurrentSeat:
			return game.ErrNotYourTurn
		case r.Game.Phase != model.PhasePlaceMeeple:
			return game.ErrWrongPhase
		}
		return ErrIllegalMeeple
	})
}

// act виконує ігрову дію під s.mu: перевіряє статус кімнати й гравця, а після успіху розсилає знімок.
// Дія має повертати помилку до будь-якої зміни стану (перевірки в game ідуть першими).
func (s *Service) act(code, playerID string, do func(r *model.Room, seat int) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.store.Get(code)
	if !ok {
		return ErrRoomNotFound
	}
	if r.Status != model.RoomPlaying || r.Game == nil {
		return ErrWrongPhase
	}
	i := slices.IndexFunc(r.Players, func(p *model.Player) bool { return p.ID == playerID })
	if i < 0 {
		return ErrInvalidToken
	}
	if err := do(r, r.Players[i].Seat); err != nil {
		return gameError(err)
	}
	s.broadcast(r)
	return nil
}

// gameError перекладає помилки game у коди docs/api.md; невідомі лишаються внутрішніми.
func gameError(err error) error {
	var e *Error
	switch {
	case errors.As(err, &e):
		return e
	case errors.Is(err, game.ErrInvalidRotation):
		return ErrBadMessage
	case errors.Is(err, game.ErrNotYourTurn):
		return ErrNotYourTurn
	case errors.Is(err, game.ErrWrongPhase):
		return ErrWrongPhase
	case errors.Is(err, game.ErrOccupied):
		return ErrCellOccupied
	case errors.Is(err, game.ErrNoNeighbor):
		return ErrNoNeighbor
	case errors.Is(err, game.ErrEdgeMismatch):
		return ErrEdgeMismatch
	}
	return err
}

// snapshot глибоко копіює кімнату (зокрема Game), щоб знімок серіалізувався поза s.mu без гонок,
// і обчислює похідні поля room_state. Викликати під s.mu.
func snapshot(r *model.Room) Snapshot {
	c := *r
	c.Players = make([]*model.Player, len(r.Players))
	for i, p := range r.Players {
		pc := *p
		c.Players[i] = &pc
	}
	snap := Snapshot{MeeplesInSupply: map[string]int{}, SeatPlayer: map[int]string{}}
	for _, p := range r.Players {
		snap.MeeplesInSupply[p.ID] = model.MeeplesPerPlayer
	}
	if r.Game != nil {
		g := cloneGame(r.Game)
		c.Game = g
		for _, p := range r.Players {
			snap.MeeplesInSupply[p.ID] = game.MeeplesInSupply(g, p.ID)
			snap.SeatPlayer[p.Seat] = p.ID
		}
		if r.Status == model.RoomPlaying && g.Phase == model.PhasePlaceTile && !game.Finished(g) {
			snap.LegalPlacements = game.LegalPlacements(g.Tiles, g.CurrentTile)
		}
		if r.Status == model.RoomFinished {
			ids := make([]string, len(r.Players)) // за порядком ходу
			for seat := range ids {
				ids[seat] = snap.SeatPlayer[seat]
			}
			snap.FinalScores = game.ZeroFinalScores(ids)
		}
	}
	snap.Room = c
	return snap
}

func cloneGame(g *model.Game) *model.Game {
	c := *g
	c.Deck = slices.Clone(g.Deck)
	c.Discarded = slices.Clone(g.Discarded)
	c.Tiles = maps.Clone(g.Tiles)
	c.Meeples = slices.Clone(g.Meeples)
	c.Scores = slices.Clone(g.Scores)
	if g.LastPlaced != nil {
		p := *g.LastPlaced
		c.LastPlaced = &p
	}
	return &c
}
