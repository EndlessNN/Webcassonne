package room

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
)

// newLobby — кімната з організатором names[0] і рештою гравців; помилки підготовки валять тест.
func newLobby(t *testing.T, s *Service, names ...string) (*model.Room, []*model.Player) {
	t.Helper()
	r, host, err := s.Create(names[0])
	if err != nil {
		t.Fatal(err)
	}
	players := []*model.Player{host}
	for _, n := range names[1:] {
		p, err := s.Join(r.Code, n)
		if err != nil {
			t.Fatal(err)
		}
		players = append(players, p)
	}
	return r, players
}

// startedGame — партія 2 гравців (Олег — організатор, Ірина); повертає активного й неактивного гравця.
func startedGame(t *testing.T, s *Service) (r *model.Room, active, passive *model.Player) {
	t.Helper()
	r, ps := newLobby(t, s, "Олег", "Ірина")
	if err := s.StartGame(r.Code, ps[0].ID); err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		if p.Seat == r.Game.CurrentSeat {
			active = p
		} else {
			passive = p
		}
	}
	if active == nil || passive == nil {
		t.Fatalf("місця після старту: %+v", ps)
	}
	return r, active, passive
}

// state — знімок кімнати під м'ютексом сервісу.
func state(s *Service, r *model.Room) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return snapshot(r)
}

// wantError — помилка з кодом і текстом з docs/api.md §1.
func wantError(t *testing.T, err error, code, message string) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Code != code || e.Message != message {
		t.Fatalf("err = %v, want %s: %s", err, code, message)
	}
}

// assertSameState — після відхиленої дії знімок кімнати (статус, гравці, партія з тайлами, стопкою, скинутими,
// фазою, ходом, а також похідні legalPlacements, міпли й рахунок) повністю збігається зі знімком до дії.
func assertSameState(t *testing.T, before, after Snapshot) {
	t.Helper()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("стан змінився попри помилку:\nдо    %s\nпісля %s", describe(before), describe(after))
	}
}

func describe(s Snapshot) string {
	out := fmt.Sprintf("status=%s host=%s finishedAt=%v players=", s.Status, s.HostPlayerID, s.FinishedAt)
	for _, p := range s.Players {
		out += fmt.Sprintf("[%s seat=%d color=%s] ", p.Nickname, p.Seat, p.Color)
	}
	if g := s.Game; g != nil {
		out += fmt.Sprintf("game={%+v lastPlaced=%v}", *g, g.LastPlaced)
	}
	return out + fmt.Sprintf(" legal=%v meeples=%v", s.LegalPlacements, s.MeeplesInSupply)
}

