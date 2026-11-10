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
	s := store.NewMemory()
	rooms := room.NewService(s)
	rooms.LeaveGrace = 0
	srv := httptest.NewServer(NewRouter(s, rooms, nil))
	t.Cleanup(srv.Close)
	return srv
}

func TestCreateAndJoin_StatusCodes(t *testing.T) {
	srv := newTestServer(t)
	var created createRoomResponse
	if code := post(t, srv, "/api/rooms", `{"nickname":"Олег"}`, &created); code != http.StatusCreated {
		t.Fatalf("create = %d", code)
	}

	tests := []struct {
		name, path, body string
		status           int
		code             string
	}{
		{"порожній нікнейм", "/api/rooms", `{"nickname":""}`, 400, "VALIDATION_ERROR"},
		{"неправильний JSON", "/api/rooms", `{`, 400, "VALIDATION_ERROR"},
		{"успішний вхід", "/api/rooms/" + created.RoomCode + "/players", `{"nickname":"Ірина"}`, 201, ""},
		{"нікнейм зайнятий", "/api/rooms/" + created.RoomCode + "/players", `{"nickname":"олег"}`, 409, "NICKNAME_TAKEN"},
		{"кімнати немає", "/api/rooms/ZZZZZZ/players", `{"nickname":"Андрій"}`, 404, "ROOM_NOT_FOUND"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body ErrorBody
			status := post(t, srv, tt.path, tt.body, &body)
			if status != tt.status || body.Error.Code != tt.code {
				t.Fatalf("got %d %q, want %d %q", status, body.Error.Code, tt.status, tt.code)
			}
		})
	}
}

func TestPreview(t *testing.T) {
	srv := newTestServer(t)
	var created createRoomResponse
	post(t, srv, "/api/rooms", `{"nickname":"Олег"}`, &created)

	res, err := http.Get(srv.URL + "/api/rooms/" + created.RoomCode)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var got roomPreview
	json.NewDecoder(res.Body).Decode(&got)
	want := roomPreview{Code: created.RoomCode, Status: "LOBBY", PlayerCount: 1, MaxPlayers: 5}
	if res.StatusCode != 200 || got != want {
		t.Fatalf("got %d %+v", res.StatusCode, got)
	}

	res, _ = http.Get(srv.URL + "/api/rooms/ZZZZZZ")
	res.Body.Close()
	if res.StatusCode != 404 {
		t.Fatalf("missing room = %d", res.StatusCode)
	}
}

func TestSocket_InvalidTokenRejectedBeforeUpgrade(t *testing.T) {
	srv := newTestServer(t)
	var created createRoomResponse
	post(t, srv, "/api/rooms", `{"nickname":"Олег"}`, &created)

	res, err := http.Get(srv.URL + "/api/rooms/" + created.RoomCode + "/ws?token=wrong")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
}

// US-04: новий гравець з'являється в лобі без перезавантаження, а той, хто вийшов, зникає.
func TestSocket_LobbyUpdates(t *testing.T) {
	srv := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var oleg createRoomResponse
	post(t, srv, "/api/rooms", `{"nickname":"Олег"}`, &oleg)
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

	var iryna joinRoomResponse
	post(t, srv, "/api/rooms/"+oleg.RoomCode+"/players", `{"nickname":"Ірина"}`, &iryna)
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
