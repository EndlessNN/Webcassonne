package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// wsMessage — будь-яке повідомлення сервера: room_state або error.
type wsMessage struct {
	Type  string    `json:"type"`
	Room  *roomView `json:"room"`
	Error *APIError `json:"error"`
	raw   []byte
}

// wsClient — WebSocket-з'єднання гравця з тестовим сервером.
type wsClient struct {
	t    *testing.T
	ctx  context.Context
	conn *websocket.Conn
}

func (c *wsClient) read() wsMessage {
	c.t.Helper()
	_, data, err := c.conn.Read(c.ctx)
	if err != nil {
		c.t.Fatal(err)
	}
	var m wsMessage
	if err := json.Unmarshal(data, &m); err != nil {
		c.t.Fatal(err)
	}
	m.raw = data
	return m
}

func (c *wsClient) send(v any) {
	c.t.Helper()
	if err := wsjson.Write(c.ctx, c.conn, v); err != nil {
		c.t.Fatal(err)
	}
}

// readState — наступне повідомлення має бути room_state.
func (c *wsClient) readState() roomView {
	c.t.Helper()
	m := c.read()
	if m.Type != "room_state" || m.Room == nil {
		c.t.Fatalf("want room_state, got %s", m.raw)
	}
	return *m.Room
}

// readError — наступне повідомлення має бути error з кодом і текстом з docs/api.md §1.
func (c *wsClient) readError(code, message string) {
	c.t.Helper()
	m := c.read()
	if m.Type != "error" || m.Error == nil || m.Error.Code != code || m.Error.Message != message {
		c.t.Fatalf("want error %s %q, got %s", code, message, m.raw)
	}
}

// gameLobby — кімната з організатором Олегом і гравцями others; усі підключені, початкові знімки прочитано.
func gameLobby(t *testing.T, others ...string) (clients []*wsClient, ids, tokens []string) {
	t.Helper()
	srv := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	var oleg createRoomResponse
	if code := post(t, srv, "/api/rooms", `{"nickname":"Олег"}`, &oleg); code != http.StatusCreated {
		t.Fatalf("create: %d", code)
	}
	ids, tokens = []string{oleg.PlayerID}, []string{oleg.Token}
	for _, n := range others {
		var p joinRoomResponse
		if code := post(t, srv, "/api/rooms/"+oleg.RoomCode+"/players", `{"nickname":"`+n+`"}`, &p); code != http.StatusCreated {
			t.Fatalf("join %s: %d", n, code)
		}
		ids, tokens = append(ids, p.PlayerID), append(tokens, p.Token)
	}

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/rooms/" + oleg.RoomCode + "/ws?token="
	for i, token := range tokens {
		conn, _, err := websocket.Dial(ctx, wsURL+token, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.CloseNow() })
		c := &wsClient{t: t, ctx: ctx, conn: conn}
		// Новий клієнт отримує свій знімок, а вже підключені — знімок про його підключення.
		for _, prev := range append(clients, c) {
			if s := prev.readState(); len(s.Players) != len(tokens) {
				t.Fatalf("клієнт %d: players = %d", i, len(s.Players))
			}
		}
		clients = append(clients, c)
	}
	return clients, ids, tokens
}

// US-05 через WebSocket: NOT_HOST лише автору, після старту обидва отримують однаковий знімок без токенів,
// повторний старт — GAME_ALREADY_STARTED.
func TestSocket_US05_StartGame(t *testing.T) {
	clients, ids, tokens := gameLobby(t, "Ірина")
	oleg, iryna := clients[0], clients[1]

	iryna.send(map[string]string{"type": "start_game"})
	iryna.readError("NOT_HOST", "Почати гру може лише організатор")

	oleg.send(map[string]string{"type": "start_game"})
	// Якби помилка Ірини дійшла й до Олега, тут було б error, а не room_state.
	var states []roomView
	for i, c := range clients {
		m := c.read()
		if m.Type != "room_state" || m.Room == nil {
			t.Fatalf("клієнт %d: want room_state, got %s", i, m.raw)
		}
		for _, token := range tokens {
			if strings.Contains(string(m.raw), token) {
				t.Fatalf("клієнт %d: знімок містить токен гравця", i)
			}
		}
		states = append(states, *m.Room)
	}

	for i, s := range states {
		if s.YouPlayerID != ids[i] {
			t.Fatalf("клієнт %d: youPlayerId = %s", i, s.YouPlayerID)
		}
		if s.Status != "PLAYING" || s.Game == nil {
			t.Fatalf("клієнт %d: status = %s", i, s.Status)
		}
		g := s.Game
		if g.DeckCount != 70 || g.CurrentTile == nil || g.TurnNumber != 1 || g.Phase != "PLACE_TILE" {
			t.Fatalf("клієнт %d: game = %+v", i, g)
		}
		if len(g.Tiles) != 1 || g.Tiles[0] != (placedTileView{X: 0, Y: 0, TileTypeID: "D", Rotation: 0}) {
			t.Fatalf("клієнт %d: tiles = %+v", i, g.Tiles)
		}
		if len(g.LegalPlacements) == 0 || g.DiscardedTiles == nil || len(g.DiscardedTiles) != 0 {
			t.Fatalf("клієнт %d: legalPlacements = %+v, discardedTiles = %+v", i, g.LegalPlacements, g.DiscardedTiles)
		}
		// US-07: currentPlayerId — гравець з місцем 0, однаковий для всіх.
		if g.CurrentPlayerID != states[0].Game.CurrentPlayerID {
			t.Fatalf("currentPlayerId різний: %s і %s", g.CurrentPlayerID, states[0].Game.CurrentPlayerID)
		}
		if len(s.Players) != 2 {
			t.Fatalf("клієнт %d: players = %d", i, len(s.Players))
		}
		for _, p := range s.Players {
			if p.Seat == nil || p.Color == nil || p.MeeplesInSupply != 7 {
				t.Fatalf("клієнт %d: player = %+v", i, p)
			}
			if (*p.Seat == 0) != (p.ID == g.CurrentPlayerID) {
				t.Fatalf("клієнт %d: currentPlayerId %s, але %s має місце %d", i, g.CurrentPlayerID, p.ID, *p.Seat)
			}
		}
	}

	oleg.send(map[string]string{"type": "start_game"})
	oleg.readError("GAME_ALREADY_STARTED", "Партія вже почалася")
}