// US-05: запуск партії. Очікування — з AC US-05, П-9 і docs/api.md.
func TestUS05_StartGame(t *testing.T) {
	colors := []model.Color{model.ColorRed, model.ColorBlue, model.ColorGreen, model.ColorYellow, model.ColorBlack}

	success := []struct {
		name  string
		names []string
	}{
		{"успішний старт", []string{"Олег", "Ірина", "Андрій"}},
		{"2 гравці — мінімум", []string{"Олег", "Ірина"}},
		{"5 гравців — максимум", []string{"Олег", "Ірина", "Андрій", "Марта", "Тарас"}},
	}
	for _, tt := range success {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService()
			r, ps := newLobby(t, s, tt.names...)
			subs := make([]*Sub, len(ps))
			for i, p := range ps {
				sub, err := s.Connect(r.Code, p.Token)
				if err != nil {
					t.Fatal(err)
				}
				subs[i] = sub
			}

			if err := s.StartGame(r.Code, ps[0].ID); err != nil {
				t.Fatalf("StartGame: %v", err)
			}

			// Усі бачать ігровий стіл: кожна підписка отримує знімок PLAYING.
			for i, sub := range subs {
				select {
				case snap := <-sub.Updates:
					if snap.Status != model.RoomPlaying || snap.Game == nil {
						t.Fatalf("гравець %d: status = %s", i, snap.Status)
					}
				case <-time.After(time.Second):
					t.Fatalf("гравець %d не отримав знімок", i)
				}
			}

			snap := state(s, r)
			if len(snap.Players) != len(tt.names) {
				t.Fatalf("players = %d, want %d", len(snap.Players), len(tt.names))
			}
			// Порядок ходів: місця — перестановка 0…n−1; колір — за місцем (П-9); 7 міплів у кожного.
			seen := map[int]bool{}
			for _, p := range snap.Players {
				if p.Seat < 0 || p.Seat >= len(tt.names) || seen[p.Seat] {
					t.Fatalf("місця не перестановка: %+v", snap.Players)
				}
				seen[p.Seat] = true
				if p.Color != colors[p.Seat] {
					t.Fatalf("%s: місце %d, колір %s, want %s", p.Nickname, p.Seat, p.Color, colors[p.Seat])
				}
				if snap.MeeplesInSupply[p.ID] != 7 {
					t.Fatalf("%s: міплів %d, want 7", p.Nickname, snap.MeeplesInSupply[p.ID])
				}
			}

			g := snap.Game
			// Стартовий тайл у центрі; лічильник стопки 70; ходить місце 0, хід 1.
			if start, ok := g.Tiles[model.Position{}]; len(g.Tiles) != 1 || !ok || start.TileTypeID != "D" || start.Rotation != 0 {
				t.Fatalf("tiles = %+v", g.Tiles)
			}
			if len(g.Deck) != 70 || g.CurrentTile == "" {
				t.Fatalf("deckCount = %d, currentTile = %q", len(g.Deck), g.CurrentTile)
			}
			if g.CurrentSeat != 0 || g.Phase != model.PhasePlaceTile || g.TurnNumber != 1 {
				t.Fatalf("seat = %d, phase = %s, turn = %d", g.CurrentSeat, g.Phase, g.TurnNumber)
			}
			// Поточний гравець — той, у кого місце 0.
			first := snap.SeatPlayer[g.CurrentSeat]
			found := false
			for _, p := range snap.Players {
				if p.ID == first {
					found = p.Seat == 0
				}
			}
			if !found {
				t.Fatalf("поточний гравець %q не має місця 0", first)
			}
			if len(snap.LegalPlacements) == 0 {
				t.Fatal("legalPlacements порожні у фазі PLACE_TILE")
			}
		})
	}

	lobbyUnchanged := func(t *testing.T, s *Service, r *model.Room) {
		t.Helper()
		snap := state(s, r)
		if snap.Status != model.RoomLobby || snap.Game != nil {
			t.Fatalf("status = %s, game = %+v", snap.Status, snap.Game)
		}
		for _, p := range snap.Players {
			if p.Seat != -1 || p.Color != "" {
				t.Fatalf("%s: seat = %d, color = %q", p.Nickname, p.Seat, p.Color)
			}
		}
	}

	t.Run("NOT_ENOUGH_PLAYERS", func(t *testing.T) {
		s := newTestService()
		r, ps := newLobby(t, s, "Олег")
		wantError(t, s.StartGame(r.Code, ps[0].ID), "NOT_ENOUGH_PLAYERS", "Потрібно щонайменше 2 гравці")
		lobbyUnchanged(t, s, r)
	})

	t.Run("NOT_HOST", func(t *testing.T) {
		s := newTestService()
		r, ps := newLobby(t, s, "Олег", "Ірина")
		wantError(t, s.StartGame(r.Code, ps[1].ID), "NOT_HOST", "Почати гру може лише організатор")
		lobbyUnchanged(t, s, r)
	})

	t.Run("GAME_ALREADY_STARTED", func(t *testing.T) {
		s := newTestService()
		r, active, passive := startedGame(t, s)
		before := state(s, r)
		// Статус перевіряється раніше за автора (api.md), тож і не організатор отримує GAME_ALREADY_STARTED.
		for _, p := range []*model.Player{active, passive} {
			wantError(t, s.StartGame(r.Code, p.ID), "GAME_ALREADY_STARTED", "Партія вже почалася")
			assertSameState(t, before, state(s, r))
		}
	})
}

