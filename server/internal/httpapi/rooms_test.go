package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/EndlessNN/Webcassonne/server/internal/room"
	"github.com/EndlessNN/Webcassonne/server/internal/store"
)

func post(t *testing.T, srv *httptest.Server, path, body string, out any) int {
	t.Helper()
	res, err := http.Post(srv.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			t.Fatal(err)
		}
	}
	return res.StatusCode
}

func newTestServer(t *testing.T) *httptest.Server {
	srv, _ := newTestServerWithRooms(t)
	return srv
}

// newTestServerWithRooms — тестовий сервер і його сервіс кімнат (щоб довести кімнату до потрібного статусу).
func newTestServerWithRooms(t *testing.T) (*httptest.Server, *room.Service) {
	s := store.NewMemory()
	rooms := room.NewService(s)
	rooms.LeaveGrace = 0
	srv := httptest.NewServer(NewRouter(s, rooms, nil))
	t.Cleanup(srv.Close)
	return srv, rooms
}

// mustCreate — POST /api/rooms; не 201 валить тест.
func mustCreate(t *testing.T, srv *httptest.Server, nickname string) createRoomResponse {
	t.Helper()
	var created createRoomResponse
	if code := post(t, srv, "/api/rooms", `{"nickname":"`+nickname+`"}`, &created); code != http.StatusCreated {
		t.Fatalf("create %s: %d", nickname, code)
	}
	return created
}

// mustJoin — POST /api/rooms/{code}/players; не 201 валить тест.
func mustJoin(t *testing.T, srv *httptest.Server, code, nickname string) joinRoomResponse {
	t.Helper()
	var joined joinRoomResponse
	if status := post(t, srv, "/api/rooms/"+code+"/players", `{"nickname":"`+nickname+`"}`, &joined); status != http.StatusCreated {
		t.Fatalf("join %s: %d", nickname, status)
	}
	return joined
}

// getError — GET з очікуваною помилкою у форматі docs/api.md §1.
func getError(t *testing.T, url string) (int, APIError) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body ErrorBody
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, body.Error
}

// US-01, US-03 через REST: статуси й коди з docs/api.md §2, тексти — з AC (для неправильного JSON api.md тексту не задає).
func TestCreateAndJoin_StatusCodes(t *testing.T) {
	long := strings.Repeat("ж", 21)
	tests := []struct {
		name    string
		setup   func(t *testing.T, srv *httptest.Server, rooms *room.Service, code string) // nil — кімната лише з Олегом
		path    string                                                                     // %s — код кімнати
		body    string
		status  int
		code    string
		message string // "" — текст не перевіряємо
		field   string // details.field; "" — без details
	}{
		{"порожній нікнейм", nil, "/api/rooms", `{"nickname":""}`, 400, "VALIDATION_ERROR", "Введіть нікнейм", "nickname"},
		{"задовгий нікнейм", nil, "/api/rooms", `{"nickname":"` + long + `"}`, 400, "VALIDATION_ERROR", "Нікнейм — від 1 до 20 символів", "nickname"},
		{"неправильний JSON", nil, "/api/rooms", `{`, 400, "VALIDATION_ERROR", "", ""},
		{"невідоме поле", nil, "/api/rooms", `{"nickname":"Олег","extra":1}`, 400, "VALIDATION_ERROR", "", ""},
		{"зайві байти після JSON", nil, "/api/rooms", `{"nickname":"Олег"}garbage`, 400, "VALIDATION_ERROR", "", ""},
		{"успішний вхід", nil, "/api/rooms/%s/players", `{"nickname":"Ірина"}`, 201, "", "", ""},
		{"нікнейм зайнятий", nil, "/api/rooms/%s/players", `{"nickname":"олег"}`, 409, "NICKNAME_TAKEN", "Цей нікнейм уже зайнятий у кімнаті", "nickname"},
		{"кімнати немає", nil, "/api/rooms/ZZZZZZ/players", `{"nickname":"Андрій"}`, 404, "ROOM_NOT_FOUND", "Кімнату не знайдено", ""},
		{"кімната заповнена", func(t *testing.T, srv *httptest.Server, _ *room.Service, code string) {
			for _, n := range []string{"Ірина", "Андрій", "Марта", "Тарас"} {
				mustJoin(t, srv, code, n)
			}
		}, "/api/rooms/%s/players", `{"nickname":"Шостий"}`, 409, "ROOM_FULL", "Кімната заповнена (максимум 5 гравців)", ""},
		{"партія вже почалася", func(t *testing.T, srv *httptest.Server, rooms *room.Service, code string) {
			mustJoin(t, srv, code, "Ірина")
			if err := rooms.StartGame(code, hostOf(t, rooms, code)); err != nil {
				t.Fatal(err)
			}
		}, "/api/rooms/%s/players", `{"nickname":"Андрій"}`, 409, "GAME_ALREADY_STARTED", "Партія вже почалася", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, rooms := newTestServerWithRooms(t)
			created := mustCreate(t, srv, "Олег")
			if tt.setup != nil {
				tt.setup(t, srv, rooms, created.RoomCode)
			}
			path := strings.Replace(tt.path, "%s", created.RoomCode, 1)
			if tt.status == http.StatusCreated {
				var joined joinRoomResponse
				if status := post(t, srv, path, tt.body, &joined); status != tt.status || joined.PlayerID == "" || len(joined.Token) != 32 {
					t.Fatalf("got %d %+v, want 201 з playerId і token", status, joined)
				}
				return
			}
			var body ErrorBody
			status := post(t, srv, path, tt.body, &body)
			e := body.Error
			if status != tt.status || e.Code != tt.code || e.Message == "" || (tt.message != "" && e.Message != tt.message) {
				t.Fatalf("got %d %+v, want %d %s %q", status, e, tt.status, tt.code, tt.message)
			}
			if field, _ := e.Details["field"].(string); field != tt.field {
				t.Fatalf("details.field = %q, want %q", field, tt.field)
			}
		})
	}
}

