package game

import (
	"errors"
	"math/rand/v2"
	"sync"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
)

// Помилки дій гравця. Коди для клієнта — у room за docs/api.md.
var (
	ErrNotYourTurn = errors.New("зараз хід іншого гравця")
	ErrWrongPhase  = errors.New("дія не відповідає фазі ходу")
	ErrNoPlayers   = errors.New("кількість гравців має бути додатною")
)

var validCatalog = sync.OnceValue(func() error { return ValidateCatalog(catalog) })

// NewDeck — стопка з 71 тайла: усі тайли каталогу, крім одного стартового, перемішані генератором rng.
// Верх стопки — Deck[0].
func NewDeck(rng *rand.Rand) []string {
	deck := make([]string, 0, TotalTiles-1)
	for _, t := range catalog {
		n := t.Count
		if t.IsStart {
			n-- // стартовий екземпляр уже на полі
		}
		for range n {
			deck = append(deck, t.ID)
		}
	}
	rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

// New — нова партія (US-05): стартовий тайл у (0, 0) з обертанням 0, перемішана стопка,
// перший у черзі (місце 0) одразу тягне тайл (US-09). Порядок місць визначає room.
func New(rng *rand.Rand) (*model.Game, error) {
	if err := validCatalog(); err != nil {
		return nil, err
	}
	start := model.Position{X: 0, Y: 0}
	g := &model.Game{
		Deck:       NewDeck(rng),
		Phase:      model.PhasePlaceTile,
		TurnNumber: 1,
		Tiles:      map[model.Position]model.PlacedTile{start: {Pos: start, TileTypeID: StartTileID()}},
	}
	draw(g)
	return g, nil
}

// MeeplesInSupply — 7 мінус міпли гравця на полі (US-16).
func MeeplesInSupply(g *model.Game, playerID string) int {
	n := model.MeeplesPerPlayer
	for _, m := range g.Meeples {
		if m.OwnerID == playerID {
			n--
		}
	}
	return n
}

// Finished — стопка скінчилася: поточного тайла немає, партія завершена.
func Finished(g *model.Game) bool { return g.CurrentTile == "" }

// PlaceTile кладе поточний тайл (US-12). Порядок перевірок — як у docs/api.md:
// формат (обертання) → партія не завершена → активний гравець → фаза → правила стикування.
// При помилці стан не змінюється.
func PlaceTile(g *model.Game, seat int, playerID string, pos model.Position, rotation int) error {
	if !ValidRotation(rotation) {
		return ErrInvalidRotation
	}
	if Finished(g) {
		return ErrWrongPhase
	}
	if seat != g.CurrentSeat {
		return ErrNotYourTurn
	}
	if g.Phase != model.PhasePlaceTile {
		return ErrWrongPhase
	}
	if err := CheckPlacement(g.Tiles, g.CurrentTile, pos, rotation); err != nil {
		return err
	}
	g.Tiles[pos] = model.PlacedTile{Pos: pos, TileTypeID: g.CurrentTile, Rotation: rotation, PlacedBy: playerID, TurnNumber: g.TurnNumber}
	g.LastPlaced = &pos
	g.Phase = model.PhasePlaceMeeple
	return nil
}

// SkipMeeple завершує хід без міпла (US-14): наступний гравець тягне тайл.
// Якщо тайлів, які можна покласти, не лишилося, партія завершується (Finished), а місце, номер ходу
// і фаза лишаються значеннями на момент завершення (docs/api.md).
func SkipMeeple(g *model.Game, seat, players int) error {
	if players <= 0 {
		return ErrNoPlayers
	}
	if Finished(g) {
		return ErrWrongPhase
	}
	if seat != g.CurrentSeat {
		return ErrNotYourTurn
	}
	if g.Phase != model.PhasePlaceMeeple {
		return ErrWrongPhase
	}
	if !draw(g) {
		return nil
	}
	g.CurrentSeat = (seat + 1) % players
	g.TurnNumber++
	g.Phase = model.PhasePlaceTile
	g.LastPlaced = nil
	return nil
}

// draw тягне тайли з верху стопки (US-09). Тайл, який нікуди не лягає, скидається в g.Discarded,
// і тягнеться наступний. Повертає false, якщо стопка скінчилася (CurrentTile = "").
func draw(g *model.Game) bool {
	g.Discarded = nil
	for len(g.Deck) > 0 {
		id := g.Deck[0]
		g.Deck = g.Deck[1:]
		if canPlaceAnywhere(g.Tiles, id) {
			g.CurrentTile = id
			return true
		}
		g.Discarded = append(g.Discarded, id)
	}
	g.CurrentTile = ""
	return false
}

func canPlaceAnywhere(tiles map[model.Position]model.PlacedTile, tileTypeID string) bool {
	for _, pos := range frontier(tiles) {
		for _, r := range Rotations {
			if CheckPlacement(tiles, tileTypeID, pos, r) == nil {
				return true
			}
		}
	}
	return false
}