// US-05 «порядок ходів визначено випадково»: один прогін випадковість не доводить, тому фіксуємо генератор
// кімнати (поле rng, AGENTS.md: генератор передається параметром) 20 сідами і перевіряємо, що порядок місць
// залежить від генератора, а не від порядку входу: трапляються різні порядки, і організатор ходить не завжди першим.
func TestUS05_TurnOrderIsShuffled(t *testing.T) {
	orders := map[string]bool{}
	hostFirst := 0
	const seeds = 20
	for seed := range uint64(seeds) {
		s := newTestService()
		s.rng = rand.New(rand.NewPCG(seed, 2))
		r, ps := newLobby(t, s, "Олег", "Ірина", "Андрій")
		if err := s.StartGame(r.Code, ps[0].ID); err != nil {
			t.Fatal(err)
		}
		snap := state(s, r) // гравці в порядку входу: перший — організатор
		order := ""
		for _, p := range snap.Players {
			order += strconv.Itoa(p.Seat)
		}
		orders[order] = true
		if snap.Players[0].Seat == 0 {
			hostFirst++
		}
	}
	if len(orders) < 2 || hostFirst == seeds {
		t.Fatalf("порядок місць не залежить від генератора: порядки %v, організатор першим у %d з %d", orders, hostFirst, seeds)
	}
}

// US-07: спроба діяти не у свій хід — дія не виконується, стан не змінюється.
func TestUS07_NotYourTurn(t *testing.T) {
	s := newTestService()
	r, _, passive := startedGame(t, s)
	snap := state(s, r)
	if len(snap.LegalPlacements) == 0 || len(snap.LegalPlacements[0].Rotations) == 0 {
		t.Fatalf("підготовка: legalPlacements = %+v", snap.LegalPlacements)
	}
	before := snap
	legal := snap.LegalPlacements[0]

	actions := []struct {
		name string
		do   func() error
	}{
		{"покласти тайл на допустиму позицію", func() error {
			return s.PlaceTile(r.Code, passive.ID, legal.Pos, legal.Rotations[0])
		}},
		// api.md: автор перевіряється раніше за правило гри.
		{"покласти тайл на зайняту клітинку", func() error { return s.PlaceTile(r.Code, passive.ID, model.Position{}, 0) }},
		// api.md: автор перевіряється раніше за фазу (skip_meeple у PLACE_TILE).
		{"пропустити міпла", func() error { return s.SkipMeeple(r.Code, passive.ID) }},
	}
	for _, a := range actions {
		t.Run(a.name, func(t *testing.T) {
			wantError(t, a.do(), "NOT_YOUR_TURN", "Зараз хід іншого гравця")
			assertSameState(t, before, state(s, r))
		})
	}
}

// setTurn — підміняє поточний тайл і стопку активного гравця, щоб сценарій не залежав від перемішування.
func setTurn(s *Service, r *model.Room, currentTile string, deck []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r.Game.CurrentTile = currentTile
	r.Game.Deck = deck
}