// hostOf — ID організатора кімнати.
func hostOf(t *testing.T, rooms *room.Service, code string) string {
	t.Helper()
	r, err := rooms.Preview(code)
	if err != nil {
		t.Fatal(err)
	}
	return r.HostPlayerID
}

// US-03: попередній перегляд кімнати і «Кімнату не знайдено».
func TestPreview(t *testing.T) {
	srv := newTestServer(t)
	created := mustCreate(t, srv, "Олег")

	res, err := http.Get(srv.URL + "/api/rooms/" + created.RoomCode)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var got roomPreview
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	want := roomPreview{Code: created.RoomCode, Status: "LOBBY", PlayerCount: 1, MaxPlayers: 5}
	if res.StatusCode != 200 || got != want {
		t.Fatalf("got %d %+v", res.StatusCode, got)
	}

	status, e := getError(t, srv.URL+"/api/rooms/ZZZZZZ")
	if status != 404 || e.Code != "ROOM_NOT_FOUND" || e.Message != "Кімнату не знайдено" {
		t.Fatalf("missing room = %d %+v", status, e)
	}
}

// api.md §3: відмова до апгрейду — REST-формат помилки.
func TestSocket_InvalidTokenRejectedBeforeUpgrade(t *testing.T) {
	srv := newTestServer(t)
	created := mustCreate(t, srv, "Олег")

	tests := []struct {
		name, path, code, message string
		status                    int
	}{
		{"чужий токен", "/api/rooms/" + created.RoomCode + "/ws?token=wrong", "INVALID_TOKEN", "Недійсний токен гравця", 401},
		{"без токена", "/api/rooms/" + created.RoomCode + "/ws", "INVALID_TOKEN", "Недійсний токен гравця", 401},
		{"кімнати немає", "/api/rooms/ZZZZZZ/ws?token=" + created.Token, "ROOM_NOT_FOUND", "Кімнату не знайдено", 404},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, e := getError(t, srv.URL+tt.path)
			if status != tt.status || e.Code != tt.code || e.Message != tt.message {
				t.Fatalf("got %d %+v, want %d %s %q", status, e, tt.status, tt.code, tt.message)
			}
		})
	}
}

func TestSocket_InvalidOriginDoesNotDisconnectExistingPlayer(t *testing.T) {
	srv := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created := mustCreate(t, srv, "Олег")
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/rooms/" + created.RoomCode + "/ws?token=" + created.Token

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()

	var first roomStateMessage
	if err := wsjson.Read(ctx, conn, &first); err != nil {
		t.Fatal(err)
	}

	_, res, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"https://evil.example"}},
	})
	if err == nil {
		t.Fatal("second dial with invalid origin unexpectedly succeeded")
	}
	if res == nil || res.StatusCode == http.StatusSwitchingProtocols {
		t.Fatalf("second dial returned unexpected status: %#v", res)
	}

	mustJoin(t, srv, created.RoomCode, "Ірина")

	var update roomStateMessage
	if err := wsjson.Read(ctx, conn, &update); err != nil {
		t.Fatal(err)
	}
	if len(update.Room.Players) != 2 {
		t.Fatalf("players after invalid origin = %d, want 2", len(update.Room.Players))
	}
}

// US-04: новий гравець з'являється в лобі без перезавантаження, а той, хто вийшов, зникає.
func TestSocket_LobbyUpdates(t *testing.T) {
	srv := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	oleg := mustCreate(t, srv, "Олег")
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/rooms/" + oleg.RoomCode + "/ws?token="

	olegConn, _, err := websocket.Dial(ctx, wsURL+oleg.Token, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer olegConn.CloseNow()
	read := func() roomStateMessage {
		var m roomStateMessage
		if err := wsjson.Read(ctx, olegConn, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}

	first := read()
	if first.Type != "room_state" || first.Room.YouPlayerID != oleg.PlayerID || first.Room.HostPlayerID != oleg.PlayerID {
		t.Fatalf("first = %+v", first)
	}
	if p := first.Room.Players[0]; p.Color != nil || p.Seat != nil || p.MeeplesInSupply != 7 {
		t.Fatalf("lobby player = %+v", p)
	}

	iryna := mustJoin(t, srv, oleg.RoomCode, "Ірина")
	if m := read(); len(m.Room.Players) != 2 || m.Room.Players[1].Nickname != "Ірина" {
		t.Fatalf("after join = %+v", m.Room.Players)
	}

	irynaConn, _, err := websocket.Dial(ctx, wsURL+iryna.Token, nil)
	if err != nil {
		t.Fatal(err)
	}
	read() // знімок після підключення Ірини
	irynaConn.Close(websocket.StatusNormalClosure, "")
	if m := read(); len(m.Room.Players) != 1 {
		t.Fatalf("after leave = %+v", m.Room.Players)
	}
}