func TestSocket_US05_NotEnoughPlayers(t *testing.T) {
	clients, _, _ := gameLobby(t)
	clients[0].send(map[string]string{"type": "start_game"})
	clients[0].readError("NOT_ENOUGH_PLAYERS", "Потрібно щонайменше 2 гравці")
}

// US-07 і US-12 через WebSocket: помилки лише автору з текстами з docs/api.md §1;
// допустимий place_tile — знімок усім.
func TestSocket_US07_US12_GameErrors(t *testing.T) {
	clients, ids, _ := gameLobby(t, "Ірина")
	clients[0].send(map[string]string{"type": "start_game"})
	s0, s1 := clients[0].readState(), clients[1].readState()
	if s0.Game == nil || s1.Game == nil {
		t.Fatal("партія не почалася")
	}

	var active, passive *wsClient
	switch s0.Game.CurrentPlayerID {
	case ids[0]:
		active, passive = clients[0], clients[1]
	case ids[1]:
		active, passive = clients[1], clients[0]
	default:
		t.Fatalf("currentPlayerId = %s", s0.Game.CurrentPlayerID)
	}
	if len(s0.Game.LegalPlacements) == 0 || len(s0.Game.LegalPlacements[0].Rotations) == 0 {
		t.Fatalf("legalPlacements = %+v", s0.Game.LegalPlacements)
	}
	legal := s0.Game.LegalPlacements[0]

	passive.send(map[string]any{"type": "place_tile", "x": legal.X, "y": legal.Y, "rotation": legal.Rotations[0]})
	passive.readError("NOT_YOUR_TURN", "Зараз хід іншого гравця")

	active.send(map[string]any{"type": "place_tile", "x": 0, "y": 0, "rotation": 0})
	active.readError("ILLEGAL_PLACEMENT", "Клітинка вже зайнята")

	active.send(map[string]any{"type": "place_tile", "x": 5, "y": 5, "rotation": 0})
	active.readError("ILLEGAL_PLACEMENT", "Тайл має межувати з іншим тайлом")

	active.send(map[string]any{"type": "place_tile", "x": legal.X, "y": legal.Y, "rotation": 45})
	active.readError("VALIDATION_ERROR", "Неправильне повідомлення")

	active.send(map[string]any{"type": "rotate_tile"})
	active.readError("VALIDATION_ERROR", "Невідомий тип повідомлення")

	// Допустимий хід: обидва отримують знімок; помилки іншого гравця сюди не потрапили.
	active.send(map[string]any{"type": "place_tile", "x": legal.X, "y": legal.Y, "rotation": legal.Rotations[0]})
	for i, c := range []*wsClient{active, passive} {
		g := c.readState().Game
		if g == nil || g.Phase != "PLACE_MEEPLE" || len(g.Tiles) != 2 || g.DeckCount != 70 {
			t.Fatalf("клієнт %d: game = %+v", i, g)
		}
		if g.LegalPlacements == nil || len(g.LegalPlacements) != 0 || g.CurrentPlayerID != s0.Game.CurrentPlayerID {
			t.Fatalf("клієнт %d: legalPlacements = %+v, current = %s", i, g.LegalPlacements, g.CurrentPlayerID)
		}
	}
}