// US-12 через сервер: успішне розміщення і тексти ILLEGAL_PLACEMENT з docs/api.md §1.
// Поле — лише стартовий D (обертання 0): N місто, E/W поле-дорога-поле, S поле. Тайл — E (місто на півночі).
func TestUS12_PlaceTile(t *testing.T) {
	errorsCases := []struct {
		name     string
		pos      model.Position
		rotation int
		code     string
		message  string
	}{
		// AC «Недопустима позиція»: місто E (обертання 0) проти поля на півдні D.
		{"недопустима позиція", model.Position{X: 0, Y: 1}, 0, "ILLEGAL_PLACEMENT", "Сторони тайлів мають збігатися"},
		{"клітинка без сусідів", model.Position{X: 5, Y: 5}, 0, "ILLEGAL_PLACEMENT", "Тайл має межувати з іншим тайлом"},
		{"зайнята клітинка", model.Position{}, 0, "ILLEGAL_PLACEMENT", "Клітинка вже зайнята"},
		{"VALIDATION_ERROR: обертання 45", model.Position{X: 0, Y: -1}, 45, "VALIDATION_ERROR", "Неправильне повідомлення"},
	}
	for _, tt := range errorsCases {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService()
			r, active, _ := startedGame(t, s)
			setTurn(s, r, "E", []string{"C", "U", "B", "B"})
			before := state(s, r)

			wantError(t, s.PlaceTile(r.Code, active.ID, tt.pos, tt.rotation), tt.code, tt.message)
			assertSameState(t, before, state(s, r))
		})
	}

	t.Run("розміщення на допустиму позицію", func(t *testing.T) {
		s := newTestService()
		r, active, _ := startedGame(t, s)
		setTurn(s, r, "E", []string{"C", "U", "B", "B"})
		seat := r.Game.CurrentSeat

		// E 180°: місто на півдні ↔ місто D на півночі.
		if err := s.PlaceTile(r.Code, active.ID, model.Position{X: 0, Y: -1}, 180); err != nil {
			t.Fatal(err)
		}
		snap := state(s, r)
		g := snap.Game
		placed, ok := g.Tiles[model.Position{X: 0, Y: -1}]
		if !ok || placed.TileTypeID != "E" || placed.Rotation != 180 || placed.PlacedBy != active.ID || len(g.Tiles) != 2 {
			t.Fatalf("tiles = %+v", g.Tiles)
		}
		// Ірині пропонується поставити міпла або пропустити: фаза PLACE_MEEPLE, хід той самий.
		if g.Phase != model.PhasePlaceMeeple || g.CurrentSeat != seat || len(g.Deck) != 4 || g.TurnNumber != 1 {
			t.Fatalf("phase = %s, seat = %d, deck = %d, turn = %d", g.Phase, g.CurrentSeat, len(g.Deck), g.TurnNumber)
		}
		// api.md: legalPlacements лише у фазі PLACE_TILE.
		if len(snap.LegalPlacements) != 0 {
			t.Fatalf("legalPlacements = %+v, want []", snap.LegalPlacements)
		}

		// api.md WRONG_PHASE: другий place_tile у фазі PLACE_MEEPLE.
		before := snap
		wantError(t, s.PlaceTile(r.Code, active.ID, model.Position{X: 0, Y: 1}, 180), "WRONG_PHASE", "Ця дія зараз недоступна")
		assertSameState(t, before, state(s, r))
	})
}

// US-09 через сервер: після ходу наступний гравець тягне тайл; тайл, який неможливо покласти, скидається.
func TestUS09_DrawAfterTurn(t *testing.T) {
	s := newTestService()
	r, active, passive := startedGame(t, s)
	setTurn(s, r, "E", []string{"C", "U", "B", "B"})

	// E 180° над D закриває місто: далі C (місто на всіх сторонах) не лягає нікуди (див. game.TestUS09_DrawTile).
	if err := s.PlaceTile(r.Code, active.ID, model.Position{X: 0, Y: -1}, 180); err != nil {
		t.Fatal(err)
	}
	if err := s.SkipMeeple(r.Code, active.ID); err != nil {
		t.Fatal(err)
	}

	snap := state(s, r)
	g := snap.Game
	if !slices.Equal(g.Discarded, []string{"C"}) || g.CurrentTile != "U" || len(g.Deck) != 2 {
		t.Fatalf("discarded = %v, currentTile = %q, deckCount = %d; want [C], U, 2", g.Discarded, g.CurrentTile, len(g.Deck))
	}
	if snap.SeatPlayer[g.CurrentSeat] != passive.ID || g.TurnNumber != 2 || g.Phase != model.PhasePlaceTile {
		t.Fatalf("current = %s, turn = %d, phase = %s", snap.SeatPlayer[g.CurrentSeat], g.TurnNumber, g.Phase)
	}
	if snap.Status != model.RoomPlaying || len(snap.LegalPlacements) == 0 {
		t.Fatalf("status = %s, legalPlacements = %+v", snap.Status, snap.LegalPlacements)
	}
}

// US-15: знімок, який отримала підписка, серіалізується поза s.mu (httpapi), тож він має бути незалежною копією:
// подальші зміни кімнати його не змінюють. -race цього не ловить, якщо читання й запис не перетнулися в часі.
func TestUS15_SnapshotIsIndependentCopy(t *testing.T) {
	t.Run("старт партії не змінює знімок лобі", func(t *testing.T) {
		s := newTestService()
		r, ps := newLobby(t, s, "Олег", "Ірина")
		sub := mustConnect(t, s, r.Code, ps[0].Token)
		snap := recv(t, sub)

		if err := s.StartGame(r.Code, ps[0].ID); err != nil {
			t.Fatal(err)
		}
		// Знімок лобі: місця −1, кольорів немає, партії немає (api.md: color, seat — null у LOBBY).
		if snap.Status != model.RoomLobby || snap.Game != nil || len(snap.Players) != 2 {
			t.Fatalf("знімок змінився: status = %s, game = %+v, players = %d", snap.Status, snap.Game, len(snap.Players))
		}
		for _, p := range snap.Players {
			if p.Seat != -1 || p.Color != "" {
				t.Fatalf("знімок лобі змінився після старту: %s seat = %d, color = %q", p.Nickname, p.Seat, p.Color)
			}
		}
	})

	t.Run("хід не змінює знімок партії", func(t *testing.T) {
		s := newTestService()
		r, active, _ := startedGame(t, s)
		setTurn(s, r, "E", []string{"C", "U", "B", "B"})
		sub := mustConnect(t, s, r.Code, active.Token)
		snap := recv(t, sub)
		seat := snap.Game.CurrentSeat

		// E 180° над D, без міпла: далі C скидається (як у TestUS09_DrawAfterTurn) — змінюються тайли, стопка,
		// поточний тайл, скинуті, фаза, місце, номер ходу, останній покладений.
		if err := s.PlaceTile(r.Code, active.ID, model.Position{X: 0, Y: -1}, 180); err != nil {
			t.Fatal(err)
		}
		if err := s.SkipMeeple(r.Code, active.ID); err != nil {
			t.Fatal(err)
		}

		g := snap.Game
		if len(g.Tiles) != 1 || g.Tiles[model.Position{}] != (model.PlacedTile{TileTypeID: "D"}) {
			t.Fatalf("тайли у знімку змінилися: %+v", g.Tiles)
		}
		if !slices.Equal(g.Deck, []string{"C", "U", "B", "B"}) || g.CurrentTile != "E" || len(g.Discarded) != 0 {
			t.Fatalf("стопка у знімку змінилася: deck = %v, currentTile = %q, discarded = %v", g.Deck, g.CurrentTile, g.Discarded)
		}
		if g.Phase != model.PhasePlaceTile || g.CurrentSeat != seat || g.TurnNumber != 1 || g.LastPlaced != nil {
			t.Fatalf("хід у знімку змінився: phase = %s, seat = %d, turn = %d, lastPlaced = %v", g.Phase, g.CurrentSeat, g.TurnNumber, g.LastPlaced)
		}
		if snap.Status != model.RoomPlaying || len(snap.LegalPlacements) != 2 {
			t.Fatalf("status = %s, legalPlacements = %+v", snap.Status, snap.LegalPlacements)
		}
	})
}
